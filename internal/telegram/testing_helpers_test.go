package telegram

import (
	"log/slog"
	"sync"
)

type sentMsg struct {
	ChatID int64
	Text   string
}

type editedMsg struct {
	ChatID    int64
	MessageID int
	Text      string
}

type deletedMsg struct {
	ChatID    int64
	MessageID int
}

type fakeSender struct {
	mu      sync.Mutex
	sent    []sentMsg
	edited  []editedMsg
	deleted []deletedMsg
	nextID  int
}

func newFakeSender() *fakeSender {
	return &fakeSender{nextID: 1}
}

func (f *fakeSender) SendMessage(chatID int64, text string) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	id := f.nextID
	f.nextID++
	f.sent = append(f.sent, sentMsg{ChatID: chatID, Text: text})
	return id, nil
}

func (f *fakeSender) EditMessage(chatID int64, messageID int, text string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.edited = append(f.edited, editedMsg{ChatID: chatID, MessageID: messageID, Text: text})
	return nil
}

func (f *fakeSender) DeleteMessage(chatID int64, messageID int) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.deleted = append(f.deleted, deletedMsg{ChatID: chatID, MessageID: messageID})
	return nil
}

func (f *fakeSender) Sent() []sentMsg {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]sentMsg, len(f.sent))
	copy(out, f.sent)
	return out
}

func (f *fakeSender) Edited() []editedMsg {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]editedMsg, len(f.edited))
	copy(out, f.edited)
	return out
}

func (f *fakeSender) Deleted() []deletedMsg {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]deletedMsg, len(f.deleted))
	copy(out, f.deleted)
	return out
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(nopWriter{}, nil))
}

type nopWriter struct{}

func (nopWriter) Write(p []byte) (int, error) { return len(p), nil }
