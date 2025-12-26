package dateparse

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Parse parses a date string and returns a time.Time.
// Supported formats:
//   - ISO date: 2025-01-15
//   - Keywords: today, tomorrow
//   - Weekday: monday, tuesday, ..., sunday
//   - Next weekday: next monday, next tuesday, ...
//   - Relative: +3d, +1w, +2m (days, weeks, months)
func Parse(s string) (time.Time, error) {
	return ParseFrom(s, time.Now())
}

// ParseFrom parses a date string relative to a given reference time.
func ParseFrom(s string, now time.Time) (time.Time, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// ISO date: 2025-01-15
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t, nil
	}

	// Keywords
	switch s {
	case "today":
		return today, nil
	case "tomorrow":
		return today.AddDate(0, 0, 1), nil
	}

	// Relative: +3d, +1w, +2m
	if relTime, ok := parseRelative(s, today); ok {
		return relTime, nil
	}

	// Weekday or "next weekday"
	if weekday, ok := parseWeekday(s); ok {
		return nextWeekday(today, weekday), nil
	}

	return time.Time{}, fmt.Errorf("cannot parse date: %s", s)
}

func parseRelative(s string, base time.Time) (time.Time, bool) {
	re := regexp.MustCompile(`^\+(\d+)([dwm])$`)
	matches := re.FindStringSubmatch(s)
	if matches == nil {
		return time.Time{}, false
	}

	n, _ := strconv.Atoi(matches[1])
	unit := matches[2]

	switch unit {
	case "d":
		return base.AddDate(0, 0, n), true
	case "w":
		return base.AddDate(0, 0, n*7), true
	case "m":
		return base.AddDate(0, n, 0), true
	}

	return time.Time{}, false
}

func parseWeekday(s string) (time.Weekday, bool) {
	s = strings.TrimPrefix(s, "next ")

	weekdays := map[string]time.Weekday{
		"sunday":    time.Sunday,
		"monday":    time.Monday,
		"tuesday":   time.Tuesday,
		"wednesday": time.Wednesday,
		"thursday":  time.Thursday,
		"friday":    time.Friday,
		"saturday":  time.Saturday,
	}

	if wd, ok := weekdays[s]; ok {
		return wd, true
	}
	return time.Sunday, false
}

func nextWeekday(from time.Time, target time.Weekday) time.Time {
	daysUntil := int(target) - int(from.Weekday())
	if daysUntil <= 0 {
		daysUntil += 7
	}
	return from.AddDate(0, 0, daysUntil)
}
