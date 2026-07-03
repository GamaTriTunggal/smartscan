package utils

import (
	"testing"
)

func TestGenerateSecureBytes(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{"3 bytes", 3},
		{"16 bytes", 16},
		{"32 bytes", 32},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bytes, err := GenerateSecureBytes(tt.length)
			if err != nil {
				t.Fatalf("GenerateSecureBytes(%d) failed: %v", tt.length, err)
			}
			if len(bytes) != tt.length {
				t.Errorf("Expected %d bytes, got %d", tt.length, len(bytes))
			}
		})
	}
}

func TestGenerateSecureOTP(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{"4 digit OTP", 4},
		{"6 digit OTP", 6},
		{"8 digit OTP", 8},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			otp, err := GenerateSecureOTP(tt.length)
			if err != nil {
				t.Fatalf("GenerateSecureOTP(%d) failed: %v", tt.length, err)
			}
			if len(otp) != tt.length {
				t.Errorf("Expected OTP length %d, got %d", tt.length, len(otp))
			}
			// Verify all characters are digits
			for _, c := range otp {
				if c < '0' || c > '9' {
					t.Errorf("OTP contains non-digit character: %c", c)
				}
			}
		})
	}
}

func TestGenerateSecureTempPassword(t *testing.T) {
	password, err := GenerateSecureTempPassword(8)
	if err != nil {
		t.Fatalf("GenerateSecureTempPassword failed: %v", err)
	}
	// 8 bytes = 16 hex chars
	if len(password) != 16 {
		t.Errorf("Expected password length 16, got %d", len(password))
	}
	// Verify all characters are hex
	for _, c := range password {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("Password contains non-hex character: %c", c)
		}
	}
}

func TestGenerateRandomBytesWithFallback(t *testing.T) {
	bytes := GenerateRandomBytesWithFallback(16)
	if len(bytes) != 16 {
		t.Errorf("Expected 16 bytes, got %d", len(bytes))
	}
}

func TestGenerateRandomIntWithFallback(t *testing.T) {
	// Run multiple times to ensure range is respected
	for i := 0; i < 100; i++ {
		n := GenerateRandomIntWithFallback(10)
		if n < 0 || n >= 10 {
			t.Errorf("GenerateRandomIntWithFallback(10) returned %d, expected [0,10)", n)
		}
	}
}

func TestGenerateRandomHexWithFallback(t *testing.T) {
	hex := GenerateRandomHexWithFallback(8)
	// 8 bytes = 16 hex chars
	if len(hex) != 16 {
		t.Errorf("Expected hex length 16, got %d", len(hex))
	}
	// Verify all characters are hex
	for _, c := range hex {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("Hex contains non-hex character: %c", c)
		}
	}
}

func TestOTPUniqueness(t *testing.T) {
	// Generate 100 OTPs and ensure they're not all the same
	otps := make(map[string]bool)
	for i := 0; i < 100; i++ {
		otp, err := GenerateSecureOTP(6)
		if err != nil {
			t.Fatalf("GenerateSecureOTP failed: %v", err)
		}
		otps[otp] = true
	}
	// We should have many unique OTPs (statistically nearly 100)
	if len(otps) < 90 {
		t.Errorf("Expected at least 90 unique OTPs, got %d", len(otps))
	}
}
