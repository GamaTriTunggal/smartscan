package middleware

import (
	"net/http"
	"net/url"
	"os"

	"github.com/gin-gonic/gin"
)

// MaxRequestBodySize is the maximum allowed request body size (10MB)
// Prevents denial of service through memory exhaustion
const MaxRequestBodySize = 10 << 20 // 10 MB

// SecurityHeaders adds essential security headers to all responses
// OWASP recommended security headers for web applications
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// X-Content-Type-Options: Prevents MIME type sniffing
		c.Header("X-Content-Type-Options", "nosniff")

		// X-Frame-Options: Prevents clickjacking attacks
		// DENY = page cannot be displayed in a frame
		c.Header("X-Frame-Options", "DENY")

		// X-XSS-Protection: Enable XSS filter in browsers (legacy but still useful)
		c.Header("X-XSS-Protection", "1; mode=block")

		// Referrer-Policy: Controls how much referrer info is sent
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// Permissions-Policy: Restrict browser features
		c.Header("Permissions-Policy", "geolocation=(self), microphone=(), camera=()")

		// Content-Security-Policy: Prevents XSS, injection attacks
		// Note: This is a base policy, adjust based on actual resource needs
		csp := "default-src 'self'; " +
			"script-src 'self' 'unsafe-inline' 'unsafe-eval'; " +
			"style-src 'self' 'unsafe-inline'; " +
			"img-src 'self' data: https:; " +
			"font-src 'self' data:; " +
			"connect-src 'self' https:; " +
			"frame-ancestors 'none'; " +
			"base-uri 'self'; " +
			"form-action 'self'"
		c.Header("Content-Security-Policy", csp)

		// Strict-Transport-Security (HSTS): Force HTTPS
		// Only set in production to avoid issues with local development
		if isProduction() {
			// max-age=31536000 = 1 year
			// includeSubDomains = apply to all subdomains
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		c.Next()
	}
}

// isProduction checks if running in production mode
func isProduction() bool {
	env := os.Getenv("APP_ENV")
	return env == "production" || env == "prod"
}

// RequestSizeLimiter limits the size of incoming request bodies
// Prevents denial of service through memory exhaustion attacks
func RequestSizeLimiter() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Limit request body size
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxRequestBodySize)
		c.Next()
	}
}

// CSRFProtection adds CSRF protection for state-changing requests
// Since this API uses JWT tokens in Authorization headers (not cookies),
// CSRF risk is minimal. However, this adds an extra layer of protection
// by requiring the Origin or Referer header to match allowed origins.
func CSRFProtection(allowedOrigins []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Only check state-changing methods
		method := c.Request.Method
		if method == "GET" || method == "HEAD" || method == "OPTIONS" {
			c.Next()
			return
		}

		// Check Origin header first
		origin := c.GetHeader("Origin")
		if origin != "" {
			for _, allowed := range allowedOrigins {
				if origin == allowed {
					c.Next()
					return
				}
			}
			// Origin present but not allowed
			c.Header("X-CSRF-Error", "Invalid origin")
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		// If no Origin, check Referer. Compare the Referer's ORIGIN (scheme+host+port)
		// against the allow-list exactly. A naive prefix match would accept an
		// attacker domain that merely starts with an allowed origin, e.g.
		// "https://app.example.com.evil.com/..." for allowed "https://app.example.com".
		referer := c.GetHeader("Referer")
		if referer != "" {
			if u, err := url.Parse(referer); err == nil && u.Scheme != "" && u.Host != "" {
				refOrigin := u.Scheme + "://" + u.Host
				for _, allowed := range allowedOrigins {
					if refOrigin == allowed {
						c.Next()
						return
					}
				}
			}
			// Referer present but not an allowed origin
			c.Header("X-CSRF-Error", "Invalid referer")
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		// If neither Origin nor Referer is present, check if this is cookie-based auth
		// For cookie-based auth (browser), reject - CSRF risk
		// For non-cookie auth (API clients like Postman/curl), allow
		if _, err := c.Cookie("access_token"); err == nil {
			// Has auth cookie - this is a browser request, require Origin/Referer
			c.Header("X-CSRF-Error", "Missing origin or referer header")
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
		if _, err := c.Cookie("loyalty_access_token"); err == nil {
			// Has loyalty auth cookie - this is a browser request
			c.Header("X-CSRF-Error", "Missing origin or referer header")
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
		if _, err := c.Cookie("distributor_access_token"); err == nil {
			// Has distributor auth cookie - this is a browser request
			c.Header("X-CSRF-Error", "Missing origin or referer header")
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		// No auth cookies - allow for stateless API clients
		c.Next()
	}
}
