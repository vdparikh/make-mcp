package llm

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type ProviderConfig struct {
	Type      string `yaml:"type"`
	BaseURL   string `yaml:"base_url"`
	Model     string `yaml:"model"`
	APIKeyEnv string `yaml:"api_key_env"`
	APIKey    string `yaml:"api_key"`
	Enabled   bool   `yaml:"enabled"`
}

type Config struct {
	DefaultProvider string                    `yaml:"default_provider"`
	Providers       map[string]ProviderConfig `yaml:"providers"`
}

func resolveConfigPath() string {
	if custom := strings.TrimSpace(os.Getenv("LLM_CONFIG_PATH")); custom != "" {
		return custom
	}
	for _, candidate := range []string{"llm-config.yaml", "../llm-config.yaml"} {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return "llm-config.yaml"
}

func loadConfig() (*Config, error) {
	path := resolveConfigPath()
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read llm config %s: %w", filepath.Clean(path), err)
	}
	var cfg Config
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return nil, fmt.Errorf("parse llm config: %w", err)
	}
	if len(cfg.Providers) == 0 {
		return nil, fmt.Errorf("no llm providers configured")
	}
	if strings.TrimSpace(cfg.DefaultProvider) == "" {
		for name, p := range cfg.Providers {
			if p.Enabled {
				cfg.DefaultProvider = name
				break
			}
		}
	}
	if strings.TrimSpace(cfg.DefaultProvider) == "" {
		return nil, fmt.Errorf("default_provider is required")
	}
	return &cfg, nil
}
