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

export interface HostedStatusResponse {
  running: boolean;
  user_id?: string;
  server_id?: string;
  server_slug?: string;
  version?: string;
  endpoint?: string;
  container_id?: string;
  host_port?: string;
}

export const hostedPublish = async (serverId: string, version?: string): Promise<HostedPublishResponse> => {
  const { data } = await api.post<HostedPublishResponse>(`/servers/${serverId}/hosted-publish`, { version: version || '' });
  return data;
};

export const hostedStatus = async (serverId: string): Promise<HostedStatusResponse> => {
  const { data } = await api.get<HostedStatusResponse>(`/servers/${serverId}/hosted-status`);
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

export const downloadMarketplaceServer = async (id: string): Promise<Blob> => {
  const { data } = await api.get<Blob>(`/marketplace/${id}/download`, {
    responseType: 'blob',
  });
  return data;
};

export default api;
