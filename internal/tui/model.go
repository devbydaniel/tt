package tui

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/devbydaniel/tt/config"
	"github.com/devbydaniel/tt/internal/app"
	"github.com/devbydaniel/tt/internal/domain/area"
	"github.com/devbydaniel/tt/internal/domain/task"
	"github.com/devbydaniel/tt/internal/output"
)

// FocusArea indicates which panel has focus
type FocusArea int

const (
	FocusSidebar FocusArea = iota
	FocusContent
	FocusDetail
)

// Model is the root TUI model
type Model struct {
	// Application
	app *app.App

	// Config
	config *config.Config

	// Styles
	styles *Styles

	// Dimensions
	width  int
	height int
	gap    int // Gap between sidebar and content (can be 0 for tight layouts)

	// Components
	sidebar          Sidebar
	content          Content
	detailPane       DetailPane
	renameModal      RenameModal
	moveModal        MoveModal
	dateModal        DateModal
	addModal         AddModal
	tagModal         TagModal
	descriptionModal DescriptionModal
	help             help.Model
	focusArea        FocusArea
	detailVisible    bool // whether the detail pane is shown

	// Cached data
	areas    []area.Area
	projects []task.Task
	tags     []string

	// Error state
	err error
}

// NewModel creates a new TUI model
func NewModel(application *app.App, theme *output.Theme, cfg *config.Config) Model {
	styles := NewStyles(theme)

	// Initialize help with theme-matching styles
	helpModel := help.New()
	helpModel.Styles.ShortKey = theme.Accent
	helpModel.Styles.ShortDesc = theme.Muted
	helpModel.Styles.ShortSeparator = theme.Muted

	return Model{
		app:              application,
		config:           cfg,
		styles:           styles,
		gap:              1, // Default gap, adjusted on resize
		sidebar:          NewSidebar(styles),
		content:          NewContent(styles),
		detailPane:       NewDetailPane(styles),
		renameModal:      NewRenameModal(styles),
		moveModal:        NewMoveModal(styles),
		dateModal:        NewDateModal(styles),
		addModal:         NewAddModal(styles),
		tagModal:         NewTagModal(styles),
		descriptionModal: NewDescriptionModal(styles),
		help:             helpModel,
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
	projects []task.Task
	tags     []string
	tasks    []task.Task
	err      error
}

// loadData fetches initial data
func (m Model) loadData() tea.Msg {
	areas, err := m.app.ListAreas.Execute()
	if err != nil {
		return loadDataMsg{err: err}
	}

	projects, err := m.app.ListProjectsWithArea.Execute()
	if err != nil {
		return loadDataMsg{err: err}
	}

	tags, err := m.app.ListTags.Execute()
	if err != nil {
		return loadDataMsg{err: err}
	}

	// Load today's tasks by default with sort from config
	sortStr := m.config.GetSort("today")
	sortOpts, _ := task.ParseSort(sortStr)
	tasks, err := m.app.ListTasks.Execute(&task.ListOptions{Schedule: "today", Sort: sortOpts})
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

		// Route keys to tag modal when active
		if m.tagModal.Active() {
			var result *TagResult
			m.tagModal, result = m.tagModal.Update(msg)
			if result != nil && !result.Canceled {
				return m, m.setTaskTags(result.TaskID, result.Tags)
			}
			return m, nil
		}

		// Route keys to description modal when active
		if m.descriptionModal.Active() {
			var result *DescriptionResult
			m.descriptionModal, result = m.descriptionModal.Update(msg)
			if result != nil && !result.Canceled {
				return m, m.setTaskDescription(result.TaskID, result.Description)
			}
			return m, nil
		}

		switch {
		case key.Matches(msg, keys.Quit):
			return m, tea.Quit

		case key.Matches(msg, keys.Enter):
			if m.focusArea == FocusSidebar {
				m.focusArea = FocusContent
				m.sidebar = m.sidebar.SetFocused(false)
				m.content = m.content.SetFocused(true)
				return m, nil
			}
			if m.focusArea == FocusContent {
				// Enter from content opens detail pane
				return m.openDetailPane()
			}
			if m.focusArea == FocusDetail {
				// Open modal for focused field
				return m.openDetailFieldModal()
			}

		case key.Matches(msg, keys.Escape), key.Matches(msg, keys.FocusSidebar):
			if m.focusArea == FocusDetail {
				// Close detail pane, return to content
				m.focusArea = FocusContent
				m.detailPane = m.detailPane.SetFocused(false)
				m.detailVisible = false
				m.content = m.content.SetShowSelection(false)
				m.content = m.content.SetFocused(true)
				m = m.recalculateLayout()
				return m, nil
			}
			if m.focusArea == FocusContent {
				m.focusArea = FocusSidebar
				m.sidebar = m.sidebar.SetFocused(true)
				m.content = m.content.SetShowSelection(false)
				m.content = m.content.SetFocused(false)
				m.detailVisible = false
				m = m.recalculateLayout()
				return m, nil
			}

		case key.Matches(msg, keys.FocusContent):
			if m.focusArea == FocusSidebar {
				m.focusArea = FocusContent
				m.sidebar = m.sidebar.SetFocused(false)
				m.content = m.content.SetFocused(true)
				return m, nil
			}
			if m.focusArea == FocusContent {
				// l from content opens detail pane
				return m.openDetailPane()
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

		case key.Matches(msg, keys.Tags):
			if m.focusArea == FocusContent {
				if selectedTask := m.content.SelectedTask(); selectedTask != nil {
					m.tagModal = m.tagModal.SetSize(m.width, m.height-1) // -1 for help bar
					m.tagModal = m.tagModal.Open(selectedTask.ID, selectedTask.Tags, m.tags)
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

		case key.Matches(msg, keys.Toggle):
			if m.focusArea == FocusContent {
				if selectedTask := m.content.SelectedTask(); selectedTask != nil {
					return m, m.toggleTask(selectedTask.ID, selectedTask.Status)
				}
			}

		case key.Matches(msg, keys.Tab):
			if m.focusArea == FocusDetail {
				m.detailPane = m.detailPane.NextField()
				return m, nil
			}
			m.sidebar = m.sidebar.NextSection()
			return m, m.loadTasksForSelection

		case key.Matches(msg, keys.ShiftTab):
			if m.focusArea == FocusDetail {
				m.detailPane = m.detailPane.PrevField()
				return m, nil
			}
			m.sidebar = m.sidebar.PrevSection()
			return m, m.loadTasksForSelection

		case key.Matches(msg, keys.Up):
			if m.focusArea == FocusDetail {
				m.detailPane = m.detailPane.PrevField()
				return m, nil
			}
			if m.focusArea == FocusContent {
				m.content = m.content.MoveUp()
				return m, nil
			}
			m.sidebar = m.sidebar.MoveUp()
			return m, m.loadTasksForSelection

		case key.Matches(msg, keys.Down):
			if m.focusArea == FocusDetail {
				m.detailPane = m.detailPane.NextField()
				return m, nil
			}
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

		// Calculate sidebar width: 1/4 of total, constrained between min/max
		sidebarWidth := m.width / 4
		minSidebar := 20
		maxSidebar := 40

		if sidebarWidth < minSidebar {
			sidebarWidth = minSidebar
		}
		if sidebarWidth > maxSidebar {
			sidebarWidth = maxSidebar
		}

		// Gap between panels (can be reduced for tight spaces)
		gap := 1
		minContentWidth := 20
		minDetailWidth := 25

		// Calculate widths based on whether detail pane is visible
		var contentWidth, detailWidth int
		if m.detailVisible {
			// Three-column layout: sidebar | content | detail
			remainingWidth := m.width - sidebarWidth - gap*2
			// Split remaining between content (60%) and detail (40%)
			contentWidth = remainingWidth * 60 / 100
			detailWidth = remainingWidth - contentWidth

			// Ensure minimum widths
			if contentWidth < minContentWidth {
				contentWidth = minContentWidth
			}
			if detailWidth < minDetailWidth {
				detailWidth = minDetailWidth
			}

			// If we exceed available space, reduce proportionally
			totalNeeded := sidebarWidth + contentWidth + detailWidth + gap*2
			if totalNeeded > m.width {
				// Reduce sidebar first
				sidebarWidth = m.width - contentWidth - detailWidth - gap*2
				if sidebarWidth < 10 {
					sidebarWidth = 10
					gap = 0
					contentWidth = (m.width - sidebarWidth - minDetailWidth) * 60 / 100
					detailWidth = m.width - sidebarWidth - contentWidth
				}
			}
		} else {
			// Two-column layout: sidebar | content
			contentWidth = m.width - sidebarWidth - gap

			// If content would be too small, shrink sidebar to give content more room
			if contentWidth < minContentWidth {
				sidebarWidth = m.width - minContentWidth - gap
				if sidebarWidth < 10 { // Absolute minimum sidebar
					sidebarWidth = 10
					gap = 0 // Remove gap entirely when very tight
					contentWidth = m.width - sidebarWidth
				} else {
					contentWidth = minContentWidth
				}
			}
		}

		// Ensure nothing goes negative
		if sidebarWidth < 1 {
			sidebarWidth = 1
		}
		if contentWidth < 1 {
			contentWidth = 1
		}

		// Calculate sidebar height that's evenly divisible by 3 (number of sections)
		// This ensures all columns end at the same row
		sidebarHeight := (availableHeight / 3) * 3

		m.sidebar = m.sidebar.SetSize(sidebarWidth, sidebarHeight)
		m.content = m.content.SetSize(contentWidth, sidebarHeight)
		if m.detailVisible {
			m.detailPane = m.detailPane.SetSize(detailWidth, sidebarHeight)
		}
		m.help.Width = m.width
		m.gap = gap // Store gap for View()
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
		// Update detail pane if showing this task
		if m.detailVisible && m.detailPane.Task() != nil && m.detailPane.Task().ID == msg.task.ID {
			m.detailPane = m.detailPane.UpdateTask(msg.task)
		}
		// Reload tasks to show the updated title
		return m, m.loadTasksForSelection

	case taskMovedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		// Update detail pane if showing this task
		if m.detailVisible && m.detailPane.Task() != nil && m.detailPane.Task().ID == msg.task.ID {
			m.detailPane = m.detailPane.UpdateTask(msg.task)
		}
		// Reload tasks to reflect the move
		return m, m.loadTasksForSelection

	case taskDateUpdatedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		// Update detail pane if showing this task
		if m.detailVisible && m.detailPane.Task() != nil && m.detailPane.Task().ID == msg.task.ID {
			m.detailPane = m.detailPane.UpdateTask(msg.task)
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

	case taskToggledMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		// Update the task status in-place (don't reload to keep task visible)
		m.content = m.content.UpdateTaskStatus(msg.taskID, msg.done)
		return m, nil

	case taskTagsUpdatedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		// Update detail pane if showing this task
		if m.detailVisible && m.detailPane.Task() != nil && m.detailPane.Task().ID == msg.task.ID {
			m.detailPane = m.detailPane.UpdateTask(msg.task)
		}
		// Reload tasks and tags (tags cache may have new tags)
		return m, m.loadDataAfterTagUpdate

	case taskDescriptionUpdatedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		// Update detail pane if showing this task
		if m.detailVisible && m.detailPane.Task() != nil && m.detailPane.Task().ID == msg.task.ID {
			m.detailPane = m.detailPane.UpdateTask(msg.task)
		}
		// Reload tasks to reflect the description change
		return m, m.loadTasksForSelection

	case tagsAndTasksUpdatedMsg:
		m.tags = msg.tags
		m.sidebar = m.sidebar.SetData(m.areas, m.projects, msg.tags)
		m.content = m.content.SetTasks(msg.tasks, msg.title, msg.groupBy, msg.hideScope)
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

// taskToggledMsg carries the result of toggling a task's done status
type taskToggledMsg struct {
	taskID int64
	done   bool // true if task was marked done, false if undone
	err    error
}

// taskTagsUpdatedMsg carries the result of updating tags
type taskTagsUpdatedMsg struct {
	task *task.Task
	err  error
}

// taskDescriptionUpdatedMsg carries the result of updating description
type taskDescriptionUpdatedMsg struct {
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

	tasks, err := m.app.ListTasks.Execute(opts)
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

		tasks, err := m.app.ListTasks.Execute(opts)
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
		updated, err := m.app.SetTaskTitle.Execute(taskID, newTitle)
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
			updated, err = m.app.SetTaskProject.Execute(taskID, name)
		case "area":
			updated, err = m.app.SetTaskArea.Execute(taskID, name)
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
			updated, err = m.app.SetPlannedDate.Execute(taskID, date)
		case DateModalDue:
			updated, err = m.app.SetDueDate.Execute(taskID, date)
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

		created, err := m.app.CreateTask.Execute(result.Title, opts)
		return taskCreatedMsg{task: created, err: err}
	}
}

// toggleTask creates a command to toggle a task's done status
func (m Model) toggleTask(taskID int64, currentStatus task.Status) tea.Cmd {
	return func() tea.Msg {
		var err error
		if currentStatus == task.StatusDone {
			// Uncomplete the task
			_, err = m.app.UncompleteTasks.Execute([]int64{taskID})
			return taskToggledMsg{taskID: taskID, done: false, err: err}
		}
		// Complete the task
		_, err = m.app.CompleteTasks.Execute([]int64{taskID})
		return taskToggledMsg{taskID: taskID, done: true, err: err}
	}
}

// setTaskTags creates a command to set a task's tags
func (m Model) setTaskTags(taskID int64, tags []string) tea.Cmd {
	return func() tea.Msg {
		updated, err := m.app.SetTags.Execute(taskID, tags)
		return taskTagsUpdatedMsg{task: updated, err: err}
	}
}

// setTaskDescription creates a command to set a task's description
func (m Model) setTaskDescription(taskID int64, description *string) tea.Cmd {
	return func() tea.Msg {
		updated, err := m.app.SetTaskDescription.Execute(taskID, description)
		return taskDescriptionUpdatedMsg{task: updated, err: err}
	}
}

// openDetailPane opens the detail pane with the selected task
func (m Model) openDetailPane() (tea.Model, tea.Cmd) {
	selectedTask := m.content.SelectedTask()
	if selectedTask == nil {
		return m, nil
	}

	m.detailVisible = true
	m.focusArea = FocusDetail
	m.content = m.content.SetShowSelection(true) // Keep showing selection
	m.content = m.content.SetFocused(false)
	m.detailPane = m.detailPane.SetTask(selectedTask)
	m.detailPane = m.detailPane.SetFocused(true)

	// Recalculate layout for three-column mode
	m = m.recalculateLayout()

	return m, nil
}

// recalculateLayout recalculates component sizes based on current state
func (m Model) recalculateLayout() Model {
	if m.width == 0 || m.height == 0 {
		return m
	}

	// Reserve 1 row for help bar at the bottom
	helpHeight := 1
	availableHeight := m.height - helpHeight

	// Calculate sidebar width: 1/4 of total, constrained between min/max
	sidebarWidth := m.width / 4
	minSidebar := 20
	maxSidebar := 40

	if sidebarWidth < minSidebar {
		sidebarWidth = minSidebar
	}
	if sidebarWidth > maxSidebar {
		sidebarWidth = maxSidebar
	}

	// Gap between panels
	gap := 1
	minContentWidth := 20
	minDetailWidth := 25

	var contentWidth, detailWidth int
	if m.detailVisible {
		// Three-column layout
		remainingWidth := m.width - sidebarWidth - gap*2
		contentWidth = remainingWidth * 60 / 100
		detailWidth = remainingWidth - contentWidth

		if contentWidth < minContentWidth {
			contentWidth = minContentWidth
		}
		if detailWidth < minDetailWidth {
			detailWidth = minDetailWidth
		}

		totalNeeded := sidebarWidth + contentWidth + detailWidth + gap*2
		if totalNeeded > m.width {
			sidebarWidth = m.width - contentWidth - detailWidth - gap*2
			if sidebarWidth < 10 {
				sidebarWidth = 10
				gap = 0
				contentWidth = (m.width - sidebarWidth - minDetailWidth) * 60 / 100
				detailWidth = m.width - sidebarWidth - contentWidth
			}
		}
	} else {
		// Two-column layout
		contentWidth = m.width - sidebarWidth - gap
		if contentWidth < minContentWidth {
			sidebarWidth = m.width - minContentWidth - gap
			if sidebarWidth < 10 {
				sidebarWidth = 10
				gap = 0
				contentWidth = m.width - sidebarWidth
			} else {
				contentWidth = minContentWidth
			}
		}
	}

	if sidebarWidth < 1 {
		sidebarWidth = 1
	}
	if contentWidth < 1 {
		contentWidth = 1
	}

	sidebarHeight := (availableHeight / 3) * 3

	m.sidebar = m.sidebar.SetSize(sidebarWidth, sidebarHeight)
	m.content = m.content.SetSize(contentWidth, sidebarHeight)
	if m.detailVisible && detailWidth > 0 {
		m.detailPane = m.detailPane.SetSize(detailWidth, sidebarHeight)
	}
	m.gap = gap

	return m
}

// openDetailFieldModal opens the appropriate modal for the currently focused field
func (m Model) openDetailFieldModal() (tea.Model, tea.Cmd) {
	selectedTask := m.detailPane.Task()
	if selectedTask == nil {
		return m, nil
	}

	switch m.detailPane.FocusedField() {
	case DetailFieldTitle:
		m.renameModal = m.renameModal.SetSize(m.width, m.height-1)
		m.renameModal = m.renameModal.Open(selectedTask.ID, selectedTask.Title)
	case DetailFieldDescription:
		m.descriptionModal = m.descriptionModal.SetSize(m.width, m.height-1)
		m.descriptionModal = m.descriptionModal.Open(selectedTask.ID, selectedTask.Description)
	case DetailFieldScope:
		m.moveModal = m.moveModal.SetSize(m.width, m.height-1)
		m.moveModal = m.moveModal.Open(selectedTask.ID, m.projects, m.areas)
	case DetailFieldPlanned:
		m.dateModal = m.dateModal.SetSize(m.width, m.height-1)
		m.dateModal = m.dateModal.Open(selectedTask.ID, DateModalPlanned, selectedTask.PlannedDate)
	case DetailFieldDue:
		m.dateModal = m.dateModal.SetSize(m.width, m.height-1)
		m.dateModal = m.dateModal.Open(selectedTask.ID, DateModalDue, selectedTask.DueDate)
	case DetailFieldTags:
		m.tagModal = m.tagModal.SetSize(m.width, m.height-1)
		m.tagModal = m.tagModal.Open(selectedTask.ID, selectedTask.Tags, m.tags)
	}

	return m, nil
}

// loadDataAfterTagUpdate reloads tags and current tasks
func (m Model) loadDataAfterTagUpdate() tea.Msg {
	// Reload tags list (may have new tags)
	tags, err := m.app.ListTags.Execute()
	if err != nil {
		return loadDataMsg{err: err}
	}

	// Build current task list options
	item := m.sidebar.SelectedItem()
	configKey := m.configKeyForSelection()
	sortStr := m.config.GetSort(configKey)
	sortOpts, _ := task.ParseSort(sortStr)
	groupBy := m.config.GetGroup(configKey)
	hideScope := m.config.GetHideScope(configKey)

	opts := m.buildListOptions(item)
	opts.Sort = sortOpts

	tasks, err := m.app.ListTasks.Execute(opts)
	if err != nil {
		return loadDataMsg{err: err}
	}

	// Return combined update
	return tagsAndTasksUpdatedMsg{
		tags:      tags,
		tasks:     tasks,
		title:     strings.TrimSpace(item.Label),
		groupBy:   groupBy,
		hideScope: hideScope,
	}
}

// tagsAndTasksUpdatedMsg carries updated tags and tasks
type tagsAndTasksUpdatedMsg struct {
	tags      []string
	tasks     []task.Task
	title     string
	groupBy   string
	hideScope bool
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
	case m.tagModal.Active():
		helpView = m.help.View(tagKeys)
	case m.descriptionModal.Active():
		helpView = m.help.View(descriptionKeys)
	case m.focusArea == FocusSidebar:
		helpView = m.help.View(sidebarKeys)
	case m.focusArea == FocusDetail:
		helpView = m.help.View(detailKeys)
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
	if m.tagModal.Active() {
		return lipgloss.JoinVertical(lipgloss.Left, m.tagModal.View(), helpView)
	}
	if m.descriptionModal.Active() {
		return lipgloss.JoinVertical(lipgloss.Left, m.descriptionModal.View(), helpView)
	}

	// Render sidebar and content side by side (gap can be 0 for tight layouts)
	contentView := lipgloss.NewStyle().MarginLeft(m.gap).Render(m.content.View())
	var mainView string
	if m.detailVisible {
		// Three-column layout: sidebar | content | detail
		detailView := lipgloss.NewStyle().MarginLeft(m.gap).Render(m.detailPane.View())
		mainView = lipgloss.JoinHorizontal(lipgloss.Top, m.sidebar.View(), contentView, detailView)
	} else {
		// Two-column layout: sidebar | content
		mainView = lipgloss.JoinHorizontal(lipgloss.Top, m.sidebar.View(), contentView)
	}

	// Combine main view with help bar at the bottom
	return lipgloss.JoinVertical(lipgloss.Left, mainView, helpView)
}
