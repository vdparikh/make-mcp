import axios from 'axios';
import type {
  Server,
  Tool,
  Resource,
  Prompt,
  Policy,
  ContextConfig,
  ToolExecution,
  HealingSuggestion,
  ServerComposition,
  TestToolResponse,
  ToolTestPreset,
  PolicyEvaluationResult,
  PolicyEvaluationResultDetailed,
  EnvProfileKey,
  EnvProfilesMap,
  ServerVersion,
  PublishRequest,
  SecurityScoreResult,
  ObservabilitySummaryResponse,
  ObservabilityDashboardResponse,
} from '../types';

const api = axios.create({
  baseURL: '/api',
  headers: {
    'Content-Type': 'application/json',
  },
});

// Add auth token to requests
api.interceptors.request.use((config) => {
  const token = localStorage.getItem('token');
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

// Handle 401 responses - but not for auth endpoints
api.interceptors.response.use(
  (response) => response,
  (error) => {
    const isAuthEndpoint = error.config?.url?.startsWith('/auth/');
    const isOnLoginPage = window.location.pathname === '/login' || window.location.pathname === '/register';
    
    if (error.response?.status === 401 && !isAuthEndpoint && !isOnLoginPage) {
      localStorage.removeItem('token');
      window.location.href = '/login';
    }
    return Promise.reject(error);
  }
);

// Auth types
interface AuthResponse {
  token: string;
  user: {
    id: string;
    email: string;
    name: string;
    created_at: string;
  };
}

interface UserResponse {
  id: string;
  email: string;
  name: string;
  created_at: string;
}

// Auth APIs (passkey-only; no password)
export const registerAccount = async (email: string, name: string): Promise<{ message: string; user: { id: string; email: string; name: string } }> => {
  const { data } = await api.post<{ message: string; user: { id: string; email: string; name: string } }>('/auth/register', { email, name });
  return data;
};

export const webauthnRegisterBegin = async (email: string): Promise<{ session_id: string; options: { publicKey: PublicKeyCredentialCreationOptionsJSON } }> => {
  const { data } = await api.post<{ session_id: string; options: { publicKey: PublicKeyCredentialCreationOptionsJSON } }>('/auth/webauthn/register/begin', { email });
  return data;
};

export const webauthnRegisterFinish = async (sessionId: string, response: CredentialCreationResponseJSON): Promise<AuthResponse> => {
  const { data } = await api.post<AuthResponse>('/auth/webauthn/register/finish', { session_id: sessionId, response });
  return data;
};

export const webauthnLoginBegin = async (email: string): Promise<{ session_id: string; options: { publicKey: PublicKeyCredentialRequestOptionsJSON } }> => {
  const { data } = await api.post<{ session_id: string; options: { publicKey: PublicKeyCredentialRequestOptionsJSON } }>('/auth/webauthn/login/begin', { email });
  return data;
};

export const webauthnLoginFinish = async (sessionId: string, response: CredentialAssertionResponseJSON): Promise<AuthResponse> => {
  const { data } = await api.post<AuthResponse>('/auth/webauthn/login/finish', { session_id: sessionId, response });
  return data;
};

// Types for WebAuthn options (backend sends JSON with base64url challenge/user.id etc.)
export interface PublicKeyCredentialCreationOptionsJSON {
  rp: { name: string; id?: string };
  user: { id: string; name: string; displayName: string };
  challenge: string;
  pubKeyCredParams: { type: string; alg: number }[];
  timeout?: number;
  attestation?: string;
  authenticatorSelection?: Record<string, unknown>;
}

export interface PublicKeyCredentialRequestOptionsJSON {
  challenge: string;
  timeout?: number;
  rpId?: string;
  allowCredentials?: { type: string; id: string; transports?: string[] }[];
  userVerification?: string;
}

export interface CredentialCreationResponseJSON {
  id: string;
  rawId: string;
  type: string;
  response: {
    clientDataJSON: string;
    attestationObject: string;
    transports?: string[];
  };
}

export interface CredentialAssertionResponseJSON {
  id: string;
  rawId: string;
  type: string;
  response: {
    clientDataJSON: string;
    authenticatorData: string;
    signature: string;
    userHandle: string | null;
  };
}

export const getCurrentUser = async (): Promise<UserResponse> => {
  const { data } = await api.get<UserResponse>('/auth/me');
  return data;
};

// Server APIs (no cache so list is always fresh for current user)
export const listServers = async (): Promise<Server[]> => {
  const { data } = await api.get<Server[]>('/servers', {
    headers: { 'Cache-Control': 'no-cache', 'Pragma': 'no-cache' },
    params: { _t: Date.now() },
  });
  return data ?? [];
};

export const getServer = async (id: string): Promise<Server> => {
  const { data } = await api.get<Server>(`/servers/${id}`);
  return data;
};

export const createServer = async (server: Partial<Server>): Promise<Server> => {
  const { data } = await api.post<Server>('/servers', server);
  return data;
};

export const createDemoServer = async (): Promise<Server> => {
  const { data } = await api.post<Server>('/servers/demo');
  return data;
};

/** Opinionated production-style template: tools (REST + webhook), resources, prompts, context, policies */
export const createBlueprintServer = async (): Promise<Server> => {
  const { data } = await api.post<Server>('/servers/blueprint');
  return data;
};

/** MCP Apps Lab: one tool per widget (table, card, image, chart, map, form) for hosts like MCP Jam */
export const createMCPAppsLabServer = async (): Promise<Server> => {
  const { data } = await api.post<Server>('/servers/mcp-apps-lab');
  return data;
};

export const updateServer = async (id: string, server: Partial<Server>): Promise<Server> => {
  const { data } = await api.put<Server>(`/servers/${id}`, server);
  return data;
};

export const deleteServer = async (id: string): Promise<void> => {
  await api.delete(`/servers/${id}`);
};

export const generateServer = async (id: string, envProfile?: 'dev' | 'staging' | 'prod'): Promise<Blob> => {
  const params = envProfile ? { env_profile: envProfile } : {};
  const { data } = await api.post(`/servers/${id}/generate`, {}, {
    params,
    responseType: 'blob',
  });
  return data;
};

export interface GitHubExportOptions {
  token: string;
  owner: string;
  repo: string;
  branch?: string;
  commit_message?: string;
  create_repo?: boolean;
  private?: boolean;
  description?: string;
}

export interface GitHubExportResponse {
  success: boolean;
  repo_url: string;
  commit_sha: string;
  files: number;
  message: string;
}

export const githubExport = async (id: string, options: GitHubExportOptions): Promise<GitHubExportResponse> => {
  const { data } = await api.post<GitHubExportResponse>(`/servers/${id}/github-export`, options);
  return data;
};

// Tool APIs
export const createTool = async (tool: Partial<Tool>): Promise<Tool> => {
  const { data } = await api.post<Tool>('/tools', tool);
  return data;
};

export const getTool = async (id: string): Promise<Tool> => {
  const { data } = await api.get<Tool>(`/tools/${id}`);
  return data;
};

export const updateTool = async (id: string, tool: Partial<Tool>): Promise<Tool> => {
  const { data } = await api.put<Tool>(`/tools/${id}`, tool);
  return data;
};

export const deleteTool = async (id: string): Promise<void> => {
  await api.delete(`/tools/${id}`);
};

export const testTool = async (
  id: string,
  input: Record<string, unknown>,
  context?: Record<string, unknown>,
  envProfile?: EnvProfileKey
): Promise<TestToolResponse> => {
  const qs = envProfile ? `?env_profile=${encodeURIComponent(envProfile)}` : '';
  const { data } = await api.post<TestToolResponse>(`/tools/${id}/test${qs}`, { input, context });
  return data;
};

export const getEnvProfiles = async (serverId: string): Promise<EnvProfilesMap> => {
  const { data } = await api.get<EnvProfilesMap>(`/servers/${serverId}/env-profiles`);
  return data ?? {};
};

export const updateEnvProfiles = async (serverId: string, profiles: EnvProfilesMap): Promise<EnvProfilesMap> => {
  const { data } = await api.put<EnvProfilesMap>(`/servers/${serverId}/env-profiles`, profiles);
  return data ?? {};
};

// Server JSON import/export
export interface ServerJSONExportPayload {
  schema_version: number;
  server: {
    name: string;
    description: string;
    version: string;
    icon?: string;
    status?: string;
    is_public?: boolean;
    env_profiles?: unknown;
  };
  tools: unknown[];
  resources: unknown[];
  prompts: unknown[];
  context_configs: unknown[];
  policies: unknown[];
}

export interface ServerJSONImportResult {
  server: Server;
  tools_created: number;
  resources_created: number;
  prompts_created: number;
  context_configs_created: number;
  policies_created: number;
}

export const exportServerJSON = async (serverId: string): Promise<ServerJSONExportPayload> => {
  const { data } = await api.get<ServerJSONExportPayload>(`/servers/${serverId}/export-json`);
  return data;
};

export const importServerJSON = async (
  payload: ServerJSONExportPayload,
  overrides?: {
    server_name_override?: string;
    description_override?: string;
    icon_override?: string;
  }
): Promise<ServerJSONImportResult> => {
  const body = {
    payload,
    ...(overrides || {}),
  };
  const { data } = await api.post<ServerJSONImportResult>(`/import/server-json`, body);
  return data;
};

export const getToolExecutions = async (id: string): Promise<ToolExecution[]> => {
  const { data } = await api.get<ToolExecution[]>(`/tools/${id}/executions`);
  return data || [];
};

export const getToolPolicies = async (id: string): Promise<Policy[]> => {
  const { data } = await api.get<Policy[]>(`/tools/${id}/policies`);
  return data || [];
};

export const getHealingSuggestions = async (id: string): Promise<HealingSuggestion[]> => {
  const { data } = await api.get<HealingSuggestion[]>(`/tools/${id}/healing`);
  return data || [];
};

export const listToolTestPresets = async (toolId: string): Promise<ToolTestPreset[]> => {
  const { data } = await api.get<ToolTestPreset[]>(`/tools/${toolId}/test-presets`);
  return data || [];
};

export const createToolTestPreset = async (
  toolId: string,
  body: { name: string; input: Record<string, unknown>; context: Record<string, unknown> }
): Promise<ToolTestPreset> => {
  const { data } = await api.post<ToolTestPreset>(`/tools/${toolId}/test-presets`, body);
  return data;
};

export const deleteToolTestPreset = async (toolId: string, presetId: string): Promise<void> => {
  await api.delete(`/tools/${toolId}/test-presets/${presetId}`);
};

// Resource APIs
export const createResource = async (resource: Partial<Resource>): Promise<Resource> => {
  const { data } = await api.post<Resource>('/resources', resource);
  return data;
};

export const deleteResource = async (id: string): Promise<void> => {
  await api.delete(`/resources/${id}`);
};

// Prompt APIs
export const createPrompt = async (prompt: Partial<Prompt>): Promise<Prompt> => {
  const { data } = await api.post<Prompt>('/prompts', prompt);
  return data;
};

export const deletePrompt = async (id: string): Promise<void> => {
  await api.delete(`/prompts/${id}`);
};

// Policy APIs
export const createPolicy = async (policy: Partial<Policy>): Promise<Policy> => {
  const { data } = await api.post<Policy>('/policies', policy);
  return data;
};

export const deletePolicy = async (id: string): Promise<void> => {
  await api.delete(`/policies/${id}`);
};

export const evaluatePolicy = async (
  toolId: string,
  input: Record<string, unknown>,
  context?: Record<string, unknown>
): Promise<PolicyEvaluationResult> => {
  const { data } = await api.post<PolicyEvaluationResult>('/policies/evaluate', {
    tool_id: toolId,
    input,
    context,
  });
  return data;
};

export const evaluatePolicyDetailed = async (
  toolId: string,
  input: Record<string, unknown>,
  context?: Record<string, unknown>
): Promise<PolicyEvaluationResultDetailed> => {
  const { data } = await api.post<PolicyEvaluationResultDetailed>('/policies/evaluate-detailed', {
    tool_id: toolId,
    input,
    context,
  });
  return data;
};

// Context Config APIs
export const getContextConfigs = async (serverId: string): Promise<ContextConfig[]> => {
  const { data } = await api.get<ContextConfig[]>(`/servers/${serverId}/context-configs`);
  return data || [];
};

export const createContextConfig = async (serverId: string, config: Partial<ContextConfig>): Promise<ContextConfig> => {
  const { data } = await api.post<ContextConfig>(`/servers/${serverId}/context-configs`, config);
  return data;
};

export const deleteContextConfig = async (id: string): Promise<void> => {
  await api.delete(`/context-configs/${id}`);
};

// Composition APIs
export const listCompositions = async (): Promise<ServerComposition[]> => {
  const { data } = await api.get<ServerComposition[]>('/compositions');
  return data || [];
};

export const getComposition = async (id: string): Promise<ServerComposition> => {
  const { data } = await api.get<ServerComposition>(`/compositions/${id}`);
  return data;
};

export const createComposition = async (composition: Partial<ServerComposition>): Promise<ServerComposition> => {
  const { data } = await api.post<ServerComposition>('/compositions', composition);
  return data;
};

export const updateComposition = async (id: string, composition: Partial<ServerComposition>): Promise<ServerComposition> => {
  const { data } = await api.put<ServerComposition>(`/compositions/${id}`, composition);
  return data;
};

export const deleteComposition = async (id: string): Promise<void> => {
  await api.delete(`/compositions/${id}`);
};

export interface CompositionExportOptions {
  prefix_tool_names: boolean;
  merge_resources: boolean;
  merge_prompts: boolean;
  env_profile?: EnvProfileKey;
}

export const exportComposition = async (id: string, options: CompositionExportOptions): Promise<Blob> => {
  const { data } = await api.post(`/compositions/${id}/export`, options, {
    responseType: 'blob',
  });
  return data;
};

// OpenAPI Import APIs
export interface OpenAPIPreviewTool {
  Name: string;
  Description: string;
  Method: string;
  Path: string;
  InputSchema: Record<string, unknown>;
  PathParams: string[];
  QueryParams: string[];
}

export interface OpenAPIPreview {
  server: {
    name: string;
    description: string;
    version: string;
    base_url: string;
  };
  tools_count: number;
  tools: OpenAPIPreviewTool[];
  auth: {
    type: string;
    header_name?: string;
    token_url?: string;
    scopes?: string[];
  } | null;
}

export interface OpenAPIImportResult {
  server: Server;
  tools_created: number;
  tools: Tool[];
}

export const previewOpenAPIImport = async (spec: string): Promise<OpenAPIPreview> => {
  const { data } = await api.post<OpenAPIPreview>('/import/openapi/preview', { spec });
  return data;
};

/** Fetches OpenAPI spec from a public URL (http/https). Returns the spec body as string. */
export const fetchOpenAPISpecFromUrl = async (url: string): Promise<{ spec: string }> => {
  const { data } = await api.post<{ spec: string }>('/import/openapi/fetch-url', { url: url.trim() });
  return data;
};

export const importOpenAPI = async (
  spec: string,
  options?: {
    server_name?: string;
    description?: string;
    base_url?: string;
  }
): Promise<OpenAPIImportResult> => {
  const { data } = await api.post<OpenAPIImportResult>('/import/openapi', {
    spec,
    ...options,
  });
  return data;
};

// Flow APIs
export interface Flow {
  id: string;
  server_id: string;
  name: string;
  description: string;
  nodes: FlowNode[];
  edges: FlowEdge[];
  created_at: string;
  updated_at: string;
}

export interface FlowNode {
  id: string;
  type: string;
  position: { x: number; y: number };
  data: Record<string, unknown>;
}

export interface FlowEdge {
  id: string;
  source: string;
  target: string;
  sourceHandle?: string;
  targetHandle?: string;
}

export interface NodeResult {
  node_id: string;
  node_type: string;
  success: boolean;
  output?: Record<string, unknown>;
  error?: string;
  duration_ms: number;
}

export interface FlowExecutionResult {
  success: boolean;
  output?: Record<string, unknown>;
  error?: string;
  duration_ms: number;
  node_results: NodeResult[];
}

export const listFlows = async (): Promise<Flow[]> => {
  const { data } = await api.get<Flow[]>('/flows');
  return data || [];
};

export const getFlow = async (id: string): Promise<Flow> => {
  const { data } = await api.get<Flow>(`/flows/${id}`);
  return data;
};

export const createFlow = async (flow: {
  server_id: string;
  name: string;
  description?: string;
  nodes: FlowNode[];
  edges: FlowEdge[];
}): Promise<Flow> => {
  const { data } = await api.post<Flow>('/flows', flow);
  return data;
};

export const updateFlow = async (id: string, flow: {
  name?: string;
  description?: string;
  nodes?: FlowNode[];
  edges?: FlowEdge[];
}): Promise<Flow> => {
  const { data } = await api.put<Flow>(`/flows/${id}`, flow);
  return data;
};

export const deleteFlow = async (id: string): Promise<void> => {
  await api.delete(`/flows/${id}`);
};

export const getServerFlows = async (serverId: string): Promise<Flow[]> => {
  const { data } = await api.get<Flow[]>(`/servers/${serverId}/flows`);
  return data || [];
};

export const getSecurityScore = async (serverId: string): Promise<SecurityScoreResult> => {
  const { data } = await api.get<SecurityScoreResult>(`/servers/${serverId}/security-score`);
  return data;
};

export const getServerObservability = async (serverId: string): Promise<ObservabilitySummaryResponse> => {
  const { data } = await api.get<ObservabilitySummaryResponse>(`/servers/${serverId}/observability`);
  return data;
};

export const enableServerObservability = async (
  serverId: string
): Promise<{ key: string; endpoint_url: string; env: { MCP_OBSERVABILITY_ENDPOINT: string; MCP_OBSERVABILITY_KEY: string } }> => {
  const { data } = await api.post<{ key: string; endpoint_url: string; env: { MCP_OBSERVABILITY_ENDPOINT: string; MCP_OBSERVABILITY_KEY: string } }>(
    `/servers/${serverId}/observability/enable`
  );
  return data;
};

export const getObservabilityDashboard = async (params?: {
  server_id?: string;
  tool_name?: string;
  client_user_id?: string;
  client_agent?: string;
  limit?: number;
}): Promise<ObservabilityDashboardResponse> => {
  const searchParams = new URLSearchParams();
  if (params?.server_id) searchParams.set('server_id', params.server_id);
  if (params?.tool_name) searchParams.set('tool_name', params.tool_name);
  if (params?.client_user_id) searchParams.set('client_user_id', params.client_user_id);
  if (params?.client_agent) searchParams.set('client_agent', params.client_agent);
  if (params?.limit) searchParams.set('limit', String(params.limit));
  const q = searchParams.toString();
  const { data } = await api.get<ObservabilityDashboardResponse>(`/observability${q ? `?${q}` : ''}`);
  return data;
};

export const executeFlow = async (id: string, input?: Record<string, unknown>, context?: Record<string, unknown>): Promise<FlowExecutionResult> => {
  const { data } = await api.post<FlowExecutionResult>(`/flows/${id}/execute`, { input, context });
  return data;
};

export const convertFlowToTool = async (id: string, toolName: string, description?: string): Promise<{ tool: Tool; message: string }> => {
  const { data } = await api.post<{ tool: Tool; message: string }>(`/flows/${id}/convert`, {
    tool_name: toolName,
    description,
  });
  return data;
};

// Server Version APIs
export const publishServer = async (serverId: string, request: PublishRequest): Promise<ServerVersion> => {
  const { data } = await api.post<ServerVersion>(`/servers/${serverId}/publish`, request);
  return data;
};

/** Hides the server from the public marketplace; snapshots and version history remain. */
export const unlistServerFromMarketplace = async (serverId: string): Promise<Server> => {
  const { data } = await api.post<Server>(`/servers/${serverId}/unlist-marketplace`);
  return data;
};

export const getServerVersions = async (serverId: string): Promise<ServerVersion[]> => {
  const { data } = await api.get<ServerVersion[]>(`/servers/${serverId}/versions`);
  return data || [];
};

export const getServerVersionSnapshot = async (serverId: string, version: string): Promise<{ version: ServerVersion; server: Server }> => {
  const { data } = await api.get<{ version: ServerVersion; server: Server }>(`/servers/${serverId}/versions/${version}`);
  return data;
};

export interface HostedPublishResponse {
  base_url: string;
  user_id: string;
  server_slug: string;
  version: string;
  endpoint: string;
  mcp_config: string;
}

export type HostedAuthMode = 'bearer_token' | 'no_auth' | 'oidc' | 'mtls';

export interface HostedStatusResponse {
  running: boolean;
  user_id?: string;
  server_id?: string;
  server_slug?: string;
  version?: string;
  snapshot_id?: string;
  snapshot_version?: string;
  started_at?: string;
  last_ensured_at?: string;
  endpoint?: string;
  mcp_config?: string;
  manifest?: Record<string, unknown>;
  container_id?: string;
  host_port?: string;
  runtime?: string;
  image?: string;
  memory_mb?: number;
  nano_cpus?: number;
  pids_limit?: number;
  idle_timeout_minutes?: number;
  network_scope?: string;
  hosted_auth_mode?: HostedAuthMode;
  require_caller_identity?: boolean;
}

export interface TryProviderInfo {
  name: string;
  type: string;
  model: string;
  enabled: boolean;
}

export interface TryConfigResponse {
  default_provider: string;
  providers: TryProviderInfo[];
}

export interface TryChatMessage {
  role: 'system' | 'user' | 'assistant';
  content: string;
}

export interface TryChatRequest {
  provider?: string;
  model?: string;
  messages: TryChatMessage[];
  target?: {
    type?: string;
    id?: string;
    name?: string;
  };
}

export interface TryChatResponse {
  provider: string;
  model: string;
  message: string;
  endpoint?: string;
  tool_calls?: Array<{
    name: string;
    arguments?: string;
    success: boolean;
    duration_ms: number;
    result?: unknown;
    error?: string;
  }>;
}

export interface HostedSession {
  id: string;
  user_id: string;
  server_id: string;
  server_name?: string;
  snapshot_version?: string;
  container_id?: string;
  host_port?: string;
  status: string;
  health: string;
  last_used_at?: string;
  last_ensured_at?: string;
  started_at?: string;
  stopped_at?: string;
  last_error?: string;
  created_at: string;
  updated_at: string;
}

export interface HostedCatalogItem {
  server_id: string;
  server_name: string;
  server_slug: string;
  publisher_user_id: string;
  snapshot_version?: string;
  endpoint: string;
  mcp_config: string;
  hosted_auth_mode?: HostedAuthMode;
  require_caller_identity?: boolean;
  last_ensured_at?: string;
}

export interface HostedCallerAPIKey {
  id: string;
  owner_user_id: string;
  key_id: string;
  caller_user_id: string;
  tenant_id?: string;
  scopes?: string[];
  allow_alias: boolean;
  expires_at?: string;
  revoked_at?: string;
  created_by?: string;
  created_at: string;
}

export interface HostedCallerAPIKeyCreateRequest {
  caller_user_id: string;
  tenant_id?: string;
  scopes?: string[];
  allow_alias?: boolean;
  expires_at?: string;
}

export interface HostedCallerAPIKeyCreateResponse {
  key: HostedCallerAPIKey;
  api_key: string;
}

export interface HostedSecurityResponse {
  hosted_auth_mode?: HostedAuthMode;
  require_caller_identity?: boolean;
  hosted_security_config?: Record<string, unknown>;
  hosted_runtime_config?: Record<string, unknown>;
  has_hosted_access_key?: boolean;
  env_header?: string;
  client_cert_header?: string;
}

export type HostedIsolationTier = 'standard' | 'restricted' | 'strict';
export type HostedEgressPolicy = 'allow_all' | 'deny_default';

export interface HostedSecurityAuditEvent {
  id: string;
  server_id: string;
  actor_user_id?: string;
  action: string;
  resource_type?: string;
  resource_id?: string;
  metadata?: Record<string, unknown>;
  created_at: string;
}

export const hostedPublish = async (
  serverId: string,
  version?: string,
  envProfile?: EnvProfileKey,
  idleTimeoutMinutes?: number,
  hostedAuthMode?: HostedAuthMode,
  requireCallerIdentity?: boolean,
  hostedSecurityConfig?: Record<string, unknown>,
  hostedRuntimeConfig?: Record<string, unknown>
): Promise<HostedPublishResponse> => {
  const { data } = await api.post<HostedPublishResponse>(`/servers/${serverId}/hosted-publish`, {
    version: version || '',
    env_profile: envProfile || undefined,
    idle_timeout_minutes: idleTimeoutMinutes,
    hosted_auth_mode: hostedAuthMode,
    require_caller_identity: requireCallerIdentity,
    hosted_security_config: hostedSecurityConfig,
    hosted_runtime_config: hostedRuntimeConfig,
  });
  return data;
};

export const getHostedSecurity = async (serverId: string): Promise<HostedSecurityResponse> => {
  const { data } = await api.get<HostedSecurityResponse>(`/servers/${serverId}/hosted-security`);
  return data;
};

export const putHostedSecurity = async (
  serverId: string,
  body: {
    hosted_auth_mode?: HostedAuthMode;
    require_caller_identity?: boolean;
    hosted_security_config?: Record<string, unknown>;
  }
): Promise<void> => {
  await api.put(`/servers/${serverId}/hosted-security`, body);
};

export const rotateHostedAccessKey = async (serverId: string): Promise<{ hosted_access_key: string; warning?: string }> => {
  const { data } = await api.post<{ hosted_access_key: string; warning?: string }>(
    `/servers/${serverId}/hosted-security/rotate-access-key`,
    {}
  );
  return data;
};

export const listHostedSecurityAudit = async (serverId: string, limit?: number): Promise<HostedSecurityAuditEvent[]> => {
  const { data } = await api.get<{ events: HostedSecurityAuditEvent[] }>(
    `/servers/${serverId}/hosted-security/audit`,
    { params: { limit: limit ?? 100 } }
  );
  return data.events || [];
};

export const downloadHostedSecurityAuditExport = async (serverId: string): Promise<Blob> => {
  const { data } = await api.get<Blob>(`/servers/${serverId}/hosted-security/audit/export`, {
    responseType: 'blob',
    params: { limit: 500 },
  });
  return data;
};

export const hostedStatus = async (serverId: string): Promise<HostedStatusResponse> => {
  const { data } = await api.get<HostedStatusResponse>(`/servers/${serverId}/hosted-status`);
  return data;
};

export const listHostedSessions = async (): Promise<HostedSession[]> => {
  const { data } = await api.get<{ sessions: HostedSession[] }>('/hosted/sessions');
  return data.sessions || [];
};

export const listHostedCatalog = async (): Promise<HostedCatalogItem[]> => {
  const { data } = await api.get<{ items: HostedCatalogItem[] }>('/hosted/catalog');
  return data.items || [];
};

export const listHostedCallerAPIKeys = async (): Promise<HostedCallerAPIKey[]> => {
  const { data } = await api.get<{ keys: HostedCallerAPIKey[] }>('/hosted/caller-keys');
  return data.keys || [];
};

export const createHostedCallerAPIKey = async (
  request: HostedCallerAPIKeyCreateRequest
): Promise<HostedCallerAPIKeyCreateResponse> => {
  const { data } = await api.post<HostedCallerAPIKeyCreateResponse>('/hosted/caller-keys', request);
  return data;
};

export const revokeHostedCallerAPIKey = async (keyId: string): Promise<void> => {
  await api.post(`/hosted/caller-keys/${encodeURIComponent(keyId)}/revoke`);
};

export const checkHostedSessionHealth = async (serverId: string): Promise<HostedSession> => {
  const { data } = await api.get<HostedSession>(`/hosted/sessions/${serverId}/health`);
  return data;
};

export const restartHostedSession = async (serverId: string): Promise<HostedSession> => {
  const { data } = await api.post<HostedSession>(`/hosted/sessions/${serverId}/restart`);
  return data;
};

export const stopHostedSession = async (serverId: string): Promise<HostedSession> => {
  const { data } = await api.post<HostedSession>(`/hosted/sessions/${serverId}/stop`);
  return data;
};

export const marketplaceHostedDeploy = async (
  serverId: string,
  envProfile?: EnvProfileKey,
  idleTimeoutMinutes?: number,
  hostedAuthMode?: HostedAuthMode,
  requireCallerIdentity?: boolean
): Promise<HostedStatusResponse> => {
  const { data } = await api.post<HostedStatusResponse>(`/marketplace/${serverId}/hosted-deploy`, {
    env_profile: envProfile || undefined,
    idle_timeout_minutes: idleTimeoutMinutes,
    hosted_auth_mode: hostedAuthMode,
    require_caller_identity: requireCallerIdentity,
  });
  return data;
};

export const marketplaceHostedStatus = async (serverId: string): Promise<HostedStatusResponse> => {
  const { data } = await api.get<HostedStatusResponse>(`/marketplace/${serverId}/hosted-status`);
  return data;
};

export const compositionHostedDeploy = async (
  compositionId: string,
  envProfile?: EnvProfileKey,
  idleTimeoutMinutes?: number,
  hostedAuthMode?: HostedAuthMode,
  requireCallerIdentity?: boolean
): Promise<HostedStatusResponse> => {
  const { data } = await api.post<HostedStatusResponse>(`/compositions/${compositionId}/hosted-deploy`, {
    env_profile: envProfile || undefined,
    idle_timeout_minutes: idleTimeoutMinutes,
    hosted_auth_mode: hostedAuthMode,
    require_caller_identity: requireCallerIdentity,
  });
  return data;
};

export const compositionHostedStatus = async (compositionId: string): Promise<HostedStatusResponse> => {
  const { data } = await api.get<HostedStatusResponse>(`/compositions/${compositionId}/hosted-status`);
  return data;
};

export const getTryConfig = async (): Promise<TryConfigResponse> => {
  const { data } = await api.get<TryConfigResponse>('/try/config');
  return data;
};

export const tryChat = async (request: TryChatRequest): Promise<TryChatResponse> => {
  const { data } = await api.post<TryChatResponse>('/try/chat', request);
  return data;
};

export const downloadServerVersion = async (serverId: string, version: string): Promise<Blob> => {
  const { data } = await api.get<Blob>(`/servers/${serverId}/versions/${version}/download`, {
    responseType: 'blob',
  });
  return data;
};

// Marketplace APIs
export const listMarketplace = async (): Promise<Server[]> => {
  const { data } = await api.get<Server[]>('/marketplace');
  return data || [];
};

export const getMarketplaceServer = async (id: string): Promise<{
  server: Server;
  versions: ServerVersion[];
  security_score?: SecurityScoreResult;
}> => {
  const { data } = await api.get<{ server: Server; versions: ServerVersion[]; security_score?: SecurityScoreResult }>(`/marketplace/${id}`);
  return data;
};

export const downloadMarketplaceServer = async (id: string, envProfile?: EnvProfileKey): Promise<Blob> => {
  const { data } = await api.get<Blob>(`/marketplace/${id}/download`, {
    params: envProfile ? { env_profile: envProfile } : undefined,
    responseType: 'blob',
  });
  return data;
};

export default api;
