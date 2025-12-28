package cli

import (
	"os"

	"github.com/devbydaniel/tt/config"
	"github.com/devbydaniel/tt/internal/domain/area"
	"github.com/devbydaniel/tt/internal/domain/project"
	"github.com/devbydaniel/tt/internal/domain/task"
	"github.com/devbydaniel/tt/internal/output"
	"github.com/spf13/cobra"
)

// validDefaultLists are the allowed values for default_list config
var validDefaultLists = map[string]bool{
	"today": true, "upcoming": true, "anytime": true,
	"someday": true, "inbox": true, "all": true,
}

type Dependencies struct {
	TaskService    *task.Service
	AreaService    *area.Service
	ProjectService *project.Service
	Config         *config.Config
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
	rootCmd.AddCommand(NewSearchCmd(deps))
	rootCmd.AddCommand(NewCompletionCmd())

	// Shorthand list commands
	rootCmd.AddCommand(NewInboxCmd(deps))
	rootCmd.AddCommand(NewTodayCmd(deps))
	rootCmd.AddCommand(NewUpcomingCmd(deps))
	rootCmd.AddCommand(NewAnytimeCmd(deps))
	rootCmd.AddCommand(NewSomedayCmd(deps))

	return rootCmd
}

func runList(deps *Dependencies) error {
	// Build options based on default_list config (defaults to "today")
	defaultList := deps.Config.DefaultList
	if !validDefaultLists[defaultList] {
		defaultList = "today"
	}

	opts := &task.ListOptions{}
	switch defaultList {
	case "today":
		opts.Today = true
	case "upcoming":
		opts.Upcoming = true
	case "anytime":
		opts.Anytime = true
	case "someday":
		opts.Someday = true
	case "inbox":
		opts.Inbox = true
	case "all":
		opts.All = true
	}

	tasks, err := deps.TaskService.List(opts)
	if err != nil {
		return err
	}

	groupBy := deps.Config.Grouping.GetForCommand(defaultList)

	formatter := output.NewFormatter(os.Stdout)
	formatter.GroupedTaskList(tasks, groupBy)
	return nil
}
