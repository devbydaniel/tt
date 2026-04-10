package tui

import (
	"fmt"
	"strings"

	"github.com/devbydaniel/tt/internal/domain/note"
	"github.com/devbydaniel/tt/internal/domain/task"
)

// DetailViewMode controls which view is shown in the detail pane
type DetailViewMode int

const (
	DetailViewData  DetailViewMode = iota // task fields
	DetailViewNotes                       // notes list
	detailViewCount                       // sentinel for wrapping
)

// DetailField represents which field is currently focused in the detail pane
type DetailField int

const (
	DetailFieldTitle DetailField = iota
	DetailFieldDescription
	DetailFieldScope
	DetailFieldPlanned
	DetailFieldDue
	DetailFieldTags
	detailFieldCount // sentinel for wrapping (comments removed from data view)
)

// DetailPane displays task details in a third column
type DetailPane struct {
	task         *task.Task
	notes        []note.Note
	selectedNote int
	focusedField DetailField
	viewMode     DetailViewMode
	width        int
	height       int
	focused      bool
	styles       *Styles
	card         *Card
}

// NewDetailPane creates a new detail pane
func NewDetailPane(styles *Styles) DetailPane {
	return DetailPane{
		styles:       styles,
		card:         NewCard(styles),
		focusedField: DetailFieldTitle,
	}
}

// SetSize updates detail pane dimensions
func (d DetailPane) SetSize(width, height int) DetailPane {
	d.width = width
	d.height = height
	return d
}

// SetTask sets the task to display
func (d DetailPane) SetTask(t *task.Task) DetailPane {
	d.task = t
	d.notes = nil
	d.selectedNote = 0
	d.focusedField = DetailFieldTitle
	d.viewMode = DetailViewData
	return d
}

// ViewMode returns the current view mode
func (d DetailPane) ViewMode() DetailViewMode {
	return d.viewMode
}

// NextViewMode cycles forward through view modes (Data → Comments → Notes → Data)
func (d DetailPane) NextViewMode() DetailPane {
	d.viewMode = (d.viewMode + 1) % detailViewCount
	return d
}

// PrevViewMode cycles backward through view modes (Data → Notes → Comments → Data)
func (d DetailPane) PrevViewMode() DetailPane {
	d.viewMode = (d.viewMode - 1 + detailViewCount) % detailViewCount
	return d
}

// SetNotes sets the notes to display
func (d DetailPane) SetNotes(notes []note.Note) DetailPane {
	d.notes = notes
	d.selectedNote = 0
	return d
}

// NextNote moves to the next note in the list
func (d DetailPane) NextNote() DetailPane {
	if len(d.notes) > 0 {
		d.selectedNote = (d.selectedNote + 1) % len(d.notes)
	}
	return d
}

// PrevNote moves to the previous note in the list
func (d DetailPane) PrevNote() DetailPane {
	if len(d.notes) > 0 {
		d.selectedNote = (d.selectedNote - 1 + len(d.notes)) % len(d.notes)
	}
	return d
}

// SelectedNote returns the currently selected note, or nil if none
func (d DetailPane) SelectedNote() *note.Note {
	if len(d.notes) == 0 {
		return nil
	}
	return &d.notes[d.selectedNote]
}

// SetFocused sets whether the detail pane has focus
func (d DetailPane) SetFocused(focused bool) DetailPane {
	d.focused = focused
	return d
}

// Focused returns whether the detail pane has focus
func (d DetailPane) Focused() bool {
	return d.focused
}

// Task returns the currently displayed task
func (d DetailPane) Task() *task.Task {
	return d.task
}

// FocusedField returns the currently focused field
func (d DetailPane) FocusedField() DetailField {
	return d.focusedField
}

// NextField moves to the next field
func (d DetailPane) NextField() DetailPane {
	if d.viewMode != DetailViewData {
		return d // no field navigation outside data view
	}
	d.focusedField = (d.focusedField + 1) % detailFieldCount
	return d
}

// PrevField moves to the previous field
func (d DetailPane) PrevField() DetailPane {
	if d.viewMode != DetailViewData {
		return d // no field navigation outside data view
	}
	d.focusedField = (d.focusedField - 1 + detailFieldCount) % detailFieldCount
	return d
}

// View renders the detail pane
func (d DetailPane) View() string {
	if d.task == nil {
		return ""
	}

	// View indicator in the title
	theme := d.styles.Theme
	labels := [2]string{"Data", "Notes"}
	var parts [2]string
	for i, label := range labels {
		if DetailViewMode(i) == d.viewMode {
			parts[i] = theme.Accent.Render(label)
		} else {
			parts[i] = theme.Muted.Render(label)
		}
	}
	sep := theme.Muted.Render(" · ")
	title := parts[0] + sep + parts[1]

	var content string
	switch d.viewMode {
	case DetailViewNotes:
		content = d.buildNotesView()
	default:
		content = d.buildContent()
	}

	return d.card.Render(title, content, d.width, d.height, d.focused)
}

// buildContent builds the detail pane content
func (d DetailPane) buildContent() string {
	if d.task == nil {
		return ""
	}

	theme := d.styles.Theme
	var sections []string

	// Title
	sections = append(sections, d.renderField(DetailFieldTitle, "Title", d.task.Title))

	// Description
	desc := "None"
	if d.task.Description != nil && *d.task.Description != "" {
		desc = *d.task.Description
	}
	sections = append(sections, d.renderField(DetailFieldDescription, "Description", desc))

	// Scope (Area > Project)
	scope := "None"
	if d.task.ParentName != nil {
		if d.task.AreaName != nil {
			scope = *d.task.AreaName + " > " + *d.task.ParentName
		} else {
			scope = *d.task.ParentName
		}
	} else if d.task.AreaName != nil {
		scope = *d.task.AreaName
	}
	sections = append(sections, d.renderField(DetailFieldScope, "Scope", scope))

	// Planned date
	planned := "None"
	if d.task.PlannedDate != nil {
		planned = theme.Icons.Date + " " + d.task.PlannedDate.Format("Jan 2, 2006")
	}
	sections = append(sections, d.renderField(DetailFieldPlanned, "Planned", planned))

	// Due date
	due := "None"
	if d.task.DueDate != nil {
		due = theme.Icons.Due + " " + d.task.DueDate.Format("Jan 2, 2006")
	}
	sections = append(sections, d.renderField(DetailFieldDue, "Due", due))

	// Tags
	tags := "None"
	if len(d.task.Tags) > 0 {
		var tagStrs []string
		for _, tag := range d.task.Tags {
			tagStrs = append(tagStrs, "#"+tag)
		}
		tags = strings.Join(tagStrs, " ")
	}
	sections = append(sections, d.renderField(DetailFieldTags, "Tags", tags))

	return strings.Join(sections, "\n\n")
}

// renderField renders a single field with label and value
func (d DetailPane) renderField(field DetailField, label, value string) string {
	theme := d.styles.Theme
	isSelected := d.focused && d.focusedField == field

	// Label
	labelStyle := theme.Muted
	if isSelected {
		labelStyle = theme.Accent
	}
	labelStr := labelStyle.Render(label)

	// Value (may be multiline for description)
	valueStr := value
	if value == "None" {
		valueStr = theme.Muted.Render(value)
	}

	// Truncate long values to fit width (leaving room for padding)
	maxWidth := d.width - 6
	if maxWidth < 10 {
		maxWidth = 10
	}

	// For description, handle multiline
	if field == DetailFieldDescription && value != "None" {
		lines := strings.Split(value, "\n")
		var truncated []string
		for i, line := range lines {
			if i >= 3 { // Show max 3 lines
				truncated = append(truncated, theme.Muted.Render("..."))
				break
			}
			if len(line) > maxWidth {
				line = line[:maxWidth-3] + "..."
			}
			truncated = append(truncated, line)
		}
		// Indent continuation lines
		valueStr = strings.Join(truncated, "\n    ")
	} else if len(valueStr) > maxWidth && value != "None" {
		valueStr = valueStr[:maxWidth-3] + "..."
	}

	// Selection indicator - only on label line
	prefix := "  "
	if isSelected {
		prefix = d.styles.SelectedItem.Render("> ")
	}

	// Combine label and value (value always indented with spaces)
	content := fmt.Sprintf("%s%s\n    %s", prefix, labelStr, valueStr)

	return content
}

// buildNotesView renders the notes list with selection indicator
func (d DetailPane) buildNotesView() string {
	theme := d.styles.Theme

	if len(d.notes) == 0 {
		return theme.Muted.Render("  No notes yet.")
	}

	maxWidth := d.width - 8
	if maxWidth < 10 {
		maxWidth = 10
	}

	var lines []string
	for i, n := range d.notes {
		prefix := "  "
		if d.focused && i == d.selectedNote {
			prefix = d.styles.SelectedItem.Render("> ")
		}
		date := n.Date.Format("Jan 2, 2006")
		entry := date + "  " + n.Title
		if len(entry) > maxWidth {
			entry = entry[:maxWidth-3] + "..."
		}
		if d.focused && i == d.selectedNote {
			lines = append(lines, prefix+theme.Accent.Render(entry))
		} else {
			lines = append(lines, prefix+entry)
		}
	}

	return strings.Join(lines, "\n")
}

// HasTask returns true if a task is set
func (d DetailPane) HasTask() bool {
	return d.task != nil
}

// UpdateTask updates the task data (e.g., after an edit)
func (d DetailPane) UpdateTask(t *task.Task) DetailPane {
	d.task = t
	return d
}
