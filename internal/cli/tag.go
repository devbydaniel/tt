package cli

import (
	"errors"
	"os"
	"strconv"

	"github.com/devbydaniel/t/internal/output"
	"github.com/spf13/cobra"
)

func NewTagCmd(deps *Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tag",
		Short: "Manage tags",
	}

	cmd.AddCommand(newTagListCmd(deps))
	cmd.AddCommand(newTagAddCmd(deps))
	cmd.AddCommand(newTagRemoveCmd(deps))

	return cmd
}

func newTagListCmd(deps *Dependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all tags in use",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			tags, err := deps.TaskService.ListTags()
			if err != nil {
				return err
			}

			formatter := output.NewFormatter(os.Stdout)
			formatter.TagList(tags)
			return nil
		},
	}
}

func newTagAddCmd(deps *Dependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "add <task-id> <tag-name>",
		Short: "Add a tag to a task",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return errors.New("invalid task ID")
			}

			tagName := args[1]
			if tagName == "" {
				return errors.New("tag name cannot be empty")
			}

			t, err := deps.TaskService.AddTag(id, tagName)
			if err != nil {
				return err
			}

			formatter := output.NewFormatter(os.Stdout)
			formatter.TaskTagAdded(t, tagName)
			return nil
		},
	}
}

func newTagRemoveCmd(deps *Dependencies) *cobra.Command {
	return &cobra.Command{
		Use:     "remove <task-id> <tag-name>",
		Aliases: []string{"rm"},
		Short:   "Remove a tag from a task",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return errors.New("invalid task ID")
			}

			tagName := args[1]
			if tagName == "" {
				return errors.New("tag name cannot be empty")
			}

			t, err := deps.TaskService.RemoveTag(id, tagName)
			if err != nil {
				return err
			}

			formatter := output.NewFormatter(os.Stdout)
			formatter.TaskTagRemoved(t, tagName)
			return nil
		},
	}
}
