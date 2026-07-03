package handlers

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gamatritunggal/smartscan/backend/internal/config"
	"github.com/gamatritunggal/smartscan/backend/internal/models"
	"github.com/gamatritunggal/smartscan/backend/internal/queue"
	sentryPkg "github.com/gamatritunggal/smartscan/backend/internal/sentry"
	"github.com/gamatritunggal/smartscan/backend/internal/services/audit"
	"github.com/gamatritunggal/smartscan/backend/internal/utils"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

// exportSemaphore limits concurrent export operations to prevent memory exhaustion
// Max 3 concurrent exports at a time
var exportSemaphore = make(chan struct{}, 3)

// exportQueueTimeout is the maximum time to wait for an export slot
var exportQueueTimeout = 30 * time.Second

// Package-level QR generation queue reference (set from main.go before router initializes handlers)
var (
	qrGenQueue   *queue.RedisQRGenerationQueue
	qrGenQueueMu sync.RWMutex
)

// SetQRGenerationQueue sets the global QR generation queue instance.
// Called from main.go after Redis initialization, before router setup.
func SetQRGenerationQueue(q *queue.RedisQRGenerationQueue) {
	qrGenQueueMu.Lock()
	defer qrGenQueueMu.Unlock()
	qrGenQueue = q
}

// getQRGenerationQueue returns the current QR generation queue instance (may be nil)
func getQRGenerationQueue() *queue.RedisQRGenerationQueue {
	qrGenQueueMu.RLock()
	defer qrGenQueueMu.RUnlock()
	return qrGenQueue
}

type QRBatchHandler struct {
	DB  *gorm.DB
	Cfg *config.Config
}

func NewQRBatchHandler(db *gorm.DB, cfg *config.Config) *QRBatchHandler {
	return &QRBatchHandler{DB: db, Cfg: cfg}
}

// ListQRBatches returns all QR batches for a tenant
func (h *QRBatchHandler) ListQRBatches(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}
	page := utils.GetPageQuery(c)
	limit := utils.GetLimitQuery(c, 20)
	status := c.Query("status")
	productID := c.Query("product_id")
	show := c.DefaultQuery("show", "active")

	offset := (page - 1) * limit

	var batches []models.QRBatch
	var total int64

	query := h.DB.Model(&models.QRBatch{}).Where("tenant_id = ?", tenantUUID)

	// Filter by soft delete status
	switch show {
	case "deleted":
		query = query.Where("deleted_at IS NOT NULL")
	case "all":
		// No filter - show everything
	default: // "active"
		query = query.Where("deleted_at IS NULL")
	}

	if status != "" {
		query = query.Where("status = ?", status)
	}
	if productID != "" {
		productUUID, err := uuid.Parse(productID)
		if err == nil {
			query = query.Where("product_id = ?", productUUID)
		}
	}

	query.Count(&total)
	query.Preload("Product").Preload("CreatedByStaff").Order("created_at DESC").Offset(offset).Limit(limit).Find(&batches)

	// Build scan count map in one query (avoid N+1)
	scanCountMap := make(map[string]int64)
	if len(batches) > 0 {
		batchIDs := make([]uuid.UUID, len(batches))
		for i, b := range batches {
			batchIDs[i] = b.ID
		}
		type scanCountRow struct {
			BatchID   uuid.UUID `gorm:"column:batch_id"`
			ScanCount int64     `gorm:"column:scan_count"`
		}
		var rows []scanCountRow
		h.DB.Raw(`
			SELECT qr_codes.batch_id, COUNT(interactions.id) AS scan_count
			FROM qr_codes
			LEFT JOIN interactions ON interactions.qr_code_id = qr_codes.id
			WHERE qr_codes.batch_id IN ?
			GROUP BY qr_codes.batch_id
		`, batchIDs).Scan(&rows)
		for _, r := range rows {
			scanCountMap[r.BatchID.String()] = r.ScanCount
		}
	}

	// Build response with scan_count per batch
	type batchWithScanCount struct {
		models.QRBatch
		ScanCount int64 `json:"scan_count"`
	}
	result := make([]batchWithScanCount, len(batches))
	for i, b := range batches {
		result[i] = batchWithScanCount{
			QRBatch:   b,
			ScanCount: scanCountMap[b.ID.String()],
		}
	}

	utils.SuccessResponse(c, http.StatusOK, "QR batches retrieved", gin.H{
		"batches": result,
		"pagination": gin.H{
			"page":       page,
			"limit":      limit,
			"total":      total,
			"total_page": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// GetQRBatch returns a single QR batch by ID
func (h *QRBatchHandler) GetQRBatch(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}
	id := c.Param("id")
	batchID, err := uuid.Parse(id)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid batch ID", err)
		return
	}

	var batch models.QRBatch
	if err := h.DB.
		Preload("Product").
		Preload("Product.DefaultValidationTemplate").
		Preload("Product.DefaultWarrantyTemplate").
		Preload("CreatedByStaff").
		Preload("ValidationTemplate").
		Preload("WarrantyTemplate").
		First(&batch, "id = ? AND tenant_id = ? AND deleted_at IS NULL", batchID, tenantUUID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "QR batch not found", err)
		return
	}

	// Get QR code count
	var qrCount int64
	h.DB.Model(&models.QRCode{}).Where("batch_id = ?", batchID).Count(&qrCount)

	utils.SuccessResponse(c, http.StatusOK, "QR batch retrieved", gin.H{
		"batch":    batch,
		"qr_count": qrCount,
	})
}

type CreateQRBatchRequest struct {
	ProductID string `json:"product_id" binding:"required"`
	BatchName string `json:"batch_name" binding:"required"`
	// QRCount — upper limit is enforced dynamically via cfg.QRGeneration.MaxBatchLimit.
	QRCount int `json:"qr_count" binding:"required,min=1"`
	Prefix                 string     `json:"prefix"`
	Suffix                 string     `json:"suffix"`
	ProductionDate         *time.Time `json:"production_date"`
	ExpiryDate             *time.Time `json:"expiry_date"`
	NeedValidation         bool       `json:"need_validation"`
	// NeedWarranty removed - warranty is now controlled at product level
	// Optional template overrides (if not set, uses product default → tenant default)
	ValidationTemplateID string `json:"validation_template_id"`
	WarrantyTemplateID   string `json:"warranty_template_id"` // Template override still allowed
	// Geofence: distribution zone (optional, Intermediate+ tier)
	GeofenceEnabled   bool     `json:"geofence_enabled"`
	GeofenceLatitude  *float64 `json:"geofence_latitude"`
	GeofenceLongitude *float64 `json:"geofence_longitude"`
	GeofenceRadiusKm  *float64 `json:"geofence_radius_km"`
	GeofenceLabel          string  `json:"geofence_label"`
	GeofenceZoneTemplateID *string `json:"geofence_zone_template_id"`
}

// CreateQRBatch creates a new QR batch and generates QR codes
func (h *QRBatchHandler) CreateQRBatch(c *gin.Context) {
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

	var req CreateQRBatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request", err)
		return
	}

	// Enforce dynamic max batch limit from config (operators can raise via env var)
	maxBatchLimit := 5000000
	if h.Cfg != nil && h.Cfg.QRGeneration.MaxBatchLimit > 0 {
		maxBatchLimit = h.Cfg.QRGeneration.MaxBatchLimit
	}
	if req.QRCount > maxBatchLimit {
		utils.ErrorResponse(c, http.StatusBadRequest,
			fmt.Sprintf("QR Count exceeds the maximum batch size (%s).",
				utils.FormatThousands(maxBatchLimit)), nil)
		return
	}

	productID, err := uuid.Parse(req.ProductID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid product ID", err)
		return
	}

	// Verify product exists and belongs to tenant
	var product models.Product
	if err := h.DB.First(&product, "id = ? AND tenant_id = ?", productID, tenantUUID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Product not found", err)
		return
	}

	// Check for duplicate batch name for the same product
	var existingBatch models.QRBatch
	if err := h.DB.Where("product_id = ? AND batch_name = ? AND deleted_at IS NULL", productID, req.BatchName).First(&existingBatch).Error; err == nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Batch name already exists for this product", nil)
		return
	}

	// Get user staff ID
	var staff models.TenantStaff
	h.DB.Where("user_id = ?", userUUID).First(&staff)

	// Generate batch code
	batchCode := generateBatchNumber()

	// Geofence: validate distribution zone (Intermediate+ only)
	geofenceEnabled := req.GeofenceEnabled
	var geofenceLat, geofenceLng, geofenceRadius *float64
	geofenceLabel := ""
	if geofenceEnabled {
		{
			if req.GeofenceLatitude == nil || req.GeofenceLongitude == nil || req.GeofenceRadiusKm == nil {
				utils.ErrorResponse(c, http.StatusBadRequest, "Geofence requires latitude, longitude, and radius", nil)
				return
			}
			if *req.GeofenceLatitude < -90 || *req.GeofenceLatitude > 90 {
				utils.ErrorResponse(c, http.StatusBadRequest, "Invalid geofence latitude (-90 to 90)", nil)
				return
			}
			if *req.GeofenceLongitude < -180 || *req.GeofenceLongitude > 180 {
				utils.ErrorResponse(c, http.StatusBadRequest, "Invalid geofence longitude (-180 to 180)", nil)
				return
			}
			if *req.GeofenceRadiusKm < 1 || *req.GeofenceRadiusKm > 500 {
				utils.ErrorResponse(c, http.StatusBadRequest, "Geofence radius must be between 1 and 500 km", nil)
				return
			}
			geofenceLat = req.GeofenceLatitude
			geofenceLng = req.GeofenceLongitude
			geofenceRadius = req.GeofenceRadiusKm
			geofenceLabel = strings.TrimSpace(req.GeofenceLabel)
			if labelRunes := []rune(geofenceLabel); len(labelRunes) > 255 {
				geofenceLabel = string(labelRunes[:255])
			}
		}
	}

	// Handle template overrides (validate before transaction)
	// Resolution: explicit batch override → product default (handled in GetPublicTemplate)
	var validationTemplateID, warrantyTemplateID *uuid.UUID

	if req.ValidationTemplateID != "" {
		tid, err := uuid.Parse(req.ValidationTemplateID)
		if err != nil {
			utils.ErrorResponse(c, http.StatusBadRequest, "Invalid validation template ID", err)
			return
		}
		// Verify template exists and belongs to tenant
		var template models.PageTemplate
		if err := h.DB.First(&template, "id = ? AND tenant_id = ? AND template_type = ?",
			tid, tenantUUID, models.TemplateTypeValidation).Error; err != nil {
			utils.ErrorResponse(c, http.StatusBadRequest, "Validation template not found", err)
			return
		}
		validationTemplateID = &tid
	}

	// Warranty template override allowed if product has warranty enabled
	if req.WarrantyTemplateID != "" && product.WarrantyEnabled {
		tid, err := uuid.Parse(req.WarrantyTemplateID)
		if err != nil {
			utils.ErrorResponse(c, http.StatusBadRequest, "Invalid warranty template ID", err)
			return
		}
		// Verify template exists and belongs to tenant
		var template models.PageTemplate
		if err := h.DB.First(&template, "id = ? AND tenant_id = ? AND template_type = ?",
			tid, tenantUUID, models.TemplateTypeWarranty).Error; err != nil {
			utils.ErrorResponse(c, http.StatusBadRequest, "Warranty template not found", err)
			return
		}
		warrantyTemplateID = &tid
	}


	// Initial status: pending_queue (updated after Redis enqueue attempt)
	initialStatus := models.QRBatchStatusPendingQueue

	batch := models.QRBatch{
		TenantID:             tenantUUID,
		ProductID:            productID,
		BatchName:            req.BatchName,
		BatchCode:            batchCode,
		QRCount:              req.QRCount,
		Status:               initialStatus,
		Prefix:               req.Prefix,
		Suffix:               req.Suffix,
		ProductionDate:       req.ProductionDate,
		ExpiryDate:           req.ExpiryDate,
		NeedValidation:       req.NeedValidation,
		ValidationTemplateID: validationTemplateID,
		// NeedWarranty removed - warranty is now at product level (product.WarrantyEnabled)
		WarrantyTemplateID: warrantyTemplateID, // Template override still allowed
		// IsStatic removed - static QR is only at product level (products.static_qr_uuid)
		// Geofence
		GeofenceEnabled:   geofenceEnabled,
		GeofenceLatitude:  geofenceLat,
		GeofenceLongitude: geofenceLng,
		GeofenceRadiusKm:  geofenceRadius,
		GeofenceLabel:     geofenceLabel,
		CreatedBy:         &staff.ID,
	}

	// Create batch in DB (single-row transaction, no need for explicit tx)
	if err := h.DB.Create(&batch).Error; err != nil {
		// Check for unique constraint violation (duplicate batch name race condition)
		if strings.Contains(err.Error(), "uniq_qr_batch_name_active") {
			utils.ErrorResponse(c, http.StatusBadRequest, "Batch name already exists for this product", nil)
			return
		}
		sentryPkg.CaptureHandlerError(c, err, "qrbatch.CreateQRBatch", sentryPkg.ErrorTypeDatabase, sentryPkg.SeverityHigh)
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to create QR batch", err)
		return
	}

	// Increment zone template usage count if template was used
	if geofenceEnabled && req.GeofenceZoneTemplateID != nil && *req.GeofenceZoneTemplateID != "" {
		if tid, err := uuid.Parse(*req.GeofenceZoneTemplateID); err == nil {
			h.DB.Model(&models.GeofenceZoneTemplate{}).
				Where("id = ? AND tenant_id = ? AND deleted_at IS NULL", tid, tenantUUID).
				UpdateColumn("usage_count", gorm.Expr("usage_count + 1"))
		}
	}

	// Enqueue generation job to Redis
	qrQueue := getQRGenerationQueue()
	if qrQueue == nil {
		// Queue not initialized. There are two distinct cases:
		// 1. QR_GENERATION_ENABLED=false — feature disabled intentionally. Reject the batch
		//    (otherwise it would sit in pending_queue forever with no worker or scanner).
		// 2. Queue was enabled but initialization failed (e.g., Redis down at startup).
		//    In this case the scanner is also not running, so we return 503 and the
		//    batch record is left in pending_queue; an operator must restart the server
		//    once Redis is back, which will retry initialization and pick up the batch.
		if h.Cfg == nil || !h.Cfg.QRGeneration.Enabled {
			// Roll back the batch record — feature is off, no point keeping a zombie row
			h.DB.Delete(&batch)
			utils.ErrorResponse(c, http.StatusServiceUnavailable,
				"Async QR generation is currently disabled on this server. Please contact an administrator.", nil)
			return
		}

		// Queue enabled but unavailable — keep the batch so the scanner can pick it up on recovery
		h.DB.Preload("Product").First(&batch, "id = ?", batch.ID)
		utils.SuccessResponse(c, http.StatusAccepted,
			"Batch saved. Generation will start automatically when the queue service becomes available.",
			batch)
		return
	}

	maxRetries := 5
	if h.Cfg != nil && h.Cfg.QRGeneration.MaxRetries > 0 {
		maxRetries = h.Cfg.QRGeneration.MaxRetries
	}
	job := queue.NewQRGenerationJob(
		batch.ID.String(),
		tenantUUID.String(),
		req.QRCount,
		req.Prefix,
		req.Suffix,
		maxRetries,
	)

	enqueueCtx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	if err := qrQueue.Enqueue(enqueueCtx, job); err != nil {
		// Enqueue failed — Redis might be down. Batch remains in DB with status=pending_queue.
		// The scanner will auto-retry when Redis is back.
		sentryPkg.CaptureHandlerError(c, err, "qrbatch.CreateQRBatch.enqueue", sentryPkg.ErrorTypeExternal, sentryPkg.SeverityMedium)
		h.DB.Preload("Product").First(&batch, "id = ?", batch.ID)
		utils.SuccessResponse(c, http.StatusAccepted,
			"Batch saved. Generation will start automatically when the queue service recovers.",
			batch)
		return
	}

	// Enqueue successful — update status to queued
	if err := h.DB.Model(&batch).Update("status", models.QRBatchStatusQueued).Error; err != nil {
		// Log but don't fail — scanner will reconcile
		sentryPkg.CaptureHandlerError(c, err, "qrbatch.CreateQRBatch.updateStatus", sentryPkg.ErrorTypeDatabase, sentryPkg.SeverityLow)
	}

	h.DB.Preload("Product").First(&batch, "id = ?", batch.ID)
	utils.SuccessResponse(c, http.StatusCreated, "QR batch created and queued for generation", batch)
}

// UpdateQRBatch updates a QR batch
func (h *QRBatchHandler) UpdateQRBatch(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}
	id := c.Param("id")
	batchID, err := uuid.Parse(id)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid batch ID", err)
		return
	}

	var req struct {
		BatchName            string   `json:"batch_name"`
		ValidationTemplateID *string  `json:"validation_template_id"`
		WarrantyTemplateID   *string  `json:"warranty_template_id"`
			GeofenceEnabled      *bool    `json:"geofence_enabled"`
		GeofenceLatitude     *float64 `json:"geofence_latitude"`
		GeofenceLongitude    *float64 `json:"geofence_longitude"`
		GeofenceRadiusKm     *float64 `json:"geofence_radius_km"`
		GeofenceLabel              *string  `json:"geofence_label"`
		GeofenceZoneTemplateID     *string  `json:"geofence_zone_template_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request", err)
		return
	}

	var batch models.QRBatch
	if err := h.DB.First(&batch, "id = ? AND tenant_id = ? AND deleted_at IS NULL", batchID, tenantUUID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "QR batch not found", err)
		return
	}

	updates := map[string]interface{}{}

	if req.BatchName != "" {
		updates["batch_name"] = req.BatchName
	}

	// Handle validation template update
	if req.ValidationTemplateID != nil {
		if *req.ValidationTemplateID == "" {
			updates["validation_template_id"] = nil
		} else {
			tid, err := uuid.Parse(*req.ValidationTemplateID)
			if err != nil {
				utils.ErrorResponse(c, http.StatusBadRequest, "Invalid validation template ID", err)
				return
			}
			var template models.PageTemplate
			if err := h.DB.First(&template, "id = ? AND tenant_id = ? AND template_type = ?",
				tid, tenantUUID, models.TemplateTypeValidation).Error; err != nil {
				utils.ErrorResponse(c, http.StatusBadRequest, "Validation template not found", err)
				return
			}
			updates["validation_template_id"] = tid
		}
	}

	// Handle warranty template update
	if req.WarrantyTemplateID != nil {
		if *req.WarrantyTemplateID == "" {
			updates["warranty_template_id"] = nil
		} else {
			tid, err := uuid.Parse(*req.WarrantyTemplateID)
			if err != nil {
				utils.ErrorResponse(c, http.StatusBadRequest, "Invalid warranty template ID", err)
				return
			}
			var template models.PageTemplate
			if err := h.DB.First(&template, "id = ? AND tenant_id = ? AND template_type = ?",
				tid, tenantUUID, models.TemplateTypeWarranty).Error; err != nil {
				utils.ErrorResponse(c, http.StatusBadRequest, "Warranty template not found", err)
				return
			}
			updates["warranty_template_id"] = tid
		}
	}

	// Handle geofence update
	if req.GeofenceEnabled != nil {
		if *req.GeofenceEnabled {
			// Validate required fields when enabling geofence
			if req.GeofenceLatitude == nil || req.GeofenceLongitude == nil || req.GeofenceRadiusKm == nil {
				utils.ErrorResponse(c, http.StatusBadRequest, "Geofence requires latitude, longitude, and radius", nil)
				return
			}
			if *req.GeofenceLatitude < -90 || *req.GeofenceLatitude > 90 {
				utils.ErrorResponse(c, http.StatusBadRequest, "Invalid geofence latitude (-90 to 90)", nil)
				return
			}
			if *req.GeofenceLongitude < -180 || *req.GeofenceLongitude > 180 {
				utils.ErrorResponse(c, http.StatusBadRequest, "Invalid geofence longitude (-180 to 180)", nil)
				return
			}
			if *req.GeofenceRadiusKm < 1 || *req.GeofenceRadiusKm > 500 {
				utils.ErrorResponse(c, http.StatusBadRequest, "Geofence radius must be between 1 and 500 km", nil)
				return
			}
			updates["geofence_enabled"] = true
			updates["geofence_latitude"] = *req.GeofenceLatitude
			updates["geofence_longitude"] = *req.GeofenceLongitude
			updates["geofence_radius_km"] = *req.GeofenceRadiusKm
			label := ""
			if req.GeofenceLabel != nil {
				label = strings.TrimSpace(*req.GeofenceLabel)
				if labelRunes := []rune(label); len(labelRunes) > 255 {
					label = string(labelRunes[:255])
				}
			}
			updates["geofence_label"] = label
		} else {
			// Disable geofence
			updates["geofence_enabled"] = false
			updates["geofence_latitude"] = nil
			updates["geofence_longitude"] = nil
			updates["geofence_radius_km"] = nil
			updates["geofence_label"] = ""
		}
	} else if req.GeofenceLatitude != nil || req.GeofenceLongitude != nil || req.GeofenceRadiusKm != nil || req.GeofenceLabel != nil {
		// Update geofence fields without toggling enabled state (only if geofence is already enabled)
		if batch.GeofenceEnabled {
			if req.GeofenceLatitude != nil {
				if *req.GeofenceLatitude < -90 || *req.GeofenceLatitude > 90 {
					utils.ErrorResponse(c, http.StatusBadRequest, "Invalid geofence latitude (-90 to 90)", nil)
					return
				}
				updates["geofence_latitude"] = *req.GeofenceLatitude
			}
			if req.GeofenceLongitude != nil {
				if *req.GeofenceLongitude < -180 || *req.GeofenceLongitude > 180 {
					utils.ErrorResponse(c, http.StatusBadRequest, "Invalid geofence longitude (-180 to 180)", nil)
					return
				}
				updates["geofence_longitude"] = *req.GeofenceLongitude
			}
			if req.GeofenceRadiusKm != nil {
				if *req.GeofenceRadiusKm < 1 || *req.GeofenceRadiusKm > 500 {
					utils.ErrorResponse(c, http.StatusBadRequest, "Geofence radius must be between 1 and 500 km", nil)
					return
				}
				updates["geofence_radius_km"] = *req.GeofenceRadiusKm
			}
			if req.GeofenceLabel != nil {
				label := strings.TrimSpace(*req.GeofenceLabel)
				if labelRunes := []rune(label); len(labelRunes) > 255 {
					label = string(labelRunes[:255])
				}
				updates["geofence_label"] = label
			}
		}
	}

	if len(updates) > 0 {
		if err := h.DB.Model(&batch).Updates(updates).Error; err != nil {
			sentryPkg.CaptureHandlerError(c, err, "qrbatch.UpdateQRBatch", sentryPkg.ErrorTypeDatabase, sentryPkg.SeverityMedium)
			utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to update batch", err)
			return
		}
	}

	// Increment zone template usage count if template was used during geofence update
	if req.GeofenceZoneTemplateID != nil && *req.GeofenceZoneTemplateID != "" {
		if tid, err := uuid.Parse(*req.GeofenceZoneTemplateID); err == nil {
			h.DB.Model(&models.GeofenceZoneTemplate{}).
				Where("id = ? AND tenant_id = ? AND deleted_at IS NULL", tid, tenantUUID).
				UpdateColumn("usage_count", gorm.Expr("usage_count + 1"))
		}
	}

	// Reload with relations
	h.DB.
		Preload("Product").
		Preload("ValidationTemplate").
		Preload("WarrantyTemplate").
		First(&batch, "id = ?", batchID)

	utils.SuccessResponse(c, http.StatusOK, "Batch updated", batch)
}

// QRCodeWithStats represents a QR code with computed scan statistics
type QRCodeWithStats struct {
	models.QRCode
	ScanCount      int64      `json:"scan_count"`
	FirstScannedAt *time.Time `json:"first_scanned_at"`
	LastScannedAt  *time.Time `json:"last_scanned_at"`
}

// ListQRCodes returns QR codes for a batch with scan statistics
func (h *QRBatchHandler) ListQRCodes(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}
	batchIDStr := c.Param("id")
	batchID, err := uuid.Parse(batchIDStr)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid batch ID", err)
		return
	}

	// Verify batch belongs to tenant
	var batch models.QRBatch
	if err := h.DB.First(&batch, "id = ? AND tenant_id = ? AND deleted_at IS NULL", batchID, tenantUUID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "QR batch not found", err)
		return
	}

	page := utils.GetPageQuery(c)
	limit := utils.GetLimitQuery(c, 100)
	status := c.Query("status")
	counterfeitStatus := c.Query("counterfeit_status")

	offset := (page - 1) * limit

	var total int64

	// Count total (with status and counterfeit filters if provided)
	countQuery := h.DB.Model(&models.QRCode{}).Where("batch_id = ?", batchID)
	if status != "" {
		countQuery = countQuery.Where("status = ?", status)
	}
	// Parse and validate counterfeit_status (supports comma-separated values)
	allowedCounterfeit := map[string]bool{"valid": true, "warning": true, "counterfeit": true}
	var validCounterfeitStatuses []string
	if counterfeitStatus != "" {
		for _, s := range strings.Split(counterfeitStatus, ",") {
			s = strings.TrimSpace(s)
			if allowedCounterfeit[s] {
				validCounterfeitStatuses = append(validCounterfeitStatuses, s)
			}
		}
	}
	if len(validCounterfeitStatuses) == 1 {
		countQuery = countQuery.Where("counterfeit_status = ?", validCounterfeitStatuses[0])
	} else if len(validCounterfeitStatuses) > 1 {
		countQuery = countQuery.Where("counterfeit_status IN (?)", validCounterfeitStatuses)
	}
	countQuery.Count(&total)

	// Build filters for raw SQL
	filterSQL := ""
	args := []interface{}{batchID}
	if status != "" {
		filterSQL += " AND qc.status = ?"
		args = append(args, status)
	}
	if len(validCounterfeitStatuses) == 1 {
		filterSQL += " AND qc.counterfeit_status = ?"
		args = append(args, validCounterfeitStatuses[0])
	} else if len(validCounterfeitStatuses) > 1 {
		filterSQL += " AND qc.counterfeit_status IN (?" + strings.Repeat(",?", len(validCounterfeitStatuses)-1) + ")"
		for _, s := range validCounterfeitStatuses {
			args = append(args, s)
		}
	}
	args = append(args, offset, limit)

	// Query QR codes with scan stats using LEFT JOIN (avoid N+1)
	var codesWithStats []QRCodeWithStats
	h.DB.Raw(`
		SELECT
			qc.*,
			COALESCE(stats.scan_count, 0) as scan_count,
			stats.first_scanned_at,
			stats.last_scanned_at
		FROM qr_codes qc
		LEFT JOIN (
			SELECT
				qr_code_id,
				COUNT(*) as scan_count,
				MIN(created_at) as first_scanned_at,
				MAX(created_at) as last_scanned_at
			FROM interactions
			WHERE interaction_subcategory = 'product_validation'
			GROUP BY qr_code_id
		) stats ON stats.qr_code_id = qc.id
		WHERE qc.batch_id = ? `+filterSQL+`
		ORDER BY qc.created_at
		OFFSET ? LIMIT ?
	`, args...).Scan(&codesWithStats)

	utils.SuccessResponse(c, http.StatusOK, "QR codes retrieved", gin.H{
		"codes": codesWithStats,
		"pagination": gin.H{
			"page":       page,
			"limit":      limit,
			"total":      total,
			"total_page": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// GetQRCodeDetail returns a single QR code with its interactions
func (h *QRBatchHandler) GetQRCodeDetail(c *gin.Context) {
	codeID := c.Param("codeId")

	// Validate UUID
	codeUUID, err := uuid.Parse(codeID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid QR code ID", err)
		return
	}

	// Get QR code with batch info (verify tenant ownership via batch)
	var code models.QRCode
	if err := h.DB.Preload("Batch").
		Preload("Batch.Product").
		First(&code, "id = ?", codeUUID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "QR code not found", err)
		return
	}

	// Verify tenant ownership
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok || code.Batch.TenantID != tenantUUID {
		utils.ErrorResponse(c, http.StatusForbidden, "Access denied", nil)
		return
	}

	// Get interactions for this QR code
	var interactions []models.Interaction
	h.DB.Where("qr_code_id = ?", codeUUID).
		Order("created_at DESC").
		Limit(100).
		Find(&interactions)

	// Calculate scan statistics from interactions (on-the-fly)
	type ScanStats struct {
		ScanCount      int64      `json:"scan_count"`
		FirstScannedAt *time.Time `json:"first_scanned_at"`
		LastScannedAt  *time.Time `json:"last_scanned_at"`
	}
	var stats ScanStats
	h.DB.Model(&models.Interaction{}).
		Select("COUNT(*) as scan_count, MIN(created_at) as first_scanned_at, MAX(created_at) as last_scanned_at").
		Where("qr_code_id = ? AND interaction_subcategory = ?", codeUUID, models.InteractionSubcategoryProductValidation).
		Scan(&stats)

	// Generate QR URL with scan endpoint format (/s/{base58})
	baseURL := h.Cfg.FrontendURL
	if baseURL == "" {
		baseURL = "http://localhost:3000"
	}
	qrURL := fmt.Sprintf("%s/s/%s", baseURL, utils.UUIDToBase58(code.QRUUID))

	utils.SuccessResponse(c, http.StatusOK, "QR code retrieved", gin.H{
		"qr_code":          code,
		"qr_url":           qrURL,
		"batch":            code.Batch,
		"product":          code.Batch.Product,
		"interactions":     interactions,
		"scan_count":       stats.ScanCount,
		"first_scanned_at": stats.FirstScannedAt,
		"last_scanned_at":  stats.LastScannedAt,
	})
}

// Helper functions
func generateBatchNumber() string {
	timestamp := time.Now().UTC().Format("20060102150405")
	randomHex := utils.GenerateRandomHexWithFallback(4)
	return fmt.Sprintf("BTH-%s-%s", timestamp, randomHex)
}



// ExportQRCodesCSV exports QR codes as CSV file for label printing
func (h *QRBatchHandler) ExportQRCodesCSV(c *gin.Context) {
	// Extract tenant UUID once (before semaphore to avoid holding slot on auth failure)
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}

	// === SEMAPHORE: Coba ambil slot untuk export ===
	select {
	case exportSemaphore <- struct{}{}: // Dapat slot!
		// Kembalikan slot setelah selesai (defer)
		defer func() { <-exportSemaphore }()
	case <-time.After(exportQueueTimeout): // Timeout setelah 30 detik
		utils.ErrorResponse(c, http.StatusServiceUnavailable,
			"Server sedang sibuk memproses export lain. Silakan coba lagi dalam beberapa saat.", nil)
		return
	}
	// === END SEMAPHORE ===
	batchIDStr := c.Param("id")
	batchID, err := uuid.Parse(batchIDStr)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid batch ID", err)
		return
	}

	// Verify batch belongs to tenant and get batch info
	var batch models.QRBatch
	if err := h.DB.Preload("Product").First(&batch, "id = ? AND tenant_id = ? AND deleted_at IS NULL", batchID, tenantUUID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "QR batch not found", err)
		return
	}

	// Get all QR codes for this batch
	var codes []models.QRCode
	if err := h.DB.Where("batch_id = ?", batchID).Order("created_at").Find(&codes).Error; err != nil {
		sentryPkg.CaptureHandlerError(c, err, "qrbatch.ExportQRCodesCSV", sentryPkg.ErrorTypeDatabase, sentryPkg.SeverityMedium)
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to fetch QR codes", err)
		return
	}

	// Create CSV
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	// Write header - compatible with label printers (BarTender, NiceLabel, ZebraDesigner)
	header := []string{
		"seq_number",
		"qr_content",
		"qr_code",
		"product_name",
		"product_code",
		"batch_name",
		"batch_code",
		"production_date",
		"expiry_date",
	}
	writer.Write(header)

	// Write data rows
	baseURL := h.Cfg.FrontendURL
	if baseURL == "" {
		baseURL = "http://localhost:3000" // FRONTEND_URL not set — dev fallback
	}

	for i, code := range codes {
		prodDate := ""
		if batch.ProductionDate != nil {
			prodDate = batch.ProductionDate.Format("2006-01-02")
		}
		expDate := ""
		if batch.ExpiryDate != nil {
			expDate = batch.ExpiryDate.Format("2006-01-02")
		}

		productName := ""
		productCode := ""
		if batch.Product != nil {
			productName = batch.Product.ProductName
			productCode = batch.Product.ProductCode
		}

		row := []string{
			strconv.Itoa(i + 1), // seq_number
			fmt.Sprintf("%s/s/%s", baseURL, utils.UUIDToBase58(code.QRUUID)), // qr_content (scan URL, Base58 encoded)
			code.QRCode,     // qr_code (raw hex for reference)
			productName,     // product_name
			productCode,     // product_code
			batch.BatchName, // batch_name
			batch.BatchCode, // batch_code
			prodDate,        // production_date
			expDate,         // expiry_date
		}
		writer.Write(row)
	}

	writer.Flush()

	// Audit log
	audit.LogFromContext(c, h.DB, models.ActionTypeExport, "qr_batch", &batchID,
		nil, map[string]interface{}{"format": "csv", "count": len(codes)})

	// Set headers for file download
	filename := fmt.Sprintf("%s_%s_qr_codes.csv", batch.BatchCode, time.Now().UTC().Format("20060102"))
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Length", strconv.Itoa(buf.Len()))

	c.Data(http.StatusOK, "text/csv; charset=utf-8", buf.Bytes())
}

// ExportQRCodesExcel exports QR codes as Excel file for label printing
func (h *QRBatchHandler) ExportQRCodesExcel(c *gin.Context) {
	// Extract tenant UUID once (before semaphore to avoid holding slot on auth failure)
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}

	// === SEMAPHORE: Coba ambil slot untuk export ===
	select {
	case exportSemaphore <- struct{}{}: // Dapat slot!
		// Kembalikan slot setelah selesai (defer)
		defer func() { <-exportSemaphore }()
	case <-time.After(exportQueueTimeout): // Timeout setelah 30 detik
		utils.ErrorResponse(c, http.StatusServiceUnavailable,
			"Server sedang sibuk memproses export lain. Silakan coba lagi dalam beberapa saat.", nil)
		return
	}
	// === END SEMAPHORE ===

	batchIDStr := c.Param("id")
	batchID, err := uuid.Parse(batchIDStr)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid batch ID", err)
		return
	}

	// Verify batch belongs to tenant and get batch info
	var batch models.QRBatch
	if err := h.DB.Preload("Product").First(&batch, "id = ? AND tenant_id = ? AND deleted_at IS NULL", batchID, tenantUUID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "QR batch not found", err)
		return
	}

	// Get all QR codes for this batch
	var codes []models.QRCode
	if err := h.DB.Where("batch_id = ?", batchID).Order("created_at").Find(&codes).Error; err != nil {
		sentryPkg.CaptureHandlerError(c, err, "qrbatch.ExportQRCodesExcel", sentryPkg.ErrorTypeDatabase, sentryPkg.SeverityMedium)
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to fetch QR codes", err)
		return
	}

	// Create Excel file
	f := excelize.NewFile()
	defer f.Close()

	sheetName := "QR Codes"
	f.SetSheetName("Sheet1", sheetName)

	// Write header - compatible with label printers
	headers := []string{
		"seq_number",
		"qr_content",
		"qr_code",
		"product_name",
		"product_code",
		"batch_name",
		"batch_code",
		"production_date",
		"expiry_date",
	}

	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetName, cell, header)
	}

	// Style header row
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"E0E0E0"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	f.SetRowStyle(sheetName, 1, 1, headerStyle)

	// Write data rows
	baseURL := h.Cfg.FrontendURL
	if baseURL == "" {
		baseURL = "http://localhost:3000" // FRONTEND_URL not set — dev fallback
	}

	for i, code := range codes {
		row := i + 2 // Start from row 2 (row 1 is header)

		prodDate := ""
		if batch.ProductionDate != nil {
			prodDate = batch.ProductionDate.Format("2006-01-02")
		}
		expDate := ""
		if batch.ExpiryDate != nil {
			expDate = batch.ExpiryDate.Format("2006-01-02")
		}

		productName := ""
		productCode := ""
		if batch.Product != nil {
			productName = batch.Product.ProductName
			productCode = batch.Product.ProductCode
		}

		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), i+1)                                                              // seq_number
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), fmt.Sprintf("%s/s/%s", baseURL, utils.UUIDToBase58(code.QRUUID))) // qr_content (scan URL, Base58)
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), code.QRCode)                                                      // qr_code (raw hex)
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), productName)                                                      // product_name
		f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), productCode)                                                      // product_code
		f.SetCellValue(sheetName, fmt.Sprintf("F%d", row), batch.BatchName)                                                  // batch_name
		f.SetCellValue(sheetName, fmt.Sprintf("G%d", row), batch.BatchCode)                                                  // batch_code
		f.SetCellValue(sheetName, fmt.Sprintf("H%d", row), prodDate)                                                         // production_date
		f.SetCellValue(sheetName, fmt.Sprintf("I%d", row), expDate)                                                          // expiry_date

	}

	// Auto-fit columns
	colCount := 9
	for i := 1; i <= colCount; i++ {
		colName, _ := excelize.ColumnNumberToName(i)
		f.SetColWidth(sheetName, colName, colName, 20)
	}
	// Make qr_content column wider
	f.SetColWidth(sheetName, "B", "B", 50)

	// Write to buffer
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		sentryPkg.CaptureHandlerError(c, err, "qrbatch.ExportQRCodesExcel", sentryPkg.ErrorTypeInternal, sentryPkg.SeverityMedium)
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to generate Excel file", err)
		return
	}

	// Audit log
	audit.LogFromContext(c, h.DB, models.ActionTypeExport, "qr_batch", &batchID,
		nil, map[string]interface{}{"format": "xlsx", "count": len(codes)})

	// Set headers for file download
	filename := fmt.Sprintf("%s_%s_qr_codes.xlsx", batch.BatchCode, time.Now().UTC().Format("20060102"))
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Length", strconv.Itoa(buf.Len()))

	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buf.Bytes())
}

// GetBatchHeatmap returns scan geolocation data for a specific batch
func (h *QRBatchHandler) GetBatchHeatmap(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}

	batchID := c.Param("id")
	if _, err := uuid.Parse(batchID); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid batch ID", nil)
		return
	}

	// Fetch batch with Product to determine QR type
	var batch models.QRBatch
	if err := h.DB.Preload("Product").First(&batch, "id = ? AND tenant_id = ? AND deleted_at IS NULL", batchID, tenantUUID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Batch not found", nil)
		return
	}

	isProTier := true

	// Parse query parameters
	fromDate := c.DefaultQuery("from", time.Now().UTC().AddDate(0, 0, -30).Format("2006-01-02"))
	toDate := c.DefaultQuery("to", time.Now().UTC().Format("2006-01-02"))
	source := c.DefaultQuery("source", "all")

	// Validate date format
	if _, err := time.Parse("2006-01-02", fromDate); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid 'from' date format (expected YYYY-MM-DD)", nil)
		return
	}
	if _, err := time.Parse("2006-01-02", toDate); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid 'to' date format (expected YYYY-MM-DD)", nil)
		return
	}

	// Set limit based on tier
	limit := 500
	if isProTier {
		limit = 2000
	}

	// Scans are recorded with qr_code_id, join via qr_codes.batch_id
	query := h.DB.Table("interactions i").
		Select(`i.id, i.geolocation, i.interaction_subcategory, i.created_at,
			qc.counterfeit_status, qc.id as qr_code_id`).
		Joins("JOIN qr_codes qc ON qc.id = i.qr_code_id").
		Where("qc.batch_id = ?", batchID).
		Where("i.tenant_id = ?", tenantUUID).
		Where("i.interaction_category = ?", models.InteractionCategoryEndUserAccess).
		Where("i.geolocation IS NOT NULL").
		Where("DATE(i.created_at) >= ? AND DATE(i.created_at) <= ?", fromDate, toDate)

	// Filter by source
	switch source {
	case "validation":
		query = query.Where("i.interaction_subcategory = ?", models.InteractionSubcategoryProductValidation)
	case "warranty":
		query = query.Where("i.interaction_subcategory = ?", models.InteractionSubcategoryWarrantyActivation)
	case "campaign":
		query = query.Where("i.interaction_subcategory = ?", models.InteractionSubcategoryCampaign)
	}

	// Execute query
	var rawData []struct {
		ID                     uuid.UUID `json:"id"`
		Geolocation            []byte    `json:"geolocation"`
		InteractionSubcategory string    `json:"interaction_subcategory"`
		CreatedAt              time.Time `json:"created_at"`
		CounterfeitStatus      string    `json:"counterfeit_status"`
		QrCodeID               uuid.UUID `json:"qr_code_id"`
	}

	if err := query.Order("i.created_at DESC").Limit(limit).Scan(&rawData).Error; err != nil {
		sentryPkg.CaptureHandlerError(c, err, "qrbatch.GetBatchHeatmap", sentryPkg.ErrorTypeDatabase, sentryPkg.SeverityMedium)
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to fetch heatmap data", err)
		return
	}

	// Parse geolocation and build response
	type HeatmapPoint struct {
		Lat               float64    `json:"lat"`
		Lng               float64    `json:"lng"`
		CounterfeitStatus string     `json:"counterfeit_status"`
		ScanType          string     `json:"scan_type"`
		CreatedAt         time.Time  `json:"created_at"`
		QrCodeID          *uuid.UUID `json:"qr_code_id"`
	}

	var points []HeatmapPoint
	for _, row := range rawData {
		if row.Geolocation == nil {
			continue
		}

		var geo struct {
			Lat float64 `json:"lat"`
			Lng float64 `json:"lng"`
		}

		if err := json.Unmarshal(row.Geolocation, &geo); err != nil {
			continue
		}

		// Skip invalid coordinates
		if geo.Lat == 0 && geo.Lng == 0 {
			continue
		}

		point := HeatmapPoint{
			Lat:      geo.Lat,
			Lng:      geo.Lng,
			ScanType: row.InteractionSubcategory,
			CreatedAt: row.CreatedAt,
		}

		point.CounterfeitStatus = row.CounterfeitStatus
		if row.QrCodeID != uuid.Nil {
			point.QrCodeID = &row.QrCodeID
		}
	

		points = append(points, point)
	}

	// Compute summary
	summary := gin.H{
		"total_points":     len(points),
		"valid_count":      0,
		"warning_count":    0,
		"counterfeit_count": 0,
	}

	validCount, counterfeitCount := 0, 0
	for _, p := range points {
		switch p.CounterfeitStatus {
		case string(models.CounterfeitStatusCounterfeit):
			counterfeitCount++
		default:
			// "valid" and legacy "warning" both count as valid
			validCount++
		}
	}
	summary["valid_count"] = validCount
	summary["warning_count"] = 0 // Deprecated: kept for API backward compat
	summary["counterfeit_count"] = counterfeitCount

	// Query geofence violations for this batch
	type GeoViolationPoint struct {
		Lat       float64   `json:"lat"`
		Lng       float64   `json:"lng"`
		Severity  string    `json:"severity"`
		CreatedAt time.Time `json:"created_at"`
	}
	var geoViolations []GeoViolationPoint
	h.DB.Model(&models.GeofenceViolation{}).
		Select("scan_latitude as lat, scan_longitude as lng, severity, created_at").
		Where("batch_id = ? AND tenant_id = ?", batchID, tenantUUID).
		Where("DATE(created_at) >= ? AND DATE(created_at) <= ?", fromDate, toDate).
		Where("scan_latitude != 0 OR scan_longitude != 0").
		Order("created_at DESC").
		Limit(limit).
		Scan(&geoViolations)

	summary["geofence_violation_count"] = len(geoViolations)

	utils.SuccessResponse(c, http.StatusOK, "Batch heatmap data", gin.H{
		"filters": gin.H{
			"from":   fromDate,
			"to":     toDate,
			"source": source,
		},
		"points":              points,
		"summary":             summary,
		"geofence_violations": geoViolations,
	})
}

// GetBatchAnalytics returns scan performance summary, trends, and top locations for a batch
func (h *QRBatchHandler) GetBatchAnalytics(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}

	batchID := c.Param("id")
	if _, err := uuid.Parse(batchID); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid batch ID", nil)
		return
	}

	// Fetch batch with Product to determine QR type
	var batch models.QRBatch
	if err := h.DB.Preload("Product").First(&batch, "id = ? AND tenant_id = ? AND deleted_at IS NULL", batchID, tenantUUID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Batch not found", nil)
		return
	}


	// ── Query 1: Summary Stats ──

	type SummaryResult struct {
		TotalScans       int64      `json:"total_scans"`
		UniqueQRScanned  int64      `json:"unique_qr_scanned"`
		ValidationScans  int64      `json:"validation_scans"`
		WarrantyScans    int64      `json:"warranty_scans"`
		CampaignScans    int64      `json:"campaign_scans"`
		UniqueCities     int64      `json:"unique_cities"`
		FirstScanAt      *time.Time `json:"first_scan_at"`
		LastScanAt       *time.Time `json:"last_scan_at"`
	}

	var summaryResult SummaryResult
	var summaryErr error

	summaryErr = h.DB.Raw(`
		SELECT
			COUNT(*) as total_scans,
			COUNT(DISTINCT i.qr_code_id) as unique_qr_scanned,
			COUNT(CASE WHEN i.interaction_subcategory = ? THEN 1 END) as validation_scans,
			COUNT(CASE WHEN i.interaction_subcategory = ? THEN 1 END) as warranty_scans,
			COUNT(CASE WHEN i.interaction_subcategory = ? THEN 1 END) as campaign_scans,
			COUNT(DISTINCT CASE WHEN i.geolocation->>'city' != '' THEN i.geolocation->>'city' END) as unique_cities,
			MIN(i.created_at) as first_scan_at,
			MAX(i.created_at) as last_scan_at
		FROM interactions i
		JOIN qr_codes qc ON qc.id = i.qr_code_id
		WHERE qc.batch_id = ? AND i.tenant_id = ?
			AND i.interaction_category = ?`,
		models.InteractionSubcategoryProductValidation,
		models.InteractionSubcategoryWarrantyActivation,
		models.InteractionSubcategoryCampaign,
		batchID, tenantUUID,
		models.InteractionCategoryEndUserAccess,
	).Scan(&summaryResult).Error


	if summaryErr != nil {
		sentryPkg.CaptureHandlerError(c, summaryErr, "qrbatch.GetBatchAnalytics.summary", sentryPkg.ErrorTypeDatabase, sentryPkg.SeverityMedium)
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to fetch analytics summary", summaryErr)
		return
	}

	// Get total QR codes in batch
	var totalQRCodes int64
	h.DB.Model(&models.QRCode{}).Where("batch_id = ?", batchID).Count(&totalQRCodes)


	// Compute scan rate
	var scanRate float64
	if totalQRCodes > 0 {
		scanRate = float64(summaryResult.UniqueQRScanned) / float64(totalQRCodes) * 100
		// Round to 1 decimal place
		scanRate = float64(int(scanRate*10)) / 10
	}

	// ── Query 2: Scan Trends ──

	// Determine granularity based on batch age
	granularity := "day"
	batchAge := time.Since(batch.CreatedAt)
	if batchAge.Hours() > 90*24 {
		granularity = "week"
	}

	type TrendRow struct {
		Date       time.Time `json:"date"`
		Validation int64     `json:"validation"`
		Warranty   int64     `json:"warranty"`
		Campaign   int64     `json:"campaign"`
		Total      int64     `json:"total"`
	}

	var trends []TrendRow

	trendTrunc := "day"
	if granularity == "week" {
		trendTrunc = "week"
	}

	var trendErr error
	trendErr = h.DB.Raw(fmt.Sprintf(`
		SELECT
			DATE_TRUNC('%s', i.created_at) as date,
			COUNT(*) FILTER (WHERE i.interaction_subcategory = ?) as validation,
			COUNT(*) FILTER (WHERE i.interaction_subcategory = ?) as warranty,
			COUNT(*) FILTER (WHERE i.interaction_subcategory = ?) as campaign,
			COUNT(*) as total
		FROM interactions i
		JOIN qr_codes qc ON qc.id = i.qr_code_id
		WHERE qc.batch_id = ? AND i.tenant_id = ?
			AND i.interaction_category = ?
		GROUP BY DATE_TRUNC('%s', i.created_at)
		ORDER BY date ASC`, trendTrunc, trendTrunc),
		models.InteractionSubcategoryProductValidation,
		models.InteractionSubcategoryWarrantyActivation,
		models.InteractionSubcategoryCampaign,
		batchID, tenantUUID,
		models.InteractionCategoryEndUserAccess,
	).Scan(&trends).Error


	if trendErr != nil {
		sentryPkg.CaptureHandlerError(c, trendErr, "qrbatch.GetBatchAnalytics.trends", sentryPkg.ErrorTypeDatabase, sentryPkg.SeverityMedium)
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to fetch scan trends", trendErr)
		return
	}

	// Format trends for response
	trendItems := make([]gin.H, 0, len(trends))
	for _, t := range trends {
		trendItems = append(trendItems, gin.H{
			"date":       t.Date.Format("2006-01-02"),
			"validation": t.Validation,
			"warranty":   t.Warranty,
			"campaign":   t.Campaign,
			"total":      t.Total,
		})
	}

	// ── Query 3: Top Locations ──

	type LocationRow struct {
		City    string `json:"city"`
		Country string `json:"country"`
		Count   int64  `json:"count"`
	}

	var locations []LocationRow
	var locationErr error

	locationErr = h.DB.Raw(`
		SELECT
			i.geolocation->>'city' as city,
			i.geolocation->>'country' as country,
			COUNT(*) as count
		FROM interactions i
		JOIN qr_codes qc ON qc.id = i.qr_code_id
		WHERE qc.batch_id = ? AND i.tenant_id = ?
			AND i.interaction_category = ?
			AND i.geolocation IS NOT NULL
			AND i.geolocation->>'city' IS NOT NULL
			AND i.geolocation->>'city' != ''
		GROUP BY i.geolocation->>'city', i.geolocation->>'country'
		ORDER BY count DESC
		LIMIT 5`,
		batchID, tenantUUID,
		models.InteractionCategoryEndUserAccess,
	).Scan(&locations).Error


	if locationErr != nil {
		sentryPkg.CaptureHandlerError(c, locationErr, "qrbatch.GetBatchAnalytics.locations", sentryPkg.ErrorTypeDatabase, sentryPkg.SeverityMedium)
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to fetch top locations", locationErr)
		return
	}

	// Compute percentages relative to geolocated scans (not total scans)
	var geolocatedTotal int64
	for _, loc := range locations {
		geolocatedTotal += loc.Count
	}

	locationItems := make([]gin.H, 0, len(locations))
	for _, loc := range locations {
		var pct float64
		if geolocatedTotal > 0 {
			pct = float64(loc.Count) / float64(geolocatedTotal) * 100
			pct = float64(int(pct*10)) / 10
		}
		locationItems = append(locationItems, gin.H{
			"city":       loc.City,
			"country":    loc.Country,
			"count":      loc.Count,
			"percentage": pct,
		})
	}

	utils.SuccessResponse(c, http.StatusOK, "Batch analytics data", gin.H{
		"summary": gin.H{
			"total_scans":       summaryResult.TotalScans,
			"unique_qr_scanned": summaryResult.UniqueQRScanned,
			"total_qr_codes":    totalQRCodes,
			"scan_rate":         scanRate,
			"validation_scans":  summaryResult.ValidationScans,
			"warranty_scans":    summaryResult.WarrantyScans,
			"campaign_scans":    summaryResult.CampaignScans,
			"unique_cities":     summaryResult.UniqueCities,
			"first_scan_at":     summaryResult.FirstScanAt,
			"last_scan_at":      summaryResult.LastScanAt,
		},
		"trends":           trendItems,
		"trend_granularity": granularity,
		"top_locations":    locationItems,
	})
}

// DeleteQRBatch soft deletes a QR batch (only if no scans exist and not currently generating)
func (h *QRBatchHandler) DeleteQRBatch(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}

	batchID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid batch ID", nil)
		return
	}

	// Find active (non-deleted) batch
	var batch models.QRBatch
	if err := h.DB.First(&batch, "id = ? AND tenant_id = ? AND deleted_at IS NULL", batchID, tenantUUID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Batch not found", nil)
		return
	}

	// Block deletion while generation is in progress
	if batch.Status.IsInProgress() {
		utils.ErrorResponse(c, http.StatusConflict,
			"Cannot delete batch while generation is in progress. Please wait until generation completes.", nil)
		return
	}

	// Check if any QR code in this batch has been scanned.
	// Redundant tenant filter via join for defense-in-depth — the batch lookup above
	// already enforces tenant ownership, but explicit scoping prevents regressions.
	var scanCount int64
	h.DB.Model(&models.Interaction{}).
		Joins("JOIN qr_codes ON qr_codes.id = interactions.qr_code_id").
		Joins("JOIN qr_batches ON qr_batches.id = qr_codes.batch_id").
		Where("qr_codes.batch_id = ? AND qr_batches.tenant_id = ?", batchID, tenantUUID).
		Count(&scanCount)

	if scanCount > 0 {
		utils.ErrorResponse(c, http.StatusConflict, fmt.Sprintf("Cannot delete batch with %d existing scan(s)", scanCount), nil)
		return
	}

	// Soft delete
	now := time.Now()
	if err := h.DB.Model(&batch).Update("deleted_at", &now).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to delete batch", nil)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Batch deleted", nil)
}

// RestoreQRBatch restores a soft-deleted QR batch
func (h *QRBatchHandler) RestoreQRBatch(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}

	batchID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid batch ID", nil)
		return
	}

	// Find deleted batch
	var batch models.QRBatch
	if err := h.DB.First(&batch, "id = ? AND tenant_id = ? AND deleted_at IS NOT NULL", batchID, tenantUUID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Batch not found or not deleted", nil)
		return
	}

	// Restore
	if err := h.DB.Model(&batch).Update("deleted_at", nil).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to restore batch", nil)
		return
	}

	// Reload with preloads
	h.DB.Preload("Product").Preload("Campaign").Preload("CreatedByStaff").
		First(&batch, "id = ?", batch.ID)

	utils.SuccessResponse(c, http.StatusOK, "Batch restored", batch)
}
