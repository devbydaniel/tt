package cli

import (
	"fmt"
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
	var group string
	var hideScope bool
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List tasks",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Determine schedule from flags
			schedule := ""
			viewCmd := "all"
			if today {
				schedule = "today"
				viewCmd = "today"
			} else if upcoming {
				schedule = "upcoming"
				viewCmd = "upcoming"
			} else if someday {
				schedule = "someday"
				viewCmd = "someday"
			} else if anytime {
				schedule = "anytime"
				viewCmd = "anytime"
			} else if inbox {
				schedule = "inbox"
				viewCmd = "inbox"
			}

			// Determine config key for settings lookup (priority: project > area > tag)
			configKey := viewCmd
			if projectName != "" {
				configKey = "project"
			} else if areaName != "" {
				configKey = "area"
			} else if tagName != "" {
				configKey = "tag"
			}

			// Resolve sorting: flag > config for view > code default
			sortToUse := sortStr
			if sortToUse == "" {
				sortToUse = deps.Config.GetSort(configKey)
			}
			sortOpts, err := task.ParseSort(sortToUse)
			if err != nil {
				return err
			}

			// Resolve grouping: flag > config for view
			groupBy := group
			if groupBy == "" {
				groupBy = deps.Config.GetGroup(configKey)
			}

			// Resolve hideScope: flag > config for view
			hideScopeToUse := hideScope
			if !hideScope {
				hideScopeToUse = deps.Config.GetHideScope(configKey)
			}

			formatter := output.NewFormatter(os.Stdout, deps.Theme)
			if schedule == "today" {
				formatter.SetHidePlannedDate(true)
			}
			if hideScopeToUse {
				formatter.SetHideScope(true)
			}

			// JSON output: single call, all tasks
			if jsonOutput {
				tasks, err := deps.TaskService.List(&task.ListOptions{
					ProjectName: projectName,
					AreaName:    areaName,
					TagName:     tagName,
					Search:      search,
					Sort:        sortOpts,
					Schedule:    schedule,
				})
				if err != nil {
					return err
				}
				return output.WriteJSON(os.Stdout, tasks)
			}

			// Schedule grouping: 4 separate queries
			if groupBy == "schedule" {
				schedules := []struct {
					name     string
					schedule string
				}{
					{"Today", "today"},
					{"Upcoming", "upcoming"},
					{"Anytime", "anytime"},
					{"Someday", "someday"},
				}

				for _, sched := range schedules {
					tasks, err := deps.TaskService.List(&task.ListOptions{
						ProjectName: projectName,
						AreaName:    areaName,
						TagName:     tagName,
						Search:      search,
						Sort:        sortOpts,
						Schedule:    sched.schedule,
					})
					if err != nil {
						return err
					}
					if len(tasks) > 0 {
						fmt.Fprintln(os.Stdout, deps.Theme.Header.Render(sched.name))
						formatter.TaskList(tasks)
					}
				}
				return nil
			}

			// Other groupings: single call, client-side grouping
			tasks, err := deps.TaskService.List(&task.ListOptions{
				ProjectName: projectName,
				AreaName:    areaName,
				TagName:     tagName,
				Search:      search,
				Sort:        sortOpts,
				Schedule:    schedule,
			})
			if err != nil {
				return err
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
	cmd.Flags().StringVarP(&group, "group", "g", "", "Group tasks by: schedule, project, area, date, none")
	cmd.Flags().BoolVar(&hideScope, "hide-scope", false, "Hide project/area columns")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")

	// Register completions
	registry := NewCompletionRegistry(deps)
	registry.RegisterAll(cmd)

	return cmd
}
