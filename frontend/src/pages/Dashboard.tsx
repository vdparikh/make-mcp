import { useState, useEffect } from 'react';
import { Link, useNavigate, useSearchParams } from 'react-router-dom';
import { toast } from 'react-toastify';
import type { Server, ServerComposition } from '../types';
import { listServers, listCompositions, createServer, createDemoServer, deleteServer } from '../services/api';
import CreateServerModal from '../components/CreateServerModal';
import CompositionsTab from '../components/CompositionsTab';
import { useAuth } from '../contexts/AuthContext';

type Tab = 'servers' | 'compositions';

export default function Dashboard() {
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const tabParam = searchParams.get('tab');
  const activeTab: Tab = tabParam === 'compositions' ? 'compositions' : 'servers';

  const { user, token } = useAuth();
  const [servers, setServers] = useState<Server[]>([]);
  const [compositions, setCompositions] = useState<ServerComposition[]>([]);
  const [loading, setLoading] = useState(true);
  const [compositionsLoading, setCompositionsLoading] = useState(false);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [showTemplates, setShowTemplates] = useState(false);
  const [openCompositionForm, setOpenCompositionForm] = useState(false);

  const setTab = (t: Tab) => {
    if (t === 'compositions') setSearchParams({ tab: 'compositions' });
    else setSearchParams({});
  };

  // Load servers whenever auth changes
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

  // Load compositions when on compositions tab
  useEffect(() => {
    if (!token || !user?.id || activeTab !== 'compositions') return;
    let cancelled = false;
    setCompositionsLoading(true);
    listCompositions()
      .then((data) => {
        if (!cancelled) setCompositions(data ?? []);
      })
      .catch(() => {
        if (!cancelled) toast.error('Failed to load compositions');
      })
      .finally(() => {
        if (!cancelled) setCompositionsLoading(false);
      });
    return () => { cancelled = true; };
  }, [token, user?.id, activeTab]);

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

  const loadCompositions = async () => {
    try {
      setCompositionsLoading(true);
      const data = await listCompositions();
      setCompositions(data ?? []);
    } catch {
      toast.error('Failed to load compositions');
    } finally {
      setCompositionsLoading(false);
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
      const server = await createDemoServer();
      toast.success('Demo server created. Open it to explore tools, resources, and prompts.');
      loadServers();
      navigate(`/servers/${server.id}`);
    } catch (error) {
      toast.error('Failed to create demo server');
    }
  };

  const handleCreateRestStarterServer = async () => {
    try {
      const server = await createServer({
        name: 'REST API Toolkit',
        description: 'Starter MCP server pre-configured for building REST API tools.',
        version: '1.0.0',
        icon: 'bi-globe',
      });
      toast.success('REST API starter server created.');
      loadServers();
      navigate(`/servers/${server.id}`);
    } catch (error) {
      toast.error('Failed to create REST starter server');
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
      <div className="page-header" style={{ alignItems: 'flex-start' }}>
        <div style={{ flex: 1, minWidth: 0 }}>
          <nav style={{ marginBottom: '0.5rem' }}>
            <span style={{ color: 'var(--text-primary)', fontSize: '0.875rem' }}>
              <i className="bi bi-house-door" style={{ marginRight: '0.375rem' }}></i>
              Dashboard
            </span>
          </nav>
          <h1 className="page-title">MCP Servers</h1>
          <p className="page-subtitle">Create and manage your Model Context Protocol servers</p>
          {/* Tabs integrated under subtitle */}
          <div
            style={{
              display: 'flex',
              gap: '0',
              marginTop: '1rem',
              borderBottom: '1px solid var(--card-border)',
              paddingBottom: '0',
            }}
          >
            <button
              type="button"
              onClick={() => setTab('servers')}
              style={{
                padding: '0.5rem 1rem 0.75rem 0',
                marginRight: '1.5rem',
                border: 'none',
                borderBottom: activeTab === 'servers' ? '2px solid var(--primary-color)' : '2px solid transparent',
                background: 'transparent',
                color: activeTab === 'servers' ? 'var(--primary-color)' : 'var(--text-muted)',
                fontWeight: activeTab === 'servers' ? 600 : 400,
                cursor: 'pointer',
                fontSize: '0.9375rem',
              }}
            >
              <i className="bi bi-server" style={{ marginRight: '0.5rem' }}></i>
              Servers
            </button>
            <button
              type="button"
              onClick={() => setTab('compositions')}
              style={{
                padding: '0.5rem 1rem 0.75rem 0',
                border: 'none',
                borderBottom: activeTab === 'compositions' ? '2px solid var(--primary-color)' : '2px solid transparent',
                background: 'transparent',
                color: activeTab === 'compositions' ? 'var(--primary-color)' : 'var(--text-muted)',
                fontWeight: activeTab === 'compositions' ? 600 : 400,
                cursor: 'pointer',
                fontSize: '0.9375rem',
              }}
            >
              <i className="bi bi-layers" style={{ marginRight: '0.5rem' }}></i>
              Compositions
            </button>
          </div>
        </div>
        <div style={{ display: 'flex', gap: '0.75rem', flexWrap: 'wrap', alignItems: 'flex-start' }}>
          {activeTab === 'servers' && (
            <>
              <button type="button" className="btn btn-secondary" onClick={() => navigate('/import/openapi')}>
                <i className="bi bi-file-earmark-code"></i>
                Import OpenAPI
              </button>
              <button type="button" className="btn btn-primary" onClick={() => setShowCreateModal(true)}>
                <i className="bi bi-plus-lg"></i>
                New Server
              </button>
            </>
          )}
          {activeTab === 'compositions' && (
            <button
              type="button"
              className="btn btn-primary"
              onClick={() => setOpenCompositionForm(true)}
            >
              <i className="bi bi-plus-lg"></i>
              New Composition
            </button>
          )}
        </div>
      </div>

      {activeTab === 'compositions' && (
        <CompositionsTab
          servers={servers}
          compositions={compositions}
          loading={compositionsLoading}
          onRefresh={loadCompositions}
          openFormRequested={openCompositionForm}
          onFormOpened={() => setOpenCompositionForm(false)}
        />
      )}

      {activeTab === 'servers' && (
        <>
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

      {!loading && servers.length > 0 && (
        <div className="card" style={{ marginTop: '1.5rem', marginBottom: '1.5rem' }}>
          <button
            type="button"
            onClick={() => setShowTemplates(!showTemplates)}
            style={{
              display: 'flex',
              justifyContent: 'space-between',
              alignItems: 'center',
              width: '100%',
              padding: '0.5rem 0 0.75rem 0',
              border: 'none',
              background: 'transparent',
              cursor: 'pointer',
            }}
          >
            <div style={{ textAlign: 'left' }}>
              <h3 className="card-title" style={{ marginBottom: '0.1rem' }}>
                <i className="bi bi-lightning" style={{ marginRight: '0.5rem', color: 'var(--primary-color)' }}></i>
                Start from a template
              </h3>
              <p style={{ margin: 0, color: 'var(--text-secondary)', fontSize: '0.875rem' }}>
                Spin up a demo or starter server, then customize tools, resources, and policies.
              </p>
            </div>
            <i
              className={`bi ${showTemplates ? 'bi-chevron-up' : 'bi-chevron-down'}`}
              style={{ fontSize: '1rem', color: 'var(--text-muted)' }}
            ></i>
          </button>

          {showTemplates && (
            <div className="server-grid" style={{ marginTop: '0.75rem' }}>
              <div className="card" style={{ cursor: 'pointer' }} onClick={handleCreateDemoServer}>
              <div style={{ display: 'flex', alignItems: 'flex-start', gap: '1rem', marginBottom: '0.75rem' }}>
                <div style={{
                  width: '40px',
                  height: '40px',
                  borderRadius: '10px',
                  background: 'var(--primary-light)',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  fontSize: '1.25rem',
                  color: 'var(--primary-color)',
                  flexShrink: 0,
                }}>
                  <i className="bi bi-box-seam"></i>
                </div>
                <div>
                  <h4 style={{ margin: 0, fontSize: '0.95rem' }}>Demo API Toolkit</h4>
                  <p style={{ margin: '0.25rem 0 0 0', fontSize: '0.8rem', color: 'var(--text-secondary)' }}>
                    Fully configured demo server with location lookup, weather, jokes, GitHub, and more.
                  </p>
                </div>
              </div>
              <button className="btn btn-primary btn-sm" type="button">
                <i className="bi bi-plus-lg"></i>
                Use template
              </button>
            </div>

              <div className="card" style={{ cursor: 'pointer' }} onClick={handleCreateRestStarterServer}>
              <div style={{ display: 'flex', alignItems: 'flex-start', gap: '1rem', marginBottom: '0.75rem' }}>
                <div style={{
                  width: '40px',
                  height: '40px',
                  borderRadius: '10px',
                  background: 'var(--primary-light)',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  fontSize: '1.25rem',
                  color: 'var(--primary-color)',
                  flexShrink: 0,
                }}>
                  <i className="bi bi-globe"></i>
                </div>
                <div>
                  <h4 style={{ margin: 0, fontSize: '0.95rem' }}>REST API Starter</h4>
                  <p style={{ margin: '0.25rem 0 0 0', fontSize: '0.8rem', color: 'var(--text-secondary)' }}>
                    Clean server ready for REST tools. Add tools for your existing APIs in minutes.
                  </p>
                </div>
              </div>
              <button className="btn btn-outline-primary btn-sm" type="button">
                <i className="bi bi-plus-lg"></i>
                Use template
              </button>
            </div>
            </div>
          )}
        </div>
      )}

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
        </>
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
