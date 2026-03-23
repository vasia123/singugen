package telegram

import (
	"context"
	"testing"

	"github.com/vasis/singugen/internal/agent"
	"github.com/vasis/singugen/internal/claude"
)

type fakeSession struct{}

func (f *fakeSession) Start(_ context.Context) error                              { return nil }
func (f *fakeSession) Send(_ context.Context, _ string) (<-chan claude.Event, error) { return nil, nil }
func (f *fakeSession) SessionID() string                                           { return "" }
func (f *fakeSession) Close() error                                                { return nil }
func (f *fakeSession) Restart(_ context.Context) error                             { return nil }

func TestCommand_Start(t *testing.T) {
	s := newFakeSender()
	a := agent.New(agent.Config{QueueSize: 1}, &fakeSession{}, discardLogger())

	handled := handleCommand(context.Background(), 123, "/start", a, &fakeSession{}, s, nil)
	if !handled {
		t.Fatal("/start not handled")
	}

	sent := s.Sent()
	if len(sent) != 1 {
		t.Fatalf("got %d messages, want 1", len(sent))
	}
	if sent[0].ChatID != 123 {
		t.Errorf("chatID = %d, want 123", sent[0].ChatID)
	}
}

func TestCommand_Status(t *testing.T) {
	s := newFakeSender()
	a := agent.New(agent.Config{QueueSize: 1}, &fakeSession{}, discardLogger())

	handleCommand(context.Background(), 123, "/status", a, nil, s, nil)

	sent := s.Sent()
	if len(sent) != 1 {
		t.Fatalf("got %d messages, want 1", len(sent))
	}
}

func TestCommand_Stop(t *testing.T) {
	s := newFakeSender()
	a := agent.New(agent.Config{QueueSize: 1}, &fakeSession{}, discardLogger())
	cancelled := false
	cancel := func() { cancelled = true }

	handleCommand(context.Background(), 123, "/stop", a, nil, s, cancel)

	if !cancelled {
		t.Error("cancel was not called")
	}
	sent := s.Sent()
	if len(sent) != 1 {
		t.Fatalf("got %d messages, want 1", len(sent))
	}
}

func TestCommand_Reset(t *testing.T) {
	s := newFakeSender()
	a := agent.New(agent.Config{QueueSize: 1}, &fakeSession{}, discardLogger())

	handleCommand(context.Background(), 123, "/reset", a, &fakeSession{}, s, nil)

	sent := s.Sent()
	if len(sent) != 1 {
		t.Fatalf("got %d messages, want 1", len(sent))
	}
	if sent[0].Text != "Session reset." {
		t.Errorf("text = %q, want Session reset.", sent[0].Text)
	}
}

func TestCommand_Unknown(t *testing.T) {
	s := newFakeSender()
	a := agent.New(agent.Config{QueueSize: 1}, &fakeSession{}, discardLogger())

	handled := handleCommand(context.Background(), 123, "/unknown", a, nil, s, nil)
	if handled {
		t.Error("unknown command should not be handled")
	}
}
