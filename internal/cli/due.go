package cli

import (
	"errors"
	"os"
	"strconv"

	"github.com/devbydaniel/tt/internal/dateparse"
	"github.com/devbydaniel/tt/internal/output"
	"github.com/spf13/cobra"
)

func NewDueCmd(deps *Dependencies) *cobra.Command {
	var clear bool

	cmd := &cobra.Command{
		Use:     "due <task-id> [date]",
		Aliases: []string{"d"},
		Short:   "Set the due date of a task",
		Long: `Set the due date of a task.

Examples:
  t due 1 today
  t due 1 tomorrow
  t due 1 friday
  t due 1 +1w
  t due 1 2025-01-15
  t due 1 --clear`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return errors.New("invalid task ID")
			}

			if clear {
				t, err := deps.TaskService.SetDueDate(id, nil)
				if err != nil {
					return err
				}
				formatter := output.NewFormatter(os.Stdout, deps.Theme)
				formatter.TaskDueDateSet(t)
				return nil
			}

			if len(args) < 2 {
				return errors.New("date required (or use --clear to remove)")
			}

			date, err := dateparse.Parse(args[1])
			if err != nil {
				return err
			}

			t, err := deps.TaskService.SetDueDate(id, &date)
			if err != nil {
				return err
			}

			formatter := output.NewFormatter(os.Stdout, deps.Theme)
			formatter.TaskDueDateSet(t)
			return nil
		},
	}

	cmd.Flags().BoolVar(&clear, "clear", false, "Clear the due date")

	return cmd
}
