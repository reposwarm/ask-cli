package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the ask CLI version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("ask %s\n", version)
	},
}
