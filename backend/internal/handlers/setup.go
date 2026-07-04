package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/gamatritunggal/smartscan/backend/internal/config"
	"github.com/gamatritunggal/smartscan/backend/internal/models"
	"github.com/gamatritunggal/smartscan/backend/internal/utils"
)

// SetupHandler implements the first-run setup wizard. A fresh deployment has
// no company and no users; the first visitor creates both. Once a company row
// exists, every endpoint here refuses to run again.
type SetupHandler struct {
	DB  *gorm.DB
	Cfg *config.Config
}

func NewSetupHandler(db *gorm.DB, cfg *config.Config) *SetupHandler {
	return &SetupHandler{DB: db, Cfg: cfg}
}

// Status reports whether first-run setup is still needed.
// GET /api/v1/setup/status
func (h *SetupHandler) Status(c *gin.Context) {
	var count int64
	if err := h.DB.Model(&models.Tenant{}).Count(&count).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to check setup status", err)
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "Setup status", gin.H{
		"needs_setup": count == 0,
	})
}

type setupRequest struct {
	CompanyName string `json:"company_name" binding:"required,max=255"`
	AdminName   string `json:"admin_name" binding:"required,max=255"`
	Email       string `json:"email" binding:"required,email,max=255"`
	Password    string `json:"password" binding:"required,min=8,max=128"`
}

// Run performs first-run setup: creates the company, the admin account, and
// signs the admin in. Refuses with 409 once a company exists (single-shot).
// POST /api/v1/setup
func (h *SetupHandler) Run(c *gin.Context) {
	var req setupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request", err)
		return
	}
	if strings.ToLower(req.Password) == "password" {
		utils.ErrorResponse(c, http.StatusBadRequest, "Please choose a stronger password", nil)
		return
	}

	req.Email = utils.NormalizeEmail(req.Email)

	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to process password", err)
		return
	}

	var tenant models.Tenant
	var user models.User

	err = h.DB.Transaction(func(tx *gorm.DB) error {
		// Single-shot guard, race-safe: lock the tenants table for this check.
		var count int64
		if err := tx.Exec("LOCK TABLE tenants IN SHARE ROW EXCLUSIVE MODE").Error; err != nil {
			return err
		}
		if err := tx.Model(&models.Tenant{}).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return errSetupAlreadyDone
		}

		tenant = models.Tenant{
			CompanyName:  req.CompanyName,
			CompanyEmail: req.Email,
		}
		if err := tx.Create(&tenant).Error; err != nil {
			return err
		}

		user = models.User{
			Email:        req.Email,
			PasswordHash: hashedPassword,
			UserType:     models.UserTypeTenantStaff,
			Status:       models.UserStatusActive,
		}
		if err := tx.Create(&user).Error; err != nil {
			return err
		}

		staff := models.TenantStaff{
			TenantID:       tenant.ID,
			UserID:         user.ID,
			FullName:       req.AdminName,
			Role:           models.TenantStaffRoleAdmin,
			IsPrimaryAdmin: true,
		}
		return tx.Create(&staff).Error
	})
	if err == errSetupAlreadyDone {
		utils.ErrorResponse(c, http.StatusConflict, "Setup has already been completed", nil)
		return
	}
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Setup failed", err)
		return
	}

	// Default landing/warranty templates are created lazily on first use by
	// the template handler (ensureDefaultTemplates).

	tokenPair, err := utils.GenerateTokenPair(
		h.Cfg.JWT.Secret,
		user.ID,
		user.Email,
		string(user.UserType),
		string(models.TenantStaffRoleAdmin),
		&tenant.ID,
		h.Cfg.JWT.ExpirationHours,
		h.Cfg.JWT.RefreshHours,
		false,
	)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Setup completed, but sign-in failed — please log in manually", err)
		return
	}
	utils.SetTokenCookies(c, tokenPair, h.Cfg.JWT.ExpirationHours, h.Cfg.JWT.RefreshHours)

	utils.SuccessResponse(c, http.StatusCreated, "Setup completed", gin.H{
		"company": gin.H{"id": tenant.ID, "company_name": tenant.CompanyName},
		"user": gin.H{
			"id":        user.ID,
			"email":     user.Email,
			"user_type": user.UserType,
			"role":      models.TenantStaffRoleAdmin,
			"full_name": req.AdminName,
			"tenant_id": tenant.ID,
		},
		"tokens":     tokenPair,
		"expires_in": h.Cfg.JWT.ExpirationHours * 3600,
	})
}

var errSetupAlreadyDone = &setupDoneError{}

type setupDoneError struct{}

func (*setupDoneError) Error() string { return "setup already completed" }
