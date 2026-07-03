package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gamatritunggal/smartscan/backend/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupProductTestFixture creates a tenant and product for testing, returns cleanup func
func setupProductTestFixture(t *testing.T) (uuid.UUID, uuid.UUID, func()) {
	t.Helper()

	tenantID, _ := uuid.NewV7()
	countryCode := "ID"
	tenant := &models.Tenant{
		ID:          tenantID,
		CompanyName: "Test Template Co",
		CompanyEmail: fmt.Sprintf("test-%s@template-test.com", tenantID.String()[:8]),
		CountryCode: &countryCode,
	}
	require.NoError(t, testDB.Create(tenant).Error)

	productID, _ := uuid.NewV7()
	product := &models.Product{
		ID:          productID,
		TenantID:    tenantID,
		ProductName: "Test Product",
		ProductCode: "TEST-001",
		Status:      models.ProductStatusActive,
	}
	require.NoError(t, testDB.Create(product).Error)

	cleanup := func() {
		testDB.Unscoped().Where("id = ?", productID).Delete(&models.Product{})
		testDB.Unscoped().Where("id = ?", tenantID).Delete(&models.Tenant{})
	}

	return tenantID, productID, cleanup
}

func setupProductTestRouter(tenantID uuid.UUID) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	h := NewProductHandler(testDB, testCfg)

	// Simulate auth middleware by injecting tenant_id
	r.Use(func(c *gin.Context) {
		c.Set("tenant_id", tenantID.String())
		c.Next()
	})

	r.PUT("/api/v1/tenant/products/:id", h.UpdateProduct)
	r.GET("/api/v1/tenant/products/:id", h.GetProduct)
	return r
}

func TestUpdateProduct_TemplateOverrides_ValidObject(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}

	tenantID, productID, cleanup := setupProductTestFixture(t)
	defer cleanup()
	router := setupProductTestRouter(tenantID)

	body := `{"template_overrides":{"header":{"badge_text":"Custom Badge","badge_bg_color":"#ff6600"},"styling":{"text_color":"#333"}}}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/v1/tenant/products/%s", productID), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.True(t, resp["success"].(bool))

	data := resp["data"].(map[string]interface{})
	overrides := data["template_overrides"].(map[string]interface{})
	header := overrides["header"].(map[string]interface{})
	assert.Equal(t, "Custom Badge", header["badge_text"])
	assert.Equal(t, "#ff6600", header["badge_bg_color"])
}

func TestUpdateProduct_TemplateOverrides_ClearWithNull(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}

	tenantID, productID, cleanup := setupProductTestFixture(t)
	defer cleanup()
	router := setupProductTestRouter(tenantID)

	// First set overrides
	body := `{"template_overrides":{"header":{"badge_text":"Temp"}}}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/v1/tenant/products/%s", productID), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	// Now clear with null
	body = `{"template_overrides":null}`
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("PUT", fmt.Sprintf("/api/v1/tenant/products/%s", productID), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	data := resp["data"].(map[string]interface{})
	// template_overrides should be absent (omitempty) or null
	_, exists := data["template_overrides"]
	assert.False(t, exists, "template_overrides should be absent after clearing")
}

func TestUpdateProduct_TemplateOverrides_ClearWithEmpty(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}

	tenantID, productID, cleanup := setupProductTestFixture(t)
	defer cleanup()
	router := setupProductTestRouter(tenantID)

	// First set overrides
	body := `{"template_overrides":{"header":{"badge_text":"Temp"}}}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/v1/tenant/products/%s", productID), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	// Clear with empty object
	body = `{"template_overrides":{}}`
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("PUT", fmt.Sprintf("/api/v1/tenant/products/%s", productID), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	data := resp["data"].(map[string]interface{})
	_, exists := data["template_overrides"]
	assert.False(t, exists, "template_overrides should be absent after clearing with {}")
}

func TestUpdateProduct_TemplateOverrides_InvalidArray(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}

	tenantID, productID, cleanup := setupProductTestFixture(t)
	defer cleanup()
	router := setupProductTestRouter(tenantID)

	body := `{"template_overrides":[1,2,3]}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/v1/tenant/products/%s", productID), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.False(t, resp["success"].(bool))
	assert.Contains(t, resp["message"].(string), "valid JSON object")
}

func TestUpdateProduct_TemplateOverrides_InvalidString(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}

	tenantID, productID, cleanup := setupProductTestFixture(t)
	defer cleanup()
	router := setupProductTestRouter(tenantID)

	body := `{"template_overrides":"just a string"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/v1/tenant/products/%s", productID), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Contains(t, resp["message"].(string), "valid JSON object")
}

func TestUpdateProduct_TemplateOverrides_TooLarge(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}

	tenantID, productID, cleanup := setupProductTestFixture(t)
	defer cleanup()
	router := setupProductTestRouter(tenantID)

	// Create a JSON object > 10KB
	largeValue := strings.Repeat("x", 11*1024)
	body := fmt.Sprintf(`{"template_overrides":{"large_key":"%s"}}`, largeValue)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/v1/tenant/products/%s", productID), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Contains(t, resp["message"].(string), "exceeds maximum size")
}

func TestUpdateProduct_TemplateOverrides_Persists(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}

	tenantID, productID, cleanup := setupProductTestFixture(t)
	defer cleanup()
	router := setupProductTestRouter(tenantID)

	// Set overrides
	body := `{"template_overrides":{"warranty_button":{"text":"Register Now","bg_color":"#e11d48"}}}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/v1/tenant/products/%s", productID), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	// GET to verify persistence
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", fmt.Sprintf("/api/v1/tenant/products/%s", productID), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	data := resp["data"].(map[string]interface{})
	overrides := data["template_overrides"].(map[string]interface{})
	warranty := overrides["warranty_button"].(map[string]interface{})
	assert.Equal(t, "Register Now", warranty["text"])
	assert.Equal(t, "#e11d48", warranty["bg_color"])
}

// --- Warranty Template Overrides Tests ---

func TestUpdateProduct_WarrantyTemplateOverrides_ValidObject(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}

	tenantID, productID, cleanup := setupProductTestFixture(t)
	defer cleanup()
	router := setupProductTestRouter(tenantID)

	body := `{"warranty_template_overrides":{"styling":{"header_bg_color":"#059669","accent_color":"#10b981"},"submit_button":{"text":"Register Warranty","bg_color":"#059669"}}}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/v1/tenant/products/%s", productID), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.True(t, resp["success"].(bool))

	data := resp["data"].(map[string]interface{})
	overrides := data["warranty_template_overrides"].(map[string]interface{})
	styling := overrides["styling"].(map[string]interface{})
	assert.Equal(t, "#059669", styling["header_bg_color"])
	assert.Equal(t, "#10b981", styling["accent_color"])
	submitBtn := overrides["submit_button"].(map[string]interface{})
	assert.Equal(t, "Register Warranty", submitBtn["text"])
}

func TestUpdateProduct_WarrantyTemplateOverrides_ClearWithNull(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}

	tenantID, productID, cleanup := setupProductTestFixture(t)
	defer cleanup()
	router := setupProductTestRouter(tenantID)

	// First set overrides
	body := `{"warranty_template_overrides":{"styling":{"header_bg_color":"#059669"}}}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/v1/tenant/products/%s", productID), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	// Clear with null
	body = `{"warranty_template_overrides":null}`
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("PUT", fmt.Sprintf("/api/v1/tenant/products/%s", productID), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	data := resp["data"].(map[string]interface{})
	_, exists := data["warranty_template_overrides"]
	assert.False(t, exists, "warranty_template_overrides should be absent after clearing")
}

func TestUpdateProduct_WarrantyTemplateOverrides_ClearWithEmpty(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}

	tenantID, productID, cleanup := setupProductTestFixture(t)
	defer cleanup()
	router := setupProductTestRouter(tenantID)

	// First set overrides
	body := `{"warranty_template_overrides":{"styling":{"header_bg_color":"#059669"}}}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/v1/tenant/products/%s", productID), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	// Clear with empty object
	body = `{"warranty_template_overrides":{}}`
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("PUT", fmt.Sprintf("/api/v1/tenant/products/%s", productID), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	data := resp["data"].(map[string]interface{})
	_, exists := data["warranty_template_overrides"]
	assert.False(t, exists, "warranty_template_overrides should be absent after clearing with {}")
}

func TestUpdateProduct_WarrantyTemplateOverrides_RejectArray(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}

	tenantID, productID, cleanup := setupProductTestFixture(t)
	defer cleanup()
	router := setupProductTestRouter(tenantID)

	body := `{"warranty_template_overrides":[1,2,3]}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/v1/tenant/products/%s", productID), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Contains(t, resp["message"].(string), "valid JSON object")
}

func TestUpdateProduct_WarrantyTemplateOverrides_SizeLimit(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}

	tenantID, productID, cleanup := setupProductTestFixture(t)
	defer cleanup()
	router := setupProductTestRouter(tenantID)

	// Create a JSON object > 10KB
	largeValue := strings.Repeat("x", 11*1024)
	body := fmt.Sprintf(`{"warranty_template_overrides":{"large_key":"%s"}}`, largeValue)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/v1/tenant/products/%s", productID), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Contains(t, resp["message"].(string), "exceeds maximum size")
}
