package models

import (
	"encoding/json"
	"time"
)

// Server represents an MCP server configuration
type Server struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Version     string          `json:"version"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
	Tools       []Tool          `json:"tools,omitempty"`
	Resources   []Resource      `json:"resources,omitempty"`
	Prompts     []Prompt        `json:"prompts,omitempty"`
	AuthConfig  json.RawMessage `json:"auth_config,omitempty"`
}

// ExecutionType defines how a tool is executed
type ExecutionType string

const (
	ExecutionTypeRestAPI   ExecutionType = "rest_api"
	ExecutionTypeGraphQL   ExecutionType = "graphql"
	ExecutionTypeDatabase  ExecutionType = "database"
	ExecutionTypeJavaScript ExecutionType = "javascript"
	ExecutionTypePython    ExecutionType = "python"
	ExecutionTypeWebhook   ExecutionType = "webhook"
)

// Tool represents an MCP tool definition
type Tool struct {
	ID             string          `json:"id"`
	ServerID       string          `json:"server_id"`
	Name           string          `json:"name"`
	Description    string          `json:"description"`
	InputSchema    json.RawMessage `json:"input_schema"`
	OutputSchema   json.RawMessage `json:"output_schema"`
	ExecutionType  ExecutionType   `json:"execution_type"`
	ExecutionConfig json.RawMessage `json:"execution_config"`
	ContextFields  []string        `json:"context_fields,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

// Resource represents an MCP resource endpoint
type Resource struct {
	ID        string          `json:"id"`
	ServerID  string          `json:"server_id"`
	Name      string          `json:"name"`
	URI       string          `json:"uri"`
	MimeType  string          `json:"mime_type"`
	Handler   json.RawMessage `json:"handler"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// Prompt represents an MCP prompt template
type Prompt struct {
	ID          string          `json:"id"`
	ServerID    string          `json:"server_id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Template    string          `json:"template"`
	Arguments   json.RawMessage `json:"arguments"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// ContextConfig defines context injection configuration
type ContextConfig struct {
	ID         string          `json:"id"`
	ServerID   string          `json:"server_id"`
	Name       string          `json:"name"`
	SourceType string          `json:"source_type"` // header, jwt, database, custom
	Config     json.RawMessage `json:"config"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
}

// PolicyRuleType defines the type of policy rule
type PolicyRuleType string

const (
	PolicyRuleApproval   PolicyRuleType = "approval_required"
	PolicyRuleMaxValue   PolicyRuleType = "max_value"
	PolicyRuleAllowedRoles PolicyRuleType = "allowed_roles"
	PolicyRuleTimeWindow PolicyRuleType = "time_window"
	PolicyRuleRateLimit  PolicyRuleType = "rate_limit"
	PolicyRuleCustom     PolicyRuleType = "custom"
)

// Policy represents an AI governance policy for tools
type Policy struct {
	ID          string          `json:"id"`
	ToolID      string          `json:"tool_id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Rules       []PolicyRule    `json:"rules"`
	Enabled     bool            `json:"enabled"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// PolicyRule defines a single governance rule
type PolicyRule struct {
	ID         string          `json:"id"`
	PolicyID   string          `json:"policy_id"`
	Type       PolicyRuleType  `json:"type"`
	Config     json.RawMessage `json:"config"`
	Priority   int             `json:"priority"`
	FailAction string          `json:"fail_action"` // deny, warn, approve
}

// ToolExecution records tool execution history for healing
type ToolExecution struct {
	ID           string          `json:"id"`
	ToolID       string          `json:"tool_id"`
	ServerID     string          `json:"server_id"`
	Input        json.RawMessage `json:"input"`
	Output       json.RawMessage `json:"output,omitempty"`
	Error        string          `json:"error,omitempty"`
	StatusCode   int             `json:"status_code"`
	DurationMs   int64           `json:"duration_ms"`
	Success      bool            `json:"success"`
	HealingApplied bool          `json:"healing_applied"`
	CreatedAt    time.Time       `json:"created_at"`
}

// HealingSuggestion represents an auto-repair suggestion
type HealingSuggestion struct {
	ID             string          `json:"id"`
	ToolID         string          `json:"tool_id"`
	ErrorPattern   string          `json:"error_pattern"`
	SuggestionType string          `json:"suggestion_type"` // refresh_token, update_schema, retry, etc.
	Suggestion     json.RawMessage `json:"suggestion"`
	AutoApply      bool            `json:"auto_apply"`
	Applied        bool            `json:"applied"`
	CreatedAt      time.Time       `json:"created_at"`
}

// ServerComposition represents composed MCP servers
type ServerComposition struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	ServerIDs   []string `json:"server_ids"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// API Request/Response types
type CreateServerRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Version     string `json:"version"`
}

type CreateToolRequest struct {
	ServerID        string          `json:"server_id" binding:"required"`
	Name            string          `json:"name" binding:"required"`
	Description     string          `json:"description"`
	InputSchema     json.RawMessage `json:"input_schema"`
	OutputSchema    json.RawMessage `json:"output_schema"`
	ExecutionType   ExecutionType   `json:"execution_type" binding:"required"`
	ExecutionConfig json.RawMessage `json:"execution_config"`
	ContextFields   []string        `json:"context_fields"`
}

type CreateResourceRequest struct {
	ServerID string          `json:"server_id" binding:"required"`
	Name     string          `json:"name" binding:"required"`
	URI      string          `json:"uri" binding:"required"`
	MimeType string          `json:"mime_type"`
	Handler  json.RawMessage `json:"handler"`
}

type CreatePromptRequest struct {
	ServerID    string          `json:"server_id" binding:"required"`
	Name        string          `json:"name" binding:"required"`
	Description string          `json:"description"`
	Template    string          `json:"template" binding:"required"`
	Arguments   json.RawMessage `json:"arguments"`
}

type CreatePolicyRequest struct {
	ToolID      string       `json:"tool_id" binding:"required"`
	Name        string       `json:"name" binding:"required"`
	Description string       `json:"description"`
	Rules       []PolicyRule `json:"rules"`
	Enabled     bool         `json:"enabled"`
}

type TestToolRequest struct {
	Input   json.RawMessage `json:"input"`
	Context json.RawMessage `json:"context,omitempty"`
}

type TestToolResponse struct {
	Success  bool            `json:"success"`
	Output   json.RawMessage `json:"output,omitempty"`
	Error    string          `json:"error,omitempty"`
	Duration int64           `json:"duration_ms"`
}

type PolicyEvaluationResult struct {
	Allowed     bool     `json:"allowed"`
	Reason      string   `json:"reason,omitempty"`
	ViolatedRules []string `json:"violated_rules,omitempty"`
}
