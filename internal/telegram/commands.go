package telegram

import (
	"context"
	"fmt"
	"strings"

	"github.com/vasis/singugen/internal/agent"
	"github.com/vasis/singugen/internal/selfupdate"
	"github.com/vasis/singugen/internal/spawner"
)

// CommandDeps holds dependencies for command handlers.
type CommandDeps struct {
	Agent      *agent.Agent
	Session    agent.SessionStarter
	Pool       *spawner.Pool
	Sender     Sender
	CancelFunc context.CancelFunc
	Updater    *selfupdate.Updater
}

// handleCommand processes slash commands. Returns true if handled.
func handleCommand(ctx context.Context, chatID int64, command, args string, deps CommandDeps) bool {
	switch command {
	case "/start":
		deps.Sender.SendMessage(chatID, "SinguGen agent ready. Send me a message.")
		return true
	case "/status":
		if deps.Pool != nil {
			var sb strings.Builder
			for _, cfg := range deps.Pool.List() {
				a, ok := deps.Pool.Get(cfg.Name)
				state := "unknown"
				if ok {
					state = a.State().String()
				}
				fmt.Fprintf(&sb, "%s: %s (%s)\n", cfg.Name, state, cfg.Description)
			}
			deps.Sender.SendMessage(chatID, sb.String())
		} else if deps.Agent != nil {
			deps.Sender.SendMessage(chatID, fmt.Sprintf("Agent state: %s", deps.Agent.State()))
		}
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
	case "/hire":
		if deps.Pool == nil {
			deps.Sender.SendMessage(chatID, "Multi-agent not available.")
			return true
		}
		name, desc := parseHireArgs(args)
		if name == "" {
			deps.Sender.SendMessage(chatID, "Usage: /hire <name> <description>")
			return true
		}
		if err := deps.Pool.Spawn(ctx, spawner.AgentConfig{Name: name, Description: desc}); err != nil {
			deps.Sender.SendMessage(chatID, fmt.Sprintf("Hire failed: %v", err))
		} else {
			deps.Sender.SendMessage(chatID, fmt.Sprintf("Agent @%s hired: %s", name, desc))
		}
		return true
	case "/fire":
		if deps.Pool == nil {
			deps.Sender.SendMessage(chatID, "Multi-agent not available.")
			return true
		}
		name := strings.TrimSpace(strings.TrimPrefix(args, "@"))
		if name == "" {
			deps.Sender.SendMessage(chatID, "Usage: /fire <name>")
			return true
		}
		if name == deps.Pool.DefaultName() {
			deps.Sender.SendMessage(chatID, "Cannot fire the main agent.")
			return true
		}
		if err := deps.Pool.Stop(name); err != nil {
			deps.Sender.SendMessage(chatID, fmt.Sprintf("Fire failed: %v", err))
		} else {
			deps.Sender.SendMessage(chatID, fmt.Sprintf("Agent @%s fired.", name))
		}
		return true
	case "/agents":
		if deps.Pool == nil {
			deps.Sender.SendMessage(chatID, "Multi-agent not available.")
			return true
		}
		agents := deps.Pool.List()
		if len(agents) == 0 {
			deps.Sender.SendMessage(chatID, "No agents running.")
			return true
		}
		var sb strings.Builder
		for _, cfg := range agents {
			a, ok := deps.Pool.Get(cfg.Name)
			state := "unknown"
			if ok {
				state = a.State().String()
			}
			fmt.Fprintf(&sb, "@%s [%s] — %s\n", cfg.Name, state, cfg.Description)
		}
		deps.Sender.SendMessage(chatID, sb.String())
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
		deps.Sender.SendMessage(chatID, "Use manual rollback: git revert HEAD && rebuild")
		return true
	default:
		return false
	}
}

func parseHireArgs(args string) (name, description string) {
	args = strings.TrimSpace(args)
	if args == "" {
		return "", ""
	}
	parts := strings.SplitN(args, " ", 2)
	name = strings.TrimPrefix(parts[0], "@")
	if len(parts) > 1 {
		description = parts[1]
	}
	return name, description
}
