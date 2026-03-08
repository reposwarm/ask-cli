package commands

import (
	"github.com/spf13/cobra"

	"github.com/reposwarm/ask-cli/internal/output"
)

var version = "dev"

// SetVersion is called from main to inject the build version.
func SetVersion(v string) {
	version = v
}

var rootCmd = &cobra.Command{
	Use:     "ask",
	Short:   "Query architecture knowledge from your codebase",
	Version: version,
	Long: `ask — the read side of RepoSwarm.

Ask questions about your codebase architecture using AI. Queries an askbox
server that holds your .arch.md files and uses LLMs to reason across repos.

Quick start:
  ask setup                                    # Set up local askbox
  ask "how does auth work across services?"    # Ask a question

Usage:
  ask "how does auth work across services?"
  ask --repos api,billing "how do they communicate?"
  ask list --status completed
  ask status
  ask refresh --url https://github.com/org/arch-hub.git

Docker management:
  ask setup        Set up local askbox with provider config
  ask up           Start the local askbox container
  ask down         Stop it
  ask logs -f      Tail logs`,
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&output.JSONMode, "json", false, "Output as JSON")
	rootCmd.PersistentFlags().BoolVar(&output.AgentMode, "for-agent", false, "Plain text output (no colors/spinners)")
	rootCmd.PersistentFlags().StringP("server", "s", "", "Askbox server URL (overrides config)")

	rootCmd.AddCommand(askCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(getCmd)
	rootCmd.AddCommand(refreshCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(versionCmd)
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

// getServerURL resolves the server URL from flag → env → config.
func getServerURL(cmd *cobra.Command) string {
	if s, _ := cmd.Flags().GetString("server"); s != "" {
		return s
	}
	cfg := loadConfig()
	return cfg.ServerURL
}
