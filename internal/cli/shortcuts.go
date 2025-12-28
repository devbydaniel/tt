package cli

import (
	"errors"
	"os"
	"strconv"

	"github.com/devbydaniel/tt/internal/output"
	"github.com/spf13/cobra"
)

func NewInboxCmd(deps *Dependencies) *cobra.Command {
	var group string
	var sortStr string
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "inbox",
		Short: "List tasks with no project, area, or dates",
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunListView(deps, "inbox", sortStr, group, jsonOutput)
		},
	}

	cmd.Flags().StringVarP(&group, "group", "g", "", "Group tasks by: project, area, date, none")
	cmd.Flags().StringVarP(&sortStr, "sort", "s", "", "Sort by field(s): id, title, planned, due, created, project, area")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	return cmd
}

func NewTodayCmd(deps *Dependencies) *cobra.Command {
	var group string
	var sortStr string
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "today",
		Short: "List tasks planned for today or overdue",
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunListView(deps, "today", sortStr, group, jsonOutput)
		},
	}

	cmd.Flags().StringVarP(&group, "group", "g", "", "Group tasks by: project, area, date, none")
	cmd.Flags().StringVarP(&sortStr, "sort", "s", "", "Sort by field(s): id, title, planned, due, created, project, area")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	return cmd
}

func NewUpcomingCmd(deps *Dependencies) *cobra.Command {
	var group string
	var sortStr string
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "upcoming",
		Short: "List tasks with future dates",
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunListView(deps, "upcoming", sortStr, group, jsonOutput)
		},
	}

	cmd.Flags().StringVarP(&group, "group", "g", "", "Group tasks by: project, area, date, none")
	cmd.Flags().StringVarP(&sortStr, "sort", "s", "", "Sort by field(s): id, title, planned, due, created, project, area")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	return cmd
}

func NewAnytimeCmd(deps *Dependencies) *cobra.Command {
	var group string
	var sortStr string
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "anytime",
		Short: "List active tasks with no specific dates",
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunListView(deps, "anytime", sortStr, group, jsonOutput)
		},
	}

	cmd.Flags().StringVarP(&group, "group", "g", "", "Group tasks by: project, area, date, none")
	cmd.Flags().StringVarP(&sortStr, "sort", "s", "", "Sort by field(s): id, title, planned, due, created, project, area")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	return cmd
}

func NewSomedayCmd(deps *Dependencies) *cobra.Command {
	var group string
	var sortStr string
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "someday",
		Short: "List tasks deferred to someday",
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunListView(deps, "someday", sortStr, group, jsonOutput)
		},
	}

	cmd.Flags().StringVarP(&group, "group", "g", "", "Group tasks by: project, area, date, none")
	cmd.Flags().StringVarP(&sortStr, "sort", "s", "", "Sort by field(s): id, title, planned, due, created, project, area")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	return cmd
}

func NewRenameCmd(deps *Dependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "rename <task-id> <new-title>",
		Short: "Rename a task (shortcut for edit --title)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return errors.New("invalid task ID: " + args[0])
			}

			if _, err := deps.TaskService.SetTitle(id, args[1]); err != nil {
				return err
			}

			formatter := output.NewFormatter(os.Stdout, deps.Theme)
			formatter.TaskEdited(id, []string{"title"})
			return nil
		},
	}
}
