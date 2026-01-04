package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// CompleteModal handles project completion confirmation
type CompleteModal struct {
	active       bool
	projectID    int64
	projectTitle string
	styles       *Styles
	width        int
	height       int
}

// CompleteResult represents the outcome of the completion modal
type CompleteResult struct {
	Confirmed    bool
	ProjectID    int64
	ProjectTitle string
}

// NewCompleteModal creates a new completion modal
func NewCompleteModal(styles *Styles) CompleteModal {
	return CompleteModal{
		styles: styles,
	}
}

// Open shows the modal for completing a project
func (m CompleteModal) Open(projectID int64, title string) CompleteModal {
	m.active = true
	m.projectID = projectID
	m.projectTitle = title
	return m
}

// Close hides the modal
func (m CompleteModal) Close() CompleteModal {
	m.active = false
	return m
}

// SetSize updates the modal dimensions for centering
func (m CompleteModal) SetSize(width, height int) CompleteModal {
	m.width = width
	m.height = height
	return m
}

// Update handles key events
func (m CompleteModal) Update(msg tea.Msg) (CompleteModal, *CompleteResult) {
	if !m.active {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEscape:
			m = m.Close()
			return m, &CompleteResult{Confirmed: false}

		case tea.KeyEnter:
			result := &CompleteResult{
				Confirmed:    true,
				ProjectID:    m.projectID,
				ProjectTitle: m.projectTitle,
			}
			m = m.Close()
			return m, result
		}
	}

	return m, nil
}

// View renders the modal
func (m CompleteModal) View() string {
	if !m.active {
		return ""
	}

	titleLine := m.styles.ModalTitle.Render("Complete Project")

	// Message showing what will be completed
	message := "Complete project \"" + m.projectTitle + "\"?"

	// Hint about child tasks
	hint := m.styles.Theme.Muted.Render("All tasks in this project will also be marked as done.")

	// Help text
	helpLine := m.styles.Theme.Muted.Render("enter: confirm  esc: cancel")

	content := lipgloss.JoinVertical(lipgloss.Left, titleLine, "", message, hint, "", helpLine)
	modal := m.styles.ModalBorder.Render(content)

	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		modal,
	)
}

// Active returns whether the modal is currently shown
func (m CompleteModal) Active() bool {
	return m.active
}
