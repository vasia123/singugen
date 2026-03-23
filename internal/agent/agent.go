package agent

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"

	"github.com/vasis/singugen/internal/claude"
)

// SessionStarter is the interface the agent uses to talk to Claude.
type SessionStarter interface {
	Start(ctx context.Context) error
	Send(ctx context.Context, message string) (<-chan claude.Event, error)
	SessionID() string
	Close() error
	Restart(ctx context.Context) error
}

// MessageHandler receives streaming events from the agent.
type MessageHandler interface {
	OnEvent(event claude.Event)
	OnComplete(result string, err error)
}

// Request is a message submitted to the agent for processing.
type Request struct {
	Message string
	Handler MessageHandler
}

// Config configures the agent.
type Config struct {
	QueueSize int
}

func (c *Config) setDefaults() {
	if c.QueueSize == 0 {
		c.QueueSize = 64
	}
}

// Agent processes user messages through a Claude session.
type Agent struct {
	cfg     Config
	session SessionStarter
	state   atomic.Int32
	queue   chan Request
	logger  *slog.Logger
}

// New creates an Agent. Call Run to start processing.
func New(cfg Config, session SessionStarter, logger *slog.Logger) *Agent {
	cfg.setDefaults()
	a := &Agent{
		cfg:     cfg,
		session: session,
		queue:   make(chan Request, cfg.QueueSize),
		logger:  logger,
	}
	a.state.Store(int32(StateStarting))
	return a
}

// State returns the current agent state.
func (a *Agent) State() State {
	return State(a.state.Load())
}

// Submit enqueues a message for processing. Returns error if queue is full.
func (a *Agent) Submit(req Request) error {
	select {
	case a.queue <- req:
		return nil
	default:
		return fmt.Errorf("agent: queue full (capacity %d)", a.cfg.QueueSize)
	}
}

// Run starts the agent processing loop. Blocks until ctx is cancelled.
// Drains remaining queue items before returning.
func (a *Agent) Run(ctx context.Context) error {
	a.state.Store(int32(StateReady))
	a.logger.Info("agent ready")

	for {
		select {
		case req := <-a.queue:
			a.processRequest(ctx, req)
		case <-ctx.Done():
			a.drain(ctx)
			a.state.Store(int32(StateStopped))
			a.logger.Info("agent stopped")
			return nil
		}
	}
}

func (a *Agent) processRequest(ctx context.Context, req Request) {
	a.state.Store(int32(StateProcessing))
	a.logger.Info("processing message", "length", len(req.Message))

	ch, err := a.session.Send(ctx, req.Message)
	if err != nil {
		a.logger.Error("session send failed", "error", err)
		req.Handler.OnComplete("", err)
		a.state.Store(int32(StateReady))
		return
	}

	var result string
	var resultErr error

	for event := range ch {
		req.Handler.OnEvent(event)

		if event.Type == claude.EventResult {
			if event.Subtype == string(claude.ResultError) {
				resultErr = fmt.Errorf("claude: %s", event.Error)
			} else {
				result = event.Result
			}
		}
	}

	req.Handler.OnComplete(result, resultErr)
	a.state.Store(int32(StateReady))
}

// drain processes remaining items in the queue without blocking.
func (a *Agent) drain(ctx context.Context) {
	for {
		select {
		case req := <-a.queue:
			a.processRequest(ctx, req)
		default:
			return
		}
	}
}
