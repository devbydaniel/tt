package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/devbydaniel/tt/internal/domain/task"
	"github.com/devbydaniel/tt/internal/recurparse"
)

// Content displays the task list in the right panel
type Content struct {
	title          string
	tasks          []task.Task
	groupBy        string          // grouping mode: none, project, area, date
	hideScope      bool            // whether to hide the project/area column
	scheduleGroups *ScheduleGroups // pre-grouped schedule data (mutually exclusive with tasks+groupBy)
	width          int
	height         int
	viewport       viewport.Model
	ready          bool
	styles         *Styles
	card           *Card
}

// NewContent creates a new content panel
func NewContent(styles *Styles) Content {
	return Content{
		title:  "Today",
		styles: styles,
		card:   NewCard(styles),
	}
}

// SetSize updates content dimensions
func (c Content) SetSize(width, height int) Content {
	c.width = width
	c.height = height

	// Content dimensions: width - border(2) - horizontal padding(2), height - border(2) - header with blank line(2)
	contentWidth := width - 4
	contentHeight := height - 4

	if contentWidth < 1 {
		contentWidth = 1
	}
	if contentHeight < 1 {
		contentHeight = 1
	}

	if !c.ready {
		c.viewport = viewport.New(contentWidth, contentHeight)
		c.viewport.SetContent(c.buildTaskList())
		c.ready = true
	} else {
		c.viewport.Width = contentWidth
		c.viewport.Height = contentHeight
	}

	return c
}

// SetTasks updates the displayed tasks with optional grouping
func (c Content) SetTasks(tasks []task.Task, title string, groupBy string, hideScope bool) Content {
	c.tasks = tasks
	c.title = title
	c.groupBy = groupBy
	c.hideScope = hideScope
	c.scheduleGroups = nil // Clear schedule groups (mutually exclusive)
	if c.ready {
		c.viewport.SetContent(c.buildTaskList())
		c.viewport.GotoTop()
	}
	return c
}

// SetScheduleGroups updates the content with pre-grouped schedule data
func (c Content) SetScheduleGroups(groups ScheduleGroups, title string, hideScope bool) Content {
	c.scheduleGroups = &groups
	c.tasks = nil // Clear tasks (mutually exclusive)
	c.groupBy = ""
	c.hideScope = hideScope
	c.title = title
	if c.ready {
		c.viewport.SetContent(c.buildTaskList())
		c.viewport.GotoTop()
	}
	return c
}

// buildTaskList renders all tasks as a string
func (c Content) buildTaskList() string {
	// Schedule grouping (pre-grouped data)
	if c.scheduleGroups != nil {
		return c.buildGroupedBySchedule()
	}

	// Check for empty tasks
	if len(c.tasks) == 0 {
		return c.styles.Theme.Muted.Render("No tasks")
	}

	// Client-side grouping
	switch c.groupBy {
	case "project":
		return c.buildGroupedByProject()
	case "area":
		return c.buildGroupedByArea()
	case "date":
		return c.buildGroupedByDate()
	default:
		return c.buildFlatTaskList()
	}
}

// buildFlatTaskList renders tasks without grouping
func (c Content) buildFlatTaskList() string {
	var rows []string
	for _, t := range c.tasks {
		row := c.renderTaskRow(&t)
		rows = append(rows, row)
	}
	return strings.Join(rows, "\n")
}

// buildGroupedByProject groups tasks by "Area > Project" hierarchy
func (c Content) buildGroupedByProject() string {
	// Group tasks: No project/no area -> "No Project", area only -> area name, project -> "Area > Project"
	noProjectNoArea := make([]task.Task, 0)
	groups := make(map[string][]task.Task)

	for _, t := range c.tasks {
		if t.ProjectName == nil {
			if t.AreaName == nil {
				noProjectNoArea = append(noProjectNoArea, t)
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

	var sections []string

	// Render "No Project" first
	if len(noProjectNoArea) > 0 {
		sections = append(sections, c.renderGroupSection("No Project", noProjectNoArea))
	}

	// Render groups alphabetically
	headers := make([]string, 0, len(groups))
	for h := range groups {
		headers = append(headers, h)
	}
	sort.Strings(headers)

	for _, header := range headers {
		sections = append(sections, c.renderGroupSection(header, groups[header]))
	}

	if len(sections) == 0 {
		return c.styles.Theme.Muted.Render("No tasks")
	}
	return strings.Join(sections, "\n\n")
}

// buildGroupedByArea groups tasks by area
func (c Content) buildGroupedByArea() string {
	noArea := make([]task.Task, 0)
	areaGroups := make(map[string][]task.Task)

	for _, t := range c.tasks {
		if t.AreaName == nil {
			noArea = append(noArea, t)
		} else {
			areaGroups[*t.AreaName] = append(areaGroups[*t.AreaName], t)
		}
	}

	var sections []string

	// Render "No Area" first
	if len(noArea) > 0 {
		sections = append(sections, c.renderGroupSection("No Area", noArea))
	}

	// Render areas alphabetically
	areaNames := make([]string, 0, len(areaGroups))
	for name := range areaGroups {
		areaNames = append(areaNames, name)
	}
	sort.Strings(areaNames)

	for _, aName := range areaNames {
		sections = append(sections, c.renderGroupSection(aName, areaGroups[aName]))
	}

	if len(sections) == 0 {
		return c.styles.Theme.Muted.Render("No tasks")
	}
	return strings.Join(sections, "\n\n")
}

// buildGroupedByDate groups tasks by date categories
func (c Content) buildGroupedByDate() string {
	now := time.Now()
	todayYear, todayMonth, todayDay := now.Date()
	today := time.Date(todayYear, todayMonth, todayDay, 0, 0, 0, 0, time.Local)
	tomorrow := today.AddDate(0, 0, 1)
	endOfWeek := today.AddDate(0, 0, 7-int(today.Weekday()))
	endOfMonth := time.Date(todayYear, todayMonth+1, 0, 0, 0, 0, 0, time.Local)
	endOfYear := time.Date(todayYear, 12, 31, 0, 0, 0, 0, time.Local)

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

	for _, t := range c.tasks {
		category := c.getDateCategory(t.PlannedDate, t.DueDate, today, tomorrow, endOfWeek, endOfMonth, endOfYear)
		dateGroups[category] = append(dateGroups[category], t)
	}

	orderedCategories := []string{"Overdue", "Today", "Tomorrow", "This Week", "This Month", "This Year", "Later", "No Date"}
	var sections []string

	for _, category := range orderedCategories {
		if len(dateGroups[category]) > 0 {
			sections = append(sections, c.renderGroupSection(category, dateGroups[category]))
		}
	}

	if len(sections) == 0 {
		return c.styles.Theme.Muted.Render("No tasks")
	}
	return strings.Join(sections, "\n\n")
}

// buildGroupedBySchedule renders pre-grouped schedule data
func (c Content) buildGroupedBySchedule() string {
	if c.scheduleGroups == nil {
		return c.styles.Theme.Muted.Render("No tasks")
	}

	schedules := []struct {
		name  string
		tasks []task.Task
	}{
		{"Today", c.scheduleGroups.Today},
		{"Upcoming", c.scheduleGroups.Upcoming},
		{"Anytime", c.scheduleGroups.Anytime},
		{"Someday", c.scheduleGroups.Someday},
	}

	var sections []string
	for _, sched := range schedules {
		if len(sched.tasks) == 0 {
			continue
		}
		sections = append(sections, c.renderGroupSection(sched.name, sched.tasks))
	}

	if len(sections) == 0 {
		return c.styles.Theme.Muted.Render("No tasks")
	}
	return strings.Join(sections, "\n\n")
}

// renderGroupSection renders a group header and its tasks
func (c Content) renderGroupSection(header string, tasks []task.Task) string {
	headerLine := c.styles.Theme.Header.Render(header)
	var rows []string
	for _, t := range tasks {
		rows = append(rows, c.renderTaskRow(&t))
	}
	return headerLine + "\n" + strings.Join(rows, "\n")
}

// getDateCategory determines which date category a task belongs to
func (c Content) getDateCategory(planned, due *time.Time, today, tomorrow, endOfWeek, endOfMonth, endOfYear time.Time) string {
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

// View renders the content panel
func (c Content) View() string {
	var content string
	if c.ready {
		content = c.viewport.View()
	} else {
		content = c.buildTaskList()
	}

	return c.card.Render(c.title, content, c.width, c.height, false)
}

// ScrollUp scrolls the content up
func (c Content) ScrollUp() Content {
	if c.ready {
		c.viewport.LineUp(1)
	}
	return c
}

// ScrollDown scrolls the content down
func (c Content) ScrollDown() Content {
	if c.ready {
		c.viewport.LineDown(1)
	}
	return c
}

// ScrollHalfPageUp scrolls up half a page
func (c Content) ScrollHalfPageUp() Content {
	if c.ready {
		c.viewport.HalfViewUp()
	}
	return c
}

// ScrollHalfPageDown scrolls down half a page
func (c Content) ScrollHalfPageDown() Content {
	if c.ready {
		c.viewport.HalfViewDown()
	}
	return c
}

// AtTop returns true if viewport is at the top
func (c Content) AtTop() bool {
	return !c.ready || c.viewport.AtTop()
}

// AtBottom returns true if viewport is at the bottom
func (c Content) AtBottom() bool {
	return !c.ready || c.viewport.AtBottom()
}

// ScrollPercent returns the scroll position as a percentage
func (c Content) ScrollPercent() float64 {
	if !c.ready {
		return 0
	}
	return c.viewport.ScrollPercent()
}

// ViewportHeight returns the viewport height for external use
func (c Content) ViewportHeight() int {
	if !c.ready {
		return 0
	}
	return c.viewport.Height
}

// TotalLines returns total content lines
func (c Content) TotalLines() int {
	return len(c.tasks)
}

// renderTaskRow formats a single task row
func (c Content) renderTaskRow(t *task.Task) string {
	theme := c.styles.Theme

	// Prefix: flag for due, star for planned today
	prefix := "  "
	if c.isDueOrOverdue(t) {
		prefix = theme.Warning.Render(theme.Icons.Due) + " "
	} else if c.isPlannedForToday(t) {
		prefix = theme.Accent.Render(theme.Icons.Planned) + " "
	}

	// ID
	id := theme.ID.Render(fmt.Sprintf("%d", t.ID))

	// Scope: "area > project" or just one
	scope := c.formatScope(t.AreaName, t.ProjectName)
	if scope != "" {
		scope = theme.Scope.Render(scope)
	}

	// Title
	title := c.sanitizeTitle(t.Title)

	// Extras: recurrence, dates, tags
	var extras []string

	if recur := c.formatRecurIndicator(t); recur != "" {
		extras = append(extras, theme.Muted.Render(recur))
	}

	if t.PlannedDate != nil {
		extras = append(extras, theme.Muted.Render(theme.Icons.Date+" "+t.PlannedDate.Format("Jan 2")))
	}

	if t.DueDate != nil {
		extras = append(extras, theme.Muted.Render(theme.Icons.Due+" "+t.DueDate.Format("Jan 2")))
	}

	if len(t.Tags) > 0 {
		extras = append(extras, theme.Muted.Render(c.formatTags(t.Tags)))
	}

	// Build row
	parts := []string{prefix + id}
	if scope != "" && !c.hideScope {
		parts = append(parts, scope)
	}
	parts = append(parts, title)
	if len(extras) > 0 {
		parts = append(parts, strings.Join(extras, " "))
	}

	return strings.Join(parts, "  ")
}

func (c Content) formatScope(areaName, projectName *string) string {
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

func (c Content) formatRecurIndicator(t *task.Task) string {
	if t.RecurType == nil {
		return ""
	}

	pattern := ""
	if t.RecurRule != nil {
		if rule, err := recurparse.FromJSON(*t.RecurRule); err == nil {
			pattern = rule.Format()
		}
	}

	symbol := "↻"
	if t.RecurPaused {
		symbol = "⏸"
	}

	if pattern != "" {
		return fmt.Sprintf("%s %s", symbol, pattern)
	}
	return symbol
}

func (c Content) formatTags(tags []string) string {
	var parts []string
	for _, tag := range tags {
		parts = append(parts, "#"+tag)
	}
	return strings.Join(parts, " ")
}

func (c Content) sanitizeTitle(title string) string {
	title = strings.ReplaceAll(title, "\r\n", " ")
	title = strings.ReplaceAll(title, "\n", " ")
	title = strings.ReplaceAll(title, "\r", " ")
	return strings.TrimSpace(title)
}

func (c Content) isPlannedForToday(t *task.Task) bool {
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

func (c Content) isDueOrOverdue(t *task.Task) bool {
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
