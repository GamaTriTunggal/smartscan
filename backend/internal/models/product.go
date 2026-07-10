package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// ProductStatus enum
type ProductStatus string

const (
	ProductStatusActive   ProductStatus = "active"
	ProductStatusInactive ProductStatus = "inactive"
)

// Product represents products registered by tenants
type Product struct {
	ID          uuid.UUID      `gorm:"type:uuid;primary_key;default:uuidv7()" json:"id"`
	TenantID    uuid.UUID      `gorm:"type:uuid;not null" json:"tenant_id"`
	ProductName string         `gorm:"type:varchar(255);not null" json:"product_name"`
	ProductCode string         `gorm:"type:varchar(100)" json:"product_code"`
	Description string         `gorm:"type:text" json:"description"`
	Status      ProductStatus  `gorm:"type:varchar(20);default:'active'" json:"status"`
	// Display config for validation page (which fields to show)
	DisplayConfig datatypes.JSON `gorm:"type:jsonb;default:'{\"product_name\": true, \"product_code\": false, \"batch_code\": false, \"production_date\": false, \"expiry_date\": false, \"brand_name\": true, \"show_verification_count\": true}'" json:"display_config"`
	// Warranty form configuration (optional fields and custom fields)
	WarrantyFieldsConfig datatypes.JSON `gorm:"type:jsonb;default:'{\"optional_fields\": {\"invoice_number\": false, \"purchase_receipt\": false}, \"custom_fields\": []}'" json:"warranty_fields_config"`
	// Landing page appearance configuration (background image, overlay, blur)
	LandingAppearanceConfig datatypes.JSON `gorm:"type:jsonb;default:'{\"background_type\": \"none\", \"preset_id\": null, \"custom_background_url\": null, \"overlay_color\": \"#000000\", \"overlay_opacity\": 30, \"card_opacity\": 90, \"card_blur\": 0}'" json:"landing_appearance_config"`
	// Product-level template customization overrides (merged on top of template config at render time)
	TemplateOverrides datatypes.JSON `gorm:"type:jsonb" json:"template_overrides,omitempty"`
	// Warranty template customization overrides (merged on top of warranty template config at render time)
	WarrantyTemplateOverrides datatypes.JSON `gorm:"type:jsonb" json:"warranty_template_overrides,omitempty"`
	// Default templates for this product (nullable - uses tenant default if null)
	DefaultValidationTemplateID *uuid.UUID `gorm:"type:uuid" json:"default_validation_template_id,omitempty"`
	DefaultWarrantyTemplateID   *uuid.UUID `gorm:"type:uuid" json:"default_warranty_template_id,omitempty"`
	// Warranty enabled (product-level setting)
	WarrantyEnabled bool `gorm:"default:false" json:"warranty_enabled"`
	// Warranty configuration
	WarrantyMonths              int        `gorm:"default:12" json:"warranty_months"`
	MaxWarrantyRegistrationDays *int       `gorm:"default:null" json:"max_warranty_registration_days,omitempty"`
	// Website link for landing page
	WebsiteURL     string `gorm:"type:varchar(500)" json:"website_url,omitempty"`
	WebsiteCaption string `gorm:"type:varchar(100)" json:"website_caption,omitempty"`
	// Video embeds for landing page
	Videos datatypes.JSON `gorm:"type:jsonb;default:'[]'" json:"videos"`
	// Product-level counterfeit threshold override (NULL = use tenant global setting)
	CounterfeitScanMax *int `gorm:"default:null" json:"counterfeit_scan_max,omitempty"`
	CreatedBy          *uuid.UUID `gorm:"type:uuid" json:"created_by,omitempty"`
	CreatedAt                   time.Time  `json:"created_at"`
	UpdatedAt                   time.Time  `json:"updated_at"`
	DeletedAt                   gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	// Relations
	Tenant                    *Tenant       `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
	CreatedByStaff            *TenantStaff  `gorm:"foreignKey:CreatedBy" json:"created_by_staff,omitempty"`
	DefaultValidationTemplate *PageTemplate `gorm:"foreignKey:DefaultValidationTemplateID" json:"default_validation_template,omitempty"`
	DefaultWarrantyTemplate   *PageTemplate `gorm:"foreignKey:DefaultWarrantyTemplateID" json:"default_warranty_template,omitempty"`
	QRBatches                 []QRBatch     `gorm:"foreignKey:ProductID" json:"qr_batches,omitempty"`
	// Gallery images and social account links
	Images             []ProductImage             `gorm:"foreignKey:ProductID" json:"images,omitempty"`
	SocialAccountLinks []ProductSocialAccountLink `gorm:"foreignKey:ProductID" json:"social_account_links,omitempty"`
}

func (Product) TableName() string {
	return "products"
}

// TemplateType enum
type TemplateType string

const (
	TemplateTypeValidation TemplateType = "validation"
	TemplateTypeWarranty   TemplateType = "warranty"
)

// PageTemplate represents custom page templates (validation, warranty)
type PageTemplate struct {
	ID               uuid.UUID      `gorm:"type:uuid;primary_key;default:uuidv7()" json:"id"`
	TenantID         uuid.UUID      `gorm:"type:uuid;not null" json:"tenant_id"`
	TemplateType     TemplateType   `gorm:"type:varchar(50);not null" json:"template_type"`
	TemplateName     string         `gorm:"type:varchar(255);not null" json:"template_name"`
	HTMLContent      string         `gorm:"type:text;not null" json:"html_content"`
	CSSContent       string         `gorm:"type:text" json:"css_content"`
	JSContent        string         `gorm:"type:text" json:"js_content"`
	CustomFields     datatypes.JSON `gorm:"type:jsonb" json:"custom_fields"`
	BackgroundConfig datatypes.JSON `gorm:"type:jsonb" json:"background_config"`
	IsActive         bool           `gorm:"default:true" json:"is_active"`
	CreatedBy        *uuid.UUID     `gorm:"type:uuid" json:"created_by,omitempty"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`

	// Relations
	Tenant         *Tenant      `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
	CreatedByStaff *TenantStaff `gorm:"foreignKey:CreatedBy" json:"created_by_staff,omitempty"`
}

// TemplateBackgroundConfig represents background configuration for templates
type TemplateBackgroundConfig struct {
	BackgroundType      string  `json:"background_type"`       // "none", "preset", "custom"
	PresetID            *string `json:"preset_id"`             // Theme preset ID
	CustomBackgroundURL *string `json:"custom_background_url"` // Custom background URL
	OverlayColor        string  `json:"overlay_color"`         // Overlay color
	OverlayOpacity      int     `json:"overlay_opacity"`       // 0-100
	CardOpacity         int     `json:"card_opacity"`          // 50-100
	CardBlur            int     `json:"card_blur"`             // 0-20px
}

func (PageTemplate) TableName() string {
	return "page_templates"
}

// QRBatchStatus enum - tracks async QR generation lifecycle
type QRBatchStatus string

const (
	QRBatchStatusPendingQueue QRBatchStatus = "pending_queue" // Created, waiting to enqueue (Redis down fallback)
	QRBatchStatusQueued       QRBatchStatus = "queued"        // Enqueued to Redis, waiting for worker
	QRBatchStatusProcessing   QRBatchStatus = "processing"    // Worker is generating QR codes
	QRBatchStatusCompleted    QRBatchStatus = "completed"     // All QR codes generated successfully
	QRBatchStatusFailed       QRBatchStatus = "failed"        // Generation failed after max retries
)

// IsTerminal returns true if the status cannot change further without user action
func (s QRBatchStatus) IsTerminal() bool {
	return s == QRBatchStatusCompleted || s == QRBatchStatusFailed
}

// IsInProgress returns true if the batch is currently being processed or waiting to process
func (s QRBatchStatus) IsInProgress() bool {
	return s == QRBatchStatusPendingQueue || s == QRBatchStatusQueued || s == QRBatchStatusProcessing
}

// QRBatch represents QR generation batches for products
type QRBatch struct {
	ID                   uuid.UUID  `gorm:"type:uuid;primary_key;default:uuidv7()" json:"id"`
	TenantID             uuid.UUID  `gorm:"type:uuid;not null" json:"tenant_id"`
	ProductID            uuid.UUID  `gorm:"type:uuid;not null" json:"product_id"`
	BatchName            string     `gorm:"type:varchar(255);not null" json:"batch_name"`
	BatchCode            string     `gorm:"type:varchar(100)" json:"batch_code"`
	QRCount              int        `gorm:"not null" json:"qr_count"`
	Status               QRBatchStatus `gorm:"type:varchar(20);default:'completed'" json:"status"`
	Prefix               string     `gorm:"type:varchar(50)" json:"prefix"`
	Suffix               string     `gorm:"type:varchar(50)" json:"suffix"`
	ProductionDate       *time.Time `gorm:"type:date" json:"production_date,omitempty"`
	ExpiryDate           *time.Time `gorm:"type:date" json:"expiry_date,omitempty"`
	LogoURL        string `gorm:"type:text" json:"logo_url"`
	CSVFileURL     string `gorm:"type:text" json:"csv_file_url"`
	NeedValidation bool   `gorm:"default:false" json:"need_validation"`
	ValidationTemplateID *uuid.UUID `gorm:"type:uuid" json:"validation_template_id,omitempty"`
	// WarrantyEnabled is now at product level - template override kept here
	WarrantyTemplateID   *uuid.UUID `gorm:"type:uuid" json:"warranty_template_id,omitempty"`
	// Geofence: distribution zone (optional)
	GeofenceEnabled   bool     `gorm:"default:false" json:"geofence_enabled"`
	GeofenceLatitude  *float64 `gorm:"type:decimal(10,7)" json:"geofence_latitude,omitempty"`
	GeofenceLongitude *float64 `gorm:"type:decimal(10,7)" json:"geofence_longitude,omitempty"`
	GeofenceRadiusKm  *float64 `gorm:"type:decimal(6,1)" json:"geofence_radius_km,omitempty"`
	GeofenceLabel     string   `gorm:"type:varchar(255)" json:"geofence_label,omitempty"`
	// Batch-level counterfeit threshold override (NULL = inherit from product/tenant)
	CounterfeitScanMax *int    `gorm:"default:null" json:"counterfeit_scan_max,omitempty"`

	CreatedBy            *uuid.UUID     `gorm:"type:uuid" json:"created_by,omitempty"`
	CreatedAt            time.Time      `json:"created_at"`
	UpdatedAt            time.Time      `json:"updated_at"`
	DeletedAt            *time.Time     `gorm:"index" json:"deleted_at,omitempty"`

	// Relations
	Tenant             *Tenant       `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
	Product            *Product      `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	ValidationTemplate *PageTemplate `gorm:"foreignKey:ValidationTemplateID" json:"validation_template,omitempty"`
	WarrantyTemplate   *PageTemplate `gorm:"foreignKey:WarrantyTemplateID" json:"warranty_template,omitempty"`
	CreatedByStaff     *TenantStaff  `gorm:"foreignKey:CreatedBy" json:"created_by_staff,omitempty"`
	QRCodes            []QRCode      `gorm:"foreignKey:BatchID" json:"qr_codes,omitempty"`
}

func (QRBatch) TableName() string {
	return "qr_batches"
}

// QRCodeStatus enum — admin control plane (enable/disable).
type QRCodeStatus string

const (
	QRCodeStatusActive   QRCodeStatus = "active"
	QRCodeStatusInactive QRCodeStatus = "inactive"
)

// CounterfeitStatus enum — fraud detection signal.
type CounterfeitStatus string

const (
	CounterfeitStatusValid       CounterfeitStatus = "valid"
	CounterfeitStatusWarning     CounterfeitStatus = "warning" // Deprecated: treated as valid. Kept for backward compat with existing DB records.
	CounterfeitStatusCounterfeit CounterfeitStatus = "counterfeit"
)

// QRCode represents individual QR codes
type QRCode struct {
	ID                uuid.UUID         `gorm:"type:uuid;primary_key;default:uuidv7()" json:"id"`
	BatchID           uuid.UUID         `gorm:"type:uuid;not null" json:"batch_id"`
	QRUUID            uuid.UUID         `gorm:"type:uuid;uniqueIndex;not null;default:uuidv7()" json:"qr_uuid"`
	QRCode            string            `gorm:"type:varchar(255);uniqueIndex;not null" json:"qr_code"`
	QRImageURL        string            `gorm:"type:text" json:"qr_image_url"`
	Status            QRCodeStatus      `gorm:"type:varchar(20);default:'active'" json:"status"`
	CounterfeitStatus CounterfeitStatus `gorm:"type:varchar(20);default:'valid'" json:"counterfeit_status"`
	IsCompressed      bool              `gorm:"default:false" json:"is_compressed"`
	CompressedData    []byte            `gorm:"type:bytea" json:"-"`
	CompressedAt      *time.Time        `json:"compressed_at,omitempty"`
	// QR-level counterfeit threshold override (NULL = inherit from batch/product/tenant)
	CounterfeitScanMax *int             `gorm:"default:null" json:"counterfeit_scan_max,omitempty"`

	CreatedAt          time.Time        `json:"created_at"`

	// Relations
	Batch *QRBatch `gorm:"foreignKey:BatchID" json:"batch,omitempty"`
}

func (QRCode) TableName() string {
	return "qr_codes"
}

// IsScannable returns true when this QR code should accept consumer scans.
// Combines the admin status and the counterfeit-detection status.
//
// NOTE: This helper does NOT check whether the parent QRBatch is soft-deleted.
// Handlers must continue to verify Batch.DeletedAt IS NULL separately (existing
// scan flow already does this).
func (q *QRCode) IsScannable() bool {
	if q.Status != QRCodeStatusActive {
		return false
	}
	// Note: CounterfeitStatusWarning is intentionally allowed (deprecated, treated as valid).
	return q.CounterfeitStatus != CounterfeitStatusCounterfeit
}

// QRGenerationQueueStatus enum
type QRGenerationQueueStatus string

const (
	QRGenerationQueueStatusQueued     QRGenerationQueueStatus = "queued"
	QRGenerationQueueStatusProcessing QRGenerationQueueStatus = "processing"
	QRGenerationQueueStatusCompleted  QRGenerationQueueStatus = "completed"
	QRGenerationQueueStatusFailed     QRGenerationQueueStatus = "failed"
)

// QRGenerationQueue represents background job queue for QR generation
type QRGenerationQueue struct {
	ID             uuid.UUID               `gorm:"type:uuid;primary_key;default:uuidv7()" json:"id"`
	BatchID        uuid.UUID               `gorm:"type:uuid;uniqueIndex;not null" json:"batch_id"`
	TotalQRCount   int                     `gorm:"not null" json:"total_qr_count"`
	GeneratedCount int                     `gorm:"default:0" json:"generated_count"`
	Status         QRGenerationQueueStatus `gorm:"type:varchar(20);default:'queued'" json:"status"`
	WorkerID       string                  `gorm:"type:varchar(100)" json:"worker_id"`
	ErrorMessage   string                  `gorm:"type:text" json:"error_message"`
	CreatedAt      time.Time               `json:"created_at"`
	StartedAt      *time.Time              `json:"started_at,omitempty"`
	CompletedAt    *time.Time              `json:"completed_at,omitempty"`

	// Relations
	Batch *QRBatch `gorm:"foreignKey:BatchID" json:"batch,omitempty"`
}

func (QRGenerationQueue) TableName() string {
	return "qr_generation_queue"
}
