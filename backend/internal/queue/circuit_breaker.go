package queue

import (
	"sync"
	"time"
)

// CircuitState represents the state of the circuit breaker
type CircuitState string

const (
	CircuitClosed   CircuitState = "closed"    // Normal operation
	CircuitOpen     CircuitState = "open"      // Blocking all requests
	CircuitHalfOpen CircuitState = "half-open" // Testing if service recovered
)

// CircuitBreaker implements the circuit breaker pattern for SMTP protection
type CircuitBreaker struct {
	mu sync.RWMutex

	state       CircuitState
	failures    int
	successes   int
	lastFailure time.Time
	lastSuccess time.Time
	lastProbeAt time.Time // when the most recent half-open probe slot was handed out

	// Configuration
	failureThreshold int           // Number of failures to open circuit
	successThreshold int           // Number of successes in half-open to close
	timeout          time.Duration // Time to wait before half-open
	halfOpenMaxCalls int           // Max concurrent calls in half-open state
	halfOpenCalls    int           // Current calls in half-open state
}

// CircuitBreakerConfig holds configuration for circuit breaker
type CircuitBreakerConfig struct {
	FailureThreshold int           // Default: 5
	SuccessThreshold int           // Default: 2
	Timeout          time.Duration // Default: 1 minute
	HalfOpenMaxCalls int           // Default: 1
}

// DefaultCircuitBreakerConfig returns default configuration
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		FailureThreshold: 5,
		SuccessThreshold: 2,
		Timeout:          time.Minute,
		HalfOpenMaxCalls: 1,
	}
}

// NewCircuitBreaker creates a new circuit breaker with the given config
func NewCircuitBreaker(cfg CircuitBreakerConfig) *CircuitBreaker {
	if cfg.FailureThreshold <= 0 {
		cfg.FailureThreshold = 5
	}
	if cfg.SuccessThreshold <= 0 {
		cfg.SuccessThreshold = 2
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = time.Minute
	}
	if cfg.HalfOpenMaxCalls <= 0 {
		cfg.HalfOpenMaxCalls = 1
	}

	return &CircuitBreaker{
		state:            CircuitClosed,
		failureThreshold: cfg.FailureThreshold,
		successThreshold: cfg.SuccessThreshold,
		timeout:          cfg.Timeout,
		halfOpenMaxCalls: cfg.HalfOpenMaxCalls,
	}
}

// Allow checks if a request should be allowed through
// Returns true if allowed, false if circuit is open
func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitClosed:
		return true

	case CircuitOpen:
		// Check if timeout has passed to transition to half-open
		if time.Since(cb.lastFailure) > cb.timeout {
			cb.state = CircuitHalfOpen
			cb.halfOpenCalls = 1 // this call consumes the first probe slot
			cb.successes = 0
			cb.lastProbeAt = time.Now()
			return true
		}
		return false

	case CircuitHalfOpen:
		// Self-heal against leaked probe slots: a caller that received a probe
		// token but returned without RecordSuccess/RecordFailure (e.g. an empty
		// queue poll or a transient dequeue error) would otherwise leave
		// halfOpenCalls pinned at the max forever, wedging the breaker in
		// half-open and permanently blocking recovery. If the outstanding probe
		// has not resolved within the timeout window, reclaim the slot so a fresh
		// probe can proceed.
		if cb.halfOpenCalls >= cb.halfOpenMaxCalls && time.Since(cb.lastProbeAt) > cb.timeout {
			cb.halfOpenCalls = 0
		}
		// Allow limited calls in half-open state
		if cb.halfOpenCalls < cb.halfOpenMaxCalls {
			cb.halfOpenCalls++
			cb.lastProbeAt = time.Now()
			return true
		}
		return false
	}

	return false
}

// RecordSuccess records a successful operation
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.lastSuccess = time.Now()

	switch cb.state {
	case CircuitHalfOpen:
		cb.successes++
		if cb.halfOpenCalls > 0 {
			cb.halfOpenCalls--
		}
		// If enough successes, close the circuit
		if cb.successes >= cb.successThreshold {
			cb.state = CircuitClosed
			cb.failures = 0
			cb.successes = 0
		}

	case CircuitClosed:
		// Reset failure count on success
		cb.failures = 0
	}
}

// RecordFailure records a failed operation
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.lastFailure = time.Now()
	cb.failures++

	switch cb.state {
	case CircuitClosed:
		// If failures exceed threshold, open the circuit
		if cb.failures >= cb.failureThreshold {
			cb.state = CircuitOpen
		}

	case CircuitHalfOpen:
		// Any failure in half-open returns to open
		cb.state = CircuitOpen
		cb.halfOpenCalls = 0
	}
}

// State returns the current state of the circuit breaker
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Stats returns current circuit breaker statistics
func (cb *CircuitBreaker) Stats() CircuitBreakerStats {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return CircuitBreakerStats{
		State:           string(cb.state),
		Failures:        cb.failures,
		Successes:       cb.successes,
		LastFailure:     cb.lastFailure,
		LastSuccess:     cb.lastSuccess,
		TimeUntilRetry:  cb.timeUntilRetry(),
		FailureThreshold: cb.failureThreshold,
	}
}

// timeUntilRetry returns duration until next retry attempt (only meaningful when open)
func (cb *CircuitBreaker) timeUntilRetry() time.Duration {
	if cb.state != CircuitOpen {
		return 0
	}
	remaining := cb.timeout - time.Since(cb.lastFailure)
	if remaining < 0 {
		return 0
	}
	return remaining
}

// Reset manually resets the circuit breaker to closed state
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.state = CircuitClosed
	cb.failures = 0
	cb.successes = 0
	cb.halfOpenCalls = 0
}

// IsOpen returns true if the circuit is open (blocking requests)
func (cb *CircuitBreaker) IsOpen() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state == CircuitOpen
}

// CircuitBreakerStats holds statistics for monitoring
type CircuitBreakerStats struct {
	State            string        `json:"state"`
	Failures         int           `json:"failures"`
	Successes        int           `json:"successes"`
	LastFailure      time.Time     `json:"last_failure"`
	LastSuccess      time.Time     `json:"last_success"`
	TimeUntilRetry   time.Duration `json:"time_until_retry"`
	FailureThreshold int           `json:"failure_threshold"`
}
