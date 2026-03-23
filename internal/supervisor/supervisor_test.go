package supervisor

import (
	"context"
	"errors"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"
)

type fakeRunner struct {
	calls  atomic.Int32
	startF func(ctx context.Context) error
}

func (f *fakeRunner) Start(ctx context.Context) error {
	f.calls.Add(1)
	return f.startF(ctx)
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(nopWriter{}, nil))
}

type nopWriter struct{}

func (nopWriter) Write(p []byte) (int, error) { return len(p), nil }

func newTestSupervisor(runner ChildRunner, breaker *CircuitBreaker) *Supervisor {
	sup := New(runner, breaker, discardLogger())
	sup.restartBackoff = 0
	return sup
}

func TestSupervisor_RestartsOnChildExit(t *testing.T) {
	runner := &fakeRunner{
		startF: func(ctx context.Context) error {
			return errors.New("crash")
		},
	}
	breaker := NewCircuitBreaker(5, 2*time.Minute)
	sup := newTestSupervisor(runner, breaker)

	err := sup.Run(context.Background())
	if !errors.Is(err, ErrCircuitBreakerOpen) {
		t.Fatalf("Run() = %v, want ErrCircuitBreakerOpen", err)
	}

	got := int(runner.calls.Load())
	if got != 5 {
		t.Errorf("child started %d times, want 5", got)
	}
}

func TestSupervisor_StopsOnContextCancel(t *testing.T) {
	runner := &fakeRunner{
		startF: func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		},
	}
	breaker := NewCircuitBreaker(5, 2*time.Minute)
	sup := newTestSupervisor(runner, breaker)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- sup.Run(ctx)
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run() = %v, want nil", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run() did not return after context cancel")
	}
}

func TestSupervisor_RestartChild(t *testing.T) {
	started := make(chan struct{}, 10)
	runner := &fakeRunner{
		startF: func(ctx context.Context) error {
			started <- struct{}{}
			<-ctx.Done()
			return ctx.Err()
		},
	}
	breaker := NewCircuitBreaker(10, 2*time.Minute)
	sup := newTestSupervisor(runner, breaker)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- sup.Run(ctx)
	}()

	// Wait for first start.
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("child did not start")
	}

	if got := int(runner.calls.Load()); got != 1 {
		t.Fatalf("calls = %d, want 1", got)
	}

	// Trigger restart and wait for second start.
	sup.RestartChild()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("child did not restart")
	}

	if got := int(runner.calls.Load()); got != 2 {
		t.Errorf("calls = %d after RestartChild, want 2", got)
	}

	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Run() did not return")
	}
}

func TestSupervisor_CircuitBreakerTrips(t *testing.T) {
	runner := &fakeRunner{
		startF: func(ctx context.Context) error {
			return errors.New("crash")
		},
	}
	breaker := NewCircuitBreaker(3, 2*time.Minute)
	sup := newTestSupervisor(runner, breaker)

	err := sup.Run(context.Background())
	if !errors.Is(err, ErrCircuitBreakerOpen) {
		t.Fatalf("Run() = %v, want ErrCircuitBreakerOpen", err)
	}
}

func TestSupervisor_CleanExitStillCounts(t *testing.T) {
	runner := &fakeRunner{
		startF: func(ctx context.Context) error {
			return nil
		},
	}
	breaker := NewCircuitBreaker(3, 2*time.Minute)
	sup := newTestSupervisor(runner, breaker)

	err := sup.Run(context.Background())
	if !errors.Is(err, ErrCircuitBreakerOpen) {
		t.Fatalf("Run() = %v, want ErrCircuitBreakerOpen (clean exits count)", err)
	}
}
