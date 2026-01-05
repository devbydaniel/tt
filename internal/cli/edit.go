package cli

import (
	"errors"
	"os"
	"strconv"

	"github.com/devbydaniel/tt/internal/dateparse"
	"github.com/devbydaniel/tt/internal/domain/task"
	"github.com/devbydaniel/tt/internal/output"
	"github.com/spf13/cobra"
)

func NewEditCmd(deps *Dependencies) *cobra.Command {
	var title string
	var description string
	var projectName string
	var areaName string
	var plannedStr string
	var dueStr string
	var today bool
	var addTags []string
	var removeTags []string
	var clearPlanned bool
	var clearDue bool
	var clearProject bool
	var clearArea bool
	var clearDescription bool
	var someday bool
	var active bool

	// Bulk edit filter flags
	var whereProject string
	var whereNotProjects []string
	var whereArea string
	var whereNotAreas []string
	var whereTags []string
	var whereNotTags []string
	var whereState string
	var dryRun bool

	cmd := &cobra.Command{
		Use:     "edit [<task-id>...]",
		Aliases: []string{"e"},
		Short:   "Edit one or more tasks",
		Long: `Edit task properties. Specify task IDs or use --where-* filters for bulk editing.

Examples:
  t edit 1 --title "New title"
  t edit 1 --project Work
  t edit 1 2 3 --project Work
  t edit 1 --area Health
  t edit 1 --due tomorrow
  t edit 1 --planned +3d
  t edit 1 --tag urgent --tag priority
  t edit 1 --untag old-tag
  t edit 1 --clear-project
  t edit 1 --clear-due
  t edit 1 --someday
  t edit 1 --active

Bulk editing with filters (all --where-* flags are ANDed together):
  t edit --where-project OldProject --project NewProject
  t edit --where-tag urgent --where-tag work --clear-planned
  t edit --where-state someday --active
  t edit --where-area Health --where-not-tag done --today
  t edit --where-not-project Work --where-not-project Personal --today
  t edit --where-project Work --dry-run`,
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			formatter := output.NewFormatter(os.Stdout, deps.Theme)

			// Determine mode: ID-based vs filter-based
			hasWhereFlags := whereProject != "" || len(whereNotProjects) > 0 ||
				whereArea != "" || len(whereNotAreas) > 0 ||
				len(whereTags) > 0 || len(whereNotTags) > 0 ||
				whereState != ""
			hasIDArgs := len(args) > 0

			// Validation: cannot mix ID args with where-* flags
			if hasIDArgs && hasWhereFlags {
				return errors.New("cannot specify both task IDs and --where-* flags")
			}

			var ids []int64

			if hasWhereFlags {
				// Bulk edit mode: query matching tasks
				tasks, err := deps.App.ListTasks.Execute(&task.ListOptions{
					TaskType:        task.TaskTypeTask, // only edit tasks, not projects
					ProjectName:     whereProject,
					NotProjectNames: whereNotProjects,
					AreaName:        whereArea,
					NotAreaNames:    whereNotAreas,
					TagNames:        whereTags,
					NotTagNames:     whereNotTags,
					State:           task.State(whereState),
				})
				if err != nil {
					return err
				}

				if len(tasks) == 0 {
					return errors.New("no tasks match the specified filters")
				}

				// Dry run: show what would be edited
				if dryRun {
					formatter.BulkEditPreview(tasks)
					return nil
				}

				for _, t := range tasks {
					ids = append(ids, t.ID)
				}
			} else if hasIDArgs {
				// ID mode: parse task IDs from args
				for _, arg := range args {
					id, err := strconv.ParseInt(arg, 10, 64)
					if err != nil {
						return errors.New("invalid task ID: " + arg)
					}
					ids = append(ids, id)
				}
			}

			// Validate mutual exclusivity
			if projectName != "" && areaName != "" {
				return errors.New("cannot specify both --project and --area")
			}
			if projectName != "" && clearProject {
				return errors.New("cannot specify both --project and --clear-project")
			}
			if areaName != "" && clearArea {
				return errors.New("cannot specify both --area and --clear-area")
			}
			if plannedStr != "" && clearPlanned {
				return errors.New("cannot specify both --planned and --clear-planned")
			}
			if today && plannedStr != "" {
				return errors.New("cannot specify both --today and --planned")
			}
			if today && clearPlanned {
				return errors.New("cannot specify both --today and --clear-planned")
			}
			if today {
				plannedStr = "today"
			}
			if dueStr != "" && clearDue {
				return errors.New("cannot specify both --due and --clear-due")
			}
			if description != "" && clearDescription {
				return errors.New("cannot specify both --description and --clear-description")
			}

			// If no changes specified and single task, show details
			hasChanges := title != "" || description != "" || projectName != "" || areaName != "" ||
				plannedStr != "" || dueStr != "" || today || clearPlanned || clearDue ||
				clearProject || clearArea || clearDescription || len(addTags) > 0 || len(removeTags) > 0 ||
				someday || active

			if !hasChanges {
				if len(ids) == 1 {
					t, err := deps.App.GetTask.Execute(ids[0])
					if err != nil {
						return err
					}
					formatter.TaskDetails(t)
					return nil
				} else if len(ids) == 0 && !hasWhereFlags {
					return errors.New("specify task IDs or --where-* filters")
				} else {
					return errors.New("no changes specified")
				}
			}

			// Build changes list once (same for all tasks)
			var changes []string
			if title != "" {
				changes = append(changes, "title")
			}
			if description != "" {
				changes = append(changes, "description")
			} else if clearDescription {
				changes = append(changes, "description cleared")
			}
			if projectName != "" {
				changes = append(changes, "project")
			} else if clearProject {
				changes = append(changes, "project cleared")
			}
			if areaName != "" {
				changes = append(changes, "area")
			} else if clearArea {
				changes = append(changes, "area cleared")
			}
			if plannedStr != "" {
				changes = append(changes, "planned date")
			} else if clearPlanned {
				changes = append(changes, "planned date cleared")
			}
			if dueStr != "" {
				changes = append(changes, "due date")
			} else if clearDue {
				changes = append(changes, "due date cleared")
			}
			if len(addTags) > 0 {
				changes = append(changes, "tags added")
			}
			if len(removeTags) > 0 {
				changes = append(changes, "tags removed")
			}
			if someday {
				changes = append(changes, "moved to someday")
			}
			if active {
				changes = append(changes, "moved to active")
			}

			// Apply changes to all tasks
			for _, id := range ids {
				if title != "" {
					if _, err := deps.App.SetTaskTitle.Execute(id, title); err != nil {
						return err
					}
				}

				if description != "" {
					if _, err := deps.App.SetTaskDescription.Execute(id, &description); err != nil {
						return err
					}
				} else if clearDescription {
					if _, err := deps.App.SetTaskDescription.Execute(id, nil); err != nil {
						return err
					}
				}

				if projectName != "" {
					if _, err := deps.App.SetTaskProject.Execute(id, projectName); err != nil {
						return err
					}
				} else if clearProject {
					if _, err := deps.App.SetTaskProject.Execute(id, ""); err != nil {
						return err
					}
				}

				if areaName != "" {
					if _, err := deps.App.SetTaskArea.Execute(id, areaName); err != nil {
						return err
					}
				} else if clearArea {
					if _, err := deps.App.SetTaskArea.Execute(id, ""); err != nil {
						return err
					}
				}

				if plannedStr != "" {
					planned, err := dateparse.Parse(plannedStr)
					if err != nil {
						return err
					}
					if _, err := deps.App.SetPlannedDate.Execute(id, &planned); err != nil {
						return err
					}
				} else if clearPlanned {
					if _, err := deps.App.SetPlannedDate.Execute(id, nil); err != nil {
						return err
					}
				}

				if dueStr != "" {
					due, err := dateparse.Parse(dueStr)
					if err != nil {
						return err
					}
					if _, err := deps.App.SetDueDate.Execute(id, &due); err != nil {
						return err
					}
				} else if clearDue {
					if _, err := deps.App.SetDueDate.Execute(id, nil); err != nil {
						return err
					}
				}

				for _, tag := range addTags {
					if _, err := deps.App.AddTag.Execute(id, tag); err != nil {
						return err
					}
				}

				for _, tag := range removeTags {
					if _, err := deps.App.RemoveTag.Execute(id, tag); err != nil {
						return err
					}
				}

				if someday {
					if _, err := deps.App.DeferTask.Execute(id); err != nil {
						return err
					}
				}

				if active {
					if _, err := deps.App.ActivateTask.Execute(id); err != nil {
						return err
					}
				}

				formatter.TaskEdited(id, changes)
			}

			// Show summary for bulk edits
			if hasWhereFlags && len(ids) > 1 {
				formatter.BulkEditSummary(len(ids))
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&title, "title", "", "Set task title")
	cmd.Flags().StringVarP(&description, "description", "d", "", "Set task description")
	cmd.Flags().StringVarP(&projectName, "project", "p", "", "Assign to project")
	cmd.Flags().StringVarP(&areaName, "area", "a", "", "Assign to area")
	cmd.Flags().StringVarP(&plannedStr, "planned", "P", "", "Set planned date")
	cmd.Flags().BoolVarP(&today, "today", "T", false, "Set planned date to today")
	cmd.Flags().StringVarP(&dueStr, "due", "D", "", "Set due date")
	cmd.Flags().StringArrayVarP(&addTags, "tag", "t", nil, "Add tag (repeatable)")
	cmd.Flags().StringArrayVar(&removeTags, "untag", nil, "Remove tag (repeatable)")
	cmd.Flags().BoolVar(&clearPlanned, "clear-planned", false, "Clear planned date")
	cmd.Flags().BoolVar(&clearDue, "clear-due", false, "Clear due date")
	cmd.Flags().BoolVar(&clearProject, "clear-project", false, "Remove from project")
	cmd.Flags().BoolVar(&clearArea, "clear-area", false, "Remove from area")
	cmd.Flags().BoolVar(&clearDescription, "clear-description", false, "Clear description")
	cmd.Flags().BoolVarP(&someday, "someday", "s", false, "Move to someday")
	cmd.Flags().BoolVarP(&active, "active", "A", false, "Move to active")
	cmd.MarkFlagsMutuallyExclusive("someday", "active")

	// Bulk edit filter flags
	cmd.Flags().StringVar(&whereProject, "where-project", "", "Filter by project")
	cmd.Flags().StringArrayVar(&whereNotProjects, "where-not-project", nil, "Exclude project (AND, repeatable)")
	cmd.Flags().StringVar(&whereArea, "where-area", "", "Filter by area")
	cmd.Flags().StringArrayVar(&whereNotAreas, "where-not-area", nil, "Exclude area (AND, repeatable)")
	cmd.Flags().StringArrayVar(&whereTags, "where-tag", nil, "Filter by tag (AND, repeatable)")
	cmd.Flags().StringArrayVar(&whereNotTags, "where-not-tag", nil, "Exclude tag (repeatable)")
	cmd.Flags().StringVar(&whereState, "where-state", "", "Filter by state (active, someday)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview matching tasks without editing")

	// Register completions
	registry := NewCompletionRegistry(deps)
	registry.RegisterAll(cmd)

	// Register where-* flag completions
	_ = cmd.RegisterFlagCompletionFunc("where-project", registry.AllProjectCompletion())
	_ = cmd.RegisterFlagCompletionFunc("where-not-project", registry.AllProjectCompletion())
	_ = cmd.RegisterFlagCompletionFunc("where-area", registry.AreaCompletion())
	_ = cmd.RegisterFlagCompletionFunc("where-not-area", registry.AreaCompletion())
	_ = cmd.RegisterFlagCompletionFunc("where-tag", registry.TagCompletion())
	_ = cmd.RegisterFlagCompletionFunc("where-not-tag", registry.TagCompletion())
	_ = cmd.RegisterFlagCompletionFunc("where-state", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"active", "someday"}, cobra.ShellCompDirectiveNoFileComp
	})

	return cmd
}
