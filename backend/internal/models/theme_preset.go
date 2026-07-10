package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PresetType enum
type PresetType string

const (
	PresetTypeLanding PresetType = "landing"
)

// ThemePreset represents admin-managed background theme presets
type ThemePreset struct {
	ID             uuid.UUID      `gorm:"type:uuid;primary_key;default:uuidv7()" json:"id"`
	Name           string         `gorm:"type:varchar(255);not null" json:"name"`
	Description    string         `gorm:"type:text" json:"description"`
	PresetType     PresetType     `gorm:"type:varchar(20);not null" json:"preset_type"`
	BackgroundURL  string         `gorm:"type:text;not null" json:"background_url"`
	ThumbnailURL   string         `gorm:"type:text" json:"thumbnail_url"`
	OverlayColor   string         `gorm:"type:varchar(7);default:'#000000'" json:"overlay_color"`
	OverlayOpacity int            `gorm:"default:30" json:"overlay_opacity"`
	CardOpacity    int            `gorm:"default:90" json:"card_opacity"`
	CardBlur       int            `gorm:"default:0" json:"card_blur"`
	IsActive       bool           `gorm:"default:true" json:"is_active"`
	DisplayOrder   int            `gorm:"default:0" json:"display_order"`
	CreatedBy      *uuid.UUID     `gorm:"type:uuid" json:"created_by,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	// Relations
}

func (ThemePreset) TableName() string {
	return "theme_presets"
}

// LandingAppearanceConfig represents the background configuration for landing pages
type LandingAppearanceConfig struct {
	BackgroundType      string  `json:"background_type"`       // "none" | "preset" | "custom"
	PresetID            *string `json:"preset_id,omitempty"`   // UUID of selected preset
	CustomBackgroundURL *string `json:"custom_background_url,omitempty"`
	OverlayColor        string  `json:"overlay_color"`
	OverlayOpacity      int     `json:"overlay_opacity"` // 0-100
	CardOpacity         int     `json:"card_opacity"`    // 50-100
	CardBlur            int     `json:"card_blur"`       // 0-20
}

// DefaultLandingAppearanceConfig returns the default configuration
func DefaultLandingAppearanceConfig() LandingAppearanceConfig {
	return LandingAppearanceConfig{
		BackgroundType: "none",
		OverlayColor:   "#000000",
		OverlayOpacity: 30,
		CardOpacity:    90,
		CardBlur:       0,
	}
}
