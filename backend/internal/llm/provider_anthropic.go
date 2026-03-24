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

// Anthropic Messages API (not OpenAI-compatible). Docs: https://docs.anthropic.com/claude/reference/messages_post
const (
	defaultAnthropicBaseURL = "https://api.anthropic.com"
	anthropicAPIVersion     = "2023-06-01"
	defaultAnthropicMaxOut  = 8192
)

type anthropicProvider struct {
	providerName string
	baseURL      string
	defaultModel string
	apiKey       string
	maxTokens    int
	client       *http.Client
}

func newAnthropicProvider(name string, cfg ProviderConfig, apiKey string) (*anthropicProvider, error) {
	base := strings.TrimSpace(cfg.BaseURL)
	if base == "" {
		base = defaultAnthropicBaseURL
	}
	max := cfg.MaxTokens
	if max <= 0 {
		max = defaultAnthropicMaxOut
	}
	if max > 64000 {
		max = 64000
	}
	return &anthropicProvider{
		providerName: name,
		baseURL:      strings.TrimSuffix(base, "/"),
		defaultModel: strings.TrimSpace(cfg.Model),
		apiKey:       apiKey,
		maxTokens:    max,
		client:       &http.Client{Timeout: 120 * time.Second},
	}, nil
}

// convertMessagesForAnthropic merges all system messages into the Messages API `system` field (order preserved)
// and keeps user/assistant turns in order for `messages`.
func convertMessagesForAnthropic(messages []ChatMessage) (system string, out []map[string]interface{}) {
	var sysParts []string
	var conv []ChatMessage
	for _, m := range messages {
		switch strings.ToLower(strings.TrimSpace(m.Role)) {
		case "system":
			if strings.TrimSpace(m.Content) != "" {
				sysParts = append(sysParts, m.Content)
			}
		case "user", "assistant":
			conv = append(conv, m)
		}
	}
	system = strings.Join(sysParts, "\n\n")
	for _, m := range conv {
		role := strings.ToLower(strings.TrimSpace(m.Role))
		if role != "user" && role != "assistant" {
			continue
		}
		out = append(out, map[string]interface{}{
			"role":    role,
			"content": m.Content,
		})
	}
	if len(out) == 0 {
		out = []map[string]interface{}{
			{"role": "user", "content": "Follow the system instructions and respond."},
		}
	}
	return system, out
}

func buildAnthropicTools(tools []ToolDefinition) []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(tools))
	for _, t := range tools {
		params := map[string]interface{}{"type": "object"}
		if t.InputSchema != nil {
			params = t.InputSchema
		}
		out = append(out, map[string]interface{}{
			"name":         t.Name,
			"description":  t.Description,
			"input_schema": params,
		})
	}
	return out
}

func (p *anthropicProvider) Chat(ctx context.Context, model string, messages []ChatMessage, tools []ToolDefinition) (*ChatResult, error) {
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

	system, anthropicMessages := convertMessagesForAnthropic(messages)

	body := map[string]interface{}{
		"model":       model,
		"max_tokens":  p.maxTokens,
		"messages":    anthropicMessages,
	}
	if strings.TrimSpace(system) != "" {
		body["system"] = system
	}
	if len(tools) > 0 {
		body["tools"] = buildAnthropicTools(tools)
	}

	raw, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal anthropic request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/v1/messages", bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("create anthropic request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", anthropicAPIVersion)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call provider %s: %w", p.providerName, err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		return nil, fmt.Errorf("read anthropic response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("provider %s returned %d: %s", p.providerName, resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var parsed struct {
		Content []struct {
			Type  string          `json:"type"`
			Text  string          `json:"text"`
			ID    string          `json:"id"`
			Name  string          `json:"name"`
			Input json.RawMessage `json:"input"`
		} `json:"content"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("decode anthropic response: %w", err)
	}

	var textParts []string
	toolCalls := make([]ToolCall, 0)
	for _, block := range parsed.Content {
		switch block.Type {
		case "text":
			textParts = append(textParts, block.Text)
		case "tool_use":
			if strings.TrimSpace(block.Name) == "" {
				continue
			}
			args := strings.TrimSpace(string(block.Input))
			if args == "" {
				args = "{}"
			}
			toolCalls = append(toolCalls, ToolCall{
				ID:        block.ID,
				Name:      block.Name,
				Arguments: args,
			})
		default:
			continue
		}
	}

	return &ChatResult{
		Provider:  p.providerName,
		Model:     model,
		Message:   strings.TrimSpace(strings.Join(textParts, "")),
		ToolCalls: toolCalls,
	}, nil
}
