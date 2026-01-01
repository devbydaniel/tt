package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/devbydaniel/tt/internal/domain/area"
	"github.com/devbydaniel/tt/internal/domain/task"
	"github.com/sahilm/fuzzy"
)

// MoveItem represents a selectable destination
type MoveItem struct {
	Type  string // "project" or "area"
	Name  string // actual name for the service call
	Label string // display text for filtering and rendering
}

// MoveModal handles moving a task to a project or area with filtering
type MoveModal struct {
	input    textinput.Model
	allItems []MoveItem
	filtered []MoveItem
	selected int
	taskID   int64
	active   bool
	styles   *Styles
	width    int
	height   int
}

// MoveResult represents the outcome of the move modal
type MoveResult struct {
	TaskID   int64
	ItemType string // "project" or "area"
	Name     string
	Canceled bool
}

// NewMoveModal creates a new move modal
func NewMoveModal(styles *Styles) MoveModal {
	ti := textinput.New()
	ti.Placeholder = "Type to filter..."
	ti.CharLimit = 100

	return MoveModal{
		input:  ti,
		styles: styles,
	}
}

// Open shows the modal with available destinations
func (m MoveModal) Open(taskID int64, projects []task.Task, areas []area.Area) MoveModal {
	m.active = true
	m.taskID = taskID
	m.selected = 0
	m.allItems = m.buildItems(projects, areas)
	m.filtered = m.allItems
	m.input.SetValue("")
	m.input.Focus()
	return m
}

// OpenForProject shows the modal with only areas (projects can't be nested under other projects)
func (m MoveModal) OpenForProject(taskID int64, areas []area.Area) MoveModal {
	m.active = true
	m.taskID = taskID
	m.selected = 0
	var items []MoveItem
	for _, a := range areas {
		items = append(items, MoveItem{
			Type:  "area",
			Name:  a.Name,
			Label: a.Name,
		})
	}
	m.allItems = items
	m.filtered = items
	m.input.SetValue("")
	m.input.Focus()
	return m
}

// buildItems creates the list of selectable destinations
func (m MoveModal) buildItems(projects []task.Task, areas []area.Area) []MoveItem {
	var items []MoveItem

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

// Close hides the modal
func (m MoveModal) Close() MoveModal {
	m.active = false
	m.input.Blur()
	return m
}

// SetSize updates the modal dimensions
func (m MoveModal) SetSize(width, height int) MoveModal {
	m.width = width
	m.height = height
	m.input.Width = 40
	return m
}

// filterItems filters items based on the current input using fuzzy matching
func (m MoveModal) filterItems() []MoveItem {
	query := strings.TrimSpace(m.input.Value())
	if query == "" {
		return m.allItems
	}

	// Build list of labels for fuzzy matching
	labels := make([]string, len(m.allItems))
	for i, item := range m.allItems {
		labels[i] = item.Label
	}

	// Fuzzy match
	matches := fuzzy.Find(query, labels)

	// Return matched items in order of match quality
	result := make([]MoveItem, len(matches))
	for i, match := range matches {
		result[i] = m.allItems[match.Index]
	}

	return result
}

// Update handles input events
func (m MoveModal) Update(msg tea.Msg) (MoveModal, *MoveResult) {
	if !m.active {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEscape:
			m = m.Close()
			return m, &MoveResult{Canceled: true}

		case tea.KeyEnter:
			if len(m.filtered) > 0 && m.selected < len(m.filtered) {
				item := m.filtered[m.selected]
				m = m.Close()
				return m, &MoveResult{
					TaskID:   m.taskID,
					ItemType: item.Type,
					Name:     item.Name,
				}
			}
			return m, nil

		case tea.KeyUp:
			if m.selected > 0 {
				m.selected--
			}
			return m, nil

		case tea.KeyDown:
			if m.selected < len(m.filtered)-1 {
				m.selected++
			}
			return m, nil
		}
	}

	// Pass other keys to the text input
	prevValue := m.input.Value()
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	_ = cmd

	// Re-filter if input changed
	if m.input.Value() != prevValue {
		m.filtered = m.filterItems()
		m.selected = 0 // Reset selection when filter changes
	}

	return m, nil
}

// View renders the modal
func (m MoveModal) View() string {
	if !m.active {
		return ""
	}

	title := m.styles.ModalTitle.Render("Move")
	input := m.input.View()
	list := m.renderList()

	content := lipgloss.JoinVertical(lipgloss.Left, title, input, "", list)
	modal := m.styles.ModalBorder.Render(content)

	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		modal,
	)
}

// renderList renders the filtered list of items with fixed height
func (m MoveModal) renderList() string {
	const maxVisible = 10

	// Calculate offset for scrolling
	offset := 0
	if m.selected >= maxVisible {
		offset = m.selected - maxVisible + 1
	}

	lines := make([]string, maxVisible)
	for i := 0; i < maxVisible; i++ {
		idx := offset + i
		if idx < len(m.filtered) {
			item := m.filtered[idx]

			if idx == m.selected {
				lines[i] = m.styles.SelectedItem.Render("> " + item.Label)
			} else {
				lines[i] = "  " + item.Label
			}
		} else if i == 0 && len(m.filtered) == 0 {
			lines[i] = m.styles.Theme.Muted.Render("No matches")
		} else {
			lines[i] = "" // Empty line to maintain height
		}
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// Active returns whether the modal is currently shown
func (m MoveModal) Active() bool {
	return m.active
}
