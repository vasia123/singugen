package claude

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"log/slog"
)

// ReadEvents reads NDJSON events from r until EOF or ctx cancellation.
// If r implements io.Closer, it will be closed on context cancellation
// to unblock the scanner. Malformed lines are logged and skipped.
// The returned channel is closed when reading stops.
func ReadEvents(ctx context.Context, r io.Reader) <-chan Event {
	ch := make(chan Event)

	// Unblock scanner.Scan() when context is cancelled.
	if rc, ok := r.(io.Closer); ok {
		go func() {
			<-ctx.Done()
			rc.Close()
		}()
	}

	go func() {
		defer close(ch)

		scanner := bufio.NewScanner(r)
		// Allow up to 1MB lines (Claude can produce large tool outputs).
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

		for scanner.Scan() {
			if ctx.Err() != nil {
				return
			}

			line := scanner.Bytes()
			if len(line) == 0 {
				continue
			}

			var event Event
			if err := json.Unmarshal(line, &event); err != nil {
				slog.Warn("claude: skipping malformed event", "error", err)
				continue
			}

			select {
			case ch <- event:
			case <-ctx.Done():
				return
			}
		}
	}()

	return ch
}
