package cli

import (
	"os"

	"github.com/devbydaniel/tt/internal/domain/task"
	"github.com/devbydaniel/tt/internal/output"
	"github.com/spf13/cobra"
)

func NewSearchCmd(deps *Dependencies) *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:     "search <query>",
		Aliases: []string{"s"},
		Short:   "Search tasks by title",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := args[0]

			opts := &task.ListOptions{
				Search: query,
				// No schedule filter = search across all tasks
			}

			tasks, err := deps.App.ListTasks.Execute(opts)
			if err != nil {
				return err
			}

			if jsonOutput {
				return output.WriteJSON(os.Stdout, tasks)
			}

			formatter := output.NewFormatter(os.Stdout, deps.Theme)
			formatter.TaskList(tasks)
			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	return cmd
}
