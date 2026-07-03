package utils

import (
	"context"
	"fmt"
	"time"

	"github.com/gamatritunggal/smartscan/backend/internal/database"
)

// Account lockout constants
const (
	// MaxLoginAttempts before lockout
	MaxLoginAttempts = 5
	// LockoutDuration after max attempts
	LockoutDuration = 15 * time.Minute
	// LoginAttemptWindow for counting failed attempts
	LoginAttemptWindow = 15 * time.Minute
)

// AccountLockout handles login attempt tracking and account lockout
type AccountLockout struct{}

// NewAccountLockout creates a new AccountLockout instance
func NewAccountLockout() *AccountLockout {
	return &AccountLockout{}
}

// getAttemptKey returns the Redis key for tracking login attempts
func (a *AccountLockout) getAttemptKey(email string) string {
	return fmt.Sprintf("login:attempts:%s", email)
}

// getLockoutKey returns the Redis key for lockout status
func (a *AccountLockout) getLockoutKey(email string) string {
	return fmt.Sprintf("login:lockout:%s", email)
}

// IsLocked checks if an account is currently locked
func (a *AccountLockout) IsLocked(email string) (bool, time.Duration) {
	if database.RedisClient == nil {
		return false, 0
	}

	ctx := context.Background()
	lockoutKey := a.getLockoutKey(email)

	ttl, err := database.RedisClient.TTL(ctx, lockoutKey).Result()
	if err != nil || ttl <= 0 {
		return false, 0
	}

	return true, ttl
}

// RecordFailedAttempt records a failed login attempt
// Returns true if account is now locked, along with remaining attempts
func (a *AccountLockout) RecordFailedAttempt(email string) (isLocked bool, remainingAttempts int) {
	if database.RedisClient == nil {
		return false, MaxLoginAttempts
	}

	ctx := context.Background()
	attemptKey := a.getAttemptKey(email)
	lockoutKey := a.getLockoutKey(email)

	// Increment attempt counter
	count, err := database.RedisClient.Incr(ctx, attemptKey).Result()
	if err != nil {
		return false, MaxLoginAttempts
	}

	// Set expiry on first attempt
	if count == 1 {
		database.RedisClient.Expire(ctx, attemptKey, LoginAttemptWindow)
	}

	remaining := MaxLoginAttempts - int(count)
	if remaining < 0 {
		remaining = 0
	}

	// Check if max attempts reached
	if count >= int64(MaxLoginAttempts) {
		// Lock the account
		database.RedisClient.Set(ctx, lockoutKey, "locked", LockoutDuration)
		// Clear the attempt counter
		database.RedisClient.Del(ctx, attemptKey)
		return true, 0
	}

	return false, remaining
}

// ClearAttempts clears failed login attempts (called on successful login)
func (a *AccountLockout) ClearAttempts(email string) {
	if database.RedisClient == nil {
		return
	}

	ctx := context.Background()
	attemptKey := a.getAttemptKey(email)
	database.RedisClient.Del(ctx, attemptKey)
}

// GetRemainingAttempts returns the number of remaining login attempts
func (a *AccountLockout) GetRemainingAttempts(email string) int {
	if database.RedisClient == nil {
		return MaxLoginAttempts
	}

	ctx := context.Background()
	attemptKey := a.getAttemptKey(email)

	count, err := database.RedisClient.Get(ctx, attemptKey).Int()
	if err != nil {
		return MaxLoginAttempts
	}

	remaining := MaxLoginAttempts - count
	if remaining < 0 {
		return 0
	}
	return remaining
}
