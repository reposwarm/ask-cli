package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/reposwarm/ask-cli/internal/client"
	"github.com/reposwarm/ask-cli/internal/output"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check askbox server status",
	RunE: func(cmd *cobra.Command, args []string) error {
		serverURL := getServerURL(cmd)
		c := client.New(serverURL)

		health, err := c.Health()
		if err != nil {
			output.Error(err.Error(), fmt.Sprintf("Is askbox running at %s?", serverURL))
			return err
		}

		if output.JSONMode {
			return output.JSON(health)
		}

		archStatus := "❌ not loaded"
		if health.ArchHubReady && health.ArchHubRepos > 0 {
			archStatus = fmt.Sprintf("✅ ready (%d repos)", health.ArchHubRepos)
		} else if health.ArchHubReady && health.ArchHubRepos == 0 {
			archStatus = "⚠️  loaded but empty (0 repos — no .arch.md files found)"
		}

		fmt.Printf("🏥 Askbox Server Status\n\n")
		fmt.Printf("  Server:    %s\n", serverURL)
		fmt.Printf("  Status:    %s\n", health.Status)
		fmt.Printf("  Arch-hub:  %s\n", archStatus)
		fmt.Printf("  Jobs:      %d total, %d running\n", health.JobsTotal, health.JobsRunning)
		fmt.Printf("  Uptime:    %.0fs\n", health.Uptime)
		return nil
	},
}
