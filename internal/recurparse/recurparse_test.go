package recurparse

import (
	"testing"
	"time"
)

func TestParseKeywords(t *testing.T) {
	tests := []struct {
		input    string
		wantType Type
		wantUnit string
		wantInt  int
	}{
		{"daily", TypeFixed, "day", 1},
		{"weekly", TypeFixed, "week", 1},
		{"monthly", TypeFixed, "month", 1},
		{"yearly", TypeFixed, "year", 1},
		{"biweekly", TypeFixed, "week", 2},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) error = %v", tt.input, err)
			}
			if result.Type != tt.wantType {
				t.Errorf("Type = %v, want %v", result.Type, tt.wantType)
			}
			if result.Rule.Unit != tt.wantUnit {
				t.Errorf("Unit = %v, want %v", result.Rule.Unit, tt.wantUnit)
			}
			if result.Rule.Interval != tt.wantInt {
				t.Errorf("Interval = %v, want %v", result.Rule.Interval, tt.wantInt)
			}
		})
	}
}

func TestParseEveryInterval(t *testing.T) {
	tests := []struct {
		input    string
		wantUnit string
		wantInt  int
	}{
		{"every day", "day", 1},
		{"every 2 days", "day", 2},
		{"every week", "week", 1},
		{"every 3 weeks", "week", 3},
		{"every month", "month", 1},
		{"every 6 months", "month", 6},
		{"every year", "year", 1},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) error = %v", tt.input, err)
			}
			if result.Type != TypeFixed {
				t.Errorf("Type = %v, want fixed", result.Type)
			}
			if result.Rule.Unit != tt.wantUnit {
				t.Errorf("Unit = %v, want %v", result.Rule.Unit, tt.wantUnit)
			}
			if result.Rule.Interval != tt.wantInt {
				t.Errorf("Interval = %v, want %v", result.Rule.Interval, tt.wantInt)
			}
		})
	}
}

func TestParseEveryWeekday(t *testing.T) {
	tests := []struct {
		input        string
		wantWeekdays []string
	}{
		{"every monday", []string{"mon"}},
		{"every mon", []string{"mon"}},
		{"every mon,wed,fri", []string{"mon", "wed", "fri"}},
		{"every tuesday", []string{"tue"}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) error = %v", tt.input, err)
			}
			if result.Type != TypeFixed {
				t.Errorf("Type = %v, want fixed", result.Type)
			}
			if len(result.Rule.Weekdays) != len(tt.wantWeekdays) {
				t.Errorf("Weekdays = %v, want %v", result.Rule.Weekdays, tt.wantWeekdays)
			}
		})
	}
}

func TestParseEveryDayOfMonth(t *testing.T) {
	tests := []struct {
		input   string
		wantDay int
	}{
		{"every 1st", 1},
		{"every 15th", 15},
		{"every 31st", 31},
		{"every 2nd", 2},
		{"every 3rd", 3},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) error = %v", tt.input, err)
			}
			if result.Type != TypeFixed {
				t.Errorf("Type = %v, want fixed", result.Type)
			}
			if result.Rule.Day != tt.wantDay {
				t.Errorf("Day = %v, want %v", result.Rule.Day, tt.wantDay)
			}
		})
	}
}

func TestParseRelative(t *testing.T) {
	tests := []struct {
		input    string
		wantUnit string
		wantInt  int
	}{
		{"3d after done", "day", 3},
		{"2w after done", "week", 2},
		{"1m after done", "month", 1},
		{"3d after completion", "day", 3},
		{"2 weeks after done", "week", 2},
		{"1 day after completion", "day", 1},
		{"after 3 days", "day", 3},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) error = %v", tt.input, err)
			}
			if result.Type != TypeRelative {
				t.Errorf("Type = %v, want relative", result.Type)
			}
			if result.Rule.Unit != tt.wantUnit {
				t.Errorf("Unit = %v, want %v", result.Rule.Unit, tt.wantUnit)
			}
			if result.Rule.Interval != tt.wantInt {
				t.Errorf("Interval = %v, want %v", result.Rule.Interval, tt.wantInt)
			}
		})
	}
}

func TestParseInvalid(t *testing.T) {
	invalids := []string{
		"",
		"invalid",
		"every",
		"every foo",
		"sometimes",
	}

	for _, input := range invalids {
		t.Run(input, func(t *testing.T) {
			_, err := Parse(input)
			if err == nil {
				t.Errorf("Parse(%q) should error", input)
			}
		})
	}
}

func TestRuleJSON(t *testing.T) {
	rule := &Rule{Interval: 1, Unit: "week", Weekdays: []string{"mon", "wed"}}

	json, err := rule.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error = %v", err)
	}

	parsed, err := FromJSON(json)
	if err != nil {
		t.Fatalf("FromJSON() error = %v", err)
	}

	if parsed.Interval != rule.Interval {
		t.Errorf("Interval = %v, want %v", parsed.Interval, rule.Interval)
	}
	if parsed.Unit != rule.Unit {
		t.Errorf("Unit = %v, want %v", parsed.Unit, rule.Unit)
	}
	if len(parsed.Weekdays) != len(rule.Weekdays) {
		t.Errorf("Weekdays = %v, want %v", parsed.Weekdays, rule.Weekdays)
	}
}

func TestRuleFormat(t *testing.T) {
	tests := []struct {
		rule *Rule
		want string
	}{
		{&Rule{Interval: 1, Unit: "day"}, "daily"},
		{&Rule{Interval: 2, Unit: "day"}, "every 2 days"},
		{&Rule{Interval: 1, Unit: "week"}, "weekly"},
		{&Rule{Interval: 2, Unit: "week"}, "biweekly"},
		{&Rule{Interval: 3, Unit: "week"}, "every 3 weeks"},
		{&Rule{Interval: 1, Unit: "month"}, "monthly"},
		{&Rule{Interval: 1, Unit: "month", Day: 15}, "every 15th"},
		{&Rule{Interval: 1, Unit: "year"}, "yearly"},
		{&Rule{Interval: 1, Unit: "week", Weekdays: []string{"mon", "wed"}}, "every mon,wed"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.rule.Format()
			if got != tt.want {
				t.Errorf("Format() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNextOccurrenceFixed(t *testing.T) {
	// For fixed recurrence, next occurrence is calculated from today
	today := time.Now()

	tests := []struct {
		name     string
		rule     *Rule
		wantDays int // days from today
	}{
		{"daily", &Rule{Interval: 1, Unit: "day"}, 1},
		{"every 3 days", &Rule{Interval: 3, Unit: "day"}, 3},
		{"weekly", &Rule{Interval: 1, Unit: "week"}, 7},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// The fromDate is ignored for fixed recurrence (uses today)
			next := NextOccurrence(tt.rule, TypeFixed, today)
			expected := today.AddDate(0, 0, tt.wantDays)

			// Compare just the dates
			if next.Year() != expected.Year() || next.YearDay() != expected.YearDay() {
				t.Errorf("NextOccurrence() = %v, want %v", next.Format("2006-01-02"), expected.Format("2006-01-02"))
			}
		})
	}
}

func TestNextOccurrenceRelative(t *testing.T) {
	// Completion date: 2025-01-15
	completionDate := time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		rule     *Rule
		wantDays int
	}{
		{"3 days after", &Rule{Interval: 3, Unit: "day"}, 3},
		{"1 week after", &Rule{Interval: 1, Unit: "week"}, 7},
		{"2 weeks after", &Rule{Interval: 2, Unit: "week"}, 14},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			next := NextOccurrence(tt.rule, TypeRelative, completionDate)
			expected := completionDate.AddDate(0, 0, tt.wantDays)

			if next.Year() != expected.Year() || next.YearDay() != expected.YearDay() {
				t.Errorf("NextOccurrence() = %v, want %v", next.Format("2006-01-02"), expected.Format("2006-01-02"))
			}
		})
	}
}
