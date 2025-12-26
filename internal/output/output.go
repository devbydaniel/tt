package output

import (
	"fmt"
	"io"
	"time"

	"github.com/devbydaniel/t/internal/domain/area"
	"github.com/devbydaniel/t/internal/domain/project"
	"github.com/devbydaniel/t/internal/domain/task"
	"github.com/devbydaniel/t/internal/recurparse"
)

type Formatter struct {
	w io.Writer
}

func NewFormatter(w io.Writer) *Formatter {
	return &Formatter{w: w}
}

func (f *Formatter) TaskCreated(t *task.Task) {
	fmt.Fprintf(f.w, "Created task #%d: %s\n", t.ID, t.Title)
}

func (f *Formatter) TaskList(tasks []task.Task) {
	if len(tasks) == 0 {
		fmt.Fprintln(f.w, "No tasks")
		return
	}

	for _, t := range tasks {
		dateStr := formatTaskDate(t.PlannedDate, t.DueDate)
		recurIndicator := formatRecurIndicator(&t)
		tagIndicator := formatTagIndicator(t.Tags)
		title := t.Title + recurIndicator + tagIndicator

		if dateStr != "" {
			fmt.Fprintf(f.w, "%d  %-10s  %s\n", t.ID, dateStr, title)
		} else {
			fmt.Fprintf(f.w, "%d  %s\n", t.ID, title)
		}
	}
}

func formatRecurIndicator(t *task.Task) string {
	if t.RecurType == nil {
		return ""
	}
	if t.RecurPaused {
		return " [paused]"
	}
	return " [recurs]"
}

func formatTagIndicator(tags []string) string {
	if len(tags) == 0 {
		return ""
	}
	result := ""
	for _, tag := range tags {
		result += " #" + tag
	}
	return result
}

func formatTaskDate(planned, due *time.Time) string {
	now := time.Now()
	todayYear, todayMonth, todayDay := now.Date()

	// Prefer planned date, fall back to due date
	var d *time.Time
	if planned != nil {
		d = planned
	} else if due != nil {
		d = due
	}

	if d == nil {
		return ""
	}

	dateYear, dateMonth, dateDay := d.Date()

	// Compare dates without time component
	if dateYear == todayYear && dateMonth == todayMonth && dateDay == todayDay {
		return "today"
	}

	tomorrow := now.AddDate(0, 0, 1)
	tomorrowYear, tomorrowMonth, tomorrowDay := tomorrow.Date()
	if dateYear == tomorrowYear && dateMonth == tomorrowMonth && dateDay == tomorrowDay {
		return "tomorrow"
	}

	// Check if overdue (before today)
	today := time.Date(todayYear, todayMonth, todayDay, 0, 0, 0, 0, time.UTC)
	dateOnly := time.Date(dateYear, dateMonth, dateDay, 0, 0, 0, 0, time.UTC)
	if dateOnly.Before(today) {
		return "overdue"
	}

	// Within 7 days, show weekday
	weekFromNow := today.AddDate(0, 0, 7)
	if dateOnly.Before(weekFromNow) {
		return d.Format("Mon")
	}

	// Show date
	return d.Format("Jan 2")
}

func (f *Formatter) TasksCompleted(results []task.CompleteResult) {
	for _, r := range results {
		fmt.Fprintf(f.w, "Completed #%d: %s\n", r.Completed.ID, r.Completed.Title)
		if r.NextTask != nil {
			nextDate := r.NextTask.PlannedDate
			if nextDate == nil {
				nextDate = r.NextTask.DueDate
			}
			if nextDate != nil {
				fmt.Fprintf(f.w, "  Next: #%d on %s\n", r.NextTask.ID, nextDate.Format("Jan 2"))
			} else {
				fmt.Fprintf(f.w, "  Next: #%d\n", r.NextTask.ID)
			}
		}
	}
}

func (f *Formatter) TasksDeleted(tasks []task.Task) {
	for _, t := range tasks {
		fmt.Fprintf(f.w, "Deleted #%d: %s\n", t.ID, t.Title)
	}
}

func (f *Formatter) Logbook(tasks []task.Task) {
	if len(tasks) == 0 {
		fmt.Fprintln(f.w, "No completed tasks")
		return
	}

	for _, t := range tasks {
		completedAt := ""
		if t.CompletedAt != nil {
			completedAt = t.CompletedAt.Format("2006-01-02 15:04")
		}
		fmt.Fprintf(f.w, "%d  %s  %s\n", t.ID, completedAt, t.Title)
	}
}

func (f *Formatter) AreaCreated(a *area.Area) {
	fmt.Fprintf(f.w, "Created area: %s\n", a.Name)
}

func (f *Formatter) AreaList(areas []area.Area) {
	if len(areas) == 0 {
		fmt.Fprintln(f.w, "No areas")
		return
	}

	for _, a := range areas {
		fmt.Fprintln(f.w, a.Name)
	}
}

func (f *Formatter) AreaDeleted(a *area.Area) {
	fmt.Fprintf(f.w, "Deleted area: %s\n", a.Name)
}

func (f *Formatter) ProjectCreated(p *project.Project) {
	fmt.Fprintf(f.w, "Created project: %s\n", p.Name)
}

func (f *Formatter) ProjectList(projects []project.Project) {
	if len(projects) == 0 {
		fmt.Fprintln(f.w, "No projects")
		return
	}

	for _, p := range projects {
		fmt.Fprintln(f.w, p.Name)
	}
}

func (f *Formatter) ProjectDeleted(p *project.Project) {
	fmt.Fprintf(f.w, "Deleted project: %s\n", p.Name)
}

func (f *Formatter) TaskPlannedDateSet(t *task.Task) {
	if t.PlannedDate != nil {
		fmt.Fprintf(f.w, "Planned #%d for %s: %s\n", t.ID, t.PlannedDate.Format("Jan 2"), t.Title)
	} else {
		fmt.Fprintf(f.w, "Cleared planned date for #%d: %s\n", t.ID, t.Title)
	}
}

func (f *Formatter) TaskDueDateSet(t *task.Task) {
	if t.DueDate != nil {
		fmt.Fprintf(f.w, "Due #%d on %s: %s\n", t.ID, t.DueDate.Format("Jan 2"), t.Title)
	} else {
		fmt.Fprintf(f.w, "Cleared due date for #%d: %s\n", t.ID, t.Title)
	}
}

func (f *Formatter) TaskRecurrenceSet(t *task.Task) {
	if t.RecurRule != nil {
		rule, err := recurparse.FromJSON(*t.RecurRule)
		if err != nil {
			fmt.Fprintf(f.w, "Set recurrence for #%d: %s\n", t.ID, t.Title)
			return
		}
		fmt.Fprintf(f.w, "Set recurrence for #%d (%s): %s\n", t.ID, rule.Format(), t.Title)
	} else {
		fmt.Fprintf(f.w, "Cleared recurrence for #%d: %s\n", t.ID, t.Title)
	}
}

func (f *Formatter) TaskRecurrencePaused(t *task.Task) {
	fmt.Fprintf(f.w, "Paused recurrence for #%d: %s\n", t.ID, t.Title)
}

func (f *Formatter) TaskRecurrenceResumed(t *task.Task) {
	fmt.Fprintf(f.w, "Resumed recurrence for #%d: %s\n", t.ID, t.Title)
}

func (f *Formatter) TaskRecurrenceEndSet(t *task.Task) {
	if t.RecurEnd != nil {
		fmt.Fprintf(f.w, "Set recurrence end date for #%d to %s: %s\n", t.ID, t.RecurEnd.Format("Jan 2"), t.Title)
	} else {
		fmt.Fprintf(f.w, "Cleared recurrence end date for #%d: %s\n", t.ID, t.Title)
	}
}

func (f *Formatter) TaskRecurrenceInfo(t *task.Task) {
	if t.RecurRule == nil {
		fmt.Fprintf(f.w, "#%d: %s (no recurrence)\n", t.ID, t.Title)
		return
	}

	rule, err := recurparse.FromJSON(*t.RecurRule)
	ruleStr := "unknown"
	if err == nil {
		ruleStr = rule.Format()
	}

	status := ""
	if t.RecurPaused {
		status = " (paused)"
	}

	endStr := ""
	if t.RecurEnd != nil {
		endStr = fmt.Sprintf(" until %s", t.RecurEnd.Format("Jan 2, 2006"))
	}

	fmt.Fprintf(f.w, "#%d: %s\n  Recurs: %s%s%s\n", t.ID, t.Title, ruleStr, endStr, status)
}

func (f *Formatter) TagList(tags []string) {
	if len(tags) == 0 {
		fmt.Fprintln(f.w, "No tags")
		return
	}

	for _, tag := range tags {
		fmt.Fprintln(f.w, tag)
	}
}

func (f *Formatter) TaskTagAdded(t *task.Task, tagName string) {
	fmt.Fprintf(f.w, "Added tag '%s' to #%d: %s\n", tagName, t.ID, t.Title)
}

func (f *Formatter) TaskTagRemoved(t *task.Task, tagName string) {
	fmt.Fprintf(f.w, "Removed tag '%s' from #%d: %s\n", tagName, t.ID, t.Title)
}

func (f *Formatter) TaskEdited(id int64, changes []string) {
	fmt.Fprintf(f.w, "Updated #%d: %s\n", id, joinChanges(changes))
}

func joinChanges(changes []string) string {
	if len(changes) == 0 {
		return "no changes"
	}
	if len(changes) == 1 {
		return changes[0]
	}
	if len(changes) == 2 {
		return changes[0] + " and " + changes[1]
	}
	return fmt.Sprintf("%s, and %s",
		fmt.Sprintf("%s", changes[0:len(changes)-1])[1:len(fmt.Sprintf("%s", changes[0:len(changes)-1]))-1],
		changes[len(changes)-1])
}

func (f *Formatter) TaskDetails(t *task.Task) {
	fmt.Fprintf(f.w, "#%d: %s\n", t.ID, t.Title)

	if t.PlannedDate != nil {
		fmt.Fprintf(f.w, "  Planned: %s\n", t.PlannedDate.Format("Jan 2, 2006"))
	}
	if t.DueDate != nil {
		fmt.Fprintf(f.w, "  Due: %s\n", t.DueDate.Format("Jan 2, 2006"))
	}
	if t.State == task.StateSomeday {
		fmt.Fprintln(f.w, "  State: someday")
	}
	if len(t.Tags) > 0 {
		fmt.Fprintf(f.w, "  Tags: %s\n", formatTagList(t.Tags))
	}
}

func formatTagList(tags []string) string {
	result := ""
	for i, tag := range tags {
		if i > 0 {
			result += ", "
		}
		result += "#" + tag
	}
	return result
}
