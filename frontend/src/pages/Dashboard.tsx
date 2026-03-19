import { useState, useEffect, useMemo, useRef } from 'react';
import { Link, useNavigate, useSearchParams } from 'react-router-dom';
import { toast } from 'react-toastify';
import type { Server, ServerComposition } from '../types';
import { listServers, listCompositions, createServer, createDemoServer, deleteServer } from '../services/api';
import CreateServerModal from '../components/CreateServerModal';
import CompositionsTab from '../components/CompositionsTab';
import { useAuth } from '../contexts/AuthContext';
import { useTryChat } from '../contexts/TryChatContext';
import ConfirmModal from '../components/ConfirmModal';

type Tab = 'servers' | 'compositions';

export default function Dashboard() {
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const tabParam = searchParams.get('tab');
  const activeTab: Tab = tabParam === 'compositions' ? 'compositions' : 'servers';

  const { user, token } = useAuth();
  const { openTryChat } = useTryChat();
  const [servers, setServers] = useState<Server[]>([]);
  const [compositions, setCompositions] = useState<ServerComposition[]>([]);
  const [loading, setLoading] = useState(true);
  const [compositionsLoading, setCompositionsLoading] = useState(false);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [deleteServerId, setDeleteServerId] = useState<string | null>(null);
  const [openCompositionForm, setOpenCompositionForm] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const [statusFilter, setStatusFilter] = useState<'all' | 'draft' | 'published' | 'archived' | 'hosted_running'>('all');
  const [sortBy, setSortBy] = useState<'updated_desc' | 'name_asc' | 'tools_desc'>('updated_desc');
  const [hostedRuntimeByServer, setHostedRuntimeByServer] = useState<Record<string, { running: boolean; health?: string }>>({});
  const [templateMenuOpen, setTemplateMenuOpen] = useState(false);
  const templateMenuRef = useRef<HTMLDivElement | null>(null);

  const mapHostedRuntimeFromServers = (items: Server[]): Record<string, { running: boolean; health?: string }> => {
    const next: Record<string, { running: boolean; health?: string }> = {};
    (items || []).forEach((s) => {
      if (s.hosted_running) {
        next[s.id] = { running: true };
      }
    });
    return next;
  };

  const setTab = (t: Tab) => {
    if (t === 'compositions') setSearchParams({ tab: 'compositions' });
    else setSearchParams({});
  };

  // Load servers whenever auth changes
  useEffect(() => {
    if (!token || !user?.id) {
      setServers([]);
      setHostedRuntimeByServer({});
      setLoading(false);
      return;
    }
    let cancelled = false;
    setLoading(true);
    listServers()
      .then((serverData) => {
        if (cancelled) return;
        const safeServers = serverData ?? [];
        setServers(safeServers);
        setHostedRuntimeByServer(mapHostedRuntimeFromServers(safeServers));
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
      const safeServers = data ?? [];
      setServers(safeServers);
      setHostedRuntimeByServer(mapHostedRuntimeFromServers(safeServers));
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

  const filteredServers = useMemo(() => {
    const query = searchQuery.trim().toLowerCase();
    let items = servers.filter((s) => {
      if (statusFilter === 'hosted_running') {
        return hostedRuntimeByServer[s.id]?.running === true;
      }
      if (statusFilter !== 'all' && (s.status || 'draft') !== statusFilter) {
        return false;
      }
      if (!query) return true;
      return (
        s.name.toLowerCase().includes(query) ||
        (s.description || '').toLowerCase().includes(query)
      );
    });
    items = [...items].sort((a, b) => {
      if (sortBy === 'name_asc') return a.name.localeCompare(b.name);
      if (sortBy === 'tools_desc') return (b.tools?.length || 0) - (a.tools?.length || 0);
      return new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime();
    });
    return items;
  }, [servers, hostedRuntimeByServer, searchQuery, sortBy, statusFilter]);

  const newestServer = useMemo(() => {
    if (!servers.length) return null;
    return [...servers].sort((a, b) => new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime())[0];
  }, [servers]);

  const publishedCount = servers.filter((s) => s.status === 'published').length;
  const hostedCount = servers.filter((s) => hostedRuntimeByServer[s.id]?.running).length;

  useEffect(() => {
    if (!templateMenuOpen) return;

    const onPointerDown = (event: PointerEvent) => {
      if (!templateMenuRef.current) return;
      if (!templateMenuRef.current.contains(event.target as Node)) {
        setTemplateMenuOpen(false);
      }
    };
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') setTemplateMenuOpen(false);
    };

    window.addEventListener('pointerdown', onPointerDown);
    window.addEventListener('keydown', onKeyDown);
    return () => {
      window.removeEventListener('pointerdown', onPointerDown);
      window.removeEventListener('keydown', onKeyDown);
    };
  }, [templateMenuOpen]);

  const renderPrimaryActions = (className = 'dashboard-action-row') => (
    <div className={className}>
      <button type="button" className="btn btn-primary dashboard-action-btn" onClick={() => setShowCreateModal(true)}>
        <i className="bi bi-plus-lg"></i>
        New Server
      </button>
      <div className="dashboard-template-dropdown" ref={templateMenuRef}>
        <button
          type="button"
          className="btn btn-secondary dashboard-action-btn"
          onClick={() => setTemplateMenuOpen((prev) => !prev)}
          aria-haspopup="menu"
          aria-expanded={templateMenuOpen}
        >
          <i className="bi bi-box-seam"></i>
          Start with Template
          <i className="bi bi-chevron-down dashboard-template-caret"></i>
        </button>
        {templateMenuOpen && (
          <div className="dashboard-template-menu" role="menu">
            <button
              type="button"
              className="dashboard-template-menu-item"
              onClick={() => {
                setTemplateMenuOpen(false);
                handleCreateDemoServer();
              }}
            >
              <i className="bi bi-box-seam"></i>
              Demo Template
            </button>
            <button
              type="button"
              className="dashboard-template-menu-item"
              onClick={() => {
                setTemplateMenuOpen(false);
                handleCreateRestStarterServer();
              }}
            >
              <i className="bi bi-globe"></i>
              REST Template
            </button>
          </div>
        )}
      </div>
      <button type="button" className="btn btn-outline-primary dashboard-action-btn" onClick={() => navigate('/import/openapi')}>
        <i className="bi bi-file-earmark-code"></i>
        Import OpenAPI
      </button>
    </div>
  );

  return (
    <div>
      <div className="page-header" style={{ alignItems: 'flex-start' }}>
        <div style={{ flex: 1, minWidth: 0 }}>
          <nav className="page-breadcrumb">
            <span className="page-breadcrumb-current">Dashboard</span>
          </nav>
          <h1 className="page-title">
            <i className="bi bi-server page-title-icon"></i>
            MCP Servers
          </h1>
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
      <div className="card dashboard-command-center">
        <div className="dashboard-command-center-head">
          <div>
            <h3 className="card-title" style={{ margin: 0 }}>Workspace command center</h3>
            <p className="dashboard-command-center-subtitle">
              Build faster with one-click starter actions and jump back into your latest server.
            </p>
          </div>
          <div className="dashboard-command-center-pills">
            <span className="dashboard-cc-pill">{servers.length} total</span>
            <span className="dashboard-cc-pill">{publishedCount} published</span>
            <span className="dashboard-cc-pill">{hostedCount} hosted live</span>
          </div>
        </div>
        {newestServer && (
          <div className="dashboard-command-center-latest">
            <div>
              <div className="dashboard-command-center-label">Continue where you left off</div>
              <div className="dashboard-command-center-name">{newestServer.name}</div>
            </div>
            <Link to={`/servers/${newestServer.id}`} className="btn btn-secondary btn-sm">
              <i className="bi bi-arrow-right-circle"></i>
              Open latest
            </Link>
          </div>
        )}
      </div>

      {!loading && servers.length > 0 && (
        <div className="card dashboard-primary-actions-card">
          {renderPrimaryActions()}
        </div>
      )}

      <div className="dashboard-kpi-strip" role="status" aria-label="Server metrics">
        <div className="dashboard-kpi-item">
          <span className="dashboard-kpi-label">Servers</span>
          <span className="dashboard-kpi-value">{servers.length}</span>
        </div>
        <div className="dashboard-kpi-item">
          <span className="dashboard-kpi-label">Published</span>
          <span className="dashboard-kpi-value">{servers.filter((s) => s.status === 'published').length}</span>
        </div>
        <div className="dashboard-kpi-item">
          <span className="dashboard-kpi-label">Tools</span>
          <span className="dashboard-kpi-value">{servers.reduce((acc, s) => acc + (s.tools?.length || 0), 0)}</span>
        </div>
        <div className="dashboard-kpi-item">
          <span className="dashboard-kpi-label">Hosted live</span>
          <span className="dashboard-kpi-value">{servers.filter((s) => hostedRuntimeByServer[s.id]?.running).length}</span>
        </div>
      </div>

      {!loading && servers.length > 0 && (
        <div className="card dashboard-controls-card">
          <div className="dashboard-controls-grid">
            <div className="form-group" style={{ marginBottom: 0 }}>
              <label className="form-label">Search</label>
              <input
                className="form-control"
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                placeholder="Search by name or description"
              />
            </div>
            <div className="form-group" style={{ marginBottom: 0 }}>
              <label className="form-label">Status</label>
              <select className="form-control" value={statusFilter} onChange={(e) => setStatusFilter(e.target.value as typeof statusFilter)}>
                <option value="all">All</option>
                <option value="draft">Draft</option>
                <option value="published">Published</option>
                <option value="archived">Archived</option>
                <option value="hosted_running">Hosted Running</option>
              </select>
            </div>
            <div className="form-group" style={{ marginBottom: 0 }}>
              <label className="form-label">Sort</label>
              <select className="form-control" value={sortBy} onChange={(e) => setSortBy(e.target.value as typeof sortBy)}>
                <option value="updated_desc">Recently updated</option>
                <option value="name_asc">Name (A-Z)</option>
                <option value="tools_desc">Most tools</option>
              </select>
            </div>
          </div>
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
          <p>Create your first MCP server, start from a template, or import from OpenAPI.</p>
          {renderPrimaryActions('dashboard-action-row dashboard-action-row-center')}
        </div>
      ) : filteredServers.length === 0 ? (
        <div className="empty-state">
          <i className="bi bi-search"></i>
          <h3>No servers match your filters</h3>
          <p>Try clearing search or selecting a different status/sort.</p>
          <button className="btn btn-secondary" onClick={() => { setSearchQuery(''); setStatusFilter('all'); setSortBy('updated_desc'); }}>
            Reset filters
          </button>
        </div>
      ) : (
        <div className="server-grid">
          {filteredServers.map((server) => (
            <div className="card dashboard-server-card" key={server.id}>
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
                  {hostedRuntimeByServer[server.id]?.running && (
                    <div className="dashboard-hosted-pill">
                      <span className="dashboard-hosted-dot" />
                      Hosted running {hostedRuntimeByServer[server.id]?.health ? `· ${hostedRuntimeByServer[server.id]?.health}` : ''}
                    </div>
                  )}
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
                    onClick={() => openTryChat({ type: 'server', id: server.id, name: server.name })}
                    data-tooltip="Try Chat"
                  >
                    <i className="bi bi-stars"></i>
                  </button>
                  <button 
                    className="btn btn-icon btn-secondary btn-sm"
                    onClick={() => setDeleteServerId(server.id)}
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
      <ConfirmModal
        open={!!deleteServerId}
        title="Delete server?"
        message="This permanently removes the server and its local configuration."
        confirmLabel="Delete"
        danger
        onCancel={() => setDeleteServerId(null)}
        onConfirm={async () => {
          if (!deleteServerId) return;
          await handleDeleteServer(deleteServerId);
          setDeleteServerId(null);
        }}
      />
    </div>
  );
}
