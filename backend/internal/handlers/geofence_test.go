package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gamatritunggal/smartscan/backend/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── Helpers ────────────────────────────────────────────────────────────────

type geofenceTestFixture struct {
	TenantID  uuid.UUID
	UserID    uuid.UUID
	BatchID   uuid.UUID
	ProductID uuid.UUID
	Handler   *GeofenceHandler
}

func setupGeofenceTest(t *testing.T) (*geofenceTestFixture, func()) {
	t.Helper()
	if testDB == nil {
		t.Skip("Database not available")
	}
	uniq := uuid.New().String()[:8]

	tenantID := uuid.Must(uuid.NewV7())
	countryCode := "ID"
	tenant := &models.Tenant{
		ID: tenantID, CompanyName: "GeoTest " + uniq,
		CompanyEmail: "geo-" + uniq + "@test.com", CountryCode: &countryCode,
	}
	require.NoError(t, testDB.Create(tenant).Error)

	userID := uuid.Must(uuid.NewV7())
	user := &models.User{
		ID: userID, Email: "geouser-" + uniq + "@test.com",
		UserType: models.UserTypeTenantStaff, Status: models.UserStatusActive,
	}
	require.NoError(t, testDB.Create(user).Error)

	staff := &models.TenantStaff{
		ID: uuid.Must(uuid.NewV7()), TenantID: tenantID, UserID: userID,
		FullName: "Geo Test Admin", Role: models.TenantStaffRoleAdmin,
	}
	require.NoError(t, testDB.Create(staff).Error)

	productID := uuid.Must(uuid.NewV7())
	product := &models.Product{
		ID: productID, TenantID: tenantID,
		ProductName: "Geo Product " + uniq, ProductCode: "GEO-" + uniq,
		Status: models.ProductStatusActive,
	}
	require.NoError(t, testDB.Create(product).Error)

	lat, lng, radius := -6.2088, 106.8456, 50.0
	batchID := uuid.Must(uuid.NewV7())
	batch := &models.QRBatch{
		ID: batchID, TenantID: tenantID, ProductID: productID,
		BatchName: "Geo Batch " + uniq, QRCount: 10,
		GeofenceEnabled: true, GeofenceLatitude: &lat,
		GeofenceLongitude: &lng, GeofenceRadiusKm: &radius,
		GeofenceLabel: "Jakarta Zone",
	}
	require.NoError(t, testDB.Create(batch).Error)

	handler := NewGeofenceHandler(testDB, testCfg)

	fix := &geofenceTestFixture{
		TenantID: tenantID, UserID: userID,
		BatchID: batchID, ProductID: productID,
		Handler: handler,
	}

	cleanup := func() {
		testDB.Exec("DELETE FROM geofence_violations WHERE tenant_id = ?", tenantID)
		testDB.Exec("DELETE FROM geofence_zone_templates WHERE tenant_id = ?", tenantID)
		testDB.Unscoped().Where("batch_id = ?", batchID).Delete(&models.QRCode{})
		testDB.Unscoped().Where("id = ?", batchID).Delete(&models.QRBatch{})
		testDB.Unscoped().Where("id = ?", productID).Delete(&models.Product{})
		testDB.Unscoped().Where("tenant_id = ?", tenantID).Delete(&models.TenantStaff{})
		testDB.Unscoped().Where("id = ?", userID).Delete(&models.User{})
		testDB.Unscoped().Where("id = ?", tenantID).Delete(&models.Tenant{})
	}
	return fix, cleanup
}

func createViolation(t *testing.T, tenantID, batchID uuid.UUID, severity string, distEdge float64) {
	t.Helper()
	v := &models.GeofenceViolation{
		TenantID:             tenantID,
		BatchID:              batchID,
		ScanLatitude:         -7.5,
		ScanLongitude:        110.0,
		DistanceFromCenterKm: distEdge + 50.0,
		DistanceFromEdgeKm:   distEdge,
		Severity:             severity,
	}
	require.NoError(t, testDB.Create(v).Error)
}

func createViolationAt(t *testing.T, tenantID, batchID uuid.UUID, severity string, distEdge, lat, lng float64) {
	t.Helper()
	v := &models.GeofenceViolation{
		TenantID:             tenantID,
		BatchID:              batchID,
		ScanLatitude:         lat,
		ScanLongitude:        lng,
		DistanceFromCenterKm: distEdge + 50.0,
		DistanceFromEdgeKm:   distEdge,
		Severity:             severity,
	}
	require.NoError(t, testDB.Create(v).Error)
}

func setupGeofenceRouter(h *GeofenceHandler, tenantID uuid.UUID) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("tenant_id", tenantID.String())
		c.Next()
	})
	r.GET("/api/v1/tenant/geofence/areas", h.GetGeofenceAreas)
	r.GET("/api/v1/tenant/geofence/violations", h.ListGeofenceViolations)
	r.GET("/api/v1/tenant/geofence/stats", h.GetGeofenceStats)
	r.GET("/api/v1/tenant/geofence/analytics", h.GetGeofenceAnalytics)
	r.GET("/api/v1/tenant/geofence/violations/export", h.ExportGeofenceViolations)
	r.GET("/api/v1/tenant/geofence/map-data", h.GetGeofenceMapData)
	r.GET("/api/v1/tenant/geofence/zone-templates", h.ListZoneTemplates)
	r.POST("/api/v1/tenant/geofence/zone-templates", h.CreateZoneTemplate)
	r.PUT("/api/v1/tenant/geofence/zone-templates/:id", h.UpdateZoneTemplate)
	r.DELETE("/api/v1/tenant/geofence/zone-templates/:id", h.DeleteZoneTemplate)
	r.GET("/api/v1/tenant/qr-batches/:id/geofence-violations", h.GetBatchGeofenceViolations)
	r.GET("/api/v1/tenant/qr-batches/:id/geofence-analytics", h.GetBatchGeofenceAnalytics)
	return r
}

func parseResponse(t *testing.T, w *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	return resp
}

func getData(t *testing.T, resp map[string]interface{}) map[string]interface{} {
	t.Helper()
	data, ok := resp["data"].(map[string]interface{})
	require.True(t, ok, "response should have 'data' field")
	return data
}

// ─── calcPreviousPeriod tests ───────────────────────────────────────────────

func TestCalcPreviousPeriod_ThisMonth(t *testing.T) {
	prevFrom, prevTo, label, err := calcPreviousPeriod("this_month", "2026-03-01", "2026-03-15")
	require.NoError(t, err)
	assert.Equal(t, "2026-02-01", prevFrom)
	assert.Equal(t, "2026-02-28", prevTo)
	assert.Equal(t, "vs last month", label)
}

func TestCalcPreviousPeriod_LastMonth(t *testing.T) {
	prevFrom, prevTo, label, err := calcPreviousPeriod("last_month", "2026-02-01", "2026-02-28")
	require.NoError(t, err)
	assert.Equal(t, "2026-01-01", prevFrom)
	assert.Equal(t, "2026-01-31", prevTo)
	assert.Contains(t, label, "vs Jan")
}

func TestCalcPreviousPeriod_Last7Days(t *testing.T) {
	prevFrom, prevTo, label, err := calcPreviousPeriod("last_7_days", "2026-02-22", "2026-02-28")
	require.NoError(t, err)
	assert.Equal(t, "2026-02-15", prevFrom)
	assert.Equal(t, "2026-02-21", prevTo)
	assert.Contains(t, label, "previous 7 days")
}

func TestCalcPreviousPeriod_Last30Days(t *testing.T) {
	prevFrom, prevTo, label, err := calcPreviousPeriod("last_30_days", "2026-01-30", "2026-02-28")
	require.NoError(t, err)
	assert.NotEmpty(t, prevFrom)
	assert.NotEmpty(t, prevTo)
	assert.Contains(t, label, "previous")
}

func TestCalcPreviousPeriod_DefaultCustom(t *testing.T) {
	prevFrom, prevTo, label, err := calcPreviousPeriod("custom", "2026-01-01", "2026-01-31")
	require.NoError(t, err)
	assert.Equal(t, "2025-01-01", prevFrom)
	assert.Equal(t, "2025-01-31", prevTo)
	assert.Equal(t, "vs same period last year", label)
}

func TestCalcPreviousPeriod_InvalidFromDate(t *testing.T) {
	_, _, _, err := calcPreviousPeriod("this_month", "bad-date", "2026-03-01")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid 'from' date")
}

func TestCalcPreviousPeriod_InvalidToDate(t *testing.T) {
	_, _, _, err := calcPreviousPeriod("this_month", "2026-03-01", "bad-date")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid 'to' date")
}

func TestCalcPreviousPeriod_YearBoundary(t *testing.T) {
	prevFrom, prevTo, _, err := calcPreviousPeriod("this_month", "2026-01-01", "2026-01-31")
	require.NoError(t, err)
	assert.Equal(t, "2025-12-01", prevFrom)
	assert.Equal(t, "2025-12-31", prevTo)
}

// ─── ListGeofenceViolations tests ───────────────────────────────────────────

func TestListGeofenceViolations_Empty(t *testing.T) {
	fix, cleanup := setupGeofenceTest(t)
	defer cleanup()

	router := setupGeofenceRouter(fix.Handler, fix.TenantID)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/tenant/geofence/violations", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseResponse(t, w)
	data := getData(t, resp)
	violations := data["violations"].([]interface{})
	assert.Len(t, violations, 0)
	pagination := data["pagination"].(map[string]interface{})
	assert.Equal(t, float64(0), pagination["total"])
}

func TestListGeofenceViolations_WithData(t *testing.T) {
	fix, cleanup := setupGeofenceTest(t)
	defer cleanup()

	createViolation(t, fix.TenantID, fix.BatchID, "low", 5)
	createViolation(t, fix.TenantID, fix.BatchID, "medium", 30)
	createViolation(t, fix.TenantID, fix.BatchID, "high", 100)

	router := setupGeofenceRouter(fix.Handler, fix.TenantID)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/tenant/geofence/violations", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseResponse(t, w)
	data := getData(t, resp)
	violations := data["violations"].([]interface{})
	assert.Len(t, violations, 3)
	pagination := data["pagination"].(map[string]interface{})
	assert.Equal(t, float64(3), pagination["total"])
}

func TestListGeofenceViolations_FilterBySeverity(t *testing.T) {
	fix, cleanup := setupGeofenceTest(t)
	defer cleanup()

	createViolation(t, fix.TenantID, fix.BatchID, "low", 5)
	createViolation(t, fix.TenantID, fix.BatchID, "high", 100)
	createViolation(t, fix.TenantID, fix.BatchID, "high", 150)

	router := setupGeofenceRouter(fix.Handler, fix.TenantID)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/tenant/geofence/violations?severity=high", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseResponse(t, w)
	data := getData(t, resp)
	violations := data["violations"].([]interface{})
	assert.Len(t, violations, 2)
}

func TestListGeofenceViolations_FilterByBatchID(t *testing.T) {
	fix, cleanup := setupGeofenceTest(t)
	defer cleanup()

	// Create a second batch
	lat, lng, radius := -7.0, 110.0, 30.0
	batch2ID := uuid.Must(uuid.NewV7())
	batch2 := &models.QRBatch{
		ID: batch2ID, TenantID: fix.TenantID, ProductID: fix.ProductID,
		BatchName: "Batch 2", QRCount: 5,
		GeofenceEnabled: true, GeofenceLatitude: &lat,
		GeofenceLongitude: &lng, GeofenceRadiusKm: &radius,
	}
	require.NoError(t, testDB.Create(batch2).Error)
	defer testDB.Unscoped().Where("id = ?", batch2ID).Delete(&models.QRBatch{})

	createViolation(t, fix.TenantID, fix.BatchID, "low", 5)
	createViolation(t, fix.TenantID, batch2ID, "high", 100)

	router := setupGeofenceRouter(fix.Handler, fix.TenantID)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/tenant/geofence/violations?batch_id=%s", batch2ID), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseResponse(t, w)
	data := getData(t, resp)
	violations := data["violations"].([]interface{})
	assert.Len(t, violations, 1)
}

func TestListGeofenceViolations_Pagination(t *testing.T) {
	fix, cleanup := setupGeofenceTest(t)
	defer cleanup()

	for i := 0; i < 25; i++ {
		createViolation(t, fix.TenantID, fix.BatchID, "low", float64(i+1))
	}

	router := setupGeofenceRouter(fix.Handler, fix.TenantID)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/tenant/geofence/violations?page=2&limit=10", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseResponse(t, w)
	data := getData(t, resp)
	violations := data["violations"].([]interface{})
	assert.Len(t, violations, 10)
	pagination := data["pagination"].(map[string]interface{})
	assert.Equal(t, float64(25), pagination["total"])
	assert.Equal(t, float64(2), pagination["page"])
	assert.Equal(t, float64(3), pagination["total_page"])
}

func TestListGeofenceViolations_TenantIsolation(t *testing.T) {
	fix, cleanup := setupGeofenceTest(t)
	defer cleanup()

	createViolation(t, fix.TenantID, fix.BatchID, "low", 5)

	// Query with a different tenant ID — should return 0 violations
	otherTenantID := uuid.Must(uuid.NewV7())
	router := setupGeofenceRouter(fix.Handler, otherTenantID)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/tenant/geofence/violations", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseResponse(t, w)
	data := getData(t, resp)
	violations := data["violations"].([]interface{})
	assert.Len(t, violations, 0)
}

// ─── GetGeofenceStats tests ─────────────────────────────────────────────────

func TestGetGeofenceStats_Empty(t *testing.T) {
	fix, cleanup := setupGeofenceTest(t)
	defer cleanup()

	router := setupGeofenceRouter(fix.Handler, fix.TenantID)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/tenant/geofence/stats", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseResponse(t, w)
	data := getData(t, resp)
	assert.Equal(t, float64(0), data["total_violations"])
}

func TestGetGeofenceStats_BySeverity(t *testing.T) {
	fix, cleanup := setupGeofenceTest(t)
	defer cleanup()

	createViolation(t, fix.TenantID, fix.BatchID, "low", 5)
	createViolation(t, fix.TenantID, fix.BatchID, "low", 8)
	createViolation(t, fix.TenantID, fix.BatchID, "high", 100)
	createViolation(t, fix.TenantID, fix.BatchID, "critical", 300)

	now := time.Now().UTC()
	from := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
	to := now.Format("2006-01-02")

	router := setupGeofenceRouter(fix.Handler, fix.TenantID)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/tenant/geofence/stats?from=%s&to=%s", from, to), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseResponse(t, w)
	data := getData(t, resp)
	assert.Equal(t, float64(4), data["total_violations"])

	bySeverity := data["by_severity"].([]interface{})
	assert.NotEmpty(t, bySeverity)
}

func TestGetGeofenceStats_InvalidFromDate(t *testing.T) {
	fix, cleanup := setupGeofenceTest(t)
	defer cleanup()

	router := setupGeofenceRouter(fix.Handler, fix.TenantID)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/tenant/geofence/stats?from=bad-date", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetGeofenceStats_TopBatches(t *testing.T) {
	fix, cleanup := setupGeofenceTest(t)
	defer cleanup()

	createViolation(t, fix.TenantID, fix.BatchID, "low", 5)
	createViolation(t, fix.TenantID, fix.BatchID, "medium", 30)

	now := time.Now().UTC()
	from := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
	to := now.Format("2006-01-02")

	router := setupGeofenceRouter(fix.Handler, fix.TenantID)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/tenant/geofence/stats?from=%s&to=%s", from, to), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseResponse(t, w)
	data := getData(t, resp)
	topBatches := data["top_batches"].([]interface{})
	assert.NotEmpty(t, topBatches)
}

// ─── GetGeofenceAnalytics tests ─────────────────────────────────────────────

func TestGetGeofenceAnalytics_Empty(t *testing.T) {
	fix, cleanup := setupGeofenceTest(t)
	defer cleanup()

	router := setupGeofenceRouter(fix.Handler, fix.TenantID)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/tenant/geofence/analytics", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseResponse(t, w)
	data := getData(t, resp)
	assert.NotNil(t, data["violation_rate"])
	assert.NotNil(t, data["distance_stats"])
}

func TestGetGeofenceAnalytics_WithViolations(t *testing.T) {
	fix, cleanup := setupGeofenceTest(t)
	defer cleanup()

	createViolation(t, fix.TenantID, fix.BatchID, "low", 5)
	createViolation(t, fix.TenantID, fix.BatchID, "high", 100)

	now := time.Now().UTC()
	from := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
	to := now.Format("2006-01-02")

	router := setupGeofenceRouter(fix.Handler, fix.TenantID)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/tenant/geofence/analytics?from=%s&to=%s", from, to), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseResponse(t, w)
	data := getData(t, resp)
	assert.NotNil(t, data["trends"])
	assert.NotNil(t, data["by_product"])
}

func TestGetGeofenceAnalytics_InvalidDate(t *testing.T) {
	fix, cleanup := setupGeofenceTest(t)
	defer cleanup()

	router := setupGeofenceRouter(fix.Handler, fix.TenantID)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/tenant/geofence/analytics?from=invalid&to=also-bad", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ─── GetBatchGeofenceViolations tests ───────────────────────────────────────

func TestGetBatchGeofenceViolations_Success(t *testing.T) {
	fix, cleanup := setupGeofenceTest(t)
	defer cleanup()

	createViolation(t, fix.TenantID, fix.BatchID, "low", 5)
	createViolation(t, fix.TenantID, fix.BatchID, "medium", 30)

	router := setupGeofenceRouter(fix.Handler, fix.TenantID)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/tenant/qr-batches/%s/geofence-violations", fix.BatchID), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseResponse(t, w)
	data := getData(t, resp)
	violations := data["violations"].([]interface{})
	assert.Len(t, violations, 2)
}

func TestGetBatchGeofenceViolations_InvalidBatchID(t *testing.T) {
	fix, cleanup := setupGeofenceTest(t)
	defer cleanup()

	router := setupGeofenceRouter(fix.Handler, fix.TenantID)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/tenant/qr-batches/not-a-uuid/geofence-violations", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetBatchGeofenceViolations_BatchNotFound(t *testing.T) {
	fix, cleanup := setupGeofenceTest(t)
	defer cleanup()

	fakeBatchID := uuid.Must(uuid.NewV7())
	router := setupGeofenceRouter(fix.Handler, fix.TenantID)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/tenant/qr-batches/%s/geofence-violations", fakeBatchID), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetBatchGeofenceViolations_WrongTenant(t *testing.T) {
	fix, cleanup := setupGeofenceTest(t)
	defer cleanup()

	// Use a random tenant ID — batch won't match
	otherTenantID := uuid.Must(uuid.NewV7())
	router := setupGeofenceRouter(fix.Handler, otherTenantID)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/tenant/qr-batches/%s/geofence-violations", fix.BatchID), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetBatchGeofenceViolations_FilterBySeverity(t *testing.T) {
	fix, cleanup := setupGeofenceTest(t)
	defer cleanup()

	createViolation(t, fix.TenantID, fix.BatchID, "low", 5)
	createViolation(t, fix.TenantID, fix.BatchID, "critical", 300)

	router := setupGeofenceRouter(fix.Handler, fix.TenantID)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/tenant/qr-batches/%s/geofence-violations?severity=critical", fix.BatchID), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseResponse(t, w)
	data := getData(t, resp)
	violations := data["violations"].([]interface{})
	assert.Len(t, violations, 1)
}

// ─── GetBatchGeofenceAnalytics tests (Pro tier gated) ───────────────────────

func TestGetBatchGeofenceAnalytics_Success(t *testing.T) {
	fix, cleanup := setupGeofenceTest(t)
	defer cleanup()

	createViolation(t, fix.TenantID, fix.BatchID, "high", 100)

	router := setupGeofenceRouter(fix.Handler, fix.TenantID)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/tenant/qr-batches/%s/geofence-analytics", fix.BatchID), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseResponse(t, w)
	data := getData(t, resp)
	assert.Equal(t, fix.BatchID.String(), data["batch_id"])
	assert.NotNil(t, data["total_violations"])
	assert.NotNil(t, data["by_severity"])
}

func TestGetBatchGeofenceAnalytics_InvalidBatchID(t *testing.T) {
	fix, cleanup := setupGeofenceTest(t)
	defer cleanup()


	router := setupGeofenceRouter(fix.Handler, fix.TenantID)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/tenant/qr-batches/not-a-uuid/geofence-analytics", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetBatchGeofenceAnalytics_BatchNotFound(t *testing.T) {
	fix, cleanup := setupGeofenceTest(t)
	defer cleanup()


	fakeBatchID := uuid.Must(uuid.NewV7())
	router := setupGeofenceRouter(fix.Handler, fix.TenantID)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/tenant/qr-batches/%s/geofence-analytics", fakeBatchID), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ─── ExportGeofenceViolations tests (Pro tier gated) ────────────────────────

func TestExportGeofenceViolations_Success(t *testing.T) {
	fix, cleanup := setupGeofenceTest(t)
	defer cleanup()

	createViolation(t, fix.TenantID, fix.BatchID, "low", 5)

	router := setupGeofenceRouter(fix.Handler, fix.TenantID)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/tenant/geofence/violations/export", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "spreadsheetml.sheet")
	assert.Contains(t, w.Header().Get("Content-Disposition"), "geofence_violations.xlsx")
	assert.Greater(t, w.Body.Len(), 0)
}

func TestExportGeofenceViolations_EmptyData(t *testing.T) {
	fix, cleanup := setupGeofenceTest(t)
	defer cleanup()


	router := setupGeofenceRouter(fix.Handler, fix.TenantID)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/tenant/geofence/violations/export", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "spreadsheetml.sheet")
	assert.Greater(t, w.Body.Len(), 0) // still valid xlsx with header row
}

// ─── GetGeofenceMapData tests ───────────────────────────────────────────────

func TestGetGeofenceMapData_Empty(t *testing.T) {
	fix, cleanup := setupGeofenceTest(t)
	defer cleanup()

	router := setupGeofenceRouter(fix.Handler, fix.TenantID)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/tenant/geofence/map-data", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseResponse(t, w)
	data := getData(t, resp)
	violations := data["violations"].([]interface{})
	assert.Len(t, violations, 0)
}

func TestGetGeofenceMapData_WithData(t *testing.T) {
	fix, cleanup := setupGeofenceTest(t)
	defer cleanup()

	createViolationAt(t, fix.TenantID, fix.BatchID, "high", 100, -7.5, 110.0)
	createViolationAt(t, fix.TenantID, fix.BatchID, "low", 5, -6.5, 107.0)

	router := setupGeofenceRouter(fix.Handler, fix.TenantID)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/tenant/geofence/map-data", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseResponse(t, w)
	data := getData(t, resp)
	violations := data["violations"].([]interface{})
	assert.Len(t, violations, 2)
	zones := data["zones"].([]interface{})
	assert.NotEmpty(t, zones) // at least the fixture batch zone
}

func TestGetGeofenceMapData_ExcludesNullIsland(t *testing.T) {
	fix, cleanup := setupGeofenceTest(t)
	defer cleanup()

	// Normal violation
	createViolationAt(t, fix.TenantID, fix.BatchID, "low", 5, -7.5, 110.0)
	// Null island violation (0,0)
	createViolationAt(t, fix.TenantID, fix.BatchID, "high", 100, 0, 0)

	router := setupGeofenceRouter(fix.Handler, fix.TenantID)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/tenant/geofence/map-data", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseResponse(t, w)
	data := getData(t, resp)
	violations := data["violations"].([]interface{})
	assert.Len(t, violations, 1, "null island (0,0) should be excluded from map data")
}

func TestGetGeofenceMapData_FilterBySeverity(t *testing.T) {
	fix, cleanup := setupGeofenceTest(t)
	defer cleanup()

	createViolationAt(t, fix.TenantID, fix.BatchID, "low", 5, -7.5, 110.0)
	createViolationAt(t, fix.TenantID, fix.BatchID, "critical", 300, -8.0, 112.0)

	router := setupGeofenceRouter(fix.Handler, fix.TenantID)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/tenant/geofence/map-data?severity=critical", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseResponse(t, w)
	data := getData(t, resp)
	violations := data["violations"].([]interface{})
	assert.Len(t, violations, 1)
}

// ─── Zone Templates (CRUD) tests ───────────────────────────────────────────

func TestListZoneTemplates_Empty(t *testing.T) {
	fix, cleanup := setupGeofenceTest(t)
	defer cleanup()

	router := setupGeofenceRouter(fix.Handler, fix.TenantID)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/tenant/geofence/zone-templates", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseResponse(t, w)
	data := getData(t, resp)
	templates := data["zone_templates"].([]interface{})
	assert.Len(t, templates, 0)
}

func TestListZoneTemplates_WithData(t *testing.T) {
	fix, cleanup := setupGeofenceTest(t)
	defer cleanup()

	zt := &models.GeofenceZoneTemplate{
		TenantID: fix.TenantID, TemplateName: "Test Zone",
		Latitude: -6.2, Longitude: 106.8, RadiusKm: 50, Label: "Jakarta",
	}
	require.NoError(t, testDB.Create(zt).Error)

	router := setupGeofenceRouter(fix.Handler, fix.TenantID)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/tenant/geofence/zone-templates", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseResponse(t, w)
	data := getData(t, resp)
	templates := data["zone_templates"].([]interface{})
	assert.Len(t, templates, 1)
}

func TestListZoneTemplates_TenantIsolation(t *testing.T) {
	fix, cleanup := setupGeofenceTest(t)
	defer cleanup()

	zt := &models.GeofenceZoneTemplate{
		TenantID: fix.TenantID, TemplateName: "My Zone",
		Latitude: -6.2, Longitude: 106.8, RadiusKm: 50,
	}
	require.NoError(t, testDB.Create(zt).Error)

	otherTenantID := uuid.Must(uuid.NewV7())
	router := setupGeofenceRouter(fix.Handler, otherTenantID)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/tenant/geofence/zone-templates", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseResponse(t, w)
	data := getData(t, resp)
	templates := data["zone_templates"].([]interface{})
	assert.Len(t, templates, 0, "should not see another tenant's templates")
}

func TestCreateZoneTemplate_Success(t *testing.T) {
	fix, cleanup := setupGeofenceTest(t)
	defer cleanup()

	body, _ := json.Marshal(map[string]interface{}{
		"template_name": "Semarang Zone",
		"latitude":      -6.9666,
		"longitude":     110.4196,
		"radius_km":     30,
		"label":         "Semarang Metro",
	})

	router := setupGeofenceRouter(fix.Handler, fix.TenantID)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/tenant/geofence/zone-templates", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	resp := parseResponse(t, w)
	data := getData(t, resp)
	template := data["zone_template"].(map[string]interface{})
	assert.Equal(t, "Semarang Zone", template["template_name"])
	assert.Equal(t, float64(30), template["radius_km"])
}

func TestCreateZoneTemplate_MissingFields(t *testing.T) {
	fix, cleanup := setupGeofenceTest(t)
	defer cleanup()

	body, _ := json.Marshal(map[string]interface{}{
		"latitude":  -6.9666,
		"longitude": 110.4196,
		"radius_km": 30,
	})

	router := setupGeofenceRouter(fix.Handler, fix.TenantID)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/tenant/geofence/zone-templates", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateZoneTemplate_InvalidLatitude(t *testing.T) {
	fix, cleanup := setupGeofenceTest(t)
	defer cleanup()

	body, _ := json.Marshal(map[string]interface{}{
		"template_name": "Bad Zone",
		"latitude":      95.0,
		"longitude":     110.0,
		"radius_km":     30,
	})

	router := setupGeofenceRouter(fix.Handler, fix.TenantID)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/tenant/geofence/zone-templates", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	resp := parseResponse(t, w)
	assert.Contains(t, resp["message"], "latitude")
}

func TestCreateZoneTemplate_InvalidLongitude(t *testing.T) {
	fix, cleanup := setupGeofenceTest(t)
	defer cleanup()

	body, _ := json.Marshal(map[string]interface{}{
		"template_name": "Bad Zone",
		"latitude":      -6.0,
		"longitude":     200.0,
		"radius_km":     30,
	})

	router := setupGeofenceRouter(fix.Handler, fix.TenantID)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/tenant/geofence/zone-templates", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	resp := parseResponse(t, w)
	assert.Contains(t, resp["message"], "longitude")
}

func TestCreateZoneTemplate_InvalidRadius_Zero(t *testing.T) {
	fix, cleanup := setupGeofenceTest(t)
	defer cleanup()

	body, _ := json.Marshal(map[string]interface{}{
		"template_name": "Bad Zone",
		"latitude":      -6.0,
		"longitude":     106.0,
		"radius_km":     0,
	})

	router := setupGeofenceRouter(fix.Handler, fix.TenantID)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/tenant/geofence/zone-templates", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateZoneTemplate_InvalidRadius_Over500(t *testing.T) {
	fix, cleanup := setupGeofenceTest(t)
	defer cleanup()

	body, _ := json.Marshal(map[string]interface{}{
		"template_name": "Bad Zone",
		"latitude":      -6.0,
		"longitude":     106.0,
		"radius_km":     600,
	})

	router := setupGeofenceRouter(fix.Handler, fix.TenantID)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/tenant/geofence/zone-templates", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateZoneTemplate_Success(t *testing.T) {
	fix, cleanup := setupGeofenceTest(t)
	defer cleanup()

	zt := &models.GeofenceZoneTemplate{
		TenantID: fix.TenantID, TemplateName: "Old Name",
		Latitude: -6.2, Longitude: 106.8, RadiusKm: 50,
	}
	require.NoError(t, testDB.Create(zt).Error)

	body, _ := json.Marshal(map[string]interface{}{
		"template_name": "Updated Name",
		"latitude":      -6.9,
		"longitude":     110.4,
		"radius_km":     100,
		"label":         "New Label",
	})

	router := setupGeofenceRouter(fix.Handler, fix.TenantID)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/v1/tenant/geofence/zone-templates/%s", zt.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseResponse(t, w)
	data := getData(t, resp)
	template := data["zone_template"].(map[string]interface{})
	assert.Equal(t, "Updated Name", template["template_name"])
	assert.Equal(t, float64(100), template["radius_km"])
}

func TestUpdateZoneTemplate_NotFound(t *testing.T) {
	fix, cleanup := setupGeofenceTest(t)
	defer cleanup()

	fakeID := uuid.Must(uuid.NewV7())
	body, _ := json.Marshal(map[string]interface{}{
		"template_name": "X",
		"latitude":      -6.0,
		"longitude":     106.0,
		"radius_km":     30,
	})

	router := setupGeofenceRouter(fix.Handler, fix.TenantID)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/v1/tenant/geofence/zone-templates/%s", fakeID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteZoneTemplate_Success(t *testing.T) {
	fix, cleanup := setupGeofenceTest(t)
	defer cleanup()

	zt := &models.GeofenceZoneTemplate{
		TenantID: fix.TenantID, TemplateName: "To Delete",
		Latitude: -6.2, Longitude: 106.8, RadiusKm: 50,
	}
	require.NoError(t, testDB.Create(zt).Error)

	router := setupGeofenceRouter(fix.Handler, fix.TenantID)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/v1/tenant/geofence/zone-templates/%s", zt.ID), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify soft-deleted
	var deleted models.GeofenceZoneTemplate
	err := testDB.Unscoped().First(&deleted, "id = ?", zt.ID).Error
	require.NoError(t, err)
	assert.NotNil(t, deleted.DeletedAt, "should be soft-deleted")

	// Verify not visible in list
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/api/v1/tenant/geofence/zone-templates", nil)
	router.ServeHTTP(w2, req2)
	resp := parseResponse(t, w2)
	data := getData(t, resp)
	templates := data["zone_templates"].([]interface{})
	assert.Len(t, templates, 0, "deleted template should not appear in list")
}

func TestDeleteZoneTemplate_NotFound(t *testing.T) {
	fix, cleanup := setupGeofenceTest(t)
	defer cleanup()

	fakeID := uuid.Must(uuid.NewV7())
	router := setupGeofenceRouter(fix.Handler, fix.TenantID)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/v1/tenant/geofence/zone-templates/%s", fakeID), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ─── UpdateQRBatch + Zone Template Usage Tracking ──────────────────────────

func setupBatchUpdateRouter(tenantID uuid.UUID) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("tenant_id", tenantID.String())
		c.Next()
	})
	batchHandler := NewQRBatchHandler(testDB, testCfg)
	r.PUT("/api/v1/tenant/qr-batches/:id", batchHandler.UpdateQRBatch)
	return r
}

func TestUpdateQRBatch_GeofenceZoneTemplate_IncrementsUsage(t *testing.T) {
	fix, cleanup := setupGeofenceTest(t)
	defer cleanup()

	// UpdateQRBatch requires an active subscription with Intermediate+ tier (geofence is gated).

	// Create a zone template with usage_count 0
	zt := &models.GeofenceZoneTemplate{
		TenantID:     fix.TenantID,
		TemplateName: "Surabaya Zone",
		Latitude:     -7.2575,
		Longitude:    112.7521,
		RadiusKm:     30.0,
		Label:        "Surabaya",
		UsageCount:   0,
	}
	require.NoError(t, testDB.Create(zt).Error)

	router := setupBatchUpdateRouter(fix.TenantID)

	lat, lng, radius := -7.2575, 112.7521, 30.0
	body, _ := json.Marshal(map[string]interface{}{
		"geofence_enabled":          true,
		"geofence_latitude":         lat,
		"geofence_longitude":        lng,
		"geofence_radius_km":        radius,
		"geofence_label":            "Surabaya",
		"geofence_zone_template_id": zt.ID.String(),
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/v1/tenant/qr-batches/%s", fix.BatchID), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify usage_count incremented
	var updated models.GeofenceZoneTemplate
	require.NoError(t, testDB.First(&updated, "id = ?", zt.ID).Error)
	assert.Equal(t, 1, updated.UsageCount, "usage_count should be incremented to 1")
}

func TestUpdateQRBatch_GeofenceZoneTemplate_InvalidID(t *testing.T) {
	fix, cleanup := setupGeofenceTest(t)
	defer cleanup()


	router := setupBatchUpdateRouter(fix.TenantID)

	lat, lng, radius := -7.2575, 112.7521, 30.0
	body, _ := json.Marshal(map[string]interface{}{
		"geofence_enabled":          true,
		"geofence_latitude":         lat,
		"geofence_longitude":        lng,
		"geofence_radius_km":        radius,
		"geofence_label":            "Test Zone",
		"geofence_zone_template_id": "not-a-uuid",
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/v1/tenant/qr-batches/%s", fix.BatchID), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	// Should still succeed — invalid template ID is silently ignored
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify batch geofence was still updated
	var batch models.QRBatch
	require.NoError(t, testDB.First(&batch, "id = ?", fix.BatchID).Error)
	assert.InDelta(t, -7.2575, *batch.GeofenceLatitude, 0.0001)
}

func TestUpdateQRBatch_GeofenceZoneTemplate_WrongTenant(t *testing.T) {
	fix, cleanup := setupGeofenceTest(t)
	defer cleanup()


	// Create a zone template under a DIFFERENT tenant
	otherTenantID := uuid.Must(uuid.NewV7())
	otherTenant := &models.Tenant{
		ID: otherTenantID, CompanyName: "Other Corp",
		CompanyEmail: "other-" + uuid.New().String()[:8] + "@test.com",
	}
	require.NoError(t, testDB.Create(otherTenant).Error)
	defer testDB.Unscoped().Where("id = ?", otherTenantID).Delete(&models.Tenant{})

	zt := &models.GeofenceZoneTemplate{
		TenantID:     otherTenantID,
		TemplateName: "Other Tenant Zone",
		Latitude:     -6.9175,
		Longitude:    107.6191,
		RadiusKm:     20.0,
		UsageCount:   5,
	}
	require.NoError(t, testDB.Create(zt).Error)
	defer testDB.Exec("DELETE FROM geofence_zone_templates WHERE tenant_id = ?", otherTenantID)

	router := setupBatchUpdateRouter(fix.TenantID)

	lat, lng, radius := -6.9175, 107.6191, 20.0
	body, _ := json.Marshal(map[string]interface{}{
		"geofence_enabled":          true,
		"geofence_latitude":         lat,
		"geofence_longitude":        lng,
		"geofence_radius_km":        radius,
		"geofence_label":            "Bandung",
		"geofence_zone_template_id": zt.ID.String(),
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/v1/tenant/qr-batches/%s", fix.BatchID), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	// Batch update should succeed
	assert.Equal(t, http.StatusOK, w.Code)

	// But other tenant's template usage_count should NOT be incremented
	var ztCheck models.GeofenceZoneTemplate
	require.NoError(t, testDB.First(&ztCheck, "id = ?", zt.ID).Error)
	assert.Equal(t, 5, ztCheck.UsageCount, "other tenant's template usage_count should not change")
}

// ─── GetGeofenceAreas tests ─────────────────────────────────────────────────

type geofenceAreaTestFixture struct {
	geofenceTestFixture
	Product2ID uuid.UUID
	Batch2ID   uuid.UUID // Product 1, different label ("Surabaya Zone")
	Batch3ID   uuid.UUID // Product 2, same label as Batch 1 ("Jakarta Zone")
}

func setupGeofenceAreaTest(t *testing.T) (*geofenceAreaTestFixture, func()) {
	t.Helper()
	fix, baseCleanup := setupGeofenceTest(t)
	uniq := uuid.New().String()[:8]

	// Create second batch for Product 1 with different label
	lat2, lng2, radius2 := -7.2575, 112.7521, 30.0
	batch2ID := uuid.Must(uuid.NewV7())
	batch2 := &models.QRBatch{
		ID: batch2ID, TenantID: fix.TenantID, ProductID: fix.ProductID,
		BatchName: "Geo Batch Sby " + uniq, QRCount: 5,
		GeofenceEnabled: true, GeofenceLatitude: &lat2,
		GeofenceLongitude: &lng2, GeofenceRadiusKm: &radius2,
		GeofenceLabel: "Surabaya Zone",
	}
	require.NoError(t, testDB.Create(batch2).Error)

	// Create second product
	product2ID := uuid.Must(uuid.NewV7())
	product2 := &models.Product{
		ID: product2ID, TenantID: fix.TenantID,
		ProductName: "Geo Product 2 " + uniq, ProductCode: "GEO2-" + uniq,
		Status: models.ProductStatusActive,
	}
	require.NoError(t, testDB.Create(product2).Error)

	// Create batch for Product 2 with same label as Batch 1
	lat3, lng3, radius3 := -6.2088, 106.8456, 40.0
	batch3ID := uuid.Must(uuid.NewV7())
	batch3 := &models.QRBatch{
		ID: batch3ID, TenantID: fix.TenantID, ProductID: product2ID,
		BatchName: "Geo Batch P2 " + uniq, QRCount: 8,
		GeofenceEnabled: true, GeofenceLatitude: &lat3,
		GeofenceLongitude: &lng3, GeofenceRadiusKm: &radius3,
		GeofenceLabel: "Jakarta Zone",
	}
	require.NoError(t, testDB.Create(batch3).Error)

	areaFix := &geofenceAreaTestFixture{
		geofenceTestFixture: *fix,
		Product2ID:          product2ID,
		Batch2ID:            batch2ID,
		Batch3ID:            batch3ID,
	}

	cleanup := func() {
		testDB.Exec("DELETE FROM geofence_violations WHERE tenant_id = ?", fix.TenantID)
		testDB.Unscoped().Where("id IN ?", []uuid.UUID{batch2ID, batch3ID}).Delete(&models.QRBatch{})
		testDB.Unscoped().Where("id = ?", product2ID).Delete(&models.Product{})
		baseCleanup()
	}
	return areaFix, cleanup
}

func TestGetGeofenceAreas_Empty(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}
	uniq := uuid.New().String()[:8]
	tenantID := uuid.Must(uuid.NewV7())
	countryCode := "ID"
	tenant := &models.Tenant{
		ID: tenantID, CompanyName: "EmptyGeo " + uniq,
		CompanyEmail: "emptygeo-" + uniq + "@test.com", CountryCode: &countryCode,
	}
	require.NoError(t, testDB.Create(tenant).Error)
	defer testDB.Unscoped().Where("id = ?", tenantID).Delete(&models.Tenant{})

	handler := NewGeofenceHandler(testDB, testCfg)
	router := setupGeofenceRouter(handler, tenantID)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/tenant/geofence/areas", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseResponse(t, w)
	data := getData(t, resp)
	areas := data["areas"].([]interface{})
	assert.Equal(t, 0, len(areas))
}

func TestGetGeofenceAreas_WithData(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}
	fix, cleanup := setupGeofenceAreaTest(t)
	defer cleanup()

	// Create violations for different batches
	createViolation(t, fix.TenantID, fix.BatchID, "low", 5.0)   // Product 1, Jakarta Zone
	createViolation(t, fix.TenantID, fix.BatchID, "high", 15.0)  // Product 1, Jakarta Zone
	createViolation(t, fix.TenantID, fix.Batch2ID, "medium", 8.0) // Product 1, Surabaya Zone
	createViolation(t, fix.TenantID, fix.Batch3ID, "low", 3.0)   // Product 2, Jakarta Zone

	router := setupGeofenceRouter(fix.Handler, fix.TenantID)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/tenant/geofence/areas", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseResponse(t, w)
	data := getData(t, resp)
	areas := data["areas"].([]interface{})

	// Should have 3 areas: Product1/Jakarta, Product1/Surabaya, Product2/Jakarta
	assert.Equal(t, 3, len(areas))

	// Verify violation counts (areas sorted by product_name ASC, geofence_label ASC)
	for _, a := range areas {
		area := a.(map[string]interface{})
		label := area["geofence_label"].(string)
		violations := int(area["total_violations"].(float64))
		batchCount := int(area["batch_count"].(float64))

		switch {
		case area["product_id"].(string) == fix.ProductID.String() && label == "Jakarta Zone":
			assert.Equal(t, 2, violations, "Product1/Jakarta should have 2 violations")
			assert.Equal(t, 1, batchCount, "Product1/Jakarta should have 1 batch")
		case area["product_id"].(string) == fix.ProductID.String() && label == "Surabaya Zone":
			assert.Equal(t, 1, violations, "Product1/Surabaya should have 1 violation")
			assert.Equal(t, 1, batchCount)
		case area["product_id"].(string) == fix.Product2ID.String() && label == "Jakarta Zone":
			assert.Equal(t, 1, violations, "Product2/Jakarta should have 1 violation")
			assert.Equal(t, 1, batchCount)
		default:
			t.Errorf("unexpected area: %s/%s", area["product_id"], label)
		}
	}
}

func TestGetGeofenceAreas_SkipsEmptyLabel(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}
	fix, cleanup := setupGeofenceTest(t)
	defer cleanup()

	// Create batch with empty geofence_label
	lat, lng, radius := -6.2088, 106.8456, 50.0
	emptyLabelBatch := &models.QRBatch{
		ID: uuid.Must(uuid.NewV7()), TenantID: fix.TenantID, ProductID: fix.ProductID,
		BatchName: "No Label Batch", QRCount: 5,
		GeofenceEnabled: true, GeofenceLatitude: &lat,
		GeofenceLongitude: &lng, GeofenceRadiusKm: &radius,
		GeofenceLabel: "",
	}
	require.NoError(t, testDB.Create(emptyLabelBatch).Error)
	defer testDB.Unscoped().Where("id = ?", emptyLabelBatch.ID).Delete(&models.QRBatch{})

	router := setupGeofenceRouter(fix.Handler, fix.TenantID)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/tenant/geofence/areas", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseResponse(t, w)
	data := getData(t, resp)
	areas := data["areas"].([]interface{})

	// Only the labeled batch from setupGeofenceTest should appear
	assert.Equal(t, 1, len(areas))
	area := areas[0].(map[string]interface{})
	assert.Equal(t, "Jakarta Zone", area["geofence_label"].(string))
}

func TestGetGeofenceAreas_TenantIsolation(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}
	fix, cleanup := setupGeofenceTest(t)
	defer cleanup()

	// Query with a different tenant ID
	otherTenantID := uuid.Must(uuid.NewV7())
	router := setupGeofenceRouter(fix.Handler, otherTenantID)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/tenant/geofence/areas", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseResponse(t, w)
	data := getData(t, resp)
	areas := data["areas"].([]interface{})
	assert.Equal(t, 0, len(areas))
}

func TestGetGeofenceAreas_SkipsDeletedBatches(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}
	fix, cleanup := setupGeofenceTest(t)
	defer cleanup()

	// Soft-delete the batch
	now := time.Now()
	testDB.Model(&models.QRBatch{}).Where("id = ?", fix.BatchID).Update("deleted_at", now)
	defer testDB.Model(&models.QRBatch{}).Where("id = ?", fix.BatchID).Update("deleted_at", nil)

	router := setupGeofenceRouter(fix.Handler, fix.TenantID)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/tenant/geofence/areas", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseResponse(t, w)
	data := getData(t, resp)
	areas := data["areas"].([]interface{})
	assert.Equal(t, 0, len(areas))
}

// ─── Area filter tests ──────────────────────────────────────────────────────

func TestListViolations_FilterByArea(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}
	fix, cleanup := setupGeofenceAreaTest(t)
	defer cleanup()

	// Create violations across different areas
	createViolation(t, fix.TenantID, fix.BatchID, "low", 5.0)    // Product 1, Jakarta Zone
	createViolation(t, fix.TenantID, fix.BatchID, "high", 15.0)   // Product 1, Jakarta Zone
	createViolation(t, fix.TenantID, fix.Batch2ID, "medium", 8.0) // Product 1, Surabaya Zone
	createViolation(t, fix.TenantID, fix.Batch3ID, "low", 3.0)    // Product 2, Jakarta Zone

	router := setupGeofenceRouter(fix.Handler, fix.TenantID)

	// Filter by Product 1 / Jakarta Zone → should return 2 violations
	w := httptest.NewRecorder()
	url := fmt.Sprintf("/api/v1/tenant/geofence/violations?product_id=%s&geofence_label=%s",
		fix.ProductID.String(), "Jakarta+Zone")
	req, _ := http.NewRequest("GET", url, nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseResponse(t, w)
	data := getData(t, resp)
	violations := data["violations"].([]interface{})
	assert.Equal(t, 2, len(violations), "Product1/Jakarta should have 2 violations")

	// Filter by Product 1 / Surabaya Zone → should return 1 violation
	w2 := httptest.NewRecorder()
	url2 := fmt.Sprintf("/api/v1/tenant/geofence/violations?product_id=%s&geofence_label=%s",
		fix.ProductID.String(), "Surabaya+Zone")
	req2, _ := http.NewRequest("GET", url2, nil)
	router.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusOK, w2.Code)
	resp2 := parseResponse(t, w2)
	data2 := getData(t, resp2)
	violations2 := data2["violations"].([]interface{})
	assert.Equal(t, 1, len(violations2), "Product1/Surabaya should have 1 violation")
}

func TestListViolations_AreaFilterMultipleBatches(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}
	fix, cleanup := setupGeofenceAreaTest(t)
	defer cleanup()

	// Create a second batch for Product 1 with same label "Jakarta Zone"
	lat, lng, radius := -6.2088, 106.8456, 60.0
	batch4ID := uuid.Must(uuid.NewV7())
	batch4 := &models.QRBatch{
		ID: batch4ID, TenantID: fix.TenantID, ProductID: fix.ProductID,
		BatchName: "Geo Batch Jkt2", QRCount: 5,
		GeofenceEnabled: true, GeofenceLatitude: &lat,
		GeofenceLongitude: &lng, GeofenceRadiusKm: &radius,
		GeofenceLabel: "Jakarta Zone",
	}
	require.NoError(t, testDB.Create(batch4).Error)
	defer testDB.Unscoped().Where("id = ?", batch4ID).Delete(&models.QRBatch{})

	// Create violations on both batches in the same area
	createViolation(t, fix.TenantID, fix.BatchID, "low", 5.0)   // Batch 1, Jakarta Zone
	createViolation(t, fix.TenantID, batch4ID, "high", 12.0)     // Batch 4, Jakarta Zone
	createViolation(t, fix.TenantID, batch4ID, "medium", 7.0)    // Batch 4, Jakarta Zone
	createViolation(t, fix.TenantID, fix.Batch2ID, "low", 3.0)   // Different area, should not be returned

	router := setupGeofenceRouter(fix.Handler, fix.TenantID)
	w := httptest.NewRecorder()
	url := fmt.Sprintf("/api/v1/tenant/geofence/violations?product_id=%s&geofence_label=%s",
		fix.ProductID.String(), "Jakarta+Zone")
	req, _ := http.NewRequest("GET", url, nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseResponse(t, w)
	data := getData(t, resp)
	violations := data["violations"].([]interface{})
	assert.Equal(t, 3, len(violations), "Should return violations from both batches in Jakarta Zone")
}

func TestGetGeofenceStats_FilterByArea(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}
	fix, cleanup := setupGeofenceAreaTest(t)
	defer cleanup()

	createViolation(t, fix.TenantID, fix.BatchID, "low", 5.0)    // Product 1, Jakarta Zone
	createViolation(t, fix.TenantID, fix.BatchID, "high", 15.0)   // Product 1, Jakarta Zone
	createViolation(t, fix.TenantID, fix.Batch2ID, "medium", 8.0) // Product 1, Surabaya Zone

	router := setupGeofenceRouter(fix.Handler, fix.TenantID)
	w := httptest.NewRecorder()
	url := fmt.Sprintf("/api/v1/tenant/geofence/stats?product_id=%s&geofence_label=%s",
		fix.ProductID.String(), "Jakarta+Zone")
	req, _ := http.NewRequest("GET", url, nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseResponse(t, w)
	data := getData(t, resp)

	totalViolations := int(data["total_violations"].(float64))
	assert.Equal(t, 2, totalViolations, "Stats should only count Jakarta Zone violations")
}

func TestGetGeofenceMapData_FilterByArea(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}
	fix, cleanup := setupGeofenceAreaTest(t)
	defer cleanup()

	createViolationAt(t, fix.TenantID, fix.BatchID, "low", 5.0, -7.5, 110.0)   // Product 1, Jakarta Zone
	createViolationAt(t, fix.TenantID, fix.Batch2ID, "medium", 8.0, -7.3, 112.7) // Product 1, Surabaya Zone

	router := setupGeofenceRouter(fix.Handler, fix.TenantID)
	w := httptest.NewRecorder()
	url := fmt.Sprintf("/api/v1/tenant/geofence/map-data?product_id=%s&geofence_label=%s",
		fix.ProductID.String(), "Jakarta+Zone")
	req, _ := http.NewRequest("GET", url, nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseResponse(t, w)
	data := getData(t, resp)

	violations := data["violations"].([]interface{})
	assert.Equal(t, 1, len(violations), "Map should only show Jakarta Zone violations")

	zones := data["zones"].([]interface{})
	assert.Equal(t, 1, len(zones), "Map should show Jakarta Zone circle")
	zone := zones[0].(map[string]interface{})
	assert.Equal(t, "Jakarta Zone", zone["label"].(string))
}

// ─── Product-only filter tests ──────────────────────────────────────────────

func TestListViolations_FilterByProductOnly(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}
	fix, cleanup := setupGeofenceAreaTest(t)
	defer cleanup()

	// Create violations across different products and areas
	createViolation(t, fix.TenantID, fix.BatchID, "low", 5.0)    // Product 1, Jakarta Zone
	createViolation(t, fix.TenantID, fix.Batch2ID, "medium", 8.0) // Product 1, Surabaya Zone
	createViolation(t, fix.TenantID, fix.Batch3ID, "high", 3.0)   // Product 2, Jakarta Zone
	createViolation(t, fix.TenantID, fix.Batch3ID, "low", 2.0)    // Product 2, Jakarta Zone

	router := setupGeofenceRouter(fix.Handler, fix.TenantID)

	// Filter by Product 1 only (no geofence_label) → should return violations from both areas
	w := httptest.NewRecorder()
	url := fmt.Sprintf("/api/v1/tenant/geofence/violations?product_id=%s", fix.ProductID.String())
	req, _ := http.NewRequest("GET", url, nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseResponse(t, w)
	data := getData(t, resp)
	violations := data["violations"].([]interface{})
	assert.Equal(t, 2, len(violations), "Product 1 should have 2 violations across both areas")

	// Filter by Product 2 only → should return 2 violations
	w2 := httptest.NewRecorder()
	url2 := fmt.Sprintf("/api/v1/tenant/geofence/violations?product_id=%s", fix.Product2ID.String())
	req2, _ := http.NewRequest("GET", url2, nil)
	router.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusOK, w2.Code)
	resp2 := parseResponse(t, w2)
	data2 := getData(t, resp2)
	violations2 := data2["violations"].([]interface{})
	assert.Equal(t, 2, len(violations2), "Product 2 should have 2 violations")
}

func TestGetGeofenceMapData_FilterByProductOnly(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}
	fix, cleanup := setupGeofenceAreaTest(t)
	defer cleanup()

	createViolationAt(t, fix.TenantID, fix.BatchID, "low", 5.0, -7.5, 110.0)    // Product 1, Jakarta Zone
	createViolationAt(t, fix.TenantID, fix.Batch2ID, "medium", 8.0, -7.3, 112.7) // Product 1, Surabaya Zone
	createViolationAt(t, fix.TenantID, fix.Batch3ID, "high", 3.0, -6.9, 107.0)   // Product 2, Jakarta Zone

	router := setupGeofenceRouter(fix.Handler, fix.TenantID)

	// Filter by Product 1 only → should show violations and zones from both areas
	w := httptest.NewRecorder()
	url := fmt.Sprintf("/api/v1/tenant/geofence/map-data?product_id=%s", fix.ProductID.String())
	req, _ := http.NewRequest("GET", url, nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	resp := parseResponse(t, w)
	data := getData(t, resp)

	violations := data["violations"].([]interface{})
	assert.Equal(t, 2, len(violations), "Map should show Product 1 violations from both areas")

	zones := data["zones"].([]interface{})
	assert.Equal(t, 2, len(zones), "Map should show both zone circles for Product 1")
}
