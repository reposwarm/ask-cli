package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectRepoSwarmConfig(t *testing.T) {
	// Create a fake .reposwarm/temporal/worker.env
	dir := t.TempDir()
	rsDir := filepath.Join(dir, ".reposwarm", "temporal")
	os.MkdirAll(rsDir, 0755)
	os.WriteFile(filepath.Join(rsDir, "worker.env"), []byte(`
CLAUDE_CODE_USE_BEDROCK=1
CLAUDE_PROVIDER=bedrock
AWS_REGION=us-west-2
ANTHROPIC_MODEL=us.anthropic.claude-sonnet-4-20250514-v1:0
`), 0644)

	// Change to the test dir and look for .reposwarm
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	vars := DetectRepoSwarmConfig()
	if vars == nil {
		t.Fatal("expected to detect reposwarm config")
	}
	if vars["CLAUDE_CODE_USE_BEDROCK"] != "1" {
		t.Errorf("expected bedrock=1, got %q", vars["CLAUDE_CODE_USE_BEDROCK"])
	}
	if vars["AWS_REGION"] != "us-west-2" {
		t.Errorf("expected us-west-2, got %q", vars["AWS_REGION"])
	}
}

func TestDetectRepoSwarmConfigNotFound(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	// Override home to avoid finding real config
	t.Setenv("HOME", dir)

	vars := DetectRepoSwarmConfig()
	if vars != nil {
		t.Error("expected nil when no reposwarm config")
	}
}

func TestParseEnvFile(t *testing.T) {
	content := `# Comment
CLAUDE_PROVIDER=bedrock
AWS_REGION=us-east-1
ANTHROPIC_API_KEY="sk-test-key"
EMPTY=
`
	vars := parseEnvFile(content)
	if vars["CLAUDE_PROVIDER"] != "bedrock" {
		t.Errorf("expected bedrock, got %q", vars["CLAUDE_PROVIDER"])
	}
	if vars["ANTHROPIC_API_KEY"] != "sk-test-key" {
		t.Errorf("expected unquoted key, got %q", vars["ANTHROPIC_API_KEY"])
	}
	if vars["EMPTY"] != "" {
		t.Errorf("expected empty, got %q", vars["EMPTY"])
	}
}

func TestResolveModelAlias(t *testing.T) {
	tests := []struct {
		provider, alias, expected string
	}{
		{"bedrock", "sonnet", "us.anthropic.claude-sonnet-4-20250514-v1:0"},
		{"bedrock", "opus", "us.anthropic.claude-opus-4-20250514-v1:0"},
		{"anthropic", "sonnet", "claude-sonnet-4-20250514"},
		{"bedrock", "custom-model-id", "custom-model-id"},
	}
	for _, tt := range tests {
		got := resolveModelAlias(tt.provider, tt.alias)
		if got != tt.expected {
			t.Errorf("resolveModelAlias(%q, %q) = %q, want %q", tt.provider, tt.alias, got, tt.expected)
		}
	}
}
