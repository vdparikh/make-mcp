import { useState } from 'react';
import { toast } from 'react-toastify';
import Editor from '@monaco-editor/react';
import type { Tool, ExecutionType } from '../types';
import { createTool, updateTool } from '../services/api';

interface Props {
  serverId: string;
  tools: Tool[];
  onToolCreated: () => void;
  onToolDeleted: (id: string) => void;
}

type AuthType = 'none' | 'api_key' | 'bearer_token' | 'basic_auth' | 'oauth2';

interface AuthConfig {
  type: AuthType;
  apiKey?: {
    headerName: string;
    prefix: string;
    value: string;
  };
  bearerToken?: {
    token: string;
  };
  basicAuth?: {
    username: string;
    password: string;
  };
  oauth2?: {
    tokenUrl: string;
    clientId: string;
    clientSecret: string;
    scope: string;
  };
}

const executionTypes: { value: ExecutionType; label: string; icon: string }[] = [
  { value: 'rest_api', label: 'REST API', icon: 'bi-globe' },
  { value: 'graphql', label: 'GraphQL', icon: 'bi-diagram-3' },
  { value: 'webhook', label: 'Webhook', icon: 'bi-link-45deg' },
  { value: 'javascript', label: 'JavaScript', icon: 'bi-filetype-js' },
  { value: 'python', label: 'Python', icon: 'bi-filetype-py' },
  { value: 'database', label: 'Database', icon: 'bi-database' },
];

const authTypes: { value: AuthType; label: string; icon: string; description: string }[] = [
  { value: 'none', label: 'No Authentication', icon: 'bi-unlock', description: 'Public API, no auth required' },
  { value: 'api_key', label: 'API Key', icon: 'bi-key', description: 'API key in header or query param' },
  { value: 'bearer_token', label: 'Bearer Token', icon: 'bi-shield-lock', description: 'JWT or OAuth access token' },
  { value: 'basic_auth', label: 'Basic Auth', icon: 'bi-person-lock', description: 'Username and password' },
  { value: 'oauth2', label: 'OAuth 2.0', icon: 'bi-shield-check', description: 'Client credentials flow' },
];

export default function ToolEditor({ serverId, tools, onToolCreated, onToolDeleted }: Props) {
  const [showForm, setShowForm] = useState(false);
  const [editingTool, setEditingTool] = useState<Tool | null>(null);
  const [activeTab, setActiveTab] = useState<'basic' | 'auth' | 'schema' | 'config'>('basic');
  
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [executionType, setExecutionType] = useState<ExecutionType>('rest_api');
  const [inputSchema, setInputSchema] = useState('{\n  "type": "object",\n  "properties": {\n    \n  }\n}');
  const [outputSchema, setOutputSchema] = useState('{\n  "type": "object",\n  "properties": {\n    \n  }\n}');
  const [executionConfig, setExecutionConfig] = useState('{\n  "url": "",\n  "method": "GET",\n  "headers": {}\n}');
  const [contextFields, setContextFields] = useState('');
  const [saving, setSaving] = useState(false);

  // Auth state
  const [authType, setAuthType] = useState<AuthType>('none');
  const [apiKeyHeader, setApiKeyHeader] = useState('X-API-Key');
  const [apiKeyPrefix, setApiKeyPrefix] = useState('');
  const [apiKeyValue, setApiKeyValue] = useState('');
  const [bearerToken, setBearerToken] = useState('');
  const [basicUsername, setBasicUsername] = useState('');
  const [basicPassword, setBasicPassword] = useState('');
  const [oauth2TokenUrl, setOauth2TokenUrl] = useState('');
  const [oauth2ClientId, setOauth2ClientId] = useState('');
  const [oauth2ClientSecret, setOauth2ClientSecret] = useState('');
  const [oauth2Scope, setOauth2Scope] = useState('');

  const resetForm = () => {
    setName('');
    setDescription('');
    setExecutionType('rest_api');
    setInputSchema('{\n  "type": "object",\n  "properties": {\n    \n  }\n}');
    setOutputSchema('{\n  "type": "object",\n  "properties": {\n    \n  }\n}');
    setExecutionConfig('{\n  "url": "",\n  "method": "GET",\n  "headers": {}\n}');
    setContextFields('');
    setEditingTool(null);
    setActiveTab('basic');
    // Reset auth
    setAuthType('none');
    setApiKeyHeader('X-API-Key');
    setApiKeyPrefix('');
    setApiKeyValue('');
    setBearerToken('');
    setBasicUsername('');
    setBasicPassword('');
    setOauth2TokenUrl('');
    setOauth2ClientId('');
    setOauth2ClientSecret('');
    setOauth2Scope('');
  };

  const extractAuthFromConfig = (config: Record<string, unknown>) => {
    const headers = (config.headers || {}) as Record<string, string>;
    const authConfig = config.auth as AuthConfig | undefined;
    
    if (authConfig) {
      setAuthType(authConfig.type);
      if (authConfig.apiKey) {
        setApiKeyHeader(authConfig.apiKey.headerName || 'X-API-Key');
        setApiKeyPrefix(authConfig.apiKey.prefix || '');
        setApiKeyValue(authConfig.apiKey.value || '');
      }
      if (authConfig.bearerToken) {
        setBearerToken(authConfig.bearerToken.token || '');
      }
      if (authConfig.basicAuth) {
        setBasicUsername(authConfig.basicAuth.username || '');
        setBasicPassword(authConfig.basicAuth.password || '');
      }
      if (authConfig.oauth2) {
        setOauth2TokenUrl(authConfig.oauth2.tokenUrl || '');
        setOauth2ClientId(authConfig.oauth2.clientId || '');
        setOauth2ClientSecret(authConfig.oauth2.clientSecret || '');
        setOauth2Scope(authConfig.oauth2.scope || '');
      }
    } else if (headers.Authorization) {
      const authHeader = headers.Authorization;
      if (authHeader.startsWith('Bearer ')) {
        setAuthType('bearer_token');
        setBearerToken(authHeader.replace('Bearer ', ''));
      } else if (authHeader.startsWith('Basic ')) {
        setAuthType('basic_auth');
      }
    } else {
      // Check for common API key headers
      for (const [key, value] of Object.entries(headers)) {
        if (key.toLowerCase().includes('api') || key.toLowerCase().includes('key')) {
          setAuthType('api_key');
          setApiKeyHeader(key);
          setApiKeyValue(value);
          break;
        }
      }
    }
  };

  const handleEdit = (tool: Tool) => {
    setEditingTool(tool);
    setName(tool.name);
    setDescription(tool.description);
    setExecutionType(tool.execution_type);
    setInputSchema(JSON.stringify(tool.input_schema || {}, null, 2));
    setOutputSchema(JSON.stringify(tool.output_schema || {}, null, 2));
    setExecutionConfig(JSON.stringify(tool.execution_config || {}, null, 2));
    setContextFields(tool.context_fields?.join(', ') || '');
    extractAuthFromConfig(tool.execution_config || {});
    setShowForm(true);
    setActiveTab('basic');
  };

  const buildAuthHeaders = (): Record<string, string> => {
    const headers: Record<string, string> = {};
    
    switch (authType) {
      case 'api_key':
        if (apiKeyValue) {
          headers[apiKeyHeader] = apiKeyPrefix ? `${apiKeyPrefix} ${apiKeyValue}` : apiKeyValue;
        }
        break;
      case 'bearer_token':
        if (bearerToken) {
          headers['Authorization'] = `Bearer ${bearerToken}`;
        }
        break;
      case 'basic_auth':
        if (basicUsername && basicPassword) {
          const encoded = btoa(`${basicUsername}:${basicPassword}`);
          headers['Authorization'] = `Basic ${encoded}`;
        }
        break;
      case 'oauth2':
        // OAuth2 requires runtime token fetch - store config
        break;
    }
    
    return headers;
  };

  const buildAuthConfig = (): AuthConfig | null => {
    if (authType === 'none') return null;
    
    const config: AuthConfig = { type: authType };
    
    switch (authType) {
      case 'api_key':
        config.apiKey = {
          headerName: apiKeyHeader,
          prefix: apiKeyPrefix,
          value: apiKeyValue,
        };
        break;
      case 'bearer_token':
        config.bearerToken = { token: bearerToken };
        break;
      case 'basic_auth':
        config.basicAuth = {
          username: basicUsername,
          password: basicPassword,
        };
        break;
      case 'oauth2':
        config.oauth2 = {
          tokenUrl: oauth2TokenUrl,
          clientId: oauth2ClientId,
          clientSecret: oauth2ClientSecret,
          scope: oauth2Scope,
        };
        break;
    }
    
    return config;
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    try {
      setSaving(true);
      
      let parsedInputSchema, parsedOutputSchema, parsedExecutionConfig;
      
      try {
        parsedInputSchema = JSON.parse(inputSchema);
      } catch {
        toast.error('Invalid Input Schema JSON');
        return;
      }
      
      try {
        parsedOutputSchema = JSON.parse(outputSchema);
      } catch {
        toast.error('Invalid Output Schema JSON');
        return;
      }
      
      try {
        parsedExecutionConfig = JSON.parse(executionConfig);
      } catch {
        toast.error('Invalid Execution Config JSON');
        return;
      }

      // Merge auth headers into execution config
      const authHeaders = buildAuthHeaders();
      const authConfig = buildAuthConfig();
      
      parsedExecutionConfig.headers = {
        ...parsedExecutionConfig.headers,
        ...authHeaders,
      };
      
      if (authConfig) {
        parsedExecutionConfig.auth = authConfig;
      }

      const toolData = {
        server_id: serverId,
        name,
        description,
        execution_type: executionType,
        input_schema: parsedInputSchema,
        output_schema: parsedOutputSchema,
        execution_config: parsedExecutionConfig,
        context_fields: contextFields.split(',').map(f => f.trim()).filter(Boolean),
      };

      if (editingTool) {
        await updateTool(editingTool.id, toolData);
        toast.success('Tool updated');
      } else {
        await createTool(toolData);
        toast.success('Tool created');
      }

      setShowForm(false);
      resetForm();
      onToolCreated();
    } catch (error) {
      toast.error('Failed to save tool');
    } finally {
      setSaving(false);
    }
  };

  const getExecutionConfigTemplate = (type: ExecutionType) => {
    switch (type) {
      case 'rest_api':
        return '{\n  "url": "https://api.example.com/endpoint",\n  "method": "GET",\n  "headers": {}\n}';
      case 'graphql':
        return '{\n  "url": "https://api.example.com/graphql",\n  "query": "query { ... }",\n  "headers": {}\n}';
      case 'webhook':
        return '{\n  "url": "https://example.com/webhook",\n  "headers": {}\n}';
      case 'database':
        return '{\n  "connection_string": "",\n  "query": "SELECT * FROM table WHERE id = {{id}}"\n}';
      default:
        return '{}';
    }
  };

  const renderAuthConfig = () => (
    <div>
      <div style={{ marginBottom: '1.5rem' }}>
        <label className="form-label">Authentication Type</label>
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(180px, 1fr))', gap: '0.75rem' }}>
          {authTypes.map((type) => (
            <div
              key={type.value}
              onClick={() => setAuthType(type.value)}
              style={{
                padding: '1rem',
                background: authType === type.value ? 'rgba(129, 140, 248, 0.15)' : 'var(--dark-bg)',
                border: `2px solid ${authType === type.value ? 'var(--primary-color)' : 'var(--card-border)'}`,
                borderRadius: '8px',
                cursor: 'pointer',
                transition: 'all 0.2s',
              }}
            >
              <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', marginBottom: '0.25rem' }}>
                <i className={`bi ${type.icon}`} style={{ color: authType === type.value ? 'var(--primary-color)' : 'var(--text-secondary)' }}></i>
                <span style={{ fontWeight: 500, color: 'var(--text-primary)' }}>{type.label}</span>
              </div>
              <p style={{ fontSize: '0.75rem', color: 'var(--text-secondary)', margin: 0 }}>
                {type.description}
              </p>
            </div>
          ))}
        </div>
      </div>

      {authType === 'api_key' && (
        <div style={{ 
          background: 'var(--dark-bg)', 
          borderRadius: '8px', 
          padding: '1.25rem',
          border: '1px solid var(--card-border)'
        }}>
          <h4 style={{ fontSize: '0.9375rem', marginBottom: '1rem', color: 'var(--text-primary)' }}>
            <i className="bi bi-key" style={{ marginRight: '0.5rem', color: 'var(--warning-color)' }}></i>
            API Key Configuration
          </h4>
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '1rem' }}>
            <div className="form-group">
              <label className="form-label">Header Name</label>
              <input
                type="text"
                className="form-control"
                value={apiKeyHeader}
                onChange={(e) => setApiKeyHeader(e.target.value)}
                placeholder="X-API-Key"
              />
            </div>
            <div className="form-group">
              <label className="form-label">Prefix (optional)</label>
              <input
                type="text"
                className="form-control"
                value={apiKeyPrefix}
                onChange={(e) => setApiKeyPrefix(e.target.value)}
                placeholder="e.g., Api-Key"
              />
            </div>
          </div>
          <div className="form-group" style={{ marginBottom: 0 }}>
            <label className="form-label">API Key Value</label>
            <input
              type="password"
              className="form-control"
              value={apiKeyValue}
              onChange={(e) => setApiKeyValue(e.target.value)}
              placeholder="Enter your API key"
            />
            <small style={{ color: 'var(--text-muted)', marginTop: '0.25rem', display: 'block' }}>
              Use {'{{ENV_VAR}}'} for environment variables
            </small>
          </div>
        </div>
      )}

      {authType === 'bearer_token' && (
        <div style={{ 
          background: 'var(--dark-bg)', 
          borderRadius: '8px', 
          padding: '1.25rem',
          border: '1px solid var(--card-border)'
        }}>
          <h4 style={{ fontSize: '0.9375rem', marginBottom: '1rem', color: 'var(--text-primary)' }}>
            <i className="bi bi-shield-lock" style={{ marginRight: '0.5rem', color: 'var(--success-color)' }}></i>
            Bearer Token Configuration
          </h4>
          <div className="form-group" style={{ marginBottom: 0 }}>
            <label className="form-label">Access Token</label>
            <input
              type="password"
              className="form-control"
              value={bearerToken}
              onChange={(e) => setBearerToken(e.target.value)}
              placeholder="Enter your bearer token"
            />
            <small style={{ color: 'var(--text-muted)', marginTop: '0.25rem', display: 'block' }}>
              Token will be sent as: Authorization: Bearer {'<token>'}
            </small>
          </div>
        </div>
      )}

      {authType === 'basic_auth' && (
        <div style={{ 
          background: 'var(--dark-bg)', 
          borderRadius: '8px', 
          padding: '1.25rem',
          border: '1px solid var(--card-border)'
        }}>
          <h4 style={{ fontSize: '0.9375rem', marginBottom: '1rem', color: 'var(--text-primary)' }}>
            <i className="bi bi-person-lock" style={{ marginRight: '0.5rem', color: 'var(--secondary-color)' }}></i>
            Basic Authentication
          </h4>
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '1rem' }}>
            <div className="form-group" style={{ marginBottom: 0 }}>
              <label className="form-label">Username</label>
              <input
                type="text"
                className="form-control"
                value={basicUsername}
                onChange={(e) => setBasicUsername(e.target.value)}
                placeholder="Username"
              />
            </div>
            <div className="form-group" style={{ marginBottom: 0 }}>
              <label className="form-label">Password</label>
              <input
                type="password"
                className="form-control"
                value={basicPassword}
                onChange={(e) => setBasicPassword(e.target.value)}
                placeholder="Password"
              />
            </div>
          </div>
        </div>
      )}

      {authType === 'oauth2' && (
        <div style={{ 
          background: 'var(--dark-bg)', 
          borderRadius: '8px', 
          padding: '1.25rem',
          border: '1px solid var(--card-border)'
        }}>
          <h4 style={{ fontSize: '0.9375rem', marginBottom: '1rem', color: 'var(--text-primary)' }}>
            <i className="bi bi-shield-check" style={{ marginRight: '0.5rem', color: 'var(--primary-color)' }}></i>
            OAuth 2.0 Client Credentials
          </h4>
          <div className="form-group">
            <label className="form-label">Token URL</label>
            <input
              type="text"
              className="form-control"
              value={oauth2TokenUrl}
              onChange={(e) => setOauth2TokenUrl(e.target.value)}
              placeholder="https://auth.example.com/oauth/token"
            />
          </div>
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '1rem' }}>
            <div className="form-group">
              <label className="form-label">Client ID</label>
              <input
                type="text"
                className="form-control"
                value={oauth2ClientId}
                onChange={(e) => setOauth2ClientId(e.target.value)}
                placeholder="Client ID"
              />
            </div>
            <div className="form-group">
              <label className="form-label">Client Secret</label>
              <input
                type="password"
                className="form-control"
                value={oauth2ClientSecret}
                onChange={(e) => setOauth2ClientSecret(e.target.value)}
                placeholder="Client Secret"
              />
            </div>
          </div>
          <div className="form-group" style={{ marginBottom: 0 }}>
            <label className="form-label">Scope (optional)</label>
            <input
              type="text"
              className="form-control"
              value={oauth2Scope}
              onChange={(e) => setOauth2Scope(e.target.value)}
              placeholder="e.g., read write"
            />
          </div>
        </div>
      )}

      {authType === 'none' && (
        <div style={{ 
          background: 'var(--dark-bg)', 
          borderRadius: '8px', 
          padding: '2rem',
          border: '1px solid var(--card-border)',
          textAlign: 'center'
        }}>
          <i className="bi bi-unlock" style={{ fontSize: '2rem', color: 'var(--text-muted)', marginBottom: '0.75rem', display: 'block' }}></i>
          <p style={{ color: 'var(--text-secondary)', margin: 0 }}>
            No authentication configured. The API will be called without auth headers.
          </p>
        </div>
      )}
    </div>
  );

  if (showForm) {
    const tabs = [
      { id: 'basic', label: 'Basic Info', icon: 'bi-info-circle' },
      { id: 'auth', label: 'Authentication', icon: 'bi-shield-lock' },
      { id: 'schema', label: 'Schema', icon: 'bi-braces' },
      { id: 'config', label: 'Execution', icon: 'bi-gear' },
    ] as const;

    return (
      <div className="card">
        <div className="card-header">
          <h3 className="card-title">
            {editingTool ? 'Edit Tool' : 'Create New Tool'}
          </h3>
          <button 
            className="btn btn-icon btn-secondary"
            onClick={() => { setShowForm(false); resetForm(); }}
          >
            <i className="bi bi-x-lg"></i>
          </button>
        </div>

        <div className="tabs" style={{ marginBottom: '1.5rem' }}>
          {tabs.map((tab) => (
            <button
              key={tab.id}
              type="button"
              className={`tab ${activeTab === tab.id ? 'active' : ''}`}
              onClick={() => setActiveTab(tab.id)}
            >
              <i className={`bi ${tab.icon}`} style={{ marginRight: '0.5rem' }}></i>
              {tab.label}
            </button>
          ))}
        </div>

        <form onSubmit={handleSubmit}>
          {activeTab === 'basic' && (
            <div>
              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '1rem' }}>
                <div className="form-group">
                  <label className="form-label">Tool Name *</label>
                  <input
                    type="text"
                    className="form-control"
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                    placeholder="e.g., get_weather"
                    required
                  />
                </div>

                <div className="form-group">
                  <label className="form-label">Execution Type *</label>
                  <select
                    className="form-control"
                    value={executionType}
                    onChange={(e) => {
                      setExecutionType(e.target.value as ExecutionType);
                      setExecutionConfig(getExecutionConfigTemplate(e.target.value as ExecutionType));
                    }}
                  >
                    {executionTypes.map((type) => (
                      <option key={type.value} value={type.value}>
                        {type.label}
                      </option>
                    ))}
                  </select>
                </div>
              </div>

              <div className="form-group">
                <label className="form-label">Description</label>
                <textarea
                  className="form-control"
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  placeholder="Describe what this tool does..."
                  rows={3}
                />
              </div>

              <div className="form-group">
                <label className="form-label">
                  Context Fields
                  <span style={{ fontWeight: 'normal', color: 'var(--text-muted)', marginLeft: '0.5rem' }}>
                    (comma-separated: user_id, organization_id, permissions, roles)
                  </span>
                </label>
                <input
                  type="text"
                  className="form-control"
                  value={contextFields}
                  onChange={(e) => setContextFields(e.target.value)}
                  placeholder="user_id, organization_id, permissions"
                />
              </div>
            </div>
          )}

          {activeTab === 'auth' && renderAuthConfig()}

          {activeTab === 'schema' && (
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '1rem' }}>
              <div className="form-group">
                <label className="form-label">Input Schema (JSON Schema)</label>
                <div className="editor-container">
                  <Editor
                    height="300px"
                    language="json"
                    theme="vs-dark"
                    value={inputSchema}
                    onChange={(value) => setInputSchema(value || '')}
                    options={{
                      minimap: { enabled: false },
                      fontSize: 13,
                      lineNumbers: 'off',
                      folding: false,
                    }}
                  />
                </div>
              </div>

              <div className="form-group">
                <label className="form-label">Output Schema (JSON Schema)</label>
                <div className="editor-container">
                  <Editor
                    height="300px"
                    language="json"
                    theme="vs-dark"
                    value={outputSchema}
                    onChange={(value) => setOutputSchema(value || '')}
                    options={{
                      minimap: { enabled: false },
                      fontSize: 13,
                      lineNumbers: 'off',
                      folding: false,
                    }}
                  />
                </div>
              </div>
            </div>
          )}

          {activeTab === 'config' && (
            <div className="form-group">
              <label className="form-label">
                Execution Configuration
                <span style={{ fontWeight: 'normal', color: 'var(--text-muted)', marginLeft: '0.5rem' }}>
                  (Use {'{{field}}'} for input variables)
                </span>
              </label>
              <div className="editor-container">
                <Editor
                  height="300px"
                  language="json"
                  theme="vs-dark"
                  value={executionConfig}
                  onChange={(value) => setExecutionConfig(value || '')}
                  options={{
                    minimap: { enabled: false },
                    fontSize: 13,
                    lineNumbers: 'off',
                    folding: false,
                  }}
                />
              </div>
              <div style={{ 
                marginTop: '1rem', 
                padding: '1rem', 
                background: 'rgba(129, 140, 248, 0.1)', 
                borderRadius: '8px',
                border: '1px solid rgba(129, 140, 248, 0.2)'
              }}>
                <p style={{ fontSize: '0.8125rem', color: 'var(--text-secondary)', margin: 0 }}>
                  <i className="bi bi-info-circle" style={{ marginRight: '0.5rem', color: 'var(--primary-color)' }}></i>
                  Auth headers from the Authentication tab will be automatically merged into the headers object.
                </p>
              </div>
            </div>
          )}

          <div style={{ display: 'flex', gap: '0.75rem', marginTop: '1.5rem' }}>
            <button 
              type="button" 
              className="btn btn-secondary"
              onClick={() => { setShowForm(false); resetForm(); }}
            >
              Cancel
            </button>
            <button type="submit" className="btn btn-primary" disabled={saving}>
              {saving ? 'Saving...' : (editingTool ? 'Update Tool' : 'Create Tool')}
            </button>
          </div>
        </form>
      </div>
    );
  }

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '1rem' }}>
        <h3 style={{ margin: 0 }}>Tools ({tools.length})</h3>
        <button className="btn btn-primary" onClick={() => setShowForm(true)}>
          <i className="bi bi-plus-lg"></i>
          Add Tool
        </button>
      </div>

      {tools.length === 0 ? (
        <div className="empty-state">
          <i className="bi bi-tools"></i>
          <h3>No tools yet</h3>
          <p>Tools are functions that AI agents can call</p>
          <button className="btn btn-primary" onClick={() => setShowForm(true)}>
            <i className="bi bi-plus-lg"></i>
            Create First Tool
          </button>
        </div>
      ) : (
        <div>
          {tools.map((tool) => {
            const hasAuth = tool.execution_config && 
              ((tool.execution_config as Record<string, unknown>).auth || 
               Object.keys((tool.execution_config as Record<string, unknown>).headers || {}).some(
                 h => h.toLowerCase() === 'authorization' || h.toLowerCase().includes('api')
               ));
            
            return (
              <div key={tool.id} className="tool-card">
                <div className="tool-icon">
                  <i className={`bi ${executionTypes.find(t => t.value === tool.execution_type)?.icon || 'bi-gear'}`}></i>
                </div>
                <div className="tool-info">
                  <div className="tool-name">{tool.name}</div>
                  <div className="tool-description">
                    {tool.description || 'No description'}
                  </div>
                  <div style={{ display: 'flex', gap: '0.5rem', marginTop: '0.5rem', flexWrap: 'wrap' }}>
                    <span className="badge badge-primary">{tool.execution_type}</span>
                    {tool.context_fields && tool.context_fields.length > 0 && (
                      <span className="badge badge-success">
                        <i className="bi bi-person-badge" style={{ marginRight: '0.25rem' }}></i>
                        Context Aware
                      </span>
                    )}
                    {hasAuth && (
                      <span className="badge badge-warning">
                        <i className="bi bi-shield-lock" style={{ marginRight: '0.25rem' }}></i>
                        Auth
                      </span>
                    )}
                  </div>
                </div>
                <div style={{ display: 'flex', gap: '0.5rem' }}>
                  <button 
                    className="btn btn-icon btn-secondary btn-sm"
                    onClick={() => handleEdit(tool)}
                    title="Edit"
                  >
                    <i className="bi bi-pencil"></i>
                  </button>
                  <button 
                    className="btn btn-icon btn-secondary btn-sm"
                    onClick={() => onToolDeleted(tool.id)}
                    title="Delete"
                  >
                    <i className="bi bi-trash"></i>
                  </button>
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
