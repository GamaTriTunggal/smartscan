package models

import (
	"time"

	"github.com/google/uuid"
)

// TenantSocialAccount represents tenant-owned social media accounts
// These can be linked to multiple products (N:M relationship)
type TenantSocialAccount struct {
	ID            uuid.UUID  `gorm:"type:uuid;primary_key;default:uuidv7()" json:"id"`
	TenantID      uuid.UUID  `gorm:"type:uuid;not null" json:"tenant_id"`
	PlatformID    uuid.UUID  `gorm:"type:uuid;not null" json:"platform_id"`
	AccountHandle string     `gorm:"type:varchar(255);not null" json:"account_handle"`
	AccountURL    string     `gorm:"type:varchar(500)" json:"account_url,omitempty"`
	IsActive      bool       `gorm:"default:true" json:"is_active"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`

	// Relations
	Tenant   *Tenant              `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
	Platform *SocialMediaPlatform `gorm:"foreignKey:PlatformID" json:"platform,omitempty"`
}

func (TenantSocialAccount) TableName() string {
	return "tenant_social_accounts"
}

// ProductSocialAccountLink represents N:M link between products and tenant social accounts
type ProductSocialAccountLink struct {
	ID              uuid.UUID `gorm:"type:uuid;primary_key;default:uuidv7()" json:"id"`
	ProductID       uuid.UUID `gorm:"type:uuid;not null" json:"product_id"`
	SocialAccountID uuid.UUID `gorm:"type:uuid;not null" json:"social_account_id"`
	SortOrder       int       `gorm:"default:0" json:"sort_order"`
	CreatedAt       time.Time `json:"created_at"`

	// Relations
	Product       *Product             `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	SocialAccount *TenantSocialAccount `gorm:"foreignKey:SocialAccountID" json:"social_account,omitempty"`
}

func (ProductSocialAccountLink) TableName() string {
	return "product_social_account_links"
}
