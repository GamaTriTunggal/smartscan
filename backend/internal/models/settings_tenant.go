package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// TenantSettings stores per-company key/value settings (e.g. counterfeit thresholds)
type TenantSettings struct {
	ID           uuid.UUID      `gorm:"type:uuid;primary_key;default:uuidv7()" json:"id"`
	TenantID     uuid.UUID      `gorm:"type:uuid;not null" json:"tenant_id"`
	SettingKey   string         `gorm:"type:varchar(100);not null" json:"setting_key"`
	SettingValue datatypes.JSON `gorm:"type:jsonb;not null" json:"setting_value"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`

	// Relations
	Tenant *Tenant `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
}

func (TenantSettings) TableName() string {
	return "tenant_settings"
}
