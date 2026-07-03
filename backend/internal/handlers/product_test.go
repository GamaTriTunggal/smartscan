package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gamatritunggal/smartscan/backend/internal/models"
	"github.com/gamatritunggal/smartscan/backend/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createProductTestTenant creates a tenant, user, and staff for product tests.
// Returns tenantID, userID, and a cleanup function.
func createProductTestTenant(t *testing.T, suffix string) (uuid.UUID, uuid.UUID, func()) {
	t.Helper()

	tenantID := uuid.Must(uuid.NewV7())
	countryCode := "ID"
	tenant := &models.Tenant{
		ID:           tenantID,
		CompanyName:  "Product Test Co " + suffix,
		CompanyEmail: "product-" + suffix + "@test.com",
		CountryCode:  &countryCode,
	}
	require.NoError(t, testDB.Create(tenant).Error)

	hashedPassword, _ := utils.HashPassword("password")
	userID := uuid.Must(uuid.NewV7())
	user := &models.User{
		ID:           userID,
		Email:        "product-user-" + suffix + "@test.com",
		PasswordHash: hashedPassword,
		UserType:     models.UserTypeTenantStaff,
		Status:       models.UserStatusActive,
	}
	require.NoError(t, testDB.Create(user).Error)

	staff := &models.TenantStaff{
		ID:             uuid.Must(uuid.NewV7()),
		TenantID:       tenantID,
		UserID:         userID,
		FullName:       "Product Test User " + suffix,
		Role:           models.TenantStaffRoleAdmin,
		IsPrimaryAdmin: true,
	}
	require.NoError(t, testDB.Create(staff).Error)

	cleanup := func() {
		testDB.Unscoped().Where("tenant_id = ?", tenantID).Delete(&models.Product{})
		testDB.Unscoped().Where("id = ?", staff.ID).Delete(&models.TenantStaff{})
		testDB.Unscoped().Where("id = ?", userID).Delete(&models.User{})
		testDB.Unscoped().Where("id = ?", tenantID).Delete(&models.Tenant{})
	}

	return tenantID, userID, cleanup
}

func setupCreateProductTestRouter(h *ProductHandler, tenantID, userID uuid.UUID) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("tenant_id", tenantID.String())
		c.Set("user_id", userID.String())
		c.Next()
	})
	r.POST("/api/v1/tenant/products", h.CreateProduct)
	return r
}

func TestCreateProduct_Success(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}

	uniq := uuid.New().String()[:8]
	tenantID, userID, cleanup := createProductTestTenant(t, uniq)
	defer cleanup()

	h := NewProductHandler(testDB, testCfg)
	router := setupCreateProductTestRouter(h, tenantID, userID)

	body := map[string]interface{}{
		"product_name": "Test Product " + uniq,
		"product_code": "PROD-" + uniq,
	}
	jsonBody, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/tenant/products", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "Product created", resp["message"])
}

func setupProductDuplicateTestRouter(h *ProductHandler, tenantID, userID uuid.UUID) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("tenant_id", tenantID.String())
		c.Set("user_id", userID.String())
		c.Next()
	})
	r.POST("/api/v1/tenant/products", h.CreateProduct)
	r.PUT("/api/v1/tenant/products/:id", h.UpdateProduct)
	return r
}

// createProductViaAPI creates a product via the API and returns the product ID.
func createProductViaAPI(t *testing.T, router *gin.Engine, name, code string) uuid.UUID {
	t.Helper()
	body := map[string]interface{}{
		"product_name": name,
		"product_code": code,
	}
	jsonBody, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/tenant/products", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code, "Product creation should succeed")

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	id, err := uuid.Parse(data["id"].(string))
	require.NoError(t, err)
	return id
}

// --- CreateProduct duplicate tests ---

func TestCreateProduct_DuplicateName_Rejected(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}
	uniq := uuid.New().String()[:8]
	tenantID, userID, cleanup := createProductTestTenant(t, uniq)
	defer cleanup()

	h := NewProductHandler(testDB, testCfg)
	router := setupProductDuplicateTestRouter(h, tenantID, userID)

	name := "Duplicate Name " + uniq
	createProductViaAPI(t, router, name, "CODE-A-"+uniq)

	// Attempt to create another product with the same name
	body := map[string]interface{}{
		"product_name": name,
		"product_code": "CODE-B-" + uniq,
	}
	jsonBody, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/tenant/products", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "Product name already exists", resp["message"])
}

func TestCreateProduct_DuplicateCode_Rejected(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}
	uniq := uuid.New().String()[:8]
	tenantID, userID, cleanup := createProductTestTenant(t, uniq)
	defer cleanup()

	h := NewProductHandler(testDB, testCfg)
	router := setupProductDuplicateTestRouter(h, tenantID, userID)

	code := "DUP-CODE-" + uniq
	createProductViaAPI(t, router, "Product A "+uniq, code)

	// Attempt to create another product with the same code
	body := map[string]interface{}{
		"product_name": "Product B " + uniq,
		"product_code": code,
	}
	jsonBody, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/tenant/products", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "Product code already exists", resp["message"])
}

func TestCreateProduct_SameNameDifferentTenant_Allowed(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}
	uniq := uuid.New().String()[:8]

	tenantA, userA, cleanupA := createProductTestTenant(t, "a-"+uniq)
	defer cleanupA()

	tenantB, userB, cleanupB := createProductTestTenant(t, "b-"+uniq)
	defer cleanupB()

	h := NewProductHandler(testDB, testCfg)
	routerA := setupProductDuplicateTestRouter(h, tenantA, userA)
	routerB := setupProductDuplicateTestRouter(h, tenantB, userB)

	sharedName := "Cross Tenant Widget " + uniq

	// Create in tenant A
	createProductViaAPI(t, routerA, sharedName, "CT-A-"+uniq)

	// Create same name in tenant B — should succeed
	body := map[string]interface{}{
		"product_name": sharedName,
		"product_code": "CT-B-" + uniq,
	}
	jsonBody, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/tenant/products", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	routerB.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code, "Same name in different tenant should be allowed")
}

func TestCreateProduct_SameNameAfterSoftDelete_Allowed(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}
	uniq := uuid.New().String()[:8]
	tenantID, userID, cleanup := createProductTestTenant(t, uniq)
	defer cleanup()

	h := NewProductHandler(testDB, testCfg)
	router := setupProductDuplicateTestRouter(h, tenantID, userID)

	name := "Soft Delete Reuse " + uniq
	productID := createProductViaAPI(t, router, name, "SD-"+uniq)

	// Soft delete the product
	require.NoError(t, testDB.Delete(&models.Product{}, "id = ?", productID).Error)

	// Create another product with the same name — should succeed
	body := map[string]interface{}{
		"product_name": name,
		"product_code": "SD2-" + uniq,
	}
	jsonBody, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/tenant/products", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code, "Reusing a soft-deleted product's name should be allowed")
}

// --- UpdateProduct duplicate tests ---

func TestUpdateProduct_DuplicateName_Rejected(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}
	uniq := uuid.New().String()[:8]
	tenantID, userID, cleanup := createProductTestTenant(t, uniq)
	defer cleanup()

	h := NewProductHandler(testDB, testCfg)
	router := setupProductDuplicateTestRouter(h, tenantID, userID)

	nameA := "Update Dup A " + uniq
	nameB := "Update Dup B " + uniq
	createProductViaAPI(t, router, nameA, "UDA-"+uniq)
	idB := createProductViaAPI(t, router, nameB, "UDB-"+uniq)

	// Rename B to A's name — should be rejected
	body := map[string]interface{}{
		"product_name": nameA,
	}
	jsonBody, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/tenant/products/"+idB.String(), bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "Product name already exists", resp["message"])
}

func TestUpdateProduct_DuplicateCode_Rejected(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}
	uniq := uuid.New().String()[:8]
	tenantID, userID, cleanup := createProductTestTenant(t, uniq)
	defer cleanup()

	h := NewProductHandler(testDB, testCfg)
	router := setupProductDuplicateTestRouter(h, tenantID, userID)

	codeA := "UDC-A-" + uniq
	codeB := "UDC-B-" + uniq
	createProductViaAPI(t, router, "Code Dup A "+uniq, codeA)
	idB := createProductViaAPI(t, router, "Code Dup B "+uniq, codeB)

	// Change B's code to A's code — should be rejected
	body := map[string]interface{}{
		"product_code": codeA,
	}
	jsonBody, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/tenant/products/"+idB.String(), bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "Product code already exists", resp["message"])
}

func TestUpdateProduct_SameName_NoChange_Allowed(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}
	uniq := uuid.New().String()[:8]
	tenantID, userID, cleanup := createProductTestTenant(t, uniq)
	defer cleanup()

	h := NewProductHandler(testDB, testCfg)
	router := setupProductDuplicateTestRouter(h, tenantID, userID)

	name := "Self Rename " + uniq
	id := createProductViaAPI(t, router, name, "SR-"+uniq)

	// Update with the same name — should succeed
	body := map[string]interface{}{
		"product_name": name,
	}
	jsonBody, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/tenant/products/"+id.String(), bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "Updating with same name should be allowed")
}

func TestUpdateProduct_SameCode_NoChange_Allowed(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}
	uniq := uuid.New().String()[:8]
	tenantID, userID, cleanup := createProductTestTenant(t, uniq)
	defer cleanup()

	h := NewProductHandler(testDB, testCfg)
	router := setupProductDuplicateTestRouter(h, tenantID, userID)

	code := "SELF-CODE-" + uniq
	id := createProductViaAPI(t, router, "Self Code "+uniq, code)

	// Update with the same code — should succeed
	body := map[string]interface{}{
		"product_code": code,
	}
	jsonBody, _ := json.Marshal(body)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/tenant/products/"+id.String(), bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "Updating with same code should be allowed")
}
