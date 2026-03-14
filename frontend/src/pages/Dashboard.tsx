import { useState, useEffect } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { toast } from 'react-toastify';
import type { Server } from '../types';
import { listServers, createServer, createDemoServer, deleteServer } from '../services/api';
import CreateServerModal from '../components/CreateServerModal';
import { useAuth } from '../contexts/AuthContext';

export default function Dashboard() {
  const navigate = useNavigate();
  const { user, token } = useAuth();
  const [servers, setServers] = useState<Server[]>([]);
  const [loading, setLoading] = useState(true);
  const [showCreateModal, setShowCreateModal] = useState(false);

  // Refetch whenever auth identity or token changes so switching users always shows correct list
  useEffect(() => {
    if (!token || !user?.id) {
      setServers([]);
      setLoading(false);
      return;
    }
    let cancelled = false;
    setLoading(true);
    listServers()
      .then((data) => {
        if (!cancelled) setServers(data ?? []);
      })
      .catch(() => {
        if (!cancelled) toast.error('Failed to load servers');
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => { cancelled = true; };
  }, [token, user?.id]);

  const loadServers = async () => {
    try {
      setLoading(true);
      const data = await listServers();
      setServers(data ?? []);
    } catch (error) {
      toast.error('Failed to load servers');
    } finally {
      setLoading(false);
    }
  };

  const handleCreateServer = async (name: string, description: string, icon: string) => {
    try {
      await createServer({ name, description, version: '1.0.0', icon });
      toast.success('Server created successfully');
      setShowCreateModal(false);
      loadServers();
    } catch (error) {
      toast.error('Failed to create server');
    }
  };

  const handleCreateDemoServer = async () => {
    try {
      await createDemoServer();
      toast.success('Demo server created. Open it to explore tools, resources, and prompts.');
      loadServers();
    } catch (error) {
      toast.error('Failed to create demo server');
    }
  };

  const handleDeleteServer = async (id: string) => {
    if (!confirm('Are you sure you want to delete this server?')) return;
    
    try {
      await deleteServer(id);
      toast.success('Server deleted');
      loadServers();
    } catch (error) {
      toast.error('Failed to delete server');
    }
  };

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString('en-US', {
      month: 'short',
      day: 'numeric',
      year: 'numeric',
    });
  };

  return (
    <div>
      <div className="page-header">
        <div>
          <nav style={{ marginBottom: '0.5rem' }}>
            <span style={{ color: 'var(--text-primary)', fontSize: '0.875rem' }}>
              <i className="bi bi-house-door" style={{ marginRight: '0.375rem' }}></i>
              Dashboard
            </span>
          </nav>
          <h1 className="page-title">MCP Servers</h1>
          <p className="page-subtitle">Create and manage your Model Context Protocol servers</p>
        </div>
        <div style={{ display: 'flex', gap: '0.75rem' }}>
          <button className="btn btn-secondary" onClick={() => navigate('/import/openapi')}>
            <i className="bi bi-file-earmark-code"></i>
            Import OpenAPI
          </button>
          <button className="btn btn-primary" onClick={() => setShowCreateModal(true)}>
            <i className="bi bi-plus-lg"></i>
            New Server
          </button>
        </div>
      </div>

      <div className="stats-grid">
        <div className="stat-card">
          <div className="stat-value">{servers.length}</div>
          <div className="stat-label">Total Servers</div>
        </div>
        <div className="stat-card">
          <div className="stat-value">
            {servers.filter(s => s.status === 'published').length}
          </div>
          <div className="stat-label">Published</div>
        </div>
        <div className="stat-card">
          <div className="stat-value">
            {servers.reduce((acc, s) => acc + (s.tools?.length || 0), 0)}
          </div>
          <div className="stat-label">Total Tools</div>
        </div>
        <div className="stat-card">
          <div className="stat-value">
            {servers.reduce((acc, s) => acc + (s.resources?.length || 0), 0)}
          </div>
          <div className="stat-label">Total Resources</div>
        </div>
      </div>

      {loading ? (
        <div className="loading">
          <div className="spinner"></div>
        </div>
      ) : servers.length === 0 ? (
        <div className="empty-state">
          <i className="bi bi-server"></i>
          <h3>No servers yet</h3>
          <p>Create your first MCP server to get started, or add the demo server to explore the system.</p>
          <div style={{ display: 'flex', gap: '0.75rem', flexWrap: 'wrap', justifyContent: 'center' }}>
            <button className="btn btn-primary" onClick={() => setShowCreateModal(true)}>
              <i className="bi bi-plus-lg"></i>
              Create Server
            </button>
            <button className="btn btn-secondary" onClick={handleCreateDemoServer}>
              <i className="bi bi-box-seam"></i>
              Create demo server
            </button>
          </div>
        </div>
      ) : (
        <div className="server-grid">
          {servers.map((server) => (
            <div className="card" key={server.id}>
              <div style={{ display: 'flex', alignItems: 'flex-start', gap: '1rem', marginBottom: '1rem' }}>
                <div style={{
                  width: '48px',
                  height: '48px',
                  borderRadius: '12px',
                  background: 'var(--primary-light)',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  fontSize: '1.5rem',
                  color: 'var(--primary-color)',
                  flexShrink: 0
                }}>
                  <i className={`bi ${server.icon || 'bi-server'}`}></i>
                </div>
                <div style={{ flex: 1, minWidth: 0 }}>
                  <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: '0.5rem' }}>
                    <h3 className="card-title" style={{ margin: 0 }}>{server.name}</h3>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
                      <span className={`status-badge ${server.status || 'draft'}`} style={{ fontSize: '0.65rem' }}>
                        {server.status === 'published' ? 'Published' : server.status === 'archived' ? 'Archived' : 'Draft'}
                      </span>
                      <span className="badge badge-primary">v{server.latest_version || server.version}</span>
                    </div>
                  </div>
                  <p className="card-description" style={{ margin: '0.25rem 0 0 0' }}>
                    {server.description || 'No description'}
                  </p>
                </div>
              </div>
              
              <div className="card-meta">
                <div className="card-meta-item">
                  <i className="bi bi-tools"></i>
                  <span>{server.tools?.length || 0} Tools</span>
                </div>
                <div className="card-meta-item">
                  <i className="bi bi-folder"></i>
                  <span>{server.resources?.length || 0} Resources</span>
                </div>
                <div className="card-meta-item">
                  <i className="bi bi-chat-text"></i>
                  <span>{server.prompts?.length || 0} Prompts</span>
                </div>
              </div>

              <div style={{ 
                display: 'flex', 
                justifyContent: 'space-between', 
                alignItems: 'center',
                marginTop: '1rem',
                paddingTop: '1rem',
                borderTop: '1px solid var(--card-border)'
              }}>
                <span style={{ fontSize: '0.75rem', color: 'var(--text-muted)' }}>
                  Updated {formatDate(server.updated_at)}
                </span>
                <div style={{ display: 'flex', gap: '0.5rem' }}>
                  <button 
                    className="btn btn-icon btn-secondary btn-sm"
                    onClick={() => handleDeleteServer(server.id)}
                    data-tooltip="Delete"
                  >
                    <i className="bi bi-trash"></i>
                  </button>
                  <Link to={`/servers/${server.id}`} className="btn btn-primary btn-sm">
                    <i className="bi bi-pencil"></i>
                    Edit
                  </Link>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}

      {showCreateModal && (
        <CreateServerModal
          onClose={() => setShowCreateModal(false)}
          onCreate={handleCreateServer}
        />
      )}
    </div>
  );
}
