package tui

import (
	"sort"

	"github.com/charmbracelet/lipgloss"
	"github.com/devbydaniel/tt/internal/domain/area"
	"github.com/devbydaniel/tt/internal/domain/task"
)

// Sidebar contains the left panel with 3 sections
type Sidebar struct {
	sections      []Section
	activeSection int
	width         int
	height        int
	boxHeight     int // height of each individual box
	focused       bool // whether sidebar has focus (vs content area)
	styles        *Styles
	card          *Card
}

// NewSidebar creates a new sidebar
func NewSidebar(styles *Styles) Sidebar {
	return Sidebar{
		sections: []Section{
			NewListsSection(styles),
			NewScopesSection(styles),
			NewTagsSection(styles),
		},
		activeSection: 0,
		focused:       true, // Sidebar starts with focus
		styles:        styles,
		card:          NewCard(styles),
	}
}

// SetFocused sets whether the sidebar has focus
func (s Sidebar) SetFocused(focused bool) Sidebar {
	s.focused = focused
	return s
}

// SetData updates sidebar sections with loaded data
func (s Sidebar) SetData(areas []area.Area, projects []task.Task, tags []string) Sidebar {
	// Update scopes section
	if scopes, ok := s.sections[1].(*ScopesSection); ok {
		s.sections[1] = scopes.SetData(areas, projects)
	}

	// Update tags section
	if tagsSection, ok := s.sections[2].(*TagsSection); ok {
		s.sections[2] = tagsSection.SetData(tags)
	}

	return s
}

// SetSize updates sidebar dimensions
func (s Sidebar) SetSize(width, height int) Sidebar {
	s.width = width
	s.height = height

	// Divide height equally among boxes
	s.boxHeight = height / len(s.sections)
	if s.boxHeight < 5 {
		s.boxHeight = 5
	}

	// Content height: box - border(2) - header with blank line(2)
	contentHeight := s.boxHeight - 4
	if contentHeight < 1 {
		contentHeight = 1
	}

	// Content width: box - border(2) - horizontal padding(2)
	contentWidth := width - 4
	if contentWidth < 1 {
		contentWidth = 1
	}

	for i := range s.sections {
		s.sections[i] = s.sections[i].SetHeight(contentHeight)
		s.sections[i] = s.sections[i].SetWidth(contentWidth)
	}

	return s
}

// NextSection cycles to the next section
func (s Sidebar) NextSection() Sidebar {
	s.sections[s.activeSection] = s.sections[s.activeSection].SetFocused(false)
	s.activeSection = (s.activeSection + 1) % len(s.sections)
	s.sections[s.activeSection] = s.sections[s.activeSection].SetFocused(true)
	return s
}

// PrevSection cycles to the previous section
func (s Sidebar) PrevSection() Sidebar {
	s.sections[s.activeSection] = s.sections[s.activeSection].SetFocused(false)
	s.activeSection = (s.activeSection - 1 + len(s.sections)) % len(s.sections)
	s.sections[s.activeSection] = s.sections[s.activeSection].SetFocused(true)
	return s
}

// MoveUp moves selection up in current section, or jumps to previous section if at first element
func (s Sidebar) MoveUp() Sidebar {
	if s.sections[s.activeSection].AtFirst() {
		// Jump to previous section and select its last item
		s.sections[s.activeSection] = s.sections[s.activeSection].SetFocused(false)
		s.activeSection = (s.activeSection - 1 + len(s.sections)) % len(s.sections)
		s.sections[s.activeSection] = s.sections[s.activeSection].SetFocused(true)
		s.sections[s.activeSection] = s.sections[s.activeSection].SelectLast()
	} else {
		s.sections[s.activeSection] = s.sections[s.activeSection].MoveUp()
	}
	return s
}

// MoveDown moves selection down in current section, or jumps to next section if at last element
func (s Sidebar) MoveDown() Sidebar {
	if s.sections[s.activeSection].AtLast() {
		// Jump to next section and select its first item
		s.sections[s.activeSection] = s.sections[s.activeSection].SetFocused(false)
		s.activeSection = (s.activeSection + 1) % len(s.sections)
		s.sections[s.activeSection] = s.sections[s.activeSection].SetFocused(true)
		s.sections[s.activeSection] = s.sections[s.activeSection].SelectFirst()
	} else {
		s.sections[s.activeSection] = s.sections[s.activeSection].MoveDown()
	}
	return s
}

// SelectedItem returns the currently selected item
func (s Sidebar) SelectedItem() SidebarItem {
	return s.sections[s.activeSection].SelectedItem()
}

// IsScopesSectionActive returns true if the Scopes section is currently active
func (s Sidebar) IsScopesSectionActive() bool {
	return s.activeSection == 1 // Scopes section is index 1
}

// View renders the sidebar as three stacked bordered boxes
func (s Sidebar) View() string {
	headers := []string{"Lists", "Scopes", "Tags"}
	var boxes []string

	for i, section := range s.sections {
		// Only highlight active section when sidebar has focus
		focused := s.focused && i == s.activeSection
		box := s.card.Render(headers[i], section.View(), s.width, s.boxHeight, focused)
		boxes = append(boxes, box)
	}

	return lipgloss.JoinVertical(lipgloss.Left, boxes...)
}

// SidebarItem represents an item in the sidebar
type SidebarItem struct {
	Type  string // "static", "area", "project", "tag"
	Key   string // Filter key (e.g., "today", area name, project name, tag name)
	Label string // Display text
}

// Section interface for sidebar sections
type Section interface {
	View() string
	SelectedItem() SidebarItem
	SetFocused(bool) Section
	SetHeight(int) Section
	SetWidth(int) Section
	MoveUp() Section
	MoveDown() Section
	AtFirst() bool
	AtLast() bool
	SelectFirst() Section
	SelectLast() Section
}

// ListsSection shows static list items (Inbox, Today, etc.)
type ListsSection struct {
	items    []SidebarItem
	selected int
	focused  bool
	height   int
	width    int
	styles   *Styles
}

// NewListsSection creates the static lists section
func NewListsSection(styles *Styles) *ListsSection {
	return &ListsSection{
		items: []SidebarItem{
			{Type: "static", Key: "inbox", Label: "Inbox"},
			{Type: "static", Key: "today", Label: "Today"},
			{Type: "static", Key: "upcoming", Label: "Upcoming"},
			{Type: "static", Key: "anytime", Label: "Anytime"},
			{Type: "static", Key: "someday", Label: "Someday"},
		},
		selected: 1, // Default to Today
		focused:  true,
		styles:   styles,
	}
}

func (s *ListsSection) View() string {
	var lines []string
	for i, item := range s.items {
		line := "  " + item.Label
		if i == s.selected && s.focused {
			line = s.styles.SelectedItem.Render("> " + item.Label)
		}
		lines = append(lines, line)
	}
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (s *ListsSection) SelectedItem() SidebarItem {
	return s.items[s.selected]
}

func (s *ListsSection) SetFocused(focused bool) Section {
	s.focused = focused
	return s
}

func (s *ListsSection) SetHeight(height int) Section {
	s.height = height
	return s
}

func (s *ListsSection) SetWidth(width int) Section {
	s.width = width
	return s
}

func (s *ListsSection) MoveUp() Section {
	if s.selected > 0 {
		s.selected--
	}
	return s
}

func (s *ListsSection) MoveDown() Section {
	if s.selected < len(s.items)-1 {
		s.selected++
	}
	return s
}

func (s *ListsSection) AtFirst() bool {
	return s.selected == 0
}

func (s *ListsSection) AtLast() bool {
	return s.selected >= len(s.items)-1
}

func (s *ListsSection) SelectFirst() Section {
	s.selected = 0
	return s
}

func (s *ListsSection) SelectLast() Section {
	if len(s.items) > 0 {
		s.selected = len(s.items) - 1
	}
	return s
}

// ScopesSection shows areas and projects
type ScopesSection struct {
	items    []SidebarItem
	selected int
	focused  bool
	height   int
	width    int
	offset   int // For scrolling
	styles   *Styles
}

// NewScopesSection creates an empty scopes section
func NewScopesSection(styles *Styles) *ScopesSection {
	return &ScopesSection{
		items:   []SidebarItem{},
		styles:  styles,
	}
}

// SetData populates the scopes section with areas and projects nested under areas
func (s *ScopesSection) SetData(areas []area.Area, projects []task.Task) *ScopesSection {
	var items []SidebarItem

	// Build a map of area name -> projects
	projectsByArea := make(map[string][]task.Task)
	var noAreaProjects []task.Task

	for _, p := range projects {
		if p.AreaName != nil {
			projectsByArea[*p.AreaName] = append(projectsByArea[*p.AreaName], p)
		} else {
			noAreaProjects = append(noAreaProjects, p)
		}
	}

	// Sort areas by name
	sortedAreas := make([]area.Area, len(areas))
	copy(sortedAreas, areas)
	sort.Slice(sortedAreas, func(i, j int) bool {
		return sortedAreas[i].Name < sortedAreas[j].Name
	})

	// Add areas with their projects nested underneath
	for _, a := range sortedAreas {
		items = append(items, SidebarItem{
			Type:  "area",
			Key:   a.Name,
			Label: a.Name,
		})

		// Add projects under this area (indented)
		areaProjects := projectsByArea[a.Name]
		sort.Slice(areaProjects, func(i, j int) bool {
			return areaProjects[i].Title < areaProjects[j].Title
		})
		for _, p := range areaProjects {
			items = append(items, SidebarItem{
				Type:  "project",
				Key:   p.Title,
				Label: "  " + p.Title, // Indent projects
			})
		}
	}

	// Add projects without areas at the end
	sort.Slice(noAreaProjects, func(i, j int) bool {
		return noAreaProjects[i].Title < noAreaProjects[j].Title
	})
	for _, p := range noAreaProjects {
		items = append(items, SidebarItem{
			Type:  "project",
			Key:   p.Title,
			Label: p.Title,
		})
	}

	s.items = items
	return s
}

func (s *ScopesSection) View() string {
	if len(s.items) == 0 {
		return s.styles.Theme.Muted.Render("  No scopes")
	}

	// Calculate visible range
	visibleCount := s.height
	if visibleCount > len(s.items) {
		visibleCount = len(s.items)
	}

	// Adjust offset if selection is out of view
	if s.selected < s.offset {
		s.offset = s.selected
	} else if s.selected >= s.offset+visibleCount {
		s.offset = s.selected - visibleCount + 1
	}

	var lines []string
	for i := s.offset; i < s.offset+visibleCount && i < len(s.items); i++ {
		item := s.items[i]
		line := "  " + item.Label
		if i == s.selected && s.focused {
			line = s.styles.SelectedItem.Render("> " + item.Label)
		}
		lines = append(lines, line)
	}
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (s *ScopesSection) SelectedItem() SidebarItem {
	if len(s.items) == 0 {
		return SidebarItem{Type: "static", Key: "today", Label: "Today"}
	}
	return s.items[s.selected]
}

func (s *ScopesSection) SetFocused(focused bool) Section {
	s.focused = focused
	return s
}

func (s *ScopesSection) SetHeight(height int) Section {
	s.height = height
	return s
}

func (s *ScopesSection) SetWidth(width int) Section {
	s.width = width
	return s
}

func (s *ScopesSection) MoveUp() Section {
	if s.selected > 0 {
		s.selected--
	}
	return s
}

func (s *ScopesSection) MoveDown() Section {
	if s.selected < len(s.items)-1 {
		s.selected++
	}
	return s
}

func (s *ScopesSection) AtFirst() bool {
	return len(s.items) == 0 || s.selected == 0
}

func (s *ScopesSection) AtLast() bool {
	return len(s.items) == 0 || s.selected >= len(s.items)-1
}

func (s *ScopesSection) SelectFirst() Section {
	s.selected = 0
	s.offset = 0
	return s
}

func (s *ScopesSection) SelectLast() Section {
	if len(s.items) > 0 {
		s.selected = len(s.items) - 1
	}
	return s
}

// TagsSection shows tags
type TagsSection struct {
	items    []SidebarItem
	selected int
	focused  bool
	height   int
	width    int
	offset   int
	styles   *Styles
}

// NewTagsSection creates an empty tags section
func NewTagsSection(styles *Styles) *TagsSection {
	return &TagsSection{
		items:  []SidebarItem{},
		styles: styles,
	}
}

// SetData populates the tags section
func (s *TagsSection) SetData(tags []string) *TagsSection {
	var items []SidebarItem
	for _, tag := range tags {
		items = append(items, SidebarItem{
			Type:  "tag",
			Key:   tag,
			Label: "#" + tag,
		})
	}
	s.items = items
	return s
}

func (s *TagsSection) View() string {
	if len(s.items) == 0 {
		return s.styles.Theme.Muted.Render("  No tags")
	}

	visibleCount := s.height
	if visibleCount > len(s.items) {
		visibleCount = len(s.items)
	}

	if s.selected < s.offset {
		s.offset = s.selected
	} else if s.selected >= s.offset+visibleCount {
		s.offset = s.selected - visibleCount + 1
	}

	var lines []string
	for i := s.offset; i < s.offset+visibleCount && i < len(s.items); i++ {
		item := s.items[i]
		line := "  " + item.Label
		if i == s.selected && s.focused {
			line = s.styles.SelectedItem.Render("> " + item.Label)
		}
		lines = append(lines, line)
	}
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (s *TagsSection) SelectedItem() SidebarItem {
	if len(s.items) == 0 {
		return SidebarItem{Type: "static", Key: "today", Label: "Today"}
	}
	return s.items[s.selected]
}

func (s *TagsSection) SetFocused(focused bool) Section {
	s.focused = focused
	return s
}

func (s *TagsSection) SetHeight(height int) Section {
	s.height = height
	return s
}

func (s *TagsSection) SetWidth(width int) Section {
	s.width = width
	return s
}

func (s *TagsSection) MoveUp() Section {
	if s.selected > 0 {
		s.selected--
	}
	return s
}

func (s *TagsSection) MoveDown() Section {
	if s.selected < len(s.items)-1 {
		s.selected++
	}
	return s
}

func (s *TagsSection) AtFirst() bool {
	return len(s.items) == 0 || s.selected == 0
}

func (s *TagsSection) AtLast() bool {
	return len(s.items) == 0 || s.selected >= len(s.items)-1
}

func (s *TagsSection) SelectFirst() Section {
	s.selected = 0
	s.offset = 0
	return s
}

func (s *TagsSection) SelectLast() Section {
	if len(s.items) > 0 {
		s.selected = len(s.items) - 1
	}
	return s
}
