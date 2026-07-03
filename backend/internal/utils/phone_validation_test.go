package utils

import (
	"testing"
)

func TestValidateAndNormalizePhone(t *testing.T) {
	tests := []struct {
		name        string
		phone       string
		countryHint string
		expected    string
		expectError bool
		errorMsg    string
	}{
		// Indonesia (ID) - local format
		{
			name:        "Indonesia local format 08",
			phone:       "08123456789",
			countryHint: "ID",
			expected:    "+628123456789",
			expectError: false,
		},
		{
			name:        "Indonesia local format with area code",
			phone:       "081234567890",
			countryHint: "ID",
			expected:    "+6281234567890",
			expectError: false,
		},
		{
			name:        "Indonesia E.164 format",
			phone:       "+628123456789",
			countryHint: "",
			expected:    "+628123456789",
			expectError: false,
		},
		{
			name:        "Indonesia with spaces",
			phone:       "0812 3456 789",
			countryHint: "ID",
			expected:    "+628123456789",
			expectError: false,
		},
		{
			name:        "Indonesia with dashes",
			phone:       "0812-3456-789",
			countryHint: "ID",
			expected:    "+628123456789",
			expectError: false,
		},

		// United States (US) - using valid NYC area code 212
		{
			name:        "US local format",
			phone:       "(212) 555-1234",
			countryHint: "US",
			expected:    "+12125551234",
			expectError: false,
		},
		{
			name:        "US E.164 format",
			phone:       "+12125551234",
			countryHint: "",
			expected:    "+12125551234",
			expectError: false,
		},

		// Singapore (SG)
		{
			name:        "Singapore local format",
			phone:       "91234567",
			countryHint: "SG",
			expected:    "+6591234567",
			expectError: false,
		},
		{
			name:        "Singapore E.164 format",
			phone:       "+6591234567",
			countryHint: "",
			expected:    "+6591234567",
			expectError: false,
		},

		// Malaysia (MY)
		{
			name:        "Malaysia local format",
			phone:       "0123456789",
			countryHint: "MY",
			expected:    "+60123456789",
			expectError: false,
		},
		{
			name:        "Malaysia E.164 format",
			phone:       "+60123456789",
			countryHint: "",
			expected:    "+60123456789",
			expectError: false,
		},

		// Australia (AU)
		{
			name:        "Australia local format",
			phone:       "0412345678",
			countryHint: "AU",
			expected:    "+61412345678",
			expectError: false,
		},

		// United Kingdom (GB)
		{
			name:        "UK local format",
			phone:       "07911123456",
			countryHint: "GB",
			expected:    "+447911123456",
			expectError: false,
		},

		// Japan (JP)
		{
			name:        "Japan local format",
			phone:       "09012345678",
			countryHint: "JP",
			expected:    "+819012345678",
			expectError: false,
		},

		// Germany (DE)
		{
			name:        "Germany local format",
			phone:       "015112345678",
			countryHint: "DE",
			expected:    "+4915112345678",
			expectError: false,
		},

		// Edge cases - empty and whitespace
		{
			name:        "empty string allowed",
			phone:       "",
			countryHint: "",
			expected:    "",
			expectError: false,
		},
		{
			name:        "whitespace only",
			phone:       "   ",
			countryHint: "",
			expected:    "",
			expectError: false,
		},

		// Error cases
		{
			name:        "local format without country hint",
			phone:       "08123456789",
			countryHint: "",
			expected:    "",
			expectError: true,
			errorMsg:    "country code is required for local phone format",
		},
		{
			name:        "too short",
			phone:       "081234",
			countryHint: "ID",
			expected:    "",
			expectError: true,
			errorMsg:    "invalid phone number for the specified country",
		},
		{
			name:        "invalid characters",
			phone:       "abc123",
			countryHint: "ID",
			expected:    "",
			expectError: true,
			errorMsg:    "invalid phone number for the specified country",
		},
		{
			name:        "wrong country format",
			phone:       "08123456789",
			countryHint: "US",
			expected:    "",
			expectError: true,
			errorMsg:    "invalid phone number for the specified country",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ValidateAndNormalizePhone(tt.phone, tt.countryHint)

			if tt.expectError {
				if err == nil {
					t.Errorf("ValidateAndNormalizePhone(%q, %q) expected error, got nil", tt.phone, tt.countryHint)
				} else if err.Error() != tt.errorMsg {
					t.Errorf("ValidateAndNormalizePhone(%q, %q) error = %q, want %q", tt.phone, tt.countryHint, err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateAndNormalizePhone(%q, %q) unexpected error: %v", tt.phone, tt.countryHint, err)
				}
				if result != tt.expected {
					t.Errorf("ValidateAndNormalizePhone(%q, %q) = %q, want %q", tt.phone, tt.countryHint, result, tt.expected)
				}
			}
		})
	}
}

func TestNormalizePhone(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "remove spaces",
			input:    "0812 3456 789",
			expected: "08123456789",
		},
		{
			name:     "remove dashes",
			input:    "0812-3456-789",
			expected: "08123456789",
		},
		{
			name:     "remove parentheses",
			input:    "(021) 12345678",
			expected: "02112345678",
		},
		{
			name:     "remove dots",
			input:    "0812.3456.789",
			expected: "08123456789",
		},
		{
			name:     "preserve plus sign",
			input:    "+62 812-3456-789",
			expected: "+628123456789",
		},
		{
			name:     "complex format",
			input:    "+1 (555) 123-4567",
			expected: "+15551234567",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "trim whitespace",
			input:    "  08123456789  ",
			expected: "08123456789",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizePhone(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizePhone(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestPhoneNormalizationConsistency ensures same phone formats normalize to same E.164
func TestPhoneNormalizationConsistency(t *testing.T) {
	// All these Indonesian phone variations should normalize to +628123456789
	indonesiaVariants := []struct {
		phone       string
		countryHint string
	}{
		{"08123456789", "ID"},
		{"0812 3456 789", "ID"},
		{"0812-3456-789", "ID"},
		{"+628123456789", ""},
		{"+62 812 3456 789", ""},
		{"+62-812-3456-789", ""},
		{"628123456789", "ID"},
	}

	expectedID := "+628123456789"
	for _, variant := range indonesiaVariants {
		result, err := ValidateAndNormalizePhone(variant.phone, variant.countryHint)
		if err != nil {
			t.Errorf("Phone variant (%q, %q) unexpected error: %v", variant.phone, variant.countryHint, err)
			continue
		}
		if result != expectedID {
			t.Errorf("Phone variant (%q, %q) normalized to %q, expected %q", variant.phone, variant.countryHint, result, expectedID)
		}
	}
}

// TestE164DetectsCountryWithoutHint tests that E.164 format works without country hint
func TestE164DetectsCountryWithoutHint(t *testing.T) {
	tests := []struct {
		phone    string
		expected string
	}{
		{"+628123456789", "+628123456789"},   // Indonesia
		{"+12125551234", "+12125551234"},     // US (NYC area code 212)
		{"+6591234567", "+6591234567"},       // Singapore
		{"+60123456789", "+60123456789"},     // Malaysia
		{"+447911123456", "+447911123456"},   // UK
		{"+819012345678", "+819012345678"},   // Japan
	}

	for _, tt := range tests {
		t.Run(tt.phone, func(t *testing.T) {
			// No country hint provided - should detect from +XX prefix
			result, err := ValidateAndNormalizePhone(tt.phone, "")
			if err != nil {
				t.Errorf("ValidateAndNormalizePhone(%q, \"\") unexpected error: %v", tt.phone, err)
			}
			if result != tt.expected {
				t.Errorf("ValidateAndNormalizePhone(%q, \"\") = %q, want %q", tt.phone, result, tt.expected)
			}
		})
	}
}

// TestLowercaseCountryHint ensures country hints work regardless of case
func TestLowercaseCountryHint(t *testing.T) {
	variants := []string{"id", "ID", "Id", "iD"}
	phone := "08123456789"
	expected := "+628123456789"

	for _, hint := range variants {
		result, err := ValidateAndNormalizePhone(phone, hint)
		if err != nil {
			t.Errorf("ValidateAndNormalizePhone(%q, %q) unexpected error: %v", phone, hint, err)
		}
		if result != expected {
			t.Errorf("ValidateAndNormalizePhone(%q, %q) = %q, want %q", phone, hint, result, expected)
		}
	}
}
