package handlers

// Consumer-driven wire-shape tests.
//
// Each test asserts that a critical API response contains EXACTLY the fields the
// frontend actually reads (presence + JSON type — not values, except where the
// value itself is the contract, e.g. success flags). The assertion lists are
// derived by reading the consumer source; every test names its consumer file.
//
// This is a regression net for the "renamed/removed field" bug class this repo
// has shipped before (warranty_expiry_date drift, geolocation {lat,lng} key
// drift). If a handler renames or drops a field one of these tests goes red.
//
// Uses the same infrastructure as geofence_test.go: the package-level testDB /
// testCfg initialized in TestMain (auth_test.go), skip when the DB is
// unavailable, httptest + gin, testify, uuid-suffixed fixtures.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gamatritunggal/smartscan/backend/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
)

// ─── Wire-shape assertion helpers ───────────────────────────────────────────

// wireField asserts the key exists and returns its raw value.
func wireField(t *testing.T, m map[string]interface{}, key string) interface{} {
	t.Helper()
	v, ok := m[key]
	require.True(t, ok, "wire contract broken: field %q missing from payload (frontend reads it)", key)
	return v
}

// wireString asserts key exists and is a JSON string.
func wireString(t *testing.T, m map[string]interface{}, key string) string {
	t.Helper()
	v := wireField(t, m, key)
	s, ok := v.(string)
	require.True(t, ok, "wire contract broken: field %q should be a string, got %T", key, v)
	return s
}

// wireNumber asserts key exists and is a JSON number.
func wireNumber(t *testing.T, m map[string]interface{}, key string) float64 {
	t.Helper()
	v := wireField(t, m, key)
	n, ok := v.(float64)
	require.True(t, ok, "wire contract broken: field %q should be a number, got %T", key, v)
	return n
}

// wireBool asserts key exists and is a JSON boolean.
func wireBool(t *testing.T, m map[string]interface{}, key string) bool {
	t.Helper()
	v := wireField(t, m, key)
	b, ok := v.(bool)
	require.True(t, ok, "wire contract broken: field %q should be a boolean, got %T", key, v)
	return b
}

// wireObject asserts key exists and is a JSON object.
func wireObject(t *testing.T, m map[string]interface{}, key string) map[string]interface{} {
	t.Helper()
	v := wireField(t, m, key)
	o, ok := v.(map[string]interface{})
	require.True(t, ok, "wire contract broken: field %q should be an object, got %T", key, v)
	return o
}

// wireArray asserts key exists and is a JSON array.
func wireArray(t *testing.T, m map[string]interface{}, key string) []interface{} {
	t.Helper()
	v := wireField(t, m, key)
	a, ok := v.([]interface{})
	require.True(t, ok, "wire contract broken: field %q should be an array, got %T", key, v)
	return a
}

// wireAbsent asserts the key is NOT in the payload (e.g. PII stripped by a
// security fix must never resurface).
func wireAbsent(t *testing.T, m map[string]interface{}, key string) {
	t.Helper()
	_, ok := m[key]
	assert.False(t, ok, "wire contract broken: field %q must NOT be present in this public payload", key)
}

// ─── Shared fixture: tenant → product → batch → QR code (+ landing content) ─

type contractFixture struct {
	TenantID  uuid.UUID
	ProductID uuid.UUID
	BatchID   uuid.UUID
	QRCodeID  uuid.UUID
	QRUUID    uuid.UUID
	QRRef     string
}

// setupContractFixture builds the minimal chain GetValidationInfo /
// GetWarrantyStatus require plus the optional landing-page content
// (images, social account, certification, videos, website) so that
// omitempty fields are actually emitted and their item shapes can be asserted.
func setupContractFixture(t *testing.T) (*contractFixture, func()) {
	t.Helper()
	if testDB == nil {
		t.Skip("Database not available")
	}
	uniq := uuid.New().String()[:8]

	tenantID := uuid.Must(uuid.NewV7())
	countryCode := "ID"
	tenant := &models.Tenant{
		ID: tenantID, CompanyName: "ContractCo " + uniq,
		CompanyEmail: "contract-" + uniq + "@test.com", CountryCode: &countryCode,
	}
	require.NoError(t, testDB.Create(tenant).Error)

	productID := uuid.Must(uuid.NewV7())
	product := &models.Product{
		ID: productID, TenantID: tenantID,
		ProductName: "Contract Product " + uniq,
		ProductCode: "CTR-" + uniq,
		Description: "Contract test product description",
		Status:      models.ProductStatusActive,
		// Explicit display config: ValidatePage.vue reads display_config.* keys.
		DisplayConfig: datatypes.JSON(`{"product_name":true,"product_code":true,"brand_name":true,"batch_code":true,"production_date":true,"expiry_date":true,"show_verification_count":true}`),
		// New-format warranty config with every customizable field optional so
		// registration only needs the four fixed fields.
		WarrantyFieldsConfig: datatypes.JSON(`{"enabled":true,"fields":{"store_name":"optional","country":"optional","province":"optional","city":"optional","address":"optional"},"custom_fields":[]}`),
		WarrantyEnabled:      true,
		WarrantyMonths:       24,
		WebsiteURL:           "https://example.com/product",
		WebsiteCaption:       "Visit Product Page",
		Videos:               datatypes.JSON(`[{"url":"https://www.youtube.com/watch?v=demo","title":"Demo"}]`),
	}
	require.NoError(t, testDB.Create(product).Error)

	prodDate := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	expDate := time.Date(2028, 1, 15, 0, 0, 0, 0, time.UTC)
	lat, lng, radius := -6.2088, 106.8456, 50.0
	batchID := uuid.Must(uuid.NewV7())
	batch := &models.QRBatch{
		ID: batchID, TenantID: tenantID, ProductID: productID,
		BatchName: "Contract Batch " + uniq, BatchCode: "CB-" + uniq,
		QRCount:        1,
		ProductionDate: &prodDate, ExpiryDate: &expDate,
		LogoURL: "https://example.com/logo.png",
		// Geofence label => distribution_zone is emitted (ValidatePage reads it).
		GeofenceEnabled: true, GeofenceLatitude: &lat,
		GeofenceLongitude: &lng, GeofenceRadiusKm: &radius,
		GeofenceLabel: "Jakarta Zone",
	}
	require.NoError(t, testDB.Create(batch).Error)

	qrCodeID := uuid.Must(uuid.NewV7())
	qrUUID := uuid.Must(uuid.NewV7())
	qrRef := "CTRQR-" + uniq
	qrCode := &models.QRCode{
		ID: qrCodeID, BatchID: batchID,
		QRUUID: qrUUID, QRCode: qrRef,
		Status:            models.QRCodeStatusActive,
		CounterfeitStatus: models.CounterfeitStatusValid,
	}
	require.NoError(t, testDB.Create(qrCode).Error)

	// Gallery images (ValidatePage reads images[].image_url/caption/is_main).
	mainImg := &models.ProductImage{
		ID: uuid.Must(uuid.NewV7()), ProductID: productID,
		ImageURL: "https://example.com/img-main.jpg", Caption: "Main", IsMain: true, SortOrder: 0,
	}
	require.NoError(t, testDB.Create(mainImg).Error)
	sideImg := &models.ProductImage{
		ID: uuid.Must(uuid.NewV7()), ProductID: productID,
		ImageURL: "https://example.com/img-2.jpg", Caption: "Side", IsMain: false, SortOrder: 1,
	}
	require.NoError(t, testDB.Create(sideImg).Error)

	// Social account chain (ValidatePage reads social_accounts[].platform_code/
	// platform_name/platform_icon/account_handle/url).
	platformID := uuid.Must(uuid.NewV7())
	platform := &models.SocialMediaPlatform{
		ID: platformID, Code: "ctr-ig-" + uniq, Name: "Instagram",
		Icon: "instagram", BaseURL: "https://instagram.com/", IsActive: true,
	}
	require.NoError(t, testDB.Create(platform).Error)
	accountID := uuid.Must(uuid.NewV7())
	account := &models.TenantSocialAccount{
		ID: accountID, TenantID: tenantID, PlatformID: platformID,
		AccountHandle: "smartscan", AccountURL: "https://instagram.com/smartscan", IsActive: true,
	}
	require.NoError(t, testDB.Create(account).Error)
	link := &models.ProductSocialAccountLink{
		ID: uuid.Must(uuid.NewV7()), ProductID: productID, SocialAccountID: accountID, SortOrder: 0,
	}
	require.NoError(t, testDB.Create(link).Error)

	// Certification chain (ValidatePage reads certifications[].name/logo_url/website_url).
	certTypeID := uuid.Must(uuid.NewV7())
	certType := &models.CertificationType{
		ID: certTypeID, Code: "CTRCERT-" + uniq, Name: "Contract Cert",
		LogoURL: "https://example.com/cert.png", WebsiteURL: "https://example.com/cert",
		IsActive: true,
	}
	require.NoError(t, testDB.Create(certType).Error)
	prodCert := &models.ProductCertification{
		ID: uuid.Must(uuid.NewV7()), ProductID: productID,
		CertificationTypeID: certTypeID, RegistrationNumber: "REG-" + uniq,
	}
	require.NoError(t, testDB.Create(prodCert).Error)

	fix := &contractFixture{
		TenantID: tenantID, ProductID: productID, BatchID: batchID,
		QRCodeID: qrCodeID, QRUUID: qrUUID, QRRef: qrRef,
	}

	cleanup := func() {
		testDB.Exec("DELETE FROM warranty_activations WHERE qr_code_id = ?", qrCodeID)
		testDB.Exec("DELETE FROM interactions WHERE tenant_id = ?", tenantID)
		testDB.Unscoped().Where("product_id = ?", productID).Delete(&models.ProductCertification{})
		testDB.Unscoped().Where("id = ?", certTypeID).Delete(&models.CertificationType{})
		testDB.Unscoped().Where("product_id = ?", productID).Delete(&models.ProductSocialAccountLink{})
		testDB.Unscoped().Where("id = ?", accountID).Delete(&models.TenantSocialAccount{})
		testDB.Unscoped().Where("id = ?", platformID).Delete(&models.SocialMediaPlatform{})
		testDB.Unscoped().Where("product_id = ?", productID).Delete(&models.ProductImage{})
		testDB.Unscoped().Where("id = ?", qrCodeID).Delete(&models.QRCode{})
		testDB.Unscoped().Where("id = ?", batchID).Delete(&models.QRBatch{})
		testDB.Unscoped().Where("id = ?", productID).Delete(&models.Product{})
		testDB.Unscoped().Where("id = ?", tenantID).Delete(&models.Tenant{})
	}
	return fix, cleanup
}

// ─── 1. POST /auth/login ────────────────────────────────────────────────────
// Consumer: frontend/src/stores/auth.js
//   - login():       response.success, response.data, response.data.user,
//                    response.data.expires_in (number, fed to setTokenExpiry)
//   - store getters: user.user_type (isTenant), user.role (isAdmin/isQCStaff/
//                    isWarehouseStaff), user.must_change_password (strict
//                    === true check), user.id (logout / tour state key)

func TestContract_Login_WireShape(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}

	testEmail := "contract-login-" + uuid.New().String()[:8] + "@example.com"
	cleanupTestUser(testDB, testEmail)
	defer func() {
		// Best-effort: remove the async audit row before deleting the user.
		var u models.User
		if err := testDB.Unscoped().Where("email = ?", testEmail).First(&u).Error; err == nil {
			testDB.Exec("DELETE FROM activity_logs WHERE user_id = ?", u.ID)
		}
		cleanupTestUser(testDB, testEmail)
		cleanupTestTenant(testDB, "authtest-"+testEmail)
	}()

	_, err := createTestAdminUser(testDB, testEmail, "password123")
	require.NoError(t, err)

	handler := NewAuthHandler(testDB, testCfg)
	router := setupTestRouter(handler)

	body, _ := json.Marshal(LoginRequest{Email: testEmail, Password: "password123"})
	req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())
	resp := parseResponse(t, w)

	// auth.js: `if (response.success && response.data)` — success value IS the contract.
	require.Equal(t, true, resp["success"], "auth.js gates the whole login flow on success === true")
	data := getData(t, resp)

	// auth.js: setTokenExpiry(response.data.expires_in) — seconds as a number.
	expiresIn := wireNumber(t, data, "expires_in")
	assert.Greater(t, expiresIn, float64(0), "expires_in must be a positive number of seconds")

	// auth.js: setUser(response.data.user)
	user := wireObject(t, data, "user")
	assert.NotEmpty(t, wireString(t, user, "id"), "user.id is read on logout (tour-state key)")
	assert.NotEmpty(t, wireString(t, user, "user_type"), "user.user_type drives isTenant")
	assert.NotEmpty(t, wireString(t, user, "role"), "user.role drives isAdmin/isQCStaff/isWarehouseStaff")
	// mustChangePassword does a strict `=== true`, so the field must be a real
	// boolean — a missing key or a string "false" would silently change behavior.
	wireBool(t, user, "must_change_password")
}

// ─── 2. GET /public/validate-info/:code ─────────────────────────────────────
// Consumer: frontend/src/pages/public/ValidatePage.vue — the app's most
// important public endpoint. Field list derived from every validationData.*
// read in that file (template + computed props + replacePlaceholders):
//   is_counterfeit (=== true), need_warranty (=== true), message,
//   validation_count, qr_code_ref, distribution_zone, website_url,
//   website_caption, display_config.*, landing_appearance_config,
//   product.{name,code,description}, batch.{batch_code,production_date,
//   expiry_date}, tenant.{company_name,brand_name,logo_url},
//   certifications[].{name,logo_url,website_url},
//   social_accounts[].{platform_code,platform_name,platform_icon,
//   account_handle,url}, images[].{image_url,caption,is_main}, videos.

func TestContract_ValidateInfo_WireShape(t *testing.T) {
	fix, cleanup := setupContractFixture(t)
	defer cleanup()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := NewValidationHandler(testDB, testCfg)
	router.GET("/api/v1/public/validate-info/:code", h.GetValidationInfo)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/public/validate-info/%s", fix.QRUUID), nil)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())
	resp := parseResponse(t, w)
	require.Equal(t, true, resp["success"], "ValidatePage gates on response.data.success")
	data := getData(t, resp)

	// Core flow flags — ValidatePage does strict boolean comparisons on these.
	assert.Equal(t, true, wireBool(t, data, "is_valid"))
	assert.Equal(t, false, wireBool(t, data, "is_counterfeit"), "authentic fixture must not be flagged")
	assert.Equal(t, true, wireBool(t, data, "need_warranty"), "warranty-enabled product drives the warranty button")
	assert.NotEmpty(t, wireString(t, data, "message"))
	wireNumber(t, data, "validation_count")
	assert.NotEmpty(t, wireString(t, data, "qr_code_ref"), "shown to users for counterfeit reports")
	assert.NotEmpty(t, wireString(t, data, "qr_code_id"))
	assert.NotEmpty(t, wireString(t, data, "batch_id"))
	assert.NotEmpty(t, wireString(t, data, "counterfeit_status"))
	assert.NotEmpty(t, wireString(t, data, "qr_status"))
	assert.Equal(t, "Jakarta Zone", wireString(t, data, "distribution_zone"))
	assert.NotEmpty(t, wireString(t, data, "website_url"))
	assert.NotEmpty(t, wireString(t, data, "website_caption"))

	// display_config: ValidatePage reads the per-field visibility keys.
	displayConfig := wireObject(t, data, "display_config")
	for _, key := range []string{"product_name", "product_code", "brand_name", "batch_code", "production_date", "expiry_date", "show_verification_count"} {
		wireField(t, displayConfig, key)
	}

	// landing_appearance_config: read for background rendering; always an object.
	landing := wireObject(t, data, "landing_appearance_config")
	wireString(t, landing, "background_type")

	// product.{name, code, description}
	product := wireObject(t, data, "product")
	assert.NotEmpty(t, wireString(t, product, "name"))
	assert.NotEmpty(t, wireString(t, product, "code"))
	assert.NotEmpty(t, wireString(t, product, "description"))

	// batch.{batch_code, production_date, expiry_date}
	batch := wireObject(t, data, "batch")
	assert.NotEmpty(t, wireString(t, batch, "batch_code"))
	assert.NotEmpty(t, wireString(t, batch, "production_date"))
	assert.NotEmpty(t, wireString(t, batch, "expiry_date"))

	// tenant.{company_name, brand_name, logo_url}
	tenant := wireObject(t, data, "tenant")
	assert.NotEmpty(t, wireString(t, tenant, "company_name"))
	assert.NotEmpty(t, wireString(t, tenant, "brand_name"))
	assert.NotEmpty(t, wireString(t, tenant, "logo_url"))

	// certifications[] item shape
	certs := wireArray(t, data, "certifications")
	require.NotEmpty(t, certs, "fixture created a certification — it must be emitted")
	cert, ok := certs[0].(map[string]interface{})
	require.True(t, ok)
	assert.NotEmpty(t, wireString(t, cert, "name"))
	wireString(t, cert, "logo_url")
	wireString(t, cert, "website_url")

	// social_accounts[] item shape
	socials := wireArray(t, data, "social_accounts")
	require.NotEmpty(t, socials, "fixture linked a social account — it must be emitted")
	social, ok := socials[0].(map[string]interface{})
	require.True(t, ok)
	assert.NotEmpty(t, wireString(t, social, "platform_code"))
	assert.NotEmpty(t, wireString(t, social, "platform_name"))
	wireString(t, social, "platform_icon")
	assert.NotEmpty(t, wireString(t, social, "account_handle"))
	assert.NotEmpty(t, wireString(t, social, "url"))

	// images[] item shape
	images := wireArray(t, data, "images")
	require.Len(t, images, 2, "fixture created two gallery images")
	img, ok := images[0].(map[string]interface{})
	require.True(t, ok)
	assert.NotEmpty(t, wireString(t, img, "image_url"))
	wireField(t, img, "caption")
	wireBool(t, img, "is_main")

	// videos: ValidatePage reads videos.length — must be an array when set.
	videos := wireArray(t, data, "videos")
	assert.NotEmpty(t, videos)
}

// ─── 3. GET /public/warranty/:code (status) ─────────────────────────────────
// Consumer: frontend/src/pages/public/WarrantyPage.vue
//   productData reads: warranty_registered, warranty_months, warranty_expiry
//   (top-level, only when registered), warranty_fields_config.{fields,
//   custom_fields}, product.{product_name,product_code}, tenant.{logo_url,
//   brand_name,company_name}. (product.description is part of the emitted
//   product object contract.)

func TestContract_WarrantyStatus_WireShape(t *testing.T) {
	fix, cleanup := setupContractFixture(t)
	defer cleanup()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := NewWarrantyHandler(testDB, testCfg)
	router.GET("/api/v1/public/warranty/:code", h.GetWarrantyStatus)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/public/warranty/%s", fix.QRUUID), nil)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())
	resp := parseResponse(t, w)
	require.Equal(t, true, resp["success"])
	data := getData(t, resp)

	wireBool(t, data, "is_valid")
	assert.Equal(t, true, wireBool(t, data, "need_warranty"))
	assert.Equal(t, false, wireBool(t, data, "warranty_registered"))
	assert.Equal(t, float64(24), wireNumber(t, data, "warranty_months"), "fixture set 24 months")

	// warranty_fields_config drives the dynamic registration form.
	cfg := wireObject(t, data, "warranty_fields_config")
	wireObject(t, cfg, "fields")
	wireField(t, cfg, "custom_fields")

	product := wireObject(t, data, "product")
	assert.NotEmpty(t, wireString(t, product, "product_name"))
	assert.NotEmpty(t, wireString(t, product, "product_code"))
	assert.NotEmpty(t, wireString(t, product, "description"))

	tenant := wireObject(t, data, "tenant")
	assert.NotEmpty(t, wireString(t, tenant, "company_name"))
	assert.NotEmpty(t, wireString(t, tenant, "brand_name"))
	assert.NotEmpty(t, wireString(t, tenant, "logo_url"))

	// Not registered yet — warranty_expiry must be absent.
	wireAbsent(t, data, "warranty_expiry")
}

// TestContract_WarrantyStatus_RegisteredPIIAbsent is the regression test for
// the PII-leak security fix: the public, unauthenticated warranty-status
// endpoint must expose ONLY the expiry date for a registered warranty — never
// the registrant's identity or purchase details (QR codes are printed on
// physical labels and can be enumerated by anyone).
func TestContract_WarrantyStatus_RegisteredPIIAbsent(t *testing.T) {
	fix, cleanup := setupContractFixture(t)
	defer cleanup()

	purchase := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	expiry := purchase.AddDate(0, 24, 0)
	geo, _ := json.Marshal(map[string]float64{"lat": -6.2, "lng": 106.8})
	countryCode := "ID"
	activation := &models.WarrantyActivation{
		QRCodeID:           fix.QRCodeID,
		CustomerName:       "PII Sentinel Buyer",
		CustomerEmail:      "pii-sentinel@example.com",
		CustomerPhone:      "+628111111111",
		PurchaseDate:       &purchase,
		PurchaseStore:      "PII Sentinel Store",
		Address:            "PII Sentinel Address 123",
		CountryCode:        &countryCode,
		WarrantyExpiryDate: &expiry,
		IPAddress:          "203.0.113.7",
		Geolocation:        datatypes.JSON(geo),
	}
	require.NoError(t, testDB.Create(activation).Error)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := NewWarrantyHandler(testDB, testCfg)
	router.GET("/api/v1/public/warranty/:code", h.GetWarrantyStatus)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/public/warranty/%s", fix.QRUUID), nil)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())
	resp := parseResponse(t, w)
	data := getData(t, resp)

	// WarrantyPage reads warranty_registered and the top-level warranty_expiry.
	assert.Equal(t, true, wireBool(t, data, "warranty_registered"))
	assert.NotEmpty(t, wireString(t, data, "warranty_expiry"))

	// The fields removed by the security fix must stay absent.
	for _, piiKey := range []string{
		"customer_name", "customer_email", "customer_phone",
		"purchase_store", "purchase_date", "address",
		"country_code", "province_id", "city_id",
		"activated_at", "ip_address", "geolocation", "activation_data",
		"warranty", // the raw activation record must never be embedded
	} {
		wireAbsent(t, data, piiKey)
	}

	// Belt and braces: none of the seeded PII values may appear anywhere in the
	// raw response body (catches leaks through nested/renamed keys too).
	body := w.Body.String()
	for _, sentinel := range []string{
		"PII Sentinel Buyer", "pii-sentinel@example.com", "+628111111111",
		"PII Sentinel Store", "PII Sentinel Address 123", "203.0.113.7",
	} {
		assert.NotContains(t, body, sentinel, "PII value leaked into the public warranty-status payload")
	}
}

// ─── POST /public/warranty/:code (registration) ─────────────────────────────
// Consumer: frontend/src/pages/public/WarrantyPage.vue submitRegistration():
//   `if (response.data.success && response.data.data?.success)` — the
//   double-success envelope: HTTP wrapper success AND business-result success.
//   On success it stores response.data.data and renders
//   registrationResult.warranty_expiry_date (the exact field name that
//   drifted in a shipped bug). On failure it shows data.data?.message.

func TestContract_WarrantyRegister_DoubleSuccessEnvelope(t *testing.T) {
	fix, cleanup := setupContractFixture(t)
	defer cleanup()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := NewWarrantyHandler(testDB, testCfg)
	router.POST("/api/v1/public/warranty/:code", h.RegisterWarranty)

	payload := map[string]interface{}{
		"customer_name": "Contract Buyer",
		"email":         "contractbuyer" + uuid.New().String()[:8] + "@gmail.com",
		"phone":         "+6281234567890",
		"purchase_date": time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02"),
	}
	body, _ := json.Marshal(payload)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/public/warranty/%s", fix.QRUUID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	// The handler stores c.ClientIP() into an inet column; direct
	// router.ServeHTTP requests have no RemoteAddr, which would yield "".
	req.RemoteAddr = "203.0.113.10:12345"
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())
	resp := parseResponse(t, w)

	// Outer envelope success (response.data.success in the Vue consumer).
	require.Equal(t, true, resp["success"], "outer envelope success must be true")
	data := getData(t, resp)

	// Inner business success (response.data.data.success) — the double-success
	// contract. Both must be booleans with these exact values.
	assert.Equal(t, true, wireBool(t, data, "success"), "inner (business) success must be true")
	assert.NotEmpty(t, wireString(t, data, "message"))
	assert.Equal(t, false, wireBool(t, data, "already_registered"))
	assert.NotEmpty(t, wireString(t, data, "warranty_expiry_date"),
		"warranty_expiry_date is rendered by WarrantyPage — this exact key drifted in a shipped bug")
	assert.NotEmpty(t, wireString(t, data, "batch_id"))

	// Second registration on the same QR: outer success stays true, inner
	// success flips false, already_registered true, expiry still present.
	// WarrantyPage reads data.data.message for the error banner in this case.
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("POST", fmt.Sprintf("/api/v1/public/warranty/%s", fix.QRUUID), bytes.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	req2.RemoteAddr = "203.0.113.10:12345"
	router.ServeHTTP(w2, req2)

	require.Equal(t, http.StatusOK, w2.Code, "body: %s", w2.Body.String())
	resp2 := parseResponse(t, w2)
	require.Equal(t, true, resp2["success"])
	data2 := getData(t, resp2)
	assert.Equal(t, false, wireBool(t, data2, "success"))
	assert.Equal(t, true, wireBool(t, data2, "already_registered"))
	assert.NotEmpty(t, wireString(t, data2, "message"))
	assert.NotEmpty(t, wireString(t, data2, "warranty_expiry_date"))
}

// ─── 4. GET /tenant/products (paginated list envelope) ──────────────────────
// Consumers: frontend/src/pages/tenant/products/DynamicQRPage.vue
//   (response.data.products, p.id / p.product_name / p.product_code) and
//   frontend/src/lib/pagination.js getPagination(response.data)
//   (pagination.page / .limit / .total / .total_page).
// Locks the utils.PaginationMeta contract end-to-end, including the singular
// "total_page" key the frontend depends on.

func TestContract_TenantProducts_ListEnvelope(t *testing.T) {
	fix, cleanup := setupContractFixture(t)
	defer cleanup()

	// Same auth pattern as geofence_test.go: tenant endpoints only need
	// tenant_id in the gin context, injected by a stand-in middleware.
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("tenant_id", fix.TenantID.String())
		c.Next()
	})
	h := NewProductHandler(testDB, testCfg)
	router.GET("/api/v1/tenant/products", h.ListProducts)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/tenant/products?page=1&limit=10", nil)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())
	resp := parseResponse(t, w)
	require.Equal(t, true, resp["success"])
	data := getData(t, resp)

	products := wireArray(t, data, "products")
	require.NotEmpty(t, products, "fixture created one product for this tenant")
	item, ok := products[0].(map[string]interface{})
	require.True(t, ok)
	assert.NotEmpty(t, wireString(t, item, "id"), "DynamicQRPage routes on product.id")
	assert.NotEmpty(t, wireString(t, item, "product_name"), "client-side search filters on product_name")
	wireString(t, item, "product_code")

	pagination := wireObject(t, data, "pagination")
	assert.Equal(t, float64(1), wireNumber(t, pagination, "page"))
	assert.Equal(t, float64(10), wireNumber(t, pagination, "limit"))
	assert.GreaterOrEqual(t, wireNumber(t, pagination, "total"), float64(1))
	assert.GreaterOrEqual(t, wireNumber(t, pagination, "total_page"), float64(1),
		"total_page (singular) is the key lib/pagination.js reads — renaming it silently breaks every paginated page")
}
