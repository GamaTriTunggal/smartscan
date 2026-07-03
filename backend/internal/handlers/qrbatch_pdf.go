package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/gamatritunggal/smartscan/backend/internal/models"
	"github.com/gamatritunggal/smartscan/backend/internal/pdf"
	"github.com/gamatritunggal/smartscan/backend/internal/utils"
)

// pdfExportChunkSize controls how many codes are loaded from the DB at once
// while rendering — memory stays flat no matter how large the batch is.
const pdfExportChunkSize = 500

// ExportQRCodesPDF streams a print-ready A4 label sheet.
// GET /tenant/qr-batches/:id/export/pdf?label=25|38|50&start=1&end=5000
//
// Small businesses print these themselves; large runs should use the CSV
// export with a label printer. The per-file cap (PDF_EXPORT_MAX_CODES,
// default 10000) keeps a single request bounded — bigger batches are
// exported in ranges.
func (h *QRBatchHandler) ExportQRCodesPDF(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}

	// Share the export slot pool with CSV/Excel so exports can't stampede.
	select {
	case exportSemaphore <- struct{}{}:
		defer func() { <-exportSemaphore }()
	case <-time.After(exportQueueTimeout):
		utils.ErrorResponse(c, http.StatusServiceUnavailable,
			"The server is busy with another export. Please try again shortly.", nil)
		return
	}

	batchID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid batch ID", err)
		return
	}

	var batch models.QRBatch
	if err := h.DB.Preload("Product").First(&batch, "id = ? AND tenant_id = ? AND deleted_at IS NULL", batchID, tenantUUID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "QR batch not found", err)
		return
	}
	if batch.Status != models.QRBatchStatusCompleted {
		utils.ErrorResponse(c, http.StatusBadRequest, "Batch generation has not completed yet", nil)
		return
	}

	spec, ok := pdf.LabelPresets[c.DefaultQuery("label", "25")]
	if !ok {
		utils.ErrorResponse(c, http.StatusBadRequest, "Unknown label size — use 25, 38 or 50", nil)
		return
	}

	maxCodes := 10000
	if h.Cfg != nil && h.Cfg.QRGeneration.PDFExportMaxCodes > 0 {
		maxCodes = h.Cfg.QRGeneration.PDFExportMaxCodes
	}

	// Optional 1-based range within the batch (ordered by creation).
	start := 1
	if v, err := strconv.Atoi(c.DefaultQuery("start", "1")); err == nil && v > 0 {
		start = v
	}
	end := batch.QRCount
	if v, err := strconv.Atoi(c.Query("end")); err == nil && v >= start {
		end = v
	}
	if end-start+1 > maxCodes {
		utils.ErrorResponse(c, http.StatusBadRequest, fmt.Sprintf(
			"A single PDF is limited to %d codes. Export a range (start/end) or use the CSV export for print vendors.",
			maxCodes), nil)
		return
	}

	baseURL := h.Cfg.FrontendURL
	if baseURL == "" {
		baseURL = "http://localhost:3000" // FRONTEND_URL not set — dev fallback
	}

	gen := pdf.NewGenerator(spec)

	offset := start - 1
	remaining := end - start + 1
	for remaining > 0 {
		limit := pdfExportChunkSize
		if remaining < limit {
			limit = remaining
		}
		var codes []models.QRCode
		if err := h.DB.Select("qr_uuid, qr_code").
			Where("batch_id = ?", batchID).
			Order("created_at, id").
			Offset(offset).Limit(limit).
			Find(&codes).Error; err != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to fetch QR codes", err)
			return
		}
		if len(codes) == 0 {
			break
		}
		for _, code := range codes {
			if err := gen.Add(pdf.Code{
				Content: fmt.Sprintf("%s/s/%s", baseURL, utils.UUIDToBase58(code.QRUUID)),
				Label:   code.QRCode,
			}); err != nil {
				utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to render QR code", err)
				return
			}
		}
		offset += len(codes)
		remaining -= len(codes)
	}

	filename := fmt.Sprintf("%s-labels-%d-%d.pdf", batch.BatchCode, start, end)
	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	if err := gen.Output(c.Writer); err != nil {
		// headers already sent; log only
		fmt.Printf("[PDF-EXPORT] write failed for batch %s: %v\n", batchID, err)
	}
}
