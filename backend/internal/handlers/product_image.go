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
	"github.com/gamatritunggal/smartscan/backend/internal/models"
	"github.com/gamatritunggal/smartscan/backend/internal/storage"
	"github.com/gamatritunggal/smartscan/backend/internal/utils"
	"gorm.io/gorm"
)

type ProductImageHandler struct {
	DB  *gorm.DB
	Cfg *config.Config
}

func NewProductImageHandler(db *gorm.DB, cfg *config.Config) *ProductImageHandler {
	return &ProductImageHandler{DB: db, Cfg: cfg}
}

// UpdateProductImageRequest represents the request to update an image
type UpdateProductImageRequest struct {
	Caption   string `json:"caption" binding:"max=255"`
	SortOrder *int   `json:"sort_order"`
}

// ReorderProductImagesRequest represents the request to reorder images
type ReorderProductImagesRequest struct {
	ImageIDs []string `json:"image_ids" binding:"required"`
}

// List returns all images for a product
func (h *ProductImageHandler) List(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")
	productID := c.Param("id")

	productUUID, err := uuid.Parse(productID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid product ID", nil)
		return
	}

	// Verify product belongs to tenant
	var product models.Product
	if err := h.DB.Where("id = ? AND tenant_id = ? AND deleted_at IS NULL", productUUID, tenantID).First(&product).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Product not found", nil)
		return
	}

	var images []models.ProductImage
	if err := h.DB.Where("product_id = ?", productUUID).
		Order("sort_order ASC, created_at ASC").
		Find(&images).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to fetch images", err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Product images retrieved", gin.H{
		"images": images,
	})
}

// Upload uploads a new image for a product
func (h *ProductImageHandler) Upload(c *gin.Context) {
	tenantIDStr, _ := c.Get("tenant_id")
	tenantUUID, err := uuid.Parse(tenantIDStr.(string))
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Invalid tenant ID", nil)
		return
	}
	productID := c.Param("id")

	productUUID, err := uuid.Parse(productID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid product ID", nil)
		return
	}

	// Verify product belongs to tenant
	var product models.Product
	if err := h.DB.Where("id = ? AND tenant_id = ? AND deleted_at IS NULL", productUUID, tenantUUID).First(&product).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Product not found", nil)
		return
	}

	// Get the uploaded file first (before transaction) so we can process it
	file, header, err := c.Request.FormFile("image")
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "No image file provided", nil)
		return
	}
	defer file.Close()

	// Process and validate image with security utils
	processed, err := utils.ProcessUploadedImage(file, header, utils.ImageUploadOptions{
		MaxFileSize:  5 * 1024 * 1024, // 5MB for gallery images
		MinDimension: 400,             // Minimum 400px
		AllowedTypes: []string{"image/jpeg", "image/png", "image/webp"},
	})
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error(), nil)
		return
	}

	// Get caption and is_main from form
	caption := c.PostForm("caption")
	isMainStr := c.PostForm("is_main")
	isMain := isMainStr == "true" || isMainStr == "1"

	// Generate filename
	filename := fmt.Sprintf("%s%s", uuid.New().String(), processed.Extension)

	// Storage path key (used for both R2 and local)
	storageKey := fmt.Sprintf("products/%s/%s/gallery/%s", tenantUUID.String(), productUUID.String(), filename)

	var imageURL string
	var localFilePath string

	// Check if R2 storage is enabled
	r2Client := storage.GetGlobalR2Client()
	if r2Client != nil {
		// Upload to R2
		url, err := r2Client.Upload(context.Background(), storageKey, processed.Data, processed.ContentType)
		if err != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to upload image to storage", err)
			return
		}
		imageURL = url
	} else {
		// Fallback to local filesystem
		uploadDir := filepath.Join(h.Cfg.UploadPath, "products", tenantUUID.String(), productUUID.String(), "gallery")
		if err := utils.EnsureDir(uploadDir); err != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to create upload directory", err)
			return
		}

		localFilePath = filepath.Join(uploadDir, filename)
		if err := os.WriteFile(localFilePath, processed.Data, 0644); err != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to save image", err)
			return
		}
		imageURL = fmt.Sprintf("/uploads/%s", storageKey)
	}

	// Use transaction to prevent race condition on count check + insert
	var image models.ProductImage
	txErr := h.DB.Transaction(func(tx *gorm.DB) error {
		// Check current image count within transaction (with row lock)
		var count int64
		if err := tx.Model(&models.ProductImage{}).
			Where("product_id = ?", productUUID).
			Count(&count).Error; err != nil {
			return err
		}

		if count >= int64(models.MaxProductImages) {
			return fmt.Errorf("maximum %d images allowed per product", models.MaxProductImages)
		}

		// If this is the first image, set it as main
		if count == 0 {
			isMain = true
		}

		// If setting as main, unset other main images
		if isMain {
			if err := tx.Model(&models.ProductImage{}).
				Where("product_id = ? AND is_main = true", productUUID).
				Update("is_main", false).Error; err != nil {
				return err
			}
		}

		// Get next sort order
		var maxOrder int
		tx.Model(&models.ProductImage{}).
			Where("product_id = ?", productUUID).
			Select("COALESCE(MAX(sort_order), -1)").
			Scan(&maxOrder)

		// Create database record
		image = models.ProductImage{
			ProductID: productUUID,
			ImageURL:  imageURL,
			Caption:   caption,
			IsMain:    isMain,
			SortOrder: maxOrder + 1,
			FileSize:  len(processed.Data),
		}

		return tx.Create(&image).Error
	})

	if txErr != nil {
		// Clean up uploaded file on DB error
		if r2Client != nil {
			r2Client.Delete(context.Background(), storageKey)
		} else if localFilePath != "" {
			os.Remove(localFilePath)
		}

		// Check if it's the max image error
		if strings.Contains(txErr.Error(), "maximum") {
			utils.ErrorResponse(c, http.StatusBadRequest, txErr.Error(), nil)
		} else {
			utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to save image record", txErr)
		}
		return
	}

	utils.SuccessResponse(c, http.StatusCreated, "Image uploaded", gin.H{
		"image": image,
	})
}

// Update updates an image's caption or sort order
func (h *ProductImageHandler) Update(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")
	productID := c.Param("id")
	imageID := c.Param("img_id")

	productUUID, err := uuid.Parse(productID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid product ID", nil)
		return
	}

	imageUUID, err := uuid.Parse(imageID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid image ID", nil)
		return
	}

	// Verify product belongs to tenant
	var product models.Product
	if err := h.DB.Where("id = ? AND tenant_id = ? AND deleted_at IS NULL", productUUID, tenantID).First(&product).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Product not found", nil)
		return
	}

	// Find the image
	var image models.ProductImage
	if err := h.DB.Where("id = ? AND product_id = ?", imageUUID, productUUID).First(&image).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Image not found", nil)
		return
	}

	var req UpdateProductImageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request", err)
		return
	}

	// Update fields
	updates := map[string]interface{}{}
	if req.Caption != "" || c.Request.ContentLength > 0 {
		updates["caption"] = req.Caption
	}
	if req.SortOrder != nil {
		updates["sort_order"] = *req.SortOrder
	}

	if len(updates) > 0 {
		if err := h.DB.Model(&image).Updates(updates).Error; err != nil {
			utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to update image", err)
			return
		}
	}

	// Reload
	h.DB.First(&image, "id = ?", image.ID)

	utils.SuccessResponse(c, http.StatusOK, "Image updated", gin.H{
		"image": image,
	})
}

// Delete deletes an image
func (h *ProductImageHandler) Delete(c *gin.Context) {
	tenantIDStr, _ := c.Get("tenant_id")
	tenantUUID, err := uuid.Parse(tenantIDStr.(string))
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Invalid tenant ID", nil)
		return
	}
	productID := c.Param("id")
	imageID := c.Param("img_id")

	productUUID, err := uuid.Parse(productID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid product ID", nil)
		return
	}

	imageUUID, err := uuid.Parse(imageID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid image ID", nil)
		return
	}

	// Verify product belongs to tenant
	var product models.Product
	if err := h.DB.Where("id = ? AND tenant_id = ? AND deleted_at IS NULL", productUUID, tenantUUID).First(&product).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Product not found", nil)
		return
	}

	// Find the image
	var image models.ProductImage
	if err := h.DB.Where("id = ? AND product_id = ?", imageUUID, productUUID).First(&image).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Image not found", nil)
		return
	}

	wasMain := image.IsMain

	// Delete the file from storage
	r2Client := storage.GetGlobalR2Client()
	if r2Client != nil && r2Client.IsR2URL(image.ImageURL) {
		// Delete from R2
		key := r2Client.ExtractKeyFromURL(image.ImageURL)
		if key != "" {
			r2Client.Delete(context.Background(), key)
		}
	} else if strings.HasPrefix(image.ImageURL, "/uploads/") {
		// Delete from local filesystem
		// Extract path after /uploads/
		relativePath := strings.TrimPrefix(image.ImageURL, "/uploads/")
		filePath := filepath.Join(h.Cfg.UploadPath, relativePath)
		os.Remove(filePath) // Ignore error if file doesn't exist
	}

	// Delete the record
	if err := h.DB.Delete(&image).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to delete image", err)
		return
	}

	// If deleted image was main, set another image as main
	if wasMain {
		var firstImage models.ProductImage
		if err := h.DB.Where("product_id = ?", productUUID).
			Order("sort_order ASC").
			First(&firstImage).Error; err == nil {
			h.DB.Model(&firstImage).Update("is_main", true)
		}
	}

	// Reorder remaining images
	var remainingImages []models.ProductImage
	h.DB.Where("product_id = ?", productUUID).Order("sort_order ASC").Find(&remainingImages)
	for i, img := range remainingImages {
		if img.SortOrder != i {
			h.DB.Model(&img).Update("sort_order", i)
		}
	}

	utils.SuccessResponse(c, http.StatusOK, "Image deleted", nil)
}

// SetMain sets an image as the main image and moves it to first position
func (h *ProductImageHandler) SetMain(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")
	productID := c.Param("id")
	imageID := c.Param("img_id")

	productUUID, err := uuid.Parse(productID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid product ID", nil)
		return
	}

	imageUUID, err := uuid.Parse(imageID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid image ID", nil)
		return
	}

	// Verify product belongs to tenant
	var product models.Product
	if err := h.DB.Where("id = ? AND tenant_id = ? AND deleted_at IS NULL", productUUID, tenantID).First(&product).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Product not found", nil)
		return
	}

	// Find the image
	var image models.ProductImage
	if err := h.DB.Where("id = ? AND product_id = ?", imageUUID, productUUID).First(&image).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Image not found", nil)
		return
	}

	// Use transaction to atomically update main flag and sort order
	txErr := h.DB.Transaction(func(tx *gorm.DB) error {
		// Unset all other main images
		if err := tx.Model(&models.ProductImage{}).
			Where("product_id = ? AND is_main = true", productUUID).
			Update("is_main", false).Error; err != nil {
			return err
		}

		// Set this image as main
		if err := tx.Model(&image).Update("is_main", true).Error; err != nil {
			return err
		}

		// Move this image to sort_order 0 by shifting other images
		// First, increment sort_order for all images with sort_order < current image's sort_order
		if image.SortOrder > 0 {
			if err := tx.Model(&models.ProductImage{}).
				Where("product_id = ? AND sort_order < ?", productUUID, image.SortOrder).
				Update("sort_order", gorm.Expr("sort_order + 1")).Error; err != nil {
				return err
			}

			// Set this image's sort_order to 0
			if err := tx.Model(&image).Update("sort_order", 0).Error; err != nil {
				return err
			}
		}

		return nil
	})

	if txErr != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to set main image", txErr)
		return
	}

	// Reload image with updated values
	h.DB.First(&image, "id = ?", image.ID)

	utils.SuccessResponse(c, http.StatusOK, "Main image set", gin.H{
		"image": image,
	})
}

// Reorder updates the sort order of images
func (h *ProductImageHandler) Reorder(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")
	productID := c.Param("id")

	productUUID, err := uuid.Parse(productID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid product ID", nil)
		return
	}

	// Verify product belongs to tenant
	var product models.Product
	if err := h.DB.Where("id = ? AND tenant_id = ? AND deleted_at IS NULL", productUUID, tenantID).First(&product).Error; err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "Product not found", nil)
		return
	}

	var req ReorderProductImagesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request", err)
		return
	}

	// Get all existing images for this product
	var existingImages []models.ProductImage
	if err := h.DB.Where("product_id = ?", productUUID).Find(&existingImages).Error; err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to fetch images", err)
		return
	}

	// Build a set of existing image IDs for validation
	existingIDSet := make(map[uuid.UUID]bool)
	for _, img := range existingImages {
		existingIDSet[img.ID] = true
	}

	// Validate: parse all IDs and check for uniqueness
	requestedIDs := make(map[uuid.UUID]bool)
	parsedIDs := make([]uuid.UUID, 0, len(req.ImageIDs))
	for _, imageIDStr := range req.ImageIDs {
		imageID, err := uuid.Parse(imageIDStr)
		if err != nil {
			utils.ErrorResponse(c, http.StatusBadRequest, "Invalid image ID in list", nil)
			return
		}

		// Check for duplicates in request
		if requestedIDs[imageID] {
			utils.ErrorResponse(c, http.StatusBadRequest, "Duplicate image ID in list", nil)
			return
		}
		requestedIDs[imageID] = true

		// Check that image exists and belongs to this product
		if !existingIDSet[imageID] {
			utils.ErrorResponse(c, http.StatusNotFound, "Image not found or does not belong to this product", nil)
			return
		}

		parsedIDs = append(parsedIDs, imageID)
	}

	// Validate: all existing images must be in the request (complete list)
	if len(parsedIDs) != len(existingImages) {
		utils.ErrorResponse(c, http.StatusBadRequest,
			fmt.Sprintf("Image ID list must contain all %d images for this product", len(existingImages)), nil)
		return
	}

	// Update sort order within a transaction
	txErr := h.DB.Transaction(func(tx *gorm.DB) error {
		for i, imageID := range parsedIDs {
			if err := tx.Model(&models.ProductImage{}).
				Where("id = ? AND product_id = ?", imageID, productUUID).
				Update("sort_order", i).Error; err != nil {
				return err
			}
		}
		return nil
	})

	if txErr != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to reorder images", txErr)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Images reordered", nil)
}
