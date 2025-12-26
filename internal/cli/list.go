package cli

import (
	"os"

	"github.com/devbydaniel/t/internal/domain/task"
	"github.com/devbydaniel/t/internal/output"
	"github.com/spf13/cobra"
)

func NewListCmd(deps *Dependencies) *cobra.Command {
	var projectName string
	var areaName string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List tasks",
		RunE: func(cmd *cobra.Command, args []string) error {
			var opts *task.ListOptions
			if projectName != "" || areaName != "" {
				opts = &task.ListOptions{
					ProjectName: projectName,
					AreaName:    areaName,
				}
			}

			tasks, err := deps.TaskService.List(opts)
			if err != nil {
				return err
			}

			formatter := output.NewFormatter(os.Stdout)
			formatter.TaskList(tasks)
			return nil
		},
	}

	cmd.Flags().StringVarP(&projectName, "project", "p", "", "Filter by project name")
	cmd.Flags().StringVarP(&areaName, "area", "a", "", "Filter by area name")

	return cmd
}
