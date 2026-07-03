package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gamatritunggal/smartscan/backend/internal/config"
	"github.com/gamatritunggal/smartscan/backend/internal/models"
	"github.com/gamatritunggal/smartscan/backend/internal/utils"
	"gorm.io/gorm"
)

type TenantLocationHandler struct {
	DB  *gorm.DB
	Cfg *config.Config
}

func NewTenantLocationHandler(db *gorm.DB, cfg *config.Config) *TenantLocationHandler {
	return &TenantLocationHandler{DB: db, Cfg: cfg}
}

// GetLocations returns all locations for a tenant
func (h *TenantLocationHandler) GetLocations(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}

	// Query parameters
	locationType := c.Query("type") // warehouse, qc_area, etc.
	status := c.DefaultQuery("status", "active")

	query := h.DB.Where("tenant_id = ?", tenantUUID)

	// Filter by type
	if locationType != "" {
		query = query.Where("location_type = ?", locationType)
	}

	// Filter by status
	switch status {
	case "deleted":
		query = query.Where("deleted_at IS NOT NULL")
	case "all":
		// No filter
	default: // "active"
		query = query.Where("deleted_at IS NULL AND status = ?", "active")
	}

	var locations []models.TenantLocation
	query.Order("location_type, location_name").Find(&locations)

	utils.SuccessResponse(c, http.StatusOK, "Locations retrieved", locations)
}

// GetLocation returns a single location
func (h *TenantLocationHandler) GetLocation(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}
	locationID := c.Param("id")

	var location models.TenantLocation
	if err := h.DB.Where("id = ? AND tenant_id = ?", locationID, tenantUUID).First(&location).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Location not found", nil)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Location retrieved", location)
}

// CreateLocation creates a new location
func (h *TenantLocationHandler) CreateLocation(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}

	var input struct {
		LocationName  string   `json:"location_name" binding:"required"`
		LocationType  string   `json:"location_type" binding:"required,oneof=warehouse qc_area production office"`
		Address       string   `json:"address"`
		City          string   `json:"city"`
		Province      string   `json:"province"`
		PostalCode    string   `json:"postal_code"`
		PhoneNumber   string   `json:"phone_number"`
		Latitude      *float64 `json:"latitude"`
		Longitude     *float64 `json:"longitude"`
		AllowedRadius *int     `json:"allowed_radius"` // meters
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request", err)
		return
	}

	// Build geolocation JSON if coordinates provided (use pointers to allow 0,0)
	var geolocation []byte
	if input.Latitude != nil && input.Longitude != nil {
		geolocation = []byte(fmt.Sprintf(`{"lat":%f,"lng":%f}`, *input.Latitude, *input.Longitude))
	}

	location := models.TenantLocation{
		TenantID:      tenantUUID,
		LocationName:  input.LocationName,
		LocationType:  models.LocationType(input.LocationType),
		Address:       input.Address,
		City:          input.City,
		Province:      input.Province,
		PostalCode:    input.PostalCode,
		PhoneNumber:   input.PhoneNumber,
		Geolocation:   geolocation,
		AllowedRadius: input.AllowedRadius,
		Status:        "active",
	}

	if err := h.DB.Create(&location).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to create location", err)
		return
	}

	utils.SuccessResponse(c, http.StatusCreated, "Location created", location)
}

// UpdateLocation updates an existing location
func (h *TenantLocationHandler) UpdateLocation(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}
	locationID := c.Param("id")

	var location models.TenantLocation
	if err := h.DB.Where("id = ? AND tenant_id = ? AND deleted_at IS NULL", locationID, tenantUUID).First(&location).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Location not found", nil)
		return
	}

	var input struct {
		LocationName  string   `json:"location_name"`
		LocationType  string   `json:"location_type" binding:"omitempty,oneof=warehouse qc_area production office"`
		Address       *string  `json:"address"`
		City          *string  `json:"city"`
		Province      *string  `json:"province"`
		PostalCode    *string  `json:"postal_code"`
		PhoneNumber   *string  `json:"phone_number"`
		Latitude      *float64 `json:"latitude"`
		Longitude     *float64 `json:"longitude"`
		AllowedRadius *int     `json:"allowed_radius"`
		Status        string   `json:"status" binding:"omitempty,oneof=active inactive"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request", err)
		return
	}

	// Build updates map
	updates := map[string]interface{}{}
	if input.LocationName != "" {
		updates["location_name"] = input.LocationName
	}
	if input.LocationType != "" {
		updates["location_type"] = models.LocationType(input.LocationType)
	}
	if input.Address != nil {
		updates["address"] = *input.Address
	}
	if input.City != nil {
		updates["city"] = *input.City
	}
	if input.Province != nil {
		updates["province"] = *input.Province
	}
	if input.PostalCode != nil {
		updates["postal_code"] = *input.PostalCode
	}
	if input.PhoneNumber != nil {
		updates["phone_number"] = *input.PhoneNumber
	}
	if input.Status != "" {
		updates["status"] = input.Status
	}
	if input.AllowedRadius != nil {
		updates["allowed_radius"] = *input.AllowedRadius
	}

	// Update geolocation if coordinates provided (use pointers to allow 0,0)
	if input.Latitude != nil && input.Longitude != nil {
		updates["geolocation"] = []byte(fmt.Sprintf(`{"lat":%f,"lng":%f}`, *input.Latitude, *input.Longitude))
	}

	if len(updates) > 0 {
		if err := h.DB.Model(&location).Updates(updates).Error; err != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to update location", err)
			return
		}
	}

	// Re-fetch for response
	h.DB.Where("id = ? AND tenant_id = ?", locationID, tenantUUID).First(&location)

	utils.SuccessResponse(c, http.StatusOK, "Location updated", location)
}

// DeleteLocation soft deletes a location
func (h *TenantLocationHandler) DeleteLocation(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}
	locationID := c.Param("id")

	var location models.TenantLocation
	if err := h.DB.Where("id = ? AND tenant_id = ? AND deleted_at IS NULL", locationID, tenantUUID).First(&location).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Location not found", nil)
		return
	}

	now := time.Now().UTC()

	if err := h.DB.Model(&location).Update("deleted_at", now).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to delete location", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Location deleted", nil)
}

// RestoreLocation restores a soft-deleted location
func (h *TenantLocationHandler) RestoreLocation(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}
	locationID := c.Param("id")

	var location models.TenantLocation
	if err := h.DB.Where("id = ? AND tenant_id = ? AND deleted_at IS NOT NULL", locationID, tenantUUID).First(&location).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Deleted location not found", nil)
		return
	}

	if err := h.DB.Model(&location).Update("deleted_at", nil).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to restore location", err)
		return
	}

	location.DeletedAt = nil
	utils.SuccessResponse(c, http.StatusOK, "Location restored", location)
}

// GetLocationsByType returns locations filtered by type (for dropdowns)
func (h *TenantLocationHandler) GetLocationsByType(c *gin.Context) {
	tenantUUID, ok := utils.GetTenantUUID(c)
	if !ok {
		utils.ErrorResponse(c, http.StatusForbidden, "Invalid tenant context", nil)
		return
	}
	locationType := c.Param("type")

	var locations []models.TenantLocation
	h.DB.Where("tenant_id = ? AND location_type = ? AND status = ? AND deleted_at IS NULL", tenantUUID, locationType, "active").
		Order("location_name").
		Find(&locations)

	utils.SuccessResponse(c, http.StatusOK, "Locations retrieved", locations)
}
