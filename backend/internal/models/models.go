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
	ID                        string          `json:"id"`
	Name                      string          `json:"name"`
	Description               string          `json:"description"`
	Version                   string          `json:"version"`
	Icon                      string          `json:"icon,omitempty"`
	Status                    ServerStatus    `json:"status"`
	PublishedAt               *time.Time      `json:"published_at,omitempty"`
	LatestVersion             string          `json:"latest_version,omitempty"`
	OwnerID                   string          `json:"owner_id,omitempty"`
	IsPublic                  bool            `json:"is_public"`
	Downloads                 int             `json:"downloads"`
	HostedRunning             bool            `json:"hosted_running"`
	HostedVirtual             bool            `json:"hosted_virtual,omitempty"`
	SecurityScore             *int            `json:"security_score,omitempty"`              // 0-100, set for marketplace list/detail
	SecurityGrade             *string         `json:"security_grade,omitempty"`              // A/B/C/D/F
	ObservabilityReportingKey string          `json:"observability_reporting_key,omitempty"` // key for runtime to report events
	HostedAccessKey           string          `json:"-"`                                     // secret key for hosted endpoint access boundary
	HostedAuthMode            string          `json:"hosted_auth_mode,omitempty"`            // "bearer_token" or "no_auth"
	RequireCallerIdentity    bool            `json:"require_caller_identity,omitempty"`     // independent toggle for X-Make-MCP-Caller-Id
	CreatedAt                 time.Time       `json:"created_at"`
	UpdatedAt                 time.Time       `json:"updated_at"`
	Tools                     []Tool          `json:"tools,omitempty"`
	Resources                 []Resource      `json:"resources,omitempty"`
	Prompts                   []Prompt        `json:"prompts,omitempty"`
	AuthConfig                json.RawMessage `json:"auth_config,omitempty"`
	EnvProfiles               json.RawMessage `json:"env_profiles,omitempty"` // Dev/Staging/Prod: base_url, db_url, env vars
	// HostedSecurityConfig is per-environment auth matrix (IP, OIDC, mTLS); see docs/hosted-security.md.
	HostedSecurityConfig json.RawMessage `json:"hosted_security_config,omitempty"`
	// HostedRuntimeConfig is isolation tier, resource limits, and egress policy for hosted containers; see docs/hosted-runtime-isolation.md.
	HostedRuntimeConfig json.RawMessage `json:"hosted_runtime_config,omitempty"`
}

// HostedSecurityAuditEvent is an admin/security action for hosted MCP (rotation, config change).
type HostedSecurityAuditEvent struct {
	ID           string          `json:"id"`
	ServerID     string          `json:"server_id"`
	ActorUserID  string          `json:"actor_user_id,omitempty"`
	Action       string          `json:"action"`
	ResourceType string          `json:"resource_type,omitempty"`
	ResourceID   string          `json:"resource_id,omitempty"`
	Metadata     json.RawMessage `json:"metadata,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
}

// EnvProfile is one environment profile (dev, staging, prod) for a server.
type EnvProfile struct {
	BaseURL string            `json:"base_url,omitempty"` // For rest_api/graphql/webhook tools
	DBURL   string            `json:"db_url,omitempty"`   // For database tools
	Env     map[string]string `json:"env,omitempty"`      // Additional env vars (documentation / codegen)
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
	OutputDisplayImage = "image"
	OutputDisplayForm  = "form"
	OutputDisplayChart = "chart"
	OutputDisplayMap   = "map"
)

// Tool represents an MCP tool definition
type Tool struct {
	ID                  string          `json:"id"`
	ServerID            string          `json:"server_id"`
	Name                string          `json:"name"`
	Description         string          `json:"description"`
	InputSchema         json.RawMessage `json:"input_schema"`
	OutputSchema        json.RawMessage `json:"output_schema"`
	ExecutionType       ExecutionType   `json:"execution_type"`
	ExecutionConfig     json.RawMessage `json:"execution_config"`
	ContextFields       []string        `json:"context_fields,omitempty"`
	OutputDisplay       string          `json:"output_display,omitempty"`          // json | table | card | image | form — MCP Apps style
	OutputDisplayConfig json.RawMessage `json:"output_display_config,omitempty"` // maps result fields to card/image/form widgets
	ReadOnlyHint        bool            `json:"read_only_hint,omitempty"`        // if true, tool is read-only; gateways may block write operations
	DestructiveHint     bool            `json:"destructive_hint,omitempty"`      // if true, tool can modify/delete; clients may require confirmation
	CreatedAt           time.Time       `json:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at"`
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
	PolicyRuleApproval     PolicyRuleType = "approval_required"
	PolicyRuleMaxValue     PolicyRuleType = "max_value"
	PolicyRuleAllowedRoles PolicyRuleType = "allowed_roles"
	PolicyRuleTimeWindow   PolicyRuleType = "time_window"
	PolicyRuleRateLimit    PolicyRuleType = "rate_limit"
	PolicyRuleCustom       PolicyRuleType = "custom"
)

// Policy represents an AI governance policy for tools
type Policy struct {
	ID          string       `json:"id"`
	ToolID      string       `json:"tool_id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Rules       []PolicyRule `json:"rules"`
	Enabled     bool         `json:"enabled"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
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

// ToolTestPreset stores saved test cases (input + context) per tool per user
type ToolTestPreset struct {
	ID          string          `json:"id"`
	ToolID      string          `json:"tool_id"`
	UserID      string          `json:"user_id"`
	Name        string          `json:"name"`
	InputJSON   json.RawMessage `json:"input_json"`
	ContextJSON json.RawMessage `json:"context_json"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// ToolExecution records tool execution history for healing and observability
type ToolExecution struct {
	ID               string          `json:"id"`
	ToolID           string          `json:"tool_id"`
	ServerID         string          `json:"server_id"`
	ToolName         string          `json:"tool_name,omitempty"`      // set for runtime-reported events
	Source           string          `json:"source,omitempty"`         // "playground" | "runtime"
	ClientUserID     string          `json:"client_user_id,omitempty"` // optional: end-user/tenant identifier
	ClientAgent      string          `json:"client_agent,omitempty"`   // optional: e.g. "Cursor", "Claude Desktop"
	ClientToken      string          `json:"client_token,omitempty"`   // optional: API key/token for correlation
	Input            json.RawMessage `json:"input,omitempty"`
	Output           json.RawMessage `json:"output,omitempty"`
	Error            string          `json:"error,omitempty"`
	StatusCode       int             `json:"status_code"`
	DurationMs       int64           `json:"duration_ms"`
	Success          bool            `json:"success"`
	HealingApplied   bool            `json:"healing_applied"`
	RepairSuggestion string          `json:"repair_suggestion,omitempty"` // from runtime or healing
	CreatedAt        time.Time       `json:"created_at"`
}

// HostedSession tracks runtime state for hosted MCP containers.
type HostedSession struct {
	ID              string     `json:"id"`
	UserID          string     `json:"user_id"`
	ServerID        string     `json:"server_id"`
	SnapshotVersion string     `json:"snapshot_version,omitempty"`
	ContainerID     string     `json:"container_id,omitempty"`
	HostPort        string     `json:"host_port,omitempty"`
	Status          string     `json:"status"` // running|starting|stopped|error
	Health          string     `json:"health"` // healthy|unhealthy|unknown
	LastUsedAt      *time.Time `json:"last_used_at,omitempty"`
	LastEnsuredAt   *time.Time `json:"last_ensured_at,omitempty"`
	StartedAt       *time.Time `json:"started_at,omitempty"`
	StoppedAt       *time.Time `json:"stopped_at,omitempty"`
	LastError       string     `json:"last_error,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// HostedCallerAPIKey stores user-owned caller identity credentials for hosted requests.
type HostedCallerAPIKey struct {
	ID           string     `json:"id"`
	OwnerUserID  string     `json:"owner_user_id"`
	KeyID        string     `json:"key_id"`
	CallerUserID string     `json:"caller_user_id"` // verified identity resolved from this key
	TenantID     string     `json:"tenant_id,omitempty"`
	Scopes       []string   `json:"scopes,omitempty"`
	AllowAlias   bool       `json:"allow_alias"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
	RevokedAt    *time.Time `json:"revoked_at,omitempty"`
	CreatedBy    string     `json:"created_by,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

// HostedCallerIdentity is the verified identity resolved from a caller API key.
type HostedCallerIdentity struct {
	CallerUserID string
	TenantID     string
	Scopes       []string
	AllowAlias   bool
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
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	ServerIDs   []string  `json:"server_ids"`
	OwnerID     string    `json:"owner_id,omitempty"`
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
	ServerID              string          `json:"server_id" binding:"required"`
	Name                  string          `json:"name" binding:"required"`
	Description           string          `json:"description"`
	InputSchema           json.RawMessage `json:"input_schema"`
	OutputSchema          json.RawMessage `json:"output_schema"`
	ExecutionType         ExecutionType   `json:"execution_type" binding:"required"`
	ExecutionConfig       json.RawMessage `json:"execution_config"`
	ContextFields         []string        `json:"context_fields"`
	OutputDisplay         string          `json:"output_display"`                   // json | table | card | image | form
	OutputDisplayConfig   json.RawMessage `json:"output_display_config,omitempty"` // optional field mapping
	ReadOnlyHint          bool            `json:"read_only_hint"`                  // tool is read-only; gateways may enforce
	DestructiveHint       bool            `json:"destructive_hint"`                 // tool can modify/delete; require user confirmation
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
	Success         bool                   `json:"success"`
	Output          json.RawMessage        `json:"output,omitempty"`
	Error           string                 `json:"error,omitempty"`
	Duration        int64                  `json:"duration_ms"`
	InjectedContext map[string]interface{} `json:"injected_context,omitempty"` // context actually passed to the tool (for simulation visibility)
}

type PolicyEvaluationResult struct {
	Allowed       bool     `json:"allowed"`
	Reason        string   `json:"reason,omitempty"`
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
	Success     bool            `json:"success"`
	Output      json.RawMessage `json:"output,omitempty"`
	Error       string          `json:"error,omitempty"`
	Duration    int64           `json:"duration_ms"`
	NodeResults []NodeResult    `json:"node_results,omitempty"`
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
// LoginRequest is unused; login is passkey-only via WebAuthn login/begin and login/finish.
type LoginRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type RegisterRequest struct {
	Email string `json:"email" binding:"required,email"`
	Name  string `json:"name" binding:"required"`
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

// Observability: runtime-reported tool execution events (from generated MCP server)
type ObservabilityEventPayload struct {
	ToolName         string `json:"tool_name" binding:"required"`
	DurationMs       int64  `json:"duration_ms"`
	Success          bool   `json:"success"`
	Error            string `json:"error,omitempty"`
	RepairSuggestion string `json:"repair_suggestion,omitempty"`
	// Optional: identify who and which client the call came from (when many users use the same MCP)
	ClientUserID string `json:"client_user_id,omitempty"` // end-user or tenant identifier
	ClientAgent  string `json:"client_agent,omitempty"`   // e.g. "Cursor", "Claude Desktop", "VS Code"
	ClientToken  string `json:"client_token,omitempty"`   // optional API key/token for correlation
}

type ObservabilityEventsRequest struct {
	Key    string                      `json:"key" binding:"required"` // observability_reporting_key
	Events []ObservabilityEventPayload `json:"events" binding:"required"`
}

// ObservabilitySummaryResponse for the server observability tab (enable reporting + env vars)
type ObservabilitySummaryResponse struct {
	ReportingKey      string                 `json:"reporting_key,omitempty"`
	EndpointURL       string                 `json:"endpoint_url,omitempty"`
	RecentEvents      []ToolExecution        `json:"recent_events"`
	LatencyByTool     []ToolLatencyStat      `json:"latency_by_tool"`
	FailuresByTool    []ToolFailureStat      `json:"failures_by_tool"`
	RepairSuggestions []RepairSuggestionItem `json:"repair_suggestions"`
}

// ServerSummary for filter dropdowns
type ServerSummary struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ObservabilityDashboardResponse for the global Observability page (filter by server, tool)
type ObservabilityDashboardResponse struct {
	Servers           []ServerSummary        `json:"servers"`
	RecentEvents      []ToolExecution        `json:"recent_events"`
	LatencyByTool     []ToolLatencyStat      `json:"latency_by_tool"`
	FailuresByTool    []ToolFailureStat      `json:"failures_by_tool"`
	RepairSuggestions []RepairSuggestionItem `json:"repair_suggestions"`
}

type ToolLatencyStat struct {
	ToolName string  `json:"tool_name"`
	ToolID   string  `json:"tool_id"`
	Count    int     `json:"count"`
	AvgMs    float64 `json:"avg_ms"`
	P95Ms    int64   `json:"p95_ms"`
}

type ToolFailureStat struct {
	ToolName  string `json:"tool_name"`
	ToolID    string `json:"tool_id"`
	Count     int    `json:"count"`
	LastError string `json:"last_error,omitempty"`
}

type RepairSuggestionItem struct {
	ToolName   string    `json:"tool_name"`
	ToolID     string    `json:"tool_id"`
	Suggestion string    `json:"suggestion"`
	CreatedAt  time.Time `json:"created_at"`
}
