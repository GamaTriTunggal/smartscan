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

type TenantHandler struct {
	DB  *gorm.DB
	Cfg *config.Config
}

func NewTenantHandler(db *gorm.DB, cfg *config.Config) *TenantHandler {
	return &TenantHandler{DB: db, Cfg: cfg}
}





// GetMyTenant returns the current user's tenant info (for tenant staff)
func (h *TenantHandler) GetMyTenant(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}

	var tenant models.Tenant
	if err := h.DB.First(&tenant, "id = ?", tenantUUID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Tenant not found", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Tenant retrieved", tenant)
}

// UpdateMyTenant updates the current user's tenant info
func (h *TenantHandler) UpdateMyTenant(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}

	var req struct {
		CompanyName    string `json:"company_name"`
		CompanyAddress string `json:"company_address"`
		PhoneNumber    string `json:"phone_number"`
		BusinessField  string `json:"business_field"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request", err)
		return
	}

	var tenant models.Tenant
	if err := h.DB.First(&tenant, "id = ?", tenantUUID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Tenant not found", err)
		return
	}

	updates := map[string]interface{}{}

	if req.CompanyName != "" {
		updates["company_name"] = req.CompanyName
	}
	if req.CompanyAddress != "" {
		updates["company_address"] = req.CompanyAddress
	}
	if req.PhoneNumber != "" {
		updates["phone_number"] = req.PhoneNumber
	}
	if req.BusinessField != "" {
		updates["business_field"] = req.BusinessField
	}

	if len(updates) > 0 {
		if err := h.DB.Model(&tenant).Updates(updates).Error; err != nil {
			sentry.CaptureHandlerError(c, err, "tenant.UpdateMyTenant", sentry.ErrorTypeDatabase, sentry.SeverityMedium)
			utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to update tenant", err)
			return
		}
	}

	// Re-fetch for response
	h.DB.First(&tenant, "id = ?", tenantUUID)

	utils.SuccessResponse(c, http.StatusOK, "Tenant updated", tenant)
}

// ResetTenantStaffPassword resets a tenant staff member's password (Super Admin only)
func (h *TenantHandler) ResetTenantStaffPassword(c *gin.Context) {
	tenantID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}

	staffID, err := uuid.Parse(c.Param("staff_id"))
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid staff ID", err)
		return
	}

	// Find staff member and verify they belong to this tenant
	var staff models.TenantStaff
	if err := h.DB.Preload("User").First(&staff, "id = ? AND tenant_id = ?", staffID, tenantID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Staff not found", nil)
		return
	}

	if staff.User == nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Staff has no linked user account", nil)
		return
	}

	if staff.User.Status != "active" {
		utils.ErrorResponse(c, http.StatusBadRequest, "Cannot reset password for inactive user", nil)
		return
	}

	// Get tenant for email context
	var tenant models.Tenant
	if err := h.DB.First(&tenant, "id = ?", tenantID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Tenant not found", nil)
		return
	}

	// Generate secure temp password
	tempPassword, err := utils.GenerateSecureTempPassword(8)
	if err != nil {
		utils.ErrorResponse(c, http.StatusServiceUnavailable, "Unable to generate secure password. Please try again.", err)
		return
	}

	hashedPassword, err := utils.HashPassword(tempPassword)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to hash password", err)
		return
	}

	// Update user password and set must_change_password flag
	if err := h.DB.Model(&models.User{}).Where("id = ?", staff.UserID).Updates(map[string]interface{}{
		"password_hash":        hashedPassword,
		"must_change_password": true,
	}).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to reset password", err)
		return
	}

	// Revoke all existing tokens (force re-login)
	utils.NewTokenBlacklist().RevokeUserTokens(staff.UserID.String(), 168*time.Hour)

	// The temp password is returned ONCE to the admin, who hands it to the staff
	// member directly. The staff member must change it on first login.
	utils.SuccessResponse(c, http.StatusOK, "Password reset", gin.H{
		"email":         staff.User.Email,
		"temp_password": tempPassword,
	})
}
