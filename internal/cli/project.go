package cli

import (
	"errors"
	"os"
	"time"

	"github.com/devbydaniel/tt/internal/dateparse"
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
	cmd.AddCommand(newProjectDoCmd(deps))
	cmd.AddCommand(newProjectUndoCmd(deps))
	cmd.AddCommand(newProjectEditCmd(deps))

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

func newProjectDoCmd(deps *Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "do <name>",
		Short: "Mark a project as complete",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Look up project by name
			project, err := deps.App.GetProjectByName.Execute(args[0])
			if err != nil {
				return err
			}

			// Complete the project (and its children)
			completed, err := deps.App.CompleteTasks.Execute([]int64{project.ID})
			if err != nil {
				return err
			}

			formatter := output.NewFormatter(os.Stdout, deps.Theme)
			formatter.ProjectsCompleted(completed)
			return nil
		},
	}

	// Register project name completion
	registry := NewCompletionRegistry(deps)
	cmd.ValidArgsFunction = registry.ProjectCompletion()

	return cmd
}

func newProjectUndoCmd(deps *Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "undo <name>",
		Short: "Mark a project as not complete",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Look up project by name
			project, err := deps.App.GetProjectByName.Execute(args[0])
			if err != nil {
				return err
			}

			// Uncomplete the project
			uncompleted, err := deps.App.UncompleteTasks.Execute([]int64{project.ID})
			if err != nil {
				return err
			}

			formatter := output.NewFormatter(os.Stdout, deps.Theme)
			formatter.ProjectsUncompleted(uncompleted)
			return nil
		},
	}

	// Register project name completion
	registry := NewCompletionRegistry(deps)
	cmd.ValidArgsFunction = registry.ProjectCompletion()

	return cmd
}

func newProjectEditCmd(deps *Dependencies) *cobra.Command {
	var title string
	var description string
	var areaName string
	var plannedStr string
	var dueStr string
	var today bool
	var addTags []string
	var removeTags []string
	var clearPlanned bool
	var clearDue bool
	var clearArea bool
	var clearDescription bool
	var someday bool
	var active bool

	cmd := &cobra.Command{
		Use:   "edit <name>",
		Short: "Edit a project",
		Long: `Edit project properties.

Examples:
  tt project edit "My Project" --title "New Title"
  tt project edit "My Project" --area Work
  tt project edit "My Project" --due tomorrow
  tt project edit "My Project" --planned +3d
  tt project edit "My Project" --tag urgent --tag priority
  tt project edit "My Project" --untag old-tag
  tt project edit "My Project" --clear-area
  tt project edit "My Project" --someday
  tt project edit "My Project" --active`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectName := args[0]

			// Look up project by name
			project, err := deps.App.GetProjectByName.Execute(projectName)
			if err != nil {
				return err
			}

			// Validate mutual exclusivity
			if areaName != "" && clearArea {
				return errors.New("cannot specify both --area and --clear-area")
			}
			if plannedStr != "" && clearPlanned {
				return errors.New("cannot specify both --planned and --clear-planned")
			}
			if today && plannedStr != "" {
				return errors.New("cannot specify both --today and --planned")
			}
			if today && clearPlanned {
				return errors.New("cannot specify both --today and --clear-planned")
			}
			if today {
				plannedStr = "today"
			}
			if dueStr != "" && clearDue {
				return errors.New("cannot specify both --due and --clear-due")
			}
			if description != "" && clearDescription {
				return errors.New("cannot specify both --description and --clear-description")
			}

			formatter := output.NewFormatter(os.Stdout, deps.Theme)

			// If no changes specified, show details
			hasChanges := title != "" || description != "" || areaName != "" ||
				plannedStr != "" || dueStr != "" || today || clearPlanned || clearDue ||
				clearArea || clearDescription || len(addTags) > 0 || len(removeTags) > 0 ||
				someday || active

			if !hasChanges {
				formatter.ProjectDetails(project)
				return nil
			}

			// Build changes list
			var changes []string
			if title != "" {
				changes = append(changes, "title")
			}
			if description != "" {
				changes = append(changes, "description")
			} else if clearDescription {
				changes = append(changes, "description cleared")
			}
			if areaName != "" {
				changes = append(changes, "area")
			} else if clearArea {
				changes = append(changes, "area cleared")
			}
			if plannedStr != "" {
				changes = append(changes, "planned date")
			} else if clearPlanned {
				changes = append(changes, "planned date cleared")
			}
			if dueStr != "" {
				changes = append(changes, "due date")
			} else if clearDue {
				changes = append(changes, "due date cleared")
			}
			if len(addTags) > 0 {
				changes = append(changes, "tags added")
			}
			if len(removeTags) > 0 {
				changes = append(changes, "tags removed")
			}
			if someday {
				changes = append(changes, "moved to someday")
			}
			if active {
				changes = append(changes, "moved to active")
			}

			// Apply changes
			if title != "" {
				if _, err := deps.App.SetTaskTitle.Execute(project.ID, title); err != nil {
					return err
				}
			}

			if description != "" {
				if _, err := deps.App.SetTaskDescription.Execute(project.ID, &description); err != nil {
					return err
				}
			} else if clearDescription {
				if _, err := deps.App.SetTaskDescription.Execute(project.ID, nil); err != nil {
					return err
				}
			}

			if areaName != "" {
				if _, err := deps.App.SetTaskArea.Execute(project.ID, areaName); err != nil {
					return err
				}
			} else if clearArea {
				if _, err := deps.App.SetTaskArea.Execute(project.ID, ""); err != nil {
					return err
				}
			}

			if plannedStr != "" {
				planned, err := dateparse.Parse(plannedStr)
				if err != nil {
					return err
				}
				if _, err := deps.App.SetPlannedDate.Execute(project.ID, &planned); err != nil {
					return err
				}
			} else if clearPlanned {
				if _, err := deps.App.SetPlannedDate.Execute(project.ID, nil); err != nil {
					return err
				}
			}

			if dueStr != "" {
				due, err := dateparse.Parse(dueStr)
				if err != nil {
					return err
				}
				if _, err := deps.App.SetDueDate.Execute(project.ID, &due); err != nil {
					return err
				}
			} else if clearDue {
				if _, err := deps.App.SetDueDate.Execute(project.ID, nil); err != nil {
					return err
				}
			}

			for _, tag := range addTags {
				if _, err := deps.App.AddTag.Execute(project.ID, tag); err != nil {
					return err
				}
			}

			for _, tag := range removeTags {
				if _, err := deps.App.RemoveTag.Execute(project.ID, tag); err != nil {
					return err
				}
			}

			if someday {
				if _, err := deps.App.DeferTask.Execute(project.ID); err != nil {
					return err
				}
			}

			if active {
				if _, err := deps.App.ActivateTask.Execute(project.ID); err != nil {
					return err
				}
			}

			formatter.ProjectEdited(projectName, changes)
			return nil
		},
	}

	cmd.Flags().StringVar(&title, "title", "", "Set project title")
	cmd.Flags().StringVarP(&description, "description", "d", "", "Set project description")
	cmd.Flags().StringVarP(&areaName, "area", "a", "", "Assign to area")
	cmd.Flags().StringVarP(&plannedStr, "planned", "P", "", "Set planned date")
	cmd.Flags().BoolVarP(&today, "today", "T", false, "Set planned date to today")
	cmd.Flags().StringVarP(&dueStr, "due", "D", "", "Set due date")
	cmd.Flags().StringArrayVarP(&addTags, "tag", "t", nil, "Add tag (repeatable)")
	cmd.Flags().StringArrayVar(&removeTags, "untag", nil, "Remove tag (repeatable)")
	cmd.Flags().BoolVar(&clearPlanned, "clear-planned", false, "Clear planned date")
	cmd.Flags().BoolVar(&clearDue, "clear-due", false, "Clear due date")
	cmd.Flags().BoolVar(&clearArea, "clear-area", false, "Remove from area")
	cmd.Flags().BoolVar(&clearDescription, "clear-description", false, "Clear description")
	cmd.Flags().BoolVarP(&someday, "someday", "s", false, "Move to someday")
	cmd.Flags().BoolVarP(&active, "active", "A", false, "Move to active")
	cmd.MarkFlagsMutuallyExclusive("someday", "active")

	// Register completions (use AllProjectCompletion to include someday projects)
	registry := NewCompletionRegistry(deps)
	cmd.ValidArgsFunction = registry.AllProjectCompletion()
	registry.RegisterAreaFlag(cmd)

	return cmd
}
