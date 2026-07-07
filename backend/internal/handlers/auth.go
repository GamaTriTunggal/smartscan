package handlers

import (
	"time"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gamatritunggal/smartscan/backend/internal/config"
	"github.com/gamatritunggal/smartscan/backend/internal/models"
	"github.com/gamatritunggal/smartscan/backend/internal/sentry"
	"github.com/gamatritunggal/smartscan/backend/internal/services/audit"
	"github.com/gamatritunggal/smartscan/backend/internal/utils"
	"gorm.io/gorm"
)

type AuthHandler struct {
	DB  *gorm.DB
	Cfg *config.Config
}

func NewAuthHandler(db *gorm.DB, cfg *config.Config) *AuthHandler {
	return &AuthHandler{DB: db, Cfg: cfg}
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email,max=255"`
	Password string `json:"password" binding:"required,min=6,max=128"`
}

type LoginResponse struct {
	User                 UserResponse                  `json:"user"`
	TokenPair            *utils.TokenPair              `json:"tokens,omitempty"` // Only included if not using cookies
	ExpiresIn            int                           `json:"expires_in"`       // seconds until access token expires
}

type UserResponse struct {
	ID                 uuid.UUID  `json:"id"`
	Email              string     `json:"email"`
	UserType           string     `json:"user_type"`
	Role               string     `json:"role"`
	FullName           string     `json:"full_name"`
	TenantID           *uuid.UUID `json:"tenant_id,omitempty"`
	TenantName         string     `json:"tenant_name,omitempty"`
	MustChangePassword bool       `json:"must_change_password"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sentry.CaptureHandlerError(c, err, "auth.Login", sentry.ErrorTypeValidation, sentry.SeverityLow)
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request", err)
		return
	}

	// Normalize email for case-insensitive matching (handles Gmail dots, +suffix for all providers)
	req.Email = utils.NormalizeEmail(req.Email)

	// Check if account is locked
	lockout := utils.NewAccountLockout()
	if isLocked, remainingTime := lockout.IsLocked(req.Email); isLocked {
		// Calculate remaining minutes with ceiling (e.g., 14m30s -> 15, 30s -> 1)
		remainingMinutes := int(remainingTime.Minutes())
		if remainingTime.Seconds()-float64(remainingMinutes*60) > 0 {
			remainingMinutes++ // Round up if there are remaining seconds
		}
		if remainingMinutes < 1 {
			remainingMinutes = 1 // Show at least 1 minute
		}
		utils.ErrorResponse(c, http.StatusTooManyRequests,
			fmt.Sprintf("Account is temporarily locked. Try again in %d minutes", remainingMinutes), nil)
		return
	}

	// Find user
	var user models.User
	if err := h.DB.Where("email = ? AND status = ?", req.Email, "active").First(&user).Error; err != nil {
		// Account not found (or inactive). Normalize timing with a dummy bcrypt
		// comparison and record the failed attempt so lockout also applies to
		// unknown emails. Return the SAME response as the wrong-password branch so
		// the two cases are indistinguishable (prevents user enumeration — see the
		// matching branch below).
		utils.CheckDummyPassword(req.Password)
		lockout.RecordFailedAttempt(req.Email)
		utils.ErrorResponse(c, http.StatusUnauthorized, "Invalid email or password", nil)
		return
	}

	// Check password
	if !utils.CheckPassword(req.Password, user.PasswordHash) {
		// Record the failed attempt for lockout, but return the IDENTICAL uniform
		// response as the account-not-found branch above — no per-attempt counter
		// and no distinct lockout status here, both of which would otherwise reveal
		// that the email belongs to a real account. Once the attempt threshold is
		// crossed the account is locked internally; the uniform "temporarily locked"
		// message is then emitted by the IsLocked check at the top of this handler
		// on the next attempt, the same way for existent and non-existent emails.
		lockout.RecordFailedAttempt(req.Email)
		utils.ErrorResponse(c, http.StatusUnauthorized, "Invalid email or password", nil)
		return
	}

	// Clear failed attempts on successful login
	lockout.ClearAttempts(req.Email)

	// Get user details based on type
	var role, fullName string
	var tenantID *uuid.UUID
	var tenantName string

	if user.UserType == models.UserTypeTenantStaff {
		var staff models.TenantStaff
		if err := h.DB.Preload("Tenant").Where("user_id = ?", user.ID).First(&staff).Error; err != nil {
			sentry.CaptureHandlerError(c, err, "auth.Login", sentry.ErrorTypeDatabase, sentry.SeverityHigh)
			utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to get user details", err)
			return
		}
		role = string(staff.Role)
		fullName = staff.FullName
		tenantID = &staff.TenantID
		if staff.Tenant != nil {
			tenantName = staff.Tenant.CompanyName
		}
	}

	// Generate tokens
	tokenPair, err := utils.GenerateTokenPair(
		h.Cfg.JWT.Secret,
		user.ID,
		user.Email,
		string(user.UserType),
		role,
		tenantID,
		h.Cfg.JWT.ExpirationHours,
		h.Cfg.JWT.RefreshHours,
		user.MustChangePassword,
	)
	if err != nil {
		sentry.CaptureHandlerError(c, err, "auth.Login", sentry.ErrorTypeInternal, sentry.SeverityCritical)
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to generate token", err)
		return
	}

	// Set HttpOnly cookies for tokens
	utils.SetTokenCookies(c, tokenPair, h.Cfg.JWT.ExpirationHours, h.Cfg.JWT.RefreshHours)

	// Initialize activity tracking for inactivity timeout
	activityTracker := utils.NewActivityTracker()
	activityTracker.UpdateActivity(user.ID.String())

	// Audit log
	audit.Log(h.DB, audit.Entry{
		UserID:     &user.ID,
		TenantID:   tenantID,
		Action:     models.ActionTypeLogin,
		EntityType: "user",
		EntityID:   &user.ID,
		IPAddress:  c.ClientIP(),
		UserAgent:  c.Request.UserAgent(),
	})

	utils.SuccessResponse(c, http.StatusOK, "Login successful", LoginResponse{
		User: UserResponse{
			ID:                 user.ID,
			Email:              user.Email,
			UserType:           string(user.UserType),
			Role:               role,
			FullName:           fullName,
			TenantID:           tenantID,
			TenantName:         tenantName,
			MustChangePassword: user.MustChangePassword,
		},
		TokenPair:            tokenPair, // Still include for backward compatibility
		ExpiresIn:            h.Cfg.JWT.ExpirationHours * 3600,
	})
}

func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var refreshToken string

	// Try to get refresh token from cookie first
	cookieToken, err := utils.GetRefreshTokenFromCookie(c)
	if err == nil && cookieToken != "" {
		refreshToken = cookieToken
	} else {
		// Fall back to request body for backward compatibility
		var req struct {
			RefreshToken string `json:"refresh_token"`
		}
		if err := c.ShouldBindJSON(&req); err != nil || req.RefreshToken == "" {
			sentry.CaptureHandlerError(c, fmt.Errorf("refresh token required"), "auth.RefreshToken", sentry.ErrorTypeValidation, sentry.SeverityLow)
			utils.ErrorResponse(c, http.StatusBadRequest, "Refresh token required", nil)
			return
		}
		refreshToken = req.RefreshToken
	}

	claims, err := utils.ValidateRefreshToken(refreshToken, h.Cfg.JWT.Secret)
	if err != nil {
		sentry.CaptureHandlerError(c, err, "auth.RefreshToken", sentry.ErrorTypeAuth, sentry.SeverityMedium)
		utils.ErrorResponse(c, http.StatusUnauthorized, "Invalid refresh token", err)
		return
	}

	// Honor revocation on the refresh path too. AuthMiddleware enforces this for
	// access tokens; without the same checks here a password reset / forced
	// logout could be defeated by minting a fresh access token from an old
	// refresh token. Fail-open when Redis is unavailable (matches middleware).
	blacklist := utils.NewTokenBlacklist()
	if blacklist.IsRevoked(refreshToken) {
		utils.ErrorResponse(c, http.StatusUnauthorized, "Refresh token has been revoked", nil)
		return
	}
	if claims.IssuedAt != nil && blacklist.IsUserTokensRevoked(claims.UserID.String(), claims.IssuedAt.Time) {
		utils.ErrorResponse(c, http.StatusUnauthorized, "Session has been revoked — please sign in again", nil)
		return
	}

	// Check inactivity timeout before allowing refresh
	if h.Cfg.JWT.InactivityTimeoutMinutes > 0 {
		activityTracker := utils.NewActivityTracker()
		if activityTracker.IsInactive(claims.UserID.String(), h.Cfg.JWT.InactivityTimeoutMinutes) {
			// User has been inactive too long, clear activity and reject refresh
			activityTracker.ClearActivity(claims.UserID.String())
			utils.ErrorResponseWithCode(c, http.StatusUnauthorized, "INACTIVITY_TIMEOUT",
				"Session expired due to inactivity")
			return
		}
		// Update activity since user is actively refreshing their session
		activityTracker.UpdateActivity(claims.UserID.String())
	}

	// Check if user still exists and active
	var user models.User
	if err := h.DB.Where("id = ? AND status = ?", claims.UserID, "active").First(&user).Error; err != nil {
		sentry.CaptureHandlerError(c, err, "auth.RefreshToken", sentry.ErrorTypeAuth, sentry.SeverityMedium)
		utils.ErrorResponse(c, http.StatusUnauthorized, "User not found or inactive", nil)
		return
	}

	// Generate new tokens
	tokenPair, err := utils.GenerateTokenPair(
		h.Cfg.JWT.Secret,
		claims.UserID,
		claims.Email,
		claims.UserType,
		claims.Role,
		claims.TenantID,
		h.Cfg.JWT.ExpirationHours,
		h.Cfg.JWT.RefreshHours,
		user.MustChangePassword, // re-read from DB so a cleared/set flag propagates on refresh
	)
	if err != nil {
		sentry.CaptureHandlerError(c, err, "auth.RefreshToken", sentry.ErrorTypeInternal, sentry.SeverityCritical)
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to generate token", err)
		return
	}

	// Set HttpOnly cookies for new tokens
	utils.SetTokenCookies(c, tokenPair, h.Cfg.JWT.ExpirationHours, h.Cfg.JWT.RefreshHours)

	// Response with tokens and expiry info
	utils.SuccessResponse(c, http.StatusOK, "Token refreshed", gin.H{
		"access_token":  tokenPair.AccessToken, // For backward compatibility
		"refresh_token": tokenPair.RefreshToken,
		"expires_in":    h.Cfg.JWT.ExpirationHours * 3600,
	})
}

// Logout clears authentication cookies and revokes the token
func (h *AuthHandler) Logout(c *gin.Context) {
	// Try to get access token to revoke it
	accessToken, _ := utils.GetAccessTokenFromCookie(c)
	if accessToken == "" {
		// Try Authorization header as fallback
		authHeader := c.GetHeader("Authorization")
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			accessToken = authHeader[7:]
		}
	}

	// Revoke the token if we found one
	if accessToken != "" {
		claims, err := utils.ValidateToken(accessToken, h.Cfg.JWT.Secret)
		if err == nil {
			// Clear activity tracking for inactivity timeout
			activityTracker := utils.NewActivityTracker()
			activityTracker.ClearActivity(claims.UserID.String())

			// Revoke the token
			if claims.ExpiresAt != nil {
				blacklist := utils.NewTokenBlacklist()
				blacklist.RevokeToken(accessToken, claims.ExpiresAt.Time)
			}
		}
	}

	// Also revoke the refresh token. Blacklisting the access token alone is not
	// enough: the refresh token outlives it (default 168h) and could mint a fresh
	// access token via /auth/refresh, so the session would survive logout.
	refreshToken, _ := utils.GetRefreshTokenFromCookie(c)
	if refreshToken == "" {
		var body struct {
			RefreshToken string `json:"refresh_token"`
		}
		if err := c.ShouldBindJSON(&body); err == nil {
			refreshToken = body.RefreshToken
		}
	}
	if refreshToken != "" {
		if claims, err := utils.ValidateRefreshToken(refreshToken, h.Cfg.JWT.Secret); err == nil && claims.ExpiresAt != nil {
			utils.NewTokenBlacklist().RevokeToken(refreshToken, claims.ExpiresAt.Time)
		}
	}

	// Clear cookies
	utils.ClearTokenCookies(c)

	// Audit log
	audit.LogFromContext(c, h.DB, models.ActionTypeLogout, "user", nil, nil, nil)

	utils.SuccessResponse(c, http.StatusOK, "Logged out successfully", nil)
}

func (h *AuthHandler) GetMe(c *gin.Context) {
	userUUID, ok := utils.GetUserUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "Invalid user context", nil)
		return
	}

	var user models.User
	if err := h.DB.Where("id = ?", userUUID).First(&user).Error; err != nil {
		sentry.CaptureHandlerError(c, err, "auth.GetMe", sentry.ErrorTypeDatabase, sentry.SeverityMedium)
		utils.ErrorResponse(c, http.StatusNotFound, "User not found", err)
		return
	}

	var role, fullName string
	var tenantID *uuid.UUID
	var tenantName string

	if user.UserType == models.UserTypeTenantStaff {
		var staff models.TenantStaff
		if err := h.DB.Preload("Tenant").Where("user_id = ?", user.ID).First(&staff).Error; err == nil {
			role = string(staff.Role)
			fullName = staff.FullName
			tenantID = &staff.TenantID
			if staff.Tenant != nil {
				tenantName = staff.Tenant.CompanyName
			}
		}
	}

	utils.SuccessResponse(c, http.StatusOK, "User profile", UserResponse{
		ID:                 user.ID,
		Email:              user.Email,
		UserType:           string(user.UserType),
		Role:               role,
		FullName:           fullName,
		TenantID:           tenantID,
		TenantName:         tenantName,
		MustChangePassword: user.MustChangePassword,
	})
}

// UpdateProfileRequest for updating user profile
type UpdateProfileRequest struct {
	FullName    string  `json:"full_name"`
	PhoneNumber *string `json:"phone_number"` // Pointer to allow clearing
	Address     *string `json:"address"`      // Pointer to allow clearing
}

// UpdateMe updates the current user's profile
func (h *AuthHandler) UpdateMe(c *gin.Context) {
	userUUID, ok := utils.GetUserUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "Invalid user context", nil)
		return
	}

	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request", err)
		return
	}

	// Get user to determine type
	var user models.User
	if err := h.DB.Where("id = ?", userUUID).First(&user).Error; err != nil {
		sentry.CaptureHandlerError(c, err, "auth.UpdateMe", sentry.ErrorTypeDatabase, sentry.SeverityMedium)
		utils.ErrorResponse(c, http.StatusNotFound, "User not found", err)
		return
	}

	// Build updates map from request
	updates := map[string]interface{}{}
	if req.FullName != "" {
		updates["full_name"] = req.FullName
	}
	if req.PhoneNumber != nil {
		updates["phone_number"] = *req.PhoneNumber
	}
	if req.Address != nil {
		updates["address"] = *req.Address
	}

	// Update based on user type
	if user.UserType == models.UserTypeTenantStaff {
		var staff models.TenantStaff
		if err := h.DB.Where("user_id = ?", userUUID).First(&staff).Error; err != nil {
			utils.ErrorResponse(c, http.StatusNotFound, "Staff profile not found", err)
			return
		}

		if len(updates) > 0 {
			if err := h.DB.Model(&staff).Updates(updates).Error; err != nil {
				sentry.CaptureHandlerError(c, err, "auth.UpdateMe", sentry.ErrorTypeDatabase, sentry.SeverityMedium)
				utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to update profile", err)
				return
			}
		}

		// Refresh for response
		h.DB.Where("user_id = ?", userUUID).First(&staff)

		utils.SuccessResponse(c, http.StatusOK, "Profile updated", gin.H{
			"full_name":    staff.FullName,
			"phone_number": staff.PhoneNumber,
			"address":      staff.Address,
		})
	} else {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid user type", nil)
	}
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=8,max=128"`
}

func (h *AuthHandler) ChangePassword(c *gin.Context) {
	userUUID, ok := utils.GetUserUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "Invalid user context", nil)
		return
	}

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sentry.CaptureHandlerError(c, err, "auth.ChangePassword", sentry.ErrorTypeValidation, sentry.SeverityLow)
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request", err)
		return
	}

	// Get current user
	var user models.User
	if err := h.DB.Where("id = ?", userUUID).First(&user).Error; err != nil {
		sentry.CaptureHandlerError(c, err, "auth.ChangePassword", sentry.ErrorTypeDatabase, sentry.SeverityMedium)
		utils.ErrorResponse(c, http.StatusNotFound, "User not found", err)
		return
	}

	// Verify current password
	if !utils.CheckPassword(req.CurrentPassword, user.PasswordHash) {
		utils.ErrorResponse(c, http.StatusUnauthorized, "Password saat ini salah", nil)
		return
	}

	// Check if new password is different from current
	if req.CurrentPassword == req.NewPassword {
		utils.ErrorResponse(c, http.StatusBadRequest, "Password baru harus berbeda dari password saat ini", nil)
		return
	}

	// Hash new password
	hashedPassword, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		sentry.CaptureHandlerError(c, err, "auth.ChangePassword", sentry.ErrorTypeInternal, sentry.SeverityCritical)
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to hash password", err)
		return
	}

	// Update password and clear must_change_password flag
	if err := h.DB.Model(&user).Updates(map[string]interface{}{
		"password_hash":        hashedPassword,
		"must_change_password": false,
	}).Error; err != nil {
		sentry.CaptureHandlerError(c, err, "auth.ChangePassword", sentry.ErrorTypeDatabase, sentry.SeverityCritical)
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to update password", err)
		return
	}

	// Evict all existing sessions (any stolen/other-device token issued before
	// now is invalidated), then re-mint the caller's tokens so this session
	// stays signed in. Fail-open when Redis is unavailable.
	_ = utils.NewTokenBlacklist().RevokeUserTokens(user.ID.String(), time.Duration(h.Cfg.JWT.RefreshHours)*time.Hour)

	var tenantID *uuid.UUID
	if tid, ok := utils.GetTenantUUID(c); ok {
		tenantID = &tid
	}
	if tokenPair, terr := utils.GenerateTokenPair(
		h.Cfg.JWT.Secret, user.ID, user.Email, string(user.UserType),
		c.GetString("role"), tenantID, h.Cfg.JWT.ExpirationHours, h.Cfg.JWT.RefreshHours, false,
	); terr == nil {
		utils.SetTokenCookies(c, tokenPair, h.Cfg.JWT.ExpirationHours, h.Cfg.JWT.RefreshHours)
	}

	// Audit log
	audit.LogFromContext(c, h.DB, models.ActionTypeUpdate, "password", &user.ID, nil, nil)

	utils.SuccessResponse(c, http.StatusOK, "Password changed successfully", nil)
}
