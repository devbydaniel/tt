package output

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/devbydaniel/tt/internal/domain/area"
	"github.com/devbydaniel/tt/internal/domain/project"
	"github.com/devbydaniel/tt/internal/domain/task"
	"github.com/devbydaniel/tt/internal/recurparse"
)

type Formatter struct {
	w               io.Writer
	hidePlannedDate bool
	theme           *Theme
}

func NewFormatter(w io.Writer, theme *Theme) *Formatter {
	if theme == nil {
		theme = DefaultTheme()
	}
	return &Formatter{w: w, theme: theme}
}

func (f *Formatter) SetHidePlannedDate(hide bool) {
	f.hidePlannedDate = hide
}

func (f *Formatter) TaskCreated(t *task.Task) {
	fmt.Fprintln(f.w, f.theme.Success.Render(fmt.Sprintf("Created task #%d: %s", t.ID, sanitizeTitle(t.Title))))
}

func (f *Formatter) TaskList(tasks []task.Task) {
	if len(tasks) == 0 {
		fmt.Fprintln(f.w, "No tasks")
		return
	}

	// Build table rows
	rows := make([][]string, 0, len(tasks))
	for _, t := range tasks {
		// Prefix: red flag for due (precedence), yellow star for planned today or earlier
		prefix := "  "
		if isDueOrOverdue(&t) {
			prefix = f.theme.Warning.Render(f.theme.Icons.Due) + " "
		} else if isPlannedForToday(&t) {
			prefix = f.theme.Accent.Render(f.theme.Icons.Planned) + " "
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
		if t.PlannedDate != nil && !f.hidePlannedDate {
			title += " " + f.theme.Muted.Render(f.theme.Icons.Date+" "+t.PlannedDate.Format("Jan 2"))
		}
		if t.DueDate != nil {
			title += " " + f.theme.Muted.Render(f.theme.Icons.Due+" "+t.DueDate.Format("Jan 2"))
		}
		if len(t.Tags) > 0 {
			title += " " + f.theme.Muted.Render(formatTagsForTable(t.Tags))
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

	// Normalize: lipgloss adds trailing \n + spaces for single-row tables
	fmt.Fprintln(f.w, strings.TrimRight(tbl.Render(), " \n"))
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
	idWidth := maxIDWidth(tasks)

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
		fmt.Fprintln(f.w, f.theme.Header.Render("No Project"))
		f.renderTaskRows(noProjectNoAreaTasks, 0, false, idWidth)
	}

	// Render groups alphabetically
	// Natural sort order puts "Area" before "Area > Project"
	headers := make([]string, 0, len(groups))
	for h := range groups {
		headers = append(headers, h)
	}
	sort.Strings(headers)

	for _, header := range headers {
		fmt.Fprintln(f.w, f.theme.Header.Render(header))
		f.renderTaskRows(groups[header], 0, false, idWidth)
	}
}

// groupedByArea displays tasks grouped by area
func (f *Formatter) groupedByArea(tasks []task.Task) {
	idWidth := maxIDWidth(tasks)

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
		fmt.Fprintln(f.w, f.theme.Header.Render("No Area"))
		f.renderTaskRows(noAreaTasks, 0, true, idWidth)
	}

	// Render areas alphabetically
	areaNames := make([]string, 0, len(areaGroups))
	for name := range areaGroups {
		areaNames = append(areaNames, name)
	}
	sort.Strings(areaNames)

	for _, aName := range areaNames {
		fmt.Fprintln(f.w, f.theme.Header.Render(aName))
		f.renderTaskRows(areaGroups[aName], 0, true, idWidth)
	}
}

// groupedByDate displays tasks grouped by date categories
func (f *Formatter) groupedByDate(tasks []task.Task) {
	idWidth := maxIDWidth(tasks)

	// Define date categories
	dateGroups := map[string][]task.Task{
		"Overdue":    {},
		"Today":      {},
		"Tomorrow":   {},
		"This Week":  {},
		"This Month": {},
		"This Year":  {},
		"Later":      {},
		"No Date":    {},
	}
	orderedCategories := []string{"Overdue", "Today", "Tomorrow", "This Week", "This Month", "This Year", "Later", "No Date"}

	now := time.Now()
	todayYear, todayMonth, todayDay := now.Date()
	today := time.Date(todayYear, todayMonth, todayDay, 0, 0, 0, 0, time.Local)
	tomorrow := today.AddDate(0, 0, 1)
	endOfWeek := today.AddDate(0, 0, 7-int(today.Weekday()))
	endOfMonth := time.Date(todayYear, todayMonth+1, 0, 0, 0, 0, 0, time.Local) // Last day of current month
	endOfYear := time.Date(todayYear, 12, 31, 0, 0, 0, 0, time.Local)

	for _, t := range tasks {
		category := getDateCategory(t.PlannedDate, t.DueDate, today, tomorrow, endOfWeek, endOfMonth, endOfYear)
		dateGroups[category] = append(dateGroups[category], t)
	}

	// Render each category
	for _, category := range orderedCategories {
		if len(dateGroups[category]) > 0 {
			fmt.Fprintln(f.w, f.theme.Header.Render(category))
			f.renderTaskRows(dateGroups[category], 0, true, idWidth)
		}
	}
}

func getDateCategory(planned, due *time.Time, today, tomorrow, endOfWeek, endOfMonth, endOfYear time.Time) string {
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
	if dateOnly.Before(endOfMonth) || dateOnly.Equal(endOfMonth) {
		return "This Month"
	}
	if dateOnly.Before(endOfYear) || dateOnly.Equal(endOfYear) {
		return "This Year"
	}
	return "Later"
}

// maxIDWidth calculates the width needed for the largest task ID
func maxIDWidth(tasks []task.Task) int {
	maxWidth := 1
	for _, t := range tasks {
		width := len(fmt.Sprintf("%d", t.ID))
		if width > maxWidth {
			maxWidth = width
		}
	}
	return maxWidth
}

// renderTaskRows renders task rows with optional indentation
func (f *Formatter) renderTaskRows(tasks []task.Task, indent int, showScope bool, idWidth int) {
	// Calculate scope width for alignment when showing scope
	scopeWidth := 0
	if showScope {
		for _, t := range tasks {
			scope := ""
			if t.ProjectName != nil {
				scope = *t.ProjectName
			} else if t.AreaName != nil {
				scope = *t.AreaName
			}
			if len(scope) > scopeWidth {
				scopeWidth = len(scope)
			}
		}
	}

	indentStr := strings.Repeat(" ", indent)
	for _, t := range tasks {
		prefix := "  "
		if isDueOrOverdue(&t) {
			prefix = f.theme.Warning.Render(f.theme.Icons.Due) + " "
		} else if isPlannedForToday(&t) {
			prefix = f.theme.Accent.Render(f.theme.Icons.Planned) + " "
		}

		title := formatTaskTitle(&t)
		if t.PlannedDate != nil && !f.hidePlannedDate {
			title += " " + f.theme.Muted.Render(f.theme.Icons.Date+" "+t.PlannedDate.Format("Jan 2"))
		}
		if t.DueDate != nil {
			title += " " + f.theme.Muted.Render(f.theme.Icons.Due+" "+t.DueDate.Format("Jan 2"))
		}
		if len(t.Tags) > 0 {
			title += " " + f.theme.Muted.Render(formatTagsForTable(t.Tags))
		}

		if showScope {
			scope := ""
			if t.ProjectName != nil {
				scope = *t.ProjectName
			} else if t.AreaName != nil {
				scope = *t.AreaName
			}
			fmt.Fprintf(f.w, "%s%s%*d  %-*s  %s\n", indentStr, prefix, idWidth, t.ID, scopeWidth, scope, title)
		} else {
			fmt.Fprintf(f.w, "%s%s%*d  %s\n", indentStr, prefix, idWidth, t.ID, title)
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

func formatTaskTitle(t *task.Task) string {
	title := sanitizeTitle(t.Title)

	// Add recurrence indicator
	title += formatRecurIndicator(t)

	return title
}

// sanitizeTitle removes newline characters from task titles to prevent display issues
func sanitizeTitle(title string) string {
	title = strings.ReplaceAll(title, "\r\n", " ")
	title = strings.ReplaceAll(title, "\n", " ")
	title = strings.ReplaceAll(title, "\r", " ")
	return strings.TrimSpace(title)
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
		fmt.Fprintln(f.w, f.theme.Success.Render(fmt.Sprintf("Completed #%d: %s", r.Completed.ID, sanitizeTitle(r.Completed.Title))))
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
		fmt.Fprintln(f.w, f.theme.Success.Render(fmt.Sprintf("Deleted #%d: %s", t.ID, sanitizeTitle(t.Title))))
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
		fmt.Fprintf(f.w, "%d  %s  %s\n", t.ID, completedAt, sanitizeTitle(t.Title))
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
		fmt.Fprintln(f.w, f.theme.Header.Render("No Project"))
		f.renderLogbookRows(noProjectNoAreaTasks)
	}

	headers := make([]string, 0, len(groups))
	for h := range groups {
		headers = append(headers, h)
	}
	sort.Strings(headers)

	for _, header := range headers {
		fmt.Fprintln(f.w, f.theme.Header.Render(header))
		f.renderLogbookRows(groups[header])
	}
}

func (f *Formatter) logbookByArea(tasks []task.Task) {
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
		fmt.Fprintln(f.w, f.theme.Header.Render("No Area"))
		f.renderLogbookRows(noAreaTasks)
	}

	areaNames := make([]string, 0, len(areaGroups))
	for name := range areaGroups {
		areaNames = append(areaNames, name)
	}
	sort.Strings(areaNames)

	for _, aName := range areaNames {
		fmt.Fprintln(f.w, f.theme.Header.Render(aName))
		f.renderLogbookRows(areaGroups[aName])
	}
}

func (f *Formatter) logbookByDate(tasks []task.Task) {
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
		fmt.Fprintln(f.w, f.theme.Header.Render(date))
		f.renderLogbookRows(dateGroups[date])
	}
}

func (f *Formatter) renderLogbookRows(tasks []task.Task) {
	for _, t := range tasks {
		completedAt := ""
		if t.CompletedAt != nil {
			completedAt = t.CompletedAt.Format("15:04")
		}
		fmt.Fprintf(f.w, "  %d  %s  %s\n", t.ID, completedAt, sanitizeTitle(t.Title))
	}
}

func (f *Formatter) AreaCreated(a *area.Area) {
	fmt.Fprintln(f.w, f.theme.Success.Render(fmt.Sprintf("Created area: %s", a.Name)))
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
	fmt.Fprintln(f.w, f.theme.Success.Render(fmt.Sprintf("Deleted area: %s", a.Name)))
}

func (f *Formatter) ProjectCreated(p *project.Project) {
	fmt.Fprintln(f.w, f.theme.Success.Render(fmt.Sprintf("Created project: %s", p.Name)))
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

func (f *Formatter) ProjectListGrouped(projects []project.ProjectWithArea, groupBy string) {
	if groupBy != "area" || groupBy == "" {
		// Fall back to simple list
		if len(projects) == 0 {
			fmt.Fprintln(f.w, "No projects")
			return
		}
		for _, p := range projects {
			fmt.Fprintln(f.w, p.Name)
		}
		return
	}

	if len(projects) == 0 {
		fmt.Fprintln(f.w, "No projects")
		return
	}

	// Group projects by area
	noAreaProjects := make([]project.ProjectWithArea, 0)
	areaGroups := make(map[string][]project.ProjectWithArea)

	for _, p := range projects {
		if p.AreaName == nil {
			noAreaProjects = append(noAreaProjects, p)
		} else {
			areaGroups[*p.AreaName] = append(areaGroups[*p.AreaName], p)
		}
	}

	// Render: No Area first
	if len(noAreaProjects) > 0 {
		fmt.Fprintln(f.w, f.theme.Header.Render("No Area"))
		for _, p := range noAreaProjects {
			fmt.Fprintf(f.w, "  %s\n", p.Name)
		}
	}

	// Render areas alphabetically
	areaNames := make([]string, 0, len(areaGroups))
	for name := range areaGroups {
		areaNames = append(areaNames, name)
	}
	sort.Strings(areaNames)

	for _, aName := range areaNames {
		fmt.Fprintln(f.w, f.theme.Header.Render(aName))
		for _, p := range areaGroups[aName] {
			fmt.Fprintf(f.w, "  %s\n", p.Name)
		}
	}
}

func (f *Formatter) ProjectDeleted(p *project.Project) {
	fmt.Fprintln(f.w, f.theme.Success.Render(fmt.Sprintf("Deleted project: %s", p.Name)))
}

func (f *Formatter) AreaRenamed(oldName, newName string) {
	fmt.Fprintln(f.w, f.theme.Success.Render(fmt.Sprintf("Renamed area: %s -> %s", oldName, newName)))
}

func (f *Formatter) ProjectRenamed(oldName, newName string) {
	fmt.Fprintln(f.w, f.theme.Success.Render(fmt.Sprintf("Renamed project: %s -> %s", oldName, newName)))
}

func (f *Formatter) ProjectMoved(p *project.Project, areaName string) {
	fmt.Fprintln(f.w, f.theme.Success.Render(fmt.Sprintf("Moved project '%s' to area: %s", p.Name, areaName)))
}

func (f *Formatter) ProjectAreaCleared(p *project.Project) {
	fmt.Fprintln(f.w, f.theme.Success.Render(fmt.Sprintf("Cleared area from project: %s", p.Name)))
}

func (f *Formatter) TaskPlannedDateSet(t *task.Task) {
	if t.PlannedDate != nil {
		fmt.Fprintln(f.w, f.theme.Success.Render(fmt.Sprintf("Planned #%d for %s: %s", t.ID, t.PlannedDate.Format("Jan 2"), sanitizeTitle(t.Title))))
	} else {
		fmt.Fprintln(f.w, f.theme.Success.Render(fmt.Sprintf("Cleared planned date for #%d: %s", t.ID, sanitizeTitle(t.Title))))
	}
}

func (f *Formatter) TaskDueDateSet(t *task.Task) {
	if t.DueDate != nil {
		fmt.Fprintln(f.w, f.theme.Success.Render(fmt.Sprintf("Due #%d on %s: %s", t.ID, t.DueDate.Format("Jan 2"), sanitizeTitle(t.Title))))
	} else {
		fmt.Fprintln(f.w, f.theme.Success.Render(fmt.Sprintf("Cleared due date for #%d: %s", t.ID, sanitizeTitle(t.Title))))
	}
}

func (f *Formatter) TaskRecurrenceSet(t *task.Task) {
	if t.RecurRule != nil {
		rule, err := recurparse.FromJSON(*t.RecurRule)
		if err != nil {
			fmt.Fprintln(f.w, f.theme.Success.Render(fmt.Sprintf("Set recurrence for #%d: %s", t.ID, sanitizeTitle(t.Title))))
			return
		}
		fmt.Fprintln(f.w, f.theme.Success.Render(fmt.Sprintf("Set recurrence for #%d (%s): %s", t.ID, rule.Format(), sanitizeTitle(t.Title))))
	} else {
		fmt.Fprintln(f.w, f.theme.Success.Render(fmt.Sprintf("Cleared recurrence for #%d: %s", t.ID, sanitizeTitle(t.Title))))
	}
}

func (f *Formatter) TaskRecurrencePaused(t *task.Task) {
	fmt.Fprintln(f.w, f.theme.Success.Render(fmt.Sprintf("Paused recurrence for #%d: %s", t.ID, sanitizeTitle(t.Title))))
}

func (f *Formatter) TaskRecurrenceResumed(t *task.Task) {
	fmt.Fprintln(f.w, f.theme.Success.Render(fmt.Sprintf("Resumed recurrence for #%d: %s", t.ID, sanitizeTitle(t.Title))))
}

func (f *Formatter) TaskRecurrenceEndSet(t *task.Task) {
	if t.RecurEnd != nil {
		fmt.Fprintln(f.w, f.theme.Success.Render(fmt.Sprintf("Set recurrence end date for #%d to %s: %s", t.ID, t.RecurEnd.Format("Jan 2"), sanitizeTitle(t.Title))))
	} else {
		fmt.Fprintln(f.w, f.theme.Success.Render(fmt.Sprintf("Cleared recurrence end date for #%d: %s", t.ID, sanitizeTitle(t.Title))))
	}
}

func (f *Formatter) TaskRecurrenceInfo(t *task.Task) {
	if t.RecurRule == nil {
		fmt.Fprintf(f.w, "#%d: %s (no recurrence)\n", t.ID, sanitizeTitle(t.Title))
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

	fmt.Fprintf(f.w, "#%d: %s\n  Recurs: %s%s%s\n", t.ID, sanitizeTitle(t.Title), ruleStr, endStr, status)
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
	fmt.Fprintln(f.w, f.theme.Success.Render(fmt.Sprintf("Added tag '%s' to #%d: %s", tagName, t.ID, sanitizeTitle(t.Title))))
}

func (f *Formatter) TaskTagRemoved(t *task.Task, tagName string) {
	fmt.Fprintln(f.w, f.theme.Success.Render(fmt.Sprintf("Removed tag '%s' from #%d: %s", tagName, t.ID, sanitizeTitle(t.Title))))
}

func (f *Formatter) TaskEdited(id int64, changes []string) {
	fmt.Fprintln(f.w, f.theme.Success.Render(fmt.Sprintf("Updated #%d: %s", id, joinChanges(changes))))
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
	fmt.Fprintf(f.w, "#%d: %s\n", t.ID, sanitizeTitle(t.Title))

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

func (f *Formatter) Error(msg string) {
	fmt.Fprintln(f.w, f.theme.Error.Render(fmt.Sprintf("Error: %s", msg)))
}
