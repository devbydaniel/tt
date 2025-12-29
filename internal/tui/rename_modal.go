package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// RenameModal handles task renaming with a text input
type RenameModal struct {
	input  textinput.Model
	taskID int64
	active bool
	styles *Styles
	width  int
	height int
}

// RenameResult represents the outcome of the rename modal
type RenameResult struct {
	TaskID   int64
	NewTitle string
	Canceled bool
}

// NewRenameModal creates a new rename modal
func NewRenameModal(styles *Styles) RenameModal {
	ti := textinput.New()
	ti.Placeholder = "Task title"
	ti.CharLimit = 500

	return RenameModal{
		input:  ti,
		styles: styles,
	}
}

// Open shows the modal with the given task ID and current title
func (m RenameModal) Open(taskID int64, currentTitle string) RenameModal {
	m.active = true
	m.taskID = taskID
	m.input.SetValue(currentTitle)
	m.input.Focus()
	m.input.CursorEnd()
	return m
}

// Close hides the modal
func (m RenameModal) Close() RenameModal {
	m.active = false
	m.input.Blur()
	return m
}

// SetSize updates the modal dimensions for centering
func (m RenameModal) SetSize(width, height int) RenameModal {
	m.width = width
	m.height = height
	// Input width is modal content width minus padding
	m.input.Width = 40
	return m
}

// Update handles input events, returns updated modal and optional result
func (m RenameModal) Update(msg tea.Msg) (RenameModal, *RenameResult) {
	if !m.active {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEscape:
			m = m.Close()
			return m, &RenameResult{Canceled: true}

		case tea.KeyEnter:
			title := strings.TrimSpace(m.input.Value())
			if title == "" {
				// Don't allow empty titles, stay in modal
				return m, nil
			}
			m = m.Close()
			return m, &RenameResult{
				TaskID:   m.taskID,
				NewTitle: title,
			}
		}
	}

	// Pass other keys to the text input
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	_ = cmd // We handle commands synchronously in the parent
	return m, nil
}

// View renders the modal
func (m RenameModal) View() string {
	if !m.active {
		return ""
	}

	// Modal content
	title := m.styles.ModalTitle.Render("Rename Task")
	input := m.input.View()

	content := lipgloss.JoinVertical(lipgloss.Left, title, input)
	modal := m.styles.ModalBorder.Render(content)

	// Center the modal
	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		modal,
	)
}

// Active returns whether the modal is currently shown
func (m RenameModal) Active() bool {
	return m.active
}

// TaskID returns the ID of the task being renamed
func (m RenameModal) TaskID() int64 {
	return m.taskID
}

// Value returns the current input value
func (m RenameModal) Value() string {
	return m.input.Value()
}
