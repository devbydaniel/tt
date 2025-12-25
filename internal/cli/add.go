package cli

import (
	"errors"
	"os"
	"strings"

	"github.com/devbydaniel/t/internal/output"
	"github.com/spf13/cobra"
)

func NewAddCmd(deps *Dependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "add [title]",
		Short: "Add a new task",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			title := strings.Join(args, " ")
			if title == "" {
				return errors.New("task title cannot be empty")
			}

			task, err := deps.TaskService.Create(title)
			if err != nil {
				return err
			}

			formatter := output.NewFormatter(os.Stdout)
			formatter.TaskCreated(task)
			return nil
		},
	}
}
