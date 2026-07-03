package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// WarrantyActivation represents product warranty activations by end-users
type WarrantyActivation struct {
	ID                 uuid.UUID      `gorm:"type:uuid;primary_key;default:uuidv7()" json:"id"`
	QRCodeID           uuid.UUID      `gorm:"type:uuid;not null" json:"qr_code_id"`
	CustomerName       string         `gorm:"type:varchar(255)" json:"customer_name"`
	CustomerEmail      string         `gorm:"type:varchar(255)" json:"customer_email"`
	CustomerPhone      string         `gorm:"type:varchar(20)" json:"customer_phone"`
	PurchaseDate       *time.Time     `gorm:"type:date" json:"purchase_date,omitempty"`
	PurchaseStore      string         `gorm:"type:varchar(255)" json:"purchase_store"`
	// Customer address fields (for warranty service and delivery)
	Address            string         `gorm:"type:text" json:"address"`
	CountryCode        *string        `gorm:"type:varchar(2)" json:"country_code,omitempty"`
	ProvinceID         *int           `json:"province_id,omitempty"`
	CityID             *int           `json:"city_id,omitempty"`
	// Other fields
	ActivationData     datatypes.JSON `gorm:"type:jsonb" json:"activation_data"`
	ActivatedAt        time.Time      `gorm:"default:CURRENT_TIMESTAMP" json:"activated_at"`
	WarrantyExpiryDate *time.Time     `gorm:"type:date" json:"warranty_expiry_date,omitempty"`
	IPAddress          string         `gorm:"type:inet" json:"ip_address"`
	Geolocation            datatypes.JSON `gorm:"type:jsonb" json:"geolocation"`
	ExpiryReminderSentAt   *time.Time     `json:"expiry_reminder_sent_at,omitempty"`

	// Anti-counterfeit signal: registration attempts on an already-activated QR.
	// A cloned label surfaces here — the legitimate owner registered first, and the
	// clone's owner then hits the duplicate path.
	DuplicateAttemptCount  int        `gorm:"not null;default:0" json:"duplicate_attempt_count"`
	LastDuplicateAttemptAt *time.Time `json:"last_duplicate_attempt_at,omitempty"`

	// Relations
	QRCode   *QRCode   `gorm:"foreignKey:QRCodeID" json:"qr_code,omitempty"`
	Country  *Country  `gorm:"foreignKey:CountryCode;references:Code" json:"country,omitempty"`
	Province *Province `gorm:"foreignKey:ProvinceID" json:"province,omitempty"`
	City     *City     `gorm:"foreignKey:CityID" json:"city,omitempty"`
}

func (WarrantyActivation) TableName() string {
	return "warranty_activations"
}

