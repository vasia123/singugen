package telegram

import (
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/vasis/singugen/internal/claude"
)

// Handler implements agent.MessageHandler, bridging Claude events
// to Telegram messages. One instance per user request.
type Handler struct {
	chatID         int64
	sender         Sender
	logger         *slog.Logger
	statusDebounce time.Duration

	mu          sync.Mutex
	statusMsgID int
	lastEdit    time.Time
	statusSent  bool
}

// NewHandler creates a per-request handler for the given chat.
func NewHandler(chatID int64, sender Sender, logger *slog.Logger) *Handler {
	return &Handler{
		chatID:         chatID,
		sender:         sender,
		logger:         logger,
		statusDebounce: 1 * time.Second,
	}
}

// OnEvent receives streaming events from Claude.
func (h *Handler) OnEvent(event claude.Event) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Send status message on first event.
	if !h.statusSent {
		id, err := h.sender.SendMessage(h.chatID, "Thinking...")
		if err != nil {
			h.logger.Error("failed to send status", "error", err)
		} else {
			h.statusMsgID = id
		}
		h.statusSent = true
	}

	// Update status on tool_use events.
	if event.Type == claude.EventAssistant && event.Message != nil {
		for _, block := range event.Message.Content {
			if block.Type == "tool_use" {
				h.updateStatus(toolLabel(block.Name))
			}
		}
	}
}

// OnComplete is called when the agent finishes processing.
func (h *Handler) OnComplete(result string, err error) {
	h.mu.Lock()
	statusMsgID := h.statusMsgID
	statusSent := h.statusSent
	h.mu.Unlock()

	// Delete status message.
	if statusSent && statusMsgID > 0 {
		if delErr := h.sender.DeleteMessage(h.chatID, statusMsgID); delErr != nil {
			h.logger.Debug("failed to delete status", "error", delErr)
		}
	}

	// Send result or error.
	if err != nil {
		h.sender.SendMessage(h.chatID, fmt.Sprintf("Error: %v", err))
		return
	}

	if result == "" {
		return
	}

	chunks := ChunkText(result, MaxMessageLen)
	for _, chunk := range chunks {
		if _, sendErr := h.sender.SendMessage(h.chatID, chunk); sendErr != nil {
			h.logger.Error("failed to send chunk", "error", sendErr)
			break
		}
	}
}

func (h *Handler) updateStatus(text string) {
	if h.statusMsgID == 0 {
		return
	}
	if time.Since(h.lastEdit) < h.statusDebounce {
		return
	}
	if err := h.sender.EditMessage(h.chatID, h.statusMsgID, text); err != nil {
		h.logger.Debug("failed to edit status", "error", err)
		return
	}
	h.lastEdit = time.Now()
}

func toolLabel(name string) string {
	switch name {
	case "Read":
		return "Reading file..."
	case "Write":
		return "Writing file..."
	case "Edit":
		return "Editing code..."
	case "Bash":
		return "Running command..."
	case "Grep":
		return "Searching code..."
	case "Glob":
		return "Finding files..."
	case "WebSearch":
		return "Searching web..."
	case "WebFetch":
		return "Fetching page..."
	default:
		return "Working..."
	}
}
