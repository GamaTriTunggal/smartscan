package utils

import (
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// isDevMode checks if running in development mode
func isDevMode() bool {
	env := os.Getenv("APP_ENV")
	return env == "" || env == "development" || env == "dev"
}

// MaxPaginationLimit is the maximum allowed pagination limit
// Prevents denial of service by limiting records per request
const MaxPaginationLimit = 100

// GetIntQuery gets an integer query parameter with a default value
func GetIntQuery(c *gin.Context, key string, defaultVal int) int {
	val := c.Query(key)
	if val == "" {
		return defaultVal
	}
	intVal, err := strconv.Atoi(val)
	if err != nil {
		return defaultVal
	}
	return intVal
}

// GetLimitQuery gets pagination limit with maximum cap (100)
// This prevents DoS attacks by limiting the number of records returned
func GetLimitQuery(c *gin.Context, defaultVal int) int {
	val := c.Query("limit")
	if val == "" {
		if defaultVal > MaxPaginationLimit {
			return MaxPaginationLimit
		}
		return defaultVal
	}
	limit, err := strconv.Atoi(val)
	if err != nil || limit < 1 {
		return defaultVal
	}
	if limit > MaxPaginationLimit {
		return MaxPaginationLimit
	}
	return limit
}

// GetPageQuery gets pagination page number (minimum 1)
func GetPageQuery(c *gin.Context) int {
	val := c.Query("page")
	if val == "" {
		return 1
	}
	page, err := strconv.Atoi(val)
	if err != nil || page < 1 {
		return 1
	}
	return page
}

// GetTenantID safely retrieves tenant_id from gin context
// Returns the tenant ID and true if found, or zero UUID and false if not found
func GetTenantID(c *gin.Context) (interface{}, bool) {
	tenantID, exists := c.Get("tenant_id")
	if !exists || tenantID == nil {
		return nil, false
	}
	return tenantID, true
}

// RequireTenantID retrieves tenant_id from gin context or returns an error response
// Use this for handlers that require a valid tenant context
func RequireTenantID(c *gin.Context) (interface{}, bool) {
	tenantID, exists := GetTenantID(c)
	if !exists {
		ErrorResponse(c, 403, "Tenant context required", nil)
		return nil, false
	}
	return tenantID, true
}

// GetTenantUUID safely retrieves and parses tenant_id from gin context as uuid.UUID
// The middleware stores tenant_id as string, so this function handles the conversion
// Returns the tenant UUID and true if successful, or zero UUID and false if not found/invalid
func GetTenantUUID(c *gin.Context) (uuid.UUID, bool) {
	tenantID, exists := c.Get("tenant_id")
	if !exists || tenantID == nil {
		return uuid.Nil, false
	}

	// Handle both string and uuid.UUID types
	switch v := tenantID.(type) {
	case string:
		parsed, err := uuid.Parse(v)
		if err != nil {
			return uuid.Nil, false
		}
		return parsed, true
	case uuid.UUID:
		return v, true
	default:
		return uuid.Nil, false
	}
}

// RequireTenantUUID retrieves tenant_id from gin context as uuid.UUID or returns an error response
// Use this for handlers that require a valid tenant context with UUID type
func RequireTenantUUID(c *gin.Context) (uuid.UUID, bool) {
	tenantUUID, ok := GetTenantUUID(c)
	if !ok {
		ErrorResponse(c, 403, "Tenant context required", nil)
		return uuid.Nil, false
	}
	return tenantUUID, true
}

// GetUserUUID safely retrieves and parses user_id from gin context as uuid.UUID
// Returns the user UUID and true if successful, or zero UUID and false if not found/invalid
func GetUserUUID(c *gin.Context) (uuid.UUID, bool) {
	userID, exists := c.Get("user_id")
	if !exists || userID == nil {
		return uuid.Nil, false
	}

	switch v := userID.(type) {
	case string:
		parsed, err := uuid.Parse(v)
		if err != nil {
			return uuid.Nil, false
		}
		return parsed, true
	case uuid.UUID:
		return v, true
	default:
		return uuid.Nil, false
	}
}

// GetStaffUUID safely retrieves and parses staff_id from gin context as uuid.UUID
// The middleware stores staff_id as string, so this function handles the conversion
// Returns the staff UUID and true if successful, or zero UUID and false if not found/invalid
func GetStaffUUID(c *gin.Context) (uuid.UUID, bool) {
	staffID, exists := c.Get("staff_id")
	if !exists || staffID == nil {
		return uuid.Nil, false
	}

	// Handle both string and uuid.UUID types
	switch v := staffID.(type) {
	case string:
		parsed, err := uuid.Parse(v)
		if err != nil {
			return uuid.Nil, false
		}
		return parsed, true
	case uuid.UUID:
		return v, true
	default:
		return uuid.Nil, false
	}
}

type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func SuccessResponse(c *gin.Context, statusCode int, message string, data interface{}) {
	c.JSON(statusCode, APIResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

func ErrorResponse(c *gin.Context, statusCode int, message string, err error) {
	// Always log the full error internally for debugging
	if err != nil {
		log.Printf("[ERROR] %s: %v (path: %s, method: %s)", message, err, c.Request.URL.Path, c.Request.Method)
	}

	// Only expose error details in development mode
	errMsg := ""
	if err != nil && isDevMode() {
		errMsg = err.Error()
	}

	c.JSON(statusCode, APIResponse{
		Success: false,
		Message: message,
		Error:   errMsg,
	})
}

// ErrorResponseWithCode returns an error response with a specific error code
// Useful for frontend to handle specific error types (e.g., INACTIVITY_TIMEOUT)
func ErrorResponseWithCode(c *gin.Context, statusCode int, code string, message string) {
	c.JSON(statusCode, gin.H{
		"success": false,
		"code":    code,
		"message": message,
	})
}

func ValidationErrorResponse(c *gin.Context, errors map[string]string) {
	c.JSON(http.StatusBadRequest, gin.H{
		"success": false,
		"message": "Validation failed",
		"errors":  errors,
	})
}
