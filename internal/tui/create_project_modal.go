package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/devbydaniel/tt/internal/domain/area"
	"github.com/sahilm/fuzzy"
)

// CreateProjectField tracks which field is currently focused
type CreateProjectField int

const (
	CreateProjectFieldTitle CreateProjectField = iota
	CreateProjectFieldArea
)

// CreateProjectModal handles creating a new project with optional area
type CreateProjectModal struct {
	// Title input
	titleInput textinput.Model

	// Area selector
	areaInput     textinput.Model
	allAreas      []MoveItem
	filteredAreas []MoveItem
	areaSelected  int
	selectedArea  *MoveItem

	// State
	activeField CreateProjectField
	active      bool
	err         error

	// Styling and dimensions
	styles *Styles
	width  int
	height int
}

// CreateProjectResult represents the outcome of the create project modal
type CreateProjectResult struct {
	Name     string
	AreaName string // empty if no area selected
	Canceled bool
}

// NewCreateProjectModal creates a new create project modal
func NewCreateProjectModal(styles *Styles) CreateProjectModal {
	titleInput := textinput.New()
	titleInput.Placeholder = "Project name"
	titleInput.CharLimit = 500

	areaInput := textinput.New()
	areaInput.Placeholder = "Type to filter areas..."
	areaInput.CharLimit = 100

	return CreateProjectModal{
		titleInput: titleInput,
		areaInput:  areaInput,
		styles:     styles,
	}
}

// buildAreas creates the list of selectable areas
func (m CreateProjectModal) buildAreas(areas []area.Area) []MoveItem {
	items := []MoveItem{
		{Type: "none", Name: "", Label: "None"},
	}

	for _, a := range areas {
		items = append(items, MoveItem{
			Type:  "area",
			Name:  a.Name,
			Label: a.Name,
		})
	}

	return items
}

// Open shows the modal
func (m CreateProjectModal) Open(areas []area.Area) CreateProjectModal {
	m.active = true
	m.activeField = CreateProjectFieldTitle
	m.err = nil

	// Reset inputs
	m.titleInput.SetValue("")
	m.areaInput.SetValue("")

	// Build area list
	m.allAreas = m.buildAreas(areas)
	m.filteredAreas = m.allAreas
	m.areaSelected = 0
	m.selectedArea = nil

	m.titleInput.Focus()
	return m
}

// Close hides the modal
func (m CreateProjectModal) Close() CreateProjectModal {
	m.active = false
	m.titleInput.Blur()
	m.areaInput.Blur()
	return m
}

// SetSize updates the modal dimensions
func (m CreateProjectModal) SetSize(width, height int) CreateProjectModal {
	m.width = width
	m.height = height
	m.titleInput.Width = 40
	m.areaInput.Width = 40
	return m
}

// Active returns whether the modal is currently shown
func (m CreateProjectModal) Active() bool {
	return m.active
}

// nextField moves to the next field
func (m CreateProjectModal) nextField() CreateProjectModal {
	m.activeField = (m.activeField + 1) % 2
	return m.updateFocus()
}

// prevField moves to the previous field
func (m CreateProjectModal) prevField() CreateProjectModal {
	m.activeField = (m.activeField - 1 + 2) % 2
	return m.updateFocus()
}

// updateFocus updates which input has focus
func (m CreateProjectModal) updateFocus() CreateProjectModal {
	m.titleInput.Blur()
	m.areaInput.Blur()

	switch m.activeField {
	case CreateProjectFieldTitle:
		m.titleInput.Focus()
	case CreateProjectFieldArea:
		m.areaInput.Focus()
	}
	return m
}

// filterAreas filters areas based on the current input using fuzzy matching
func (m CreateProjectModal) filterAreas() []MoveItem {
	query := strings.TrimSpace(m.areaInput.Value())
	if query == "" {
		return m.allAreas
	}

	// Build list of labels for fuzzy matching
	labels := make([]string, len(m.allAreas))
	for i, item := range m.allAreas {
		labels[i] = item.Label
	}

	// Fuzzy match
	matches := fuzzy.Find(query, labels)

	// Return matched items in order of match quality
	result := make([]MoveItem, len(matches))
	for i, match := range matches {
		result[i] = m.allAreas[match.Index]
	}

	return result
}

// Update handles input events
func (m CreateProjectModal) Update(msg tea.Msg) (CreateProjectModal, *CreateProjectResult) {
	if !m.active {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEscape:
			m = m.Close()
			return m, &CreateProjectResult{Canceled: true}

		case tea.KeyTab:
			m = m.nextField()
			return m, nil

		case tea.KeyShiftTab:
			m = m.prevField()
			return m, nil

		case tea.KeyEnter:
			if m.activeField == CreateProjectFieldArea {
				// Select current area and move to submit
				if len(m.filteredAreas) > 0 && m.areaSelected < len(m.filteredAreas) {
					selected := m.filteredAreas[m.areaSelected]
					m.selectedArea = &selected
				}
				// Try to submit
				return m.trySubmit()
			}
			// Try to submit from title field
			return m.trySubmit()

		case tea.KeyUp:
			if m.activeField == CreateProjectFieldArea {
				if m.areaSelected > 0 {
					m.areaSelected--
				}
				return m, nil
			}

		case tea.KeyDown:
			if m.activeField == CreateProjectFieldArea {
				if m.areaSelected < len(m.filteredAreas)-1 {
					m.areaSelected++
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
func (m CreateProjectModal) updateActiveInput(msg tea.Msg) CreateProjectModal {
	var cmd tea.Cmd

	switch m.activeField {
	case CreateProjectFieldTitle:
		m.titleInput, cmd = m.titleInput.Update(msg)
	case CreateProjectFieldArea:
		prevValue := m.areaInput.Value()
		m.areaInput, cmd = m.areaInput.Update(msg)
		// Re-filter if input changed
		if m.areaInput.Value() != prevValue {
			m.filteredAreas = m.filterAreas()
			m.areaSelected = 0
		}
	}
	_ = cmd

	return m
}

// trySubmit attempts to submit the form
func (m CreateProjectModal) trySubmit() (CreateProjectModal, *CreateProjectResult) {
	name := strings.TrimSpace(m.titleInput.Value())
	if name == "" {
		m.err = errProjectNameRequired
		return m, nil
	}

	result := &CreateProjectResult{
		Name: name,
	}

	// Set area if selected and not "None"
	if m.selectedArea != nil && m.selectedArea.Type == "area" {
		result.AreaName = m.selectedArea.Name
	}

	m = m.Close()
	return m, result
}

// View renders the modal
func (m CreateProjectModal) View() string {
	if !m.active {
		return ""
	}

	title := m.styles.ModalTitle.Render("Create Project")

	// Build field views
	fields := []string{
		m.renderField("Name", m.titleInput.View(), CreateProjectFieldTitle),
		m.renderAreaField(),
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
func (m CreateProjectModal) renderField(label, input string, field CreateProjectField) string {
	prefix := "  "
	if m.activeField == field {
		prefix = "> "
	}
	return prefix + label + ": " + input
}

// renderAreaField renders the area selector field
func (m CreateProjectModal) renderAreaField() string {
	prefix := "  "
	if m.activeField == CreateProjectFieldArea {
		prefix = "> "
	}

	var areaView string
	if m.activeField == CreateProjectFieldArea {
		areaView = m.areaInput.View() + "\n" + m.renderAreaList()
	} else if m.selectedArea != nil {
		areaView = m.selectedArea.Label
	} else {
		areaView = "(None)"
	}

	return prefix + "Area: " + areaView
}

// renderAreaList renders the filtered list of areas
func (m CreateProjectModal) renderAreaList() string {
	const maxVisible = 5

	// Calculate offset for scrolling
	offset := 0
	if m.areaSelected >= maxVisible {
		offset = m.areaSelected - maxVisible + 1
	}

	var lines []string
	for i := 0; i < maxVisible; i++ {
		idx := offset + i
		if idx < len(m.filteredAreas) {
			item := m.filteredAreas[idx]
			if idx == m.areaSelected {
				lines = append(lines, m.styles.SelectedItem.Render("     > "+item.Label))
			} else {
				lines = append(lines, "       "+item.Label)
			}
		}
	}

	if len(m.filteredAreas) == 0 {
		lines = append(lines, m.styles.Theme.Muted.Render("       No matches"))
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// Error messages
var errProjectNameRequired = &createProjectError{"Project name is required"}

type createProjectError struct {
	msg string
}

func (e *createProjectError) Error() string {
	return e.msg
}
