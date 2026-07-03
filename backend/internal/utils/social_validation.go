package utils

import (
	"errors"
	"net/mail"
	"net/url"
	"regexp"
	"strings"
)

// Social media validation types
const (
	ValidationTypePhone    = "phone"
	ValidationTypeUsername = "username"
	ValidationTypeEmail    = "email"
	ValidationTypeURL      = "url"
	ValidationTypeText     = "text"
)

var (
	// Username regex: alphanumeric, underscore, period, 1-30 chars
	usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_.]{1,30}$`)
)

// ValidateSocialHandle validates and normalizes social media handles based on validation type.
// Returns the normalized value and an error if validation fails.
func ValidateSocialHandle(validationType, handle string) (string, error) {
	handle = strings.TrimSpace(handle)
	if handle == "" {
		return "", errors.New("handle cannot be empty")
	}

	switch validationType {
	case ValidationTypePhone:
		return ValidateSocialPhone(handle)
	case ValidationTypeUsername:
		return ValidateSocialUsername(handle)
	case ValidationTypeEmail:
		return ValidateSocialEmail(handle)
	case ValidationTypeURL:
		return ValidateSocialURL(handle)
	case ValidationTypeText:
		// No validation for text type, just return as-is
		return handle, nil
	default:
		// Unknown type, return as-is
		return handle, nil
	}
}

// ValidateSocialPhone validates and normalizes phone numbers for social media (WhatsApp, Zalo).
// Requires E.164 format with country code (must start with +).
func ValidateSocialPhone(phone string) (string, error) {
	// Remove common formatting characters
	normalized := strings.Map(func(r rune) rune {
		switch r {
		case ' ', '-', '(', ')', '.':
			return -1 // Remove
		default:
			return r
		}
	}, phone)

	// Must start with +
	if !strings.HasPrefix(normalized, "+") {
		return normalized, errors.New("phone number must start with country code (e.g., +62 for Indonesia)")
	}

	// Basic E.164 validation: + followed by 7-15 digits
	if len(normalized) < 8 || len(normalized) > 16 {
		return normalized, errors.New("phone number must be 7-15 digits after country code")
	}

	// Check that rest is digits
	for i, r := range normalized {
		if i == 0 {
			continue // Skip the +
		}
		if r < '0' || r > '9' {
			return normalized, errors.New("phone number can only contain digits after country code")
		}
	}

	// Additional check: first digit after + should be 1-9 (no leading zero in country code)
	if len(normalized) > 1 && normalized[1] == '0' {
		return normalized, errors.New("invalid country code (cannot start with 0)")
	}

	return normalized, nil
}

// ValidateSocialUsername validates and normalizes social media usernames.
// Strips @ prefix and validates format (alphanumeric, underscore, period).
func ValidateSocialUsername(username string) (string, error) {
	// Strip @ prefix if present
	normalized := strings.TrimPrefix(username, "@")

	// Check if empty after stripping
	if normalized == "" {
		return "", errors.New("username cannot be empty")
	}

	// Check length (most platforms: 1-30 chars)
	if len(normalized) > 30 {
		return normalized, errors.New("username must be 30 characters or less")
	}

	// Validate format
	if !usernameRegex.MatchString(normalized) {
		return normalized, errors.New("username can only contain letters, numbers, underscores (_), and periods (.)")
	}

	return normalized, nil
}

// ValidateSocialEmail validates and normalizes email addresses.
// Converts to lowercase.
func ValidateSocialEmail(email string) (string, error) {
	// Normalize to lowercase
	normalized := strings.ToLower(email)

	// Parse email
	addr, err := mail.ParseAddress(normalized)
	if err != nil {
		return normalized, errors.New("invalid email format")
	}

	return addr.Address, nil
}

// ValidateSocialURL validates and normalizes URLs.
// Adds https:// if scheme is missing.
func ValidateSocialURL(rawURL string) (string, error) {
	normalized := strings.TrimSpace(rawURL)

	// If no scheme, assume https
	if !strings.HasPrefix(normalized, "http://") && !strings.HasPrefix(normalized, "https://") {
		normalized = "https://" + normalized
	}

	// Parse URL
	parsed, err := url.Parse(normalized)
	if err != nil {
		return normalized, errors.New("invalid URL format")
	}

	// Must have a host
	if parsed.Host == "" {
		return normalized, errors.New("URL must include a domain")
	}

	// Must be http or https
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return normalized, errors.New("URL must use http or https")
	}

	return normalized, nil
}
