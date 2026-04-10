package tui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// HelpModal displays context-sensitive key bindings in a modal overlay.
type HelpModal struct {
	active   bool
	styles   *Styles
	width    int
	height   int
	bindings []key.Binding
}

// NewHelpModal creates a new help modal.
func NewHelpModal(styles *Styles) HelpModal {
	return HelpModal{styles: styles}
}

// Open shows the modal with the given bindings.
func (m HelpModal) Open(bindings []key.Binding) HelpModal {
	m.active = true
	m.bindings = bindings
	return m
}

// Close hides the modal.
func (m HelpModal) Close() HelpModal {
	m.active = false
	return m
}

// Active returns whether the modal is currently shown.
func (m HelpModal) Active() bool {
	return m.active
}

// SetSize updates the modal dimensions for centering.
func (m HelpModal) SetSize(width, height int) HelpModal {
	m.width = width
	m.height = height
	return m
}

// Update handles key events. Returns the updated modal and whether it was closed.
func (m HelpModal) Update(msg tea.Msg) (HelpModal, bool) {
	if !m.active {
		return m, false
	}

	if msg, ok := msg.(tea.KeyMsg); ok {
		switch {
		case msg.Type == tea.KeyEscape, key.Matches(msg, keys.Help):
			m = m.Close()
			return m, true
		}
	}

	return m, false
}

// View renders the modal.
func (m HelpModal) View() string {
	if !m.active {
		return ""
	}

	titleLine := m.styles.ModalTitle.Render("Key Bindings")

	keyStyle := m.styles.Theme.Accent.Width(12)
	descStyle := m.styles.Theme.Muted

	var rows string
	for _, b := range m.bindings {
		h := b.Help()
		row := keyStyle.Render(h.Key) + descStyle.Render(h.Desc)
		if rows == "" {
			rows = row
		} else {
			rows = rows + "\n" + row
		}
	}

	content := lipgloss.JoinVertical(lipgloss.Left, titleLine, "", rows)
	modal := m.styles.ModalBorder.Render(content)

	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		modal,
	)
}
