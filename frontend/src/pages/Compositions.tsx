import { useState, useEffect } from 'react';
import { toast } from 'react-toastify';
import type { Server, ServerComposition } from '../types';
import { listServers, listCompositions, createComposition } from '../services/api';

export default function Compositions() {
  const [servers, setServers] = useState<Server[]>([]);
  const [compositions, setCompositions] = useState<ServerComposition[]>([]);
  const [loading, setLoading] = useState(true);
  const [showForm, setShowForm] = useState(false);
  
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [selectedServers, setSelectedServers] = useState<string[]>([]);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    loadData();
  }, []);

  const loadData = async () => {
    try {
      setLoading(true);
      const [serversData, compositionsData] = await Promise.all([
        listServers(),
        listCompositions(),
      ]);
      setServers(serversData);
      setCompositions(compositionsData);
    } catch (error) {
      toast.error('Failed to load data');
    } finally {
      setLoading(false);
    }
  };

  const resetForm = () => {
    setName('');
    setDescription('');
    setSelectedServers([]);
  };

  const toggleServer = (serverId: string) => {
    if (selectedServers.includes(serverId)) {
      setSelectedServers(selectedServers.filter(id => id !== serverId));
    } else {
      setSelectedServers([...selectedServers, serverId]);
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    if (selectedServers.length < 2) {
      toast.error('Select at least 2 servers to compose');
      return;
    }

    try {
      setSaving(true);
      await createComposition({
        name,
        description,
        server_ids: selectedServers,
      });
      toast.success('Composition created');
      setShowForm(false);
      resetForm();
      loadData();
    } catch (error) {
      toast.error('Failed to create composition');
    } finally {
      setSaving(false);
    }
  };

  const getServerName = (serverId: string) => {
    return servers.find(s => s.id === serverId)?.name || serverId;
  };

  if (loading) {
    return (
      <div className="loading">
        <div className="spinner"></div>
      </div>
    );
  }

  return (
    <div>
      <div className="page-header">
        <div>
          <h1 className="page-title">Server Compositions</h1>
          <p className="page-subtitle">Combine multiple MCP servers into one unified interface</p>
        </div>
        <button className="btn btn-primary" onClick={() => setShowForm(true)}>
          <i className="bi bi-plus-lg"></i>
          New Composition
        </button>
      </div>

      <div className="card" style={{ marginBottom: '1.5rem' }}>
        <h3 className="card-title" style={{ marginBottom: '0.75rem' }}>
          <i className="bi bi-layers" style={{ marginRight: '0.75rem', color: 'var(--primary-color)' }}></i>
          MCP Server Composition
        </h3>
        <p style={{ color: 'var(--text-secondary)', marginBottom: '1rem' }}>
          Compose multiple MCP servers into a single unified interface. This enables complex AI workflows 
          that span multiple services.
        </p>
        
        <div style={{ 
          background: 'linear-gradient(135deg, rgba(129, 140, 248, 0.15), rgba(56, 189, 248, 0.08))',
          border: '1px solid rgba(129, 140, 248, 0.3)',
          borderRadius: '8px',
          padding: '1rem',
        }}>
          <h4 style={{ fontSize: '0.875rem', marginBottom: '0.5rem', color: 'var(--text-primary)' }}>
            <i className="bi bi-diagram-3" style={{ marginRight: '0.5rem', color: 'var(--secondary-color)' }}></i>
            Example: Sales Agent Composition
          </h4>
          <div style={{ display: 'flex', gap: '1rem', flexWrap: 'wrap', marginBottom: '0.5rem', alignItems: 'center' }}>
            <span className="badge badge-primary">Stripe MCP</span>
            <span style={{ color: 'var(--text-secondary)' }}>+</span>
            <span className="badge badge-primary">Salesforce MCP</span>
            <span style={{ color: 'var(--text-secondary)' }}>+</span>
            <span className="badge badge-primary">Slack MCP</span>
            <span style={{ color: 'var(--text-secondary)' }}>=</span>
            <span className="badge badge-success">Sales Agent MCP</span>
          </div>
          <p style={{ fontSize: '0.8125rem', color: '#e2e8f0', margin: 0 }}>
            AI workflow: Find lead (Salesforce) → Create invoice (Stripe) → Notify team (Slack)
          </p>
        </div>
      </div>

      {showForm && (
        <div className="card" style={{ marginBottom: '1.5rem' }}>
          <div className="card-header">
            <h3 className="card-title">Create New Composition</h3>
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
                <label className="form-label">Composition Name *</label>
                <input
                  type="text"
                  className="form-control"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  placeholder="e.g., Sales Agent"
                  required
                />
              </div>
              <div className="form-group">
                <label className="form-label">Description</label>
                <input
                  type="text"
                  className="form-control"
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  placeholder="Describe this composition..."
                />
              </div>
            </div>

            <div className="form-group">
              <label className="form-label">Select Servers to Compose ({selectedServers.length} selected)</label>
              <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(200px, 1fr))', gap: '0.75rem' }}>
                {servers.map((server) => (
                  <div
                    key={server.id}
                    onClick={() => toggleServer(server.id)}
                    style={{
                      padding: '1rem',
                      background: selectedServers.includes(server.id) ? 'rgba(99, 102, 241, 0.15)' : 'var(--dark-bg)',
                      border: `2px solid ${selectedServers.includes(server.id) ? 'var(--primary-color)' : 'var(--card-border)'}`,
                      borderRadius: '8px',
                      cursor: 'pointer',
                      transition: 'all 0.2s',
                    }}
                  >
                    <div style={{ display: 'flex', alignItems: 'center', gap: '0.75rem' }}>
                      <div style={{ 
                        width: 20, 
                        height: 20, 
                        borderRadius: '4px',
                        border: `2px solid ${selectedServers.includes(server.id) ? 'var(--primary-color)' : 'var(--card-border)'}`,
                        background: selectedServers.includes(server.id) ? 'var(--primary-color)' : 'transparent',
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                      }}>
                        {selectedServers.includes(server.id) && (
                          <i className="bi bi-check" style={{ color: 'white', fontSize: '0.75rem' }}></i>
                        )}
                      </div>
                      <div>
                        <div style={{ fontWeight: 500, color: 'var(--text-primary)' }}>{server.name}</div>
                        <div style={{ fontSize: '0.75rem', color: 'var(--text-secondary)' }}>
                          {server.tools?.length || 0} tools
                        </div>
                      </div>
                    </div>
                  </div>
                ))}
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
              <button 
                type="submit" 
                className="btn btn-primary" 
                disabled={saving || selectedServers.length < 2}
              >
                {saving ? 'Creating...' : 'Create Composition'}
              </button>
            </div>
          </form>
        </div>
      )}

      {compositions.length === 0 && !showForm ? (
        <div className="empty-state">
          <i className="bi bi-layers"></i>
          <h3>No compositions yet</h3>
          <p>Combine multiple MCP servers into one unified interface</p>
          <button className="btn btn-primary" onClick={() => setShowForm(true)}>
            <i className="bi bi-plus-lg"></i>
            Create First Composition
          </button>
        </div>
      ) : (
        <div className="server-grid">
          {compositions.map((composition) => (
            <div className="card" key={composition.id}>
              <div className="card-header">
                <div>
                  <h3 className="card-title">{composition.name}</h3>
                  <p className="card-description">
                    {composition.description || 'No description'}
                  </p>
                </div>
                <span className="badge badge-success">
                  <i className="bi bi-layers" style={{ marginRight: '0.25rem' }}></i>
                  Composed
                </span>
              </div>
              
              <div className="composition-servers">
                {composition.server_ids.map((serverId) => (
                  <span key={serverId} className="composition-server-badge">
                    <i className="bi bi-server"></i>
                    {getServerName(serverId)}
                  </span>
                ))}
              </div>

              <div className="card-meta">
                <div className="card-meta-item">
                  <i className="bi bi-boxes"></i>
                  <span>{composition.server_ids.length} Servers</span>
                </div>
                <div className="card-meta-item">
                  <i className="bi bi-tools"></i>
                  <span>
                    {composition.server_ids.reduce((acc, id) => {
                      const server = servers.find(s => s.id === id);
                      return acc + (server?.tools?.length || 0);
                    }, 0)} Total Tools
                  </span>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
