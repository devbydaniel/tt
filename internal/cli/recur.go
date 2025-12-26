package cli

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/devbydaniel/t/internal/dateparse"
	"github.com/devbydaniel/t/internal/output"
	"github.com/devbydaniel/t/internal/recurparse"
	"github.com/spf13/cobra"
)

func NewRecurCmd(deps *Dependencies) *cobra.Command {
	var clear bool
	var pause bool
	var resume bool
	var endStr string
	var show bool

	cmd := &cobra.Command{
		Use:   "recur <id> [pattern]",
		Short: "Set, clear, or manage task recurrence",
		Long: `Set, clear, or manage task recurrence.

Examples:
  t recur 5 "every monday"      Set weekly recurrence on Mondays
  t recur 5 "daily"             Set daily recurrence
  t recur 5 "3d after done"     Recur 3 days after completion
  t recur 5 --clear             Clear recurrence
  t recur 5 --pause             Pause recurrence
  t recur 5 --resume            Resume paused recurrence
  t recur 5 --end 2025-12-31    Set recurrence end date
  t recur 5 --show              Show current recurrence info`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return errors.New("invalid task ID: " + args[0])
			}

			formatter := output.NewFormatter(os.Stdout)

			// Handle --show
			if show {
				t, err := deps.TaskService.GetByID(id)
				if err != nil {
					return err
				}
				formatter.TaskRecurrenceInfo(t)
				return nil
			}

			// Handle --clear
			if clear {
				t, err := deps.TaskService.SetRecurrence(id, nil, nil, nil)
				if err != nil {
					return err
				}
				formatter.TaskRecurrenceSet(t)
				return nil
			}

			// Handle --pause
			if pause {
				t, err := deps.TaskService.PauseRecurrence(id)
				if err != nil {
					return err
				}
				formatter.TaskRecurrencePaused(t)
				return nil
			}

			// Handle --resume
			if resume {
				t, err := deps.TaskService.ResumeRecurrence(id)
				if err != nil {
					return err
				}
				formatter.TaskRecurrenceResumed(t)
				return nil
			}

			// Handle --end (without pattern)
			if endStr != "" && len(args) == 1 {
				var endDate *time.Time
				end, err := dateparse.Parse(endStr)
				if err != nil {
					return err
				}
				endDate = &end
				t, err := deps.TaskService.SetRecurrenceEnd(id, endDate)
				if err != nil {
					return err
				}
				formatter.TaskRecurrenceEndSet(t)
				return nil
			}

			// Need a pattern to set recurrence
			if len(args) < 2 {
				return errors.New("recurrence pattern required (e.g., 'daily', 'every monday', '3d after done')")
			}

			pattern := strings.Join(args[1:], " ")
			result, err := recurparse.Parse(pattern)
			if err != nil {
				return err
			}

			ruleJSON, err := result.Rule.ToJSON()
			if err != nil {
				return err
			}

			recurType := string(result.Type)

			// Parse end date if provided
			var endDate *time.Time
			if endStr != "" {
				end, err := dateparse.Parse(endStr)
				if err != nil {
					return err
				}
				endDate = &end
			}

			t, err := deps.TaskService.SetRecurrence(id, &recurType, &ruleJSON, endDate)
			if err != nil {
				return err
			}

			formatter.TaskRecurrenceSet(t)
			return nil
		},
	}

	cmd.Flags().BoolVar(&clear, "clear", false, "Clear recurrence from task")
	cmd.Flags().BoolVar(&pause, "pause", false, "Pause recurrence (keeps rule)")
	cmd.Flags().BoolVar(&resume, "resume", false, "Resume paused recurrence")
	cmd.Flags().StringVar(&endStr, "end", "", "Set recurrence end date")
	cmd.Flags().BoolVar(&show, "show", false, "Show current recurrence info")

	return cmd
}
