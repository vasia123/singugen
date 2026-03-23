package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/vasis/singugen/internal/config"
	"github.com/vasis/singugen/internal/supervisor"
)

var version = "dev"

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		os.Exit(1)
	}

	logger := setupLogger(cfg.Log)
	logger.Info("singugen supervisor starting", "version", version)

	runner := supervisor.NewExecRunner(cfg.Supervisor.ChildBinary)
	breaker := supervisor.NewCircuitBreaker(
		cfg.Supervisor.MaxRestarts,
		cfg.Supervisor.RestartWindow,
	)
	sup := supervisor.New(runner, breaker, logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGUSR1)

	go func() {
		for sig := range sigCh {
			switch sig {
			case syscall.SIGINT, syscall.SIGTERM:
				logger.Info("received shutdown signal", "signal", sig)
				cancel()
				return
			case syscall.SIGUSR1:
				logger.Info("received update signal, restarting child")
				sup.RestartChild()
			}
		}
	}()

	if err := sup.Run(ctx); err != nil {
		logger.Error("supervisor exited with error", "error", err)
		os.Exit(1)
	}

	logger.Info("supervisor stopped")
}

func setupLogger(cfg config.LogConfig) *slog.Logger {
	level := slog.LevelInfo
	switch cfg.Level {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}

	opts := &slog.HandlerOptions{Level: level}

	var handler slog.Handler
	if cfg.Format == "json" {
		handler = slog.NewJSONHandler(os.Stderr, opts)
	} else {
		handler = slog.NewTextHandler(os.Stderr, opts)
	}

	return slog.New(handler)
}
