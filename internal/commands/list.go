package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/reposwarm/ask/internal/client"
	"github.com/reposwarm/ask/internal/output"
)

var (
	listStatus string
	listLimit  int
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List previous questions and their status",
	Long: `List questions submitted to askbox.

Examples:
  ask list
  ask list --status completed
  ask list --status running --limit 5`,
	RunE: func(cmd *cobra.Command, args []string) error {
		serverURL := getServerURL(cmd)
		c := client.New(serverURL)

		jobs, err := c.ListJobs(listStatus, listLimit)
		if err != nil {
			output.Error(err.Error(), "")
			return err
		}

		if output.JSONMode {
			return output.JSON(jobs)
		}

		if len(jobs) == 0 {
			fmt.Println("No questions found.")
			return nil
		}

		fmt.Printf("📋 Questions (%d)\n\n", len(jobs))
		for _, j := range jobs {
			icon := statusIcon(j.Status)
			q := j.Question
			if len(q) > 60 {
				q = q[:57] + "..."
			}
			chars := ""
			if j.Answer != "" {
				chars = fmt.Sprintf(" (%d chars)", len(j.Answer))
			}
			fmt.Printf("  %s %-10s %s  %s%s\n", icon, j.Status, j.ID[:8], q, chars)
		}
		return nil
	},
}

func init() {
	listCmd.Flags().StringVar(&listStatus, "status", "", "Filter by status (queued, running, completed, failed)")
	listCmd.Flags().IntVar(&listLimit, "limit", 0, "Limit number of results")
}

func statusIcon(status string) string {
	switch status {
	case "completed":
		return "✅"
	case "failed":
		return "❌"
	case "running":
		return "⏳"
	case "queued":
		return "📥"
	default:
		return "❓"
	}
}
