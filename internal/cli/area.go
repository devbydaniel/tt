package cli

import (
	"os"

	"github.com/devbydaniel/tt/internal/output"
	"github.com/spf13/cobra"
)

func NewAreaCmd(deps *Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "area",
		Short: "Manage areas",
	}

	cmd.AddCommand(newAreaListCmd(deps))
	cmd.AddCommand(newAreaAddCmd(deps))
	cmd.AddCommand(newAreaDeleteCmd(deps))
	cmd.AddCommand(newAreaRenameCmd(deps))

	return cmd
}

func newAreaListCmd(deps *Dependencies) *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all areas",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			areas, err := deps.App.ListAreas.Execute()
			if err != nil {
				return err
			}

			if jsonOutput {
				return output.WriteJSON(os.Stdout, areas)
			}

			formatter := output.NewFormatter(os.Stdout, deps.Theme)
			formatter.AreaList(areas)
			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	return cmd
}

func newAreaAddCmd(deps *Dependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "add <name>",
		Short: "Create a new area",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			area, err := deps.App.CreateArea.Execute(args[0])
			if err != nil {
				return err
			}

			formatter := output.NewFormatter(os.Stdout, deps.Theme)
			formatter.AreaCreated(area)
			return nil
		},
	}
}

func newAreaDeleteCmd(deps *Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete an area",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			area, err := deps.App.DeleteArea.Execute(args[0])
			if err != nil {
				return err
			}

			formatter := output.NewFormatter(os.Stdout, deps.Theme)
			formatter.AreaDeleted(area)
			return nil
		},
	}

	// Register area name completion
	registry := NewCompletionRegistry(deps)
	cmd.ValidArgsFunction = registry.AreaCompletion()

	return cmd
}

func newAreaRenameCmd(deps *Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rename <old-name> <new-name>",
		Short: "Rename an area",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			oldName := args[0]
			newName := args[1]

			_, err := deps.App.RenameArea.Execute(oldName, newName)
			if err != nil {
				return err
			}

			formatter := output.NewFormatter(os.Stdout, deps.Theme)
			formatter.AreaRenamed(oldName, newName)
			return nil
		},
	}

	// Register area name completion for first argument
	registry := NewCompletionRegistry(deps)
	cmd.ValidArgsFunction = registry.AreaCompletion()

	return cmd
}
