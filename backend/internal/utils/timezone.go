package utils

import (
	"fmt"
	"time"
)

// ExportTimezone holds parsed timezone information for exports.
type ExportTimezone struct {
	Location     *time.Location
	Abbreviation string // e.g., "WIB", "WITA", "WIT", "UTC", "JST"
}

// ParseExportTimezone parses the ?tz query parameter using the IANA timezone database.
// Returns nil, nil if tz is empty (backward compatible — caller uses UTC).
// Returns error if timezone is not a valid IANA timezone identifier.
func ParseExportTimezone(tz string) (*ExportTimezone, error) {
	if tz == "" {
		return nil, nil
	}

	loc, err := time.LoadLocation(tz)
	if err != nil {
		return nil, fmt.Errorf("invalid timezone %q", tz)
	}

	// Get the current abbreviation for this timezone (e.g., "WIB", "JST", "EST")
	abbr, _ := time.Now().In(loc).Zone()

	return &ExportTimezone{
		Location:     loc,
		Abbreviation: abbr,
	}, nil
}

// FormatExportTime formats a time.Time for export in the given timezone.
// If et is nil, formats in UTC (backward compatible).
func FormatExportTime(t time.Time, et *ExportTimezone, layout string) string {
	if et != nil {
		t = t.In(et.Location)
	}
	return t.Format(layout)
}

// FormatExportTimePtr is like FormatExportTime but for *time.Time.
// Returns empty string if t is nil.
func FormatExportTimePtr(t *time.Time, et *ExportTimezone, layout string) string {
	if t == nil {
		return ""
	}
	return FormatExportTime(*t, et, layout)
}

// TimezoneHeaderSuffix returns a suffix like " (WIB)" for column headers.
// Returns " (UTC)" if et is nil.
func TimezoneHeaderSuffix(et *ExportTimezone) string {
	if et == nil {
		return " (UTC)"
	}
	return " (" + et.Abbreviation + ")"
}
