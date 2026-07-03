package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gamatritunggal/smartscan/backend/internal/config"
	"github.com/gamatritunggal/smartscan/backend/internal/models"
	"github.com/gamatritunggal/smartscan/backend/internal/utils"
	"gorm.io/gorm"
)

type ThemePresetHandler struct {
	DB  *gorm.DB
	Cfg *config.Config
}

func NewThemePresetHandler(db *gorm.DB, cfg *config.Config) *ThemePresetHandler {
	return &ThemePresetHandler{DB: db, Cfg: cfg}
}

// ListThemePresets returns all theme presets with filtering
// GET /tenant/theme-presets
func (h *ThemePresetHandler) ListThemePresets(c *gin.Context) {
	page := utils.GetPageQuery(c)
	limit := utils.GetLimitQuery(c, 50)
	presetType := c.Query("type")
	status := c.DefaultQuery("status", "active")

	offset := (page - 1) * limit

	var presets []models.ThemePreset
	var total int64

	query := h.DB.Model(&models.ThemePreset{})

	// Status filter
	switch status {
	case "deleted":
		query = query.Unscoped().Where("deleted_at IS NOT NULL")
	case "all":
		query = query.Unscoped()
	default: // active
		query = query.Where("deleted_at IS NULL")
	}

	// Type filter
	if presetType != "" {
		query = query.Where("preset_type = ?", presetType)
	}

	query.Count(&total)
	query.Preload("Creator").Order("display_order ASC, name ASC").Offset(offset).Limit(limit).Find(&presets)

	utils.SuccessResponse(c, http.StatusOK, "Theme presets retrieved", gin.H{
		"theme_presets": presets,
		"pagination": gin.H{
			"page":       page,
			"limit":      limit,
			"total":      total,
			"total_page": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// GetThemePreset returns a single theme preset
// GET /tenant/theme-presets/:id
func (h *ThemePresetHandler) GetThemePreset(c *gin.Context) {
	id := c.Param("id")
	presetID, err := uuid.Parse(id)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid theme preset ID", err)
		return
	}

	var preset models.ThemePreset
	if err := h.DB.Unscoped().Preload("Creator").First(&preset, "id = ?", presetID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Theme preset not found", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Theme preset retrieved", preset)
}

// CreateThemePreset creates a new theme preset
// POST /tenant/theme-presets
func (h *ThemePresetHandler) CreateThemePreset(c *gin.Context) {
	staffID := c.GetString("staff_id")

	var req struct {
		Name           string `json:"name" binding:"required"`
		Description    string `json:"description"`
		PresetType     string `json:"preset_type" binding:"required,oneof=landing campaign"`
		BackgroundURL  string `json:"background_url" binding:"required"`
		ThumbnailURL   string `json:"thumbnail_url"`
		OverlayColor   string `json:"overlay_color"`
		OverlayOpacity *int   `json:"overlay_opacity"`
		CardOpacity    *int   `json:"card_opacity"`
		CardBlur       int    `json:"card_blur"`
		DisplayOrder   int    `json:"display_order"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request", err)
		return
	}

	// Set defaults using pointers to distinguish nil from 0
	if req.OverlayColor == "" {
		req.OverlayColor = "#000000"
	}
	overlayOpacity := 30
	if req.OverlayOpacity != nil {
		overlayOpacity = *req.OverlayOpacity
	}
	cardOpacity := 90
	if req.CardOpacity != nil {
		cardOpacity = *req.CardOpacity
	}

	// Validate ranges
	if overlayOpacity < 0 || overlayOpacity > 100 {
		utils.ErrorResponse(c, http.StatusBadRequest, "Overlay opacity must be between 0 and 100", nil)
		return
	}
	if cardOpacity < 50 || cardOpacity > 100 {
		utils.ErrorResponse(c, http.StatusBadRequest, "Card opacity must be between 50 and 100", nil)
		return
	}
	if req.CardBlur < 0 || req.CardBlur > 20 {
		utils.ErrorResponse(c, http.StatusBadRequest, "Card blur must be between 0 and 20", nil)
		return
	}

	var createdBy *uuid.UUID
	if staffID != "" {
		parsed, _ := uuid.Parse(staffID)
		createdBy = &parsed
	}

	preset := models.ThemePreset{
		Name:           req.Name,
		Description:    req.Description,
		PresetType:     models.PresetType(req.PresetType),
		BackgroundURL:  req.BackgroundURL,
		ThumbnailURL:   req.ThumbnailURL,
		OverlayColor:   req.OverlayColor,
		OverlayOpacity: overlayOpacity,
		CardOpacity:    cardOpacity,
		CardBlur:       req.CardBlur,
		IsActive:       true,
		DisplayOrder:   req.DisplayOrder,
		CreatedBy:      createdBy,
	}

	if err := h.DB.Create(&preset).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to create theme preset", err)
		return
	}

	utils.SuccessResponse(c, http.StatusCreated, "Theme preset created", preset)
}

// UpdateThemePreset updates an existing theme preset
// PUT /tenant/theme-presets/:id
func (h *ThemePresetHandler) UpdateThemePreset(c *gin.Context) {
	id := c.Param("id")
	presetID, err := uuid.Parse(id)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid theme preset ID", err)
		return
	}

	var preset models.ThemePreset
	if err := h.DB.First(&preset, "id = ?", presetID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Theme preset not found", err)
		return
	}

	var req struct {
		Name           string `json:"name"`
		Description    string `json:"description"`
		PresetType     string `json:"preset_type"`
		BackgroundURL  string `json:"background_url"`
		ThumbnailURL   string `json:"thumbnail_url"`
		OverlayColor   string `json:"overlay_color"`
		OverlayOpacity *int   `json:"overlay_opacity"`
		CardOpacity    *int   `json:"card_opacity"`
		CardBlur       *int   `json:"card_blur"`
		IsActive       *bool  `json:"is_active"`
		DisplayOrder   *int   `json:"display_order"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request", err)
		return
	}

	// Update fields if provided
	updates := map[string]interface{}{}

	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.PresetType != "" {
		if req.PresetType != "landing" && req.PresetType != "campaign" {
			utils.ErrorResponse(c, http.StatusBadRequest, "Invalid preset type", nil)
			return
		}
		updates["preset_type"] = req.PresetType
	}
	if req.BackgroundURL != "" {
		updates["background_url"] = req.BackgroundURL
	}
	if req.ThumbnailURL != "" {
		updates["thumbnail_url"] = req.ThumbnailURL
	}
	if req.OverlayColor != "" {
		updates["overlay_color"] = req.OverlayColor
	}
	if req.OverlayOpacity != nil {
		if *req.OverlayOpacity < 0 || *req.OverlayOpacity > 100 {
			utils.ErrorResponse(c, http.StatusBadRequest, "Overlay opacity must be between 0 and 100", nil)
			return
		}
		updates["overlay_opacity"] = *req.OverlayOpacity
	}
	if req.CardOpacity != nil {
		if *req.CardOpacity < 50 || *req.CardOpacity > 100 {
			utils.ErrorResponse(c, http.StatusBadRequest, "Card opacity must be between 50 and 100", nil)
			return
		}
		updates["card_opacity"] = *req.CardOpacity
	}
	if req.CardBlur != nil {
		if *req.CardBlur < 0 || *req.CardBlur > 20 {
			utils.ErrorResponse(c, http.StatusBadRequest, "Card blur must be between 0 and 20", nil)
			return
		}
		updates["card_blur"] = *req.CardBlur
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}
	if req.DisplayOrder != nil {
		updates["display_order"] = *req.DisplayOrder
	}

	if len(updates) > 0 {
		if err := h.DB.Model(&preset).Updates(updates).Error; err != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to update theme preset", err)
			return
		}
	}

	// Re-fetch for response
	h.DB.First(&preset, "id = ?", presetID)

	utils.SuccessResponse(c, http.StatusOK, "Theme preset updated", preset)
}

// DeleteThemePreset soft deletes a theme preset
// DELETE /tenant/theme-presets/:id
func (h *ThemePresetHandler) DeleteThemePreset(c *gin.Context) {
	id := c.Param("id")
	presetID, err := uuid.Parse(id)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid theme preset ID", err)
		return
	}

	var preset models.ThemePreset
	if err := h.DB.First(&preset, "id = ?", presetID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Theme preset not found", err)
		return
	}

	if err := h.DB.Delete(&preset).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to delete theme preset", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Theme preset deleted", nil)
}

// RestoreThemePreset restores a soft-deleted theme preset
// POST /tenant/theme-presets/:id/restore
func (h *ThemePresetHandler) RestoreThemePreset(c *gin.Context) {
	id := c.Param("id")
	presetID, err := uuid.Parse(id)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid theme preset ID", err)
		return
	}

	var preset models.ThemePreset
	if err := h.DB.Unscoped().First(&preset, "id = ?", presetID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Theme preset not found", err)
		return
	}

	if preset.DeletedAt.Time.IsZero() {
		utils.ErrorResponse(c, http.StatusBadRequest, "Theme preset is not deleted", nil)
		return
	}

	if err := h.DB.Unscoped().Model(&preset).Update("deleted_at", nil).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to restore theme preset", err)
		return
	}

	preset.DeletedAt = gorm.DeletedAt{}
	utils.SuccessResponse(c, http.StatusOK, "Theme preset restored", preset)
}

// ReorderThemePresets updates the display order of theme presets
// PUT /tenant/theme-presets/reorder
func (h *ThemePresetHandler) ReorderThemePresets(c *gin.Context) {
	var req struct {
		Orders []struct {
			ID           string `json:"id"`
			DisplayOrder int    `json:"display_order"`
		} `json:"orders"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request", err)
		return
	}

	tx := h.DB.Begin()
	for _, order := range req.Orders {
		presetID, err := uuid.Parse(order.ID)
		if err != nil {
			tx.Rollback()
			utils.ErrorResponse(c, http.StatusBadRequest, "Invalid preset ID: "+order.ID, err)
			return
		}

		if err := tx.Model(&models.ThemePreset{}).Where("id = ?", presetID).Update("display_order", order.DisplayOrder).Error; err != nil {
			tx.Rollback()
			utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to update display order", err)
			return
		}
	}
	if err := tx.Commit().Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to commit transaction", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Theme presets reordered", nil)
}

// ==================== PUBLIC ENDPOINTS ====================

// ListPublicThemePresets returns active theme presets for public use
// GET /public/theme-presets
func (h *ThemePresetHandler) ListPublicThemePresets(c *gin.Context) {
	presetType := c.Query("type")

	var presets []models.ThemePreset

	query := h.DB.Where("is_active = ? AND deleted_at IS NULL", true)

	if presetType != "" {
		query = query.Where("preset_type = ?", presetType)
	}

	query.Order("display_order ASC, name ASC").Find(&presets)

	utils.SuccessResponse(c, http.StatusOK, "Theme presets retrieved", gin.H{
		"theme_presets": presets,
	})
}

// GetPublicThemePreset returns a single active theme preset
// GET /public/theme-presets/:id
func (h *ThemePresetHandler) GetPublicThemePreset(c *gin.Context) {
	id := c.Param("id")
	presetID, err := uuid.Parse(id)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid theme preset ID", err)
		return
	}

	var preset models.ThemePreset
	if err := h.DB.Where("is_active = ? AND deleted_at IS NULL", true).First(&preset, "id = ?", presetID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Theme preset not found", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Theme preset retrieved", preset)
}

// ==================== TENANT ENDPOINTS ====================

// ListTenantThemePresets returns active theme presets for tenant selection
// GET /tenant/theme-presets
func (h *ThemePresetHandler) ListTenantThemePresets(c *gin.Context) {
	presetType := c.Query("type")

	var presets []models.ThemePreset

	query := h.DB.Where("is_active = ? AND deleted_at IS NULL", true)

	if presetType != "" {
		query = query.Where("preset_type = ?", presetType)
	}

	query.Order("display_order ASC, name ASC").Find(&presets)

	utils.SuccessResponse(c, http.StatusOK, "Theme presets retrieved", gin.H{
		"theme_presets": presets,
	})
}
