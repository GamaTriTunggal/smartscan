package models

import (
	"time"

	"github.com/google/uuid"
)

// ProductImage represents product gallery images
// Max 15 images per product
type ProductImage struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:uuidv7()" json:"id"`
	ProductID uuid.UUID `gorm:"type:uuid;not null" json:"product_id"`
	ImageURL  string    `gorm:"type:text;not null" json:"image_url"`
	Caption   string    `gorm:"type:varchar(255)" json:"caption,omitempty"`
	IsMain    bool      `gorm:"default:false" json:"is_main"`
	SortOrder int       `gorm:"default:0" json:"sort_order"`
	FileSize  int       `json:"file_size,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Relations
	Product *Product `gorm:"foreignKey:ProductID" json:"product,omitempty"`
}

func (ProductImage) TableName() string {
	return "product_images"
}

// MaxProductImages is the maximum number of images per product
const MaxProductImages = 15
