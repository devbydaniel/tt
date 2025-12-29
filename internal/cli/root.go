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
	Theme          *output.Theme
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

	// Shorthand task commands
	rootCmd.AddCommand(NewRenameCmd(deps))

	// Interactive TUI
	rootCmd.AddCommand(NewTUICmd(deps))

	return rootCmd
}

func runList(deps *Dependencies) error {
	// Build options based on default_list config (defaults to "today")
	defaultList := deps.Config.DefaultList
	if !validDefaultLists[defaultList] {
		defaultList = "today"
	}

	return RunListView(deps, defaultList, "", "", false)
}

// RunListView runs a list view with the given view name, optional sort and group overrides.
// This is used by all shortcut commands (today, upcoming, etc.) and the list command.
func RunListView(deps *Dependencies, viewCmd, sortOverride, groupOverride string, jsonOutput bool) error {
	// Build list options based on view command
	opts := &task.ListOptions{}
	switch viewCmd {
	case "today":
		opts.Schedule = "today"
	case "upcoming":
		opts.Schedule = "upcoming"
	case "anytime":
		opts.Schedule = "anytime"
	case "someday":
		opts.Schedule = "someday"
	case "inbox":
		opts.Schedule = "inbox"
	case "all":
		// no schedule filter
	}

	// Resolve sorting: override > config > code default
	sortToUse := sortOverride
	if sortToUse == "" {
		sortToUse = deps.Config.GetSort(viewCmd)
	}
	sortOpts, err := task.ParseSort(sortToUse)
	if err != nil {
		return err
	}
	opts.Sort = sortOpts

	tasks, err := deps.TaskService.List(opts)
	if err != nil {
		return err
	}

	if jsonOutput {
		return output.WriteJSON(os.Stdout, tasks)
	}

	// Resolve grouping: override > config > none
	groupBy := groupOverride
	if groupBy == "" {
		groupBy = deps.Config.GetGroup(viewCmd)
	}

	formatter := output.NewFormatter(os.Stdout, deps.Theme)
	if viewCmd == "today" {
		formatter.SetHidePlannedDate(true)
	}
	formatter.GroupedTaskList(tasks, groupBy)
	return nil
}
