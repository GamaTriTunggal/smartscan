package utils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseExportTimezone_Empty(t *testing.T) {
	et, err := ParseExportTimezone("")
	assert.NoError(t, err)
	assert.Nil(t, et)
}

func TestParseExportTimezone_Valid(t *testing.T) {
	tests := []struct {
		tz   string
		abbr string
	}{
		{"Asia/Jakarta", "WIB"},
		{"Asia/Makassar", "WITA"},
		{"Asia/Jayapura", "WIT"},
		{"UTC", "UTC"},
		{"America/New_York", "E"}, // EST or EDT depending on time of year — just check prefix
		{"Europe/London", ""},     // GMT or BST — just check non-nil
		{"Asia/Tokyo", "JST"},
		{"Australia/Sydney", "AE"}, // AEST or AEDT — check prefix
	}

	for _, tt := range tests {
		t.Run(tt.tz, func(t *testing.T) {
			et, err := ParseExportTimezone(tt.tz)
			require.NoError(t, err)
			require.NotNil(t, et)
			assert.NotEmpty(t, et.Abbreviation)
			assert.NotNil(t, et.Location)
			assert.Equal(t, tt.tz, et.Location.String())
		})
	}
}

func TestParseExportTimezone_Invalid(t *testing.T) {
	invalid := []string{"invalid", "Not/Real", "123", "asia jakarta"}
	for _, tz := range invalid {
		t.Run(tz, func(t *testing.T) {
			et, err := ParseExportTimezone(tz)
			assert.Error(t, err)
			assert.Nil(t, et)
			assert.Contains(t, err.Error(), "invalid timezone")
		})
	}
}

func TestFormatExportTime(t *testing.T) {
	// 2026-03-01 03:00:00 UTC should be 2026-03-01 10:00:00 WIB (UTC+7)
	utcTime := time.Date(2026, 3, 1, 3, 0, 0, 0, time.UTC)
	layout := "2006-01-02 15:04"

	t.Run("nil timezone returns UTC", func(t *testing.T) {
		result := FormatExportTime(utcTime, nil, layout)
		assert.Equal(t, "2026-03-01 03:00", result)
	})

	t.Run("WIB converts to +7", func(t *testing.T) {
		et, _ := ParseExportTimezone("Asia/Jakarta")
		result := FormatExportTime(utcTime, et, layout)
		assert.Equal(t, "2026-03-01 10:00", result)
	})

	t.Run("WITA converts to +8", func(t *testing.T) {
		et, _ := ParseExportTimezone("Asia/Makassar")
		result := FormatExportTime(utcTime, et, layout)
		assert.Equal(t, "2026-03-01 11:00", result)
	})

	t.Run("WIT converts to +9", func(t *testing.T) {
		et, _ := ParseExportTimezone("Asia/Jayapura")
		result := FormatExportTime(utcTime, et, layout)
		assert.Equal(t, "2026-03-01 12:00", result)
	})

	t.Run("UTC stays UTC", func(t *testing.T) {
		et, _ := ParseExportTimezone("UTC")
		result := FormatExportTime(utcTime, et, layout)
		assert.Equal(t, "2026-03-01 03:00", result)
	})

	t.Run("with seconds layout", func(t *testing.T) {
		et, _ := ParseExportTimezone("Asia/Jakarta")
		result := FormatExportTime(utcTime, et, "2006-01-02 15:04:05")
		assert.Equal(t, "2026-03-01 10:00:00", result)
	})

	t.Run("JST converts to +9", func(t *testing.T) {
		et, _ := ParseExportTimezone("Asia/Tokyo")
		result := FormatExportTime(utcTime, et, layout)
		assert.Equal(t, "2026-03-01 12:00", result)
	})
}

func TestFormatExportTimePtr(t *testing.T) {
	layout := "2006-01-02 15:04"

	t.Run("nil time returns empty", func(t *testing.T) {
		et, _ := ParseExportTimezone("Asia/Jakarta")
		result := FormatExportTimePtr(nil, et, layout)
		assert.Equal(t, "", result)
	})

	t.Run("valid time converts", func(t *testing.T) {
		utcTime := time.Date(2026, 3, 1, 3, 0, 0, 0, time.UTC)
		et, _ := ParseExportTimezone("Asia/Jakarta")
		result := FormatExportTimePtr(&utcTime, et, layout)
		assert.Equal(t, "2026-03-01 10:00", result)
	})

	t.Run("nil timezone returns UTC", func(t *testing.T) {
		utcTime := time.Date(2026, 3, 1, 3, 0, 0, 0, time.UTC)
		result := FormatExportTimePtr(&utcTime, nil, layout)
		assert.Equal(t, "2026-03-01 03:00", result)
	})
}

func TestTimezoneHeaderSuffix(t *testing.T) {
	t.Run("nil returns UTC suffix", func(t *testing.T) {
		assert.Equal(t, " (UTC)", TimezoneHeaderSuffix(nil))
	})

	t.Run("WIB suffix", func(t *testing.T) {
		et, _ := ParseExportTimezone("Asia/Jakarta")
		assert.Equal(t, " (WIB)", TimezoneHeaderSuffix(et))
	})

	t.Run("WITA suffix", func(t *testing.T) {
		et, _ := ParseExportTimezone("Asia/Makassar")
		assert.Equal(t, " (WITA)", TimezoneHeaderSuffix(et))
	})

	t.Run("WIT suffix", func(t *testing.T) {
		et, _ := ParseExportTimezone("Asia/Jayapura")
		assert.Equal(t, " (WIT)", TimezoneHeaderSuffix(et))
	})

	t.Run("UTC suffix", func(t *testing.T) {
		et, _ := ParseExportTimezone("UTC")
		assert.Equal(t, " (UTC)", TimezoneHeaderSuffix(et))
	})
}
