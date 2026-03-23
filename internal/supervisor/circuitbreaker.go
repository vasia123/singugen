package supervisor

import (
	"sync"
	"time"
)

// CircuitBreaker trips when the child process restarts more than
// maxRestarts times within window. This prevents crash loops.
type CircuitBreaker struct {
	maxRestarts int
	window      time.Duration

	mu         sync.Mutex
	timestamps []time.Time
	now        func() time.Time
}

func NewCircuitBreaker(maxRestarts int, window time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		maxRestarts: maxRestarts,
		window:      window,
		now:         time.Now,
	}
}

// Record registers a restart event.
func (cb *CircuitBreaker) Record() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.timestamps = append(cb.timestamps, cb.now())
	cb.prune()
}

// IsOpen returns true if too many restarts occurred within the window.
func (cb *CircuitBreaker) IsOpen() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.prune()
	return len(cb.timestamps) >= cb.maxRestarts
}

// Reset clears the restart history.
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.timestamps = cb.timestamps[:0]
}

// prune removes entries older than window. Must be called with mu held.
func (cb *CircuitBreaker) prune() {
	cutoff := cb.now().Add(-cb.window)
	i := 0
	for i < len(cb.timestamps) && cb.timestamps[i].Before(cutoff) {
		i++
	}
	if i > 0 {
		cb.timestamps = cb.timestamps[i:]
	}
}
