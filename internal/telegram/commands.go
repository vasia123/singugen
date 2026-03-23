package telegram

import (
	"context"
	"fmt"

	"github.com/vasis/singugen/internal/agent"
)

// handleCommand processes slash commands. Returns true if handled.
func handleCommand(ctx context.Context, chatID int64, command string, a *agent.Agent, session agent.SessionStarter, sender Sender, cancelFunc context.CancelFunc) bool {
	switch command {
	case "/start":
		sender.SendMessage(chatID, "SinguGen agent ready. Send me a message.")
		return true
	case "/status":
		state := a.State()
		sender.SendMessage(chatID, fmt.Sprintf("Agent state: %s", state))
		return true
	case "/stop":
		sender.SendMessage(chatID, "Shutting down...")
		if cancelFunc != nil {
			cancelFunc()
		}
		return true
	case "/reset":
		if session == nil {
			sender.SendMessage(chatID, "No session to reset.")
			return true
		}
		if err := session.Restart(ctx); err != nil {
			sender.SendMessage(chatID, fmt.Sprintf("Reset failed: %v", err))
		} else {
			sender.SendMessage(chatID, "Session reset.")
		}
		return true
	default:
		return false
	}
}
