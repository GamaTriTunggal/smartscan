package utils

import (
	"crypto/rand"
	"encoding/base64"
	"time"
)

const (
	// ScanSessionTTL is the time-to-live for scan session in Redis
	ScanSessionTTL = 5 * time.Minute
)

// GenerateScanSessionID creates a cryptographically secure 8-char session ID
// Uses URL-safe base64 encoding (A-Za-z0-9-_)
// 6 bytes = 48 bits of entropy = 281 trillion combinations
func GenerateScanSessionID() (string, error) {
	bytes := make([]byte, 6) // 6 bytes = 8 base64 chars
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}
