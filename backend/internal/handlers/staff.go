package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gamatritunggal/smartscan/backend/internal/config"
	"github.com/gamatritunggal/smartscan/backend/internal/models"
	"github.com/gamatritunggal/smartscan/backend/internal/sentry"
	"github.com/gamatritunggal/smartscan/backend/internal/services/audit"
	"github.com/gamatritunggal/smartscan/backend/internal/utils"
	"gorm.io/gorm"
)

type StaffHandler struct {
	DB  *gorm.DB
	Cfg *config.Config
}

func NewStaffHandler(db *gorm.DB, cfg *config.Config) *StaffHandler {
	return &StaffHandler{DB: db, Cfg: cfg}
}







// ========== TENANT STAFF ==========

// ListTenantStaff returns all staff for a tenant
func (h *StaffHandler) ListTenantStaff(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}
	page := utils.GetPageQuery(c)
	limit := utils.GetLimitQuery(c, 20)
	search := c.Query("search")
	role := c.Query("role")

	offset := (page - 1) * limit

	var staff []models.TenantStaff
	var total int64

	query := h.DB.Model(&models.TenantStaff{}).Where("tenant_id = ?", tenantUUID)

	if search != "" {
		query = query.Where("full_name ILIKE ?", "%"+search+"%")
	}
	if role != "" {
		query = query.Where("role = ?", role)
	}

	query.Count(&total)
	query.Preload("User").Order("created_at DESC").Offset(offset).Limit(limit).Find(&staff)

	utils.SuccessResponse(c, http.StatusOK, "Tenant staff retrieved", gin.H{
		"staff": staff,
		"pagination": gin.H{
			"page":       page,
			"limit":      limit,
			"total":      total,
			"total_page": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// CreateTenantStaff creates a new tenant staff member
func (h *StaffHandler) CreateTenantStaff(c *gin.Context) {
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

	var req struct {
		Email       string `json:"email" binding:"required,email"`
		Password    string `json:"password" binding:"required,min=8"`
		FullName    string `json:"full_name" binding:"required"`
		PhoneNumber string `json:"phone_number"`
		Position    string `json:"position"`
		Role        string `json:"role" binding:"required,oneof=admin qc_staff warehouse_staff"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request", err)
		return
	}

	// Validate and normalize email (disposable check + Gmail normalization + +suffix removal)
	normalizedEmail, err := utils.ValidateEmailForCampaign(req.Email)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error(), nil)
		return
	}
	req.Email = normalizedEmail

	// Validate and normalize phone number
	if req.PhoneNumber != "" {
		normalizedPhone, err := utils.ValidateAndNormalizePhone(req.PhoneNumber, "")
		if err != nil {
			utils.ErrorResponse(c, http.StatusBadRequest, "Invalid phone number: "+err.Error(), nil)
			return
		}
		req.PhoneNumber = normalizedPhone
	}


	// Check if email exists
	var existingUser models.User
	if err := h.DB.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Email already registered", nil)
		return
	}

	// Get creator staff ID
	var creatorStaff models.TenantStaff
	h.DB.Where("user_id = ?", userUUID).First(&creatorStaff)

	tx := h.DB.Begin()

	// Hash password
	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		tx.Rollback()
		sentry.CaptureHandlerError(c, err, "staff.CreateTenantStaff", sentry.ErrorTypeInternal, sentry.SeverityMedium)
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to hash password", err)
		return
	}

	// Create user
	user := models.User{
		Email:        req.Email,
		PasswordHash: hashedPassword,
		UserType:     models.UserTypeTenantStaff,
		Status:       models.UserStatusActive,
	}

	if err := tx.Create(&user).Error; err != nil {
		tx.Rollback()
		sentry.CaptureHandlerError(c, err, "staff.CreateTenantStaff", sentry.ErrorTypeDatabase, sentry.SeverityMedium)
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to create user", err)
		return
	}

	// Create staff
	staff := models.TenantStaff{
		TenantID:    tenantUUID,
		UserID:      user.ID,
		FullName:    req.FullName,
		PhoneNumber: req.PhoneNumber,
		Position:    req.Position,
		Role:        models.TenantStaffRole(req.Role),
	}

	if err := tx.Create(&staff).Error; err != nil {
		tx.Rollback()
		sentry.CaptureHandlerError(c, err, "staff.CreateTenantStaff", sentry.ErrorTypeDatabase, sentry.SeverityMedium)
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to create staff", err)
		return
	}

	tx.Commit()

	// Audit log
	audit.LogFromContext(c, h.DB, models.ActionTypeCreate, "tenant_staff", &staff.ID,
		nil, map[string]interface{}{"email": req.Email, "role": req.Role})


	staff.User = &user
	utils.SuccessResponse(c, http.StatusCreated, "Tenant staff created", staff)
}

// UpdateTenantStaff updates a tenant staff member
func (h *StaffHandler) UpdateTenantStaff(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}
	id := c.Param("id")
	staffID, err := uuid.Parse(id)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid staff ID", err)
		return
	}

	var staff models.TenantStaff
	if err := h.DB.Preload("User").First(&staff, "id = ? AND tenant_id = ?", staffID, tenantUUID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Staff not found", err)
		return
	}

	var req struct {
		FullName    string `json:"full_name"`
		PhoneNumber string `json:"phone_number"`
		Position    string `json:"position"`
		Role        string `json:"role"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request", err)
		return
	}

	updates := map[string]interface{}{}

	if req.FullName != "" {
		updates["full_name"] = req.FullName
	}
	if req.PhoneNumber != "" {
		updates["phone_number"] = req.PhoneNumber
	}
	if req.Position != "" {
		updates["position"] = req.Position
	}
	if req.Role != "" {
		updates["role"] = req.Role
	}

	// Detect a role change BEFORE Updates mutates staff.Role.
	roleChanged := req.Role != "" && req.Role != string(staff.Role)

	if len(updates) > 0 {
		if err := h.DB.Model(&staff).Updates(updates).Error; err != nil {
			sentry.CaptureHandlerError(c, err, "staff.UpdateTenantStaff", sentry.ErrorTypeDatabase, sentry.SeverityMedium)
			utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to update staff", err)
			return
		}
	}

	// A role change must take effect immediately. The role guards authorize on JWT
	// claims and RefreshToken re-mints from the presented token's (now stale) role,
	// so without evicting sessions the old privilege would persist for the whole
	// refresh window. Force a re-login so the next token carries the new role.
	if roleChanged {
		_ = utils.NewTokenBlacklist().RevokeUserTokens(staff.UserID.String(), time.Duration(h.Cfg.JWT.RefreshHours)*time.Hour)
	}

	// Re-fetch for response with preloads
	h.DB.Preload("User").First(&staff, "id = ?", staffID)

	// Audit log
	audit.LogFromContext(c, h.DB, models.ActionTypeUpdate, "tenant_staff", &staff.ID, nil, updates)

	utils.SuccessResponse(c, http.StatusOK, "Staff updated", staff)
}

// DeleteTenantStaff soft deletes a tenant staff member
func (h *StaffHandler) DeleteTenantStaff(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}
	id := c.Param("id")
	staffID, err := uuid.Parse(id)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid staff ID", err)
		return
	}

	var staff models.TenantStaff
	if err := h.DB.Preload("User").First(&staff, "id = ? AND tenant_id = ?", staffID, tenantUUID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Staff not found", err)
		return
	}

	// Cannot delete primary admin
	if staff.IsPrimaryAdmin {
		utils.ErrorResponse(c, http.StatusBadRequest, "Cannot delete primary admin", nil)
		return
	}


	// Soft delete staff and user
	h.DB.Delete(&staff)
	h.DB.Delete(&models.User{}, "id = ?", staff.UserID)

	// Evict the deleted user's sessions. AuthMiddleware authorizes purely on JWT
	// claims and never re-checks the DB, so without this the access token keeps
	// working until it expires (default 24h). Fail-open if Redis is unavailable.
	_ = utils.NewTokenBlacklist().RevokeUserTokens(staff.UserID.String(), time.Duration(h.Cfg.JWT.RefreshHours)*time.Hour)

	// Audit log
	audit.LogFromContext(c, h.DB, models.ActionTypeDelete, "tenant_staff", &staff.ID, map[string]interface{}{"full_name": staff.FullName, "role": staff.Role}, nil)

	utils.SuccessResponse(c, http.StatusOK, "Staff deleted", nil)
}
