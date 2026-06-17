package config

import (
	"os"
	"strconv"

	"gopkg.in/yaml.v3"
)

// Config holds all proxy configuration.
type Config struct {
	APIKey        string `yaml:"api_key"`
	BaseURL       string `yaml:"base_url"`
	Port          int    `yaml:"port"`
	BufferSize    int    `yaml:"buffer_size"`
	LogDir        string `yaml:"log_dir"`
	MaxLogFiles   int    `yaml:"max_log_files"`
	SQLiteEnabled bool   `yaml:"sqlite_enabled"`
}

// DefaultConfig returns config with default values.
func DefaultConfig() *Config {
	return &Config{
		BaseURL:     "https://api.anthropic.com",
		Port:        8080,
		BufferSize:  500,
		LogDir:      "logs",
		MaxLogFiles: 1000,
	}
}

// Load reads config from file path, then overrides with environment variables.
func Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	if err == nil {
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, err
		}
	}

	if v := os.Getenv("BRAINPROXY_API_KEY"); v != "" {
		cfg.APIKey = v
	}
	if v := os.Getenv("BRAINPROXY_BASE_URL"); v != "" {
		cfg.BaseURL = v
	}
	if v := os.Getenv("BRAINPROXY_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.Port = port
		}
	}
	if v := os.Getenv("BRAINPROXY_LOG_DIR"); v != "" {
		cfg.LogDir = v
	}

	return cfg, nil
}
