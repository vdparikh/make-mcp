import { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';
import { toast } from 'react-toastify';
import type { Server } from '../types';
import { listServers, createServer, deleteServer } from '../services/api';
import CreateServerModal from '../components/CreateServerModal';

export default function Dashboard() {
  const [servers, setServers] = useState<Server[]>([]);
  const [loading, setLoading] = useState(true);
  const [showCreateModal, setShowCreateModal] = useState(false);

  useEffect(() => {
    loadServers();
  }, []);

  const loadServers = async () => {
    try {
      setLoading(true);
      const data = await listServers();
      setServers(data);
    } catch (error) {
      toast.error('Failed to load servers');
    } finally {
      setLoading(false);
    }
  };

  const handleCreateServer = async (name: string, description: string) => {
    try {
      await createServer({ name, description, version: '1.0.0' });
      toast.success('Server created successfully');
      setShowCreateModal(false);
      loadServers();
    } catch (error) {
      toast.error('Failed to create server');
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
          <h1 className="page-title">MCP Servers</h1>
          <p className="page-subtitle">Create and manage your Model Context Protocol servers</p>
        </div>
        <button className="btn btn-primary" onClick={() => setShowCreateModal(true)}>
          <i className="bi bi-plus-lg"></i>
          New Server
        </button>
      </div>

      <div className="stats-grid">
        <div className="stat-card">
          <div className="stat-value">{servers.length}</div>
          <div className="stat-label">Total Servers</div>
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
        <div className="stat-card">
          <div className="stat-value">
            {servers.reduce((acc, s) => acc + (s.prompts?.length || 0), 0)}
          </div>
          <div className="stat-label">Total Prompts</div>
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
          <p>Create your first MCP server to get started</p>
          <button className="btn btn-primary" onClick={() => setShowCreateModal(true)}>
            <i className="bi bi-plus-lg"></i>
            Create Server
          </button>
        </div>
      ) : (
        <div className="server-grid">
          {servers.map((server) => (
            <div className="card" key={server.id}>
              <div className="card-header">
                <div>
                  <h3 className="card-title">{server.name}</h3>
                  <p className="card-description">
                    {server.description || 'No description'}
                  </p>
                </div>
                <span className="badge badge-primary">v{server.version}</span>
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
                    title="Delete"
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
