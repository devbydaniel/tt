package cli

import (
	"os"

	"github.com/devbydaniel/tt/internal/domain/task"
	"github.com/devbydaniel/tt/internal/output"
	"github.com/spf13/cobra"
)

func NewSearchCmd(deps *Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "search <query>",
		Aliases: []string{"s"},
		Short:   "Search tasks by title",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := args[0]

			opts := &task.ListOptions{
				Search: query,
				All:    true, // search across all active tasks
			}

			tasks, err := deps.TaskService.List(opts)
			if err != nil {
				return err
			}

			formatter := output.NewFormatter(os.Stdout)
			formatter.TaskList(tasks)
			return nil
		},
	}

	return cmd
}
