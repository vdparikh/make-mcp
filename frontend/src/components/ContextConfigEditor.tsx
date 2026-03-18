import { useState } from 'react';
import { toast } from 'react-toastify';
import Editor from '@monaco-editor/react';
import type { ContextConfig } from '../types';
import { createContextConfig } from '../services/api';

interface Props {
  serverId: string;
  configs: ContextConfig[];
  onConfigCreated: () => void;
  onConfigDeleted: (id: string) => void;
}

const sourceTypes = [
  { value: 'header', label: 'HTTP Header', icon: 'bi-arrow-down-up', description: 'Extract context from request headers' },
  { value: 'jwt', label: 'JWT Claims', icon: 'bi-key', description: 'Extract context from JWT token claims' },
  { value: 'query', label: 'Query Parameters', icon: 'bi-question-circle', description: 'Extract context from URL query params' },
  { value: 'custom', label: 'Custom', icon: 'bi-gear', description: 'Custom context extraction logic' },
];

const configTemplates: Record<string, string> = {
  header: '{\n  "header_name": "X-User-ID",\n  "target_field": "user_id"\n}',
  jwt: '{\n  "header_name": "Authorization",\n  "claims_map": {\n    "sub": "user_id",\n    "org": "organization_id",\n    "roles": "roles",\n    "permissions": "permissions"\n  }\n}',
  query: '{\n  "header_name": "user_id",\n  "target_field": "user_id"\n}',
  custom: '{\n  "type": "custom",\n  "config": {}\n}',
};

export default function ContextConfigEditor({ serverId, configs, onConfigCreated, onConfigDeleted }: Props) {
  const [showForm, setShowForm] = useState(false);
  const [name, setName] = useState('');
  const [sourceType, setSourceType] = useState('header');
  const [config, setConfig] = useState(configTemplates.header);
  const [saving, setSaving] = useState(false);

  const resetForm = () => {
    setName('');
    setSourceType('header');
    setConfig(configTemplates.header);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    try {
      setSaving(true);
      
      let parsedConfig;
      try {
        parsedConfig = JSON.parse(config);
      } catch {
        toast.error('Invalid Config JSON');
        return;
      }

      await createContextConfig(serverId, {
        name,
        source_type: sourceType as ContextConfig['source_type'],
        config: parsedConfig,
      });

      toast.success('Context config created');
      setShowForm(false);
      resetForm();
      onConfigCreated();
    } catch (error) {
      toast.error('Failed to create context config');
    } finally {
      setSaving(false);
    }
  };

  return (
    <div>
      <div className="card" style={{ marginBottom: '1.5rem' }}>
        <h3 className="card-title" style={{ marginBottom: '0.75rem' }}>
          <i className="bi bi-person-badge" style={{ marginRight: '0.75rem', color: 'var(--primary-color)' }}></i>
          Context-Aware Tool Execution
        </h3>
        <p style={{ color: 'var(--text-secondary)', marginBottom: '1rem' }}>
          Configure how context (user identity, permissions, organization data) is automatically injected into tool executions.
          This enables multi-tenant AI agents to safely operate with proper user context.
        </p>
        
        <div className="info-box" style={{ 
          borderRadius: '8px',
          padding: '1rem',
        }}>
          <h4 style={{ fontSize: '0.875rem', marginBottom: '0.5rem', color: 'var(--text-primary)' }}>
            <i className="bi bi-lightbulb" style={{ marginRight: '0.5rem', color: 'var(--warning-color)' }}></i>
            How It Works
          </h4>
          <ul style={{ color: '#e2e8f0', fontSize: '0.8125rem', margin: 0, paddingLeft: '1.25rem' }}>
            <li>AI asks: <code style={{ color: '#a5f3fc', background: 'rgb(0,0,0)', padding: '0.125rem 0.375rem', borderRadius: '3px' }}>"Show me my invoices"</code></li>
            <li>Context Engine extracts <code style={{ color: '#c4b5fd', background: 'rgba(0,0,0)', padding: '0.125rem 0.375rem', borderRadius: '3px' }}>user_id</code> from JWT/headers</li>
            <li>Tool automatically receives: <code style={{ color: '#86efac', background: 'rgba(0,0,0)', padding: '0.125rem 0.375rem', borderRadius: '3px' }}>customer_id = current_user</code></li>
            <li>No prompt engineering needed!</li>
          </ul>
          <div style={{ marginTop: '0.75rem', fontSize: '0.8rem' }}>
            <a
              href="https://github.com/vdparikh/make-mcp/blob/main/docs/creating-servers.md#context-engine"
              target="_blank"
              rel="noopener noreferrer"
              style={{ color: '#38bdf8', textDecoration: 'none' }}
            >
              <i className="bi bi-box-arrow-up-right" style={{ marginRight: '0.25rem' }}></i>
              Read full Context Engine guide
            </a>
          </div>
        </div>
      </div>

      {!showForm && (
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '1rem' }}>
          <h3 style={{ margin: 0 }}>Context Configurations ({configs.length})</h3>
          <button className="btn btn-primary" onClick={() => setShowForm(true)}>
            <i className="bi bi-plus-lg"></i>
            Add Configuration
          </button>
        </div>
      )}

      {showForm && (
        <div className="card">
          <div className="card-header">
            <h3 className="card-title">Add Context Configuration</h3>
            <button 
              className="btn btn-icon btn-secondary"
              onClick={() => { setShowForm(false); resetForm(); }}
            >
              <i className="bi bi-x-lg"></i>
            </button>
          </div>

          <form onSubmit={handleSubmit}>
            <div className="form-group">
              <label className="form-label">Configuration Name *</label>
              <input
                type="text"
                className="form-control"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="e.g., JWT User Context"
                required
              />
            </div>

            <div className="form-group">
              <label className="form-label">Source Type *</label>
              <div style={{ display: 'grid', gridTemplateColumns: 'repeat(2, 1fr)', gap: '0.75rem' }}>
                {sourceTypes.map((type) => (
                  <div
                    key={type.value}
                    onClick={() => {
                      setSourceType(type.value);
                      setConfig(configTemplates[type.value]);
                    }}
                    style={{
                      padding: '1rem',
                      background: sourceType === type.value ? 'rgba(99, 102, 241, 0.15)' : 'var(--dark-bg)',
                      border: `1px solid ${sourceType === type.value ? 'var(--primary-color)' : 'var(--card-border)'}`,
                      borderRadius: '8px',
                      cursor: 'pointer',
                      transition: 'all 0.2s',
                    }}
                  >
                    <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', marginBottom: '0.25rem' }}>
                      <i className={`bi ${type.icon}`} style={{ color: 'var(--primary-color)' }}></i>
                      <span style={{ fontWeight: 500 }}>{type.label}</span>
                    </div>
                    <p style={{ fontSize: '0.75rem', color: 'var(--text-muted)', margin: 0 }}>
                      {type.description}
                    </p>
                  </div>
                ))}
              </div>
            </div>

            <div className="form-group">
              <label className="form-label">Configuration</label>
              <div className="editor-container">
                <Editor
                  height="200px"
                  language="json"
                  theme="vs-dark"
                  value={config}
                  onChange={(value) => setConfig(value || '')}
                  options={{
                    minimap: { enabled: false },
                    fontSize: 13,
                    lineNumbers: 'off',
                    folding: false,
                  }}
                />
              </div>
            </div>

            <div style={{ display: 'flex', gap: '0.75rem', marginTop: '1.5rem' }}>
              <button 
                type="button" 
                className="btn btn-secondary"
                onClick={() => { setShowForm(false); resetForm(); }}
              >
                Cancel
              </button>
              <button type="submit" className="btn btn-primary" disabled={saving}>
                {saving ? 'Creating...' : 'Create Configuration'}
              </button>
            </div>
          </form>
        </div>
      )}

      {!showForm && configs.length === 0 ? (
        <div className="empty-state">
          <i className="bi bi-person-badge"></i>
          <h3>No context configurations</h3>
          <p>Add configurations to enable context-aware tool execution</p>
          <button className="btn btn-primary" onClick={() => setShowForm(true)}>
            <i className="bi bi-plus-lg"></i>
            Add Configuration
          </button>
        </div>
      ) : !showForm && (
        <div>
          {configs.map((cfg) => (
            <div key={cfg.id} className="tool-card">
              <div className="tool-icon" style={{ background: '#8b5cf6', color: 'white' }}>
                <i className={`bi ${sourceTypes.find(t => t.value === cfg.source_type)?.icon || 'bi-gear'}`}></i>
              </div>
              <div className="tool-info">
                <div className="tool-name">{cfg.name}</div>
                <div className="tool-description">
                  {sourceTypes.find(t => t.value === cfg.source_type)?.label || cfg.source_type}
                </div>
                <div style={{ marginTop: '0.5rem' }}>
                  <span className="badge badge-primary">{cfg.source_type}</span>
                </div>
              </div>
              <div className="tool-actions">
                <button 
                  className="btn btn-icon btn-secondary btn-sm"
                  onClick={() => {
                    setName(cfg.name);
                    setSourceType(cfg.source_type);
                    setConfig(JSON.stringify(cfg.config || {}, null, 2));
                    setShowForm(true);
                  }}
                  data-tooltip="Edit"
                >
                  <i className="bi bi-pencil"></i>
                </button>
                <button 
                  className="btn btn-icon btn-secondary btn-sm"
                  onClick={() => onConfigDeleted(cfg.id)}
                  data-tooltip="Delete"
                >
                  <i className="bi bi-trash"></i>
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
