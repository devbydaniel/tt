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
	displayTasks   []task.Task      // tasks in display order (computed once when set)
	taskSchedules  map[int64]string // task ID -> schedule name (for schedule grouping)
	groupBy        string           // grouping mode: none, scope, date, schedule
	hideScope      bool             // whether to hide the project/area column
	width          int
	height         int
	viewport       viewport.Model
	ready          bool
	styles         *Styles
	card           *Card
	focused        bool // whether content panel has focus
	showSelection  bool // whether to show selection indicator (even when not focused)
	selectedIndex  int  // index into displayTasks (-1 = none)
}

// NewContent creates a new content panel
func NewContent(styles *Styles) Content {
	return Content{
		title:         "Today",
		styles:        styles,
		card:          NewCard(styles),
		selectedIndex: -1,
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
	c.title = title
	c.groupBy = groupBy
	c.hideScope = hideScope
	c.displayTasks = c.computeDisplayOrder(tasks, groupBy)
	// Reset selection when tasks change
	if c.focused && len(c.displayTasks) > 0 {
		c.selectedIndex = 0
	} else {
		c.selectedIndex = -1
	}
	if c.ready {
		c.viewport.SetContent(c.buildTaskList())
		c.viewport.GotoTop()
	}
	return c
}

// SetScheduleGroups updates the content with pre-grouped schedule data
func (c Content) SetScheduleGroups(groups ScheduleGroups, title string, hideScope bool) Content {
	c.groupBy = "schedule"
	c.hideScope = hideScope
	c.title = title
	// Build display order and schedule map
	c.taskSchedules = make(map[int64]string)
	var all []task.Task
	for _, t := range groups.Today {
		c.taskSchedules[t.ID] = "Today"
		all = append(all, t)
	}
	for _, t := range groups.Upcoming {
		c.taskSchedules[t.ID] = "Upcoming"
		all = append(all, t)
	}
	for _, t := range groups.Anytime {
		c.taskSchedules[t.ID] = "Anytime"
		all = append(all, t)
	}
	for _, t := range groups.Someday {
		c.taskSchedules[t.ID] = "Someday"
		all = append(all, t)
	}
	c.displayTasks = all
	// Reset selection when tasks change
	if c.focused && len(c.displayTasks) > 0 {
		c.selectedIndex = 0
	} else {
		c.selectedIndex = -1
	}
	if c.ready {
		c.viewport.SetContent(c.buildTaskList())
		c.viewport.GotoTop()
	}
	return c
}

// buildTaskList renders all tasks as a string
func (c Content) buildTaskList() string {
	if len(c.displayTasks) == 0 {
		return c.styles.Theme.Muted.Render("No tasks")
	}

	switch c.groupBy {
	case "scope":
		return c.buildGroupedByScope()
	case "date":
		return c.buildGroupedByDate()
	case "schedule":
		return c.buildGroupedBySchedule()
	default:
		return c.buildFlatTaskList()
	}
}

// buildFlatTaskList renders tasks without grouping
func (c Content) buildFlatTaskList() string {
	var rows []string
	for i := range c.displayTasks {
		rows = append(rows, c.renderTaskRow(&c.displayTasks[i], i))
	}
	return strings.Join(rows, "\n")
}

// buildGroupedByScope groups tasks by scope ("Area > Project", "Area", or "Project")
// Projects appear as standalone header-style lines with metadata but no ID.
func (c Content) buildGroupedByScope() string {
	// Separate projects from regular tasks
	var projects []*task.Task
	var regularTasks []*task.Task
	projectIndices := make(map[*task.Task]int)
	regularIndices := make(map[*task.Task]int)

	for i := range c.displayTasks {
		t := &c.displayTasks[i]
		if t.IsProject() {
			projects = append(projects, t)
			projectIndices[t] = i
		} else {
			regularTasks = append(regularTasks, t)
			regularIndices[t] = i
		}
	}

	// Group regular tasks by scope
	noScopeTasks := make([]*task.Task, 0)
	groups := make(map[string][]*task.Task)

	for _, t := range regularTasks {
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

	// Build project scope map for sorting
	projectsByScope := make(map[string]*task.Task)
	for _, p := range projects {
		scope := c.sanitizeTitle(p.Title)
		if p.AreaName != nil {
			scope = *p.AreaName + " > " + c.sanitizeTitle(p.Title)
		}
		projectsByScope[scope] = p
	}

	var sections []string

	// Render: No Scope first
	if len(noScopeTasks) > 0 {
		header := c.styles.Theme.Header.Render("No Scope")
		var rows []string
		for _, t := range noScopeTasks {
			rows = append(rows, c.renderTaskRow(t, regularIndices[t]))
		}
		sections = append(sections, header+"\n"+strings.Join(rows, "\n"))
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
			sections = append(sections, c.renderProjectHeaderLine(proj, projectIndices[proj]))
		} else if tasks, isGroup := groups[header]; isGroup {
			headerLine := c.styles.Theme.Header.Render(header)
			var rows []string
			for _, t := range tasks {
				rows = append(rows, c.renderTaskRow(t, regularIndices[t]))
			}
			sections = append(sections, headerLine+"\n"+strings.Join(rows, "\n"))
		}
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

	return c.buildGroupedList(func(t *task.Task) string {
		return c.getDateCategory(t.PlannedDate, t.DueDate, today, tomorrow, endOfWeek, endOfMonth, endOfYear)
	})
}

// buildGroupedBySchedule renders pre-grouped schedule data
func (c Content) buildGroupedBySchedule() string {
	return c.buildGroupedList(func(t *task.Task) string {
		if sched, ok := c.taskSchedules[t.ID]; ok {
			return sched
		}
		return "Unknown"
	})
}

// buildGroupedList renders displayTasks with headers when group changes
func (c Content) buildGroupedList(getGroup func(*task.Task) string) string {
	var sections []string
	var currentGroup string
	var currentRows []string

	for i := range c.displayTasks {
		t := &c.displayTasks[i]
		group := getGroup(t)

		if group != currentGroup {
			// Flush previous group
			if len(currentRows) > 0 {
				header := c.styles.Theme.Header.Render(currentGroup)
				sections = append(sections, header+"\n"+strings.Join(currentRows, "\n"))
			}
			currentGroup = group
			currentRows = nil
		}
		currentRows = append(currentRows, c.renderTaskRow(t, i))
	}

	// Flush last group
	if len(currentRows) > 0 {
		header := c.styles.Theme.Header.Render(currentGroup)
		sections = append(sections, header+"\n"+strings.Join(currentRows, "\n"))
	}

	return strings.Join(sections, "\n\n")
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

	return c.card.Render(c.title, content, c.width, c.height, c.focused)
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
	return len(c.displayTasks)
}

// renderTaskRow formats a single task row
func (c Content) renderTaskRow(t *task.Task, index int) string {
	theme := c.styles.Theme
	isSelected := (c.focused || c.showSelection) && index == c.selectedIndex

	// Prefix: selection indicator, check for done, flag for due, star for planned today
	prefix := "  "
	if t.Status == task.StatusDone {
		prefix = theme.Success.Render(theme.Icons.Done) + " "
	} else if c.isDueOrOverdue(t) {
		prefix = theme.Warning.Render(theme.Icons.Due) + " "
	} else if c.isPlannedForToday(t) {
		prefix = theme.Accent.Render(theme.Icons.Planned) + " "
	}

	// ID
	id := theme.ID.Render(fmt.Sprintf("%d", t.ID))

	// Build scope and title differently for projects vs tasks
	var scope, title string
	if t.IsProject() {
		// For projects in scope-hidden view (e.g., area view), show project name as title with scope styling
		// Otherwise, show full scope (Area > ProjectName)
		if c.hideScope {
			title = theme.Scope.Render(c.sanitizeTitle(t.Title))
		} else {
			scope = c.formatProjectScope(t.AreaName, t.Title)
			scope = theme.Scope.Render(scope)
		}
	} else {
		// For regular tasks: scope and title
		scope = c.formatScope(t.AreaName, t.ParentName)
		if scope != "" {
			scope = theme.Scope.Render(scope)
		}
		title = c.sanitizeTitle(t.Title)
	}

	// Extras: recurrence, dates, tags (only recurrence for regular tasks)
	var extras []string

	if !t.IsProject() {
		if recur := c.formatRecurIndicator(t); recur != "" {
			extras = append(extras, theme.Muted.Render(recur))
		}
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
	if title != "" {
		parts = append(parts, title)
	}
	if len(extras) > 0 {
		parts = append(parts, strings.Join(extras, " "))
	}

	row := strings.Join(parts, "  ")

	// Apply selection highlighting
	if isSelected {
		row = c.styles.SelectedItem.Render("> " + row)
	} else {
		row = "  " + row
	}

	return row
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

// formatProjectScope returns the scope display for a project row.
// Shows "area > projectTitle" when area exists, just "projectTitle" otherwise.
func (c Content) formatProjectScope(areaName *string, projectTitle string) string {
	if areaName != nil {
		return *areaName + " > " + c.sanitizeTitle(projectTitle)
	}
	return c.sanitizeTitle(projectTitle)
}

// renderProjectHeaderLine renders a project as a standalone header-style line
// for scope-grouped views. Format: [Area > ProjectName]  [planned] [due] [tags]
func (c Content) renderProjectHeaderLine(t *task.Task, index int) string {
	theme := c.styles.Theme
	isSelected := (c.focused || c.showSelection) && index == c.selectedIndex

	// Show full scope: "Area > ProjectName" or just "ProjectName"
	scope := c.sanitizeTitle(t.Title)
	if t.AreaName != nil {
		scope = *t.AreaName + " > " + scope
	}
	parts := []string{theme.Header.Render(scope)}

	if t.PlannedDate != nil {
		parts = append(parts, theme.Muted.Render(theme.Icons.Date+" "+t.PlannedDate.Format("Jan 2")))
	}
	if t.DueDate != nil {
		parts = append(parts, theme.Muted.Render(theme.Icons.Due+" "+t.DueDate.Format("Jan 2")))
	}
	if len(t.Tags) > 0 {
		parts = append(parts, theme.Muted.Render(c.formatTags(t.Tags)))
	}

	row := strings.Join(parts, "  ")

	if isSelected {
		row = c.styles.SelectedItem.Render("> " + row)
	}

	return row
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

// SetFocused sets whether the content panel has focus
func (c Content) SetFocused(focused bool) Content {
	c.focused = focused
	if focused && len(c.displayTasks) > 0 {
		if c.selectedIndex < 0 {
			c.selectedIndex = 0
		}
	} else if !c.showSelection {
		c.selectedIndex = -1
	}
	if c.ready {
		c.viewport.SetContent(c.buildTaskList())
	}
	return c
}

// SetShowSelection sets whether to show the selection indicator even when not focused
func (c Content) SetShowSelection(show bool) Content {
	c.showSelection = show
	if c.ready {
		c.viewport.SetContent(c.buildTaskList())
	}
	return c
}

// MoveUp moves selection up
func (c Content) MoveUp() Content {
	if c.selectedIndex > 0 {
		c.selectedIndex--
		if c.ready {
			c.viewport.SetContent(c.buildTaskList())
			c = c.ensureSelectionVisible()
		}
	}
	return c
}

// MoveDown moves selection down
func (c Content) MoveDown() Content {
	if c.selectedIndex < len(c.displayTasks)-1 {
		c.selectedIndex++
		if c.ready {
			c.viewport.SetContent(c.buildTaskList())
			c = c.ensureSelectionVisible()
		}
	}
	return c
}

// selectedTaskLine calculates the line number of the selected task in rendered output
func (c Content) selectedTaskLine() int {
	if c.selectedIndex < 0 || len(c.displayTasks) == 0 {
		return 0
	}

	// For flat lists, line = index
	if c.groupBy == "" || c.groupBy == "none" {
		return c.selectedIndex
	}

	// For grouped lists, count headers and blank lines
	var getGroup func(*task.Task) string
	var isProjectItem func(*task.Task) bool

	switch c.groupBy {
	case "scope":
		getGroup = func(t *task.Task) string {
			// Projects are their own groups
			if t.IsProject() {
				scope := c.sanitizeTitle(t.Title)
				if t.AreaName != nil {
					scope = *t.AreaName + " > " + c.sanitizeTitle(t.Title)
				}
				return "project:" + scope // prefix to distinguish from regular groups
			}
			if t.ParentName == nil {
				if t.AreaName == nil {
					return "No Scope"
				}
				return *t.AreaName
			}
			if t.AreaName != nil {
				return *t.AreaName + " > " + *t.ParentName
			}
			return *t.ParentName
		}
		isProjectItem = func(t *task.Task) bool {
			return t.IsProject()
		}
	case "schedule":
		getGroup = func(t *task.Task) string {
			if sched, ok := c.taskSchedules[t.ID]; ok {
				return sched
			}
			return "Unknown"
		}
		isProjectItem = func(t *task.Task) bool { return false }
	case "date":
		now := time.Now()
		todayYear, todayMonth, todayDay := now.Date()
		today := time.Date(todayYear, todayMonth, todayDay, 0, 0, 0, 0, time.Local)
		tomorrow := today.AddDate(0, 0, 1)
		endOfWeek := today.AddDate(0, 0, 7-int(today.Weekday()))
		endOfMonth := time.Date(todayYear, todayMonth+1, 0, 0, 0, 0, 0, time.Local)
		endOfYear := time.Date(todayYear, 12, 31, 0, 0, 0, 0, time.Local)
		getGroup = func(t *task.Task) string {
			return c.getDateCategory(t.PlannedDate, t.DueDate, today, tomorrow, endOfWeek, endOfMonth, endOfYear)
		}
		isProjectItem = func(t *task.Task) bool { return false }
	default:
		return c.selectedIndex
	}

	line := 0
	currentGroup := ""
	for i := 0; i <= c.selectedIndex; i++ {
		t := &c.displayTasks[i]
		group := getGroup(t)
		if group != currentGroup {
			if currentGroup != "" {
				line++ // blank line between groups
			}
			// For projects in scope view, the "header" IS the selectable item
			// For regular groups, there's a header line followed by task lines
			if !isProjectItem(t) {
				line++ // header line (only for non-project groups)
			}
			currentGroup = group
		}
		if i == c.selectedIndex {
			return line
		}
		line++ // task/project line
	}
	return line
}

// ensureSelectionVisible scrolls viewport to keep selected task visible
func (c Content) ensureSelectionVisible() Content {
	if !c.ready || c.selectedIndex < 0 {
		return c
	}

	line := c.selectedTaskLine()
	yOffset := c.viewport.YOffset
	height := c.viewport.Height

	if line < yOffset {
		c.viewport.SetYOffset(line)
	} else if line >= yOffset+height {
		c.viewport.SetYOffset(line - height + 1)
	}

	return c
}

// SelectedTask returns the currently selected task, or nil if none
func (c Content) SelectedTask() *task.Task {
	if !c.focused || c.selectedIndex < 0 || c.selectedIndex >= len(c.displayTasks) {
		return nil
	}
	return &c.displayTasks[c.selectedIndex]
}

// UpdateTaskStatus updates a task's status in-place and refreshes the viewport
func (c Content) UpdateTaskStatus(taskID int64, done bool) Content {
	for i := range c.displayTasks {
		if c.displayTasks[i].ID == taskID {
			if done {
				c.displayTasks[i].Status = task.StatusDone
			} else {
				c.displayTasks[i].Status = task.StatusTodo
			}
			break
		}
	}
	if c.ready {
		c.viewport.SetContent(c.buildTaskList())
	}
	return c
}

// computeDisplayOrder returns tasks sorted by the given grouping mode
func (c Content) computeDisplayOrder(tasks []task.Task, groupBy string) []task.Task {
	switch groupBy {
	case "scope":
		return c.orderByScope(tasks)
	case "date":
		return c.orderByDate(tasks)
	default:
		return tasks
	}
}

// orderByScope sorts tasks by scope grouping
// Projects are treated as standalone entries sorted by their scope (Area > ProjectTitle)
func (c Content) orderByScope(tasks []task.Task) []task.Task {
	noScope := make([]task.Task, 0)
	groups := make(map[string][]task.Task)
	projectsByScope := make(map[string]task.Task)

	for _, t := range tasks {
		if t.IsProject() {
			// Projects are standalone entries sorted by their scope
			scope := c.sanitizeTitle(t.Title)
			if t.AreaName != nil {
				scope = *t.AreaName + " > " + c.sanitizeTitle(t.Title)
			}
			projectsByScope[scope] = t
			continue
		}

		if t.ParentName == nil {
			if t.AreaName == nil {
				noScope = append(noScope, t)
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

	var result []task.Task
	result = append(result, noScope...)

	// Combine all headers (group headers + project scopes) and sort
	allHeaders := make([]string, 0, len(groups)+len(projectsByScope))
	for h := range groups {
		allHeaders = append(allHeaders, h)
	}
	for h := range projectsByScope {
		allHeaders = append(allHeaders, h)
	}
	sort.Strings(allHeaders)

	for _, header := range allHeaders {
		if proj, isProject := projectsByScope[header]; isProject {
			result = append(result, proj)
		} else if tasks, isGroup := groups[header]; isGroup {
			result = append(result, tasks...)
		}
	}
	return result
}

// orderByDate sorts tasks by date category
func (c Content) orderByDate(tasks []task.Task) []task.Task {
	now := time.Now()
	todayYear, todayMonth, todayDay := now.Date()
	today := time.Date(todayYear, todayMonth, todayDay, 0, 0, 0, 0, time.Local)
	tomorrow := today.AddDate(0, 0, 1)
	endOfWeek := today.AddDate(0, 0, 7-int(today.Weekday()))
	endOfMonth := time.Date(todayYear, todayMonth+1, 0, 0, 0, 0, 0, time.Local)
	endOfYear := time.Date(todayYear, 12, 31, 0, 0, 0, 0, time.Local)

	dateGroups := map[string][]task.Task{
		"Overdue": {}, "Today": {}, "Tomorrow": {}, "This Week": {},
		"This Month": {}, "This Year": {}, "Later": {}, "No Date": {},
	}

	for _, t := range tasks {
		category := c.getDateCategory(t.PlannedDate, t.DueDate, today, tomorrow, endOfWeek, endOfMonth, endOfYear)
		dateGroups[category] = append(dateGroups[category], t)
	}

	orderedCategories := []string{"Overdue", "Today", "Tomorrow", "This Week", "This Month", "This Year", "Later", "No Date"}
	var result []task.Task
	for _, category := range orderedCategories {
		result = append(result, dateGroups[category]...)
	}
	return result
}
