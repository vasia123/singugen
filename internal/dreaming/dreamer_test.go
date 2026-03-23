package dreaming

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"testing"

	"github.com/vasis/singugen/internal/claude"
	"github.com/vasis/singugen/internal/memory"
)

type nopWriter struct{}

func (nopWriter) Write(p []byte) (int, error) { return len(p), nil }

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(nopWriter{}, nil))
}

type fakeDreamSession struct {
	mu       sync.Mutex
	messages []string
	response string
	err      error
}

func (f *fakeDreamSession) Send(_ context.Context, message string) (<-chan claude.Event, error) {
	f.mu.Lock()
	f.messages = append(f.messages, message)
	resp := f.response
	sendErr := f.err
	f.mu.Unlock()

	if sendErr != nil {
		return nil, sendErr
	}

	ch := make(chan claude.Event, 1)
	ch <- claude.Event{
		Type:    claude.EventResult,
		Subtype: string(claude.ResultSuccess),
		Result:  resp,
	}
	close(ch)
	return ch, nil
}

func TestDreamer_UpdatesMemory(t *testing.T) {
	dir := t.TempDir()
	store := memory.New(dir, testLogger())
	store.Init()
	store.Save("user", "old user info")

	sess := &fakeDreamSession{
		response: `<<<MEMORY_UPDATE>>>
<<<FILE:user.md>>>
updated user info
<<<END_MEMORY_UPDATE>>>`,
	}

	d := New(store, sess, testLogger())
	if err := d.Dream(context.Background()); err != nil {
		t.Fatalf("Dream() error: %v", err)
	}

	entry, err := store.Load("user")
	if err != nil {
		t.Fatal(err)
	}
	if entry.Content != "updated user info" {
		t.Errorf("content = %q, want updated user info", entry.Content)
	}
}

func TestDreamer_NoChanges(t *testing.T) {
	dir := t.TempDir()
	store := memory.New(dir, testLogger())
	store.Init()
	store.Save("user", "original")

	sess := &fakeDreamSession{
		response: "Everything looks great.\n<<<NO_CHANGES>>>",
	}

	d := New(store, sess, testLogger())
	if err := d.Dream(context.Background()); err != nil {
		t.Fatal(err)
	}

	entry, _ := store.Load("user")
	if entry.Content != "original" {
		t.Errorf("content changed to %q, should remain original", entry.Content)
	}
}

func TestDreamer_SessionError(t *testing.T) {
	dir := t.TempDir()
	store := memory.New(dir, testLogger())
	store.Init()

	sess := &fakeDreamSession{err: fmt.Errorf("connection failed")}

	d := New(store, sess, testLogger())
	err := d.Dream(context.Background())
	if err == nil {
		t.Error("Dream() should return error on session failure")
	}
}

func TestDreamer_MalformedResponse(t *testing.T) {
	dir := t.TempDir()
	store := memory.New(dir, testLogger())
	store.Init()
	store.Save("user", "original")

	sess := &fakeDreamSession{response: "just some random text without markers"}

	d := New(store, sess, testLogger())
	err := d.Dream(context.Background())
	if err == nil {
		t.Error("Dream() should return error on malformed response")
	}

	// Memory should be unchanged.
	entry, _ := store.Load("user")
	if entry.Content != "original" {
		t.Errorf("content changed to %q after malformed response", entry.Content)
	}
}

func TestDreamer_ContextCancelled(t *testing.T) {
	dir := t.TempDir()
	store := memory.New(dir, testLogger())
	store.Init()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	sess := &fakeDreamSession{response: "<<<NO_CHANGES>>>"}
	d := New(store, sess, testLogger())

	// Should handle cancelled context gracefully.
	_ = d.Dream(ctx)
}
