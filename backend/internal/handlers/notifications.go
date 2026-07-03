package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/gamatritunggal/smartscan/backend/internal/config"
	"github.com/gamatritunggal/smartscan/backend/internal/models"
	"github.com/gamatritunggal/smartscan/backend/internal/utils"
)

type NotificationsHandler struct {
	DB  *gorm.DB
	Cfg *config.Config
}

func NewNotificationsHandler(db *gorm.DB, cfg *config.Config) *NotificationsHandler {
	return &NotificationsHandler{DB: db, Cfg: cfg}
}

// parsePositiveInt parses s as an int in [0, max].
func parsePositiveInt(s string, max int) (int, error) {
	v, err := strconv.Atoi(s)
	if err != nil {
		return 0, err
	}
	if v < 0 {
		v = 0
	}
	if v > max {
		v = max
	}
	return v, nil
}

// Notify creates an in-app notification and fires the outbound webhook (if one
// is configured for this event type). Best-effort: failures are logged, never
// returned — a notification must not break the flow that triggered it.
func Notify(db *gorm.DB, tenantID uuid.UUID, ntype models.NotificationType, title, body, link string, data map[string]interface{}) {
	var payload []byte
	if data != nil {
		payload, _ = json.Marshal(data)
	}

	n := models.Notification{
		TenantID: tenantID,
		Type:     ntype,
		Title:    title,
		Body:     body,
		Link:     link,
		Data:     payload,
	}
	if err := db.Create(&n).Error; err != nil {
		log.Printf("[NOTIFY] failed to create notification (%s): %v", ntype, err)
	}

	utils.SendWebhook(db, tenantID, string(ntype), map[string]interface{}{
		"event": string(ntype),
		"title": title,
		"body":  body,
		"link":  link,
		"data":  data,
	})
}

// List returns the newest notifications plus the unread count.
// GET /tenant/notifications?limit=20&offset=0&unread_only=true
func (h *NotificationsHandler) List(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}

	limit := 20
	if l := c.Query("limit"); l != "" {
		if v, err := parsePositiveInt(l, 100); err == nil {
			limit = v
		}
	}
	offset := 0
	if o := c.Query("offset"); o != "" {
		if v, err := parsePositiveInt(o, 1<<30); err == nil {
			offset = v
		}
	}

	query := h.DB.Where("tenant_id = ?", tenantUUID)
	if c.Query("unread_only") == "true" {
		query = query.Where("read_at IS NULL")
	}

	var notifications []models.Notification
	if err := query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&notifications).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to load notifications", err)
		return
	}

	var unread int64
	h.DB.Model(&models.Notification{}).Where("tenant_id = ? AND read_at IS NULL", tenantUUID).Count(&unread)

	utils.SuccessResponse(c, http.StatusOK, "Notifications", gin.H{
		"notifications": notifications,
		"unread_count":  unread,
	})
}

// MarkRead marks one notification as read.
// POST /tenant/notifications/:id/read
func (h *NotificationsHandler) MarkRead(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid notification ID", err)
		return
	}

	now := time.Now().UTC()
	res := h.DB.Model(&models.Notification{}).
		Where("id = ? AND tenant_id = ? AND read_at IS NULL", id, tenantUUID).
		Update("read_at", now)
	if res.Error != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to mark notification read", res.Error)
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "Notification marked read", nil)
}

// MarkAllRead marks every unread notification as read.
// POST /tenant/notifications/read-all
func (h *NotificationsHandler) MarkAllRead(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}

	now := time.Now().UTC()
	if err := h.DB.Model(&models.Notification{}).
		Where("tenant_id = ? AND read_at IS NULL", tenantUUID).
		Update("read_at", now).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to mark notifications read", err)
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "All notifications marked read", nil)
}
