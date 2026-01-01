package cli

import (
	"os"

	"github.com/devbydaniel/tt/internal/output"
	"github.com/spf13/cobra"
)

func NewTagsCmd(deps *Dependencies) *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "tags",
		Short: "List all tags alphabetically",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			tags, err := deps.App.ListTags.Execute()
			if err != nil {
				return err
			}

			if jsonOutput {
				return output.WriteJSON(os.Stdout, tags)
			}

			formatter := output.NewFormatter(os.Stdout, deps.Theme)
			formatter.TagList(tags)
			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	return cmd
}
