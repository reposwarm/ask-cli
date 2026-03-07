package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/reposwarm/ask/internal/client"
	"github.com/reposwarm/ask/internal/output"
)

var getCmd = &cobra.Command{
	Use:   "get <job-id>",
	Short: "Get the result of a previous question",
	Long: `Retrieve the answer for a previously submitted question.

If the job is still running, polls until completion (unless --no-wait).

Examples:
  ask get abc12345
  ask get abc12345 --json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		serverURL := getServerURL(cmd)
		c := client.New(serverURL)
		id := args[0]

		job, err := c.GetJob(id)
		if err != nil {
			output.Error(err.Error(), "")
			return err
		}

		// If still running or queued, poll
		if job.Status == "running" || job.Status == "queued" {
			if noWaitFlag {
				if output.JSONMode {
					return output.JSON(job)
				}
				fmt.Printf("⏳ Job %s is %s (%d tool calls)\n", id, job.Status, job.ToolCalls)
				return nil
			}
			output.Info(fmt.Sprintf("⏳ Job %s is %s, waiting...", id, job.Status))
			return pollJob(c, id)
		}

		// Completed or failed
		if output.JSONMode {
			return output.JSON(job)
		}

		if job.Status == "completed" {
			fmt.Printf("✅ Job %s — %d tool calls\n\n", id, job.ToolCalls)
			fmt.Println(job.Answer)
		} else if job.Status == "failed" {
			fmt.Printf("❌ Job %s failed: %s\n", id, job.Error)
		}

		return nil
	},
}
