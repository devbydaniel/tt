package tui

import (
	"strings"

	"github.com/charmbracelet/glamour"
	glamourstyles "github.com/charmbracelet/glamour/styles"

	"github.com/devbydaniel/tt/internal/domain/note"
)

// NotePreviewPane displays a rendered markdown preview of a note
type NotePreviewPane struct {
	note     *note.Note
	rendered string // glamour-rendered content
	rawPath  string // path of currently rendered note (for cache check)
	width    int
	height   int
	scroll   int // viewport scroll offset
	focused  bool
	styles   *Styles
	card     *Card
	isLight  bool // whether to use light glamour style
}

// NewNotePreviewPane creates a new note preview pane
func NewNotePreviewPane(styles *Styles, isLight bool) NotePreviewPane {
	return NotePreviewPane{
		styles:  styles,
		card:    NewCard(styles),
		isLight: isLight,
	}
}

// SetNote sets the note and its rendered markdown content
func (p NotePreviewPane) SetNote(n *note.Note, rawContent string) NotePreviewPane {
	p.note = n
	p.rawPath = n.Path
	p.scroll = 0
	p.rendered = p.renderMarkdown(rawContent)
	return p
}

// SetSize updates dimensions. Re-rendering on resize would require keeping
// the raw markdown around — for now the rendered content is only refreshed
// when SetNote is called.
func (p NotePreviewPane) SetSize(width, height int) NotePreviewPane {
	p.width = width
	p.height = height
	return p
}

// SetFocused sets whether the preview pane has focus
func (p NotePreviewPane) SetFocused(focused bool) NotePreviewPane {
	p.focused = focused
	return p
}

// Focused returns whether the preview pane has focus
func (p NotePreviewPane) Focused() bool {
	return p.focused
}

// ScrollDown scrolls the viewport down
func (p NotePreviewPane) ScrollDown() NotePreviewPane {
	maxScroll := p.maxScroll()
	if p.scroll < maxScroll {
		p.scroll++
	}
	return p
}

// ScrollUp scrolls the viewport up
func (p NotePreviewPane) ScrollUp() NotePreviewPane {
	if p.scroll > 0 {
		p.scroll--
	}
	return p
}

// Note returns the currently displayed note
func (p NotePreviewPane) Note() *note.Note {
	return p.note
}

// View renders the preview pane
func (p NotePreviewPane) View() string {
	if p.note == nil {
		return ""
	}

	title := p.note.Title

	// Apply scroll to rendered content
	content := p.scrolledContent()

	return p.card.Render(title, "", content, p.width, p.height, p.focused)
}

// scrolledContent returns the visible portion of rendered content
func (p NotePreviewPane) scrolledContent() string {
	lines := strings.Split(p.rendered, "\n")
	if p.scroll >= len(lines) {
		return ""
	}
	// Available height = total height - border(2) - header line - blank line
	viewHeight := p.height - 4
	if viewHeight < 1 {
		viewHeight = 1
	}
	end := p.scroll + viewHeight
	if end > len(lines) {
		end = len(lines)
	}
	return strings.Join(lines[p.scroll:end], "\n")
}

// maxScroll returns the maximum scroll offset
func (p NotePreviewPane) maxScroll() int {
	lines := strings.Split(p.rendered, "\n")
	viewHeight := p.height - 4
	if viewHeight < 1 {
		viewHeight = 1
	}
	maxOff := len(lines) - viewHeight
	if maxOff < 0 {
		return 0
	}
	return maxOff
}

// renderMarkdown renders raw markdown through glamour
func (p NotePreviewPane) renderMarkdown(raw string) string {
	wordWrap := p.width - 6 // border(2) + padding(2) + margin(2)
	if wordWrap < 20 {
		wordWrap = 20
	}

	styleConfig := glamourstyles.DarkStyleConfig
	if p.isLight {
		styleConfig = glamourstyles.LightStyleConfig
	}

	renderer, err := glamour.NewTermRenderer(
		glamour.WithStyles(styleConfig),
		glamour.WithWordWrap(wordWrap),
	)
	if err != nil {
		return raw // fallback to raw content
	}

	rendered, err := renderer.Render(raw)
	if err != nil {
		return raw
	}

	// Trim trailing whitespace/newlines
	return strings.TrimRight(rendered, "\n ")
}
