package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// CommentModal handles adding a comment to a task with a multiline text area
type CommentModal struct {
	textarea textarea.Model
	taskID   int64
	active   bool
	styles   *Styles
	width    int
	height   int
}

// CommentResult represents the outcome of the comment modal
type CommentResult struct {
	TaskID   int64
	Body     string
	Canceled bool
}

// NewCommentModal creates a new comment modal
func NewCommentModal(styles *Styles) CommentModal {
	ta := textarea.New()
	ta.Placeholder = "Write your comment..."
	ta.CharLimit = 2000
	ta.ShowLineNumbers = false

	return CommentModal{
		textarea: ta,
		styles:   styles,
	}
}

// Open shows the modal for the given task
func (m CommentModal) Open(taskID int64) CommentModal {
	m.active = true
	m.taskID = taskID
	m.textarea.SetValue("")
	m.textarea.Focus()
	return m
}

// Close hides the modal
func (m CommentModal) Close() CommentModal {
	m.active = false
	m.textarea.Blur()
	return m
}

// SetSize updates the modal dimensions for centering
func (m CommentModal) SetSize(width, height int) CommentModal {
	m.width = width
	m.height = height
	m.textarea.SetWidth(50)
	m.textarea.SetHeight(8)
	return m
}

// Update handles input events, returns updated modal and optional result
func (m CommentModal) Update(msg tea.Msg) (CommentModal, *CommentResult) {
	if !m.active {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEscape:
			m = m.Close()
			return m, &CommentResult{Canceled: true}

		case tea.KeyCtrlS:
			return m.submit()
		}

		// Check for Alt+Enter
		if msg.Type == tea.KeyEnter && msg.Alt {
			return m.submit()
		}
	}

	// Pass other keys to the text area
	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(msg)
	_ = cmd
	return m, nil
}

// submit handles submitting the comment
func (m CommentModal) submit() (CommentModal, *CommentResult) {
	body := strings.TrimSpace(m.textarea.Value())
	m = m.Close()

	if body == "" {
		return m, &CommentResult{Canceled: true}
	}

	return m, &CommentResult{
		TaskID: m.taskID,
		Body:   body,
	}
}

// View renders the modal
func (m CommentModal) View() string {
	if !m.active {
		return ""
	}

	title := m.styles.ModalTitle.Render("Add Comment")
	textArea := m.textarea.View()
	help := m.styles.Theme.Muted.Render("ctrl+s/alt+enter: save  esc: cancel")

	content := lipgloss.JoinVertical(lipgloss.Left, title, "", textArea, "", help)
	modal := m.styles.ModalBorder.Render(content)

	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		modal,
	)
}

// Active returns whether the modal is currently shown
func (m CommentModal) Active() bool {
	return m.active
}
