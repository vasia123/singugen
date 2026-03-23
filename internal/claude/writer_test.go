package claude

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestWriter_WriteMessage(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)

	msg := NewUserInput("Hello, Claude!")
	if err := w.WriteMessage(msg); err != nil {
		t.Fatalf("WriteMessage() error: %v", err)
	}

	line := buf.String()

	// Must end with newline (NDJSON).
	if line[len(line)-1] != '\n' {
		t.Error("output does not end with newline")
	}

	// Must be valid JSON.
	var parsed InputMessage
	if err := json.Unmarshal([]byte(line), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if parsed.Type != "user" {
		t.Errorf("type = %q, want user", parsed.Type)
	}
	if parsed.Message.Role != "user" {
		t.Errorf("role = %q, want user", parsed.Message.Role)
	}
	if parsed.Message.Content != "Hello, Claude!" {
		t.Errorf("content = %q, want Hello, Claude!", parsed.Message.Content)
	}
	if parsed.ParentToolUseID != nil {
		t.Errorf("parent_tool_use_id = %v, want nil", parsed.ParentToolUseID)
	}
}

func TestWriter_WriteToolResponse(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)

	msg := NewToolResponse("tool_123", "Option 2")
	if err := w.WriteMessage(msg); err != nil {
		t.Fatalf("WriteMessage() error: %v", err)
	}

	var parsed InputMessage
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if parsed.ParentToolUseID == nil || *parsed.ParentToolUseID != "tool_123" {
		t.Errorf("parent_tool_use_id = %v, want tool_123", parsed.ParentToolUseID)
	}
}
