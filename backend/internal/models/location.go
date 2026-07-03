package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// Country represents countries table
type Country struct {
	Code      string     `gorm:"type:varchar(2);primary_key" json:"code"`
	Name      string     `gorm:"type:varchar(100);not null" json:"name"`
	PhoneCode string     `gorm:"type:varchar(5)" json:"phone_code"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `gorm:"index" json:"deleted_at,omitempty"`
}

func (Country) TableName() string {
	return "countries"
}

// Province represents provinces table
type Province struct {
	ID          int        `gorm:"primary_key" json:"id"`
	CountryCode string     `gorm:"type:varchar(2);not null" json:"country_code"`
	Name        string     `gorm:"type:varchar(100);not null" json:"name"`
	Code        string     `gorm:"type:varchar(10)" json:"code"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   *time.Time `gorm:"index" json:"deleted_at,omitempty"`

	Country *Country `gorm:"foreignKey:CountryCode" json:"country,omitempty"`
}

func (Province) TableName() string {
	return "provinces"
}

// City represents cities table
type City struct {
	ID               int        `gorm:"primary_key" json:"id"`
	ProvinceID       int        `gorm:"not null" json:"province_id"`
	CountryCode      string     `gorm:"type:varchar(2);not null" json:"country_code"`
	Name             string     `gorm:"type:varchar(100);not null" json:"name"`
	PostalCodePrefix string     `gorm:"type:varchar(10)" json:"postal_code_prefix"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	DeletedAt        *time.Time `gorm:"index" json:"deleted_at,omitempty"`

	Province *Province `gorm:"foreignKey:ProvinceID" json:"province,omitempty"`
	Country  *Country  `gorm:"foreignKey:CountryCode" json:"country,omitempty"`
}

func (City) TableName() string {
	return "cities"
}

// LocationType enum
type LocationType string

const (
	LocationTypeWarehouse  LocationType = "warehouse"
	LocationTypeQCArea     LocationType = "qc_area"
	LocationTypeProduction LocationType = "production"
	LocationTypeOffice     LocationType = "office"
)

// TenantLocation represents tenant locations (warehouses, QC areas, etc.)
type TenantLocation struct {
	ID            uuid.UUID      `gorm:"type:uuid;primary_key;default:uuidv7()" json:"id"`
	TenantID      uuid.UUID      `gorm:"type:uuid;not null" json:"tenant_id"`
	LocationName  string         `gorm:"type:varchar(255);not null" json:"location_name"`
	LocationType  LocationType   `gorm:"type:varchar(50);not null;default:'warehouse'" json:"location_type"`
	Address       string         `gorm:"type:text" json:"address"`
	City          string         `gorm:"type:varchar(100)" json:"city"`
	Province      string         `gorm:"type:varchar(100)" json:"province"`
	PostalCode    string         `gorm:"type:varchar(10)" json:"postal_code"`
	PhoneNumber   string         `gorm:"type:varchar(20)" json:"phone_number"`
	Geolocation   datatypes.JSON `gorm:"type:jsonb" json:"geolocation"`
	AllowedRadius *int           `gorm:"type:integer" json:"allowed_radius,omitempty"` // in meters, NULL = no limit
	Status        string         `gorm:"type:varchar(20);default:'active'" json:"status"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     *time.Time     `gorm:"index" json:"deleted_at,omitempty"`

	// Relations
	Tenant *Tenant `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
}

func (TenantLocation) TableName() string {
	return "tenant_locations"
}

// MovementType enum
type MovementType string

const (
	MovementTypeIn  MovementType = "in"
	MovementTypeOut MovementType = "out"
)

// InventoryMovement represents product movement tracking
type InventoryMovement struct {
	ID              uuid.UUID      `gorm:"type:uuid;primary_key;default:uuidv7()" json:"id"`
	LocationID      uuid.UUID      `gorm:"type:uuid;not null" json:"location_id"`
	QRCodeID        uuid.UUID      `gorm:"type:uuid;not null" json:"qr_code_id"`
	MovementType    MovementType   `gorm:"type:varchar(20)" json:"movement_type"`
	ScannedBy       *uuid.UUID     `gorm:"type:uuid" json:"scanned_by,omitempty"`
	ScanGeolocation datatypes.JSON `gorm:"type:jsonb" json:"scan_geolocation,omitempty"`
	ScannedAt       time.Time      `gorm:"default:CURRENT_TIMESTAMP" json:"scanned_at"`

	// Relations
	Location       *TenantLocation `gorm:"foreignKey:LocationID" json:"location,omitempty"`
	QRCode         *QRCode         `gorm:"foreignKey:QRCodeID" json:"qr_code,omitempty"`
	ScannedByStaff *TenantStaff    `gorm:"foreignKey:ScannedBy" json:"scanned_by_staff,omitempty"`
}

func (InventoryMovement) TableName() string {
	return "inventory_movements"
}

// QCStatus enum
type QCStatus string

const (
	QCStatusPass   QCStatus = "pass"
	QCStatusFailed QCStatus = "failed"
)

// QCScan represents quality control scan records
type QCScan struct {
	ID               uuid.UUID      `gorm:"type:uuid;primary_key;default:uuidv7()" json:"id"`
	LocationID       *uuid.UUID     `gorm:"type:uuid" json:"location_id,omitempty"`
	QRCodeID         uuid.UUID      `gorm:"type:uuid;not null" json:"qr_code_id"`
	QCStatus         QCStatus       `gorm:"type:varchar(20)" json:"qc_status"`
	ScannedBy        *uuid.UUID     `gorm:"type:uuid" json:"scanned_by,omitempty"`
	ScanGeolocation  datatypes.JSON `gorm:"type:jsonb" json:"scan_geolocation,omitempty"`
	IsCorrection     bool           `gorm:"default:false" json:"is_correction"`
	CorrectsScanID   *uuid.UUID     `gorm:"type:uuid" json:"corrects_scan_id,omitempty"`
	CorrectionReason string         `gorm:"type:text" json:"correction_reason,omitempty"`
	ScannedAt        time.Time      `gorm:"default:CURRENT_TIMESTAMP" json:"scanned_at"`

	// Relations
	Location       *TenantLocation `gorm:"foreignKey:LocationID" json:"location,omitempty"`
	QRCode         *QRCode         `gorm:"foreignKey:QRCodeID" json:"qr_code,omitempty"`
	ScannedByStaff *TenantStaff    `gorm:"foreignKey:ScannedBy" json:"scanned_by_staff,omitempty"`
	CorrectsScan   *QCScan         `gorm:"foreignKey:CorrectsScanID" json:"corrects_scan,omitempty"`
}

func (QCScan) TableName() string {
	return "qc_scans"
}
