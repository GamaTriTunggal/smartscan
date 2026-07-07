package utils

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"mime/multipart"

	_ "golang.org/x/image/webp" // WebP decoder
)

// ImageUploadOptions configures image upload validation
type ImageUploadOptions struct {
	MaxFileSize  int64    // Maximum file size in bytes
	MinDimension int      // Minimum for the smaller dimension (width or height)
	AllowedTypes []string // Allowed MIME types
}

// MaxImagePixels caps the total decoded pixel count (width*height) accepted for
// an upload, guarding against decompression bombs: a small, highly-compressible
// image can declare enormous dimensions that would allocate gigabytes when
// decoded (width*height*4 bytes). 40 megapixels comfortably covers legitimate
// photography while keeping the worst-case allocation bounded (~160 MB).
const MaxImagePixels = 40_000_000

// MaxImageDimension caps either side of an uploaded image.
const MaxImageDimension = 12000

// ProcessedImage contains the cleaned/validated image data
type ProcessedImage struct {
	Data        []byte // Clean, re-encoded image data (metadata stripped)
	ContentType string // Actual content type after processing
	Extension   string // File extension (e.g., ".jpg", ".png")
	Width       int
	Height      int
}

// DefaultUploadOptions returns standard options for image uploads
func DefaultUploadOptions() ImageUploadOptions {
	return ImageUploadOptions{
		MaxFileSize:  2 * 1024 * 1024, // 2MB
		MinDimension: 720,
		AllowedTypes: []string{"image/jpeg", "image/png", "image/webp"},
	}
}

// ProcessUploadedImage validates and sanitizes an uploaded image file
// It performs: size check, content-type validation, magic bytes verification,
// metadata stripping via re-encoding, and dimension validation
func ProcessUploadedImage(file multipart.File, header *multipart.FileHeader, opts ImageUploadOptions) (*ProcessedImage, error) {
	// 1. Check file size
	if header.Size > opts.MaxFileSize {
		return nil, fmt.Errorf("file size exceeds maximum limit of %d MB", opts.MaxFileSize/(1024*1024))
	}

	// 2. Check content type against allowed types
	contentType := header.Header.Get("Content-Type")
	allowed := false
	for _, t := range opts.AllowedTypes {
		if t == contentType {
			allowed = true
			break
		}
	}
	if !allowed {
		return nil, errors.New("invalid file type. Allowed: JPEG, PNG, WebP")
	}

	// 3. Read file content
	imgData, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read image file: %w", err)
	}

	// 4. Validate magic bytes match claimed content type
	if !ValidateMagicBytes(imgData, contentType) {
		return nil, errors.New("file content does not match declared image type")
	}

	// 4b. Decompression-bomb guard: read ONLY the image header (DecodeConfig does
	// not allocate the pixel buffer) and reject oversized dimensions BEFORE the
	// full decode in ReencodeImage. Without this a ~500KB solid-color PNG could
	// declare 25000x25000 and force a multi-GB allocation, OOM-killing the process.
	if cfg, _, cfgErr := image.DecodeConfig(bytes.NewReader(imgData)); cfgErr == nil {
		if cfg.Width <= 0 || cfg.Height <= 0 {
			return nil, errors.New("invalid image dimensions")
		}
		if cfg.Width > MaxImageDimension || cfg.Height > MaxImageDimension ||
			int64(cfg.Width)*int64(cfg.Height) > MaxImagePixels {
			return nil, fmt.Errorf("image dimensions too large (max %d x %d and %d total pixels)",
				MaxImageDimension, MaxImageDimension, MaxImagePixels)
		}
	} else {
		return nil, fmt.Errorf("failed to read image header: %w", cfgErr)
	}

	// 5. Re-encode image to strip metadata (EXIF, XMP, etc.)
	cleanData, actualContentType, err := ReencodeImage(imgData, contentType)
	if err != nil {
		return nil, fmt.Errorf("failed to process image: %w", err)
	}

	// 6. Validate dimensions
	width, height, valid, err := ValidateImageDimensions(cleanData, opts.MinDimension)
	if err != nil {
		return nil, fmt.Errorf("failed to read image dimensions: %w", err)
	}
	if !valid {
		smallerDim := width
		if height < width {
			smallerDim = height
		}
		return nil, fmt.Errorf("image smallest dimension must be at least %dpx. Current: %dpx (%dx%d)", opts.MinDimension, smallerDim, width, height)
	}

	// 7. Determine file extension based on actual content type
	var ext string
	switch actualContentType {
	case "image/jpeg":
		ext = ".jpg"
	case "image/png":
		ext = ".png"
	case "image/webp":
		ext = ".webp"
	default:
		ext = ".png" // Fallback
	}

	return &ProcessedImage{
		Data:        cleanData,
		ContentType: actualContentType,
		Extension:   ext,
		Width:       width,
		Height:      height,
	}, nil
}

// Magic bytes for supported file types
var (
	jpegMagic = []byte{0xFF, 0xD8, 0xFF}
	pngMagic  = []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	gifMagic  = []byte{0x47, 0x49, 0x46, 0x38} // "GIF8" (both GIF87a and GIF89a)
	webpRIFF  = []byte{0x52, 0x49, 0x46, 0x46} // "RIFF"
	webpWEBP  = []byte{0x57, 0x45, 0x42, 0x50} // "WEBP"
	pdfMagic  = []byte{0x25, 0x50, 0x44, 0x46} // "%PDF"
)

// ValidateMagicBytes checks if file content matches claimed MIME type
// This prevents attacks where malicious files are disguised as images
func ValidateMagicBytes(data []byte, contentType string) bool {
	if len(data) < 12 {
		return false
	}

	switch contentType {
	case "image/jpeg":
		return bytes.HasPrefix(data, jpegMagic)
	case "image/png":
		return bytes.HasPrefix(data, pngMagic)
	case "image/gif":
		return bytes.HasPrefix(data, gifMagic)
	case "image/webp":
		// WebP format: RIFF....WEBP
		return bytes.HasPrefix(data, webpRIFF) && bytes.Equal(data[8:12], webpWEBP)
	case "application/pdf":
		return bytes.HasPrefix(data, pdfMagic)
	default:
		return false
	}
}

// ReencodeImage decodes and re-encodes an image to strip all metadata
// This removes EXIF, XMP, IPTC data and any potentially malicious embedded content
// Returns the clean image data and any error encountered
func ReencodeImage(data []byte, contentType string) ([]byte, string, error) {
	// Decode the image
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, "", errors.New("failed to decode image: " + err.Error())
	}

	var buf bytes.Buffer
	var outputType string

	switch contentType {
	case "image/jpeg":
		// Re-encode as JPEG with high quality
		err = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 92})
		outputType = "image/jpeg"
	case "image/png":
		// Re-encode as PNG
		err = png.Encode(&buf, img)
		outputType = "image/png"
	case "image/gif":
		// Re-encode as GIF (first frame only for animated GIFs)
		// This strips metadata while preserving the image
		err = gif.Encode(&buf, img, nil)
		outputType = "image/gif"
	case "image/webp":
		// Go doesn't have a standard WebP encoder, convert to PNG
		// This ensures we strip metadata while maintaining quality
		err = png.Encode(&buf, img)
		outputType = "image/png"
	default:
		return nil, "", errors.New("unsupported image type: " + contentType)
	}

	if err != nil {
		return nil, "", errors.New("failed to encode image: " + err.Error())
	}

	return buf.Bytes(), outputType, nil
}

// GetImageDimensions returns the width and height of an image
func GetImageDimensions(data []byte) (width, height int, err error) {
	config, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return 0, 0, err
	}
	return config.Width, config.Height, nil
}

// ValidateImageDimensions checks if image meets minimum dimension requirements
// Uses flexible validation: smallest dimension must be at least minDimension
func ValidateImageDimensions(data []byte, minDimension int) (width, height int, valid bool, err error) {
	width, height, err = GetImageDimensions(data)
	if err != nil {
		return 0, 0, false, err
	}

	// Check that the smaller dimension meets minimum requirement
	smallerDim := width
	if height < width {
		smallerDim = height
	}

	valid = smallerDim >= minDimension
	return width, height, valid, nil
}
