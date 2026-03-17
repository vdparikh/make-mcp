export type ExecutionType = 
  | 'rest_api' 
  | 'graphql' 
  | 'database' 
  | 'javascript' 
  | 'python' 
  | 'webhook'
  | 'cli'
  | 'flow';

export type ServerStatus = 'draft' | 'published' | 'archived';

export interface Server {
  id: string;
  name: string;
  description: string;
  version: string;
  icon?: string;
  status: ServerStatus;
  published_at?: string;
  latest_version?: string;
  owner_id?: string;
  is_public: boolean;
  downloads: number;
  security_score?: number;
  security_grade?: string;
  observability_reporting_key?: string;
  created_at: string;
  updated_at: string;
  tools?: Tool[];
  resources?: Resource[];
  prompts?: Prompt[];
  auth_config?: Record<string, unknown>;
}

export interface ServerVersion {
  id: string;
  server_id: string;
  version: string;
  release_notes: string;
  snapshot: Record<string, unknown>;
  published_by: string;
  published_at: string;
}

export interface PublishRequest {
  version: string;
  release_notes: string;
  is_public: boolean;
}

/** Security score from SlowMist MCP Security Checklist */
export interface SecurityCriterionResult {
  id: string;
  name: string;
  priority: 'high' | 'medium' | 'low';
  met: boolean;
  reason?: string;
}

export interface SecurityScoreResult {
  score: number;
  grade: string;
  max_points: number;
  earned: number;
  criteria: SecurityCriterionResult[];
  checklist_url: string;
}

/** How tool output is presented in MCP Apps–capable clients (e.g. table, card, json) */
export type OutputDisplay = 'json' | 'table' | 'card';

export interface Tool {
  id: string;
  server_id: string;
  name: string;
  description: string;
  input_schema: Record<string, unknown>;
  output_schema: Record<string, unknown>;
  execution_type: ExecutionType;
  execution_config: Record<string, unknown>;
  context_fields?: string[];
  output_display?: OutputDisplay;
  /** If true, tool is read-only; gateways may block write operations (MCP security best practice). */
  read_only_hint?: boolean;
  /** If true, tool can modify/delete data; clients should require user confirmation (MCP security best practice). */
  destructive_hint?: boolean;
  created_at: string;
  updated_at: string;
}

/** MCP Apps format: clients that support it render the widget; others use text fallback */
export type MCPAppPayload =
  | {
      widget: 'table';
      props: {
        columns: { key: string; label: string }[];
        rows: Record<string, unknown>[];
      };
    }
  | {
      widget: 'card';
      props: {
        content: string;
        title?: string;
      };
    };

export function isMCPAppOutput(output: unknown): output is { text?: string; _mcp_app: MCPAppPayload } {
  return (
    typeof output === 'object' &&
    output !== null &&
    '_mcp_app' in output &&
    typeof (output as { _mcp_app: unknown })._mcp_app === 'object'
  );
}

export interface Resource {
  id: string;
  server_id: string;
  name: string;
  uri: string;
  mime_type: string;
  handler: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export interface Prompt {
  id: string;
  server_id: string;
  name: string;
  description: string;
  template: string;
  arguments: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export interface ContextConfig {
  id: string;
  server_id: string;
  name: string;
  source_type: 'header' | 'jwt' | 'query' | 'database' | 'custom';
  config: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export type PolicyRuleType = 
  | 'approval_required'
  | 'max_value'
  | 'allowed_roles'
  | 'time_window'
  | 'rate_limit'
  | 'custom';

export interface PolicyRule {
  id: string;
  policy_id: string;
  type: PolicyRuleType;
  config: Record<string, unknown>;
  priority: number;
  fail_action: 'deny' | 'warn' | 'approve';
}

export interface Policy {
  id: string;
  tool_id: string;
  name: string;
  description: string;
  rules: PolicyRule[];
  enabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface ToolExecution {
  id: string;
  tool_id: string;
  server_id: string;
  tool_name?: string;
  source?: string;
  client_user_id?: string;
  client_agent?: string;
  client_token?: string;
  input?: Record<string, unknown>;
  output?: Record<string, unknown>;
  error?: string;
  status_code?: number;
  duration_ms: number;
  success: boolean;
  healing_applied?: boolean;
  repair_suggestion?: string;
  created_at: string;
}

export interface ToolLatencyStat {
  tool_name: string;
  tool_id: string;
  count: number;
  avg_ms: number;
  p95_ms: number;
}

export interface ToolFailureStat {
  tool_name: string;
  tool_id: string;
  count: number;
  last_error?: string;
}

export interface RepairSuggestionItem {
  tool_name: string;
  tool_id: string;
  suggestion: string;
  created_at: string;
}

export interface ObservabilitySummaryResponse {
  reporting_key?: string;
  endpoint_url?: string;
  recent_events: ToolExecution[];
  latency_by_tool: ToolLatencyStat[];
  failures_by_tool: ToolFailureStat[];
  repair_suggestions: RepairSuggestionItem[];
}

export interface ServerSummary {
  id: string;
  name: string;
}

export interface ObservabilityDashboardResponse {
  servers: ServerSummary[];
  recent_events: ToolExecution[];
  latency_by_tool: ToolLatencyStat[];
  failures_by_tool: ToolFailureStat[];
  repair_suggestions: RepairSuggestionItem[];
}

export interface HealingSuggestion {
  id: string;
  tool_id: string;
  error_pattern: string;
  suggestion_type: string;
  suggestion: {
    type: string;
    message: string;
    auto_fix: boolean;
    fix_action: string;
    fix_params?: Record<string, unknown>;
    confidence: number;
    description: string;
  };
  auto_apply: boolean;
  applied: boolean;
  created_at: string;
}

export interface ServerComposition {
  id: string;
  name: string;
  description: string;
  server_ids: string[];
  created_at: string;
  updated_at: string;
}

export interface TestToolResponse {
  success: boolean;
  output?: Record<string, unknown>;
  error?: string;
  duration_ms: number;
  /** Context actually passed to the tool (for simulation visibility) */
  injected_context?: Record<string, unknown>;
}

export interface SchemaField {
  name: string;
  type: 'string' | 'number' | 'boolean' | 'object' | 'array';
  required: boolean;
  description?: string;
}

export interface PolicyEvaluationResult {
  allowed: boolean;
  reason?: string;
  violated_rules?: string[];
  requires_approval?: boolean;
  approval_reason?: string;
}
