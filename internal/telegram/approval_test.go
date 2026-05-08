package telegram

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestApprovalQueue_RequestCreatesButtons(t *testing.T) {
	s := newFakeSender()
	q := NewApprovalQueue(s, discardLogger())

	err := q.Request(123, "Send to @alice: hello", func() error { return nil })
	if err != nil {
		t.Fatal(err)
	}

	sent := s.Sent()
	if len(sent) != 1 {
		t.Fatalf("got %d messages, want 1", len(sent))
	}
	if sent[0].ChatID != 123 {
		t.Errorf("chatID = %d, want 123", sent[0].ChatID)
	}
}

func TestApprovalQueue_ApproveExecutesAction(t *testing.T) {
	s := newFakeSender()
	q := NewApprovalQueue(s, discardLogger())

	var executed atomic.Bool
	q.Request(123, "test action", func() error {
		executed.Store(true)
		return nil
	})

	// Get the action ID from the pending map.
	var actionID string
	q.mu.Lock()
	for id := range q.pending {
		actionID = id
	}
	q.mu.Unlock()

	err := q.Approve(actionID)
	if err != nil {
		t.Fatal(err)
	}
	if !executed.Load() {
		t.Error("action was not executed on approve")
	}
}

func TestApprovalQueue_RejectDiscardsAction(t *testing.T) {
	s := newFakeSender()
	q := NewApprovalQueue(s, discardLogger())

	var executed atomic.Bool
	q.Request(123, "test action", func() error {
		executed.Store(true)
		return nil
	})

	var actionID string
	q.mu.Lock()
	for id := range q.pending {
		actionID = id
	}
	q.mu.Unlock()

	err := q.Reject(actionID)
	if err != nil {
		t.Fatal(err)
	}
	if executed.Load() {
		t.Error("action should NOT be executed on reject")
	}

	// Should be removed from pending.
	q.mu.Lock()
	if len(q.pending) != 0 {
		t.Error("pending should be empty after reject")
	}
	q.mu.Unlock()
}

func TestApprovalQueue_HandleCallback(t *testing.T) {
	s := newFakeSender()
	q := NewApprovalQueue(s, discardLogger())

	var executed atomic.Bool
	q.Request(123, "test", func() error {
		executed.Store(true)
		return nil
	})

	var actionID string
	q.mu.Lock()
	for id := range q.pending {
		actionID = id
	}
	q.mu.Unlock()

	q.HandleCallback("cb1", "approve:"+actionID)

	time.Sleep(50 * time.Millisecond)
	if !executed.Load() {
		t.Error("callback approve should execute action")
	}
}

func TestApprovalQueue_HandleCallback_Reject(t *testing.T) {
	s := newFakeSender()
	q := NewApprovalQueue(s, discardLogger())

	q.Request(123, "test", func() error { return nil })

	var actionID string
	q.mu.Lock()
	for id := range q.pending {
		actionID = id
	}
	q.mu.Unlock()

	q.HandleCallback("cb2", "reject:"+actionID)

	q.mu.Lock()
	if len(q.pending) != 0 {
		t.Error("reject should remove from pending")
	}
	q.mu.Unlock()
}
