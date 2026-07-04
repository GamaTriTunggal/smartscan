package testutil

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gamatritunggal/smartscan/backend/internal/config"
	"github.com/gamatritunggal/smartscan/backend/internal/models"
	"github.com/gamatritunggal/smartscan/backend/internal/utils"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// TestConfig provides a test configuration
func TestConfig() *config.Config {
	return &config.Config{
		AppEnv:      "test",
		FrontendURL: "http://localhost:3000",
		JWT: config.JWTConfig{
			Secret:          "test-secret-key-for-jwt-signing-minimum-32-chars",
			ExpirationHours: 24,
			RefreshHours:    168,
		},
	}
}

// SetupTestDB creates a connection to the test database
func SetupTestDB(t *testing.T) *gorm.DB {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "host=postgres port=5432 user=smartscan password=smartscan dbname=smartscan_test sslmode=disable TimeZone=UTC"
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Skip("Test database not available")
		return nil
	}

	return db
}

// RequireTestDB returns test DB or fails the test
func RequireTestDB(t *testing.T) *gorm.DB {
	db := SetupTestDB(t)
	require.NotNil(t, db, "Test database is required for this test")
	return db
}

// SetupTestRouter creates a Gin router in test mode
func SetupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

// CleanupUser removes a user and related records by email
func CleanupUser(db *gorm.DB, email string) {
	var user models.User
	if err := db.Unscoped().Where("email = ?", email).First(&user).Error; err == nil {
		db.Unscoped().Where("user_id = ?", user.ID).Delete(&models.TenantStaff{})
		db.Unscoped().Delete(&user)
	}
}

// CleanupTenant removes a tenant and related records by company email
func CleanupTenant(db *gorm.DB, companyEmail string) {
	var tenant models.Tenant
	if err := db.Unscoped().Where("company_email = ?", companyEmail).First(&tenant).Error; err == nil {
		db.Unscoped().Where("tenant_id = ?", tenant.ID).Delete(&models.TenantSettings{})
		db.Unscoped().Where("tenant_id = ?", tenant.ID).Delete(&models.TenantStaff{})
		db.Unscoped().Delete(&tenant)
	}
}

// CleanupQRBatch removes a QR batch and related QR codes
func CleanupQRBatch(db *gorm.DB, batchID uuid.UUID) {
	db.Unscoped().Where("batch_id = ?", batchID).Delete(&models.QRCode{})
	db.Unscoped().Where("id = ?", batchID).Delete(&models.QRBatch{})
}

// CreateTenantUser creates a test tenant user with tenant
func CreateTenantUser(t *testing.T, db *gorm.DB, email, password string, role models.TenantStaffRole) (*models.User, *models.Tenant) {
	hashedPassword, err := utils.HashPassword(password)
	require.NoError(t, err)

	countryCode := "ID"
	tenant := &models.Tenant{
		ID:           uuid.Must(uuid.NewV7()),
		CompanyName:  "Test Company",
		CompanyEmail: "company-" + email,
		CountryCode:  &countryCode,
	}
	require.NoError(t, db.Create(tenant).Error)

	user := &models.User{
		ID:           uuid.Must(uuid.NewV7()),
		Email:        email,
		PasswordHash: hashedPassword,
		UserType:     models.UserTypeTenantStaff,
		Status:       models.UserStatusActive,
	}
	require.NoError(t, db.Create(user).Error)

	staff := &models.TenantStaff{
		ID:       uuid.Must(uuid.NewV7()),
		UserID:   user.ID,
		TenantID: tenant.ID,
		FullName: "Test Tenant User",
		Role:     role,
	}
	require.NoError(t, db.Create(staff).Error)

	return user, tenant
}

// GenerateTestToken creates a valid JWT token for testing
func GenerateTestToken(t *testing.T, cfg *config.Config, user *models.User, role string, tenantID *uuid.UUID) string {
	tokenPair, err := utils.GenerateTokenPair(
		cfg.JWT.Secret,
		user.ID,
		user.Email,
		string(user.UserType),
		role,
		tenantID,
		cfg.JWT.ExpirationHours,
		cfg.JWT.RefreshHours,
		false,
	)
	require.NoError(t, err, "Failed to generate token")
	return tokenPair.AccessToken
}

// HTTPTestCase represents a test case for HTTP handlers
type HTTPTestCase struct {
	Name           string
	Method         string
	Path           string
	Body           interface{}
	Headers        map[string]string
	ExpectedStatus int
	CheckResponse  func(t *testing.T, body []byte)
}

// RunHTTPTest executes an HTTP test case
func RunHTTPTest(t *testing.T, router *gin.Engine, tc HTTPTestCase) *httptest.ResponseRecorder {
	var bodyReader *bytes.Buffer
	if tc.Body != nil {
		bodyBytes, err := json.Marshal(tc.Body)
		require.NoError(t, err)
		bodyReader = bytes.NewBuffer(bodyBytes)
	} else {
		bodyReader = bytes.NewBuffer(nil)
	}

	req, err := http.NewRequest(tc.Method, tc.Path, bodyReader)
	require.NoError(t, err)

	req.Header.Set("Content-Type", "application/json")
	for k, v := range tc.Headers {
		req.Header.Set(k, v)
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if tc.ExpectedStatus != 0 {
		require.Equal(t, tc.ExpectedStatus, w.Code, "Response body: %s", w.Body.String())
	}

	if tc.CheckResponse != nil {
		tc.CheckResponse(t, w.Body.Bytes())
	}

	return w
}

// ParseAPIResponse parses the standard API response
func ParseAPIResponse(t *testing.T, body []byte) utils.APIResponse {
	var response utils.APIResponse
	require.NoError(t, json.Unmarshal(body, &response))
	return response
}

// RequireSuccess checks that the API response indicates success
func RequireSuccess(t *testing.T, body []byte) {
	response := ParseAPIResponse(t, body)
	require.True(t, response.Success, "Expected success=true, got message: %s", response.Message)
}

// RequireFailure checks that the API response indicates failure
func RequireFailure(t *testing.T, body []byte) {
	response := ParseAPIResponse(t, body)
	require.False(t, response.Success, "Expected success=false")
}

// WithAuthHeader returns headers with Authorization token
func WithAuthHeader(token string) map[string]string {
	return map[string]string{
		"Authorization": "Bearer " + token,
	}
}
