package supervisor

import (
	"testing"
	"time"
)

func TestCircuitBreaker_StaysClosedUnderLimit(t *testing.T) {
	cb := NewCircuitBreaker(5, 2*time.Minute)

	for i := 0; i < 4; i++ {
		cb.Record()
	}

	if cb.IsOpen() {
		t.Error("breaker is open after 4 records, want closed (limit 5)")
	}
}

func TestCircuitBreaker_OpensAtLimit(t *testing.T) {
	now := time.Now()
	cb := NewCircuitBreaker(5, 2*time.Minute)
	cb.now = func() time.Time { return now }

	for i := 0; i < 5; i++ {
		cb.Record()
	}

	if !cb.IsOpen() {
		t.Error("breaker is closed after 5 records, want open")
	}
}

func TestCircuitBreaker_ClosesAfterWindowExpires(t *testing.T) {
	now := time.Now()
	cb := NewCircuitBreaker(3, 1*time.Minute)
	cb.now = func() time.Time { return now }

	for i := 0; i < 3; i++ {
		cb.Record()
	}

	if !cb.IsOpen() {
		t.Error("breaker should be open after 3 records")
	}

	// Advance time past the window.
	now = now.Add(2 * time.Minute)

	if cb.IsOpen() {
		t.Error("breaker should be closed after window expired")
	}
}

func TestCircuitBreaker_Reset(t *testing.T) {
	cb := NewCircuitBreaker(3, 1*time.Minute)

	for i := 0; i < 3; i++ {
		cb.Record()
	}

	if !cb.IsOpen() {
		t.Error("breaker should be open")
	}

	cb.Reset()

	if cb.IsOpen() {
		t.Error("breaker should be closed after reset")
	}
}

func TestCircuitBreaker_SlidingWindow(t *testing.T) {
	now := time.Now()
	cb := NewCircuitBreaker(3, 1*time.Minute)
	cb.now = func() time.Time { return now }

	// Record 2 events.
	cb.Record()
	cb.Record()

	// Advance time by 50 seconds and record 1 more (total 3 in window, but first 2 are old).
	now = now.Add(50 * time.Second)
	cb.Record()

	if !cb.IsOpen() {
		t.Error("breaker should be open (3 within window)")
	}

	// Advance another 15 seconds — first 2 events fall outside window.
	now = now.Add(15 * time.Second)

	if cb.IsOpen() {
		t.Error("breaker should be closed (only 1 event within window)")
	}
}
