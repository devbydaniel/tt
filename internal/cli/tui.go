package cli

import (
	"github.com/devbydaniel/tt/internal/tui"
	"github.com/spf13/cobra"
)

func NewTUICmd(deps *Dependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "ui",
		Short: "Open interactive TUI",
		RunE: func(cmd *cobra.Command, args []string) error {
			return tui.Run(deps.App, deps.Theme, deps.Config)
		},
	}
}
