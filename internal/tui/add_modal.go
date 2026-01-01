package tui

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/devbydaniel/tt/internal/dateparse"
	"github.com/devbydaniel/tt/internal/domain/area"
	"github.com/devbydaniel/tt/internal/domain/task"
	"github.com/sahilm/fuzzy"
)

// AddModalField tracks which field is currently focused
type AddModalField int

const (
	AddFieldTitle AddModalField = iota
	AddFieldDescription
	AddFieldScope
	AddFieldPlanned
	AddFieldDue
	AddFieldTags
)

// AddModal handles task creation with multiple fields
type AddModal struct {
	// Text inputs
	titleInput textinput.Model
	descInput  textinput.Model
	tagsInput  textinput.Model

	// Date inputs
	plannedInput textinput.Model
	dueInput     textinput.Model

	// Scope selector
	scopeInput     textinput.Model
	allScopes      []MoveItem
	filteredScopes []MoveItem
	scopeSelected  int
	selectedScope  *MoveItem

	// State
	activeField AddModalField
	active      bool
	err         error

	// Styling and dimensions
	styles *Styles
	width  int
	height int
}

// AddResult represents the outcome of the add modal
type AddResult struct {
	Title       string
	Description string
	ProjectName string
	AreaName    string
	PlannedDate *time.Time
	DueDate     *time.Time
	Tags        []string
	Canceled    bool
}

// NewAddModal creates a new add modal
func NewAddModal(styles *Styles) AddModal {
	titleInput := textinput.New()
	titleInput.Placeholder = "Task title (required)"
	titleInput.CharLimit = 500

	descInput := textinput.New()
	descInput.Placeholder = "Description (optional)"
	descInput.CharLimit = 2000

	scopeInput := textinput.New()
	scopeInput.Placeholder = "Type to filter projects/areas..."
	scopeInput.CharLimit = 100

	plannedInput := textinput.New()
	plannedInput.Placeholder = "today, tomorrow, +3d, monday..."
	plannedInput.CharLimit = 50

	dueInput := textinput.New()
	dueInput.Placeholder = "Due date (optional)"
	dueInput.CharLimit = 50

	tagsInput := textinput.New()
	tagsInput.Placeholder = "tag1, tag2, tag3"
	tagsInput.CharLimit = 200

	return AddModal{
		titleInput:   titleInput,
		descInput:    descInput,
		scopeInput:   scopeInput,
		plannedInput: plannedInput,
		dueInput:     dueInput,
		tagsInput:    tagsInput,
		styles:       styles,
	}
}

// buildScopes creates the list of selectable scopes
func (m AddModal) buildScopes(projects []task.Task, areas []area.Area) []MoveItem {
	items := []MoveItem{
		{Type: "none", Name: "", Label: "None (Inbox)"},
	}

	// Add projects
	for _, p := range projects {
		label := p.Title
		if p.AreaName != nil {
			label = *p.AreaName + " > " + p.Title
		}
		items = append(items, MoveItem{
			Type:  "project",
			Name:  p.Title,
			Label: label,
		})
	}

	// Add areas
	for _, a := range areas {
		items = append(items, MoveItem{
			Type:  "area",
			Name:  a.Name,
			Label: a.Name,
		})
	}

	return items
}

// Open shows the modal with optional pre-filled scope from sidebar context
func (m AddModal) Open(projects []task.Task, areas []area.Area, sidebarItem *SidebarItem) AddModal {
	m.active = true
	m.activeField = AddFieldTitle
	m.err = nil

	// Reset all inputs
	m.titleInput.SetValue("")
	m.descInput.SetValue("")
	m.scopeInput.SetValue("")
	m.plannedInput.SetValue("")
	m.dueInput.SetValue("")
	m.tagsInput.SetValue("")

	// Build scope list
	m.allScopes = m.buildScopes(projects, areas)
	m.filteredScopes = m.allScopes
	m.scopeSelected = 0
	m.selectedScope = nil

	// Pre-fill scope based on sidebar context
	if sidebarItem != nil {
		switch sidebarItem.Type {
		case "project":
			for i, item := range m.allScopes {
				if item.Type == "project" && item.Name == sidebarItem.Key {
					m.scopeSelected = i
					m.selectedScope = &m.allScopes[i]
					break
				}
			}
		case "area":
			for i, item := range m.allScopes {
				if item.Type == "area" && item.Name == sidebarItem.Key {
					m.scopeSelected = i
					m.selectedScope = &m.allScopes[i]
					break
				}
			}
		}
	}

	m.titleInput.Focus()
	return m
}

// Close hides the modal
func (m AddModal) Close() AddModal {
	m.active = false
	m.titleInput.Blur()
	m.descInput.Blur()
	m.scopeInput.Blur()
	m.plannedInput.Blur()
	m.dueInput.Blur()
	m.tagsInput.Blur()
	return m
}

// SetSize updates the modal dimensions
func (m AddModal) SetSize(width, height int) AddModal {
	m.width = width
	m.height = height
	inputWidth := 40
	m.titleInput.Width = inputWidth
	m.descInput.Width = inputWidth
	m.scopeInput.Width = inputWidth
	m.plannedInput.Width = inputWidth
	m.dueInput.Width = inputWidth
	m.tagsInput.Width = inputWidth
	return m
}

// Active returns whether the modal is currently shown
func (m AddModal) Active() bool {
	return m.active
}

// nextField moves to the next field
func (m AddModal) nextField() AddModal {
	m.activeField = (m.activeField + 1) % 6
	return m.updateFocus()
}

// prevField moves to the previous field
func (m AddModal) prevField() AddModal {
	m.activeField = (m.activeField - 1 + 6) % 6
	return m.updateFocus()
}

// updateFocus updates which input has focus
func (m AddModal) updateFocus() AddModal {
	// Blur all inputs
	m.titleInput.Blur()
	m.descInput.Blur()
	m.scopeInput.Blur()
	m.plannedInput.Blur()
	m.dueInput.Blur()
	m.tagsInput.Blur()

	// Focus the active field
	switch m.activeField {
	case AddFieldTitle:
		m.titleInput.Focus()
	case AddFieldDescription:
		m.descInput.Focus()
	case AddFieldScope:
		m.scopeInput.Focus()
	case AddFieldPlanned:
		m.plannedInput.Focus()
	case AddFieldDue:
		m.dueInput.Focus()
	case AddFieldTags:
		m.tagsInput.Focus()
	}
	return m
}

// filterScopes filters scopes based on the current input using fuzzy matching
func (m AddModal) filterScopes() []MoveItem {
	query := strings.TrimSpace(m.scopeInput.Value())
	if query == "" {
		return m.allScopes
	}

	// Build list of labels for fuzzy matching
	labels := make([]string, len(m.allScopes))
	for i, item := range m.allScopes {
		labels[i] = item.Label
	}

	// Fuzzy match
	matches := fuzzy.Find(query, labels)

	// Return matched items in order of match quality
	result := make([]MoveItem, len(matches))
	for i, match := range matches {
		result[i] = m.allScopes[match.Index]
	}

	return result
}

// Update handles input events
func (m AddModal) Update(msg tea.Msg) (AddModal, *AddResult) {
	if !m.active {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEscape:
			m = m.Close()
			return m, &AddResult{Canceled: true}

		case tea.KeyTab:
			m = m.nextField()
			return m, nil

		case tea.KeyShiftTab:
			m = m.prevField()
			return m, nil

		case tea.KeyEnter:
			if m.activeField == AddFieldScope {
				// Select current scope and move to next field
				if len(m.filteredScopes) > 0 && m.scopeSelected < len(m.filteredScopes) {
					selected := m.filteredScopes[m.scopeSelected]
					m.selectedScope = &selected
				}
				m = m.nextField()
				return m, nil
			}
			// Try to submit the form
			return m.trySubmit()

		case tea.KeyUp:
			if m.activeField == AddFieldScope {
				if m.scopeSelected > 0 {
					m.scopeSelected--
				}
				return m, nil
			}

		case tea.KeyDown:
			if m.activeField == AddFieldScope {
				if m.scopeSelected < len(m.filteredScopes)-1 {
					m.scopeSelected++
				}
				return m, nil
			}
		}
	}

	// Route to active field's input
	m = m.updateActiveInput(msg)
	return m, nil
}

// updateActiveInput updates the currently active input field
func (m AddModal) updateActiveInput(msg tea.Msg) AddModal {
	var cmd tea.Cmd

	switch m.activeField {
	case AddFieldTitle:
		m.titleInput, cmd = m.titleInput.Update(msg)
	case AddFieldDescription:
		m.descInput, cmd = m.descInput.Update(msg)
	case AddFieldScope:
		prevValue := m.scopeInput.Value()
		m.scopeInput, cmd = m.scopeInput.Update(msg)
		// Re-filter if input changed
		if m.scopeInput.Value() != prevValue {
			m.filteredScopes = m.filterScopes()
			m.scopeSelected = 0
		}
	case AddFieldPlanned:
		m.plannedInput, cmd = m.plannedInput.Update(msg)
	case AddFieldDue:
		m.dueInput, cmd = m.dueInput.Update(msg)
	case AddFieldTags:
		m.tagsInput, cmd = m.tagsInput.Update(msg)
	}
	_ = cmd

	return m
}

// trySubmit attempts to submit the form
func (m AddModal) trySubmit() (AddModal, *AddResult) {
	title := strings.TrimSpace(m.titleInput.Value())
	if title == "" {
		m.err = errTitleRequired
		return m, nil
	}

	result := &AddResult{
		Title:       title,
		Description: strings.TrimSpace(m.descInput.Value()),
	}

	// Parse scope
	if m.selectedScope != nil {
		switch m.selectedScope.Type {
		case "project":
			result.ProjectName = m.selectedScope.Name
		case "area":
			result.AreaName = m.selectedScope.Name
		}
	}

	// Parse planned date
	if v := strings.TrimSpace(m.plannedInput.Value()); v != "" {
		parsed, err := dateparse.Parse(v)
		if err != nil {
			m.err = errInvalidPlannedDate
			return m, nil
		}
		result.PlannedDate = &parsed
	}

	// Parse due date
	if v := strings.TrimSpace(m.dueInput.Value()); v != "" {
		parsed, err := dateparse.Parse(v)
		if err != nil {
			m.err = errInvalidDueDate
			return m, nil
		}
		result.DueDate = &parsed
	}

	// Parse tags
	if v := strings.TrimSpace(m.tagsInput.Value()); v != "" {
		parts := strings.Split(v, ",")
		for _, p := range parts {
			if tag := strings.TrimSpace(p); tag != "" {
				result.Tags = append(result.Tags, tag)
			}
		}
	}

	m = m.Close()
	return m, result
}

// View renders the modal
func (m AddModal) View() string {
	if !m.active {
		return ""
	}

	title := m.styles.ModalTitle.Render("Add Task")

	// Build field views
	fields := []string{
		m.renderField("Title", m.titleInput.View(), AddFieldTitle),
		m.renderField("Description", m.descInput.View(), AddFieldDescription),
		m.renderScopeField(),
		m.renderField("Planned", m.plannedInput.View(), AddFieldPlanned),
		m.renderField("Due", m.dueInput.View(), AddFieldDue),
		m.renderField("Tags", m.tagsInput.View(), AddFieldTags),
	}

	// Error display
	var errView string
	if m.err != nil {
		errView = m.styles.Theme.Error.Render(m.err.Error())
	}

	// Combine
	parts := []string{title}
	parts = append(parts, fields...)
	if errView != "" {
		parts = append(parts, "", errView)
	}

	content := lipgloss.JoinVertical(lipgloss.Left, parts...)
	modal := m.styles.ModalBorder.Render(content)

	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		modal,
	)
}

// renderField renders a single field with focus indicator
func (m AddModal) renderField(label, input string, field AddModalField) string {
	prefix := "  "
	if m.activeField == field {
		prefix = "> "
	}
	return prefix + label + ": " + input
}

// renderScopeField renders the scope selector field
func (m AddModal) renderScopeField() string {
	prefix := "  "
	if m.activeField == AddFieldScope {
		prefix = "> "
	}

	var scopeView string
	if m.activeField == AddFieldScope {
		scopeView = m.scopeInput.View() + "\n" + m.renderScopeList()
	} else if m.selectedScope != nil {
		scopeView = m.selectedScope.Label
	} else {
		scopeView = "(Inbox)"
	}

	return prefix + "Scope: " + scopeView
}

// renderScopeList renders the filtered list of scopes
func (m AddModal) renderScopeList() string {
	const maxVisible = 5

	// Calculate offset for scrolling
	offset := 0
	if m.scopeSelected >= maxVisible {
		offset = m.scopeSelected - maxVisible + 1
	}

	var lines []string
	for i := 0; i < maxVisible; i++ {
		idx := offset + i
		if idx < len(m.filteredScopes) {
			item := m.filteredScopes[idx]
			if idx == m.scopeSelected {
				lines = append(lines, m.styles.SelectedItem.Render("     > "+item.Label))
			} else {
				lines = append(lines, "       "+item.Label)
			}
		}
	}

	if len(m.filteredScopes) == 0 {
		lines = append(lines, m.styles.Theme.Muted.Render("       No matches"))
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// Error messages
var (
	errTitleRequired     = &addModalError{"Title is required"}
	errInvalidPlannedDate = &addModalError{"Invalid planned date"}
	errInvalidDueDate    = &addModalError{"Invalid due date"}
)

type addModalError struct {
	msg string
}

func (e *addModalError) Error() string {
	return e.msg
}
