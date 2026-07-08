package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gamatritunggal/smartscan/backend/internal/config"
	"github.com/gamatritunggal/smartscan/backend/internal/models"
	"github.com/gamatritunggal/smartscan/backend/internal/utils"
	"gorm.io/gorm"
)

type WarrantyAdminHandler struct {
	DB  *gorm.DB
	Cfg *config.Config
}

func NewWarrantyAdminHandler(db *gorm.DB, cfg *config.Config) *WarrantyAdminHandler {
	return &WarrantyAdminHandler{DB: db, Cfg: cfg}
}

// ListWarranties returns all warranty activations for the tenant
// GET /api/v1/tenant/warranties
// Query params:
//   - status: active|expired|all (default: all)
//   - product_id: UUID (optional filter by product)
//   - search: string (search by customer_name or customer_email)
//   - from_date: YYYY-MM-DD (filter by activation date from)
//   - to_date: YYYY-MM-DD (filter by activation date to)
//   - page: int (default: 1)
//   - limit: int (default: 20, max: 100)
func (h *WarrantyAdminHandler) ListWarranties(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}

	page := utils.GetPageQuery(c)
	limit := utils.GetLimitQuery(c, 20)
	status := c.DefaultQuery("status", "all")
	productID := c.Query("product_id")
	search := c.Query("search")
	fromDate := c.Query("from_date")
	toDate := c.Query("to_date")

	offset := (page - 1) * limit

	// Base query: join through qr_codes and qr_batches to filter by tenant
	query := h.DB.Model(&models.WarrantyActivation{}).
		Joins("JOIN qr_codes ON qr_codes.id = warranty_activations.qr_code_id").
		Joins("JOIN qr_batches ON qr_batches.id = qr_codes.batch_id").
		Where("qr_batches.tenant_id = ?", tenantUUID)

	now := time.Now().UTC()

	// Filter by status (active = not expired, expired = past expiry date)
	switch status {
	case "active":
		query = query.Where("warranty_activations.warranty_expiry_date > ?", now)
	case "expired":
		query = query.Where("warranty_activations.warranty_expiry_date <= ?", now)
	}

	// Filter by product
	if productID != "" {
		if productUUID, err := uuid.Parse(productID); err == nil {
			query = query.Where("qr_batches.product_id = ?", productUUID)
		}
	}

	// Search by customer name or email
	if search != "" {
		query = query.Where(
			"warranty_activations.customer_name ILIKE ? OR warranty_activations.customer_email ILIKE ?",
			"%"+search+"%", "%"+search+"%",
		)
	}

	// Date range filter (by activated_at)
	if fromDate != "" {
		if t, err := time.Parse("2006-01-02", fromDate); err == nil {
			query = query.Where("warranty_activations.activated_at >= ?", t)
		}
	}
	if toDate != "" {
		if t, err := time.Parse("2006-01-02", toDate); err == nil {
			// Add 1 day to include the entire end date
			query = query.Where("warranty_activations.activated_at < ?", t.AddDate(0, 0, 1))
		}
	}

	// Count total before pagination
	var total int64
	query.Count(&total)

	// Fetch with relations
	var warranties []models.WarrantyActivation
	query.Preload("QRCode.Batch.Product").
		Preload("Province").
		Preload("City").
		Order("warranty_activations.activated_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&warranties)

	// Add computed is_expired field to response
	type WarrantyResponse struct {
		models.WarrantyActivation
		IsExpired bool `json:"is_expired"`
	}

	var response []WarrantyResponse
	for _, w := range warranties {
		wr := WarrantyResponse{
			WarrantyActivation: w,
			IsExpired:          w.WarrantyExpiryDate != nil && w.WarrantyExpiryDate.Before(now),
		}
		response = append(response, wr)
	}

	utils.SuccessResponse(c, http.StatusOK, "Warranties retrieved", gin.H{
		"warranties": response,
		"pagination": utils.PaginationMeta(page, limit, total),
	})
}

// GetWarrantyDetail returns a single warranty activation detail
// GET /api/v1/tenant/warranties/:id
func (h *WarrantyAdminHandler) GetWarrantyDetail(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}

	warrantyID := c.Param("id")
	id, err := uuid.Parse(warrantyID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid warranty ID", err)
		return
	}

	var warranty models.WarrantyActivation
	if err := h.DB.
		Preload("QRCode.Batch.Product").
		Preload("Province").
		Preload("City").
		Joins("JOIN qr_codes ON qr_codes.id = warranty_activations.qr_code_id").
		Joins("JOIN qr_batches ON qr_batches.id = qr_codes.batch_id").
		Where("warranty_activations.id = ? AND qr_batches.tenant_id = ?", id, tenantUUID).
		First(&warranty).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Warranty not found", err)
		return
	}

	now := time.Now().UTC()
	isExpired := warranty.WarrantyExpiryDate != nil && warranty.WarrantyExpiryDate.Before(now)

	utils.SuccessResponse(c, http.StatusOK, "Warranty retrieved", gin.H{
		"warranty":   warranty,
		"is_expired": isExpired,
	})
}

// GetWarrantyStats returns warranty statistics for the tenant
// GET /api/v1/tenant/warranties/stats
func (h *WarrantyAdminHandler) GetWarrantyStats(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}

	now := time.Now().UTC()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	var stats struct {
		TotalWarranties       int64 `json:"total_warranties"`
		ActiveWarranties      int64 `json:"active_warranties"`
		ExpiredWarranties     int64 `json:"expired_warranties"`
		WarrantiesThisMonth   int64 `json:"warranties_this_month"`
		ExpiringIn30Days      int64 `json:"expiring_in_30_days"`
	}

	baseQuery := h.DB.Model(&models.WarrantyActivation{}).
		Joins("JOIN qr_codes ON qr_codes.id = warranty_activations.qr_code_id").
		Joins("JOIN qr_batches ON qr_batches.id = qr_codes.batch_id").
		Where("qr_batches.tenant_id = ?", tenantUUID)

	// Total warranties
	baseQuery.Count(&stats.TotalWarranties)

	// Active (not expired)
	h.DB.Model(&models.WarrantyActivation{}).
		Joins("JOIN qr_codes ON qr_codes.id = warranty_activations.qr_code_id").
		Joins("JOIN qr_batches ON qr_batches.id = qr_codes.batch_id").
		Where("qr_batches.tenant_id = ? AND warranty_activations.warranty_expiry_date > ?", tenantUUID, now).
		Count(&stats.ActiveWarranties)

	// Expired
	h.DB.Model(&models.WarrantyActivation{}).
		Joins("JOIN qr_codes ON qr_codes.id = warranty_activations.qr_code_id").
		Joins("JOIN qr_batches ON qr_batches.id = qr_codes.batch_id").
		Where("qr_batches.tenant_id = ? AND warranty_activations.warranty_expiry_date <= ?", tenantUUID, now).
		Count(&stats.ExpiredWarranties)

	// This month registrations
	h.DB.Model(&models.WarrantyActivation{}).
		Joins("JOIN qr_codes ON qr_codes.id = warranty_activations.qr_code_id").
		Joins("JOIN qr_batches ON qr_batches.id = qr_codes.batch_id").
		Where("qr_batches.tenant_id = ? AND warranty_activations.activated_at >= ?", tenantUUID, startOfMonth).
		Count(&stats.WarrantiesThisMonth)

	// Expiring in 30 days
	in30Days := now.AddDate(0, 0, 30)
	h.DB.Model(&models.WarrantyActivation{}).
		Joins("JOIN qr_codes ON qr_codes.id = warranty_activations.qr_code_id").
		Joins("JOIN qr_batches ON qr_batches.id = qr_codes.batch_id").
		Where("qr_batches.tenant_id = ? AND warranty_activations.warranty_expiry_date > ? AND warranty_activations.warranty_expiry_date <= ?", tenantUUID, now, in30Days).
		Count(&stats.ExpiringIn30Days)

	utils.SuccessResponse(c, http.StatusOK, "Warranty stats retrieved", stats)
}

// ExportWarrantiesToCSV exports warranty activations to CSV
// GET /api/v1/tenant/warranties/export
func (h *WarrantyAdminHandler) ExportWarrantiesToCSV(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}

	// Parse timezone for export
	exportTZ, err := utils.ParseExportTimezone(c.Query("tz"))
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid timezone: "+c.Query("tz"), nil)
		return
	}

	status := c.DefaultQuery("status", "all")
	productID := c.Query("product_id")
	fromDate := c.Query("from_date")
	toDate := c.Query("to_date")

	// Build query
	query := h.DB.Model(&models.WarrantyActivation{}).
		Joins("JOIN qr_codes ON qr_codes.id = warranty_activations.qr_code_id").
		Joins("JOIN qr_batches ON qr_batches.id = qr_codes.batch_id").
		Where("qr_batches.tenant_id = ?", tenantUUID)

	now := time.Now().UTC()

	// Filter by status
	switch status {
	case "active":
		query = query.Where("warranty_activations.warranty_expiry_date > ?", now)
	case "expired":
		query = query.Where("warranty_activations.warranty_expiry_date <= ?", now)
	}

	// Filter by product
	if productID != "" {
		if productUUID, err := uuid.Parse(productID); err == nil {
			query = query.Where("qr_batches.product_id = ?", productUUID)
		}
	}

	// Date range filter
	if fromDate != "" {
		if t, err := time.Parse("2006-01-02", fromDate); err == nil {
			query = query.Where("warranty_activations.activated_at >= ?", t)
		}
	}
	if toDate != "" {
		if t, err := time.Parse("2006-01-02", toDate); err == nil {
			query = query.Where("warranty_activations.activated_at < ?", t.AddDate(0, 0, 1))
		}
	}

	// Fetch warranties with relations (limit to prevent memory exhaustion)
	const maxExportRecords = 50000
	var warranties []models.WarrantyActivation
	query.Preload("QRCode.Batch.Product").
		Preload("Province").
		Preload("City").
		Order("warranty_activations.activated_at DESC").
		Limit(maxExportRecords).
		Find(&warranties)

	// Generate CSV
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Disposition", "attachment; filename=warranties_export.csv")
	c.Header("Content-Type", "text/csv; charset=utf-8")

	// Write BOM for Excel UTF-8 compatibility
	c.Writer.Write([]byte{0xEF, 0xBB, 0xBF})

	// Write header
	c.Writer.WriteString("Customer Name,Email,Phone,Product,Batch,Purchase Date,Purchase Store,Address,Province,City,Registered At" + utils.TimezoneHeaderSuffix(exportTZ) + ",Expiry Date,Status\n")

	// Write data rows
	for _, w := range warranties {
		productName := ""
		batchName := ""
		if w.QRCode != nil && w.QRCode.Batch != nil {
			batchName = w.QRCode.Batch.BatchName
			if w.QRCode.Batch.Product != nil {
				productName = w.QRCode.Batch.Product.ProductName
			}
		}

		provinceName := ""
		if w.Province != nil {
			provinceName = w.Province.Name
		}

		cityName := ""
		if w.City != nil {
			cityName = w.City.Name
		}

		purchaseDate := ""
		if w.PurchaseDate != nil {
			purchaseDate = w.PurchaseDate.Format("2006-01-02")
		}

		expiryDate := ""
		isExpired := false
		if w.WarrantyExpiryDate != nil {
			expiryDate = w.WarrantyExpiryDate.Format("2006-01-02")
			isExpired = w.WarrantyExpiryDate.Before(now)
		}

		status := "Active"
		if isExpired {
			status = "Expired"
		}

		// Escape CSV fields
		row := escapeCSVRow(
			w.CustomerName,
			w.CustomerEmail,
			w.CustomerPhone,
			productName,
			batchName,
			purchaseDate,
			w.PurchaseStore,
			w.Address,
			provinceName,
			cityName,
			utils.FormatExportTime(w.ActivatedAt, exportTZ, "2006-01-02 15:04"),
			expiryDate,
			status,
		)
		c.Writer.WriteString(row + "\n")
	}
}

// escapeCSVRow escapes and joins CSV fields
func escapeCSVRow(fields ...string) string {
	escaped := make([]string, len(fields))
	for i, field := range fields {
		// Prevent CSV injection by prefixing dangerous characters
		if len(field) > 0 {
			firstChar := field[0]
			if firstChar == '=' || firstChar == '+' || firstChar == '-' || firstChar == '@' ||
				firstChar == '\t' || firstChar == '\r' || firstChar == '\n' {
				field = "'" + field
			}
		}
		// Escape quotes and wrap in quotes if contains comma, quote, or newline
		if containsSpecialChar(field) {
			field = "\"" + escapeQuotes(field) + "\""
		}
		escaped[i] = field
	}
	return joinCSV(escaped)
}

func containsSpecialChar(s string) bool {
	for _, c := range s {
		if c == ',' || c == '"' || c == '\n' || c == '\r' {
			return true
		}
	}
	return false
}

func escapeQuotes(s string) string {
	result := ""
	for _, c := range s {
		if c == '"' {
			result += "\"\""
		} else {
			result += string(c)
		}
	}
	return result
}

func joinCSV(fields []string) string {
	result := ""
	for i, f := range fields {
		if i > 0 {
			result += ","
		}
		result += f
	}
	return result
}
