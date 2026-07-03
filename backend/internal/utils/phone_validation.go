package utils

import (
	"errors"
	"strings"

	"github.com/nyaruka/phonenumbers"
)

// ValidateAndNormalizePhone validates and normalizes a phone number using libphonenumber.
// Accepts both E.164 format (+628123456789) and local format (08123456789).
// Returns the normalized E.164 format or an error.
//
// Parameters:
//   - phone: The phone number to validate
//   - countryHint: ISO 3166-1 alpha-2 country code (e.g., "ID", "US") for local format parsing
//
// If countryHint is empty and phone doesn't start with +, validation will fail.
func ValidateAndNormalizePhone(phone string, countryHint string) (string, error) {
	phone = strings.TrimSpace(phone)
	if phone == "" {
		return "", nil // Empty phone is allowed (may not be required)
	}

	// Normalize country hint to uppercase
	countryHint = strings.ToUpper(strings.TrimSpace(countryHint))

	// Handle country hint for E.164 vs local format
	if countryHint == "" {
		if strings.HasPrefix(phone, "+") {
			countryHint = "ZZ" // Unknown region, let library detect from +XX
		} else {
			return "", errors.New("country code is required for local phone format")
		}
	}

	// Parse phone number with country hint
	num, err := phonenumbers.Parse(phone, countryHint)
	if err != nil {
		return "", errors.New("invalid phone number format")
	}

	// Validate the parsed number
	if !phonenumbers.IsValidNumber(num) {
		return "", errors.New("invalid phone number for the specified country")
	}

	// Format to E.164 for consistent storage
	e164 := phonenumbers.Format(num, phonenumbers.E164)
	return e164, nil
}

// NormalizePhone removes common formatting characters from phone numbers.
// This is a lightweight normalization without full validation.
func NormalizePhone(phone string) string {
	phone = strings.TrimSpace(phone)
	if phone == "" {
		return ""
	}

	var normalized strings.Builder
	for _, r := range phone {
		switch r {
		case ' ', '-', '(', ')', '.':
			// Skip formatting characters
		default:
			normalized.WriteRune(r)
		}
	}

	return normalized.String()
}
