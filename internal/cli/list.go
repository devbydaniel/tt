package cli

import (
	"os"

	"github.com/devbydaniel/tt/internal/domain/task"
	"github.com/devbydaniel/tt/internal/output"
	"github.com/spf13/cobra"
)

func NewListCmd(deps *Dependencies) *cobra.Command {
	var projectName string
	var areaName string
	var tagName string
	var today bool
	var upcoming bool
	var someday bool
	var anytime bool
	var inbox bool
	var all bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List tasks",
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := &task.ListOptions{
				ProjectName: projectName,
				AreaName:    areaName,
				TagName:     tagName,
				Today:       today,
				Upcoming:    upcoming,
				Someday:     someday,
				Anytime:     anytime,
				Inbox:       inbox,
				All:         all,
			}

			// Default to --today if no filter specified
			if !today && !upcoming && !someday && !anytime && !inbox && !all {
				opts.Today = true
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

	cmd.Flags().StringVarP(&projectName, "project", "p", "", "Filter by project name")
	cmd.Flags().StringVarP(&areaName, "area", "a", "", "Filter by area name")
	cmd.Flags().StringVar(&tagName, "tag", "", "Filter by tag")
	cmd.Flags().BoolVar(&today, "today", false, "Show tasks planned for today or overdue")
	cmd.Flags().BoolVar(&upcoming, "upcoming", false, "Show tasks with future dates")
	cmd.Flags().BoolVar(&someday, "someday", false, "Show someday tasks")
	cmd.Flags().BoolVar(&anytime, "anytime", false, "Show active tasks with no dates")
	cmd.Flags().BoolVar(&inbox, "inbox", false, "Show tasks with no project, area, or dates")
	cmd.Flags().BoolVar(&all, "all", false, "Show all active tasks")

	return cmd
}
