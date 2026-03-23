package supervisor

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/exec"
	"syscall"
	"time"
)

var ErrCircuitBreakerOpen = errors.New("supervisor: circuit breaker open, too many restarts")

// ChildRunner abstracts how a child process is launched.
type ChildRunner interface {
	Start(ctx context.Context) error
}

// Supervisor manages the lifecycle of a child process, restarting it
// on exit and tripping the circuit breaker on crash loops.
type Supervisor struct {
	runner         ChildRunner
	breaker        *CircuitBreaker
	logger         *slog.Logger
	restartCh      chan struct{}
	restartBackoff time.Duration
}

func New(runner ChildRunner, breaker *CircuitBreaker, logger *slog.Logger) *Supervisor {
	return &Supervisor{
		runner:         runner,
		breaker:        breaker,
		logger:         logger,
		restartCh:      make(chan struct{}, 1),
		restartBackoff: 1 * time.Second,
	}
}

// RestartChild signals the supervision loop to kill and restart the child.
func (s *Supervisor) RestartChild() {
	select {
	case s.restartCh <- struct{}{}:
	default:
	}
}

// Run starts the supervision loop. It blocks until ctx is cancelled
// or the circuit breaker trips.
func (s *Supervisor) Run(ctx context.Context) error {
	for {
		if ctx.Err() != nil {
			return nil
		}
		if s.breaker.IsOpen() {
			s.logger.Error("circuit breaker open")
			return ErrCircuitBreakerOpen
		}

		childCtx, childCancel := context.WithCancel(ctx)

		go func() {
			select {
			case <-s.restartCh:
				s.logger.Info("restart requested")
				childCancel()
			case <-childCtx.Done():
			}
		}()

		s.logger.Info("starting child")
		err := s.runner.Start(childCtx)
		childCancel()

		if ctx.Err() != nil {
			return nil
		}

		s.breaker.Record()
		if err != nil {
			s.logger.Error("child exited with error", "error", err)
		} else {
			s.logger.Info("child exited cleanly")
		}

		select {
		case <-ctx.Done():
			return nil
		case <-time.After(s.restartBackoff):
		}
	}
}

// ExecRunner launches a child process via os/exec.
type ExecRunner struct {
	Binary string
	Args   []string
}

func NewExecRunner(binary string, args ...string) *ExecRunner {
	return &ExecRunner{Binary: binary, Args: args}
}

func (r *ExecRunner) Start(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, r.Binary, r.Args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Send SIGTERM instead of SIGKILL on context cancel.
	cmd.Cancel = func() error {
		return cmd.Process.Signal(syscall.SIGTERM)
	}
	cmd.WaitDelay = 5 * time.Second

	return cmd.Run()
}
