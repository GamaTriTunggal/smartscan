package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/gamatritunggal/smartscan/backend/internal/config"
	"github.com/gamatritunggal/smartscan/backend/internal/models"
	"github.com/gamatritunggal/smartscan/backend/internal/utils"
)

// CompanyContact is what the brand owner chooses to show consumers on every
// public page — most importantly on the counterfeit verdict, where "believe
// this is an error? contact us" matters most. Every field is optional; only
// filled fields are rendered.
type CompanyContact struct {
	Phone    string `json:"phone"`
	WhatsApp string `json:"whatsapp"`
	Email    string `json:"email"`
	Website  string `json:"website"`
	Address  string `json:"address"`
}

const companyContactKey = "public_contact"

type CompanyContactHandler struct {
	DB  *gorm.DB
	Cfg *config.Config
}

func NewCompanyContactHandler(db *gorm.DB, cfg *config.Config) *CompanyContactHandler {
	return &CompanyContactHandler{DB: db, Cfg: cfg}
}

// loadCompanyContact loads the contact for the globally-oldest tenant. Used only
// by the public endpoint, which has no tenant context (single-brand consumer pages).
func loadCompanyContact(db *gorm.DB) (CompanyContact, string) {
	var tenant models.Tenant
	if err := db.Select("id, company_name").Order("created_at ASC").First(&tenant).Error; err != nil {
		return CompanyContact{}, ""
	}
	return loadCompanyContactForTenant(db, tenant.ID)
}

// loadCompanyContactForTenant loads the contact for a specific tenant. Used by the
// authenticated admin endpoint so the settings form always reflects the caller's
// own tenant (and round-trips consistently with Update).
func loadCompanyContactForTenant(db *gorm.DB, tenantID uuid.UUID) (CompanyContact, string) {
	var contact CompanyContact
	companyName := ""

	var tenant models.Tenant
	if err := db.Select("id, company_name").Where("id = ?", tenantID).First(&tenant).Error; err != nil {
		return contact, companyName
	}
	companyName = tenant.CompanyName

	var row struct{ SettingValue []byte }
	err := db.Table("tenant_settings").
		Select("setting_value").
		Where("tenant_id = ? AND setting_key = ?", tenant.ID, companyContactKey).
		Scan(&row).Error
	if err == nil && len(row.SettingValue) > 0 {
		_ = json.Unmarshal(row.SettingValue, &contact)
	}
	return contact, companyName
}

// GetPublic returns the company contact for public pages (no auth).
// GET /api/v1/public/company-contact
func (h *CompanyContactHandler) GetPublic(c *gin.Context) {
	contact, companyName := loadCompanyContact(h.DB)
	utils.SuccessResponse(c, http.StatusOK, "Company contact", gin.H{
		"company_name": companyName,
		"contact":      contact,
	})
}

// Get returns the contact settings for the admin UI, scoped to the caller's tenant.
// GET /tenant/company-contact
func (h *CompanyContactHandler) Get(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}
	contact, companyName := loadCompanyContactForTenant(h.DB, tenantUUID)
	utils.SuccessResponse(c, http.StatusOK, "Company contact", gin.H{
		"company_name": companyName,
		"contact":      contact,
	})
}

// Update stores the contact settings.
// PUT /tenant/company-contact (admin)
func (h *CompanyContactHandler) Update(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}

	var req CompanyContact
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request", err)
		return
	}

	value, err := json.Marshal(req)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to encode contact", err)
		return
	}

	setting := models.TenantSettings{
		TenantID:     tenantUUID,
		SettingKey:   companyContactKey,
		SettingValue: value,
		UpdatedAt:    time.Now().UTC(),
	}
	if err := h.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "tenant_id"}, {Name: "setting_key"}},
		DoUpdates: clause.AssignmentColumns([]string{"setting_value", "updated_at"}),
	}).Create(&setting).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to save contact", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Company contact saved", nil)
}
