package cli

import (
	"errors"
	"os"
	"strings"

	"github.com/devbydaniel/tt/internal/dateparse"
	"github.com/devbydaniel/tt/internal/domain/task"
	"github.com/devbydaniel/tt/internal/output"
	"github.com/devbydaniel/tt/internal/recurparse"
	"github.com/spf13/cobra"
)

func NewAddCmd(deps *Dependencies) *cobra.Command {
	var projectName string
	var areaName string
	var description string
	var plannedStr string
	var dueStr string
	var today bool
	var someday bool
	var recurStr string
	var recurEndStr string
	var tags []string

	cmd := &cobra.Command{
		Use:   "add [title]",
		Short: "Add a new task",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			title := strings.Join(args, " ")
			if title == "" {
				return errors.New("task title cannot be empty")
			}

			if projectName != "" && areaName != "" {
				return errors.New("cannot specify both --project and --area")
			}

			if today && plannedStr != "" {
				return errors.New("cannot specify both --today and --planned")
			}
			if today {
				plannedStr = "today"
			}

			opts := &task.CreateOptions{
				ProjectName: projectName,
				AreaName:    areaName,
				Description: description,
				Someday:     someday,
				Tags:        tags,
			}

			if plannedStr != "" {
				planned, err := dateparse.Parse(plannedStr)
				if err != nil {
					return err
				}
				opts.PlannedDate = &planned
			}

			if dueStr != "" {
				due, err := dateparse.Parse(dueStr)
				if err != nil {
					return err
				}
				opts.DueDate = &due
			}

			// Parse recurrence if provided
			if recurStr != "" {
				result, err := recurparse.Parse(recurStr)
				if err != nil {
					return err
				}
				ruleJSON, err := result.Rule.ToJSON()
				if err != nil {
					return err
				}
				recurType := string(result.Type)
				opts.RecurType = &recurType
				opts.RecurRule = &ruleJSON
			}

			// Parse recurrence end date if provided
			if recurEndStr != "" {
				recurEnd, err := dateparse.Parse(recurEndStr)
				if err != nil {
					return err
				}
				opts.RecurEnd = &recurEnd
			}

			t, err := deps.TaskService.Create(title, opts)
			if err != nil {
				return err
			}

			formatter := output.NewFormatter(os.Stdout)
			formatter.TaskCreated(t)
			return nil
		},
	}

	cmd.Flags().StringVarP(&projectName, "project", "p", "", "Assign to project")
	cmd.Flags().StringVarP(&areaName, "area", "a", "", "Assign to area")
	cmd.Flags().StringVarP(&description, "description", "d", "", "Task description")
	cmd.Flags().StringVarP(&plannedStr, "planned", "P", "", "Planned date (e.g., today, tomorrow, +3d, 2025-01-15)")
	cmd.Flags().BoolVarP(&today, "today", "T", false, "Set planned date to today")
	cmd.Flags().StringVarP(&dueStr, "due", "D", "", "Due date (e.g., today, tomorrow, +3d, 2025-01-15)")
	cmd.Flags().BoolVar(&someday, "someday", false, "Create task in someday state")
	cmd.Flags().StringVarP(&recurStr, "recur", "r", "", "Recurrence pattern (e.g., daily, every monday, 3d after done)")
	cmd.Flags().StringVar(&recurEndStr, "recur-end", "", "Recurrence end date")
	cmd.Flags().StringArrayVarP(&tags, "tag", "t", nil, "Add tag (repeatable)")

	// Register completions
	registry := NewCompletionRegistry(deps)
	registry.RegisterAll(cmd)

	return cmd
}
