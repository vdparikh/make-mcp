package api

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/vdparikh/make-mcp/backend/internal/mcpvalidate"
	"github.com/vdparikh/make-mcp/backend/internal/models"
)

const serverJSONExportSchemaVersion = 1

type ServerExportMeta struct {
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Version     string              `json:"version"`
	Icon        string              `json:"icon,omitempty"`
	Status      models.ServerStatus `json:"status,omitempty"`
	IsPublic    bool                `json:"is_public,omitempty"`
	EnvProfiles json.RawMessage     `json:"env_profiles,omitempty"`
}

type ExportTool struct {
	Name                 string               `json:"name"`
	Description          string               `json:"description"`
	InputSchema          json.RawMessage      `json:"input_schema"`
	OutputSchema         json.RawMessage      `json:"output_schema"`
	ExecutionType        models.ExecutionType `json:"execution_type"`
	ExecutionConfig      json.RawMessage      `json:"execution_config"`
	ContextFields        []string             `json:"context_fields,omitempty"`
	OutputDisplay        string               `json:"output_display,omitempty"`
	OutputDisplayConfig  json.RawMessage      `json:"output_display_config,omitempty"`
	ReadOnlyHint         bool                 `json:"read_only_hint,omitempty"`
	DestructiveHint      bool                 `json:"destructive_hint,omitempty"`
}

type ExportResource struct {
	Name     string          `json:"name"`
	URI      string          `json:"uri"`
	MimeType string          `json:"mime_type"`
	Handler  json.RawMessage `json:"handler"`
}

type ExportPrompt struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Template    string          `json:"template"`
	Arguments   json.RawMessage `json:"arguments"`
}

type ExportContextConfig struct {
	Name       string          `json:"name"`
	SourceType string          `json:"source_type"`
	Config     json.RawMessage `json:"config"`
}

type ExportPolicyRule struct {
	Type       models.PolicyRuleType `json:"type"`
	Config     json.RawMessage       `json:"config"`
	Priority   int                   `json:"priority"`
	FailAction string                `json:"fail_action"`
}

type ExportPolicy struct {
	ToolName    string             `json:"tool_name"`
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Enabled     bool               `json:"enabled"`
	Rules       []ExportPolicyRule `json:"rules"`
}

type ServerJSONExportPayload struct {
	SchemaVersion  int                   `json:"schema_version"`
	Server         ServerExportMeta      `json:"server"`
	Tools          []ExportTool          `json:"tools"`
	Resources      []ExportResource      `json:"resources"`
	Prompts        []ExportPrompt        `json:"prompts"`
	ContextConfigs []ExportContextConfig `json:"context_configs"`
	Policies       []ExportPolicy        `json:"policies"`
}

func validateServerJSONExportPayload(payload *ServerJSONExportPayload) error {
	if payload == nil {
		return fmt.Errorf("payload is required")
	}
	if payload.SchemaVersion != serverJSONExportSchemaVersion {
		return fmt.Errorf("unsupported schema_version %d", payload.SchemaVersion)
	}
	if strings.TrimSpace(payload.Server.Name) == "" {
		return fmt.Errorf("server.name is required")
	}
	if strings.TrimSpace(payload.Server.Version) == "" {
		// The server version drives codegen; require it so downstream imports are stable.
		return fmt.Errorf("server.version is required")
	}
	if len(payload.Tools) == 0 {
		return fmt.Errorf("at least one tool is required")
	}
	if len(payload.Tools) > 500 {
		return fmt.Errorf("too many tools (%d)", len(payload.Tools))
	}

	toolNameSeen := make(map[string]bool, len(payload.Tools))
	for i := range payload.Tools {
		t := payload.Tools[i]
		nm := strings.TrimSpace(t.Name)
		if nm == "" {
			return fmt.Errorf("tool[%d].name is required", i)
		}
		if err := mcpvalidate.ValidateToolName(nm); err != nil {
			return fmt.Errorf("tool[%d].name: %w", i, err)
		}
		if toolNameSeen[nm] {
			return fmt.Errorf("duplicate tool name %q", nm)
		}
		toolNameSeen[nm] = true
		if t.ExecutionType == "" {
			return fmt.Errorf("tool[%d].execution_type is required", i)
		}
	}

	if len(payload.Policies) > 2000 {
		return fmt.Errorf("too many policies (%d)", len(payload.Policies))
	}

	allowedFailActions := map[string]bool{"deny": true, "warn": true, "approve": true}
	for i := range payload.Policies {
		p := payload.Policies[i]
		ptn := strings.TrimSpace(p.ToolName)
		if ptn == "" {
			return fmt.Errorf("policies[%d].tool_name is required", i)
		}
		if err := mcpvalidate.ValidateToolName(ptn); err != nil {
			return fmt.Errorf("policies[%d].tool_name: %w", i, err)
		}
		if len(p.Rules) == 0 {
			return fmt.Errorf("policies[%d].rules must not be empty", i)
		}
		for j := range p.Rules {
			r := p.Rules[j]
			if r.Priority < 0 {
				return fmt.Errorf("policies[%d].rules[%d].priority must be >= 0", i, j)
			}
			if !allowedFailActions[r.FailAction] {
				return fmt.Errorf("policies[%d].rules[%d].fail_action must be deny|warn|approve", i, j)
			}
		}
	}
	return nil
}

func ensureJSONObjectRaw(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return json.RawMessage(`{}`)
	}
	return raw
}

func parseImportServerJSONBody(body []byte) (ServerJSONExportPayload, ImportServerJSONRequest, error) {
	var payload ServerJSONExportPayload
	var wrapped ImportServerJSONRequest
	// Try wrapped format first.
	if err := json.Unmarshal(body, &wrapped); err == nil && wrapped.Payload.SchemaVersion != 0 {
		payload = wrapped.Payload
		return payload, wrapped, nil
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return ServerJSONExportPayload{}, ImportServerJSONRequest{}, fmt.Errorf("invalid JSON payload")
	}
	return payload, wrapped, nil
}
