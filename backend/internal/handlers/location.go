package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gamatritunggal/smartscan/backend/internal/config"
	"github.com/gamatritunggal/smartscan/backend/internal/models"
	"github.com/gamatritunggal/smartscan/backend/internal/utils"
	"gorm.io/gorm"
)

type LocationHandler struct {
	DB  *gorm.DB
	Cfg *config.Config
}

func NewLocationHandler(db *gorm.DB, cfg *config.Config) *LocationHandler {
	return &LocationHandler{DB: db, Cfg: cfg}
}

// GetCountries returns all countries
func (h *LocationHandler) GetCountries(c *gin.Context) {
	var countries []models.Country
	h.DB.Order("name").Find(&countries)

	utils.SuccessResponse(c, http.StatusOK, "Countries retrieved", countries)
}

// GetProvinces returns provinces for a country
func (h *LocationHandler) GetProvinces(c *gin.Context) {
	countryCode := c.Param("country_code")

	var provinces []models.Province
	h.DB.Where("country_code = ?", countryCode).Order("name").Find(&provinces)

	utils.SuccessResponse(c, http.StatusOK, "Provinces retrieved", provinces)
}

// GetCities returns cities for a province
func (h *LocationHandler) GetCities(c *gin.Context) {
	provinceIDStr := c.Param("province_id")
	provinceID, err := strconv.Atoi(provinceIDStr)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid province ID", err)
		return
	}

	var cities []models.City
	h.DB.Where("province_id = ?", provinceID).Order("name").Find(&cities)

	utils.SuccessResponse(c, http.StatusOK, "Cities retrieved", cities)
}
