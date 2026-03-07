package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/reposwarm/ask/internal/config"
	"github.com/reposwarm/ask/internal/output"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage ask configuration",
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Load()

		if output.JSONMode {
			return output.JSON(cfg)
		}

		fmt.Printf("⚙️  Configuration (%s)\n\n", config.Path())
		fmt.Printf("  serverUrl:  %s\n", cfg.ServerURL)
		fmt.Printf("  adapter:    %s\n", valueOrDefault(cfg.Adapter, "(default)"))
		fmt.Printf("  model:      %s\n", valueOrDefault(cfg.Model, "(default)"))
		return nil
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Long: fmt.Sprintf(`Set a configuration value.

Valid keys: %v

Examples:
  ask config set serverUrl http://askbox.internal:8082
  ask config set adapter strands
  ask config set model us.anthropic.claude-sonnet-4-20250514-v1:0`, config.ValidKeys),
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key, value := args[0], args[1]
		cfg := config.Load()

		if err := cfg.Set(key, value); err != nil {
			output.Error(err.Error(), "")
			return err
		}

		if err := config.Save(cfg); err != nil {
			output.Error(err.Error(), "")
			return err
		}

		output.Success(fmt.Sprintf("Set %s = %s", key, value))
		return nil
	},
}

func init() {
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configSetCmd)
}

func valueOrDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}
