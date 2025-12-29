package tui

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/devbydaniel/tt/internal/dateparse"
	datepicker "github.com/ethanefung/bubble-datepicker"
)

// DateModalMode indicates whether we're setting planned or due date
type DateModalMode int

const (
	DateModalPlanned DateModalMode = iota
	DateModalDue
)

// DateModal handles setting planned or due dates for a task
type DateModal struct {
	input      textinput.Model
	datepicker datepicker.Model
	mode       DateModalMode
	taskID     int64
	active     bool
	focusInput bool // true = input focused, false = picker focused
	err        error
	styles     *Styles
	width      int
	height     int
}

// DateResult represents the outcome of the date modal
type DateResult struct {
	TaskID   int64
	Date     *time.Time
	Mode     DateModalMode
	Canceled bool
}

// NewDateModal creates a new date modal
func NewDateModal(styles *Styles) DateModal {
	ti := textinput.New()
	ti.Placeholder = "today, tomorrow, +3d, monday, 2025-01-15"
	ti.CharLimit = 30

	dp := datepicker.New(time.Now())
	dp.Styles = datepicker.DefaultStyles()

	return DateModal{
		input:      ti,
		datepicker: dp,
		focusInput: true,
		styles:     styles,
	}
}

// Open shows the modal for the given task and mode
func (m DateModal) Open(taskID int64, mode DateModalMode, currentDate *time.Time) DateModal {
	m.active = true
	m.taskID = taskID
	m.mode = mode
	m.err = nil
	m.focusInput = true

	// Set initial date
	initialDate := time.Now()
	if currentDate != nil {
		initialDate = *currentDate
		m.input.SetValue(currentDate.Format("2006-01-02"))
	} else {
		m.input.SetValue("")
	}

	m.datepicker.SetTime(initialDate)
	m.datepicker.SelectDate()
	m.input.Focus()

	return m
}

// Close hides the modal
func (m DateModal) Close() DateModal {
	m.active = false
	m.input.Blur()
	m.datepicker.Blur()
	return m
}

// SetSize updates the modal dimensions
func (m DateModal) SetSize(width, height int) DateModal {
	m.width = width
	m.height = height
	m.input.Width = 30
	return m
}

// Update handles input events
func (m DateModal) Update(msg tea.Msg) (DateModal, *DateResult) {
	if !m.active {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEscape:
			m = m.Close()
			return m, &DateResult{Canceled: true}

		case tea.KeyTab:
			m.focusInput = !m.focusInput
			if m.focusInput {
				m.input.Focus()
				m.datepicker.Blur()
			} else {
				m.input.Blur()
				m.datepicker.SetFocus(datepicker.FocusCalendar)
			}
			return m, nil

		case tea.KeyEnter:
			if m.focusInput {
				// Parse input and submit
				value := strings.TrimSpace(m.input.Value())
				if value == "" {
					// Empty = clear date
					m = m.Close()
					return m, &DateResult{
						TaskID: m.taskID,
						Date:   nil,
						Mode:   m.mode,
					}
				}

				parsed, err := dateparse.Parse(value)
				if err != nil {
					m.err = err
					return m, nil
				}

				m = m.Close()
				return m, &DateResult{
					TaskID: m.taskID,
					Date:   &parsed,
					Mode:   m.mode,
				}
			} else {
				// Picker focused - submit with picker's date
				selectedDate := m.datepicker.Time
				m = m.Close()
				return m, &DateResult{
					TaskID: m.taskID,
					Date:   &selectedDate,
					Mode:   m.mode,
				}
			}
		}

		// Handle other keys based on focus
		if m.focusInput {
			// Pass to text input
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			_ = cmd
			m.err = nil // Clear error on typing
		} else {
			// Pass to datepicker
			var cmd tea.Cmd
			m.datepicker, cmd = m.datepicker.Update(msg)
			_ = cmd
		}
	}

	return m, nil
}

// View renders the modal
func (m DateModal) View() string {
	if !m.active {
		return ""
	}

	// Title
	titleText := "Set Planned Date"
	if m.mode == DateModalDue {
		titleText = "Set Due Date"
	}
	title := m.styles.ModalTitle.Render(titleText)

	// Input field with focus indicator
	var input string
	if m.focusInput {
		input = "> " + m.input.View()
	} else {
		input = "  " + m.input.View()
	}

	// Error message
	var errView string
	if m.err != nil {
		errView = m.styles.Theme.Error.Render(m.err.Error())
	}

	// Datepicker with focus indicator
	var picker string
	if !m.focusInput {
		picker = m.styles.SelectedItem.Render(m.datepicker.View())
	} else {
		picker = m.datepicker.View()
	}

	// Build content
	var parts []string
	parts = append(parts, title, "", input)
	if errView != "" {
		parts = append(parts, errView)
	}
	parts = append(parts, "", picker)

	content := lipgloss.JoinVertical(lipgloss.Left, parts...)
	modal := m.styles.ModalBorder.Render(content)

	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		modal,
	)
}

// Active returns whether the modal is currently shown
func (m DateModal) Active() bool {
	return m.active
}

// FocusInput returns whether the input field is focused (vs the picker)
func (m DateModal) FocusInput() bool {
	return m.focusInput
}
