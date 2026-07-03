package utils

import (
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	// Cookie names
	AccessTokenCookie  = "access_token"
	RefreshTokenCookie = "refresh_token"
)

// CookieOptions contains configuration for secure cookies
type CookieOptions struct {
	Domain   string
	Path     string
	MaxAge   int // in seconds
	Secure   bool
	HttpOnly bool
	SameSite http.SameSite
}

// isProduction checks if running in production mode
func isProductionEnv() bool {
	env := os.Getenv("APP_ENV")
	return env == "production" || env == "prod"
}

// getCookieDomain returns the cookie domain based on environment
func getCookieDomain() string {
	// In production, this should be set to your domain
	// In development, empty string works for localhost
	domain := os.Getenv("COOKIE_DOMAIN")
	if domain == "" && isProductionEnv() {
		// In production, if not set, don't restrict to a specific domain
		return ""
	}
	return domain
}

// getDefaultCookieOptions returns secure cookie options based on environment
func getDefaultCookieOptions() CookieOptions {
	return CookieOptions{
		Domain:   getCookieDomain(),
		Path:     "/",
		Secure:   isProductionEnv(), // Only secure in production (requires HTTPS)
		HttpOnly: true,              // Always HttpOnly to prevent XSS access
		SameSite: http.SameSiteLaxMode, // Lax allows top-level navigations
	}
}

// SetAccessTokenCookie sets the access token in an HttpOnly cookie
func SetAccessTokenCookie(c *gin.Context, token string, expirationHours int) {
	opts := getDefaultCookieOptions()
	opts.MaxAge = expirationHours * 3600 // Convert hours to seconds

	c.SetSameSite(opts.SameSite)
	c.SetCookie(
		AccessTokenCookie,
		token,
		opts.MaxAge,
		opts.Path,
		opts.Domain,
		opts.Secure,
		opts.HttpOnly,
	)
}

// SetRefreshTokenCookie sets the refresh token in an HttpOnly cookie
func SetRefreshTokenCookie(c *gin.Context, token string, expirationHours int) {
	opts := getDefaultCookieOptions()
	opts.MaxAge = expirationHours * 3600 // Convert hours to seconds
	// Refresh token path is restricted to auth endpoints for extra security
	opts.Path = "/api/v1/auth"

	c.SetSameSite(opts.SameSite)
	c.SetCookie(
		RefreshTokenCookie,
		token,
		opts.MaxAge,
		opts.Path,
		opts.Domain,
		opts.Secure,
		opts.HttpOnly,
	)
}

// SetTokenCookies sets both access and refresh token cookies
func SetTokenCookies(c *gin.Context, tokenPair *TokenPair, accessExpHours, refreshExpHours int) {
	SetAccessTokenCookie(c, tokenPair.AccessToken, accessExpHours)
	SetRefreshTokenCookie(c, tokenPair.RefreshToken, refreshExpHours)
}

// ClearTokenCookies clears both token cookies (for logout)
func ClearTokenCookies(c *gin.Context) {
	opts := getDefaultCookieOptions()

	// Clear access token cookie
	c.SetSameSite(opts.SameSite)
	c.SetCookie(
		AccessTokenCookie,
		"",
		-1, // Negative MaxAge deletes the cookie
		"/",
		opts.Domain,
		opts.Secure,
		opts.HttpOnly,
	)

	// Clear refresh token cookie
	c.SetSameSite(opts.SameSite)
	c.SetCookie(
		RefreshTokenCookie,
		"",
		-1,
		"/api/v1/auth",
		opts.Domain,
		opts.Secure,
		opts.HttpOnly,
	)
}

// GetAccessTokenFromCookie retrieves the access token from cookie
func GetAccessTokenFromCookie(c *gin.Context) (string, error) {
	return c.Cookie(AccessTokenCookie)
}

// GetRefreshTokenFromCookie retrieves the refresh token from cookie
func GetRefreshTokenFromCookie(c *gin.Context) (string, error) {
	return c.Cookie(RefreshTokenCookie)
}

// TokenPairWithExpiry includes expiration info for frontend
type TokenPairWithExpiry struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	ExpiresIn    int       `json:"expires_in"` // seconds until access token expires
}
