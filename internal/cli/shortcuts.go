package cli

import (
	"os"

	"github.com/devbydaniel/tt/internal/domain/task"
	"github.com/devbydaniel/tt/internal/output"
	"github.com/spf13/cobra"
)

func NewInboxCmd(deps *Dependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "inbox",
		Short: "List tasks with no project, area, or dates",
		RunE: func(cmd *cobra.Command, args []string) error {
			tasks, err := deps.TaskService.List(&task.ListOptions{Inbox: true})
			if err != nil {
				return err
			}

			formatter := output.NewFormatter(os.Stdout)
			formatter.TaskList(tasks)
			return nil
		},
	}
}

func NewTodayCmd(deps *Dependencies) *cobra.Command {
	var group string

	cmd := &cobra.Command{
		Use:   "today",
		Short: "List tasks planned for today or overdue",
		RunE: func(cmd *cobra.Command, args []string) error {
			tasks, err := deps.TaskService.List(&task.ListOptions{Today: true})
			if err != nil {
				return err
			}

			groupBy := group
			if groupBy == "" {
				groupBy = deps.Config.Grouping.GetForCommand("today")
			}

			formatter := output.NewFormatter(os.Stdout)
			formatter.SetHidePlannedDate(true)
			formatter.GroupedTaskList(tasks, groupBy)
			return nil
		},
	}

	cmd.Flags().StringVarP(&group, "group", "g", "", "Group tasks by: project, area, date, none")
	return cmd
}

func NewUpcomingCmd(deps *Dependencies) *cobra.Command {
	var group string

	cmd := &cobra.Command{
		Use:   "upcoming",
		Short: "List tasks with future dates",
		RunE: func(cmd *cobra.Command, args []string) error {
			tasks, err := deps.TaskService.List(&task.ListOptions{Upcoming: true})
			if err != nil {
				return err
			}

			groupBy := group
			if groupBy == "" {
				groupBy = deps.Config.Grouping.GetForCommand("upcoming")
			}

			formatter := output.NewFormatter(os.Stdout)
			formatter.GroupedTaskList(tasks, groupBy)
			return nil
		},
	}

	cmd.Flags().StringVarP(&group, "group", "g", "", "Group tasks by: project, area, date, none")
	return cmd
}

func NewAnytimeCmd(deps *Dependencies) *cobra.Command {
	var group string

	cmd := &cobra.Command{
		Use:   "anytime",
		Short: "List active tasks with no specific dates",
		RunE: func(cmd *cobra.Command, args []string) error {
			tasks, err := deps.TaskService.List(&task.ListOptions{Anytime: true})
			if err != nil {
				return err
			}

			groupBy := group
			if groupBy == "" {
				groupBy = deps.Config.Grouping.GetForCommand("anytime")
			}

			formatter := output.NewFormatter(os.Stdout)
			formatter.GroupedTaskList(tasks, groupBy)
			return nil
		},
	}

	cmd.Flags().StringVarP(&group, "group", "g", "", "Group tasks by: project, area, date, none")
	return cmd
}

func NewSomedayCmd(deps *Dependencies) *cobra.Command {
	var group string

	cmd := &cobra.Command{
		Use:   "someday",
		Short: "List tasks deferred to someday",
		RunE: func(cmd *cobra.Command, args []string) error {
			tasks, err := deps.TaskService.List(&task.ListOptions{Someday: true})
			if err != nil {
				return err
			}

			groupBy := group
			if groupBy == "" {
				groupBy = deps.Config.Grouping.GetForCommand("someday")
			}

			formatter := output.NewFormatter(os.Stdout)
			formatter.GroupedTaskList(tasks, groupBy)
			return nil
		},
	}

	cmd.Flags().StringVarP(&group, "group", "g", "", "Group tasks by: project, area, date, none")
	return cmd
}
