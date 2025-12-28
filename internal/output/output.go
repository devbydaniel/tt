package output

import (
	"fmt"
	"io"
	"sort"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/devbydaniel/tt/internal/domain/area"
	"github.com/devbydaniel/tt/internal/domain/project"
	"github.com/devbydaniel/tt/internal/domain/task"
	"github.com/devbydaniel/tt/internal/recurparse"
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

	// Styles
	mutedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	starStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("226"))
	flagStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))

	// Build table rows
	rows := make([][]string, 0, len(tasks))
	for _, t := range tasks {
		// Prefix: red flag for due (precedence), yellow star for planned today or earlier
		prefix := "  "
		if isDueOrOverdue(&t) {
			prefix = flagStyle.Render("⚑") + " "
		} else if isPlannedForToday(&t) {
			prefix = starStyle.Render("★") + " "
		}

		// Scope: project name if present, else area name
		scope := ""
		if t.ProjectName != nil {
			scope = *t.ProjectName
		} else if t.AreaName != nil {
			scope = *t.AreaName
		}

		// Task title with recurrence indicator, dates, and tags
		title := formatTaskTitle(&t)
		if t.PlannedDate != nil {
			title += " " + mutedStyle.Render("📅 "+t.PlannedDate.Format("Jan 2"))
		}
		if t.DueDate != nil {
			title += " " + mutedStyle.Render("⚑ "+t.DueDate.Format("Jan 2"))
		}
		if len(t.Tags) > 0 {
			title += " " + mutedStyle.Render(formatTagsForTable(t.Tags))
		}

		rows = append(rows, []string{
			prefix + fmt.Sprintf("%d", t.ID),
			scope,
			title,
		})
	}

	// Create minimal table (no borders, no headers)
	tbl := table.New().
		Rows(rows...).
		Border(lipgloss.HiddenBorder()).
		StyleFunc(func(row, col int) lipgloss.Style {
			return lipgloss.NewStyle().PaddingRight(2)
		})

	fmt.Fprintln(f.w, tbl.Render())
}

// GroupedTaskList displays tasks grouped by the specified field.
// groupBy can be: "project", "area", "date", or "none" (falls back to TaskList)
func (f *Formatter) GroupedTaskList(tasks []task.Task, groupBy string) {
	if groupBy == "none" || groupBy == "" {
		f.TaskList(tasks)
		return
	}

	if len(tasks) == 0 {
		fmt.Fprintln(f.w, "No tasks")
		return
	}

	switch groupBy {
	case "project":
		f.groupedByProject(tasks)
	case "area":
		f.groupedByArea(tasks)
	case "date":
		f.groupedByDate(tasks)
	default:
		f.TaskList(tasks)
	}
}

// groupedByProject displays tasks grouped by "Area > Project"
// Tasks with area but no project appear under just the area name,
// sorted before "Area > Project" groups (alphabetically, area-only headers come first)
func (f *Formatter) groupedByProject(tasks []task.Task) {
	headerStyle := lipgloss.NewStyle().Bold(true)

	// Group tasks:
	// - No area, no project -> "No Project"
	// - Area but no project -> area name (e.g., "Work")
	// - Project -> "Area > Project" or just "Project"
	noProjectNoAreaTasks := make([]task.Task, 0)
	groups := make(map[string][]task.Task)

	for _, t := range tasks {
		if t.ProjectName == nil {
			if t.AreaName == nil {
				noProjectNoAreaTasks = append(noProjectNoAreaTasks, t)
			} else {
				// Area but no project - use area name as header
				groups[*t.AreaName] = append(groups[*t.AreaName], t)
			}
			continue
		}

		// Build header: "Area > Project" or just "Project"
		header := *t.ProjectName
		if t.AreaName != nil {
			header = *t.AreaName + " > " + *t.ProjectName
		}
		groups[header] = append(groups[header], t)
	}

	// Render: No Project (no area) first
	if len(noProjectNoAreaTasks) > 0 {
		fmt.Fprintln(f.w, headerStyle.Render("No Project"))
		f.renderTaskRows(noProjectNoAreaTasks, 0, false)
		fmt.Fprintln(f.w)
	}

	// Render groups alphabetically
	// Natural sort order puts "Area" before "Area > Project"
	headers := make([]string, 0, len(groups))
	for h := range groups {
		headers = append(headers, h)
	}
	sort.Strings(headers)

	for _, header := range headers {
		fmt.Fprintln(f.w, headerStyle.Render(header))
		f.renderTaskRows(groups[header], 0, false)
		fmt.Fprintln(f.w)
	}
}

// groupedByArea displays tasks grouped by area
func (f *Formatter) groupedByArea(tasks []task.Task) {
	headerStyle := lipgloss.NewStyle().Bold(true)

	// Group by area
	noAreaTasks := make([]task.Task, 0)
	areaGroups := make(map[string][]task.Task)

	for _, t := range tasks {
		if t.AreaName == nil {
			noAreaTasks = append(noAreaTasks, t)
		} else {
			areaGroups[*t.AreaName] = append(areaGroups[*t.AreaName], t)
		}
	}

	// Render: No Area first
	if len(noAreaTasks) > 0 {
		fmt.Fprintln(f.w, headerStyle.Render("No Area"))
		f.renderTaskRows(noAreaTasks, 0, true)
		fmt.Fprintln(f.w)
	}

	// Render areas alphabetically
	areaNames := make([]string, 0, len(areaGroups))
	for name := range areaGroups {
		areaNames = append(areaNames, name)
	}
	sort.Strings(areaNames)

	for _, aName := range areaNames {
		fmt.Fprintln(f.w, headerStyle.Render(aName))
		f.renderTaskRows(areaGroups[aName], 0, true)
		fmt.Fprintln(f.w)
	}
}

// groupedByDate displays tasks grouped by date categories
func (f *Formatter) groupedByDate(tasks []task.Task) {
	headerStyle := lipgloss.NewStyle().Bold(true)

	// Define date categories
	dateGroups := map[string][]task.Task{
		"Overdue":   {},
		"Today":     {},
		"Tomorrow":  {},
		"This Week": {},
		"Next Week": {},
		"Later":     {},
		"No Date":   {},
	}
	orderedCategories := []string{"Overdue", "Today", "Tomorrow", "This Week", "Next Week", "Later", "No Date"}

	now := time.Now()
	todayYear, todayMonth, todayDay := now.Date()
	today := time.Date(todayYear, todayMonth, todayDay, 0, 0, 0, 0, time.Local)
	tomorrow := today.AddDate(0, 0, 1)
	endOfWeek := today.AddDate(0, 0, 7-int(today.Weekday()))
	endOfNextWeek := endOfWeek.AddDate(0, 0, 7)

	for _, t := range tasks {
		category := getDateCategory(t.PlannedDate, t.DueDate, today, tomorrow, endOfWeek, endOfNextWeek)
		dateGroups[category] = append(dateGroups[category], t)
	}

	// Render each category
	for _, category := range orderedCategories {
		if len(dateGroups[category]) > 0 {
			fmt.Fprintln(f.w, headerStyle.Render(category))
			f.renderTaskRows(dateGroups[category], 0, true)
			fmt.Fprintln(f.w)
		}
	}
}

func getDateCategory(planned, due *time.Time, today, tomorrow, endOfWeek, endOfNextWeek time.Time) string {
	var d *time.Time
	if planned != nil {
		d = planned
	} else if due != nil {
		d = due
	}

	if d == nil {
		return "No Date"
	}

	dateYear, dateMonth, dateDay := d.Date()
	dateOnly := time.Date(dateYear, dateMonth, dateDay, 0, 0, 0, 0, time.Local)

	if dateOnly.Before(today) {
		return "Overdue"
	}
	if dateOnly.Equal(today) {
		return "Today"
	}
	if dateOnly.Equal(tomorrow) {
		return "Tomorrow"
	}
	if dateOnly.Before(endOfWeek) || dateOnly.Equal(endOfWeek) {
		return "This Week"
	}
	if dateOnly.Before(endOfNextWeek) || dateOnly.Equal(endOfNextWeek) {
		return "Next Week"
	}
	return "Later"
}

// renderTaskRows renders task rows with optional indentation
func (f *Formatter) renderTaskRows(tasks []task.Task, indent int, showScope bool) {
	mutedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	starStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("226"))
	flagStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))

	rows := make([][]string, 0, len(tasks))
	for _, t := range tasks {
		prefix := "  "
		if isDueOrOverdue(&t) {
			prefix = flagStyle.Render("⚑") + " "
		} else if isPlannedForToday(&t) {
			prefix = starStyle.Render("★") + " "
		}

		scope := ""
		if showScope {
			if t.ProjectName != nil {
				scope = *t.ProjectName
			} else if t.AreaName != nil {
				scope = *t.AreaName
			}
		}

		title := formatTaskTitle(&t)
		if t.PlannedDate != nil {
			title += " " + mutedStyle.Render("📅 "+t.PlannedDate.Format("Jan 2"))
		}
		if t.DueDate != nil {
			title += " " + mutedStyle.Render("⚑ "+t.DueDate.Format("Jan 2"))
		}
		if len(t.Tags) > 0 {
			title += " " + mutedStyle.Render(formatTagsForTable(t.Tags))
		}

		rows = append(rows, []string{
			prefix + fmt.Sprintf("%d", t.ID),
			scope,
			title,
		})
	}

	tbl := table.New().
		Rows(rows...).
		Border(lipgloss.HiddenBorder()).
		BorderTop(false).
		BorderBottom(false).
		BorderHeader(false).
		StyleFunc(func(row, col int) lipgloss.Style {
			style := lipgloss.NewStyle().PaddingRight(2)
			if col == 0 && indent > 0 {
				style = style.PaddingLeft(indent)
			}
			return style
		})

	fmt.Fprint(f.w, tbl.Render())
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

func formatTaskTitle(t *task.Task) string {
	title := t.Title

	// Add recurrence indicator
	title += formatRecurIndicator(t)

	return title
}

func isPlannedForToday(t *task.Task) bool {
	if t.PlannedDate == nil {
		return false
	}
	now := time.Now()
	todayYear, todayMonth, todayDay := now.Date()
	today := time.Date(todayYear, todayMonth, todayDay, 0, 0, 0, 0, time.Local)
	dateYear, dateMonth, dateDay := t.PlannedDate.Date()
	plannedDate := time.Date(dateYear, dateMonth, dateDay, 0, 0, 0, 0, time.Local)
	return !plannedDate.After(today)
}

func isDueOrOverdue(t *task.Task) bool {
	if t.DueDate == nil {
		return false
	}
	now := time.Now()
	todayYear, todayMonth, todayDay := now.Date()
	today := time.Date(todayYear, todayMonth, todayDay, 0, 0, 0, 0, time.Local)
	dateYear, dateMonth, dateDay := t.DueDate.Date()
	dueDate := time.Date(dateYear, dateMonth, dateDay, 0, 0, 0, 0, time.Local)
	return !dueDate.After(today)
}

func formatTagsForTable(tags []string) string {
	if len(tags) == 0 {
		return ""
	}
	result := ""
	for i, tag := range tags {
		if i > 0 {
			result += " "
		}
		result += "#" + tag
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

// GroupedLogbook displays completed tasks grouped by the specified field.
// For date grouping, it uses CompletedAt instead of PlannedDate.
func (f *Formatter) GroupedLogbook(tasks []task.Task, groupBy string) {
	if groupBy == "none" || groupBy == "" {
		f.Logbook(tasks)
		return
	}

	if len(tasks) == 0 {
		fmt.Fprintln(f.w, "No completed tasks")
		return
	}

	switch groupBy {
	case "project":
		f.logbookByProject(tasks)
	case "area":
		f.logbookByArea(tasks)
	case "date":
		f.logbookByDate(tasks)
	default:
		f.Logbook(tasks)
	}
}

func (f *Formatter) logbookByProject(tasks []task.Task) {
	headerStyle := lipgloss.NewStyle().Bold(true)

	noProjectNoAreaTasks := make([]task.Task, 0)
	groups := make(map[string][]task.Task)

	for _, t := range tasks {
		if t.ProjectName == nil {
			if t.AreaName == nil {
				noProjectNoAreaTasks = append(noProjectNoAreaTasks, t)
			} else {
				groups[*t.AreaName] = append(groups[*t.AreaName], t)
			}
			continue
		}

		header := *t.ProjectName
		if t.AreaName != nil {
			header = *t.AreaName + " > " + *t.ProjectName
		}
		groups[header] = append(groups[header], t)
	}

	if len(noProjectNoAreaTasks) > 0 {
		fmt.Fprintln(f.w, headerStyle.Render("No Project"))
		f.renderLogbookRows(noProjectNoAreaTasks)
		fmt.Fprintln(f.w)
	}

	headers := make([]string, 0, len(groups))
	for h := range groups {
		headers = append(headers, h)
	}
	sort.Strings(headers)

	for _, header := range headers {
		fmt.Fprintln(f.w, headerStyle.Render(header))
		f.renderLogbookRows(groups[header])
		fmt.Fprintln(f.w)
	}
}

func (f *Formatter) logbookByArea(tasks []task.Task) {
	headerStyle := lipgloss.NewStyle().Bold(true)

	noAreaTasks := make([]task.Task, 0)
	areaGroups := make(map[string][]task.Task)

	for _, t := range tasks {
		if t.AreaName == nil {
			noAreaTasks = append(noAreaTasks, t)
		} else {
			areaGroups[*t.AreaName] = append(areaGroups[*t.AreaName], t)
		}
	}

	if len(noAreaTasks) > 0 {
		fmt.Fprintln(f.w, headerStyle.Render("No Area"))
		f.renderLogbookRows(noAreaTasks)
		fmt.Fprintln(f.w)
	}

	areaNames := make([]string, 0, len(areaGroups))
	for name := range areaGroups {
		areaNames = append(areaNames, name)
	}
	sort.Strings(areaNames)

	for _, aName := range areaNames {
		fmt.Fprintln(f.w, headerStyle.Render(aName))
		f.renderLogbookRows(areaGroups[aName])
		fmt.Fprintln(f.w)
	}
}

func (f *Formatter) logbookByDate(tasks []task.Task) {
	headerStyle := lipgloss.NewStyle().Bold(true)

	// Group by completed date
	dateGroups := make(map[string][]task.Task)

	for _, t := range tasks {
		dateKey := "Unknown"
		if t.CompletedAt != nil {
			dateKey = t.CompletedAt.Format("2006-01-02")
		}
		dateGroups[dateKey] = append(dateGroups[dateKey], t)
	}

	// Sort dates in reverse order (most recent first)
	dates := make([]string, 0, len(dateGroups))
	for d := range dateGroups {
		dates = append(dates, d)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(dates)))

	for _, date := range dates {
		fmt.Fprintln(f.w, headerStyle.Render(date))
		f.renderLogbookRows(dateGroups[date])
		fmt.Fprintln(f.w)
	}
}

func (f *Formatter) renderLogbookRows(tasks []task.Task) {
	for _, t := range tasks {
		completedAt := ""
		if t.CompletedAt != nil {
			completedAt = t.CompletedAt.Format("15:04")
		}
		fmt.Fprintf(f.w, "  %d  %s  %s\n", t.ID, completedAt, t.Title)
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

	if t.Description != nil && *t.Description != "" {
		fmt.Fprintf(f.w, "  Description: %s\n", *t.Description)
	}
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
