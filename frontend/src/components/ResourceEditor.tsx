import { useState } from 'react';
import { toast } from 'react-toastify';
import Editor from '@monaco-editor/react';
import type { Resource } from '../types';
import { createResource } from '../services/api';

interface Props {
  serverId: string;
  resources: Resource[];
  onResourceCreated: () => void;
  onResourceDeleted: (id: string) => void;
}

export default function ResourceEditor({ serverId, resources, onResourceCreated, onResourceDeleted }: Props) {
  const [showForm, setShowForm] = useState(false);
  const [name, setName] = useState('');
  const [uri, setUri] = useState('');
  const [mimeType, setMimeType] = useState('application/json');
  const [handler, setHandler] = useState('{\n  "type": "static",\n  "data": {}\n}');
  const [saving, setSaving] = useState(false);

  const resetForm = () => {
    setName('');
    setUri('');
    setMimeType('application/json');
    setHandler('{\n  "type": "static",\n  "data": {}\n}');
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    try {
      setSaving(true);
      
      let parsedHandler;
      try {
        parsedHandler = JSON.parse(handler);
      } catch {
        toast.error('Invalid Handler JSON');
        return;
      }

      await createResource({
        server_id: serverId,
        name,
        uri,
        mime_type: mimeType,
        handler: parsedHandler,
      });

      toast.success('Resource created');
      setShowForm(false);
      resetForm();
      onResourceCreated();
    } catch (error) {
      toast.error('Failed to create resource');
    } finally {
      setSaving(false);
    }
  };

  if (showForm) {
    return (
      <div className="card">
        <div className="card-header">
          <h3 className="card-title">Create New Resource</h3>
          <button 
            className="btn btn-icon btn-secondary"
            onClick={() => { setShowForm(false); resetForm(); }}
          >
            <i className="bi bi-x-lg"></i>
          </button>
        </div>

        <form onSubmit={handleSubmit}>
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '1rem' }}>
            <div className="form-group">
              <label className="form-label">Resource Name *</label>
              <input
                type="text"
                className="form-control"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="e.g., company_docs"
                required
              />
            </div>

            <div className="form-group">
              <label className="form-label">MIME Type</label>
              <select
                className="form-control"
                value={mimeType}
                onChange={(e) => setMimeType(e.target.value)}
              >
                <option value="application/json">application/json</option>
                <option value="text/plain">text/plain</option>
                <option value="text/markdown">text/markdown</option>
                <option value="text/html">text/html</option>
              </select>
            </div>
          </div>

          <div className="form-group">
            <label className="form-label">URI *</label>
            <input
              type="text"
              className="form-control"
              value={uri}
              onChange={(e) => setUri(e.target.value)}
              placeholder="e.g., mcp://docs/company"
              required
            />
          </div>

          <div className="form-group">
            <label className="form-label">Handler Configuration</label>
            <div className="editor-container">
              <Editor
                height="200px"
                language="json"
                theme="vs-dark"
                value={handler}
                onChange={(value) => setHandler(value || '')}
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
              {saving ? 'Creating...' : 'Create Resource'}
            </button>
          </div>
        </form>
      </div>
    );
  }

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '1rem' }}>
        <h3 style={{ margin: 0 }}>Resources ({resources.length})</h3>
        <button className="btn btn-primary" onClick={() => setShowForm(true)}>
          <i className="bi bi-plus-lg"></i>
          Add Resource
        </button>
      </div>

      {resources.length === 0 ? (
        <div className="empty-state">
          <i className="bi bi-folder"></i>
          <h3>No resources yet</h3>
          <p>Resources provide structured data endpoints</p>
          <button className="btn btn-primary" onClick={() => setShowForm(true)}>
            <i className="bi bi-plus-lg"></i>
            Create First Resource
          </button>
        </div>
      ) : (
        <div>
          {resources.map((resource) => (
            <div key={resource.id} className="tool-card">
              <div className="tool-icon" style={{ background: 'linear-gradient(135deg, #10b981, #059669)' }}>
                <i className="bi bi-folder-fill"></i>
              </div>
              <div className="tool-info">
                <div className="tool-name">{resource.name}</div>
                <div className="tool-description">
                  <code style={{ background: 'var(--dark-bg)', padding: '0.125rem 0.5rem', borderRadius: '4px' }}>
                    {resource.uri}
                  </code>
                </div>
                <div style={{ marginTop: '0.5rem' }}>
                  <span className="badge badge-success">{resource.mime_type}</span>
                </div>
              </div>
              <div>
                <button 
                  className="btn btn-icon btn-secondary btn-sm"
                  onClick={() => onResourceDeleted(resource.id)}
                  title="Delete"
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
