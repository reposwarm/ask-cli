package commands

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/reposwarm/ask/internal/client"
	"github.com/reposwarm/ask/internal/output"
)

var (
	reposFlag   string
	adapterFlag string
	modelFlag   string
	noWaitFlag  bool
)

var askCmd = &cobra.Command{
	Use:   "ask [question]",
	Short: "Ask an architecture question",
	Long: `Submit a question about your codebase architecture.

The question is sent to the askbox server, which uses AI to analyze
your .arch.md files and return a detailed answer.

Examples:
  ask "how does authentication work?"
  ask --repos api,billing "how do these services communicate?"
  ask --adapter strands "what databases are used?"
  ask --no-wait --json "summarize the architecture"`,
	Args:                  cobra.MinimumNArgs(1),
	DisableFlagsInUseLine: true,
	RunE:                  runAsk,
}

func init() {
	askCmd.Flags().StringVar(&reposFlag, "repos", "", "Comma-separated list of repos to scope the question to")
	askCmd.Flags().StringVar(&adapterFlag, "adapter", "", "LLM adapter (claude-agent, strands)")
	askCmd.Flags().StringVar(&modelFlag, "model", "", "Model override")
	askCmd.Flags().BoolVar(&noWaitFlag, "no-wait", false, "Submit and return immediately (don't poll for answer)")

	// Make "ask" the default command — bare `ask "question"` works
	rootCmd.Args = askCmd.Args
	rootCmd.RunE = runAsk
	rootCmd.Flags().StringVar(&reposFlag, "repos", "", "Comma-separated list of repos to scope the question to")
	rootCmd.Flags().StringVar(&adapterFlag, "adapter", "", "LLM adapter (claude-agent, strands)")
	rootCmd.Flags().StringVar(&modelFlag, "model", "", "Model override")
	rootCmd.Flags().BoolVar(&noWaitFlag, "no-wait", false, "Submit and return immediately (don't poll for answer)")
}

func runAsk(cmd *cobra.Command, args []string) error {
	question := strings.Join(args, " ")
	serverURL := getServerURL(cmd)
	cfg := loadConfig()

	c := client.New(serverURL)

	// Build request
	req := &client.AskRequest{
		Question: question,
	}
	if reposFlag != "" {
		req.Repos = strings.Split(reposFlag, ",")
	}
	if adapterFlag != "" {
		req.Adapter = adapterFlag
	} else if cfg.Adapter != "" {
		req.Adapter = cfg.Adapter
	}
	if modelFlag != "" {
		req.Model = modelFlag
	} else if cfg.Model != "" {
		req.Model = cfg.Model
	}

	// Submit
	output.Info(fmt.Sprintf("🔍 Asking: %s", question))
	resp, err := c.Ask(req)
	if err != nil {
		output.Error(err.Error(), "Is askbox running? Check: ask status")
		return err
	}

	output.Info(fmt.Sprintf("📋 Job %s submitted", resp.ID))

	if noWaitFlag {
		if output.JSONMode {
			return output.JSON(map[string]any{
				"success": true,
				"id":      resp.ID,
				"status":  resp.Status,
			})
		}
		fmt.Printf("Job ID: %s\nStatus: %s\nPoll:   ask get %s\n", resp.ID, resp.Status, resp.ID)
		return nil
	}

	// Poll until done
	return pollJob(c, resp.ID)
}

func pollJob(c *client.Client, id string) error {
	start := time.Now()
	attempt := 0

	for {
		time.Sleep(3 * time.Second)
		attempt++

		job, err := c.GetJob(id)
		if err != nil {
			output.Error(err.Error(), "")
			return err
		}

		switch job.Status {
		case "completed":
			elapsed := time.Since(start).Round(time.Second)
			output.StatusLine("") // clear status line

			if output.JSONMode {
				return output.JSON(map[string]any{
					"success":   true,
					"id":        job.ID,
					"status":    job.Status,
					"answer":    job.Answer,
					"toolCalls": job.ToolCalls,
					"duration":  elapsed.Seconds(),
				})
			}

			fmt.Printf("\n✅ Answered in %s (%d tool calls)\n\n", elapsed, job.ToolCalls)
			fmt.Println(job.Answer)
			return nil

		case "failed":
			output.StatusLine("")
			errMsg := job.Error
			if errMsg == "" {
				errMsg = "unknown error"
			}
			output.Error(fmt.Sprintf("Job %s failed: %s", id, errMsg), "Check askbox logs")
			return fmt.Errorf("job failed: %s", errMsg)

		default:
			elapsed := time.Since(start).Round(time.Second)
			detail := ""
			if job.ToolCalls > 0 {
				detail = fmt.Sprintf(" (%d tool calls)", job.ToolCalls)
			}
			output.StatusLine(fmt.Sprintf("⏳ %s %s%s", job.Status, elapsed, detail))
		}
	}
}
