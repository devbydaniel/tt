package output

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/devbydaniel/tt/internal/domain/area"
	"github.com/devbydaniel/tt/internal/domain/task"
	"github.com/devbydaniel/tt/internal/recurparse"
)

type Formatter struct {
	w               io.Writer
	hidePlannedDate bool
	hideScope       bool
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

func (f *Formatter) SetHideScope(hide bool) {
	f.hideScope = hide
}

func (f *Formatter) TaskCreated(t *task.Task) {
	fmt.Fprintln(f.w, f.theme.Success.Render(fmt.Sprintf("Created task #%d: %s", t.ID, sanitizeTitle(t.Title))))
}

func (f *Formatter) TaskList(tasks []task.Task) {
	if len(tasks) == 0 {
		fmt.Fprintln(f.w, "No tasks")
		return
	}

	idWidth := maxIDWidth(tasks)
	f.renderTaskRows(tasks, 0, !f.hideScope, idWidth)
}

// GroupedTaskList displays tasks grouped by the specified field.
// groupBy can be: "scope", "date", or "none" (falls back to TaskList)
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
	case "scope":
		f.groupedByScope(tasks)
	case "date":
		f.groupedByDate(tasks)
	default:
		f.TaskList(tasks)
	}
}

// groupedByScope displays tasks grouped by scope ("Area > Project", "Area", or "Project")
// Tasks with area but no project appear under just the area name,
// sorted before "Area > Project" groups (alphabetically, area-only headers come first)
// Projects appear as standalone header-style lines with metadata but no ID.
func (f *Formatter) groupedByScope(tasks []task.Task) {
	idWidth := maxIDWidth(tasks)

	// Separate projects from regular tasks
	var projects []task.Task
	var regularTasks []task.Task
	for _, t := range tasks {
		if t.IsProject() {
			projects = append(projects, t)
		} else {
			regularTasks = append(regularTasks, t)
		}
	}

	// Group regular tasks:
	// - No area, no project -> "No Scope"
	// - Area but no project -> area name (e.g., "Work")
	// - Project -> "Area > Project" or just "Project"
	noScopeTasks := make([]task.Task, 0)
	groups := make(map[string][]task.Task)

	for _, t := range regularTasks {
		if t.ParentName == nil {
			if t.AreaName == nil {
				noScopeTasks = append(noScopeTasks, t)
			} else {
				// Area but no project - use area name as header
				groups[*t.AreaName] = append(groups[*t.AreaName], t)
			}
			continue
		}

		// Build header: "Area > Project" or just "Project"
		header := *t.ParentName
		if t.AreaName != nil {
			header = *t.AreaName + " > " + *t.ParentName
		}
		groups[header] = append(groups[header], t)
	}

	// Build project scope map for sorting
	projectsByScope := make(map[string]*task.Task)
	for i := range projects {
		p := &projects[i]
		scope := sanitizeTitle(p.Title)
		if p.AreaName != nil {
			scope = *p.AreaName + " > " + sanitizeTitle(p.Title)
		}
		projectsByScope[scope] = p
	}

	// Render: No Scope first
	if len(noScopeTasks) > 0 {
		fmt.Fprintln(f.w, f.theme.Header.Render("No Scope"))
		f.renderTaskRows(noScopeTasks, 0, !f.hideScope, idWidth)
	}

	// Combine all headers (group headers + project scopes) and sort
	allHeaders := make([]string, 0, len(groups)+len(projectsByScope))
	for h := range groups {
		allHeaders = append(allHeaders, h)
	}
	for h := range projectsByScope {
		allHeaders = append(allHeaders, h)
	}
	sort.Strings(allHeaders)

	// Render sorted headers (both groups and projects)
	for _, header := range allHeaders {
		if proj, isProject := projectsByScope[header]; isProject {
			// Render project as header-style line (no ID, with metadata)
			f.renderProjectHeaderLine(proj)
		} else if tasks, isGroup := groups[header]; isGroup {
			fmt.Fprintln(f.w, f.theme.Header.Render(header))
			f.renderTaskRows(tasks, 0, !f.hideScope, idWidth)
		}
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
		// Planned dates in past show as "Today", only due dates are "Overdue"
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
	indentStr := strings.Repeat(" ", indent)
	for _, t := range tasks {
		prefix := "  "
		if isDueOrOverdue(&t) {
			prefix = f.theme.Warning.Render(f.theme.Icons.Due) + " "
		} else if isPlannedForToday(&t) {
			prefix = f.theme.Accent.Render(f.theme.Icons.Planned) + " "
		}

		// ID styled with padding
		idStr := fmt.Sprintf("%*d", idWidth, t.ID)
		id := f.theme.ID.Render(idStr)

		// Build display differently for projects vs tasks
		var display string
		if t.IsProject() {
			// For projects: when hiding scope, show only project name (no area)
			if showScope {
				display = f.theme.Scope.Render(formatProjectScope(t.AreaName, t.Title))
			} else {
				display = f.theme.Scope.Render(sanitizeTitle(t.Title))
			}
		} else {
			// For regular tasks: optional scope prefix + title
			if showScope {
				scope := formatScope(t.AreaName, t.ParentName)
				if scope != "" {
					display = f.theme.Scope.Render(scope) + "  "
				}
			}
			display += formatTaskTitle(&t)
			if recur := formatRecurIndicator(&t); recur != "" {
				display += " " + f.theme.Muted.Render(recur)
			}
		}

		// Add dates and tags (common to both projects and tasks)
		if t.PlannedDate != nil && !f.hidePlannedDate {
			display += " " + f.theme.Muted.Render(f.theme.Icons.Date+" "+t.PlannedDate.Format("Jan 2"))
		}
		if t.DueDate != nil {
			display += " " + f.theme.Muted.Render(f.theme.Icons.Due+" "+t.DueDate.Format("Jan 2"))
		}
		if len(t.Tags) > 0 {
			display += " " + f.theme.Muted.Render(formatTagsForTable(t.Tags))
		}

		fmt.Fprintf(f.w, "%s%s%s  %s\n", indentStr, prefix, id, display)
	}
}

// formatScope returns the scope display string for a task.
// Shows "area > project" when both exist, just "area" or "project" when only one exists.
func formatScope(areaName, projectName *string) string {
	if projectName != nil {
		if areaName != nil {
			return *areaName + " > " + *projectName
		}
		return *projectName
	}
	if areaName != nil {
		return *areaName
	}
	return ""
}

// formatProjectScope returns the scope display for a project row.
// Shows "area > projectTitle" when area exists, just "projectTitle" otherwise.
func formatProjectScope(areaName *string, projectTitle string) string {
	if areaName != nil {
		return *areaName + " > " + sanitizeTitle(projectTitle)
	}
	return sanitizeTitle(projectTitle)
}

// renderProjectHeaderLine renders a project as a standalone header-style line
// for scope-grouped views. Format: [Area > ProjectName]  [planned] [due] [tags]
func (f *Formatter) renderProjectHeaderLine(p *task.Task) {
	// Show full scope: "Area > ProjectName" or just "ProjectName"
	scope := sanitizeTitle(p.Title)
	if p.AreaName != nil {
		scope = *p.AreaName + " > " + scope
	}
	parts := []string{f.theme.Header.Render(scope)}

	if p.PlannedDate != nil && !f.hidePlannedDate {
		parts = append(parts, f.theme.Muted.Render(f.theme.Icons.Date+" "+p.PlannedDate.Format("Jan 2")))
	}
	if p.DueDate != nil {
		parts = append(parts, f.theme.Muted.Render(f.theme.Icons.Due+" "+p.DueDate.Format("Jan 2")))
	}
	if len(p.Tags) > 0 {
		parts = append(parts, f.theme.Muted.Render(formatTagsForTable(p.Tags)))
	}

	fmt.Fprintln(f.w, strings.Join(parts, "  "))
}

func formatRecurIndicator(t *task.Task) string {
	if t.RecurType == nil {
		return ""
	}

	pattern := ""
	if t.RecurRule != nil {
		if rule, err := recurparse.FromJSON(*t.RecurRule); err == nil {
			pattern = rule.Format()
		}
	}

	// Use ⏸ for paused, ↻ for active
	symbol := "↻"
	if t.RecurPaused {
		symbol = "⏸"
	}

	if pattern != "" {
		return fmt.Sprintf("%s %s", symbol, pattern)
	}
	return symbol
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
	return sanitizeTitle(t.Title)
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

func (f *Formatter) TasksUncompleted(tasks []task.Task) {
	for _, t := range tasks {
		fmt.Fprintln(f.w, f.theme.Success.Render(fmt.Sprintf("Uncompleted #%d: %s", t.ID, sanitizeTitle(t.Title))))
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
	case "scope":
		f.logbookByScope(tasks)
	case "date":
		f.logbookByDate(tasks)
	default:
		f.Logbook(tasks)
	}
}

func (f *Formatter) logbookByScope(tasks []task.Task) {
	noScopeTasks := make([]task.Task, 0)
	groups := make(map[string][]task.Task)

	for _, t := range tasks {
		if t.ParentName == nil {
			if t.AreaName == nil {
				noScopeTasks = append(noScopeTasks, t)
			} else {
				groups[*t.AreaName] = append(groups[*t.AreaName], t)
			}
			continue
		}

		header := *t.ParentName
		if t.AreaName != nil {
			header = *t.AreaName + " > " + *t.ParentName
		}
		groups[header] = append(groups[header], t)
	}

	if len(noScopeTasks) > 0 {
		fmt.Fprintln(f.w, f.theme.Header.Render("No Scope"))
		f.renderLogbookRows(noScopeTasks)
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

func (f *Formatter) ProjectCreated(p *task.Task) {
	fmt.Fprintln(f.w, f.theme.Success.Render(fmt.Sprintf("Created project: %s", p.Title)))
}

func (f *Formatter) ProjectList(projects []task.Task) {
	if len(projects) == 0 {
		fmt.Fprintln(f.w, "No projects")
		return
	}

	for _, p := range projects {
		fmt.Fprintln(f.w, p.Title)
	}
}

func (f *Formatter) ProjectListGrouped(projects []task.Task, groupBy string) {
	if groupBy != "area" || groupBy == "" {
		// Fall back to simple list
		if len(projects) == 0 {
			fmt.Fprintln(f.w, "No projects")
			return
		}
		for _, p := range projects {
			fmt.Fprintln(f.w, p.Title)
		}
		return
	}

	if len(projects) == 0 {
		fmt.Fprintln(f.w, "No projects")
		return
	}

	// Group projects by area
	noAreaProjects := make([]task.Task, 0)
	areaGroups := make(map[string][]task.Task)

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
			fmt.Fprintf(f.w, "  %s\n", p.Title)
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
			fmt.Fprintf(f.w, "  %s\n", p.Title)
		}
	}
}

func (f *Formatter) ProjectDeleted(p *task.Task) {
	fmt.Fprintln(f.w, f.theme.Success.Render(fmt.Sprintf("Deleted project: %s", p.Title)))
}

func (f *Formatter) AreaRenamed(oldName, newName string) {
	fmt.Fprintln(f.w, f.theme.Success.Render(fmt.Sprintf("Renamed area: %s -> %s", oldName, newName)))
}

func (f *Formatter) ProjectRenamed(oldName, newName string) {
	fmt.Fprintln(f.w, f.theme.Success.Render(fmt.Sprintf("Renamed project: %s -> %s", oldName, newName)))
}

func (f *Formatter) ProjectMoved(p *task.Task, areaName string) {
	fmt.Fprintln(f.w, f.theme.Success.Render(fmt.Sprintf("Moved project '%s' to area: %s", p.Title, areaName)))
}

func (f *Formatter) ProjectAreaCleared(p *task.Task) {
	fmt.Fprintln(f.w, f.theme.Success.Render(fmt.Sprintf("Cleared area from project: %s", p.Title)))
}

func (f *Formatter) ProjectsCompleted(results []task.CompleteResult) {
	for _, r := range results {
		fmt.Fprintln(f.w, f.theme.Success.Render(fmt.Sprintf("Completed project: %s", sanitizeTitle(r.Completed.Title))))
	}
}

func (f *Formatter) ProjectsUncompleted(projects []task.Task) {
	for _, p := range projects {
		fmt.Fprintln(f.w, f.theme.Success.Render(fmt.Sprintf("Uncompleted project: %s", sanitizeTitle(p.Title))))
	}
}

func (f *Formatter) ProjectEdited(name string, changes []string) {
	fmt.Fprintln(f.w, f.theme.Success.Render(fmt.Sprintf("Updated project '%s': %s", name, joinChanges(changes))))
}

func (f *Formatter) ProjectDetails(p *task.Task) {
	fmt.Fprintf(f.w, "Project: %s\n", sanitizeTitle(p.Title))

	if p.Description != nil && *p.Description != "" {
		fmt.Fprintf(f.w, "  Description: %s\n", *p.Description)
	}
	if p.AreaName != nil {
		fmt.Fprintf(f.w, "  Area: %s\n", *p.AreaName)
	}
	if p.PlannedDate != nil {
		fmt.Fprintf(f.w, "  Planned: %s\n", p.PlannedDate.Format("Jan 2, 2006"))
	}
	if p.DueDate != nil {
		fmt.Fprintf(f.w, "  Due: %s\n", p.DueDate.Format("Jan 2, 2006"))
	}
	if p.State == task.StateSomeday {
		fmt.Fprintln(f.w, "  State: someday")
	}
	if len(p.Tags) > 0 {
		fmt.Fprintf(f.w, "  Tags: %s\n", formatTagList(p.Tags))
	}
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
