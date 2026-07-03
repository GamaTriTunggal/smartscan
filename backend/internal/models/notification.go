package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// NotificationType identifies what triggered an in-app notification.
type NotificationType string

const (
	NotificationTypeCounterfeitAlert  NotificationType = "counterfeit_alert"
	NotificationTypeGeofenceViolation NotificationType = "geofence_violation"
	NotificationTypeQRBatchReady      NotificationType = "qr_batch_ready"
)

// Notification is an in-app notification shown in the notification center.
// This replaces the email pipeline of the original SaaS: admins see alerts on
// the bell icon / dashboard instead of receiving emails.
type Notification struct {
	ID       uuid.UUID        `gorm:"type:uuid;primary_key;default:uuidv7()" json:"id"`
	TenantID uuid.UUID        `gorm:"type:uuid;not null;index" json:"tenant_id"`
	Type     NotificationType `gorm:"type:varchar(50);not null" json:"type"`
	Title    string           `gorm:"type:varchar(255);not null" json:"title"`
	Body     string           `gorm:"type:text" json:"body"`
	// Link is a frontend route the notification points at (e.g. /tenant/counterfeit).
	Link string `gorm:"type:text" json:"link"`
	// Data carries optional structured payload for the webhook / UI.
	Data      datatypes.JSON `gorm:"type:jsonb" json:"data,omitempty"`
	ReadAt    *time.Time     `json:"read_at,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
}

func (Notification) TableName() string {
	return "notifications"
}
