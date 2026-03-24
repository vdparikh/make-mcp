import { useState, useEffect, useRef } from 'react';
import { toast } from 'react-toastify';
import Editor from '@monaco-editor/react';
import type { Resource } from '../types';
import { createResource } from '../services/api';

interface Props {
  serverId: string;
  resources: Resource[];
  onResourceCreated: () => void;
  onResourceDeleted: (id: string) => void;
  /** When set (e.g. sidebar), open this resource in the editor */
  focusResourceId?: string | null;
  /** Clear parent sidebar selection when leaving the editor */
  onCloseEdit?: () => void;
}

export default function ResourceEditor({
  serverId,
  resources,
  onResourceCreated,
  onResourceDeleted,
  focusResourceId,
  onCloseEdit,
}: Props) {
  const openedViaSidebarRef = useRef(false);
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

  const loadResourceIntoForm = (resource: Resource) => {
    setName(resource.name);
    setUri(resource.uri);
    setMimeType(resource.mime_type);
    setHandler(JSON.stringify(resource.handler || {}, null, 2));
  };

  // Sidebar / parent focus: open the same edit view as the card "Edit" button
  useEffect(() => {
    if (!focusResourceId) {
      if (openedViaSidebarRef.current) {
        setShowForm(false);
        resetForm();
        openedViaSidebarRef.current = false;
      }
      return;
    }
    if (resources.length === 0) return;
    const resource = resources.find((r) => r.id === focusResourceId);
    if (resource) {
      openedViaSidebarRef.current = true;
      loadResourceIntoForm(resource);
      setShowForm(true);
    }
  }, [focusResourceId, resources]);

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
      onCloseEdit?.();
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
            type="button"
            className="btn btn-outline-primary btn-sm"
            onClick={() => { setShowForm(false); resetForm(); }}
            title="Return to resources list"
            style={{ fontWeight: 600 }}
          >
            <i className="bi bi-arrow-left" style={{ marginRight: '0.35rem' }}></i>
            Back to list
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
              onClick={() => {
                setShowForm(false);
                resetForm();
                openedViaSidebarRef.current = false;
                onCloseEdit?.();
              }}
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
        <button
          className="btn btn-primary"
          onClick={() => {
            openedViaSidebarRef.current = false;
            onCloseEdit?.();
            resetForm();
            setShowForm(true);
          }}
        >
          <i className="bi bi-plus-lg"></i>
          Add Resource
        </button>
      </div>

      {resources.length === 0 ? (
        <div className="empty-state">
          <i className="bi bi-folder"></i>
          <h3>No resources yet</h3>
          <p>Resources provide structured data endpoints</p>
          <button
            className="btn btn-primary"
            onClick={() => {
              openedViaSidebarRef.current = false;
              onCloseEdit?.();
              resetForm();
              setShowForm(true);
            }}
          >
            <i className="bi bi-plus-lg"></i>
            Create First Resource
          </button>
        </div>
      ) : (
        <div>
          {resources.map((resource) => (
            <div key={resource.id} className="tool-card">
              <div className="tool-icon" style={{ background: '#10b981', color: 'white' }}>
                <i className="bi bi-folder-fill"></i>
              </div>
              <div className="tool-info">
                <div className="tool-name">{resource.name}</div>
                <div className="tool-description">
                  <code style={{ background: '#1a1a2e', color: '#e5e7eb', padding: '0.125rem 0.5rem', borderRadius: '4px', fontSize: '0.75rem' }}>
                    {resource.uri}
                  </code>
                </div>
                <div style={{ marginTop: '0.5rem' }}>
                  <span className="badge badge-success">{resource.mime_type}</span>
                </div>
              </div>
              <div className="tool-actions">
                <button 
                  className="btn btn-icon btn-secondary btn-sm"
                  onClick={() => {
                    openedViaSidebarRef.current = false;
                    onCloseEdit?.();
                    loadResourceIntoForm(resource);
                    setShowForm(true);
                  }}
                  data-tooltip="Edit"
                >
                  <i className="bi bi-pencil"></i>
                </button>
                <button 
                  className="btn btn-icon btn-secondary btn-sm"
                  onClick={() => onResourceDeleted(resource.id)}
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
