package recurparse

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Rule represents a parsed recurrence rule.
type Rule struct {
	Interval int      `json:"interval"`          // e.g., 1, 2, 3
	Unit     string   `json:"unit"`              // "day", "week", "month", "year"
	Weekdays []string `json:"weekdays,omitempty"` // e.g., ["mon", "wed", "fri"]
	Day      int      `json:"day,omitempty"`      // day of month (1-31)
}

// Type indicates whether recurrence is fixed (schedule-based) or relative (from completion).
type Type string

const (
	TypeFixed    Type = "fixed"
	TypeRelative Type = "relative"
)

// ParseResult contains the parsed rule and its type.
type ParseResult struct {
	Rule *Rule
	Type Type
}

// Parse parses a natural language recurrence string.
// Returns the rule and whether it's fixed or relative.
//
// Supported formats:
//   - daily, weekly, monthly, yearly, biweekly
//   - every N days/weeks/months/years
//   - every monday, every mon,wed,fri
//   - every 1st, every 15th (day of month)
//   - 3d after done, 2w after done (relative)
func Parse(s string) (*ParseResult, error) {
	s = strings.TrimSpace(strings.ToLower(s))

	// Check for relative pattern: "Nd after done" or "Nw after done"
	if result, ok := parseRelative(s); ok {
		return result, nil
	}

	// Fixed patterns
	if result, ok := parseKeyword(s); ok {
		return result, nil
	}

	if result, ok := parseEveryInterval(s); ok {
		return result, nil
	}

	if result, ok := parseEveryWeekday(s); ok {
		return result, nil
	}

	if result, ok := parseEveryDayOfMonth(s); ok {
		return result, nil
	}

	return nil, fmt.Errorf("cannot parse recurrence: %s", s)
}

// ToJSON converts a Rule to its JSON representation.
func (r *Rule) ToJSON() (string, error) {
	b, err := json.Marshal(r)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// FromJSON parses a JSON string into a Rule.
func FromJSON(s string) (*Rule, error) {
	var r Rule
	if err := json.Unmarshal([]byte(s), &r); err != nil {
		return nil, err
	}
	return &r, nil
}

// Format returns a human-readable string for the rule.
func (r *Rule) Format() string {
	switch r.Unit {
	case "day":
		if r.Interval == 1 {
			return "daily"
		}
		return fmt.Sprintf("every %d days", r.Interval)
	case "week":
		if len(r.Weekdays) > 0 {
			return "every " + strings.Join(r.Weekdays, ",")
		}
		if r.Interval == 1 {
			return "weekly"
		}
		if r.Interval == 2 {
			return "biweekly"
		}
		return fmt.Sprintf("every %d weeks", r.Interval)
	case "month":
		if r.Day > 0 {
			return fmt.Sprintf("every %s", ordinal(r.Day))
		}
		if r.Interval == 1 {
			return "monthly"
		}
		return fmt.Sprintf("every %d months", r.Interval)
	case "year":
		if r.Interval == 1 {
			return "yearly"
		}
		return fmt.Sprintf("every %d years", r.Interval)
	}
	return "unknown"
}

// parseRelative handles "3d after done", "2w after done", "1 week after completion"
func parseRelative(s string) (*ParseResult, bool) {
	// Match patterns like "3d after done", "2w after completion"
	re := regexp.MustCompile(`^(\d+)\s*([dwmy])\s+after\s+(done|completion)$`)
	if matches := re.FindStringSubmatch(s); matches != nil {
		n, _ := strconv.Atoi(matches[1])
		unit := unitFromShort(matches[2])
		return &ParseResult{
			Rule: &Rule{Interval: n, Unit: unit},
			Type: TypeRelative,
		}, true
	}

	// Match patterns like "1 week after done", "2 days after completion"
	re2 := regexp.MustCompile(`^(\d+)\s+(day|days|week|weeks|month|months|year|years)\s+after\s+(done|completion)$`)
	if matches := re2.FindStringSubmatch(s); matches != nil {
		n, _ := strconv.Atoi(matches[1])
		unit := normalizeUnit(matches[2])
		return &ParseResult{
			Rule: &Rule{Interval: n, Unit: unit},
			Type: TypeRelative,
		}, true
	}

	// Match "after N days/weeks"
	re3 := regexp.MustCompile(`^after\s+(\d+)\s+(day|days|week|weeks|month|months|year|years)$`)
	if matches := re3.FindStringSubmatch(s); matches != nil {
		n, _ := strconv.Atoi(matches[1])
		unit := normalizeUnit(matches[2])
		return &ParseResult{
			Rule: &Rule{Interval: n, Unit: unit},
			Type: TypeRelative,
		}, true
	}

	return nil, false
}

// parseKeyword handles daily, weekly, monthly, yearly, biweekly
func parseKeyword(s string) (*ParseResult, bool) {
	keywords := map[string]*Rule{
		"daily":    {Interval: 1, Unit: "day"},
		"weekly":   {Interval: 1, Unit: "week"},
		"monthly":  {Interval: 1, Unit: "month"},
		"yearly":   {Interval: 1, Unit: "year"},
		"biweekly": {Interval: 2, Unit: "week"},
	}

	if rule, ok := keywords[s]; ok {
		return &ParseResult{Rule: rule, Type: TypeFixed}, true
	}
	return nil, false
}

// parseEveryInterval handles "every N days/weeks/months/years" or "every day/week"
func parseEveryInterval(s string) (*ParseResult, bool) {
	// "every day", "every week", etc.
	re1 := regexp.MustCompile(`^every\s+(day|week|month|year)$`)
	if matches := re1.FindStringSubmatch(s); matches != nil {
		return &ParseResult{
			Rule: &Rule{Interval: 1, Unit: matches[1]},
			Type: TypeFixed,
		}, true
	}

	// "every 2 days", "every 3 weeks", etc.
	re2 := regexp.MustCompile(`^every\s+(\d+)\s+(day|days|week|weeks|month|months|year|years)$`)
	if matches := re2.FindStringSubmatch(s); matches != nil {
		n, _ := strconv.Atoi(matches[1])
		unit := normalizeUnit(matches[2])
		return &ParseResult{
			Rule: &Rule{Interval: n, Unit: unit},
			Type: TypeFixed,
		}, true
	}

	return nil, false
}

// parseEveryWeekday handles "every monday", "every mon,wed,fri"
func parseEveryWeekday(s string) (*ParseResult, bool) {
	if !strings.HasPrefix(s, "every ") {
		return nil, false
	}
	rest := strings.TrimPrefix(s, "every ")

	// Check if it's a comma-separated list of weekdays
	parts := strings.Split(rest, ",")
	var weekdays []string

	for _, p := range parts {
		p = strings.TrimSpace(p)
		if wd := normalizeWeekday(p); wd != "" {
			weekdays = append(weekdays, wd)
		} else {
			return nil, false
		}
	}

	if len(weekdays) > 0 {
		return &ParseResult{
			Rule: &Rule{Interval: 1, Unit: "week", Weekdays: weekdays},
			Type: TypeFixed,
		}, true
	}

	return nil, false
}

// parseEveryDayOfMonth handles "every 1st", "every 15th"
func parseEveryDayOfMonth(s string) (*ParseResult, bool) {
	re := regexp.MustCompile(`^every\s+(\d+)(st|nd|rd|th)$`)
	if matches := re.FindStringSubmatch(s); matches != nil {
		day, _ := strconv.Atoi(matches[1])
		if day >= 1 && day <= 31 {
			return &ParseResult{
				Rule: &Rule{Interval: 1, Unit: "month", Day: day},
				Type: TypeFixed,
			}, true
		}
	}
	return nil, false
}

// unitFromShort converts short unit codes to full names.
func unitFromShort(short string) string {
	switch short {
	case "d":
		return "day"
	case "w":
		return "week"
	case "m":
		return "month"
	case "y":
		return "year"
	}
	return short
}

// normalizeUnit converts plural units to singular.
func normalizeUnit(s string) string {
	s = strings.TrimSuffix(s, "s")
	return s
}

// normalizeWeekday converts weekday names to short form.
func normalizeWeekday(s string) string {
	weekdays := map[string]string{
		"monday": "mon", "mon": "mon",
		"tuesday": "tue", "tue": "tue",
		"wednesday": "wed", "wed": "wed",
		"thursday": "thu", "thu": "thu",
		"friday": "fri", "fri": "fri",
		"saturday": "sat", "sat": "sat",
		"sunday": "sun", "sun": "sun",
	}
	return weekdays[s]
}

// ordinal returns the ordinal string for a number (1st, 2nd, 3rd, etc.)
func ordinal(n int) string {
	suffix := "th"
	switch n % 10 {
	case 1:
		if n%100 != 11 {
			suffix = "st"
		}
	case 2:
		if n%100 != 12 {
			suffix = "nd"
		}
	case 3:
		if n%100 != 13 {
			suffix = "rd"
		}
	}
	return fmt.Sprintf("%d%s", n, suffix)
}

// NextOccurrence calculates the next occurrence date based on the rule.
// For fixed rules, it calculates from today.
// For relative rules, fromDate should be the completion date.
func NextOccurrence(rule *Rule, recurrenceType Type, fromDate time.Time) time.Time {
	// Normalize to start of day
	from := time.Date(fromDate.Year(), fromDate.Month(), fromDate.Day(), 0, 0, 0, 0, fromDate.Location())
	today := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, fromDate.Location())

	switch recurrenceType {
	case TypeRelative:
		// For relative, add interval from the fromDate (completion date)
		return addInterval(from, rule)

	case TypeFixed:
		// For fixed, find the next valid occurrence after today
		if len(rule.Weekdays) > 0 {
			return nextWeekdayOccurrence(today, rule.Weekdays)
		}
		if rule.Day > 0 {
			return nextDayOfMonth(today, rule.Day)
		}
		// Simple interval: find next occurrence after today
		return addInterval(today, rule)
	}

	return addInterval(from, rule)
}

// addInterval adds the rule's interval to a date.
func addInterval(from time.Time, rule *Rule) time.Time {
	switch rule.Unit {
	case "day":
		return from.AddDate(0, 0, rule.Interval)
	case "week":
		return from.AddDate(0, 0, rule.Interval*7)
	case "month":
		return from.AddDate(0, rule.Interval, 0)
	case "year":
		return from.AddDate(rule.Interval, 0, 0)
	}
	return from
}

// nextWeekdayOccurrence finds the next occurrence of any of the given weekdays.
func nextWeekdayOccurrence(from time.Time, weekdays []string) time.Time {
	weekdayMap := map[string]time.Weekday{
		"sun": time.Sunday, "mon": time.Monday, "tue": time.Tuesday,
		"wed": time.Wednesday, "thu": time.Thursday, "fri": time.Friday,
		"sat": time.Saturday,
	}

	// Find the nearest upcoming weekday
	minDays := 8
	for _, wd := range weekdays {
		target := weekdayMap[wd]
		days := int(target) - int(from.Weekday())
		if days <= 0 {
			days += 7
		}
		if days < minDays {
			minDays = days
		}
	}

	return from.AddDate(0, 0, minDays)
}

// nextDayOfMonth finds the next occurrence of a specific day of month.
func nextDayOfMonth(from time.Time, day int) time.Time {
	// If we're before that day this month, use this month
	if from.Day() < day {
		result := time.Date(from.Year(), from.Month(), day, 0, 0, 0, 0, from.Location())
		// Check if the day is valid for this month
		if result.Day() == day {
			return result
		}
	}

	// Otherwise, try next month
	next := from.AddDate(0, 1, 0)
	result := time.Date(next.Year(), next.Month(), day, 0, 0, 0, 0, from.Location())

	// If day overflows (e.g., 31st in a 30-day month), use last day of month
	if result.Day() != day {
		// Go to first of next-next month, then back one day
		result = time.Date(next.Year(), next.Month()+1, 1, 0, 0, 0, 0, from.Location()).AddDate(0, 0, -1)
	}

	return result
}
