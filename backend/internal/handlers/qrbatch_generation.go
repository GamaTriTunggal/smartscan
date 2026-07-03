package handlers

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gamatritunggal/smartscan/backend/internal/models"
	"github.com/gamatritunggal/smartscan/backend/internal/queue"
	sentryPkg "github.com/gamatritunggal/smartscan/backend/internal/sentry"
	"github.com/gamatritunggal/smartscan/backend/internal/utils"
	"gorm.io/gorm"
)

// GenerationStatusResponse is the response for generation status endpoints
type GenerationStatusResponse struct {
	BatchID         string                `json:"batch_id"`
	BatchName       string                `json:"batch_name"`
	Status          models.QRBatchStatus  `json:"status"`
	TotalQRCount    int                   `json:"total_qr_count"`
	GeneratedCount  int                   `json:"generated_count"`
	ProgressPercent int                   `json:"progress_percent"` // 0-100
	StartedAt       *time.Time            `json:"started_at,omitempty"`
	CompletedAt     *time.Time            `json:"completed_at,omitempty"`
	ETASeconds      *int                  `json:"eta_seconds,omitempty"`
	ErrorMessage    string                `json:"error_message,omitempty"`
}

// buildGenerationStatusResponse builds the status response from batch and queue record
func buildGenerationStatusResponse(batch *models.QRBatch, queueRec *models.QRGenerationQueue) GenerationStatusResponse {
	resp := GenerationStatusResponse{
		BatchID:      batch.ID.String(),
		BatchName:    batch.BatchName,
		Status:       batch.Status,
		TotalQRCount: batch.QRCount,
	}

	if queueRec != nil {
		resp.GeneratedCount = queueRec.GeneratedCount
		resp.StartedAt = queueRec.StartedAt
		resp.CompletedAt = queueRec.CompletedAt
		resp.ErrorMessage = queueRec.ErrorMessage

		// Calculate progress percent (cap at 100)
		if batch.QRCount > 0 {
			pct := (queueRec.GeneratedCount * 100) / batch.QRCount
			if pct > 100 {
				pct = 100
			}
			resp.ProgressPercent = pct
		}

		// Calculate ETA if in progress
		if batch.Status == models.QRBatchStatusProcessing && queueRec.StartedAt != nil && queueRec.GeneratedCount > 0 {
			elapsed := time.Since(*queueRec.StartedAt).Seconds()
			rate := float64(queueRec.GeneratedCount) / elapsed // QR codes per second
			if rate > 0 {
				remaining := batch.QRCount - queueRec.GeneratedCount
				etaSec := int(float64(remaining) / rate)
				if etaSec < 0 {
					etaSec = 0
				}
				resp.ETASeconds = &etaSec
			}
		}
	} else {
		// No queue record yet
		if batch.Status == models.QRBatchStatusCompleted {
			// Legacy sync-generated batch or static product
			resp.GeneratedCount = batch.QRCount
			resp.ProgressPercent = 100
		}
	}

	return resp
}

// GetGenerationStatus returns the current generation status of a specific batch
// Route: GET /tenant/qr-batches/:id/generation-status
func (h *QRBatchHandler) GetGenerationStatus(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}

	batchID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid batch ID", err)
		return
	}

	// Find batch (enforce tenant isolation)
	var batch models.QRBatch
	if err := h.DB.First(&batch, "id = ? AND tenant_id = ? AND deleted_at IS NULL", batchID, tenantUUID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.ErrorResponse(c, http.StatusNotFound, "QR batch not found", nil)
			return
		}
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to load batch", err)
		return
	}

	// Load queue record (may not exist for legacy batches)
	var queueRec *models.QRGenerationQueue
	var qr models.QRGenerationQueue
	if err := h.DB.Where("batch_id = ?", batchID).First(&qr).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to load queue record", err)
			return
		}
		// Not found is OK — legacy batch without queue record
	} else {
		queueRec = &qr
	}

	resp := buildGenerationStatusResponse(&batch, queueRec)
	utils.SuccessResponse(c, http.StatusOK, "Generation status retrieved", resp)
}

// GetActiveGenerations returns all batches currently in non-terminal state for the tenant.
// Used by the frontend global polling composable to track active generations across pages.
// Route: GET /tenant/qr-batches/active-generations
func (h *QRBatchHandler) GetActiveGenerations(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}

	// Load active batches (not in terminal state)
	// Limit to prevent abuse / excessive response size
	activeStatuses := []models.QRBatchStatus{
		models.QRBatchStatusPendingQueue,
		models.QRBatchStatusQueued,
		models.QRBatchStatusProcessing,
	}

	const activeGenerationsLimit = 200
	var batches []models.QRBatch
	if err := h.DB.
		Where("tenant_id = ? AND deleted_at IS NULL AND status IN ?", tenantUUID, activeStatuses).
		Order("created_at DESC").
		Limit(activeGenerationsLimit).
		Find(&batches).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to load active generations", err)
		return
	}

	if len(batches) == 0 {
		utils.SuccessResponse(c, http.StatusOK, "No active generations", gin.H{
			"active_generations": []GenerationStatusResponse{},
			"truncated":          false,
		})
		return
	}

	// Load all queue records in one query to avoid N+1
	batchIDs := make([]uuid.UUID, len(batches))
	for i, b := range batches {
		batchIDs[i] = b.ID
	}

	var queueRecords []models.QRGenerationQueue
	h.DB.Where("batch_id IN ?", batchIDs).Find(&queueRecords)

	queueMap := make(map[uuid.UUID]*models.QRGenerationQueue, len(queueRecords))
	for i := range queueRecords {
		queueMap[queueRecords[i].BatchID] = &queueRecords[i]
	}

	// Build response
	responses := make([]GenerationStatusResponse, len(batches))
	for i, batch := range batches {
		queueRec := queueMap[batch.ID]
		responses[i] = buildGenerationStatusResponse(&batch, queueRec)
	}

	utils.SuccessResponse(c, http.StatusOK, "Active generations retrieved", gin.H{
		"active_generations": responses,
		"truncated":          len(batches) >= activeGenerationsLimit,
	})
}

// RetryFailedGeneration re-enqueues a failed batch for retry.
// Uses existing generated_count to resume from last successful chunk.
// Route: POST /tenant/qr-batches/:id/retry-generation
func (h *QRBatchHandler) RetryFailedGeneration(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}

	batchID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid batch ID", err)
		return
	}

	// Atomic status transition: only transition from `failed` → `pending_queue`.
	// Uses RowsAffected to detect a race with a concurrent retry click.
	res := h.DB.Model(&models.QRBatch{}).
		Where("id = ? AND tenant_id = ? AND deleted_at IS NULL AND status = ?",
			batchID, tenantUUID, models.QRBatchStatusFailed).
		Update("status", models.QRBatchStatusPendingQueue)
	if res.Error != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to update batch status", res.Error)
		return
	}
	if res.RowsAffected == 0 {
		// Either the batch doesn't exist, belongs to another tenant, is deleted,
		// or is not currently in `failed` state. All cases return the same error to avoid leaking.
		var batch models.QRBatch
		if loadErr := h.DB.First(&batch, "id = ? AND tenant_id = ? AND deleted_at IS NULL", batchID, tenantUUID).Error; loadErr != nil {
			if errors.Is(loadErr, gorm.ErrRecordNotFound) {
				utils.ErrorResponse(c, http.StatusNotFound, "QR batch not found", nil)
				return
			}
			utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to load batch", loadErr)
			return
		}
		utils.ErrorResponse(c, http.StatusConflict,
			"Batch is not in failed state (current status: "+string(batch.Status)+"). It may already be retrying.", nil)
		return
	}

	// Load batch for enqueue data (guaranteed to exist since RowsAffected was 1)
	var batch models.QRBatch
	if err := h.DB.First(&batch, "id = ?", batchID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to load batch after update", err)
		return
	}

	// Reset queue record: clear error, clear started_at (so ETA is recalculated from scratch),
	// reset status. Keep generated_count so the worker resumes from where it left off.
	if err := h.DB.Model(&models.QRGenerationQueue{}).
		Where("batch_id = ?", batchID).
		Updates(map[string]interface{}{
			"status":        models.QRGenerationQueueStatusQueued,
			"error_message": "",
			"completed_at":  nil,
			"started_at":    nil, // reset so worker assigns a fresh start time (correct ETA)
			"worker_id":     "",
		}).Error; err != nil {
		// Not fatal — scanner will reconcile eventually
		sentryPkg.CaptureHandlerError(c, err, "qrbatch.RetryFailedGeneration.resetQueue",
			sentryPkg.ErrorTypeDatabase, sentryPkg.SeverityLow)
	}

	// Try to enqueue immediately if queue is available
	finalStatus := models.QRBatchStatusPendingQueue
	qrQueue := getQRGenerationQueue()
	if qrQueue != nil {
		maxRetries := 5
		if h.Cfg != nil && h.Cfg.QRGeneration.MaxRetries > 0 {
			maxRetries = h.Cfg.QRGeneration.MaxRetries
		}
		job := queue.NewQRGenerationJob(
			batch.ID.String(),
			tenantUUID.String(),
			batch.QRCount,
			batch.Prefix,
			batch.Suffix,
			maxRetries,
		)
		enqueueCtx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		if err := qrQueue.Enqueue(enqueueCtx, job); err != nil {
			sentryPkg.CaptureHandlerError(c, err, "qrbatch.RetryFailedGeneration.enqueue",
				sentryPkg.ErrorTypeExternal, sentryPkg.SeverityMedium)
			// Fall through — status=pending_queue, scanner will retry
		} else {
			finalStatus = models.QRBatchStatusQueued
		}
	}

	// Update batch to final status (queued if enqueue succeeded, else stay pending_queue)
	if finalStatus != models.QRBatchStatusPendingQueue {
		if err := h.DB.Model(&batch).Update("status", finalStatus).Error; err != nil {
			sentryPkg.CaptureHandlerError(c, err, "qrbatch.RetryFailedGeneration.finalStatus",
				sentryPkg.ErrorTypeDatabase, sentryPkg.SeverityLow)
		}
	}

	// Reload and return
	h.DB.First(&batch, "id = ?", batchID)
	utils.SuccessResponse(c, http.StatusOK, "Batch retry queued", batch)
}
