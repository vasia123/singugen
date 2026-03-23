package claude

import (
	"context"
	"strings"
	"testing"
	"time"
)

func collectEvents(ch <-chan Event) []Event {
	var events []Event
	for e := range ch {
		events = append(events, e)
	}
	return events
}

func TestReadEvents_TextResponse(t *testing.T) {
	input := strings.Join([]string{
		`{"type":"system","subtype":"init","session_id":"sess-123"}`,
		`{"type":"assistant","message":{"content":[{"type":"text","text":"Hello!"}]}}`,
		`{"type":"result","subtype":"success","session_id":"sess-123","result":"Hello!"}`,
	}, "\n") + "\n"

	events := collectEvents(ReadEvents(context.Background(), strings.NewReader(input)))

	if len(events) != 3 {
		t.Fatalf("got %d events, want 3", len(events))
	}

	if events[0].Type != EventSystem || events[0].Subtype != "init" || events[0].SessionID != "sess-123" {
		t.Errorf("event[0] = %+v, want system/init", events[0])
	}

	if events[1].Type != EventAssistant {
		t.Fatalf("event[1].Type = %q, want assistant", events[1].Type)
	}
	if len(events[1].Message.Content) != 1 || events[1].Message.Content[0].Text != "Hello!" {
		t.Errorf("event[1] content = %+v, want text Hello!", events[1].Message.Content)
	}

	if events[2].Type != EventResult || events[2].Subtype != string(ResultSuccess) {
		t.Errorf("event[2] = %+v, want result/success", events[2])
	}
}

func TestReadEvents_ToolUse(t *testing.T) {
	input := `{"type":"assistant","message":{"content":[{"type":"tool_use","id":"tu_1","name":"ReadFile","input":{"path":"/tmp/x"}}]}}` + "\n"

	events := collectEvents(ReadEvents(context.Background(), strings.NewReader(input)))

	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}

	block := events[0].Message.Content[0]
	if block.Type != "tool_use" || block.ID != "tu_1" || block.Name != "ReadFile" {
		t.Errorf("block = %+v, want tool_use tu_1 ReadFile", block)
	}
	if string(block.Input) != `{"path":"/tmp/x"}` {
		t.Errorf("input = %s, want {\"path\":\"/tmp/x\"}", block.Input)
	}
}

func TestReadEvents_ErrorResult(t *testing.T) {
	input := `{"type":"result","subtype":"error","error":"something broke"}` + "\n"

	events := collectEvents(ReadEvents(context.Background(), strings.NewReader(input)))

	if len(events) != 1 {
		t.Fatalf("got %d events, want 1", len(events))
	}
	if events[0].Error != "something broke" {
		t.Errorf("error = %q, want something broke", events[0].Error)
	}
}

func TestReadEvents_ContextCancel(t *testing.T) {
	// Blocking reader that never returns data.
	pr, _ := newTestPipe()

	ctx, cancel := context.WithCancel(context.Background())
	ch := ReadEvents(ctx, pr)

	cancel()

	// Channel should close promptly.
	select {
	case _, ok := <-ch:
		if ok {
			t.Error("expected channel to be closed")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("channel not closed after context cancel")
	}
}

func TestReadEvents_MalformedJSON(t *testing.T) {
	input := strings.Join([]string{
		`{"type":"system","subtype":"init","session_id":"s1"}`,
		`{{{bad json`,
		`{"type":"result","subtype":"success","result":"done"}`,
	}, "\n") + "\n"

	events := collectEvents(ReadEvents(context.Background(), strings.NewReader(input)))

	// Bad line skipped, should get 2 events.
	if len(events) != 2 {
		t.Fatalf("got %d events, want 2 (malformed line skipped)", len(events))
	}
	if events[0].Type != EventSystem {
		t.Errorf("events[0].Type = %q, want system", events[0].Type)
	}
	if events[1].Type != EventResult {
		t.Errorf("events[1].Type = %q, want result", events[1].Type)
	}
}
