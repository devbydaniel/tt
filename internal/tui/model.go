package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/devbydaniel/tt/config"
	"github.com/devbydaniel/tt/internal/domain/area"
	"github.com/devbydaniel/tt/internal/domain/project"
	"github.com/devbydaniel/tt/internal/domain/task"
	"github.com/devbydaniel/tt/internal/output"
)

// Model is the root TUI model
type Model struct {
	// Services
	taskService    *task.Service
	areaService    *area.Service
	projectService *project.Service

	// Config
	config *config.Config

	// Styles
	styles *Styles

	// Dimensions
	width  int
	height int

	// Components
	sidebar Sidebar
	content Content

	// Cached data
	areas    []area.Area
	projects []project.ProjectWithArea
	tags     []string

	// Error state
	err error
}

// NewModel creates a new TUI model
func NewModel(taskService *task.Service, areaService *area.Service, projectService *project.Service, theme *output.Theme, cfg *config.Config) Model {
	styles := NewStyles(theme)

	return Model{
		taskService:    taskService,
		areaService:    areaService,
		projectService: projectService,
		config:         cfg,
		styles:         styles,
		sidebar:        NewSidebar(styles),
		content:        NewContent(styles),
	}
}

// configKeyForSelection returns the config key for the current sidebar selection
func (m Model) configKeyForSelection() string {
	item := m.sidebar.SelectedItem()
	switch item.Type {
	case "static":
		return item.Key // "inbox", "today", "upcoming", "anytime", "someday"
	case "project":
		return "project"
	case "area":
		return "area"
	case "tag":
		return "tag"
	}
	return "all"
}

// Init implements tea.Model
func (m Model) Init() tea.Cmd {
	return m.loadData
}

// loadDataMsg carries loaded data
type loadDataMsg struct {
	areas    []area.Area
	projects []project.ProjectWithArea
	tags     []string
	tasks    []task.Task
	err      error
}

// loadData fetches initial data
func (m Model) loadData() tea.Msg {
	areas, err := m.areaService.List()
	if err != nil {
		return loadDataMsg{err: err}
	}

	projects, err := m.projectService.ListWithArea()
	if err != nil {
		return loadDataMsg{err: err}
	}

	tags, err := m.taskService.ListTags()
	if err != nil {
		return loadDataMsg{err: err}
	}

	// Load today's tasks by default with sort from config
	sortStr := m.config.GetSort("today")
	sortOpts, _ := task.ParseSort(sortStr)
	tasks, err := m.taskService.List(&task.ListOptions{Schedule: "today", Sort: sortOpts})
	if err != nil {
		return loadDataMsg{err: err}
	}

	return loadDataMsg{
		areas:    areas,
		projects: projects,
		tags:     tags,
		tasks:    tasks,
	}
}

// Update implements tea.Model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Quit):
			return m, tea.Quit

		case key.Matches(msg, keys.Tab):
			m.sidebar = m.sidebar.NextSection()
			return m, m.loadTasksForSelection

		case key.Matches(msg, keys.ShiftTab):
			m.sidebar = m.sidebar.PrevSection()
			return m, m.loadTasksForSelection

		case key.Matches(msg, keys.Up):
			m.sidebar = m.sidebar.MoveUp()
			return m, m.loadTasksForSelection

		case key.Matches(msg, keys.Down):
			m.sidebar = m.sidebar.MoveDown()
			return m, m.loadTasksForSelection
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Sidebar gets 1/4, content gets rest minus 1-char gap between columns
		sidebarWidth := m.width / 4
		if sidebarWidth < 20 {
			sidebarWidth = 20
		}
		contentWidth := m.width - sidebarWidth - 1

		m.sidebar = m.sidebar.SetSize(sidebarWidth, m.height)
		m.content = m.content.SetSize(contentWidth, m.height)
		return m, nil

	case loadDataMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.areas = msg.areas
		m.projects = msg.projects
		m.tags = msg.tags
		m.sidebar = m.sidebar.SetData(msg.areas, msg.projects, msg.tags)
		// Get groupBy and hideScope for initial "today" view
		groupBy := m.config.GetGroup("today")
		hideScope := m.config.GetHideScope("today")
		m.content = m.content.SetTasks(msg.tasks, "Today", groupBy, hideScope)
		return m, nil

	case tasksLoadedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.content = m.content.SetTasks(msg.tasks, msg.title, msg.groupBy, msg.hideScope)
		return m, nil

	case scheduleTasksLoadedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.content = m.content.SetScheduleGroups(msg.groups, msg.title, msg.hideScope)
		return m, nil
	}

	return m, nil
}

// tasksLoadedMsg carries loaded tasks for a selection
type tasksLoadedMsg struct {
	tasks     []task.Task
	title     string
	groupBy   string
	hideScope bool
	err       error
}

// ScheduleGroups holds tasks grouped by schedule
type ScheduleGroups struct {
	Today    []task.Task
	Upcoming []task.Task
	Anytime  []task.Task
	Someday  []task.Task
}

// scheduleTasksLoadedMsg carries schedule-grouped tasks
type scheduleTasksLoadedMsg struct {
	groups    ScheduleGroups
	title     string
	hideScope bool
	err       error
}

// loadTasksForSelection loads tasks based on sidebar selection
func (m Model) loadTasksForSelection() tea.Msg {
	item := m.sidebar.SelectedItem()
	title := strings.TrimSpace(item.Label)

	// Get sort, group, and hideScope settings from config
	configKey := m.configKeyForSelection()
	groupBy := m.config.GetGroup(configKey)
	sortStr := m.config.GetSort(configKey)
	hideScope := m.config.GetHideScope(configKey)
	sortOpts, _ := task.ParseSort(sortStr)

	// Schedule grouping requires 4 separate queries
	if groupBy == "schedule" {
		return m.loadScheduleGroups(item, title, sortOpts, hideScope)
	}

	// Build list options based on selection
	opts := m.buildListOptions(item)
	opts.Sort = sortOpts

	tasks, err := m.taskService.List(opts)
	if err != nil {
		return tasksLoadedMsg{err: err}
	}

	return tasksLoadedMsg{tasks: tasks, title: title, groupBy: groupBy, hideScope: hideScope}
}

// buildListOptions creates ListOptions based on sidebar selection
func (m Model) buildListOptions(item SidebarItem) *task.ListOptions {
	opts := &task.ListOptions{}

	switch item.Type {
	case "static":
		opts.Schedule = item.Key
	case "area":
		opts.AreaName = item.Key
	case "project":
		opts.ProjectName = item.Key
	case "tag":
		opts.TagName = item.Key
	}

	return opts
}

// loadScheduleGroups loads tasks grouped by schedule (4 separate queries)
func (m Model) loadScheduleGroups(item SidebarItem, title string, sortOpts []task.SortOption, hideScope bool) tea.Msg {
	var groups ScheduleGroups

	schedules := []struct {
		schedule string
		target   *[]task.Task
	}{
		{"today", &groups.Today},
		{"upcoming", &groups.Upcoming},
		{"anytime", &groups.Anytime},
		{"someday", &groups.Someday},
	}

	for _, sched := range schedules {
		opts := m.buildListOptions(item)
		opts.Schedule = sched.schedule
		opts.Sort = sortOpts

		tasks, err := m.taskService.List(opts)
		if err != nil {
			return scheduleTasksLoadedMsg{err: err}
		}
		*sched.target = tasks
	}

	return scheduleTasksLoadedMsg{groups: groups, title: title, hideScope: hideScope}
}

// View implements tea.Model
func (m Model) View() string {
	if m.err != nil {
		return "Error: " + m.err.Error() + "\n\nPress q to quit."
	}

	if m.width == 0 {
		return "Loading..."
	}

	// Render sidebar and content side by side with 1-char gap
	contentView := lipgloss.NewStyle().MarginLeft(1).Render(m.content.View())
	return lipgloss.JoinHorizontal(lipgloss.Top, m.sidebar.View(), contentView)
}
