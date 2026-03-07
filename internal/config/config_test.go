package config

import (
	"os"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	// Point config to nonexistent dir so defaults are used
	t.Setenv("ASK_CONFIG_DIR", "/tmp/ask-test-nonexistent")
	os.Unsetenv("ASK_SERVER_URL")

	cfg := Load()
	if cfg.ServerURL != DefaultServerURL {
		t.Errorf("expected default server URL %q, got %q", DefaultServerURL, cfg.ServerURL)
	}
	if cfg.Adapter != "" {
		t.Errorf("expected empty adapter, got %q", cfg.Adapter)
	}
}

func TestLoadEnvOverride(t *testing.T) {
	t.Setenv("ASK_CONFIG_DIR", "/tmp/ask-test-nonexistent")
	t.Setenv("ASK_SERVER_URL", "http://custom:9999")

	cfg := Load()
	if cfg.ServerURL != "http://custom:9999" {
		t.Errorf("expected env override, got %q", cfg.ServerURL)
	}
}

func TestSetValidKey(t *testing.T) {
	cfg := &Config{}
	if err := cfg.Set("serverUrl", "http://test:8082"); err != nil {
		t.Fatal(err)
	}
	if cfg.ServerURL != "http://test:8082" {
		t.Errorf("expected set value, got %q", cfg.ServerURL)
	}
}

func TestSetInvalidKey(t *testing.T) {
	cfg := &Config{}
	if err := cfg.Set("invalid", "value"); err == nil {
		t.Fatal("expected error for invalid key")
	}
}

func TestSaveLoad(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ASK_CONFIG_DIR", dir)
	os.Unsetenv("ASK_SERVER_URL")

	cfg := &Config{
		ServerURL: "http://saved:1234",
		Adapter:   "strands",
		Model:     "test-model",
	}
	if err := Save(cfg); err != nil {
		t.Fatal(err)
	}

	loaded := Load()
	if loaded.ServerURL != "http://saved:1234" {
		t.Errorf("expected saved URL, got %q", loaded.ServerURL)
	}
	if loaded.Adapter != "strands" {
		t.Errorf("expected saved adapter, got %q", loaded.Adapter)
	}
	if loaded.Model != "test-model" {
		t.Errorf("expected saved model, got %q", loaded.Model)
	}
}
