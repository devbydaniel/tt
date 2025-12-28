package cli

import (
	"github.com/spf13/cobra"
)

func NewInboxCmd(deps *Dependencies) *cobra.Command {
	var group string
	var sortStr string

	cmd := &cobra.Command{
		Use:   "inbox",
		Short: "List tasks with no project, area, or dates",
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunListView(deps, "inbox", sortStr, group)
		},
	}

	cmd.Flags().StringVarP(&group, "group", "g", "", "Group tasks by: project, area, date, none")
	cmd.Flags().StringVarP(&sortStr, "sort", "s", "", "Sort by field(s): id, title, planned, due, created, project, area")
	return cmd
}

func NewTodayCmd(deps *Dependencies) *cobra.Command {
	var group string
	var sortStr string

	cmd := &cobra.Command{
		Use:   "today",
		Short: "List tasks planned for today or overdue",
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunListView(deps, "today", sortStr, group)
		},
	}

	cmd.Flags().StringVarP(&group, "group", "g", "", "Group tasks by: project, area, date, none")
	cmd.Flags().StringVarP(&sortStr, "sort", "s", "", "Sort by field(s): id, title, planned, due, created, project, area")
	return cmd
}

func NewUpcomingCmd(deps *Dependencies) *cobra.Command {
	var group string
	var sortStr string

	cmd := &cobra.Command{
		Use:   "upcoming",
		Short: "List tasks with future dates",
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunListView(deps, "upcoming", sortStr, group)
		},
	}

	cmd.Flags().StringVarP(&group, "group", "g", "", "Group tasks by: project, area, date, none")
	cmd.Flags().StringVarP(&sortStr, "sort", "s", "", "Sort by field(s): id, title, planned, due, created, project, area")
	return cmd
}

func NewAnytimeCmd(deps *Dependencies) *cobra.Command {
	var group string
	var sortStr string

	cmd := &cobra.Command{
		Use:   "anytime",
		Short: "List active tasks with no specific dates",
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunListView(deps, "anytime", sortStr, group)
		},
	}

	cmd.Flags().StringVarP(&group, "group", "g", "", "Group tasks by: project, area, date, none")
	cmd.Flags().StringVarP(&sortStr, "sort", "s", "", "Sort by field(s): id, title, planned, due, created, project, area")
	return cmd
}

func NewSomedayCmd(deps *Dependencies) *cobra.Command {
	var group string
	var sortStr string

	cmd := &cobra.Command{
		Use:   "someday",
		Short: "List tasks deferred to someday",
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunListView(deps, "someday", sortStr, group)
		},
	}

	cmd.Flags().StringVarP(&group, "group", "g", "", "Group tasks by: project, area, date, none")
	cmd.Flags().StringVarP(&sortStr, "sort", "s", "", "Sort by field(s): id, title, planned, due, created, project, area")
	return cmd
}
