package telegram

import (
	"context"
	"testing"
	"time"

	"github.com/vasis/singugen/internal/agent"
	"github.com/vasis/singugen/internal/claude"
)

func newTestAgent(events ...claude.Event) *agent.Agent {
	sess := &fakeAgentSession{events: events}
	return agent.New(agent.Config{QueueSize: 8}, sess, discardLogger())
}

type fakeAgentSession struct {
	events []claude.Event
}

func (f *fakeAgentSession) Start(_ context.Context) error { return nil }
func (f *fakeAgentSession) Send(_ context.Context, _ string) (<-chan claude.Event, error) {
	ch := make(chan claude.Event, len(f.events))
	for _, e := range f.events {
		ch <- e
	}
	close(ch)
	return ch, nil
}
func (f *fakeAgentSession) SessionID() string            { return "" }
func (f *fakeAgentSession) Close() error                 { return nil }
func (f *fakeAgentSession) Restart(_ context.Context) error { return nil }

func TestBot_AuthRejectsUnauthorized(t *testing.T) {
	s := newFakeSender()
	a := newTestAgent()
	go a.Run(context.Background())

	bot := NewBot(a, nil, s, BotConfig{AllowFrom: []int64{111}}, discardLogger(), nil)

	bot.HandleText(context.Background(), 100, 999, "hello")

	sent := s.Sent()
	if len(sent) != 0 {
		t.Errorf("got %d messages for unauthorized user, want 0", len(sent))
	}
}

func TestBot_TextDispatchToAgent(t *testing.T) {
	s := newFakeSender()
	a := newTestAgent(
		claude.Event{Type: claude.EventResult, Subtype: string(claude.ResultSuccess), Result: "response"},
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go a.Run(ctx)
	time.Sleep(50 * time.Millisecond)

	bot := NewBot(a, nil, s, BotConfig{}, discardLogger(), nil)

	bot.HandleText(ctx, 100, 42, "hello")

	// Wait for agent to process.
	time.Sleep(100 * time.Millisecond)

	sent := s.Sent()
	// Should have: status message + result message.
	if len(sent) < 2 {
		t.Fatalf("got %d messages, want at least 2 (status + result)", len(sent))
	}

	// Last sent should be the result.
	lastSent := sent[len(sent)-1]
	if lastSent.Text != "response" {
		t.Errorf("last message = %q, want response", lastSent.Text)
	}
}

func TestBot_QueueFullFeedback(t *testing.T) {
	s := newFakeSender()
	// Agent with queue size 1, pre-fill to make it full.
	// Use a session that blocks so the queue item isn't consumed.
	blockSess := &fakeAgentSession{events: []claude.Event{}}
	a := agent.New(agent.Config{QueueSize: 1}, blockSess, discardLogger())

	// Fill the queue without starting Run (so nothing is consumed).
	a.Submit(agent.Request{Message: "blocker", Handler: &noopHandler{}})

	bot := NewBot(a, nil, s, BotConfig{}, discardLogger(), nil)

	bot.HandleText(context.Background(), 100, 42, "hello")

	sent := s.Sent()
	if len(sent) != 1 {
		t.Fatalf("got %d messages, want 1 (busy feedback)", len(sent))
	}
}

func TestBot_CommandDispatch(t *testing.T) {
	s := newFakeSender()
	a := newTestAgent()

	bot := NewBot(a, nil, s, BotConfig{}, discardLogger(), nil)

	bot.HandleText(context.Background(), 100, 42, "/start")

	sent := s.Sent()
	if len(sent) != 1 {
		t.Fatalf("got %d messages, want 1", len(sent))
	}
}

func TestBot_EmptyMessageIgnored(t *testing.T) {
	s := newFakeSender()
	a := newTestAgent()

	bot := NewBot(a, nil, s, BotConfig{}, discardLogger(), nil)

	bot.HandleText(context.Background(), 100, 42, "  ")

	if len(s.Sent()) != 0 {
		t.Error("empty message should be ignored")
	}
}
