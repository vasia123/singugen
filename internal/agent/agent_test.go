package agent

import (
	"context"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/vasis/singugen/internal/claude"
)

type fakeSession struct {
	mu       sync.Mutex
	messages []string
	events   []claude.Event
}

func newFakeSession(events ...claude.Event) *fakeSession {
	return &fakeSession{events: events}
}

func (f *fakeSession) Start(_ context.Context) error { return nil }

func (f *fakeSession) Send(_ context.Context, message string) (<-chan claude.Event, error) {
	f.mu.Lock()
	f.messages = append(f.messages, message)
	events := make([]claude.Event, len(f.events))
	copy(events, f.events)
	f.mu.Unlock()

	ch := make(chan claude.Event, len(events))
	for _, e := range events {
		ch <- e
	}
	close(ch)
	return ch, nil
}

func (f *fakeSession) SessionID() string { return "fake-session" }
func (f *fakeSession) Close() error      { return nil }
func (f *fakeSession) Restart(_ context.Context) error { return nil }

func (f *fakeSession) Messages() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	msgs := make([]string, len(f.messages))
	copy(msgs, f.messages)
	return msgs
}

type recordingHandler struct {
	mu     sync.Mutex
	events []claude.Event
	result string
	err    error
	done   chan struct{}
}

func newRecordingHandler() *recordingHandler {
	return &recordingHandler{done: make(chan struct{})}
}

func (h *recordingHandler) OnEvent(event claude.Event) {
	h.mu.Lock()
	h.events = append(h.events, event)
	h.mu.Unlock()
}

func (h *recordingHandler) OnComplete(result string, err error) {
	h.mu.Lock()
	h.result = result
	h.err = err
	h.mu.Unlock()
	close(h.done)
}

func (h *recordingHandler) waitDone(t *testing.T) {
	t.Helper()
	select {
	case <-h.done:
	case <-time.After(2 * time.Second):
		t.Fatal("handler.OnComplete not called")
	}
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(nopWriter{}, nil))
}

type nopWriter struct{}

func (nopWriter) Write(p []byte) (int, error) { return len(p), nil }

func TestAgent_StateTransitions(t *testing.T) {
	sess := newFakeSession(
		claude.Event{Type: claude.EventResult, Subtype: string(claude.ResultSuccess), Result: "ok"},
	)
	a := New(Config{QueueSize: 8}, sess, discardLogger())

	if a.State() != StateStarting {
		t.Errorf("initial state = %v, want starting", a.State())
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- a.Run(ctx) }()

	// Wait for ready state.
	time.Sleep(50 * time.Millisecond)
	if a.State() != StateReady {
		t.Errorf("state after Run = %v, want ready", a.State())
	}

	// Submit a message to trigger processing.
	handler := newRecordingHandler()
	if err := a.Submit(Request{Message: "hi", Handler: handler}); err != nil {
		t.Fatal(err)
	}
	handler.waitDone(t)

	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return")
	}

	if a.State() != StateStopped {
		t.Errorf("state after cancel = %v, want stopped", a.State())
	}
}

func TestAgent_SubmitAndProcess(t *testing.T) {
	sess := newFakeSession(
		claude.Event{Type: claude.EventAssistant, Message: &claude.AssistantBody{
			Content: []claude.ContentBlock{{Type: "text", Text: "response"}},
		}},
		claude.Event{Type: claude.EventResult, Subtype: string(claude.ResultSuccess), Result: "response"},
	)
	a := New(Config{QueueSize: 8}, sess, discardLogger())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go a.Run(ctx)
	time.Sleep(50 * time.Millisecond)

	handler := newRecordingHandler()
	if err := a.Submit(Request{Message: "hello", Handler: handler}); err != nil {
		t.Fatal(err)
	}
	handler.waitDone(t)

	msgs := sess.Messages()
	if len(msgs) != 1 || msgs[0] != "hello" {
		t.Errorf("session messages = %v, want [hello]", msgs)
	}

	handler.mu.Lock()
	defer handler.mu.Unlock()
	if handler.result != "response" {
		t.Errorf("result = %q, want response", handler.result)
	}
	if len(handler.events) != 2 {
		t.Errorf("got %d events, want 2", len(handler.events))
	}
}

func TestAgent_QueueFull(t *testing.T) {
	sess := newFakeSession() // no events — will block on Send
	a := New(Config{QueueSize: 1}, sess, discardLogger())

	// Don't start Run — queue will fill up.
	if err := a.Submit(Request{Message: "first", Handler: newRecordingHandler()}); err != nil {
		t.Fatalf("first submit should succeed: %v", err)
	}

	err := a.Submit(Request{Message: "second", Handler: newRecordingHandler()})
	if err == nil {
		t.Error("second submit should fail when queue is full")
	}
}

func TestAgent_HandlerReceivesEvents(t *testing.T) {
	sess := newFakeSession(
		claude.Event{Type: claude.EventSystem, Subtype: "init", SessionID: "s1"},
		claude.Event{Type: claude.EventAssistant, Message: &claude.AssistantBody{
			Content: []claude.ContentBlock{{Type: "text", Text: "hello"}},
		}},
		claude.Event{Type: claude.EventResult, Subtype: string(claude.ResultSuccess), Result: "hello"},
	)
	a := New(Config{QueueSize: 8}, sess, discardLogger())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go a.Run(ctx)
	time.Sleep(50 * time.Millisecond)

	handler := newRecordingHandler()
	a.Submit(Request{Message: "test", Handler: handler})
	handler.waitDone(t)

	handler.mu.Lock()
	defer handler.mu.Unlock()
	if len(handler.events) != 3 {
		t.Fatalf("got %d events, want 3", len(handler.events))
	}
	if handler.events[0].Type != claude.EventSystem {
		t.Errorf("event[0].Type = %q, want system", handler.events[0].Type)
	}
	if handler.err != nil {
		t.Errorf("handler.err = %v, want nil", handler.err)
	}
}

func TestAgent_ShutdownDrainsQueue(t *testing.T) {
	callCount := 0
	sess := &fakeSession{}

	// Return events on each call.
	origSend := sess.Send
	_ = origSend
	sess.events = []claude.Event{
		{Type: claude.EventResult, Subtype: string(claude.ResultSuccess), Result: "ok"},
	}

	a := New(Config{QueueSize: 8}, sess, discardLogger())

	ctx, cancel := context.WithCancel(context.Background())
	go a.Run(ctx)
	time.Sleep(50 * time.Millisecond)

	// Submit multiple messages.
	handlers := make([]*recordingHandler, 3)
	for i := range 3 {
		handlers[i] = newRecordingHandler()
		a.Submit(Request{Message: "msg", Handler: handlers[i]})
		callCount++
	}

	// Cancel and wait for all to complete.
	cancel()
	for _, h := range handlers {
		h.waitDone(t)
	}

	msgs := sess.Messages()
	if len(msgs) != 3 {
		t.Errorf("processed %d messages, want 3 (all drained)", len(msgs))
	}
}
