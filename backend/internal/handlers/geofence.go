package handlers

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gamatritunggal/smartscan/backend/internal/config"
	"github.com/gamatritunggal/smartscan/backend/internal/models"
	"github.com/gamatritunggal/smartscan/backend/internal/utils"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

type GeofenceHandler struct {
	DB  *gorm.DB
	Cfg *config.Config
}

func NewGeofenceHandler(db *gorm.DB, cfg *config.Config) *GeofenceHandler {
	return &GeofenceHandler{DB: db, Cfg: cfg}
}

// resolveBatchIDs returns batch IDs matching the active filter from query params.
// Supports single batch_id OR product_id+geofence_label (area filter).
// Returns nil if no batch filter is active.
func (h *GeofenceHandler) resolveBatchIDs(c *gin.Context, tenantUUID uuid.UUID) []uuid.UUID {
	if batchID := c.Query("batch_id"); batchID != "" {
		if bid, err := uuid.Parse(batchID); err == nil {
			return []uuid.UUID{bid}
		}
	}
	productID := c.Query("product_id")
	geofenceLabel := c.Query("geofence_label")
	if productID != "" {
		if pid, err := uuid.Parse(productID); err == nil {
			var ids []uuid.UUID
			q := h.DB.Model(&models.QRBatch{}).Select("id").
				Where("product_id = ? AND tenant_id = ? AND geofence_enabled = true AND deleted_at IS NULL",
					pid, tenantUUID)
			if geofenceLabel != "" {
				q = q.Where("geofence_label = ?", geofenceLabel)
			}
			q.Pluck("id", &ids)
			return ids
		}
	}
	return nil
}

// applyViolationFilters applies shared query filters for geofence violations.
func (h *GeofenceHandler) applyViolationFilters(c *gin.Context, query *gorm.DB) *gorm.DB {
	if severity := c.Query("severity"); severity != "" {
		allowed := map[string]bool{"low": true, "medium": true, "high": true, "critical": true}
		if allowed[severity] {
			query = query.Where("severity = ?", severity)
		}
	}
	if batchID := c.Query("batch_id"); batchID != "" {
		if bid, err := uuid.Parse(batchID); err == nil {
			query = query.Where("batch_id = ?", bid)
		}
	}
	// Area filter: product_id + optional geofence_label → resolve matching batch IDs
	productID := c.Query("product_id")
	geofenceLabel := c.Query("geofence_label")
	if productID != "" {
		if pid, err := uuid.Parse(productID); err == nil {
			tenantUUID, _ := utils.GetTenantUUID(c)
			batchSubquery := h.DB.Model(&models.QRBatch{}).Select("id").
				Where("product_id = ? AND tenant_id = ? AND geofence_enabled = true AND deleted_at IS NULL",
					pid, tenantUUID)
			if geofenceLabel != "" {
				batchSubquery = batchSubquery.Where("geofence_label = ?", geofenceLabel)
			}
			query = query.Where("batch_id IN (?)", batchSubquery)
		}
	}
	// Support both from/to (new preset-based) and date_from/date_to (legacy)
	dateFrom := c.Query("from")
	if dateFrom == "" {
		dateFrom = c.Query("date_from")
	}
	if dateFrom != "" {
		if t, err := time.Parse(time.DateOnly, dateFrom); err == nil {
			query = query.Where("geofence_violations.created_at >= ?", t)
		}
	}
	dateTo := c.Query("to")
	if dateTo == "" {
		dateTo = c.Query("date_to")
	}
	if dateTo != "" {
		if t, err := time.Parse(time.DateOnly, dateTo); err == nil {
			query = query.Where("geofence_violations.created_at < ?", t.AddDate(0, 0, 1))
		}
	}
	return query
}

// GetGeofenceAreas returns product-grouped geofence areas for the filter dropdown.
// GET /api/v1/tenant/geofence/areas
func (h *GeofenceHandler) GetGeofenceAreas(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}

	type AreaResult struct {
		ProductID       uuid.UUID `json:"product_id"`
		ProductName     string    `json:"product_name"`
		GeofenceLabel   string    `json:"geofence_label"`
		BatchCount      int64     `json:"batch_count"`
		TotalViolations int64     `json:"total_violations"`
	}

	var areas []AreaResult
	h.DB.Model(&models.QRBatch{}).
		Select(`products.id AS product_id, products.product_name,
			qr_batches.geofence_label,
			COUNT(DISTINCT qr_batches.id) AS batch_count,
			COALESCE(SUM(vc.violation_count), 0) AS total_violations`).
		Joins("JOIN products ON products.id = qr_batches.product_id AND products.tenant_id = ?", tenantUUID).
		Joins(`LEFT JOIN (
			SELECT batch_id, COUNT(*) AS violation_count
			FROM geofence_violations
			WHERE tenant_id = ?
			GROUP BY batch_id
		) vc ON vc.batch_id = qr_batches.id`, tenantUUID).
		Where("qr_batches.tenant_id = ? AND qr_batches.geofence_enabled = true AND qr_batches.deleted_at IS NULL AND qr_batches.geofence_label != ''", tenantUUID).
		Group("products.id, products.product_name, qr_batches.geofence_label").
		Order("products.product_name ASC, qr_batches.geofence_label ASC").
		Scan(&areas)

	if areas == nil {
		areas = []AreaResult{}
	}

	utils.SuccessResponse(c, http.StatusOK, "Geofence areas", gin.H{
		"areas": areas,
	})
}

// ListGeofenceViolations returns paginated list of geofence violations for a tenant.
// GET /api/v1/tenant/geofence/violations
// Query params: page, limit, severity, batch_id, product_id, geofence_label, date_from, date_to
func (h *GeofenceHandler) ListGeofenceViolations(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}

	page := utils.GetPageQuery(c)
	limit := utils.GetLimitQuery(c, 20)
	offset := (page - 1) * limit

	query := h.DB.Model(&models.GeofenceViolation{}).Where("geofence_violations.tenant_id = ?", tenantUUID)
	query = h.applyViolationFilters(c, query)

	var total int64
	query.Count(&total)

	var violations []models.GeofenceViolation
	if err := query.Preload("Batch", func(db *gorm.DB) *gorm.DB {
		return db.Select("id, batch_name, batch_code, product_id")
	}).Preload("Batch.Product", func(db *gorm.DB) *gorm.DB {
		return db.Select("id, product_name, product_code")
	}).Order("geofence_violations.created_at DESC").
		Offset(offset).Limit(limit).
		Find(&violations).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to fetch violations", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Geofence violations", gin.H{
		"violations": violations,
		"pagination": utils.PaginationMeta(page, limit, total),
	})
}

// calcPreviousPeriod computes previous period date range and comparison label for a given preset.
func calcPreviousPeriod(preset, fromDate, toDate string) (prevFrom, prevTo, comparisonLabel string, err error) {
	currentStart, err := time.Parse("2006-01-02", fromDate)
	if err != nil {
		return "", "", "", fmt.Errorf("invalid 'from' date: %s", fromDate)
	}
	currentEnd, err := time.Parse("2006-01-02", toDate)
	if err != nil {
		return "", "", "", fmt.Errorf("invalid 'to' date: %s", toDate)
	}
	_ = currentEnd // used in switch cases below

	switch preset {
	case "this_month":
		prevMonthStart := time.Date(currentStart.Year(), currentStart.Month()-1, 1, 0, 0, 0, 0, time.UTC)
		prevMonthEnd := time.Date(currentStart.Year(), currentStart.Month(), 0, 0, 0, 0, 0, time.UTC)
		prevFrom = prevMonthStart.Format("2006-01-02")
		prevTo = prevMonthEnd.Format("2006-01-02")
		comparisonLabel = "vs last month"
	case "last_month":
		twoMonthsAgoStart := time.Date(currentStart.Year(), currentStart.Month()-1, 1, 0, 0, 0, 0, time.UTC)
		twoMonthsAgoEnd := time.Date(currentStart.Year(), currentStart.Month(), 0, 0, 0, 0, 0, time.UTC)
		prevFrom = twoMonthsAgoStart.Format("2006-01-02")
		prevTo = twoMonthsAgoEnd.Format("2006-01-02")
		comparisonLabel = "vs " + twoMonthsAgoStart.Month().String()[:3]
	case "last_7_days", "last_30_days":
		periodDays := int(currentEnd.Sub(currentStart).Hours()/24) + 1
		previousEnd := currentStart.AddDate(0, 0, -1)
		previousStart := previousEnd.AddDate(0, 0, -periodDays+1)
		prevFrom = previousStart.Format("2006-01-02")
		prevTo = previousEnd.Format("2006-01-02")
		comparisonLabel = fmt.Sprintf("vs previous %d days", periodDays)
	default:
		prevFrom = currentStart.AddDate(-1, 0, 0).Format("2006-01-02")
		prevTo = currentEnd.AddDate(-1, 0, 0).Format("2006-01-02")
		comparisonLabel = "vs same period last year"
	}
	return
}

// GetGeofenceStats returns aggregated geofence violation statistics.
// GET /api/v1/tenant/geofence/stats
func (h *GeofenceHandler) GetGeofenceStats(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}

	// Parse date range and preset for period comparison
	now := time.Now().UTC()
	som := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	fromDate := c.DefaultQuery("from", som.Format("2006-01-02"))
	toDate := c.DefaultQuery("to", now.Format("2006-01-02"))
	preset := c.DefaultQuery("preset", "this_month")

	prevFromDate, prevToDate, comparisonLabel, err := calcPreviousPeriod(preset, fromDate, toDate)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid date format (expected YYYY-MM-DD)", nil)
		return
	}

	// Count by severity (single query, filtered)
	type SeverityCount struct {
		Severity string `json:"severity"`
		Count    int64  `json:"count"`
	}
	var severityCounts []SeverityCount
	severityQuery := h.DB.Model(&models.GeofenceViolation{}).
		Select("severity, COUNT(*) as count").
		Where("tenant_id = ?", tenantUUID)
	severityQuery = h.applyViolationFilters(c, severityQuery)
	severityQuery.Group("severity").Scan(&severityCounts)

	// Total count
	var totalCount int64
	for _, sc := range severityCounts {
		totalCount += sc.Count
	}

	// Previous period severity counts
	severity := c.Query("severity")
	batchIDs := h.resolveBatchIDs(c, tenantUUID)

	prevFromTime, _ := time.Parse("2006-01-02", prevFromDate)
	prevToTime, _ := time.Parse("2006-01-02", prevToDate)
	prevToExclusive := prevToTime.AddDate(0, 0, 1)

	var prevSeverityCounts []SeverityCount
	prevQuery := h.DB.Model(&models.GeofenceViolation{}).
		Select("severity, COUNT(*) as count").
		Where("tenant_id = ? AND created_at >= ? AND created_at < ?", tenantUUID, prevFromTime, prevToExclusive)
	if severity != "" {
		allowed := map[string]bool{"low": true, "medium": true, "high": true, "critical": true}
		if allowed[severity] {
			prevQuery = prevQuery.Where("severity = ?", severity)
		}
	}
	if len(batchIDs) > 0 {
		prevQuery = prevQuery.Where("batch_id IN ?", batchIDs)
	}
	prevQuery.Group("severity").Scan(&prevSeverityCounts)

	// Top batches by violation count (single query, limit 5, filtered)
	type BatchViolation struct {
		BatchID        uuid.UUID `json:"batch_id"`
		BatchName      string    `json:"batch_name"`
		ProductName    string    `json:"product_name"`
		ViolationCount int64     `json:"violation_count"`
	}
	var topBatches []BatchViolation
	topQuery := h.DB.Model(&models.GeofenceViolation{}).
		Select("geofence_violations.batch_id, qr_batches.batch_name, COALESCE(products.product_name, '') as product_name, COUNT(*) as violation_count").
		Joins("JOIN qr_batches ON qr_batches.id = geofence_violations.batch_id AND qr_batches.tenant_id = ?", tenantUUID).
		Joins("LEFT JOIN products ON products.id = qr_batches.product_id AND products.tenant_id = ?", tenantUUID).
		Where("geofence_violations.tenant_id = ?", tenantUUID)
	topQuery = h.applyViolationFilters(c, topQuery)
	topQuery.Group("geofence_violations.batch_id, qr_batches.batch_name, products.product_name").
		Order("violation_count DESC").
		Limit(5).
		Scan(&topBatches)

	// Recent violations (last 5, filtered)
	recentQuery := h.DB.Where("tenant_id = ?", tenantUUID)
	recentQuery = h.applyViolationFilters(c, recentQuery)
	var recentViolations []models.GeofenceViolation
	recentQuery.Preload("Batch", func(db *gorm.DB) *gorm.DB {
		return db.Select("id, batch_name, batch_code")
	}).
		Order("created_at DESC").
		Limit(5).
		Find(&recentViolations)

	utils.SuccessResponse(c, http.StatusOK, "Geofence stats", gin.H{
		"total_violations":    totalCount,
		"by_severity":         severityCounts,
		"previous_by_severity": prevSeverityCounts,
		"date_filter": gin.H{
			"from":             fromDate,
			"to":               toDate,
			"previous_from":    prevFromDate,
			"previous_to":      prevToDate,
			"comparison_label": comparisonLabel,
			"preset":           preset,
		},
		"top_batches":       topBatches,
		"recent_violations": recentViolations,
	})
}

// GeofenceComparisonStats holds metrics for period comparison in geofence analytics.
type GeofenceComparisonStats struct {
	TotalViolations int64
	TotalScans      int64
	ViolationRate   float64
	AvgKm           float64
	MaxKm           float64
}

// getGeofenceComparisonStats computes violation rate and distance stats for a given date range.
func (h *GeofenceHandler) getGeofenceComparisonStats(tenantUUID uuid.UUID, from, to, severity string, batchIDs []uuid.UUID) GeofenceComparisonStats {
	var stats GeofenceComparisonStats

	fromTime, _ := time.Parse("2006-01-02", from)
	toTime, _ := time.Parse("2006-01-02", to)
	toTimeExclusive := toTime.AddDate(0, 0, 1)

	// Count violations in date range
	violationQuery := h.DB.Model(&models.GeofenceViolation{}).
		Where("geofence_violations.tenant_id = ?", tenantUUID).
		Where("geofence_violations.created_at >= ? AND geofence_violations.created_at < ?", fromTime, toTimeExclusive)
	if severity != "" {
		allowed := map[string]bool{"low": true, "medium": true, "high": true, "critical": true}
		if allowed[severity] {
			violationQuery = violationQuery.Where("severity = ?", severity)
		}
	}
	if len(batchIDs) > 0 {
		violationQuery = violationQuery.Where("batch_id IN ?", batchIDs)
	}
	violationQuery.Count(&stats.TotalViolations)

	// Count total geofenced scans in date range
	scanQuery := h.DB.Model(&models.Interaction{}).
		Joins("JOIN qr_codes ON qr_codes.id = interactions.qr_code_id").
		Joins("JOIN qr_batches ON qr_batches.id = qr_codes.batch_id").
		Where("qr_batches.tenant_id = ? AND qr_batches.geofence_enabled = true", tenantUUID).
		Where("interactions.interaction_subcategory = ?", models.InteractionSubcategoryProductValidation).
		Where("interactions.created_at >= ? AND interactions.created_at < ?", fromTime, toTimeExclusive)
	if len(batchIDs) > 0 {
		scanQuery = scanQuery.Where("qr_batches.id IN ?", batchIDs)
	}
	scanQuery.Count(&stats.TotalScans)

	// Violation rate
	if stats.TotalScans > 0 {
		stats.ViolationRate = float64(stats.TotalViolations) / float64(stats.TotalScans) * 100
	}

	// Distance stats (avg/max from zone edge)
	type DistResult struct {
		AvgKm *float64
		MaxKm *float64
	}
	var dist DistResult
	distQuery := h.DB.Model(&models.GeofenceViolation{}).
		Select("ROUND(AVG(distance_from_edge_km)::numeric, 1) as avg_km, ROUND(MAX(distance_from_edge_km)::numeric, 1) as max_km").
		Where("geofence_violations.tenant_id = ?", tenantUUID).
		Where("geofence_violations.created_at >= ? AND geofence_violations.created_at < ?", fromTime, toTimeExclusive)
	if severity != "" {
		allowed := map[string]bool{"low": true, "medium": true, "high": true, "critical": true}
		if allowed[severity] {
			distQuery = distQuery.Where("severity = ?", severity)
		}
	}
	if len(batchIDs) > 0 {
		distQuery = distQuery.Where("batch_id IN ?", batchIDs)
	}
	distQuery.Scan(&dist)

	if dist.AvgKm != nil {
		stats.AvgKm = *dist.AvgKm
	}
	if dist.MaxKm != nil {
		stats.MaxKm = *dist.MaxKm
	}

	return stats
}

// GetGeofenceAnalytics returns strategic analytics for geofence violations.
// GET /api/v1/tenant/geofence/analytics
func (h *GeofenceHandler) GetGeofenceAnalytics(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}

	// Parse date range and preset (matching dashboard pattern)
	now := time.Now().UTC()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	fromDate := c.DefaultQuery("from", startOfMonth.Format("2006-01-02"))
	toDate := c.DefaultQuery("to", now.Format("2006-01-02"))
	preset := c.DefaultQuery("preset", "this_month")

	prevFromDate, prevToDate, comparisonLabel, err := calcPreviousPeriod(preset, fromDate, toDate)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid date format (expected YYYY-MM-DD)", nil)
		return
	}

	// Parse filters for comparison stats helper
	severity := c.Query("severity")
	batchIDs := h.resolveBatchIDs(c, tenantUUID)

	// Get comparison stats for current and previous periods
	currentStats := h.getGeofenceComparisonStats(tenantUUID, fromDate, toDate, severity, batchIDs)
	previousStats := h.getGeofenceComparisonStats(tenantUUID, prevFromDate, prevToDate, severity, batchIDs)

	// 1. Trend data — violations per week
	type TrendPoint struct {
		Week  time.Time `json:"week"`
		Count int64     `json:"count"`
	}
	var trends []TrendPoint
	trendQuery := h.DB.Model(&models.GeofenceViolation{}).
		Select("date_trunc('week', geofence_violations.created_at) as week, COUNT(*) as count").
		Where("geofence_violations.tenant_id = ?", tenantUUID)
	trendQuery = h.applyViolationFilters(c, trendQuery)
	trendQuery.Group("week").Order("week ASC").Scan(&trends)

	// 2. Top cities — JOIN with interactions
	type CityCount struct {
		City     string `json:"city"`
		Province string `json:"province"`
		Count    int64  `json:"count"`
	}
	var topCities []CityCount
	cityQuery := h.DB.Model(&models.GeofenceViolation{}).
		Select("i.geolocation->>'city' AS city, i.geolocation->>'province' AS province, COUNT(*) as count").
		Joins("JOIN interactions i ON i.id = geofence_violations.interaction_id AND i.tenant_id = ?", tenantUUID).
		Where("geofence_violations.tenant_id = ?", tenantUUID).
		Where("i.geolocation->>'city' != '' AND i.geolocation->>'city' IS NOT NULL")
	cityQuery = h.applyViolationFilters(c, cityQuery)
	cityQuery.Group("i.geolocation->>'city', i.geolocation->>'province'").Order("count DESC").Limit(10).Scan(&topCities)

	// 3. By product — violations grouped by product
	type ProductCount struct {
		ProductID   uuid.UUID `json:"product_id"`
		ProductName string    `json:"product_name"`
		Count       int64     `json:"count"`
	}
	var byProduct []ProductCount
	productQuery := h.DB.Model(&models.GeofenceViolation{}).
		Select("products.id as product_id, products.product_name, COUNT(*) as count").
		Joins("JOIN qr_batches ON qr_batches.id = geofence_violations.batch_id AND qr_batches.tenant_id = ?", tenantUUID).
		Joins("JOIN products ON products.id = qr_batches.product_id AND products.tenant_id = ?", tenantUUID).
		Where("geofence_violations.tenant_id = ?", tenantUUID)
	productQuery = h.applyViolationFilters(c, productQuery)
	productQuery.Group("products.id, products.product_name").Order("count DESC").Limit(10).Scan(&byProduct)

	// Assemble response
	utils.SuccessResponse(c, http.StatusOK, "Geofence analytics", gin.H{
		"trends":     trends,
		"top_cities": topCities,
		"violation_rate": gin.H{
			"total_scans":      currentStats.TotalScans,
			"total_violations": currentStats.TotalViolations,
			"rate":             currentStats.ViolationRate,
		},
		"distance_stats": gin.H{
			"avg_km": currentStats.AvgKm,
			"max_km": currentStats.MaxKm,
		},
		"by_product": byProduct,
		"previous_stats": gin.H{
			"violation_rate": gin.H{
				"total_scans":      previousStats.TotalScans,
				"total_violations": previousStats.TotalViolations,
				"rate":             previousStats.ViolationRate,
			},
			"distance_stats": gin.H{
				"avg_km": previousStats.AvgKm,
				"max_km": previousStats.MaxKm,
			},
		},
		"date_filter": gin.H{
			"from":             fromDate,
			"to":               toDate,
			"previous_from":    prevFromDate,
			"previous_to":      prevToDate,
			"comparison_label": comparisonLabel,
			"preset":           preset,
		},
	})
}

// GetBatchGeofenceViolations returns violations for a specific batch.
// GET /api/v1/tenant/qr-batches/:id/geofence-violations
func (h *GeofenceHandler) GetBatchGeofenceViolations(c *gin.Context) {
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

	// Verify batch belongs to tenant
	var batch models.QRBatch
	if err := h.DB.Select("id, tenant_id, geofence_enabled").
		First(&batch, "id = ? AND tenant_id = ? AND deleted_at IS NULL", batchID, tenantUUID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Batch not found", nil)
		return
	}

	page := utils.GetPageQuery(c)
	limit := utils.GetLimitQuery(c, 20)
	offset := (page - 1) * limit

	var total int64
	countQuery := h.DB.Model(&models.GeofenceViolation{}).
		Where("batch_id = ? AND tenant_id = ?", batchID, tenantUUID)
	countQuery = h.applyViolationFilters(c, countQuery)
	countQuery.Count(&total)

	var violations []models.GeofenceViolation
	fetchQuery := h.DB.Where("batch_id = ? AND tenant_id = ?", batchID, tenantUUID)
	fetchQuery = h.applyViolationFilters(c, fetchQuery)
	fetchQuery.
		Preload("Batch", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, batch_name, batch_code, product_id")
		}).Preload("Batch.Product", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, product_name, product_code")
		}).
		Order("created_at DESC").
		Offset(offset).Limit(limit).
		Find(&violations)

	utils.SuccessResponse(c, http.StatusOK, "Batch geofence violations", gin.H{
		"violations": violations,
		"pagination": utils.PaginationMeta(page, limit, total),
	})
}

// GetBatchGeofenceAnalytics returns detailed analytics for a batch's geofence.
// GET /api/v1/tenant/qr-batches/:id/geofence-analytics
func (h *GeofenceHandler) GetBatchGeofenceAnalytics(c *gin.Context) {
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

	// Verify batch belongs to tenant
	var batch models.QRBatch
	if err := h.DB.Select("id, tenant_id, qr_count, geofence_enabled, geofence_label, geofence_radius_km").
		First(&batch, "id = ? AND tenant_id = ? AND deleted_at IS NULL", batchID, tenantUUID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Batch not found", nil)
		return
	}

	// Total scans for this batch (from interactions)
	var totalScans int64
	h.DB.Model(&models.Interaction{}).
		Joins("JOIN qr_codes ON qr_codes.id = interactions.qr_code_id").
		Where("qr_codes.batch_id = ? AND interactions.interaction_subcategory = ?",
			batchID, models.InteractionSubcategoryProductValidation).
		Count(&totalScans)

	// Total violations
	var totalViolations int64
	h.DB.Model(&models.GeofenceViolation{}).
		Where("batch_id = ? AND tenant_id = ?", batchID, tenantUUID).
		Count(&totalViolations)

	// Violations by severity
	type SeverityCount struct {
		Severity string `json:"severity"`
		Count    int64  `json:"count"`
	}
	var bySeverity []SeverityCount
	h.DB.Model(&models.GeofenceViolation{}).
		Select("severity, COUNT(*) as count").
		Where("batch_id = ? AND tenant_id = ?", batchID, tenantUUID).
		Group("severity").
		Scan(&bySeverity)

	// Violation rate
	var violationRate float64
	if totalScans > 0 {
		violationRate = float64(totalViolations) / float64(totalScans) * 100
	}

	inZoneCount := totalScans - totalViolations
	if inZoneCount < 0 {
		inZoneCount = 0
	}

	utils.SuccessResponse(c, http.StatusOK, "Batch geofence analytics", gin.H{
		"batch_id":          batchID,
		"geofence_label":    batch.GeofenceLabel,
		"geofence_radius_km": batch.GeofenceRadiusKm,
		"total_scans":       totalScans,
		"total_violations":  totalViolations,
		"violation_rate":    violationRate,
		"by_severity":       bySeverity,
		"in_zone_count":     inZoneCount,
		"out_of_zone_count": totalViolations,
	})
}

// ExportGeofenceViolations exports violations to Excel.
// GET /api/v1/tenant/geofence/violations/export
func (h *GeofenceHandler) ExportGeofenceViolations(c *gin.Context) {
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

	// Fetch all violations with JOIN (single query, no N+1)
	type exportRow struct {
		CreatedAt            time.Time
		BatchName            string
		ProductName          string
		Severity             string
		DistanceFromCenterKm float64
		DistanceFromEdgeKm   float64
		ScanLatitude         float64
		ScanLongitude        float64
	}

	query := h.DB.Model(&models.GeofenceViolation{}).
		Select(`geofence_violations.created_at, qr_batches.batch_name,
			COALESCE(products.product_name, '') as product_name,
			geofence_violations.severity, geofence_violations.distance_from_center_km,
			geofence_violations.distance_from_edge_km, geofence_violations.scan_latitude,
			geofence_violations.scan_longitude`).
		Joins("JOIN qr_batches ON qr_batches.id = geofence_violations.batch_id").
		Joins("LEFT JOIN products ON products.id = qr_batches.product_id").
		Where("geofence_violations.tenant_id = ?", tenantUUID).
		Order("geofence_violations.created_at DESC").
		Limit(10000)

	query = h.applyViolationFilters(c, query)

	var rows []exportRow
	query.Scan(&rows)

	f := excelize.NewFile()
	sheet := "Geofence Violations"
	f.SetSheetName("Sheet1", sheet)

	headers := []string{"Date" + utils.TimezoneHeaderSuffix(exportTZ), "Batch", "Product", "Severity", "Distance from Center (km)", "Distance from Edge (km)", "Scan Latitude", "Scan Longitude"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
	}

	for i, r := range rows {
		row := i + 2
		f.SetCellValue(sheet, geoCellName(1, row), utils.FormatExportTime(r.CreatedAt, exportTZ, "2006-01-02 15:04"))
		f.SetCellValue(sheet, geoCellName(2, row), r.BatchName)
		f.SetCellValue(sheet, geoCellName(3, row), r.ProductName)
		f.SetCellValue(sheet, geoCellName(4, row), strings.ToUpper(r.Severity))
		f.SetCellValue(sheet, geoCellName(5, row), r.DistanceFromCenterKm)
		f.SetCellValue(sheet, geoCellName(6, row), r.DistanceFromEdgeKm)
		f.SetCellValue(sheet, geoCellName(7, row), r.ScanLatitude)
		f.SetCellValue(sheet, geoCellName(8, row), r.ScanLongitude)
	}

	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", "attachment; filename=geofence_violations.xlsx")
	f.Write(c.Writer)
}

// geoCellName is a helper for Excel coordinate to cell name conversion.
func geoCellName(col, row int) string {
	name, _ := excelize.CoordinatesToCellName(col, row)
	return name
}

// GetGeofenceMapData returns lightweight violation coordinates and batch zone circles for map rendering.
// GET /api/v1/tenant/geofence/map-data
// Query params: severity, batch_id, date_from, date_to
func (h *GeofenceHandler) GetGeofenceMapData(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}

	// Query 1: Violation coordinates (max 500, latest first)
	type MapViolation struct {
		Lat       float64   `json:"lat"`
		Lng       float64   `json:"lng"`
		Severity  string    `json:"severity"`
		BatchName string    `json:"batch_name"`
		BatchID   uuid.UUID `json:"batch_id"`
		CreatedAt time.Time `json:"created_at"`
	}

	violationQuery := h.DB.Model(&models.GeofenceViolation{}).
		Select("geofence_violations.scan_latitude AS lat, geofence_violations.scan_longitude AS lng, geofence_violations.severity, qr_batches.batch_name, geofence_violations.batch_id, geofence_violations.created_at").
		Joins("JOIN qr_batches ON qr_batches.id = geofence_violations.batch_id AND qr_batches.tenant_id = ?", tenantUUID).
		Where("geofence_violations.tenant_id = ?", tenantUUID).
		Where("geofence_violations.scan_latitude != 0 OR geofence_violations.scan_longitude != 0")

	violationQuery = h.applyViolationFilters(c, violationQuery)

	violations := make([]MapViolation, 0)
	if err := violationQuery.Order("geofence_violations.created_at DESC").
		Limit(500).
		Scan(&violations).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to fetch map data", err)
		return
	}

	// Query 2: Batch zone circles
	type MapZone struct {
		BatchID   uuid.UUID `json:"batch_id"`
		BatchName string    `json:"batch_name"`
		Lat       float64   `json:"lat"`
		Lng       float64   `json:"lng"`
		RadiusKm  float64   `json:"radius_km"`
		Label     string    `json:"label"`
	}

	zones := make([]MapZone, 0)
	// When product/area filter is active, show all zones for that selection
	productID := c.Query("product_id")
	geofenceLabel := c.Query("geofence_label")
	if productID != "" {
		if pid, err := uuid.Parse(productID); err == nil {
			zoneQuery := h.DB.Model(&models.QRBatch{}).
				Select("id AS batch_id, batch_name, geofence_latitude AS lat, geofence_longitude AS lng, geofence_radius_km AS radius_km, geofence_label AS label").
				Where("product_id = ? AND tenant_id = ? AND geofence_enabled = true AND deleted_at IS NULL AND geofence_latitude IS NOT NULL AND geofence_longitude IS NOT NULL",
					pid, tenantUUID)
			if geofenceLabel != "" {
				zoneQuery = zoneQuery.Where("geofence_label = ?", geofenceLabel)
			}
			zoneQuery.Limit(100).Scan(&zones)
		}
	} else {
		// Default: show zones only for batches with violations
		batchSubquery := h.DB.Model(&models.GeofenceViolation{}).
			Select("DISTINCT batch_id").
			Where("tenant_id = ?", tenantUUID).
			Where("scan_latitude != 0 OR scan_longitude != 0")
		batchSubquery = h.applyViolationFilters(c, batchSubquery)

		h.DB.Model(&models.QRBatch{}).
			Select("id AS batch_id, batch_name, geofence_latitude AS lat, geofence_longitude AS lng, geofence_radius_km AS radius_km, geofence_label AS label").
			Where("id IN (?) AND tenant_id = ? AND geofence_enabled = true AND geofence_latitude IS NOT NULL AND geofence_longitude IS NOT NULL", batchSubquery, tenantUUID).
			Limit(100).
			Scan(&zones)
	}

	utils.SuccessResponse(c, http.StatusOK, "Geofence map data", gin.H{
		"violations": violations,
		"zones":      zones,
	})
}

// ---- Zone Templates ----

// ListZoneTemplates returns paginated zone templates for a tenant.
// GET /api/v1/tenant/geofence/zone-templates
func (h *GeofenceHandler) ListZoneTemplates(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}

	page := utils.GetPageQuery(c)
	limit := utils.GetLimitQuery(c, 50)
	offset := (page - 1) * limit

	// Status filter: active (default), deleted, all
	status := c.DefaultQuery("status", "active")
	query := h.DB.Where("tenant_id = ?", tenantUUID)
	switch status {
	case "deleted":
		query = query.Where("deleted_at IS NOT NULL")
	case "all":
		// no additional filter
	default: // "active"
		query = query.Where("deleted_at IS NULL")
	}

	var total int64
	query.Model(&models.GeofenceZoneTemplate{}).Count(&total)

	var templates []models.GeofenceZoneTemplate
	query.Model(&models.GeofenceZoneTemplate{}).
		Order("usage_count DESC, created_at DESC").
		Offset(offset).Limit(limit).
		Find(&templates)

	utils.SuccessResponse(c, http.StatusOK, "Zone templates", gin.H{
		"zone_templates": templates,
		"pagination": utils.PaginationMeta(page, limit, total),
	})
}

type CreateZoneTemplateRequest struct {
	TemplateName string  `json:"template_name" binding:"required"`
	Latitude     float64 `json:"latitude" binding:"required"`
	Longitude    float64 `json:"longitude" binding:"required"`
	RadiusKm     float64 `json:"radius_km" binding:"required"`
	Label        string  `json:"label"`
}

// CreateZoneTemplate creates a new zone template.
// POST /api/v1/tenant/geofence/zone-templates
func (h *GeofenceHandler) CreateZoneTemplate(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}

	var req CreateZoneTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request", err)
		return
	}

	// Validate coordinates
	if req.Latitude < -90 || req.Latitude > 90 {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid latitude (-90 to 90)", nil)
		return
	}
	if req.Longitude < -180 || req.Longitude > 180 {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid longitude (-180 to 180)", nil)
		return
	}
	if req.RadiusKm < 1 || req.RadiusKm > 500 {
		utils.ErrorResponse(c, http.StatusBadRequest, "Radius must be between 1 and 500 km", nil)
		return
	}

	name := strings.TrimSpace(req.TemplateName)
	if nameRunes := []rune(name); len(nameRunes) > 255 {
		name = string(nameRunes[:255])
	}
	label := strings.TrimSpace(req.Label)
	if labelRunes := []rune(label); len(labelRunes) > 255 {
		label = string(labelRunes[:255])
	}

	template := models.GeofenceZoneTemplate{
		TenantID:     tenantUUID,
		TemplateName: name,
		Latitude:     req.Latitude,
		Longitude:    req.Longitude,
		RadiusKm:     req.RadiusKm,
		Label:        label,
	}

	if err := h.DB.Create(&template).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to create zone template", err)
		return
	}

	utils.SuccessResponse(c, http.StatusCreated, "Zone template created", gin.H{
		"zone_template": template,
	})
}

// UpdateZoneTemplate updates an existing zone template.
// PUT /api/v1/tenant/geofence/zone-templates/:id
func (h *GeofenceHandler) UpdateZoneTemplate(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}

	templateID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid template ID", nil)
		return
	}

	var template models.GeofenceZoneTemplate
	if err := h.DB.First(&template, "id = ? AND tenant_id = ? AND deleted_at IS NULL", templateID, tenantUUID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Zone template not found", nil)
		return
	}

	var req CreateZoneTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request", err)
		return
	}

	if req.Latitude < -90 || req.Latitude > 90 {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid latitude (-90 to 90)", nil)
		return
	}
	if req.Longitude < -180 || req.Longitude > 180 {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid longitude (-180 to 180)", nil)
		return
	}
	if req.RadiusKm < 1 || req.RadiusKm > 500 {
		utils.ErrorResponse(c, http.StatusBadRequest, "Radius must be between 1 and 500 km", nil)
		return
	}

	name := strings.TrimSpace(req.TemplateName)
	if nameRunes := []rune(name); len(nameRunes) > 255 {
		name = string(nameRunes[:255])
	}
	label := strings.TrimSpace(req.Label)
	if labelRunes := []rune(label); len(labelRunes) > 255 {
		label = string(labelRunes[:255])
	}

	if err := h.DB.Model(&template).Updates(map[string]interface{}{
		"template_name": name,
		"latitude":      req.Latitude,
		"longitude":     req.Longitude,
		"radius_km":     req.RadiusKm,
		"label":         label,
	}).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to update zone template", err)
		return
	}

	// Reload to return fresh data
	h.DB.First(&template, "id = ? AND tenant_id = ?", templateID, tenantUUID)

	utils.SuccessResponse(c, http.StatusOK, "Zone template updated", gin.H{
		"zone_template": template,
	})
}

// DeleteZoneTemplate soft-deletes a zone template.
// DELETE /api/v1/tenant/geofence/zone-templates/:id
func (h *GeofenceHandler) DeleteZoneTemplate(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}

	templateID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid template ID", nil)
		return
	}

	var template models.GeofenceZoneTemplate
	if err := h.DB.First(&template, "id = ? AND tenant_id = ? AND deleted_at IS NULL", templateID, tenantUUID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Zone template not found", nil)
		return
	}

	now := time.Now()
	h.DB.Model(&template).Update("deleted_at", &now)

	utils.SuccessResponse(c, http.StatusOK, "Zone template deleted", nil)
}

// RestoreZoneTemplate restores a soft-deleted zone template.
// POST /api/v1/tenant/geofence/zone-templates/:id/restore
func (h *GeofenceHandler) RestoreZoneTemplate(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}

	templateID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid template ID", nil)
		return
	}

	var template models.GeofenceZoneTemplate
	if err := h.DB.Where("id = ? AND tenant_id = ? AND deleted_at IS NOT NULL", templateID, tenantUUID).First(&template).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Zone template not found or not deleted", nil)
		return
	}

	if err := h.DB.Model(&template).Update("deleted_at", nil).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to restore zone template", err)
		return
	}

	template.DeletedAt = nil
	utils.SuccessResponse(c, http.StatusOK, "Zone template restored", gin.H{
		"zone_template": template,
	})
}
