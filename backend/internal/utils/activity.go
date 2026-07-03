package utils

import (
	"context"
	"fmt"
	"time"

	"github.com/gamatritunggal/smartscan/backend/internal/database"
)

// ActivityTracker handles user activity tracking for inactivity timeout
type ActivityTracker struct{}

// NewActivityTracker creates a new ActivityTracker instance
func NewActivityTracker() *ActivityTracker {
	return &ActivityTracker{}
}

// getActivityKey returns the Redis key for user activity
func (a *ActivityTracker) getActivityKey(userID string) string {
	return fmt.Sprintf("user:activity:%s", userID)
}

// UpdateActivity updates the last activity timestamp for a user
func (a *ActivityTracker) UpdateActivity(userID string) error {
	if database.RedisClient == nil {
		// If Redis is not available, fail-open
		return nil
	}

	ctx := context.Background()
	key := a.getActivityKey(userID)

	// Store current timestamp with 24h TTL (auto-cleanup)
	// TTL is longer than inactivity timeout to allow proper checking
	return database.RedisClient.Set(ctx, key, time.Now().Unix(), 24*time.Hour).Err()
}

// GetLastActivity returns the last activity timestamp for a user
func (a *ActivityTracker) GetLastActivity(userID string) (time.Time, error) {
	if database.RedisClient == nil {
		// If Redis is not available, return current time (no timeout)
		return time.Now(), nil
	}

	ctx := context.Background()
	key := a.getActivityKey(userID)

	timestamp, err := database.RedisClient.Get(ctx, key).Int64()
	if err != nil {
		// No activity record found - treat as new session
		return time.Time{}, err
	}

	return time.Unix(timestamp, 0), nil
}

// IsInactive checks if user has been inactive for longer than the timeout
// Returns true if user is inactive and should be logged out
func (a *ActivityTracker) IsInactive(userID string, timeoutMinutes int) bool {
	if database.RedisClient == nil {
		// If Redis is not available, fail-open (not inactive)
		return false
	}

	lastActivity, err := a.GetLastActivity(userID)
	if err != nil {
		// No activity record - this is first request after login
		// Don't mark as inactive, let middleware update activity
		return false
	}

	inactiveDuration := time.Since(lastActivity)
	timeoutDuration := time.Duration(timeoutMinutes) * time.Minute

	return inactiveDuration > timeoutDuration
}

// ClearActivity removes the activity record for a user (on logout)
func (a *ActivityTracker) ClearActivity(userID string) error {
	if database.RedisClient == nil {
		return nil
	}

	ctx := context.Background()
	key := a.getActivityKey(userID)

	return database.RedisClient.Del(ctx, key).Err()
}
