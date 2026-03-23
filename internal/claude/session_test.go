package claude

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"slices"
	"testing"
	"time"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(nopWriter{}, nil))
}

type nopWriter struct{}

func (nopWriter) Write(p []byte) (int, error) { return len(p), nil }

func writeEvent(w io.Writer, e Event) {
	data, _ := json.Marshal(e)
	fmt.Fprintf(w, "%s\n", data)
}

func TestSession_SendAndReceive(t *testing.T) {
	fake := NewFakeLauncher()
	sess := NewSession(SessionConfig{}, fake, testLogger())
	if err := sess.Start(context.Background()); err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	defer sess.Close()

	conn := <-fake.Conns
	defer conn.Close()

	go func() {
		scanner := bufio.NewScanner(conn.StdinReader)
		scanner.Scan()

		var msg InputMessage
		if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
			t.Errorf("failed to parse input: %v", err)
			return
		}
		if msg.Message.Content != "Hello!" {
			t.Errorf("message content = %q, want Hello!", msg.Message.Content)
		}

		writeEvent(conn.StdoutWriter, Event{Type: EventSystem, Subtype: "init", SessionID: "sess-1"})
		writeEvent(conn.StdoutWriter, Event{Type: EventAssistant, Message: &AssistantBody{
			Content: []ContentBlock{{Type: "text", Text: "Hi there!"}},
		}})
		writeEvent(conn.StdoutWriter, Event{Type: EventResult, Subtype: string(ResultSuccess), SessionID: "sess-1", Result: "Hi there!"})
	}()

	ch, err := sess.Send(context.Background(), "Hello!")
	if err != nil {
		t.Fatalf("Send() error: %v", err)
	}

	var events []Event
	for e := range ch {
		events = append(events, e)
	}

	if len(events) != 3 {
		t.Fatalf("got %d events, want 3", len(events))
	}
	if events[0].Type != EventSystem {
		t.Errorf("events[0].Type = %q, want system", events[0].Type)
	}
	if events[1].Message.Content[0].Text != "Hi there!" {
		t.Errorf("events[1] text = %q, want Hi there!", events[1].Message.Content[0].Text)
	}
	if events[2].Type != EventResult {
		t.Errorf("events[2].Type = %q, want result", events[2].Type)
	}
}

func TestSession_SessionID(t *testing.T) {
	fake := NewFakeLauncher()
	sess := NewSession(SessionConfig{}, fake, testLogger())
	if err := sess.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer sess.Close()

	conn := <-fake.Conns
	defer conn.Close()

	if id := sess.SessionID(); id != "" {
		t.Errorf("SessionID() = %q before init, want empty", id)
	}

	go func() {
		scanner := bufio.NewScanner(conn.StdinReader)
		scanner.Scan()

		writeEvent(conn.StdoutWriter, Event{Type: EventSystem, Subtype: "init", SessionID: "my-session-42"})
		writeEvent(conn.StdoutWriter, Event{Type: EventResult, Subtype: string(ResultSuccess), SessionID: "my-session-42", Result: "ok"})
	}()

	ch, err := sess.Send(context.Background(), "test")
	if err != nil {
		t.Fatal(err)
	}
	for range ch {
	}

	if id := sess.SessionID(); id != "my-session-42" {
		t.Errorf("SessionID() = %q, want my-session-42", id)
	}
}

func TestSession_Close(t *testing.T) {
	fake := NewFakeLauncher()
	sess := NewSession(SessionConfig{}, fake, testLogger())
	if err := sess.Start(context.Background()); err != nil {
		t.Fatal(err)
	}

	conn := <-fake.Conns
	defer conn.Close()

	if err := sess.Close(); err != nil {
		t.Fatalf("Close() error: %v", err)
	}

	_, err := sess.Send(context.Background(), "hello")
	if err == nil {
		t.Error("Send() after Close() should return error")
	}
}

func TestSession_Timeout(t *testing.T) {
	fake := NewFakeLauncher()
	sess := NewSession(SessionConfig{Timeout: 100 * time.Millisecond}, fake, testLogger())
	if err := sess.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer sess.Close()

	conn := <-fake.Conns
	defer conn.Close()

	go func() {
		scanner := bufio.NewScanner(conn.StdinReader)
		scanner.Scan()
		// Never respond — trigger watchdog.
	}()

	ch, err := sess.Send(context.Background(), "hello")
	if err != nil {
		t.Fatal(err)
	}

	var lastEvent Event
	for e := range ch {
		lastEvent = e
	}

	if lastEvent.Type != EventResult || lastEvent.Subtype != string(ResultError) {
		t.Errorf("expected timeout error event, got %+v", lastEvent)
	}
	if lastEvent.Error == "" {
		t.Error("timeout event should have error message")
	}
}

func TestSession_BuildArgs(t *testing.T) {
	sess := NewSession(SessionConfig{
		Model:        "claude-sonnet-4-6",
		SystemPrompt: "You are helpful",
	}, nil, testLogger())

	args := sess.buildArgs()

	expected := map[string]bool{
		"-p":                              true,
		"--input-format":                  true,
		"stream-json":                     true,
		"--output-format":                 true,
		"--verbose":                       true,
		"--dangerously-skip-permissions":  true,
		"--model":                         true,
		"claude-sonnet-4-6":               true,
		"--system-prompt":                 true,
		"You are helpful":                 true,
	}

	for _, arg := range args {
		if !expected[arg] {
			t.Errorf("unexpected arg: %q", arg)
		}
	}
}

func TestSession_Resume(t *testing.T) {
	fake := NewFakeLauncher()
	sess := NewSession(SessionConfig{}, fake, testLogger())

	// First start — no session ID.
	if err := sess.Start(context.Background()); err != nil {
		t.Fatal(err)
	}

	conn1 := <-fake.Conns
	args1 := fake.ArgsLog()[0]
	if slices.Contains(args1, "--resume") {
		t.Error("first launch should not have --resume")
	}

	// Simulate getting a session ID.
	go func() {
		scanner := bufio.NewScanner(conn1.StdinReader)
		scanner.Scan()
		writeEvent(conn1.StdoutWriter, Event{Type: EventSystem, Subtype: "init", SessionID: "sess-abc"})
		writeEvent(conn1.StdoutWriter, Event{Type: EventResult, Subtype: string(ResultSuccess), SessionID: "sess-abc", Result: "ok"})
	}()

	ch, err := sess.Send(context.Background(), "first")
	if err != nil {
		t.Fatal(err)
	}
	for range ch {
	}
	conn1.Close()

	// Restart — should use --resume with session ID.
	if err := sess.Restart(context.Background()); err != nil {
		t.Fatal(err)
	}

	conn2 := <-fake.Conns
	defer conn2.Close()

	args2 := fake.ArgsLog()[1]
	resumeIdx := slices.Index(args2, "--resume")
	if resumeIdx == -1 {
		t.Fatal("second launch should have --resume")
	}
	if args2[resumeIdx+1] != "sess-abc" {
		t.Errorf("resume session ID = %q, want sess-abc", args2[resumeIdx+1])
	}
}

func TestSession_ProcessDeath(t *testing.T) {
	fake := NewFakeLauncher()
	sess := NewSession(SessionConfig{Timeout: 2 * time.Second}, fake, testLogger())
	if err := sess.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer sess.Close()

	conn := <-fake.Conns

	go func() {
		scanner := bufio.NewScanner(conn.StdinReader)
		scanner.Scan()

		// Send one event then close stdout (simulate crash).
		writeEvent(conn.StdoutWriter, Event{Type: EventAssistant, Message: &AssistantBody{
			Content: []ContentBlock{{Type: "text", Text: "partial"}},
		}})
		conn.StdoutWriter.Close()
	}()

	ch, err := sess.Send(context.Background(), "hello")
	if err != nil {
		t.Fatal(err)
	}

	var events []Event
	for e := range ch {
		events = append(events, e)
	}

	// Should get the partial event, then channel closes (no result event).
	if len(events) != 1 {
		t.Fatalf("got %d events, want 1 (partial before crash)", len(events))
	}
	if events[0].Message.Content[0].Text != "partial" {
		t.Errorf("event text = %q, want partial", events[0].Message.Content[0].Text)
	}
}
