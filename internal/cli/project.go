package cli

import (
	"os"
	"time"

	"github.com/devbydaniel/tt/internal/domain/task/usecases"
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
				groupBy = deps.Config.GetGroup("project-list")
			}

			projects, err := deps.App.ListProjectsWithArea.Execute()
			if err != nil {
				return err
			}
			if jsonOutput {
				return output.WriteJSON(os.Stdout, projects)
			}
			formatter := output.NewFormatter(os.Stdout, deps.Theme)
			if groupBy == "area" {
				formatter.ProjectListGrouped(projects, groupBy)
			} else {
				formatter.ProjectList(projects)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&group, "group", "g", "", "Group projects by: area")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")

	return cmd
}

func newProjectAddCmd(deps *Dependencies) *cobra.Command {
	var areaName string
	var plannedStr string
	var dueStr string
	var someday bool
	var description string

	cmd := &cobra.Command{
		Use:   "add <name>",
		Short: "Create a new project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := &usecases.CreateProjectOptions{
				AreaName:    areaName,
				Description: description,
				Someday:     someday,
			}

			if plannedStr != "" {
				t, err := parseDate(plannedStr)
				if err != nil {
					return err
				}
				opts.PlannedDate = &t
			}
			if dueStr != "" {
				t, err := parseDate(dueStr)
				if err != nil {
					return err
				}
				opts.DueDate = &t
			}

			project, err := deps.App.CreateProject.Execute(args[0], opts)
			if err != nil {
				return err
			}

			formatter := output.NewFormatter(os.Stdout, deps.Theme)
			formatter.ProjectCreated(project)
			return nil
		},
	}

	cmd.Flags().StringVar(&areaName, "area", "", "Assign to area")
	cmd.Flags().StringVarP(&plannedStr, "planned", "p", "", "Set planned date (YYYY-MM-DD or 'today', 'tomorrow', etc.)")
	cmd.Flags().StringVarP(&dueStr, "due", "d", "", "Set due date (YYYY-MM-DD or 'today', 'tomorrow', etc.)")
	cmd.Flags().BoolVarP(&someday, "someday", "s", false, "Create in someday state")
	cmd.Flags().StringVar(&description, "desc", "", "Set project description")

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
			// Look up project by name
			project, err := deps.App.GetProjectByName.Execute(args[0])
			if err != nil {
				return err
			}

			// Delete the project (and its children via cascade)
			_, err = deps.App.DeleteTasks.Execute([]int64{project.ID})
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

			// Look up project by name
			project, err := deps.App.GetProjectByName.Execute(oldName)
			if err != nil {
				return err
			}

			// Rename using SetTaskTitle
			_, err = deps.App.SetTaskTitle.Execute(project.ID, newName)
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

			// Look up project by name
			project, err := deps.App.GetProjectByName.Execute(projectName)
			if err != nil {
				return err
			}

			if clearArea {
				// Clear area using SetTaskArea with empty string
				project, err = deps.App.SetTaskArea.Execute(project.ID, "")
				if err != nil {
					return err
				}
				formatter.ProjectAreaCleared(project)
				return nil
			}

			if areaName == "" {
				return cmd.Help()
			}

			// Set area using SetTaskArea
			project, err = deps.App.SetTaskArea.Execute(project.ID, areaName)
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

// parseDate parses a date string in various formats
func parseDate(s string) (time.Time, error) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	switch s {
	case "today":
		return today, nil
	case "tomorrow":
		return today.AddDate(0, 0, 1), nil
	default:
		return time.Parse("2006-01-02", s)
	}
}
