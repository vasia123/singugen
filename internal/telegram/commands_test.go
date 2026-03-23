package telegram

import (
	"context"
	"testing"

	"github.com/vasis/singugen/internal/agent"
	"github.com/vasis/singugen/internal/claude"
)

type fakeSession struct{}

func (f *fakeSession) Start(_ context.Context) error                                { return nil }
func (f *fakeSession) Send(_ context.Context, _ string) (<-chan claude.Event, error) { return nil, nil }
func (f *fakeSession) SessionID() string                                            { return "" }
func (f *fakeSession) Close() error                                                 { return nil }
func (f *fakeSession) Restart(_ context.Context) error                              { return nil }

func testDeps(s *fakeSender) CommandDeps {
	a := agent.New(agent.Config{QueueSize: 1}, &fakeSession{}, discardLogger())
	return CommandDeps{
		Agent:   a,
		Session: &fakeSession{},
		Sender:  s,
	}
}

func TestCommand_Start(t *testing.T) {
	s := newFakeSender()
	deps := testDeps(s)

	handled := handleCommand(context.Background(), 123, "/start", deps)
	if !handled {
		t.Fatal("/start not handled")
	}

	sent := s.Sent()
	if len(sent) != 1 || sent[0].ChatID != 123 {
		t.Errorf("unexpected sent: %v", sent)
	}
}

func TestCommand_Status(t *testing.T) {
	s := newFakeSender()
	deps := testDeps(s)

	handleCommand(context.Background(), 123, "/status", deps)

	if len(s.Sent()) != 1 {
		t.Fatalf("got %d messages, want 1", len(s.Sent()))
	}
}

func TestCommand_Stop(t *testing.T) {
	s := newFakeSender()
	deps := testDeps(s)
	cancelled := false
	deps.CancelFunc = func() { cancelled = true }

	handleCommand(context.Background(), 123, "/stop", deps)

	if !cancelled {
		t.Error("cancel was not called")
	}
}

func TestCommand_Reset(t *testing.T) {
	s := newFakeSender()
	deps := testDeps(s)

	handleCommand(context.Background(), 123, "/reset", deps)

	sent := s.Sent()
	if len(sent) != 1 || sent[0].Text != "Session reset." {
		t.Errorf("unexpected: %v", sent)
	}
}

func TestCommand_Unknown(t *testing.T) {
	s := newFakeSender()
	deps := testDeps(s)

	handled := handleCommand(context.Background(), 123, "/unknown", deps)
	if handled {
		t.Error("unknown command should not be handled")
	}
}

func TestCommand_UpdateDisabled(t *testing.T) {
	s := newFakeSender()
	deps := testDeps(s)
	deps.Updater = nil

	handleCommand(context.Background(), 123, "/update", deps)

	sent := s.Sent()
	if len(sent) != 1 || sent[0].Text != "Self-update is disabled." {
		t.Errorf("unexpected: %v", sent)
	}
}
