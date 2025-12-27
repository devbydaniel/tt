package cli

import (
	"os"
	"time"

	"github.com/devbydaniel/tt/internal/output"
	"github.com/spf13/cobra"
)

func NewLogCmd(deps *Dependencies) *cobra.Command {
	var sinceStr string

	cmd := &cobra.Command{
		Use:   "log",
		Short: "Show completed tasks (logbook)",
		RunE: func(cmd *cobra.Command, args []string) error {
			var since *time.Time
			if sinceStr != "" {
				parsed, err := time.Parse("2006-01-02", sinceStr)
				if err != nil {
					return err
				}
				since = &parsed
			}

			tasks, err := deps.TaskService.ListCompleted(since)
			if err != nil {
				return err
			}

			formatter := output.NewFormatter(os.Stdout)
			formatter.Logbook(tasks)
			return nil
		},
	}

	cmd.Flags().StringVar(&sinceStr, "since", "", "Show tasks completed since date (YYYY-MM-DD)")

	return cmd
}
