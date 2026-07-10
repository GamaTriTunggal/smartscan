package handlers

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gamatritunggal/smartscan/backend/internal/config"
	"github.com/gamatritunggal/smartscan/backend/internal/models"
	"github.com/gamatritunggal/smartscan/backend/internal/sentry"
	"github.com/gamatritunggal/smartscan/backend/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type AppSettingsHandler struct {
	DB  *gorm.DB
	Cfg *config.Config
}

func NewAppSettingsHandler(db *gorm.DB, cfg *config.Config) *AppSettingsHandler {
	return &AppSettingsHandler{DB: db, Cfg: cfg}
}

// Branding cache
var (
	brandingCache     *models.BrandingSettings
	brandingCacheMu   sync.RWMutex
	brandingCacheTime time.Time
	brandingCacheTTL  = 5 * time.Minute
)

// GetBrandingFromDB fetches branding settings from database with caching
func GetBrandingFromDB(db *gorm.DB) models.BrandingSettings {
	brandingCacheMu.RLock()
	if brandingCache != nil && time.Since(brandingCacheTime) < brandingCacheTTL {
		result := *brandingCache
		brandingCacheMu.RUnlock()
		return result
	}
	brandingCacheMu.RUnlock()

	brandingCacheMu.Lock()
	defer brandingCacheMu.Unlock()

	// Double-check after acquiring write lock
	if brandingCache != nil && time.Since(brandingCacheTime) < brandingCacheTTL {
		return *brandingCache
	}

	var settings models.AppSettings
	if err := db.Where("setting_key = ?", "branding").First(&settings).Error; err != nil {
		// Return default if not found
		defaultBranding := models.DefaultBrandingSettings()
		brandingCache = &defaultBranding
		brandingCacheTime = time.Now()
		return defaultBranding
	}

	var branding models.BrandingSettings
	if err := json.Unmarshal(settings.SettingValue, &branding); err != nil {
		defaultBranding := models.DefaultBrandingSettings()
		brandingCache = &defaultBranding
		brandingCacheTime = time.Now()
		return defaultBranding
	}

	brandingCache = &branding
	brandingCacheTime = time.Now()
	return branding
}

// InvalidateBrandingCache clears the branding cache
func InvalidateBrandingCache() {
	brandingCacheMu.Lock()
	defer brandingCacheMu.Unlock()
	brandingCache = nil
}

// =============================================================================
// PUBLIC ENDPOINTS
// =============================================================================

// GetBrandingPublic returns branding settings (public, no auth required)
// Used by frontend to show app name, logo, etc.
func (h *AppSettingsHandler) GetBrandingPublic(c *gin.Context) {
	branding := GetBrandingFromDB(h.DB)
	utils.SuccessResponse(c, http.StatusOK, "Branding retrieved", branding)
}

// =============================================================================
// AUTHENTICATED ENDPOINTS (admin)
// =============================================================================

// GetBranding returns branding settings for authenticated users
func (h *AppSettingsHandler) GetBranding(c *gin.Context) {
	var settings models.AppSettings
	if err := h.DB.Where("setting_key = ?", "branding").First(&settings).Error; err != nil {
		// Return default if not found
		utils.SuccessResponse(c, http.StatusOK, "Branding retrieved (default)", models.DefaultBrandingSettings())
		return
	}

	var branding models.BrandingSettings
	if err := json.Unmarshal(settings.SettingValue, &branding); err != nil {
		sentry.CaptureHandlerError(c, err, "appSettings.GetBranding", sentry.ErrorTypeInternal, sentry.SeverityMedium)
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to parse branding settings", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Branding retrieved", branding)
}

// UpdateBranding updates branding settings (Super Admin only)
func (h *AppSettingsHandler) UpdateBranding(c *gin.Context) {
	userUUID, ok := utils.GetUserUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "Invalid user context", nil)
		return
	}

	var req models.BrandingSettings
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request", err)
		return
	}

	// Validate required fields
	if req.AppName == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "App name is required", nil)
		return
	}

	// Validate color formats (basic hex validation)
	colors := []string{
		req.HeaderGradientStart,
		req.HeaderGradientEnd,
		req.HeaderTextColor,
		req.ButtonBgColor,
		req.ButtonTextColor,
	}
	for _, color := range colors {
		if color != "" && !isValidHexColor(color) {
			utils.ErrorResponse(c, http.StatusBadRequest, "Invalid color format: "+color, nil)
			return
		}
	}

	// Get staff ID from user ID
	var staff models.TenantStaff
	if err := h.DB.Where("user_id = ?", userUUID).First(&staff).Error; err != nil {
		sentry.CaptureHandlerError(c, err, "appSettings.UpdateBranding", sentry.ErrorTypeDatabase, sentry.SeverityMedium)
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to get staff info", err)
		return
	}

	// Marshal the settings to JSON
	settingValue, err := json.Marshal(req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to marshal settings", err)
		return
	}

	// Check if branding setting exists
	var settings models.AppSettings
	if err := h.DB.Where("setting_key = ?", "branding").First(&settings).Error; err != nil {
		// Create new
		settings = models.AppSettings{
			ID:           uuid.Must(uuid.NewV7()),
			SettingKey:   "branding",
			SettingValue: datatypes.JSON(settingValue),
			UpdatedBy:    &staff.ID,
		}
		if err := h.DB.Create(&settings).Error; err != nil {
			sentry.CaptureHandlerError(c, err, "appSettings.UpdateBranding", sentry.ErrorTypeDatabase, sentry.SeverityMedium)
			utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to create branding settings", err)
			return
		}
	} else {
		// Update existing
		brandingUpdates := map[string]interface{}{
			"setting_value": datatypes.JSON(settingValue),
			"updated_by":    staff.ID,
			"updated_at":    time.Now().UTC(),
		}
		if err := h.DB.Model(&settings).Updates(brandingUpdates).Error; err != nil {
			sentry.CaptureHandlerError(c, err, "appSettings.UpdateBranding", sentry.ErrorTypeDatabase, sentry.SeverityMedium)
			utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to update branding settings", err)
			return
		}
	}

	// Invalidate cache
	InvalidateBrandingCache()

	utils.SuccessResponse(c, http.StatusOK, "Branding updated successfully", req)
}

// isValidHexColor validates if a string is a valid hex color
func isValidHexColor(color string) bool {
	if len(color) != 7 && len(color) != 4 {
		return false
	}
	if color[0] != '#' {
		return false
	}
	for _, c := range color[1:] {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}
