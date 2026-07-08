package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gamatritunggal/smartscan/backend/internal/config"
	"github.com/gamatritunggal/smartscan/backend/internal/models"
	"github.com/gamatritunggal/smartscan/backend/internal/utils"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type ProductHandler struct {
	DB  *gorm.DB
	Cfg *config.Config
}

func NewProductHandler(db *gorm.DB, cfg *config.Config) *ProductHandler {
	return &ProductHandler{DB: db, Cfg: cfg}
}


type CreateProductRequest struct {
	ProductCode                 string          `json:"product_code" binding:"max=100"`
	ProductName                 string          `json:"product_name" binding:"required,max=255"`
	Description                 string          `json:"description" binding:"max=2000"`
	DisplayConfig               json.RawMessage `json:"display_config"`
	WarrantyFieldsConfig        json.RawMessage `json:"warranty_fields_config"`
	WarrantyEnabled             bool            `json:"warranty_enabled"`                // Enable warranty registration for this product
	WarrantyMonths              *int            `json:"warranty_months"`                 // Warranty period in months (default 12)
	MaxWarrantyRegistrationDays *int            `json:"max_warranty_registration_days"`  // Max days after purchase to register (null = unlimited)
	WebsiteURL                  *string         `json:"website_url"`                     // Product website URL
	WebsiteCaption              *string         `json:"website_caption"`                 // Button caption for website link
	CounterfeitScanMax          *int            `json:"counterfeit_scan_max"`            // Product-level counterfeit threshold (NULL = use tenant global)
}

// ListProducts returns all products for a tenant
func (h *ProductHandler) ListProducts(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}
	page := utils.GetPageQuery(c)
	limit := utils.GetLimitQuery(c, 20)
	search := c.Query("search")
	status := c.Query("status")

	offset := (page - 1) * limit

	var products []models.Product
	var total int64

	query := h.DB.Model(&models.Product{}).Where("tenant_id = ?", tenantUUID)

	if search != "" {
		query = query.Where("product_name ILIKE ? OR product_code ILIKE ?", "%"+search+"%", "%"+search+"%")
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	query.Count(&total)
	query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&products)


	utils.SuccessResponse(c, http.StatusOK, "Products retrieved", gin.H{
		"products": products,
		"pagination": utils.PaginationMeta(page, limit, total),
	})
}

// GetProduct returns a single product by ID
func (h *ProductHandler) GetProduct(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}
	id := c.Param("id")
	productID, err := uuid.Parse(id)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid product ID", err)
		return
	}

	var product models.Product
	if err := h.DB.Preload("DefaultValidationTemplate").Preload("DefaultWarrantyTemplate").
		First(&product, "id = ? AND tenant_id = ?", productID, tenantUUID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Product not found", err)
		return
	}


	utils.SuccessResponse(c, http.StatusOK, "Product retrieved", product)
}

// CreateProduct creates a new product
func (h *ProductHandler) CreateProduct(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}
	userUUID, ok := utils.GetUserUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "Invalid user context", nil)
		return
	}

	var req CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request", err)
		return
	}

	// Check for duplicate product code if provided
	if req.ProductCode != "" {
		var existing models.Product
		if err := h.DB.Where("tenant_id = ? AND product_code = ?", tenantUUID, req.ProductCode).First(&existing).Error; err == nil {
			utils.ErrorResponse(c, http.StatusBadRequest, "Product code already exists", nil)
			return
		}
	}

	// Check for duplicate product name
	{
		var existing models.Product
		if err := h.DB.Where("tenant_id = ? AND product_name = ?", tenantUUID, req.ProductName).First(&existing).Error; err == nil {
			utils.ErrorResponse(c, http.StatusBadRequest, "Product name already exists", nil)
			return
		}
	}

	// Get user staff ID
	var staff models.TenantStaff
	h.DB.Where("user_id = ?", userUUID).First(&staff)

	product := models.Product{
		TenantID:    tenantUUID,
		ProductCode: req.ProductCode,
		ProductName: req.ProductName,
		Description: req.Description,
		Status:      models.ProductStatusActive,
		CreatedBy:   &staff.ID,
	}

	// Set display config if provided
	if len(req.DisplayConfig) > 0 {
		product.DisplayConfig = datatypes.JSON(req.DisplayConfig)
	}

	// Set website URL and caption if provided (with validation)
	if req.WebsiteURL != nil && *req.WebsiteURL != "" {
		normalizedURL, err := utils.ValidateSocialURL(*req.WebsiteURL)
		if err != nil {
			utils.ErrorResponse(c, http.StatusBadRequest, "Invalid website URL: "+err.Error(), nil)
			return
		}
		product.WebsiteURL = normalizedURL
	}
	if req.WebsiteCaption != nil {
		product.WebsiteCaption = *req.WebsiteCaption
	}
	if req.CounterfeitScanMax != nil && *req.CounterfeitScanMax >= 1 {
		product.CounterfeitScanMax = req.CounterfeitScanMax
	}

	// Set warranty enabled
	product.WarrantyEnabled = req.WarrantyEnabled

	// Validate warranty_fields_config custom fields
	if err := validateWarrantyFieldsConfig(req.WarrantyFieldsConfig); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	// Set warranty fields config if provided
	if len(req.WarrantyFieldsConfig) > 0 {
		product.WarrantyFieldsConfig = datatypes.JSON(req.WarrantyFieldsConfig)
	}

	// Set warranty duration config
	if req.WarrantyMonths != nil {
		if *req.WarrantyMonths < 1 || *req.WarrantyMonths > 120 {
			utils.ErrorResponse(c, http.StatusBadRequest,
				"Warranty period must be between 1 and 120 months", nil)
			return
		}
		product.WarrantyMonths = *req.WarrantyMonths
	} else {
		product.WarrantyMonths = 12 // Default 12 months
	}
	if req.MaxWarrantyRegistrationDays != nil {
		if *req.MaxWarrantyRegistrationDays < 0 || *req.MaxWarrantyRegistrationDays > 365 {
			utils.ErrorResponse(c, http.StatusBadRequest,
				"Max registration days must be between 0 and 365 (0 = unlimited)", nil)
			return
		}
		// Store 0 as nil (unlimited)
		if *req.MaxWarrantyRegistrationDays == 0 {
			product.MaxWarrantyRegistrationDays = nil
		} else {
			product.MaxWarrantyRegistrationDays = req.MaxWarrantyRegistrationDays
		}
	}

	if err := h.DB.Create(&product).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to create product", err)
		return
	}

	utils.SuccessResponse(c, http.StatusCreated, "Product created", product)
}

// UpdateProduct updates an existing product
func (h *ProductHandler) UpdateProduct(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}
	id := c.Param("id")
	productID, err := uuid.Parse(id)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid product ID", err)
		return
	}

	var product models.Product
	if err := h.DB.First(&product, "id = ? AND tenant_id = ?", productID, tenantUUID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Product not found", err)
		return
	}

	var req struct {
		ProductName                 string          `json:"product_name"`
		ProductCode                 string          `json:"product_code"`
		Description                 *string         `json:"description"`
		Status                      string          `json:"status"`
		DisplayConfig               json.RawMessage `json:"display_config"`
		WarrantyFieldsConfig        json.RawMessage `json:"warranty_fields_config"`
		WarrantyEnabled             *bool           `json:"warranty_enabled"` // Enable warranty registration
		WarrantyMonths              *int            `json:"warranty_months"`
		MaxWarrantyRegistrationDays *int            `json:"max_warranty_registration_days"`
		DefaultValidationTemplateID *string         `json:"default_validation_template_id"`
		DefaultWarrantyTemplateID   *string         `json:"default_warranty_template_id"`
			// Landing page features
		WebsiteURL         *string         `json:"website_url"`
		WebsiteCaption     *string         `json:"website_caption"`
		Videos             json.RawMessage `json:"videos"`
		CounterfeitScanMax *int            `json:"counterfeit_scan_max"` // Product-level counterfeit threshold (NULL/0 = use tenant global)
		TemplateOverrides         json.RawMessage `json:"template_overrides"`          // Product-level template customization overrides
		WarrantyTemplateOverrides json.RawMessage `json:"warranty_template_overrides"` // Warranty template customization overrides
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request", err)
		return
	}

	// Check for duplicate product name when changing
	if req.ProductName != "" && req.ProductName != product.ProductName {
		var existing models.Product
		if err := h.DB.Where("tenant_id = ? AND product_name = ? AND id != ?", tenantUUID, req.ProductName, productID).First(&existing).Error; err == nil {
			utils.ErrorResponse(c, http.StatusBadRequest, "Product name already exists", nil)
			return
		}
	}

	// Check for duplicate product code when changing
	if req.ProductCode != "" && req.ProductCode != product.ProductCode {
		var existing models.Product
		if err := h.DB.Where("tenant_id = ? AND product_code = ? AND id != ?", tenantUUID, req.ProductCode, productID).First(&existing).Error; err == nil {
			utils.ErrorResponse(c, http.StatusBadRequest, "Product code already exists", nil)
			return
		}
	}

	updates := map[string]interface{}{}

	if req.ProductName != "" {
		updates["product_name"] = req.ProductName
	}
	if req.ProductCode != "" {
		updates["product_code"] = req.ProductCode
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.Status != "" {
		updates["status"] = req.Status
	}
	if len(req.DisplayConfig) > 0 {
		updates["display_config"] = datatypes.JSON(req.DisplayConfig)
	}

	// Update template overrides (product-level template customization)
	if req.TemplateOverrides != nil {
		raw := strings.TrimSpace(string(req.TemplateOverrides))
		if raw == "null" || raw == "{}" {
			updates["template_overrides"] = nil
		} else {
			// Size validation (max 10KB, consistent with custom_fields limit)
			if len(req.TemplateOverrides) > 10*1024 {
				utils.ErrorResponse(c, http.StatusBadRequest, "template_overrides exceeds maximum size of 10KB", nil)
				return
			}
			// Structural validation: must be a valid JSON object
			var check map[string]interface{}
			if err := json.Unmarshal(req.TemplateOverrides, &check); err != nil {
				utils.ErrorResponse(c, http.StatusBadRequest, "template_overrides must be a valid JSON object", nil)
				return
			}
			updates["template_overrides"] = datatypes.JSON(req.TemplateOverrides)
		}
	}

	// Update warranty template overrides (warranty page customization)
	if req.WarrantyTemplateOverrides != nil {
		raw := strings.TrimSpace(string(req.WarrantyTemplateOverrides))
		if raw == "null" || raw == "{}" {
			updates["warranty_template_overrides"] = nil
		} else {
			if len(req.WarrantyTemplateOverrides) > 10*1024 {
				utils.ErrorResponse(c, http.StatusBadRequest, "warranty_template_overrides exceeds maximum size of 10KB", nil)
				return
			}
			var check map[string]interface{}
			if err := json.Unmarshal(req.WarrantyTemplateOverrides, &check); err != nil {
				utils.ErrorResponse(c, http.StatusBadRequest, "warranty_template_overrides must be a valid JSON object", nil)
				return
			}
			updates["warranty_template_overrides"] = datatypes.JSON(req.WarrantyTemplateOverrides)
		}
	}

	// Update warranty enabled
	if req.WarrantyEnabled != nil {
		updates["warranty_enabled"] = *req.WarrantyEnabled
	}

	// Validate warranty_fields_config custom fields
	if err := validateWarrantyFieldsConfig(req.WarrantyFieldsConfig); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	if len(req.WarrantyFieldsConfig) > 0 {
		updates["warranty_fields_config"] = datatypes.JSON(req.WarrantyFieldsConfig)
	}

	// Handle warranty duration config
	if req.WarrantyMonths != nil {
		if *req.WarrantyMonths < 1 || *req.WarrantyMonths > 120 {
			utils.ErrorResponse(c, http.StatusBadRequest,
				"Warranty period must be between 1 and 120 months", nil)
			return
		}
		updates["warranty_months"] = *req.WarrantyMonths
	}
	if req.MaxWarrantyRegistrationDays != nil {
		if *req.MaxWarrantyRegistrationDays < 0 || *req.MaxWarrantyRegistrationDays > 365 {
			utils.ErrorResponse(c, http.StatusBadRequest,
				"Max registration days must be between 0 and 365 (0 = unlimited)", nil)
			return
		}
		// Store 0 as nil (unlimited)
		if *req.MaxWarrantyRegistrationDays == 0 {
			updates["max_warranty_registration_days"] = nil
		} else {
			updates["max_warranty_registration_days"] = *req.MaxWarrantyRegistrationDays
		}
	}

	// Handle default template ID updates
	if req.DefaultValidationTemplateID != nil {
		if *req.DefaultValidationTemplateID == "" {
			// Clear the template
			updates["default_validation_template_id"] = nil
		} else {
			// Set the template - validate it exists and belongs to tenant
			templateUUID, err := uuid.Parse(*req.DefaultValidationTemplateID)
			if err != nil {
				utils.ErrorResponse(c, http.StatusBadRequest, "Invalid validation template ID", err)
				return
			}
			var template models.PageTemplate
			if err := h.DB.First(&template, "id = ? AND tenant_id = ? AND template_type = ?",
				templateUUID, tenantUUID, models.TemplateTypeValidation).Error; err != nil {
				utils.ErrorResponse(c, http.StatusBadRequest, "Validation template not found", err)
				return
			}
			updates["default_validation_template_id"] = templateUUID
		}
	}

	if req.DefaultWarrantyTemplateID != nil {
		if *req.DefaultWarrantyTemplateID == "" {
			// Clear the template
			updates["default_warranty_template_id"] = nil
		} else {
			// Set the template - validate it exists and belongs to tenant
			templateUUID, err := uuid.Parse(*req.DefaultWarrantyTemplateID)
			if err != nil {
				utils.ErrorResponse(c, http.StatusBadRequest, "Invalid warranty template ID", err)
				return
			}
			var template models.PageTemplate
			if err := h.DB.First(&template, "id = ? AND tenant_id = ? AND template_type = ?",
				templateUUID, tenantUUID, models.TemplateTypeWarranty).Error; err != nil {
				utils.ErrorResponse(c, http.StatusBadRequest, "Warranty template not found", err)
				return
			}
			updates["default_warranty_template_id"] = templateUUID
		}
	}

	// Handle landing page features (with URL validation)
	if req.WebsiteURL != nil {
		if *req.WebsiteURL != "" {
			normalizedURL, err := utils.ValidateSocialURL(*req.WebsiteURL)
			if err != nil {
				utils.ErrorResponse(c, http.StatusBadRequest, "Invalid website URL: "+err.Error(), nil)
				return
			}
			updates["website_url"] = normalizedURL
		} else {
			updates["website_url"] = "" // Allow clearing the URL
		}
	}
	if req.WebsiteCaption != nil {
		updates["website_caption"] = *req.WebsiteCaption
	}
	if req.CounterfeitScanMax != nil {
		if *req.CounterfeitScanMax >= 1 {
			updates["counterfeit_scan_max"] = *req.CounterfeitScanMax
		} else {
			// 0 or negative = reset to tenant global (NULL)
			updates["counterfeit_scan_max"] = nil
		}
	}
	if len(req.Videos) > 0 {
		if err := validateVideos(req.Videos); err != nil {
			utils.ErrorResponse(c, http.StatusBadRequest, err.Error(), nil)
			return
		}
		updates["videos"] = datatypes.JSON(req.Videos)
	}

	if len(updates) > 0 {
		if err := h.DB.Model(&product).Updates(updates).Error; err != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to update product", err)
			return
		}
	}

	// Refresh product for response
	h.DB.First(&product, "id = ? AND tenant_id = ?", productID, tenantUUID)


	utils.SuccessResponse(c, http.StatusOK, "Product updated", product)
}

// DeleteProduct soft deletes a product
func (h *ProductHandler) DeleteProduct(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}
	id := c.Param("id")
	productID, err := uuid.Parse(id)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid product ID", err)
		return
	}

	var product models.Product
	if err := h.DB.First(&product, "id = ? AND tenant_id = ?", productID, tenantUUID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Product not found", err)
		return
	}

	if err := h.DB.Delete(&product).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to delete product", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Product deleted", nil)
}


// NOTE: Landing appearance config has been moved to page templates.
// Use template's background_config instead of product's landing_appearance_config.
// See TemplateHandler for background configuration.

// validFieldTypes defines allowed field types for warranty custom fields
var validFieldTypes = map[string]bool{
	"text":     true,
	"textarea": true,
	"number":   true,
	"date":     true,
	"select":   true,
	"email":    true,
	"phone":    true,
}

// WarrantyCustomField represents a custom field in warranty_fields_config
type WarrantyCustomField struct {
	ID       string   `json:"id"`
	Label    string   `json:"label"`
	Type     string   `json:"type"`
	Required bool     `json:"required"`
	Options  []string `json:"options,omitempty"`
}

// WarrantyFieldsConfigParsed represents the parsed warranty_fields_config
type WarrantyFieldsConfigParsed struct {
	Enabled      bool                  `json:"enabled"`
	Fields       map[string]string     `json:"fields"`
	CustomFields []WarrantyCustomField `json:"custom_fields"`
}

// validateWarrantyFieldsConfig validates the warranty_fields_config JSON
// Returns error if custom_fields have invalid field types or missing required properties
func validateWarrantyFieldsConfig(config json.RawMessage) error {
	if len(config) == 0 {
		return nil
	}

	var parsed WarrantyFieldsConfigParsed
	if err := json.Unmarshal(config, &parsed); err != nil {
		return fmt.Errorf("invalid warranty_fields_config format: %w", err)
	}

	// Validate custom fields
	seenIDs := make(map[string]bool)
	for i, field := range parsed.CustomFields {
		// Validate required properties
		if field.ID == "" {
			return fmt.Errorf("custom field #%d: id is required", i+1)
		}
		if field.Label == "" {
			return fmt.Errorf("custom field '%s': label is required", field.ID)
		}
		if field.Type == "" {
			return fmt.Errorf("custom field '%s': type is required", field.ID)
		}

		// Validate field type
		if !validFieldTypes[field.Type] {
			return fmt.Errorf("custom field '%s': invalid type '%s'. Valid types: text, textarea, number, date, select, email, phone", field.ID, field.Type)
		}

		// Validate select type has options
		if field.Type == "select" && len(field.Options) == 0 {
			return errors.New("custom field '" + field.ID + "': select type requires at least one option")
		}

		// Check for duplicate IDs
		if seenIDs[field.ID] {
			return fmt.Errorf("custom field '%s': duplicate id", field.ID)
		}
		seenIDs[field.ID] = true
	}

	return nil
}

// VideoEmbed represents a video embed configuration
type VideoEmbed struct {
	Platform    string `json:"platform"`
	VideoID     string `json:"video_id"`
	Autoplay    bool   `json:"autoplay"`
	Caption     string `json:"caption"`
	AspectRatio string `json:"aspect_ratio"` // "landscape" or "portrait"
}

// validAspectRatios defines allowed aspect ratios
var validAspectRatios = map[string]bool{
	"landscape": true,
	"portrait":  true,
	"":          true, // empty allowed, will use platform default
}

// validVideoPlatforms defines allowed video platforms
var validVideoPlatforms = map[string]bool{
	"youtube":   true,
	"tiktok":    true,
	"instagram": true,
}

// validateVideos validates the videos JSON array
// Returns error if videos have invalid platform or missing video_id
func validateVideos(videos json.RawMessage) error {
	if len(videos) == 0 || string(videos) == "[]" || string(videos) == "null" {
		return nil
	}

	var embeds []VideoEmbed
	if err := json.Unmarshal(videos, &embeds); err != nil {
		return fmt.Errorf("invalid videos format: %w", err)
	}

	if len(embeds) > 5 {
		return errors.New("maximum 5 videos allowed")
	}

	for i, v := range embeds {
		if !validVideoPlatforms[v.Platform] {
			return fmt.Errorf("video #%d: invalid platform '%s'. Valid platforms: youtube, tiktok, instagram", i+1, v.Platform)
		}
		if v.VideoID == "" {
			return fmt.Errorf("video #%d: video_id is required", i+1)
		}
		if len(v.VideoID) > 100 {
			return fmt.Errorf("video #%d: video_id too long (max 100 chars)", i+1)
		}
		if len(v.Caption) > 255 {
			return fmt.Errorf("video #%d: caption too long (max 255 chars)", i+1)
		}
		if !validAspectRatios[v.AspectRatio] {
			return fmt.Errorf("video #%d: invalid aspect_ratio '%s'. Valid values: landscape, portrait", i+1, v.AspectRatio)
		}
	}

	return nil
}

