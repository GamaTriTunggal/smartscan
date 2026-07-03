package utils

import "testing"

func TestFormatThousands(t *testing.T) {
	cases := []struct {
		in  int
		out string
	}{
		{0, "0"},
		{5, "5"},
		{100, "100"},
		{999, "999"},
		{1000, "1,000"},
		{1234, "1,234"},
		{12345, "12,345"},
		{123456, "123,456"},
		{1234567, "1,234,567"},
		{5000000, "5,000,000"},
		{1000000000, "1,000,000,000"},
		{-1, "-1"},
		{-1234, "-1,234"},
		{-1000000, "-1,000,000"},
	}
	for _, tc := range cases {
		got := FormatThousands(tc.in)
		if got != tc.out {
			t.Errorf("FormatThousands(%d) = %q, want %q", tc.in, got, tc.out)
		}
	}
}
