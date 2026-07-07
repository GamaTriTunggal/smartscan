package queue

import (
	"testing"
	"time"
)

// TestCircuitBreaker_NormalRecovery exercises the closed -> open -> half-open ->
// closed lifecycle.
func TestCircuitBreaker_NormalRecovery(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		FailureThreshold: 3,
		SuccessThreshold: 2,
		Timeout:          20 * time.Millisecond,
		HalfOpenMaxCalls: 1,
	})

	// Trip the breaker.
	for i := 0; i < 3; i++ {
		cb.RecordFailure()
	}
	if cb.State() != CircuitOpen {
		t.Fatalf("expected open after %d failures, got %s", 3, cb.State())
	}
	if cb.Allow() {
		t.Fatal("expected requests to be blocked while open")
	}

	// After the timeout, the first probe is allowed and consumes the only slot.
	time.Sleep(30 * time.Millisecond)
	if !cb.Allow() {
		t.Fatal("expected first half-open probe to be allowed")
	}
	if cb.Allow() {
		t.Fatal("expected second concurrent probe to be blocked (HalfOpenMaxCalls=1)")
	}

	// Two successes close the circuit.
	cb.RecordSuccess()
	if !cb.Allow() {
		t.Fatal("expected a fresh probe after the first success released the slot")
	}
	cb.RecordSuccess()
	if cb.State() != CircuitClosed {
		t.Fatalf("expected closed after %d successes, got %s", 2, cb.State())
	}
}

// TestCircuitBreaker_HalfOpenSelfHealsLeakedProbe is the regression guard for the
// permanent-wedge bug: a caller that received a half-open probe token but returned
// without RecordSuccess/RecordFailure (e.g. an empty queue poll) must not pin the
// breaker in half-open forever. After the timeout elapses with the probe
// unresolved, a fresh probe must be handed out. This test FAILS against the old
// implementation, which had no reclaim path.
func TestCircuitBreaker_HalfOpenSelfHealsLeakedProbe(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		FailureThreshold: 3,
		SuccessThreshold: 2,
		Timeout:          20 * time.Millisecond,
		HalfOpenMaxCalls: 1,
	})

	for i := 0; i < 3; i++ {
		cb.RecordFailure()
	}

	// Enter half-open and take the probe slot, then LEAK it (never Record*).
	time.Sleep(30 * time.Millisecond)
	if !cb.Allow() {
		t.Fatal("expected first half-open probe to be allowed")
	}
	if cb.Allow() {
		t.Fatal("expected the single probe slot to be consumed")
	}

	// With the probe unresolved past the timeout, the breaker must reclaim the slot
	// and allow a new probe instead of blocking forever.
	time.Sleep(30 * time.Millisecond)
	if !cb.Allow() {
		t.Fatal("circuit wedged: half-open never reclaimed the leaked probe slot")
	}
}

// TestCircuitBreaker_HalfOpenFailureReopens verifies a failed probe re-opens the
// circuit.
func TestCircuitBreaker_HalfOpenFailureReopens(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		FailureThreshold: 2,
		SuccessThreshold: 2,
		Timeout:          20 * time.Millisecond,
		HalfOpenMaxCalls: 1,
	})
	cb.RecordFailure()
	cb.RecordFailure()
	time.Sleep(30 * time.Millisecond)
	if !cb.Allow() {
		t.Fatal("expected half-open probe")
	}
	cb.RecordFailure()
	if cb.State() != CircuitOpen {
		t.Fatalf("expected re-open after half-open failure, got %s", cb.State())
	}
}
