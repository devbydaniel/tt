package output

import (
	"fmt"
	"io"
	"time"

	"github.com/devbydaniel/t/internal/domain/area"
	"github.com/devbydaniel/t/internal/domain/project"
	"github.com/devbydaniel/t/internal/domain/task"
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
		if dateStr != "" {
			fmt.Fprintf(f.w, "%d  %-10s  %s\n", t.ID, dateStr, t.Title)
		} else {
			fmt.Fprintf(f.w, "%d  %s\n", t.ID, t.Title)
		}
	}
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

func (f *Formatter) TasksCompleted(tasks []task.Task) {
	for _, t := range tasks {
		fmt.Fprintf(f.w, "Completed #%d: %s\n", t.ID, t.Title)
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
