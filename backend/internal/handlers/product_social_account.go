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

type ProductSocialAccountHandler struct {
	DB  *gorm.DB
	Cfg *config.Config
}

func NewProductSocialAccountHandler(db *gorm.DB, cfg *config.Config) *ProductSocialAccountHandler {
	return &ProductSocialAccountHandler{DB: db, Cfg: cfg}
}

// LinkSocialAccountRequest represents the request to link a social account to a product
type LinkSocialAccountRequest struct {
	SocialAccountID string `json:"social_account_id" binding:"required"`
}

// ReorderSocialAccountsRequest represents the request to reorder social account links
type ReorderSocialAccountsRequest struct {
	LinkIDs []string `json:"link_ids" binding:"required"`
}

// ProductSocialAccountLinkResponse represents a social account link with full details
type ProductSocialAccountLinkResponse struct {
	ID            uuid.UUID                         `json:"id"`
	SocialAccount *TenantSocialAccountWithPlatform  `json:"social_account"`
	SortOrder     int                               `json:"sort_order"`
	CreatedAt     string                            `json:"created_at"`
}

// TenantSocialAccountWithPlatform includes platform info
type TenantSocialAccountWithPlatform struct {
	ID            uuid.UUID                   `json:"id"`
	Platform      *models.SocialMediaPlatform `json:"platform"`
	AccountHandle string                      `json:"account_handle"`
	AccountURL    string                      `json:"account_url,omitempty"`
}

// List returns all social account links for a product
func (h *ProductSocialAccountHandler) List(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")
	productID := c.Param("id")

	productUUID, err := uuid.Parse(productID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid product ID", nil)
		return
	}

	// Verify product belongs to tenant
	var product models.Product
	if err := h.DB.Where("id = ? AND tenant_id = ? AND deleted_at IS NULL", productUUID, tenantID).First(&product).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Product not found", nil)
		return
	}

	var links []models.ProductSocialAccountLink
	if err := h.DB.Where("product_id = ?", productUUID).
		Preload("SocialAccount.Platform").
		Order("sort_order ASC").
		Find(&links).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to fetch social account links", err)
		return
	}

	var responses []ProductSocialAccountLinkResponse
	for _, link := range links {
		var accountResp *TenantSocialAccountWithPlatform
		if link.SocialAccount != nil {
			accountResp = &TenantSocialAccountWithPlatform{
				ID:            link.SocialAccount.ID,
				Platform:      link.SocialAccount.Platform,
				AccountHandle: link.SocialAccount.AccountHandle,
				AccountURL:    link.SocialAccount.AccountURL,
			}
		}
		responses = append(responses, ProductSocialAccountLinkResponse{
			ID:            link.ID,
			SocialAccount: accountResp,
			SortOrder:     link.SortOrder,
			CreatedAt:     link.CreatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	utils.SuccessResponse(c, http.StatusOK, "Social account links retrieved", gin.H{
		"links": responses,
	})
}

// Link links a social account to a product
func (h *ProductSocialAccountHandler) Link(c *gin.Context) {
	tenantIDStr, _ := c.Get("tenant_id")
	tenantUUID, err := uuid.Parse(tenantIDStr.(string))
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Invalid tenant ID", nil)
		return
	}
	productID := c.Param("id")

	productUUID, err := uuid.Parse(productID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid product ID", nil)
		return
	}

	// Verify product belongs to tenant
	var product models.Product
	if err := h.DB.Where("id = ? AND tenant_id = ? AND deleted_at IS NULL", productUUID, tenantUUID).First(&product).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Product not found", nil)
		return
	}

	var req LinkSocialAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request", err)
		return
	}

	socialAccountID, err := uuid.Parse(req.SocialAccountID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid social account ID", nil)
		return
	}

	// Verify social account belongs to same tenant and is active
	var socialAccount models.TenantSocialAccount
	if err := h.DB.Where("id = ? AND tenant_id = ? AND is_active = true", socialAccountID, tenantUUID).
		First(&socialAccount).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Social account not found or inactive", nil)
		return
	}

	// Check if already linked
	var existing models.ProductSocialAccountLink
	if err := h.DB.Where("product_id = ? AND social_account_id = ?", productUUID, socialAccountID).
		First(&existing).Error; err == nil {
		utils.ErrorResponse(c, http.StatusConflict, "Social account already linked to this product", nil)
		return
	}

	// Get next sort order
	var maxOrder int
	h.DB.Model(&models.ProductSocialAccountLink{}).
		Where("product_id = ?", productUUID).
		Select("COALESCE(MAX(sort_order), -1)").
		Scan(&maxOrder)

	link := models.ProductSocialAccountLink{
		ProductID:       productUUID,
		SocialAccountID: socialAccountID,
		SortOrder:       maxOrder + 1,
	}

	if err := h.DB.Create(&link).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to link social account", err)
		return
	}

	// Reload with relations
	h.DB.Preload("SocialAccount.Platform").First(&link, "id = ?", link.ID)

	utils.SuccessResponse(c, http.StatusCreated, "Social account linked", gin.H{
		"link": ProductSocialAccountLinkResponse{
			ID: link.ID,
			SocialAccount: &TenantSocialAccountWithPlatform{
				ID:            link.SocialAccount.ID,
				Platform:      link.SocialAccount.Platform,
				AccountHandle: link.SocialAccount.AccountHandle,
				AccountURL:    link.SocialAccount.AccountURL,
			},
			SortOrder: link.SortOrder,
			CreatedAt: link.CreatedAt.Format("2006-01-02T15:04:05Z"),
		},
	})
}

// Unlink removes a social account link from a product
func (h *ProductSocialAccountHandler) Unlink(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")
	productID := c.Param("id")
	linkID := c.Param("link_id")

	productUUID, err := uuid.Parse(productID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid product ID", nil)
		return
	}

	linkUUID, err := uuid.Parse(linkID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid link ID", nil)
		return
	}

	// Verify product belongs to tenant
	var product models.Product
	if err := h.DB.Where("id = ? AND tenant_id = ? AND deleted_at IS NULL", productUUID, tenantID).First(&product).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Product not found", nil)
		return
	}

	// Find and delete the link
	var link models.ProductSocialAccountLink
	if err := h.DB.Where("id = ? AND product_id = ?", linkUUID, productUUID).First(&link).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Link not found", nil)
		return
	}

	if err := h.DB.Delete(&link).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to unlink social account", err)
		return
	}

	// Reorder remaining links
	var remainingLinks []models.ProductSocialAccountLink
	h.DB.Where("product_id = ?", productUUID).Order("sort_order ASC").Find(&remainingLinks)
	for i, l := range remainingLinks {
		if l.SortOrder != i {
			h.DB.Model(&l).Update("sort_order", i)
		}
	}

	utils.SuccessResponse(c, http.StatusOK, "Social account unlinked", nil)
}

// Reorder updates the sort order of social account links
func (h *ProductSocialAccountHandler) Reorder(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")
	productID := c.Param("id")

	productUUID, err := uuid.Parse(productID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid product ID", nil)
		return
	}

	// Verify product belongs to tenant
	var product models.Product
	if err := h.DB.Where("id = ? AND tenant_id = ? AND deleted_at IS NULL", productUUID, tenantID).First(&product).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Product not found", nil)
		return
	}

	var req ReorderSocialAccountsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request", err)
		return
	}

	// Validate: no duplicate IDs
	seen := make(map[string]bool)
	for _, id := range req.LinkIDs {
		if seen[id] {
			utils.ErrorResponse(c, http.StatusBadRequest, "Duplicate link ID in list", nil)
			return
		}
		seen[id] = true
	}

	// Validate: must include all links for this product
	var existingCount int64
	h.DB.Model(&models.ProductSocialAccountLink{}).Where("product_id = ?", productUUID).Count(&existingCount)
	if int64(len(req.LinkIDs)) != existingCount {
		utils.ErrorResponse(c, http.StatusBadRequest, "Must include all links in reorder request", nil)
		return
	}

	// Update sort_order within a transaction
	tx := h.DB.Begin()
	for i, linkIDStr := range req.LinkIDs {
		linkID, err := uuid.Parse(linkIDStr)
		if err != nil {
			tx.Rollback()
			utils.ErrorResponse(c, http.StatusBadRequest, "Invalid link ID in list", nil)
			return
		}

		var link models.ProductSocialAccountLink
		if err := tx.Where("id = ? AND product_id = ?", linkID, productUUID).First(&link).Error; err != nil {
			tx.Rollback()
			utils.ErrorResponse(c, http.StatusNotFound, "Link not found or does not belong to this product", nil)
			return
		}

		if err := tx.Model(&link).Update("sort_order", i).Error; err != nil {
			tx.Rollback()
			utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to update sort order", err)
			return
		}
	}

	if err := tx.Commit().Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to commit reorder", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Social account links reordered", nil)
}
