package audit

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gamatritunggal/smartscan/backend/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var testDB *gorm.DB

func TestMain(m *testing.M) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "host=localhost port=5433 user=smartscan password=smartscan dbname=smartscan sslmode=disable TimeZone=UTC"
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		os.Exit(0)
	}

	testDB = db
	os.Exit(m.Run())
}

// createTestUser creates a real user in DB for FK constraints
func createTestUser(t *testing.T) uuid.UUID {
	t.Helper()
	userID := uuid.Must(uuid.NewV7())
	user := models.User{
		ID:           userID,
		Email:        "auditunit-" + uuid.Must(uuid.NewV7()).String()[:8] + "@test.com",
		PasswordHash: "$2a$10$test",
		UserType:     models.UserTypeTenantStaff,
		Status:       models.UserStatusActive,
	}
	require.NoError(t, testDB.Create(&user).Error)
	t.Cleanup(func() {
		testDB.Unscoped().Where("id = ?", userID).Delete(&models.User{})
	})
	return userID
}

// createTestTenant creates a real tenant in DB for FK constraints
func createTestTenant(t *testing.T) uuid.UUID {
	t.Helper()
	tenantID := uuid.Must(uuid.NewV7())
	tenant := models.Tenant{
		ID:                 tenantID,
		CompanyName:        "Audit Unit Test Corp",
		CompanyEmail:       "auditunit-" + uuid.Must(uuid.NewV7()).String()[:8] + "@test.com",
		Slug:               "auditunit-" + uuid.Must(uuid.NewV7()).String()[:8],
	}
	require.NoError(t, testDB.Create(&tenant).Error)
	t.Cleanup(func() {
		testDB.Unscoped().Where("id = ?", tenantID).Delete(&models.Tenant{})
	})
	return tenantID
}

// cleanupAuditLog removes test audit logs by entity_id
func cleanupAuditLog(db *gorm.DB, entityID uuid.UUID) {
	db.Unscoped().Where("entity_id = ?", entityID).Delete(&models.ActivityLog{})
}

// cleanupAuditLogByIP removes test audit logs by ip_address
func cleanupAuditLogByIP(db *gorm.DB, ip string) {
	db.Unscoped().Where("ip_address = ?", ip).Delete(&models.ActivityLog{})
}

// waitForAuditLog waits for goroutine to write to DB
func waitForAuditLog() {
	time.Sleep(200 * time.Millisecond)
}

func TestLog_BasicWrite(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}

	entityID := uuid.Must(uuid.NewV7())
	userID := createTestUser(t)
	tenantID := createTestTenant(t)
	defer cleanupAuditLog(testDB, entityID)

	Log(testDB, Entry{
		UserID:     &userID,
		TenantID:   &tenantID,
		Action:     models.ActionTypeCreate,
		EntityType: "test_entity",
		EntityID:   &entityID,
		OldValues:  map[string]interface{}{"field": "old"},
		NewValues:  map[string]interface{}{"field": "new"},
		IPAddress:  "192.168.1.100",
		UserAgent:  "TestAgent/1.0",
	})

	waitForAuditLog()

	var log models.ActivityLog
	err := testDB.Where("entity_id = ?", entityID).First(&log).Error
	require.NoError(t, err, "Audit log should be written to DB")

	assert.Equal(t, userID, *log.UserID)
	assert.Equal(t, tenantID, *log.TenantID)
	assert.Equal(t, models.ActionTypeCreate, log.ActionType)
	assert.Equal(t, "test_entity", log.EntityType)
	assert.Equal(t, entityID, *log.EntityID)
	assert.Equal(t, "192.168.1.100", log.IPAddress)
	assert.Equal(t, "TestAgent/1.0", log.UserAgent)
	assert.WithinDuration(t, time.Now(), log.CreatedAt, 5*time.Second)

	// Verify JSON values
	var oldVals map[string]interface{}
	require.NoError(t, json.Unmarshal(log.OldValues, &oldVals))
	assert.Equal(t, "old", oldVals["field"])

	var newVals map[string]interface{}
	require.NoError(t, json.Unmarshal(log.NewValues, &newVals))
	assert.Equal(t, "new", newVals["field"])
}

func TestLog_NilValues(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}

	entityID := uuid.Must(uuid.NewV7())
	userID := createTestUser(t)
	defer cleanupAuditLog(testDB, entityID)

	Log(testDB, Entry{
		UserID:     &userID,
		Action:     models.ActionTypeLogin,
		EntityType: "user",
		EntityID:   &entityID,
		OldValues:  nil,
		NewValues:  nil,
		IPAddress:  "10.0.0.1",
	})

	waitForAuditLog()

	var log models.ActivityLog
	err := testDB.Where("entity_id = ?", entityID).First(&log).Error
	require.NoError(t, err)

	assert.Nil(t, []byte(log.OldValues), "OldValues should be nil/empty for nil input")
	assert.Nil(t, []byte(log.NewValues), "NewValues should be nil/empty for nil input")
}

func TestLog_ComplexNestedValues(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}

	entityID := uuid.Must(uuid.NewV7())
	defer cleanupAuditLog(testDB, entityID)

	complexData := map[string]interface{}{
		"product_name": "Test Product",
		"certifications": []map[string]interface{}{
			{"cert_name": "ISO 9001", "display_order": 1},
			{"cert_name": "Halal", "display_order": 2},
		},
		"display_config": map[string]interface{}{
			"enabled": true,
			"limit":   100,
		},
	}

	Log(testDB, Entry{
		Action:     models.ActionTypeCreate,
		EntityType: "product",
		EntityID:   &entityID,
		NewValues:  complexData,
		IPAddress:  "10.0.0.1",
	})

	waitForAuditLog()

	var log models.ActivityLog
	err := testDB.Where("entity_id = ?", entityID).First(&log).Error
	require.NoError(t, err)

	var newVals map[string]interface{}
	require.NoError(t, json.Unmarshal(log.NewValues, &newVals))
	assert.Equal(t, "Test Product", newVals["product_name"])

	certs, ok := newVals["certifications"].([]interface{})
	require.True(t, ok, "certifications should be an array")
	assert.Len(t, certs, 2)

	displayConfig, ok := newVals["display_config"].(map[string]interface{})
	require.True(t, ok, "display_config should be a map")
	assert.Equal(t, true, displayConfig["enabled"])
}

func TestLog_NilUserAndTenant(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}

	entityID := uuid.Must(uuid.NewV7())
	defer cleanupAuditLog(testDB, entityID)

	Log(testDB, Entry{
		UserID:     nil,
		TenantID:   nil,
		Action:     models.ActionTypeLogin,
		EntityType: "user",
		EntityID:   &entityID,
		IPAddress:  "10.0.0.1",
	})

	waitForAuditLog()

	var log models.ActivityLog
	err := testDB.Where("entity_id = ?", entityID).First(&log).Error
	require.NoError(t, err)

	assert.Nil(t, log.UserID, "UserID should be NULL")
	assert.Nil(t, log.TenantID, "TenantID should be NULL")
}

func TestLog_NilEntityID(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}

	// Use unique IP to find this record since entity_id is nil
	uniqueIP := "198.51.100.1"
	defer cleanupAuditLogByIP(testDB, uniqueIP)

	Log(testDB, Entry{
		UserID:     nil, // nil to avoid FK constraint
		Action:     models.ActionTypeLogout,
		EntityType: "user",
		EntityID:   nil,
		IPAddress:  uniqueIP,
	})

	waitForAuditLog()

	var log models.ActivityLog
	err := testDB.Where("ip_address = ? AND action_type = ?", uniqueIP, models.ActionTypeLogout).First(&log).Error
	require.NoError(t, err)

	assert.Nil(t, log.EntityID, "EntityID should be NULL")
	assert.Equal(t, models.ActionTypeLogout, log.ActionType)
}

func TestLog_AllActionTypes(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}

	actionTypes := []models.ActionType{
		models.ActionTypeLogin,
		models.ActionTypeLogout,
		models.ActionTypeCreate,
		models.ActionTypeUpdate,
		models.ActionTypeDelete,
		models.ActionTypeExport,
		models.ActionTypePasswordReset,
	}

	uniqueIP := "198.51.100.2"
	defer cleanupAuditLogByIP(testDB, uniqueIP)

	for _, action := range actionTypes {
		entityID := uuid.Must(uuid.NewV7())
		Log(testDB, Entry{
			Action:     action,
			EntityType: "test_action_type",
			EntityID:   &entityID,
			IPAddress:  uniqueIP,
		})
	}

	waitForAuditLog()

	var logs []models.ActivityLog
	err := testDB.Where("ip_address = ? AND entity_type = ?", uniqueIP, "test_action_type").Find(&logs).Error
	require.NoError(t, err)
	assert.Len(t, logs, 7, "All 7 action types should be written")

	foundActions := make(map[models.ActionType]bool)
	for _, log := range logs {
		foundActions[log.ActionType] = true
	}
	for _, action := range actionTypes {
		assert.True(t, foundActions[action], "Action type %s should be found", action)
	}
}

func TestLogFromContext_ExtractsContext(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}

	entityID := uuid.Must(uuid.NewV7())
	userID := createTestUser(t)
	tenantID := createTestTenant(t)
	defer cleanupAuditLog(testDB, entityID)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.SetTrustedProxies(nil) // Trust all proxies for test
	r.POST("/test", func(ctx *gin.Context) {
		ctx.Set("user_id", userID.String())
		ctx.Set("tenant_id", tenantID.String())
		LogFromContext(ctx, testDB, models.ActionTypeLogin, "user", &entityID, nil, nil)
		ctx.JSON(200, gin.H{"ok": true})
	})

	req, _ := http.NewRequest("POST", "/test", nil)
	req.Header.Set("User-Agent", "TestBrowser/2.0")
	req.RemoteAddr = "203.0.113.50:12345"
	r.ServeHTTP(w, req)

	waitForAuditLog()

	var log models.ActivityLog
	err := testDB.Where("entity_id = ?", entityID).First(&log).Error
	require.NoError(t, err)

	assert.Equal(t, userID, *log.UserID, "UserID should be extracted from context")
	assert.Equal(t, tenantID, *log.TenantID, "TenantID should be extracted from context")
	assert.Equal(t, "TestBrowser/2.0", log.UserAgent, "UserAgent should be extracted from request")
	assert.NotEmpty(t, log.IPAddress, "IPAddress should be captured")
	assert.Equal(t, models.ActionTypeLogin, log.ActionType)
}

func TestLogFromContext_MissingUserID(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}

	entityID := uuid.Must(uuid.NewV7())
	tenantID := createTestTenant(t)
	defer cleanupAuditLog(testDB, entityID)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.SetTrustedProxies(nil)
	r.POST("/test", func(ctx *gin.Context) {
		ctx.Set("tenant_id", tenantID.String())
		LogFromContext(ctx, testDB, models.ActionTypeCreate, "test", &entityID, nil, nil)
		ctx.JSON(200, gin.H{"ok": true})
	})

	req, _ := http.NewRequest("POST", "/test", nil)
	req.RemoteAddr = "203.0.113.51:12345"
	r.ServeHTTP(w, req)

	waitForAuditLog()

	var log models.ActivityLog
	err := testDB.Where("entity_id = ?", entityID).First(&log).Error
	require.NoError(t, err)

	assert.Nil(t, log.UserID, "UserID should be NULL when not in context")
	assert.Equal(t, tenantID, *log.TenantID, "TenantID should still be set")
}

func TestLogFromContext_MissingTenantID(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}

	entityID := uuid.Must(uuid.NewV7())
	userID := createTestUser(t)
	defer cleanupAuditLog(testDB, entityID)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.SetTrustedProxies(nil)
	r.POST("/test", func(ctx *gin.Context) {
		ctx.Set("user_id", userID.String())
		LogFromContext(ctx, testDB, models.ActionTypeLogout, "user", &entityID, nil, nil)
		ctx.JSON(200, gin.H{"ok": true})
	})

	req, _ := http.NewRequest("POST", "/test", nil)
	req.RemoteAddr = "203.0.113.52:12345"
	r.ServeHTTP(w, req)

	waitForAuditLog()

	var log models.ActivityLog
	err := testDB.Where("entity_id = ?", entityID).First(&log).Error
	require.NoError(t, err)

	assert.Equal(t, userID, *log.UserID, "UserID should still be set")
	assert.Nil(t, log.TenantID, "TenantID should be NULL when not in context")
}
