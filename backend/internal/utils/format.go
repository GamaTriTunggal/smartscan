package utils

import (
	"strconv"
	"strings"
)

// FormatThousands renders an integer with comma thousand separators.
// Used in user-facing error / flash messages so numbers like 5000000 read as "5,000,000".
//
// Locale note: this is English-style (comma separator). When the app gains i18n support,
// this should be replaced with a locale-aware formatter (e.g. golang.org/x/text/message).
func FormatThousands(n int) string {
	s := strconv.Itoa(n)
	negative := false
	if strings.HasPrefix(s, "-") {
		negative = true
		s = s[1:]
	}

	if len(s) <= 3 {
		if negative {
			return "-" + s
		}
		return s
	}

	var b strings.Builder
	if negative {
		b.WriteByte('-')
	}
	// Insert commas from the right every three digits.
	first := len(s) % 3
	if first > 0 {
		b.WriteString(s[:first])
		if len(s) > first {
			b.WriteByte(',')
		}
	}
	for i := first; i < len(s); i += 3 {
		b.WriteString(s[i : i+3])
		if i+3 < len(s) {
			b.WriteByte(',')
		}
	}
	return b.String()
}
