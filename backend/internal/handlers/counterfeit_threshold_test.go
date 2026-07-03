package handlers

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/gamatritunggal/smartscan/backend/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func thresholdIntPtr(n int) *int {
	return &n
}

func setupThresholdTestData(t *testing.T) (product *models.Product, batch *models.QRBatch, qrCode *models.QRCode, cleanup func()) {
	t.Helper()

	tenantID := uuid.Must(uuid.NewV7())
	tenant := &models.Tenant{
		ID:          tenantID,
		CompanyName: "Threshold Test Tenant " + tenantID.String()[:8],
	}
	require.NoError(t, testDB.Create(tenant).Error)

	product = &models.Product{
		TenantID:    tenantID,
		ProductName: "Test Product",
	}
	require.NoError(t, testDB.Create(product).Error)

	batch = &models.QRBatch{
		TenantID:  tenantID,
		ProductID: product.ID,
		BatchName: "Test Batch",
		QRCount:   1,
	}
	require.NoError(t, testDB.Create(batch).Error)

	qrCode = &models.QRCode{
		BatchID: batch.ID,
		QRUUID:  uuid.Must(uuid.NewV7()),
		QRCode:  "TEST-" + uuid.Must(uuid.NewV7()).String()[:8],
	}
	require.NoError(t, testDB.Create(qrCode).Error)

	// Preload relations
	qrCode.Batch = batch
	qrCode.Batch.Product = product

	cleanup = func() {
		testDB.Unscoped().Delete(qrCode)
		testDB.Unscoped().Delete(batch)
		testDB.Unscoped().Delete(product)
		testDB.Unscoped().Delete(tenant)
		testDB.Where("tenant_id = ? AND setting_key = ?", tenantID, "counterfeit_thresholds").Delete(&models.TenantSettings{})
	}

	return product, batch, qrCode, cleanup
}

func TestResolveCounterfeitThreshold_Default(t *testing.T) {
	if testDB == nil {
		t.Skip("Test database not available")
	}

	_, _, qrCode, cleanup := setupThresholdTestData(t)
	defer cleanup()

	// No overrides at any level - should return default (3)
	result := ResolveCounterfeitThreshold(testDB, qrCode)
	assert.Equal(t, 3, result)
}

func TestResolveCounterfeitThreshold_TenantSetting(t *testing.T) {
	if testDB == nil {
		t.Skip("Test database not available")
	}

	_, _, qrCode, cleanup := setupThresholdTestData(t)
	defer cleanup()

	// Set tenant-level threshold
	settingVal, _ := json.Marshal(map[string]int{"end_user_scan_max": 5})
	setting := models.TenantSettings{
		TenantID:     qrCode.Batch.TenantID,
		SettingKey:   "counterfeit_thresholds",
		SettingValue: settingVal,
	}
	require.NoError(t, testDB.Create(&setting).Error)

	result := ResolveCounterfeitThreshold(testDB, qrCode)
	assert.Equal(t, 5, result)
}

func TestResolveCounterfeitThreshold_ProductOverride(t *testing.T) {
	if testDB == nil {
		t.Skip("Test database not available")
	}

	product, _, qrCode, cleanup := setupThresholdTestData(t)
	defer cleanup()

	// Set tenant-level to 5, product-level to 10
	settingVal, _ := json.Marshal(map[string]int{"end_user_scan_max": 5})
	setting := models.TenantSettings{
		TenantID:     qrCode.Batch.TenantID,
		SettingKey:   "counterfeit_thresholds",
		SettingValue: settingVal,
	}
	require.NoError(t, testDB.Create(&setting).Error)

	product.CounterfeitScanMax = thresholdIntPtr(10)
	require.NoError(t, testDB.Save(product).Error)
	qrCode.Batch.Product = product

	result := ResolveCounterfeitThreshold(testDB, qrCode)
	assert.Equal(t, 10, result)
}

func TestResolveCounterfeitThreshold_BatchOverride(t *testing.T) {
	if testDB == nil {
		t.Skip("Test database not available")
	}

	product, batch, qrCode, cleanup := setupThresholdTestData(t)
	defer cleanup()

	// Product = 10, Batch = 20
	product.CounterfeitScanMax = thresholdIntPtr(10)
	require.NoError(t, testDB.Save(product).Error)
	qrCode.Batch.Product = product

	batch.CounterfeitScanMax = thresholdIntPtr(20)
	require.NoError(t, testDB.Save(batch).Error)
	qrCode.Batch = batch
	qrCode.Batch.Product = product

	result := ResolveCounterfeitThreshold(testDB, qrCode)
	assert.Equal(t, 20, result)
}

func TestResolveCounterfeitThreshold_QROverride(t *testing.T) {
	if testDB == nil {
		t.Skip("Test database not available")
	}

	product, batch, qrCode, cleanup := setupThresholdTestData(t)
	defer cleanup()

	// Product = 10, Batch = 20, QR = 50
	product.CounterfeitScanMax = thresholdIntPtr(10)
	require.NoError(t, testDB.Save(product).Error)

	batch.CounterfeitScanMax = thresholdIntPtr(20)
	require.NoError(t, testDB.Save(batch).Error)

	qrCode.CounterfeitScanMax = thresholdIntPtr(50)
	require.NoError(t, testDB.Save(qrCode).Error)

	qrCode.Batch = batch
	qrCode.Batch.Product = product

	result := ResolveCounterfeitThreshold(testDB, qrCode)
	assert.Equal(t, 50, result)
}

func TestResolveCounterfeitThreshold_ZeroDisabled(t *testing.T) {
	if testDB == nil {
		t.Skip("Test database not available")
	}

	_, _, qrCode, cleanup := setupThresholdTestData(t)
	defer cleanup()

	// Tenant setting = 0 should disable counterfeit check
	settingVal, _ := json.Marshal(map[string]int{"end_user_scan_max": 0})
	setting := models.TenantSettings{
		TenantID:     qrCode.Batch.TenantID,
		SettingKey:   "counterfeit_thresholds",
		SettingValue: settingVal,
	}
	require.NoError(t, testDB.Create(&setting).Error)

	result := ResolveCounterfeitThreshold(testDB, qrCode)
	assert.Equal(t, 0, result)
}

func TestResolveCounterfeitThreshold_NilCascade(t *testing.T) {
	if testDB == nil {
		t.Skip("Test database not available")
	}

	product, _, qrCode, cleanup := setupThresholdTestData(t)
	defer cleanup()

	// Only product override set, QR and Batch are nil - should cascade to product
	product.CounterfeitScanMax = thresholdIntPtr(15)
	require.NoError(t, testDB.Save(product).Error)
	qrCode.Batch.Product = product

	// QR and Batch have nil CounterfeitScanMax
	assert.Nil(t, qrCode.CounterfeitScanMax)
	assert.Nil(t, qrCode.Batch.CounterfeitScanMax)

	result := ResolveCounterfeitThreshold(testDB, qrCode)
	assert.Equal(t, 15, result)
}

func TestResolveCounterfeitThreshold_ProductSetBatchNil(t *testing.T) {
	if testDB == nil {
		t.Skip("Test database not available")
	}

	product, batch, qrCode, cleanup := setupThresholdTestData(t)
	defer cleanup()

	// Product has override, batch explicitly nil
	product.CounterfeitScanMax = thresholdIntPtr(12)
	require.NoError(t, testDB.Save(product).Error)

	// Batch has no override (nil)
	assert.Nil(t, batch.CounterfeitScanMax)

	qrCode.Batch = batch
	qrCode.Batch.Product = product

	result := ResolveCounterfeitThreshold(testDB, qrCode)
	assert.Equal(t, 12, result)
}

func TestResolveCounterfeitThreshold_TenantSettingMissingKey(t *testing.T) {
	if testDB == nil {
		t.Skip("Test database not available")
	}

	_, _, qrCode, cleanup := setupThresholdTestData(t)
	defer cleanup()

	// Insert valid JSON but without the expected key
	setting := models.TenantSettings{
		TenantID:     qrCode.Batch.TenantID,
		SettingKey:   "counterfeit_thresholds",
		SettingValue: []byte(`{"velocity_check_enabled": true}`),
	}
	require.NoError(t, testDB.Create(&setting).Error)

	// Should fallback to default (3) when end_user_scan_max key is missing
	result := ResolveCounterfeitThreshold(testDB, qrCode)
	assert.Equal(t, 3, result)
}
