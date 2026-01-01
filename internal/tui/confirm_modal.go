package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// DeleteTarget represents what type of item is being deleted
type DeleteTarget int

const (
	DeleteTargetTask DeleteTarget = iota
	DeleteTargetProject
	DeleteTargetArea
)

// ConfirmModal handles delete confirmation
type ConfirmModal struct {
	active     bool
	target     DeleteTarget
	targetID   int64  // Task/Project ID (for tasks and projects)
	targetName string // Display name for confirmation message
	styles     *Styles
	width      int
	height     int
}

// ConfirmResult represents the outcome of the confirmation modal
type ConfirmResult struct {
	Confirmed  bool
	Target     DeleteTarget
	TargetID   int64
	TargetName string
}

// NewConfirmModal creates a new confirmation modal
func NewConfirmModal(styles *Styles) ConfirmModal {
	return ConfirmModal{
		styles: styles,
	}
}

// OpenForTask shows the modal for deleting a task
func (m ConfirmModal) OpenForTask(taskID int64, title string) ConfirmModal {
	m.active = true
	m.target = DeleteTargetTask
	m.targetID = taskID
	m.targetName = title
	return m
}

// OpenForProject shows the modal for deleting a project
func (m ConfirmModal) OpenForProject(projectID int64, title string) ConfirmModal {
	m.active = true
	m.target = DeleteTargetProject
	m.targetID = projectID
	m.targetName = title
	return m
}

// OpenForArea shows the modal for deleting an area
func (m ConfirmModal) OpenForArea(name string) ConfirmModal {
	m.active = true
	m.target = DeleteTargetArea
	m.targetID = 0
	m.targetName = name
	return m
}

// Close hides the modal
func (m ConfirmModal) Close() ConfirmModal {
	m.active = false
	return m
}

// SetSize updates the modal dimensions for centering
func (m ConfirmModal) SetSize(width, height int) ConfirmModal {
	m.width = width
	m.height = height
	return m
}

// Update handles key events
func (m ConfirmModal) Update(msg tea.Msg) (ConfirmModal, *ConfirmResult) {
	if !m.active {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEscape:
			m = m.Close()
			return m, &ConfirmResult{Confirmed: false}

		case tea.KeyEnter:
			result := &ConfirmResult{
				Confirmed:  true,
				Target:     m.target,
				TargetID:   m.targetID,
				TargetName: m.targetName,
			}
			m = m.Close()
			return m, result
		}
	}

	return m, nil
}

// View renders the modal
func (m ConfirmModal) View() string {
	if !m.active {
		return ""
	}

	// Title based on target type
	var title, itemType string
	switch m.target {
	case DeleteTargetTask:
		title = "Delete Task"
		itemType = "task"
	case DeleteTargetProject:
		title = "Delete Project"
		itemType = "project"
	case DeleteTargetArea:
		title = "Delete Area"
		itemType = "area"
	}

	titleLine := m.styles.ModalTitle.Render(title)

	// Message showing what will be deleted
	message := "Delete " + itemType + " \"" + m.targetName + "\"?"

	// Help text
	helpLine := m.styles.Theme.Muted.Render("enter: confirm  esc: cancel")

	content := lipgloss.JoinVertical(lipgloss.Left, titleLine, "", message, "", helpLine)
	modal := m.styles.ModalBorder.Render(content)

	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		modal,
	)
}

// Active returns whether the modal is currently shown
func (m ConfirmModal) Active() bool {
	return m.active
}
