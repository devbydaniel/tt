package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/devbydaniel/tt/internal/domain/area"
	"github.com/devbydaniel/tt/internal/domain/project"
	"github.com/devbydaniel/tt/internal/domain/task"
	"github.com/devbydaniel/tt/internal/output"
)

// Run starts the TUI application
func Run(taskService *task.Service, areaService *area.Service, projectService *project.Service, theme *output.Theme) error {
	model := NewModel(taskService, areaService, projectService, theme)
	p := tea.NewProgram(model, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
