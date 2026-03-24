package llm

import (
	"context"
	"fmt"
	"os"
	"strings"
)

type ProviderInfo struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Model   string `json:"model"`
	Enabled bool   `json:"enabled"`
}

type Service struct {
	defaultProvider string
	providers       map[string]Provider
	meta            map[string]ProviderInfo
}

// NewService builds an LLM service from the llm: block in config/config.yaml.
// If cfg is nil or has no providers, returns (nil, nil) so LLM features are off.
// If providers are listed but none have API keys, returns an error (callers may ignore).
func NewService(cfg *Config) (*Service, error) {
	if cfg == nil || len(cfg.Providers) == 0 {
		return nil, nil
	}
	defaultProvider := strings.TrimSpace(cfg.DefaultProvider)
	if defaultProvider == "" {
		for name, p := range cfg.Providers {
			if p.Enabled {
				defaultProvider = name
				break
			}
		}
	}
	if defaultProvider == "" {
		return nil, fmt.Errorf("llm: default_provider is required when providers are configured")
	}
	svc := &Service{
		defaultProvider: defaultProvider,
		providers:       make(map[string]Provider),
		meta:            make(map[string]ProviderInfo),
	}
	for name, p := range cfg.Providers {
		enabled := p.Enabled
		if !enabled {
			continue
		}
		apiKey := strings.TrimSpace(p.APIKey)
		if apiKey == "" && strings.TrimSpace(p.APIKeyEnv) != "" {
			apiKey = strings.TrimSpace(os.Getenv(p.APIKeyEnv))
		}
		if apiKey == "" {
			continue
		}
		pt := strings.TrimSpace(p.Type)
		var provider Provider
		var err error
		switch pt {
		case "openai_compatible":
			provider, err = newOpenAICompatibleProvider(name, p, apiKey)
		case "anthropic":
			provider, err = newAnthropicProvider(name, p, apiKey)
		default:
			continue
		}
		if err != nil {
			return nil, err
		}
		svc.providers[name] = provider
		svc.meta[name] = ProviderInfo{
			Name:    name,
			Type:    p.Type,
			Model:   p.Model,
			Enabled: true,
		}
	}
	if len(svc.providers) == 0 {
		return nil, fmt.Errorf("no enabled llm providers with api keys")
	}
	if _, ok := svc.providers[svc.defaultProvider]; !ok {
		for n := range svc.providers {
			svc.defaultProvider = n
			break
		}
	}
	return svc, nil
}

func (s *Service) ProviderInfos() []ProviderInfo {
	out := make([]ProviderInfo, 0, len(s.meta))
	for _, m := range s.meta {
		out = append(out, m)
	}
	return out
}

func (s *Service) DefaultProvider() string {
	return s.defaultProvider
}

func (s *Service) Chat(ctx context.Context, providerName, model string, messages []ChatMessage, tools []ToolDefinition) (*ChatResult, error) {
	name := strings.TrimSpace(providerName)
	if name == "" {
		name = s.defaultProvider
	}
	provider, ok := s.providers[name]
	if !ok {
		return nil, fmt.Errorf("provider %s is not available", name)
	}
	return provider.Chat(ctx, model, messages, tools)
}
