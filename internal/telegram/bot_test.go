package telegram

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/vasis/singugen/internal/claude"
	"github.com/vasis/singugen/internal/comms"
	"github.com/vasis/singugen/internal/spawner"
)

type fakeBotLauncher struct{}

func (fakeBotLauncher) Launch(_ context.Context, _ []string) (io.WriteCloser, io.ReadCloser, func() error, error) {
	stdinR, stdinW := io.Pipe()
	stdoutR, stdoutW := io.Pipe()
	_ = stdinR
	_ = stdoutW
	wait := func() error { select {} }
	return stdinW, stdoutR, wait, nil
}

func newTestPool(t *testing.T, events ...claude.Event) *spawner.Pool {
	t.Helper()
	dir := t.TempDir()
	bus := comms.New()
	pool := spawner.NewPool(fakeBotLauncher{}, bus, dir, "main", discardLogger())

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() {
		cancel()
		pool.ShutdownAll()
	})

	pool.Spawn(ctx, spawner.AgentConfig{Name: "main", Description: "Main agent"})
	time.Sleep(50 * time.Millisecond)
	return pool
}

func TestBot_AuthRejectsUnauthorized(t *testing.T) {
	s := newFakeSender()
	pool := newTestPool(t)

	bot := NewBot(pool, s, BotConfig{AllowFrom: []int64{111}}, discardLogger(), nil)

	bot.HandleText(context.Background(), 100, 999, "hello")

	if len(s.Sent()) != 0 {
		t.Errorf("got %d messages for unauthorized user, want 0", len(s.Sent()))
	}
}

func TestBot_CommandDispatch(t *testing.T) {
	s := newFakeSender()
	pool := newTestPool(t)

	bot := NewBot(pool, s, BotConfig{}, discardLogger(), nil)

	bot.HandleText(context.Background(), 100, 42, "/start")

	if len(s.Sent()) != 1 {
		t.Fatalf("got %d messages, want 1", len(s.Sent()))
	}
}

func TestBot_EmptyMessageIgnored(t *testing.T) {
	s := newFakeSender()
	pool := newTestPool(t)

	bot := NewBot(pool, s, BotConfig{}, discardLogger(), nil)

	bot.HandleText(context.Background(), 100, 42, "  ")

	if len(s.Sent()) != 0 {
		t.Error("empty message should be ignored")
	}
}

func TestBot_RoutesToDefaultAgent(t *testing.T) {
	s := newFakeSender()
	pool := newTestPool(t)

	bot := NewBot(pool, s, BotConfig{}, discardLogger(), nil)

	// Should not error — message submitted to main agent queue.
	bot.HandleText(context.Background(), 100, 42, "hello")

	// No error message sent = routing succeeded.
	sent := s.Sent()
	for _, m := range sent {
		if m.Text != "" && m.Text[0] == 'A' { // "Agent @main: ..."
			t.Errorf("unexpected error message: %s", m.Text)
		}
	}
}

func TestBot_RoutesToNamedAgent(t *testing.T) {
	s := newFakeSender()
	pool := newTestPool(t)

	bot := NewBot(pool, s, BotConfig{}, discardLogger(), nil)

	// Try to route to non-existent agent — should get error message.
	bot.HandleText(context.Background(), 100, 42, "@researcher find something")

	sent := s.Sent()
	if len(sent) != 1 {
		t.Fatalf("got %d messages, want 1 (error)", len(sent))
	}
}
