package telegram

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"
)

// PendingAction represents an action awaiting user approval.
type PendingAction struct {
	ID          string
	Description string
	Action      func() error
	ChatID      int64
	MessageID   int
	CreatedAt   time.Time
}

// ApprovalQueue manages pending actions with inline button approval.
type ApprovalQueue struct {
	pending map[string]*PendingAction
	mu      sync.Mutex
	sender  Sender
	logger  *slog.Logger
}

// NewApprovalQueue creates an approval queue.
func NewApprovalQueue(sender Sender, logger *slog.Logger) *ApprovalQueue {
	return &ApprovalQueue{
		pending: make(map[string]*PendingAction),
		sender:  sender,
		logger:  logger,
	}
}

// Request creates a pending action and sends an approval message with inline buttons.
func (q *ApprovalQueue) Request(chatID int64, description string, action func() error) error {
	id := generateID()

	buttons := [][]InlineButton{
		{
			{Label: "Approve ✓", Data: "approve:" + id},
			{Label: "Reject ✗", Data: "reject:" + id},
		},
	}

	msgID, err := q.sender.SendMessageWithButtons(chatID, fmt.Sprintf("🔔 Approval needed:\n%s", description), buttons)
	if err != nil {
		return fmt.Errorf("approval: send buttons: %w", err)
	}

	q.mu.Lock()
	q.pending[id] = &PendingAction{
		ID:          id,
		Description: description,
		Action:      action,
		ChatID:      chatID,
		MessageID:   msgID,
		CreatedAt:   time.Now(),
	}
	q.mu.Unlock()

	q.logger.Info("approval requested", "id", id, "description", description)
	return nil
}

// Approve executes the pending action and removes it.
func (q *ApprovalQueue) Approve(actionID string) error {
	q.mu.Lock()
	action, ok := q.pending[actionID]
	if !ok {
		q.mu.Unlock()
		return fmt.Errorf("approval: action %s not found", actionID)
	}
	delete(q.pending, actionID)
	q.mu.Unlock()

	q.logger.Info("action approved", "id", actionID)

	if err := action.Action(); err != nil {
		q.sender.EditMessage(action.ChatID, action.MessageID, fmt.Sprintf("✗ Failed: %v", err))
		return err
	}

	q.sender.EditMessage(action.ChatID, action.MessageID, fmt.Sprintf("✓ Approved: %s", action.Description))
	return nil
}

// Reject discards the pending action.
func (q *ApprovalQueue) Reject(actionID string) error {
	q.mu.Lock()
	action, ok := q.pending[actionID]
	if !ok {
		q.mu.Unlock()
		return fmt.Errorf("approval: action %s not found", actionID)
	}
	delete(q.pending, actionID)
	q.mu.Unlock()

	q.logger.Info("action rejected", "id", actionID)
	q.sender.EditMessage(action.ChatID, action.MessageID, fmt.Sprintf("✗ Rejected: %s", action.Description))
	return nil
}

// HandleCallback processes an inline button callback.
func (q *ApprovalQueue) HandleCallback(callbackID, data string) {
	parts := strings.SplitN(data, ":", 2)
	if len(parts) != 2 {
		return
	}

	action := parts[0]
	actionID := parts[1]

	switch action {
	case "approve":
		if err := q.Approve(actionID); err != nil {
			q.logger.Warn("approve failed", "error", err)
		}
		q.sender.AnswerCallback(callbackID, "Approved")
	case "reject":
		if err := q.Reject(actionID); err != nil {
			q.logger.Warn("reject failed", "error", err)
		}
		q.sender.AnswerCallback(callbackID, "Rejected")
	}
}

func generateID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}
