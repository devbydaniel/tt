package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	commentusecases "github.com/devbydaniel/tt/internal/domain/comment/usecases"
	"github.com/devbydaniel/tt/internal/output"
)

// NewCommentCmd builds the `tt comment` command tree.
func NewCommentCmd(deps *Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "comment",
		Aliases: []string{"comments"},
		Short:   "Manage task comments",
	}

	cmd.AddCommand(newCommentAddCmd(deps))
	cmd.AddCommand(newCommentListCmd(deps))

	return cmd
}

func newCommentAddCmd(deps *Dependencies) *cobra.Command {
	var taskID int64
	var author string
	var bodyFile string

	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a comment to a task",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			var body string

			if bodyFile != "" {
				if bodyFile == "-" {
					b, err := io.ReadAll(os.Stdin)
					if err != nil {
						return err
					}
					body = string(b)
				} else {
					b, err := os.ReadFile(bodyFile)
					if err != nil {
						return err
					}
					body = string(b)
				}
			} else {
				body = strings.Join(args, " ")
			}

			body = strings.TrimSpace(body)
			if body == "" {
				return fmt.Errorf("comment body is required")
			}

			c, err := deps.App.AddComment.Execute(commentusecases.AddOptions{
				TaskID: taskID,
				Author: author,
				Body:   body,
			})
			if err != nil {
				return err
			}

			fmt.Fprintln(os.Stdout, c.ID)
			return nil
		},
	}

	cmd.Flags().Int64Var(&taskID, "task", 0, "Task ID (required)")
	_ = cmd.MarkFlagRequired("task")
	cmd.Flags().StringVar(&author, "author", "user", "Comment author")
	cmd.Flags().StringVar(&bodyFile, "body-file", "", "Read body from file (use - for stdin)")

	return cmd
}

func newCommentListCmd(deps *Dependencies) *cobra.Command {
	var taskID int64
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "ls",
		Short: "List comments for a task",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			comments, err := deps.App.ListComments.Execute(taskID)
			if err != nil {
				return err
			}

			if jsonOutput {
				return output.WriteJSON(os.Stdout, comments)
			}

			for i, c := range comments {
				if i > 0 {
					fmt.Fprintln(os.Stdout)
				}
				fmt.Fprintf(os.Stdout, "--- %s @ %s ---\n", c.Author, c.CreatedAt.Format("Jan 2, 2006 3:04 PM"))
				fmt.Fprintln(os.Stdout, c.Body)
			}
			return nil
		},
	}

	cmd.Flags().Int64Var(&taskID, "task", 0, "Task ID (required)")
	_ = cmd.MarkFlagRequired("task")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")

	return cmd
}
