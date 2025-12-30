package tui

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/devbydaniel/tt/config"
	"github.com/devbydaniel/tt/internal/domain/area"
	"github.com/devbydaniel/tt/internal/domain/project"
	"github.com/devbydaniel/tt/internal/domain/task"
	"github.com/devbydaniel/tt/internal/output"
)

// FocusArea indicates which panel has focus
type FocusArea int

const (
	FocusSidebar FocusArea = iota
	FocusContent
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
	sidebar     Sidebar
	content     Content
	renameModal RenameModal
	moveModal   MoveModal
	dateModal   DateModal
	addModal    AddModal
	help        help.Model
	focusArea   FocusArea

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

	// Initialize help with theme-matching styles
	helpModel := help.New()
	helpModel.Styles.ShortKey = theme.Accent
	helpModel.Styles.ShortDesc = theme.Muted
	helpModel.Styles.ShortSeparator = theme.Muted

	return Model{
		taskService:    taskService,
		areaService:    areaService,
		projectService: projectService,
		config:         cfg,
		styles:         styles,
		sidebar:        NewSidebar(styles),
		content:        NewContent(styles),
		renameModal:    NewRenameModal(styles),
		moveModal:      NewMoveModal(styles),
		dateModal:      NewDateModal(styles),
		addModal:       NewAddModal(styles),
		help:           helpModel,
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
		// Route keys to add modal when active
		if m.addModal.Active() {
			var result *AddResult
			m.addModal, result = m.addModal.Update(msg)
			if result != nil && !result.Canceled {
				return m, m.createTask(result)
			}
			return m, nil
		}

		// Route keys to rename modal when active
		if m.renameModal.Active() {
			var result *RenameResult
			m.renameModal, result = m.renameModal.Update(msg)
			if result != nil && !result.Canceled {
				// Rename was confirmed, update the task
				return m, m.renameTask(result.TaskID, result.NewTitle)
			}
			return m, nil
		}

		// Route keys to move modal when active
		if m.moveModal.Active() {
			var result *MoveResult
			m.moveModal, result = m.moveModal.Update(msg)
			if result != nil && !result.Canceled {
				return m, m.moveTask(result.TaskID, result.ItemType, result.Name)
			}
			return m, nil
		}

		// Route keys to date modal when active
		if m.dateModal.Active() {
			var result *DateResult
			m.dateModal, result = m.dateModal.Update(msg)
			if result != nil && !result.Canceled {
				return m, m.setTaskDate(result.TaskID, result.Date, result.Mode)
			}
			return m, nil
		}

		switch {
		case key.Matches(msg, keys.Quit):
			return m, tea.Quit

		case key.Matches(msg, keys.Enter):
			if m.focusArea == FocusSidebar {
				m.focusArea = FocusContent
				m.content = m.content.SetFocused(true)
				return m, nil
			}

		case key.Matches(msg, keys.Escape), key.Matches(msg, keys.FocusSidebar):
			if m.focusArea == FocusContent {
				m.focusArea = FocusSidebar
				m.content = m.content.SetFocused(false)
				return m, nil
			}

		case key.Matches(msg, keys.FocusContent):
			if m.focusArea == FocusSidebar {
				m.focusArea = FocusContent
				m.content = m.content.SetFocused(true)
				return m, nil
			}

		case key.Matches(msg, keys.Rename):
			if m.focusArea == FocusContent {
				if selectedTask := m.content.SelectedTask(); selectedTask != nil {
					m.renameModal = m.renameModal.SetSize(m.width, m.height-1) // -1 for help bar
					m.renameModal = m.renameModal.Open(selectedTask.ID, selectedTask.Title)
					return m, nil
				}
			}

		case key.Matches(msg, keys.Move):
			if m.focusArea == FocusContent {
				if selectedTask := m.content.SelectedTask(); selectedTask != nil {
					m.moveModal = m.moveModal.SetSize(m.width, m.height-1) // -1 for help bar
					m.moveModal = m.moveModal.Open(selectedTask.ID, m.projects, m.areas)
					return m, nil
				}
			}

		case key.Matches(msg, keys.Planned):
			if m.focusArea == FocusContent {
				if selectedTask := m.content.SelectedTask(); selectedTask != nil {
					m.dateModal = m.dateModal.SetSize(m.width, m.height-1) // -1 for help bar
					m.dateModal = m.dateModal.Open(selectedTask.ID, DateModalPlanned, selectedTask.PlannedDate)
					return m, nil
				}
			}

		case key.Matches(msg, keys.Due):
			if m.focusArea == FocusContent {
				if selectedTask := m.content.SelectedTask(); selectedTask != nil {
					m.dateModal = m.dateModal.SetSize(m.width, m.height-1) // -1 for help bar
					m.dateModal = m.dateModal.Open(selectedTask.ID, DateModalDue, selectedTask.DueDate)
					return m, nil
				}
			}

		case key.Matches(msg, keys.Add):
			// Open add modal, pre-fill scope if viewing a project or area
			sidebarItem := m.sidebar.SelectedItem()
			var prefill *SidebarItem
			if sidebarItem.Type == "project" || sidebarItem.Type == "area" {
				prefill = &sidebarItem
			}
			m.addModal = m.addModal.SetSize(m.width, m.height-1)
			m.addModal = m.addModal.Open(m.projects, m.areas, prefill)
			return m, nil

		case key.Matches(msg, keys.Tab):
			m.sidebar = m.sidebar.NextSection()
			return m, m.loadTasksForSelection

		case key.Matches(msg, keys.ShiftTab):
			m.sidebar = m.sidebar.PrevSection()
			return m, m.loadTasksForSelection

		case key.Matches(msg, keys.Up):
			if m.focusArea == FocusContent {
				m.content = m.content.MoveUp()
				return m, nil
			}
			m.sidebar = m.sidebar.MoveUp()
			return m, m.loadTasksForSelection

		case key.Matches(msg, keys.Down):
			if m.focusArea == FocusContent {
				m.content = m.content.MoveDown()
				return m, nil
			}
			m.sidebar = m.sidebar.MoveDown()
			return m, m.loadTasksForSelection
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Reserve 1 row for help bar at the bottom
		helpHeight := 1
		availableHeight := m.height - helpHeight

		// Sidebar gets 1/4, content gets rest minus 1-char gap between columns
		sidebarWidth := m.width / 4
		if sidebarWidth < 20 {
			sidebarWidth = 20
		}
		contentWidth := m.width - sidebarWidth - 1

		// Calculate sidebar height that's evenly divisible by 3 (number of sections)
		// This ensures both columns end at the same row
		sidebarHeight := (availableHeight / 3) * 3

		m.sidebar = m.sidebar.SetSize(sidebarWidth, sidebarHeight)
		m.content = m.content.SetSize(contentWidth, sidebarHeight)
		m.help.Width = m.width
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

	case taskRenamedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		// Reload tasks to show the updated title
		return m, m.loadTasksForSelection

	case taskMovedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		// Reload tasks to reflect the move
		return m, m.loadTasksForSelection

	case taskDateUpdatedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		// Reload tasks to reflect the date change
		return m, m.loadTasksForSelection

	case taskCreatedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		// Reload tasks to show the new task
		return m, m.loadTasksForSelection
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

// taskRenamedMsg carries the result of a task rename
type taskRenamedMsg struct {
	task *task.Task
	err  error
}

// taskMovedMsg carries the result of a task move
type taskMovedMsg struct {
	task *task.Task
	err  error
}

// taskDateUpdatedMsg carries the result of a date update
type taskDateUpdatedMsg struct {
	task *task.Task
	err  error
}

// taskCreatedMsg carries the result of creating a task
type taskCreatedMsg struct {
	task *task.Task
	err  error
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

// renameTask creates a command to rename a task
func (m Model) renameTask(taskID int64, newTitle string) tea.Cmd {
	return func() tea.Msg {
		updated, err := m.taskService.SetTitle(taskID, newTitle)
		return taskRenamedMsg{task: updated, err: err}
	}
}

// moveTask creates a command to move a task to a project or area
func (m Model) moveTask(taskID int64, itemType, name string) tea.Cmd {
	return func() tea.Msg {
		var updated *task.Task
		var err error

		switch itemType {
		case "project":
			updated, err = m.taskService.SetProject(taskID, name)
		case "area":
			updated, err = m.taskService.SetArea(taskID, name)
		}

		return taskMovedMsg{task: updated, err: err}
	}
}

// setTaskDate creates a command to set a task's planned or due date
func (m Model) setTaskDate(taskID int64, date *time.Time, mode DateModalMode) tea.Cmd {
	return func() tea.Msg {
		var updated *task.Task
		var err error

		switch mode {
		case DateModalPlanned:
			updated, err = m.taskService.SetPlannedDate(taskID, date)
		case DateModalDue:
			updated, err = m.taskService.SetDueDate(taskID, date)
		}

		return taskDateUpdatedMsg{task: updated, err: err}
	}
}

// createTask creates a command to create a new task
func (m Model) createTask(result *AddResult) tea.Cmd {
	return func() tea.Msg {
		opts := &task.CreateOptions{
			ProjectName: result.ProjectName,
			AreaName:    result.AreaName,
			Description: result.Description,
			PlannedDate: result.PlannedDate,
			DueDate:     result.DueDate,
			Tags:        result.Tags,
		}

		created, err := m.taskService.Create(result.Title, opts)
		return taskCreatedMsg{task: created, err: err}
	}
}

// View implements tea.Model
func (m Model) View() string {
	if m.err != nil {
		return "Error: " + m.err.Error() + "\n\nPress q to quit."
	}

	if m.width == 0 {
		return "Loading..."
	}

	// Determine which help keys to show based on current state
	var helpView string
	switch {
	case m.addModal.Active():
		helpView = m.help.View(addKeys)
	case m.renameModal.Active():
		helpView = m.help.View(renameKeys)
	case m.moveModal.Active():
		helpView = m.help.View(moveKeys)
	case m.dateModal.Active():
		if m.dateModal.FocusInput() {
			helpView = m.help.View(dateInputKeys)
		} else {
			helpView = m.help.View(datePickerKeys)
		}
	case m.focusArea == FocusSidebar:
		helpView = m.help.View(sidebarKeys)
	default:
		helpView = m.help.View(contentKeys)
	}
	helpView = lipgloss.PlaceHorizontal(m.width, lipgloss.Center, helpView)

	// Render modal if active (with help bar below)
	if m.addModal.Active() {
		return lipgloss.JoinVertical(lipgloss.Left, m.addModal.View(), helpView)
	}
	if m.renameModal.Active() {
		return lipgloss.JoinVertical(lipgloss.Left, m.renameModal.View(), helpView)
	}
	if m.moveModal.Active() {
		return lipgloss.JoinVertical(lipgloss.Left, m.moveModal.View(), helpView)
	}
	if m.dateModal.Active() {
		return lipgloss.JoinVertical(lipgloss.Left, m.dateModal.View(), helpView)
	}

	// Render sidebar and content side by side with 1-char gap
	contentView := lipgloss.NewStyle().MarginLeft(1).Render(m.content.View())
	mainView := lipgloss.JoinHorizontal(lipgloss.Top, m.sidebar.View(), contentView)

	// Combine main view with help bar at the bottom
	return lipgloss.JoinVertical(lipgloss.Left, mainView, helpView)
}
