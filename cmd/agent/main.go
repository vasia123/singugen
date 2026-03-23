package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/mymmrac/telego"

	"github.com/vasis/singugen/internal/agent"
	"github.com/vasis/singugen/internal/claude"
	"github.com/vasis/singugen/internal/config"
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

	launcher := claude.NewExecLauncher(cfg.Agent.ClaudeBinary)
	sess := claude.NewSession(claude.SessionConfig{
		Model:      cfg.Agent.ClaudeModel,
		Timeout:    cfg.Agent.ClaudeTimeout,
		MaxRetries: cfg.Agent.ClaudeMaxRetries,
	}, launcher, logger)

	a := agent.New(agent.Config{
		QueueSize: cfg.Agent.QueueSize,
	}, sess, logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		logger.Info("received signal, shutting down", "signal", sig)
		cancel()
	}()

	// CLI test mode.
	if msg := os.Getenv("SINGUGEN_AGENT_MESSAGE"); msg != "" {
		runSingleMessage(ctx, sess, a, msg, logger)
		return
	}

	// Start Claude session.
	if err := sess.Start(ctx); err != nil {
		logger.Error("failed to start claude session", "error", err)
		os.Exit(1)
	}
	defer sess.Close()

	// Start Telegram bot if token is configured.
	if cfg.Telegram.Token != "" {
		startTelegramBot(ctx, cfg, a, sess, logger, cancel)
	}

	logger.Info("agent started, waiting for messages")
	if err := a.Run(ctx); err != nil {
		logger.Error("agent exited with error", "error", err)
		os.Exit(1)
	}

	logger.Info("agent stopped")
}

func startTelegramBot(ctx context.Context, cfg *config.Config, a *agent.Agent, sess *claude.Session, logger *slog.Logger, cancel context.CancelFunc) {
	bot, err := telego.NewBot(cfg.Telegram.Token)
	if err != nil {
		logger.Error("failed to create telegram bot", "error", err)
		os.Exit(1)
	}

	sender := tg.NewTelegoSender(ctx, bot)
	tgBot := tg.NewBot(a, sess, sender, tg.BotConfig{
		AllowFrom: cfg.Telegram.AllowFrom,
	}, logger, cancel)

	// Run long-polling in background.
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

func runSingleMessage(ctx context.Context, sess *claude.Session, a *agent.Agent, msg string, logger *slog.Logger) {
	if err := sess.Start(ctx); err != nil {
		logger.Error("failed to start claude session", "error", err)
		os.Exit(1)
	}
	defer sess.Close()

	go a.Run(ctx)

	handler := &printHandler{logger: logger, done: make(chan struct{})}
	if err := a.Submit(agent.Request{Message: msg, Handler: handler}); err != nil {
		logger.Error("submit failed", "error", err)
		os.Exit(1)
	}

	<-handler.done
}

type printHandler struct {
	logger *slog.Logger
	done   chan struct{}
}

func (h *printHandler) OnEvent(event claude.Event) {
	if event.Type == claude.EventAssistant && event.Message != nil {
		for _, block := range event.Message.Content {
			if block.Type == "text" {
				fmt.Print(block.Text)
			}
		}
	}
}

func (h *printHandler) OnComplete(result string, err error) {
	if err != nil {
		h.logger.Error("error", "error", err)
	}
	fmt.Println()
	close(h.done)
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
