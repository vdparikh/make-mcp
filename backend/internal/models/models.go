package models

import (
	"encoding/json"
	"time"
)

// ServerStatus represents the publication status of a server
type ServerStatus string

const (
	ServerStatusDraft     ServerStatus = "draft"
	ServerStatusPublished ServerStatus = "published"
	ServerStatusArchived  ServerStatus = "archived"
)

// Server represents an MCP server configuration
type Server struct {
	ID             string          `json:"id"`
	Name           string          `json:"name"`
	Description    string          `json:"description"`
	Version        string          `json:"version"`
	Icon           string          `json:"icon,omitempty"`
	Status         ServerStatus    `json:"status"`
	PublishedAt    *time.Time      `json:"published_at,omitempty"`
	LatestVersion  string          `json:"latest_version,omitempty"`
	OwnerID        string          `json:"owner_id,omitempty"`
	IsPublic       bool            `json:"is_public"`
	Downloads      int             `json:"downloads"`
	SecurityScore  *int            `json:"security_score,omitempty"`  // 0-100, set for marketplace list/detail
	SecurityGrade  *string         `json:"security_grade,omitempty"`  // A/B/C/D/F
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
	Tools          []Tool          `json:"tools,omitempty"`
	Resources      []Resource      `json:"resources,omitempty"`
	Prompts        []Prompt        `json:"prompts,omitempty"`
	AuthConfig     json.RawMessage `json:"auth_config,omitempty"`
}

// ServerVersion represents a published snapshot of a server
type ServerVersion struct {
	ID           string          `json:"id"`
	ServerID     string          `json:"server_id"`
	Version      string          `json:"version"`
	ReleaseNotes string          `json:"release_notes"`
	Snapshot     json.RawMessage `json:"snapshot"`
	PublishedBy  string          `json:"published_by"`
	PublishedAt  time.Time       `json:"published_at"`
}

// PublishRequest represents a request to publish a server version
type PublishRequest struct {
	Version      string `json:"version" binding:"required"`
	ReleaseNotes string `json:"release_notes"`
	IsPublic     bool   `json:"is_public"`
}

// ExecutionType defines how a tool is executed
type ExecutionType string

const (
	ExecutionTypeRestAPI    ExecutionType = "rest_api"
	ExecutionTypeGraphQL    ExecutionType = "graphql"
	ExecutionTypeDatabase   ExecutionType = "database"
	ExecutionTypeJavaScript ExecutionType = "javascript"
	ExecutionTypePython     ExecutionType = "python"
	ExecutionTypeWebhook    ExecutionType = "webhook"
	ExecutionTypeCLI        ExecutionType = "cli"
	ExecutionTypeFlow       ExecutionType = "flow"
)

// OutputDisplay defines how tool output is presented in MCP Apps–capable clients (e.g. table, card, json).
const (
	OutputDisplayJSON  = "json"
	OutputDisplayTable = "table"
	OutputDisplayCard  = "card"
)

// Tool represents an MCP tool definition
type Tool struct {
	ID              string          `json:"id"`
	ServerID        string          `json:"server_id"`
	Name            string          `json:"name"`
	Description     string          `json:"description"`
	InputSchema     json.RawMessage `json:"input_schema"`
	OutputSchema    json.RawMessage `json:"output_schema"`
	ExecutionType   ExecutionType   `json:"execution_type"`
	ExecutionConfig json.RawMessage `json:"execution_config"`
	ContextFields   []string        `json:"context_fields,omitempty"`
	OutputDisplay   string          `json:"output_display,omitempty"`   // "json" or "table" — MCP Apps style
	ReadOnlyHint    bool            `json:"read_only_hint,omitempty"`   // if true, tool is read-only; gateways may block write operations
	DestructiveHint bool            `json:"destructive_hint,omitempty"` // if true, tool can modify/delete; clients may require confirmation
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
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
	OwnerID     string   `json:"owner_id,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// API Request/Response types
type CreateServerRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Version     string `json:"version"`
	Icon        string `json:"icon"`
	OwnerID     string `json:"-"` // Set by API from auth; not accepted from client
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
	OutputDisplay   string          `json:"output_display"`   // "json" or "table"
	ReadOnlyHint    bool            `json:"read_only_hint"`   // tool is read-only; gateways may enforce
	DestructiveHint bool            `json:"destructive_hint"` // tool can modify/delete; require user confirmation
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

// Flow represents a visual tool flow/pipeline
type Flow struct {
	ID          string          `json:"id"`
	ServerID    string          `json:"server_id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Nodes       json.RawMessage `json:"nodes"`
	Edges       json.RawMessage `json:"edges"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// FlowNode represents a node in a flow
type FlowNode struct {
	ID       string          `json:"id"`
	Type     string          `json:"type"`
	Position FlowPosition    `json:"position"`
	Data     json.RawMessage `json:"data"`
}

type FlowPosition struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// FlowEdge represents a connection between nodes
type FlowEdge struct {
	ID           string `json:"id"`
	Source       string `json:"source"`
	Target       string `json:"target"`
	SourceHandle string `json:"sourceHandle,omitempty"`
	TargetHandle string `json:"targetHandle,omitempty"`
}

type CreateFlowRequest struct {
	ServerID    string          `json:"server_id" binding:"required"`
	Name        string          `json:"name" binding:"required"`
	Description string          `json:"description"`
	Nodes       json.RawMessage `json:"nodes" binding:"required"`
	Edges       json.RawMessage `json:"edges" binding:"required"`
}

type UpdateFlowRequest struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Nodes       json.RawMessage `json:"nodes"`
	Edges       json.RawMessage `json:"edges"`
}

// FlowExecutionRequest for testing a flow
type FlowExecutionRequest struct {
	FlowID  string          `json:"flow_id" binding:"required"`
	Input   json.RawMessage `json:"input"`
	Context json.RawMessage `json:"context,omitempty"`
}

type FlowExecutionResponse struct {
	Success    bool              `json:"success"`
	Output     json.RawMessage   `json:"output,omitempty"`
	Error      string            `json:"error,omitempty"`
	Duration   int64             `json:"duration_ms"`
	NodeResults []NodeResult     `json:"node_results,omitempty"`
}

type NodeResult struct {
	NodeID   string          `json:"node_id"`
	NodeType string          `json:"node_type"`
	Success  bool            `json:"success"`
	Output   json.RawMessage `json:"output,omitempty"`
	Error    string          `json:"error,omitempty"`
	Duration int64           `json:"duration_ms"`
}

// User represents a user account
type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	Name         string    `json:"name"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Auth request/response types
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Name     string `json:"name" binding:"required"`
	Password string `json:"password" binding:"required,min=6"`
}

type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

type UserResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}
