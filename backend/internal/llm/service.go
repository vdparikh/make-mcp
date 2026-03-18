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

func NewService() (*Service, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, err
	}
	svc := &Service{
		defaultProvider: cfg.DefaultProvider,
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
		if strings.TrimSpace(p.Type) != "openai_compatible" {
			continue
		}
		provider, err := newOpenAICompatibleProvider(name, p, apiKey)
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
