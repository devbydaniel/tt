package cli

import (
	"os"

	"github.com/devbydaniel/t/internal/domain/area"
	"github.com/devbydaniel/t/internal/domain/project"
	"github.com/devbydaniel/t/internal/domain/task"
	"github.com/devbydaniel/t/internal/output"
	"github.com/spf13/cobra"
)

type Dependencies struct {
	TaskService    *task.Service
	AreaService    *area.Service
	ProjectService *project.Service
}

func NewRootCmd(deps *Dependencies) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "t",
		Short: "A CLI task manager",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(deps)
		},
	}

	rootCmd.AddCommand(NewAddCmd(deps))
	rootCmd.AddCommand(NewListCmd(deps))
	rootCmd.AddCommand(NewDoneCmd(deps))
	rootCmd.AddCommand(NewLogCmd(deps))
	rootCmd.AddCommand(NewAreaCmd(deps))
	rootCmd.AddCommand(NewProjectCmd(deps))

	return rootCmd
}

func runList(deps *Dependencies) error {
	tasks, err := deps.TaskService.List()
	if err != nil {
		return err
	}

	formatter := output.NewFormatter(os.Stdout)
	formatter.TaskList(tasks)
	return nil
}
