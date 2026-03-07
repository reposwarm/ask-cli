package main

import (
	"fmt"
	"os"

	"github.com/reposwarm/ask-cli/internal/commands"
)

// Version is set at build time via -ldflags
var Version = "dev"

func main() {
	commands.SetVersion(Version)
	if err := commands.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
