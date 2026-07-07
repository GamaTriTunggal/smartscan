package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gamatritunggal/smartscan/backend/internal/config"
	"github.com/gamatritunggal/smartscan/backend/internal/models"
	"github.com/gamatritunggal/smartscan/backend/internal/sentry"
	"github.com/gamatritunggal/smartscan/backend/internal/utils"
	"gorm.io/gorm"
)

type SocialMediaHandler struct {
	DB  *gorm.DB
	Cfg *config.Config
}

// isSafeIconValue rejects platform icon values containing HTML/attribute-significant
// characters. The icon is either a platform-code reference or bare SVG path data,
// neither of which needs <, >, ", ', or &. This is defense-in-depth against stored
// XSS: the frontend already renders icons from a trusted static map, but blocking
// poisoned values at write time keeps the master-data table clean regardless.
func isSafeIconValue(icon string) bool {
	return !strings.ContainsAny(icon, "<>\"'&")
}

func NewSocialMediaHandler(db *gorm.DB, cfg *config.Config) *SocialMediaHandler {
	return &SocialMediaHandler{DB: db, Cfg: cfg}
}

// ==================== SOCIAL MEDIA PLATFORMS (Super Admin) ====================

// ListSocialMediaPlatforms returns all social media platforms
func (h *SocialMediaHandler) ListSocialMediaPlatforms(c *gin.Context) {
	page := utils.GetPageQuery(c)
	limit := utils.GetLimitQuery(c, 50)
	search := c.Query("search")
	status := c.DefaultQuery("status", "active")

	offset := (page - 1) * limit

	var platforms []models.SocialMediaPlatform
	var total int64

	query := h.DB.Model(&models.SocialMediaPlatform{})

	// Status filter
	switch status {
	case "deleted":
		query = query.Unscoped().Where("deleted_at IS NOT NULL")
	case "all":
		query = query.Unscoped()
	default: // active
		query = query.Where("deleted_at IS NULL")
	}

	if search != "" {
		query = query.Where("name ILIKE ? OR code ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	query.Count(&total)
	query.Order("display_order ASC, name ASC").Offset(offset).Limit(limit).Find(&platforms)

	utils.SuccessResponse(c, http.StatusOK, "Social media platforms retrieved", gin.H{
		"platforms": platforms,
		"pagination": gin.H{
			"page":       page,
			"limit":      limit,
			"total":      total,
			"total_page": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// GetSocialMediaPlatform returns a single social media platform
func (h *SocialMediaHandler) GetSocialMediaPlatform(c *gin.Context) {
	id := c.Param("id")
	platformID, err := uuid.Parse(id)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid platform ID", err)
		return
	}

	var platform models.SocialMediaPlatform
	if err := h.DB.Unscoped().First(&platform, "id = ?", platformID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Social media platform not found", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Social media platform retrieved", platform)
}

// CreateSocialMediaPlatform creates a new social media platform
func (h *SocialMediaHandler) CreateSocialMediaPlatform(c *gin.Context) {
	var req struct {
		Code            string `json:"code" binding:"required"`
		Name            string `json:"name" binding:"required"`
		Icon            string `json:"icon"`
		BaseURL         string `json:"base_url"`
		DeepLinkPattern string `json:"deep_link_pattern"`
		PlaceholderText string `json:"placeholder_text"`
		DisplayOrder    int    `json:"display_order"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request", err)
		return
	}

	if !isSafeIconValue(req.Icon) {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid icon value", nil)
		return
	}

	// Check for duplicate code
	var existing models.SocialMediaPlatform
	if err := h.DB.Unscoped().Where("code = ?", req.Code).First(&existing).Error; err == nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Platform code already exists", nil)
		return
	}

	platform := models.SocialMediaPlatform{
		Code:            req.Code,
		Name:            req.Name,
		Icon:            req.Icon,
		BaseURL:         req.BaseURL,
		DeepLinkPattern: req.DeepLinkPattern,
		PlaceholderText: req.PlaceholderText,
		IsActive:        true,
		DisplayOrder:    req.DisplayOrder,
	}

	if err := h.DB.Create(&platform).Error; err != nil {
		sentry.CaptureHandlerError(c, err, "socialMedia.CreateSocialMediaPlatform", sentry.ErrorTypeDatabase, sentry.SeverityLow)
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to create platform", err)
		return
	}

	utils.SuccessResponse(c, http.StatusCreated, "Social media platform created", platform)
}

// UpdateSocialMediaPlatform updates a social media platform
func (h *SocialMediaHandler) UpdateSocialMediaPlatform(c *gin.Context) {
	id := c.Param("id")
	platformID, err := uuid.Parse(id)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid platform ID", err)
		return
	}

	var platform models.SocialMediaPlatform
	if err := h.DB.First(&platform, "id = ?", platformID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Social media platform not found", err)
		return
	}

	var req struct {
		Code            string `json:"code"`
		Name            string `json:"name"`
		Icon            string `json:"icon"`
		BaseURL         string `json:"base_url"`
		DeepLinkPattern string `json:"deep_link_pattern"`
		PlaceholderText string `json:"placeholder_text"`
		IsActive        *bool  `json:"is_active"`
		DisplayOrder    *int   `json:"display_order"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request", err)
		return
	}

	if !isSafeIconValue(req.Icon) {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid icon value", nil)
		return
	}

	updates := map[string]interface{}{}

	// Check for duplicate code if changing
	if req.Code != "" && req.Code != platform.Code {
		var existing models.SocialMediaPlatform
		if err := h.DB.Unscoped().Where("code = ? AND id != ?", req.Code, platformID).First(&existing).Error; err == nil {
			utils.ErrorResponse(c, http.StatusBadRequest, "Platform code already exists", nil)
			return
		}
		updates["code"] = req.Code
	}

	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Icon != "" {
		updates["icon"] = req.Icon
	}
	if req.BaseURL != "" {
		updates["base_url"] = req.BaseURL
	}
	if req.DeepLinkPattern != "" {
		updates["deep_link_pattern"] = req.DeepLinkPattern
	}
	if req.PlaceholderText != "" {
		updates["placeholder_text"] = req.PlaceholderText
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}
	if req.DisplayOrder != nil {
		updates["display_order"] = *req.DisplayOrder
	}

	if len(updates) > 0 {
		if err := h.DB.Model(&platform).Updates(updates).Error; err != nil {
			sentry.CaptureHandlerError(c, err, "socialMedia.UpdateSocialMediaPlatform", sentry.ErrorTypeDatabase, sentry.SeverityLow)
			utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to update platform", err)
			return
		}
	}

	// Re-fetch for response
	h.DB.First(&platform, "id = ?", platformID)

	utils.SuccessResponse(c, http.StatusOK, "Social media platform updated", platform)
}

// DeleteSocialMediaPlatform soft deletes a social media platform
func (h *SocialMediaHandler) DeleteSocialMediaPlatform(c *gin.Context) {
	id := c.Param("id")
	platformID, err := uuid.Parse(id)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid platform ID", err)
		return
	}

	var platform models.SocialMediaPlatform
	if err := h.DB.First(&platform, "id = ?", platformID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Social media platform not found", err)
		return
	}

	now := time.Now().UTC()

	if err := h.DB.Model(&platform).Update("deleted_at", now).Error; err != nil {
		sentry.CaptureHandlerError(c, err, "socialMedia.DeleteSocialMediaPlatform", sentry.ErrorTypeDatabase, sentry.SeverityLow)
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to delete platform", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Social media platform deleted", nil)
}

// RestoreSocialMediaPlatform restores a soft-deleted social media platform
func (h *SocialMediaHandler) RestoreSocialMediaPlatform(c *gin.Context) {
	id := c.Param("id")
	platformID, err := uuid.Parse(id)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid platform ID", err)
		return
	}

	var platform models.SocialMediaPlatform
	if err := h.DB.Unscoped().First(&platform, "id = ?", platformID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Social media platform not found", err)
		return
	}

	if err := h.DB.Unscoped().Model(&platform).Update("deleted_at", nil).Error; err != nil {
		sentry.CaptureHandlerError(c, err, "socialMedia.RestoreSocialMediaPlatform", sentry.ErrorTypeDatabase, sentry.SeverityLow)
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to restore platform", err)
		return
	}

	platform.DeletedAt = nil
	utils.SuccessResponse(c, http.StatusOK, "Social media platform restored", platform)
}

// ==================== SOCIAL MEDIA REQUESTS (Super Admin Review) ====================





// ==================== TENANT ENDPOINTS ====================

// GetAvailableSocialMediaPlatforms returns active platforms for tenant dropdown
func (h *SocialMediaHandler) GetAvailableSocialMediaPlatforms(c *gin.Context) {
	var platforms []models.SocialMediaPlatform

	h.DB.Where("is_active = ? AND deleted_at IS NULL", true).
		Order("display_order ASC, name ASC").
		Find(&platforms)

	utils.SuccessResponse(c, http.StatusOK, "Social media platforms retrieved", platforms)
}

// GetProductSocialLinks returns social links for a specific product
func (h *SocialMediaHandler) GetProductSocialLinks(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}
	productID := c.Param("product_id")
	productUUIDParsed, err := uuid.Parse(productID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid product ID", err)
		return
	}

	// Verify product belongs to tenant
	var product models.Product
	if err := h.DB.First(&product, "id = ? AND tenant_id = ?", productUUIDParsed, tenantUUID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Product not found", err)
		return
	}

	var links []models.ProductSocialLink
	h.DB.Where("product_id = ?", productUUIDParsed).
		Preload("Platform").
		Find(&links)

	utils.SuccessResponse(c, http.StatusOK, "Product social links retrieved", links)
}

// AddProductSocialLink adds a social link to a product
func (h *SocialMediaHandler) AddProductSocialLink(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}
	productID := c.Param("product_id")
	productUUIDParsed, err := uuid.Parse(productID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid product ID", err)
		return
	}

	var req struct {
		PlatformID  string `json:"platform_id" binding:"required"`
		HandleOrURL string `json:"handle_or_url" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request", err)
		return
	}

	platformUUID, err := uuid.Parse(req.PlatformID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid platform ID", err)
		return
	}

	// Verify product belongs to tenant
	var product models.Product
	if err := h.DB.First(&product, "id = ? AND tenant_id = ?", productUUIDParsed, tenantUUID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Product not found", err)
		return
	}

	// Verify platform exists and is active
	var platform models.SocialMediaPlatform
	if err := h.DB.First(&platform, "id = ? AND is_active = ? AND deleted_at IS NULL", platformUUID, true).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Social media platform not found or not active", err)
		return
	}

	// Check if already added
	var existing models.ProductSocialLink
	if err := h.DB.Where("product_id = ? AND platform_id = ?", productUUIDParsed, platformUUID).First(&existing).Error; err == nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Product already has a link for this platform", nil)
		return
	}

	link := models.ProductSocialLink{
		ProductID:   productUUIDParsed,
		PlatformID:  platformUUID,
		HandleOrURL: req.HandleOrURL,
	}

	if err := h.DB.Create(&link).Error; err != nil {
		sentry.CaptureHandlerError(c, err, "socialMedia.AddProductSocialLink", sentry.ErrorTypeDatabase, sentry.SeverityLow)
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to add social link", err)
		return
	}

	// Load relation
	h.DB.Preload("Platform").First(&link, "id = ?", link.ID)

	utils.SuccessResponse(c, http.StatusCreated, "Social link added to product", link)
}

// UpdateProductSocialLink updates a product social link
func (h *SocialMediaHandler) UpdateProductSocialLink(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}
	productID := c.Param("product_id")
	linkID := c.Param("link_id")

	productUUIDParsed, err := uuid.Parse(productID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid product ID", err)
		return
	}
	linkUUIDParsed, err := uuid.Parse(linkID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid link ID", err)
		return
	}

	// Verify product belongs to tenant
	var product models.Product
	if err := h.DB.First(&product, "id = ? AND tenant_id = ?", productUUIDParsed, tenantUUID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Product not found", err)
		return
	}

	var link models.ProductSocialLink
	if err := h.DB.First(&link, "id = ? AND product_id = ?", linkUUIDParsed, productUUIDParsed).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Product social link not found", err)
		return
	}

	var req struct {
		HandleOrURL string `json:"handle_or_url" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request", err)
		return
	}

	if err := h.DB.Model(&link).Update("handle_or_url", req.HandleOrURL).Error; err != nil {
		sentry.CaptureHandlerError(c, err, "socialMedia.UpdateProductSocialLink", sentry.ErrorTypeDatabase, sentry.SeverityLow)
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to update social link", err)
		return
	}

	h.DB.Preload("Platform").First(&link, "id = ?", link.ID)

	utils.SuccessResponse(c, http.StatusOK, "Product social link updated", link)
}

// RemoveProductSocialLink removes a social link from a product
func (h *SocialMediaHandler) RemoveProductSocialLink(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}
	productID := c.Param("product_id")
	linkID := c.Param("link_id")

	productUUIDParsed, err := uuid.Parse(productID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid product ID", err)
		return
	}
	linkUUIDParsed, err := uuid.Parse(linkID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid link ID", err)
		return
	}

	// Verify product belongs to tenant
	var product models.Product
	if err := h.DB.First(&product, "id = ? AND tenant_id = ?", productUUIDParsed, tenantUUID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Product not found", err)
		return
	}

	var link models.ProductSocialLink
	if err := h.DB.First(&link, "id = ? AND product_id = ?", linkUUIDParsed, productUUIDParsed).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Product social link not found", err)
		return
	}

	if err := h.DB.Delete(&link).Error; err != nil {
		sentry.CaptureHandlerError(c, err, "socialMedia.RemoveProductSocialLink", sentry.ErrorTypeDatabase, sentry.SeverityLow)
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to remove social link", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Social link removed from product", nil)
}


