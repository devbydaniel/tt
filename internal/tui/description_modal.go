package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// DescriptionModal handles task description editing with a multiline text area
type DescriptionModal struct {
	textarea textarea.Model
	taskID   int64
	active   bool
	styles   *Styles
	width    int
	height   int
}

// DescriptionResult represents the outcome of the description modal
type DescriptionResult struct {
	TaskID      int64
	Description *string // nil means clear description
	Canceled    bool
}

// NewDescriptionModal creates a new description modal
func NewDescriptionModal(styles *Styles) DescriptionModal {
	ta := textarea.New()
	ta.Placeholder = "Enter description..."
	ta.CharLimit = 2000
	ta.ShowLineNumbers = false

	return DescriptionModal{
		textarea: ta,
		styles:   styles,
	}
}

// Open shows the modal with the given task ID and current description
func (m DescriptionModal) Open(taskID int64, currentDescription *string) DescriptionModal {
	m.active = true
	m.taskID = taskID
	if currentDescription != nil {
		m.textarea.SetValue(*currentDescription)
	} else {
		m.textarea.SetValue("")
	}
	m.textarea.Focus()
	m.textarea.CursorEnd()
	return m
}

// Close hides the modal
func (m DescriptionModal) Close() DescriptionModal {
	m.active = false
	m.textarea.Blur()
	return m
}

// SetSize updates the modal dimensions for centering
func (m DescriptionModal) SetSize(width, height int) DescriptionModal {
	m.width = width
	m.height = height
	// Text area size: modal content width minus padding
	m.textarea.SetWidth(50)
	m.textarea.SetHeight(8)
	return m
}

// Update handles input events, returns updated modal and optional result
func (m DescriptionModal) Update(msg tea.Msg) (DescriptionModal, *DescriptionResult) {
	if !m.active {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEscape:
			m = m.Close()
			return m, &DescriptionResult{Canceled: true}

		case tea.KeyCtrlS:
			// Submit with Ctrl+S
			return m.submit()
		}

		// Check for Ctrl+Enter
		if msg.Type == tea.KeyEnter && msg.Alt {
			return m.submit()
		}
	}

	// Pass other keys to the text area
	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(msg)
	_ = cmd // We handle commands synchronously in the parent
	return m, nil
}

// submit handles submitting the description
func (m DescriptionModal) submit() (DescriptionModal, *DescriptionResult) {
	desc := strings.TrimSpace(m.textarea.Value())
	m = m.Close()

	var descPtr *string
	if desc != "" {
		descPtr = &desc
	}

	return m, &DescriptionResult{
		TaskID:      m.taskID,
		Description: descPtr,
	}
}

// View renders the modal
func (m DescriptionModal) View() string {
	if !m.active {
		return ""
	}

	// Modal content
	title := m.styles.ModalTitle.Render("Edit Description")
	textArea := m.textarea.View()
	help := m.styles.Theme.Muted.Render("ctrl+s/alt+enter: save  esc: cancel")

	content := lipgloss.JoinVertical(lipgloss.Left, title, "", textArea, "", help)
	modal := m.styles.ModalBorder.Render(content)

	// Center the modal
	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		modal,
	)
}

// Active returns whether the modal is currently shown
func (m DescriptionModal) Active() bool {
	return m.active
}

// TaskID returns the ID of the task being edited
func (m DescriptionModal) TaskID() int64 {
	return m.taskID
}

// Value returns the current textarea value
func (m DescriptionModal) Value() string {
	return m.textarea.Value()
}
