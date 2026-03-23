package claude

import "encoding/json"

// EventType discriminates output events from Claude.
type EventType string

const (
	EventSystem    EventType = "system"
	EventAssistant EventType = "assistant"
	EventResult    EventType = "result"
)

// ResultSubtype distinguishes success from error results.
type ResultSubtype string

const (
	ResultSuccess ResultSubtype = "success"
	ResultError   ResultSubtype = "error"
)

// Event is a single NDJSON line from Claude's stdout.
type Event struct {
	Type      EventType      `json:"type"`
	Subtype   string         `json:"subtype,omitempty"`
	SessionID string         `json:"session_id,omitempty"`
	Message   *AssistantBody `json:"message,omitempty"`
	Result    string         `json:"result,omitempty"`
	Error     string         `json:"error,omitempty"`
}

// AssistantBody holds the content array from an assistant message.
type AssistantBody struct {
	Content []ContentBlock `json:"content"`
}

// ContentBlock is a text or tool_use block within an assistant message.
type ContentBlock struct {
	Type  string          `json:"type"`
	Text  string          `json:"text,omitempty"`
	ID    string          `json:"id,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`
}

// InputMessage is the envelope sent to Claude's stdin.
type InputMessage struct {
	Type            string      `json:"type"`
	Message         UserMessage `json:"message"`
	ParentToolUseID *string     `json:"parent_tool_use_id"`
}

// UserMessage is the user's message content.
type UserMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// NewUserInput creates an InputMessage for a top-level user message.
func NewUserInput(content string) InputMessage {
	return InputMessage{
		Type:            "user",
		Message:         UserMessage{Role: "user", Content: content},
		ParentToolUseID: nil,
	}
}

// NewToolResponse creates an InputMessage responding to a tool use.
func NewToolResponse(toolUseID, content string) InputMessage {
	return InputMessage{
		Type:            "user",
		Message:         UserMessage{Role: "user", Content: content},
		ParentToolUseID: &toolUseID,
	}
}
