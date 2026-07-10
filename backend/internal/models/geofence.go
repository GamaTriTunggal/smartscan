package models

import (
	"time"

	"github.com/google/uuid"
)

// GeofenceViolation records a consumer scan that occurred outside the designated distribution zone.
type GeofenceViolation struct {
	ID                   uuid.UUID  `gorm:"type:uuid;primary_key;default:uuidv7()" json:"id"`
	TenantID             uuid.UUID  `gorm:"type:uuid;not null" json:"tenant_id"`
	BatchID              uuid.UUID  `gorm:"type:uuid;not null" json:"batch_id"`
	QRCodeID             *uuid.UUID `gorm:"type:uuid" json:"qr_code_id,omitempty"`
	ProductID            *uuid.UUID `gorm:"type:uuid" json:"product_id,omitempty"`
	InteractionID        *uuid.UUID `gorm:"type:uuid" json:"interaction_id,omitempty"`
	ScanLatitude         float64    `gorm:"type:decimal(10,7);not null" json:"scan_latitude"`
	ScanLongitude        float64    `gorm:"type:decimal(10,7);not null" json:"scan_longitude"`
	DistanceFromCenterKm float64    `gorm:"type:decimal(8,2);not null" json:"distance_from_center_km"`
	DistanceFromEdgeKm   float64    `gorm:"type:decimal(8,2);not null" json:"distance_from_edge_km"`
	GPSAccuracyMeters    *float64   `gorm:"type:decimal(8,2)" json:"gps_accuracy_meters,omitempty"`
	Severity             string     `gorm:"type:varchar(20);not null" json:"severity"`
	CreatedAt            time.Time  `json:"created_at"`

	// Relations
	Tenant      *Tenant      `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
	Batch       *QRBatch     `gorm:"foreignKey:BatchID" json:"batch,omitempty"`
	QRCode      *QRCode      `gorm:"foreignKey:QRCodeID" json:"qr_code,omitempty"`
	Product     *Product     `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	Interaction *Interaction `gorm:"foreignKey:InteractionID" json:"interaction,omitempty"`
}

func (GeofenceViolation) TableName() string {
	return "geofence_violations"
}

// GeofenceZoneTemplate is a saved geofence zone for reuse in batch creation.
type GeofenceZoneTemplate struct {
	ID           uuid.UUID  `gorm:"type:uuid;primary_key;default:uuidv7()" json:"id"`
	TenantID     uuid.UUID  `gorm:"type:uuid;not null" json:"tenant_id"`
	TemplateName string     `gorm:"type:varchar(255);not null" json:"template_name"`
	Latitude     float64    `gorm:"type:decimal(10,7);not null" json:"latitude"`
	Longitude    float64    `gorm:"type:decimal(10,7);not null" json:"longitude"`
	RadiusKm     float64    `gorm:"type:decimal(6,1);not null" json:"radius_km"`
	Label        string     `gorm:"type:varchar(255)" json:"label,omitempty"`
	UsageCount   int        `gorm:"default:0" json:"usage_count"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	DeletedAt    *time.Time `gorm:"index" json:"deleted_at,omitempty"`

	// Relations
	Tenant *Tenant `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
}

func (GeofenceZoneTemplate) TableName() string {
	return "geofence_zone_templates"
}
