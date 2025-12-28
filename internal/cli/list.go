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
	var search string
	var today bool
	var upcoming bool
	var someday bool
	var anytime bool
	var inbox bool
	var all bool
	var group string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List tasks",
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := &task.ListOptions{
				ProjectName: projectName,
				AreaName:    areaName,
				TagName:     tagName,
				Search:      search,
				Today:       today,
				Upcoming:    upcoming,
				Someday:     someday,
				Anytime:     anytime,
				Inbox:       inbox,
				All:         all,
			}

			// Apply default_list config if no view filter specified
			// When filtering by project/area, default to showing all tasks in that filter
			viewCmd := "list"
			if !today && !upcoming && !someday && !anytime && !inbox && !all {
				if projectName != "" || areaName != "" {
					// Project/area filter: show all tasks in that project/area
					opts.All = true
					viewCmd = "all"
				} else {
					switch deps.Config.DefaultList {
					case "upcoming":
						opts.Upcoming = true
						viewCmd = "upcoming"
					case "anytime":
						opts.Anytime = true
						viewCmd = "anytime"
					case "someday":
						opts.Someday = true
						viewCmd = "someday"
					case "inbox":
						opts.Inbox = true
						viewCmd = "inbox"
					case "all":
						opts.All = true
						viewCmd = "all"
					default:
						opts.Today = true
						viewCmd = "today"
					}
				}
			}

			tasks, err := deps.TaskService.List(opts)
			if err != nil {
				return err
			}

			// Resolve grouping: flag > config for view > config for list > none
			groupBy := group
			if groupBy == "" {
				groupBy = deps.Config.Grouping.GetForCommand(viewCmd)
			}

			formatter := output.NewFormatter(os.Stdout)
			if opts.Today {
				formatter.SetHidePlannedDate(true)
			}
			formatter.GroupedTaskList(tasks, groupBy)
			return nil
		},
	}

	cmd.Flags().StringVarP(&projectName, "project", "p", "", "Filter by project name")
	cmd.Flags().StringVarP(&areaName, "area", "a", "", "Filter by area name")
	cmd.Flags().StringVar(&tagName, "tag", "", "Filter by tag")
	cmd.Flags().StringVarP(&search, "search", "s", "", "Search task titles")
	cmd.Flags().BoolVar(&today, "today", false, "Show tasks planned for today or overdue")
	cmd.Flags().BoolVar(&upcoming, "upcoming", false, "Show tasks with future dates")
	cmd.Flags().BoolVar(&someday, "someday", false, "Show someday tasks")
	cmd.Flags().BoolVar(&anytime, "anytime", false, "Show active tasks with no dates")
	cmd.Flags().BoolVar(&inbox, "inbox", false, "Show tasks with no project, area, or dates")
	cmd.Flags().BoolVar(&all, "all", false, "Show all active tasks")
	cmd.Flags().StringVarP(&group, "group", "g", "", "Group tasks by: project, area, date, none")

	// Register completions
	registry := NewCompletionRegistry(deps)
	registry.RegisterAll(cmd)

	return cmd
}
