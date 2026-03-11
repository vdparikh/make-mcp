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
  PolicyEvaluationResult,
} from '../types';

const api = axios.create({
  baseURL: '/api',
  headers: {
    'Content-Type': 'application/json',
  },
});

// Server APIs
export const listServers = async (): Promise<Server[]> => {
  const { data } = await api.get<Server[]>('/servers');
  return data || [];
};

export const getServer = async (id: string): Promise<Server> => {
  const { data } = await api.get<Server>(`/servers/${id}`);
  return data;
};

export const createServer = async (server: Partial<Server>): Promise<Server> => {
  const { data } = await api.post<Server>('/servers', server);
  return data;
};

export const updateServer = async (id: string, server: Partial<Server>): Promise<Server> => {
  const { data } = await api.put<Server>(`/servers/${id}`, server);
  return data;
};

export const deleteServer = async (id: string): Promise<void> => {
  await api.delete(`/servers/${id}`);
};

export const generateServer = async (id: string): Promise<Blob> => {
  const { data } = await api.post(`/servers/${id}/generate`, {}, {
    responseType: 'blob',
  });
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

export const testTool = async (id: string, input: Record<string, unknown>, context?: Record<string, unknown>): Promise<TestToolResponse> => {
  const { data } = await api.post<TestToolResponse>(`/tools/${id}/test`, { input, context });
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

// Context Config APIs
export const getContextConfigs = async (serverId: string): Promise<ContextConfig[]> => {
  const { data } = await api.get<ContextConfig[]>(`/servers/${serverId}/context-configs`);
  return data || [];
};

export const createContextConfig = async (serverId: string, config: Partial<ContextConfig>): Promise<ContextConfig> => {
  const { data } = await api.post<ContextConfig>(`/servers/${serverId}/context-configs`, config);
  return data;
};

// Composition APIs
export const listCompositions = async (): Promise<ServerComposition[]> => {
  const { data } = await api.get<ServerComposition[]>('/compositions');
  return data || [];
};

export const createComposition = async (composition: Partial<ServerComposition>): Promise<ServerComposition> => {
  const { data } = await api.post<ServerComposition>('/compositions', composition);
  return data;
};

export default api;
