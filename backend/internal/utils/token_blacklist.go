package utils

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/gamatritunggal/smartscan/backend/internal/database"
)

// TokenBlacklist handles JWT token revocation using Redis
type TokenBlacklist struct{}

// NewTokenBlacklist creates a new TokenBlacklist instance
func NewTokenBlacklist() *TokenBlacklist {
	return &TokenBlacklist{}
}

// getBlacklistKey returns the Redis key for a blacklisted token
// Uses SHA256 hash of the token to avoid storing the full token
func (t *TokenBlacklist) getBlacklistKey(token string) string {
	hash := sha256.Sum256([]byte(token))
	return fmt.Sprintf("token:blacklist:%s", hex.EncodeToString(hash[:]))
}

// RevokeToken adds a token to the blacklist
// The token will remain blacklisted until its original expiry time
func (t *TokenBlacklist) RevokeToken(token string, expiresAt time.Time) error {
	if database.RedisClient == nil {
		// If Redis is not available, fail-open (allow the operation)
		return nil
	}

	ctx := context.Background()
	key := t.getBlacklistKey(token)

	// Calculate TTL based on token's expiry time
	ttl := time.Until(expiresAt)
	if ttl <= 0 {
		// Token already expired, no need to blacklist
		return nil
	}

	// Store in Redis with expiry
	return database.RedisClient.Set(ctx, key, "revoked", ttl).Err()
}

// IsRevoked checks if a token has been revoked
func (t *TokenBlacklist) IsRevoked(token string) bool {
	if database.RedisClient == nil {
		// If Redis is not available, fail-open (assume not revoked)
		return false
	}

	ctx := context.Background()
	key := t.getBlacklistKey(token)

	exists, err := database.RedisClient.Exists(ctx, key).Result()
	if err != nil {
		// On error, fail-open
		return false
	}

	return exists > 0
}

// RevokeUserTokens revokes all tokens for a specific user
// This is useful when a user changes password or logs out from all devices
func (t *TokenBlacklist) RevokeUserTokens(userID string, expiry time.Duration) error {
	if database.RedisClient == nil {
		return nil
	}

	ctx := context.Background()
	key := fmt.Sprintf("token:user_revoked:%s", userID)

	// Store the revocation timestamp
	return database.RedisClient.Set(ctx, key, time.Now().Unix(), expiry).Err()
}

// IsUserTokensRevoked checks if all tokens for a user have been revoked
// This checks if a global revocation was issued after the token was created
func (t *TokenBlacklist) IsUserTokensRevoked(userID string, tokenIssuedAt time.Time) bool {
	if database.RedisClient == nil {
		return false
	}

	ctx := context.Background()
	key := fmt.Sprintf("token:user_revoked:%s", userID)

	revocationTime, err := database.RedisClient.Get(ctx, key).Int64()
	if err != nil {
		// No revocation record found
		return false
	}

	// If revocation happened after token was issued, token is invalid
	return revocationTime > tokenIssuedAt.Unix()
}
