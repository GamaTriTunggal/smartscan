package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/gamatritunggal/smartscan/backend/internal/config"
	"github.com/gamatritunggal/smartscan/backend/internal/database"
	"github.com/gamatritunggal/smartscan/backend/internal/models"
	"github.com/gamatritunggal/smartscan/backend/internal/services/audit"
	"github.com/gamatritunggal/smartscan/backend/internal/storage"
	"github.com/gamatritunggal/smartscan/backend/internal/utils"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type CounterfeitHandler struct {
	DB  *gorm.DB
	Cfg *config.Config
}

func NewCounterfeitHandler(db *gorm.DB, cfg *config.Config) *CounterfeitHandler {
	return &CounterfeitHandler{DB: db, Cfg: cfg}
}

// ListCounterfeitDetections returns counterfeit detections for a tenant
func (h *CounterfeitHandler) ListCounterfeitDetections(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}

	page := utils.GetPageQuery(c)
	limit := utils.GetLimitQuery(c, 20)
	offset := (page - 1) * limit

	status := c.Query("status") // active, false_positive, or empty for all
	search := c.Query("search") // Search by QR code or product name

	query := h.DB.Model(&models.CounterfeitDetection{}).
		Where("counterfeit_detections.tenant_id = ?", tenantUUID)

	if status != "" {
		query = query.Where("counterfeit_detections.status = ?", status)
	}

	if search != "" {
		query = query.Joins("JOIN qr_codes ON qr_codes.id = counterfeit_detections.qr_code_id").
			Joins("JOIN qr_batches ON qr_batches.id = qr_codes.batch_id").
			Joins("JOIN products ON products.id = qr_batches.product_id").
			Where("qr_codes.qr_code ILIKE ? OR products.product_name ILIKE ?",
				"%"+search+"%", "%"+search+"%")
	}

	// Date range filter
	if from := c.Query("from"); from != "" {
		if t, err := time.Parse("2006-01-02", from); err == nil {
			query = query.Where("counterfeit_detections.created_at >= ?", t)
		}
	}
	if to := c.Query("to"); to != "" {
		if t, err := time.Parse("2006-01-02", to); err == nil {
			query = query.Where("counterfeit_detections.created_at < ?", t.AddDate(0, 0, 1))
		}
	}

	var total int64
	query.Count(&total)

	var detections []models.CounterfeitDetection
	query.Preload("QRCode.Batch.Product").
		Preload("ResolvedByStaff.User").
		Order("counterfeit_detections.created_at DESC").
		Offset(offset).Limit(limit).
		Find(&detections)

	utils.SuccessResponse(c, http.StatusOK, "Counterfeit detections retrieved", gin.H{
		"detections":  detections,
		"page":        page,
		"limit":       limit,
		"total":       total,
		"total_pages": (total + int64(limit) - 1) / int64(limit),
	})
}

// GetCounterfeitDetection returns a single counterfeit detection with interaction history
func (h *CounterfeitHandler) GetCounterfeitDetection(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}
	detectionID := c.Param("id")

	detectionUUID, err := uuid.Parse(detectionID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid detection ID", err)
		return
	}

	var detection models.CounterfeitDetection
	if err := h.DB.Preload("QRCode.Batch.Product").
		Preload("ResolvedByStaff.User").
		Preload("Tenant").
		Where("id = ? AND tenant_id = ?", detectionUUID, tenantUUID).
		First(&detection).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.ErrorResponse(c, http.StatusNotFound, "Counterfeit detection not found", nil)
			return
		}
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to get counterfeit detection", err)
		return
	}

	// Get all interactions for this QR code
	var interactions []models.Interaction
	h.DB.Where("qr_code_id = ?", detection.QRCodeID).
		Order("created_at ASC").
		Find(&interactions)

	// Calculate velocity data (distance and time between consecutive scans)
	var velocityData []gin.H
	for i := 1; i < len(interactions); i++ {
		prev := interactions[i-1]
		curr := interactions[i]

		// Parse geolocations (ignore errors - corrupted data should not break velocity calculation)
		var prevGeo, currGeo Geolocation
		if prev.Geolocation != nil {
			_ = json.Unmarshal(prev.Geolocation, &prevGeo)
		}
		if curr.Geolocation != nil {
			_ = json.Unmarshal(curr.Geolocation, &currGeo)
		}

		// Calculate distance and time
		var distance float64
		if prevGeo.Lat != 0 && prevGeo.Lng != 0 && currGeo.Lat != 0 && currGeo.Lng != 0 {
			distance = calculateDistanceHaversine(prevGeo.Lat, prevGeo.Lng, currGeo.Lat, currGeo.Lng)
		}

		timeDiff := curr.CreatedAt.Sub(prev.CreatedAt)
		var speed float64
		if timeDiff.Seconds() > 0 && distance > 0 {
			speed = distance / timeDiff.Seconds() // meters per second
		}

		velocityData = append(velocityData, gin.H{
			"from_time":         prev.CreatedAt,
			"to_time":           curr.CreatedAt,
			"from_location":     prevGeo,
			"to_location":       currGeo,
			"distance_meters":   distance,
			"time_seconds":      timeDiff.Seconds(),
			"speed_mps":         speed,
			"speed_kmh":         speed * 3.6,
			"is_impossible":     speed > 277.78, // > 1000 km/h is impossible
			"interaction_index": i,
		})
	}

	utils.SuccessResponse(c, http.StatusOK, "Counterfeit detection retrieved", gin.H{
		"detection":     detection,
		"interactions":  interactions,
		"velocity_data": velocityData,
	})
}

// MarkAsFalsePositive marks a counterfeit detection as false positive
func (h *CounterfeitHandler) MarkAsFalsePositive(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}
	userUUID, ok := utils.GetUserUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid user context", nil)
		return
	}
	detectionID := c.Param("id")

	detectionUUID, err := uuid.Parse(detectionID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid detection ID", err)
		return
	}

	var input struct {
		Reason string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Reason is required", err)
		return
	}

	// Get staff ID from user
	var staff models.TenantStaff
	if err := h.DB.Where("user_id = ? AND tenant_id = ?", userUUID, tenantUUID).First(&staff).Error; err != nil {
		utils.ErrorResponse(c, http.StatusUnauthorized, "Staff not found", nil)
		return
	}

	var detection models.CounterfeitDetection
	if err := h.DB.Where("id = ? AND tenant_id = ?", detectionUUID, tenantUUID).First(&detection).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.ErrorResponse(c, http.StatusNotFound, "Counterfeit detection not found", nil)
			return
		}
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to get counterfeit detection", err)
		return
	}

	if detection.Status != models.CounterfeitDetectionStatusActive {
		utils.ErrorResponse(c, http.StatusBadRequest, "Detection must be active to perform this action", nil)
		return
	}

	now := time.Now().UTC()
	newReason := detection.DetectionReason + " | False positive: " + input.Reason

	tx := h.DB.Begin()

	if err := tx.Model(&detection).Updates(map[string]interface{}{
		"status":           models.CounterfeitDetectionStatusFalsePositive,
		"resolved_by":      staff.ID,
		"resolved_at":      now,
		"detection_reason": newReason,
	}).Error; err != nil {
		tx.Rollback()
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to mark as false positive", err)
		return
	}

	// Inline AfterUpdate hook: sync QR code counterfeit_status to "valid"
	if err := tx.Model(&models.QRCode{}).Where("id = ?", detection.QRCodeID).
		Update("counterfeit_status", models.CounterfeitStatusValid).Error; err != nil {
		tx.Rollback()
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to sync QR code counterfeit status", err)
		return
	}

	if err := tx.Commit().Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to commit false positive update", err)
		return
	}

	h.DB.Preload("QRCode.Batch.Product").Preload("ResolvedByStaff.User").First(&detection, detection.ID)

	utils.SuccessResponse(c, http.StatusOK, "Marked as false positive", detection)
}

// OverrideThreshold overrides the counterfeit scan threshold at QR, batch, or product level
// and marks the detection as false positive. This is the main action for handling false positives
// caused by store displays, warehouse scanning, etc.
func (h *CounterfeitHandler) OverrideThreshold(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}
	userUUID, ok := utils.GetUserUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid user context", nil)
		return
	}

	detectionID := c.Param("id")
	detectionUUID, err := uuid.Parse(detectionID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid detection ID", err)
		return
	}

	var input struct {
		Level        string `json:"level" binding:"required,oneof=qr batch product"`
		NewThreshold int    `json:"new_threshold" binding:"required,min=1"`
		Reason       string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request: level (qr/batch/product), new_threshold (>= 1), and reason are required", err)
		return
	}

	// Get staff
	var staff models.TenantStaff
	if err := h.DB.Where("user_id = ? AND tenant_id = ?", userUUID, tenantUUID).First(&staff).Error; err != nil {
		utils.ErrorResponse(c, http.StatusUnauthorized, "Staff not found", nil)
		return
	}

	// Get detection with QR > Batch > Product preloaded
	var detection models.CounterfeitDetection
	if err := h.DB.Preload("QRCode.Batch.Product").
		Where("id = ? AND tenant_id = ?", detectionUUID, tenantUUID).
		First(&detection).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.ErrorResponse(c, http.StatusNotFound, "Counterfeit detection not found", nil)
			return
		}
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to get detection", err)
		return
	}

	if detection.Status != models.CounterfeitDetectionStatusActive {
		utils.ErrorResponse(c, http.StatusBadRequest, "Detection must be active to perform this action", nil)
		return
	}

	// Validate threshold > current scan count (use detection's stored count, works for all scan types)
	scanCount := detection.TotalInteractionsCount

	if input.NewThreshold <= scanCount {
		utils.ErrorResponse(c, http.StatusBadRequest,
			fmt.Sprintf("New threshold (%d) must be greater than current scan count (%d)", input.NewThreshold, scanCount), nil)
		return
	}

	// Capture old threshold for audit trail
	oldThreshold := ResolveCounterfeitThreshold(h.DB, detection.QRCode)

	tx := h.DB.Begin()

	// Apply threshold override based on level
	var entityType string
	var entityID uuid.UUID
	switch input.Level {
	case "qr":
		entityType = "qr_code"
		entityID = detection.QRCodeID
		if err := tx.Model(&models.QRCode{}).Where("id = ?", detection.QRCodeID).
			Update("counterfeit_scan_max", input.NewThreshold).Error; err != nil {
			tx.Rollback()
			utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to update QR threshold", err)
			return
		}
	case "batch":
		if detection.QRCode == nil || detection.QRCode.Batch == nil {
			tx.Rollback()
			utils.ErrorResponse(c, http.StatusBadRequest, "QR code batch not found", nil)
			return
		}
		entityType = "qr_batch"
		entityID = detection.QRCode.BatchID
		if err := tx.Model(&models.QRBatch{}).Where("id = ?", detection.QRCode.BatchID).
			Update("counterfeit_scan_max", input.NewThreshold).Error; err != nil {
			tx.Rollback()
			utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to update batch threshold", err)
			return
		}
	case "product":
		if detection.QRCode == nil || detection.QRCode.Batch == nil || detection.QRCode.Batch.Product == nil {
			tx.Rollback()
			utils.ErrorResponse(c, http.StatusBadRequest, "Product not found", nil)
			return
		}
		entityType = "product"
		entityID = detection.QRCode.Batch.ProductID
		if err := tx.Model(&models.Product{}).Where("id = ?", detection.QRCode.Batch.ProductID).
			Update("counterfeit_scan_max", input.NewThreshold).Error; err != nil {
			tx.Rollback()
			utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to update product threshold", err)
			return
		}
	}

	// Mark detection as false positive
	now := time.Now().UTC()
	newReason := detection.DetectionReason + " | Threshold override (" + input.Level + "): " + input.Reason
	if err := tx.Model(&detection).Updates(map[string]interface{}{
		"status":           models.CounterfeitDetectionStatusFalsePositive,
		"resolved_by":      staff.ID,
		"resolved_at":      now,
		"detection_reason": newReason,
	}).Error; err != nil {
		tx.Rollback()
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to update detection", err)
		return
	}

	// Reset QR counterfeit_status to valid
	if err := tx.Model(&models.QRCode{}).Where("id = ?", detection.QRCodeID).
		Update("counterfeit_status", models.CounterfeitStatusValid).Error; err != nil {
		tx.Rollback()
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to reset QR status", err)
		return
	}

	if err := tx.Commit().Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to commit override", err)
		return
	}

	// Audit log
	audit.LogFromContext(c, h.DB, models.ActionTypeThresholdOverride, entityType, &entityID, map[string]interface{}{
		"detection_id":  detection.ID,
		"level":         input.Level,
		"old_threshold": oldThreshold,
		"scan_count":    scanCount,
	}, map[string]interface{}{
		"new_threshold": input.NewThreshold,
		"reason":        input.Reason,
	})

	// Reload for response
	h.DB.Preload("QRCode.Batch.Product").Preload("ResolvedByStaff.User").First(&detection, detection.ID)

	utils.SuccessResponse(c, http.StatusOK, "Threshold overridden and detection resolved", gin.H{
		"detection":     detection,
		"level":         input.Level,
		"new_threshold": input.NewThreshold,
	})
}




// GetCounterfeitStats returns counterfeit statistics for dashboard
func (h *CounterfeitHandler) GetCounterfeitStats(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}

	// Build base query with optional date range filter
	baseQuery := h.DB.Model(&models.CounterfeitDetection{}).Where("tenant_id = ?", tenantUUID)
	if from := c.Query("from"); from != "" {
		if t, err := time.Parse("2006-01-02", from); err == nil {
			baseQuery = baseQuery.Where("created_at >= ?", t)
		}
	}
	if to := c.Query("to"); to != "" {
		if t, err := time.Parse("2006-01-02", to); err == nil {
			baseQuery = baseQuery.Where("created_at < ?", t.AddDate(0, 0, 1))
		}
	}

	var activeCount int64
	var falsePositiveCount int64
	var totalCount int64

	baseQuery.Session(&gorm.Session{}).Where("status = ?", "active").Count(&activeCount)
	baseQuery.Session(&gorm.Session{}).Where("status = ?", "false_positive").Count(&falsePositiveCount)
	baseQuery.Session(&gorm.Session{}).Count(&totalCount)

	// Recent detections (last 7 days within the date range)
	sevenDaysAgo := time.Now().UTC().AddDate(0, 0, -7)
	var recentCount int64
	baseQuery.Session(&gorm.Session{}).Where("created_at >= ?", sevenDaysAgo).Count(&recentCount)

	// Detection by reason
	type ReasonStat struct {
		Reason string `json:"reason"`
		Count  int64  `json:"count"`
	}
	var reasonStats []ReasonStat
	baseQuery.Session(&gorm.Session{}).
		Select("CASE WHEN detection_reason LIKE '%velocity%' OR detection_reason LIKE '%impossible%' THEN 'velocity_anomaly' WHEN detection_reason LIKE '%threshold%' THEN 'threshold_exceeded' ELSE 'other' END as reason, COUNT(*) as count").
		Group("reason").
		Scan(&reasonStats)

	utils.SuccessResponse(c, http.StatusOK, "Counterfeit stats retrieved", gin.H{
		"active":         activeCount,
		"false_positive": falsePositiveCount,
		"total":          totalCount,
		"recent_7_days":  recentCount,
		"by_reason":      reasonStats,
	})
}

// GetCounterfeitSettings returns the counterfeit threshold settings
func (h *CounterfeitHandler) GetCounterfeitSettings(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}

	var setting models.TenantSettings
	if err := h.DB.Where("tenant_id = ? AND setting_key = ?", tenantUUID, "counterfeit_thresholds").First(&setting).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// Return default settings
			utils.SuccessResponse(c, http.StatusOK, "Counterfeit settings retrieved", gin.H{
				"qc_scan_max":               0,
				"warehouse_scan_max":        0,
				"end_user_scan_max":         3,
				"velocity_check_enabled":    false,
				"max_speed_kmh":             1000,
				"alert_on_detection":        true,
				"auto_flag_suspicious":      true,
			})
			return
		}
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to get settings", err)
		return
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(setting.SettingValue, &settings); err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to parse settings", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Counterfeit settings retrieved", settings)
}

// UpdateCounterfeitSettings updates the counterfeit threshold settings
func (h *CounterfeitHandler) UpdateCounterfeitSettings(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}

	var input struct {
		QCScanMax            int  `json:"qc_scan_max"`
		WarehouseScanMax     int  `json:"warehouse_scan_max"`
		EndUserScanMax       int  `json:"end_user_scan_max"`
		VelocityCheckEnabled bool `json:"velocity_check_enabled"`
		MaxSpeedKmh          int  `json:"max_speed_kmh"`
		AlertOnDetection     bool `json:"alert_on_detection"`
		AutoFlagSuspicious   bool `json:"auto_flag_suspicious"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request", err)
		return
	}

	settingValue, err := json.Marshal(map[string]interface{}{
		"qc_scan_max":            input.QCScanMax,
		"warehouse_scan_max":     input.WarehouseScanMax,
		"end_user_scan_max":      input.EndUserScanMax,
		"velocity_check_enabled": input.VelocityCheckEnabled,
		"max_speed_kmh":          input.MaxSpeedKmh,
		"alert_on_detection":     input.AlertOnDetection,
		"auto_flag_suspicious":   input.AutoFlagSuspicious,
	})
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to serialize settings", err)
		return
	}

	var setting models.TenantSettings
	if err := h.DB.Where("tenant_id = ? AND setting_key = ?", tenantUUID, "counterfeit_thresholds").First(&setting).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// Create new setting
			setting = models.TenantSettings{
				TenantID:     tenantUUID,
				SettingKey:   "counterfeit_thresholds",
				SettingValue: settingValue,
			}
			if err := h.DB.Create(&setting).Error; err != nil {
				utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to create settings", err)
				return
			}
			utils.SuccessResponse(c, http.StatusCreated, "Counterfeit settings created", input)
			return
		}
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to get settings", err)
		return
	}

	if err := h.DB.Model(&setting).Update("setting_value", datatypes.JSON(settingValue)).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to update settings", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Counterfeit settings updated", input)
}

// GetCounterfeitGeolocations returns geolocation data for map visualization
func (h *CounterfeitHandler) GetCounterfeitGeolocations(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}
	detectionID := c.Param("id")

	detectionUUID, err := uuid.Parse(detectionID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid detection ID", err)
		return
	}

	var detection models.CounterfeitDetection
	if err := h.DB.Where("id = ? AND tenant_id = ?", detectionUUID, tenantUUID).First(&detection).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			utils.ErrorResponse(c, http.StatusNotFound, "Counterfeit detection not found", nil)
			return
		}
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to get counterfeit detection", err)
		return
	}

	// Get all interactions for this QR code with geolocation
	var interactions []models.Interaction
	h.DB.Where("qr_code_id = ? AND geolocation IS NOT NULL", detection.QRCodeID).
		Order("created_at ASC").
		Find(&interactions)

	var geopoints []gin.H
	for i, interaction := range interactions {
		var geo Geolocation
		if interaction.Geolocation != nil {
			if err := json.Unmarshal(interaction.Geolocation, &geo); err == nil && geo.Lat != 0 && geo.Lng != 0 {
				geopoints = append(geopoints, gin.H{
					"index":       i + 1,
					"lat":         geo.Lat,
					"lng":         geo.Lng,
					"timestamp":   interaction.CreatedAt,
					"category":    interaction.InteractionCategory,
					"subcategory": interaction.InteractionSubcategory,
					"status":      interaction.InteractionStatus,
				})
			}
		}
	}

	utils.SuccessResponse(c, http.StatusOK, "Geolocations retrieved", gin.H{
		"detection_id": detection.ID,
		"qr_code_id":   detection.QRCodeID,
		"geopoints":    geopoints,
		"total_points": len(geopoints),
	})
}

// Helper function for Haversine distance calculation
func calculateDistanceHaversine(lat1, lng1, lat2, lng2 float64) float64 {
	const earthRadius = 6371000 // meters

	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	deltaLat := (lat2 - lat1) * math.Pi / 180
	deltaLng := (lng2 - lng1) * math.Pi / 180

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(deltaLng/2)*math.Sin(deltaLng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadius * c
}

// =============================================================================
// COUNTERFEIT REPORTS (End-user submissions)
// =============================================================================

// SubmitCounterfeitReport - Public endpoint for end-users to report counterfeit products
// Accepts multipart form with text fields + optional photo uploads (max 5)
func (h *CounterfeitHandler) SubmitCounterfeitReport(c *gin.Context) {
	// Parse multipart form (32MB max memory)
	if err := c.Request.ParseMultipartForm(32 << 20); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid form data", err)
		return
	}

	// Read text fields
	qrCodeParam := c.PostForm("qr_code")
	if qrCodeParam == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "QR code is required", nil)
		return
	}
	description := c.PostForm("description")
	storeName := c.PostForm("store_name")

	// Parse QR code (supports Base58, UUID, and Hex formats)
	lookup, err := utils.ParseQRCodeParam(qrCodeParam)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid QR code", err)
		return
	}

	// Find QR code and verify it exists
	var qrCode models.QRCode
	query := h.DB.Preload("Batch")
	if lookup.LookupByCode {
		query = query.Where("qr_code = ?", lookup.OriginalCode)
	} else {
		query = query.Where("qr_uuid = ?", lookup.QRUUID)
	}
	if err := query.First(&qrCode).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "QR code not found", err)
		return
	}

	// Check if QR code is actually flagged as counterfeit
	if qrCode.CounterfeitStatus != models.CounterfeitStatusCounterfeit {
		utils.ErrorResponse(c, http.StatusBadRequest, "This product is not flagged as counterfeit", nil)
		return
	}

	// Find active counterfeit detection for this QR code
	var detection models.CounterfeitDetection
	var detectionID *uuid.UUID
	if err := h.DB.Where("qr_code_id = ? AND status = ?", qrCode.ID, models.CounterfeitDetectionStatusActive).
		First(&detection).Error; err == nil {
		detectionID = &detection.ID
	}

	// Rate limit: 1 report per IP per QR per hour
	if h.checkReportRateLimit(c, qrCode.ID.String()) {
		return
	}

	// Process photo uploads (optional, max 5)
	var photoURLs []string
	var uploadedKeys []string // for cleanup on failure
	files := c.Request.MultipartForm.File["photos"]
	if len(files) > 5 {
		utils.ErrorResponse(c, http.StatusBadRequest, "Maximum 5 photos allowed", nil)
		return
	}

	reportID := uuid.New().String()
	tenantID := qrCode.Batch.TenantID.String()
	uploadOpts := utils.ImageUploadOptions{
		MaxFileSize:  5 * 1024 * 1024, // 5MB per photo
		MinDimension: 0,               // No dimension requirement for evidence photos
		AllowedTypes: []string{"image/jpeg", "image/png", "image/webp"},
	}

	for _, fileHeader := range files {
		file, err := fileHeader.Open()
		if err != nil {
			h.cleanupUploads(uploadedKeys)
			utils.ErrorResponse(c, http.StatusBadRequest, "Failed to read uploaded photo", err)
			return
		}

		processed, err := utils.ProcessUploadedImage(file, fileHeader, uploadOpts)
		file.Close()
		if err != nil {
			h.cleanupUploads(uploadedKeys)
			utils.ErrorResponse(c, http.StatusBadRequest, err.Error(), nil)
			return
		}

		// Generate unique filename and storage path
		filename := uuid.New().String() + processed.Extension
		storageKey := fmt.Sprintf("counterfeit-reports/%s/%s/%s", tenantID, reportID, filename)

		url, err := h.uploadFile(storageKey, processed.Data, processed.ContentType)
		if err != nil {
			h.cleanupUploads(uploadedKeys)
			utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to upload photo", err)
			return
		}

		photoURLs = append(photoURLs, url)
		uploadedKeys = append(uploadedKeys, storageKey)
	}

	// Prepare photos JSON
	var photosJSON []byte
	if len(photoURLs) > 0 {
		photosJSON, _ = json.Marshal(photoURLs)
	}

	// Prepare geolocation (from browser geolocation API if available)
	geolocation, _ := json.Marshal(map[string]interface{}{})

	// Create counterfeit report
	report := models.CounterfeitReport{
		QRCodeID:               qrCode.ID,
		TenantID:               qrCode.Batch.TenantID,
		CounterfeitDetectionID: detectionID,
		Description:            description,
		Photos:                 photosJSON,
		StoreName:              storeName,
		IPAddress:              c.ClientIP(),
		UserAgent:              c.GetHeader("User-Agent"),
		Geolocation:            geolocation,
	}

	if err := h.DB.Create(&report).Error; err != nil {
		h.cleanupUploads(uploadedKeys)
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to submit report", err)
		return
	}

	utils.SuccessResponse(c, http.StatusCreated, "Thank you for your report. We will investigate this matter.", gin.H{
		"report_id": report.ID,
	})
}

// uploadFile uploads a file to R2 or local filesystem
func (h *CounterfeitHandler) uploadFile(storageKey string, data []byte, contentType string) (string, error) {
	r2Client := storage.GetGlobalR2Client()
	if r2Client != nil {
		return r2Client.Upload(context.Background(), storageKey, data, contentType)
	}

	// Fallback to local filesystem
	fullDir := filepath.Join(h.Cfg.UploadPath, filepath.Dir(storageKey))
	if err := os.MkdirAll(fullDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create upload directory: %w", err)
	}

	filePath := filepath.Join(h.Cfg.UploadPath, storageKey)
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to save file: %w", err)
	}

	return fmt.Sprintf("/uploads/%s", storageKey), nil
}

// cleanupUploads removes uploaded files on failure (rollback)
func (h *CounterfeitHandler) cleanupUploads(keys []string) {
	r2Client := storage.GetGlobalR2Client()
	for _, key := range keys {
		if r2Client != nil {
			r2Client.Delete(context.Background(), key)
		} else {
			os.Remove(filepath.Join(h.Cfg.UploadPath, key))
		}
	}
}

// checkReportRateLimit enforces 1 report per IP per QR per hour via Redis.
// Returns true if rate-limited (caller should return). Fail-open if Redis unavailable.
func (h *CounterfeitHandler) checkReportRateLimit(c *gin.Context, qrCodeID string) bool {
	if database.RedisClient == nil {
		return false
	}

	clientIP := c.ClientIP()
	key := fmt.Sprintf("rl:counterfeit-report:%s:%s", clientIP, qrCodeID)
	ctx := context.Background()

	// Atomic INCR + conditional EXPIRE (same pattern as ratelimit.go)
	script := redis.NewScript(`
		local count = redis.call("INCR", KEYS[1])
		if count == 1 then
			redis.call("EXPIRE", KEYS[1], ARGV[1])
		end
		return count
	`)

	count, err := script.Run(ctx, database.RedisClient, []string{key}, 3600).Int64()
	if err != nil {
		return false
	}

	if count > 1 {
		ttl, _ := database.RedisClient.TTL(ctx, key).Result()
		c.Header("Retry-After", fmt.Sprintf("%d", int(ttl.Seconds())))
		utils.ErrorResponse(c, http.StatusTooManyRequests,
			"You have already submitted a report for this product recently. Please try again later.", nil)
		c.Abort()
		return true
	}

	return false
}

// ListCounterfeitReports returns counterfeit reports for a tenant
func (h *CounterfeitHandler) ListCounterfeitReports(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}

	page := utils.GetPageQuery(c)
	limit := utils.GetLimitQuery(c, 20)
	offset := (page - 1) * limit

	search := c.Query("search") // Search by store name

	query := h.DB.Model(&models.CounterfeitReport{}).
		Where("counterfeit_reports.tenant_id = ?", tenantUUID)

	if search != "" {
		query = query.Where("store_name ILIKE ?", "%"+search+"%")
	}

	var total int64
	query.Count(&total)

	var reports []models.CounterfeitReport
	query.Preload("QRCode.Batch.Product").
		Preload("CounterfeitDetection").
		Order("counterfeit_reports.created_at DESC").
		Offset(offset).Limit(limit).
		Find(&reports)

	// Group by store name for summary
	type StoreSummary struct {
		StoreName   string `json:"store_name"`
		ReportCount int64  `json:"report_count"`
	}
	var storeSummary []StoreSummary
	h.DB.Model(&models.CounterfeitReport{}).
		Select("store_name, COUNT(*) as report_count").
		Where("tenant_id = ?", tenantUUID).
		Group("store_name").
		Order("report_count DESC").
		Limit(10).
		Scan(&storeSummary)

	utils.SuccessResponse(c, http.StatusOK, "Counterfeit reports retrieved", gin.H{
		"reports":       reports,
		"page":          page,
		"limit":         limit,
		"total":         total,
		"total_pages":   (total + int64(limit) - 1) / int64(limit),
		"top_stores":    storeSummary,
	})
}

// GetCounterfeitReport returns a single counterfeit report
func (h *CounterfeitHandler) GetCounterfeitReport(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}
	reportID := c.Param("id")

	reportUUID, err := uuid.Parse(reportID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid report ID", err)
		return
	}

	var report models.CounterfeitReport
	if err := h.DB.Preload("QRCode.Batch.Product").
		Preload("CounterfeitDetection").
		Where("id = ? AND tenant_id = ?", reportUUID, tenantUUID).
		First(&report).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Report not found", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Counterfeit report retrieved", report)
}

// GetCounterfeitReportStats returns statistics about counterfeit reports
func (h *CounterfeitHandler) GetCounterfeitReportStats(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}

	var totalReports int64
	var todayReports int64
	var thisWeekReports int64
	var thisMonthReports int64

	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	weekAgo := now.AddDate(0, 0, -7)
	monthAgo := now.AddDate(0, -1, 0)

	h.DB.Model(&models.CounterfeitReport{}).Where("tenant_id = ?", tenantUUID).Count(&totalReports)
	h.DB.Model(&models.CounterfeitReport{}).Where("tenant_id = ? AND created_at >= ?", tenantUUID, today).Count(&todayReports)
	h.DB.Model(&models.CounterfeitReport{}).Where("tenant_id = ? AND created_at >= ?", tenantUUID, weekAgo).Count(&thisWeekReports)
	h.DB.Model(&models.CounterfeitReport{}).Where("tenant_id = ? AND created_at >= ?", tenantUUID, monthAgo).Count(&thisMonthReports)

	// Top reported stores
	type TopStore struct {
		StoreName   string `json:"store_name"`
		Province    string `json:"province"`
		City        string `json:"city"`
		ReportCount int64  `json:"report_count"`
	}
	var topStores []TopStore
	h.DB.Model(&models.CounterfeitReport{}).
		Select("store_name, province, city, COUNT(*) as report_count").
		Where("tenant_id = ?", tenantUUID).
		Group("store_name, province, city").
		Order("report_count DESC").
		Limit(10).
		Scan(&topStores)

	// Reports by province
	type ProvinceStats struct {
		Province    string `json:"province"`
		ReportCount int64  `json:"report_count"`
	}
	var provinceStats []ProvinceStats
	h.DB.Model(&models.CounterfeitReport{}).
		Select("COALESCE(province, 'Unknown') as province, COUNT(*) as report_count").
		Where("tenant_id = ?", tenantUUID).
		Group("province").
		Order("report_count DESC").
		Scan(&provinceStats)

	utils.SuccessResponse(c, http.StatusOK, "Counterfeit report stats retrieved", gin.H{
		"total":           totalReports,
		"today":           todayReports,
		"this_week":       thisWeekReports,
		"this_month":      thisMonthReports,
		"top_stores":      topStores,
		"by_province":     provinceStats,
	})
}
