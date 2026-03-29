import { useState, useEffect } from 'react';
import { Node } from 'reactflow';
import Editor from '@monaco-editor/react';

interface NodeConfigPanelProps {
  node: Node;
  onUpdate: (nodeId: string, data: Record<string, unknown>) => void;
  onDelete: (nodeId: string) => void;
  onClose: () => void;
}

export default function NodeConfigPanel({ node, onUpdate, onDelete, onClose }: NodeConfigPanelProps) {
  const [label, setLabel] = useState(node.data.label || '');
  const [description, setDescription] = useState(node.data.description || '');
  const [config, setConfig] = useState(JSON.stringify(node.data.config || {}, null, 2));

  useEffect(() => {
    setLabel(node.data.label || '');
    setDescription(node.data.description || '');
    setConfig(JSON.stringify(node.data.config || {}, null, 2));
  }, [node]);

  const handleSave = () => {
    try {
      const parsedConfig = JSON.parse(config);
      onUpdate(node.id, { label, description, config: parsedConfig });
    } catch {
      // Invalid JSON, just save without config update
      onUpdate(node.id, { label, description });
    }
  };

  const getNodeTypeInfo = () => {
    switch (node.type) {
      case 'trigger':
        return { icon: 'bi-play-circle', color: '#10b981', name: 'Trigger' };
      case 'api':
        return { icon: 'bi-globe', color: '#6366f1', name: 'API Call' };
      case 'cli':
        return { icon: 'bi-terminal', color: '#f59e0b', name: 'CLI Command' };
      case 'transform':
        return { icon: 'bi-shuffle', color: '#06b6d4', name: 'Transform' };
      case 'condition':
        return { icon: 'bi-signpost-split', color: '#f97316', name: 'Condition' };
      case 'output':
        return { icon: 'bi-box-arrow-right', color: '#ef4444', name: 'Output' };
      default:
        return { icon: 'bi-circle', color: '#6b7280', name: 'Node' };
    }
  };

  const typeInfo = getNodeTypeInfo();

  const renderTypeSpecificConfig = () => {
    switch (node.type) {
      case 'api':
        return (
          <>
            <div className="config-field">
              <label>Method</label>
              <select
                value={node.data.config?.method || 'GET'}
                onChange={(e) => {
                  const newConfig = { ...node.data.config, method: e.target.value };
                  setConfig(JSON.stringify(newConfig, null, 2));
                  onUpdate(node.id, { config: newConfig });
                }}
              >
                <option value="GET">GET</option>
                <option value="POST">POST</option>
                <option value="PUT">PUT</option>
                <option value="DELETE">DELETE</option>
                <option value="PATCH">PATCH</option>
              </select>
            </div>
            <div className="config-field">
              <label>URL</label>
              <input
                type="text"
                placeholder="https://api.example.com/endpoint"
                value={node.data.config?.url || ''}
                onChange={(e) => {
                  const newConfig = { ...node.data.config, url: e.target.value };
                  setConfig(JSON.stringify(newConfig, null, 2));
                  onUpdate(node.id, { config: newConfig });
                }}
              />
            </div>
          </>
        );
      
      case 'cli':
        return (
          <>
            <div className="config-field">
              <label>Command</label>
              <input
                type="text"
                placeholder="kubectl get pods -n {{namespace}}"
                value={node.data.config?.command || ''}
                onChange={(e) => {
                  const newConfig = { ...node.data.config, command: e.target.value };
                  setConfig(JSON.stringify(newConfig, null, 2));
                  onUpdate(node.id, { config: newConfig });
                }}
              />
            </div>
            <div className="config-field">
              <label>Timeout (ms)</label>
              <input
                type="number"
                value={node.data.config?.timeout || 30000}
                onChange={(e) => {
                  const newConfig = { ...node.data.config, timeout: parseInt(e.target.value) };
                  setConfig(JSON.stringify(newConfig, null, 2));
                  onUpdate(node.id, { config: newConfig });
                }}
              />
            </div>
          </>
        );
      
      case 'transform':
        return (
          <div className="config-field">
            <label>Field path (Test Flow)</label>
            <input
              type="text"
              placeholder="origin"
              value={node.data.config?.expression || ''}
              onChange={(e) => {
                const newConfig = { ...node.data.config, expression: e.target.value };
                setConfig(JSON.stringify(newConfig, null, 2));
                onUpdate(node.id, { config: newConfig });
              }}
            />
            <small className="text-muted d-block mt-1">
              Dot path into the previous node&apos;s JSON (e.g. <code>origin</code>, <code>headers.Host</code>). Leave empty to pass data through. The generated tool can use richer expressions when you convert the flow.
            </small>
          </div>
        );
      
      case 'condition':
        return (
          <>
            <div className="config-field">
              <label>Condition</label>
              <input
                type="text"
                placeholder="data.status === 'success'"
                value={node.data.config?.condition || ''}
                onChange={(e) => {
                  const newConfig = { ...node.data.config, condition: e.target.value };
                  setConfig(JSON.stringify(newConfig, null, 2));
                  onUpdate(node.id, { config: newConfig });
                }}
              />
            </div>
          </>
        );
      
      default:
        return null;
    }
  };

  return (
    <div className="node-config-panel">
      <div className="panel-header">
        <div className="panel-title">
          <i className={`bi ${typeInfo.icon}`} style={{ color: typeInfo.color }}></i>
          <span>{typeInfo.name}</span>
        </div>
        <button className="btn btn-icon btn-sm" onClick={onClose}>
          <i className="bi bi-x-lg"></i>
        </button>
      </div>

      <div className="panel-content">
        <div className="config-section">
          <div className="config-field">
            <label>Label</label>
            <input
              type="text"
              value={label}
              onChange={(e) => setLabel(e.target.value)}
              onBlur={() => onUpdate(node.id, { label })}
            />
          </div>

          <div className="config-field">
            <label>Description</label>
            <input
              type="text"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              onBlur={() => onUpdate(node.id, { description })}
            />
          </div>
        </div>

        <div className="config-section">
          <h4>Configuration</h4>
          {renderTypeSpecificConfig()}
        </div>

        <div className="config-section">
          <h4>Advanced (JSON)</h4>
          <div className="editor-container" style={{ height: '200px' }}>
            <Editor
              height="100%"
              language="json"
              theme="vs-dark"
              value={config}
              onChange={(value) => setConfig(value || '{}')}
              options={{
                minimap: { enabled: false },
                fontSize: 12,
                lineNumbers: 'off',
                folding: false,
              }}
            />
          </div>
        </div>
      </div>

      <div className="panel-footer">
        <button className="btn btn-danger btn-sm" onClick={() => onDelete(node.id)}>
          <i className="bi bi-trash"></i>
          Delete Node
        </button>
        <button className="btn btn-primary btn-sm" onClick={handleSave}>
          <i className="bi bi-check"></i>
          Apply
        </button>
      </div>
    </div>
  );
}
