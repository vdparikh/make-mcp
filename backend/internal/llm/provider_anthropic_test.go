package llm

import (
	"strings"
	"testing"
)

func TestConvertMessagesForAnthropic(t *testing.T) {
	tests := []struct {
		name       string
		messages   []ChatMessage
		wantSys    string
		wantMinMsg int
	}{
		{
			name: "system and user",
			messages: []ChatMessage{
				{Role: "system", Content: "You are helpful."},
				{Role: "user", Content: "Hi"},
			},
			wantSys:    "You are helpful.",
			wantMinMsg: 1,
		},
		{
			name: "multiple system merged",
			messages: []ChatMessage{
				{Role: "system", Content: "A"},
				{Role: "user", Content: "Q"},
				{Role: "system", Content: "B"},
				{Role: "assistant", Content: "A"},
			},
			wantSys:    "A\n\nB",
			wantMinMsg: 2,
		},
		{
			name: "only system gets fallback user",
			messages: []ChatMessage{
				{Role: "system", Content: "Instructions only."},
			},
			wantSys:    "Instructions only.",
			wantMinMsg: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sys, out := convertMessagesForAnthropic(tt.messages)
			if sys != tt.wantSys {
				t.Fatalf("system: got %q want %q", sys, tt.wantSys)
			}
			if len(out) < tt.wantMinMsg {
				t.Fatalf("messages len: got %d want >= %d", len(out), tt.wantMinMsg)
			}
			if len(out) > 0 && (out[0]["role"] != "user" && out[0]["role"] != "assistant") {
				t.Fatalf("first message role invalid: %v", out[0]["role"])
			}
		})
	}
}

func TestBuildAnthropicTools(t *testing.T) {
	tools := []ToolDefinition{
		{Name: "t1", Description: "d", InputSchema: map[string]interface{}{"type": "object", "properties": map[string]interface{}{"x": map[string]string{"type": "string"}}}},
	}
	wire := buildAnthropicTools(tools)
	if len(wire) != 1 || wire[0]["name"] != "t1" {
		t.Fatalf("unexpected wire: %+v", wire)
	}
	if _, ok := wire[0]["input_schema"]; !ok {
		t.Fatal("expected input_schema")
	}
}

func TestNewAnthropicProvider_DefaultBaseURL(t *testing.T) {
	p, err := newAnthropicProvider("a", ProviderConfig{Model: "claude-haiku-4-5-20251001", BaseURL: ""}, "k")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(p.baseURL, "api.anthropic.com") {
		t.Fatalf("base: %s", p.baseURL)
	}
}
