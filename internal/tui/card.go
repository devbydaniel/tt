package tui

import "github.com/charmbracelet/lipgloss"

// Card renders a bordered box with a title and content
type Card struct {
	styles *Styles
}

// NewCard creates a new card renderer
func NewCard(styles *Styles) *Card {
	return &Card{styles: styles}
}

// Render creates a bordered card with title and content at fixed dimensions
func (c *Card) Render(title, content string, width, height int, focused bool) string {
	// Select border style based on focus
	borderStyle := c.styles.UnfocusedSection
	if focused {
		borderStyle = c.styles.FocusedSection
	}

	// Render header (use base style without margin to control spacing ourselves)
	header := c.styles.Theme.Header.Bold(true).Render(title)

	// Combine header and content with blank line between
	innerContent := header + "\n\n" + content

	// Inner dimensions: total - border(2) - horizontal padding(2)
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

	// Apply horizontal padding (1 char left/right)
	padded := lipgloss.NewStyle().Padding(0, 1).Render(placed)

	return borderStyle.Render(padded)
}
