package tui

import (
	"fmt"
	"strings"

	"github.com/devbydaniel/tt/internal/domain/comment"
	"github.com/devbydaniel/tt/internal/domain/task"
)

// DetailViewMode controls which view is shown in the detail pane
type DetailViewMode int

const (
	DetailViewData     DetailViewMode = iota // task fields
	DetailViewComments                       // full comment list
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
	comments     []comment.Comment
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
	d.comments = nil
	d.focusedField = DetailFieldTitle
	d.viewMode = DetailViewData
	return d
}

// ViewMode returns the current view mode
func (d DetailPane) ViewMode() DetailViewMode {
	return d.viewMode
}

// ToggleViewMode switches between data and comments views
func (d DetailPane) ToggleViewMode() DetailPane {
	if d.viewMode == DetailViewData {
		d.viewMode = DetailViewComments
	} else {
		d.viewMode = DetailViewData
	}
	return d
}

// SetComments sets the comments to display
func (d DetailPane) SetComments(comments []comment.Comment) DetailPane {
	d.comments = comments
	return d
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
	if d.viewMode == DetailViewComments {
		return d // no field navigation in comments view
	}
	d.focusedField = (d.focusedField + 1) % detailFieldCount
	return d
}

// PrevField moves to the previous field
func (d DetailPane) PrevField() DetailPane {
	if d.viewMode == DetailViewComments {
		return d // no field navigation in comments view
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
	var title string
	if d.viewMode == DetailViewData {
		title = theme.Accent.Render("Data") + theme.Muted.Render(" · Comments")
	} else {
		title = theme.Muted.Render("Data · ") + theme.Accent.Render("Comments")
	}

	var content string
	if d.viewMode == DetailViewComments {
		content = d.buildCommentsView()
	} else {
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

// buildCommentsView renders the full comments list using available height
func (d DetailPane) buildCommentsView() string {
	theme := d.styles.Theme

	if len(d.comments) == 0 {
		return theme.Muted.Render("  No comments yet. Press c to add one.")
	}

	bodyMax := d.width - 8
	if bodyMax < 10 {
		bodyMax = 10
	}

	// Build each comment block
	var blocks []string
	for _, c := range d.comments {
		header := "  " + theme.Muted.Render(fmt.Sprintf("%s @ %s", c.Author, c.CreatedAt.Format("Jan 2 15:04")))
		// Wrap/truncate body lines
		bodyLines := strings.Split(c.Body, "\n")
		var rendered []string
		for _, line := range bodyLines {
			if len(line) > bodyMax {
				line = line[:bodyMax-3] + "..."
			}
			rendered = append(rendered, "    "+line)
		}
		blocks = append(blocks, header+"\n"+strings.Join(rendered, "\n"))
	}

	// Calculate available lines (height minus border/padding ~4 lines)
	availableLines := d.height - 4
	if availableLines < 3 {
		availableLines = 3
	}

	// Join all blocks and count lines
	fullContent := strings.Join(blocks, "\n\n")
	allLines := strings.Split(fullContent, "\n")

	if len(allLines) <= availableLines {
		return fullContent
	}

	// Bottom-anchor: show most recent comments that fit
	visibleLines := allLines[len(allLines)-availableLines+1:]
	earlier := fmt.Sprintf("  ... earlier comments above")
	return theme.Muted.Render(earlier) + "\n" + strings.Join(visibleLines, "\n")
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

// HasTask returns true if a task is set
func (d DetailPane) HasTask() bool {
	return d.task != nil
}

// UpdateTask updates the task data (e.g., after an edit)
func (d DetailPane) UpdateTask(t *task.Task) DetailPane {
	d.task = t
	return d
}
