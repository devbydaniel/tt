package tui

import (
	"fmt"
	"strings"

	"github.com/devbydaniel/tt/internal/domain/task"
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
	detailFieldCount // sentinel for wrapping
)

// DetailPane displays task details in a third column
type DetailPane struct {
	task         *task.Task
	focusedField DetailField
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
	d.focusedField = DetailFieldTitle
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
	d.focusedField = (d.focusedField + 1) % detailFieldCount
	return d
}

// PrevField moves to the previous field
func (d DetailPane) PrevField() DetailPane {
	d.focusedField = (d.focusedField - 1 + detailFieldCount) % detailFieldCount
	return d
}

// View renders the detail pane
func (d DetailPane) View() string {
	if d.task == nil {
		return ""
	}

	content := d.buildContent()
	return d.card.Render("Details", content, d.width, d.height, d.focused)
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

// HasTask returns true if a task is set
func (d DetailPane) HasTask() bool {
	return d.task != nil
}

// UpdateTask updates the task data (e.g., after an edit)
func (d DetailPane) UpdateTask(t *task.Task) DetailPane {
	d.task = t
	return d
}
