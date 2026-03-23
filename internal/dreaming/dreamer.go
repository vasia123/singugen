package dreaming

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/vasis/singugen/internal/claude"
	"github.com/vasis/singugen/internal/memory"
)

// SessionSender sends messages to Claude.
type SessionSender interface {
	Send(ctx context.Context, message string) (<-chan claude.Event, error)
}

// Dreamer orchestrates memory reorganization through Claude.
type Dreamer struct {
	store   *memory.Store
	session SessionSender
	logger  *slog.Logger
}

// New creates a Dreamer.
func New(store *memory.Store, session SessionSender, logger *slog.Logger) *Dreamer {
	return &Dreamer{
		store:   store,
		session: session,
		logger:  logger,
	}
}

// Dream executes one dreaming cycle: loads memory, asks Claude to
// reorganize, parses response, applies updates.
func (d *Dreamer) Dream(ctx context.Context) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	entries, err := d.store.LoadAll()
	if err != nil {
		return fmt.Errorf("dreaming: load memory: %w", err)
	}

	prompt := buildDreamPrompt(entries)

	d.logger.Info("dreaming: sending reorganization request", "entries", len(entries))

	ch, err := d.session.Send(ctx, prompt)
	if err != nil {
		return fmt.Errorf("dreaming: send: %w", err)
	}

	var result string
	for event := range ch {
		if event.Type == claude.EventResult {
			if event.Subtype == string(claude.ResultError) {
				return fmt.Errorf("dreaming: claude error: %s", event.Error)
			}
			result = event.Result
		}
	}

	if result == "" {
		return fmt.Errorf("dreaming: empty result from claude")
	}

	update, err := ParseDreamResponse(result)
	if err != nil {
		return fmt.Errorf("dreaming: parse response: %w", err)
	}

	if !update.Changed {
		d.logger.Info("dreaming: no changes needed")
		return nil
	}

	for name, content := range update.Files {
		if err := d.store.Save(name, content); err != nil {
			d.logger.Error("dreaming: failed to save", "file", name, "error", err)
		}
	}

	d.logger.Info("dreaming: updated memory", "files", len(update.Files))
	return nil
}

func buildDreamPrompt(entries []memory.Entry) string {
	var sb strings.Builder

	sb.WriteString("You are entering a dreaming phase. Your task is to review and reorganize your memory files.\n\n")
	sb.WriteString("Current memory contents:\n---\n")

	for _, e := range entries {
		content := strings.TrimSpace(e.Content)
		if content == "" {
			continue
		}
		fmt.Fprintf(&sb, "## %s.md\n%s\n\n", e.Name, content)
	}

	sb.WriteString("---\n\n")
	sb.WriteString("Instructions:\n")
	sb.WriteString("1. Review all memory files for accuracy, relevance, and organization\n")
	sb.WriteString("2. Remove outdated or redundant information\n")
	sb.WriteString("3. Restructure entries for clarity\n\n")
	sb.WriteString("Respond with ONLY the updated memory files in this exact format:\n")
	sb.WriteString("<<<MEMORY_UPDATE>>>\n")
	sb.WriteString("<<<FILE:user.md>>>\n")
	sb.WriteString("<updated contents>\n")
	sb.WriteString("<<<FILE:projects.md>>>\n")
	sb.WriteString("<updated contents>\n")
	sb.WriteString("<<<END_MEMORY_UPDATE>>>\n\n")
	sb.WriteString("If no changes are needed, respond with:\n")
	sb.WriteString("<<<NO_CHANGES>>>")

	return sb.String()
}
