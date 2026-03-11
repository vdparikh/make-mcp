import { useState } from 'react';

interface Props {
  onClose: () => void;
  onCreate: (name: string, description: string) => void;
}

export default function CreateServerModal({ onClose, onCreate }: Props) {
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim()) return;

    setLoading(true);
    await onCreate(name, description);
    setLoading(false);
  };

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal-content" onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <h2 className="modal-title">Create New Server</h2>
          <button 
            className="btn btn-icon btn-secondary" 
            onClick={onClose}
          >
            <i className="bi bi-x-lg"></i>
          </button>
        </div>

        <form onSubmit={handleSubmit}>
          <div className="modal-body">
            <div className="form-group">
              <label className="form-label">Server Name *</label>
              <input
                type="text"
                className="form-control"
                placeholder="e.g., weather-server"
                value={name}
                onChange={(e) => setName(e.target.value)}
                required
                autoFocus
              />
            </div>

            <div className="form-group">
              <label className="form-label">Description</label>
              <textarea
                className="form-control"
                placeholder="Describe what this MCP server does..."
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                rows={3}
              />
            </div>

            <div style={{ 
              background: 'var(--dark-bg)', 
              borderRadius: '8px', 
              padding: '1rem',
              marginTop: '1rem'
            }}>
              <h4 style={{ fontSize: '0.875rem', marginBottom: '0.5rem', color: 'var(--text-primary)' }}>
                <i className="bi bi-lightbulb" style={{ marginRight: '0.5rem', color: 'var(--warning-color)' }}></i>
                What is an MCP Server?
              </h4>
              <p style={{ fontSize: '0.8125rem', color: 'var(--text-secondary)', margin: 0 }}>
                An MCP (Model Context Protocol) server exposes tools, resources, and prompts 
                that AI agents can use. Once created, you can add tools (functions AI can call), 
                resources (data endpoints), and prompts (templated instructions).
              </p>
            </div>
          </div>

          <div className="modal-footer">
            <button type="button" className="btn btn-secondary" onClick={onClose}>
              Cancel
            </button>
            <button 
              type="submit" 
              className="btn btn-primary" 
              disabled={!name.trim() || loading}
            >
              {loading ? (
                <>
                  <span className="spinner" style={{ width: 16, height: 16, borderWidth: 2 }}></span>
                  Creating...
                </>
              ) : (
                <>
                  <i className="bi bi-plus-lg"></i>
                  Create Server
                </>
              )}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
