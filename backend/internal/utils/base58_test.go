package utils

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUUIDToBase58_Roundtrip(t *testing.T) {
	// Test with multiple random UUIDs
	for i := 0; i < 100; i++ {
		original, err := uuid.NewV7()
		require.NoError(t, err)

		encoded := UUIDToBase58(original)
		decoded, err := Base58ToUUID(encoded)

		assert.NoError(t, err)
		assert.Equal(t, original, decoded, "Roundtrip failed for UUID: %s", original)
	}
}

func TestUUIDToBase58_Length(t *testing.T) {
	// UUIDv7 should encode to 21-22 characters
	for i := 0; i < 100; i++ {
		u, _ := uuid.NewV7()
		encoded := UUIDToBase58(u)
		assert.GreaterOrEqual(t, len(encoded), 21, "Encoded length too short: %d", len(encoded))
		assert.LessOrEqual(t, len(encoded), 22, "Encoded length too long: %d", len(encoded))
	}
}

func TestUUIDToBase58_KnownValues(t *testing.T) {
	tests := []struct {
		name     string
		uuid     string
		expected string // Manually verified
	}{
		{
			name:     "Nil UUID",
			uuid:     "00000000-0000-0000-0000-000000000000",
			expected: "1111111111111111", // 16 bytes of zeros = 16 '1's
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := uuid.Parse(tt.uuid)
			require.NoError(t, err)

			encoded := UUIDToBase58(u)
			assert.Equal(t, tt.expected, encoded)

			// Verify roundtrip
			decoded, err := Base58ToUUID(encoded)
			assert.NoError(t, err)
			assert.Equal(t, u, decoded)
		})
	}
}

func TestBase58ToUUID_InvalidInput(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "Empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "Invalid character 0",
			input:   "0ABC123",
			wantErr: true,
		},
		{
			name:    "Invalid character O",
			input:   "OABC123",
			wantErr: true,
		},
		{
			name:    "Invalid character I",
			input:   "IABC123",
			wantErr: true,
		},
		{
			name:    "Invalid character l",
			input:   "lABC123",
			wantErr: true,
		},
		{
			name:    "Special characters",
			input:   "ABC!@#123",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Base58ToUUID(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestIsValidBase58(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"Valid base58", "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", true},
		{"Valid short", "abc123XYZ", true},
		{"Empty string", "", false},
		{"Contains 0", "abc0123", false},
		{"Contains O", "abcO123", false},
		{"Contains I", "abcI123", false},
		{"Contains l", "abcl123", false},
		{"Contains space", "abc 123", false},
		{"Contains special", "abc!123", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidBase58(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsBase58UUID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"Valid 22 chars", "2n9BKjVcFzVW0ZvVL0Fa5K", false}, // Contains 0
		{"Valid 22 chars correct", "2n9BKjVcFzVWaZvVLaFa5K", true},
		{"Valid 21 chars", "n9BKjVcFzVWaZvVLaFa5K", true},
		{"Too short 20", "9BKjVcFzVWaZvVLaFa5K", false},
		{"Too long 23", "2n9BKjVcFzVWaZvVLaFa5Ka", false},
		{"Invalid chars", "2n9BKjVcFzVW0ZvVL0Fa5K", false}, // Contains 0
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsBase58UUID(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestUUIDToBase58_Deterministic(t *testing.T) {
	// Same UUID should always produce same Base58
	u, _ := uuid.Parse("550e8400-e29b-41d4-a716-446655440000")
	encoded1 := UUIDToBase58(u)
	encoded2 := UUIDToBase58(u)
	assert.Equal(t, encoded1, encoded2)
}

func TestUUIDToBase58_URLSafe(t *testing.T) {
	// All Base58 characters are URL-safe (no encoding needed)
	for i := 0; i < 10; i++ {
		u, _ := uuid.NewV7()
		encoded := UUIDToBase58(u)

		for _, c := range encoded {
			// Check character is alphanumeric
			isAlphaNum := (c >= 'a' && c <= 'z') ||
				(c >= 'A' && c <= 'Z') ||
				(c >= '0' && c <= '9')
			assert.True(t, isAlphaNum, "Character %c is not URL-safe", c)
		}
	}
}

func BenchmarkUUIDToBase58(b *testing.B) {
	u, _ := uuid.NewV7()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		UUIDToBase58(u)
	}
}

func BenchmarkBase58ToUUID(b *testing.B) {
	u, _ := uuid.NewV7()
	encoded := UUIDToBase58(u)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Base58ToUUID(encoded)
	}
}
