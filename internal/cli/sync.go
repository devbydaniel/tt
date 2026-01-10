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
		Short: "Sync management commands",
	}

	pushCmd := &cobra.Command{
		Use:   "push",
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

	resetCmd := &cobra.Command{
		Use:   "reset",
		Short: "Clear all local sync events",
		RunE: func(cmd *cobra.Command, args []string) error {
			if deps.App.ResetSync == nil {
				return fmt.Errorf("sync not configured: set client_id in config file")
			}

			fmt.Print("This will delete all local sync events. Continue? [y/N]: ")
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

			count, err := deps.App.ResetSync.Execute()
			if err != nil {
				return fmt.Errorf("reset failed: %w", err)
			}

			fmt.Fprintf(os.Stdout, "Deleted %d sync event(s).\n", count)
			return nil
		},
	}

	cmd.AddCommand(pushCmd, resetCmd)

	// Make "push" the default when running "tt sync" with no subcommand
	cmd.RunE = pushCmd.RunE

	return cmd
}
