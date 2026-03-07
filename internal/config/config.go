package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config holds the ask CLI configuration.
type Config struct {
	ServerURL string          `json:"serverUrl"`          // Askbox server URL
	Adapter   string          `json:"adapter,omitempty"`  // Default adapter (claude-agent-sdk, strands)
	Model     string          `json:"model,omitempty"`    // Default model override
	Provider  *ProviderConfig `json:"provider,omitempty"` // LLM provider settings (for local askbox)
	ArchHub   *ArchHubConfig  `json:"archHub,omitempty"`  // Arch-hub settings
}

// ProviderConfig holds LLM provider settings.
type ProviderConfig struct {
	Name          string `json:"name"`                    // anthropic, bedrock, litellm
	Region        string `json:"region,omitempty"`        // AWS region (bedrock)
	AuthMethod    string `json:"authMethod,omitempty"`    // iam-role, access-keys, profile, api-keys
	Model         string `json:"model,omitempty"`         // Model ID or alias
	ProxyURL      string `json:"proxyUrl,omitempty"`      // LiteLLM proxy URL
}

// ArchHubConfig holds arch-hub settings.
type ArchHubConfig struct {
	URL    string `json:"url,omitempty"`    // Git URL
	Branch string `json:"branch,omitempty"` // Git branch
}

// DefaultServerURL is the default askbox server address.
const DefaultServerURL = "http://localhost:8082"

// ValidKeys lists all config keys that can be set.
var ValidKeys = []string{"serverUrl", "adapter", "model", "provider.name", "provider.region", "archHub.url"}

// configDir returns the config directory path.
func configDir() string {
	if d := os.Getenv("ASK_CONFIG_DIR"); d != "" {
		return d
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "ask")
}

// DataDir returns the data directory for compose files etc.
func DataDir() string {
	if d := os.Getenv("ASK_DATA_DIR"); d != "" {
		return d
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".ask")
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
	cfg.Provider = fileCfg.Provider
	cfg.ArchHub = fileCfg.ArchHub

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

// HasProvider returns true if a provider is configured.
func (c *Config) HasProvider() bool {
	return c.Provider != nil && c.Provider.Name != ""
}

// ModelID returns the resolved model ID from provider config or model override.
func (c *Config) ModelID() string {
	if c.Model != "" {
		return c.Model
	}
	if c.Provider != nil && c.Provider.Model != "" {
		return resolveModelAlias(c.Provider.Name, c.Provider.Model)
	}
	return ""
}

// resolveModelAlias expands short aliases to full model IDs.
func resolveModelAlias(provider, alias string) string {
	if provider == "bedrock" {
		switch alias {
		case "sonnet":
			return "us.anthropic.claude-sonnet-4-20250514-v1:0"
		case "opus":
			return "us.anthropic.claude-opus-4-20250514-v1:0"
		case "haiku":
			return "us.anthropic.claude-3-5-haiku-20241022-v1:0"
		}
	} else {
		switch alias {
		case "sonnet":
			return "claude-sonnet-4-20250514"
		case "opus":
			return "claude-opus-4-20250514"
		case "haiku":
			return "claude-3-5-haiku-20241022"
		}
	}
	return alias
}

// DetectRepoSwarmConfig looks for RepoSwarm config in the current directory tree.
// Returns provider env vars if found.
func DetectRepoSwarmConfig() map[string]string {
	// Look for .reposwarm directory (up to 3 levels up)
	dirs := []string{
		".reposwarm",
		filepath.Join("..", ".reposwarm"),
		filepath.Join("..", "..", ".reposwarm"),
	}

	for _, d := range dirs {
		envPath := filepath.Join(d, "temporal", "worker.env")
		if data, err := os.ReadFile(envPath); err == nil {
			return parseEnvFile(string(data))
		}
	}

	// Also check home directory
	home, _ := os.UserHomeDir()
	envPath := filepath.Join(home, ".reposwarm", "temporal", "worker.env")
	if data, err := os.ReadFile(envPath); err == nil {
		return parseEnvFile(string(data))
	}

	return nil
}

func parseEnvFile(content string) map[string]string {
	vars := map[string]string{}
	for _, line := range splitLines(content) {
		line = trimSpace(line)
		if line == "" || line[0] == '#' {
			continue
		}
		eq := indexOf(line, '=')
		if eq > 0 {
			key := line[:eq]
			val := line[eq+1:]
			// Strip quotes
			if len(val) >= 2 && ((val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'')) {
				val = val[1 : len(val)-1]
			}
			vars[key] = val
		}
	}
	return vars
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func trimSpace(s string) string {
	i, j := 0, len(s)
	for i < j && (s[i] == ' ' || s[i] == '\t' || s[i] == '\r') {
		i++
	}
	for j > i && (s[j-1] == ' ' || s[j-1] == '\t' || s[j-1] == '\r') {
		j--
	}
	return s[i:j]
}

func indexOf(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}
