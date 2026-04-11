package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/devbydaniel/tt/internal/domain/note"
)

// CreateNoteModal handles creating a new note
type CreateNoteModal struct {
	input      textinput.Model
	active     bool
	err        error
	styles     *Styles
	width      int
	height     int
	entityType note.EntityType
	entityUUID string
	taskID     int64 // non-zero for task/project detail notes
}

// CreateNoteResult represents the outcome of the create note modal
type CreateNoteResult struct {
	Title      string
	EntityType note.EntityType
	EntityUUID string
	TaskID     int64
	Canceled   bool
}

// NewCreateNoteModal creates a new create note modal
func NewCreateNoteModal(styles *Styles) CreateNoteModal {
	ti := textinput.New()
	ti.Placeholder = "Note title"
	ti.CharLimit = 100

	return CreateNoteModal{
		input:  ti,
		styles: styles,
	}
}

// Open shows the modal with entity context
func (m CreateNoteModal) Open(entityType note.EntityType, entityUUID string, taskID int64) CreateNoteModal {
	m.active = true
	m.err = nil
	m.entityType = entityType
	m.entityUUID = entityUUID
	m.taskID = taskID
	m.input.SetValue("")
	m.input.Focus()
	return m
}

// Close hides the modal
func (m CreateNoteModal) Close() CreateNoteModal {
	m.active = false
	m.input.Blur()
	return m
}

// SetSize updates the modal dimensions
func (m CreateNoteModal) SetSize(width, height int) CreateNoteModal {
	m.width = width
	m.height = height
	m.input.Width = 40
	return m
}

// Active returns whether the modal is currently shown
func (m CreateNoteModal) Active() bool {
	return m.active
}

// Update handles input events
func (m CreateNoteModal) Update(msg tea.Msg) (CreateNoteModal, *CreateNoteResult) {
	if !m.active {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEscape:
			m = m.Close()
			return m, &CreateNoteResult{Canceled: true}

		case tea.KeyEnter:
			title := strings.TrimSpace(m.input.Value())
			if title == "" {
				m.err = errNoteTitleRequired
				return m, nil
			}
			m = m.Close()
			return m, &CreateNoteResult{
				Title:      title,
				EntityType: m.entityType,
				EntityUUID: m.entityUUID,
				TaskID:     m.taskID,
			}
		}
	}

	// Pass other keys to the text input
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	_ = cmd

	return m, nil
}

// View renders the modal
func (m CreateNoteModal) View() string {
	if !m.active {
		return ""
	}

	title := m.styles.ModalTitle.Render("Create Note")
	input := m.input.View()

	var errView string
	if m.err != nil {
		errView = m.styles.Theme.Error.Render(m.err.Error())
	}

	var parts []string
	parts = append(parts, title, "", input)
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

var errNoteTitleRequired = &modalError{"Note title is required"}
