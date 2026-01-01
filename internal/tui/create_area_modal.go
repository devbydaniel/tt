package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// CreateAreaModal handles creating a new area
type CreateAreaModal struct {
	input  textinput.Model
	active bool
	err    error
	styles *Styles
	width  int
	height int
}

// CreateAreaResult represents the outcome of the create area modal
type CreateAreaResult struct {
	Name     string
	Canceled bool
}

// NewCreateAreaModal creates a new create area modal
func NewCreateAreaModal(styles *Styles) CreateAreaModal {
	ti := textinput.New()
	ti.Placeholder = "Area name"
	ti.CharLimit = 100

	return CreateAreaModal{
		input:  ti,
		styles: styles,
	}
}

// Open shows the modal
func (m CreateAreaModal) Open() CreateAreaModal {
	m.active = true
	m.err = nil
	m.input.SetValue("")
	m.input.Focus()
	return m
}

// Close hides the modal
func (m CreateAreaModal) Close() CreateAreaModal {
	m.active = false
	m.input.Blur()
	return m
}

// SetSize updates the modal dimensions
func (m CreateAreaModal) SetSize(width, height int) CreateAreaModal {
	m.width = width
	m.height = height
	m.input.Width = 40
	return m
}

// Active returns whether the modal is currently shown
func (m CreateAreaModal) Active() bool {
	return m.active
}

// Update handles input events
func (m CreateAreaModal) Update(msg tea.Msg) (CreateAreaModal, *CreateAreaResult) {
	if !m.active {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEscape:
			m = m.Close()
			return m, &CreateAreaResult{Canceled: true}

		case tea.KeyEnter:
			name := strings.TrimSpace(m.input.Value())
			if name == "" {
				m.err = errAreaNameRequired
				return m, nil
			}
			m = m.Close()
			return m, &CreateAreaResult{Name: name}
		}
	}

	// Pass other keys to the text input
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	_ = cmd

	return m, nil
}

// View renders the modal
func (m CreateAreaModal) View() string {
	if !m.active {
		return ""
	}

	title := m.styles.ModalTitle.Render("Create Area")
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

// Error messages
var errAreaNameRequired = &createAreaError{"Area name is required"}

type createAreaError struct {
	msg string
}

func (e *createAreaError) Error() string {
	return e.msg
}
