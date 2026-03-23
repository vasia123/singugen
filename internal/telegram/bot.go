package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/vasis/singugen/internal/agent"
	"github.com/vasis/singugen/internal/selfupdate"
	"github.com/vasis/singugen/internal/spawner"
)

// BotConfig holds bot configuration.
type BotConfig struct {
	AllowFrom []int64
}

// Bot bridges Telegram updates to agents.
type Bot struct {
	pool      *spawner.Pool
	sender    Sender
	updater   *selfupdate.Updater
	allowFrom map[int64]bool
	logger    *slog.Logger
	cancel    context.CancelFunc
}

// NewBot creates a Bot.
func NewBot(pool *spawner.Pool, sender Sender, cfg BotConfig, logger *slog.Logger, cancel context.CancelFunc) *Bot {
	allow := make(map[int64]bool, len(cfg.AllowFrom))
	for _, id := range cfg.AllowFrom {
		allow[id] = true
	}

	return &Bot{
		pool:      pool,
		sender:    sender,
		allowFrom: allow,
		logger:    logger,
		cancel:    cancel,
	}
}

// SetUpdater configures the self-update pipeline.
func (b *Bot) SetUpdater(u *selfupdate.Updater) {
	b.updater = u
}

// HandleText processes an incoming text message from a user.
func (b *Bot) HandleText(ctx context.Context, chatID int64, userID int64, text string) {
	if !IsAuthorized(userID, b.allowFrom) {
		b.logger.Debug("unauthorized user", "user_id", userID)
		return
	}

	text = strings.TrimSpace(text)
	if text == "" {
		return
	}

	// Check for commands.
	if strings.HasPrefix(text, "/") {
		parts := strings.SplitN(text, " ", 2)
		command := parts[0]
		args := ""
		if len(parts) > 1 {
			args = parts[1]
		}
		deps := CommandDeps{
			Pool:       b.pool,
			Sender:     b.sender,
			CancelFunc: b.cancel,
			Updater:    b.updater,
		}
		if handleCommand(ctx, chatID, command, args, deps) {
			return
		}
	}

	// Route to agent via @prefix or default.
	agentName, cleanedMsg := spawner.ParseAgentPrefix(text)
	if agentName == "" {
		agentName = b.pool.DefaultName()
	}
	if cleanedMsg == "" {
		cleanedMsg = text
	}

	handler := NewHandler(chatID, b.sender, b.logger)
	if err := b.pool.SubmitTo(agentName, agent.Request{Message: cleanedMsg, Handler: handler}); err != nil {
		b.logger.Warn("submit failed", "agent", agentName, "error", err)
		b.sender.SendMessage(chatID, fmt.Sprintf("Agent @%s: %v", agentName, err))
	}
}
