package handlers

import (
	"encoding/json"
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/gamatritunggal/smartscan/backend/internal/config"
	"github.com/gamatritunggal/smartscan/backend/internal/models"
	"github.com/gamatritunggal/smartscan/backend/internal/utils"
)

type WebhookSettingsHandler struct {
	DB  *gorm.DB
	Cfg *config.Config
}

func NewWebhookSettingsHandler(db *gorm.DB, cfg *config.Config) *WebhookSettingsHandler {
	return &WebhookSettingsHandler{DB: db, Cfg: cfg}
}

// knownWebhookEvents lists every event type the app can emit.
var knownWebhookEvents = []string{
	"warranty_registered",
	"counterfeit_alert",
	"geofence_violation",
	"qr_batch_ready",
}

// Get returns the current webhook configuration (secret masked).
// GET /tenant/integrations/webhook
func (h *WebhookSettingsHandler) Get(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}

	cfg, err := utils.LoadWebhookConfig(h.DB, tenantUUID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to load webhook settings", err)
		return
	}

	hasSecret := cfg.Secret != ""
	cfg.Secret = "" // never echo the secret back

	utils.SuccessResponse(c, http.StatusOK, "Webhook settings", gin.H{
		"config":     cfg,
		"has_secret": hasSecret,
		"events":     knownWebhookEvents,
	})
}

type updateWebhookRequest struct {
	URL     string   `json:"url" binding:"omitempty,max=2000"`
	Secret  *string  `json:"secret"` // nil = keep existing secret
	Enabled bool     `json:"enabled"`
	Events  []string `json:"events"`
}

// Update stores the webhook configuration in tenant_settings.
// PUT /tenant/integrations/webhook
func (h *WebhookSettingsHandler) Update(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}

	var req updateWebhookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request", err)
		return
	}

	if req.Enabled {
		u, err := url.Parse(req.URL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			utils.ErrorResponse(c, http.StatusBadRequest, "Webhook URL must be a valid http(s) URL", nil)
			return
		}
	}
	for _, e := range req.Events {
		valid := false
		for _, k := range knownWebhookEvents {
			if e == k {
				valid = true
				break
			}
		}
		if !valid {
			utils.ErrorResponse(c, http.StatusBadRequest, "Unknown event type: "+e, nil)
			return
		}
	}

	existing, err := utils.LoadWebhookConfig(h.DB, tenantUUID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to load webhook settings", err)
		return
	}

	newCfg := utils.WebhookConfig{
		URL:     req.URL,
		Enabled: req.Enabled,
		Events:  req.Events,
		Secret:  existing.Secret,
	}
	if req.Secret != nil {
		newCfg.Secret = *req.Secret
	}

	value, err := json.Marshal(newCfg)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to encode webhook settings", err)
		return
	}

	setting := models.TenantSettings{
		TenantID:     tenantUUID,
		SettingKey:   "webhook",
		SettingValue: value,
		UpdatedAt:    time.Now().UTC(),
	}
	if err := h.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "tenant_id"}, {Name: "setting_key"}},
		DoUpdates: clause.AssignmentColumns([]string{"setting_value", "updated_at"}),
	}).Create(&setting).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to save webhook settings", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Webhook settings saved", nil)
}
