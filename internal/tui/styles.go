package tui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/devbydaniel/tt/internal/output"
)

// Styles holds TUI-specific styles extending the base theme
type Styles struct {
	Theme *output.Theme

	// Sidebar styles
	SidebarBorder   lipgloss.Style
	SectionHeader   lipgloss.Style
	SelectedItem    lipgloss.Style
	UnselectedItem  lipgloss.Style
	FocusedSection  lipgloss.Style
	UnfocusedSection lipgloss.Style

	// Content styles
	ContentBorder lipgloss.Style
	ContentHeader lipgloss.Style
	TaskRow       lipgloss.Style

	// Modal styles
	ModalBorder lipgloss.Style
	ModalTitle  lipgloss.Style
}

// NewStyles creates TUI styles from the base theme
func NewStyles(theme *output.Theme) *Styles {
	if theme == nil {
		theme = output.DefaultTheme()
	}

	// Get accent color for focused elements
	accentColor := theme.Accent.GetForeground()
	mutedColor := theme.Muted.GetForeground()
	headerColor := theme.Header.GetForeground()

	return &Styles{
		Theme: theme,

		SidebarBorder: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(mutedColor),

		SectionHeader: lipgloss.NewStyle().
			Bold(true).
			Foreground(headerColor).
			MarginBottom(1),

		SelectedItem: lipgloss.NewStyle().
			Bold(true).
			Foreground(accentColor),

		UnselectedItem: lipgloss.NewStyle(),

		FocusedSection: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(accentColor),

		UnfocusedSection: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(mutedColor),

		ContentBorder: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(mutedColor),

		ContentHeader: lipgloss.NewStyle().
			Bold(true).
			Foreground(headerColor).
			MarginBottom(1),

		TaskRow: lipgloss.NewStyle(),

		ModalBorder: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accentColor).
			Padding(1, 2),

		ModalTitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(headerColor).
			MarginBottom(1),
	}
}
