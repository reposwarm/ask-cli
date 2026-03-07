package commands

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/reposwarm/ask/internal/config"
	"github.com/reposwarm/ask/internal/output"
)

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Start the local askbox server",
	Long: `Start the askbox Docker container.

Requires prior setup via 'ask setup'. Uses the docker-compose.yml in ~/.ask/.

Examples:
  ask up`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dataDir := config.DataDir()
		composePath := dataDir + "/docker-compose.yml"
		if _, err := os.Stat(composePath); os.IsNotExist(err) {
			output.Error("No local askbox configured", "Run: ask setup")
			return fmt.Errorf("no docker-compose.yml found at %s", composePath)
		}

		output.Info("🐳 Starting askbox...")
		c := exec.Command("docker", "compose", "up", "-d")
		c.Dir = dataDir
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		if err := c.Run(); err != nil {
			output.Error("Failed to start", "Check Docker is running")
			return err
		}
		output.Success("Askbox started")
		return nil
	},
}

var downCmd = &cobra.Command{
	Use:   "down",
	Short: "Stop the local askbox server",
	RunE: func(cmd *cobra.Command, args []string) error {
		dataDir := config.DataDir()
		output.Info("🛑 Stopping askbox...")
		c := exec.Command("docker", "compose", "down")
		c.Dir = dataDir
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		if err := c.Run(); err != nil {
			output.Error("Failed to stop", "")
			return err
		}
		output.Success("Askbox stopped")
		return nil
	},
}

var logsFollowFlag bool

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Show askbox server logs",
	RunE: func(cmd *cobra.Command, args []string) error {
		dataDir := config.DataDir()
		cmdArgs := []string{"compose", "logs"}
		if logsFollowFlag {
			cmdArgs = append(cmdArgs, "-f")
		}
		cmdArgs = append(cmdArgs, "askbox")
		c := exec.Command("docker", cmdArgs...)
		c.Dir = dataDir
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		return c.Run()
	},
}

func init() {
	logsCmd.Flags().BoolVarP(&logsFollowFlag, "follow", "f", false, "Follow log output")

	rootCmd.AddCommand(upCmd)
	rootCmd.AddCommand(downCmd)
	rootCmd.AddCommand(logsCmd)
}
