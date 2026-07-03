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
	"gorm.io/datatypes"
)

// ==================== PARTIAL UPDATE TESTS ====================

// TestSaveToUpdates_ProductPartialUpdate verifies that updating one field
// does NOT overwrite other fields (the original bug that prompted this refactor).
func TestSaveToUpdates_ProductPartialUpdate(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}

	tenantID, productID, cleanup := setupProductTestFixture(t)
	defer cleanup()
	router := setupProductTestRouter(tenantID)

	// Step 1: Set website_url and website_caption
	body := `{"website_url":"https://example.com","website_caption":"Visit Us"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/v1/tenant/products/%s", productID), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	// Step 2: Update ONLY product_name (should NOT clear website_url)
	body = `{"product_name":"Updated Name"}`
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("PUT", fmt.Sprintf("/api/v1/tenant/products/%s", productID), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	// Step 3: Verify via GET that website_url is still set
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", fmt.Sprintf("/api/v1/tenant/products/%s", productID), nil)
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	data := resp["data"].(map[string]interface{})
	assert.Equal(t, "Updated Name", data["product_name"])
	assert.Equal(t, "https://example.com", data["website_url"])
	assert.Equal(t, "Visit Us", data["website_caption"])
}

// TestSaveToUpdates_ProductEmptyBodyNoChange verifies that sending an empty
// body does not cause errors or change any fields.
func TestSaveToUpdates_ProductEmptyBodyNoChange(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}

	tenantID, productID, cleanup := setupProductTestFixture(t)
	defer cleanup()
	router := setupProductTestRouter(tenantID)

	// Get original state
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/tenant/products/%s", productID), nil)
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var originalResp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &originalResp))
	originalData := originalResp["data"].(map[string]interface{})
	originalName := originalData["product_name"]

	// Send empty update
	body := `{}`
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("PUT", fmt.Sprintf("/api/v1/tenant/products/%s", productID), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify nothing changed
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", fmt.Sprintf("/api/v1/tenant/products/%s", productID), nil)
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var afterResp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &afterResp))
	afterData := afterResp["data"].(map[string]interface{})
	assert.Equal(t, originalName, afterData["product_name"])
}

// ==================== SOFT DELETE / RESTORE TESTS ====================

func setupCountryTestFixture(t *testing.T) (string, func()) {
	t.Helper()
	code := fmt.Sprintf("Z%s", uuid.Must(uuid.NewV7()).String()[:1])
	country := &models.Country{
		Code:      code,
		Name:      "Test Country",
		PhoneCode: "+99",
	}
	require.NoError(t, testDB.Create(country).Error)
	cleanup := func() {
		testDB.Unscoped().Where("code = ?", code).Delete(&models.Country{})
	}
	return code, cleanup
}

func setupLocationMasterTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewLocationMasterHandler(testDB, testCfg)
	r.PUT("/api/v1/tenant/location-master/countries/:code", h.UpdateCountry)
	r.DELETE("/api/v1/tenant/location-master/countries/:code", h.DeleteCountry)
	r.POST("/api/v1/tenant/location-master/countries/:code/restore", h.RestoreCountry)
	return r
}

// TestSaveToUpdates_SoftDeleteOnlyDeletedAt verifies soft delete sets only deleted_at.
func TestSaveToUpdates_SoftDeleteOnlyDeletedAt(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}

	code, cleanup := setupCountryTestFixture(t)
	defer cleanup()
	router := setupLocationMasterTestRouter()

	// Delete
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/v1/tenant/location-master/countries/%s", code), nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify in DB: deleted_at is set, name/phone_code unchanged
	var country models.Country
	err := testDB.Where("code = ?", code).First(&country).Error
	require.NoError(t, err)
	assert.NotNil(t, country.DeletedAt, "deleted_at should be set")
	assert.Equal(t, "Test Country", country.Name, "name should be unchanged")
	assert.Equal(t, "+99", country.PhoneCode, "phone_code should be unchanged")
}

// TestSaveToUpdates_RestoreClearsDeletedAt verifies restore sets deleted_at to nil.
func TestSaveToUpdates_RestoreClearsDeletedAt(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}

	code, cleanup := setupCountryTestFixture(t)
	defer cleanup()
	router := setupLocationMasterTestRouter()

	// First delete
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/v1/tenant/location-master/countries/%s", code), nil)
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	// Restore
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", fmt.Sprintf("/api/v1/tenant/location-master/countries/%s/restore", code), nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify in DB: deleted_at is nil again
	var country models.Country
	err := testDB.Where("code = ?", code).First(&country).Error
	require.NoError(t, err)
	assert.Nil(t, country.DeletedAt, "deleted_at should be nil after restore")
	assert.Equal(t, "Test Country", country.Name, "name should be unchanged")
}

// ==================== COUNTRY PARTIAL UPDATE TEST ====================

// TestSaveToUpdates_CountryPartialUpdate verifies updating only name does not overwrite phone_code.
func TestSaveToUpdates_CountryPartialUpdate(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}

	code, cleanup := setupCountryTestFixture(t)
	defer cleanup()
	router := setupLocationMasterTestRouter()

	// Update only name
	body := `{"name":"New Country Name"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/v1/tenant/location-master/countries/%s", code), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Verify DB: name changed, phone_code unchanged
	var country models.Country
	err := testDB.Where("code = ?", code).First(&country).Error
	require.NoError(t, err)
	assert.Equal(t, "New Country Name", country.Name)
	assert.Equal(t, "+99", country.PhoneCode, "phone_code should be unchanged")
}

// ==================== COUNTERFEIT QR SYNC TEST (HOOK INLINING) ====================

func setupCounterfeitTestFixture(t *testing.T) (uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, func()) {
	t.Helper()

	tenantID, _ := uuid.NewV7()
	companyName := fmt.Sprintf("Counterfeit Test Co %s", tenantID.String())
	companyEmail := fmt.Sprintf("test-%s@counterfeit-test.com", tenantID.String())

	// Pre-cleanup: remove orphaned data from interrupted previous runs
	testDB.Unscoped().Where("company_name LIKE 'Counterfeit Test Co %'").Delete(&models.Tenant{})

	countryCode := "ID"
	tenant := &models.Tenant{
		ID:           tenantID,
		CompanyName:  companyName,
		CompanyEmail: companyEmail,
		CountryCode:  &countryCode,
	}
	require.NoError(t, testDB.Create(tenant).Error)

	productID, _ := uuid.NewV7()
	product := &models.Product{
		ID:          productID,
		TenantID:    tenantID,
		ProductName: "Counterfeit Test Product",
		ProductCode: "CF-001",
		Status:      models.ProductStatusActive,
	}
	require.NoError(t, testDB.Create(product).Error)

	batchID, _ := uuid.NewV7()
	batch := &models.QRBatch{
		ID:        batchID,
		TenantID:  tenantID,
		ProductID: productID,
		BatchName: "Test Batch",
		QRCount:   1,
	}
	require.NoError(t, testDB.Create(batch).Error)

	qrCodeID, _ := uuid.NewV7()
	qrCode := &models.QRCode{
		ID:                qrCodeID,
		BatchID:           batchID,
		QRUUID:            uuid.Must(uuid.NewV7()),
		QRCode:            fmt.Sprintf("QR-%s", qrCodeID.String()[:8]),
		CounterfeitStatus: models.CounterfeitStatusCounterfeit,
	}
	require.NoError(t, testDB.Create(qrCode).Error)

	now := time.Now().UTC()
	detection := &models.CounterfeitDetection{
		QRCodeID:               qrCodeID,
		TenantID:               tenantID,
		Status:                 models.CounterfeitDetectionStatusActive,
		DetectionReason:        "Exceeded scan threshold",
		TotalInteractionsCount: 5,
		FirstInteractionAt:     &now,
		LastInteractionAt:      &now,
	}
	require.NoError(t, testDB.Create(detection).Error)

	cleanup := func() {
		testDB.Unscoped().Where("qr_code_id = ?", qrCodeID).Delete(&models.CounterfeitDetection{})
		testDB.Unscoped().Where("id = ?", qrCodeID).Delete(&models.QRCode{})
		testDB.Unscoped().Where("id = ?", batchID).Delete(&models.QRBatch{})
		testDB.Unscoped().Where("id = ?", productID).Delete(&models.Product{})
		testDB.Unscoped().Where("id = ?", tenantID).Delete(&models.Tenant{})
	}

	return tenantID, qrCodeID, detection.ID, detection.ID, cleanup
}

// ==================== STATUS TRANSITION TEST ====================
func TestSaveToUpdates_ProductNullableFields(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}

	tenantID, productID, cleanup := setupProductTestFixture(t)
	defer cleanup()
	router := setupProductTestRouter(tenantID)

	// Step 1: Set warranty_months and max_warranty_registration_days
	body := `{"warranty_enabled":true,"warranty_months":12,"max_warranty_registration_days":30}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/v1/tenant/products/%s", productID), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	// Step 2: Clear max_warranty_registration_days by sending 0 (= unlimited/NULL)
	body = `{"max_warranty_registration_days":0}`
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("PUT", fmt.Sprintf("/api/v1/tenant/products/%s", productID), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	// Step 3: Verify in DB
	var product models.Product
	err := testDB.First(&product, "id = ? AND tenant_id = ?", productID, tenantID).Error
	require.NoError(t, err)
	assert.Nil(t, product.MaxWarrantyRegistrationDays, "max_warranty_registration_days should be nil (unlimited)")
	assert.Equal(t, 12, product.WarrantyMonths, "warranty_months should be unchanged")
}

// Suppress unused import warning
var _ = datatypes.JSON{}
