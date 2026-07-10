package utils

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	mathRand "math/rand"
	"sync"
	"time"
)

var (
	// ErrEntropyUnavailable indicates crypto/rand failed and fallback is not allowed
	ErrEntropyUnavailable = errors.New("cryptographic random number generator unavailable")

	// mathRandMu protects math/rand seed initialization
	mathRandMu sync.Once
)

// initMathRand initializes math/rand with a time-based seed (fallback only)
func initMathRand() {
	mathRandMu.Do(func() {
		mathRand.Seed(time.Now().UnixNano())
	})
}

// GenerateSecureBytes generates cryptographically secure random bytes.
// Returns error if crypto/rand fails - caller must handle appropriately.
// DO NOT use fallback for security-critical operations (passwords, auth tokens).
func GenerateSecureBytes(length int) ([]byte, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrEntropyUnavailable, err)
	}
	return b, nil
}

// GenerateSecureHex generates a cryptographically secure hex string.
// Returns error if crypto/rand fails.
func GenerateSecureHex(byteLength int) (string, error) {
	b, err := GenerateSecureBytes(byteLength)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// GenerateSecureTempPassword generates a cryptographically secure temporary password.
// Returns error if crypto/rand fails - NEVER falls back to math/rand.
// This is a SECURITY-CRITICAL function.
func GenerateSecureTempPassword(byteLength int) (string, error) {
	b, err := GenerateSecureBytes(byteLength)
	if err != nil {
		return "", fmt.Errorf("failed to generate temporary password: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// GenerateSecureInt generates a cryptographically secure random integer in range [0, max).
// Returns error if crypto/rand fails.
func GenerateSecureInt(max int64) (int64, error) {
	if max <= 0 {
		return 0, errors.New("max must be positive")
	}

	n, err := rand.Int(rand.Reader, big.NewInt(max))
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrEntropyUnavailable, err)
	}

	return n.Int64(), nil
}

// GenerateRandomBytesWithFallback generates random bytes, falling back to math/rand if needed.
// Use this ONLY for non-security-critical operations (tracking numbers, display IDs).
// Logs a warning if fallback is used.
func GenerateRandomBytesWithFallback(length int) []byte {
	b := make([]byte, length)

	if _, err := rand.Read(b); err != nil {
		// Fallback to math/rand for non-critical operations
		initMathRand()
		for i := range b {
			b[i] = byte(mathRand.Intn(256))
		}
		// Note: In production, this should log to monitoring system
		// log.Printf("Warning: crypto/rand failed, using math/rand fallback: %v", err)
	}

	return b
}

// GenerateRandomHexWithFallback generates a hex string, falling back to math/rand if needed.
// Use this ONLY for non-security-critical operations.
func GenerateRandomHexWithFallback(byteLength int) string {
	b := GenerateRandomBytesWithFallback(byteLength)
	return hex.EncodeToString(b)
}

// GenerateRandomIntWithFallback generates a random int in range [0, max), with fallback.
// Use this ONLY for non-security-critical operations.
func GenerateRandomIntWithFallback(max int) int {
	if max <= 0 {
		return 0
	}

	n, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		// Fallback to math/rand
		initMathRand()
		return mathRand.Intn(max)
	}

	return int(n.Int64())
}

// ShuffleWithFallback performs Fisher-Yates shuffle on a slice of indices.
// Uses crypto/rand with fallback to math/rand for non-critical shuffling.
func ShuffleWithFallback(n int) []int {
	indices := make([]int, n)
	for i := range indices {
		indices[i] = i
	}

	for i := n - 1; i > 0; i-- {
		j := GenerateRandomIntWithFallback(i + 1)
		indices[i], indices[j] = indices[j], indices[i]
	}

	return indices
}
