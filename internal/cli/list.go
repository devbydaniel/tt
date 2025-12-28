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
	var sortStr string
	var today bool
	var upcoming bool
	var someday bool
	var anytime bool
	var inbox bool
	var all bool
	var group string
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List tasks",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Determine view command first (needed for config lookup)
			viewCmd := "all"
			if today {
				viewCmd = "today"
			} else if upcoming {
				viewCmd = "upcoming"
			} else if someday {
				viewCmd = "someday"
			} else if anytime {
				viewCmd = "anytime"
			} else if inbox {
				viewCmd = "inbox"
			}

			// Resolve sorting: flag > config for view > code default
			sortToUse := sortStr
			if sortToUse == "" {
				sortToUse = deps.Config.Sorting.GetForCommand(viewCmd)
			}
			sortOpts, err := task.ParseSort(sortToUse)
			if err != nil {
				return err
			}

			opts := &task.ListOptions{
				ProjectName: projectName,
				AreaName:    areaName,
				TagName:     tagName,
				Search:      search,
				Sort:        sortOpts,
				Today:       today,
				Upcoming:    upcoming,
				Someday:     someday,
				Anytime:     anytime,
				Inbox:       inbox,
				All:         all,
			}

			// Default to showing all active tasks (default_list config only applies to bare "tt" command)
			if viewCmd == "all" {
				opts.All = true
			}

			tasks, err := deps.TaskService.List(opts)
			if err != nil {
				return err
			}

			if jsonOutput {
				return output.WriteJSON(os.Stdout, tasks)
			}

			// Resolve grouping: flag > config for view > config for list > none
			groupBy := group
			if groupBy == "" {
				groupBy = deps.Config.Grouping.GetForCommand(viewCmd)
			}

			formatter := output.NewFormatter(os.Stdout, deps.Theme)
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
	cmd.Flags().StringVarP(&search, "search", "S", "", "Search task titles")
	cmd.Flags().StringVarP(&sortStr, "sort", "s", "", "Sort by field(s): id, title, planned, due, created, project, area (e.g. due,title:desc)")
	cmd.Flags().BoolVar(&today, "today", false, "Show tasks planned for today or overdue")
	cmd.Flags().BoolVar(&upcoming, "upcoming", false, "Show tasks with future dates")
	cmd.Flags().BoolVar(&someday, "someday", false, "Show someday tasks")
	cmd.Flags().BoolVar(&anytime, "anytime", false, "Show active tasks with no dates")
	cmd.Flags().BoolVar(&inbox, "inbox", false, "Show tasks with no project, area, or dates")
	cmd.Flags().BoolVar(&all, "all", false, "Show all active tasks")
	cmd.Flags().StringVarP(&group, "group", "g", "", "Group tasks by: project, area, date, none")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")

	// Register completions
	registry := NewCompletionRegistry(deps)
	registry.RegisterAll(cmd)

	return cmd
}
