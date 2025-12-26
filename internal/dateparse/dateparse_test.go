package dateparse

import (
	"testing"
	"time"
)

func TestParse(t *testing.T) {
	// Use a fixed reference date: Wednesday, January 15, 2025
	ref := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)
	today := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		input    string
		expected time.Time
	}{
		// ISO date
		{"2025-01-20", time.Date(2025, 1, 20, 0, 0, 0, 0, time.UTC)},
		{"2025-12-31", time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)},

		// Keywords
		{"today", today},
		{"TODAY", today},
		{"tomorrow", today.AddDate(0, 0, 1)},

		// Relative dates
		{"+1d", today.AddDate(0, 0, 1)},
		{"+3d", today.AddDate(0, 0, 3)},
		{"+1w", today.AddDate(0, 0, 7)},
		{"+2w", today.AddDate(0, 0, 14)},
		{"+1m", today.AddDate(0, 1, 0)},
		{"+3m", today.AddDate(0, 3, 0)},

		// Weekdays (ref is Wednesday Jan 15)
		{"monday", time.Date(2025, 1, 20, 0, 0, 0, 0, time.UTC)},    // next Monday
		{"friday", time.Date(2025, 1, 17, 0, 0, 0, 0, time.UTC)},    // this Friday
		{"wednesday", time.Date(2025, 1, 22, 0, 0, 0, 0, time.UTC)}, // next Wednesday (not today)

		// Next weekday
		{"next monday", time.Date(2025, 1, 20, 0, 0, 0, 0, time.UTC)},
		{"next friday", time.Date(2025, 1, 17, 0, 0, 0, 0, time.UTC)},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseFrom(tt.input, ref)
			if err != nil {
				t.Fatalf("ParseFrom(%q) error: %v", tt.input, err)
			}
			if !got.Equal(tt.expected) {
				t.Errorf("ParseFrom(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestParseError(t *testing.T) {
	tests := []string{
		"invalid",
		"2025-13-01", // invalid month
		"yesterday",
		"++1d",
		"1d",
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			_, err := Parse(input)
			if err == nil {
				t.Errorf("Parse(%q) expected error, got nil", input)
			}
		})
	}
}
