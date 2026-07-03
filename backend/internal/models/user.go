package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserType enum
type UserType string

const (
	UserTypeTenantStaff      UserType = "tenant_staff"
)

// UserStatus enum
type UserStatus string

const (
	UserStatusActive   UserStatus = "active"
	UserStatusInactive UserStatus = "inactive"
)

// User represents the main users table
type User struct {
	ID                 uuid.UUID      `gorm:"type:uuid;primary_key;default:uuidv7()" json:"id"`
	Email              string         `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`
	PasswordHash       string         `gorm:"type:varchar(255);not null" json:"-"`
	UserType           UserType       `gorm:"type:varchar(50);not null" json:"user_type"`
	Status             UserStatus     `gorm:"type:varchar(20);default:'active'" json:"status"`
	MustChangePassword bool           `gorm:"default:false" json:"must_change_password"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
	DeletedAt          gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	// Relations
	TenantStaff    *TenantStaff    `gorm:"foreignKey:UserID" json:"tenant_staff,omitempty"`
}

func (User) TableName() string {
	return "users"
}

// TenantStaffRole enum
type TenantStaffRole string

const (
	TenantStaffRoleAdmin          TenantStaffRole = "admin"
	TenantStaffRoleQCStaff        TenantStaffRole = "qc_staff"
	TenantStaffRoleWarehouseStaff TenantStaffRole = "warehouse_staff"
)

// TenantStaff represents staff members of tenant companies
type TenantStaff struct {
	ID             uuid.UUID       `gorm:"type:uuid;primary_key;default:uuidv7()" json:"id"`
	TenantID       uuid.UUID       `gorm:"type:uuid;not null" json:"tenant_id"`
	UserID         uuid.UUID       `gorm:"type:uuid;not null" json:"user_id"`
	FullName       string          `gorm:"type:varchar(255);not null" json:"full_name"`
	PhoneNumber    string          `gorm:"type:varchar(20)" json:"phone_number"`
	Address        string          `gorm:"type:text" json:"address"`
	Position       string          `gorm:"type:varchar(100)" json:"position"`
	Role           TenantStaffRole `gorm:"type:varchar(50);not null" json:"role"`
	IsPrimaryAdmin bool            `gorm:"default:false" json:"is_primary_admin"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`

	// Relations
	User   *User   `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Tenant *Tenant `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
}

func (TenantStaff) TableName() string {
	return "tenant_staff"
}
