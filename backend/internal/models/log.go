package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// ActionType enum
type ActionType string

const (
	ActionTypeLogin         ActionType = "login"
	ActionTypeLogout        ActionType = "logout"
	ActionTypeCreate        ActionType = "create"
	ActionTypeUpdate        ActionType = "update"
	ActionTypeDelete        ActionType = "delete"
	ActionTypeExport        ActionType = "export"
	ActionTypePasswordReset      ActionType = "password_reset"
	ActionTypeThresholdOverride  ActionType = "threshold_override"
)

// ActivityLog represents audit trail for all system activities
type ActivityLog struct {
	ID                  uuid.UUID      `gorm:"type:uuid;primary_key;default:uuidv7()" json:"id"`
	UserID              *uuid.UUID     `gorm:"type:uuid" json:"user_id,omitempty"`
	TenantID            *uuid.UUID     `gorm:"type:uuid" json:"tenant_id,omitempty"`
	ActionType          ActionType     `gorm:"type:varchar(100)" json:"action_type"`
	EntityType          string         `gorm:"type:varchar(100)" json:"entity_type"`
	EntityID            *uuid.UUID     `gorm:"type:uuid" json:"entity_id,omitempty"`
	OldValues           datatypes.JSON `gorm:"type:jsonb" json:"old_values"`
	NewValues           datatypes.JSON `gorm:"type:jsonb" json:"new_values"`
	IPAddress           string         `gorm:"type:inet" json:"ip_address"`
	UserAgent           string         `gorm:"type:text" json:"user_agent"`
	CreatedAt           time.Time      `json:"created_at"`

	// Relations
	User   *User   `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Tenant *Tenant `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
}

func (ActivityLog) TableName() string {
	return "activity_logs"
}
