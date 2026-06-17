package setup

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Options holds the setup configuration.
type Options struct {
	APIKey  string
	BaseURL string
	Port    int
	LogDir  string
}

// configData is the YAML structure written to config.yaml.
type configData struct {
	APIKey      string `yaml:"api_key"`
	BaseURL     string `yaml:"base_url"`
	Port        int    `yaml:"port"`
	BufferSize  int    `yaml:"buffer_size"`
	LogDir      string `yaml:"log_dir"`
	MaxLogFiles int    `yaml:"max_log_files"`
}

// Run writes config.yaml and creates the logs directory.
func Run(configPath string, opts Options) error {
	if opts.APIKey == "" {
		return fmt.Errorf("--api-key is required")
	}
	if opts.BaseURL == "" {
		return fmt.Errorf("--base-url is required")
	}
	if opts.Port == 0 {
		opts.Port = 8080
	}
	if opts.LogDir == "" {
		opts.LogDir = "logs"
	}

	cfg := configData{
		APIKey:      opts.APIKey,
		BaseURL:     opts.BaseURL,
		Port:        opts.Port,
		BufferSize:  500,
		LogDir:      opts.LogDir,
		MaxLogFiles: 1000,
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", configPath, err)
	}
	fmt.Printf("✓ 已创建 %s\n", configPath)

	if err := os.MkdirAll(opts.LogDir, 0o755); err != nil {
		return fmt.Errorf("create %s: %w", opts.LogDir, err)
	}
	fmt.Printf("✓ 已创建 %s/\n", opts.LogDir)

	fmt.Println()
	fmt.Println("现在可以运行：")
	fmt.Println("  brainproxy claude     # 自动启动代理 + Claude Code")
	fmt.Println("  brainproxy            # 只启动代理（手动模式）")

	return nil
}
