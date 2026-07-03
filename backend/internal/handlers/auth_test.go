package handlers

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
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var testDB *gorm.DB
var testCfg *config.Config

func TestMain(m *testing.M) {
	// Setup test database
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "host=localhost port=5433 user=smartscan password=smartscan dbname=smartscan sslmode=disable TimeZone=UTC"
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		// Skip tests if DB not available
		os.Exit(0)
	}

	testDB = db
	testCfg = &config.Config{
		AppEnv: "test",
		JWT: config.JWTConfig{
			Secret:          "test-secret-key-for-jwt-signing-minimum-32-chars",
			ExpirationHours: 24,
			RefreshHours:    168,
		},
	}

	// Run tests
	code := m.Run()

	os.Exit(code)
}

func setupTestRouter(h *AuthHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/v1/auth/login", h.Login)
	r.POST("/api/v1/auth/refresh", h.RefreshToken)
	return r
}

func cleanupTestUser(db *gorm.DB, email string) {
	var user models.User
	if err := db.Unscoped().Where("email = ?", email).First(&user).Error; err == nil {
		db.Unscoped().Where("user_id = ?", user.ID).Delete(&models.TenantStaff{})
		db.Unscoped().Where("user_id = ?", user.ID).Delete(&models.TenantStaff{})
		db.Unscoped().Delete(&user)
	}
}

func cleanupTestTenant(db *gorm.DB, companyEmail string) {
	var tenant models.Tenant
	if err := db.Unscoped().Where("company_email = ?", companyEmail).First(&tenant).Error; err == nil {
		db.Unscoped().Where("tenant_id = ?", tenant.ID).Delete(&models.TenantSettings{})
		db.Unscoped().Where("tenant_id = ?", tenant.ID).Delete(&models.TenantStaff{})
		db.Unscoped().Where("tenant_id = ?", tenant.ID).Delete(&models.PageTemplate{})
		db.Unscoped().Delete(&tenant)
	}
}

func createTestAdminUser(db *gorm.DB, email, password string) (*models.User, error) {
	hashedPassword, err := utils.HashPassword(password)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		ID:           uuid.Must(uuid.NewV7()),
		Email:        email,
		PasswordHash: hashedPassword,
		UserType:     models.UserTypeTenantStaff,
		Status:       models.UserStatusActive,
	}

	if err := db.Create(user).Error; err != nil {
		return nil, err
	}

	countryCode := "ID"
	tenant := &models.Tenant{
		ID:           uuid.Must(uuid.NewV7()),
		CompanyName:  "AuthTest " + email,
		CompanyEmail: "authtest-" + email,
		CountryCode:  &countryCode,
	}
	if err := db.Create(tenant).Error; err != nil {
		return nil, err
	}
	staff := &models.TenantStaff{
		ID:       uuid.Must(uuid.NewV7()),
		UserID:   user.ID,
		TenantID: tenant.ID,
		FullName: "Test User",
		Role:     models.TenantStaffRoleAdmin,
	}

	if err := db.Create(staff).Error; err != nil {
		return nil, err
	}

	return user, nil
}

func TestLogin_Success(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}

	// Per-invocation unique email so a killed prior run can't leak the row and trip the
	testEmail := "testlogin-" + uuid.New().String()[:8] + "@example.com"
	cleanupTestUser(testDB, testEmail)
	defer cleanupTestUser(testDB, testEmail)

	handler := NewAuthHandler(testDB, testCfg)
	router := setupTestRouter(handler)

	// Create test user
	_, err := createTestAdminUser(testDB, testEmail, "password123")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Test login
	loginReq := LoginRequest{
		Email:    testEmail,
		Password: "password123",
	}
	body, _ := json.Marshal(loginReq)

	req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var response utils.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if !response.Success {
		t.Errorf("Expected success=true, got false. Message: %s", response.Message)
	}
}

func TestLogin_InvalidEmail(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}

	handler := NewAuthHandler(testDB, testCfg)
	router := setupTestRouter(handler)

	loginReq := LoginRequest{
		Email:    "nonexistent@example.com",
		Password: "password123",
	}
	body, _ := json.Marshal(loginReq)

	req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}

	var response utils.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response.Success {
		t.Errorf("Expected success=false for invalid credentials")
	}
}

func TestLogin_InvalidPassword(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}

	testEmail := "testbadpw-" + uuid.New().String()[:8] + "@example.com"
	cleanupTestUser(testDB, testEmail)
	defer cleanupTestUser(testDB, testEmail)

	handler := NewAuthHandler(testDB, testCfg)
	router := setupTestRouter(handler)

	// Create test user
	_, err := createTestAdminUser(testDB, testEmail, "password123")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	loginReq := LoginRequest{
		Email:    testEmail,
		Password: "wrongpassword",
	}
	body, _ := json.Marshal(loginReq)

	req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestLogin_InvalidRequestFormat(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}

	handler := NewAuthHandler(testDB, testCfg)
	router := setupTestRouter(handler)

	// Send invalid JSON
	req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestLogin_MissingEmail(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}

	handler := NewAuthHandler(testDB, testCfg)
	router := setupTestRouter(handler)

	loginReq := map[string]string{
		"password": "password123",
	}
	body, _ := json.Marshal(loginReq)

	req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestRefreshToken_Success(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}

	testEmail := "testrefresh-" + uuid.New().String()[:8] + "@example.com"
	cleanupTestUser(testDB, testEmail)
	defer cleanupTestUser(testDB, testEmail)

	handler := NewAuthHandler(testDB, testCfg)
	router := setupTestRouter(handler)

	// Create test user
	user, err := createTestAdminUser(testDB, testEmail, "password123")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Generate a valid refresh token
	tokenPair, err := utils.GenerateTokenPair(
		testCfg.JWT.Secret,
		user.ID,
		user.Email,
		string(user.UserType),
		"super_admin",
		nil,
		testCfg.JWT.ExpirationHours,
		testCfg.JWT.RefreshHours,
	)
	if err != nil {
		t.Fatalf("Failed to generate token pair: %v", err)
	}

	refreshReq := map[string]string{
		"refresh_token": tokenPair.RefreshToken,
	}
	body, _ := json.Marshal(refreshReq)

	req, _ := http.NewRequest("POST", "/api/v1/auth/refresh", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var response utils.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if !response.Success {
		t.Errorf("Expected success=true, got false")
	}
}

func TestRefreshToken_InvalidToken(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}

	handler := NewAuthHandler(testDB, testCfg)
	router := setupTestRouter(handler)

	refreshReq := map[string]string{
		"refresh_token": "invalid-token",
	}
	body, _ := json.Marshal(refreshReq)

	req, _ := http.NewRequest("POST", "/api/v1/auth/refresh", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestLogin_InactiveUser(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}

	testEmail := "testinactive-" + uuid.New().String()[:8] + "@example.com"
	cleanupTestUser(testDB, testEmail)
	defer cleanupTestUser(testDB, testEmail)

	handler := NewAuthHandler(testDB, testCfg)
	router := setupTestRouter(handler)

	// Create inactive user
	hashedPassword, _ := utils.HashPassword("password123")
	user := &models.User{
		ID:           uuid.Must(uuid.NewV7()),
		Email:        testEmail,
		PasswordHash: hashedPassword,
		UserType:     models.UserTypeTenantStaff,
		Status:       models.UserStatusInactive, // Inactive status
	}
	testDB.Create(user)

	staff := &models.TenantStaff{
		ID:       uuid.Must(uuid.NewV7()),
		UserID:   user.ID,
		FullName: "Inactive User",
		Role:     models.TenantStaffRoleAdmin,
	}
	testDB.Create(staff)

	loginReq := LoginRequest{
		Email:    testEmail,
		Password: "password123",
	}
	body, _ := json.Marshal(loginReq)

	req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Inactive users should not be able to login
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d for inactive user, got %d. Body: %s", http.StatusUnauthorized, w.Code, w.Body.String())
	}
}

func TestLogout_Success(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}

	testEmail := "testlogout-" + uuid.New().String()[:8] + "@example.com"
	cleanupTestUser(testDB, testEmail)
	defer cleanupTestUser(testDB, testEmail)

	handler := NewAuthHandler(testDB, testCfg)

	// Setup router with logout endpoint
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/v1/auth/logout", handler.Logout)

	// Create test user
	user, err := createTestAdminUser(testDB, testEmail, "password123")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Generate token
	tokenPair, err := utils.GenerateTokenPair(
		testCfg.JWT.Secret,
		user.ID,
		user.Email,
		string(user.UserType),
		"super_admin",
		nil,
		testCfg.JWT.ExpirationHours,
		testCfg.JWT.RefreshHours,
	)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	req, _ := http.NewRequest("POST", "/api/v1/auth/logout", nil)
	req.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var response utils.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if !response.Success {
		t.Errorf("Expected success=true, got false")
	}
}

func TestRefreshToken_MissingToken(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}

	handler := NewAuthHandler(testDB, testCfg)
	router := setupTestRouter(handler)

	// Empty request body
	req, _ := http.NewRequest("POST", "/api/v1/auth/refresh", bytes.NewBuffer([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d for missing token, got %d", http.StatusBadRequest, w.Code)
	}
}
