package handlers

import (
	"bytes"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gamatritunggal/smartscan/backend/internal/config"
	"github.com/gamatritunggal/smartscan/backend/internal/models"
	"github.com/gamatritunggal/smartscan/backend/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// scanningStringPtr is a helper to create string pointer
func scanningStringPtr(s string) *string {
	return &s
}

func getScanningTestConfig() *config.Config {
	return &config.Config{
		AppEnv:      "test",
		FrontendURL: "http://localhost:3000",
		JWT: config.JWTConfig{
			Secret:          "test-secret-key-for-jwt-signing",
			ExpirationHours: 24,
			RefreshHours:    168,
		},
	}
}

func setupScanningTestRouter(h *ScanningHandler, tenantID, userID uuid.UUID) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	// Middleware to inject tenant_id and user_id
	r.Use(func(c *gin.Context) {
		c.Set("tenant_id", tenantID)
		c.Set("user_id", userID)
		c.Next()
	})

	r.POST("/api/v1/tenant/scanning/qc", h.QCScan)
	r.GET("/api/v1/tenant/scanning/qc/history", h.GetQCHistory)
	return r
}

// setupQCScanTestData creates tenant + user + staff for QC scan tests.
// The handler checks staff membership BEFORE input validation, so all tests
// need a valid staff record to reach the validation code path.
func setupQCScanTestData(t *testing.T) (tenantID, userID uuid.UUID, cleanup func()) {
	t.Helper()
	uniq := uuid.Must(uuid.NewV7()).String()[:8]

	tenantID = uuid.Must(uuid.NewV7())
	tenant := &models.Tenant{
		ID:           tenantID,
		CompanyName:  "QCScan Test " + uniq,
		CompanyEmail: "qcscan-" + uniq + "@test.com",
		CountryCode:  scanningStringPtr("ID"),
	}
	require.NoError(t, testDB.Create(tenant).Error)

	userID = uuid.Must(uuid.NewV7())
	user := &models.User{
		ID:       userID,
		Email:    "qcstaff-" + uniq + "@test.com",
		UserType: models.UserTypeTenantStaff,
		Status:   models.UserStatusActive,
	}
	require.NoError(t, testDB.Create(user).Error)

	staff := &models.TenantStaff{
		ID:       uuid.Must(uuid.NewV7()),
		UserID:   userID,
		TenantID: tenantID,
		FullName: "QC Staff " + uniq,
		Role:     models.TenantStaffRoleQCStaff,
	}
	require.NoError(t, testDB.Create(staff).Error)

	cleanup = func() {
		testDB.Unscoped().Delete(staff)
		testDB.Unscoped().Delete(user)
		testDB.Unscoped().Delete(tenant)
	}
	return
}

// Test the Haversine distance calculation
func TestCalculateDistance(t *testing.T) {
	// Test with known distances
	tests := []struct {
		name      string
		lat1      float64
		lng1      float64
		lat2      float64
		lng2      float64
		expected  float64 // expected distance in meters (approximate)
		tolerance float64
	}{
		{
			name:      "Same point",
			lat1:      -6.2088,
			lng1:      106.8456,
			lat2:      -6.2088,
			lng2:      106.8456,
			expected:  0,
			tolerance: 1,
		},
		{
			name:      "Jakarta to Bandung (approx 120km)",
			lat1:      -6.2088,
			lng1:      106.8456,
			lat2:      -6.9175,
			lng2:      107.6191,
			expected:  118000, // ~118 km
			tolerance: 5000,   // 5km tolerance
		},
		{
			name:      "Short distance (100m)",
			lat1:      -6.2088,
			lng1:      106.8456,
			lat2:      -6.2097,
			lng2:      106.8456,
			expected:  100,
			tolerance: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			distance := calculateDistance(tt.lat1, tt.lng1, tt.lat2, tt.lng2)
			diff := math.Abs(distance - tt.expected)
			assert.LessOrEqual(t, diff, tt.tolerance,
				"Distance %.0f should be within %.0f of expected %.0f", distance, tt.tolerance, tt.expected)
		})
	}
}

func TestQCScan_InvalidRequest(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}

	tenantID, userID, cleanup := setupQCScanTestData(t)
	defer cleanup()

	cfg := getScanningTestConfig()
	handler := NewScanningHandler(testDB, cfg)
	router := setupScanningTestRouter(handler, tenantID, userID)

	// Send invalid JSON
	req, _ := http.NewRequest("POST", "/api/v1/tenant/scanning/qc", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestQCScan_MissingRequiredFields(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}

	tenantID, userID, cleanup := setupQCScanTestData(t)
	defer cleanup()

	cfg := getScanningTestConfig()
	handler := NewScanningHandler(testDB, cfg)
	router := setupScanningTestRouter(handler, tenantID, userID)

	// Missing location_id and qr_code
	input := map[string]string{
		"status": "pass",
	}
	body, _ := json.Marshal(input)

	req, _ := http.NewRequest("POST", "/api/v1/tenant/scanning/qc", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestQCScan_InvalidLocationID(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}

	tenantID, userID, cleanup := setupQCScanTestData(t)
	defer cleanup()

	cfg := getScanningTestConfig()
	handler := NewScanningHandler(testDB, cfg)
	router := setupScanningTestRouter(handler, tenantID, userID)

	input := map[string]interface{}{
		"location_id": "invalid-uuid",
		"qr_code":     "QR123",
		"status":      "pass",
	}
	body, _ := json.Marshal(input)

	req, _ := http.NewRequest("POST", "/api/v1/tenant/scanning/qc", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestQCScan_QCAreaNotFound(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}

	tenantID, userID, cleanup := setupQCScanTestData(t)
	defer cleanup()

	cfg := getScanningTestConfig()
	handler := NewScanningHandler(testDB, cfg)
	router := setupScanningTestRouter(handler, tenantID, userID)

	// Valid UUID but non-existent location
	input := map[string]interface{}{
		"location_id": uuid.Must(uuid.NewV7()).String(),
		"qr_code":     "QR123",
		"status":      "pass",
	}
	body, _ := json.Marshal(input)

	req, _ := http.NewRequest("POST", "/api/v1/tenant/scanning/qc", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response utils.APIResponse
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Contains(t, response.Message, "QC area not found")
}

func TestQCScan_StaffNotFound(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}

	cfg := getScanningTestConfig()
	handler := NewScanningHandler(testDB, cfg)

	// Create tenant and location but NOT staff — this tests the 401 path
	uniq := uuid.Must(uuid.NewV7()).String()[:8]
	tenantID := uuid.Must(uuid.NewV7())
	tenant := &models.Tenant{
		ID:           tenantID,
		CompanyName:  "QC StaffNotFound " + uniq,
		CompanyEmail: "qcstaffnotfound-" + uniq + "@test.com",
		CountryCode:  scanningStringPtr("ID"),
	}
	require.NoError(t, testDB.Create(tenant).Error)
	defer testDB.Unscoped().Delete(tenant)

	// Create QC location
	locationID := uuid.Must(uuid.NewV7())
	location := &models.TenantLocation{
		ID:           locationID,
		TenantID:     tenantID,
		LocationName: "Test QC Area",
		LocationType: models.LocationTypeQCArea,
		Status:       "active",
	}
	require.NoError(t, testDB.Create(location).Error)
	defer testDB.Unscoped().Delete(location)

	// Use non-existent user ID
	userID := uuid.Must(uuid.NewV7())
	router := setupScanningTestRouter(handler, tenantID, userID)

	input := map[string]interface{}{
		"location_id": locationID.String(),
		"qr_code":     "QR123",
		"status":      "pass",
	}
	body, _ := json.Marshal(input)

	req, _ := http.NewRequest("POST", "/api/v1/tenant/scanning/qc", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestQCScan_QRCodeNotFound(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}

	cfg := getScanningTestConfig()
	handler := NewScanningHandler(testDB, cfg)

	// Create full setup (tenant, user, staff, location)
	uniq := uuid.Must(uuid.NewV7()).String()[:8]
	tenantID := uuid.Must(uuid.NewV7())
	tenant := &models.Tenant{
		ID:           tenantID,
		CompanyName:  "QC QRNotFound " + uniq,
		CompanyEmail: "qcqrnotfound-" + uniq + "@test.com",
		CountryCode:  scanningStringPtr("ID"),
	}
	require.NoError(t, testDB.Create(tenant).Error)
	defer testDB.Unscoped().Delete(tenant)

	userID := uuid.Must(uuid.NewV7())
	user := &models.User{
		ID:       userID,
		Email:    "qcstaff-qrnf-" + uniq + "@test.com",
		UserType: models.UserTypeTenantStaff,
		Status:   models.UserStatusActive,
	}
	require.NoError(t, testDB.Create(user).Error)
	defer testDB.Unscoped().Delete(user)

	staff := &models.TenantStaff{
		ID:       uuid.Must(uuid.NewV7()),
		UserID:   userID,
		TenantID: tenantID,
		FullName: "QC Staff",
		Role:     models.TenantStaffRoleQCStaff,
	}
	require.NoError(t, testDB.Create(staff).Error)
	defer testDB.Unscoped().Delete(staff)

	locationID := uuid.Must(uuid.NewV7())
	location := &models.TenantLocation{
		ID:           locationID,
		TenantID:     tenantID,
		LocationName: "Test QC Area",
		LocationType: models.LocationTypeQCArea,
		Status:       "active",
	}
	require.NoError(t, testDB.Create(location).Error)
	defer testDB.Unscoped().Delete(location)

	router := setupScanningTestRouter(handler, tenantID, userID)

	// Use a valid UUID that doesn't exist as a QR code
	input := map[string]interface{}{
		"location_id": locationID.String(),
		"qr_code":     uuid.Must(uuid.NewV7()).String(),
		"status":      "pass",
	}
	body, _ := json.Marshal(input)

	req, _ := http.NewRequest("POST", "/api/v1/tenant/scanning/qc", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestQCScan_Success(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}

	cfg := getScanningTestConfig()
	handler := NewScanningHandler(testDB, cfg)

	// Create full test data with unique identifiers
	uniq := uuid.Must(uuid.NewV7()).String()[:8]
	tenantID := uuid.Must(uuid.NewV7())
	tenant := &models.Tenant{
		ID:           tenantID,
		CompanyName:  "QC Success " + uniq,
		CompanyEmail: "qcsuccess-" + uniq + "@test.com",
		CountryCode:  scanningStringPtr("ID"),
	}
	require.NoError(t, testDB.Create(tenant).Error)
	defer testDB.Unscoped().Delete(tenant)

	userID := uuid.Must(uuid.NewV7())
	user := &models.User{
		ID:       userID,
		Email:    "qcstaff-success-" + uniq + "@test.com",
		UserType: models.UserTypeTenantStaff,
		Status:   models.UserStatusActive,
	}
	require.NoError(t, testDB.Create(user).Error)
	defer testDB.Unscoped().Delete(user)

	staffID := uuid.Must(uuid.NewV7())
	staff := &models.TenantStaff{
		ID:       staffID,
		UserID:   userID,
		TenantID: tenantID,
		FullName: "QC Staff " + uniq,
		Role:     models.TenantStaffRoleQCStaff,
	}
	require.NoError(t, testDB.Create(staff).Error)
	defer testDB.Unscoped().Delete(staff)

	locationID := uuid.Must(uuid.NewV7())
	location := &models.TenantLocation{
		ID:           locationID,
		TenantID:     tenantID,
		LocationName: "Test QC Area " + uniq,
		LocationType: models.LocationTypeQCArea,
		Status:       "active",
	}
	require.NoError(t, testDB.Create(location).Error)
	defer testDB.Unscoped().Delete(location)

	productID := uuid.Must(uuid.NewV7())
	product := &models.Product{
		ID:          productID,
		TenantID:    tenantID,
		ProductName: "Test Product " + uniq,
		ProductCode: "TST-" + uniq,
		Status:      models.ProductStatusActive,
	}
	require.NoError(t, testDB.Create(product).Error)
	defer testDB.Unscoped().Delete(product)

	batchID := uuid.Must(uuid.NewV7())
	batch := &models.QRBatch{
		ID:        batchID,
		TenantID:  tenantID,
		ProductID: productID,
		BatchName: "Test QC Batch " + uniq,
		QRCount:   10,
	}
	require.NoError(t, testDB.Create(batch).Error)
	defer testDB.Unscoped().Delete(batch)

	qrCodeID := uuid.Must(uuid.NewV7())
	qrUUID := uuid.Must(uuid.NewV7())
	qrCode := &models.QRCode{
		ID:      qrCodeID,
		BatchID: batchID,
		QRCode:  "QC-" + uniq,
		QRUUID:  qrUUID,
		Status:  models.QRCodeStatusActive,
	}
	require.NoError(t, testDB.Create(qrCode).Error)
	defer func() {
		testDB.Unscoped().Where("qr_code_id = ?", qrCodeID).Delete(&models.QCScan{})
		testDB.Unscoped().Where("qr_code_id = ?", qrCodeID).Delete(&models.Interaction{})
		testDB.Unscoped().Delete(qrCode)
	}()

	router := setupScanningTestRouter(handler, tenantID, userID)

	// Use qr_uuid (UUID format) — the handler's OR query needs a valid UUID
	// for the qr_uuid column comparison to avoid PostgreSQL type cast error
	input := map[string]interface{}{
		"location_id": locationID.String(),
		"qr_code":     qrUUID.String(),
		"status":      "pass",
		"latitude":    -6.2088,
		"longitude":   106.8456,
	}
	body, _ := json.Marshal(input)

	req, _ := http.NewRequest("POST", "/api/v1/tenant/scanning/qc", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	// httptest requests have no RemoteAddr; production always does, and the
	// handler stores c.ClientIP() into an inet column
	req.RemoteAddr = "203.0.113.10:52341"

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response utils.APIResponse
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.True(t, response.Success)

	// Verify QC scan was created
	var qcScan models.QCScan
	err := testDB.Where("qr_code_id = ?", qrCodeID).First(&qcScan).Error
	assert.NoError(t, err)
	assert.Equal(t, models.QCStatusPass, qcScan.QCStatus)

	// Verify the analytics interaction row was persisted — regression test:
	// interactions.scanned_by has an FK to users(id), so writing the staff ID
	// here used to make the insert fail silently and drop the row.
	var interaction models.Interaction
	err = testDB.Where("qr_code_id = ? AND interaction_subcategory = ?", qrCodeID, models.InteractionSubcategoryQCScan).First(&interaction).Error
	assert.NoError(t, err, "QC interaction row must be persisted")
	if assert.NotNil(t, interaction.ScannedBy) {
		assert.Equal(t, userID, *interaction.ScannedBy, "interaction.scanned_by must be the user ID (users FK), not the staff ID")
	}
}

func TestQCScan_InvalidStatus(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}

	tenantID, userID, cleanup := setupQCScanTestData(t)
	defer cleanup()

	cfg := getScanningTestConfig()
	handler := NewScanningHandler(testDB, cfg)
	router := setupScanningTestRouter(handler, tenantID, userID)

	input := map[string]interface{}{
		"location_id": uuid.Must(uuid.NewV7()).String(),
		"qr_code":     "QR123",
		"status":      "invalid_status", // Should be "pass" or "failed"
	}
	body, _ := json.Marshal(input)

	req, _ := http.NewRequest("POST", "/api/v1/tenant/scanning/qc", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetQCHistory_Empty(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}

	cfg := getScanningTestConfig()
	handler := NewScanningHandler(testDB, cfg)

	// Use a tenant with no QC history
	tenantID := uuid.Must(uuid.NewV7())
	userID := uuid.Must(uuid.NewV7())

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("tenant_id", tenantID)
		c.Set("user_id", userID)
		c.Next()
	})
	router.GET("/api/v1/tenant/scanning/qc/history", handler.GetQCHistory)

	req, _ := http.NewRequest("GET", "/api/v1/tenant/scanning/qc/history", nil)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response utils.APIResponse
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.True(t, response.Success)
}
