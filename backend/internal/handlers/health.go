package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gamatritunggal/smartscan/backend/internal/utils"
)

func HealthCheck(c *gin.Context) {
	utils.SuccessResponse(c, http.StatusOK, "Smart Label API is running", gin.H{
		"status":  "healthy",
		"version": "1.0.0",
	})
}
