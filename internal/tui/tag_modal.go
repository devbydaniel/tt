package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sahilm/fuzzy"
)

// TagItem represents a tag in the modal list
type TagItem struct {
	Name     string
	Selected bool // true if tag is currently on the task
	IsNew    bool // true if this is a "create new" option
}

// TagModal handles editing tags on a task
type TagModal struct {
	input       textinput.Model
	allTags     []string        // all system tags
	taskTags    map[string]bool // current task's tags (for quick lookup)
	filtered    []TagItem       // filtered list for display
	selectedIdx int             // cursor position in filtered list
	taskID      int64
	active      bool
	styles      *Styles
	width       int
	height      int
}

// TagResult represents the outcome of the tag modal
type TagResult struct {
	TaskID   int64
	Tags     []string
	Canceled bool
}

// NewTagModal creates a new tag modal
func NewTagModal(styles *Styles) TagModal {
	ti := textinput.New()
	ti.Placeholder = "Type to filter or add..."
	ti.CharLimit = 100

	return TagModal{
		input:    ti,
		taskTags: make(map[string]bool),
		styles:   styles,
	}
}

// Open shows the modal with task's current tags and all available tags
func (m TagModal) Open(taskID int64, currentTags []string, allTags []string) TagModal {
	m.active = true
	m.taskID = taskID
	m.selectedIdx = 0
	m.allTags = allTags

	// Build map of current tags for quick lookup
	m.taskTags = make(map[string]bool)
	for _, tag := range currentTags {
		m.taskTags[tag] = true
	}

	m.filtered = m.buildTagList()
	m.input.SetValue("")
	m.input.Focus()
	return m
}

// buildTagList creates the list of tags with their selected state
func (m TagModal) buildTagList() []TagItem {
	query := strings.TrimSpace(m.input.Value())
	var items []TagItem

	if query == "" {
		// Show all tags
		for _, tag := range m.allTags {
			items = append(items, TagItem{
				Name:     tag,
				Selected: m.taskTags[tag],
			})
		}
		return items
	}

	// Fuzzy filter
	matches := fuzzy.Find(query, m.allTags)

	// Check if input exactly matches any existing tag
	exactMatch := false
	for _, tag := range m.allTags {
		if strings.EqualFold(tag, query) {
			exactMatch = true
			break
		}
	}

	// Add "create new" option if input doesn't match existing tag exactly
	if !exactMatch && query != "" {
		items = append(items, TagItem{
			Name:     query,
			Selected: m.taskTags[query],
			IsNew:    true,
		})
	}

	// Add matched items
	for _, match := range matches {
		items = append(items, TagItem{
			Name:     m.allTags[match.Index],
			Selected: m.taskTags[m.allTags[match.Index]],
		})
	}

	return items
}

// Close hides the modal
func (m TagModal) Close() TagModal {
	m.active = false
	m.input.Blur()
	return m
}

// SetSize updates the modal dimensions
func (m TagModal) SetSize(width, height int) TagModal {
	m.width = width
	m.height = height
	m.input.Width = 40
	return m
}

// toggleSelected toggles the tag at the current selection
func (m TagModal) toggleSelected() TagModal {
	if m.selectedIdx >= 0 && m.selectedIdx < len(m.filtered) {
		item := m.filtered[m.selectedIdx]
		if m.taskTags[item.Name] {
			delete(m.taskTags, item.Name)
		} else {
			m.taskTags[item.Name] = true
		}
		// Rebuild list to reflect change
		m.filtered = m.buildTagList()
	}
	return m
}

// getSelectedTags returns all currently selected tags as a slice
func (m TagModal) getSelectedTags() []string {
	var tags []string
	for tag := range m.taskTags {
		tags = append(tags, tag)
	}
	return tags
}

// Update handles input events
func (m TagModal) Update(msg tea.Msg) (TagModal, *TagResult) {
	if !m.active {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEscape:
			m = m.Close()
			return m, &TagResult{Canceled: true}

		case tea.KeyEnter:
			// Confirm and return result
			m = m.Close()
			return m, &TagResult{
				TaskID: m.taskID,
				Tags:   m.getSelectedTags(),
			}

		case tea.KeySpace:
			// Toggle selected tag
			m = m.toggleSelected()
			return m, nil

		case tea.KeyUp:
			if m.selectedIdx > 0 {
				m.selectedIdx--
			}
			return m, nil

		case tea.KeyDown:
			if m.selectedIdx < len(m.filtered)-1 {
				m.selectedIdx++
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
		m.filtered = m.buildTagList()
		m.selectedIdx = 0
	}

	return m, nil
}

// View renders the modal
func (m TagModal) View() string {
	if !m.active {
		return ""
	}

	title := m.styles.ModalTitle.Render("Edit Tags")
	input := m.input.View()
	list := m.renderList()
	help := m.styles.Theme.Muted.Render("space: toggle  enter: confirm  esc: cancel")

	content := lipgloss.JoinVertical(lipgloss.Left, title, input, "", list, "", help)
	modal := m.styles.ModalBorder.Render(content)

	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		modal,
	)
}

// renderList renders the filtered list of tags
func (m TagModal) renderList() string {
	const maxVisible = 10

	if len(m.filtered) == 0 {
		return m.styles.Theme.Muted.Render("No tags. Type to create one.")
	}

	// Calculate offset for scrolling
	offset := 0
	if m.selectedIdx >= maxVisible {
		offset = m.selectedIdx - maxVisible + 1
	}

	lines := make([]string, 0, maxVisible)
	for i := 0; i < maxVisible; i++ {
		idx := offset + i
		if idx >= len(m.filtered) {
			break
		}

		item := m.filtered[idx]

		// Build checkbox indicator
		checkbox := "[ ]"
		if item.Selected {
			checkbox = "[x]"
		}

		// Build label
		var label string
		if item.IsNew {
			label = checkbox + " #" + item.Name + " (new)"
		} else {
			label = checkbox + " #" + item.Name
		}

		// Apply selection styling
		if idx == m.selectedIdx {
			lines = append(lines, m.styles.SelectedItem.Render("> "+label))
		} else {
			lines = append(lines, "  "+label)
		}
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// Active returns whether the modal is currently shown
func (m TagModal) Active() bool {
	return m.active
}
