package claude

import (
	"encoding/json"
	"io"
	"sync"
)

// Writer writes NDJSON input messages to Claude's stdin.
type Writer struct {
	mu  sync.Mutex
	enc *json.Encoder
}

// NewWriter creates a Writer that writes to w.
func NewWriter(w io.Writer) *Writer {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return &Writer{enc: enc}
}

// WriteMessage serializes msg as a single NDJSON line.
func (w *Writer) WriteMessage(msg InputMessage) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.enc.Encode(msg)
}
