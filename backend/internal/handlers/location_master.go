package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gamatritunggal/smartscan/backend/internal/config"
	"github.com/gamatritunggal/smartscan/backend/internal/models"
	"github.com/gamatritunggal/smartscan/backend/internal/sentry"
	"github.com/gamatritunggal/smartscan/backend/internal/utils"
	"gorm.io/gorm"
)

type LocationMasterHandler struct {
	DB  *gorm.DB
	Cfg *config.Config
}

func NewLocationMasterHandler(db *gorm.DB, cfg *config.Config) *LocationMasterHandler {
	return &LocationMasterHandler{DB: db, Cfg: cfg}
}

// ==================== COUNTRIES ====================

// ListCountries returns all countries
func (h *LocationMasterHandler) ListCountries(c *gin.Context) {
	page := utils.GetPageQuery(c)
	limit := utils.GetLimitQuery(c, 50)
	search := c.Query("search")
	status := c.DefaultQuery("status", "active")

	offset := (page - 1) * limit

	var countries []models.Country
	var total int64

	query := h.DB.Model(&models.Country{})

	// Status filter
	switch status {
	case "deleted":
		query = query.Where("deleted_at IS NOT NULL")
	case "all":
		// No filter - show all
	default: // active
		query = query.Where("deleted_at IS NULL")
	}

	if search != "" {
		query = query.Where("name ILIKE ? OR code ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	query.Count(&total)
	query.Order("name ASC").Offset(offset).Limit(limit).Find(&countries)

	utils.SuccessResponse(c, http.StatusOK, "Countries retrieved", gin.H{
		"countries": countries,
		"pagination": gin.H{
			"page":       page,
			"limit":      limit,
			"total":      total,
			"total_page": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// GetCountry returns a single country
func (h *LocationMasterHandler) GetCountry(c *gin.Context) {
	code := c.Param("code")

	var country models.Country
	if err := h.DB.First(&country, "code = ?", code).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Country not found", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Country retrieved", country)
}

// CreateCountry creates a new country
func (h *LocationMasterHandler) CreateCountry(c *gin.Context) {
	var req struct {
		Code      string `json:"code" binding:"required,len=2"`
		Name      string `json:"name" binding:"required"`
		PhoneCode string `json:"phone_code"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request", err)
		return
	}

	// Check for duplicate code
	var existing models.Country
	if err := h.DB.Where("code = ?", req.Code).First(&existing).Error; err == nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Country code already exists", nil)
		return
	}

	country := models.Country{
		Code:      req.Code,
		Name:      req.Name,
		PhoneCode: req.PhoneCode,
	}

	if err := h.DB.Create(&country).Error; err != nil {
		sentry.CaptureHandlerError(c, err, "locationMaster.CreateCountry", sentry.ErrorTypeDatabase, sentry.SeverityLow)
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to create country", err)
		return
	}

	utils.SuccessResponse(c, http.StatusCreated, "Country created", country)
}

// UpdateCountry updates a country
func (h *LocationMasterHandler) UpdateCountry(c *gin.Context) {
	code := c.Param("code")

	var country models.Country
	if err := h.DB.First(&country, "code = ?", code).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Country not found", err)
		return
	}

	var req struct {
		Name      string `json:"name"`
		PhoneCode string `json:"phone_code"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request", err)
		return
	}

	updates := map[string]interface{}{}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.PhoneCode != "" {
		updates["phone_code"] = req.PhoneCode
	}

	if len(updates) > 0 {
		if err := h.DB.Model(&country).Updates(updates).Error; err != nil {
			sentry.CaptureHandlerError(c, err, "locationMaster.UpdateCountry", sentry.ErrorTypeDatabase, sentry.SeverityLow)
			utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to update country", err)
			return
		}
	}

	// Refresh for response
	h.DB.First(&country, "code = ?", code)

	utils.SuccessResponse(c, http.StatusOK, "Country updated", country)
}

// DeleteCountry soft deletes a country
func (h *LocationMasterHandler) DeleteCountry(c *gin.Context) {
	code := c.Param("code")

	var country models.Country
	if err := h.DB.First(&country, "code = ?", code).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Country not found", err)
		return
	}

	// Check if there are active provinces referencing this country
	var provinceCount int64
	h.DB.Model(&models.Province{}).Where("country_code = ? AND deleted_at IS NULL", code).Count(&provinceCount)
	if provinceCount > 0 {
		utils.ErrorResponse(c, http.StatusBadRequest, "Cannot delete country with active provinces. Delete provinces first.", nil)
		return
	}

	now := time.Now().UTC()

	if err := h.DB.Model(&country).Update("deleted_at", now).Error; err != nil {
		sentry.CaptureHandlerError(c, err, "locationMaster.DeleteCountry", sentry.ErrorTypeDatabase, sentry.SeverityLow)
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to delete country", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Country deleted", nil)
}

// RestoreCountry restores a soft-deleted country
func (h *LocationMasterHandler) RestoreCountry(c *gin.Context) {
	code := c.Param("code")

	var country models.Country
	if err := h.DB.Where("code = ?", code).First(&country).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Country not found", err)
		return
	}

	if err := h.DB.Model(&country).Update("deleted_at", nil).Error; err != nil {
		sentry.CaptureHandlerError(c, err, "locationMaster.RestoreCountry", sentry.ErrorTypeDatabase, sentry.SeverityLow)
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to restore country", err)
		return
	}

	country.DeletedAt = nil
	utils.SuccessResponse(c, http.StatusOK, "Country restored", country)
}

// ==================== PROVINCES ====================

// ListProvinces returns all provinces
func (h *LocationMasterHandler) ListProvinces(c *gin.Context) {
	page := utils.GetPageQuery(c)
	limit := utils.GetLimitQuery(c, 50)
	search := c.Query("search")
	countryCode := c.Query("country_code")
	status := c.DefaultQuery("status", "active")

	offset := (page - 1) * limit

	var provinces []models.Province
	var total int64

	query := h.DB.Model(&models.Province{})

	// Status filter
	switch status {
	case "deleted":
		query = query.Where("provinces.deleted_at IS NOT NULL")
	case "all":
		// No filter - show all
	default: // active
		query = query.Where("provinces.deleted_at IS NULL")
	}

	if search != "" {
		query = query.Where("provinces.name ILIKE ? OR provinces.code ILIKE ?", "%"+search+"%", "%"+search+"%")
	}
	if countryCode != "" {
		query = query.Where("provinces.country_code = ?", countryCode)
	}

	query.Count(&total)
	query.Preload("Country").Order("provinces.name ASC").Offset(offset).Limit(limit).Find(&provinces)

	utils.SuccessResponse(c, http.StatusOK, "Provinces retrieved", gin.H{
		"provinces": provinces,
		"pagination": gin.H{
			"page":       page,
			"limit":      limit,
			"total":      total,
			"total_page": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// GetProvince returns a single province
func (h *LocationMasterHandler) GetProvince(c *gin.Context) {
	id := c.Param("id")
	provinceID, err := strconv.Atoi(id)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid province ID", err)
		return
	}

	var province models.Province
	if err := h.DB.Preload("Country").First(&province, "id = ?", provinceID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Province not found", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Province retrieved", province)
}

// CreateProvince creates a new province
func (h *LocationMasterHandler) CreateProvince(c *gin.Context) {
	var req struct {
		CountryCode string `json:"country_code" binding:"required,len=2"`
		Name        string `json:"name" binding:"required"`
		Code        string `json:"code"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request", err)
		return
	}

	// Verify country exists and is active
	var country models.Country
	if err := h.DB.Where("code = ? AND deleted_at IS NULL", req.CountryCode).First(&country).Error; err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Country not found or inactive", err)
		return
	}

	province := models.Province{
		CountryCode: req.CountryCode,
		Name:        req.Name,
		Code:        req.Code,
	}

	if err := h.DB.Create(&province).Error; err != nil {
		sentry.CaptureHandlerError(c, err, "locationMaster.CreateProvince", sentry.ErrorTypeDatabase, sentry.SeverityLow)
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to create province", err)
		return
	}

	// Load relation
	h.DB.Preload("Country").First(&province, "id = ?", province.ID)

	utils.SuccessResponse(c, http.StatusCreated, "Province created", province)
}

// UpdateProvince updates a province
func (h *LocationMasterHandler) UpdateProvince(c *gin.Context) {
	id := c.Param("id")
	provinceID, err := strconv.Atoi(id)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid province ID", err)
		return
	}

	var province models.Province
	if err := h.DB.First(&province, "id = ?", provinceID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Province not found", err)
		return
	}

	var req struct {
		CountryCode string `json:"country_code"`
		Name        string `json:"name"`
		Code        string `json:"code"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request", err)
		return
	}

	updates := map[string]interface{}{}
	if req.CountryCode != "" && req.CountryCode != province.CountryCode {
		// Verify new country exists and is active
		var country models.Country
		if err := h.DB.Where("code = ? AND deleted_at IS NULL", req.CountryCode).First(&country).Error; err != nil {
			utils.ErrorResponse(c, http.StatusBadRequest, "Country not found or inactive", err)
			return
		}
		updates["country_code"] = req.CountryCode
	}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Code != "" {
		updates["code"] = req.Code
	}

	if len(updates) > 0 {
		if err := h.DB.Model(&province).Updates(updates).Error; err != nil {
			sentry.CaptureHandlerError(c, err, "locationMaster.UpdateProvince", sentry.ErrorTypeDatabase, sentry.SeverityLow)
			utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to update province", err)
			return
		}
	}

	h.DB.Preload("Country").First(&province, "id = ?", province.ID)

	utils.SuccessResponse(c, http.StatusOK, "Province updated", province)
}

// DeleteProvince soft deletes a province
func (h *LocationMasterHandler) DeleteProvince(c *gin.Context) {
	id := c.Param("id")
	provinceID, err := strconv.Atoi(id)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid province ID", err)
		return
	}

	var province models.Province
	if err := h.DB.First(&province, "id = ?", provinceID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Province not found", err)
		return
	}

	// Check if there are active cities referencing this province
	var cityCount int64
	h.DB.Model(&models.City{}).Where("province_id = ? AND deleted_at IS NULL", provinceID).Count(&cityCount)
	if cityCount > 0 {
		utils.ErrorResponse(c, http.StatusBadRequest, "Cannot delete province with active cities. Delete cities first.", nil)
		return
	}

	now := time.Now().UTC()

	if err := h.DB.Model(&province).Update("deleted_at", now).Error; err != nil {
		sentry.CaptureHandlerError(c, err, "locationMaster.DeleteProvince", sentry.ErrorTypeDatabase, sentry.SeverityLow)
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to delete province", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Province deleted", nil)
}

// RestoreProvince restores a soft-deleted province
func (h *LocationMasterHandler) RestoreProvince(c *gin.Context) {
	id := c.Param("id")
	provinceID, err := strconv.Atoi(id)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid province ID", err)
		return
	}

	var province models.Province
	if err := h.DB.Where("id = ?", provinceID).First(&province).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Province not found", err)
		return
	}

	// Check if parent country is active
	var country models.Country
	if err := h.DB.Where("code = ? AND deleted_at IS NULL", province.CountryCode).First(&country).Error; err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Cannot restore province - parent country is deleted", err)
		return
	}

	if err := h.DB.Model(&province).Update("deleted_at", nil).Error; err != nil {
		sentry.CaptureHandlerError(c, err, "locationMaster.RestoreProvince", sentry.ErrorTypeDatabase, sentry.SeverityLow)
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to restore province", err)
		return
	}

	h.DB.Preload("Country").First(&province, "id = ?", province.ID)

	utils.SuccessResponse(c, http.StatusOK, "Province restored", province)
}

// ==================== CITIES ====================

// ListCities returns all cities
func (h *LocationMasterHandler) ListCities(c *gin.Context) {
	page := utils.GetPageQuery(c)
	limit := utils.GetLimitQuery(c, 50)
	search := c.Query("search")
	provinceID := c.Query("province_id")
	countryCode := c.Query("country_code")
	status := c.DefaultQuery("status", "active")

	offset := (page - 1) * limit

	var cities []models.City
	var total int64

	query := h.DB.Model(&models.City{})

	// Status filter
	switch status {
	case "deleted":
		query = query.Where("cities.deleted_at IS NOT NULL")
	case "all":
		// No filter - show all
	default: // active
		query = query.Where("cities.deleted_at IS NULL")
	}

	if search != "" {
		query = query.Where("cities.name ILIKE ?", "%"+search+"%")
	}
	if provinceID != "" {
		query = query.Where("cities.province_id = ?", provinceID)
	}
	if countryCode != "" {
		query = query.Where("cities.country_code = ?", countryCode)
	}

	query.Count(&total)
	query.Preload("Province").Preload("Country").Order("cities.name ASC").Offset(offset).Limit(limit).Find(&cities)

	utils.SuccessResponse(c, http.StatusOK, "Cities retrieved", gin.H{
		"cities": cities,
		"pagination": gin.H{
			"page":       page,
			"limit":      limit,
			"total":      total,
			"total_page": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// GetCity returns a single city
func (h *LocationMasterHandler) GetCity(c *gin.Context) {
	id := c.Param("id")
	cityID, err := strconv.Atoi(id)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid city ID", err)
		return
	}

	var city models.City
	if err := h.DB.Preload("Province").Preload("Country").First(&city, "id = ?", cityID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "City not found", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "City retrieved", city)
}

// CreateCity creates a new city
func (h *LocationMasterHandler) CreateCity(c *gin.Context) {
	var req struct {
		ProvinceID       int    `json:"province_id" binding:"required"`
		Name             string `json:"name" binding:"required"`
		PostalCodePrefix string `json:"postal_code_prefix"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request", err)
		return
	}

	// Verify province exists and is active
	var province models.Province
	if err := h.DB.Where("id = ? AND deleted_at IS NULL", req.ProvinceID).First(&province).Error; err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Province not found or inactive", err)
		return
	}

	city := models.City{
		ProvinceID:       req.ProvinceID,
		CountryCode:      province.CountryCode,
		Name:             req.Name,
		PostalCodePrefix: req.PostalCodePrefix,
	}

	if err := h.DB.Create(&city).Error; err != nil {
		sentry.CaptureHandlerError(c, err, "locationMaster.CreateCity", sentry.ErrorTypeDatabase, sentry.SeverityLow)
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to create city", err)
		return
	}

	// Load relations
	h.DB.Preload("Province").Preload("Country").First(&city, "id = ?", city.ID)

	utils.SuccessResponse(c, http.StatusCreated, "City created", city)
}

// UpdateCity updates a city
func (h *LocationMasterHandler) UpdateCity(c *gin.Context) {
	id := c.Param("id")
	cityID, err := strconv.Atoi(id)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid city ID", err)
		return
	}

	var city models.City
	if err := h.DB.First(&city, "id = ?", cityID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "City not found", err)
		return
	}

	var req struct {
		ProvinceID       *int   `json:"province_id"`
		Name             string `json:"name"`
		PostalCodePrefix string `json:"postal_code_prefix"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request", err)
		return
	}

	updates := map[string]interface{}{}
	if req.ProvinceID != nil && *req.ProvinceID != city.ProvinceID {
		// Verify new province exists and is active
		var province models.Province
		if err := h.DB.Where("id = ? AND deleted_at IS NULL", *req.ProvinceID).First(&province).Error; err != nil {
			utils.ErrorResponse(c, http.StatusBadRequest, "Province not found or inactive", err)
			return
		}
		updates["province_id"] = *req.ProvinceID
		updates["country_code"] = province.CountryCode
	}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.PostalCodePrefix != "" {
		updates["postal_code_prefix"] = req.PostalCodePrefix
	}

	if len(updates) > 0 {
		if err := h.DB.Model(&city).Updates(updates).Error; err != nil {
			sentry.CaptureHandlerError(c, err, "locationMaster.UpdateCity", sentry.ErrorTypeDatabase, sentry.SeverityLow)
			utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to update city", err)
			return
		}
	}

	h.DB.Preload("Province").Preload("Country").First(&city, "id = ?", city.ID)

	utils.SuccessResponse(c, http.StatusOK, "City updated", city)
}

// DeleteCity soft deletes a city
func (h *LocationMasterHandler) DeleteCity(c *gin.Context) {
	id := c.Param("id")
	cityID, err := strconv.Atoi(id)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid city ID", err)
		return
	}

	var city models.City
	if err := h.DB.First(&city, "id = ?", cityID).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "City not found", err)
		return
	}

	// Check if there are tenants referencing this city
	var tenantCount int64
	h.DB.Model(&models.Tenant{}).Where("city_id = ? AND deleted_at IS NULL", cityID).Count(&tenantCount)
	if tenantCount > 0 {
		utils.ErrorResponse(c, http.StatusBadRequest, "Cannot delete city - some tenants are using it", nil)
		return
	}

	now := time.Now().UTC()

	if err := h.DB.Model(&city).Update("deleted_at", now).Error; err != nil {
		sentry.CaptureHandlerError(c, err, "locationMaster.DeleteCity", sentry.ErrorTypeDatabase, sentry.SeverityLow)
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to delete city", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "City deleted", nil)
}

// RestoreCity restores a soft-deleted city
func (h *LocationMasterHandler) RestoreCity(c *gin.Context) {
	id := c.Param("id")
	cityID, err := strconv.Atoi(id)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid city ID", err)
		return
	}

	var city models.City
	if err := h.DB.Where("id = ?", cityID).First(&city).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "City not found", err)
		return
	}

	// Check if parent province is active
	var province models.Province
	if err := h.DB.Where("id = ? AND deleted_at IS NULL", city.ProvinceID).First(&province).Error; err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Cannot restore city - parent province is deleted", err)
		return
	}

	if err := h.DB.Model(&city).Update("deleted_at", nil).Error; err != nil {
		sentry.CaptureHandlerError(c, err, "locationMaster.RestoreCity", sentry.ErrorTypeDatabase, sentry.SeverityLow)
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to restore city", err)
		return
	}

	h.DB.Preload("Province").Preload("Country").First(&city, "id = ?", city.ID)

	utils.SuccessResponse(c, http.StatusOK, "City restored", city)
}
