package cli

import (
	"os"

	"github.com/devbydaniel/tt/internal/domain/area"
	"github.com/devbydaniel/tt/internal/domain/project"
	"github.com/devbydaniel/tt/internal/domain/task"
	"github.com/devbydaniel/tt/internal/output"
	"github.com/spf13/cobra"
)

type Dependencies struct {
	TaskService    *task.Service
	AreaService    *area.Service
	ProjectService *project.Service
}

func NewRootCmd(deps *Dependencies) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "tt",
		Short: "A CLI task manager",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(deps)
		},
	}

	rootCmd.AddCommand(NewAddCmd(deps))
	rootCmd.AddCommand(NewListCmd(deps))
	rootCmd.AddCommand(NewEditCmd(deps))
	rootCmd.AddCommand(NewDoneCmd(deps))
	rootCmd.AddCommand(NewDeleteCmd(deps))
	rootCmd.AddCommand(NewLogCmd(deps))
	rootCmd.AddCommand(NewAreaCmd(deps))
	rootCmd.AddCommand(NewProjectCmd(deps))
	rootCmd.AddCommand(NewPlanCmd(deps))
	rootCmd.AddCommand(NewDueCmd(deps))
	rootCmd.AddCommand(NewRecurCmd(deps))
	rootCmd.AddCommand(NewTagCmd(deps))

	// Shorthand list commands
	rootCmd.AddCommand(NewInboxCmd(deps))
	rootCmd.AddCommand(NewTodayCmd(deps))
	rootCmd.AddCommand(NewUpcomingCmd(deps))
	rootCmd.AddCommand(NewAnytimeCmd(deps))
	rootCmd.AddCommand(NewSomedayCmd(deps))

	return rootCmd
}

func runList(deps *Dependencies) error {
	tasks, err := deps.TaskService.List(nil)
	if err != nil {
		return err
	}

	formatter := output.NewFormatter(os.Stdout)
	formatter.TaskList(tasks)
	return nil
}
