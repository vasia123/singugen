package spawner

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/vasis/singugen/internal/agent"
	"github.com/vasis/singugen/internal/claude"
	"github.com/vasis/singugen/internal/comms"
)

type nopWriter struct{}

func (nopWriter) Write(p []byte) (int, error) { return len(p), nil }

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(nopWriter{}, nil))
}

// fakeLauncher for pool tests — each Launch creates connected pipes.
type fakeLauncher struct{}

func (fakeLauncher) Launch(_ context.Context, _ []string) (io.WriteCloser, io.ReadCloser, func() error, error) {
	stdinR, stdinW := io.Pipe()
	stdoutR, stdoutW := io.Pipe()
	_ = stdinR
	_ = stdoutW
	wait := func() error {
		select {} // block forever
	}
	return stdinW, stdoutR, wait, nil
}

func newTestPool(t *testing.T) (*Pool, context.CancelFunc) {
	t.Helper()
	dir := t.TempDir()
	bus := comms.New()
	p := NewPool(fakeLauncher{}, bus, dir, "main", testLogger())
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() {
		cancel()
		p.ShutdownAll()
	})

	// Spawn default agent.
	if err := p.Spawn(ctx, AgentConfig{Name: "main", Description: "Main agent"}); err != nil {
		t.Fatalf("spawn main: %v", err)
	}
	time.Sleep(50 * time.Millisecond) // let Run() start

	return p, cancel
}

func TestPool_SpawnAndGet(t *testing.T) {
	p, _ := newTestPool(t)

	a, ok := p.Get("main")
	if !ok || a == nil {
		t.Fatal("main agent not found")
	}
	if a.State() != agent.StateReady {
		t.Errorf("state = %v, want ready", a.State())
	}
}

func TestPool_SpawnDuplicate(t *testing.T) {
	p, _ := newTestPool(t)
	ctx := context.Background()

	err := p.Spawn(ctx, AgentConfig{Name: "main"})
	if err == nil {
		t.Error("duplicate spawn should fail")
	}
}

func TestPool_Stop(t *testing.T) {
	p, _ := newTestPool(t)
	ctx := context.Background()

	p.Spawn(ctx, AgentConfig{Name: "researcher", Description: "Research"})
	time.Sleep(50 * time.Millisecond)

	if err := p.Stop("researcher"); err != nil {
		t.Fatalf("Stop() error: %v", err)
	}

	_, ok := p.Get("researcher")
	if ok {
		t.Error("agent should be removed after stop")
	}
}

func TestPool_StopUnknown(t *testing.T) {
	p, _ := newTestPool(t)

	err := p.Stop("nonexistent")
	if err == nil {
		t.Error("stopping unknown agent should fail")
	}
}

func TestPool_List(t *testing.T) {
	p, _ := newTestPool(t)
	ctx := context.Background()

	p.Spawn(ctx, AgentConfig{Name: "researcher", Description: "Research"})
	time.Sleep(50 * time.Millisecond)

	list := p.List()
	if len(list) != 2 {
		t.Fatalf("got %d agents, want 2", len(list))
	}
}

func TestPool_SubmitTo(t *testing.T) {
	p, _ := newTestPool(t)

	// Submit should not error (agent exists and queue not full).
	handler := &noopHandler{}
	err := p.SubmitTo("main", agent.Request{Message: "hello", Handler: handler})
	if err != nil {
		t.Fatalf("SubmitTo() error: %v", err)
	}
}

func TestPool_SubmitToUnknown(t *testing.T) {
	p, _ := newTestPool(t)

	err := p.SubmitTo("nonexistent", agent.Request{Message: "hello", Handler: &noopHandler{}})
	if err == nil {
		t.Error("submit to unknown should fail")
	}
}

func TestPool_Default(t *testing.T) {
	p, _ := newTestPool(t)

	a, ok := p.Default()
	if !ok || a == nil {
		t.Fatal("default agent not found")
	}
}

func TestPool_ShutdownAll(t *testing.T) {
	dir := t.TempDir()
	bus := comms.New()
	p := NewPool(fakeLauncher{}, bus, dir, "main", testLogger())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	p.Spawn(ctx, AgentConfig{Name: "main"})
	p.Spawn(ctx, AgentConfig{Name: "researcher"})
	time.Sleep(50 * time.Millisecond)

	p.ShutdownAll()

	if len(p.List()) != 0 {
		t.Error("all agents should be removed after shutdown")
	}
}

type noopHandler struct{}

func (noopHandler) OnEvent(_ claude.Event)       {}
func (noopHandler) OnComplete(_ string, _ error) {}
