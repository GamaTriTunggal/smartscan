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

type TenantSocialAccountHandler struct {
	DB  *gorm.DB
	Cfg *config.Config
}

func NewTenantSocialAccountHandler(db *gorm.DB, cfg *config.Config) *TenantSocialAccountHandler {
	return &TenantSocialAccountHandler{DB: db, Cfg: cfg}
}

// CreateTenantSocialAccountRequest represents the request to create a social account
type CreateTenantSocialAccountRequest struct {
	PlatformID    string `json:"platform_id" binding:"required"`
	AccountHandle string `json:"account_handle" binding:"required,max=255"`
	AccountURL    string `json:"account_url" binding:"max=500"`
}

// UpdateTenantSocialAccountRequest represents the request to update a social account
type UpdateTenantSocialAccountRequest struct {
	AccountHandle string `json:"account_handle" binding:"max=255"`
	AccountURL    string `json:"account_url" binding:"max=500"`
	IsActive      *bool  `json:"is_active"`
}

// TenantSocialAccountResponse represents the response with product count
type TenantSocialAccountResponse struct {
	ID            uuid.UUID                   `json:"id"`
	TenantID      uuid.UUID                   `json:"tenant_id"`
	PlatformID    uuid.UUID                   `json:"platform_id"`
	Platform      *models.SocialMediaPlatform `json:"platform,omitempty"`
	AccountHandle string                      `json:"account_handle"`
	AccountURL    string                      `json:"account_url,omitempty"`
	IsActive      bool                        `json:"is_active"`
	ProductCount  int                         `json:"product_count"`
	CreatedAt     string                      `json:"created_at"`
	UpdatedAt     string                      `json:"updated_at"`
}

// List returns all social accounts for the tenant
func (h *TenantSocialAccountHandler) List(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")

	var accounts []models.TenantSocialAccount
	if err := h.DB.Where("tenant_id = ?", tenantID).
		Preload("Platform").
		Order("created_at DESC").
		Find(&accounts).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to fetch social accounts", err)
		return
	}

	// Get all product counts in a single query (fix N+1)
	accountIDs := make([]uuid.UUID, len(accounts))
	for i, acc := range accounts {
		accountIDs[i] = acc.ID
	}

	// Build count map from single GROUP BY query
	countMap := make(map[uuid.UUID]int64)
	if len(accountIDs) > 0 {
		type AccountCount struct {
			SocialAccountID uuid.UUID `gorm:"column:social_account_id"`
			Count           int64     `gorm:"column:count"`
		}
		var counts []AccountCount
		h.DB.Model(&models.ProductSocialAccountLink{}).
			Select("social_account_id, count(*) as count").
			Where("social_account_id IN ?", accountIDs).
			Group("social_account_id").
			Find(&counts)

		for _, c := range counts {
			countMap[c.SocialAccountID] = c.Count
		}
	}

	// Build response using pre-fetched counts
	responses := make([]TenantSocialAccountResponse, 0, len(accounts))
	for _, acc := range accounts {
		responses = append(responses, TenantSocialAccountResponse{
			ID:            acc.ID,
			TenantID:      acc.TenantID,
			PlatformID:    acc.PlatformID,
			Platform:      acc.Platform,
			AccountHandle: acc.AccountHandle,
			AccountURL:    acc.AccountURL,
			IsActive:      acc.IsActive,
			ProductCount:  int(countMap[acc.ID]),
			CreatedAt:     acc.CreatedAt.Format("2006-01-02T15:04:05Z"),
			UpdatedAt:     acc.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	utils.SuccessResponse(c, http.StatusOK, "Social accounts retrieved", gin.H{
		"accounts": responses,
	})
}

// Get returns a single social account
func (h *TenantSocialAccountHandler) Get(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")
	accountID := c.Param("id")

	accountUUID, err := uuid.Parse(accountID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid account ID", nil)
		return
	}

	var account models.TenantSocialAccount
	if err := h.DB.Where("id = ? AND tenant_id = ?", accountUUID, tenantID).
		Preload("Platform").
		First(&account).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Social account not found", nil)
		return
	}

	// Get product count
	var count int64
	h.DB.Model(&models.ProductSocialAccountLink{}).Where("social_account_id = ?", account.ID).Count(&count)

	utils.SuccessResponse(c, http.StatusOK, "Social account retrieved", gin.H{
		"account": TenantSocialAccountResponse{
			ID:            account.ID,
			TenantID:      account.TenantID,
			PlatformID:    account.PlatformID,
			Platform:      account.Platform,
			AccountHandle: account.AccountHandle,
			AccountURL:    account.AccountURL,
			IsActive:      account.IsActive,
			ProductCount:  int(count),
			CreatedAt:     account.CreatedAt.Format("2006-01-02T15:04:05Z"),
			UpdatedAt:     account.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		},
	})
}

// Create creates a new social account for the tenant
func (h *TenantSocialAccountHandler) Create(c *gin.Context) {
	tenantIDStr, _ := c.Get("tenant_id")
	tenantUUID, err := uuid.Parse(tenantIDStr.(string))
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Invalid tenant ID", nil)
		return
	}

	var req CreateTenantSocialAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request", err)
		return
	}

	platformID, err := uuid.Parse(req.PlatformID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid platform ID", nil)
		return
	}

	// Verify platform exists and is active
	var platform models.SocialMediaPlatform
	if err := h.DB.Where("id = ? AND is_active = true AND deleted_at IS NULL", platformID).First(&platform).Error; err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Platform not found or inactive", nil)
		return
	}

	// Validate and normalize handle based on platform's validation type
	normalizedHandle, err := utils.ValidateSocialHandle(platform.ValidationType, req.AccountHandle)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	// Find or create: Check if account already exists (same tenant, platform, handle)
	var existing models.TenantSocialAccount
	if err := h.DB.Where("tenant_id = ? AND platform_id = ? AND account_handle = ?",
		tenantUUID, platformID, normalizedHandle).Preload("Platform").First(&existing).Error; err == nil {
		// Account already exists - return it (find-or-create behavior)
		utils.SuccessResponse(c, http.StatusOK, "Social account already exists", gin.H{
			"account": existing,
		})
		return
	}

	account := models.TenantSocialAccount{
		TenantID:      tenantUUID,
		PlatformID:    platformID,
		AccountHandle: normalizedHandle, // Use normalized handle
		AccountURL:    req.AccountURL,
		IsActive:      true,
	}

	if err := h.DB.Create(&account).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to create social account", err)
		return
	}

	// Reload with platform
	h.DB.Preload("Platform").First(&account, "id = ?", account.ID)

	utils.SuccessResponse(c, http.StatusCreated, "Social account created", gin.H{
		"account": account,
	})
}

// Update updates a social account
func (h *TenantSocialAccountHandler) Update(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")
	accountID := c.Param("id")

	accountUUID, err := uuid.Parse(accountID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid account ID", nil)
		return
	}

	var account models.TenantSocialAccount
	if err := h.DB.Preload("Platform").Where("id = ? AND tenant_id = ?", accountUUID, tenantID).First(&account).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Social account not found", nil)
		return
	}

	var req UpdateTenantSocialAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request", err)
		return
	}

	updates := map[string]interface{}{}

	// Check for duplicate if handle is being changed
	if req.AccountHandle != "" && req.AccountHandle != account.AccountHandle {
		// Validate and normalize handle based on platform's validation type
		validationType := "text"
		if account.Platform != nil {
			validationType = account.Platform.ValidationType
		}
		normalizedHandle, err := utils.ValidateSocialHandle(validationType, req.AccountHandle)
		if err != nil {
			utils.ErrorResponse(c, http.StatusBadRequest, err.Error(), nil)
			return
		}

		var existing models.TenantSocialAccount
		if err := h.DB.Where("tenant_id = ? AND platform_id = ? AND account_handle = ? AND id != ?",
			account.TenantID, account.PlatformID, normalizedHandle, account.ID).First(&existing).Error; err == nil {
			utils.ErrorResponse(c, http.StatusConflict, "Account with this handle already exists for this platform", nil)
			return
		}
		updates["account_handle"] = normalizedHandle
	}

	if req.AccountURL != "" {
		updates["account_url"] = req.AccountURL
	}

	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}

	if len(updates) > 0 {
		if err := h.DB.Model(&account).Updates(updates).Error; err != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to update social account", err)
			return
		}
	}

	// Reload with platform
	h.DB.Preload("Platform").First(&account, "id = ?", accountUUID)

	utils.SuccessResponse(c, http.StatusOK, "Social account updated", gin.H{
		"account": account,
	})
}

// Delete deletes a social account
func (h *TenantSocialAccountHandler) Delete(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")
	accountID := c.Param("id")

	accountUUID, err := uuid.Parse(accountID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid account ID", nil)
		return
	}

	var account models.TenantSocialAccount
	if err := h.DB.Where("id = ? AND tenant_id = ?", accountUUID, tenantID).First(&account).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Social account not found", nil)
		return
	}

	// Check if account is linked to any products
	var linkCount int64
	h.DB.Model(&models.ProductSocialAccountLink{}).Where("social_account_id = ?", accountUUID).Count(&linkCount)
	if linkCount > 0 {
		utils.ErrorResponse(c, http.StatusConflict, "Cannot delete account that is linked to products. Unlink from all products first.", nil)
		return
	}

	if err := h.DB.Delete(&account).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to delete social account", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Social account deleted", nil)
}
