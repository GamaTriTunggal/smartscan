package handlers

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gamatritunggal/smartscan/backend/internal/config"
	"github.com/gamatritunggal/smartscan/backend/internal/storage"
	"github.com/gamatritunggal/smartscan/backend/internal/utils"
	"gorm.io/gorm"
)

// isPathSafe checks if the resolved path is within the allowed base directory
// Prevents path traversal attacks (e.g., ../../etc/passwd)
func isPathSafe(basePath, requestedPath string) bool {
	// Clean and resolve the paths
	absBase, err := filepath.Abs(basePath)
	if err != nil {
		return false
	}
	absRequested, err := filepath.Abs(requestedPath)
	if err != nil {
		return false
	}

	// Check if requested path starts with base path
	return strings.HasPrefix(absRequested, absBase)
}

type UploadHandler struct {
	DB  *gorm.DB
	Cfg *config.Config
}

func NewUploadHandler(db *gorm.DB, cfg *config.Config) *UploadHandler {
	return &UploadHandler{DB: db, Cfg: cfg}
}

// UploadBackground handles background image upload for landing pages
// POST /tenant/uploads/background
func (h *UploadHandler) UploadBackground(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		utils.ErrorResponse(c, http.StatusUnauthorized, "Tenant ID not found", nil)
		return
	}

	// Only "landing" type is supported (campaign backgrounds removed)
	uploadType := c.DefaultPostForm("type", "landing")
	if uploadType != "landing" {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid upload type. Only 'landing' is supported", nil)
		return
	}

	// Get the file from form
	file, header, err := c.Request.FormFile("image")
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "No image file provided", err)
		return
	}
	defer file.Close()

	// Process and validate image using standard security checks
	processed, err := utils.ProcessUploadedImage(file, header, utils.DefaultUploadOptions())
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	// Generate unique filename
	filename := uuid.New().String() + processed.Extension

	// Storage path key
	storageKey := fmt.Sprintf("backgrounds/tenants/%s/%s/%s", tenantID, uploadType, filename)

	var urlPath string

	// Check if R2 storage is enabled
	r2Client := storage.GetGlobalR2Client()
	if r2Client != nil {
		// Upload to R2
		url, err := r2Client.Upload(context.Background(), storageKey, processed.Data, processed.ContentType)
		if err != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to upload image to storage", err)
			return
		}
		urlPath = url
	} else {
		// Fallback to local filesystem
		uploadDir := filepath.Join(h.Cfg.UploadPath, "backgrounds", "tenants", tenantID, uploadType)
		if err := os.MkdirAll(uploadDir, 0755); err != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to create upload directory", err)
			return
		}

		filePath := filepath.Join(uploadDir, filename)
		if err := os.WriteFile(filePath, processed.Data, 0644); err != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to save image file", err)
			return
		}
		urlPath = fmt.Sprintf("/uploads/%s", storageKey)
	}

	utils.SuccessResponse(c, http.StatusOK, "Background image uploaded successfully", gin.H{
		"url":      urlPath,
		"filename": filename,
		"type":     uploadType,
	})
}

// DeleteBackground deletes an uploaded background image
// DELETE /tenant/uploads/background/:filename
func (h *UploadHandler) DeleteBackground(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		utils.ErrorResponse(c, http.StatusUnauthorized, "Tenant ID not found", nil)
		return
	}

	filename := c.Param("filename")
	if filename == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "Filename is required", nil)
		return
	}

	// Only "landing" type is supported (campaign backgrounds removed)
	uploadType := c.DefaultQuery("type", "landing")
	if uploadType != "landing" {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid upload type. Only 'landing' is supported", nil)
		return
	}

	// Check if R2 storage is enabled
	r2Client := storage.GetGlobalR2Client()
	if r2Client != nil {
		// Delete from R2
		storageKey := fmt.Sprintf("backgrounds/tenants/%s/%s/%s", tenantID, uploadType, filename)
		if err := r2Client.Delete(context.Background(), storageKey); err != nil {
			// Log error but don't fail - file might not exist
			// utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to delete file from storage", err)
			// return
		}
	} else {
		// Fallback to local filesystem
		filePath := filepath.Join(h.Cfg.UploadPath, "backgrounds", "tenants", tenantID, uploadType, filename)

		// Check for path traversal attacks
		if !isPathSafe(filepath.Join(h.Cfg.UploadPath, "backgrounds"), filePath) {
			utils.ErrorResponse(c, http.StatusBadRequest, "Invalid filename", nil)
			return
		}

		// Check if file exists and belongs to this tenant
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			utils.ErrorResponse(c, http.StatusNotFound, "File not found", nil)
			return
		}

		// Delete the file
		if err := os.Remove(filePath); err != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to delete file", err)
			return
		}
	}

	utils.SuccessResponse(c, http.StatusOK, "Background image deleted successfully", nil)
}

// UploadPresetBackground handles background image upload for theme presets (admin)
// POST /tenant/theme-presets/upload
func (h *UploadHandler) UploadPresetBackground(c *gin.Context) {
	// Get the file from form
	file, header, err := c.Request.FormFile("image")
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "No image file provided", err)
		return
	}
	defer file.Close()

	// Process and validate image using standard security checks
	processed, err := utils.ProcessUploadedImage(file, header, utils.DefaultUploadOptions())
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	// Generate unique filename
	filename := uuid.New().String() + processed.Extension

	// Storage path key
	storageKey := fmt.Sprintf("backgrounds/presets/%s", filename)

	var urlPath string

	// Check if R2 storage is enabled
	r2Client := storage.GetGlobalR2Client()
	if r2Client != nil {
		// Upload to R2
		url, err := r2Client.Upload(context.Background(), storageKey, processed.Data, processed.ContentType)
		if err != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to upload image to storage", err)
			return
		}
		urlPath = url
	} else {
		// Fallback to local filesystem
		uploadDir := filepath.Join(h.Cfg.UploadPath, "backgrounds", "presets")
		if err := os.MkdirAll(uploadDir, 0755); err != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to create upload directory", err)
			return
		}

		filePath := filepath.Join(uploadDir, filename)
		if err := os.WriteFile(filePath, processed.Data, 0644); err != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to save image file", err)
			return
		}
		urlPath = fmt.Sprintf("/uploads/%s", storageKey)
	}

	utils.SuccessResponse(c, http.StatusOK, "Preset background image uploaded successfully", gin.H{
		"url":      urlPath,
		"filename": filename,
	})
}

// ServeBackgroundFile serves background images with proper access control
// GET /uploads/backgrounds/*filepath
//
// Access Control Logic:
// - /presets/*: Public access (for all users viewing validation pages)
// - /tenants/{tenant_id}/*:
//   - If requester is authenticated as that tenant: Allow
//   - If requester is authenticated as different tenant: Deny (403)
//   - If requester is not authenticated (public page): Allow (needed for validation page rendering)
//
// This prevents logged-in tenants from snooping on other tenants' backgrounds
// while still allowing public pages to display the backgrounds.
func (h *UploadHandler) ServeBackgroundFile(c *gin.Context) {
	// Get the filepath parameter (everything after /uploads/backgrounds/)
	filePath := c.Param("filepath")
	if filePath == "" {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	// Remove leading slash if present
	filePath = strings.TrimPrefix(filePath, "/")

	// Construct full file path
	fullPath := filepath.Join(h.Cfg.UploadPath, "backgrounds", filePath)

	// Security: Prevent path traversal attacks
	if !isPathSafe(filepath.Join(h.Cfg.UploadPath, "backgrounds"), fullPath) {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	// Check if file exists
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	// Tenant isolation check for tenant-specific backgrounds
	if strings.HasPrefix(filePath, "tenants/") {
		// Parse path: tenants/{tenant_id}/{type}/{filename}
		parts := strings.Split(filePath, "/")
		if len(parts) >= 2 {
			pathTenantID := parts[1]

			// Check if user is authenticated as a tenant
			requestingTenantID := c.GetString("tenant_id")

			if requestingTenantID != "" {
				// User is logged in as a tenant
				if requestingTenantID != pathTenantID {
					// Trying to access another tenant's background - deny
					c.AbortWithStatus(http.StatusForbidden)
					return
				}
				// Same tenant - allow
			}
			// If not logged in (requestingTenantID == ""), allow access
			// This is needed for public validation pages to render backgrounds
		}
	}
	// Preset backgrounds (/presets/*) are always publicly accessible

	// Determine content type based on extension
	ext := strings.ToLower(filepath.Ext(fullPath))
	contentType := "application/octet-stream"
	switch ext {
	case ".jpg", ".jpeg":
		contentType = "image/jpeg"
	case ".png":
		contentType = "image/png"
	case ".webp":
		contentType = "image/webp"
	case ".gif":
		contentType = "image/gif"
	}

	// Security headers
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Content-Security-Policy", "default-src 'none'")
	c.Header("Cache-Control", "public, max-age=86400") // 24 hours
	c.Header("Content-Type", contentType)

	// Serve the file
	c.File(fullPath)
}

// ServeProductFile serves product gallery images with proper access control
// GET /uploads/products/*filepath
//
// Path structure: /uploads/products/{tenant_id}/{product_id}/gallery/{filename}
//
// Access Control Logic:
// - If requester is authenticated as that tenant: Allow
// - If requester is authenticated as different tenant: Deny (403)
// - If requester is not authenticated (public page): Allow (needed for QR scan landing pages)
func (h *UploadHandler) ServeProductFile(c *gin.Context) {
	// Get the filepath parameter (everything after /uploads/products/)
	filePath := c.Param("filepath")
	if filePath == "" {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	// Remove leading slash if present
	filePath = strings.TrimPrefix(filePath, "/")

	// Construct full file path
	fullPath := filepath.Join(h.Cfg.UploadPath, "products", filePath)

	// Security: Prevent path traversal attacks
	if !isPathSafe(filepath.Join(h.Cfg.UploadPath, "products"), fullPath) {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	// Check if file exists
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	// Tenant isolation check
	// Path structure: {tenant_id}/{product_id}/gallery/{filename}
	parts := strings.Split(filePath, "/")
	if len(parts) >= 1 {
		pathTenantID := parts[0]

		// Check if user is authenticated as a tenant
		requestingTenantID := c.GetString("tenant_id")

		if requestingTenantID != "" {
			// User is logged in as a tenant
			if requestingTenantID != pathTenantID {
				// Trying to access another tenant's product images - deny
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
			// Same tenant - allow
		}
		// If not logged in (requestingTenantID == ""), allow access
		// This is needed for public QR scan landing pages to display product images
	}

	// Determine content type based on extension
	ext := strings.ToLower(filepath.Ext(fullPath))
	contentType := "application/octet-stream"
	switch ext {
	case ".jpg", ".jpeg":
		contentType = "image/jpeg"
	case ".png":
		contentType = "image/png"
	case ".webp":
		contentType = "image/webp"
	case ".gif":
		contentType = "image/gif"
	}

	// Security headers
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Content-Security-Policy", "default-src 'none'")
	c.Header("Cache-Control", "public, max-age=86400") // 24 hours
	c.Header("Content-Type", contentType)

	// Serve the file
	c.File(fullPath)
}

// ServeTemplateFile serves template logo images from local filesystem
// GET /uploads/templates/*filepath
// Access Control: same as ServeProductFile (public for landing pages, tenant-isolated for logged-in users)
func (h *UploadHandler) ServeTemplateFile(c *gin.Context) {
	filePath := c.Param("filepath")
	if filePath == "" {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	filePath = strings.TrimPrefix(filePath, "/")
	fullPath := filepath.Join(h.Cfg.UploadPath, "templates", filePath)

	if !isPathSafe(filepath.Join(h.Cfg.UploadPath, "templates"), fullPath) {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	// Tenant isolation: path structure is {tenant_id}/logos/{filename}
	parts := strings.Split(filePath, "/")
	if len(parts) >= 1 {
		pathTenantID := parts[0]
		requestingTenantID := c.GetString("tenant_id")
		if requestingTenantID != "" && requestingTenantID != pathTenantID {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
	}

	ext := strings.ToLower(filepath.Ext(fullPath))
	contentType := "application/octet-stream"
	switch ext {
	case ".jpg", ".jpeg":
		contentType = "image/jpeg"
	case ".png":
		contentType = "image/png"
	case ".webp":
		contentType = "image/webp"
	}

	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Content-Security-Policy", "default-src 'none'")
	c.Header("Cache-Control", "public, max-age=86400")
	c.Header("Content-Type", contentType)

	c.File(fullPath)
}

// ListTenantBackgrounds lists all background images uploaded by the current tenant
// GET /tenant/uploads/backgrounds
func (h *UploadHandler) ListTenantBackgrounds(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		utils.ErrorResponse(c, http.StatusUnauthorized, "Tenant ID not found", nil)
		return
	}

	var files []gin.H

	// Scan landing directory only (campaign backgrounds removed)
	dirPath := filepath.Join(h.Cfg.UploadPath, "backgrounds", "tenants", tenantID, "landing")
	entries, err := os.ReadDir(dirPath)
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			info, err := entry.Info()
			if err != nil {
				continue
			}
			files = append(files, gin.H{
				"filename":   entry.Name(),
				"type":       "landing",
				"url":        fmt.Sprintf("/uploads/backgrounds/tenants/%s/landing/%s", tenantID, entry.Name()),
				"size":       info.Size(),
				"created_at": info.ModTime(),
			})
		}
	}

	utils.SuccessResponse(c, http.StatusOK, "Background images retrieved", gin.H{
		"backgrounds": files,
		"total":       len(files),
	})
}

// ServeCounterfeitReportFile serves counterfeit report photos
// GET /uploads/counterfeit-reports/*filepath
// Public access — these are photos uploaded by end-users via public form
// Path structure: {tenant_id}/{report_id}/{filename}
func (h *UploadHandler) ServeCounterfeitReportFile(c *gin.Context) {
	filePath := c.Param("filepath")
	if filePath == "" {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	filePath = strings.TrimPrefix(filePath, "/")
	fullPath := filepath.Join(h.Cfg.UploadPath, "counterfeit-reports", filePath)

	if !isPathSafe(filepath.Join(h.Cfg.UploadPath, "counterfeit-reports"), fullPath) {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	// Tenant isolation for logged-in users, public access for anonymous
	parts := strings.Split(filePath, "/")
	if len(parts) >= 1 {
		pathTenantID := parts[0]
		requestingTenantID := c.GetString("tenant_id")
		if requestingTenantID != "" && requestingTenantID != pathTenantID {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
	}

	ext := strings.ToLower(filepath.Ext(fullPath))
	contentType := "application/octet-stream"
	switch ext {
	case ".jpg", ".jpeg":
		contentType = "image/jpeg"
	case ".png":
		contentType = "image/png"
	case ".webp":
		contentType = "image/webp"
	}

	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Content-Security-Policy", "default-src 'none'")
	c.Header("Cache-Control", "public, max-age=86400")
	c.Header("Content-Type", contentType)
	c.File(fullPath)
}

// ServePaymentProofFile serves payment proof images (marketing-uploaded)
// GET /uploads/payment-proofs/*filepath
func (h *UploadHandler) ServePaymentProofFile(c *gin.Context) {
	filePath := c.Param("filepath")
	if filePath == "" {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	filePath = strings.TrimPrefix(filePath, "/")
	fullPath := filepath.Join(h.Cfg.UploadPath, "payment-proofs", filePath)

	if !isPathSafe(filepath.Join(h.Cfg.UploadPath, "payment-proofs"), fullPath) {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	ext := strings.ToLower(filepath.Ext(fullPath))
	contentType := "application/octet-stream"
	switch ext {
	case ".jpg", ".jpeg":
		contentType = "image/jpeg"
	case ".png":
		contentType = "image/png"
	}

	c.Header("Cache-Control", "private, max-age=86400")
	c.Header("Content-Type", contentType)
	c.File(fullPath)
}

// ServeMsgProofFile serves topup proof images
// GET /uploads/msg-proofs/*filepath
func (h *UploadHandler) ServeMsgProofFile(c *gin.Context) {
	h.serveMessagingFile(c, "msg-proofs", "image")
}

// ServeMsgImportFile serves import CSV files and error reports
// GET /uploads/msg-imports/*filepath
func (h *UploadHandler) ServeMsgImportFile(c *gin.Context) {
	h.serveMessagingFile(c, "msg-imports", "file")
}

// serveMessagingFile is a shared helper for serving messaging upload files
func (h *UploadHandler) serveMessagingFile(c *gin.Context, subDir, fileType string) {
	filePath := c.Param("filepath")
	if filePath == "" {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	filePath = strings.TrimPrefix(filePath, "/")
	fullPath := filepath.Join(h.Cfg.UploadPath, subDir, filePath)

	if !isPathSafe(filepath.Join(h.Cfg.UploadPath, subDir), fullPath) {
		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	// Tenant isolation: path structure is {tenant_id}/{filename}
	parts := strings.Split(filePath, "/")
	if len(parts) >= 1 {
		pathTenantID := parts[0]
		if userTenantID, exists := c.Get("tenant_id"); exists {
			if tid, ok := userTenantID.(string); ok && tid != pathTenantID {
				// Also check uuid.UUID type
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
			if tid, ok := userTenantID.(uuid.UUID); ok && tid.String() != pathTenantID {
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
		}
	}

	// Determine content type
	ext := strings.ToLower(filepath.Ext(fullPath))
	contentType := "application/octet-stream"
	switch ext {
	case ".jpg", ".jpeg":
		contentType = "image/jpeg"
	case ".png":
		contentType = "image/png"
	case ".csv":
		contentType = "text/csv"
	}

	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Content-Security-Policy", "default-src 'none'")
	c.Header("Cache-Control", "public, max-age=86400")
	c.Header("Content-Type", contentType)
	c.File(fullPath)
}
