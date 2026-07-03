package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/gamatritunggal/smartscan/backend/internal/database"
	"github.com/gamatritunggal/smartscan/backend/internal/utils"
)

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	// Requests per window
	Requests int
	// Window duration
	Window time.Duration
	// Key prefix for Redis
	KeyPrefix string
}

// Default rate limit configurations
var (
	// GeneralRateLimit for most API endpoints
	GeneralRateLimit = RateLimitConfig{
		Requests:  100,
		Window:    time.Minute,
		KeyPrefix: "rl:general",
	}

	// AuthRateLimit for login/register endpoints (stricter)
	AuthRateLimit = RateLimitConfig{
		Requests:  10,
		Window:    time.Minute,
		KeyPrefix: "rl:auth",
	}

	// PublicRateLimit for public endpoints
	PublicRateLimit = RateLimitConfig{
		Requests:  60,
		Window:    time.Minute,
		KeyPrefix: "rl:public",
	}

	// ScanRateLimit for QR scan redirect endpoint (stricter to prevent abuse)
	ScanRateLimit = RateLimitConfig{
		Requests:  30,
		Window:    time.Minute,
		KeyPrefix: "rl:scan",
	}

	// ExportRateLimit for CSV/Excel export endpoints (prevent abuse)
	ExportRateLimit = RateLimitConfig{
		Requests:  5,
		Window:    time.Minute,
		KeyPrefix: "rl:export",
	}
)

// RateLimiter creates a rate limiting middleware using Redis
// Falls back to allowing requests if Redis is unavailable (fail-open)
func RateLimiter(config RateLimitConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip if Redis is not available (fail-open for development)
		if database.RedisClient == nil {
			c.Next()
			return
		}

		// Get client identifier (IP address)
		clientIP := c.ClientIP()
		key := fmt.Sprintf("%s:%s", config.KeyPrefix, clientIP)

		ctx := context.Background()

		// Use Lua script for atomic INCR + EXPIRE to prevent race conditions
		// Script increments counter and sets expiry atomically
		script := redis.NewScript(`
			local count = redis.call("INCR", KEYS[1])
			if count == 1 then
				redis.call("EXPIRE", KEYS[1], ARGV[1])
			end
			return count
		`)

		windowSeconds := int(config.Window.Seconds())
		count, err := script.Run(ctx, database.RedisClient, []string{key}, windowSeconds).Int64()
		if err != nil {
			// Redis error - fail open (allow request)
			c.Next()
			return
		}

		// Check if limit exceeded
		if count > int64(config.Requests) {
			// Get TTL to inform client when to retry
			ttl, _ := database.RedisClient.TTL(ctx, key).Result()

			c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", config.Requests))
			c.Header("X-RateLimit-Remaining", "0")
			c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(ttl).Unix()))
			c.Header("Retry-After", fmt.Sprintf("%d", int(ttl.Seconds())))

			utils.ErrorResponse(c, http.StatusTooManyRequests, "Rate limit exceeded. Please try again later.", nil)
			c.Abort()
			return
		}

		// Set rate limit headers
		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", config.Requests))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", config.Requests-int(count)))

		c.Next()
	}
}

// RateLimiterByUserID creates a rate limiter keyed by user ID (for authenticated endpoints)
func RateLimiterByUserID(config RateLimitConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip if Redis is not available
		if database.RedisClient == nil {
			c.Next()
			return
		}

		// Get user ID from context (set by auth middleware)
		userID, exists := c.Get("user_id")
		if !exists {
			// Fall back to IP-based limiting if not authenticated
			RateLimiter(config)(c)
			return
		}

		key := fmt.Sprintf("%s:user:%v", config.KeyPrefix, userID)

		ctx := context.Background()

		count, err := database.RedisClient.Incr(ctx, key).Result()
		if err != nil {
			c.Next()
			return
		}

		if count == 1 {
			database.RedisClient.Expire(ctx, key, config.Window)
		}

		if count > int64(config.Requests) {
			ttl, _ := database.RedisClient.TTL(ctx, key).Result()

			c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", config.Requests))
			c.Header("X-RateLimit-Remaining", "0")
			c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(ttl).Unix()))
			c.Header("Retry-After", fmt.Sprintf("%d", int(ttl.Seconds())))

			utils.ErrorResponse(c, http.StatusTooManyRequests, "Rate limit exceeded. Please try again later.", nil)
			c.Abort()
			return
		}

		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", config.Requests))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", config.Requests-int(count)))

		c.Next()
	}
}
