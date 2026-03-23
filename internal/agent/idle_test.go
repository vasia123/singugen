package agent

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/vasis/singugen/internal/claude"
)

func TestAgent_IdleTimerFires(t *testing.T) {
	var idleCalled atomic.Int32
	sess := newFakeSession(
		claude.Event{Type: claude.EventResult, Subtype: string(claude.ResultSuccess), Result: "ok"},
	)

	a := New(Config{
		QueueSize:   8,
		IdleTimeout: 100 * time.Millisecond,
		OnIdle: func(ctx context.Context) {
			idleCalled.Add(1)
		},
	}, sess, discardLogger())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go a.Run(ctx)

	// Wait for idle to fire.
	time.Sleep(250 * time.Millisecond)

	if got := idleCalled.Load(); got == 0 {
		t.Error("OnIdle was not called")
	}

	// After idle, state should be Ready again.
	if a.State() != StateReady {
		t.Errorf("state after idle = %v, want ready", a.State())
	}
}

func TestAgent_IdleTimerResetsOnMessage(t *testing.T) {
	var idleCalled atomic.Int32
	sess := newFakeSession(
		claude.Event{Type: claude.EventResult, Subtype: string(claude.ResultSuccess), Result: "ok"},
	)

	a := New(Config{
		QueueSize:   8,
		IdleTimeout: 150 * time.Millisecond,
		OnIdle: func(ctx context.Context) {
			idleCalled.Add(1)
		},
	}, sess, discardLogger())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go a.Run(ctx)

	// Submit messages every 100ms, keeping idle timer from firing (150ms).
	for range 3 {
		time.Sleep(100 * time.Millisecond)
		h := newRecordingHandler()
		a.Submit(Request{Message: "keepalive", Handler: h})
		h.waitDone(t)
	}

	if got := idleCalled.Load(); got != 0 {
		t.Errorf("OnIdle called %d times, want 0 (messages kept resetting)", got)
	}
}

func TestAgent_NoIdleDuringProcessing(t *testing.T) {
	var idleCalled atomic.Int32

	// Session that takes 200ms to process.
	slowSess := &slowFakeSession{
		delay:  200 * time.Millisecond,
		events: []claude.Event{{Type: claude.EventResult, Subtype: string(claude.ResultSuccess), Result: "ok"}},
	}

	a := New(Config{
		QueueSize:   8,
		IdleTimeout: 50 * time.Millisecond,
		OnIdle: func(ctx context.Context) {
			idleCalled.Add(1)
		},
	}, slowSess, discardLogger())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go a.Run(ctx)

	time.Sleep(20 * time.Millisecond)

	// Submit a message that takes 200ms to process.
	h := newRecordingHandler()
	a.Submit(Request{Message: "slow", Handler: h})
	h.waitDone(t)

	// Idle should NOT have fired during the 200ms processing
	// (even though idle timeout is 50ms) because timer was stopped.
	if got := idleCalled.Load(); got != 0 {
		t.Errorf("OnIdle called %d times during processing, want 0", got)
	}
}

func TestAgent_NoIdleWithoutConfig(t *testing.T) {
	sess := newFakeSession(
		claude.Event{Type: claude.EventResult, Subtype: string(claude.ResultSuccess), Result: "ok"},
	)
	// No IdleTimeout, no OnIdle — should not panic.
	a := New(Config{QueueSize: 8}, sess, discardLogger())

	ctx, cancel := context.WithCancel(context.Background())
	go a.Run(ctx)
	time.Sleep(50 * time.Millisecond)
	cancel()

	time.Sleep(50 * time.Millisecond)
	if a.State() != StateStopped {
		t.Errorf("state = %v, want stopped", a.State())
	}
}

func TestAgent_ShutdownHookCalled(t *testing.T) {
	var shutdownCalled atomic.Int32
	sess := newFakeSession()

	a := New(Config{
		QueueSize: 8,
		OnShutdown: func(ctx context.Context) {
			shutdownCalled.Add(1)
		},
	}, sess, discardLogger())

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- a.Run(ctx) }()

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return")
	}

	if got := shutdownCalled.Load(); got != 1 {
		t.Errorf("OnShutdown called %d times, want 1", got)
	}
}

func TestAgent_MultipleIdleCycles(t *testing.T) {
	var idleCalled atomic.Int32
	sess := newFakeSession(
		claude.Event{Type: claude.EventResult, Subtype: string(claude.ResultSuccess), Result: "ok"},
	)

	a := New(Config{
		QueueSize:   8,
		IdleTimeout: 50 * time.Millisecond,
		OnIdle: func(ctx context.Context) {
			idleCalled.Add(1)
		},
	}, sess, discardLogger())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go a.Run(ctx)

	// Wait for at least 2 idle cycles.
	time.Sleep(150 * time.Millisecond)

	if got := idleCalled.Load(); got < 2 {
		t.Errorf("OnIdle called %d times, want at least 2", got)
	}
}

// slowFakeSession adds a delay before returning events.
type slowFakeSession struct {
	delay  time.Duration
	events []claude.Event
}

func (s *slowFakeSession) Start(_ context.Context) error { return nil }
func (s *slowFakeSession) Send(ctx context.Context, _ string) (<-chan claude.Event, error) {
	ch := make(chan claude.Event)
	go func() {
		defer close(ch)
		select {
		case <-time.After(s.delay):
		case <-ctx.Done():
			return
		}
		for _, e := range s.events {
			ch <- e
		}
	}()
	return ch, nil
}
func (s *slowFakeSession) SessionID() string            { return "" }
func (s *slowFakeSession) Close() error                 { return nil }
func (s *slowFakeSession) Restart(_ context.Context) error { return nil }
