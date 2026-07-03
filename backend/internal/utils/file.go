package utils

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// SanitizeFileName removes potentially dangerous characters from a filename
// to prevent path traversal and other attacks
func SanitizeFileName(filename string) string {
	// Remove path separators
	filename = filepath.Base(filename)

	// Remove null bytes
	filename = strings.ReplaceAll(filename, "\x00", "")

	// Remove directory traversal patterns
	filename = strings.ReplaceAll(filename, "..", "")
	filename = strings.ReplaceAll(filename, "/", "")
	filename = strings.ReplaceAll(filename, "\\", "")

	// Remove leading dots (hidden files)
	filename = strings.TrimLeft(filename, ".")

	// Replace spaces with underscores
	filename = strings.ReplaceAll(filename, " ", "_")

	// Only allow alphanumeric, dots, underscores, and hyphens
	reg := regexp.MustCompile(`[^a-zA-Z0-9._-]`)
	filename = reg.ReplaceAllString(filename, "")

	// Limit length
	if len(filename) > 200 {
		filename = filename[:200]
	}

	// Default name if empty
	if filename == "" {
		filename = "file"
	}

	return filename
}

// EnsureDir creates a directory if it doesn't exist
func EnsureDir(path string) error {
	return os.MkdirAll(path, 0755)
}

// DeleteFile deletes a file at the given path
// Returns nil if file doesn't exist
func DeleteFile(path string) error {
	err := os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
