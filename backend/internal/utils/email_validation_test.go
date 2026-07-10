package utils

import (
	"testing"
)

func TestNormalizeEmail(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Basic normalization (ALL providers)
		{
			name:     "lowercase conversion",
			input:    "John.Doe@Company.COM",
			expected: "john.doe@company.com",
		},
		{
			name:     "trim whitespace",
			input:    "  user@example.com  ",
			expected: "user@example.com",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "invalid format without @",
			input:    "notanemail",
			expected: "notanemail",
		},

		// +suffix removal (ALL providers)
		{
			name:     "outlook +suffix removal",
			input:    "john+spam@outlook.com",
			expected: "john@outlook.com",
		},
		{
			name:     "hotmail +suffix removal",
			input:    "user+newsletter@hotmail.com",
			expected: "user@hotmail.com",
		},
		{
			name:     "protonmail +suffix removal",
			input:    "secure+promo@protonmail.com",
			expected: "secure@protonmail.com",
		},
		{
			name:     "icloud +suffix removal",
			input:    "apple+test@icloud.com",
			expected: "apple@icloud.com",
		},
		{
			name:     "yahoo +suffix removal",
			input:    "user+tag@yahoo.com",
			expected: "user@yahoo.com",
		},
		{
			name:     "company domain +suffix removal",
			input:    "john+news@company.co.id",
			expected: "john@company.co.id",
		},
		{
			name:     "fastmail +suffix removal",
			input:    "user+alias@fastmail.com",
			expected: "user@fastmail.com",
		},

		// Gmail-specific: dots AND +suffix removal
		{
			name:     "gmail dots removal",
			input:    "j.o.h.n@gmail.com",
			expected: "john@gmail.com",
		},
		{
			name:     "gmail +suffix removal",
			input:    "john+promo@gmail.com",
			expected: "john@gmail.com",
		},
		{
			name:     "gmail dots and +suffix combined",
			input:    "j.o.h.n+promo@gmail.com",
			expected: "john@gmail.com",
		},
		{
			name:     "gmail uppercase with dots and +suffix",
			input:    "J.O.H.N+SPAM@Gmail.com",
			expected: "john@gmail.com",
		},

		// googlemail.com → gmail.com normalization
		{
			name:     "googlemail to gmail",
			input:    "user@googlemail.com",
			expected: "user@gmail.com",
		},
		{
			name:     "googlemail with dots",
			input:    "u.s.e.r@googlemail.com",
			expected: "user@gmail.com",
		},
		{
			name:     "googlemail with +suffix",
			input:    "user+test@googlemail.com",
			expected: "user@gmail.com",
		},

		// Non-Gmail: dots PRESERVED (critical!)
		{
			name:     "outlook dots preserved",
			input:    "john.doe@outlook.com",
			expected: "john.doe@outlook.com",
		},
		{
			name:     "company email dots preserved",
			input:    "john.doe@company.co.id",
			expected: "john.doe@company.co.id",
		},
		{
			name:     "yahoo dots preserved",
			input:    "first.last@yahoo.com",
			expected: "first.last@yahoo.com",
		},
		{
			name:     "hotmail dots preserved",
			input:    "user.name@hotmail.com",
			expected: "user.name@hotmail.com",
		},

		// Combined scenarios
		{
			name:     "outlook dots preserved + suffix removed",
			input:    "john.doe+promo@outlook.com",
			expected: "john.doe@outlook.com",
		},
		{
			name:     "company email dots preserved + suffix removed",
			input:    "first.last+newsletter@company.com",
			expected: "first.last@company.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeEmail(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeEmail(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsDisposableEmail(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		expected bool
	}{
		// Disposable emails (should return true)
		{
			name:     "tempmail.com",
			email:    "user@tempmail.com",
			expected: true,
		},
		{
			name:     "mailinator.com",
			email:    "test@mailinator.com",
			expected: true,
		},
		{
			name:     "guerrillamail.com",
			email:    "anon@guerrillamail.com",
			expected: true,
		},
		{
			name:     "10minutemail.com",
			email:    "temp@10minutemail.com",
			expected: true,
		},
		{
			name:     "yopmail.com",
			email:    "throwaway@yopmail.com",
			expected: true,
		},
		{
			name:     "maildrop.cc",
			email:    "random@maildrop.cc",
			expected: true,
		},
		{
			name:     "burnermail.io",
			email:    "burn@burnermail.io",
			expected: true,
		},
		{
			name:     "trashmail.com",
			email:    "trash@trashmail.com",
			expected: true,
		},

		// Legitimate emails (should return false)
		{
			name:     "gmail.com",
			email:    "user@gmail.com",
			expected: false,
		},
		{
			name:     "outlook.com",
			email:    "user@outlook.com",
			expected: false,
		},
		{
			name:     "yahoo.com",
			email:    "user@yahoo.com",
			expected: false,
		},
		{
			name:     "company domain",
			email:    "employee@company.co.id",
			expected: false,
		},
		{
			name:     "protonmail.com",
			email:    "secure@protonmail.com",
			expected: false,
		},
		{
			name:     "icloud.com",
			email:    "apple@icloud.com",
			expected: false,
		},

		// Edge cases
		{
			name:     "empty string",
			email:    "",
			expected: false,
		},
		{
			name:     "invalid format",
			email:    "notanemail",
			expected: false,
		},
		{
			name:     "case insensitive check",
			email:    "USER@TEMPMAIL.COM",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsDisposableEmail(tt.email)
			if result != tt.expected {
				t.Errorf("IsDisposableEmail(%q) = %v, want %v", tt.email, result, tt.expected)
			}
		})
	}
}

func TestValidateAndNormalizeEmail(t *testing.T) {
	tests := []struct {
		name        string
		email       string
		expected    string
		expectError bool
		errorMsg    string
	}{
		// Successful validations with normalization
		{
			name:        "gmail with dots and suffix",
			email:       "j.o.h.n+promo@gmail.com",
			expected:    "john@gmail.com",
			expectError: false,
		},
		{
			name:        "outlook with suffix",
			email:       "john+spam@outlook.com",
			expected:    "john@outlook.com",
			expectError: false,
		},
		{
			name:        "outlook with dots preserved",
			email:       "john.doe@outlook.com",
			expected:    "john.doe@outlook.com",
			expectError: false,
		},
		{
			name:        "company email",
			email:       "John.Doe+Test@Company.COM",
			expected:    "john.doe@company.com",
			expectError: false,
		},
		{
			name:        "empty string allowed",
			email:       "",
			expected:    "",
			expectError: false,
		},

		// Disposable email rejections
		{
			name:        "tempmail blocked",
			email:       "user@tempmail.com",
			expected:    "",
			expectError: true,
			errorMsg:    "disposable email addresses are not allowed",
		},
		{
			name:        "mailinator blocked",
			email:       "test@mailinator.com",
			expected:    "",
			expectError: true,
			errorMsg:    "disposable email addresses are not allowed",
		},
		{
			name:        "guerrillamail blocked",
			email:       "anon@guerrillamail.com",
			expected:    "",
			expectError: true,
			errorMsg:    "disposable email addresses are not allowed",
		},
		{
			name:        "10minutemail blocked",
			email:       "temp@10minutemail.com",
			expected:    "",
			expectError: true,
			errorMsg:    "disposable email addresses are not allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ValidateAndNormalizeEmail(tt.email)

			if tt.expectError {
				if err == nil {
					t.Errorf("ValidateAndNormalizeEmail(%q) expected error, got nil", tt.email)
				} else if err.Error() != tt.errorMsg {
					t.Errorf("ValidateAndNormalizeEmail(%q) error = %q, want %q", tt.email, err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateAndNormalizeEmail(%q) unexpected error: %v", tt.email, err)
				}
				if result != tt.expected {
					t.Errorf("ValidateAndNormalizeEmail(%q) = %q, want %q", tt.email, result, tt.expected)
				}
			}
		})
	}
}

// TestEmailNormalizationConsistency ensures the same user can't create multiple accounts
func TestEmailNormalizationConsistency(t *testing.T) {
	// All these Gmail variations should normalize to the same address
	gmailVariants := []string{
		"john@gmail.com",
		"j.o.h.n@gmail.com",
		"john+promo@gmail.com",
		"j.o.h.n+spam@gmail.com",
		"JOHN@Gmail.com",
		"J.O.H.N+Newsletter@GMAIL.COM",
		"john@googlemail.com",
		"j.o.h.n@googlemail.com",
	}

	expectedGmail := "john@gmail.com"
	for _, variant := range gmailVariants {
		result := NormalizeEmail(variant)
		if result != expectedGmail {
			t.Errorf("Gmail variant %q normalized to %q, expected %q", variant, result, expectedGmail)
		}
	}

	// All these Outlook variations should normalize to john@outlook.com
	// Note: dots are PRESERVED for Outlook!
	outlookVariantsToJohn := []string{
		"john@outlook.com",
		"john+promo@outlook.com",
		"john+spam@outlook.com",
		"JOHN@Outlook.com",
		"John+Newsletter@OUTLOOK.COM",
	}

	expectedOutlook := "john@outlook.com"
	for _, variant := range outlookVariantsToJohn {
		result := NormalizeEmail(variant)
		if result != expectedOutlook {
			t.Errorf("Outlook variant %q normalized to %q, expected %q", variant, result, expectedOutlook)
		}
	}

	// These Outlook emails with dots should normalize differently (dots preserved!)
	// john@outlook.com != john.doe@outlook.com
	outlookJohn := NormalizeEmail("john@outlook.com")
	outlookJohnDoe := NormalizeEmail("john.doe@outlook.com")
	if outlookJohn == outlookJohnDoe {
		t.Errorf("Outlook should preserve dots: %q should NOT equal %q", outlookJohn, outlookJohnDoe)
	}
}

// TestPlusSuffixAllProviders verifies +suffix is removed for all providers
func TestPlusSuffixAllProviders(t *testing.T) {
	providers := map[string]string{
		"gmail.com":      "user@gmail.com",
		"outlook.com":    "user@outlook.com",
		"hotmail.com":    "user@hotmail.com",
		"yahoo.com":      "user@yahoo.com",
		"protonmail.com": "user@protonmail.com",
		"icloud.com":     "user@icloud.com",
		"fastmail.com":   "user@fastmail.com",
		"company.co.id":  "user@company.co.id",
		"university.edu": "user@university.edu",
	}

	for domain, expected := range providers {
		input := "user+spam@" + domain
		result := NormalizeEmail(input)
		if result != expected {
			t.Errorf("Provider %s: NormalizeEmail(%q) = %q, want %q", domain, input, result, expected)
		}
	}
}
