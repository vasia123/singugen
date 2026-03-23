package telegram

import (
	"context"
	"fmt"

	"github.com/vasis/singugen/internal/agent"
	"github.com/vasis/singugen/internal/selfupdate"
)

// CommandDeps holds dependencies for command handlers.
type CommandDeps struct {
	Agent      *agent.Agent
	Session    agent.SessionStarter
	Sender     Sender
	CancelFunc context.CancelFunc
	Updater    *selfupdate.Updater // nil if self-update disabled
}

// handleCommand processes slash commands. Returns true if handled.
func handleCommand(ctx context.Context, chatID int64, command string, deps CommandDeps) bool {
	switch command {
	case "/start":
		deps.Sender.SendMessage(chatID, "SinguGen agent ready. Send me a message.")
		return true
	case "/status":
		state := deps.Agent.State()
		deps.Sender.SendMessage(chatID, fmt.Sprintf("Agent state: %s", state))
		return true
	case "/stop":
		deps.Sender.SendMessage(chatID, "Shutting down...")
		if deps.CancelFunc != nil {
			deps.CancelFunc()
		}
		return true
	case "/reset":
		if deps.Session == nil {
			deps.Sender.SendMessage(chatID, "No session to reset.")
			return true
		}
		if err := deps.Session.Restart(ctx); err != nil {
			deps.Sender.SendMessage(chatID, fmt.Sprintf("Reset failed: %v", err))
		} else {
			deps.Sender.SendMessage(chatID, "Session reset.")
		}
		return true
	case "/update":
		if deps.Updater == nil {
			deps.Sender.SendMessage(chatID, "Self-update is disabled.")
			return true
		}
		deps.Sender.SendMessage(chatID, "Running self-update pipeline...")
		result, err := deps.Updater.Apply(ctx)
		if err != nil {
			deps.Sender.SendMessage(chatID, fmt.Sprintf("Update failed: %v", err))
			return true
		}
		if !result.Validated {
			deps.Sender.SendMessage(chatID, fmt.Sprintf("Validation failed:\n%s", result.Output))
			return true
		}
		if !result.Committed {
			deps.Sender.SendMessage(chatID, "No changes to commit.")
			return true
		}
		deps.Sender.SendMessage(chatID, fmt.Sprintf("Update committed: %s\nRestarting...", result.CommitHash))
		if err := selfupdate.SignalParentRestart(); err != nil {
			deps.Sender.SendMessage(chatID, fmt.Sprintf("Restart signal failed: %v", err))
		}
		return true
	case "/rollback":
		if deps.Updater == nil {
			deps.Sender.SendMessage(chatID, "Self-update is disabled.")
			return true
		}
		deps.Sender.SendMessage(chatID, "Rolling back last commit...")
		// Rollback is handled via git revert through the updater's git operator.
		// For now, simple notification — full rollback requires git access.
		deps.Sender.SendMessage(chatID, "Use manual rollback: git revert HEAD && rebuild")
		return true
	default:
		return false
	}
}
