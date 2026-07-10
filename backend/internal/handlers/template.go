package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gamatritunggal/smartscan/backend/internal/config"
	"github.com/gamatritunggal/smartscan/backend/internal/models"
	"github.com/gamatritunggal/smartscan/backend/internal/sentry"
	"github.com/gamatritunggal/smartscan/backend/internal/storage"
	"github.com/gamatritunggal/smartscan/backend/internal/utils"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Valid section IDs for section_order in custom_fields
var validSectionIDs = map[string]bool{
	"images":           true,
	"videos":           true,
	"social_accounts":  true,
	"certifications":   true,
	"website_link":     true,
	"description":      true,
	"warranty_button":  true,
}

// Maximum size for custom_fields JSON (10KB)
const maxCustomFieldsSize = 10 * 1024

// validateCustomFields validates the custom_fields map
func validateCustomFields(fields map[string]interface{}) error {
	if fields == nil {
		return nil
	}

	// Check max size
	jsonBytes, err := json.Marshal(fields)
	if err != nil {
		return fmt.Errorf("invalid custom_fields format: %w", err)
	}
	if len(jsonBytes) > maxCustomFieldsSize {
		return fmt.Errorf("custom_fields exceeds maximum size of %d bytes", maxCustomFieldsSize)
	}

	// Validate section_order if present
	if sectionOrder, ok := fields["section_order"]; ok {
		if err := validateSectionOrder(sectionOrder); err != nil {
			return err
		}
	}

	return nil
}

// validateSectionOrder validates the section_order array
func validateSectionOrder(order interface{}) error {
	arr, ok := order.([]interface{})
	if !ok {
		return fmt.Errorf("section_order must be an array")
	}

	if len(arr) == 0 {
		return fmt.Errorf("section_order cannot be empty, use null to reset to defaults")
	}

	if len(arr) > 20 {
		return fmt.Errorf("section_order cannot have more than 20 items")
	}

	seen := make(map[string]bool)
	for i, item := range arr {
		id, ok := item.(string)
		if !ok {
			return fmt.Errorf("section_order[%d] must be a string", i)
		}
		if !validSectionIDs[id] {
			return fmt.Errorf("invalid section_id at index %d: %s", i, id)
		}
		if seen[id] {
			return fmt.Errorf("duplicate section_id: %s", id)
		}
		seen[id] = true
	}

	return nil
}

// Hex color regex pattern
var hexColorRegex = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

// validateBackgroundConfig validates the background_config map
func validateBackgroundConfig(config map[string]interface{}) error {
	if config == nil {
		return nil
	}

	// Check max size (5KB should be plenty for background config)
	jsonBytes, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("invalid background_config format: %w", err)
	}
	if len(jsonBytes) > 5*1024 {
		return fmt.Errorf("background_config exceeds maximum size")
	}

	// Validate color fields
	colorFields := []string{"overlay_color", "card_bg_color", "text_color", "label_color"}
	for _, field := range colorFields {
		if val, ok := config[field]; ok {
			if color, ok := val.(string); ok {
				if color != "" && !hexColorRegex.MatchString(color) {
					return fmt.Errorf("invalid hex color for %s: %s", field, color)
				}
			}
		}
	}

	// Validate opacity fields (0-100)
	opacityFields := []string{"overlay_opacity", "card_opacity"}
	for _, field := range opacityFields {
		if val, ok := config[field]; ok {
			var opacity float64
			switch v := val.(type) {
			case float64:
				opacity = v
			case int:
				opacity = float64(v)
			default:
				continue // Skip non-numeric values
			}
			if opacity < 0 || opacity > 100 {
				return fmt.Errorf("%s must be between 0 and 100", field)
			}
		}
	}

	// Validate blur field (0-20)
	if val, ok := config["card_blur"]; ok {
		var blur float64
		switch v := val.(type) {
		case float64:
			blur = v
		case int:
			blur = float64(v)
		default:
			blur = 0
		}
		if blur < 0 || blur > 20 {
			return fmt.Errorf("card_blur must be between 0 and 20")
		}
	}

	// Validate custom_background_url if present
	if val, ok := config["custom_background_url"]; ok {
		if urlStr, ok := val.(string); ok && urlStr != "" {
			// Must be a valid URL
			parsed, err := url.Parse(urlStr)
			if err != nil {
				return fmt.Errorf("invalid custom_background_url: %w", err)
			}
			// Must be http or https
			if parsed.Scheme != "http" && parsed.Scheme != "https" {
				return fmt.Errorf("custom_background_url must use http or https scheme")
			}
			// Block javascript: and data: URLs
			lowerURL := strings.ToLower(urlStr)
			if strings.HasPrefix(lowerURL, "javascript:") || strings.HasPrefix(lowerURL, "data:text") {
				return fmt.Errorf("custom_background_url contains forbidden scheme")
			}
		}
	}

	return nil
}

type TemplateHandler struct {
	DB  *gorm.DB
	Cfg *config.Config
}

func NewTemplateHandler(db *gorm.DB, cfg *config.Config) *TemplateHandler {
	return &TemplateHandler{DB: db, Cfg: cfg}
}

type CreateTemplateRequest struct {
	TemplateName     string                 `json:"template_name" binding:"required,max=255"`
	TemplateType     string                 `json:"template_type" binding:"required,oneof=validation warranty"`
	HTMLContent      string                 `json:"html_content" binding:"required,max=100000"`
	CSSContent       string                 `json:"css_content" binding:"max=50000"`
	JSContent        string                 `json:"js_content" binding:"max=50000"`
	CustomFields     map[string]interface{} `json:"custom_fields"`
	BackgroundConfig map[string]interface{} `json:"background_config"`
	IsActive         bool                   `json:"is_active"`
}

type UpdateTemplateRequest struct {
	TemplateName     string                 `json:"template_name" binding:"max=255"`
	HTMLContent      string                 `json:"html_content" binding:"max=100000"`
	CSSContent       string                 `json:"css_content" binding:"max=50000"`
	JSContent        string                 `json:"js_content" binding:"max=50000"`
	CustomFields     map[string]interface{} `json:"custom_fields"`
	BackgroundConfig map[string]interface{} `json:"background_config"`
	IsActive         *bool                  `json:"is_active"`
}

// ensureDefaultTemplates creates default validation and warranty templates if they don't exist
func (h *TemplateHandler) ensureDefaultTemplates(tenantID uuid.UUID) error {
	// Default custom_fields for validation templates (must match CSS colors)
	defaultValidationCustomFields := datatypes.JSON(`{"header":{"bg_color":"#3b82f6","badge_text":"Authentic Product","badge_bg_color":"#22c55e","badge_text_color":"#ffffff","logo_enabled":false},"styling":{"card_bg_color":"#ffffff","field_bg_color":"#f9fafb","text_color":"#1f2937"},"warranty_button":{"text":"Activate Warranty","bg_color":"#9333ea","text_color":"#ffffff"}}`)

	// Check and create validation template
	var validationTemplate models.PageTemplate
	if err := h.DB.Where("tenant_id = ? AND template_type = ?", tenantID, models.TemplateTypeValidation).First(&validationTemplate).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// Create default validation template (uses section placeholders to respect display_config)
			validationTemplate = models.PageTemplate{
				TenantID:     tenantID,
				TemplateType: models.TemplateTypeValidation,
				TemplateName: "Default Validation Template",
				HTMLContent: `<div class="validation-page">
	<div class="header">
		<img src="{{logo_url}}" alt="Logo" class="logo" onerror="this.style.display='none'" />
		<div class="badge">
			<svg class="badge-icon" viewBox="0 0 20 20" fill="currentColor">
				<path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd" />
			</svg>
			Authentic Product
		</div>
	</div>
	<div class="content">
		<div class="product-section">
			<h1 class="product-name">{{product_name}}</h1>
		</div>
		{{product_code_section}}
		{{brand_name_section}}
		{{batch_code_section}}
		{{production_date_section}}
		{{expiry_date_section}}
		{{verification_count_section}}
		{{certifications_section}}
		{{social_links_section}}
		{{action_buttons_section}}
	</div>
</div>`,
				CSSContent: `.validation-page{font-family:system-ui,-apple-system,sans-serif;max-width:480px;margin:0 auto;min-height:100vh;background:#f3f4f6}.header{background:#3b82f6;padding:24px 16px;text-align:center}.logo{max-height:60px;margin-bottom:12px}.badge{display:inline-flex;align-items:center;gap:6px;background:#22c55e;color:white;padding:6px 12px;border-radius:9999px;font-size:12px;font-weight:500}.badge-icon{width:14px;height:14px}.content{padding:16px}.product-section{text-align:center;margin-bottom:16px}.product-name{font-size:20px;font-weight:700;color:#1f2937;margin:0 0 4px}.product-code{font-size:14px;color:#6b7280;margin:0}.field-card{background:white;border-radius:8px;padding:12px;margin-bottom:12px}.field-label{font-size:10px;text-transform:uppercase;letter-spacing:0.05em;color:#6b7280;margin:0 0 4px}.field-value{font-size:16px;font-weight:500;color:#1f2937;margin:0}.section-card{background:white;border-radius:8px;padding:12px;margin-bottom:12px}.section-title{font-size:10px;text-transform:uppercase;letter-spacing:0.05em;color:#6b7280;margin:0 0 8px;font-weight:600}.cert-list{display:flex;flex-wrap:wrap;gap:8px}.cert-item{display:flex;align-items:center;gap:6px;padding:6px 10px;background:#f9fafb;border-radius:6px;text-decoration:none;color:#374151;font-size:13px}.cert-logo{width:20px;height:20px;object-fit:contain}.cert-name{font-weight:500}.social-list{display:flex;flex-wrap:wrap;gap:8px}.social-item{display:flex;align-items:center;gap:6px;padding:8px 12px;background:#f9fafb;border-radius:6px;text-decoration:none;color:#374151;font-size:13px}.social-name{font-weight:500}.action-buttons{margin-top:16px;display:flex;flex-direction:column;gap:12px}.btn-warranty{display:block;width:100%;padding:14px;background:#9333ea;color:white;border:none;border-radius:8px;font-size:16px;font-weight:600;text-align:center;text-decoration:none;cursor:pointer}.btn-warranty:hover{background:#7c22ce}`,
				CustomFields: defaultValidationCustomFields,
				IsActive:     true,
			}
			if err := h.DB.Create(&validationTemplate).Error; err != nil {
				return err
			}
			// Auto-set as tenant default for validation
			h.DB.Model(&models.Tenant{}).Where("id = ?", tenantID).
				Update("default_validation_template_id", validationTemplate.ID)
		} else {
			return err
		}
	} else {
		// Template exists — backfill custom_fields if empty (created by older version)
		if len(validationTemplate.CustomFields) == 0 || string(validationTemplate.CustomFields) == "null" {
			h.DB.Model(&validationTemplate).Update("custom_fields", defaultValidationCustomFields)
		}
	}

	// Check and create warranty template
	var warrantyTemplate models.PageTemplate
	if err := h.DB.Where("tenant_id = ? AND template_type = ?", tenantID, models.TemplateTypeWarranty).First(&warrantyTemplate).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// Create default warranty template
			warrantyTemplate = models.PageTemplate{
				TenantID:     tenantID,
				TemplateType: models.TemplateTypeWarranty,
				TemplateName: "Default Warranty Template",
				HTMLContent: `<div class="warranty-container">
	<div class="logo-section">
		<img src="{{logo_url}}" alt="Logo" class="logo" onerror="this.style.display='none'" />
	</div>
	<div class="product-info">
		<h1 class="product-name">{{product_name}}</h1>
		<p class="subtitle">Register Your Warranty</p>
	</div>
	<form id="warranty-form" class="warranty-form">
		<div class="form-group">
			<input type="text" name="customer_name" placeholder="Your Full Name" required />
		</div>
		<div class="form-group">
			<input type="email" name="email" placeholder="Email Address" required />
		</div>
		<div class="form-group">
			<input type="tel" name="phone" placeholder="Phone Number" required />
		</div>
		<div class="form-group">
			<input type="date" name="purchase_date" placeholder="Purchase Date" required />
		</div>
		<div class="form-group">
			<input type="text" name="store_name" placeholder="Store Name" />
		</div>
		<button type="submit" class="submit-btn">Register Warranty</button>
	</form>
</div>`,
				CSSContent: `.warranty-container{font-family:system-ui,-apple-system,sans-serif;max-width:400px;margin:0 auto;padding:24px}.logo{max-width:120px;display:block;margin:0 auto 16px}.product-name{font-size:24px;font-weight:700;text-align:center;margin:0 0 4px}.subtitle{text-align:center;color:#666;margin:0 0 24px}.warranty-form{display:flex;flex-direction:column;gap:16px}.form-group input{width:100%;padding:12px 16px;border:1px solid #ddd;border-radius:8px;font-size:16px;box-sizing:border-box}.form-group input:focus{outline:none;border-color:#007bff}.submit-btn{background:#007bff;color:white;border:none;padding:14px 24px;border-radius:8px;font-size:16px;font-weight:600;cursor:pointer}.submit-btn:hover{background:#0056b3}`,
				IsActive:   true,
			}
			if err := h.DB.Create(&warrantyTemplate).Error; err != nil {
				return err
			}
			// Auto-set as tenant default for warranty
			h.DB.Model(&models.Tenant{}).Where("id = ?", tenantID).
				Update("default_warranty_template_id", warrantyTemplate.ID)
		} else {
			return err
		}
	}

	return nil
}

// ListTemplates returns all templates for a tenant
func (h *TemplateHandler) ListTemplates(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}

	page := utils.GetPageQuery(c)
	limit := utils.GetLimitQuery(c, 100)
	templateType := c.Query("type")
	status := c.Query("status")

	// Ensure default validation and warranty templates exist
	if err := h.ensureDefaultTemplates(tenantUUID); err != nil {
		// Log error but don't fail the request
		// Templates might already exist or there could be a minor issue
	}

	offset := (page - 1) * limit

	var templates []models.PageTemplate
	var total int64

	query := h.DB.Model(&models.PageTemplate{}).Where("tenant_id = ?", tenantUUID)

	// Filter by type
	if templateType != "" && templateType != "all" {
		query = query.Where("template_type = ?", templateType)
	}

	// Filter by status
	switch status {
	case "active":
		query = query.Where("is_active = ?", true)
	case "inactive":
		query = query.Where("is_active = ?", false)
		// "all" or empty - no filter
	}

	query.Count(&total)
	query.Order("template_type ASC, created_at DESC").Offset(offset).Limit(limit).Find(&templates)

	utils.SuccessResponse(c, http.StatusOK, "Templates retrieved", gin.H{
		"templates": templates,
		"pagination": utils.PaginationMeta(page, limit, total),
	})
}

// GetTemplate returns a single template by ID
func (h *TemplateHandler) GetTemplate(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}

	id := c.Param("id")
	templateID, err := uuid.Parse(id)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid template ID", err)
		return
	}

	var template models.PageTemplate
	if err := h.DB.First(&template, "id = ? AND tenant_id = ?", templateID, tenantUUID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Template not found", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Template retrieved", template)
}

// CreateTemplate creates a new template
func (h *TemplateHandler) CreateTemplate(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}

	staffID, hasStaffID := utils.GetStaffUUID(c) // Optional - for audit logging

	var req CreateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request", err)
		return
	}

	// All template types can have multiple templates per tenant
	// Check for duplicate template name within same type
	var existingCount int64
	h.DB.Model(&models.PageTemplate{}).Where(
		"tenant_id = ? AND template_type = ? AND template_name = ?",
		tenantUUID, req.TemplateType, req.TemplateName,
	).Count(&existingCount)

	if existingCount > 0 {
		utils.ErrorResponse(c, http.StatusBadRequest,
			"A template with this name already exists for this type.",
			nil)
		return
	}

	// Validate custom fields
	if err := validateCustomFields(req.CustomFields); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	// Validate background config
	if err := validateBackgroundConfig(req.BackgroundConfig); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	// Convert custom fields to JSON
	var customFieldsJSON datatypes.JSON
	if req.CustomFields != nil {
		jsonBytes, err := json.Marshal(req.CustomFields)
		if err != nil {
			utils.ErrorResponse(c, http.StatusBadRequest, "Failed to serialize custom_fields", err)
			return
		}
		customFieldsJSON = datatypes.JSON(jsonBytes)
	}

	// Convert background config to JSON
	var backgroundConfigJSON datatypes.JSON
	if req.BackgroundConfig != nil {
		jsonBytes, err := json.Marshal(req.BackgroundConfig)
		if err != nil {
			utils.ErrorResponse(c, http.StatusBadRequest, "Failed to serialize background_config", err)
			return
		}
		backgroundConfigJSON = datatypes.JSON(jsonBytes)
	}

	// Set created by staff ID
	var createdBy *uuid.UUID
	if hasStaffID {
		createdBy = &staffID
	}

	template := models.PageTemplate{
		TenantID:         tenantUUID,
		TemplateType:     models.TemplateType(req.TemplateType),
		TemplateName:     req.TemplateName,
		HTMLContent:      req.HTMLContent,
		CSSContent:       req.CSSContent,
		JSContent:        req.JSContent,
		CustomFields:     customFieldsJSON,
		BackgroundConfig: backgroundConfigJSON,
		IsActive:         req.IsActive,
		CreatedBy:        createdBy,
	}

	if err := h.DB.Create(&template).Error; err != nil {
		sentry.CaptureHandlerError(c, err, "template.CreateTemplate", sentry.ErrorTypeDatabase, sentry.SeverityLow)
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to create template", err)
		return
	}

	// Auto-set as tenant default if no default exists for this type
	if template.TemplateType == models.TemplateTypeValidation || template.TemplateType == models.TemplateTypeWarranty {
		var tenant models.Tenant
		if err := h.DB.First(&tenant, "id = ?", tenantUUID).Error; err == nil {
			shouldSetDefault := false
			if template.TemplateType == models.TemplateTypeValidation && tenant.DefaultValidationTemplateID == nil {
				shouldSetDefault = true
			} else if template.TemplateType == models.TemplateTypeWarranty && tenant.DefaultWarrantyTemplateID == nil {
				shouldSetDefault = true
			}

			if shouldSetDefault {
				if template.TemplateType == models.TemplateTypeValidation {
					h.DB.Model(&tenant).Update("default_validation_template_id", template.ID)
				} else {
					h.DB.Model(&tenant).Update("default_warranty_template_id", template.ID)
				}
			}
		}
	}

	utils.SuccessResponse(c, http.StatusCreated, "Template created successfully", template)
}

// UpdateTemplate updates an existing template
func (h *TemplateHandler) UpdateTemplate(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}

	id := c.Param("id")
	templateID, err := uuid.Parse(id)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid template ID", err)
		return
	}

	var template models.PageTemplate
	if err := h.DB.First(&template, "id = ? AND tenant_id = ?", templateID, tenantUUID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Template not found", err)
		return
	}

	var req UpdateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request", err)
		return
	}

	// Validate custom fields if provided
	if req.CustomFields != nil {
		if err := validateCustomFields(req.CustomFields); err != nil {
			utils.ErrorResponse(c, http.StatusBadRequest, err.Error(), nil)
			return
		}
	}

	// Validate background config if provided
	if req.BackgroundConfig != nil {
		if err := validateBackgroundConfig(req.BackgroundConfig); err != nil {
			utils.ErrorResponse(c, http.StatusBadRequest, err.Error(), nil)
			return
		}
	}

	// Build updates map with only provided fields
	updates := map[string]interface{}{}
	if req.TemplateName != "" {
		updates["template_name"] = req.TemplateName
	}
	if req.HTMLContent != "" {
		updates["html_content"] = req.HTMLContent
	}
	if req.CSSContent != "" {
		updates["css_content"] = req.CSSContent
	}
	if req.JSContent != "" {
		updates["js_content"] = req.JSContent
	}
	if req.CustomFields != nil {
		jsonBytes, err := json.Marshal(req.CustomFields)
		if err != nil {
			utils.ErrorResponse(c, http.StatusBadRequest, "Failed to serialize custom_fields", err)
			return
		}
		updates["custom_fields"] = datatypes.JSON(jsonBytes)
	}
	if req.BackgroundConfig != nil {
		jsonBytes, err := json.Marshal(req.BackgroundConfig)
		if err != nil {
			utils.ErrorResponse(c, http.StatusBadRequest, "Failed to serialize background_config", err)
			return
		}
		updates["background_config"] = datatypes.JSON(jsonBytes)
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}

	if len(updates) > 0 {
		if err := h.DB.Model(&template).Updates(updates).Error; err != nil {
			sentry.CaptureHandlerError(c, err, "template.UpdateTemplate", sentry.ErrorTypeDatabase, sentry.SeverityLow)
			utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to update template", err)
			return
		}
	}

	// Refresh for response
	h.DB.First(&template, "id = ? AND tenant_id = ?", templateID, tenantUUID)

	utils.SuccessResponse(c, http.StatusOK, "Template updated successfully", template)
}

// DeleteTemplate deletes a template
func (h *TemplateHandler) DeleteTemplate(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}

	id := c.Param("id")
	templateID, err := uuid.Parse(id)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid template ID", err)
		return
	}

	var template models.PageTemplate
	if err := h.DB.First(&template, "id = ? AND tenant_id = ?", templateID, tenantUUID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Template not found", err)
		return
	}

	// Check if template is in use by any products (as default template)
	// IMPORTANT: Filter by tenant_id to ensure proper tenant isolation
	var productCount int64
	h.DB.Model(&models.Product{}).Where(
		"tenant_id = ? AND (default_validation_template_id = ? OR default_warranty_template_id = ?)",
		tenantUUID, templateID, templateID,
	).Count(&productCount)

	if productCount > 0 {
		utils.ErrorResponse(c, http.StatusBadRequest, "Template is in use as default template by products and cannot be deleted", nil)
		return
	}

	// Check if template is in use by any QR batches
	// IMPORTANT: Filter by tenant_id to ensure proper tenant isolation
	var batchCount int64
	h.DB.Model(&models.QRBatch{}).Where(
		"tenant_id = ? AND deleted_at IS NULL AND (validation_template_id = ? OR warranty_template_id = ?)",
		tenantUUID, templateID, templateID,
	).Count(&batchCount)

	if batchCount > 0 {
		utils.ErrorResponse(c, http.StatusBadRequest, "Template is in use by QR batches and cannot be deleted", nil)
		return
	}

	// Prevent deleting the last template of a type (ensure at least one exists)
	var typeCount int64
	h.DB.Model(&models.PageTemplate{}).Where(
		"tenant_id = ? AND template_type = ?",
		tenantUUID, template.TemplateType,
	).Count(&typeCount)

	if typeCount <= 1 && (template.TemplateType == models.TemplateTypeValidation || template.TemplateType == models.TemplateTypeWarranty) {
		utils.ErrorResponse(c, http.StatusBadRequest,
			"Cannot delete the last "+string(template.TemplateType)+" template. At least one must exist.",
			nil)
		return
	}

	// Clean up logo file from storage before deleting template
	h.deleteLogoFile(template.CustomFields)

	if err := h.DB.Delete(&template).Error; err != nil {
		sentry.CaptureHandlerError(c, err, "template.DeleteTemplate", sentry.ErrorTypeDatabase, sentry.SeverityLow)
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to delete template", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Template deleted successfully", nil)
}

// deleteLogoFile removes the logo file from R2 or local filesystem
func (h *TemplateHandler) deleteLogoFile(customFields datatypes.JSON) {
	if customFields == nil || len(customFields) == 0 {
		return
	}

	var fields map[string]interface{}
	if err := json.Unmarshal(customFields, &fields); err != nil {
		return
	}

	headerMap, ok := fields["header"].(map[string]interface{})
	if !ok {
		return
	}

	logoURL, ok := headerMap["logo_url"].(string)
	if !ok || logoURL == "" {
		return
	}

	r2Client := storage.GetGlobalR2Client()
	if r2Client != nil && r2Client.IsR2URL(logoURL) {
		key := r2Client.ExtractKeyFromURL(logoURL)
		if key != "" {
			r2Client.Delete(context.Background(), key)
		}
	} else if strings.HasPrefix(logoURL, "/uploads/") {
		relativePath := strings.TrimPrefix(logoURL, "/uploads/")
		filePath := filepath.Join(h.Cfg.UploadPath, relativePath)
		if isPathSafe(filepath.Join(h.Cfg.UploadPath, "templates"), filePath) {
			os.Remove(filePath)
		}
	}
}

// UploadTemplateLogo uploads a logo image for a template header
// POST /tenant/templates/:id/logo
func (h *TemplateHandler) UploadTemplateLogo(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}

	id := c.Param("id")
	templateID, err := uuid.Parse(id)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid template ID", err)
		return
	}

	// Verify template belongs to tenant
	var template models.PageTemplate
	if err := h.DB.First(&template, "id = ? AND tenant_id = ?", templateID, tenantUUID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Template not found", err)
		return
	}

	// Get the uploaded file
	file, header, err := c.Request.FormFile("logo")
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "No logo file provided", nil)
		return
	}
	defer file.Close()

	// Process and validate image
	processed, err := utils.ProcessUploadedImage(file, header, utils.ImageUploadOptions{
		MaxFileSize:  2 * 1024 * 1024, // 2MB for logos
		MinDimension: 50,              // Logos can be small
		AllowedTypes: []string{"image/jpeg", "image/png", "image/webp"},
	})
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	// Generate filename and storage key
	filename := fmt.Sprintf("%s%s", uuid.New().String(), processed.Extension)
	storageKey := fmt.Sprintf("templates/%s/logos/%s", tenantUUID.String(), filename)

	var imageURL string

	// Upload to R2 or local filesystem
	r2Client := storage.GetGlobalR2Client()
	if r2Client != nil {
		url, err := r2Client.Upload(context.Background(), storageKey, processed.Data, processed.ContentType)
		if err != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to upload logo to storage", err)
			return
		}
		imageURL = url
	} else {
		uploadDir := filepath.Join(h.Cfg.UploadPath, "templates", tenantUUID.String(), "logos")
		if err := utils.EnsureDir(uploadDir); err != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to create upload directory", err)
			return
		}

		localFilePath := filepath.Join(uploadDir, filename)
		if err := os.WriteFile(localFilePath, processed.Data, 0644); err != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to save logo", err)
			return
		}
		imageURL = fmt.Sprintf("/uploads/%s", storageKey)
	}

	// Save old custom_fields for cleanup after DB save succeeds
	oldCustomFields := make(datatypes.JSON, len(template.CustomFields))
	copy(oldCustomFields, template.CustomFields)

	// Update custom_fields.header.logo_url
	var fields map[string]interface{}
	if template.CustomFields != nil && len(template.CustomFields) > 0 {
		if err := json.Unmarshal(template.CustomFields, &fields); err != nil {
			fields = make(map[string]interface{})
		}
	} else {
		fields = make(map[string]interface{})
	}

	headerMap, ok2 := fields["header"].(map[string]interface{})
	if !ok2 {
		headerMap = make(map[string]interface{})
	}
	headerMap["logo_url"] = imageURL
	headerMap["logo_enabled"] = true
	fields["header"] = headerMap

	jsonBytes, err := json.Marshal(fields)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to update template", err)
		return
	}

	if err := h.DB.Model(&template).Update("custom_fields", datatypes.JSON(jsonBytes)).Error; err != nil {
		sentry.CaptureHandlerError(c, err, "template.UploadTemplateLogo", sentry.ErrorTypeDatabase, sentry.SeverityMedium)
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to update template", err)
		return
	}

	// Delete old logo file AFTER DB save succeeds (prevents data loss on save failure)
	h.deleteLogoFile(oldCustomFields)

	utils.SuccessResponse(c, http.StatusOK, "Logo uploaded", gin.H{
		"logo_url": imageURL,
	})
}

// DeleteTemplateLogo removes the logo from a template
// DELETE /tenant/templates/:id/logo
func (h *TemplateHandler) DeleteTemplateLogo(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}

	id := c.Param("id")
	templateID, err := uuid.Parse(id)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid template ID", err)
		return
	}

	// Verify template belongs to tenant
	var template models.PageTemplate
	if err := h.DB.First(&template, "id = ? AND tenant_id = ?", templateID, tenantUUID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Template not found", err)
		return
	}

	// Delete the logo file from storage
	h.deleteLogoFile(template.CustomFields)

	// Clear logo_url from custom_fields
	var fields map[string]interface{}
	if template.CustomFields != nil && len(template.CustomFields) > 0 {
		if err := json.Unmarshal(template.CustomFields, &fields); err != nil {
			fields = make(map[string]interface{})
		}
	} else {
		fields = make(map[string]interface{})
	}

	if headerMap, ok := fields["header"].(map[string]interface{}); ok {
		headerMap["logo_url"] = ""
		fields["header"] = headerMap
	}

	jsonBytes, err := json.Marshal(fields)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to update template", err)
		return
	}

	if err := h.DB.Model(&template).Update("custom_fields", datatypes.JSON(jsonBytes)).Error; err != nil {
		sentry.CaptureHandlerError(c, err, "template.DeleteTemplateLogo", sentry.ErrorTypeDatabase, sentry.SeverityMedium)
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to update template", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Logo deleted", nil)
}

// GetPublicTemplate returns template data for public QR scan pages
func (h *TemplateHandler) GetPublicTemplate(c *gin.Context) {
	qrParam := c.Param("uuid")
	templateType := c.DefaultQuery("type", "validation")

	// Find QR code - support Base58, UUID, and hex formats
	var qrCode models.QRCode
	var qrUUID uuid.UUID
	var lookupByQRCode bool

	// Parse input format (same logic as validation handlers)
	inputLen := len(qrParam)
	switch {
	case inputLen >= 21 && inputLen <= 22 && utils.IsBase58UUID(qrParam):
		// Base58 encoded UUID (new format)
		var err error
		qrUUID, err = utils.Base58ToUUID(qrParam)
		if err != nil {
			h.returnDefaultTemplate(c, templateType, nil, nil)
			return
		}
	case inputLen == 36:
		// Standard UUID format
		var err error
		qrUUID, err = uuid.Parse(qrParam)
		if err != nil {
			h.returnDefaultTemplate(c, templateType, nil, nil)
			return
		}
	case inputLen == 32:
		// Legacy hex format (qr_code string)
		lookupByQRCode = true
	default:
		h.returnDefaultTemplate(c, templateType, nil, nil)
		return
	}

	// Lookup QR code
	var lookupErr error
	if lookupByQRCode {
		lookupErr = h.DB.Preload("Batch.Product").Preload("Batch.Tenant").
			First(&qrCode, "qr_code = ?", qrParam).Error
	} else {
		lookupErr = h.DB.Preload("Batch.Product").Preload("Batch.Tenant").
			First(&qrCode, "qr_uuid = ?", qrUUID).Error
	}

	if lookupErr != nil {
		h.returnDefaultTemplate(c, templateType, nil, nil)
		return
	}

	// Nil safety — if batch relationship is broken, return default
	if qrCode.Batch == nil {
		h.returnDefaultTemplate(c, templateType, nil, nil)
		return
	}

	// Template resolution priority:
	// 1. Batch-level override (explicit template on batch)
	// 2. Product-level default (default template on product)
	// 3. Tenant explicit default (set via Set as Default)
	// 4. Tenant's oldest template of this type (fallback)
	var templateID *uuid.UUID
	var needsFeature bool

	// Get tenant's explicit defaults
	var tenant models.Tenant
	h.DB.Select("default_validation_template_id", "default_warranty_template_id").
		First(&tenant, "id = ?", qrCode.Batch.TenantID)

	switch templateType {
	case "validation":
		// Priority: batch override → product default → tenant explicit default → tenant oldest
		if qrCode.Batch.ValidationTemplateID != nil {
			templateID = qrCode.Batch.ValidationTemplateID
		} else if qrCode.Batch.Product != nil && qrCode.Batch.Product.DefaultValidationTemplateID != nil {
			templateID = qrCode.Batch.Product.DefaultValidationTemplateID
		} else if tenant.DefaultValidationTemplateID != nil {
			templateID = tenant.DefaultValidationTemplateID
		}
		needsFeature = qrCode.Batch.NeedValidation
	case "warranty":
		// Priority: batch override → product default → tenant explicit default → tenant oldest
		if qrCode.Batch.WarrantyTemplateID != nil {
			templateID = qrCode.Batch.WarrantyTemplateID
		} else if qrCode.Batch.Product != nil && qrCode.Batch.Product.DefaultWarrantyTemplateID != nil {
			templateID = qrCode.Batch.Product.DefaultWarrantyTemplateID
		} else if tenant.DefaultWarrantyTemplateID != nil {
			templateID = tenant.DefaultWarrantyTemplateID
		}
		needsFeature = qrCode.Batch.Product != nil && qrCode.Batch.Product.WarrantyEnabled
	}

	// Fallback to tenant's oldest active template if no explicit assignment
	if templateID == nil {
		var oldestTemplate models.PageTemplate
		if err := h.DB.Where("tenant_id = ? AND template_type = ? AND is_active = ?",
			qrCode.Batch.TenantID, templateType, true).
			Order("created_at ASC").
			First(&oldestTemplate).Error; err == nil {
			templateID = &oldestTemplate.ID
		}
	}

	// Check if features is enabled (skip for validation - it's the landing page entry point)
	// Validation/landing page should always be accessible regardless of need_validation flag
	if templateType != "validation" && !needsFeature {
		utils.ErrorResponse(c, http.StatusNotFound, "This features is not enabled for this product", nil)
		return
	}

	// Fetch template or use default
	var template *models.PageTemplate
	if templateID != nil {
		var t models.PageTemplate
		if err := h.DB.First(&t, "id = ? AND is_active = ?", templateID, true).Error; err == nil {
			template = &t
		}
	}

	// Get validation count for validation pages
	var validationCount int64
	if templateType == "validation" {
		h.DB.Model(&models.Interaction{}).Where(
			"qr_code_id = ? AND interaction_subcategory = ?",
			qrCode.ID, "product_validation",
		).Count(&validationCount)
	}

	// Build response data
	responseData := gin.H{
		"qr_code": gin.H{
			"id":      qrCode.ID,
			"qr_uuid": qrCode.QRUUID,
			"status":  qrCode.Status,
		},
		"product":          nil,
		"batch":            nil,
		"tenant":           nil,
		"validation_count": validationCount,
	}

	if qrCode.Batch != nil {
		responseData["batch"] = gin.H{
			"id":         qrCode.Batch.ID,
			"batch_name": qrCode.Batch.BatchName,
			"batch_code": qrCode.Batch.BatchCode,
			"logo_url":   qrCode.Batch.LogoURL,
		}

		if qrCode.Batch.Product != nil {
			responseData["product"] = gin.H{
				"id":           qrCode.Batch.Product.ID,
				"product_name": qrCode.Batch.Product.ProductName,
				"product_code": qrCode.Batch.Product.ProductCode,
				"description":  qrCode.Batch.Product.Description,
			}
		}

		if qrCode.Batch.Tenant != nil {
			responseData["tenant"] = gin.H{
				"company_name": qrCode.Batch.Tenant.CompanyName,
				"logo_url":     "", // Add tenant logo if available
			}
		}
	}

	// Return template with data
	if template != nil {
		// Resolve background config with preset URL
		backgroundConfig := h.resolveTemplateBackgroundConfig(template.BackgroundConfig)

		// Merge product-level template overrides into custom_fields
		customFields := template.CustomFields
		if templateType == "validation" && qrCode.Batch != nil && qrCode.Batch.Product != nil &&
			len(qrCode.Batch.Product.TemplateOverrides) > 0 {
			customFields = utils.DeepMergeJSON(template.CustomFields, qrCode.Batch.Product.TemplateOverrides)
		}
		if templateType == "warranty" && qrCode.Batch != nil && qrCode.Batch.Product != nil &&
			len(qrCode.Batch.Product.WarrantyTemplateOverrides) > 0 {
			customFields = utils.DeepMergeJSON(template.CustomFields, qrCode.Batch.Product.WarrantyTemplateOverrides)
		}

		utils.SuccessResponse(c, http.StatusOK, "Template retrieved", gin.H{
			"template": gin.H{
				"id":                template.ID,
				"template_type":     template.TemplateType,
				"template_name":     template.TemplateName,
				"html_content":      template.HTMLContent,
				"css_content":       template.CSSContent,
				"js_content":        template.JSContent,
				"custom_fields":     customFields,
				"background_config": backgroundConfig,
			},
			"data": responseData,
		})
	} else {
		h.returnDefaultTemplate(c, templateType, &qrCode, responseData)
	}
}

// resolveTemplateBackgroundConfig resolves background config, fetching preset URL if needed
func (h *TemplateHandler) resolveTemplateBackgroundConfig(config datatypes.JSON) map[string]interface{} {
	if config == nil || len(config) == 0 {
		return map[string]interface{}{
			"background_type":       "none",
			"preset_id":             nil,
			"custom_background_url": nil,
			"background_url":        nil,
			"overlay_color":         "#000000",
			"overlay_opacity":       30,
			"card_opacity":          90,
			"card_blur":             0,
		}
	}

	var bgConfig map[string]interface{}
	if err := json.Unmarshal(config, &bgConfig); err != nil {
		return map[string]interface{}{
			"background_type":       "none",
			"preset_id":             nil,
			"custom_background_url": nil,
			"background_url":        nil,
			"overlay_color":         "#000000",
			"overlay_opacity":       30,
			"card_opacity":          90,
			"card_blur":             0,
		}
	}

	// Set default values
	result := map[string]interface{}{
		"background_type":       "none",
		"preset_id":             nil,
		"custom_background_url": nil,
		"background_url":        nil,
		"overlay_color":         "#000000",
		"overlay_opacity":       30,
		"card_opacity":          90,
		"card_blur":             0,
	}

	// Copy values from config
	if v, ok := bgConfig["background_type"]; ok {
		result["background_type"] = v
	}
	if v, ok := bgConfig["preset_id"]; ok {
		result["preset_id"] = v
	}
	if v, ok := bgConfig["custom_background_url"]; ok {
		result["custom_background_url"] = v
	}
	if v, ok := bgConfig["overlay_color"]; ok {
		result["overlay_color"] = v
	}
	if v, ok := bgConfig["overlay_opacity"]; ok {
		result["overlay_opacity"] = v
	}
	if v, ok := bgConfig["card_opacity"]; ok {
		result["card_opacity"] = v
	}
	if v, ok := bgConfig["card_blur"]; ok {
		result["card_blur"] = v
	}

	// Resolve background URL based on type
	bgType, _ := result["background_type"].(string)
	switch bgType {
	case "preset":
		if presetID, ok := result["preset_id"].(string); ok && presetID != "" {
			if parsedID, err := uuid.Parse(presetID); err == nil {
				var preset models.ThemePreset
				if err := h.DB.First(&preset, "id = ? AND is_active = ? AND deleted_at IS NULL", parsedID, true).Error; err == nil {
					result["background_url"] = preset.BackgroundURL
					// Copy preset's styling if not overridden
					if result["overlay_color"] == "#000000" {
						result["overlay_color"] = preset.OverlayColor
					}
					if result["overlay_opacity"] == 30 {
						result["overlay_opacity"] = preset.OverlayOpacity
					}
					if result["card_opacity"] == 90 {
						result["card_opacity"] = preset.CardOpacity
					}
					if result["card_blur"] == 0 {
						result["card_blur"] = preset.CardBlur
					}
				}
			}
		}
	case "custom":
		if customURL, ok := result["custom_background_url"].(string); ok && customURL != "" {
			result["background_url"] = customURL
		}
	}

	return result
}

// SetAsDefault sets a template as the tenant's default for its type
func (h *TemplateHandler) SetAsDefault(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}

	id := c.Param("id")
	templateID, err := uuid.Parse(id)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid template ID", err)
		return
	}

	// Find the template and verify ownership
	var template models.PageTemplate
	if err := h.DB.First(&template, "id = ? AND tenant_id = ?", templateID, tenantUUID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Template not found", err)
		return
	}

	// Update tenant's default template
	var updates map[string]interface{}
	switch template.TemplateType {
	case models.TemplateTypeValidation:
		updates = map[string]interface{}{"default_validation_template_id": templateID}
	case models.TemplateTypeWarranty:
		updates = map[string]interface{}{"default_warranty_template_id": templateID}
	}

	if err := h.DB.Model(&models.Tenant{}).Where("id = ?", tenantUUID).Updates(updates).Error; err != nil {
		sentry.CaptureHandlerError(c, err, "template.SetAsDefault", sentry.ErrorTypeDatabase, sentry.SeverityLow)
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to set template as default", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Template set as default successfully", gin.H{
		"template_id":   templateID,
		"template_type": template.TemplateType,
	})
}

// GetTenantDefaults returns the tenant's default template IDs
func (h *TemplateHandler) GetTenantDefaults(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}

	var tenant models.Tenant
	if err := h.DB.Select("default_validation_template_id", "default_warranty_template_id").
		First(&tenant, "id = ?", tenantUUID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Tenant not found", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Tenant defaults retrieved", gin.H{
		"default_validation_template_id": tenant.DefaultValidationTemplateID,
		"default_warranty_template_id":   tenant.DefaultWarrantyTemplateID,
	})
}

// returnDefaultTemplate returns nil template to let frontend use Vue component default
// The Vue component has a built-in warranty button that the old HTML template lacked
func (h *TemplateHandler) returnDefaultTemplate(c *gin.Context, templateType string, qrCode *models.QRCode, data gin.H) {
	utils.SuccessResponse(c, http.StatusOK, "Using default Vue component", gin.H{
		"template": nil,
		"data":     data,
	})
}

