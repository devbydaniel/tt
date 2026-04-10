package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Card renders a bordered box with a title and content
type Card struct {
	styles *Styles
}

// NewCard creates a new card renderer
func NewCard(styles *Styles) *Card {
	return &Card{styles: styles}
}

// Render creates a bordered card with title and content at fixed dimensions.
// If tabs is non-empty, the header shows heading left-aligned and tabs right-aligned.
func (c *Card) Render(heading, tabs, content string, width, height int, focused bool) string {
	// Select border style based on focus
	borderStyle := c.styles.UnfocusedSection
	if focused {
		borderStyle = c.styles.FocusedSection
	}

	// Render header
	var header string
	if tabs == "" {
		header = c.styles.Theme.Header.Bold(true).Render(heading)
	} else {
		innerWidth := width - 4 // border(2) + padding(2)
		styledHeading := c.styles.Theme.Header.Bold(true).Render(heading)
		headingW := lipgloss.Width(styledHeading)
		tabsW := lipgloss.Width(tabs)
		gap := innerWidth - headingW - tabsW
		if gap < 1 {
			gap = 1
		}
		header = styledHeading + strings.Repeat(" ", gap) + tabs
	}

	// Combine header and content with blank line between
	innerContent := header + "\n\n" + content

	// Inner dimensions accounting for border(2) and padding(2)
	innerWidth := width - 4
	innerHeight := height - 2

	if innerWidth < 1 {
		innerWidth = 1
	}
	if innerHeight < 1 {
		innerHeight = 1
	}

	// Place content at top-left within fixed-size container
	placed := lipgloss.Place(innerWidth, innerHeight, lipgloss.Left, lipgloss.Top, innerContent)

	// Apply padding and set fixed width to enforce box dimensions
	// MaxWidth truncates lines that are too long (handles ANSI codes properly)
	padded := lipgloss.NewStyle().
		Padding(0, 1).
		MaxWidth(width - 2). // Total width minus border
		Render(placed)

	return borderStyle.Render(padded)
}
