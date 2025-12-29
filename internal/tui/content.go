package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/devbydaniel/tt/internal/domain/task"
	"github.com/devbydaniel/tt/internal/recurparse"
)

// Content displays the task list in the right panel
type Content struct {
	title    string
	tasks    []task.Task
	width    int
	height   int
	viewport viewport.Model
	ready    bool
	styles   *Styles
	card     *Card
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

// SetTasks updates the displayed tasks
func (c Content) SetTasks(tasks []task.Task, title string) Content {
	c.tasks = tasks
	c.title = title
	if c.ready {
		c.viewport.SetContent(c.buildTaskList())
		c.viewport.GotoTop()
	}
	return c
}

// buildTaskList renders all tasks as a string
func (c Content) buildTaskList() string {
	if len(c.tasks) == 0 {
		return c.styles.Theme.Muted.Render("No tasks")
	}

	var rows []string
	for _, t := range c.tasks {
		row := c.renderTaskRow(&t)
		rows = append(rows, row)
	}

	return strings.Join(rows, "\n")
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
	if scope != "" {
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
