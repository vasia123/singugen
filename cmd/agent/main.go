package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/mymmrac/telego"

	"github.com/vasis/singugen/internal/claude"
	"github.com/vasis/singugen/internal/comms"
	"github.com/vasis/singugen/internal/config"
	"github.com/vasis/singugen/internal/kanban"
	"github.com/vasis/singugen/internal/selfupdate"
	"github.com/vasis/singugen/internal/spawner"
	tg "github.com/vasis/singugen/internal/telegram"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		os.Exit(1)
	}

	logger := setupLogger(cfg.Log)
	logger.Info("agent starting")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		logger.Info("received signal, shutting down", "signal", sig)
		cancel()
	}()

	// Create message bus and agent pool.
	bus := comms.New()
	launcher := claude.NewExecLauncher(cfg.Agent.ClaudeBinary)
	pool := spawner.NewPool(launcher, bus, cfg.Agents.BaseDir, cfg.Agents.DefaultAgent, logger)
	defer pool.ShutdownAll()

	// Spawn agents from config.
	for name, def := range cfg.Agents.Definitions {
		if !def.Enabled {
			continue
		}
		model := def.Model
		if model == "" {
			model = cfg.Agent.ClaudeModel
		}
		if err := pool.Spawn(ctx, spawner.AgentConfig{
			Name:        name,
			Description: def.Description,
			Model:       model,
		}); err != nil {
			logger.Error("failed to spawn agent", "name", name, "error", err)
			os.Exit(1)
		}
	}

	// Create self-update pipeline if enabled.
	var updater *selfupdate.Updater
	if cfg.SelfUpdate.Enabled {
		projectDir, _ := os.Getwd()
		updater = selfupdate.NewUpdater(projectDir, selfupdate.ExecCommandRunner{}, logger)
		updater.SetProtectedDirs(cfg.SelfUpdate.ProtectedDirs)
		updater.SetAutoPush(cfg.SelfUpdate.AutoPush, cfg.SelfUpdate.PushBranch)
		logger.Info("self-update enabled")
	}

	// Initialize kanban board.
	board := kanban.NewBoard(cfg.Kanban.Path, logger)
	if err := board.Init(); err != nil {
		logger.Error("failed to init kanban board", "error", err)
		os.Exit(1)
	}

	// Start Telegram bot if token is configured.
	if cfg.Telegram.Token != "" {
		startTelegramBot(ctx, cfg, pool, board, updater, logger, cancel)
	}

	logger.Info("agent started",
		"agents", len(cfg.Agents.Definitions),
		"default", cfg.Agents.DefaultAgent,
	)

	// Block until context is cancelled.
	<-ctx.Done()
	logger.Info("agent stopped")
}

func startTelegramBot(ctx context.Context, cfg *config.Config, pool *spawner.Pool, board *kanban.Board, updater *selfupdate.Updater, logger *slog.Logger, cancel context.CancelFunc) {
	bot, err := telego.NewBot(cfg.Telegram.Token)
	if err != nil {
		logger.Error("failed to create telegram bot", "error", err)
		os.Exit(1)
	}

	sender := tg.NewTelegoSender(ctx, bot)
	tgBot := tg.NewBot(pool, sender, tg.BotConfig{
		AllowFrom: cfg.Telegram.AllowFrom,
	}, logger, cancel)
	if updater != nil {
		tgBot.SetUpdater(updater)
	}
	if board != nil {
		tgBot.SetBoard(board)
	}

	go func() {
		updates, err := bot.UpdatesViaLongPolling(ctx, nil)
		if err != nil {
			logger.Error("telegram long-polling failed", "error", err)
			cancel()
			return
		}

		logger.Info("telegram bot started")

		for update := range updates {
			if update.Message == nil {
				continue
			}
			msg := update.Message
			if msg.From == nil {
				continue
			}
			tgBot.HandleText(ctx, msg.Chat.ID, msg.From.ID, msg.Text)
		}

		logger.Info("telegram bot stopped")
	}()
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
