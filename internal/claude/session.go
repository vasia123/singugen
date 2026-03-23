package claude

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"
)

// SessionConfig configures a Claude Code session.
type SessionConfig struct {
	Model        string
	SystemPrompt string
	Timeout      time.Duration // watchdog per-interaction, default 3min
	MaxRetries   int           // auto-restart attempts, default 10
}

func (c *SessionConfig) setDefaults() {
	if c.Timeout == 0 {
		c.Timeout = 3 * time.Minute
	}
	if c.MaxRetries == 0 {
		c.MaxRetries = 10
	}
}

// Session manages a running Claude Code process.
type Session struct {
	cfg      SessionConfig
	launcher ProcessLauncher
	logger   *slog.Logger

	mu        sync.Mutex
	sessionID string
	stdin     io.WriteCloser
	stdout    io.ReadCloser
	wait      func() error
	writer    *Writer
	cancel    context.CancelFunc
	running   bool
}

// NewSession creates a new Session. Call Start before Send.
func NewSession(cfg SessionConfig, launcher ProcessLauncher, logger *slog.Logger) *Session {
	cfg.setDefaults()
	return &Session{
		cfg:      cfg,
		launcher: launcher,
		logger:   logger,
	}
}

// Start launches the claude process. Must be called before Send.
func (s *Session) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return nil
	}

	args := s.buildArgs()

	childCtx, cancel := context.WithCancel(ctx)
	stdin, stdout, wait, err := s.launcher.Launch(childCtx, args)
	if err != nil {
		cancel()
		return fmt.Errorf("claude: launch: %w", err)
	}

	s.stdin = stdin
	s.stdout = stdout
	s.wait = wait
	s.writer = NewWriter(stdin)
	s.cancel = cancel
	s.running = true

	return nil
}

// Send writes a user message and returns a channel that yields
// all output events until a result event is received.
func (s *Session) Send(ctx context.Context, message string) (<-chan Event, error) {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return nil, fmt.Errorf("claude: session not started")
	}
	writer := s.writer
	stdout := s.stdout
	timeout := s.cfg.Timeout
	s.mu.Unlock()

	msg := NewUserInput(message)
	if err := writer.WriteMessage(msg); err != nil {
		return nil, fmt.Errorf("claude: write message: %w", err)
	}

	out := make(chan Event)
	events := ReadEvents(ctx, stdout)

	go func() {
		defer close(out)

		timer := time.NewTimer(timeout)
		defer timer.Stop()

		for {
			select {
			case event, ok := <-events:
				if !ok {
					return
				}

				timer.Reset(timeout)

				// Track session ID from init event.
				if event.Type == EventSystem && event.Subtype == "init" {
					s.mu.Lock()
					s.sessionID = event.SessionID
					s.mu.Unlock()
				}

				select {
				case out <- event:
				case <-ctx.Done():
					return
				}

				// Result event signals end of this interaction.
				if event.Type == EventResult {
					// Update session ID from result.
					if event.SessionID != "" {
						s.mu.Lock()
						s.sessionID = event.SessionID
						s.mu.Unlock()
					}
					return
				}

			case <-timer.C:
				s.logger.Warn("claude: watchdog timeout", "timeout", timeout)
				out <- Event{
					Type:    EventResult,
					Subtype: string(ResultError),
					Error:   "watchdog timeout: no response from claude",
				}
				return

			case <-ctx.Done():
				return
			}
		}
	}()

	return out, nil
}

// SessionID returns the session ID from Claude, or empty if unknown.
func (s *Session) SessionID() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sessionID
}

// Restart stops the current process and starts a new one,
// resuming the session if a session ID is known.
func (s *Session) Restart(ctx context.Context) error {
	if err := s.Close(); err != nil {
		return err
	}
	return s.Start(ctx)
}

// Close terminates the claude process gracefully.
func (s *Session) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	s.running = false

	if s.cancel != nil {
		s.cancel()
	}
	if s.stdin != nil {
		s.stdin.Close()
	}

	return nil
}

func (s *Session) buildArgs() []string {
	args := []string{
		"-p",
		"--input-format", "stream-json",
		"--output-format", "stream-json",
		"--verbose",
		"--dangerously-skip-permissions",
	}

	if s.cfg.Model != "" {
		args = append(args, "--model", s.cfg.Model)
	}
	if s.cfg.SystemPrompt != "" {
		args = append(args, "--system-prompt", s.cfg.SystemPrompt)
	}
	if s.sessionID != "" {
		args = append(args, "--resume", s.sessionID)
	}

	return args
}
