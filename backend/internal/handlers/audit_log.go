package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gamatritunggal/smartscan/backend/internal/config"
	"github.com/gamatritunggal/smartscan/backend/internal/models"
	"github.com/gamatritunggal/smartscan/backend/internal/utils"
	"gorm.io/gorm"
)

type AuditLogHandler struct {
	DB  *gorm.DB
	Cfg *config.Config
}

func NewAuditLogHandler(db *gorm.DB, cfg *config.Config) *AuditLogHandler {
	return &AuditLogHandler{DB: db, Cfg: cfg}
}

// ListAuditLogs returns paginated audit logs with filters for the caller's tenant.
// GET /api/v1/tenant/audit-logs
func (h *AuditLogHandler) ListAuditLogs(c *gin.Context) {
	// Tenant isolation: audit logs must never cross tenant boundaries. Derive the
	// tenant from the authenticated context and scope every query to it — the
	// client-supplied tenant_id filter is intentionally ignored.
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Tenant context required", nil)
		return
	}

	page := utils.GetPageQuery(c)
	limit := utils.GetLimitQuery(c, 20)
	offset := (page - 1) * limit

	query := h.DB.Model(&models.ActivityLog{}).Where("tenant_id = ?", tenantUUID)

	// Filters
	if userID := c.Query("user_id"); userID != "" {
		query = query.Where("user_id = ?", userID)
	}
	if actionType := c.Query("action_type"); actionType != "" {
		query = query.Where("action_type = ?", actionType)
	}
	if entityType := c.Query("entity_type"); entityType != "" {
		query = query.Where("entity_type = ?", entityType)
	}
	if dateFrom := c.Query("date_from"); dateFrom != "" {
		if t, err := time.Parse("2006-01-02", dateFrom); err == nil {
			query = query.Where("created_at >= ?", t)
		}
	}
	if dateTo := c.Query("date_to"); dateTo != "" {
		if t, err := time.Parse("2006-01-02", dateTo); err == nil {
			query = query.Where("created_at < ?", t.AddDate(0, 0, 1))
		}
	}
	if search := c.Query("search"); search != "" {
		query = query.Where("ip_address::text ILIKE ? OR entity_type ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	var total int64
	query.Count(&total)

	var logs []models.ActivityLog
	query.Preload("User").Preload("Tenant").
		Order("created_at DESC").
		Offset(offset).Limit(limit).
		Find(&logs)

	// Build response with flattened user/tenant info
	type AuditLogResponse struct {
		models.ActivityLog
		UserEmail   string `json:"user_email,omitempty"`
		CompanyName string `json:"company_name,omitempty"`
	}

	results := make([]AuditLogResponse, len(logs))
	for i, log := range logs {
		results[i] = AuditLogResponse{ActivityLog: log}
		if log.User != nil {
			results[i].UserEmail = log.User.Email
		}
		if log.Tenant != nil {
			results[i].CompanyName = log.Tenant.CompanyName
		}
	}

	utils.SuccessResponse(c, http.StatusOK, "Audit logs retrieved", gin.H{
		"logs": results,
		"pagination": gin.H{
			"page":       page,
			"limit":      limit,
			"total":      total,
			"total_page": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// GetAuditLogStats returns aggregated audit log statistics for the visual dashboard
// GET /api/v1/tenant/audit-logs/stats?period=30d
func (h *AuditLogHandler) GetAuditLogStats(c *gin.Context) {
	// Tenant isolation: aggregate only over the caller's own tenant.
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Tenant context required", nil)
		return
	}

	// Parse period
	period := c.DefaultQuery("period", "30d")
	days := 30
	switch period {
	case "7d":
		days = 7
	case "90d":
		days = 90
	default:
		period = "30d"
		days = 30
	}

	cutoff := time.Now().UTC().AddDate(0, 0, -days)

	// 1. Summary counts (single query with conditional aggregation)
	type SummaryRow struct {
		TotalEvents    int64
		SecurityEvents int64
		UniqueUsers    int64
		UniqueIPs      int64
	}
	var summary SummaryRow
	h.DB.Model(&models.ActivityLog{}).Where("tenant_id = ?", tenantUUID).Where("created_at >= ?", cutoff).
		Select(`COUNT(*) AS total_events,
			COUNT(*) FILTER (WHERE action_type IN ('delete','password_reset','export')) AS security_events,
			COUNT(DISTINCT user_id) AS unique_users,
			COUNT(DISTINCT NULLIF(ip_address::text, '')) AS unique_ips`).
		Scan(&summary)

	// 2. By action type
	type ActionCount struct {
		ActionType string `json:"action_type"`
		Count      int64  `json:"count"`
	}
	var byAction []ActionCount
	h.DB.Model(&models.ActivityLog{}).Where("tenant_id = ?", tenantUUID).Where("created_at >= ?", cutoff).
		Select("action_type, COUNT(*) as count").
		Group("action_type").
		Order("count DESC").
		Find(&byAction)

	// 3. By entity type (top 10)
	type EntityCount struct {
		EntityType string `json:"entity_type"`
		Count      int64  `json:"count"`
	}
	var byEntity []EntityCount
	h.DB.Model(&models.ActivityLog{}).Where("tenant_id = ?", tenantUUID).Where("created_at >= ?", cutoff).
		Select("entity_type, COUNT(*) as count").
		Group("entity_type").
		Order("count DESC").
		Limit(10).
		Find(&byEntity)

	// 4. Daily trend
	type DailyCount struct {
		Date  string `json:"date"`
		Count int64  `json:"count"`
	}
	var dailyTrend []DailyCount
	h.DB.Model(&models.ActivityLog{}).Where("tenant_id = ?", tenantUUID).Where("created_at >= ?", cutoff).
		Select("TO_CHAR(created_at, 'YYYY-MM-DD') as date, COUNT(*) as count").
		Group("date").
		Order("date ASC").
		Find(&dailyTrend)

	// 5. Top 5 users
	type TopUser struct {
		UserID     string `json:"user_id"`
		Email      string `json:"email"`
		EventCount int64  `json:"event_count"`
	}
	var topUsers []TopUser
	h.DB.Model(&models.ActivityLog{}).
		Select("activity_logs.user_id, users.email, COUNT(*) as event_count").
		Joins("JOIN users ON users.id = activity_logs.user_id").
		Where("activity_logs.tenant_id = ?", tenantUUID).
		Where("activity_logs.created_at >= ?", cutoff).
		Where("activity_logs.user_id IS NOT NULL").
		Group("activity_logs.user_id, users.email").
		Order("event_count DESC").
		Limit(5).
		Find(&topUsers)

	utils.SuccessResponse(c, http.StatusOK, "Audit log stats retrieved", gin.H{
		"period": period,
		"summary": gin.H{
			"total_events":    summary.TotalEvents,
			"security_events": summary.SecurityEvents,
			"unique_users":    summary.UniqueUsers,
			"unique_ips":      summary.UniqueIPs,
		},
		"by_action":   byAction,
		"by_entity":   byEntity,
		"daily_trend": dailyTrend,
		"top_users":   topUsers,
	})
}
