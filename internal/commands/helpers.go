package commands

import (
	"github.com/reposwarm/ask-cli/internal/config"
)

// loadConfig is a helper used by commands.
func loadConfig() *config.Config {
	return config.Load()
}
