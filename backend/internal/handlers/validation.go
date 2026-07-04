package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gamatritunggal/smartscan/backend/internal/config"
	"github.com/gamatritunggal/smartscan/backend/internal/database"
	"github.com/gamatritunggal/smartscan/backend/internal/models"
	"github.com/gamatritunggal/smartscan/backend/internal/utils"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type ValidationHandler struct {
	DB  *gorm.DB
	Cfg *config.Config
}

func NewValidationHandler(db *gorm.DB, cfg *config.Config) *ValidationHandler {
	return &ValidationHandler{DB: db, Cfg: cfg}
}



type ValidationResult struct {
	IsValid           bool        `json:"is_valid"`
	IsCounterfeit     bool        `json:"is_counterfeit"`
	CounterfeitStatus string      `json:"counterfeit_status,omitempty"` // valid, counterfeit (dynamic only; legacy "warning" treated as valid)
	Message           string      `json:"message"`
	Product           interface{} `json:"product,omitempty"`
	ValidationCount   int         `json:"validation_count"` // For dynamic QR (per-item count)
	QRStatus          string      `json:"qr_status,omitempty"`
	// Batch configuration for flow logic
	NeedWarranty bool   `json:"need_warranty"`
	BatchID      string `json:"batch_id,omitempty"`
	// Batch info (batch_code, production_date, expiry_date)
	Batch interface{} `json:"batch,omitempty"`
	// Tenant info for branding
	Tenant interface{} `json:"tenant,omitempty"`
	// Product certifications and social links (legacy)
	Certifications []interface{} `json:"certifications,omitempty"`
	SocialLinks    []interface{} `json:"social_links,omitempty"` // DEPRECATED: Use SocialAccounts instead
	// NEW: N:M social accounts
	SocialAccounts []interface{} `json:"social_accounts,omitempty"`
	// NEW: Product gallery images
	Images []interface{} `json:"images,omitempty"`
	// NEW: Video embeds
	Videos interface{} `json:"videos,omitempty"`
	// NEW: Website link
	WebsiteURL     string `json:"website_url,omitempty"`
	WebsiteCaption string `json:"website_caption,omitempty"`
	// Display config from product (controls which fields to show)
	DisplayConfig interface{} `json:"display_config,omitempty"`
	// Landing appearance config for background customization (Intermediate+ tier)
	LandingAppearanceConfig interface{} `json:"landing_appearance_config,omitempty"`
	// QR Code ID for counterfeit report submission (dynamic only)
	QRCodeID string `json:"qr_code_id,omitempty"`
	// Readable QR code string for end-user reference (e.g. counterfeit report)
	QRCodeRef string `json:"qr_code_ref,omitempty"`
	// Geofence: distribution zone label for consumer info (e.g. "Semarang Metro")
	DistributionZone string `json:"distribution_zone,omitempty"`
}

// resolveValidationTemplateID resolves the template ID to use for validation
// Priority: batch override → product default → tenant explicit default → tenant oldest template
func (h *ValidationHandler) resolveValidationTemplateID(batch *models.QRBatch) *uuid.UUID {
	if batch == nil {
		return nil
	}

	// 1. Batch override (highest priority)
	if batch.ValidationTemplateID != nil {
		return batch.ValidationTemplateID
	}

	// 2. Product default
	if batch.Product != nil && batch.Product.DefaultValidationTemplateID != nil {
		return batch.Product.DefaultValidationTemplateID
	}

	// 3. Tenant explicit default
	var tenant models.Tenant
	if err := h.DB.Select("default_validation_template_id").
		First(&tenant, "id = ?", batch.TenantID).Error; err == nil {
		if tenant.DefaultValidationTemplateID != nil {
			return tenant.DefaultValidationTemplateID
		}
	}

	// 4. Tenant fallback (oldest active validation template)
	var template models.PageTemplate
	if err := h.DB.Where("tenant_id = ? AND template_type = ? AND is_active = ?",
		batch.TenantID, models.TemplateTypeValidation, true).
		Order("created_at ASC").
		First(&template).Error; err == nil {
		return &template.ID
	}

	return nil
}

// resolveTemplateBackgroundConfig resolves the background config from the validation template
// If background_type is "preset", it fetches the preset and adds the background_url
func (h *ValidationHandler) resolveTemplateBackgroundConfig(batch *models.QRBatch) interface{} {
	if batch == nil {
		return nil
	}

	// Resolve template ID first
	templateID := h.resolveValidationTemplateID(batch)
	if templateID == nil {
		return gin.H{
			"background_type": "none",
		}
	}

	// Fetch the template's background config
	var template models.PageTemplate
	if err := h.DB.Select("background_config").First(&template, "id = ?", templateID).Error; err != nil {
		return gin.H{
			"background_type": "none",
		}
	}

	if template.BackgroundConfig == nil || len(template.BackgroundConfig) == 0 {
		return gin.H{
			"background_type": "none",
		}
	}

	var config models.TemplateBackgroundConfig
	if err := json.Unmarshal(template.BackgroundConfig, &config); err != nil {
		return gin.H{
			"background_type": "none",
		}
	}

	// If background type is "none", return minimal config
	if config.BackgroundType == "none" || config.BackgroundType == "" {
		return gin.H{
			"background_type": "none",
		}
	}

	result := gin.H{
		"background_type":  config.BackgroundType,
		"overlay_color":    config.OverlayColor,
		"overlay_opacity":  config.OverlayOpacity,
		"card_opacity":     config.CardOpacity,
		"card_blur":        config.CardBlur,
	}

	// If preset, fetch the preset and get its background URL
	if config.BackgroundType == "preset" && config.PresetID != nil {
		presetID, err := uuid.Parse(*config.PresetID)
		if err == nil {
			var preset models.ThemePreset
			if err := h.DB.First(&preset, "id = ? AND is_active = true AND deleted_at IS NULL", presetID).Error; err == nil {
				result["background_url"] = preset.BackgroundURL
				result["preset_id"] = preset.ID.String()
			}
		}
	}

	// If custom, use the custom background URL
	if config.BackgroundType == "custom" && config.CustomBackgroundURL != nil {
		result["background_url"] = *config.CustomBackgroundURL
	}

	return result
}

// GetProductInfo returns public product info by QR UUID
func (h *ValidationHandler) GetProductInfo(c *gin.Context) {
	code := c.Param("code")

	// Parse QR code parameter (supports Base58, UUID, hex formats)
	lookup, err := utils.ParseQRCodeParam(code)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	// Lookup QR code
	var qrCode models.QRCode
	var lookupErr error
	if lookup.LookupByCode {
		lookupErr = h.DB.Preload("Batch.Product").Preload("Batch.Tenant").First(&qrCode, "qr_code = ?", lookup.OriginalCode).Error
	} else {
		lookupErr = h.DB.Preload("Batch.Product").Preload("Batch.Tenant").First(&qrCode, "qr_uuid = ?", lookup.QRUUID).Error
	}
	if lookupErr != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Product not found", lookupErr)
		return
	}

	var result gin.H
	if qrCode.Batch != nil && qrCode.Batch.Product != nil {
		result = gin.H{
			"product": gin.H{
				"name":        qrCode.Batch.Product.ProductName,
				"code":        qrCode.Batch.Product.ProductCode,
				"description": qrCode.Batch.Product.Description,
			},
			"production_date": qrCode.Batch.ProductionDate,
			"expiry_date":     qrCode.Batch.ExpiryDate,
		}
		if qrCode.Batch.Tenant != nil {
			result["brand"] = gin.H{"name": qrCode.Batch.Tenant.CompanyName}
		}
	}

	utils.SuccessResponse(c, http.StatusOK, "Product info", result)
}

// ScanRedirect handles QR scan with redirect pattern (like me-qr.com)
// GET /s/:code - records interaction and redirects to /v/:code?s=SESSION_ID
// This prevents scan count from incrementing on page refresh
// The session ID stored in Redis ensures geolocation requirement can't be bypassed
func (h *ValidationHandler) ScanRedirect(c *gin.Context) {
	code := c.Param("code")
	frontendURL := h.Cfg.FrontendURL

	// Generate short session ID and store in Redis
	sessionID, err := utils.GenerateScanSessionID()
	if err != nil {
		// Fallback: redirect without session (no geo requirement)
		log.Printf("Failed to generate session ID: %v", err)
		c.Redirect(http.StatusFound, fmt.Sprintf("%s/v/%s", frontendURL, code))
		return
	}

	// Store session in Redis with 5 min TTL
	// Format: "code:timestamp" for validation
	if database.RedisClient != nil {
		sessionKey := fmt.Sprintf("scan:sess:%s", sessionID)
		sessionData := fmt.Sprintf("%s:%d", code, time.Now().Unix())
		ctx := context.Background()
		if err := database.RedisClient.Set(ctx, sessionKey, sessionData, utils.ScanSessionTTL).Err(); err != nil {
			log.Printf("Failed to store scan session in Redis: %v", err)
		}
	}

	// Redirect with short session param (~43 chars total, under 50!)
	signedRedirect := fmt.Sprintf("%s/v/%s?s=%s", frontendURL, code, sessionID)

	// Parse QR code input to UUID for lookup
	// Supports: Base58 (21-22 chars), UUID string (36 chars), legacy hex (32 chars)
	var qrUUID uuid.UUID
	var lookupByQRCode bool

	inputLen := len(code)
	switch {
	case inputLen >= 21 && inputLen <= 22 && utils.IsBase58UUID(code):
		// Base58 encoded UUID (new format)
		var err error
		qrUUID, err = utils.Base58ToUUID(code)
		if err != nil {
			c.Redirect(http.StatusFound, fmt.Sprintf("%s/v/%s?error=invalid", frontendURL, code))
			return
		}
	case inputLen == 36:
		// Standard UUID format
		var err error
		qrUUID, err = uuid.Parse(code)
		if err != nil {
			c.Redirect(http.StatusFound, fmt.Sprintf("%s/v/%s?error=invalid", frontendURL, code))
			return
		}
	case inputLen == 32:
		// Legacy hex format
		lookupByQRCode = true
	default:
		c.Redirect(http.StatusFound, fmt.Sprintf("%s/v/%s?error=invalid", frontendURL, code))
		return
	}

	// Find QR code with batch, product, and tenant
	var qrCode models.QRCode
	var lookupErr error

	if lookupByQRCode {
		lookupErr = h.DB.Preload("Batch.Product").Preload("Batch.Tenant").First(&qrCode, "qr_code = ?", code).Error
	} else {
		lookupErr = h.DB.Preload("Batch.Product").Preload("Batch.Tenant").First(&qrCode, "qr_uuid = ?", qrUUID).Error
	}

	if lookupErr != nil {
		c.Redirect(http.StatusFound, fmt.Sprintf("%s/v/%s?error=not_found", frontendURL, code))
		return
	}

	// IsScannable() combines status + counterfeit_status.
	if !qrCode.IsScannable() {
		c.Redirect(http.StatusFound, signedRedirect)
		return
	}

	// A missing batch (soft-deleted / orphaned) or an explicitly soft-deleted batch
	// is treated as "not found". recordDynamicQRScan dereferences qrCode.Batch, so a
	// nil batch must be caught here rather than panicking downstream.
	if qrCode.Batch == nil || qrCode.Batch.DeletedAt != nil {
		c.Redirect(http.StatusFound, fmt.Sprintf("%s/v/%s?error=not_found", frontendURL, code))
		return
	}

	h.recordDynamicQRScan(c, &qrCode)
	c.Redirect(http.StatusFound, signedRedirect)
}


// recordDynamicQRScan records a scan for dynamic QR with counterfeit detection
func (h *ValidationHandler) recordDynamicQRScan(c *gin.Context, qrCode *models.QRCode) {
	// Resolve counterfeit threshold via 4-level hierarchy: QR > Batch > Product > Tenant > Default(3)
	threshold := ResolveCounterfeitThreshold(h.DB, qrCode)

	// Resolve template ID for A/B testing analytics
	templateID := h.resolveValidationTemplateID(qrCode.Batch)

	// Record interaction FIRST
	interaction := models.Interaction{
		QRCodeID:               &qrCode.ID,
		TenantID:               qrCode.Batch.TenantID,
		InteractionCategory:    models.InteractionCategoryEndUserAccess,
		InteractionSubcategory: models.InteractionSubcategoryProductValidation,
		InteractionStatus:      models.InteractionStatusSuccess,
		IPAddress:              c.ClientIP(),
		UserAgent:              c.GetHeader("User-Agent"),
		ValidationTemplateID:   templateID,
	}
	h.DB.Create(&interaction)

	// 0 = disabled, skip counterfeit check (consistent with scanning.go)
	if threshold == 0 {
		return
	}

	// Count validations AFTER insert to avoid race condition
	var validationCount int64
	h.DB.Model(&models.Interaction{}).Where(
		"qr_code_id = ? AND interaction_subcategory = ?",
		qrCode.ID, models.InteractionSubcategoryProductValidation,
	).Count(&validationCount)

	scanCount := int(validationCount)

	// Determine counterfeit status based on threshold (2-state: valid or counterfeit)
	var counterfeitStatus models.CounterfeitStatus
	if scanCount > threshold {
		counterfeitStatus = models.CounterfeitStatusCounterfeit
	} else {
		counterfeitStatus = models.CounterfeitStatusValid
	}

	// Update QR code counterfeit status if changed
	if qrCode.CounterfeitStatus != counterfeitStatus {
		h.DB.Model(&qrCode).Update("counterfeit_status", counterfeitStatus)
	}

	// Create counterfeit detection for excessive validations
	if scanCount > threshold {
		var existing models.CounterfeitDetection
		if err := h.DB.Where("qr_code_id = ? AND status = ?", qrCode.ID, "active").First(&existing).Error; err != nil {
			interactionIDs, _ := json.Marshal([]string{interaction.ID.String()})
			now := time.Now().UTC()
			detection := models.CounterfeitDetection{
				QRCodeID:               qrCode.ID,
				TenantID:               qrCode.Batch.TenantID,
				DetectionReason:        fmt.Sprintf("Excessive validation attempts: %d (threshold: %d)", scanCount, threshold),
				InteractionIDs:         datatypes.JSON(interactionIDs),
				TotalInteractionsCount: scanCount,
				FirstInteractionAt:     &now,
				LastInteractionAt:      &now,
				Status:                 models.CounterfeitDetectionStatusActive,
			}
			h.DB.Create(&detection)
		} else {
			// Update existing detection with latest scan count
			now := time.Now().UTC()
			h.DB.Model(&existing).Updates(map[string]interface{}{
				"total_interactions_count": scanCount,
				"last_interaction_at":      now,
			})
		}
	}
}

// GetValidationInfo returns validation data WITHOUT recording a new interaction
// GET /api/v1/public/validate-info/:code
// Used by frontend after redirect from /s/:code to prevent double counting
func (h *ValidationHandler) GetValidationInfo(c *gin.Context) {
	code := c.Param("code")

	// Parse QR code input to UUID for lookup
	var qrUUID uuid.UUID
	var lookupByQRCode bool

	inputLen := len(code)
	switch {
	case inputLen >= 21 && inputLen <= 22 && utils.IsBase58UUID(code):
		var err error
		qrUUID, err = utils.Base58ToUUID(code)
		if err != nil {
			utils.SuccessResponse(c, http.StatusOK, "Validation result", ValidationResult{
				IsValid:       false,
				IsCounterfeit: true,
				Message:       "Invalid QR code format",
			})
			return
		}
	case inputLen == 36:
		var err error
		qrUUID, err = uuid.Parse(code)
		if err != nil {
			utils.SuccessResponse(c, http.StatusOK, "Validation result", ValidationResult{
				IsValid:       false,
				IsCounterfeit: true,
				Message:       "Invalid QR code format",
			})
			return
		}
	case inputLen == 32:
		lookupByQRCode = true
	default:
		utils.SuccessResponse(c, http.StatusOK, "Validation result", ValidationResult{
			IsValid:       false,
			IsCounterfeit: true,
			Message:       "Invalid QR code format",
		})
		return
	}

	// Find QR code
	var qrCode models.QRCode
	var lookupErr error

	if lookupByQRCode {
		lookupErr = h.DB.Preload("Batch.Product").Preload("Batch.Tenant").First(&qrCode, "qr_code = ?", code).Error
	} else {
		lookupErr = h.DB.Preload("Batch.Product").Preload("Batch.Tenant").First(&qrCode, "qr_uuid = ?", qrUUID).Error
	}

	if lookupErr != nil {
		utils.SuccessResponse(c, http.StatusOK, "Validation result", ValidationResult{
			IsValid:       false,
			IsCounterfeit: true,
			Message:       "Invalid QR code - Product not found",
		})
		return
	}

	// Check if batch has been soft-deleted
	if qrCode.Batch != nil && qrCode.Batch.DeletedAt != nil {
		utils.SuccessResponse(c, http.StatusOK, "Validation result", ValidationResult{
			IsValid:       false,
			IsCounterfeit: true,
			Message:       "Invalid QR code - Product not found",
		})
		return
	}

	// IsScannable() combines status + counterfeit_status.
	if !qrCode.IsScannable() {
		utils.SuccessResponse(c, http.StatusOK, "Validation result", ValidationResult{
			IsValid:       false,
			IsCounterfeit: false,
			Message:       "QR code is not active",
			QRStatus:      string(qrCode.Status),
		})
		return
	}

	// Verify batch and product relationships are loaded
	// If product was soft-deleted, reload with Unscoped to still show product info
	if qrCode.Batch == nil || qrCode.Batch.Product == nil || qrCode.Batch.Tenant == nil {
		// Try reloading with Unscoped to include soft-deleted products
		var batch models.QRBatch
		if err := h.DB.Unscoped().
			Preload("Product", func(db *gorm.DB) *gorm.DB { return db.Unscoped() }).
			Preload("Tenant").
			First(&batch, "id = ?", qrCode.BatchID).Error; err == nil {
			qrCode.Batch = &batch
		}

		// If still missing, return error
		if qrCode.Batch == nil || qrCode.Batch.Product == nil {
			utils.SuccessResponse(c, http.StatusOK, "Validation result", ValidationResult{
				IsValid:       false,
				IsCounterfeit: false,
				Message:       "Product information unavailable",
				QRStatus:      string(qrCode.Status),
			})
			return
		}
	}

	h.getDynamicQRInfo(c, &qrCode)
}

// getDynamicQRInfo returns dynamic QR info without recording
func (h *ValidationHandler) getDynamicQRInfo(c *gin.Context, qrCode *models.QRCode) {
	// Get existing validation count (no increment)
	var validationCount int64
	h.DB.Model(&models.Interaction{}).Where(
		"qr_code_id = ? AND interaction_subcategory = ?",
		qrCode.ID, models.InteractionSubcategoryProductValidation,
	).Count(&validationCount)

	scanCount := int(validationCount)

	// Determine counterfeit status from stored value
	counterfeitStatus := qrCode.CounterfeitStatus
	var isCounterfeit bool
	var statusMessage string

	switch counterfeitStatus {
	case models.CounterfeitStatusCounterfeit:
		isCounterfeit = true
		statusMessage = "Warning: This product may be counterfeit. Multiple validation attempts detected."
	default:
		// "valid" and legacy "warning" records are both treated as authentic
		isCounterfeit = false
		statusMessage = "This is an authentic product"
	}

	// Prepare product info - initialize slices as empty (not nil) for consistent JSON
	var productInfo gin.H
	certifications := make([]interface{}, 0)
	socialLinks := make([]interface{}, 0)
	socialAccounts := make([]interface{}, 0)
	images := make([]interface{}, 0)
	var videos interface{}
	var websiteURL, websiteCaption string

	if qrCode.Batch != nil && qrCode.Batch.Product != nil {
		product := qrCode.Batch.Product
		productInfo = gin.H{
			"name":        product.ProductName,
			"code":        product.ProductCode,
			"description": product.Description,
		}
		if qrCode.Batch.Tenant != nil {
			productInfo["brand"] = qrCode.Batch.Tenant.CompanyName
		}

		// Fetch product certifications
		var productCerts []models.ProductCertification
		h.DB.Where("product_id = ?", product.ID).
			Preload("CertificationType").
			Preload("CertificationType.Country").
			Find(&productCerts)

		for _, cert := range productCerts {
			if cert.CertificationType != nil {
				countryName := "International"
				if cert.CertificationType.Country != nil {
					countryName = cert.CertificationType.Country.Name
				}
				certifications = append(certifications, gin.H{
					"name":                cert.CertificationType.Name,
					"code":                cert.CertificationType.Code,
					"country":             countryName,
					"registration_number": cert.RegistrationNumber,
					"logo_url":            cert.CertificationType.LogoURL,
					"website_url":         cert.CertificationType.WebsiteURL,
				})
			}
		}

		// Fetch product social links (legacy - deprecated)
		var productSocials []models.ProductSocialLink
		h.DB.Where("product_id = ?", product.ID).
			Preload("Platform").
			Find(&productSocials)

		for _, link := range productSocials {
			if link.Platform != nil {
				socialLinks = append(socialLinks, gin.H{
					"platform":          link.Platform.Name,
					"code":              link.Platform.Code,
					"icon":              link.Platform.Icon,
					"handle_or_url":     link.HandleOrURL,
					"base_url":          link.Platform.BaseURL,
					"deep_link_pattern": link.Platform.DeepLinkPattern,
				})
			}
		}

		// Fetch product social accounts (N:M - new structure)
		var accountLinks []models.ProductSocialAccountLink
		h.DB.Where("product_id = ?", product.ID).
			Preload("SocialAccount.Platform").
			Order("sort_order ASC").
			Find(&accountLinks)

		for _, link := range accountLinks {
			if link.SocialAccount != nil && link.SocialAccount.Platform != nil {
				// Build URL from base_url + handle or use custom account_url
				accountURL := link.SocialAccount.AccountURL
				if accountURL == "" && link.SocialAccount.Platform.BaseURL != "" {
					accountURL = link.SocialAccount.Platform.BaseURL + link.SocialAccount.AccountHandle
				}
				socialAccounts = append(socialAccounts, gin.H{
					"platform_code":  link.SocialAccount.Platform.Code,
					"platform_name":  link.SocialAccount.Platform.Name,
					"platform_icon":  link.SocialAccount.Platform.Icon,
					"account_handle": link.SocialAccount.AccountHandle,
					"url":            accountURL,
				})
			}
		}

		// Fetch product images
		var productImages []models.ProductImage
		h.DB.Where("product_id = ?", product.ID).
			Order("sort_order ASC").
			Find(&productImages)

		for _, img := range productImages {
			images = append(images, gin.H{
				"image_url": img.ImageURL,
				"caption":   img.Caption,
				"is_main":   img.IsMain,
			})
		}

		// Get videos from product
		if len(product.Videos) > 0 && string(product.Videos) != "[]" && string(product.Videos) != "null" {
			var videoList []map[string]interface{}
			if err := json.Unmarshal(product.Videos, &videoList); err == nil {
				videos = videoList
			}
		}

		// Get website URL and caption
		websiteURL = product.WebsiteURL
		websiteCaption = product.WebsiteCaption
	}

	// Prepare tenant info for branding
	var tenantInfo gin.H
	if qrCode.Batch != nil && qrCode.Batch.Tenant != nil {
		tenantInfo = gin.H{
			"company_name": qrCode.Batch.Tenant.CompanyName,
			"brand_name":   qrCode.Batch.Tenant.CompanyName,
			"logo_url":     qrCode.Batch.LogoURL,
		}
	}

	// Prepare batch info
	var batchInfo gin.H
	var distributionZone string
	if qrCode.Batch != nil {
		batchInfo = gin.H{
			"batch_code":      qrCode.Batch.BatchCode,
			"production_date": qrCode.Batch.ProductionDate,
			"expiry_date":     qrCode.Batch.ExpiryDate,
		}
		// Include geofence distribution zone label (subtle info for consumer)
		if qrCode.Batch.GeofenceEnabled && qrCode.Batch.GeofenceLabel != "" {
			distributionZone = qrCode.Batch.GeofenceLabel
		}
	}

	// Get display config from product
	var displayConfig interface{}
	if qrCode.Batch != nil && qrCode.Batch.Product != nil {
		displayConfig = qrCode.Batch.Product.DisplayConfig
	}

	// Get background config from template
	var landingAppearanceConfig interface{}
	if qrCode.Batch != nil {
		landingAppearanceConfig = h.resolveTemplateBackgroundConfig(qrCode.Batch)
	}

	// Get batch flags for flow logic
	var needWarranty bool
	var batchID string
	if qrCode.Batch != nil {
		// Warranty is controlled at product level
		if qrCode.Batch.Product != nil {
			needWarranty = qrCode.Batch.Product.WarrantyEnabled
		}
		batchID = qrCode.Batch.ID.String()
	}

	// Convert to []interface{} for response
	var certsInterface []interface{}
	for _, cert := range certifications {
		certsInterface = append(certsInterface, cert)
	}
	var socialsInterface []interface{}
	for _, link := range socialLinks {
		socialsInterface = append(socialsInterface, link)
	}

	utils.SuccessResponse(c, http.StatusOK, "Validation result", ValidationResult{
		IsValid:                 true,
		IsCounterfeit:           isCounterfeit,
		CounterfeitStatus:       string(counterfeitStatus),
		Message:                 statusMessage,
		Product:                 productInfo,
		ValidationCount:         scanCount,
		QRStatus:                string(qrCode.Status),
		NeedWarranty:            needWarranty,
		BatchID:                 batchID,
		Batch:                   batchInfo,
		Tenant:                  tenantInfo,
		Certifications:          certsInterface,
		SocialLinks:             socialsInterface,
		SocialAccounts:          socialAccounts,
		Images:                  images,
		Videos:                  videos,
		WebsiteURL:              websiteURL,
		WebsiteCaption:          websiteCaption,
		DisplayConfig:           displayConfig,
		LandingAppearanceConfig: landingAppearanceConfig,
		QRCodeID:                qrCode.ID.String(),
		QRCodeRef:               qrCode.QRCode,
		DistributionZone:        distributionZone,
	})
}

// UpdateScanLocationRequest is the request body for updating scan location
type UpdateScanLocationRequest struct {
	QRCode    string  `json:"qr_code" binding:"required"`
	Latitude  float64 `json:"latitude" binding:"required"`
	Longitude float64 `json:"longitude" binding:"required"`
	Accuracy  float64 `json:"accuracy"`
}

// UpdateScanLocation updates the geolocation of the most recent scan interaction
// POST /api/v1/public/scan-location
// This is called by the frontend after getting user's geolocation permission
func (h *ValidationHandler) UpdateScanLocation(c *gin.Context) {
	var req UpdateScanLocationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request", nil)
		return
	}

	// Validate coordinates
	if req.Latitude < -90 || req.Latitude > 90 || req.Longitude < -180 || req.Longitude > 180 {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid coordinates", nil)
		return
	}

	// Parse QR code to find the QR code record
	lookup, err := utils.ParseQRCodeParam(req.QRCode)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid QR code format", nil)
		return
	}

	// First try to find as dynamic QR code
	var qrCode models.QRCode
	var qrCodeFound bool
	if lookup.LookupByCode {
		qrCodeFound = h.DB.First(&qrCode, "qr_code = ?", lookup.OriginalCode).Error == nil
	} else {
		qrCodeFound = h.DB.First(&qrCode, "qr_uuid = ?", lookup.QRUUID).Error == nil
	}

	// Find the most recent interaction for this QR code within last 5 minutes
	// This prevents abuse and ensures we're updating the correct scan
	cutoffTime := time.Now().Add(-5 * time.Minute)
	var interaction models.Interaction

	if !qrCodeFound {
		// QR code not found - silently succeed to not leak info
		utils.SuccessResponse(c, http.StatusOK, "Location noted", nil)
		return
	}
	if err := h.DB.Where(
		"qr_code_id = ? AND interaction_subcategory = ? AND created_at > ? AND geolocation IS NULL",
		qrCode.ID, models.InteractionSubcategoryProductValidation, cutoffTime,
	).Order("created_at DESC").First(&interaction).Error; err != nil {
		// No recent interaction found or already has geolocation - silently succeed
		utils.SuccessResponse(c, http.StatusOK, "Location noted", nil)
		return
	}

	// Impossible-travel check — must run BEFORE this interaction's geolocation
	// is written, so the comparison targets the PREVIOUS geolocated scan.
	// A clone of this label scanned in another city minutes apart trips this.
	if qrCodeFound {
		if exceeded, reason := checkVelocityAnomalyShared(h.DB, interaction.TenantID, qrCode.ID, req.Latitude, req.Longitude); exceeded {
			go createCounterfeitDetection(h.DB, interaction.TenantID, qrCode.ID, reason, interaction.ID)
		}
	}

	// Build geolocation JSON
	geoData := map[string]float64{
		"lat": req.Latitude,
		"lng": req.Longitude,
	}
	if req.Accuracy > 0 {
		geoData["accuracy"] = req.Accuracy
	}
	geoJSON, _ := json.Marshal(geoData)

	// Update the interaction with geolocation
	if err := h.DB.Model(&interaction).Update("geolocation", datatypes.JSON(geoJSON)).Error; err != nil {
		log.Printf("Failed to update scan location: %v", err)
		// Still return success to not block the user
		utils.SuccessResponse(c, http.StatusOK, "Location noted", nil)
		return
	}

	// Geofence check (non-blocking goroutine - never delays consumer response)
	if qrCodeFound {
		go checkBatchGeofence(h.DB, h.Cfg, interaction.ID, qrCode.BatchID, qrCode.ID, nil, req.Latitude, req.Longitude, req.Accuracy)
	}

	// Reverse geocoding enrichment (non-blocking goroutine - adds city/province/country)
	if h.Cfg.Geocoding.BigDataCloudAPIKey != "" {
		go enrichGeolocation(h.DB, h.Cfg.Geocoding.BigDataCloudAPIKey, interaction.ID, req.Latitude, req.Longitude)
	}

	utils.SuccessResponse(c, http.StatusOK, "Location updated", nil)
}

// checkBatchGeofence checks if a scan location is outside the batch's geofence zone
// and records a violation if so. Runs in a goroutine to avoid blocking consumer response.
func checkBatchGeofence(db *gorm.DB, cfg *config.Config, interactionID, batchID, qrCodeID uuid.UUID, productID *uuid.UUID, scanLat, scanLng, gpsAccuracy float64) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered panic in checkBatchGeofence (batch=%s, interaction=%s): %v", batchID, interactionID, r)
		}
	}()

	// Skip if scan coordinates are (0,0) — this typically means "no GPS data" from the browser.
	// Note: (0,0) is technically a valid coordinate (Null Island, Gulf of Guinea) but in practice
	// browser Geolocation API returns 0,0 when GPS is unavailable.
	if scanLat == 0 && scanLng == 0 {
		log.Printf("Geofence check skipped: null island coordinates (0,0) for interaction=%s", interactionID)
		return
	}

	var batch models.QRBatch
	if err := db.Select("id, tenant_id, product_id, batch_name, geofence_enabled, geofence_latitude, geofence_longitude, geofence_radius_km, geofence_label").
		First(&batch, "id = ?", batchID).Error; err != nil {
		return
	}

	if !batch.GeofenceEnabled || batch.GeofenceLatitude == nil ||
		batch.GeofenceLongitude == nil || batch.GeofenceRadiusKm == nil {
		return
	}

	distanceMeters := utils.HaversineDistance(scanLat, scanLng, *batch.GeofenceLatitude, *batch.GeofenceLongitude)
	distanceKm := distanceMeters / 1000.0
	radiusKm := *batch.GeofenceRadiusKm

	// Apply dynamic GPS buffer to reduce false positives
	// Uses actual GPS accuracy when it exceeds the 2km default, capped at 10km
	gpsBufferKm := 2.0
	if gpsAccuracy > 0 {
		accuracyKm := gpsAccuracy / 1000.0
		if accuracyKm > gpsBufferKm {
			gpsBufferKm = accuracyKm
		}
		if gpsBufferKm > 10.0 {
			gpsBufferKm = 10.0
		}
	}
	distanceFromEdge := distanceKm - radiusKm - gpsBufferKm

	if distanceFromEdge <= 0 {
		return // Inside zone or within buffer
	}

	severity := utils.GeofenceSeverity(distanceFromEdge)

	if productID == nil {
		productID = &batch.ProductID
	}

	violation := models.GeofenceViolation{
		TenantID:             batch.TenantID,
		BatchID:              batch.ID,
		QRCodeID:             &qrCodeID,
		ProductID:            productID,
		InteractionID:        &interactionID,
		ScanLatitude:         scanLat,
		ScanLongitude:        scanLng,
		DistanceFromCenterKm: distanceKm,
		DistanceFromEdgeKm:   distanceFromEdge,
		Severity:             severity,
	}

	// Store GPS accuracy if available
	if gpsAccuracy > 0 {
		violation.GPSAccuracyMeters = &gpsAccuracy
	}

	if err := db.Create(&violation).Error; err != nil {
		// Handle duplicate key error gracefully (concurrent goroutines for same interaction)
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "idx_geofence_violations_interaction_batch") {
			log.Printf("Duplicate geofence violation skipped (interaction=%s, batch=%s)", interactionID, batchID)
			return
		}
		log.Printf("Failed to record geofence violation: %v", err)
		return
	}

	// Queue email notification for high/critical severity violations
	if severity == "high" || severity == "critical" {
		go func() {
			var admin models.TenantStaff
			if err := db.Preload("User").Where("tenant_id = ? AND is_primary_admin = ?", batch.TenantID, true).
				First(&admin).Error; err != nil || admin.User == nil {
				return
			}

			// Get product name
			productName := "Unknown Product"
			if productID != nil {
				var product models.Product
				if err := db.Select("product_name").First(&product, "id = ?", *productID).Error; err == nil {
					productName = product.ProductName
				}
			}

			zoneLabel := "Distribution Zone"
			if batch.GeofenceLabel != "" {
				zoneLabel = batch.GeofenceLabel
			}

			Notify(db, batch.TenantID, models.NotificationTypeGeofenceViolation,
				"Out-of-zone scan detected",
				fmt.Sprintf("%s (batch %s) scanned %.1f km outside %s — severity %s",
					productName, batch.BatchName, distanceFromEdge, zoneLabel, strings.ToUpper(severity)),
				"/tenant/geofence",
				map[string]interface{}{
					"product_name":       productName,
					"batch_name":         batch.BatchName,
					"zone_label":         zoneLabel,
					"severity":           severity,
					"distance_from_edge": distanceFromEdge,
				})
		}()
	}
}

// enrichGeolocation enriches an interaction's geolocation JSON with city, province, and country
// from BigDataCloud reverse geocoding API. Runs in a goroutine, never blocks consumer response.
func enrichGeolocation(db *gorm.DB, apiKey string, interactionID uuid.UUID, lat, lng float64) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered panic in enrichGeolocation (interaction=%s): %v", interactionID, r)
		}
	}()

	result, err := utils.ReverseGeocode(lat, lng, apiKey)
	if err != nil {
		log.Printf("Reverse geocode failed for interaction=%s: %v", interactionID, err)
		return
	}

	if result.City == "" && result.Province == "" && result.Country == "" {
		return
	}

	// Build update using jsonb_set to merge into existing geolocation
	updates := map[string]interface{}{}
	if result.City != "" {
		updates["city"] = result.City
	}
	if result.Province != "" {
		updates["province"] = result.Province
	}
	if result.Country != "" {
		updates["country"] = result.Country
	}
	if result.CountryCode != "" {
		updates["country_code"] = result.CountryCode
	}

	// Read existing geolocation, merge, and write back
	var interaction models.Interaction
	if err := db.Select("id, geolocation").First(&interaction, "id = ?", interactionID).Error; err != nil {
		return
	}

	var geoData map[string]interface{}
	if interaction.Geolocation != nil {
		if err := json.Unmarshal(interaction.Geolocation, &geoData); err != nil {
			geoData = map[string]interface{}{}
		}
	} else {
		geoData = map[string]interface{}{}
	}

	for k, v := range updates {
		geoData[k] = v
	}

	geoJSON, err := json.Marshal(geoData)
	if err != nil {
		return
	}

	db.Model(&models.Interaction{}).Where("id = ?", interactionID).
		Update("geolocation", datatypes.JSON(geoJSON))
}

// VerifyScanSession verifies the session ID from scan redirect
// GET /api/v1/public/verify-scan-session?code=X&s=Y
// Returns { geo_required: true } if session is valid and not expired
func (h *ValidationHandler) VerifyScanSession(c *gin.Context) {
	code := c.Query("code")
	sessionID := c.Query("s")

	// If session param is missing, no geolocation required (direct URL access)
	if sessionID == "" {
		utils.SuccessResponse(c, http.StatusOK, "Session verification", gin.H{
			"geo_required": false,
		})
		return
	}

	// Check if Redis is available
	if database.RedisClient == nil {
		utils.SuccessResponse(c, http.StatusOK, "Session verification", gin.H{
			"geo_required": false,
		})
		return
	}

	// Check Redis for session
	sessionKey := fmt.Sprintf("scan:sess:%s", sessionID)
	ctx := context.Background()
	sessionData, err := database.RedisClient.Get(ctx, sessionKey).Result()

	if err != nil || sessionData == "" {
		// Session expired or invalid
		utils.SuccessResponse(c, http.StatusOK, "Session verification", gin.H{
			"geo_required": false,
		})
		return
	}

	// Parse session data: "code:timestamp"
	parts := strings.Split(sessionData, ":")
	if len(parts) < 2 || parts[0] != code {
		// Code mismatch - possible URL manipulation
		utils.SuccessResponse(c, http.StatusOK, "Session verification", gin.H{
			"geo_required": false,
		})
		return
	}

	// Valid session - require geolocation
	utils.SuccessResponse(c, http.StatusOK, "Session verification", gin.H{
		"geo_required": true,
	})
}
