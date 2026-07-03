package middleware

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

// AuditLogger logs security-sensitive operations
// This provides an audit trail for compliance and security monitoring
func AuditLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Start timer
		startTime := time.Now()

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(startTime)

		// Get request info
		clientIP := c.ClientIP()
		method := c.Request.Method
		path := c.Request.URL.Path
		statusCode := c.Writer.Status()
		userAgent := c.Request.UserAgent()

		// Get user info if authenticated
		userID, _ := c.Get("user_id")
		email, _ := c.Get("email")
		userType, _ := c.Get("user_type")

		// Log security-sensitive endpoints
		if isSensitiveEndpoint(path, method) {
			log.Printf("[AUDIT] %s | %d | %s | %s | %13v | user_id=%v | email=%v | user_type=%v | ua=%s",
				method,
				statusCode,
				clientIP,
				path,
				latency,
				userID,
				email,
				userType,
				truncateString(userAgent, 50),
			)
		}

		// Log failed authentication attempts
		if statusCode == 401 || statusCode == 403 {
			log.Printf("[SECURITY] Auth failure | %d | %s | %s | user_id=%v | email=%v",
				statusCode,
				clientIP,
				path,
				userID,
				email,
			)
		}

		// Log rate limit violations
		if statusCode == 429 {
			log.Printf("[SECURITY] Rate limit exceeded | %s | %s | user_id=%v",
				clientIP,
				path,
				userID,
			)
		}
	}
}

// Path prefix constants for audit logging
const (
	tenantPathPrefix = "/api/v1/tenant/"
	tenantPathLen    = len(tenantPathPrefix)
)

// isSensitiveEndpoint checks if an endpoint is security-sensitive
func isSensitiveEndpoint(path, method string) bool {
	// Authentication endpoints
	if path == "/api/v1/auth/login" || path == "/api/v1/auth/refresh" || path == "/api/v1/setup" {
		return true
	}

	// Tenant admin operations
	if len(path) > tenantPathLen && path[:tenantPathLen] == tenantPathPrefix {
		if method == "POST" || method == "DELETE" {
			return true
		}
	}

	return false
}

// truncateString truncates a string to maxLen characters
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
