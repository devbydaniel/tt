package cli

import (
	"errors"
	"os"
	"strconv"

	"github.com/devbydaniel/t/internal/dateparse"
	"github.com/devbydaniel/t/internal/output"
	"github.com/spf13/cobra"
)

func NewPlanCmd(deps *Dependencies) *cobra.Command {
	var clear bool

	cmd := &cobra.Command{
		Use:   "plan <task-id> [date]",
		Short: "Set the planned date of a task",
		Long: `Set the planned date of a task.

Examples:
  t plan 1 today
  t plan 1 tomorrow
  t plan 1 monday
  t plan 1 +3d
  t plan 1 2025-01-15
  t plan 1 --clear`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return errors.New("invalid task ID")
			}

			if clear {
				t, err := deps.TaskService.SetPlannedDate(id, nil)
				if err != nil {
					return err
				}
				formatter := output.NewFormatter(os.Stdout)
				formatter.TaskPlannedDateSet(t)
				return nil
			}

			if len(args) < 2 {
				return errors.New("date required (or use --clear to remove)")
			}

			date, err := dateparse.Parse(args[1])
			if err != nil {
				return err
			}

			t, err := deps.TaskService.SetPlannedDate(id, &date)
			if err != nil {
				return err
			}

			formatter := output.NewFormatter(os.Stdout)
			formatter.TaskPlannedDateSet(t)
			return nil
		},
	}

	cmd.Flags().BoolVar(&clear, "clear", false, "Clear the planned date")

	return cmd
}
