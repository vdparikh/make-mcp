package llm

// ProviderConfig describes one LLM endpoint (OpenAI-compatible or Anthropic Messages API).
type ProviderConfig struct {
	Type      string `yaml:"type"`
	BaseURL   string `yaml:"base_url"`
	Model     string `yaml:"model"`
	APIKeyEnv string `yaml:"api_key_env"`
	APIKey    string `yaml:"api_key"`
	Enabled   bool   `yaml:"enabled"`
	// MaxTokens is required by Anthropic Messages API; defaults in code if unset (0).
	MaxTokens int `yaml:"max_tokens"`
}

// Config is the llm: section in config/config.yaml.
type Config struct {
	DefaultProvider string                    `yaml:"default_provider"`
	Providers       map[string]ProviderConfig `yaml:"providers"`
}
