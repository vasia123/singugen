package selfupdate

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
)

type runResult struct {
	output []byte
	err    error
}

// fakeRunner records commands and returns predetermined results.
// Key format: "command subcommand" (e.g., "go build", "git diff").
type fakeRunner struct {
	mu      sync.Mutex
	results map[string]runResult
	calls   []string
}

func (f *fakeRunner) Run(_ context.Context, dir, name string, args ...string) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Build key from command + first arg.
	key := name
	if len(args) > 0 {
		key = name + " " + args[0]
	}
	f.calls = append(f.calls, key)

	if r, ok := f.results[key]; ok {
		return r.output, r.err
	}
	return nil, fmt.Errorf("fakeRunner: no result for %q", key)
}

func (f *fakeRunner) Called(key string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, c := range f.calls {
		if strings.HasPrefix(c, key) {
			return true
		}
	}
	return false
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(nopWriter{}, nil))
}

type nopWriter struct{}

func (nopWriter) Write(p []byte) (int, error) { return len(p), nil }
