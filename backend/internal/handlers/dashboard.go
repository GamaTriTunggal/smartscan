package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gamatritunggal/smartscan/backend/internal/config"
	"github.com/gamatritunggal/smartscan/backend/internal/models"
	"github.com/gamatritunggal/smartscan/backend/internal/utils"
	"gorm.io/gorm"
)

type DashboardHandler struct {
	DB  *gorm.DB
	Cfg *config.Config
}

func NewDashboardHandler(db *gorm.DB, cfg *config.Config) *DashboardHandler {
	return &DashboardHandler{DB: db, Cfg: cfg}
}

// ========================================
// Tenant Dashboard Response Structures
// ========================================

// EnhancedStats contains enhanced dashboard statistics with new metrics
type EnhancedStats struct {
	TotalScans       int64   `json:"total_scans"`
	TotalQRCodes     int64   `json:"total_qr_codes"`
	ScanQRRatio      float64 `json:"scan_qr_ratio"`
	UniqueCities     int64   `json:"unique_cities"`
	CounterfeitRate  float64 `json:"counterfeit_rate"`
	CounterfeitCount int64   `json:"counterfeit_count"`
}

// ScanTrendByType for stacked area chart showing scan breakdown by type
type ScanTrendByType struct {
	Date       string `json:"date"`
	Validation int64  `json:"validation"`
	Warranty   int64  `json:"warranty"`
	Campaign   int64  `json:"campaign"`
}

// RegionStat for top performing regions widget
type RegionStat struct {
	City    string `json:"city"`
	Country string `json:"country"`
	Count   int64  `json:"count"`
}

// ProductCounterfeit for per-product counterfeit breakdown
type ProductCounterfeit struct {
	ProductID        string  `json:"product_id"`
	ProductName      string  `json:"product_name"`
	TotalScans       int64   `json:"total_scans"`
	CounterfeitCount int64   `json:"counterfeit_count"`
	Rate             float64 `json:"rate"`
	RiskLevel        string  `json:"risk_level"`
}

// CounterfeitHotspot for counterfeit hotspot locations
type CounterfeitHotspot struct {
	City    string `json:"city"`
	Country string `json:"country"`
	Count   int64  `json:"count"`
}

// TemplateWithTrend for template performance with period comparison
type TemplateWithTrend struct {
	TemplateID    *uuid.UUID `json:"template_id"`
	TemplateName  string     `json:"template_name"`
	Count         int64      `json:"count"`
	PreviousCount int64      `json:"previous_count"`
	ChangePercent float64    `json:"change_percent"`
}

// GetTenantDashboard returns tenant dashboard data with enhanced metrics
func (h *DashboardHandler) GetTenantDashboard(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}

	// Parse date range parameters (default: start of current month to today)
	now := time.Now().UTC()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	fromDate := c.DefaultQuery("from", startOfMonth.Format("2006-01-02"))
	toDate := c.DefaultQuery("to", now.Format("2006-01-02"))
	preset := c.DefaultQuery("preset", "this_month")
	var dynamicProductCount int64
	h.DB.Model(&models.Product{}).
		Where("tenant_id = ? AND deleted_at IS NULL", tenantUUID).
		Count(&dynamicProductCount)
	hasDynamicProducts := dynamicProductCount > 0

	// Calculate previous period dates based on preset type (calendar-based comparison)
	currentStart, _ := time.Parse("2006-01-02", fromDate)
	currentEnd, _ := time.Parse("2006-01-02", toDate)

	var prevFromDate, prevToDate, comparisonLabel string

	switch preset {
	case "this_month":
		// This month vs Last month (full previous month)
		prevMonthStart := time.Date(currentStart.Year(), currentStart.Month()-1, 1, 0, 0, 0, 0, time.UTC)
		prevMonthEnd := time.Date(currentStart.Year(), currentStart.Month(), 0, 0, 0, 0, 0, time.UTC) // Day 0 = last day of previous month
		prevFromDate = prevMonthStart.Format("2006-01-02")
		prevToDate = prevMonthEnd.Format("2006-01-02")
		comparisonLabel = "vs last month"

	case "last_month":
		// Last month vs month before that
		twoMonthsAgoStart := time.Date(currentStart.Year(), currentStart.Month()-1, 1, 0, 0, 0, 0, time.UTC)
		twoMonthsAgoEnd := time.Date(currentStart.Year(), currentStart.Month(), 0, 0, 0, 0, 0, time.UTC)
		prevFromDate = twoMonthsAgoStart.Format("2006-01-02")
		prevToDate = twoMonthsAgoEnd.Format("2006-01-02")
		comparisonLabel = "vs " + twoMonthsAgoStart.Month().String()[:3]

	case "last_7_days", "last_30_days":
		// Rolling: previous N days before current period
		periodDays := int(currentEnd.Sub(currentStart).Hours()/24) + 1
		previousEnd := currentStart.AddDate(0, 0, -1)
		previousStart := previousEnd.AddDate(0, 0, -periodDays+1)
		prevFromDate = previousStart.Format("2006-01-02")
		prevToDate = previousEnd.Format("2006-01-02")
		comparisonLabel = fmt.Sprintf("vs previous %d days", periodDays)

	default: // "custom" or any other
		// Year-over-Year: compare with same period last year
		prevFromDate = currentStart.AddDate(-1, 0, 0).Format("2006-01-02")
		prevToDate = currentEnd.AddDate(-1, 0, 0).Format("2006-01-02")
		comparisonLabel = "vs same period last year"
	}

	// All features are always enabled in the self-hosted edition.
	isIntermediatePlus := true

	// Helper function to get enhanced stats for a period
	getEnhancedStats := func(from, to string) EnhancedStats {
		var stats EnhancedStats

		// Total scans — filter by QR type
		h.DB.Model(&models.Interaction{}).
			Where("tenant_id = ? AND (created_at AT TIME ZONE 'UTC')::date >= ? AND (created_at AT TIME ZONE 'UTC')::date <= ?",
				tenantUUID, from, to).
			Where("qr_code_id IS NOT NULL").
			Count(&stats.TotalScans)

		// Total QR codes
		h.DB.Model(&models.QRCode{}).Joins("JOIN qr_batches ON qr_batches.id = qr_codes.batch_id").
			Where("qr_batches.tenant_id = ? AND qr_batches.deleted_at IS NULL AND (qr_codes.created_at AT TIME ZONE 'UTC')::date >= ? AND (qr_codes.created_at AT TIME ZONE 'UTC')::date <= ?",
				tenantUUID, from, to).Count(&stats.TotalQRCodes)

		// Scan-QR Ratio
		if stats.TotalQRCodes > 0 {
			stats.ScanQRRatio = float64(stats.TotalScans) / float64(stats.TotalQRCodes) * 100
		}

		// Unique cities (geographic coverage)
		qrTypeFilter := "AND qr_code_id IS NOT NULL"
		h.DB.Raw(fmt.Sprintf(`
			SELECT COUNT(DISTINCT geolocation->>'city')
			FROM interactions
			WHERE tenant_id = ?
				AND geolocation IS NOT NULL
				AND geolocation->>'city' IS NOT NULL
				AND geolocation->>'city' != ''
				AND (created_at AT TIME ZONE 'UTC')::date >= ?
				AND (created_at AT TIME ZONE 'UTC')::date <= ?
				%s
		`, qrTypeFilter), tenantUUID, from, to).Scan(&stats.UniqueCities)

		// Counterfeit count and rate
		if isIntermediatePlus {
			h.DB.Model(&models.CounterfeitDetection{}).
				Where("tenant_id = ? AND status = ? AND (created_at AT TIME ZONE 'UTC')::date >= ? AND (created_at AT TIME ZONE 'UTC')::date <= ?",
					tenantUUID, "active", from, to).
				Count(&stats.CounterfeitCount)

			if stats.TotalScans > 0 {
				stats.CounterfeitRate = float64(stats.CounterfeitCount) / float64(stats.TotalScans) * 100
			}
		}

		return stats
	}

	// Get current and previous period stats
	currentStats := getEnhancedStats(fromDate, toDate)
	previousStats := getEnhancedStats(prevFromDate, prevToDate)

	// Scan filter for raw SQL queries
	qrTypeSQL := "AND qr_code_id IS NOT NULL"

	// Scan trends by type (for stacked area chart) — filtered by QR type
	var scanTrendsByType []ScanTrendByType
	h.DB.Raw(fmt.Sprintf(`
		SELECT
			(created_at AT TIME ZONE 'UTC')::date::text as date,
			COUNT(*) FILTER (WHERE interaction_subcategory = 'product_validation') as validation,
			COUNT(*) FILTER (WHERE interaction_subcategory = 'warranty_activation') as warranty,
			COUNT(*) FILTER (WHERE interaction_subcategory = 'campaign') as campaign
		FROM interactions
		WHERE tenant_id = ?
			AND (created_at AT TIME ZONE 'UTC')::date >= ?
			AND (created_at AT TIME ZONE 'UTC')::date <= ?
			%s
		GROUP BY (created_at AT TIME ZONE 'UTC')::date
		ORDER BY (created_at AT TIME ZONE 'UTC')::date ASC
	`, qrTypeSQL), tenantUUID, fromDate, toDate).Scan(&scanTrendsByType)

	// Top performing regions — filtered by QR type
	var topRegions []RegionStat
	h.DB.Raw(fmt.Sprintf(`
		SELECT
			geolocation->>'city' as city,
			geolocation->>'country' as country,
			COUNT(*) as count
		FROM interactions
		WHERE tenant_id = ?
			AND geolocation IS NOT NULL
			AND geolocation->>'city' IS NOT NULL
			AND geolocation->>'city' != ''
			AND (created_at AT TIME ZONE 'UTC')::date >= ?
			AND (created_at AT TIME ZONE 'UTC')::date <= ?
			%s
		GROUP BY geolocation->>'city', geolocation->>'country'
		ORDER BY count DESC
		LIMIT 10
	`, qrTypeSQL), tenantUUID, fromDate, toDate).Scan(&topRegions)

	// Counterfeit Intelligence (for Intermediate+ tier)
	var productCounterfeit []ProductCounterfeit
	var counterfeitHotspots []CounterfeitHotspot

	if isIntermediatePlus {
		// Per-product counterfeit breakdown (dynamic QR only)
		h.DB.Raw(`
			SELECT
				p.id::text as product_id,
				p.product_name,
				COUNT(DISTINCT i.id) as total_scans,
				COUNT(DISTINCT CASE WHEN qc.counterfeit_status = 'counterfeit' THEN i.id END) as counterfeit_count,
				CASE
					WHEN COUNT(DISTINCT i.id) = 0 THEN 0
					ELSE ROUND((COUNT(DISTINCT CASE WHEN qc.counterfeit_status = 'counterfeit' THEN i.id END)::numeric / COUNT(DISTINCT i.id)::numeric) * 100, 2)
				END as rate
			FROM products p
			LEFT JOIN qr_batches qb ON qb.product_id = p.id
			LEFT JOIN qr_codes qc ON qc.batch_id = qb.id
			LEFT JOIN interactions i ON i.qr_code_id = qc.id
				AND (i.created_at AT TIME ZONE 'UTC')::date >= ?
				AND (i.created_at AT TIME ZONE 'UTC')::date <= ?
			WHERE p.tenant_id = ? AND p.deleted_at IS NULL
			GROUP BY p.id, p.product_name
			HAVING COUNT(DISTINCT i.id) > 0
			ORDER BY rate DESC
			LIMIT 10
		`, fromDate, toDate, tenantUUID).Scan(&productCounterfeit)

		// Add risk level to each product
		for i := range productCounterfeit {
			rate := productCounterfeit[i].Rate
			switch {
			case rate > 2:
				productCounterfeit[i].RiskLevel = "high"
			case rate > 0.5:
				productCounterfeit[i].RiskLevel = "medium"
			case rate > 0:
				productCounterfeit[i].RiskLevel = "low"
			default:
				productCounterfeit[i].RiskLevel = "safe"
			}
		}

		// Counterfeit hotspot locations
		h.DB.Raw(`
			SELECT
				i.geolocation->>'city' as city,
				i.geolocation->>'country' as country,
				COUNT(DISTINCT cd.id) as count
			FROM counterfeit_detections cd
			JOIN qr_codes qc ON qc.id = cd.qr_code_id
			JOIN interactions i ON i.qr_code_id = qc.id
			WHERE cd.tenant_id = ?
				AND cd.status = 'active'
				AND i.geolocation IS NOT NULL
				AND i.geolocation->>'city' IS NOT NULL
				AND (cd.created_at AT TIME ZONE 'UTC')::date >= ?
				AND (cd.created_at AT TIME ZONE 'UTC')::date <= ?
			GROUP BY i.geolocation->>'city', i.geolocation->>'country'
			ORDER BY count DESC
			LIMIT 3
		`, tenantUUID, fromDate, toDate).Scan(&counterfeitHotspots)
	}

	// Template performance with trend comparison
	type TemplateUsage struct {
		TemplateID    *uuid.UUID `json:"template_id"`
		TemplateName  string     `json:"template_name"`
		Count         int64      `json:"count"`
		PreviousCount int64      `json:"previous_count"`
		ChangePercent float64    `json:"change_percent"`
	}

	// Helper to calculate change percent
	calcChangePercent := func(current, previous int64) float64 {
		if previous == 0 {
			if current > 0 {
				return 100
			}
			return 0
		}
		return float64(current-previous) / float64(previous) * 100
	}

	// 1. Validation (Landing Page) Template Performance - available for all tiers
	var validationCurrentRaw []struct {
		TemplateID   *uuid.UUID
		TemplateName string
		Count        int64
	}
	h.DB.Raw(fmt.Sprintf(`
		WITH template_counts AS (
			SELECT
				i.validation_template_id as template_id,
				COALESCE(pt.template_name, 'Default Template') as template_name,
				COUNT(*) as count
			FROM interactions i
			LEFT JOIN page_templates pt ON pt.id = i.validation_template_id
			WHERE i.tenant_id = ?
				AND i.interaction_subcategory = 'product_validation'
				AND (i.created_at AT TIME ZONE 'UTC')::date >= ?
				AND (i.created_at AT TIME ZONE 'UTC')::date <= ?
				%s
			GROUP BY i.validation_template_id, pt.template_name
			ORDER BY count DESC
		),
		top_10 AS (
			SELECT template_id, template_name, count
			FROM template_counts
			LIMIT 10
		),
		others AS (
			SELECT NULL::uuid as template_id, 'Others' as template_name, COALESCE(SUM(count), 0) as count
			FROM template_counts
			WHERE (template_id, template_name) NOT IN (SELECT template_id, template_name FROM top_10)
		)
		SELECT * FROM top_10
		UNION ALL
		SELECT * FROM others WHERE count > 0
	`, qrTypeSQL), tenantUUID, fromDate, toDate).Scan(&validationCurrentRaw)

	// Get previous period counts for validation templates
	var validationPreviousRaw []struct {
		TemplateID   *uuid.UUID
		TemplateName string
		Count        int64
	}
	h.DB.Raw(fmt.Sprintf(`
		SELECT
			i.validation_template_id as template_id,
			COALESCE(pt.template_name, 'Default Template') as template_name,
			COUNT(*) as count
		FROM interactions i
		LEFT JOIN page_templates pt ON pt.id = i.validation_template_id
		WHERE i.tenant_id = ?
			AND i.interaction_subcategory = 'product_validation'
			AND (i.created_at AT TIME ZONE 'UTC')::date >= ?
			AND (i.created_at AT TIME ZONE 'UTC')::date <= ?
			%s
		GROUP BY i.validation_template_id, pt.template_name
	`, qrTypeSQL), tenantUUID, prevFromDate, prevToDate).Scan(&validationPreviousRaw)

	// Build map for previous counts
	prevValidationMap := make(map[string]int64)
	for _, p := range validationPreviousRaw {
		prevValidationMap[p.TemplateName] = p.Count
	}

	// Combine current with previous
	var validationTemplatePerformance []TemplateUsage
	for _, c := range validationCurrentRaw {
		prev := prevValidationMap[c.TemplateName]
		validationTemplatePerformance = append(validationTemplatePerformance, TemplateUsage{
			TemplateID:    c.TemplateID,
			TemplateName:  c.TemplateName,
			Count:         c.Count,
			PreviousCount: prev,
			ChangePercent: calcChangePercent(c.Count, prev),
		})
	}

	// Response data
	responseData := gin.H{
		"stats":               currentStats,
		"previous_stats":      previousStats,
		"scan_trends_by_type": scanTrendsByType,
		"top_regions":         topRegions,
		"date_filter": gin.H{
			"from":             fromDate,
			"to":               toDate,
			"previous_from":    prevFromDate,
			"previous_to":      prevToDate,
			"comparison_label": comparisonLabel,
			"preset":           preset,
		},
		"template_performance": gin.H{
			"validation": validationTemplatePerformance,
		},
		"has_dynamic_products": hasDynamicProducts,
	}

	// Add counterfeit intelligence for Intermediate+ tier
	if isIntermediatePlus {
		responseData["counterfeit"] = gin.H{
			"overall_rate":      currentStats.CounterfeitRate,
			"previous_rate":     previousStats.CounterfeitRate,
			"per_product":       productCounterfeit,
			"hotspot_locations": counterfeitHotspots,
		}
	}

	// Add geofence distribution alerts for Intermediate+ tier (dynamic only)
	if isIntermediatePlus {
		// Count violations by severity for current period
		type SeverityCount struct {
			Severity string `json:"severity"`
			Count    int64  `json:"count"`
		}
		var geoSeverityCounts []SeverityCount
		h.DB.Model(&models.GeofenceViolation{}).
			Select("severity, COUNT(*) as count").
			Where("tenant_id = ?", tenantUUID).
			Where("(created_at AT TIME ZONE 'UTC')::date >= ? AND (created_at AT TIME ZONE 'UTC')::date <= ?", fromDate, toDate).
			Group("severity").
			Scan(&geoSeverityCounts)

		var geoTotalViolations int64
		for _, sc := range geoSeverityCounts {
			geoTotalViolations += sc.Count
		}

		// Only include geofence data if there are violations
		if geoTotalViolations > 0 {
			// Top batches by violation count (limit 5)
			type BatchViolation struct {
				BatchID   uuid.UUID `json:"batch_id"`
				BatchName string    `json:"batch_name"`
				Count     int64     `json:"count"`
			}
			var geoTopBatches []BatchViolation
			h.DB.Model(&models.GeofenceViolation{}).
				Select("geofence_violations.batch_id, qr_batches.batch_name, COUNT(*) as count").
				Joins("JOIN qr_batches ON qr_batches.id = geofence_violations.batch_id AND qr_batches.tenant_id = ?", tenantUUID).
				Where("geofence_violations.tenant_id = ?", tenantUUID).
				Where("(geofence_violations.created_at AT TIME ZONE 'UTC')::date >= ? AND (geofence_violations.created_at AT TIME ZONE 'UTC')::date <= ?", fromDate, toDate).
				Group("geofence_violations.batch_id, qr_batches.batch_name").
				Order("count DESC").
				Limit(5).
				Scan(&geoTopBatches)

			geofenceData := gin.H{
				"total_violations": geoTotalViolations,
				"by_severity":     geoSeverityCounts,
				"top_batches":     geoTopBatches,
			}

			// Violation rate and trend
			isProTier := true
			if isProTier {
				// Total scans for geofence-enabled batches in current period
				var geoTotalScans int64
				h.DB.Raw(`
					SELECT COUNT(*)
					FROM interactions i
					JOIN qr_codes qc ON qc.id = i.qr_code_id
					JOIN qr_batches qb ON qb.id = qc.batch_id
					WHERE qb.tenant_id = ? AND qb.geofence_enabled = true
						AND i.interaction_subcategory = 'product_validation'
						AND (i.created_at AT TIME ZONE 'UTC')::date >= ?
						AND (i.created_at AT TIME ZONE 'UTC')::date <= ?
				`, tenantUUID, fromDate, toDate).Scan(&geoTotalScans)

				var geoViolationRate float64
				if geoTotalScans > 0 {
					geoViolationRate = float64(geoTotalViolations) / float64(geoTotalScans) * 100
				}

				// Previous period violation rate
				var prevGeoViolations int64
				h.DB.Model(&models.GeofenceViolation{}).
					Where("tenant_id = ?", tenantUUID).
					Where("(created_at AT TIME ZONE 'UTC')::date >= ? AND (created_at AT TIME ZONE 'UTC')::date <= ?", prevFromDate, prevToDate).
					Count(&prevGeoViolations)

				var prevGeoScans int64
				h.DB.Raw(`
					SELECT COUNT(*)
					FROM interactions i
					JOIN qr_codes qc ON qc.id = i.qr_code_id
					JOIN qr_batches qb ON qb.id = qc.batch_id
					WHERE qb.tenant_id = ? AND qb.geofence_enabled = true
						AND i.interaction_subcategory = 'product_validation'
						AND (i.created_at AT TIME ZONE 'UTC')::date >= ?
						AND (i.created_at AT TIME ZONE 'UTC')::date <= ?
				`, tenantUUID, prevFromDate, prevToDate).Scan(&prevGeoScans)

				var prevGeoViolationRate float64
				if prevGeoScans > 0 {
					prevGeoViolationRate = float64(prevGeoViolations) / float64(prevGeoScans) * 100
				}

				geofenceData["violation_rate"] = geoViolationRate
				geofenceData["previous_violation_rate"] = prevGeoViolationRate
			}

			responseData["geofence"] = geofenceData
		}
	}

	// 2. Warranty and Campaign Template Performance - only for Intermediate+ tier
	if isIntermediatePlus {
		// Warranty Template Performance with trend
		var warrantyCurrentRaw []struct {
			TemplateID   *uuid.UUID
			TemplateName string
			Count        int64
		}
		h.DB.Raw(`
			WITH template_counts AS (
				SELECT
					qb.warranty_template_id as template_id,
					COALESCE(pt.template_name, 'Default Template') as template_name,
					COUNT(*) as count
				FROM warranty_activations wa
				JOIN qr_codes qc ON qc.id = wa.qr_code_id
				JOIN qr_batches qb ON qb.id = qc.batch_id
				LEFT JOIN page_templates pt ON pt.id = qb.warranty_template_id
				WHERE qb.tenant_id = ?
					AND (wa.activated_at AT TIME ZONE 'UTC')::date >= ?
					AND (wa.activated_at AT TIME ZONE 'UTC')::date <= ?
				GROUP BY qb.warranty_template_id, pt.template_name
				ORDER BY count DESC
			),
			top_10 AS (
				SELECT template_id, template_name, count
				FROM template_counts
				LIMIT 10
			),
			others AS (
				SELECT NULL::uuid as template_id, 'Others' as template_name, COALESCE(SUM(count), 0) as count
				FROM template_counts
				WHERE (template_id, template_name) NOT IN (SELECT template_id, template_name FROM top_10)
			)
			SELECT * FROM top_10
			UNION ALL
			SELECT * FROM others WHERE count > 0
		`, tenantUUID, fromDate, toDate).Scan(&warrantyCurrentRaw)

		var warrantyPreviousRaw []struct {
			TemplateID   *uuid.UUID
			TemplateName string
			Count        int64
		}
		h.DB.Raw(`
			SELECT
				qb.warranty_template_id as template_id,
				COALESCE(pt.template_name, 'Default Template') as template_name,
				COUNT(*) as count
			FROM warranty_activations wa
			JOIN qr_codes qc ON qc.id = wa.qr_code_id
			JOIN qr_batches qb ON qb.id = qc.batch_id
			LEFT JOIN page_templates pt ON pt.id = qb.warranty_template_id
			WHERE qb.tenant_id = ?
				AND (wa.activated_at AT TIME ZONE 'UTC')::date >= ?
				AND (wa.activated_at AT TIME ZONE 'UTC')::date <= ?
			GROUP BY qb.warranty_template_id, pt.template_name
		`, tenantUUID, prevFromDate, prevToDate).Scan(&warrantyPreviousRaw)

		prevWarrantyMap := make(map[string]int64)
		for _, p := range warrantyPreviousRaw {
			prevWarrantyMap[p.TemplateName] = p.Count
		}

		var warrantyTemplatePerformance []TemplateUsage
		for _, c := range warrantyCurrentRaw {
			prev := prevWarrantyMap[c.TemplateName]
			warrantyTemplatePerformance = append(warrantyTemplatePerformance, TemplateUsage{
				TemplateID:    c.TemplateID,
				TemplateName:  c.TemplateName,
				Count:         c.Count,
				PreviousCount: prev,
				ChangePercent: calcChangePercent(c.Count, prev),
			})
		}

		// Add to response — warranty only for dynamic (uses batch QR codes)
		templatePerf := gin.H{
			"validation": validationTemplatePerformance,
			"warranty":   warrantyTemplatePerformance,
		}
		responseData["template_performance"] = templatePerf
	}

	utils.SuccessResponse(c, http.StatusOK, "Dashboard data", responseData)
}

// GetAnalytics returns detailed analytics for a tenant
func (h *DashboardHandler) GetAnalytics(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}

	// Parse date range parameters (same as GetTenantDashboard)
	now := time.Now().UTC()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	fromDate := c.DefaultQuery("from", startOfMonth.Format("2006-01-02"))
	toDate := c.DefaultQuery("to", now.Format("2006-01-02"))

	// Parse dates
	startDate, _ := time.Parse("2006-01-02", fromDate)
	endDate, _ := time.Parse("2006-01-02", toDate)
	endDate = endDate.Add(24*time.Hour - time.Second) // End of day

	// Scan breakdown by type
	var scansByType []struct {
		Type  string `json:"type"`
		Count int64  `json:"count"`
	}
	h.DB.Model(&models.Interaction{}).
		Select("interaction_subcategory as type, count(*) as count").
		Where("tenant_id = ? AND created_at >= ? AND created_at <= ?", tenantUUID, startDate, endDate).
		Group("interaction_subcategory").
		Scan(&scansByType)

	// Top products by scans (join through qr_batches since product_id is there, not in qr_codes)
	var topProducts []struct {
		ProductID   uuid.UUID `json:"product_id"`
		ProductName string    `json:"product_name"`
		ScanCount   int64     `json:"scan_count"`
	}
	h.DB.Model(&models.Interaction{}).
		Select("qb.product_id, p.product_name, count(*) as scan_count").
		Joins("JOIN qr_codes qc ON qc.id = interactions.qr_code_id").
		Joins("JOIN qr_batches qb ON qb.id = qc.batch_id").
		Joins("JOIN products p ON p.id = qb.product_id").
		Where("interactions.tenant_id = ? AND interactions.created_at >= ? AND interactions.created_at <= ?", tenantUUID, startDate, endDate).
		Group("qb.product_id, p.product_name").
		Order("scan_count DESC").
		Limit(10).
		Scan(&topProducts)

	// Counterfeit detection trend - single GROUP BY query (fixed N+1 bug)
	var counterfeitTrend []struct {
		Date  string `json:"date"`
		Count int64  `json:"count"`
	}
	h.DB.Model(&models.CounterfeitDetection{}).
		Select("(created_at AT TIME ZONE 'UTC')::date::text as date, COUNT(*) as count").
		Where("tenant_id = ? AND (created_at AT TIME ZONE 'UTC')::date >= ? AND (created_at AT TIME ZONE 'UTC')::date <= ?",
			tenantUUID, fromDate, toDate).
		Group("(created_at AT TIME ZONE 'UTC')::date").
		Order("(created_at AT TIME ZONE 'UTC')::date ASC").
		Scan(&counterfeitTrend)

	// Template performance (A/B testing analytics)
	// Only for product_validation interactions that have a validation_template_id recorded
	var templatePerformance []struct {
		TemplateID   *uuid.UUID `json:"template_id"`
		TemplateName string     `json:"template_name"`
		ScanCount    int64      `json:"scan_count"`
	}
	h.DB.Model(&models.Interaction{}).
		Select("interactions.validation_template_id as template_id, COALESCE(page_templates.template_name, 'Default Template') as template_name, count(*) as scan_count").
		Joins("LEFT JOIN page_templates ON page_templates.id = interactions.validation_template_id").
		Where("interactions.tenant_id = ? AND interactions.created_at >= ? AND interactions.created_at <= ? AND interactions.interaction_subcategory = ?",
			tenantUUID, startDate, endDate, models.InteractionSubcategoryProductValidation).
		Group("interactions.validation_template_id, page_templates.template_name").
		Order("scan_count DESC").
		Scan(&templatePerformance)

	utils.SuccessResponse(c, http.StatusOK, "Analytics data", gin.H{
		"from":                 fromDate,
		"to":                   toDate,
		"scans_by_type":        scansByType,
		"top_products":         topProducts,
		"counterfeit_trend":    counterfeitTrend,
		"template_performance": templatePerformance,
	})
}

// GetScanHeatmap returns geolocation data for heatmap visualization
// Available only for Intermediate tier subscribers
func (h *DashboardHandler) GetScanHeatmap(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}

	isProTier := true

	// Parse query parameters
	fromDate := c.DefaultQuery("from", time.Now().UTC().AddDate(0, 0, -30).Format("2006-01-02"))
	toDate := c.DefaultQuery("to", time.Now().UTC().Format("2006-01-02"))
	source := c.DefaultQuery("source", "all") // all, validation, warranty, campaign

	// Scan filter for raw queries
	qrTypeSQL := "AND i.qr_code_id IS NOT NULL"

	// Pro-only parameters
	countryFilter := c.Query("country")                    // Filter by country code (Pro only)
	aggregateMode := c.DefaultQuery("aggregate", "points") // points, country, province, city (Pro only)

	// Set limit based on tier (Pro gets higher limit)
	limit := 1000
	if isProTier {
		limit = 5000
	}

	// Response structure
	response := gin.H{
		"filters": gin.H{
			"from":      fromDate,
			"to":        toDate,
			"source":    source,
			"country":   countryFilter,
			"aggregate": aggregateMode,
		},
	}

	// For Pro tier, get available countries list
	if isProTier {
		var availableCountries []struct {
			CountryCode string `json:"country_code"`
			CountryName string `json:"country_name"`
			ScanCount   int64  `json:"scan_count"`
		}
		h.DB.Raw(fmt.Sprintf(`
			SELECT
				COALESCE(c.code, UPPER(SUBSTRING(geolocation->>'country', 1, 2))) as country_code,
				COALESCE(c.name, geolocation->>'country') as country_name,
				COUNT(*) as scan_count
			FROM interactions i
			LEFT JOIN countries c ON c.name = i.geolocation->>'country'
			WHERE i.tenant_id = ?
				AND i.interaction_category = ?
				AND i.geolocation IS NOT NULL
				AND i.geolocation->>'country' IS NOT NULL
				%s
			GROUP BY c.code, c.name, geolocation->>'country'
			ORDER BY scan_count DESC
		`, qrTypeSQL), tenantUUID, models.InteractionCategoryEndUserAccess).Scan(&availableCountries)
		response["available_countries"] = availableCountries
	}

	// Handle aggregation modes (Pro only)
	if aggregateMode != "points" && isProTier {
		switch aggregateMode {
		case "country":
			var countryAggregates []struct {
				CountryCode string  `json:"country_code"`
				CountryName string  `json:"country_name"`
				TotalScans  int64   `json:"total_scans"`
				CenterLat   float64 `json:"center_lat"`
				CenterLng   float64 `json:"center_lng"`
			}
			h.DB.Raw(fmt.Sprintf(`
				SELECT
					COALESCE(c.code, UPPER(SUBSTRING(geolocation->>'country', 1, 2))) as country_code,
					COALESCE(c.name, geolocation->>'country') as country_name,
					COUNT(*) as total_scans,
					AVG((geolocation->>'lat')::float) as center_lat,
					AVG((geolocation->>'lng')::float) as center_lng
				FROM interactions i
				LEFT JOIN countries c ON c.name = i.geolocation->>'country'
				WHERE i.tenant_id = ?
					AND i.interaction_category = ?
					AND i.geolocation IS NOT NULL
					AND DATE(i.created_at) >= ? AND DATE(i.created_at) <= ?
					%s
				GROUP BY c.code, c.name, geolocation->>'country'
				ORDER BY total_scans DESC
				LIMIT 50
			`, qrTypeSQL), tenantUUID, models.InteractionCategoryEndUserAccess, fromDate, toDate).Scan(&countryAggregates)
			response["country_aggregates"] = countryAggregates
			response["points"] = []interface{}{}
			response["summary"] = gin.H{"total_points": 0, "valid_count": 0, "warning_count": 0, "counterfeit_count": 0}
			utils.SuccessResponse(c, http.StatusOK, "Heatmap data (country aggregation)", response)
			return

		case "province":
			// Aggregate from interactions.geolocation JSON (province populated by the
			// reverse-geocoder), mirroring the "country" case. The former
			// campaign_participations source was removed with the campaign module.
			var provinceAggregates []struct {
				ProvinceName string  `json:"province_name"`
				CountryName  string  `json:"country_name"`
				TotalScans   int64   `json:"total_scans"`
				CenterLat    float64 `json:"center_lat"`
				CenterLng    float64 `json:"center_lng"`
			}
			query := fmt.Sprintf(`
				SELECT
					i.geolocation->>'province' as province_name,
					i.geolocation->>'country' as country_name,
					COUNT(*) as total_scans,
					AVG((i.geolocation->>'lat')::float) as center_lat,
					AVG((i.geolocation->>'lng')::float) as center_lng
				FROM interactions i
				WHERE i.tenant_id = ?
					AND i.interaction_category = ?
					AND i.geolocation IS NOT NULL
					AND COALESCE(i.geolocation->>'province', '') <> ''
					AND DATE(i.created_at) >= ? AND DATE(i.created_at) <= ?
					%s`, qrTypeSQL)
			args := []interface{}{tenantUUID, models.InteractionCategoryEndUserAccess, fromDate, toDate}

			if countryFilter != "" {
				query += " AND UPPER(SUBSTRING(COALESCE(i.geolocation->>'country',''),1,2)) = UPPER(?)"
				args = append(args, countryFilter)
			}

			query += " GROUP BY i.geolocation->>'province', i.geolocation->>'country' ORDER BY total_scans DESC LIMIT 50"
			h.DB.Raw(query, args...).Scan(&provinceAggregates)
			response["province_aggregates"] = provinceAggregates
			response["points"] = []interface{}{}
			response["summary"] = gin.H{"total_points": 0, "valid_count": 0, "warning_count": 0, "counterfeit_count": 0}
			utils.SuccessResponse(c, http.StatusOK, "Heatmap data (province aggregation)", response)
			return

		case "city":
			var cityAggregates []struct {
				CityName   string  `json:"city_name"`
				TotalScans int64   `json:"total_scans"`
				CenterLat  float64 `json:"center_lat"`
				CenterLng  float64 `json:"center_lng"`
			}
			query := fmt.Sprintf(`
				SELECT
					i.geolocation->>'city' as city_name,
					COUNT(*) as total_scans,
					AVG((i.geolocation->>'lat')::float) as center_lat,
					AVG((i.geolocation->>'lng')::float) as center_lng
				FROM interactions i
				WHERE i.tenant_id = ?
					AND i.interaction_category = ?
					AND i.geolocation IS NOT NULL
					AND COALESCE(i.geolocation->>'city', '') <> ''
					AND DATE(i.created_at) >= ? AND DATE(i.created_at) <= ?
					%s`, qrTypeSQL)
			args := []interface{}{tenantUUID, models.InteractionCategoryEndUserAccess, fromDate, toDate}

			if countryFilter != "" {
				query += " AND UPPER(SUBSTRING(COALESCE(i.geolocation->>'country',''),1,2)) = UPPER(?)"
				args = append(args, countryFilter)
			}

			query += " GROUP BY i.geolocation->>'city' ORDER BY total_scans DESC LIMIT 100"
			h.DB.Raw(query, args...).Scan(&cityAggregates)
			response["city_aggregates"] = cityAggregates
			response["points"] = []interface{}{}
			response["summary"] = gin.H{"total_points": 0, "valid_count": 0, "warning_count": 0, "counterfeit_count": 0}
			utils.SuccessResponse(c, http.StatusOK, "Heatmap data (city aggregation)", response)
			return
		}
	}

	// Build query for interactions with geolocation (points mode)
	var query *gorm.DB
	// Dynamic QR: interactions linked through qr_codes
	query = h.DB.Table("interactions i").
		Select(`
			i.id,
			i.geolocation,
			i.interaction_subcategory,
			i.created_at,
			p.product_name,
			qc.counterfeit_status,
			qc.id as qr_code_id,
			qb.id as batch_id
		`).
		Joins("JOIN qr_codes qc ON qc.id = i.qr_code_id").
		Joins("JOIN qr_batches qb ON qb.id = qc.batch_id").
		Joins("JOIN products p ON p.id = qb.product_id").
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

	// Filter by country (Pro only)
	if countryFilter != "" && isProTier {
		query = query.Where("UPPER(i.geolocation->>'country') = UPPER(?)", countryFilter)
	}

	// Execute query
	var rawData []struct {
		ID                     uuid.UUID `json:"id"`
		Geolocation            []byte    `json:"geolocation"`
		InteractionSubcategory string    `json:"interaction_subcategory"`
		CreatedAt              time.Time `json:"created_at"`
		ProductName            string    `json:"product_name"`
		CounterfeitStatus      string    `json:"counterfeit_status"`
		QrCodeID               uuid.UUID `json:"qr_code_id"`
		BatchID                uuid.UUID `json:"batch_id"`
	}

	if err := query.Order("i.created_at DESC").Limit(limit).Scan(&rawData).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to fetch heatmap data", err)
		return
	}

	// Parse geolocation and build response
	type HeatmapPoint struct {
		Lat               float64   `json:"lat"`
		Lng               float64   `json:"lng"`
		ProductName       string    `json:"product_name"`
		CounterfeitStatus string    `json:"counterfeit_status"`
		ScanType          string    `json:"scan_type"`
		CreatedAt         time.Time `json:"created_at"`
		QrCodeID          uuid.UUID `json:"qr_code_id"`
		BatchID           uuid.UUID `json:"batch_id"`
	}

	var heatmapData []HeatmapPoint
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

		// Skip if no valid coordinates
		if geo.Lat == 0 && geo.Lng == 0 {
			continue
		}

		heatmapData = append(heatmapData, HeatmapPoint{
			Lat:               geo.Lat,
			Lng:               geo.Lng,
			ProductName:       row.ProductName,
			CounterfeitStatus: row.CounterfeitStatus,
			ScanType:          row.InteractionSubcategory,
			CreatedAt:         row.CreatedAt,
			QrCodeID:          row.QrCodeID,
			BatchID:           row.BatchID,
		})
	}

	// Get summary stats
	var summary struct {
		TotalPoints            int `json:"total_points"`
		ValidCount             int `json:"valid_count"`
		WarningCount           int `json:"warning_count"`
		CounterfeitCount       int `json:"counterfeit_count"`
		GeofenceViolationCount int `json:"geofence_violation_count"`
	}

	summary.TotalPoints = len(heatmapData)
	for _, point := range heatmapData {
		switch point.CounterfeitStatus {
		case string(models.CounterfeitStatusCounterfeit):
			summary.CounterfeitCount++
		default:
			// "valid" and legacy "warning" both count as valid
			summary.ValidCount++
		}
	}

	// Query geofence violations for the same date range (dynamic QR only)
	type GeoViolationPoint struct {
		Lat       float64   `json:"lat"`
		Lng       float64   `json:"lng"`
		Severity  string    `json:"severity"`
		BatchName string    `json:"batch_name"`
		CreatedAt time.Time `json:"created_at"`
		BatchID   uuid.UUID `json:"batch_id"`
	}
	var geoViolations []GeoViolationPoint
	if true {
		h.DB.Model(&models.GeofenceViolation{}).
			Select("geofence_violations.scan_latitude as lat, geofence_violations.scan_longitude as lng, geofence_violations.severity, qr_batches.batch_name, geofence_violations.created_at, geofence_violations.batch_id").
			Joins("JOIN qr_batches ON qr_batches.id = geofence_violations.batch_id AND qr_batches.tenant_id = ?", tenantUUID).
			Where("geofence_violations.tenant_id = ?", tenantUUID).
			Where("DATE(geofence_violations.created_at) >= ? AND DATE(geofence_violations.created_at) <= ?", fromDate, toDate).
			Where("geofence_violations.scan_latitude != 0 OR geofence_violations.scan_longitude != 0").
			Order("geofence_violations.created_at DESC").
			Limit(limit).
			Scan(&geoViolations)
	}

	summary.GeofenceViolationCount = len(geoViolations)

	response["points"] = heatmapData
	response["summary"] = summary
	response["geofence_violations"] = geoViolations

	utils.SuccessResponse(c, http.StatusOK, "Heatmap data", response)
}
