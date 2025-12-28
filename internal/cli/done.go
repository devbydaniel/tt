package cli

import (
	"errors"
	"os"
	"strconv"

	"github.com/devbydaniel/tt/internal/output"
	"github.com/spf13/cobra"
)

func NewDoneCmd(deps *Dependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "done <id> [id...]",
		Short: "Mark task(s) as complete",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ids := make([]int64, 0, len(args))
			for _, arg := range args {
				id, err := strconv.ParseInt(arg, 10, 64)
				if err != nil {
					return errors.New("invalid task ID: " + arg)
				}
				ids = append(ids, id)
			}

			completed, err := deps.TaskService.Complete(ids)
			if err != nil {
				return err
			}

			formatter := output.NewFormatter(os.Stdout, deps.Theme)
			formatter.TasksCompleted(completed)
			return nil
		},
	}
}
