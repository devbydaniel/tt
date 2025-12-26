package cli

import (
	"errors"
	"os"
	"strings"

	"github.com/devbydaniel/t/internal/dateparse"
	"github.com/devbydaniel/t/internal/domain/task"
	"github.com/devbydaniel/t/internal/output"
	"github.com/devbydaniel/t/internal/recurparse"
	"github.com/spf13/cobra"
)

func NewAddCmd(deps *Dependencies) *cobra.Command {
	var projectName string
	var areaName string
	var plannedStr string
	var dueStr string
	var someday bool
	var recurStr string
	var recurEndStr string

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

			opts := &task.CreateOptions{
				ProjectName: projectName,
				AreaName:    areaName,
				Someday:     someday,
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
	cmd.Flags().StringVar(&plannedStr, "planned", "", "Planned date (e.g., today, tomorrow, +3d, 2025-01-15)")
	cmd.Flags().StringVarP(&dueStr, "due", "d", "", "Due date (e.g., today, tomorrow, +3d, 2025-01-15)")
	cmd.Flags().BoolVar(&someday, "someday", false, "Create task in someday state")
	cmd.Flags().StringVarP(&recurStr, "recur", "r", "", "Recurrence pattern (e.g., daily, every monday, 3d after done)")
	cmd.Flags().StringVar(&recurEndStr, "recur-end", "", "Recurrence end date")

	return cmd
}
