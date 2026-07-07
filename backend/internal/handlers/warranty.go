package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gamatritunggal/smartscan/backend/internal/config"
	"github.com/gamatritunggal/smartscan/backend/internal/models"
	"github.com/gamatritunggal/smartscan/backend/internal/sentry"
	"github.com/gamatritunggal/smartscan/backend/internal/utils"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type WarrantyHandler struct {
	DB  *gorm.DB
	Cfg *config.Config
}

func NewWarrantyHandler(db *gorm.DB, cfg *config.Config) *WarrantyHandler {
	return &WarrantyHandler{DB: db, Cfg: cfg}
}

type RegisterWarrantyRequest struct {
	// Fixed required fields (always required)
	CustomerName  string   `json:"customer_name" binding:"required,max=255"`
	Email         string   `json:"email" binding:"required,email,max=255"`
	Phone         string   `json:"phone" binding:"required,max=20"`
	PurchaseDate  string   `json:"purchase_date" binding:"required,max=50"`
	// Customizable fields (can be hidden/optional/required based on product config)
	PurchaseStore string   `json:"purchase_store" binding:"max=255"`
	Address       string   `json:"address" binding:"max=500"`
	CountryCode   *string  `json:"country_code"`
	ProvinceID    *int     `json:"province_id"`
	CityID        *int     `json:"city_id"`
	// Geolocation (optional, captured from browser)
	Latitude      *float64 `json:"latitude"`
	Longitude     *float64 `json:"longitude"`
	// Custom fields (dynamic fields defined per product)
	CustomFields  map[string]interface{} `json:"custom_fields"`
}

// WarrantyFieldsConfig represents the product-level warranty field configuration
type WarrantyFieldsConfig struct {
	Enabled      bool                        `json:"enabled"`
	Fields       map[string]string           `json:"fields"` // "hidden" | "optional" | "required"
	CustomFields []WarrantyCustomFieldConfig `json:"custom_fields"`
}

// WarrantyCustomFieldConfig represents a custom field definition
type WarrantyCustomFieldConfig struct {
	ID       string   `json:"id"`
	Label    string   `json:"label"`
	Type     string   `json:"type"` // text, textarea, number, date, select, email, phone
	Required bool     `json:"required"`
	Options  []string `json:"options,omitempty"` // for select type
}

// getDefaultWarrantyFieldsConfig returns the default config when product has no config
func getDefaultWarrantyFieldsConfig() WarrantyFieldsConfig {
	return WarrantyFieldsConfig{
		Enabled: true,
		Fields: map[string]string{
			"store_name": "optional",
			"country":    "required",
			"province":   "required",
			"city":       "required",
			"address":    "required",
		},
	}
}

type WarrantyRegistrationResult struct {
	Success             bool        `json:"success"`
	Message             string      `json:"message"`
	AlreadyRegistered   bool        `json:"already_registered"`
	WarrantyExpiryDate  *time.Time  `json:"warranty_expiry_date,omitempty"`
	// Flow logic flags
	BatchID             string      `json:"batch_id,omitempty"`
}

// RegisterWarranty handles public warranty registration
func (h *WarrantyHandler) RegisterWarranty(c *gin.Context) {
	code := c.Param("code")

	var req RegisterWarrantyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request", err)
		return
	}

	// Validate and normalize email (disposable check + Gmail normalization + +suffix removal)
	normalizedEmail, err := utils.ValidateEmailForCampaign(req.Email)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error(), nil)
		return
	}
	req.Email = normalizedEmail

	// Validate and normalize phone (E.164 format)
	countryHint := ""
	if req.CountryCode != nil {
		countryHint = *req.CountryCode
	}
	normalizedPhone, err := utils.ValidateAndNormalizePhone(req.Phone, countryHint)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error(), nil)
		return
	}
	req.Phone = normalizedPhone

	// Parse QR code parameter (supports Base58, UUID, hex formats)
	lookup, err := utils.ParseQRCodeParam(code)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	// Find QR code with batch and product info
	var qrCode models.QRCode
	var lookupErr error
	if lookup.LookupByCode {
		lookupErr = h.DB.Preload("Batch.Product").Preload("Batch.Tenant").First(&qrCode, "qr_code = ?", lookup.OriginalCode).Error
	} else {
		lookupErr = h.DB.Preload("Batch.Product").Preload("Batch.Tenant").First(&qrCode, "qr_uuid = ?", lookup.QRUUID).Error
	}
	if lookupErr != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Product not found", lookupErr)
		return
	}

	// IsScannable() combines status + counterfeit_status.
	if !qrCode.IsScannable() {
		utils.SuccessResponse(c, http.StatusOK, "Warranty registration result", WarrantyRegistrationResult{
			Success:           false,
			Message:           "QR code is not active",
			AlreadyRegistered: false,
		})
		return
	}

	// Check if warranty is enabled for this product (product-level setting)
	if qrCode.Batch == nil || qrCode.Batch.Product == nil || !qrCode.Batch.Product.WarrantyEnabled {
		utils.SuccessResponse(c, http.StatusOK, "Warranty registration result", WarrantyRegistrationResult{
			Success:           false,
			Message:           "Warranty registration is not available for this product",
			AlreadyRegistered: false,
		})
		return
	}

	// Check if warranty already activated
	var existingWarranty models.WarrantyActivation
	if err := h.DB.Where("qr_code_id = ?", qrCode.ID).First(&existingWarranty).Error; err == nil {
		// Anti-counterfeit signal: a second registration attempt on the same QR is a
		// classic cloned-label indicator. Track it so the counterfeit dashboard can
		// surface repeat offenders. Best-effort — never blocks the response.
		now := time.Now().UTC()
		h.DB.Model(&existingWarranty).UpdateColumns(map[string]interface{}{
			"duplicate_attempt_count":   gorm.Expr("duplicate_attempt_count + 1"),
			"last_duplicate_attempt_at": now,
		})

		utils.SuccessResponse(c, http.StatusOK, "Warranty registration result", WarrantyRegistrationResult{
			Success:            false,
			Message:            "Warranty has already been activated for this product",
			AlreadyRegistered:  true,
			WarrantyExpiryDate: existingWarranty.WarrantyExpiryDate,
			BatchID:            qrCode.Batch.ID.String(),
		})
		return
	}

	// Parse purchase date
	purchaseDate, err := time.Parse("2006-01-02", req.PurchaseDate)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid purchase date format", err)
		return
	}

	// Validate purchase date not in future
	if purchaseDate.After(time.Now()) {
		utils.ErrorResponse(c, http.StatusBadRequest, "Purchase date cannot be in the future", nil)
		return
	}

	// Validate warranty registration window (if configured on product)
	if qrCode.Batch.Product != nil && qrCode.Batch.Product.MaxWarrantyRegistrationDays != nil {
		maxDays := *qrCode.Batch.Product.MaxWarrantyRegistrationDays
		if maxDays > 0 {
			daysSincePurchase := int(time.Since(purchaseDate).Hours() / 24)
			if daysSincePurchase > maxDays {
				utils.ErrorResponse(c, http.StatusBadRequest,
					fmt.Sprintf("Warranty registration period has expired. Must register within %d days of purchase.", maxDays), nil)
				return
			}
		}
	}

	// Load product and get warranty fields config
	var product models.Product
	var fieldsConfig WarrantyFieldsConfig
	if qrCode.Batch.ProductID != uuid.Nil {
		if err := h.DB.First(&product, "id = ?", qrCode.Batch.ProductID).Error; err == nil {
			if product.WarrantyFieldsConfig != nil {
				// Try to parse the new format first
				if err := json.Unmarshal(product.WarrantyFieldsConfig, &fieldsConfig); err != nil || fieldsConfig.Fields == nil {
					// Fall back to default if parsing fails or old format
					fieldsConfig = getDefaultWarrantyFieldsConfig()
				}
			} else {
				fieldsConfig = getDefaultWarrantyFieldsConfig()
			}
		} else {
			fieldsConfig = getDefaultWarrantyFieldsConfig()
		}
	} else {
		fieldsConfig = getDefaultWarrantyFieldsConfig()
	}

	// Conditional validation based on product's warranty fields config
	if fieldsConfig.Fields["store_name"] == "required" && req.PurchaseStore == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "Store name is required", nil)
		return
	}
	if fieldsConfig.Fields["country"] == "required" && (req.CountryCode == nil || *req.CountryCode == "") {
		utils.ErrorResponse(c, http.StatusBadRequest, "Country is required", nil)
		return
	}
	if fieldsConfig.Fields["province"] == "required" && req.ProvinceID == nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Province is required", nil)
		return
	}
	if fieldsConfig.Fields["city"] == "required" && req.CityID == nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "City is required", nil)
		return
	}
	if fieldsConfig.Fields["address"] == "required" && req.Address == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "Address is required", nil)
		return
	}

	// Validate custom fields based on product config
	for _, customField := range fieldsConfig.CustomFields {
		value := req.CustomFields[customField.ID]
		if customField.Required {
			if value == nil {
				utils.ErrorResponse(c, http.StatusBadRequest,
					fmt.Sprintf("Field '%s' is required", customField.Label), nil)
				return
			}
			// Check for empty string
			if strVal, ok := value.(string); ok && strVal == "" {
				utils.ErrorResponse(c, http.StatusBadRequest,
					fmt.Sprintf("Field '%s' is required", customField.Label), nil)
				return
			}
		}
		// Validate select type has valid option
		if value != nil && customField.Type == "select" && len(customField.Options) > 0 {
			strVal, ok := value.(string)
			if ok && strVal != "" {
				validOption := false
				for _, opt := range customField.Options {
					if opt == strVal {
						validOption = true
						break
					}
				}
				if !validOption {
					utils.ErrorResponse(c, http.StatusBadRequest,
						fmt.Sprintf("Invalid value for field '%s'", customField.Label), nil)
					return
				}
			}
		}
	}

	// Calculate warranty expiry (use product config or default 12 months)
	warrantyMonths := 12
	if qrCode.Batch.Product != nil && qrCode.Batch.Product.WarrantyMonths > 0 {
		warrantyMonths = qrCode.Batch.Product.WarrantyMonths
	}
	warrantyExpiry := purchaseDate.AddDate(0, warrantyMonths, 0)

	// Prepare geolocation. Only attach it when GPS coords are present, and use the
	// {"lat","lng"} shape every other writer/reader uses (velocity.go, dashboard.go,
	// validation.go). A nil datatypes.JSON is stored as SQL NULL.
	var geolocation datatypes.JSON
	if req.Latitude != nil && req.Longitude != nil {
		geolocationData, _ := json.Marshal(map[string]float64{
			"lat": *req.Latitude,
			"lng": *req.Longitude,
		})
		geolocation = datatypes.JSON(geolocationData)
	}

	// Prepare activation data (includes custom fields)
	activationData := make(map[string]interface{})
	if len(req.CustomFields) > 0 {
		activationData["custom_fields"] = req.CustomFields
	}
	activationDataJSON, _ := json.Marshal(activationData)

	// Create warranty activation
	warranty := models.WarrantyActivation{
		QRCodeID:           qrCode.ID,
		CustomerName:       req.CustomerName,
		CustomerEmail:      req.Email,
		CustomerPhone:      req.Phone,
		PurchaseDate:       &purchaseDate,
		PurchaseStore:      req.PurchaseStore,
		// Customer address fields
		Address:            req.Address,
		CountryCode:        req.CountryCode,
		ProvinceID:         req.ProvinceID,
		CityID:             req.CityID,
		// Other fields
		WarrantyExpiryDate: &warrantyExpiry,
		IPAddress:          c.ClientIP(),
		Geolocation:        geolocation,
		ActivationData:     datatypes.JSON(activationDataJSON),
	}

	if err := h.DB.Create(&warranty).Error; err != nil {
		// Check for unique constraint violation (race condition protection)
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "uniq_warranty_activations_qr_code") {
			utils.ErrorResponse(c, http.StatusConflict, "Warranty has already been registered for this product", nil)
			return
		}
		sentry.CaptureHandlerError(c, err, "warranty.RegisterWarranty", sentry.ErrorTypeDatabase, sentry.SeverityHigh)
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to register warranty", err)
		return
	}

	// Record interaction. Only attach geolocation when GPS coords are present, using
	// the {"lat","lng"} shape. The impossible-travel velocity check (velocity.go) and
	// the heatmap (dashboard.go) parse ONLY that shape, and the velocity query keys
	// off `geolocation IS NOT NULL` — writing a coordinate-less non-null blob here
	// would break those readers AND mask the anti-counterfeit check by shadowing the
	// previous geolocated scan with a 0,0 position.
	var interactionGeolocation datatypes.JSON
	if req.Latitude != nil && req.Longitude != nil {
		interactionGeolocationData, _ := json.Marshal(map[string]float64{
			"lat": *req.Latitude,
			"lng": *req.Longitude,
		})
		interactionGeolocation = datatypes.JSON(interactionGeolocationData)
	}

	interaction := models.Interaction{
		QRCodeID:               &qrCode.ID,
		TenantID:               qrCode.Batch.TenantID,
		InteractionCategory:    models.InteractionCategoryEndUserAccess,
		InteractionSubcategory: models.InteractionSubcategoryWarrantyActivation,
		InteractionStatus:      models.InteractionStatusSuccess,
		IPAddress:              c.ClientIP(),
		UserAgent:              c.GetHeader("User-Agent"),
		Geolocation:            interactionGeolocation,
	}
	h.DB.Create(&interaction)

	// Reverse-geocode enrichment (adds country/province/city so warranty activations
	// appear in the heatmap's geographic aggregations, mirroring UpdateScanLocation).
	if h.Cfg.Geocoding.BigDataCloudAPIKey != "" && req.Latitude != nil && req.Longitude != nil {
		go enrichGeolocation(h.DB, h.Cfg.Geocoding.BigDataCloudAPIKey, interaction.ID, *req.Latitude, *req.Longitude)
	}

	// Non-blocking geofence check (records violation for awareness, does not block registration)
	if req.Latitude != nil && req.Longitude != nil && qrCode.Batch != nil {
		go checkBatchGeofence(h.DB, h.Cfg, interaction.ID, qrCode.Batch.ID, qrCode.ID, nil, *req.Latitude, *req.Longitude, 0)
	}

	// Stream the registration to the company's systems (webhook is opt-in).
	{
		productName := "Product"
		if qrCode.Batch != nil && qrCode.Batch.Product != nil {
			productName = qrCode.Batch.Product.ProductName
		}
		utils.SendWebhook(h.DB, qrCode.Batch.TenantID, "warranty_registered", map[string]interface{}{
			"event":         "warranty_registered",
			"product_name":  productName,
			"qr_code":       code,
			"customer_name": req.CustomerName,
			"email":         req.Email,
			"phone":         req.Phone,
			"purchase_date": req.PurchaseDate,
			"expires_at":    warrantyExpiry.Format("2006-01-02"),
		})
	}

	utils.SuccessResponse(c, http.StatusOK, "Warranty registered successfully", WarrantyRegistrationResult{
		Success:            true,
		Message:            "Warranty has been successfully activated",
		AlreadyRegistered:  false,
		WarrantyExpiryDate: &warrantyExpiry,
		BatchID:            qrCode.Batch.ID.String(),
	})
}

// GetWarrantyStatus checks warranty status for a QR code
func (h *WarrantyHandler) GetWarrantyStatus(c *gin.Context) {
	code := c.Param("code")

	// Parse QR code parameter (supports Base58, UUID, hex formats)
	lookup, err := utils.ParseQRCodeParam(code)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	// Find QR code with batch info
	var qrCode models.QRCode
	var lookupErr error
	if lookup.LookupByCode {
		lookupErr = h.DB.Preload("Batch.Product").Preload("Batch.Tenant").First(&qrCode, "qr_code = ?", lookup.OriginalCode).Error
	} else {
		lookupErr = h.DB.Preload("Batch.Product").Preload("Batch.Tenant").First(&qrCode, "qr_uuid = ?", lookup.QRUUID).Error
	}
	if lookupErr != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Product not found", lookupErr)
		return
	}

	// Check if warranty already activated
	var existingWarranty models.WarrantyActivation
	warrantyRegistered := h.DB.Where("qr_code_id = ?", qrCode.ID).First(&existingWarranty).Error == nil

	// Get warranty fields config from product
	var warrantyFieldsConfig WarrantyFieldsConfig
	if qrCode.Batch != nil && qrCode.Batch.Product != nil && qrCode.Batch.Product.WarrantyFieldsConfig != nil {
		// Try to parse the new format first
		if err := json.Unmarshal(qrCode.Batch.Product.WarrantyFieldsConfig, &warrantyFieldsConfig); err != nil || warrantyFieldsConfig.Fields == nil {
			// Fall back to default if parsing fails or old format
			warrantyFieldsConfig = getDefaultWarrantyFieldsConfig()
		}
	} else {
		warrantyFieldsConfig = getDefaultWarrantyFieldsConfig()
	}

	// Prepare product info
	var productInfo gin.H
	if qrCode.Batch != nil && qrCode.Batch.Product != nil {
		productInfo = gin.H{
			"product_name": qrCode.Batch.Product.ProductName,
			"product_code": qrCode.Batch.Product.ProductCode,
			"description":  qrCode.Batch.Product.Description,
		}
	}

	// Prepare tenant info
	var tenantInfo gin.H
	if qrCode.Batch != nil && qrCode.Batch.Tenant != nil {
		tenantInfo = gin.H{
			"company_name": qrCode.Batch.Tenant.CompanyName,
			"brand_name":   qrCode.Batch.Tenant.CompanyName, // Use company name as brand
			"logo_url":     qrCode.Batch.LogoURL,            // Logo from batch
		}
	}

	warrantyMonths := 12
	if qrCode.Batch != nil && qrCode.Batch.Product != nil && qrCode.Batch.Product.WarrantyMonths > 0 {
		warrantyMonths = qrCode.Batch.Product.WarrantyMonths
	}

	result := gin.H{
		"is_valid":               qrCode.Status == models.QRCodeStatusActive,
		"need_warranty":          qrCode.Batch != nil && qrCode.Batch.Product != nil && qrCode.Batch.Product.WarrantyEnabled,
		"warranty_registered":    warrantyRegistered,
		"warranty_months":        warrantyMonths,
		"warranty_fields_config": warrantyFieldsConfig,
		"product":                productInfo,
		"tenant":                 tenantInfo,
	}

	if warrantyRegistered {
		// Only the (non-PII) expiry date is exposed publicly. Registrant PII —
		// customer_name, purchase_store, purchase_date, activated_at — is deliberately
		// NOT returned here: this endpoint is unauthenticated and the QR code is
		// printed on the physical product label, so anyone who scans or enumerates a
		// code could otherwise harvest the buyer's name and purchase location. The
		// registrant sees their own details on the post-registration confirmation
		// (from the data they just submitted); tenant admins retrieve full details via
		// the authenticated, tenant-scoped /warranties endpoints.
		result["warranty_expiry"] = existingWarranty.WarrantyExpiryDate
	}

	utils.SuccessResponse(c, http.StatusOK, "Warranty status", result)
}
