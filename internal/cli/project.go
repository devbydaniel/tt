package cli

import (
	"os"

	"github.com/devbydaniel/t/internal/output"
	"github.com/spf13/cobra"
)

func NewProjectCmd(deps *Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project",
		Short: "Manage projects",
	}

	cmd.AddCommand(newProjectListCmd(deps))
	cmd.AddCommand(newProjectAddCmd(deps))
	cmd.AddCommand(newProjectDeleteCmd(deps))

	return cmd
}

func newProjectListCmd(deps *Dependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all projects",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			projects, err := deps.ProjectService.List()
			if err != nil {
				return err
			}

			formatter := output.NewFormatter(os.Stdout)
			formatter.ProjectList(projects)
			return nil
		},
	}
}

func newProjectAddCmd(deps *Dependencies) *cobra.Command {
	var areaName string

	cmd := &cobra.Command{
		Use:   "add <name>",
		Short: "Create a new project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			project, err := deps.ProjectService.Create(args[0], areaName)
			if err != nil {
				return err
			}

			formatter := output.NewFormatter(os.Stdout)
			formatter.ProjectCreated(project)
			return nil
		},
	}

	cmd.Flags().StringVar(&areaName, "area", "", "Assign to area")

	return cmd
}

func newProjectDeleteCmd(deps *Dependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			project, err := deps.ProjectService.Delete(args[0])
			if err != nil {
				return err
			}

			formatter := output.NewFormatter(os.Stdout)
			formatter.ProjectDeleted(project)
			return nil
		},
	}
}
