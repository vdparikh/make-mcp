export type ExecutionType = 
  | 'rest_api' 
  | 'graphql' 
  | 'database' 
  | 'javascript' 
  | 'python' 
  | 'webhook';

export interface Server {
  id: string;
  name: string;
  description: string;
  version: string;
  created_at: string;
  updated_at: string;
  tools?: Tool[];
  resources?: Resource[];
  prompts?: Prompt[];
  auth_config?: Record<string, unknown>;
}

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
  created_at: string;
  updated_at: string;
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
  input: Record<string, unknown>;
  output?: Record<string, unknown>;
  error?: string;
  status_code: number;
  duration_ms: number;
  success: boolean;
  healing_applied: boolean;
  created_at: string;
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
