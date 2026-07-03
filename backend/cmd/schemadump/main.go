// Command schemadump creates the full smartscan schema in an empty database
// using GORM AutoMigrate over every model. It exists to (re)generate the
// baseline migration: run it against a scratch Postgres, then pg_dump the
// result. Not shipped in production images.
package main

import (
	"fmt"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/gamatritunggal/smartscan/backend/internal/models"
)

func main() {
	dsn := os.Getenv("SCHEMA_DUMP_DSN")
	if dsn == "" {
		fmt.Fprintln(os.Stderr, "SCHEMA_DUMP_DSN is required")
		os.Exit(1)
	}

	// Pass 1: create all tables without FK constraints (sidesteps circular
	// references like tenants ↔ page_templates and GORM's relation pull-in order).
	dbNoFK, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger:                                   logger.Default.LogMode(logger.Warn),
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "connect: %v\n", err)
		os.Exit(1)
	}
	// Pass 2: same models with constraints enabled adds the FKs.
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Warn)})
	if err != nil {
		fmt.Fprintf(os.Stderr, "connect: %v\n", err)
		os.Exit(1)
	}

	all := []interface{}{
		&models.Country{},
		&models.Province{},
		&models.City{},
		&models.User{},
		&models.Tenant{},
		&models.TenantStaff{},
		&models.TenantSettings{},
		&models.TenantLocation{},
		&models.AppSettings{},
		&models.CertificationType{},
		&models.SocialMediaPlatform{},
		&models.ThemePreset{},
		&models.Product{},
		&models.ProductImage{},
		&models.ProductCertification{},
		&models.ProductSocialLink{},
		&models.TenantSocialAccount{},
		&models.ProductSocialAccountLink{},
		&models.PageTemplate{},
		&models.QRBatch{},
		&models.QRCode{},
		&models.QRGenerationQueue{},
		&models.Interaction{},
		&models.CounterfeitDetection{},
		&models.CounterfeitReport{},
		&models.GeofenceViolation{},
		&models.GeofenceZoneTemplate{},
		&models.QCScan{},
		&models.InventoryMovement{},
		&models.WarrantyActivation{},
		&models.ActivityLog{},
		&models.Notification{},
	}

	if err := dbNoFK.AutoMigrate(all...); err != nil {
		fmt.Fprintf(os.Stderr, "automigrate (tables): %v\n", err)
		os.Exit(1)
	}
	if err := db.AutoMigrate(all...); err != nil {
		fmt.Fprintf(os.Stderr, "automigrate (constraints): %v\n", err)
		os.Exit(1)
	}
	fmt.Println("schema created")
}
