package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gamatritunggal/smartscan/backend/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupQRBatchTestRouter(t *testing.T, tenantID uuid.UUID) *gin.Engine {
	t.Helper()
	handler := NewQRBatchHandler(testDB, testCfg)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", uuid.Must(uuid.NewV7()).String())
		c.Set("tenant_id", tenantID.String())
		c.Next()
	})
	r.GET("/api/v1/tenant/qr-batches", handler.ListQRBatches)
	return r
}

func TestListQRBatches_GeofenceFields(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}
	uniq := uuid.New().String()[:8]

	// Setup: tenant, product, batch with geofence
	tenantID := uuid.Must(uuid.NewV7())
	countryCode := "ID"
	tenant := &models.Tenant{
		ID:           tenantID,
		CompanyName:  "ListGeoTest " + uniq,
		CompanyEmail: "listgeo-" + uniq + "@test.com",
		CountryCode:  &countryCode,
	}
	require.NoError(t, testDB.Create(tenant).Error)

	productID := uuid.Must(uuid.NewV7())
	product := &models.Product{
		ID:          productID,
		TenantID:    tenantID,
		ProductName: "ListGeo Product " + uniq,
		ProductCode: "LGP-" + uniq,
		Status:      models.ProductStatusActive,
	}
	require.NoError(t, testDB.Create(product).Error)

	lat, lng, radius := -6.2088, 106.8456, 50.0
	batchID := uuid.Must(uuid.NewV7())
	batch := &models.QRBatch{
		ID:                batchID,
		TenantID:          tenantID,
		ProductID:         productID,
		BatchName:         "ListGeo Batch " + uniq,
		QRCount:           10,
		GeofenceEnabled:   true,
		GeofenceLatitude:  &lat,
		GeofenceLongitude: &lng,
		GeofenceRadiusKm:  &radius,
		GeofenceLabel:     "Jakarta Zone",
	}
	require.NoError(t, testDB.Create(batch).Error)

	t.Cleanup(func() {
		testDB.Unscoped().Where("id = ?", batchID).Delete(&models.QRBatch{})
		testDB.Unscoped().Where("id = ?", productID).Delete(&models.Product{})
		testDB.Unscoped().Where("id = ?", tenantID).Delete(&models.Tenant{})
	})

	r := setupQRBatchTestRouter(t, tenantID)

	req, _ := http.NewRequest("GET", "/api/v1/tenant/qr-batches?product_id="+productID.String(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

	data := resp["data"].(map[string]interface{})
	batches := data["batches"].([]interface{})
	require.Len(t, batches, 1)

	b := batches[0].(map[string]interface{})
	assert.Equal(t, true, b["geofence_enabled"])
	assert.InDelta(t, -6.2088, b["geofence_latitude"].(float64), 0.001)
	assert.InDelta(t, 106.8456, b["geofence_longitude"].(float64), 0.001)
	assert.InDelta(t, 50.0, b["geofence_radius_km"].(float64), 0.1)
	assert.Equal(t, "Jakarta Zone", b["geofence_label"])
}

func TestListQRBatches_NoGeofence(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}
	uniq := uuid.New().String()[:8]

	// Setup: tenant, product, batch WITHOUT geofence
	tenantID := uuid.Must(uuid.NewV7())
	countryCode := "ID"
	tenant := &models.Tenant{
		ID:           tenantID,
		CompanyName:  "ListNoGeo " + uniq,
		CompanyEmail: "listnogeo-" + uniq + "@test.com",
		CountryCode:  &countryCode,
	}
	require.NoError(t, testDB.Create(tenant).Error)

	productID := uuid.Must(uuid.NewV7())
	product := &models.Product{
		ID:          productID,
		TenantID:    tenantID,
		ProductName: "NoGeo Product " + uniq,
		ProductCode: "NGP-" + uniq,
		Status:      models.ProductStatusActive,
	}
	require.NoError(t, testDB.Create(product).Error)

	batchID := uuid.Must(uuid.NewV7())
	batch := &models.QRBatch{
		ID:              batchID,
		TenantID:        tenantID,
		ProductID:       productID,
		BatchName:       "NoGeo Batch " + uniq,
		QRCount:         5,
		GeofenceEnabled: false,
	}
	require.NoError(t, testDB.Create(batch).Error)

	t.Cleanup(func() {
		testDB.Unscoped().Where("id = ?", batchID).Delete(&models.QRBatch{})
		testDB.Unscoped().Where("id = ?", productID).Delete(&models.Product{})
		testDB.Unscoped().Where("id = ?", tenantID).Delete(&models.Tenant{})
	})

	r := setupQRBatchTestRouter(t, tenantID)

	req, _ := http.NewRequest("GET", "/api/v1/tenant/qr-batches?product_id="+productID.String(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

	data := resp["data"].(map[string]interface{})
	batches := data["batches"].([]interface{})
	require.Len(t, batches, 1)

	b := batches[0].(map[string]interface{})
	assert.Equal(t, false, b["geofence_enabled"])
	// geofence_latitude should be nil/absent (omitempty on *float64)
	_, hasLat := b["geofence_latitude"]
	assert.False(t, hasLat, "geofence_latitude should be omitted when nil")
	_, hasLng := b["geofence_longitude"]
	assert.False(t, hasLng, "geofence_longitude should be omitted when nil")
	_, hasRadius := b["geofence_radius_km"]
	assert.False(t, hasRadius, "geofence_radius_km should be omitted when nil")
}
