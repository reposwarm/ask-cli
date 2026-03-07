package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config holds the ask CLI configuration.
type Config struct {
	ServerURL string `json:"serverUrl"` // Askbox server URL (e.g. http://localhost:8082)
	Adapter   string `json:"adapter"`   // Default adapter (claude-agent, strands)
	Model     string `json:"model"`     // Default model override
}

// DefaultServerURL is the default askbox server address.
const DefaultServerURL = "http://localhost:8082"

// ValidKeys lists all config keys that can be set.
var ValidKeys = []string{"serverUrl", "adapter", "model"}

// configDir returns the config directory path.
func configDir() string {
	if d := os.Getenv("ASK_CONFIG_DIR"); d != "" {
		return d
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "ask")
}

// configPath returns the full path to the config file.
func configPath() string {
	return filepath.Join(configDir(), "config.json")
}

// Load reads the config file. Returns defaults if file doesn't exist.
func Load() *Config {
	cfg := &Config{
		ServerURL: DefaultServerURL,
	}

	// Environment overrides
	if u := os.Getenv("ASK_SERVER_URL"); u != "" {
		cfg.ServerURL = u
	}

	data, err := os.ReadFile(configPath())
	if err != nil {
		return cfg
	}

	var fileCfg Config
	if err := json.Unmarshal(data, &fileCfg); err != nil {
		return cfg
	}

	// File values override defaults
	if fileCfg.ServerURL != "" {
		cfg.ServerURL = fileCfg.ServerURL
	}
	if fileCfg.Adapter != "" {
		cfg.Adapter = fileCfg.Adapter
	}
	if fileCfg.Model != "" {
		cfg.Model = fileCfg.Model
	}

	// Env overrides file
	if u := os.Getenv("ASK_SERVER_URL"); u != "" {
		cfg.ServerURL = u
	}

	return cfg
}

// Save writes the config file.
func Save(cfg *Config) error {
	dir := configDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	return os.WriteFile(configPath(), data, 0644)
}

// Set updates a single config key.
func (c *Config) Set(key, value string) error {
	switch key {
	case "serverUrl":
		c.ServerURL = value
	case "adapter":
		c.Adapter = value
	case "model":
		c.Model = value
	default:
		return fmt.Errorf("unknown config key %q (valid: %v)", key, ValidKeys)
	}
	return nil
}

// Path returns the config file path (for display).
func Path() string {
	return configPath()
}
