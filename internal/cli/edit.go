package cli

import (
	"errors"
	"os"
	"strconv"

	"github.com/devbydaniel/t/internal/dateparse"
	"github.com/devbydaniel/t/internal/output"
	"github.com/spf13/cobra"
)

func NewEditCmd(deps *Dependencies) *cobra.Command {
	var title string
	var projectName string
	var areaName string
	var plannedStr string
	var dueStr string
	var addTags []string
	var removeTags []string
	var clearPlanned bool
	var clearDue bool
	var clearProject bool
	var clearArea bool

	cmd := &cobra.Command{
		Use:   "edit <task-id>",
		Short: "Edit a task",
		Long: `Edit a task's properties.

Examples:
  t edit 1 --title "New title"
  t edit 1 --project Work
  t edit 1 --area Health
  t edit 1 --due tomorrow
  t edit 1 --planned +3d
  t edit 1 --tag urgent --tag priority
  t edit 1 --untag old-tag
  t edit 1 --clear-project
  t edit 1 --clear-due`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return errors.New("invalid task ID")
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
			if dueStr != "" && clearDue {
				return errors.New("cannot specify both --due and --clear-due")
			}

			formatter := output.NewFormatter(os.Stdout)
			var task interface{}
			var changes []string

			// Apply title change
			if title != "" {
				t, err := deps.TaskService.SetTitle(id, title)
				if err != nil {
					return err
				}
				task = t
				changes = append(changes, "title")
			}

			// Apply project change
			if projectName != "" {
				t, err := deps.TaskService.SetProject(id, projectName)
				if err != nil {
					return err
				}
				task = t
				changes = append(changes, "project")
			} else if clearProject {
				t, err := deps.TaskService.SetProject(id, "")
				if err != nil {
					return err
				}
				task = t
				changes = append(changes, "project cleared")
			}

			// Apply area change
			if areaName != "" {
				t, err := deps.TaskService.SetArea(id, areaName)
				if err != nil {
					return err
				}
				task = t
				changes = append(changes, "area")
			} else if clearArea {
				t, err := deps.TaskService.SetArea(id, "")
				if err != nil {
					return err
				}
				task = t
				changes = append(changes, "area cleared")
			}

			// Apply planned date change
			if plannedStr != "" {
				planned, err := dateparse.Parse(plannedStr)
				if err != nil {
					return err
				}
				t, err := deps.TaskService.SetPlannedDate(id, &planned)
				if err != nil {
					return err
				}
				task = t
				changes = append(changes, "planned date")
			} else if clearPlanned {
				t, err := deps.TaskService.SetPlannedDate(id, nil)
				if err != nil {
					return err
				}
				task = t
				changes = append(changes, "planned date cleared")
			}

			// Apply due date change
			if dueStr != "" {
				due, err := dateparse.Parse(dueStr)
				if err != nil {
					return err
				}
				t, err := deps.TaskService.SetDueDate(id, &due)
				if err != nil {
					return err
				}
				task = t
				changes = append(changes, "due date")
			} else if clearDue {
				t, err := deps.TaskService.SetDueDate(id, nil)
				if err != nil {
					return err
				}
				task = t
				changes = append(changes, "due date cleared")
			}

			// Apply tag additions
			for _, tag := range addTags {
				t, err := deps.TaskService.AddTag(id, tag)
				if err != nil {
					return err
				}
				task = t
			}
			if len(addTags) > 0 {
				changes = append(changes, "tags added")
			}

			// Apply tag removals
			for _, tag := range removeTags {
				t, err := deps.TaskService.RemoveTag(id, tag)
				if err != nil {
					return err
				}
				task = t
			}
			if len(removeTags) > 0 {
				changes = append(changes, "tags removed")
			}

			// If no changes were made, show current task
			if len(changes) == 0 {
				t, err := deps.TaskService.GetByID(id)
				if err != nil {
					return err
				}
				formatter.TaskDetails(t)
				return nil
			}

			// Show confirmation
			formatter.TaskEdited(id, changes)
			_ = task // last task state after all changes
			return nil
		},
	}

	cmd.Flags().StringVar(&title, "title", "", "Set task title")
	cmd.Flags().StringVarP(&projectName, "project", "p", "", "Assign to project")
	cmd.Flags().StringVarP(&areaName, "area", "a", "", "Assign to area")
	cmd.Flags().StringVar(&plannedStr, "planned", "", "Set planned date")
	cmd.Flags().StringVarP(&dueStr, "due", "d", "", "Set due date")
	cmd.Flags().StringArrayVarP(&addTags, "tag", "t", nil, "Add tag (repeatable)")
	cmd.Flags().StringArrayVar(&removeTags, "untag", nil, "Remove tag (repeatable)")
	cmd.Flags().BoolVar(&clearPlanned, "clear-planned", false, "Clear planned date")
	cmd.Flags().BoolVar(&clearDue, "clear-due", false, "Clear due date")
	cmd.Flags().BoolVar(&clearProject, "clear-project", false, "Remove from project")
	cmd.Flags().BoolVar(&clearArea, "clear-area", false, "Remove from area")

	return cmd
}
