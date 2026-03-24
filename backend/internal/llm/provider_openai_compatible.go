package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatResult struct {
	Provider  string     `json:"provider"`
	Model     string     `json:"model"`
	Message   string     `json:"message"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

type Provider interface {
	Chat(ctx context.Context, model string, messages []ChatMessage, tools []ToolDefinition) (*ChatResult, error)
}

type ToolDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

type ToolCall struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type openAICompatibleProvider struct {
	providerName string
	baseURL      string
	defaultModel string
	apiKey       string
	client       *http.Client
}

func newOpenAICompatibleProvider(name string, cfg ProviderConfig, apiKey string) (*openAICompatibleProvider, error) {
	base := strings.TrimSpace(cfg.BaseURL)
	if base == "" {
		return nil, fmt.Errorf("provider %s base_url is required", name)
	}
	return &openAICompatibleProvider{
		providerName: name,
		baseURL:      strings.TrimSuffix(base, "/"),
		defaultModel: strings.TrimSpace(cfg.Model),
		apiKey:       apiKey,
		client:       &http.Client{Timeout: 45 * time.Second},
	}, nil
}

func (p *openAICompatibleProvider) Chat(ctx context.Context, model string, messages []ChatMessage, tools []ToolDefinition) (*ChatResult, error) {
	model = strings.TrimSpace(model)
	if model == "" {
		model = p.defaultModel
	}
	if model == "" {
		return nil, fmt.Errorf("model is required")
	}
	if len(messages) == 0 {
		return nil, fmt.Errorf("at least one message is required")
	}

	reqPayload := map[string]interface{}{
		"model":    model,
		"messages": messages,
	}
	if len(tools) > 0 {
		wireTools := make([]map[string]interface{}, 0, len(tools))
		for _, t := range tools {
			parameters := map[string]interface{}{
				"type": "object",
			}
			if t.InputSchema != nil {
				parameters = t.InputSchema
			}
			wireTools = append(wireTools, map[string]interface{}{
				"type": "function",
				"function": map[string]interface{}{
					"name":        t.Name,
					"description": t.Description,
					"parameters":  parameters,
				},
			})
		}
		reqPayload["tools"] = wireTools
		reqPayload["tool_choice"] = "auto"
	}
	body, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, fmt.Errorf("marshal chat request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create chat request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call provider %s: %w", p.providerName, err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		return nil, fmt.Errorf("read provider response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("provider %s returned %d: %s", p.providerName, resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var parsed struct {
		Choices []struct {
			Message struct {
				Content   string `json:"content"`
				ToolCalls []struct {
					ID       string `json:"id"`
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("decode provider response: %w", err)
	}
	if len(parsed.Choices) == 0 {
		return nil, fmt.Errorf("provider %s returned no choices", p.providerName)
	}
	toolCalls := make([]ToolCall, 0)
	for _, tc := range parsed.Choices[0].Message.ToolCalls {
		if strings.TrimSpace(tc.Function.Name) == "" {
			continue
		}
		toolCalls = append(toolCalls, ToolCall{
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: tc.Function.Arguments,
		})
	}
	return &ChatResult{
		Provider:  p.providerName,
		Model:     model,
		Message:   parsed.Choices[0].Message.Content,
		ToolCalls: toolCalls,
	}, nil
}
