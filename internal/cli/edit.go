package cli

import (
	"errors"
	"os"
	"strconv"

	"github.com/devbydaniel/tt/internal/dateparse"
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

	cmd := &cobra.Command{
		Use:     "edit <task-id>...",
		Aliases: []string{"e"},
		Short:   "Edit one or more tasks",
		Long: `Edit task properties. Supports multiple task IDs.

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
  t edit 1 --active`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Parse all task IDs first
			var ids []int64
			for _, arg := range args {
				id, err := strconv.ParseInt(arg, 10, 64)
				if err != nil {
					return errors.New("invalid task ID: " + arg)
				}
				ids = append(ids, id)
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

			formatter := output.NewFormatter(os.Stdout, deps.Theme)

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
				} else {
					return errors.New("no changes specified")
				}
				return nil
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

	// Register completions
	registry := NewCompletionRegistry(deps)
	registry.RegisterAll(cmd)

	return cmd
}
