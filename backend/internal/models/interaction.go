package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// InteractionCategory enum
type InteractionCategory string

const (
	InteractionCategoryTenantAccess  InteractionCategory = "tenant_access"
	InteractionCategoryEndUserAccess InteractionCategory = "end_user_access"
)

// InteractionSubcategory enum
type InteractionSubcategory string

const (
	InteractionSubcategoryQCScan            InteractionSubcategory = "qc_scan"
	InteractionSubcategoryWarehouseScan     InteractionSubcategory = "warehouse_scan"
	InteractionSubcategoryProductValidation InteractionSubcategory = "product_validation"
	InteractionSubcategoryWarrantyActivation InteractionSubcategory = "warranty_activation"
)

// InteractionStatus enum
type InteractionStatus string

const (
	InteractionStatusSuccess InteractionStatus = "success"
	InteractionStatusFailed  InteractionStatus = "failed"
)

// Interaction represents all QR code interactions
type Interaction struct {
	ID                     uuid.UUID              `gorm:"type:uuid;primary_key;default:uuidv7()" json:"id"`
	QRCodeID               *uuid.UUID             `gorm:"type:uuid" json:"qr_code_id,omitempty"`
	TenantID               uuid.UUID              `gorm:"type:uuid;not null" json:"tenant_id"`
	InteractionCategory    InteractionCategory    `gorm:"type:varchar(50)" json:"interaction_category"`
	InteractionSubcategory InteractionSubcategory `gorm:"type:varchar(50)" json:"interaction_subcategory"`
	InteractionStatus      InteractionStatus      `gorm:"type:varchar(20)" json:"interaction_status"`
	ScannedBy              *uuid.UUID             `gorm:"type:uuid" json:"scanned_by,omitempty"`
	IPAddress              string                 `gorm:"type:inet" json:"ip_address"`
	UserAgent              string                 `gorm:"type:text" json:"user_agent"`
	Geolocation            datatypes.JSON         `gorm:"type:jsonb" json:"geolocation"`
	AdditionalData         datatypes.JSON         `gorm:"type:jsonb" json:"additional_data"`
	ValidationTemplateID   *uuid.UUID             `gorm:"type:uuid" json:"validation_template_id,omitempty"`
	CreatedAt              time.Time              `json:"created_at"`

	// Relations
	QRCode             *QRCode       `gorm:"foreignKey:QRCodeID" json:"qr_code,omitempty"`
	Tenant             *Tenant       `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
	ScannedByUser      *User         `gorm:"foreignKey:ScannedBy" json:"scanned_by_user,omitempty"`
	ValidationTemplate *PageTemplate `gorm:"foreignKey:ValidationTemplateID" json:"validation_template,omitempty"`
}

func (Interaction) TableName() string {
	return "interactions"
}

// CounterfeitDetectionStatus enum
type CounterfeitDetectionStatus string

const (
	CounterfeitDetectionStatusActive        CounterfeitDetectionStatus = "active"
	CounterfeitDetectionStatusResolved      CounterfeitDetectionStatus = "resolved"
	CounterfeitDetectionStatusFalsePositive CounterfeitDetectionStatus = "false_positive"
)

// CounterfeitDetection represents detected counterfeit products
type CounterfeitDetection struct {
	ID                    uuid.UUID                  `gorm:"type:uuid;primary_key;default:uuidv7()" json:"id"`
	QRCodeID              uuid.UUID                  `gorm:"type:uuid;not null" json:"qr_code_id"`
	TenantID              uuid.UUID                  `gorm:"type:uuid;not null" json:"tenant_id"`
	DetectionReason       string                     `gorm:"type:text" json:"detection_reason"`
	InteractionIDs        datatypes.JSON             `gorm:"type:jsonb" json:"interaction_ids"`
	TotalInteractionsCount int                       `json:"total_interactions_count"`
	FirstInteractionAt    *time.Time                 `json:"first_interaction_at,omitempty"`
	LastInteractionAt     *time.Time                 `json:"last_interaction_at,omitempty"`
	Status                CounterfeitDetectionStatus `gorm:"type:varchar(20);default:'active'" json:"status"`
	ResolvedBy            *uuid.UUID                 `gorm:"type:uuid" json:"resolved_by,omitempty"`
	ResolvedAt            *time.Time                 `json:"resolved_at,omitempty"`
	CreatedAt             time.Time                  `json:"created_at"`

	// Relations
	QRCode         *QRCode      `gorm:"foreignKey:QRCodeID" json:"qr_code,omitempty"`
	Tenant         *Tenant      `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
	ResolvedByStaff *TenantStaff `gorm:"foreignKey:ResolvedBy" json:"resolved_by_staff,omitempty"`
}

func (CounterfeitDetection) TableName() string {
	return "counterfeit_detections"
}

// AfterCreate syncs QR code status when new active detection is created
func (d *CounterfeitDetection) AfterCreate(tx *gorm.DB) error {
	if d.Status == CounterfeitDetectionStatusActive {
		return tx.Model(&QRCode{}).Where("id = ?", d.QRCodeID).
			Update("counterfeit_status", CounterfeitStatusCounterfeit).Error
	}
	return nil
}

// NOTE: AfterUpdate hook removed — QR code counterfeit_status sync is now
// inlined directly in counterfeit.go handlers (ResolveCounterfeitDetection,
// MarkAsFalsePositive) because Updates(map) does not trigger GORM hooks.

// CounterfeitReport represents end-user reports of counterfeit products
type CounterfeitReport struct {
	ID                     uuid.UUID      `gorm:"type:uuid;primary_key;default:uuidv7()" json:"id"`
	QRCodeID               uuid.UUID      `gorm:"type:uuid;not null" json:"qr_code_id"`
	TenantID               uuid.UUID      `gorm:"type:uuid;not null" json:"tenant_id"`
	CounterfeitDetectionID *uuid.UUID     `gorm:"type:uuid" json:"counterfeit_detection_id,omitempty"`
	Description            string         `gorm:"type:text" json:"description"`
	Photos                 datatypes.JSON `gorm:"type:jsonb" json:"photos"`
	StoreName              string         `gorm:"type:varchar(255)" json:"store_name"`
	Province               string         `gorm:"type:varchar(100)" json:"province"`
	City                   string         `gorm:"type:varchar(100)" json:"city"`
	IPAddress              string         `gorm:"type:inet" json:"ip_address"`
	UserAgent              string         `gorm:"type:text" json:"user_agent"`
	Geolocation            datatypes.JSON `gorm:"type:jsonb" json:"geolocation"`
	CreatedAt              time.Time      `json:"created_at"`

	// Relations
	QRCode               *QRCode               `gorm:"foreignKey:QRCodeID" json:"qr_code,omitempty"`
	Tenant               *Tenant               `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
	CounterfeitDetection *CounterfeitDetection `gorm:"foreignKey:CounterfeitDetectionID" json:"counterfeit_detection,omitempty"`
}

func (CounterfeitReport) TableName() string {
	return "counterfeit_reports"
}
