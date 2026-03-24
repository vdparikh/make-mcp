import { useState } from 'react';

interface Props {
  onClose: () => void;
  onCreate: (name: string, description: string, icon: string) => void;
}

const ICON_OPTIONS = [
  { icon: 'bi-server', label: 'Server' },
  { icon: 'bi-cloud', label: 'Cloud' },
  { icon: 'bi-database', label: 'Database' },
  { icon: 'bi-globe', label: 'Web' },
  { icon: 'bi-robot', label: 'Robot' },
  { icon: 'bi-cpu', label: 'CPU' },
  { icon: 'bi-terminal', label: 'Terminal' },
  { icon: 'bi-code-slash', label: 'Code' },
  { icon: 'bi-gear', label: 'Settings' },
  { icon: 'bi-lightning', label: 'Fast' },
  { icon: 'bi-shield-check', label: 'Security' },
  { icon: 'bi-graph-up', label: 'Analytics' },
  { icon: 'bi-chat-dots', label: 'Chat' },
  { icon: 'bi-envelope', label: 'Email' },
  { icon: 'bi-calendar', label: 'Calendar' },
  { icon: 'bi-file-text', label: 'Documents' },
  { icon: 'bi-currency-dollar', label: 'Finance' },
  { icon: 'bi-cart', label: 'Commerce' },
  { icon: 'bi-person', label: 'User' },
  { icon: 'bi-building', label: 'Business' },
  { icon: 'bi-box', label: 'Package' },
  { icon: 'bi-palette', label: 'Design' },
  { icon: 'bi-music-note', label: 'Music' },
  { icon: 'bi-camera', label: 'Media' },
];

export default function CreateServerModal({ onClose, onCreate }: Props) {
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [icon, setIcon] = useState('bi-server');
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim()) return;

    setLoading(true);
    await onCreate(name, description, icon);
    setLoading(false);
  };

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal-content" onClick={(e) => e.stopPropagation()} style={{ maxWidth: '550px' }}>
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
                rows={2}
              />
            </div>

            <div className="form-group">
              <label className="form-label">Choose an Icon</label>
              <div style={{ 
                display: 'grid', 
                gridTemplateColumns: 'repeat(8, 1fr)', 
                gap: '0.5rem',
                padding: '0.75rem',
                background: 'var(--dark-bg)',
                borderRadius: '8px',
                maxHeight: '180px',
                overflowY: 'auto'
              }}>
                {ICON_OPTIONS.map(({ icon: iconClass, label }) => (
                  <button
                    key={iconClass}
                    type="button"
                    onClick={() => setIcon(iconClass)}
                    title={label}
                    style={{
                      width: '44px',
                      height: '44px',
                      borderRadius: '8px',
                      border: icon === iconClass ? '2px solid var(--primary-color)' : '2px solid transparent',
                      background: icon === iconClass ? 'var(--primary-light)' : 'var(--card-bg)',
                      cursor: 'pointer',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      fontSize: '1.25rem',
                      color: icon === iconClass ? 'var(--primary-color)' : 'var(--text-secondary)',
                      transition: 'all 0.15s'
                    }}
                  >
                    <i className={`bi ${iconClass}`}></i>
                  </button>
                ))}
              </div>
            </div>

            {/* Preview */}
            <div style={{ 
              display: 'flex',
              alignItems: 'center',
              gap: '1rem',
              padding: '1rem',
              background: 'var(--card-bg)',
              border: '1px solid var(--card-border)',
              borderRadius: '8px',
              marginTop: '1rem'
            }}>
              <div style={{
                width: '48px',
                height: '48px',
                borderRadius: '12px',
                background: 'var(--primary-light)',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                fontSize: '1.5rem',
                color: 'var(--primary-color)'
              }}>
                <i className={`bi ${icon}`}></i>
              </div>
              <div>
                <div style={{ fontWeight: 600, color: 'var(--text-primary)' }}>
                  {name || 'Server Name'}
                </div>
                <div style={{ fontSize: '0.8125rem', color: 'var(--text-secondary)' }}>
                  {description || 'Server description'}
                </div>
              </div>
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
