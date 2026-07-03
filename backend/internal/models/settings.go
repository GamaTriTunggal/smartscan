package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// AppSettings represents application-wide settings (key-value with JSONB)
type AppSettings struct {
	ID           uuid.UUID      `gorm:"type:uuid;primary_key;default:uuidv7()" json:"id"`
	SettingKey   string         `gorm:"type:varchar(100);uniqueIndex;not null" json:"setting_key"`
	SettingValue datatypes.JSON `gorm:"type:jsonb;not null" json:"setting_value"`
	UpdatedBy    *uuid.UUID     `gorm:"type:uuid" json:"updated_by,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`

	// Relations
}

func (AppSettings) TableName() string {
	return "app_settings"
}

// BrandingSettings represents the branding configuration structure
type BrandingSettings struct {
	AppName             string `json:"app_name"`
	LogoURL             string `json:"logo_url"`
	HeaderGradientStart string `json:"header_gradient_start"`
	HeaderGradientEnd   string `json:"header_gradient_end"`
	HeaderTextColor     string `json:"header_text_color"`
	ButtonBgColor       string `json:"button_bg_color"`
	ButtonTextColor     string `json:"button_text_color"`
}

// DefaultBrandingSettings returns the default branding configuration
func DefaultBrandingSettings() BrandingSettings {
	return BrandingSettings{
		AppName:             "smartscan",
		LogoURL:             "",
		HeaderGradientStart: "#0d9488",
		HeaderGradientEnd:   "#30e3e3",
		HeaderTextColor:     "#ffffff",
		ButtonBgColor:       "#00b4b4",
		ButtonTextColor:     "#ffffff",
	}
}

