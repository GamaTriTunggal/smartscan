package handlers

import (
	"bytes"
	"encoding/json"
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

func setupOverrideTestData(t *testing.T) (tenantID, userID uuid.UUID, detection *models.CounterfeitDetection, cleanup func()) {
	t.Helper()

	tenantID = uuid.Must(uuid.NewV7())
	tenant := &models.Tenant{
		ID:          tenantID,
		CompanyName: "Override Test " + tenantID.String()[:8],
	}
	require.NoError(t, testDB.Create(tenant).Error)

	userID = uuid.Must(uuid.NewV7())
	user := &models.User{
		ID:           userID,
		Email:        "override-" + userID.String()[:8] + "@test.com",
		PasswordHash: "$2a$10$4CL4Pjw5L4aQvPXj37VTiO7.FeSs0UjIQsSgBcrh/.jxBqGGm6ji6",
		UserType:     models.UserTypeTenantStaff,
	}
	require.NoError(t, testDB.Create(user).Error)

	staff := &models.TenantStaff{
		TenantID:       tenantID,
		UserID:         userID,
		FullName:       "Test Staff",
		IsPrimaryAdmin: true,
	}
	require.NoError(t, testDB.Create(staff).Error)

	product := &models.Product{
		TenantID:    tenantID,
		ProductName: "Override Product",
	}
	require.NoError(t, testDB.Create(product).Error)

	batch := &models.QRBatch{
		TenantID:  tenantID,
		ProductID: product.ID,
		BatchName: "Override Batch",
		QRCount:   1,
	}
	require.NoError(t, testDB.Create(batch).Error)

	qrCode := &models.QRCode{
		BatchID: batch.ID,
		QRUUID:  uuid.Must(uuid.NewV7()),
		QRCode:  "OVR-" + uuid.Must(uuid.NewV7()).String()[:8],
	}
	require.NoError(t, testDB.Create(qrCode).Error)

	// Create some interactions so scan count > 0
	for i := 0; i < 5; i++ {
		interaction := &models.Interaction{
			QRCodeID:               &qrCode.ID,
			TenantID:               tenantID,
			InteractionCategory:    models.InteractionCategoryEndUserAccess,
			InteractionSubcategory: models.InteractionSubcategoryProductValidation,
			InteractionStatus:      models.InteractionStatusSuccess,
			IPAddress:              "127.0.0.1",
		}
		require.NoError(t, testDB.Create(interaction).Error)
	}

	// Create active detection
	now := time.Now().UTC()
	detection = &models.CounterfeitDetection{
		QRCodeID:               qrCode.ID,
		TenantID:               tenantID,
		DetectionReason:        "Excessive validation attempts: 5 (threshold: 3)",
		TotalInteractionsCount: 5,
		FirstInteractionAt:     &now,
		LastInteractionAt:      &now,
		Status:                 models.CounterfeitDetectionStatusActive,
	}
	require.NoError(t, testDB.Create(detection).Error)

	cleanup = func() {
		// Wait for async audit log goroutine to complete
		time.Sleep(300 * time.Millisecond)
		// Delete audit logs first (FK to users)
		testDB.Where("user_id = ?", userID).Delete(&models.ActivityLog{})
		testDB.Where("tenant_id = ?", tenantID).Delete(&models.ActivityLog{})
		testDB.Where("qr_code_id = ?", qrCode.ID).Delete(&models.Interaction{})
		testDB.Unscoped().Delete(detection)
		testDB.Unscoped().Delete(qrCode)
		testDB.Unscoped().Delete(batch)
		testDB.Unscoped().Delete(product)
		testDB.Unscoped().Delete(staff)
		testDB.Unscoped().Delete(user)
		testDB.Unscoped().Delete(tenant)
	}

	return tenantID, userID, detection, cleanup
}

func setupOverrideRouter(h *CounterfeitHandler, tenantID, userID uuid.UUID) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("tenant_id", tenantID)
		c.Set("user_id", userID)
		c.Next()
	})
	r.POST("/api/v1/tenant/counterfeit/:id/override-threshold", h.OverrideThreshold)
	return r
}

func TestOverrideThreshold_QRLevel(t *testing.T) {
	if testDB == nil {
		t.Skip("Test database not available")
	}

	tenantID, userID, detection, cleanup := setupOverrideTestData(t)
	defer cleanup()

	handler := NewCounterfeitHandler(testDB, testCfg)
	router := setupOverrideRouter(handler, tenantID, userID)

	body, _ := json.Marshal(map[string]interface{}{
		"level":         "qr",
		"new_threshold": 15,
		"reason":        "Store display item",
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/tenant/counterfeit/"+detection.ID.String()+"/override-threshold", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify QR code got the threshold
	var qr models.QRCode
	testDB.First(&qr, "id = ?", detection.QRCodeID)
	require.NotNil(t, qr.CounterfeitScanMax)
	assert.Equal(t, 15, *qr.CounterfeitScanMax)

	// Verify QR status reset to valid
	assert.Equal(t, models.CounterfeitStatusValid, qr.CounterfeitStatus)

	// Verify detection marked as false_positive
	var d models.CounterfeitDetection
	testDB.First(&d, "id = ?", detection.ID)
	assert.Equal(t, models.CounterfeitDetectionStatusFalsePositive, d.Status)
}

func TestOverrideThreshold_BatchLevel(t *testing.T) {
	if testDB == nil {
		t.Skip("Test database not available")
	}

	tenantID, userID, detection, cleanup := setupOverrideTestData(t)
	defer cleanup()

	handler := NewCounterfeitHandler(testDB, testCfg)
	router := setupOverrideRouter(handler, tenantID, userID)

	body, _ := json.Marshal(map[string]interface{}{
		"level":         "batch",
		"new_threshold": 20,
		"reason":        "Warehouse scanning",
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/tenant/counterfeit/"+detection.ID.String()+"/override-threshold", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify batch got the threshold
	var qr models.QRCode
	testDB.First(&qr, "id = ?", detection.QRCodeID)
	var batch models.QRBatch
	testDB.First(&batch, "id = ?", qr.BatchID)
	require.NotNil(t, batch.CounterfeitScanMax)
	assert.Equal(t, 20, *batch.CounterfeitScanMax)
}

func TestOverrideThreshold_ProductLevel(t *testing.T) {
	if testDB == nil {
		t.Skip("Test database not available")
	}

	tenantID, userID, detection, cleanup := setupOverrideTestData(t)
	defer cleanup()

	handler := NewCounterfeitHandler(testDB, testCfg)
	router := setupOverrideRouter(handler, tenantID, userID)

	body, _ := json.Marshal(map[string]interface{}{
		"level":         "product",
		"new_threshold": 25,
		"reason":        "High-traffic product",
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/tenant/counterfeit/"+detection.ID.String()+"/override-threshold", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify product got the threshold
	var qr models.QRCode
	testDB.Preload("Batch").First(&qr, "id = ?", detection.QRCodeID)
	var product models.Product
	testDB.First(&product, "id = ?", qr.Batch.ProductID)
	require.NotNil(t, product.CounterfeitScanMax)
	assert.Equal(t, 25, *product.CounterfeitScanMax)
}

func TestOverrideThreshold_ThresholdMustExceedScanCount(t *testing.T) {
	if testDB == nil {
		t.Skip("Test database not available")
	}

	tenantID, userID, detection, cleanup := setupOverrideTestData(t)
	defer cleanup()

	handler := NewCounterfeitHandler(testDB, testCfg)
	router := setupOverrideRouter(handler, tenantID, userID)

	// Try setting threshold to 3 (less than scan count of 5)
	body, _ := json.Marshal(map[string]interface{}{
		"level":         "qr",
		"new_threshold": 3,
		"reason":        "Too low",
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/tenant/counterfeit/"+detection.ID.String()+"/override-threshold", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestOverrideThreshold_DetectionMustBeActive(t *testing.T) {
	if testDB == nil {
		t.Skip("Test database not available")
	}

	tenantID, userID, detection, cleanup := setupOverrideTestData(t)
	defer cleanup()

	// Mark as false positive first (non-active status)
	testDB.Model(detection).Update("status", models.CounterfeitDetectionStatusFalsePositive)

	handler := NewCounterfeitHandler(testDB, testCfg)
	router := setupOverrideRouter(handler, tenantID, userID)

	body, _ := json.Marshal(map[string]interface{}{
		"level":         "qr",
		"new_threshold": 15,
		"reason":        "Already marked false positive",
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/tenant/counterfeit/"+detection.ID.String()+"/override-threshold", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestOverrideThreshold_TenantIsolation(t *testing.T) {
	if testDB == nil {
		t.Skip("Test database not available")
	}

	_, _, detection, cleanup := setupOverrideTestData(t)
	defer cleanup()

	// Use a different tenant ID
	differentTenantID := uuid.Must(uuid.NewV7())
	differentUserID := uuid.Must(uuid.NewV7())

	// Create the different tenant + user + staff
	diffTenant := &models.Tenant{
		ID:          differentTenantID,
		CompanyName: "Different Tenant " + differentTenantID.String()[:8],
	}
	require.NoError(t, testDB.Create(diffTenant).Error)
	defer testDB.Unscoped().Delete(diffTenant)

	diffUser := &models.User{
		ID:           differentUserID,
		Email:        "diff-" + differentUserID.String()[:8] + "@test.com",
		PasswordHash: "$2a$10$4CL4Pjw5L4aQvPXj37VTiO7.FeSs0UjIQsSgBcrh/.jxBqGGm6ji6",
		UserType:     models.UserTypeTenantStaff,
	}
	require.NoError(t, testDB.Create(diffUser).Error)
	defer testDB.Unscoped().Delete(diffUser)

	diffStaff := &models.TenantStaff{
		TenantID:       differentTenantID,
		UserID:         differentUserID,
		FullName:       "Diff Staff",
		IsPrimaryAdmin: true,
	}
	require.NoError(t, testDB.Create(diffStaff).Error)
	defer testDB.Unscoped().Delete(diffStaff)

	handler := NewCounterfeitHandler(testDB, testCfg)
	router := setupOverrideRouter(handler, differentTenantID, differentUserID)

	body, _ := json.Marshal(map[string]interface{}{
		"level":         "qr",
		"new_threshold": 15,
		"reason":        "Cross-tenant attempt",
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/tenant/counterfeit/"+detection.ID.String()+"/override-threshold", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	// Should be 404 because detection belongs to a different tenant
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestOverrideThreshold_AuditLogCreated(t *testing.T) {
	if testDB == nil {
		t.Skip("Test database not available")
	}

	tenantID, userID, detection, cleanup := setupOverrideTestData(t)
	defer cleanup()

	handler := NewCounterfeitHandler(testDB, testCfg)
	router := setupOverrideRouter(handler, tenantID, userID)

	body, _ := json.Marshal(map[string]interface{}{
		"level":         "qr",
		"new_threshold": 15,
		"reason":        "Audit test",
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/tenant/counterfeit/"+detection.ID.String()+"/override-threshold", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Wait for async audit log
	time.Sleep(200 * time.Millisecond)

	// Verify audit log
	var log models.ActivityLog
	err := testDB.Where("action_type = ? AND tenant_id = ?", models.ActionTypeThresholdOverride, tenantID).
		Order("created_at DESC").First(&log).Error
	assert.NoError(t, err)
	assert.Equal(t, "qr_code", log.EntityType)
}

func TestOverrideThreshold_BoundaryThresholdEqualsScanCount(t *testing.T) {
	if testDB == nil {
		t.Skip("Test database not available")
	}

	tenantID, userID, detection, cleanup := setupOverrideTestData(t)
	defer cleanup()

	handler := NewCounterfeitHandler(testDB, testCfg)
	router := setupOverrideRouter(handler, tenantID, userID)

	// Threshold exactly equal to scan count (5) should fail
	body, _ := json.Marshal(map[string]interface{}{
		"level":         "qr",
		"new_threshold": 5,
		"reason":        "Boundary test",
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/tenant/counterfeit/"+detection.ID.String()+"/override-threshold", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestOverrideThreshold_InvalidLevel(t *testing.T) {
	if testDB == nil {
		t.Skip("Test database not available")
	}

	tenantID, userID, detection, cleanup := setupOverrideTestData(t)
	defer cleanup()

	handler := NewCounterfeitHandler(testDB, testCfg)
	router := setupOverrideRouter(handler, tenantID, userID)

	// level = "tenant" is not valid (only qr/batch/product)
	body, _ := json.Marshal(map[string]interface{}{
		"level":         "tenant",
		"new_threshold": 15,
		"reason":        "Invalid level test",
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/tenant/counterfeit/"+detection.ID.String()+"/override-threshold", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestOverrideThreshold_MissingReason(t *testing.T) {
	if testDB == nil {
		t.Skip("Test database not available")
	}

	tenantID, userID, detection, cleanup := setupOverrideTestData(t)
	defer cleanup()

	handler := NewCounterfeitHandler(testDB, testCfg)
	router := setupOverrideRouter(handler, tenantID, userID)

	// Missing reason field
	body, _ := json.Marshal(map[string]interface{}{
		"level":         "qr",
		"new_threshold": 15,
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/tenant/counterfeit/"+detection.ID.String()+"/override-threshold", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
