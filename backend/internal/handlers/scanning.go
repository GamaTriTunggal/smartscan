package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gamatritunggal/smartscan/backend/internal/config"
	"github.com/gamatritunggal/smartscan/backend/internal/models"
	"github.com/gamatritunggal/smartscan/backend/internal/sentry"
	"github.com/gamatritunggal/smartscan/backend/internal/utils"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type ScanningHandler struct {
	DB  *gorm.DB
	Cfg *config.Config
}

func NewScanningHandler(db *gorm.DB, cfg *config.Config) *ScanningHandler {
	return &ScanningHandler{DB: db, Cfg: cfg}
}

// Geolocation struct for parsing JSON
type Geolocation struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

// calculateDistance calculates distance between two points in meters using Haversine formula
func calculateDistance(lat1, lng1, lat2, lng2 float64) float64 {
	return utils.HaversineDistance(lat1, lng1, lat2, lng2)
}

// validateScanLocation checks if scan location is within allowed radius
func (h *ScanningHandler) validateScanLocation(location *models.TenantLocation, scanLat, scanLng float64) (bool, float64) {
	if location.AllowedRadius == nil || *location.AllowedRadius == 0 {
		return true, 0 // No radius limit
	}

	if location.Geolocation == nil {
		return true, 0 // No location set
	}

	var locationGeo Geolocation
	if err := json.Unmarshal(location.Geolocation, &locationGeo); err != nil {
		return true, 0 // Can't parse, allow
	}

	distance := calculateDistance(locationGeo.Lat, locationGeo.Lng, scanLat, scanLng)
	return distance <= float64(*location.AllowedRadius), distance
}


// checkCounterfeitThreshold checks if scan count exceeds threshold.
// For QC/warehouse scans, it first checks for explicit QR > Batch > Product overrides.
// If an override exists, it uses the 4-level hierarchy (shared with end-user validation).
// Otherwise, falls back to tenant-level per-scan-type settings (qc_scan_max, warehouse_scan_max).
func (h *ScanningHandler) checkCounterfeitThreshold(tenantID uuid.UUID, qrCodeID uuid.UUID, scanType string) (bool, int, int) {
	// Load QR with hierarchy for override check
	var qrCode models.QRCode
	if err := h.DB.Preload("Batch.Product").First(&qrCode, "id = ?", qrCodeID).Error; err != nil {
		return false, 0, 0
	}

	// If there's an explicit override at QR/batch/product level, use hierarchy for all scan types
	hasExplicitOverride := qrCode.CounterfeitScanMax != nil ||
		(qrCode.Batch != nil && qrCode.Batch.CounterfeitScanMax != nil) ||
		(qrCode.Batch != nil && qrCode.Batch.Product != nil && qrCode.Batch.Product.CounterfeitScanMax != nil)

	if hasExplicitOverride {
		maxCount := ResolveCounterfeitThreshold(h.DB, &qrCode)
		if maxCount == 0 {
			return false, 0, 0
		}
		var currentCount int64
		switch scanType {
		case "qc_scan":
			h.DB.Model(&models.QCScan{}).Where("qr_code_id = ?", qrCodeID).Count(&currentCount)
		case "warehouse_scan":
			h.DB.Model(&models.InventoryMovement{}).Where("qr_code_id = ?", qrCodeID).Count(&currentCount)
		}
		return int(currentCount) >= maxCount, int(currentCount), maxCount
	}

	// Fallback: tenant-level per-scan-type settings
	var setting models.TenantSettings
	if err := h.DB.Where("tenant_id = ? AND setting_key = ?", tenantID, "counterfeit_thresholds").First(&setting).Error; err != nil {
		return false, 0, 0
	}

	// The blob mixes ints and bools (velocity_check_enabled etc.); decoding into
	// map[string]int errors on the bools and disabled detection entirely. Use a
	// typed struct so the int thresholds are read correctly.
	var thresholds struct {
		QCScanMax        int `json:"qc_scan_max"`
		WarehouseScanMax int `json:"warehouse_scan_max"`
	}
	if err := json.Unmarshal(setting.SettingValue, &thresholds); err != nil {
		return false, 0, 0
	}

	var maxCount int
	switch scanType {
	case "qc_scan":
		maxCount = thresholds.QCScanMax
	case "warehouse_scan":
		maxCount = thresholds.WarehouseScanMax
	default:
		return false, 0, 0
	}

	if maxCount == 0 {
		return false, 0, 0
	}

	var currentCount int64
	switch scanType {
	case "qc_scan":
		h.DB.Model(&models.QCScan{}).Where("qr_code_id = ?", qrCodeID).Count(&currentCount)
	case "warehouse_scan":
		h.DB.Model(&models.InventoryMovement{}).Where("qr_code_id = ?", qrCodeID).Count(&currentCount)
	}

	return int(currentCount) >= maxCount, int(currentCount), maxCount
}

// isUniqueViolation reports whether a DB error is a unique-constraint violation.
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "duplicate key") ||
		strings.Contains(msg, "SQLSTATE 23505") ||
		strings.Contains(msg, "uniq_counterfeit_detection_active")
}

// accumulateCounterfeitDetection folds a triggering interaction into an existing
// active detection (increments the count, records the interaction id, bumps the
// timestamp). Best-effort — logs on failure.
func accumulateCounterfeitDetection(db *gorm.DB, existing *models.CounterfeitDetection, interactionID uuid.UUID) {
	now := time.Now().UTC()
	ids := []string{}
	if len(existing.InteractionIDs) > 0 {
		_ = json.Unmarshal(existing.InteractionIDs, &ids)
	}
	ids = append(ids, interactionID.String())
	idsJSON, _ := json.Marshal(ids)
	if err := db.Model(existing).Updates(map[string]interface{}{
		"total_interactions_count": existing.TotalInteractionsCount + 1,
		"last_interaction_at":      now,
		"interaction_ids":          datatypes.JSON(idsJSON),
	}).Error; err != nil {
		fmt.Printf("[COUNTERFEIT] Failed to update detection count for QR %s: %v\n", existing.QRCodeID, err)
	}
}

// createCounterfeitDetection creates a counterfeit detection record
func createCounterfeitDetection(db *gorm.DB, tenantID, qrCodeID uuid.UUID, reason string, interactionID uuid.UUID) {
	// Check if detection already exists
	var existing models.CounterfeitDetection
	if err := db.Where("qr_code_id = ? AND status = ?", qrCodeID, "active").First(&existing).Error; err == nil {
		accumulateCounterfeitDetection(db, &existing, interactionID)
		return
	}

	// Create new detection
	now := time.Now().UTC()
	idsJSON, _ := json.Marshal([]string{interactionID.String()})
	detection := models.CounterfeitDetection{
		QRCodeID:               qrCodeID,
		TenantID:               tenantID,
		DetectionReason:        reason,
		TotalInteractionsCount: 1,
		FirstInteractionAt:     &now,
		LastInteractionAt:      &now,
		InteractionIDs:         datatypes.JSON(idsJSON),
		Status:                 "active",
	}
	if err := db.Create(&detection).Error; err != nil {
		// Lost the check-then-create race: another concurrent scan already inserted
		// the active detection (blocked by uniq_counterfeit_detection_active). Fall
		// back to accumulating into that existing row so no scan is dropped and no
		// duplicate active detection is created.
		if isUniqueViolation(err) {
			if err := db.Where("qr_code_id = ? AND status = ?", qrCodeID, "active").First(&existing).Error; err == nil {
				accumulateCounterfeitDetection(db, &existing, interactionID)
			}
			return
		}
		fmt.Printf("[COUNTERFEIT] Failed to create detection for QR %s: %v\n", qrCodeID, err)
		return
	}

	// Queue counterfeit alert notification for tenant admin
	var qrCode models.QRCode
	productName := "Unknown Product"
	if err := db.Preload("Batch.Product").First(&qrCode, "id = ?", qrCodeID).Error; err == nil {
		if qrCode.Batch != nil && qrCode.Batch.Product != nil {
			productName = qrCode.Batch.Product.ProductName
		}
	}

	Notify(db, tenantID, models.NotificationTypeCounterfeitAlert,
		"Potential counterfeit detected",
		fmt.Sprintf("%s — QR %s flagged: %s", productName, qrCode.QRCode, reason),
		"/tenant/counterfeit",
		map[string]interface{}{
			"product_name": productName,
			"qr_code":      qrCode.QRCode,
			"reason":       reason,
		})
}

// ===============================
// QC Scanning Endpoints
// ===============================

// QCScan performs a QC scan
func (h *ScanningHandler) QCScan(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}
	userUUID, ok := utils.GetUserUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "Invalid user context", nil)
		return
	}

	// Get staff ID from user
	var staff models.TenantStaff
	if err := h.DB.Where("user_id = ? AND tenant_id = ?", userUUID, tenantUUID).First(&staff).Error; err != nil {
		utils.ErrorResponse(c, http.StatusUnauthorized, "Staff not found", nil)
		return
	}
	staffID := staff.ID

	var input struct {
		LocationID  string  `json:"location_id" binding:"required"`
		QRCode      string  `json:"qr_code" binding:"required"`
		Status      string  `json:"status" binding:"required,oneof=pass failed"`
		Latitude    float64 `json:"latitude"`
		Longitude   float64 `json:"longitude"`
		// For re-scan (correction)
		IsCorrection     bool   `json:"is_correction"`
		CorrectsScanID   string `json:"corrects_scan_id"`
		CorrectionReason string `json:"correction_reason"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request", err)
		return
	}

	locationUUID, err := uuid.Parse(input.LocationID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid location ID", err)
		return
	}

	// Verify location belongs to tenant and is a QC area
	var location models.TenantLocation
	if err := h.DB.Where("id = ? AND tenant_id = ? AND location_type = ? AND deleted_at IS NULL",
		locationUUID, tenantUUID, "qc_area").First(&location).Error; err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "QC area not found or not accessible", nil)
		return
	}

	// Find QR code. Parse the scanned value first so a non-UUID code (a batch with a
	// prefix/suffix, or a Base58 short code) is never compared against the uuid column
	// qr_uuid — that raises "invalid input syntax for type uuid" and fails the whole
	// lookup with a spurious 404. Mirrors the public scan/validation handlers.
	lookup, err := utils.ParseQRCodeParam(input.QRCode)
	if err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "QR code not found", nil)
		return
	}
	var qrCode models.QRCode
	qrQuery := h.DB.Preload("Batch")
	if lookup.LookupByCode {
		qrQuery = qrQuery.Where("qr_code = ?", lookup.OriginalCode)
	} else {
		qrQuery = qrQuery.Where("qr_uuid = ?", lookup.QRUUID)
	}
	if err := qrQuery.First(&qrCode).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "QR code not found", nil)
		return
	}

	// Verify QR belongs to tenant AND its batch is not soft-deleted. QRBatch.DeletedAt
	// is a plain *time.Time (not gorm.DeletedAt), so Preload still returns soft-deleted
	// batches as non-nil — they must be rejected explicitly, matching the public
	// validation paths (validation.go). Recording scans against a deleted batch
	// corrupts analytics and blocks the batch from being re-deleted after restore.
	if qrCode.Batch == nil || qrCode.Batch.DeletedAt != nil || qrCode.Batch.TenantID != tenantUUID {
		utils.ErrorResponse(c, http.StatusForbidden, "QR code does not belong to this tenant", nil)
		return
	}

	// Check location validation
	var warnings []string
	if input.Latitude != 0 && input.Longitude != 0 {
		valid, distance := h.validateScanLocation(&location, input.Latitude, input.Longitude)
		if !valid {
			warnings = append(warnings, fmt.Sprintf("Scan location is %.0fm from allowed area", distance))
		}
	}

	// Check if this is a correction
	var correctsScanID *uuid.UUID
	if input.IsCorrection && input.CorrectsScanID != "" {
		corrID, err := uuid.Parse(input.CorrectsScanID)
		if err != nil {
			utils.ErrorResponse(c, http.StatusBadRequest, "Invalid corrects_scan_id", err)
			return
		}

		// Find the scan being corrected
		var originalScan models.QCScan
		if err := h.DB.Where("id = ? AND qr_code_id = ?", corrID, qrCode.ID).First(&originalScan).Error; err != nil {
			utils.ErrorResponse(c, http.StatusNotFound, "Original scan not found", nil)
			return
		}

		// Check if correction is within 5 minutes (no reason required)
		timeSinceOriginal := time.Since(originalScan.ScannedAt)
		if timeSinceOriginal > 5*time.Minute && input.CorrectionReason == "" {
			utils.ErrorResponse(c, http.StatusBadRequest, "Correction reason is required for scans older than 5 minutes", nil)
			return
		}

		correctsScanID = &corrID
	} else {
		// Check if QR was already scanned (not a correction)
		var existingScan models.QCScan
		if err := h.DB.Where("qr_code_id = ? AND is_correction = ?", qrCode.ID, false).
			Order("scanned_at DESC").First(&existingScan).Error; err == nil {
			// Already scanned - check if it's a re-scan
			if !input.IsCorrection {
				warnings = append(warnings, fmt.Sprintf("QR already scanned with status: %s", existingScan.QCStatus))
			}
		}
	}

	// Check counterfeit threshold
	exceeded, current, max := h.checkCounterfeitThreshold(tenantUUID, qrCode.ID, "qc_scan")

	// Check velocity anomaly (impossible travel)
	velocityExceeded, velocityReason := checkVelocityAnomalyShared(h.DB, tenantUUID, qrCode.ID, input.Latitude, input.Longitude)
	if velocityExceeded {
		warnings = append(warnings, "Counterfeit alert: Impossible travel detected")
	}

	// Build geolocation
	var scanGeo []byte
	if input.Latitude != 0 && input.Longitude != 0 {
		scanGeo = []byte(fmt.Sprintf(`{"lat":%f,"lng":%f}`, input.Latitude, input.Longitude))
	}

	// Create QC scan record
	qcScan := models.QCScan{
		LocationID:       &locationUUID,
		QRCodeID:         qrCode.ID,
		QCStatus:         models.QCStatus(input.Status),
		ScannedBy:        &staffID,
		ScanGeolocation:  scanGeo,
		IsCorrection:     input.IsCorrection,
		CorrectsScanID:   correctsScanID,
		CorrectionReason: input.CorrectionReason,
	}

	if err := h.DB.Create(&qcScan).Error; err != nil {
		sentry.CaptureHandlerError(c, err, "scanning.QCScan", sentry.ErrorTypeDatabase, sentry.SeverityCritical)
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to record QC scan", err)
		return
	}

	// Create interaction record. ScannedBy must be the user ID, not the
	// staff ID — interactions.scanned_by has an FK to users(id).
	interaction := models.Interaction{
		QRCodeID:               &qrCode.ID,
		TenantID:               tenantUUID,
		InteractionCategory:    models.InteractionCategoryTenantAccess,
		InteractionSubcategory: models.InteractionSubcategoryQCScan,
		InteractionStatus:      models.InteractionStatusSuccess,
		ScannedBy:              &userUUID,
		IPAddress:              c.ClientIP(), // inet column rejects the empty string
		Geolocation:            scanGeo,
	}
	if err := h.DB.Create(&interaction).Error; err != nil {
		sentry.CaptureHandlerError(c, err, "scanning.QCScan.interaction", sentry.ErrorTypeDatabase, sentry.SeverityHigh)
	}

	// Handle counterfeit detection
	counterfeitDetected := false
	if exceeded {
		reason := fmt.Sprintf("QC scan count (%d) exceeded threshold (%d)", current+1, max)
		createCounterfeitDetection(h.DB, tenantUUID, qrCode.ID, reason, interaction.ID)
		warnings = append(warnings, "Counterfeit alert: Scan threshold exceeded")
		counterfeitDetected = true
	}

	if velocityExceeded {
		createCounterfeitDetection(h.DB, tenantUUID, qrCode.ID, velocityReason, interaction.ID)
		counterfeitDetected = true
	}

	// Load relations for response
	h.DB.Preload("Location").Preload("QRCode.Batch.Product").Preload("ScannedByStaff").First(&qcScan, qcScan.ID)

	response := gin.H{
		"scan":     qcScan,
		"warnings": warnings,
	}

	if counterfeitDetected {
		response["counterfeit_alert"] = true
	}

	utils.SuccessResponse(c, http.StatusCreated, "QC scan recorded", response)
}

// GetQCHistory returns QC scan history for a tenant
func (h *ScanningHandler) GetQCHistory(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}

	page := utils.GetPageQuery(c)
	limit := utils.GetLimitQuery(c, 20)
	offset := (page - 1) * limit

	status := c.Query("status") // pass, failed, or empty for all
	locationID := c.Query("location_id")

	query := h.DB.Model(&models.QCScan{}).
		Joins("JOIN qr_codes ON qr_codes.id = qc_scans.qr_code_id").
		Joins("JOIN qr_batches ON qr_batches.id = qr_codes.batch_id").
		Where("qr_batches.tenant_id = ?", tenantUUID)

	if status != "" {
		query = query.Where("qc_scans.qc_status = ?", status)
	}
	if locationID != "" {
		query = query.Where("qc_scans.location_id = ?", locationID)
	}

	var total int64
	query.Count(&total)

	var scans []models.QCScan
	query.Preload("Location").Preload("QRCode.Batch.Product").Preload("ScannedByStaff.User").
		Order("qc_scans.scanned_at DESC").
		Offset(offset).Limit(limit).
		Find(&scans)

	utils.SuccessResponse(c, http.StatusOK, "QC history retrieved", gin.H{
		"scans": scans,
		"pagination": utils.PaginationMeta(page, limit, total),
	})
}

// ===============================
// Warehouse Scanning Endpoints
// ===============================

// WarehouseScan performs a warehouse in/out scan
func (h *ScanningHandler) WarehouseScan(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}
	userUUID, ok := utils.GetUserUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusUnauthorized, "Invalid user context", nil)
		return
	}

	// Get staff ID from user
	var staff models.TenantStaff
	if err := h.DB.Where("user_id = ? AND tenant_id = ?", userUUID, tenantUUID).First(&staff).Error; err != nil {
		utils.ErrorResponse(c, http.StatusUnauthorized, "Staff not found", nil)
		return
	}
	staffID := staff.ID

	var input struct {
		LocationID   string  `json:"location_id" binding:"required"`
		QRCode       string  `json:"qr_code" binding:"required"`
		MovementType string  `json:"movement_type" binding:"required,oneof=in out"`
		Latitude     float64 `json:"latitude"`
		Longitude    float64 `json:"longitude"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request", err)
		return
	}

	locationUUID, err := uuid.Parse(input.LocationID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid location ID", err)
		return
	}

	// Verify location belongs to tenant and is a warehouse
	var location models.TenantLocation
	if err := h.DB.Where("id = ? AND tenant_id = ? AND location_type = ? AND deleted_at IS NULL",
		locationUUID, tenantUUID, "warehouse").First(&location).Error; err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Warehouse not found or not accessible", nil)
		return
	}

	// Find QR code. Parse the scanned value first so a non-UUID code (a batch with a
	// prefix/suffix, or a Base58 short code) is never compared against the uuid column
	// qr_uuid — that raises "invalid input syntax for type uuid" and fails the whole
	// lookup with a spurious 404. Mirrors the public scan/validation handlers.
	lookup, err := utils.ParseQRCodeParam(input.QRCode)
	if err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "QR code not found", nil)
		return
	}
	var qrCode models.QRCode
	qrQuery := h.DB.Preload("Batch")
	if lookup.LookupByCode {
		qrQuery = qrQuery.Where("qr_code = ?", lookup.OriginalCode)
	} else {
		qrQuery = qrQuery.Where("qr_uuid = ?", lookup.QRUUID)
	}
	if err := qrQuery.First(&qrCode).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "QR code not found", nil)
		return
	}

	// Verify QR belongs to tenant AND its batch is not soft-deleted. QRBatch.DeletedAt
	// is a plain *time.Time (not gorm.DeletedAt), so Preload still returns soft-deleted
	// batches as non-nil — they must be rejected explicitly, matching the public
	// validation paths (validation.go). Recording scans against a deleted batch
	// corrupts analytics and blocks the batch from being re-deleted after restore.
	if qrCode.Batch == nil || qrCode.Batch.DeletedAt != nil || qrCode.Batch.TenantID != tenantUUID {
		utils.ErrorResponse(c, http.StatusForbidden, "QR code does not belong to this tenant", nil)
		return
	}

	// Check location validation
	var warnings []string
	if input.Latitude != 0 && input.Longitude != 0 {
		valid, distance := h.validateScanLocation(&location, input.Latitude, input.Longitude)
		if !valid {
			warnings = append(warnings, fmt.Sprintf("Scan location is %.0fm from allowed area", distance))
		}
	}

	// Check last movement for this QR at this location
	var lastMovement models.InventoryMovement
	if err := h.DB.Where("qr_code_id = ? AND location_id = ?", qrCode.ID, locationUUID).
		Order("scanned_at DESC").First(&lastMovement).Error; err == nil {
		if string(lastMovement.MovementType) == input.MovementType {
			if input.MovementType == "out" {
				warnings = append(warnings, "Product already marked as OUT from this location")
			} else {
				warnings = append(warnings, "Product already marked as IN at this location")
			}
		}
	}

	// Check counterfeit threshold
	exceeded, current, max := h.checkCounterfeitThreshold(tenantUUID, qrCode.ID, "warehouse_scan")

	// Check velocity anomaly (impossible travel)
	velocityExceeded, velocityReason := checkVelocityAnomalyShared(h.DB, tenantUUID, qrCode.ID, input.Latitude, input.Longitude)
	if velocityExceeded {
		warnings = append(warnings, "Counterfeit alert: Impossible travel detected")
	}

	// Build geolocation
	var scanGeo []byte
	if input.Latitude != 0 && input.Longitude != 0 {
		scanGeo = []byte(fmt.Sprintf(`{"lat":%f,"lng":%f}`, input.Latitude, input.Longitude))
	}

	// Create inventory movement record
	movement := models.InventoryMovement{
		LocationID:      locationUUID,
		QRCodeID:        qrCode.ID,
		MovementType:    models.MovementType(input.MovementType),
		ScannedBy:       &staffID,
		ScanGeolocation: scanGeo,
	}

	if err := h.DB.Create(&movement).Error; err != nil {
		sentry.CaptureHandlerError(c, err, "scanning.WarehouseScan", sentry.ErrorTypeDatabase, sentry.SeverityCritical)
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to record warehouse scan", err)
		return
	}

	// Create interaction record. ScannedBy must be the user ID, not the
	// staff ID — interactions.scanned_by has an FK to users(id).
	interaction := models.Interaction{
		QRCodeID:               &qrCode.ID,
		TenantID:               tenantUUID,
		InteractionCategory:    models.InteractionCategoryTenantAccess,
		InteractionSubcategory: models.InteractionSubcategoryWarehouseScan,
		InteractionStatus:      models.InteractionStatusSuccess,
		ScannedBy:              &userUUID,
		IPAddress:              c.ClientIP(), // inet column rejects the empty string
		Geolocation:            scanGeo,
	}
	if err := h.DB.Create(&interaction).Error; err != nil {
		sentry.CaptureHandlerError(c, err, "scanning.WarehouseScan.interaction", sentry.ErrorTypeDatabase, sentry.SeverityHigh)
	}

	// Handle counterfeit detection
	counterfeitDetected := false
	if exceeded {
		reason := fmt.Sprintf("Warehouse scan count (%d) exceeded threshold (%d)", current+1, max)
		createCounterfeitDetection(h.DB, tenantUUID, qrCode.ID, reason, interaction.ID)
		warnings = append(warnings, "Counterfeit alert: Scan threshold exceeded")
		counterfeitDetected = true
	}

	if velocityExceeded {
		createCounterfeitDetection(h.DB, tenantUUID, qrCode.ID, velocityReason, interaction.ID)
		counterfeitDetected = true
	}

	// Load relations for response
	h.DB.Preload("Location").Preload("QRCode.Batch.Product").Preload("ScannedByStaff").First(&movement, movement.ID)

	response := gin.H{
		"movement": movement,
		"warnings": warnings,
	}

	if counterfeitDetected {
		response["counterfeit_alert"] = true
	}

	utils.SuccessResponse(c, http.StatusCreated, "Warehouse scan recorded", response)
}

// GetWarehouseHistory returns warehouse movement history for a tenant
func (h *ScanningHandler) GetWarehouseHistory(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}

	page := utils.GetPageQuery(c)
	limit := utils.GetLimitQuery(c, 20)
	offset := (page - 1) * limit

	movementType := c.Query("type") // in, out, or empty for all
	locationID := c.Query("location_id")

	query := h.DB.Model(&models.InventoryMovement{}).
		Joins("JOIN qr_codes ON qr_codes.id = inventory_movements.qr_code_id").
		Joins("JOIN qr_batches ON qr_batches.id = qr_codes.batch_id").
		Where("qr_batches.tenant_id = ?", tenantUUID)

	if movementType != "" {
		query = query.Where("inventory_movements.movement_type = ?", movementType)
	}
	if locationID != "" {
		query = query.Where("inventory_movements.location_id = ?", locationID)
	}

	var total int64
	query.Count(&total)

	var movements []models.InventoryMovement
	query.Preload("Location").Preload("QRCode.Batch.Product").Preload("ScannedByStaff.User").
		Order("inventory_movements.scanned_at DESC").
		Offset(offset).Limit(limit).
		Find(&movements)

	utils.SuccessResponse(c, http.StatusOK, "Warehouse history retrieved", gin.H{
		"movements": movements,
		"pagination": utils.PaginationMeta(page, limit, total),
	})
}

// GetInventoryStock returns current stock at a warehouse
func (h *ScanningHandler) GetInventoryStock(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}
	locationID := c.Query("location_id") // Optional: filter by specific location

	// Get all QR codes with their last movement at warehouse locations
	var stockItems []struct {
		QRCodeID         uuid.UUID `json:"qr_code_id"`
		QRCode           string    `json:"qr_code"`
		ProductName      string    `json:"product_name"`
		BatchCode        string    `json:"batch_code"`
		LocationName     string    `json:"location_name"`
		LastMovementType string    `json:"last_movement_type"`
		LastMovementAt   time.Time `json:"last_movement_at"`
	}

	var query string
	var args []interface{}

	if locationID != "" {
		// Query for specific location
		query = `
			WITH latest_movements AS (
				SELECT DISTINCT ON (qr_code_id)
					qr_code_id, location_id, movement_type, scanned_at
				FROM inventory_movements
				WHERE location_id = ?
				ORDER BY qr_code_id, scanned_at DESC
			)
			SELECT
				qc.id as qr_code_id,
				qc.qr_code,
				p.product_name,
				qb.batch_code,
				tl.location_name,
				lm.movement_type as last_movement_type,
				lm.scanned_at as last_movement_at
			FROM latest_movements lm
			JOIN qr_codes qc ON qc.id = lm.qr_code_id
			JOIN qr_batches qb ON qb.id = qc.batch_id
			JOIN products p ON p.id = qb.product_id
			JOIN tenant_locations tl ON tl.id = lm.location_id
			WHERE qb.tenant_id = ? AND lm.movement_type = 'in'
			ORDER BY lm.scanned_at DESC
		`
		args = []interface{}{locationID, tenantUUID}
	} else {
		// Query for all warehouse locations
		query = `
			WITH warehouse_locations AS (
				SELECT id FROM tenant_locations
				WHERE tenant_id = ? AND location_type = 'warehouse' AND deleted_at IS NULL
			),
			latest_movements AS (
				SELECT DISTINCT ON (qr_code_id)
					qr_code_id, location_id, movement_type, scanned_at
				FROM inventory_movements
				WHERE location_id IN (SELECT id FROM warehouse_locations)
				ORDER BY qr_code_id, scanned_at DESC
			)
			SELECT
				qc.id as qr_code_id,
				qc.qr_code,
				p.product_name,
				qb.batch_code,
				tl.location_name,
				lm.movement_type as last_movement_type,
				lm.scanned_at as last_movement_at
			FROM latest_movements lm
			JOIN qr_codes qc ON qc.id = lm.qr_code_id
			JOIN qr_batches qb ON qb.id = qc.batch_id
			JOIN products p ON p.id = qb.product_id
			JOIN tenant_locations tl ON tl.id = lm.location_id
			WHERE qb.tenant_id = ? AND lm.movement_type = 'in'
			ORDER BY lm.scanned_at DESC
		`
		args = []interface{}{tenantUUID, tenantUUID}
	}

	h.DB.Raw(query, args...).Scan(&stockItems)

	utils.SuccessResponse(c, http.StatusOK, "Inventory stock retrieved", gin.H{
		"stock_count": len(stockItems),
		"items":       stockItems,
	})
}

// GetQCLocations returns QC area locations for dropdown
func (h *ScanningHandler) GetQCLocations(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}

	var locations []models.TenantLocation
	h.DB.Where("tenant_id = ? AND location_type = ? AND status = ? AND deleted_at IS NULL",
		tenantUUID, "qc_area", "active").
		Order("location_name").
		Find(&locations)

	utils.SuccessResponse(c, http.StatusOK, "QC locations retrieved", locations)
}

// GetWarehouseLocations returns warehouse locations for dropdown
func (h *ScanningHandler) GetWarehouseLocations(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}

	var locations []models.TenantLocation
	h.DB.Where("tenant_id = ? AND location_type = ? AND status = ? AND deleted_at IS NULL",
		tenantUUID, "warehouse", "active").
		Order("location_name").
		Find(&locations)

	utils.SuccessResponse(c, http.StatusOK, "Warehouse locations retrieved", locations)
}
