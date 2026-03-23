package telegram

import (
	"fmt"
	"testing"
	"time"

	"github.com/vasis/singugen/internal/claude"
)

func TestHandler_FirstEventSendsStatus(t *testing.T) {
	s := newFakeSender()
	h := NewHandler(123, s, discardLogger())

	h.OnEvent(claude.Event{Type: claude.EventSystem, Subtype: "init"})

	sent := s.Sent()
	if len(sent) != 1 {
		t.Fatalf("got %d sent messages, want 1", len(sent))
	}
	if sent[0].ChatID != 123 {
		t.Errorf("chatID = %d, want 123", sent[0].ChatID)
	}
}

func TestHandler_ToolUseUpdatesStatus(t *testing.T) {
	s := newFakeSender()
	h := NewHandler(123, s, discardLogger())

	h.OnEvent(claude.Event{Type: claude.EventSystem, Subtype: "init"})

	h.OnEvent(claude.Event{
		Type: claude.EventAssistant,
		Message: &claude.AssistantBody{
			Content: []claude.ContentBlock{{Type: "tool_use", Name: "Read"}},
		},
	})

	edited := s.Edited()
	if len(edited) != 1 {
		t.Fatalf("got %d edits, want 1", len(edited))
	}
	if edited[0].MessageID != 1 {
		t.Errorf("edited message ID = %d, want 1", edited[0].MessageID)
	}
}

func TestHandler_DebounceEdits(t *testing.T) {
	s := newFakeSender()
	h := NewHandler(123, s, discardLogger())
	h.statusDebounce = 100 * time.Millisecond

	h.OnEvent(claude.Event{Type: claude.EventSystem, Subtype: "init"})

	// Two tool_use events in quick succession — only first should edit.
	h.OnEvent(claude.Event{
		Type:    claude.EventAssistant,
		Message: &claude.AssistantBody{Content: []claude.ContentBlock{{Type: "tool_use", Name: "Read"}}},
	})
	h.OnEvent(claude.Event{
		Type:    claude.EventAssistant,
		Message: &claude.AssistantBody{Content: []claude.ContentBlock{{Type: "tool_use", Name: "Write"}}},
	})

	edited := s.Edited()
	if len(edited) != 1 {
		t.Errorf("got %d edits, want 1 (debounced)", len(edited))
	}
}

func TestHandler_OnCompleteDeletesStatusAndSendsResult(t *testing.T) {
	s := newFakeSender()
	h := NewHandler(123, s, discardLogger())

	h.OnEvent(claude.Event{Type: claude.EventSystem, Subtype: "init"})

	h.OnComplete("Hello, world!", nil)

	deleted := s.Deleted()
	if len(deleted) != 1 {
		t.Fatalf("got %d deletes, want 1", len(deleted))
	}

	sent := s.Sent()
	// First sent is status, second is result.
	if len(sent) != 2 {
		t.Fatalf("got %d sent, want 2 (status + result)", len(sent))
	}
	if sent[1].Text != "Hello, world!" {
		t.Errorf("result text = %q, want Hello, world!", sent[1].Text)
	}
}

func TestHandler_OnCompleteWithError(t *testing.T) {
	s := newFakeSender()
	h := NewHandler(123, s, discardLogger())

	h.OnEvent(claude.Event{Type: claude.EventSystem, Subtype: "init"})

	h.OnComplete("", fmt.Errorf("something broke"))

	sent := s.Sent()
	if len(sent) != 2 {
		t.Fatalf("got %d sent, want 2", len(sent))
	}
	if sent[1].Text == "" {
		t.Error("error message should not be empty")
	}
}

func TestHandler_OnCompleteChunksLongResult(t *testing.T) {
	s := newFakeSender()
	h := NewHandler(123, s, discardLogger())

	h.OnEvent(claude.Event{Type: claude.EventSystem, Subtype: "init"})

	longText := make([]byte, 5000)
	for i := range longText {
		longText[i] = 'a'
	}
	h.OnComplete(string(longText), nil)

	sent := s.Sent()
	// 1 status + 2 chunks = 3.
	if len(sent) != 3 {
		t.Fatalf("got %d sent, want 3 (status + 2 chunks)", len(sent))
	}
}
