package commands

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/reposwarm/ask/internal/config"
	"github.com/reposwarm/ask/internal/output"
)

var (
	setupProviderFlag   string
	setupRegionFlag     string
	setupAuthFlag       string
	setupModelFlag      string
	setupProxyURLFlag   string
	setupProxyKeyFlag   string
	setupArchHubFlag    string
	setupPortFlag       string
	setupNonInterFlag   bool
	setupSkipDockerFlag bool
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Set up a local askbox server with Docker",
	Long: `Interactive setup for running askbox locally.

Configures the LLM provider, starts askbox via Docker, and connects the CLI.

If a RepoSwarm installation is detected, its provider settings can be reused.

Examples:
  ask setup
  ask setup --provider bedrock --region us-east-1 --model sonnet
  ask setup --provider anthropic --model opus
  ask setup --arch-hub https://github.com/org/arch-hub.git
  ask setup --skip-docker   # Configure only, don't start Docker`,
	RunE: runSetup,
}

func init() {
	setupCmd.Flags().StringVar(&setupProviderFlag, "provider", "", "LLM provider (anthropic, bedrock, litellm)")
	setupCmd.Flags().StringVar(&setupRegionFlag, "region", "", "AWS region (bedrock)")
	setupCmd.Flags().StringVar(&setupAuthFlag, "auth", "", "Auth method (iam-role, access-keys, profile, api-keys)")
	setupCmd.Flags().StringVar(&setupModelFlag, "model", "", "Model ID or alias (sonnet, opus, haiku)")
	setupCmd.Flags().StringVar(&setupProxyURLFlag, "proxy-url", "", "LiteLLM proxy URL")
	setupCmd.Flags().StringVar(&setupProxyKeyFlag, "proxy-key", "", "LiteLLM proxy API key")
	setupCmd.Flags().StringVar(&setupArchHubFlag, "arch-hub", "", "Arch-hub git URL")
	setupCmd.Flags().StringVar(&setupPortFlag, "port", "8082", "Askbox port")
	setupCmd.Flags().BoolVar(&setupNonInterFlag, "non-interactive", false, "Non-interactive mode (requires flags)")
	setupCmd.Flags().BoolVar(&setupSkipDockerFlag, "skip-docker", false, "Configure only, don't start Docker")

	rootCmd.AddCommand(setupCmd)
}

func runSetup(cmd *cobra.Command, args []string) error {
	cfg := config.Load()
	reader := bufio.NewReader(os.Stdin)

	provider := setupProviderFlag
	region := setupRegionFlag
	authMethod := setupAuthFlag
	model := setupModelFlag
	proxyURL := setupProxyURLFlag
	proxyKey := setupProxyKeyFlag
	archHub := setupArchHubFlag
	port := setupPortFlag

	// Secrets (only collected during setup, written to env file, NOT saved to config)
	var apiKey, awsKey, awsSecret, awsProfile, bedrockAPIKey string

	// ── Step 1: Auto-detect RepoSwarm config ──
	if provider == "" && !setupNonInterFlag {
		rsVars := config.DetectRepoSwarmConfig()
		if rsVars != nil {
			output.Info("🔍 Detected RepoSwarm configuration!")
			output.Info("")

			// Summarize what we found
			rsProvider := "anthropic"
			if rsVars["CLAUDE_CODE_USE_BEDROCK"] == "1" || rsVars["CLAUDE_PROVIDER"] == "bedrock" {
				rsProvider = "bedrock"
			} else if rsVars["ANTHROPIC_BASE_URL"] != "" {
				rsProvider = "litellm"
			}

			rsModel := rsVars["ANTHROPIC_MODEL"]
			if rsModel == "" {
				rsModel = rsVars["MODEL_ID"]
			}
			rsRegion := rsVars["AWS_REGION"]
			if rsRegion == "" {
				rsRegion = rsVars["AWS_DEFAULT_REGION"]
			}

			output.Info(fmt.Sprintf("  Provider: %s", rsProvider))
			if rsRegion != "" {
				output.Info(fmt.Sprintf("  Region:   %s", rsRegion))
			}
			if rsModel != "" {
				output.Info(fmt.Sprintf("  Model:    %s", rsModel))
			}
			output.Info("")

			reuse := promptYesNo(reader, "Reuse these settings?", true)
			if reuse {
				provider = rsProvider
				if rsRegion != "" {
					region = rsRegion
				}
				if rsModel != "" {
					model = rsModel
				}
				// Carry forward all env vars for the env file
				if rsVars["ANTHROPIC_API_KEY"] != "" {
					apiKey = rsVars["ANTHROPIC_API_KEY"]
				}
				if rsVars["AWS_ACCESS_KEY_ID"] != "" {
					awsKey = rsVars["AWS_ACCESS_KEY_ID"]
				}
				if rsVars["AWS_SECRET_ACCESS_KEY"] != "" {
					awsSecret = rsVars["AWS_SECRET_ACCESS_KEY"]
				}
				if rsVars["AWS_PROFILE"] != "" {
					awsProfile = rsVars["AWS_PROFILE"]
				}
				if rsVars["ANTHROPIC_BASE_URL"] != "" {
					proxyURL = rsVars["ANTHROPIC_BASE_URL"]
				}
				if rsVars["LITELLM_API_KEY"] != "" {
					proxyKey = rsVars["LITELLM_API_KEY"]
				}

				// Detect auth method from env vars
				if provider == "bedrock" {
					if awsKey != "" {
						authMethod = "access-keys"
					} else if awsProfile != "" {
						authMethod = "profile"
					} else if rsVars["AWS_BEARER_TOKEN_BEDROCK"] != "" {
						authMethod = "api-keys"
						bedrockAPIKey = rsVars["AWS_BEARER_TOKEN_BEDROCK"]
					} else {
						authMethod = "iam-role"
					}
				}
			}
		}
	}

	// ── Step 2: Interactive provider setup ──
	if provider == "" && !setupNonInterFlag && !output.AgentMode {
		output.Info("")
		output.Info("⚙️  Askbox Provider Setup")
		output.Info("")
		output.Info("  Which LLM provider should askbox use?")
		output.Info("")
		output.Info("  1) anthropic  — Direct Anthropic API (API key)")
		output.Info("  2) bedrock    — Amazon Bedrock (AWS credentials)")
		output.Info("  3) litellm    — LiteLLM proxy (custom endpoint)")
		output.Info("")

		provider = promptChoice(reader, "Provider [1/2/3]", map[string]string{
			"1": "anthropic", "2": "bedrock", "3": "litellm",
			"anthropic": "anthropic", "bedrock": "bedrock", "litellm": "litellm",
		}, "anthropic")
	}

	if provider == "" {
		return fmt.Errorf("--provider is required in non-interactive mode")
	}

	// Provider-specific prompts
	if !setupNonInterFlag && !output.AgentMode {
		switch provider {
		case "bedrock":
			if region == "" {
				region = promptString(reader, "AWS Region", "us-east-1")
			}
			if authMethod == "" {
				output.Info("")
				output.Info("  How should askbox authenticate with AWS?")
				output.Info("")
				output.Info("  1) iam-role     — EC2/ECS instance role (recommended)")
				output.Info("  2) access-keys  — AWS access key + secret")
				output.Info("  3) profile      — Named AWS profile")
				output.Info("  4) api-keys     — Bedrock API keys")
				output.Info("")

				authMethod = promptChoice(reader, "Auth method [1/2/3/4]", map[string]string{
					"1": "iam-role", "2": "access-keys", "3": "profile", "4": "api-keys",
					"iam-role": "iam-role", "access-keys": "access-keys", "profile": "profile", "api-keys": "api-keys",
				}, "iam-role")
			}

			switch authMethod {
			case "access-keys":
				if awsKey == "" {
					awsKey = promptString(reader, "AWS Access Key ID", "")
				}
				if awsSecret == "" {
					awsSecret = promptString(reader, "AWS Secret Access Key", "")
				}
			case "profile":
				if awsProfile == "" {
					awsProfile = promptString(reader, "AWS Profile name", "default")
				}
			case "api-keys":
				if bedrockAPIKey == "" {
					bedrockAPIKey = promptString(reader, "Bedrock API Key", "")
				}
			}

		case "anthropic":
			if apiKey == "" {
				apiKey = promptString(reader, "Anthropic API Key", "")
			}

		case "litellm":
			if proxyURL == "" {
				proxyURL = promptString(reader, "LiteLLM proxy URL", "http://localhost:4000")
			}
			if proxyKey == "" {
				proxyKey = promptString(reader, "LiteLLM API key (blank if none)", "")
			}
		}

		if model == "" {
			output.Info("")
			output.Info("  Model aliases: sonnet, opus, haiku")
			model = promptString(reader, "Model", "sonnet")
		}
	}

	// ── Step 3: Arch-hub URL ──
	if archHub == "" && !setupNonInterFlag && !output.AgentMode {
		output.Info("")
		archHub = promptString(reader, "Arch-hub git URL (blank to skip)", "")
	}

	// ── Step 4: Save config ──
	cfg.Provider = &config.ProviderConfig{
		Name:       provider,
		Region:     region,
		AuthMethod: authMethod,
		Model:      model,
		ProxyURL:   proxyURL,
	}
	if archHub != "" {
		cfg.ArchHub = &config.ArchHubConfig{URL: archHub}
	}
	cfg.ServerURL = fmt.Sprintf("http://localhost:%s", port)

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	output.Success(fmt.Sprintf("Config saved to %s", config.Path()))

	// ── Step 5: Write env file and docker-compose ──
	dataDir := config.DataDir()
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}

	// Write askbox.env (secrets go here, NOT in config.json)
	envVars := buildEnvVars(provider, region, authMethod, model, apiKey, awsKey, awsSecret, awsProfile, bedrockAPIKey, proxyURL, proxyKey, archHub)
	envPath := filepath.Join(dataDir, "askbox.env")
	if err := writeEnvFile(envPath, envVars); err != nil {
		return fmt.Errorf("write env file: %w", err)
	}
	output.Success(fmt.Sprintf("Env file written to %s", envPath))

	// Write docker-compose.yml
	composePath := filepath.Join(dataDir, "docker-compose.yml")
	if err := writeComposeFile(composePath, port, archHub); err != nil {
		return fmt.Errorf("write compose file: %w", err)
	}
	output.Success(fmt.Sprintf("Docker compose written to %s", composePath))

	// ── Step 6: Start Docker (optional) ──
	if setupSkipDockerFlag {
		output.Info("")
		output.Info("Skipping Docker startup. To start manually:")
		output.Info(fmt.Sprintf("  cd %s && docker compose up -d", dataDir))
		return nil
	}

	if !checkDocker() {
		output.Info("")
		output.Info("⚠️  Docker not found. To start askbox manually:")
		output.Info(fmt.Sprintf("  cd %s && docker compose up -d", dataDir))
		return nil
	}

	output.Info("")
	output.Info("🐳 Starting askbox...")

	startCmd := exec.Command("docker", "compose", "up", "-d")
	startCmd.Dir = dataDir
	startCmd.Stdout = os.Stdout
	startCmd.Stderr = os.Stderr
	if err := startCmd.Run(); err != nil {
		output.Error("Failed to start askbox", fmt.Sprintf("Run manually: cd %s && docker compose up -d", dataDir))
		return err
	}

	output.Info("")
	output.Success("Askbox is running!")
	output.Info("")
	output.Info("  Check status:  ask status")
	output.Info("  Ask something: ask \"how does auth work?\"")
	if archHub == "" {
		output.Info("")
		output.Info("  💡 No arch-hub configured. Load one with:")
		output.Info("     ask refresh --url https://github.com/org/arch-hub.git")
	}
	return nil
}

// buildEnvVars creates the environment variables for the askbox container.
func buildEnvVars(provider, region, authMethod, model, apiKey, awsKey, awsSecret, awsProfile, bedrockAPIKey, proxyURL, proxyKey, archHub string) map[string]string {
	vars := map[string]string{}

	switch provider {
	case "bedrock":
		vars["CLAUDE_CODE_USE_BEDROCK"] = "1"
		vars["CLAUDE_PROVIDER"] = "bedrock"
		if region != "" {
			vars["AWS_REGION"] = region
		}
		switch authMethod {
		case "access-keys":
			if awsKey != "" {
				vars["AWS_ACCESS_KEY_ID"] = awsKey
			}
			if awsSecret != "" {
				vars["AWS_SECRET_ACCESS_KEY"] = awsSecret
			}
		case "profile":
			if awsProfile != "" {
				vars["AWS_PROFILE"] = awsProfile
			}
		case "api-keys":
			if bedrockAPIKey != "" {
				vars["AWS_BEARER_TOKEN_BEDROCK"] = bedrockAPIKey
			}
		}

	case "anthropic":
		vars["CLAUDE_PROVIDER"] = "anthropic"
		if apiKey != "" {
			vars["ANTHROPIC_API_KEY"] = apiKey
		}

	case "litellm":
		vars["CLAUDE_PROVIDER"] = "litellm"
		if proxyURL != "" {
			vars["ANTHROPIC_BASE_URL"] = proxyURL
		}
		if proxyKey != "" {
			vars["LITELLM_API_KEY"] = proxyKey
		}
	}

	if model != "" {
		vars["ANTHROPIC_MODEL"] = model
	}
	if archHub != "" {
		vars["ARCH_HUB_URL"] = archHub
	}

	return vars
}

func writeEnvFile(path string, vars map[string]string) error {
	var lines []string
	// Write in deterministic order
	keyOrder := []string{
		"CLAUDE_PROVIDER", "CLAUDE_CODE_USE_BEDROCK",
		"AWS_REGION", "AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY", "AWS_PROFILE", "AWS_BEARER_TOKEN_BEDROCK",
		"ANTHROPIC_API_KEY", "ANTHROPIC_MODEL", "ANTHROPIC_BASE_URL",
		"LITELLM_API_KEY", "ARCH_HUB_URL",
	}
	for _, k := range keyOrder {
		if v, ok := vars[k]; ok {
			lines = append(lines, fmt.Sprintf("%s=%s", k, v))
		}
	}
	// Any remaining keys
	for k, v := range vars {
		found := false
		for _, kk := range keyOrder {
			if k == kk {
				found = true
				break
			}
		}
		if !found {
			lines = append(lines, fmt.Sprintf("%s=%s", k, v))
		}
	}
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0600) // 0600 for secrets
}

func writeComposeFile(path, port, archHub string) error {
	compose := fmt.Sprintf(`services:
  askbox:
    container_name: askbox
    image: ghcr.io/reposwarm/askbox:latest
    network_mode: host
    env_file:
      - askbox.env
    environment:
      - ASKBOX_PORT=%s
    volumes:
      - askbox-arch-hub:/tmp/arch-hub
    healthcheck:
      test: ["CMD", "python3", "-c", "import urllib.request; urllib.request.urlopen('http://localhost:%s/health')"]
      interval: 10s
      timeout: 5s
      retries: 5
    restart: unless-stopped

volumes:
  askbox-arch-hub:
`, port, port)

	return os.WriteFile(path, []byte(compose), 0644)
}

func checkDocker() bool {
	cmd := exec.Command("docker", "info")
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run() == nil
}

// ── Interactive prompts ──

func promptString(reader *bufio.Reader, label, defaultVal string) string {
	if output.AgentMode {
		return defaultVal
	}
	if defaultVal != "" {
		fmt.Printf("  %s [%s]: ", label, defaultVal)
	} else {
		fmt.Printf("  %s: ", label)
	}
	line, _ := reader.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return defaultVal
	}
	return line
}

func promptChoice(reader *bufio.Reader, label string, choices map[string]string, defaultVal string) string {
	if output.AgentMode {
		return defaultVal
	}
	for {
		fmt.Printf("  %s: ", label)
		line, _ := reader.ReadString('\n')
		line = strings.TrimSpace(line)
		if line == "" {
			return defaultVal
		}
		if val, ok := choices[strings.ToLower(line)]; ok {
			return val
		}
		fmt.Println("  Invalid choice, try again.")
	}
}

func promptYesNo(reader *bufio.Reader, label string, defaultYes bool) bool {
	if output.AgentMode {
		return defaultYes
	}
	def := "Y/n"
	if !defaultYes {
		def = "y/N"
	}
	fmt.Printf("  %s [%s]: ", label, def)
	line, _ := reader.ReadString('\n')
	line = strings.TrimSpace(strings.ToLower(line))
	if line == "" {
		return defaultYes
	}
	return line == "y" || line == "yes"
}
