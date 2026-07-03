package models

import (
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)


// Tenant represents the tenants table (companies subscribing to the SaaS)
type Tenant struct {
	ID                 uuid.UUID          `gorm:"type:uuid;primary_key;default:uuidv7()" json:"id"`
	CompanyName        string             `gorm:"type:varchar(255);not null" json:"company_name"`
	CompanyAddress     string             `gorm:"type:text" json:"company_address"`
	Country            string             `gorm:"type:varchar(100)" json:"country"`
	Province           string             `gorm:"type:varchar(100)" json:"province"`
	City               string             `gorm:"type:varchar(100)" json:"city"`
	CountryCode        *string            `gorm:"type:varchar(2)" json:"country_code,omitempty"`
	ProvinceID         *int               `gorm:"type:integer" json:"province_id,omitempty"`
	CityID             *int               `gorm:"type:integer" json:"city_id,omitempty"`
	PostalCode         string             `gorm:"type:varchar(10)" json:"postal_code"`
	BusinessField      string             `gorm:"type:varchar(100)" json:"business_field"`
	PhoneNumber        string             `gorm:"type:varchar(20)" json:"phone_number"`
	CompanyEmail       string             `gorm:"type:varchar(255)" json:"company_email"`
	// Default templates (explicitly set, falls back to oldest if NULL)
	DefaultValidationTemplateID *uuid.UUID `gorm:"type:uuid" json:"default_validation_template_id,omitempty"`
	DefaultWarrantyTemplateID   *uuid.UUID `gorm:"type:uuid" json:"default_warranty_template_id,omitempty"`
	Slug               string             `gorm:"type:varchar(100);uniqueIndex;not null" json:"slug"`
	CreatedAt          time.Time          `json:"created_at"`
	UpdatedAt          time.Time          `json:"updated_at"`
	DeletedAt          gorm.DeletedAt     `gorm:"index" json:"deleted_at,omitempty"`

	// Relations
	CountryRef   *Country        `gorm:"foreignKey:CountryCode" json:"country_ref,omitempty"`
	ProvinceRef  *Province       `gorm:"foreignKey:ProvinceID" json:"province_ref,omitempty"`
	CityRef      *City           `gorm:"foreignKey:CityID" json:"city_ref,omitempty"`
	Staff        []TenantStaff   `gorm:"foreignKey:TenantID" json:"staff,omitempty"`
	Products     []Product       `gorm:"foreignKey:TenantID" json:"products,omitempty"`
	Locations    []TenantLocation `gorm:"foreignKey:TenantID" json:"locations,omitempty"`
	DefaultValidationTemplate *PageTemplate `gorm:"foreignKey:DefaultValidationTemplateID" json:"default_validation_template,omitempty"`
	DefaultWarrantyTemplate   *PageTemplate `gorm:"foreignKey:DefaultWarrantyTemplateID" json:"default_warranty_template,omitempty"`
}

// BeforeCreate auto-generates a unique slug from CompanyName if not explicitly set.
// In production, slug is always set by the registration handler. This hook is a
// safety net that also fixes test helpers which create tenants without slugs.
func (t *Tenant) BeforeCreate(tx *gorm.DB) error {
	if t.Slug == "" {
		base := strings.ToLower(t.CompanyName)
		base = strings.ReplaceAll(base, " ", "-")
		// Strip non-alphanumeric characters (keep hyphens)
		reg := regexp.MustCompile(`[^a-z0-9-]+`)
		base = reg.ReplaceAllString(base, "")
		// Append random suffix for uniqueness
		t.Slug = base + "-" + uuid.New().String()[:8]
	}
	return nil
}

func (Tenant) TableName() string {
	return "tenants"
}
