package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func NewSyncCmd(deps *Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync with remote server (bidirectional)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if deps.App.SyncEvents == nil {
				return fmt.Errorf("sync not configured: set sync.url and sync.api_key in config file")
			}

			fmt.Fprintln(os.Stdout, "Syncing with server...")

			result, err := deps.App.SyncEvents.Execute()
			if err != nil {
				return fmt.Errorf("sync failed: %w", err)
			}

			if result.Pushed == 0 && result.Pulled == 0 && result.Rejected == 0 {
				fmt.Fprintln(os.Stdout, "Already up to date.")
				return nil
			}

			if result.Pushed > 0 {
				fmt.Fprintf(os.Stdout, "Pushed %d event(s).\n", result.Pushed)
			}

			if result.Applied > 0 {
				fmt.Fprintf(os.Stdout, "Applied %d change(s) from server.\n", result.Applied)
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

	pushCmd := &cobra.Command{
		Use:   "push",
		Short: "Push local sync events to the remote server (push only, no pull)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if deps.App.PushEvents == nil {
				return fmt.Errorf("sync not configured: set sync.url and sync.api_key in config file")
			}

			fmt.Fprintln(os.Stdout, "Pushing events to server...")

			result, err := deps.App.PushEvents.Execute()
			if err != nil {
				return fmt.Errorf("push failed: %w", err)
			}

			if result.Pushed == 0 && result.Rejected == 0 {
				fmt.Fprintln(os.Stdout, "No events to push.")
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

	resetCmd := &cobra.Command{
		Use:   "reset",
		Short: "Reset sync: clear server data, regenerate events from local database",
		RunE: func(cmd *cobra.Command, args []string) error {
			if deps.App.ResetSync == nil {
				return fmt.Errorf("sync not configured: set client_id in config file")
			}

			fmt.Fprintln(os.Stdout, "This will:")
			fmt.Fprintln(os.Stdout, "  1. Clear all data on the sync server")
			fmt.Fprintln(os.Stdout, "  2. Clear local sync events and cursor")
			fmt.Fprintln(os.Stdout, "  3. Regenerate sync events for all local tasks and areas")
			fmt.Print("Continue? [y/N]: ")
			reader := bufio.NewReader(os.Stdin)
			response, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("failed to read input: %w", err)
			}
			response = strings.TrimSpace(strings.ToLower(response))
			if response != "y" && response != "yes" {
				fmt.Fprintln(os.Stdout, "Aborted.")
				return nil
			}

			result, err := deps.App.ResetSync.Execute()
			if err != nil {
				return fmt.Errorf("reset failed: %w", err)
			}

			fmt.Fprintf(os.Stdout, "Deleted %d sync event(s).\n", result.DeletedEvents)
			fmt.Fprintf(os.Stdout, "Regenerated %d sync event(s).\n", result.RegeneratedEvents)
			fmt.Fprintln(os.Stdout, "Run 'tt sync' to push to the server.")
			return nil
		},
	}

	cmd.AddCommand(pushCmd, resetCmd)

	return cmd
}
