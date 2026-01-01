package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/devbydaniel/tt/config"
	"github.com/devbydaniel/tt/internal/app"
	"github.com/devbydaniel/tt/internal/output"
)

// Run starts the TUI application
func Run(application *app.App, theme *output.Theme, cfg *config.Config) error {
	model := NewModel(application, theme, cfg)
	p := tea.NewProgram(model, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
