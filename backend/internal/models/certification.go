package models

import (
	"time"

	"github.com/google/uuid"
)

// CertificationType represents master list of certification types
type CertificationType struct {
	ID           uuid.UUID  `gorm:"type:uuid;primary_key;default:uuidv7()" json:"id"`
	CountryCode  *string    `gorm:"type:varchar(2)" json:"country_code"` // NULL for international
	Code         string     `gorm:"type:varchar(50);unique;not null" json:"code"`
	Name         string     `gorm:"type:varchar(255);not null" json:"name"`
	Description  string     `gorm:"type:text" json:"description"`
	LogoURL      string     `gorm:"type:text" json:"logo_url"`
	WebsiteURL   string     `gorm:"type:text" json:"website_url"`
	IsActive     bool       `gorm:"default:true" json:"is_active"`
	DisplayOrder int        `gorm:"default:0" json:"display_order"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	DeletedAt    *time.Time `gorm:"index" json:"deleted_at,omitempty"`

	// Relations
	Country *Country `gorm:"foreignKey:CountryCode;references:Code" json:"country,omitempty"`
}

func (CertificationType) TableName() string {
	return "certification_types"
}

// ProductCertification represents product certification registrations
type ProductCertification struct {
	ID                  uuid.UUID `gorm:"type:uuid;primary_key;default:uuidv7()" json:"id"`
	ProductID           uuid.UUID `gorm:"type:uuid;not null" json:"product_id"`
	CertificationTypeID uuid.UUID `gorm:"type:uuid;not null" json:"certification_type_id"`
	RegistrationNumber  string    `gorm:"type:varchar(255);not null" json:"registration_number"`
	SortOrder           int       `gorm:"default:0" json:"sort_order"`
	CreatedAt           time.Time `json:"created_at"`

	// Relations
	Product           *Product           `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	CertificationType *CertificationType `gorm:"foreignKey:CertificationTypeID" json:"certification_type,omitempty"`
}

func (ProductCertification) TableName() string {
	return "product_certifications"
}


// SocialMediaPlatform represents master list of social media platforms
type SocialMediaPlatform struct {
	ID              uuid.UUID  `gorm:"type:uuid;primary_key;default:uuidv7()" json:"id"`
	Code            string     `gorm:"type:varchar(50);unique;not null" json:"code"`
	Name            string     `gorm:"type:varchar(100);not null" json:"name"`
	Icon            string     `gorm:"type:varchar(50)" json:"icon"`
	BaseURL         string     `gorm:"type:text" json:"base_url"`
	DeepLinkPattern string     `gorm:"type:text" json:"deep_link_pattern"`
	PlaceholderText string     `gorm:"type:varchar(255)" json:"placeholder_text"`
	ValidationType  string     `gorm:"type:varchar(20);default:text" json:"validation_type"` // phone, username, email, url, text
	IsActive        bool       `gorm:"default:true" json:"is_active"`
	DisplayOrder    int        `gorm:"default:0" json:"display_order"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	DeletedAt       *time.Time `gorm:"index" json:"deleted_at,omitempty"`
}

func (SocialMediaPlatform) TableName() string {
	return "social_media_platforms"
}

// ProductSocialLink represents product social media links
type ProductSocialLink struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;default:uuidv7()" json:"id"`
	ProductID   uuid.UUID `gorm:"type:uuid;not null" json:"product_id"`
	PlatformID  uuid.UUID `gorm:"type:uuid;not null" json:"platform_id"`
	HandleOrURL string    `gorm:"type:varchar(500);not null" json:"handle_or_url"`
	CreatedAt   time.Time `json:"created_at"`

	// Relations
	Product  *Product             `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	Platform *SocialMediaPlatform `gorm:"foreignKey:PlatformID" json:"platform,omitempty"`
}

func (ProductSocialLink) TableName() string {
	return "product_social_links"
}

