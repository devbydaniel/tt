package cli

import (
	"os"

	"github.com/devbydaniel/tt/internal/output"
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
	cmd.AddCommand(newProjectRenameCmd(deps))
	cmd.AddCommand(newProjectMoveCmd(deps))

	return cmd
}

func newProjectListCmd(deps *Dependencies) *cobra.Command {
	var group string
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all projects",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Use flag if provided, otherwise use config
			groupBy := group
			if groupBy == "" {
				groupBy = deps.Config.Grouping.GetForCommand("project-list")
			}

			if groupBy == "area" {
				projects, err := deps.ProjectService.ListWithArea()
				if err != nil {
					return err
				}
				if jsonOutput {
					return output.WriteJSON(os.Stdout, projects)
				}
				formatter := output.NewFormatter(os.Stdout, deps.Theme)
				formatter.ProjectListGrouped(projects, groupBy)
				return nil
			}

			projects, err := deps.ProjectService.List()
			if err != nil {
				return err
			}
			if jsonOutput {
				return output.WriteJSON(os.Stdout, projects)
			}
			formatter := output.NewFormatter(os.Stdout, deps.Theme)
			formatter.ProjectList(projects)
			return nil
		},
	}

	cmd.Flags().StringVarP(&group, "group", "g", "", "Group projects by: area")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")

	return cmd
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

			formatter := output.NewFormatter(os.Stdout, deps.Theme)
			formatter.ProjectCreated(project)
			return nil
		},
	}

	cmd.Flags().StringVar(&areaName, "area", "", "Assign to area")

	// Register area completion
	registry := NewCompletionRegistry(deps)
	registry.RegisterAreaFlag(cmd)

	return cmd
}

func newProjectDeleteCmd(deps *Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			project, err := deps.ProjectService.Delete(args[0])
			if err != nil {
				return err
			}

			formatter := output.NewFormatter(os.Stdout, deps.Theme)
			formatter.ProjectDeleted(project)
			return nil
		},
	}

	// Register project name completion
	registry := NewCompletionRegistry(deps)
	cmd.ValidArgsFunction = registry.ProjectCompletion()

	return cmd
}

func newProjectRenameCmd(deps *Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rename <old-name> <new-name>",
		Short: "Rename a project",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			oldName := args[0]
			newName := args[1]

			_, err := deps.ProjectService.Rename(oldName, newName)
			if err != nil {
				return err
			}

			formatter := output.NewFormatter(os.Stdout, deps.Theme)
			formatter.ProjectRenamed(oldName, newName)
			return nil
		},
	}

	// Register project name completion for first argument
	registry := NewCompletionRegistry(deps)
	cmd.ValidArgsFunction = registry.ProjectCompletion()

	return cmd
}

func newProjectMoveCmd(deps *Dependencies) *cobra.Command {
	var areaName string
	var clearArea bool

	cmd := &cobra.Command{
		Use:   "move <name>",
		Short: "Move a project to an area or clear its area",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectName := args[0]
			formatter := output.NewFormatter(os.Stdout, deps.Theme)

			if clearArea {
				project, err := deps.ProjectService.ClearArea(projectName)
				if err != nil {
					return err
				}
				formatter.ProjectAreaCleared(project)
				return nil
			}

			if areaName == "" {
				return cmd.Help()
			}

			project, err := deps.ProjectService.SetArea(projectName, areaName)
			if err != nil {
				return err
			}
			formatter.ProjectMoved(project, areaName)
			return nil
		},
	}

	cmd.Flags().StringVar(&areaName, "area", "", "Move to area")
	cmd.Flags().BoolVar(&clearArea, "clear", false, "Clear area assignment")
	cmd.MarkFlagsMutuallyExclusive("area", "clear")

	// Register completions
	registry := NewCompletionRegistry(deps)
	cmd.ValidArgsFunction = registry.ProjectCompletion()
	registry.RegisterAreaFlag(cmd)

	return cmd
}
