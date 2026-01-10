package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func NewSyncCmd(deps *Dependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Push local sync events to the remote server",
		RunE: func(cmd *cobra.Command, args []string) error {
			if deps.App.PushEvents == nil {
				return fmt.Errorf("sync not configured: set sync.url and sync.api_key in config file")
			}

			fmt.Fprintln(os.Stdout, "Syncing events to server...")

			result, err := deps.App.PushEvents.Execute()
			if err != nil {
				return fmt.Errorf("sync failed: %w", err)
			}

			if result.Pushed == 0 && result.Rejected == 0 {
				fmt.Fprintln(os.Stdout, "No events to sync.")
				return nil
			}

			if result.Pushed > 0 {
				fmt.Fprintf(os.Stdout, "Pushed %d event(s) successfully.\n", result.Pushed)
			}

			if result.Rejected > 0 {
				fmt.Fprintf(os.Stderr, "%d event(s) were rejected:\n", result.Rejected)
				for _, errMsg := range result.Errors {
					fmt.Fprintln(os.Stderr, "  - "+errMsg)
				}
			}

			return nil
		},
	}
}
