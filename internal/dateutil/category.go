package dateutil

import "time"

// Boundaries returns rolling-window date boundaries for date categorization.
func Boundaries() (today, tomorrow, endOfWeek, endOfMonth, endOfQuarter, endOfYear time.Time) { //nolint:gocritic // named returns are clearer here
	now := time.Now()
	y, m, d := now.Date()
	today = time.Date(y, m, d, 0, 0, 0, 0, time.Local)
	tomorrow = today.AddDate(0, 0, 1)
	endOfWeek = today.AddDate(0, 0, 7)     // rolling 7 days
	endOfMonth = today.AddDate(0, 0, 30)   // rolling 30 days
	endOfQuarter = today.AddDate(0, 0, 90) // rolling 90 days
	endOfYear = today.AddDate(0, 0, 365)   // rolling 365 days
	return
}

// Category determines which date category a task belongs to based on its
// planned and due dates. Planned dates in the past are treated as "Today"
// rather than "Overdue".
func Category(planned, due *time.Time, today, tomorrow, endOfWeek, endOfMonth, endOfQuarter, endOfYear time.Time) string {
	var d *time.Time
	isPlanned := false
	if planned != nil {
		d = planned
		isPlanned = true
	} else if due != nil {
		d = due
	}

	if d == nil {
		return "No Date"
	}

	dateYear, dateMonth, dateDay := d.Date()
	dateOnly := time.Date(dateYear, dateMonth, dateDay, 0, 0, 0, 0, time.Local)

	if dateOnly.Before(today) {
		if isPlanned {
			return "Today"
		}
		return "Overdue"
	}
	if dateOnly.Equal(today) {
		return "Today"
	}
	if dateOnly.Equal(tomorrow) {
		return "Tomorrow"
	}
	if dateOnly.Before(endOfWeek) || dateOnly.Equal(endOfWeek) {
		return "Next 7 Days"
	}
	if dateOnly.Before(endOfMonth) || dateOnly.Equal(endOfMonth) {
		return "Next 30 Days"
	}
	if dateOnly.Before(endOfQuarter) || dateOnly.Equal(endOfQuarter) {
		return "Next 90 Days"
	}
	if dateOnly.Before(endOfYear) || dateOnly.Equal(endOfYear) {
		return "Next 365 Days"
	}
	return "Later"
}

// OrderedCategories returns the date categories in display order.
func OrderedCategories() []string {
	return []string{"Overdue", "Today", "Tomorrow", "Next 7 Days", "Next 30 Days", "Next 90 Days", "Next 365 Days", "Later", "No Date"}
}
