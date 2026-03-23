package telegram

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/vasis/singugen/internal/agent"
)

// BotConfig holds bot configuration.
type BotConfig struct {
	AllowFrom []int64
}

// Bot bridges Telegram updates to the agent.
type Bot struct {
	agent     *agent.Agent
	session   agent.SessionStarter
	sender    Sender
	allowFrom map[int64]bool
	logger    *slog.Logger
	cancel    context.CancelFunc
}

// NewBot creates a Bot.
func NewBot(a *agent.Agent, session agent.SessionStarter, sender Sender, cfg BotConfig, logger *slog.Logger, cancel context.CancelFunc) *Bot {
	allow := make(map[int64]bool, len(cfg.AllowFrom))
	for _, id := range cfg.AllowFrom {
		allow[id] = true
	}

	return &Bot{
		agent:     a,
		session:   session,
		sender:    sender,
		allowFrom: allow,
		logger:    logger,
		cancel:    cancel,
	}
}

// HandleText processes an incoming text message from a user.
// This is the entry point called by the Telegram update handler.
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
		command := strings.SplitN(text, " ", 2)[0]
		if handleCommand(ctx, chatID, command, b.agent, b.session, b.sender, b.cancel) {
			return
		}
	}

	// Submit to agent.
	handler := NewHandler(chatID, b.sender, b.logger)
	if err := b.agent.Submit(agent.Request{Message: text, Handler: handler}); err != nil {
		b.logger.Warn("queue full", "error", err)
		b.sender.SendMessage(chatID, fmt.Sprintf("Busy, try again later: %v", err))
	}
}
