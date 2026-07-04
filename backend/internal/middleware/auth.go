package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gamatritunggal/smartscan/backend/internal/config"
	"github.com/gamatritunggal/smartscan/backend/internal/models"
	"github.com/gamatritunggal/smartscan/backend/internal/utils"
	"gorm.io/gorm"
)

// OptionalAuth extracts user info from JWT if present, but doesn't require authentication.
// Useful for routes that can work with or without authentication (e.g., file serving with tenant isolation).
// Sets tenant_id in context if user is authenticated as a tenant.
func OptionalAuth(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenString string

		// Try to get token from cookie first
		cookieToken, err := utils.GetAccessTokenFromCookie(c)
		if err == nil && cookieToken != "" {
			tokenString = cookieToken
		} else {
			// Try Authorization header
			authHeader := c.GetHeader("Authorization")
			if authHeader != "" {
				parts := strings.Split(authHeader, " ")
				if len(parts) == 2 && parts[0] == "Bearer" {
					tokenString = parts[1]
				}
			}
		}

		// If no token found, continue without authentication
		if tokenString == "" {
			c.Next()
			return
		}

		// Validate token - use ValidateAccessToken to prevent refresh tokens being used as access tokens
		claims, err := utils.ValidateAccessToken(tokenString, cfg.JWT.Secret)
		if err != nil {
			// Invalid token - continue without authentication
			c.Next()
			return
		}

		// Check if token has been revoked
		blacklist := utils.NewTokenBlacklist()
		if blacklist.IsRevoked(tokenString) {
			c.Next()
			return
		}

		// Check if all user tokens have been revoked
		if claims.IssuedAt != nil {
			if blacklist.IsUserTokensRevoked(claims.UserID.String(), claims.IssuedAt.Time) {
				c.Next()
				return
			}
		}

		// Set user info in context (if authenticated)
		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)
		c.Set("user_type", claims.UserType)
		c.Set("role", claims.Role)
		if claims.TenantID != nil {
			c.Set("tenant_id", claims.TenantID.String())
		}

		c.Next()
	}
}

func AuthMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenString string

		// Try to get token from cookie first (more secure)
		cookieToken, err := utils.GetAccessTokenFromCookie(c)
		if err == nil && cookieToken != "" {
			tokenString = cookieToken
		} else {
			// Fall back to Authorization header for backward compatibility
			authHeader := c.GetHeader("Authorization")
			if authHeader == "" {
				utils.ErrorResponse(c, http.StatusUnauthorized, "Authorization required", nil)
				c.Abort()
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				utils.ErrorResponse(c, http.StatusUnauthorized, "Invalid authorization header format", nil)
				c.Abort()
				return
			}

			tokenString = parts[1]
		}
		claims, err := utils.ValidateAccessToken(tokenString, cfg.JWT.Secret)
		if err != nil {
			utils.ErrorResponse(c, http.StatusUnauthorized, "Invalid or expired token", err)
			c.Abort()
			return
		}

		// Check if token has been revoked
		blacklist := utils.NewTokenBlacklist()
		if blacklist.IsRevoked(tokenString) {
			utils.ErrorResponse(c, http.StatusUnauthorized, "Token has been revoked", nil)
			c.Abort()
			return
		}

		// Check if all user tokens have been revoked (e.g., password change)
		if claims.IssuedAt != nil {
			if blacklist.IsUserTokensRevoked(claims.UserID.String(), claims.IssuedAt.Time) {
				utils.ErrorResponse(c, http.StatusUnauthorized, "Session expired. Please login again", nil)
				c.Abort()
				return
			}
		}

		// Check inactivity timeout (if enabled)
		if cfg.JWT.InactivityTimeoutMinutes > 0 {
			activityTracker := utils.NewActivityTracker()
			if activityTracker.IsInactive(claims.UserID.String(), cfg.JWT.InactivityTimeoutMinutes) {
				// Clear activity record
				activityTracker.ClearActivity(claims.UserID.String())
				utils.ErrorResponseWithCode(c, http.StatusUnauthorized, "INACTIVITY_TIMEOUT", "Session expired due to inactivity")
				c.Abort()
				return
			}
			// Update last activity timestamp
			activityTracker.UpdateActivity(claims.UserID.String())
		}

		// Set user info in context
		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)
		c.Set("user_type", claims.UserType)
		c.Set("role", claims.Role)
		if claims.TenantID != nil {
			c.Set("tenant_id", claims.TenantID.String())
		}

		// Force password change: an account flagged must_change_password (e.g. after
		// an admin reset) may only reach the change-password endpoint until it
		// rotates the temporary password. Enforced server-side so the requirement
		// can't be bypassed by calling the API directly — the frontend router guard
		// is not a security boundary.
		if claims.MustChangePassword && !strings.HasSuffix(c.FullPath(), "/auth/change-password") {
			utils.ErrorResponseWithCode(c, http.StatusForbidden, "PASSWORD_CHANGE_REQUIRED",
				"You must change your temporary password before continuing")
			c.Abort()
			return
		}

		c.Next()
	}
}



func TenantOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		userType, exists := c.Get("user_type")
		if !exists || userType != "tenant_staff" {
			utils.ErrorResponse(c, http.StatusForbidden, "Access denied: Tenant staff only", nil)
			c.Abort()
			return
		}
		c.Next()
	}
}

func TenantAdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		userType, exists := c.Get("user_type")
		role, roleExists := c.Get("role")
		if !exists || !roleExists || userType != "tenant_staff" || role != "admin" {
			utils.ErrorResponse(c, http.StatusForbidden, "Access denied: Tenant Admin only", nil)
			c.Abort()
			return
		}
		c.Next()
	}
}



// QCStaffOrAdmin allows access for QC staff or tenant admin
func QCStaffOrAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		userType, exists := c.Get("user_type")
		role, roleExists := c.Get("role")
		if !exists || !roleExists || userType != "tenant_staff" {
			utils.ErrorResponse(c, http.StatusForbidden, "Access denied: Tenant staff only", nil)
			c.Abort()
			return
		}

		roleStr := role.(string)
		if roleStr != "admin" && roleStr != "qc_staff" {
			utils.ErrorResponse(c, http.StatusForbidden, "Access denied: QC Staff or Admin only", nil)
			c.Abort()
			return
		}
		c.Next()
	}
}

// WarehouseStaffOrAdmin allows access for warehouse staff or tenant admin
func WarehouseStaffOrAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		userType, exists := c.Get("user_type")
		role, roleExists := c.Get("role")
		if !exists || !roleExists || userType != "tenant_staff" {
			utils.ErrorResponse(c, http.StatusForbidden, "Access denied: Tenant staff only", nil)
			c.Abort()
			return
		}

		roleStr := role.(string)
		if roleStr != "admin" && roleStr != "warehouse_staff" {
			utils.ErrorResponse(c, http.StatusForbidden, "Access denied: Warehouse Staff or Admin only", nil)
			c.Abort()
			return
		}
		c.Next()
	}
}



// SetTenantStaffID middleware looks up the tenant_staff record by user_id
// and sets staff_id in context. Required for handlers that need to track
// which staff member performed an action (e.g., claims, notes).
func SetTenantStaffID(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Only process for tenant_staff users
		userType, exists := c.Get("user_type")
		if !exists || userType != "tenant_staff" {
			c.Next()
			return
		}

		// Get user_id from context (set by AuthMiddleware)
		userIDStr, exists := c.Get("user_id")
		if !exists {
			c.Next()
			return
		}

		var userID uuid.UUID
		switch v := userIDStr.(type) {
		case string:
			parsed, err := uuid.Parse(v)
			if err != nil {
				c.Next()
				return
			}
			userID = parsed
		case uuid.UUID:
			userID = v
		default:
			c.Next()
			return
		}

		// Look up tenant_staff record
		var staff models.TenantStaff
		if err := db.Where("user_id = ?", userID).First(&staff).Error; err != nil {
			// Staff not found - continue without setting staff_id
			// This allows read-only operations to still work
			c.Next()
			return
		}

		// Set staff_id in context
		c.Set("staff_id", staff.ID.String())
		c.Next()
	}
}
