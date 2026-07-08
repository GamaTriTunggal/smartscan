// seed.go implements `smartscan-admin seed-demo`: it populates a fresh
// deployment with a realistic demo company so an evaluator can explore every
// feature (products, QR batches, scan analytics, heatmap, geofence, warranty,
// counterfeit, QC/warehouse jobs) without clicking through the setup wizard.
//
// The command is idempotent: if a tenant with the demo company email already
// exists it prints "Demo data already seeded" and exits 0 without changes.
// All inserts run inside a single transaction.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/gamatritunggal/smartscan/backend/internal/config"
	"github.com/gamatritunggal/smartscan/backend/internal/models"
	"github.com/gamatritunggal/smartscan/backend/internal/utils"
)

const (
	demoTenantEmail = "demo@smartscan.local"
	demoPassword    = "DemoPass123!"

	demoAdminEmail     = "demo-admin@smartscan.local"
	demoQCEmail        = "demo-qc@smartscan.local"
	demoWarehouseEmail = "demo-warehouse@smartscan.local"

	// Distribution zone for the geofence-enabled batch (Jakarta metro).
	zoneLat      = -6.2000
	zoneLng      = 106.8167
	zoneRadiusKm = 30.0
	zoneLabel    = "Jakarta Metro Distribution Zone"

	// checkBatchGeofence applies a 2 km GPS buffer before flagging a violation;
	// seeded violations mirror that so the numbers match what the live scan
	// flow would have produced.
	gpsBufferKm = 2.0
)

var errAlreadySeeded = errors.New("demo data already seeded")

func newID() uuid.UUID { return uuid.Must(uuid.NewV7()) }

func mustJSON(v interface{}) datatypes.JSON {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err) // static seed data; cannot fail
	}
	return datatypes.JSON(b)
}

func strPtr(s string) *string { return &s }

// geoSpot is a named coordinate used for seeded scan locations.
type geoSpot struct {
	lat, lng       float64
	city, province string
}

var scanUserAgents = []string{
	"Mozilla/5.0 (Linux; Android 14; SM-A546B) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Mobile Safari/537.36",
	"Mozilla/5.0 (iPhone; CPU iPhone OS 17_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.5 Mobile/15E148 Safari/604.1",
	"Mozilla/5.0 (Linux; Android 13; Redmi Note 12) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Mobile Safari/537.36",
	"Mozilla/5.0 (Linux; Android 14; Pixel 8) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Mobile Safari/537.36",
}

// seedStats collects row counts for the summary printed on success.
type seedStats struct {
	products, batches, qrCodes        int
	interactions, violations          int
	warranties, counterfeitDetections int
	qcScans, warehouseMovements       int
}

func seedDemo(db *gorm.DB, cfg *config.Config) {
	// The development connection logs every SQL statement; ~200 seed inserts
	// would bury the credentials block. Errors are still reported by hand.
	db = db.Session(&gorm.Session{Logger: db.Logger.LogMode(gormlogger.Silent)})

	var (
		stats seedStats
		// Captured for printing example public URLs after commit.
		validateQR models.QRCode // a QR of the plain product (Classic Snack)
		warrantyQR models.QRCode // an unregistered QR of the warranty product (Premium Coffee Beans)
	)

	err := db.Transaction(func(tx *gorm.DB) error {
		// ---- Idempotency sentinel -------------------------------------------
		var existing int64
		if err := tx.Model(&models.Tenant{}).
			Where("company_email = ?", demoTenantEmail).
			Count(&existing).Error; err != nil {
			return fmt.Errorf("checking for existing demo tenant (are migrations up to date?): %w", err)
		}
		if existing > 0 {
			return errAlreadySeeded
		}

		rng := rand.New(rand.NewSource(20260708)) // deterministic layout
		now := time.Now().UTC()
		daysAgo := func(d int) time.Time { return now.AddDate(0, 0, -d) }

		randomIP := func() string {
			// Indonesian consumer ISP-looking ranges.
			prefixes := []string{"114.124", "36.68", "103.147", "180.244"}
			return fmt.Sprintf("%s.%d.%d", prefixes[rng.Intn(len(prefixes))], rng.Intn(254)+1, rng.Intn(254)+1)
		}

		// ---- Reference data (seeded by migrations — link, don't create) -----
		certByCode := func(code string) (*models.CertificationType, error) {
			var ct models.CertificationType
			if err := tx.Where("code = ?", code).First(&ct).Error; err != nil {
				return nil, fmt.Errorf("certification type %q not found (run migrations first): %w", code, err)
			}
			return &ct, nil
		}
		platformByCode := func(code string) (*models.SocialMediaPlatform, error) {
			var p models.SocialMediaPlatform
			if err := tx.Where("code = ?", code).First(&p).Error; err != nil {
				return nil, fmt.Errorf("social platform %q not found (run migrations first): %w", code, err)
			}
			return &p, nil
		}

		certBPOM, err := certByCode("BPOM")
		if err != nil {
			return err
		}
		certHalal, err := certByCode("HALAL_BPJPH")
		if err != nil {
			return err
		}
		certSNI, err := certByCode("SNI")
		if err != nil {
			return err
		}
		certPIRT, err := certByCode("PIRT")
		if err != nil {
			return err
		}
		platInstagram, err := platformByCode("instagram")
		if err != nil {
			return err
		}
		platWhatsApp, err := platformByCode("whatsapp")
		if err != nil {
			return err
		}
		platTikTok, err := platformByCode("tiktok")
		if err != nil {
			return err
		}

		var provJakarta, provJabar models.Province
		if err := tx.Where("country_code = ? AND name = ?", "ID", "DKI Jakarta").First(&provJakarta).Error; err != nil {
			return fmt.Errorf("province 'DKI Jakarta' not found (run migrations first): %w", err)
		}
		if err := tx.Where("country_code = ? AND name = ?", "ID", "Jawa Barat").First(&provJabar).Error; err != nil {
			return fmt.Errorf("province 'Jawa Barat' not found: %w", err)
		}
		var cityJaksel, cityBandung models.City
		if err := tx.Where("province_id = ? AND name = ?", provJakarta.ID, "Kota Jakarta Selatan").First(&cityJaksel).Error; err != nil {
			return fmt.Errorf("city 'Kota Jakarta Selatan' not found: %w", err)
		}
		if err := tx.Where("province_id = ? AND name = ?", provJabar.ID, "Kota Bandung").First(&cityBandung).Error; err != nil {
			return fmt.Errorf("city 'Kota Bandung' not found: %w", err)
		}

		// ---- 1. Tenant + settings -------------------------------------------
		tenant := models.Tenant{
			ID:             newID(),
			CompanyName:    "Demo Company",
			CompanyAddress: "Jl. Jenderal Sudirman No. 88, Senayan",
			Country:        "Indonesia",
			Province:       "DKI Jakarta",
			City:           "Kota Jakarta Selatan",
			CountryCode:    strPtr("ID"),
			ProvinceID:     &provJakarta.ID,
			CityID:         &cityJaksel.ID,
			PostalCode:     "12190",
			BusinessField:  "Food & Beverage",
			PhoneNumber:    "+62215550188",
			CompanyEmail:   demoTenantEmail,
			Slug:           "demo-company",
			CreatedAt:      daysAgo(45),
		}
		if err := tx.Create(&tenant).Error; err != nil {
			return fmt.Errorf("creating demo tenant: %w", err)
		}

		// Same shape the counterfeit settings page writes (UpdateCounterfeitSettings).
		tenantSettings := []models.TenantSettings{
			{
				ID:         newID(),
				TenantID:   tenant.ID,
				SettingKey: "counterfeit_thresholds",
				SettingValue: mustJSON(map[string]interface{}{
					"qc_scan_max":            0,
					"warehouse_scan_max":     0,
					"end_user_scan_max":      3,
					"velocity_check_enabled": false,
					"max_speed_kmh":          1000,
					"alert_on_detection":     true,
					"auto_flag_suspicious":   true,
				}),
			},
			{
				ID:         newID(),
				TenantID:   tenant.ID,
				SettingKey: "public_contact", // shape from company_contact.go
				SettingValue: mustJSON(map[string]string{
					"phone":    "+62215550188",
					"whatsapp": "+6281180001234",
					"email":    demoTenantEmail,
					"website":  "https://demo-company.example.com",
					"address":  "Jl. Jenderal Sudirman No. 88, Jakarta Selatan 12190",
				}),
			},
		}
		if err := tx.Create(&tenantSettings).Error; err != nil {
			return fmt.Errorf("creating tenant settings: %w", err)
		}

		// ---- 2. Users + staff -----------------------------------------------
		passwordHash, err := utils.HashPassword(demoPassword)
		if err != nil {
			return fmt.Errorf("hashing demo password: %w", err)
		}

		type staffSeed struct {
			email, fullName, position string
			role                      models.TenantStaffRole
			primary                   bool
		}
		staffSeeds := []staffSeed{
			{demoAdminEmail, "Demo Admin", "Operations Manager", models.TenantStaffRoleAdmin, true},
			{demoQCEmail, "Dewi Lestari", "QC Inspector", models.TenantStaffRoleQCStaff, false},
			{demoWarehouseEmail, "Budi Santoso", "Warehouse Operator", models.TenantStaffRoleWarehouseStaff, false},
		}
		staffByRole := map[models.TenantStaffRole]models.TenantStaff{}
		for _, s := range staffSeeds {
			user := models.User{
				ID:                 newID(),
				Email:              s.email,
				PasswordHash:       passwordHash,
				UserType:           models.UserTypeTenantStaff,
				Status:             models.UserStatusActive,
				MustChangePassword: false,
				CreatedAt:          daysAgo(45),
			}
			if err := tx.Create(&user).Error; err != nil {
				return fmt.Errorf("creating user %s: %w", s.email, err)
			}
			staff := models.TenantStaff{
				ID:             newID(),
				TenantID:       tenant.ID,
				UserID:         user.ID,
				FullName:       s.fullName,
				PhoneNumber:    "+62812" + fmt.Sprintf("%08d", rng.Intn(100000000)),
				Position:       s.position,
				Role:           s.role,
				IsPrimaryAdmin: s.primary,
				CreatedAt:      daysAgo(45),
			}
			if err := tx.Create(&staff).Error; err != nil {
				return fmt.Errorf("creating staff %s: %w", s.email, err)
			}
			staffByRole[s.role] = staff
		}
		adminStaff := staffByRole[models.TenantStaffRoleAdmin]
		qcStaff := staffByRole[models.TenantStaffRoleQCStaff]
		warehouseStaff := staffByRole[models.TenantStaffRoleWarehouseStaff]

		// ---- Locations (needed by QC / warehouse job pages) ------------------
		warehouseLoc := models.TenantLocation{
			ID:           newID(),
			TenantID:     tenant.ID,
			LocationName: "Jakarta Main Warehouse",
			LocationType: models.LocationTypeWarehouse,
			Address:      "Jl. Raya Cakung Cilincing KM 2, Jakarta Utara",
			City:         "Jakarta",
			Province:     "DKI Jakarta",
			PostalCode:   "14140",
			Geolocation:  mustJSON(map[string]float64{"lat": -6.1352, "lng": 106.9133}),
			Status:       "active",
			CreatedAt:    daysAgo(44),
		}
		qcLoc := models.TenantLocation{
			ID:           newID(),
			TenantID:     tenant.ID,
			LocationName: "Cikarang QC Line",
			LocationType: models.LocationTypeQCArea,
			Address:      "Kawasan Industri Jababeka Blok C-12, Cikarang",
			City:         "Bekasi",
			Province:     "Jawa Barat",
			PostalCode:   "17530",
			Geolocation:  mustJSON(map[string]float64{"lat": -6.3346, "lng": 107.1281}),
			Status:       "active",
			CreatedAt:    daysAgo(44),
		}
		if err := tx.Create(&[]models.TenantLocation{warehouseLoc, qcLoc}).Error; err != nil {
			return fmt.Errorf("creating tenant locations: %w", err)
		}

		// ---- 3. Products ------------------------------------------------------
		displayConfigFull := mustJSON(map[string]bool{
			"product_name": true, "product_code": true, "batch_code": true,
			"production_date": true, "expiry_date": true, "brand_name": true,
			"show_verification_count": true,
		})
		displayConfigBasic := mustJSON(map[string]bool{
			"product_name": true, "product_code": false, "batch_code": false,
			"production_date": false, "expiry_date": true, "brand_name": true,
			"show_verification_count": true,
		})
		// Matches handlers.WarrantyFieldsConfig — the shape RegisterWarranty
		// validates against and WarrantyPage renders.
		coffeeWarrantyConfig := mustJSON(map[string]interface{}{
			"enabled": true,
			"fields": map[string]string{
				"store_name": "optional",
				"country":    "required",
				"province":   "required",
				"city":       "required",
				"address":    "required",
			},
			"custom_fields": []map[string]interface{}{
				{"id": "serial_number", "label": "Serial Number", "type": "text", "required": false},
			},
		})

		productCoffee := models.Product{
			ID:                   newID(),
			TenantID:             tenant.ID,
			ProductName:          "Premium Coffee Beans",
			ProductCode:          "COF-001",
			Description:          "Single-origin arabica beans from the Gayo highlands, medium roast, 250g vacuum pack. Every pack carries a unique QR code for authenticity verification and a 12-month freshness warranty.",
			Status:               models.ProductStatusActive,
			DisplayConfig:        displayConfigFull,
			WarrantyFieldsConfig: coffeeWarrantyConfig,
			WarrantyEnabled:      true,
			WarrantyMonths:       12,
			WebsiteURL:           "https://demo-company.example.com/coffee",
			WebsiteCaption:       "Shop Premium Coffee",
			Videos:               datatypes.JSON("[]"),
			CreatedBy:            &adminStaff.ID,
			CreatedAt:            daysAgo(43),
		}
		productHerbal := models.Product{
			ID:            newID(),
			TenantID:      tenant.ID,
			ProductName:   "Herbal Supplement",
			ProductCode:   "HRB-002",
			Description:   "Traditional jamu-based immunity supplement, 60 capsules. Distribution is restricted to the Jakarta metropolitan area — out-of-zone scans are flagged for the anti-counterfeit team.",
			Status:        models.ProductStatusActive,
			DisplayConfig: displayConfigFull,
			Videos:        datatypes.JSON("[]"),
			CreatedBy:     &adminStaff.ID,
			CreatedAt:     daysAgo(43),
		}
		productSnack := models.Product{
			ID:            newID(),
			TenantID:      tenant.ID,
			ProductName:   "Classic Snack",
			ProductCode:   "SNK-003",
			Description:   "Crispy cassava chips with balado seasoning, 180g family pack.",
			Status:        models.ProductStatusActive,
			DisplayConfig: displayConfigBasic,
			Videos:        datatypes.JSON("[]"),
			CreatedBy:     &adminStaff.ID,
			CreatedAt:     daysAgo(42),
		}
		if err := tx.Create(&[]models.Product{productCoffee, productHerbal, productSnack}).Error; err != nil {
			return fmt.Errorf("creating products: %w", err)
		}
		stats.products = 3

		// Certifications: link to migration-seeded certification types.
		certs := []models.ProductCertification{
			{ID: newID(), ProductID: productCoffee.ID, CertificationTypeID: certBPOM.ID, RegistrationNumber: "MD 867013002114", SortOrder: 0, CreatedAt: daysAgo(43)},
			{ID: newID(), ProductID: productCoffee.ID, CertificationTypeID: certHalal.ID, RegistrationNumber: "ID31110003420626", SortOrder: 1, CreatedAt: daysAgo(43)},
			{ID: newID(), ProductID: productHerbal.ID, CertificationTypeID: certBPOM.ID, RegistrationNumber: "TR 213688091", SortOrder: 0, CreatedAt: daysAgo(43)},
			{ID: newID(), ProductID: productHerbal.ID, CertificationTypeID: certSNI.ID, RegistrationNumber: "SNI 8664:2018-0451", SortOrder: 1, CreatedAt: daysAgo(43)},
			{ID: newID(), ProductID: productSnack.ID, CertificationTypeID: certPIRT.ID, RegistrationNumber: "P-IRT 2153171010885-27", SortOrder: 0, CreatedAt: daysAgo(42)},
		}
		if err := tx.Create(&certs).Error; err != nil {
			return fmt.Errorf("creating product certifications: %w", err)
		}

		// Tenant social accounts (N:M) + one legacy per-product social link.
		igAccount := models.TenantSocialAccount{
			ID: newID(), TenantID: tenant.ID, PlatformID: platInstagram.ID,
			AccountHandle: "demo.company", IsActive: true, CreatedAt: daysAgo(43),
		}
		waAccount := models.TenantSocialAccount{
			ID: newID(), TenantID: tenant.ID, PlatformID: platWhatsApp.ID,
			AccountHandle: "6281180001234", IsActive: true, CreatedAt: daysAgo(43),
		}
		if err := tx.Create(&[]models.TenantSocialAccount{igAccount, waAccount}).Error; err != nil {
			return fmt.Errorf("creating tenant social accounts: %w", err)
		}
		accountLinks := []models.ProductSocialAccountLink{
			{ID: newID(), ProductID: productCoffee.ID, SocialAccountID: igAccount.ID, SortOrder: 0, CreatedAt: daysAgo(43)},
			{ID: newID(), ProductID: productCoffee.ID, SocialAccountID: waAccount.ID, SortOrder: 1, CreatedAt: daysAgo(43)},
			{ID: newID(), ProductID: productHerbal.ID, SocialAccountID: igAccount.ID, SortOrder: 0, CreatedAt: daysAgo(43)},
			{ID: newID(), ProductID: productSnack.ID, SocialAccountID: waAccount.ID, SortOrder: 0, CreatedAt: daysAgo(42)},
		}
		if err := tx.Create(&accountLinks).Error; err != nil {
			return fmt.Errorf("creating product social account links: %w", err)
		}
		legacyLink := models.ProductSocialLink{
			ID: newID(), ProductID: productCoffee.ID, PlatformID: platTikTok.ID,
			HandleOrURL: "demo.company.id", CreatedAt: daysAgo(43),
		}
		if err := tx.Create(&legacyLink).Error; err != nil {
			return fmt.Errorf("creating legacy product social link: %w", err)
		}

		// ---- 4. QR batches + codes -------------------------------------------
		makeBatch := func(product *models.Product, name, code string, count int, created time.Time, prodDaysAgo, shelfMonths int) models.QRBatch {
			prodDate := daysAgo(prodDaysAgo)
			expDate := prodDate.AddDate(0, shelfMonths, 0)
			return models.QRBatch{
				ID:             newID(),
				TenantID:       tenant.ID,
				ProductID:      product.ID,
				BatchName:      name,
				BatchCode:      code,
				QRCount:        count,
				Status:         models.QRBatchStatusCompleted, // worker's terminal success value
				ProductionDate: &prodDate,
				ExpiryDate:     &expDate,
				CreatedBy:      &adminStaff.ID,
				CreatedAt:      created,
			}
		}

		batchCoffee1 := makeBatch(&productCoffee, "Coffee May Production", "COF-2605", 30, daysAgo(40), 42, 18)
		batchCoffee2 := makeBatch(&productCoffee, "Coffee June Production", "COF-2606", 30, daysAgo(20), 21, 18)
		batchHerbal := makeBatch(&productHerbal, "Herbal June Production", "HRB-2606", 25, daysAgo(17), 18, 24)
		// Created within the last week so the dashboard's default "this month"
		// view has QR codes created in range (its total_qr_codes tile filters
		// by qr_codes.created_at).
		batchSnack := makeBatch(&productSnack, "Snack July Production", "SNK-2607", 20, daysAgo(6), 8, 6)

		// Geofence distribution zone on the herbal batch.
		lat, lng, radius := zoneLat, zoneLng, zoneRadiusKm
		batchHerbal.GeofenceEnabled = true
		batchHerbal.GeofenceLatitude = &lat
		batchHerbal.GeofenceLongitude = &lng
		batchHerbal.GeofenceRadiusKm = &radius
		batchHerbal.GeofenceLabel = zoneLabel

		batches := []*models.QRBatch{&batchCoffee1, &batchCoffee2, &batchHerbal, &batchSnack}
		for _, b := range batches {
			if err := tx.Create(b).Error; err != nil {
				return fmt.Errorf("creating batch %s: %w", b.BatchCode, err)
			}
		}
		stats.batches = len(batches)

		// Saved zone template (reusable in batch creation).
		zoneTemplate := models.GeofenceZoneTemplate{
			ID: newID(), TenantID: tenant.ID, TemplateName: "Jakarta Metro",
			Latitude: zoneLat, Longitude: zoneLng, RadiusKm: zoneRadiusKm,
			Label: zoneLabel, UsageCount: 1, CreatedAt: daysAgo(18),
		}
		if err := tx.Create(&zoneTemplate).Error; err != nil {
			return fmt.Errorf("creating geofence zone template: %w", err)
		}

		// QR codes: same format the async worker generates (prefix + 32-char hex + suffix).
		makeCodes := func(batch *models.QRBatch) ([]models.QRCode, error) {
			codes := make([]models.QRCode, batch.QRCount)
			for i := range codes {
				codes[i] = models.QRCode{
					ID:                newID(),
					BatchID:           batch.ID,
					QRUUID:            newID(),
					QRCode:            utils.GenerateRandomHexWithFallback(16),
					Status:            models.QRCodeStatusActive,
					CounterfeitStatus: models.CounterfeitStatusValid,
					CreatedAt:         batch.CreatedAt,
				}
			}
			if err := tx.CreateInBatches(codes, 500).Error; err != nil {
				return nil, fmt.Errorf("creating QR codes for batch %s: %w", batch.BatchCode, err)
			}
			return codes, nil
		}
		codesCoffee1, err := makeCodes(&batchCoffee1)
		if err != nil {
			return err
		}
		codesCoffee2, err := makeCodes(&batchCoffee2)
		if err != nil {
			return err
		}
		codesHerbal, err := makeCodes(&batchHerbal)
		if err != nil {
			return err
		}
		codesSnack, err := makeCodes(&batchSnack)
		if err != nil {
			return err
		}
		stats.qrCodes = len(codesCoffee1) + len(codesCoffee2) + len(codesHerbal) + len(codesSnack)

		// A couple of admin-disabled codes for status variety.
		disabledIDs := []uuid.UUID{codesSnack[18].ID, codesSnack[19].ID}
		if err := tx.Model(&models.QRCode{}).Where("id IN ?", disabledIDs).
			Update("status", models.QRCodeStatusInactive).Error; err != nil {
			return fmt.Errorf("disabling sample QR codes: %w", err)
		}
		codesSnack[18].Status = models.QRCodeStatusInactive
		codesSnack[19].Status = models.QRCodeStatusInactive

		// Completed generation-queue records (what the worker leaves behind).
		for _, b := range batches {
			started := b.CreatedAt.Add(2 * time.Second)
			completed := b.CreatedAt.Add(9 * time.Second)
			q := models.QRGenerationQueue{
				ID: newID(), BatchID: b.ID,
				TotalQRCount: b.QRCount, GeneratedCount: b.QRCount,
				Status: models.QRGenerationQueueStatusCompleted, WorkerID: "seed-demo",
				CreatedAt: b.CreatedAt, StartedAt: &started, CompletedAt: &completed,
			}
			if err := tx.Create(&q).Error; err != nil {
				return fmt.Errorf("creating generation queue record: %w", err)
			}
		}

		// ---- 5. Scan history (consumer validations) ---------------------------
		insideSpots := []geoSpot{
			{-6.2088, 106.8456, "Jakarta", "DKI Jakarta"},
			{-6.2607, 106.7816, "Jakarta", "DKI Jakarta"},
			{-6.1751, 106.8650, "Jakarta", "DKI Jakarta"},
			{-6.9175, 107.6191, "Bandung", "Jawa Barat"},
			{-6.9034, 107.5731, "Bandung", "Jawa Barat"},
			{-7.2575, 112.7521, "Surabaya", "Jawa Timur"},
			{-7.2892, 112.7344, "Surabaya", "Jawa Timur"},
		}
		// Scans on the geofenced herbal batch that land INSIDE the zone must use
		// Jakarta coordinates only.
		jakartaSpots := insideSpots[:3]

		var interactions []models.Interaction

		// newScan builds a product_validation interaction. spot==nil → scan
		// without geolocation (consumer denied the GPS prompt).
		newScan := func(qr *models.QRCode, at time.Time, spot *geoSpot) models.Interaction {
			it := models.Interaction{
				ID:                     newID(),
				QRCodeID:               &qr.ID,
				TenantID:               tenant.ID,
				InteractionCategory:    models.InteractionCategoryEndUserAccess,
				InteractionSubcategory: models.InteractionSubcategoryProductValidation,
				InteractionStatus:      models.InteractionStatusSuccess,
				IPAddress:              randomIP(),
				UserAgent:              scanUserAgents[rng.Intn(len(scanUserAgents))],
				CreatedAt:              at,
			}
			if spot != nil {
				it.Geolocation = mustJSON(map[string]interface{}{
					"lat":          spot.lat + (rng.Float64()-0.5)*0.04,
					"lng":          spot.lng + (rng.Float64()-0.5)*0.04,
					"accuracy":     10 + rng.Float64()*40,
					"city":         spot.city,
					"province":     spot.province,
					"country":      "Indonesia",
					"country_code": "ID",
				})
			}
			return it
		}
		randomTime := func(maxDaysAgo int) time.Time {
			return now.Add(-time.Duration(rng.Float64()*float64(maxDaysAgo)*24) * time.Hour)
		}

		// Coffee batch 1: one scan per code over the past 35 days.
		for i := range codesCoffee1 {
			var spot *geoSpot
			if i%5 != 0 { // ~20% of consumers deny geolocation
				spot = &insideSpots[rng.Intn(len(insideSpots))]
			}
			interactions = append(interactions, newScan(&codesCoffee1[i], randomTime(35), spot))
		}
		// Coffee batch 2: one scan per code (except index 0, the counterfeit case below).
		for i := 1; i < len(codesCoffee2); i++ {
			var spot *geoSpot
			if i%5 != 0 {
				spot = &insideSpots[rng.Intn(len(insideSpots))]
			}
			interactions = append(interactions, newScan(&codesCoffee2[i], randomTime(19), spot))
		}
		// Snack: one scan per active code + a dozen repeat scans.
		for i := 0; i < 18; i++ {
			var spot *geoSpot
			if i%4 != 0 {
				spot = &insideSpots[rng.Intn(len(insideSpots))]
			}
			interactions = append(interactions, newScan(&codesSnack[i], randomTime(6), spot))
			if i < 12 {
				spot2 := insideSpots[rng.Intn(len(insideSpots))]
				interactions = append(interactions, newScan(&codesSnack[i], randomTime(4), &spot2))
			}
		}

		// Herbal (geofenced): two scans per code. The second scan on the first
		// seven codes happens OUTSIDE the distribution zone → violation rows.
		outsideSpots := []geoSpot{
			{-6.5500, 106.7800, "Kabupaten Bogor", "Jawa Barat"}, // ~7 km past edge → low
			{-6.5950, 106.8166, "Bogor", "Jawa Barat"},           // ~12 km → medium
			{-6.6200, 106.8100, "Bogor", "Jawa Barat"},           // ~15 km → medium
			{-6.9175, 107.6191, "Bandung", "Jawa Barat"},         // ~86 km → high
			{-6.7320, 108.5523, "Cirebon", "Jawa Barat"},         // ~168 km → high
			{-7.2575, 112.7521, "Surabaya", "Jawa Timur"},        // ~630 km → critical
			{-7.2892, 112.7344, "Surabaya", "Jawa Timur"},        // ~630 km → critical
		}
		type violationSeed struct {
			interactionIdx int
			qr             *models.QRCode
			lat, lng       float64
			accuracy       float64
			at             time.Time
		}
		var violationSeeds []violationSeed
		for i := range codesHerbal {
			spot := jakartaSpots[rng.Intn(len(jakartaSpots))]
			interactions = append(interactions, newScan(&codesHerbal[i], randomTime(16), &spot))

			if i < len(outsideSpots) {
				// Out-of-zone scan. Exact coordinates (no jitter) so severity is deterministic.
				at := now.Add(-time.Duration(float64(i*2)*24+rng.Float64()*20) * time.Hour)
				out := outsideSpots[i]
				accuracy := 15 + rng.Float64()*25
				it := models.Interaction{
					ID:                     newID(),
					QRCodeID:               &codesHerbal[i].ID,
					TenantID:               tenant.ID,
					InteractionCategory:    models.InteractionCategoryEndUserAccess,
					InteractionSubcategory: models.InteractionSubcategoryProductValidation,
					InteractionStatus:      models.InteractionStatusSuccess,
					IPAddress:              randomIP(),
					UserAgent:              scanUserAgents[rng.Intn(len(scanUserAgents))],
					CreatedAt:              at,
					Geolocation: mustJSON(map[string]interface{}{
						"lat": out.lat, "lng": out.lng, "accuracy": accuracy,
						"city": out.city, "province": out.province,
						"country": "Indonesia", "country_code": "ID",
					}),
				}
				interactions = append(interactions, it)
				violationSeeds = append(violationSeeds, violationSeed{
					interactionIdx: len(interactions) - 1,
					qr:             &codesHerbal[i],
					lat:            out.lat, lng: out.lng,
					accuracy: accuracy, at: at,
				})
			} else {
				spot2 := jakartaSpots[rng.Intn(len(jakartaSpots))]
				interactions = append(interactions, newScan(&codesHerbal[i], randomTime(12), &spot2))
			}
		}

		// Counterfeit case: one coffee QR scanned 6× across distant cities in 3
		// days — past the end_user_scan_max threshold of 3.
		counterfeitQR := &codesCoffee2[0]
		counterfeitSpots := []geoSpot{
			{-6.2088, 106.8456, "Jakarta", "DKI Jakarta"},
			{-6.2607, 106.7816, "Jakarta", "DKI Jakarta"},
			{3.5952, 98.6722, "Medan", "Sumatera Utara"},
			{-7.2575, 112.7521, "Surabaya", "Jawa Timur"},
			{3.5952, 98.6722, "Medan", "Sumatera Utara"},
			{-6.9175, 107.6191, "Bandung", "Jawa Barat"},
		}
		var counterfeitInteractionIDs []string
		var counterfeitFirst, counterfeitLast time.Time
		for i, spot := range counterfeitSpots {
			s := spot
			at := daysAgo(4).Add(time.Duration(i*11) * time.Hour)
			it := newScan(counterfeitQR, at, &s)
			interactions = append(interactions, it)
			counterfeitInteractionIDs = append(counterfeitInteractionIDs, it.ID.String())
			if i == 0 {
				counterfeitFirst = at
			}
			counterfeitLast = at
		}

		// ---- 6b/7. Staff scans (QC + warehouse job pages) ---------------------
		var qcScans []models.QCScan
		for i := 0; i < 10; i++ {
			qr := &codesCoffee2[10+i]
			at := daysAgo(18).Add(time.Duration(i*36) * time.Hour)
			status := models.QCStatusPass
			if i == 3 || i == 7 {
				status = models.QCStatusFailed
			}
			scanLat := -6.3346 + (rng.Float64()-0.5)*0.002
			scanLng := 107.1281 + (rng.Float64()-0.5)*0.002
			geo := mustJSON(map[string]float64{"lat": scanLat, "lng": scanLng})
			qcScans = append(qcScans, models.QCScan{
				ID: newID(), LocationID: &qcLoc.ID, QRCodeID: qr.ID,
				QCStatus: status, ScannedBy: &qcStaff.ID,
				ScanGeolocation: geo, ScannedAt: at,
			})
			interactions = append(interactions, models.Interaction{
				ID: newID(), QRCodeID: &qr.ID, TenantID: tenant.ID,
				InteractionCategory:    models.InteractionCategoryTenantAccess,
				InteractionSubcategory: models.InteractionSubcategoryQCScan,
				InteractionStatus:      models.InteractionStatusSuccess,
				// interactions.scanned_by FKs to users(id) — unlike qc_scans,
				// whose scanned_by FKs to tenant_staff(id).
				ScannedBy:   &qcStaff.UserID,
				IPAddress:   "10.20.0.15",
				Geolocation: geo,
				CreatedAt:   at,
			})
		}
		if err := tx.Create(&qcScans).Error; err != nil {
			return fmt.Errorf("creating QC scans: %w", err)
		}
		stats.qcScans = len(qcScans)

		var movements []models.InventoryMovement
		addMovement := func(qr *models.QRCode, mtype models.MovementType, at time.Time) {
			geo := mustJSON(map[string]float64{
				"lat": -6.1352 + (rng.Float64()-0.5)*0.002,
				"lng": 106.9133 + (rng.Float64()-0.5)*0.002,
			})
			movements = append(movements, models.InventoryMovement{
				ID: newID(), LocationID: warehouseLoc.ID, QRCodeID: qr.ID,
				MovementType: mtype, ScannedBy: &warehouseStaff.ID,
				ScanGeolocation: geo, ScannedAt: at,
			})
			subcat := models.InteractionSubcategoryWarehouseScan
			interactions = append(interactions, models.Interaction{
				ID: newID(), QRCodeID: &qr.ID, TenantID: tenant.ID,
				InteractionCategory:    models.InteractionCategoryTenantAccess,
				InteractionSubcategory: subcat,
				InteractionStatus:      models.InteractionStatusSuccess,
				// interactions.scanned_by FKs to users(id), not tenant_staff(id).
				ScannedBy:   &warehouseStaff.UserID,
				IPAddress:   "10.20.0.31",
				Geolocation: geo,
				CreatedAt:   at,
			})
		}
		for i := 0; i < 12; i++ {
			addMovement(&codesCoffee1[10+i], models.MovementTypeIn, daysAgo(34).Add(time.Duration(i*7)*time.Hour))
		}
		for i := 0; i < 4; i++ {
			addMovement(&codesCoffee1[10+i], models.MovementTypeOut, daysAgo(9).Add(time.Duration(i*13)*time.Hour))
		}
		if err := tx.Create(&movements).Error; err != nil {
			return fmt.Errorf("creating warehouse movements: %w", err)
		}
		stats.warehouseMovements = len(movements)

		// ---- 6. Warranty activations ------------------------------------------
		type warrantySeed struct {
			qr            *models.QRCode
			name, email   string
			phone, store  string
			address       string
			provinceID    int
			cityID        int
			provinceName  string
			cityName      string
			spot          geoSpot
			purchasedDays int
			activatedDays int
			serial        string
			dupAttempts   int
		}
		warrantySeeds := []warrantySeed{
			{
				qr: &codesCoffee1[1], name: "Rina Wijaya", email: "rina.wijaya@example.com",
				phone: "+6281234500011", store: "Demo Official Store — Tokopedia",
				address:    "Jl. Kemang Raya No. 12, Jakarta Selatan",
				provinceID: provJakarta.ID, cityID: cityJaksel.ID,
				provinceName: "DKI Jakarta", cityName: "Jakarta",
				spot:          geoSpot{-6.2607, 106.7816, "Jakarta", "DKI Jakarta"},
				purchasedDays: 26, activatedDays: 25, serial: "SN-2026-0117", dupAttempts: 0,
			},
			{
				qr: &codesCoffee1[2], name: "Agus Pratama", email: "agus.pratama@example.com",
				phone: "+6281234500012", store: "Kopi Corner Bandung",
				address:    "Jl. Dago No. 45, Bandung",
				provinceID: provJabar.ID, cityID: cityBandung.ID,
				provinceName: "Jawa Barat", cityName: "Bandung",
				spot:          geoSpot{-6.9175, 107.6191, "Bandung", "Jawa Barat"},
				purchasedDays: 13, activatedDays: 12, serial: "SN-2026-0242", dupAttempts: 0,
			},
			{
				qr: &codesCoffee2[5], name: "Siti Rahma", email: "siti.rahma@example.com",
				phone: "+6281234500013", store: "",
				address:    "Jl. Tebet Barat Dalam No. 3, Jakarta Selatan",
				provinceID: provJakarta.ID, cityID: cityJaksel.ID,
				provinceName: "DKI Jakarta", cityName: "Jakarta",
				spot:          geoSpot{-6.2088, 106.8456, "Jakarta", "DKI Jakarta"},
				purchasedDays: 6, activatedDays: 5, serial: "SN-2026-0388", dupAttempts: 1,
			},
		}
		countryID := "ID"
		for _, w := range warrantySeeds {
			purchase := daysAgo(w.purchasedDays)
			activated := daysAgo(w.activatedDays)
			expiry := purchase.AddDate(0, productCoffee.WarrantyMonths, 0)
			geo := mustJSON(map[string]float64{"lat": w.spot.lat, "lng": w.spot.lng})
			pid, cid := w.provinceID, w.cityID
			wa := models.WarrantyActivation{
				ID:                    newID(),
				QRCodeID:              w.qr.ID,
				CustomerName:          w.name,
				CustomerEmail:         w.email,
				CustomerPhone:         w.phone,
				PurchaseDate:          &purchase,
				PurchaseStore:         w.store,
				Address:               w.address,
				CountryCode:           &countryID,
				ProvinceID:            &pid,
				CityID:                &cid,
				ActivationData:        mustJSON(map[string]interface{}{"custom_fields": map[string]string{"serial_number": w.serial}}),
				ActivatedAt:           activated,
				WarrantyExpiryDate:    &expiry,
				IPAddress:             randomIP(),
				Geolocation:           geo,
				DuplicateAttemptCount: w.dupAttempts,
			}
			if w.dupAttempts > 0 {
				lastDup := daysAgo(2)
				wa.LastDuplicateAttemptAt = &lastDup
			}
			if err := tx.Create(&wa).Error; err != nil {
				return fmt.Errorf("creating warranty activation for %s: %w", w.name, err)
			}
			// The warranty_activation interaction RegisterWarranty records.
			interactions = append(interactions, models.Interaction{
				ID: newID(), QRCodeID: &w.qr.ID, TenantID: tenant.ID,
				InteractionCategory:    models.InteractionCategoryEndUserAccess,
				InteractionSubcategory: models.InteractionSubcategoryWarrantyActivation,
				InteractionStatus:      models.InteractionStatusSuccess,
				IPAddress:              randomIP(),
				UserAgent:              scanUserAgents[rng.Intn(len(scanUserAgents))],
				Geolocation: mustJSON(map[string]interface{}{
					"lat": w.spot.lat, "lng": w.spot.lng,
					"city": w.spot.city, "province": w.spot.province,
					"country": "Indonesia", "country_code": "ID",
				}),
				CreatedAt: activated,
			})
			stats.warranties++
		}

		// ---- Persist all interactions ------------------------------------------
		if err := tx.CreateInBatches(interactions, 500).Error; err != nil {
			return fmt.Errorf("creating interactions: %w", err)
		}
		stats.interactions = len(interactions)

		// ---- Geofence violations (paired with the out-of-zone interactions) ----
		var worstViolation *models.GeofenceViolation
		for _, vs := range violationSeeds {
			distKm := utils.HaversineDistance(vs.lat, vs.lng, zoneLat, zoneLng) / 1000.0
			edgeKm := distKm - zoneRadiusKm - gpsBufferKm
			if edgeKm <= 0 {
				return fmt.Errorf("seed bug: violation spot (%f,%f) is inside the zone", vs.lat, vs.lng)
			}
			interactionID := interactions[vs.interactionIdx].ID
			accuracy := vs.accuracy
			v := models.GeofenceViolation{
				ID:                   newID(),
				TenantID:             tenant.ID,
				BatchID:              batchHerbal.ID,
				QRCodeID:             &vs.qr.ID,
				ProductID:            &productHerbal.ID,
				InteractionID:        &interactionID,
				ScanLatitude:         vs.lat,
				ScanLongitude:        vs.lng,
				DistanceFromCenterKm: distKm,
				DistanceFromEdgeKm:   edgeKm,
				GPSAccuracyMeters:    &accuracy,
				Severity:             utils.GeofenceSeverity(edgeKm),
				CreatedAt:            vs.at,
			}
			if err := tx.Create(&v).Error; err != nil {
				return fmt.Errorf("creating geofence violation: %w", err)
			}
			if worstViolation == nil || v.DistanceFromEdgeKm > worstViolation.DistanceFromEdgeKm {
				worstViolation = &v
			}
			stats.violations++
		}

		// ---- Counterfeit detection ---------------------------------------------
		// Reason format matches validation.go recordDynamicQRScan. The AfterCreate
		// hook flips the QR's counterfeit_status to 'counterfeit'.
		idsJSON := mustJSON(counterfeitInteractionIDs)
		detection := models.CounterfeitDetection{
			ID:                     newID(),
			QRCodeID:               counterfeitQR.ID,
			TenantID:               tenant.ID,
			DetectionReason:        fmt.Sprintf("Excessive validation attempts: %d (threshold: %d)", len(counterfeitSpots), 3),
			InteractionIDs:         idsJSON,
			TotalInteractionsCount: len(counterfeitSpots),
			FirstInteractionAt:     &counterfeitFirst,
			LastInteractionAt:      &counterfeitLast,
			Status:                 models.CounterfeitDetectionStatusActive,
			CreatedAt:              counterfeitFirst.Add(33 * time.Hour), // when the 4th scan crossed the threshold
		}
		if err := tx.Create(&detection).Error; err != nil {
			return fmt.Errorf("creating counterfeit detection: %w", err)
		}
		stats.counterfeitDetections = 1

		// In-app notifications the live flows would have raised.
		notifications := []models.Notification{
			{
				ID: newID(), TenantID: tenant.ID,
				Type:  models.NotificationTypeCounterfeitAlert,
				Title: "Potential counterfeit detected",
				Body:  fmt.Sprintf("%s — QR %s flagged: %s", productCoffee.ProductName, counterfeitQR.QRCode, detection.DetectionReason),
				Link:  "/tenant/counterfeit",
				Data: mustJSON(map[string]interface{}{
					"product_name": productCoffee.ProductName,
					"qr_code":      counterfeitQR.QRCode,
					"reason":       detection.DetectionReason,
				}),
				CreatedAt: detection.CreatedAt,
			},
		}
		if worstViolation != nil {
			notifications = append(notifications, models.Notification{
				ID: newID(), TenantID: tenant.ID,
				Type:  models.NotificationTypeGeofenceViolation,
				Title: "Out-of-zone scan detected",
				Body: fmt.Sprintf("%s (batch %s) scanned %.1f km outside %s — severity %s",
					productHerbal.ProductName, batchHerbal.BatchName,
					worstViolation.DistanceFromEdgeKm, zoneLabel, "CRITICAL"),
				Link: "/tenant/geofence",
				Data: mustJSON(map[string]interface{}{
					"product_name":       productHerbal.ProductName,
					"batch_name":         batchHerbal.BatchName,
					"zone_label":         zoneLabel,
					"severity":           worstViolation.Severity,
					"distance_from_edge": worstViolation.DistanceFromEdgeKm,
				}),
				CreatedAt: worstViolation.CreatedAt,
			})
		}
		if err := tx.Create(&notifications).Error; err != nil {
			return fmt.Errorf("creating notifications: %w", err)
		}

		// QRs for the printed example URLs:
		//   validate → a plain-product code with scan history
		//   warranty → a coffee code that is NOT registered and NOT counterfeit,
		//              so the evaluator can walk the full registration flow.
		validateQR = codesSnack[0]
		warrantyQR = codesCoffee2[1]
		return nil
	})

	if errors.Is(err, errAlreadySeeded) {
		fmt.Println("Demo data already seeded")
		return
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to seed demo data: %v\n", err)
		os.Exit(1)
	}

	frontendURL := cfg.FrontendURL
	if frontendURL == "" {
		frontendURL = "http://localhost:3000"
	}
	validateB58 := utils.UUIDToBase58(validateQR.QRUUID)
	warrantyB58 := utils.UUIDToBase58(warrantyQR.QRUUID)

	fmt.Println("Demo data seeded successfully.")
	fmt.Println()
	fmt.Printf("Sign in at %s/login (the setup wizard is skipped — a company now exists):\n", frontendURL)
	fmt.Println()
	fmt.Println("  ROLE        EMAIL                            PASSWORD")
	fmt.Printf("  Admin       %-32s %s\n", demoAdminEmail, demoPassword)
	fmt.Printf("  QC          %-32s %s\n", demoQCEmail, demoPassword)
	fmt.Printf("  Warehouse   %-32s %s\n", demoWarehouseEmail, demoPassword)
	fmt.Println()
	fmt.Println("Example public QR URLs (the exact format printed inside generated QR codes):")
	fmt.Println()
	fmt.Printf("  Validate flow (Classic Snack):         %s/s/%s\n", frontendURL, validateB58)
	fmt.Printf("  Warranty flow (Premium Coffee Beans):  %s/s/%s\n", frontendURL, warrantyB58)
	fmt.Println()
	fmt.Printf("  The warranty QR lands on the validation page, whose \"Activate Warranty\"\n")
	fmt.Printf("  button opens %s/w/%s\n", frontendURL, warrantyB58)
	fmt.Println()
	fmt.Printf("Seeded: 1 company, 3 users, %d products, %d QR batches, %d QR codes,\n",
		stats.products, stats.batches, stats.qrCodes)
	fmt.Printf("%d interactions, %d geofence violations, %d warranties, %d counterfeit case,\n",
		stats.interactions, stats.violations, stats.warranties, stats.counterfeitDetections)
	fmt.Printf("%d QC scans, %d warehouse movements.\n", stats.qcScans, stats.warehouseMovements)
}
