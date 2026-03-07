package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/reposwarm/ask-cli/internal/client"
	"github.com/reposwarm/ask-cli/internal/output"
)

var (
	refreshURL    string
	refreshBranch string
)

var refreshCmd = &cobra.Command{
	Use:   "refresh",
	Short: "Refresh the arch-hub (re-clone or pull latest)",
	Long: `Trigger the askbox to refresh its arch-hub content.

This re-clones or pulls the latest .arch.md files from the arch-hub git repo.

Examples:
  ask refresh
  ask refresh --url https://github.com/org/arch-hub.git
  ask refresh --branch develop`,
	RunE: func(cmd *cobra.Command, args []string) error {
		serverURL := getServerURL(cmd)
		c := client.New(serverURL)

		output.Info("🔄 Refreshing arch-hub...")

		if err := c.Refresh(refreshURL, refreshBranch); err != nil {
			output.Error(err.Error(), "")
			return err
		}

		// Show updated status
		health, err := c.Health()
		if err != nil {
			output.Success("Refresh triggered")
			return nil
		}

		if output.JSONMode {
			return output.JSON(map[string]any{
				"success": true,
				"repos":   health.ArchHubRepos,
			})
		}

		output.Success(fmt.Sprintf("Arch-hub refreshed — %d repos indexed", health.ArchHubRepos))
		return nil
	},
}

func init() {
	refreshCmd.Flags().StringVar(&refreshURL, "url", "", "Arch-hub git URL (overrides server default)")
	refreshCmd.Flags().StringVar(&refreshBranch, "branch", "", "Git branch to pull")
}
