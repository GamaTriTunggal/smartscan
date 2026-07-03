package audit

import (
	"encoding/json"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gamatritunggal/smartscan/backend/internal/models"
	"github.com/gamatritunggal/smartscan/backend/internal/utils"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Entry represents data needed to create an audit log
type Entry struct {
	UserID     *uuid.UUID
	TenantID   *uuid.UUID
	Action     models.ActionType
	EntityType string
	EntityID   *uuid.UUID
	OldValues  interface{}
	NewValues  interface{}
	IPAddress  string
	UserAgent  string
}

// Log writes an audit log entry in a background goroutine (non-blocking)
func Log(db *gorm.DB, e Entry) {
	go func() {
		var oldJSON, newJSON datatypes.JSON
		if e.OldValues != nil {
			if b, err := json.Marshal(e.OldValues); err == nil {
				oldJSON = b
			}
		}
		if e.NewValues != nil {
			if b, err := json.Marshal(e.NewValues); err == nil {
				newJSON = b
			}
		}

		// PostgreSQL INET type rejects empty string — default to 0.0.0.0
		ipAddress := e.IPAddress
		if ipAddress == "" {
			ipAddress = "0.0.0.0"
		}

		record := models.ActivityLog{
			UserID:     e.UserID,
			TenantID:   e.TenantID,
			ActionType: e.Action,
			EntityType: e.EntityType,
			EntityID:   e.EntityID,
			OldValues:  oldJSON,
			NewValues:  newJSON,
			IPAddress:  ipAddress,
			UserAgent:  e.UserAgent,
		}

		if err := db.Create(&record).Error; err != nil {
			log.Printf("[AUDIT-ERROR] Failed to write audit log: %v", err)
		}
	}()
}

// LogFromContext extracts user/tenant/IP from Gin context and logs asynchronously.
func LogFromContext(c *gin.Context, db *gorm.DB, action models.ActionType, entityType string, entityID *uuid.UUID, oldValues, newValues interface{}) {
	var userID *uuid.UUID
	if uid, ok := utils.GetUserUUID(c); ok {
		userID = &uid
	}

	var tenantID *uuid.UUID
	if tid, ok := utils.GetTenantUUID(c); ok {
		tenantID = &tid
	}

	Log(db, Entry{
		UserID:     userID,
		TenantID:   tenantID,
		Action:     action,
		EntityType: entityType,
		EntityID:   entityID,
		OldValues:  oldValues,
		NewValues:  newValues,
		IPAddress:  c.ClientIP(),
		UserAgent:  c.Request.UserAgent(),
	})
}
