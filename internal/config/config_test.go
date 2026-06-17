package config

import (
	"os"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Port != 8080 {
		t.Errorf("expected port 8080, got %d", cfg.Port)
	}
	if cfg.BaseURL != "https://api.anthropic.com" {
		t.Errorf("expected base URL https://api.anthropic.com, got %s", cfg.BaseURL)
	}
	if cfg.BufferSize != 500 {
		t.Errorf("expected buffer size 500, got %d", cfg.BufferSize)
	}
}

func TestLoadFromFile(t *testing.T) {
	content := []byte(`
api_key: "test-key"
port: 9090
base_url: "https://custom.api.com"
`)
	tmpFile := t.TempDir() + "/config.yaml"
	if err := os.WriteFile(tmpFile, content, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(tmpFile)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.APIKey != "test-key" {
		t.Errorf("expected api_key test-key, got %s", cfg.APIKey)
	}
	if cfg.Port != 9090 {
		t.Errorf("expected port 9090, got %d", cfg.Port)
	}
}

func TestEnvOverride(t *testing.T) {
	os.Setenv("BRAINPROXY_API_KEY", "env-key")
	os.Setenv("BRAINPROXY_PORT", "3000")
	defer os.Unsetenv("BRAINPROXY_API_KEY")
	defer os.Unsetenv("BRAINPROXY_PORT")

	cfg, err := Load("/nonexistent/path")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.APIKey != "env-key" {
		t.Errorf("expected env-key, got %s", cfg.APIKey)
	}
	if cfg.Port != 3000 {
		t.Errorf("expected port 3000, got %d", cfg.Port)
	}
}
