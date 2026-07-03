package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gamatritunggal/smartscan/backend/internal/config"
	"github.com/gamatritunggal/smartscan/backend/internal/models"
	"github.com/gamatritunggal/smartscan/backend/internal/sentry"
	"github.com/gamatritunggal/smartscan/backend/internal/utils"
	"gorm.io/gorm"
)

type CertificationHandler struct {
	DB  *gorm.DB
	Cfg *config.Config
}

func NewCertificationHandler(db *gorm.DB, cfg *config.Config) *CertificationHandler {
	return &CertificationHandler{DB: db, Cfg: cfg}
}

// ==================== CERTIFICATION TYPES (Super Admin) ====================

// ListCertificationTypes returns all certification types
func (h *CertificationHandler) ListCertificationTypes(c *gin.Context) {
	page := utils.GetPageQuery(c)
	limit := utils.GetLimitQuery(c, 50)
	search := c.Query("search")
	countryCode := c.Query("country_code")
	status := c.DefaultQuery("status", "active")

	offset := (page - 1) * limit

	var types []models.CertificationType
	var total int64

	query := h.DB.Model(&models.CertificationType{})

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
	if countryCode == "international" {
		query = query.Where("country_code IS NULL")
	} else if countryCode != "" {
		query = query.Where("country_code = ?", countryCode)
	}

	query.Count(&total)
	query.Preload("Country").Order("display_order ASC, name ASC").Offset(offset).Limit(limit).Find(&types)

	utils.SuccessResponse(c, http.StatusOK, "Certification types retrieved", gin.H{
		"certification_types": types,
		"pagination": gin.H{
			"page":       page,
			"limit":      limit,
			"total":      total,
			"total_page": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// GetCertificationType returns a single certification type
func (h *CertificationHandler) GetCertificationType(c *gin.Context) {
	id := c.Param("id")
	typeID, err := uuid.Parse(id)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid certification type ID", err)
		return
	}

	var certType models.CertificationType
	if err := h.DB.Unscoped().Preload("Country").First(&certType, "id = ?", typeID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Certification type not found", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Certification type retrieved", certType)
}

// CreateCertificationType creates a new certification type
func (h *CertificationHandler) CreateCertificationType(c *gin.Context) {
	var req struct {
		CountryCode  *string `json:"country_code"` // NULL for international
		Code         string  `json:"code" binding:"required"`
		Name         string  `json:"name" binding:"required"`
		Description  string  `json:"description"`
		LogoURL      string  `json:"logo_url"`
		WebsiteURL   string  `json:"website_url"`
		DisplayOrder int     `json:"display_order"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request", err)
		return
	}

	// Check for duplicate code
	var existing models.CertificationType
	if err := h.DB.Unscoped().Where("code = ?", req.Code).First(&existing).Error; err == nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Certification code already exists", nil)
		return
	}

	certType := models.CertificationType{
		CountryCode:  req.CountryCode,
		Code:         req.Code,
		Name:         req.Name,
		Description:  req.Description,
		LogoURL:      req.LogoURL,
		WebsiteURL:   req.WebsiteURL,
		IsActive:     true,
		DisplayOrder: req.DisplayOrder,
	}

	if err := h.DB.Create(&certType).Error; err != nil {
		sentry.CaptureHandlerError(c, err, "certification.CreateCertificationType", sentry.ErrorTypeDatabase, sentry.SeverityMedium)
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to create certification type", err)
		return
	}

	utils.SuccessResponse(c, http.StatusCreated, "Certification type created", certType)
}

// UpdateCertificationType updates a certification type
func (h *CertificationHandler) UpdateCertificationType(c *gin.Context) {
	id := c.Param("id")
	typeID, err := uuid.Parse(id)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid certification type ID", err)
		return
	}

	var certType models.CertificationType
	if err := h.DB.First(&certType, "id = ?", typeID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Certification type not found", err)
		return
	}

	var req struct {
		CountryCode  *string  `json:"country_code"`
		Code         string   `json:"code"`
		Name         string   `json:"name"`
		Description  *string  `json:"description"`
		LogoURL      *string  `json:"logo_url"`
		WebsiteURL   *string  `json:"website_url"`
		IsActive     *bool    `json:"is_active"`
		DisplayOrder *int     `json:"display_order"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request", err)
		return
	}

	updates := map[string]interface{}{}

	// Check for duplicate code if changing
	if req.Code != "" && req.Code != certType.Code {
		var existing models.CertificationType
		if err := h.DB.Unscoped().Where("code = ? AND id != ?", req.Code, typeID).First(&existing).Error; err == nil {
			utils.ErrorResponse(c, http.StatusBadRequest, "Certification code already exists", nil)
			return
		}
		updates["code"] = req.Code
	}

	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.LogoURL != nil {
		updates["logo_url"] = *req.LogoURL
	}
	if req.WebsiteURL != nil {
		updates["website_url"] = *req.WebsiteURL
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}
	if req.DisplayOrder != nil {
		updates["display_order"] = *req.DisplayOrder
	}
	// CountryCode can be set to NULL — always include it
	if req.CountryCode != nil {
		updates["country_code"] = *req.CountryCode
	} else {
		updates["country_code"] = nil
	}

	if len(updates) > 0 {
		if err := h.DB.Model(&certType).Updates(updates).Error; err != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to update certification type", err)
			return
		}
	}

	// Re-fetch for response
	h.DB.First(&certType, "id = ?", typeID)

	utils.SuccessResponse(c, http.StatusOK, "Certification type updated", certType)
}

// DeleteCertificationType soft deletes a certification type
func (h *CertificationHandler) DeleteCertificationType(c *gin.Context) {
	id := c.Param("id")
	typeID, err := uuid.Parse(id)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid certification type ID", err)
		return
	}

	var certType models.CertificationType
	if err := h.DB.First(&certType, "id = ?", typeID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Certification type not found", err)
		return
	}

	now := time.Now().UTC()

	if err := h.DB.Model(&certType).Update("deleted_at", now).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to delete certification type", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Certification type deleted", nil)
}

// RestoreCertificationType restores a soft-deleted certification type
func (h *CertificationHandler) RestoreCertificationType(c *gin.Context) {
	id := c.Param("id")
	typeID, err := uuid.Parse(id)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid certification type ID", err)
		return
	}

	var certType models.CertificationType
	if err := h.DB.Unscoped().First(&certType, "id = ?", typeID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Certification type not found", err)
		return
	}

	if err := h.DB.Unscoped().Model(&certType).Update("deleted_at", nil).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to restore certification type", err)
		return
	}

	certType.DeletedAt = nil
	utils.SuccessResponse(c, http.StatusOK, "Certification type restored", certType)
}

// ==================== CERTIFICATION REQUESTS (Super Admin Review) ====================





// ==================== TENANT ENDPOINTS ====================

// GetAvailableCertificationTypes returns active certification types for tenant dropdown
// Auto-filters by tenant's country + international certifications
func (h *CertificationHandler) GetAvailableCertificationTypes(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}

	// Get tenant's country
	var tenant models.Tenant
	if err := h.DB.First(&tenant, "id = ?", tenantUUID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to get tenant info", err)
		return
	}

	var types []models.CertificationType

	query := h.DB.Where("is_active = ? AND deleted_at IS NULL", true)

	// Filter by tenant's country + international (country_code IS NULL)
	if tenant.CountryCode != nil && *tenant.CountryCode != "" {
		query = query.Where("country_code = ? OR country_code IS NULL", *tenant.CountryCode)
	}

	query.Preload("Country").Order("display_order ASC, name ASC").Find(&types)

	utils.SuccessResponse(c, http.StatusOK, "Certification types retrieved", types)
}

// GetProductCertifications returns certifications for a specific product
func (h *CertificationHandler) GetProductCertifications(c *gin.Context) {
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

	var certifications []models.ProductCertification
	h.DB.Where("product_id = ?", productUUIDParsed).
		Preload("CertificationType").
		Preload("CertificationType.Country").
		Order("sort_order ASC").
		Find(&certifications)

	utils.SuccessResponse(c, http.StatusOK, "Product certifications retrieved", certifications)
}

// AddProductCertification adds a certification to a product
func (h *CertificationHandler) AddProductCertification(c *gin.Context) {
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
		CertificationTypeID string `json:"certification_type_id" binding:"required"`
		RegistrationNumber  string `json:"registration_number" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request", err)
		return
	}

	certTypeID, err := uuid.Parse(req.CertificationTypeID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid certification type ID", err)
		return
	}

	// Verify product belongs to tenant
	var product models.Product
	if err := h.DB.First(&product, "id = ? AND tenant_id = ?", productUUIDParsed, tenantUUID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Product not found", err)
		return
	}

	// Verify certification type exists and is active
	var certType models.CertificationType
	if err := h.DB.First(&certType, "id = ? AND is_active = ? AND deleted_at IS NULL", certTypeID, true).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Certification type not found or not active", err)
		return
	}

	// Check if already added
	var existing models.ProductCertification
	if err := h.DB.Where("product_id = ? AND certification_type_id = ?", productUUIDParsed, certTypeID).First(&existing).Error; err == nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Product already has this certification", nil)
		return
	}

	// Get max sort_order for this product
	var maxOrder int
	h.DB.Model(&models.ProductCertification{}).
		Where("product_id = ?", productUUIDParsed).
		Select("COALESCE(MAX(sort_order), -1)").
		Scan(&maxOrder)

	certification := models.ProductCertification{
		ProductID:           productUUIDParsed,
		CertificationTypeID: certTypeID,
		RegistrationNumber:  req.RegistrationNumber,
		SortOrder:           maxOrder + 1,
	}

	if err := h.DB.Create(&certification).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to add certification", err)
		return
	}

	// Load relations
	h.DB.Preload("CertificationType").Preload("CertificationType.Country").First(&certification, "id = ?", certification.ID)

	utils.SuccessResponse(c, http.StatusCreated, "Certification added to product", certification)
}

// UpdateProductCertification updates a product certification (registration number)
func (h *CertificationHandler) UpdateProductCertification(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}
	productID := c.Param("product_id")
	certID := c.Param("cert_id")

	productUUIDParsed, err := uuid.Parse(productID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid product ID", err)
		return
	}
	certUUIDParsed, err := uuid.Parse(certID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid certification ID", err)
		return
	}

	// Verify product belongs to tenant
	var product models.Product
	if err := h.DB.First(&product, "id = ? AND tenant_id = ?", productUUIDParsed, tenantUUID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Product not found", err)
		return
	}

	var certification models.ProductCertification
	if err := h.DB.First(&certification, "id = ? AND product_id = ?", certUUIDParsed, productUUIDParsed).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Product certification not found", err)
		return
	}

	var req struct {
		RegistrationNumber string `json:"registration_number" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request", err)
		return
	}

	if err := h.DB.Model(&certification).Update("registration_number", req.RegistrationNumber).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to update certification", err)
		return
	}

	h.DB.Preload("CertificationType").Preload("CertificationType.Country").First(&certification, "id = ?", certification.ID)

	utils.SuccessResponse(c, http.StatusOK, "Product certification updated", certification)
}

// RemoveProductCertification removes a certification from a product
func (h *CertificationHandler) RemoveProductCertification(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}
	productID := c.Param("product_id")
	certID := c.Param("cert_id")

	productUUIDParsed, err := uuid.Parse(productID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid product ID", err)
		return
	}
	certUUIDParsed, err := uuid.Parse(certID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid certification ID", err)
		return
	}

	// Verify product belongs to tenant
	var product models.Product
	if err := h.DB.First(&product, "id = ? AND tenant_id = ?", productUUIDParsed, tenantUUID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Product not found", err)
		return
	}

	var certification models.ProductCertification
	if err := h.DB.First(&certification, "id = ? AND product_id = ?", certUUIDParsed, productUUIDParsed).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Product certification not found", err)
		return
	}

	if err := h.DB.Delete(&certification).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to remove certification", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Certification removed from product", nil)
}

// ReorderProductCertifications updates the sort order of product certifications
func (h *CertificationHandler) ReorderProductCertifications(c *gin.Context) {
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

	var req struct {
		CertIDs []string `json:"cert_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request", err)
		return
	}

	// Validate: no duplicate IDs
	seen := make(map[string]bool)
	for _, id := range req.CertIDs {
		if seen[id] {
			utils.ErrorResponse(c, http.StatusBadRequest, "Duplicate certification ID in list", nil)
			return
		}
		seen[id] = true
	}

	// Validate: must include all certifications for this product
	var existingCount int64
	h.DB.Model(&models.ProductCertification{}).Where("product_id = ?", productUUIDParsed).Count(&existingCount)
	if int64(len(req.CertIDs)) != existingCount {
		utils.ErrorResponse(c, http.StatusBadRequest, "Must include all certifications in reorder request", nil)
		return
	}

	// Update sort_order within a transaction
	tx := h.DB.Begin()
	for i, certIDStr := range req.CertIDs {
		certID, err := uuid.Parse(certIDStr)
		if err != nil {
			tx.Rollback()
			utils.ErrorResponse(c, http.StatusBadRequest, "Invalid certification ID in list", nil)
			return
		}

		var cert models.ProductCertification
		if err := tx.Where("id = ? AND product_id = ?", certID, productUUIDParsed).First(&cert).Error; err != nil {
			tx.Rollback()
			utils.ErrorResponse(c, http.StatusNotFound, "Certification not found or does not belong to this product", nil)
			return
		}

		if err := tx.Model(&cert).Update("sort_order", i).Error; err != nil {
			tx.Rollback()
			utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to update sort order", err)
			return
		}
	}

	if err := tx.Commit().Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to commit reorder", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Product certifications reordered", nil)
}


