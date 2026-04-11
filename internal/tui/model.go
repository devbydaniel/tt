package tui

import (
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/devbydaniel/tt/config"
	"github.com/devbydaniel/tt/internal/app"
	"github.com/devbydaniel/tt/internal/domain/area"
	"github.com/devbydaniel/tt/internal/domain/note"
	noteusecases "github.com/devbydaniel/tt/internal/domain/note/usecases"
	"github.com/devbydaniel/tt/internal/domain/task"
	taskusecases "github.com/devbydaniel/tt/internal/domain/task/usecases"
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
	sidebar            Sidebar
	content            Content
	detailPane         DetailPane
	renameModal        RenameModal
	moveModal          MoveModal
	dateModal          DateModal
	addModal           AddModal
	tagModal           TagModal
	descriptionModal   DescriptionModal
	confirmModal       ConfirmModal
	completeModal      CompleteModal
	createProjectModal CreateProjectModal
	createAreaModal    CreateAreaModal
	createNoteModal    CreateNoteModal
	helpModal          HelpModal

	help               help.Model
	focusArea          FocusArea
	detailVisible      bool // whether the detail pane is shown
	notePreviewPane    NotePreviewPane
	notePreviewVisible bool // whether the note preview pane is shown
	followMode         bool // follow mode: detail auto-updates with selection

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

	// Detect light theme for glamour markdown rendering
	isLightTheme := cfg.Theme.Name == "solarized-light" || cfg.Theme.Name == "catppuccin-latte"

	return Model{
		app:                application,
		config:             cfg,
		styles:             styles,
		gap:                1, // Default gap, adjusted on resize
		sidebar:            NewSidebar(styles),
		content:            NewContent(styles),
		detailPane:         NewDetailPane(styles),
		notePreviewPane:    NewNotePreviewPane(styles, isLightTheme),
		renameModal:        NewRenameModal(styles),
		moveModal:          NewMoveModal(styles),
		dateModal:          NewDateModal(styles),
		addModal:           NewAddModal(styles),
		tagModal:           NewTagModal(styles),
		descriptionModal:   NewDescriptionModal(styles),
		confirmModal:       NewConfirmModal(styles),
		completeModal:      NewCompleteModal(styles),
		createProjectModal: NewCreateProjectModal(styles),
		createAreaModal:    NewCreateAreaModal(styles),
		createNoteModal:    NewCreateNoteModal(styles),
		helpModal:          NewHelpModal(styles),

		help: helpModel,
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

// getSelectedProject returns the project selected in the sidebar, or nil
func (m Model) getSelectedProject() *task.Task {
	item := m.sidebar.SelectedItem()
	if item.Type != "project" {
		return nil
	}
	for i := range m.projects {
		if m.projects[i].Title == item.Key {
			return &m.projects[i]
		}
	}
	return nil
}

// isProjectID returns true if the given ID belongs to a project
func (m Model) isProjectID(id int64) bool {
	for _, p := range m.projects {
		if p.ID == id {
			return true
		}
	}
	return false
}

// updateProjectCache updates a project in the cached m.projects slice
func (m *Model) updateProjectCache(updated *task.Task) {
	for i := range m.projects {
		if m.projects[i].ID == updated.ID {
			m.projects[i] = *updated
			return
		}
	}
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
	if err := m.app.EnrichIndicators.Execute(tasks); err != nil {
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
				// Rename was confirmed
				if result.ItemType == "area" {
					return m, m.renameArea(result.ItemKey, result.NewTitle)
				}
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

		// Route keys to confirm modal when active
		if m.confirmModal.Active() {
			var result *ConfirmResult
			m.confirmModal, result = m.confirmModal.Update(msg)
			if result != nil && result.Confirmed {
				return m, m.deleteItem(result)
			}
			return m, nil
		}

		// Route keys to complete modal when active
		if m.completeModal.Active() {
			var result *CompleteResult
			m.completeModal, result = m.completeModal.Update(msg)
			if result != nil && result.Confirmed {
				return m, m.completeProject(result)
			}
			return m, nil
		}

		// Route keys to create project modal when active
		if m.createProjectModal.Active() {
			var result *CreateProjectResult
			m.createProjectModal, result = m.createProjectModal.Update(msg)
			if result != nil && !result.Canceled {
				return m, m.createProject(result)
			}
			return m, nil
		}

		// Route keys to create area modal when active
		if m.createAreaModal.Active() {
			var result *CreateAreaResult
			m.createAreaModal, result = m.createAreaModal.Update(msg)
			if result != nil && !result.Canceled {
				return m, m.createArea(result)
			}
			return m, nil
		}

		// Route keys to create note modal when active
		if m.createNoteModal.Active() {
			var result *CreateNoteResult
			m.createNoteModal, result = m.createNoteModal.Update(msg)
			if result != nil && !result.Canceled {
				return m, m.createAndOpenNote(result)
			}
			return m, nil
		}

		// Route keys to help modal when active
		if m.helpModal.Active() {
			var closed bool
			m.helpModal, closed = m.helpModal.Update(msg)
			if closed {
				return m, nil
			}
			return m, nil
		}

		switch {
		case key.Matches(msg, keys.Quit):
			return m, tea.Quit

		case key.Matches(msg, keys.Help):
			m.helpModal = m.helpModal.SetSize(m.width, m.height-1)
			m.helpModal = m.helpModal.Open(m.currentHelpBindings())
			return m, nil

		case key.Matches(msg, keys.Enter):
			if m.focusArea == FocusSidebar {
				m.focusArea = FocusContent
				m.sidebar = m.sidebar.SetFocused(false)
				m.content = m.content.SetFocused(true)
				return m, nil
			}
			if m.focusArea == FocusContent {
				if m.content.ViewMode() == ContentViewNotes {
					if n := m.content.SelectedNote(); n != nil {
						return m, m.openScopeNoteInEditor(n.Path)
					}
					return m, nil
				}
				// Enter from content opens detail pane
				return m.openDetailPane()
			}
			if m.focusArea == FocusDetail && m.notePreviewVisible {
				// Enter in note preview opens note in editor
				if n := m.notePreviewPane.Note(); n != nil {
					return m, m.openScopeNoteInEditor(n.Path)
				}
				return m, nil
			}
			if m.focusArea == FocusDetail {
				if m.detailPane.ViewMode() == DetailViewNotes {
					if n := m.detailPane.SelectedNote(); n != nil {
						t := m.detailPane.Task()
						return m, m.openNoteInEditor(n.Path, t.ID, t.UUID)
					}
					return m, nil
				}
				// Open modal for focused field
				return m.openDetailFieldModal()
			}

		case key.Matches(msg, keys.Escape):
			if m.focusArea == FocusDetail && m.notePreviewVisible {
				// Close note preview pane, return to content
				m.followMode = false
				m.focusArea = FocusContent
				m.notePreviewPane = m.notePreviewPane.SetFocused(false)
				m.notePreviewVisible = false
				m.content = m.content.SetFocused(true)
				m = m.recalculateLayout()
				return m, nil
			}
			if m.focusArea == FocusDetail {
				// Close detail pane, return to content
				m.followMode = false
				m.focusArea = FocusContent
				m.detailPane = m.detailPane.SetFocused(false)
				m.detailVisible = false
				m.content = m.content.SetShowSelection(false)
				m.content = m.content.SetFocused(true)
				m = m.recalculateLayout()
				return m, nil
			}
			if m.focusArea == FocusContent {
				// If follow mode or note preview is visible, close it first
				if m.followMode || m.notePreviewVisible {
					m.followMode = false
					m.notePreviewVisible = false
					m.detailVisible = false
					m = m.recalculateLayout()
					return m, nil
				}
				m.focusArea = FocusSidebar
				m.sidebar = m.sidebar.SetFocused(true)
				m.content = m.content.SetShowSelection(false)
				m.content = m.content.SetFocused(false)
				m.detailVisible = false
				m = m.recalculateLayout()
				return m, nil
			}

		case key.Matches(msg, keys.FocusSidebar):
			if m.focusArea == FocusDetail && m.notePreviewVisible {
				// Close note preview and return to content
				m.focusArea = FocusContent
				m.notePreviewPane = m.notePreviewPane.SetFocused(false)
				m.notePreviewVisible = false
				m.content = m.content.SetFocused(true)
				m = m.recalculateLayout()
				return m, nil
			}
			if m.focusArea == FocusDetail {
				// Always close detail pane and return to content
				m.focusArea = FocusContent
				m.detailPane = m.detailPane.SetFocused(false)
				m.detailVisible = false
				m.content = m.content.SetShowSelection(false)
				m.content = m.content.SetFocused(true)
				m = m.recalculateLayout()
				return m, nil
			}
			if m.focusArea == FocusContent {
				m.followMode = false
				m.focusArea = FocusSidebar
				m.sidebar = m.sidebar.SetFocused(true)
				m.content = m.content.SetShowSelection(false)
				m.content = m.content.SetFocused(false)
				m.detailVisible = false
				m.notePreviewVisible = false
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
				if m.followMode {
					// Transition from follow mode to full detail mode
					m.followMode = false
					m.focusArea = FocusDetail
					m.content = m.content.SetFocused(false)
					m.content = m.content.SetShowSelection(true)
					if m.notePreviewVisible {
						m.notePreviewPane = m.notePreviewPane.SetFocused(true)
					} else {
						m.detailPane = m.detailPane.SetFocused(true)
					}
					return m, nil
				}
				// l from content opens detail pane (tasks view) or note preview (notes view)
				if m.content.ViewMode() == ContentViewTasks {
					return m.openDetailPane()
				}
				if m.content.ViewMode() == ContentViewNotes {
					return m.openNotePreview()
				}
				return m, nil
			}

		case key.Matches(msg, keys.Follow):
			if m.focusArea == FocusContent {
				m.followMode = !m.followMode
				if m.followMode {
					return m.activateFollowMode()
				}
				return m.deactivateFollowMode()
			}

		case key.Matches(msg, keys.Rename):
			if m.focusArea == FocusContent {
				if selectedTask := m.content.SelectedTask(); selectedTask != nil {
					m.renameModal = m.renameModal.SetSize(m.width, m.height-1) // -1 for help bar
					m.renameModal = m.renameModal.Open(selectedTask.ID, selectedTask.Title)
					return m, nil
				}
			}
			if m.focusArea == FocusSidebar {
				item := m.sidebar.SelectedItem()
				if item.Type == "project" {
					if proj := m.getSelectedProject(); proj != nil {
						m.renameModal = m.renameModal.SetSize(m.width, m.height-1)
						m.renameModal = m.renameModal.Open(proj.ID, proj.Title)
						return m, nil
					}
				}
				if item.Type == "area" {
					m.renameModal = m.renameModal.SetSize(m.width, m.height-1)
					m.renameModal = m.renameModal.OpenForArea(item.Key)
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
			if m.focusArea == FocusSidebar {
				if proj := m.getSelectedProject(); proj != nil {
					m.moveModal = m.moveModal.SetSize(m.width, m.height-1)
					m.moveModal = m.moveModal.OpenForProject(proj.ID, m.areas)
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
			if m.focusArea == FocusSidebar {
				if proj := m.getSelectedProject(); proj != nil {
					m.dateModal = m.dateModal.SetSize(m.width, m.height-1)
					m.dateModal = m.dateModal.Open(proj.ID, DateModalPlanned, proj.PlannedDate)
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
			if m.focusArea == FocusSidebar {
				if proj := m.getSelectedProject(); proj != nil {
					m.dateModal = m.dateModal.SetSize(m.width, m.height-1)
					m.dateModal = m.dateModal.Open(proj.ID, DateModalDue, proj.DueDate)
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
			if m.focusArea == FocusSidebar {
				if proj := m.getSelectedProject(); proj != nil {
					m.tagModal = m.tagModal.SetSize(m.width, m.height-1)
					m.tagModal = m.tagModal.Open(proj.ID, proj.Tags, m.tags)
					return m, nil
				}
			}

		case key.Matches(msg, keys.Add):
			// In content notes view, open create note modal for scope
			if m.focusArea == FocusContent && m.content.ViewMode() == ContentViewNotes {
				entityType, entityUUID := m.resolveScopeEntity()
				if entityUUID != "" {
					m.createNoteModal = m.createNoteModal.SetSize(m.width, m.height-1)
					m.createNoteModal = m.createNoteModal.Open(entityType, entityUUID, 0)
					return m, nil
				}
			}
			// In detail notes view, open create note modal for task
			if m.focusArea == FocusDetail && m.detailPane.ViewMode() == DetailViewNotes {
				if t := m.detailPane.Task(); t != nil {
					entityType := note.EntityTask
					if t.TaskType == task.TaskTypeProject {
						entityType = note.EntityProject
					}
					m.createNoteModal = m.createNoteModal.SetSize(m.width, m.height-1)
					m.createNoteModal = m.createNoteModal.Open(entityType, t.UUID, t.ID)
					return m, nil
				}
			}
			// If sidebar is focused and scopes section is active, open create project modal
			if m.focusArea == FocusSidebar && m.sidebar.IsScopesSectionActive() {
				m.createProjectModal = m.createProjectModal.SetSize(m.width, m.height-1)
				m.createProjectModal = m.createProjectModal.Open(m.areas)
				return m, nil
			}
			// Otherwise, open add task modal (pre-fill scope if viewing a project or area)
			sidebarItem := m.sidebar.SelectedItem()
			var prefill *SidebarItem
			if sidebarItem.Type == "project" || sidebarItem.Type == "area" {
				prefill = &sidebarItem
			}
			m.addModal = m.addModal.SetSize(m.width, m.height-1)
			m.addModal = m.addModal.Open(m.projects, m.areas, prefill)
			return m, nil

		case key.Matches(msg, keys.AddArea):
			// Only works when sidebar is focused and scopes section is active
			if m.focusArea == FocusSidebar && m.sidebar.IsScopesSectionActive() {
				m.createAreaModal = m.createAreaModal.SetSize(m.width, m.height-1)
				m.createAreaModal = m.createAreaModal.Open()
				return m, nil
			}

		case key.Matches(msg, keys.Toggle):
			if m.focusArea == FocusContent {
				if selectedTask := m.content.SelectedTask(); selectedTask != nil {
					return m, m.toggleTask(selectedTask.ID, selectedTask.Status)
				}
			}
			if m.focusArea == FocusSidebar {
				if proj := m.getSelectedProject(); proj != nil {
					m.completeModal = m.completeModal.SetSize(m.width, m.height-1)
					m.completeModal = m.completeModal.Open(proj.ID, proj.Title)
					return m, nil
				}
			}

		case key.Matches(msg, keys.Someday):
			// In notes views, 's' launches fzf note search
			if m.focusArea == FocusContent && m.content.ViewMode() == ContentViewNotes {
				entityType, entityUUID := m.resolveScopeEntity()
				if entityUUID != "" {
					notesDir := m.app.SearchNotes.Repo.EntityDir(entityType, entityUUID)
					return m, launchNoteSearchScope(notesDir)
				}
			}
			if m.focusArea == FocusDetail && m.detailPane.ViewMode() == DetailViewNotes {
				if t := m.detailPane.Task(); t != nil {
					notesDir := m.app.SearchNotes.Repo.EntityDir(note.EntityTask, t.UUID)
					return m, launchNoteSearchTask(notesDir, t.ID, t.UUID)
				}
			}
			if m.focusArea == FocusContent {
				if selectedTask := m.content.SelectedTask(); selectedTask != nil {
					return m, m.toggleTaskState(selectedTask.ID, selectedTask.State)
				}
			}
			if m.focusArea == FocusSidebar {
				if proj := m.getSelectedProject(); proj != nil {
					return m, m.toggleTaskState(proj.ID, proj.State)
				}
			}

		case key.Matches(msg, keys.AISync):
			var targetTask *task.Task
			switch m.focusArea {
			case FocusContent:
				targetTask = m.content.SelectedTask()
			case FocusDetail:
				targetTask = m.detailPane.Task()
			case FocusSidebar:
				binary, err := findAIBinary(&m.config.AI)
				if err != nil {
					m.err = err
					return m, nil
				}
				if proj := m.getSelectedProject(); proj != nil {
					return m, launchAISyncForProject(proj, binary, m.config.AI.Workspace)
				}
				if item := m.sidebar.SelectedItem(); item.Type == "area" {
					for i := range m.areas {
						if m.areas[i].Name == item.Key {
							return m, launchAISyncForArea(&m.areas[i], binary, m.config.AI.Workspace)
						}
					}
				}
			}
			if targetTask != nil {
				binary, err := findAIBinary(&m.config.AI)
				if err != nil {
					m.err = err
					return m, nil
				}
				return m, launchAISync(targetTask, binary, m.config.AI.Workspace)
			}

		case key.Matches(msg, keys.Delete):
			// In content notes view, delete the selected note
			if m.focusArea == FocusContent && m.content.ViewMode() == ContentViewNotes {
				if n := m.content.SelectedNote(); n != nil {
					m.confirmModal = m.confirmModal.SetSize(m.width, m.height-1)
					m.confirmModal = m.confirmModal.OpenForNote(n.Title, n.Path)
					return m, nil
				}
			}
			// In detail notes view, delete the selected note
			if m.focusArea == FocusDetail && m.detailPane.ViewMode() == DetailViewNotes {
				if n := m.detailPane.SelectedNote(); n != nil {
					m.confirmModal = m.confirmModal.SetSize(m.width, m.height-1)
					m.confirmModal = m.confirmModal.OpenForNote(n.Title, n.Path)
					return m, nil
				}
			}
			if m.focusArea == FocusContent {
				if selectedTask := m.content.SelectedTask(); selectedTask != nil {
					m.confirmModal = m.confirmModal.SetSize(m.width, m.height-1)
					m.confirmModal = m.confirmModal.OpenForTask(selectedTask.ID, selectedTask.Title)
					return m, nil
				}
			}
			if m.focusArea == FocusSidebar {
				item := m.sidebar.SelectedItem()
				if item.Type == "project" {
					if proj := m.getSelectedProject(); proj != nil {
						m.confirmModal = m.confirmModal.SetSize(m.width, m.height-1)
						m.confirmModal = m.confirmModal.OpenForProject(proj.ID, proj.Title)
						return m, nil
					}
				}
				if item.Type == "area" {
					m.confirmModal = m.confirmModal.SetSize(m.width, m.height-1)
					m.confirmModal = m.confirmModal.OpenForArea(item.Key)
					return m, nil
				}
			}

		case key.Matches(msg, keys.Tab):
			if m.focusArea == FocusDetail {
				m.detailPane = m.detailPane.NextViewMode()
				return m, nil
			}
			if m.focusArea == FocusContent {
				if !m.hasScopeSelected() {
					return m, nil
				}
				// Close current panes when switching views
				m.notePreviewVisible = false
				m.detailVisible = false
				m.content = m.content.ToggleViewMode()
				if m.followMode {
					// Re-activate follow mode for the new view
					m = m.recalculateLayout()
					if m.content.ViewMode() == ContentViewNotes {
						// Need to load notes first, then activate follow in scopeNotesLoadedMsg
						return m, m.loadScopeNotes()
					}
					return m.activateFollowMode()
				}
				m = m.recalculateLayout()
				if m.content.ViewMode() == ContentViewNotes {
					return m, m.loadScopeNotes()
				}
				return m, nil
			}
			m.sidebar = m.sidebar.NextSection()
			return m, m.loadTasksForSelection

		case key.Matches(msg, keys.ShiftTab):
			if m.focusArea == FocusDetail {
				m.detailPane = m.detailPane.PrevViewMode()
				return m, nil
			}
			if m.focusArea == FocusContent {
				if !m.hasScopeSelected() {
					return m, nil
				}
				// Close current panes when switching views
				m.notePreviewVisible = false
				m.detailVisible = false
				m.content = m.content.ToggleViewMode()
				if m.followMode {
					// Re-activate follow mode for the new view
					m = m.recalculateLayout()
					if m.content.ViewMode() == ContentViewNotes {
						return m, m.loadScopeNotes()
					}
					return m.activateFollowMode()
				}
				m = m.recalculateLayout()
				if m.content.ViewMode() == ContentViewNotes {
					return m, m.loadScopeNotes()
				}
				return m, nil
			}
			m.sidebar = m.sidebar.PrevSection()
			return m, m.loadTasksForSelection

		case key.Matches(msg, keys.Up):
			if m.focusArea == FocusDetail && m.notePreviewVisible {
				m.notePreviewPane = m.notePreviewPane.ScrollUp()
				return m, nil
			}
			if m.focusArea == FocusDetail {
				if m.detailPane.ViewMode() == DetailViewNotes {
					m.detailPane = m.detailPane.PrevNote()
				} else {
					m.detailPane = m.detailPane.PrevField()
				}
				return m, nil
			}
			if m.focusArea == FocusContent {
				if m.content.ViewMode() == ContentViewNotes {
					m.content = m.content.NoteUp()
					if m.notePreviewVisible {
						return m, m.updateNotePreview()
					}
				} else {
					m.content = m.content.MoveUp()
					if m.followMode && m.detailVisible {
						if sel := m.content.SelectedTask(); sel != nil {
							m.detailPane = m.detailPane.SetTask(sel)
							return m, m.loadNotes(sel.UUID, sel.ID)
						}
					}
				}
				return m, nil
			}
			m.sidebar = m.sidebar.MoveUp()
			return m, m.loadTasksForSelection

		case key.Matches(msg, keys.Down):
			if m.focusArea == FocusDetail && m.notePreviewVisible {
				m.notePreviewPane = m.notePreviewPane.ScrollDown()
				return m, nil
			}
			if m.focusArea == FocusDetail {
				if m.detailPane.ViewMode() == DetailViewNotes {
					m.detailPane = m.detailPane.NextNote()
				} else {
					m.detailPane = m.detailPane.NextField()
				}
				return m, nil
			}
			if m.focusArea == FocusContent {
				if m.content.ViewMode() == ContentViewNotes {
					m.content = m.content.NoteDown()
					if m.notePreviewVisible {
						return m, m.updateNotePreview()
					}
				} else {
					m.content = m.content.MoveDown()
					if m.followMode && m.detailVisible {
						if sel := m.content.SelectedTask(); sel != nil {
							m.detailPane = m.detailPane.SetTask(sel)
							return m, m.loadNotes(sel.UUID, sel.ID)
						}
					}
				}
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

		sidebarHeight := availableHeight

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
		// Initial load - reset to top (0, 0)
		m.content = m.content.SetTasks(msg.tasks, "Today", groupBy, hideScope, 0, 0)
		return m, nil

	case tasksLoadedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		// ResetViewMode forces back to tasks view; close any note preview pane
		m.notePreviewVisible = false
		m.content = m.content.ResetViewMode()
		m.content = m.content.SetShowTabs(m.hasScopeSelected())
		m.content = m.content.SetTasks(msg.tasks, msg.title, msg.groupBy, msg.hideScope, msg.preserveTaskID, msg.preserveIndex)
		if msg.scopeNotes != nil {
			m.content = m.content.SetScopeNotes(msg.scopeNotes)
		}
		if m.followMode && m.focusArea == FocusContent {
			if sel := m.content.SelectedTask(); sel != nil {
				m.detailVisible = true
				m.detailPane = m.detailPane.SetTask(sel)
				m.detailPane = m.detailPane.SetFocused(false)
				m = m.recalculateLayout()
				return m, m.loadNotes(sel.UUID, sel.ID)
			}
			// No tasks available — deactivate follow mode
			m.followMode = false
			m.detailVisible = false
			m = m.recalculateLayout()
		}
		return m, nil

	case scheduleTasksLoadedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		// ResetViewMode forces back to tasks view; close any note preview pane
		m.notePreviewVisible = false
		m.content = m.content.ResetViewMode()
		m.content = m.content.SetShowTabs(m.hasScopeSelected())
		m.content = m.content.SetScheduleGroups(msg.groups, msg.title, msg.hideScope, msg.preserveTaskID, msg.preserveIndex)
		if msg.scopeNotes != nil {
			m.content = m.content.SetScopeNotes(msg.scopeNotes)
		}
		if m.followMode && m.focusArea == FocusContent {
			if sel := m.content.SelectedTask(); sel != nil {
				m.detailVisible = true
				m.detailPane = m.detailPane.SetTask(sel)
				m.detailPane = m.detailPane.SetFocused(false)
				m = m.recalculateLayout()
				return m, m.loadNotes(sel.UUID, sel.ID)
			}
			// No tasks available — deactivate follow mode
			m.followMode = false
			m.detailVisible = false
			m = m.recalculateLayout()
		}
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
		// If a project was renamed, reload sidebar too
		if m.isProjectID(msg.task.ID) {
			return m, m.loadData
		}
		// Reload tasks to show the updated title, preserving selection
		return m, m.loadTasksPreserveSelection

	case areaRenamedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		// Reload sidebar to show the renamed area
		return m, m.loadData

	case taskMovedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		// Update detail pane if showing this task
		if m.detailVisible && m.detailPane.Task() != nil && m.detailPane.Task().ID == msg.task.ID {
			m.detailPane = m.detailPane.UpdateTask(msg.task)
		}
		// If a project was moved, reload sidebar too
		if m.isProjectID(msg.task.ID) {
			return m, m.loadData
		}
		// Reload tasks to reflect the move, preserving selection
		return m, m.loadTasksPreserveSelection

	case taskDateUpdatedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		// Update detail pane if showing this task
		if m.detailVisible && m.detailPane.Task() != nil && m.detailPane.Task().ID == msg.task.ID {
			m.detailPane = m.detailPane.UpdateTask(msg.task)
		}
		// Update project cache if this is a project (so header title updates)
		if m.isProjectID(msg.task.ID) {
			m.updateProjectCache(msg.task)
		}
		// Reload tasks to reflect the date change, preserving selection
		return m, m.loadTasksPreserveSelection

	case taskCreatedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		// Reload tasks to show the new task
		return m, m.loadTasksForSelection

	case projectCreatedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		// Reload sidebar to show the new project
		return m, m.loadData

	case areaCreatedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		// Reload sidebar to show the new area
		return m, m.loadData

	case taskToggledMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		// Update the task status in-place (don't reload to keep task visible)
		m.content = m.content.UpdateTaskStatus(msg.taskID, msg.done)
		return m, nil

	case projectCompletedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		// Reload everything since project and its tasks are now completed
		return m, m.loadData

	case taskStateUpdatedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		// Update detail pane if showing this task
		if m.detailVisible && m.detailPane.Task() != nil && m.detailPane.Task().ID == msg.task.ID {
			m.detailPane = m.detailPane.UpdateTask(msg.task)
		}
		// If it's a project, reload everything to update sidebar (project may appear/disappear)
		if m.isProjectID(msg.task.ID) {
			return m, m.loadData
		}
		// Reload tasks to reflect the state change, preserving selection
		return m, m.loadTasksPreserveSelection

	case taskTagsUpdatedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		// Update detail pane if showing this task
		if m.detailVisible && m.detailPane.Task() != nil && m.detailPane.Task().ID == msg.task.ID {
			m.detailPane = m.detailPane.UpdateTask(msg.task)
		}
		// Update project cache if this is a project (so header title updates)
		if m.isProjectID(msg.task.ID) {
			m.updateProjectCache(msg.task)
		}
		// Reload tasks and tags (tags cache may have new tags), preserving selection
		return m, m.loadDataAfterTagUpdatePreserveSelection

	case taskDescriptionUpdatedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		// Update detail pane if showing this task
		if m.detailVisible && m.detailPane.Task() != nil && m.detailPane.Task().ID == msg.task.ID {
			m.detailPane = m.detailPane.UpdateTask(msg.task)
		}
		// Reload tasks to reflect the description change, preserving selection
		return m, m.loadTasksPreserveSelection

	case itemDeletedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		// Close detail pane if it was showing the deleted task
		if m.detailVisible && m.detailPane.Task() != nil && m.detailPane.Task().ID == msg.targetID {
			m.detailVisible = false
			m.focusArea = FocusContent
			m.detailPane = m.detailPane.SetFocused(false)
			m.content = m.content.SetFocused(true)
			m = m.recalculateLayout()
		}
		// Reload data based on what was deleted
		if msg.target == DeleteTargetNote {
			// Reload notes for the current view
			if m.focusArea == FocusDetail || (m.detailVisible && m.detailPane.Task() != nil) {
				t := m.detailPane.Task()
				return m, m.loadNotes(t.UUID, t.ID)
			}
			return m, m.loadScopeNotes()
		}
		if msg.target == DeleteTargetArea || msg.target == DeleteTargetProject {
			return m, m.loadData
		}
		// Preserve index (not task ID) so cursor moves to next task
		return m, m.loadTasksPreserveIndex

	case tagsAndTasksUpdatedMsg:
		m.tags = msg.tags
		m.sidebar = m.sidebar.SetData(m.areas, m.projects, msg.tags)
		m.content = m.content.SetTasks(msg.tasks, msg.title, msg.groupBy, msg.hideScope, msg.preserveTaskID, msg.preserveIndex)
		return m, nil

	case aiSyncFinishedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		return m, m.loadData

	case notesLoadedMsg:
		if msg.err != nil {
			return m, nil // silently ignore note load errors
		}
		if m.detailVisible && m.detailPane.Task() != nil && m.detailPane.Task().ID == msg.taskID {
			m.detailPane = m.detailPane.SetNotes(msg.notes)
		}
		return m, nil

	case noteEditorFinishedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		// Reload notes after editor closes (user may have edited/saved)
		return m, m.loadNotes(msg.taskUUID, msg.taskID)

	case scopeNotesLoadedMsg:
		if msg.err != nil {
			return m, nil
		}
		m.content = m.content.SetScopeNotes(msg.notes)
		if m.followMode && !m.notePreviewVisible && m.focusArea == FocusContent {
			// Activate follow mode now that notes are loaded (e.g. after Tab switch)
			return m.activateFollowMode()
		}
		if m.notePreviewVisible {
			return m, m.updateNotePreview()
		}
		return m, nil

	case noteContentMsg:
		if msg.err != nil {
			return m, nil
		}
		if n := m.content.SelectedNote(); n != nil && n.Path == msg.path {
			m.notePreviewPane = m.notePreviewPane.SetNote(n, msg.content)
		}
		return m, nil

	case scopeNoteEditorFinishedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		// After editor closes, return focus to content notes list
		if m.notePreviewVisible {
			m.focusArea = FocusContent
			m.notePreviewPane = m.notePreviewPane.SetFocused(false)
			m.content = m.content.SetFocused(true)
		}
		return m, m.loadScopeNotes()

	case noteCreatedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		if msg.taskID != 0 {
			return m, m.openNoteInEditor(msg.note.Path, msg.taskID, msg.entityUUID)
		}
		return m, m.openScopeNoteInEditor(msg.note.Path)
	}

	return m, nil
}

// tasksLoadedMsg carries loaded tasks for a selection
type tasksLoadedMsg struct {
	tasks          []task.Task
	title          string
	groupBy        string
	hideScope      bool
	preserveTaskID int64       // task ID to try to restore selection to
	preserveIndex  int         // fallback index if task not found
	scopeNotes     []note.Note // notes for the selected scope (if any)
	err            error
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
	groups         ScheduleGroups
	title          string
	hideScope      bool
	preserveTaskID int64       // task ID to try to restore selection to
	preserveIndex  int         // fallback index if task not found
	scopeNotes     []note.Note // notes for the selected scope (if any)
	err            error
}

// taskRenamedMsg carries the result of a task rename
type taskRenamedMsg struct {
	task *task.Task
	err  error
}

// areaRenamedMsg carries the result of an area rename
type areaRenamedMsg struct {
	area *area.Area
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

// projectCompletedMsg carries the result of completing a project
type projectCompletedMsg struct {
	projectID int64
	err       error
}

// taskStateUpdatedMsg carries the result of toggling a task's someday/active state
type taskStateUpdatedMsg struct {
	task *task.Task
	err  error
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

// itemDeletedMsg carries the result of a delete operation
type itemDeletedMsg struct {
	target     DeleteTarget
	targetID   int64
	targetName string
	err        error
}

// projectCreatedMsg carries the result of creating a project
type projectCreatedMsg struct {
	project *task.Task
	err     error
}

// areaCreatedMsg carries the result of creating an area
type areaCreatedMsg struct {
	area *area.Area
	err  error
}

// formatProjectTitle builds a project title with metadata for the content header.
// Format: "ProjectName  📅 Jan 2  🏁 Jan 15  #tag1 #tag2"
func (m Model) formatProjectTitle(proj *task.Task) string {
	theme := m.styles.Theme
	parts := []string{strings.TrimSpace(proj.Title)}

	if proj.PlannedDate != nil {
		parts = append(parts, theme.Muted.Render(theme.Icons.Date+" "+proj.PlannedDate.Format("Jan 2")))
	}
	if proj.DueDate != nil {
		parts = append(parts, theme.Muted.Render(theme.Icons.Due+" "+proj.DueDate.Format("Jan 2")))
	}
	if len(proj.Tags) > 0 {
		var tagParts []string
		for _, tag := range proj.Tags {
			tagParts = append(tagParts, "#"+tag)
		}
		parts = append(parts, theme.Muted.Render(strings.Join(tagParts, " ")))
	}

	return strings.Join(parts, "  ")
}

// loadTasksForSelection loads tasks based on sidebar selection
func (m Model) loadTasksForSelection() tea.Msg {
	item := m.sidebar.SelectedItem()
	title := strings.TrimSpace(item.Label)

	// For projects, include metadata in the title
	if item.Type == "project" {
		for i := range m.projects {
			if m.projects[i].Title == item.Key {
				title = m.formatProjectTitle(&m.projects[i])
				break
			}
		}
	}

	// Get sort, group, and hideScope settings from config
	configKey := m.configKeyForSelection()
	groupBy := m.config.GetGroup(configKey)
	sortStr := m.config.GetSort(configKey)
	hideScope := m.config.GetHideScope(configKey)
	// Auto-hide scope when already viewing a specific area or project
	if configKey == "area" || configKey == "project" {
		hideScope = true
	}
	sortOpts, _ := task.ParseSort(sortStr)

	// Load scope notes for project/area views
	var scopeNotes []note.Note
	if item.Type == "project" || item.Type == "area" {
		entityType, entityUUID := m.resolveScopeEntity()
		if entityUUID != "" {
			scopeNotes, _ = m.app.ListNotes.Execute(noteusecases.ListOptions{
				EntityType: entityType,
				EntityUUID: entityUUID,
			})
		}
	}

	// Schedule grouping requires 4 separate queries
	if groupBy == "schedule" {
		msg := m.loadScheduleGroups(item, title, sortOpts, hideScope)
		if typedMsg, ok := msg.(scheduleTasksLoadedMsg); ok {
			typedMsg.scopeNotes = scopeNotes
			return typedMsg
		}
		return msg
	}

	// Build list options based on selection
	opts := m.buildListOptions(item)
	opts.Sort = sortOpts

	tasks, err := m.app.ListTasks.Execute(opts)
	if err != nil {
		return tasksLoadedMsg{err: err}
	}
	if err := m.app.EnrichIndicators.Execute(tasks); err != nil {
		return tasksLoadedMsg{err: err}
	}

	return tasksLoadedMsg{tasks: tasks, title: title, groupBy: groupBy, hideScope: hideScope, scopeNotes: scopeNotes}
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
		if err := m.app.EnrichIndicators.Execute(tasks); err != nil {
			return scheduleTasksLoadedMsg{err: err}
		}
		*sched.target = tasks
	}

	return scheduleTasksLoadedMsg{groups: groups, title: title, hideScope: hideScope}
}

// loadTasksPreserveSelection loads tasks while preserving current selection
func (m Model) loadTasksPreserveSelection() tea.Msg {
	preserveTaskID := m.content.SelectedTaskID()
	preserveIndex := m.content.SelectedIndex()

	msg := m.loadTasksForSelection()

	switch typedMsg := msg.(type) {
	case tasksLoadedMsg:
		typedMsg.preserveTaskID = preserveTaskID
		typedMsg.preserveIndex = preserveIndex
		return typedMsg
	case scheduleTasksLoadedMsg:
		typedMsg.preserveTaskID = preserveTaskID
		typedMsg.preserveIndex = preserveIndex
		return typedMsg
	}
	return msg
}

// loadTasksPreserveIndex loads tasks while preserving only the index (for delete)
func (m Model) loadTasksPreserveIndex() tea.Msg {
	preserveIndex := m.content.SelectedIndex()

	msg := m.loadTasksForSelection()

	switch typedMsg := msg.(type) {
	case tasksLoadedMsg:
		typedMsg.preserveTaskID = 0 // Don't try to find deleted task
		typedMsg.preserveIndex = preserveIndex
		return typedMsg
	case scheduleTasksLoadedMsg:
		typedMsg.preserveTaskID = 0
		typedMsg.preserveIndex = preserveIndex
		return typedMsg
	}
	return msg
}

// renameTask creates a command to rename a task
func (m Model) renameTask(taskID int64, newTitle string) tea.Cmd {
	return func() tea.Msg {
		updated, err := m.app.SetTaskTitle.Execute(taskID, newTitle)
		return taskRenamedMsg{task: updated, err: err}
	}
}

// renameArea creates a command to rename an area
func (m Model) renameArea(oldName, newName string) tea.Cmd {
	return func() tea.Msg {
		updated, err := m.app.RenameArea.Execute(oldName, newName)
		return areaRenamedMsg{area: updated, err: err}
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

// createProject creates a command to create a new project
func (m Model) createProject(result *CreateProjectResult) tea.Cmd {
	return func() tea.Msg {
		opts := &taskusecases.CreateProjectOptions{
			AreaName: result.AreaName,
		}
		created, err := m.app.CreateProject.Execute(result.Name, opts)
		return projectCreatedMsg{project: created, err: err}
	}
}

// createArea creates a command to create a new area
func (m Model) createArea(result *CreateAreaResult) tea.Cmd {
	return func() tea.Msg {
		created, err := m.app.CreateArea.Execute(result.Name)
		return areaCreatedMsg{area: created, err: err}
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

// completeProject creates a command to complete a project and all its tasks
func (m Model) completeProject(result *CompleteResult) tea.Cmd {
	return func() tea.Msg {
		_, err := m.app.CompleteTasks.Execute([]int64{result.ProjectID})
		return projectCompletedMsg{projectID: result.ProjectID, err: err}
	}
}

// toggleTaskState creates a command to toggle a task's someday/active state
func (m Model) toggleTaskState(taskID int64, currentState task.State) tea.Cmd {
	return func() tea.Msg {
		var updated *task.Task
		var err error
		if currentState == task.StateSomeday {
			updated, err = m.app.ActivateTask.Execute(taskID)
		} else {
			updated, err = m.app.DeferTask.Execute(taskID)
		}
		return taskStateUpdatedMsg{task: updated, err: err}
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

// deleteItem creates a command to delete an item (task, project, or area)
func (m Model) deleteItem(result *ConfirmResult) tea.Cmd {
	return func() tea.Msg {
		var err error
		switch result.Target {
		case DeleteTargetTask, DeleteTargetProject:
			// Both tasks and projects use DeleteTasks
			_, err = m.app.DeleteTasks.Execute([]int64{result.TargetID})
		case DeleteTargetArea:
			_, err = m.app.DeleteArea.Execute(result.TargetName)
		case DeleteTargetNote:
			err = m.app.DeleteNote.Execute(result.TargetPath)
		}
		return itemDeletedMsg{
			target:     result.Target,
			targetID:   result.TargetID,
			targetName: result.TargetName,
			err:        err,
		}
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

	return m, m.loadNotes(selectedTask.UUID, selectedTask.ID)
}

// openNotePreview opens the note preview pane with the selected note
func (m Model) openNotePreview() (tea.Model, tea.Cmd) {
	selectedNote := m.content.SelectedNote()
	if selectedNote == nil {
		return m, nil
	}

	m.notePreviewVisible = true
	m.focusArea = FocusDetail
	m.content = m.content.SetFocused(false)
	m.notePreviewPane = m.notePreviewPane.SetFocused(true)

	// Recalculate layout for three-column mode
	m = m.recalculateLayout()

	// Load note content asynchronously
	return m, m.loadNoteContent(selectedNote)
}

// activateFollowMode opens the detail/preview pane without moving focus from content
func (m Model) activateFollowMode() (tea.Model, tea.Cmd) {
	if m.content.ViewMode() == ContentViewTasks {
		selectedTask := m.content.SelectedTask()
		if selectedTask == nil {
			m.followMode = false
			return m, nil
		}
		m.detailVisible = true
		m.notePreviewVisible = false
		m.detailPane = m.detailPane.SetTask(selectedTask)
		m.detailPane = m.detailPane.SetFocused(false)
		m.content = m.content.SetShowSelection(true)
		m = m.recalculateLayout()
		return m, m.loadNotes(selectedTask.UUID, selectedTask.ID)
	}
	if m.content.ViewMode() == ContentViewNotes {
		selectedNote := m.content.SelectedNote()
		if selectedNote == nil {
			m.followMode = false
			return m, nil
		}
		m.notePreviewVisible = true
		m.detailVisible = false
		m.notePreviewPane = m.notePreviewPane.SetFocused(false)
		m = m.recalculateLayout()
		return m, m.updateNotePreview()
	}
	m.followMode = false
	return m, nil
}

// deactivateFollowMode closes the detail/preview pane
func (m Model) deactivateFollowMode() (tea.Model, tea.Cmd) {
	m.detailVisible = false
	m.notePreviewVisible = false
	m.content = m.content.SetShowSelection(false)
	m = m.recalculateLayout()
	return m, nil
}

// noteContentMsg carries the loaded note content
type noteContentMsg struct {
	path    string
	content string
	err     error
}

// loadNoteContent reads a note file and returns a command
func (m Model) loadNoteContent(n *note.Note) tea.Cmd {
	path := n.Path
	return func() tea.Msg {
		data, err := os.ReadFile(path)
		return noteContentMsg{path: path, content: string(data), err: err}
	}
}

// updateNotePreview loads the currently selected note into the preview
func (m Model) updateNotePreview() tea.Cmd {
	selectedNote := m.content.SelectedNote()
	if selectedNote == nil {
		return nil
	}
	return m.loadNoteContent(selectedNote)
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
	if m.detailVisible || m.notePreviewVisible {
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
	if m.notePreviewVisible && detailWidth > 0 {
		m.notePreviewPane = m.notePreviewPane.SetSize(detailWidth, sidebarHeight)
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
	// Auto-hide scope when already viewing a specific area or project
	if configKey == "area" || configKey == "project" {
		hideScope = true
	}

	opts := m.buildListOptions(item)
	opts.Sort = sortOpts

	tasks, err := m.app.ListTasks.Execute(opts)
	if err != nil {
		return loadDataMsg{err: err}
	}
	if err := m.app.EnrichIndicators.Execute(tasks); err != nil {
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

// loadDataAfterTagUpdatePreserveSelection reloads tags and tasks while preserving selection
func (m Model) loadDataAfterTagUpdatePreserveSelection() tea.Msg {
	preserveTaskID := m.content.SelectedTaskID()
	preserveIndex := m.content.SelectedIndex()

	msg := m.loadDataAfterTagUpdate()

	if typedMsg, ok := msg.(tagsAndTasksUpdatedMsg); ok {
		typedMsg.preserveTaskID = preserveTaskID
		typedMsg.preserveIndex = preserveIndex
		return typedMsg
	}
	return msg
}

// tagsAndTasksUpdatedMsg carries updated tags and tasks
type tagsAndTasksUpdatedMsg struct {
	tags           []string
	tasks          []task.Task
	title          string
	groupBy        string
	hideScope      bool
	preserveTaskID int64 // task ID to try to restore selection to
	preserveIndex  int   // fallback index if task not found
}

// notesLoadedMsg carries loaded notes for the detail pane
type notesLoadedMsg struct {
	taskID int64
	notes  []note.Note
	err    error
}

// noteEditorFinishedMsg is sent when the editor closes after editing a note
type noteEditorFinishedMsg struct {
	taskID   int64
	taskUUID string
	err      error
}

// loadNotes fetches notes for a task
func (m Model) loadNotes(taskUUID string, taskID int64) tea.Cmd {
	return func() tea.Msg {
		notes, err := m.app.ListNotes.Execute(noteusecases.ListOptions{
			EntityType: note.EntityTask,
			EntityUUID: taskUUID,
		})
		return notesLoadedMsg{taskID: taskID, notes: notes, err: err}
	}
}

// scopeNotesLoadedMsg carries notes for the selected project/area
type scopeNotesLoadedMsg struct {
	notes []note.Note
	err   error
}

// scopeNoteEditorFinishedMsg is sent when the editor closes after editing a scope note
type scopeNoteEditorFinishedMsg struct {
	err error
}

// resolveScopeEntity returns the entity type and UUID for the selected sidebar scope
func (m Model) resolveScopeEntity() (entityType note.EntityType, entityUUID string) {
	item := m.sidebar.SelectedItem()
	switch item.Type {
	case "project":
		for _, p := range m.projects {
			if p.Title == item.Key {
				return note.EntityProject, p.UUID
			}
		}
	case "area":
		for _, a := range m.areas {
			if a.Name == item.Key {
				return note.EntityArea, a.UUID
			}
		}
	}
	return "", ""
}

// loadScopeNotes fetches notes for the selected project or area
func (m Model) loadScopeNotes() tea.Cmd {
	entityType, entityUUID := m.resolveScopeEntity()
	if entityUUID == "" {
		return nil
	}

	return func() tea.Msg {
		notes, err := m.app.ListNotes.Execute(noteusecases.ListOptions{
			EntityType: entityType,
			EntityUUID: entityUUID,
		})
		return scopeNotesLoadedMsg{notes: notes, err: err}
	}
}

// noteCreatedMsg is sent after a note is created from the modal
type noteCreatedMsg struct {
	note       *note.Note
	taskID     int64
	entityUUID string
	err        error
}

// createAndOpenNote creates a note and then opens it in the editor
func (m Model) createAndOpenNote(result *CreateNoteResult) tea.Cmd {
	return func() tea.Msg {
		created, err := m.app.CreateNote.Execute(noteusecases.CreateOptions{
			EntityType: result.EntityType,
			EntityUUID: result.EntityUUID,
			Title:      result.Title,
		})
		return noteCreatedMsg{
			note:       created,
			taskID:     result.TaskID,
			entityUUID: result.EntityUUID,
			err:        err,
		}
	}
}

// openScopeNoteInEditor launches $EDITOR for a scope note
func (m Model) openScopeNoteInEditor(path string) tea.Cmd {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		editor = "vi"
	}
	fields := strings.Fields(editor)
	c := exec.Command(fields[0], append(fields[1:], path)...)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return scopeNoteEditorFinishedMsg{err: err}
	})
}

// hasScopeSelected returns true if a project or area is selected in the sidebar
func (m Model) hasScopeSelected() bool {
	item := m.sidebar.SelectedItem()
	return item.Type == "project" || item.Type == "area"
}

// openNoteInEditor launches $EDITOR for the given note path
func (m Model) openNoteInEditor(path string, taskID int64, taskUUID string) tea.Cmd {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		editor = "vi"
	}
	fields := strings.Fields(editor)
	c := exec.Command(fields[0], append(fields[1:], path)...)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return noteEditorFinishedMsg{taskID: taskID, taskUUID: taskUUID, err: err}
	})
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
	case m.confirmModal.Active():
		helpView = m.help.View(confirmKeys)
	case m.createProjectModal.Active():
		helpView = m.help.View(createProjectKeys)
	case m.createAreaModal.Active():
		helpView = m.help.View(createAreaKeys)
	case m.createNoteModal.Active():
		helpView = m.help.View(createNoteKeys)
	case m.helpModal.Active():
		helpView = m.help.View(helpModalKeys)
	default:
		helpView = m.help.View(m.currentHelpKeys())
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
	if m.confirmModal.Active() {
		return lipgloss.JoinVertical(lipgloss.Left, m.confirmModal.View(), helpView)
	}
	if m.completeModal.Active() {
		return lipgloss.JoinVertical(lipgloss.Left, m.completeModal.View(), helpView)
	}
	if m.createProjectModal.Active() {
		return lipgloss.JoinVertical(lipgloss.Left, m.createProjectModal.View(), helpView)
	}
	if m.createAreaModal.Active() {
		return lipgloss.JoinVertical(lipgloss.Left, m.createAreaModal.View(), helpView)
	}
	if m.createNoteModal.Active() {
		return lipgloss.JoinVertical(lipgloss.Left, m.createNoteModal.View(), helpView)
	}
	if m.helpModal.Active() {
		return lipgloss.JoinVertical(lipgloss.Left, m.helpModal.View(), helpView)
	}
	// Render sidebar and content side by side (gap can be 0 for tight layouts)
	contentView := lipgloss.NewStyle().MarginLeft(m.gap).Render(m.content.View())
	var mainView string
	if m.notePreviewVisible {
		// Three-column layout: sidebar | content | note preview
		previewView := lipgloss.NewStyle().MarginLeft(m.gap).Render(m.notePreviewPane.View())
		mainView = lipgloss.JoinHorizontal(lipgloss.Top, m.sidebar.View(), contentView, previewView)
	} else if m.detailVisible {
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

// currentHelpKeys returns the help keymap for the current context.
func (m Model) currentHelpKeys() help.KeyMap {
	switch m.focusArea {
	case FocusSidebar:
		if m.getSelectedProject() != nil {
			return sidebarProjectKeys
		} else if m.sidebar.SelectedItem().Type == "area" {
			return sidebarAreaKeys
		} else if m.sidebar.IsScopesSectionActive() {
			return sidebarScopesKeys
		}
		return sidebarKeys
	case FocusDetail:
		if m.notePreviewVisible {
			return notePreviewKeys
		}
		switch m.detailPane.ViewMode() {
		case DetailViewNotes:
			return detailNotesKeys
		default:
			return detailDataKeys
		}
	default:
		if m.content.ViewMode() == ContentViewNotes {
			return contentNotesKeys
		}
		return contentKeys
	}
}

// currentHelpBindings returns the key bindings for the current context.
func (m Model) currentHelpBindings() []key.Binding {
	km := m.currentHelpKeys()
	return km.ShortHelp()
}
