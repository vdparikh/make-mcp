import { useState, useEffect, useMemo } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import type { Server, ServerVersion, Tool, Resource, Prompt, SecurityScoreResult } from '../types';
import { listMarketplace, getMarketplaceServer, downloadMarketplaceServer, marketplaceHostedDeploy, marketplaceHostedStatus } from '../services/api';
import DeployOptionsModal from '../components/DeployOptionsModal';
import { useTryChat } from '../contexts/TryChatContext';

type InspectorTab = 'tools' | 'resources' | 'prompts' | 'versions' | 'security';

export default function Marketplace() {
  const [servers, setServers] = useState<Server[]>([]);
  const [loading, setLoading] = useState(true);
  const [selectedServer, setSelectedServer] = useState<Server | null>(null);
  const [versions, setVersions] = useState<ServerVersion[]>([]);
  const [securityScore, setSecurityScore] = useState<SecurityScoreResult | null>(null);
  const [showModal, setShowModal] = useState(false);
  const [showDeployModal, setShowDeployModal] = useState(false);
  const [deployServer, setDeployServer] = useState<Server | null>(null);
  const [downloading, setDownloading] = useState(false);
  const [inspectorTab, setInspectorTab] = useState<InspectorTab>('tools');
  const [selectedTool, setSelectedTool] = useState<Tool | null>(null);
  const [selectedResource, setSelectedResource] = useState<Resource | null>(null);
  const [selectedPrompt, setSelectedPrompt] = useState<Prompt | null>(null);
  const [searchQuery, setSearchQuery] = useState('');
  const [sortBy, setSortBy] = useState<'updated_desc' | 'downloads_desc' | 'name_asc'>('updated_desc');
  const navigate = useNavigate();
  const { openTryChat } = useTryChat();

  useEffect(() => {
    loadMarketplace();
  }, []);

  const loadMarketplace = async () => {
    try {
      const data = await listMarketplace();
      setServers(data);
    } catch (error) {
      console.error('Error loading marketplace:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleViewServer = async (server: Server) => {
    try {
      const data = await getMarketplaceServer(server.id);
      setSelectedServer(data.server);
      setVersions(data.versions);
      setSecurityScore(data.security_score || null);
      setInspectorTab('tools');
      setSelectedTool(data.server.tools?.[0] || null);
      setSelectedResource(null);
      setSelectedPrompt(null);
      setShowModal(true);
    } catch (error) {
      console.error('Error loading server details:', error);
    }
  };

  const serverSlug = (name: string) => name.replace(/\s+/g, '-').toLowerCase().replace(/[^a-z0-9-]/g, '');

  const handleDownload = async (serverId: string, serverName: string) => {
    setDownloading(true);
    try {
      const blob = await downloadMarketplaceServer(serverId);
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `${serverSlug(serverName)}-mcp-server.zip`;
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      document.body.removeChild(a);
    } catch (error) {
      console.error('Error downloading:', error);
    } finally {
      setDownloading(false);
    }
  };

  const openDeploy = (server: Server) => {
    setDeployServer(server);
    setShowDeployModal(true);
  };

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
    });
  };

  const renderSchemaProperties = (schema: Record<string, unknown>) => {
    const properties = schema?.properties as Record<string, { type?: string; description?: string }> | undefined;
    const required = (schema?.required as string[]) || [];
    
    if (!properties) return <span style={{ color: 'var(--text-muted)' }}>No parameters</span>;
    
    return (
      <div style={{ display: 'flex', flexDirection: 'column', gap: '0.5rem' }}>
        {Object.entries(properties).map(([key, value]) => (
          <div key={key} style={{ 
            padding: '0.5rem 0.75rem',
            background: 'var(--background-secondary)',
            borderRadius: '6px',
            fontSize: '0.85rem',
          }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
              <code style={{ 
                color: 'var(--primary-color)', 
                fontWeight: 500,
                background: 'transparent',
              }}>{key}</code>
              <span style={{ 
                color: 'var(--text-muted)',
                fontSize: '0.75rem',
                padding: '0.125rem 0.375rem',
                background: 'var(--card-border)',
                borderRadius: '4px',
              }}>{value.type || 'any'}</span>
              {required.includes(key) && (
                <span style={{ 
                  color: '#dc2626',
                  fontSize: '0.7rem',
                  fontWeight: 500,
                }}>required</span>
              )}
            </div>
            {value.description && (
              <div style={{ color: 'var(--text-secondary)', marginTop: '0.25rem', fontSize: '0.8rem' }}>
                {value.description}
              </div>
            )}
          </div>
        ))}
      </div>
    );
  };

  const filteredServers = useMemo(() => {
    const query = searchQuery.trim().toLowerCase();
    let items = servers.filter((server) => {
      if (!query) return true;
      return (
        server.name.toLowerCase().includes(query) ||
        (server.description || '').toLowerCase().includes(query)
      );
    });
    items = [...items].sort((a, b) => {
      if (sortBy === 'name_asc') return a.name.localeCompare(b.name);
      if (sortBy === 'downloads_desc') return (b.downloads || 0) - (a.downloads || 0);
      const left = new Date(a.published_at || a.updated_at).getTime();
      const right = new Date(b.published_at || b.updated_at).getTime();
      return right - left;
    });
    return items;
  }, [servers, searchQuery, sortBy]);

  return (
    <div className="dashboard">
      <div className="page-header">
        <div>
          <nav className="page-breadcrumb">
            <Link to="/" className="page-breadcrumb-link">
              Dashboard
            </Link>
            <span className="page-breadcrumb-sep">/</span>
            <span className="page-breadcrumb-current">Marketplace</span>
          </nav>
          <h1 className="page-title">
            <i className="bi bi-shop page-title-icon"></i>
            Marketplace
          </h1>
          <p className="page-subtitle">
            Browse, inspect, and deploy published MCP servers from the community
          </p>
        </div>
      </div>

      {!loading && servers.length > 0 && (
        <div className="card page-quick-actions-card" style={{ marginBottom: '1rem' }}>
          <div className="page-quick-actions-head">
            <div>
              <h3 className="card-title" style={{ margin: 0 }}>Marketplace browse</h3>
              <p className="page-quick-actions-subtitle">Find a server, inspect details, then deploy to hosted runtime.</p>
            </div>
            <button className="btn btn-secondary" onClick={() => navigate('/')}>
              <i className="bi bi-plus-lg"></i>
              Create your own server
            </button>
          </div>
          <div className="page-quick-actions-toolbar marketplace-toolbar-grid">
            <div className="form-group" style={{ marginBottom: 0 }}>
              <label className="form-label">Search</label>
              <input
                className="form-control"
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                placeholder="Search by server name or description"
              />
            </div>
            <div className="form-group" style={{ marginBottom: 0 }}>
              <label className="form-label">Sort</label>
              <select className="form-control" value={sortBy} onChange={(e) => setSortBy(e.target.value as typeof sortBy)}>
                <option value="updated_desc">Recently published</option>
                <option value="downloads_desc">Most downloaded</option>
                <option value="name_asc">Name (A-Z)</option>
              </select>
            </div>
          </div>
        </div>
      )}

      {!loading && filteredServers.length > 0 && (
        <div className="card marketplace-spotlight-card">
          <div className="marketplace-spotlight-content">
            <div className="marketplace-spotlight-kicker">Featured this week</div>
            <h3 className="marketplace-spotlight-title">{filteredServers[0].name}</h3>
            <p className="marketplace-spotlight-desc">{filteredServers[0].description || 'Community published MCP server ready to inspect and deploy.'}</p>
            <div className="marketplace-spotlight-actions">
              <button className="btn btn-secondary" onClick={() => handleViewServer(filteredServers[0])}>
                <i className="bi bi-eye"></i>
                Inspect
              </button>
              <button className="btn btn-primary" onClick={() => openDeploy(filteredServers[0])}>
                <i className="bi bi-cloud-arrow-up"></i>
                Deploy now
              </button>
            </div>
          </div>
          <div className="marketplace-spotlight-meta">
            <div>
              <strong>{filteredServers[0].tools?.length || 0}</strong>
              <span>Tools</span>
            </div>
            <div>
              <strong>{filteredServers[0].downloads || 0}</strong>
              <span>Downloads</span>
            </div>
            <div>
              <strong>v{filteredServers[0].latest_version || filteredServers[0].version}</strong>
              <span>Version</span>
            </div>
          </div>
        </div>
      )}

      {loading ? (
        <div className="loading">Loading marketplace...</div>
      ) : servers.length === 0 ? (
        <div className="empty-state">
          <i className="bi bi-shop" style={{ fontSize: '3rem', color: 'var(--text-muted)' }}></i>
          <h3>No published servers yet</h3>
          <p>Be the first to publish a server to the marketplace!</p>
          <button className="btn btn-primary" onClick={() => navigate('/')}>
            Go to Dashboard
          </button>
        </div>
      ) : filteredServers.length === 0 ? (
        <div className="empty-state">
          <i className="bi bi-search"></i>
          <h3>No marketplace servers match your filters</h3>
          <p>Try a different search or reset the sort.</p>
          <button className="btn btn-secondary" onClick={() => { setSearchQuery(''); setSortBy('updated_desc'); }}>
            Reset filters
          </button>
        </div>
      ) : (
        <div className="marketplace-grid">
          {filteredServers.map((server) => (
            <div key={server.id} className="marketplace-card">
              <div className="marketplace-card-header">
                <div className="marketplace-icon">
                  <i className={`bi ${server.icon || 'bi-server'}`}></i>
                </div>
                <div className="marketplace-badges">
                  <span className="badge badge-version">v{server.latest_version || server.version}</span>
                  {server.security_score != null && server.security_grade && (
                    <span
                      className="badge"
                      style={{
                        background: server.security_grade === 'A' ? '#dcfce7' : server.security_grade === 'B' ? '#dbeafe' : server.security_grade === 'C' ? '#fef9c3' : server.security_grade === 'D' ? '#ffedd5' : '#fee2e2',
                        color: server.security_grade === 'A' ? '#166534' : server.security_grade === 'B' ? '#1e40af' : server.security_grade === 'C' ? '#854d0e' : server.security_grade === 'D' ? '#c2410c' : '#991b1b',
                      }}
                      title="Security score (SlowMist checklist)"
                    >
                      <i className="bi bi-shield-lock" style={{ marginRight: '4px' }}></i>
                      {server.security_score}% {server.security_grade}
                    </span>
                  )}
                </div>
              </div>
              
              <div className="marketplace-card-body">
                <h3>{server.name}</h3>
                <p className="marketplace-description">{server.description || 'No description'}</p>
                
                <div className="marketplace-stats">
                  <div className="stat">
                    <i className="bi bi-download"></i>
                    <span>{server.downloads || 0}</span>
                  </div>
                  <div className="stat">
                    <i className="bi bi-tools"></i>
                    <span>{server.tools?.length || 0} tools</span>
                  </div>
                  <div className="stat">
                    <i className="bi bi-calendar"></i>
                    <span>{server.published_at ? formatDate(server.published_at) : 'N/A'}</span>
                  </div>
                </div>
              </div>
              
              <div className="marketplace-card-footer">
                <button 
                  className="btn btn-secondary"
                  onClick={() => handleViewServer(server)}
                >
                  <i className="bi bi-eye"></i> Inspect
                </button>
                <button
                  className="btn btn-secondary"
                  onClick={() => openTryChat({ type: 'marketplace', id: server.id, name: server.name })}
                >
                  <i className="bi bi-stars"></i> Try
                </button>
                <button
                  className="btn btn-primary"
                  onClick={() => openDeploy(server)}
                >
                  <i className="bi bi-cloud-arrow-up"></i> Deploy
                </button>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Inspector Modal */}
      {showModal && selectedServer && (
        <div 
          className="marketplace-inspector-overlay"
          onClick={() => setShowModal(false)}
        >
          <div 
            className="marketplace-inspector-modal"
            style={{
              backgroundColor: '#1e293b',
              borderRadius: '12px',
              width: '95%',
              maxWidth: '1100px',
              height: '85vh',
              overflow: 'hidden',
              display: 'flex',
              flexDirection: 'column',
              color: '#e2e8f0',
            }}
            onClick={(e) => e.stopPropagation()}
          >
            {/* Header */}
            <div className="marketplace-inspector-header" style={{ 
              padding: '1rem 1.5rem', 
              borderBottom: '1px solid #334155',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
              background: '#0f172a',
            }}>
              <div className="marketplace-inspector-header-main" style={{ display: 'flex', alignItems: 'center', gap: '1rem' }}>
                <div className="marketplace-inspector-icon" style={{
                  width: '40px',
                  height: '40px',
                  borderRadius: '10px',
                  backgroundColor: '#3b82f6',
                  color: 'white',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  fontSize: '1.25rem',
                }}>
                  <i className={`bi ${selectedServer.icon || 'bi-server'}`}></i>
                </div>
                <div>
                  <h2 style={{ margin: 0, fontSize: '1.25rem', color: '#f1f5f9' }}>{selectedServer.name}</h2>
                  <div style={{ display: 'flex', alignItems: 'center', gap: '0.75rem', marginTop: '0.25rem', flexWrap: 'wrap' }}>
                    <span style={{ 
                      background: '#22c55e',
                      color: '#052e16',
                      padding: '0.125rem 0.5rem',
                      borderRadius: '4px',
                      fontSize: '0.7rem',
                      fontWeight: 600,
                    }}>v{selectedServer.latest_version || selectedServer.version}</span>
                    {selectedServer.security_score != null && selectedServer.security_grade && (
                      <span style={{
                        background: selectedServer.security_grade === 'A' ? '#dcfce7' : selectedServer.security_grade === 'B' ? '#dbeafe' : selectedServer.security_grade === 'C' ? '#fef9c3' : selectedServer.security_grade === 'D' ? '#ffedd5' : '#fee2e2',
                        color: selectedServer.security_grade === 'A' ? '#166534' : selectedServer.security_grade === 'B' ? '#1e40af' : selectedServer.security_grade === 'C' ? '#854d0e' : selectedServer.security_grade === 'D' ? '#c2410c' : '#991b1b',
                        padding: '0.125rem 0.5rem',
                        borderRadius: '4px',
                        fontSize: '0.7rem',
                        fontWeight: 600,
                      }} title="Security score (SlowMist MCP checklist)">
                        <i className="bi bi-shield-lock" style={{ marginRight: '0.25rem' }}></i>
                        {selectedServer.security_score}% {selectedServer.security_grade}
                      </span>
                    )}
                    <span style={{ color: '#94a3b8', fontSize: '0.8rem' }}>
                      <i className="bi bi-download" style={{ marginRight: '0.25rem' }}></i>
                      {selectedServer.downloads || 0} downloads
                    </span>
                  </div>
                </div>
              </div>
              <div className="marketplace-inspector-header-actions" style={{ display: 'flex', alignItems: 'center', gap: '0.75rem' }}>
                <button
                  className="btn btn-secondary"
                  onClick={() => openTryChat({ type: 'marketplace', id: selectedServer.id, name: selectedServer.name })}
                  style={{ padding: '0.5rem 1rem' }}
                >
                  <i className="bi bi-stars"></i>
                  Try Chat
                </button>
                <button
                  className="btn btn-primary"
                  onClick={() => openDeploy(selectedServer)}
                  style={{ padding: '0.5rem 1rem' }}
                >
                  <i className="bi bi-cloud-arrow-up"></i>
                  Deploy
                </button>
                <button 
                  onClick={() => setShowModal(false)}
                  style={{
                    background: 'transparent',
                    border: 'none',
                    fontSize: '1.5rem',
                    cursor: 'pointer',
                    color: '#94a3b8',
                    padding: '0.25rem',
                  }}
                >
                  <i className="bi bi-x-lg"></i>
                </button>
              </div>
            </div>

            {/* Tabs */}
            <div className="marketplace-inspector-tabs" style={{ 
              display: 'flex', 
              gap: '0',
              background: '#0f172a',
              borderBottom: '1px solid #334155',
              padding: '0 1rem',
            }}>
              {[
                { id: 'tools' as InspectorTab, label: 'Tools', icon: 'bi-tools', count: selectedServer.tools?.length || 0 },
                { id: 'resources' as InspectorTab, label: 'Resources', icon: 'bi-folder', count: selectedServer.resources?.length || 0 },
                { id: 'prompts' as InspectorTab, label: 'Prompts', icon: 'bi-chat-text', count: selectedServer.prompts?.length || 0 },
                { id: 'versions' as InspectorTab, label: 'Versions', icon: 'bi-clock-history', count: versions.length },
                ...(securityScore ? [{ id: 'security' as InspectorTab, label: 'Security', icon: 'bi-shield-lock', count: securityScore.score }] : []),
              ].map(tab => (
                <button
                  key={tab.id}
                  onClick={() => {
                    setInspectorTab(tab.id);
                    if (tab.id === 'tools') setSelectedTool(selectedServer.tools?.[0] || null);
                    else if (tab.id === 'resources') setSelectedResource(selectedServer.resources?.[0] || null);
                    else if (tab.id === 'prompts') setSelectedPrompt(selectedServer.prompts?.[0] || null);
                  }}
                  style={{
                    padding: '0.875rem 1.25rem',
                    background: 'transparent',
                    border: 'none',
                    borderBottom: inspectorTab === tab.id ? '2px solid #3b82f6' : '2px solid transparent',
                    color: inspectorTab === tab.id ? '#f1f5f9' : '#94a3b8',
                    cursor: 'pointer',
                    display: 'flex',
                    alignItems: 'center',
                    gap: '0.5rem',
                    fontSize: '0.875rem',
                    fontWeight: 500,
                    transition: 'all 0.15s',
                  }}
                >
                  <i className={`bi ${tab.icon}`}></i>
                  {tab.label}
                  <span style={{
                    background: inspectorTab === tab.id ? '#3b82f6' : '#334155',
                    padding: '0.125rem 0.5rem',
                    borderRadius: '10px',
                    fontSize: '0.7rem',
                  }}>{tab.count}</span>
                </button>
              ))}
            </div>

            {/* Content */}
            <div className="marketplace-inspector-body" style={{ flex: 1, display: 'flex', overflow: 'hidden' }}>
              {/* Left Panel - List */}
              <div className="marketplace-inspector-list" style={{ 
                width: '320px', 
                borderRight: '1px solid #334155',
                overflowY: 'auto',
                background: '#1e293b',
              }}>
                {inspectorTab === 'tools' && (
                  selectedServer.tools?.length ? (
                    selectedServer.tools.map(tool => (
                      <div
                        key={tool.id}
                        className={`marketplace-inspector-item ${selectedTool?.id === tool.id ? 'active' : ''}`}
                        onClick={() => setSelectedTool(tool)}
                        style={{
                          padding: '1rem',
                          borderBottom: '1px solid #334155',
                          cursor: 'pointer',
                          background: selectedTool?.id === tool.id ? '#334155' : 'transparent',
                        }}
                      >
                        <div style={{ fontWeight: 500, color: '#f1f5f9', marginBottom: '0.25rem' }}>
                          {tool.name}
                        </div>
                        <div style={{ 
                          fontSize: '0.8rem', 
                          color: '#94a3b8',
                          display: '-webkit-box',
                          WebkitLineClamp: 2,
                          WebkitBoxOrient: 'vertical',
                          overflow: 'hidden',
                        }}>
                          {tool.description || 'No description'}
                        </div>
                      </div>
                    ))
                  ) : (
                    <div style={{ padding: '2rem', textAlign: 'center', color: '#64748b' }}>
                      No tools available
                    </div>
                  )
                )}

                {inspectorTab === 'resources' && (
                  selectedServer.resources?.length ? (
                    selectedServer.resources.map(resource => (
                      <div
                        key={resource.id}
                        className={`marketplace-inspector-item ${selectedResource?.id === resource.id ? 'active' : ''}`}
                        onClick={() => setSelectedResource(resource)}
                        style={{
                          padding: '1rem',
                          borderBottom: '1px solid #334155',
                          cursor: 'pointer',
                          background: selectedResource?.id === resource.id ? '#334155' : 'transparent',
                        }}
                      >
                        <div style={{ fontWeight: 500, color: '#f1f5f9', marginBottom: '0.25rem' }}>
                          {resource.name}
                        </div>
                        <div style={{ fontSize: '0.75rem', color: '#64748b', fontFamily: 'monospace' }}>
                          {resource.uri}
                        </div>
                      </div>
                    ))
                  ) : (
                    <div style={{ padding: '2rem', textAlign: 'center', color: '#64748b' }}>
                      No resources available
                    </div>
                  )
                )}

                {inspectorTab === 'prompts' && (
                  selectedServer.prompts?.length ? (
                    selectedServer.prompts.map(prompt => (
                      <div
                        key={prompt.id}
                        className={`marketplace-inspector-item ${selectedPrompt?.id === prompt.id ? 'active' : ''}`}
                        onClick={() => setSelectedPrompt(prompt)}
                        style={{
                          padding: '1rem',
                          borderBottom: '1px solid #334155',
                          cursor: 'pointer',
                          background: selectedPrompt?.id === prompt.id ? '#334155' : 'transparent',
                        }}
                      >
                        <div style={{ fontWeight: 500, color: '#f1f5f9', marginBottom: '0.25rem' }}>
                          {prompt.name}
                        </div>
                        <div style={{ 
                          fontSize: '0.8rem', 
                          color: '#94a3b8',
                          display: '-webkit-box',
                          WebkitLineClamp: 2,
                          WebkitBoxOrient: 'vertical',
                          overflow: 'hidden',
                        }}>
                          {prompt.description || 'No description'}
                        </div>
                      </div>
                    ))
                  ) : (
                    <div style={{ padding: '2rem', textAlign: 'center', color: '#64748b' }}>
                      No prompts available
                    </div>
                  )
                )}

                {inspectorTab === 'security' && securityScore && (
                  <div style={{ padding: '1rem', color: '#94a3b8', fontSize: '0.9rem' }}>
                    <i className="bi bi-shield-lock" style={{ marginRight: '0.5rem' }}></i>
                    Security score based on SlowMist MCP checklist. See details on the right.
                  </div>
                )}

                {inspectorTab === 'versions' && (
                  versions.length ? (
                    versions.map(version => (
                      <div
                        key={version.id}
                        className="marketplace-inspector-item"
                        style={{
                          padding: '1rem',
                          borderBottom: '1px solid #334155',
                        }}
                      >
                        <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', marginBottom: '0.25rem' }}>
                          <span style={{ 
                            fontWeight: 600, 
                            color: '#22c55e',
                          }}>v{version.version}</span>
                          {version.version === selectedServer.latest_version && (
                            <span style={{
                              background: '#22c55e',
                              color: '#052e16',
                              padding: '0.125rem 0.375rem',
                              borderRadius: '4px',
                              fontSize: '0.65rem',
                              fontWeight: 600,
                            }}>LATEST</span>
                          )}
                        </div>
                        {version.release_notes && (
                          <div style={{ fontSize: '0.8rem', color: '#94a3b8', marginBottom: '0.25rem' }}>
                            {version.release_notes}
                          </div>
                        )}
                        <div style={{ fontSize: '0.75rem', color: '#64748b' }}>
                          {formatDate(version.published_at)}
                        </div>
                      </div>
                    ))
                  ) : (
                    <div style={{ padding: '2rem', textAlign: 'center', color: '#64748b' }}>
                      No versions available
                    </div>
                  )
                )}
              </div>

              {/* Right Panel - Details */}
              <div className="marketplace-inspector-details" style={{ 
                flex: 1, 
                overflowY: 'auto',
                padding: '1.5rem',
                background: '#0f172a',
              }}>
                {inspectorTab === 'tools' && selectedTool && (
                  <div>
                    <h3 style={{ margin: '0 0 0.5rem', color: '#f1f5f9', fontSize: '1.25rem' }}>
                      {selectedTool.name}
                    </h3>
                    <p style={{ color: '#94a3b8', marginBottom: '1.5rem', fontSize: '0.9rem' }}>
                      {selectedTool.description || 'No description'}
                    </p>

                    <div style={{ marginBottom: '1.5rem' }}>
                      <div style={{ 
                        display: 'flex', 
                        alignItems: 'center', 
                        gap: '0.5rem',
                        marginBottom: '0.75rem',
                        color: '#94a3b8',
                        fontSize: '0.8rem',
                        textTransform: 'uppercase',
                        fontWeight: 600,
                      }}>
                        <i className="bi bi-box-arrow-in-right"></i>
                        Input Parameters
                      </div>
                      {renderSchemaProperties(selectedTool.input_schema)}
                    </div>

                    <div style={{ marginBottom: '1.5rem' }}>
                      <div style={{ 
                        display: 'flex', 
                        alignItems: 'center', 
                        gap: '0.5rem',
                        marginBottom: '0.75rem',
                        color: '#94a3b8',
                        fontSize: '0.8rem',
                        textTransform: 'uppercase',
                        fontWeight: 600,
                      }}>
                        <i className="bi bi-box-arrow-right"></i>
                        Output Schema
                      </div>
                      {selectedTool.output_schema && Object.keys(selectedTool.output_schema).length > 0 ? (
                        <pre style={{
                          background: '#1e293b',
                          padding: '1rem',
                          borderRadius: '8px',
                          fontSize: '0.8rem',
                          color: '#a5f3fc',
                          overflow: 'auto',
                          margin: 0,
                        }}>
                          {JSON.stringify(selectedTool.output_schema, null, 2)}
                        </pre>
                      ) : (
                        <span style={{ color: '#64748b', fontSize: '0.85rem' }}>No output schema defined</span>
                      )}
                    </div>

                    <div>
                      <div style={{ 
                        display: 'flex', 
                        alignItems: 'center', 
                        gap: '0.5rem',
                        marginBottom: '0.75rem',
                        color: '#94a3b8',
                        fontSize: '0.8rem',
                        textTransform: 'uppercase',
                        fontWeight: 600,
                      }}>
                        <i className="bi bi-gear"></i>
                        Execution Type
                      </div>
                      <span style={{
                        background: '#334155',
                        padding: '0.375rem 0.75rem',
                        borderRadius: '6px',
                        fontSize: '0.85rem',
                        color: '#f1f5f9',
                      }}>{selectedTool.execution_type}</span>
                    </div>
                  </div>
                )}

                {inspectorTab === 'security' && securityScore && (
                  <div>
                    <h3 style={{ margin: '0 0 0.5rem', color: '#f1f5f9', fontSize: '1.25rem' }}>
                      <i className="bi bi-shield-lock" style={{ marginRight: '0.5rem' }}></i>
                      Security Score
                    </h3>
                    <p style={{ color: '#94a3b8', marginBottom: '1rem', fontSize: '0.9rem' }}>
                      Based on the <a href={securityScore.checklist_url} target="_blank" rel="noopener noreferrer" style={{ color: '#60a5fa' }}>SlowMist MCP Security Checklist</a>
                    </p>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '1.5rem', marginBottom: '1.5rem' }}>
                      <div style={{
                        width: 56,
                        height: 56,
                        borderRadius: '50%',
                        background: securityScore.grade === 'A' ? '#16653422' : securityScore.grade === 'B' ? '#1e40af22' : securityScore.grade === 'C' ? '#854d0e22' : securityScore.grade === 'D' ? '#c2410c22' : '#991b1b22',
                        border: `2px solid ${securityScore.grade === 'A' ? '#16a34a' : securityScore.grade === 'B' ? '#2563eb' : securityScore.grade === 'C' ? '#ca8a04' : securityScore.grade === 'D' ? '#ea580c' : '#dc2626'}`,
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                        fontSize: '1.5rem',
                        fontWeight: 700,
                        color: securityScore.grade === 'A' ? '#16a34a' : securityScore.grade === 'B' ? '#2563eb' : securityScore.grade === 'C' ? '#ca8a04' : securityScore.grade === 'D' ? '#ea580c' : '#dc2626',
                      }}>
                        {securityScore.grade}
                      </div>
                      <div>
                        <div style={{ fontSize: '1.25rem', fontWeight: 700, color: '#f1f5f9' }}>{securityScore.score}%</div>
                        <div style={{ fontSize: '0.8rem', color: '#94a3b8' }}>{securityScore.earned} / {securityScore.max_points} points</div>
                      </div>
                    </div>
                    <h4 style={{ marginBottom: '0.75rem', fontSize: '0.95rem', color: '#e2e8f0' }}>Checklist criteria</h4>
                    <ul style={{ listStyle: 'none', padding: 0, margin: 0 }}>
                      {securityScore.criteria.map((c) => (
                        <li key={c.id} style={{ display: 'flex', alignItems: 'flex-start', gap: '0.5rem', padding: '0.5rem 0', borderBottom: '1px solid #334155' }}>
                          <i className={c.met ? 'bi bi-check-circle-fill' : 'bi bi-x-circle'} style={{ color: c.met ? '#16a34a' : '#64748b', marginTop: '2px', flexShrink: 0 }} />
                          <div>
                            <span style={{ fontWeight: 500, color: '#f1f5f9' }}>{c.name}</span>
                            <span style={{ marginLeft: '0.5rem', fontSize: '0.75rem', color: '#64748b' }}>({c.priority})</span>
                            {c.reason && !c.met && <div style={{ fontSize: '0.8rem', color: '#94a3b8', marginTop: '0.25rem' }}>{c.reason}</div>}
                          </div>
                        </li>
                      ))}
                    </ul>
                  </div>
                )}

                {inspectorTab === 'resources' && selectedResource && (
                  <div>
                    <h3 style={{ margin: '0 0 0.5rem', color: '#f1f5f9', fontSize: '1.25rem' }}>
                      {selectedResource.name}
                    </h3>
                    
                    <div style={{ marginBottom: '1.5rem' }}>
                      <div style={{ 
                        display: 'flex', 
                        alignItems: 'center', 
                        gap: '0.5rem',
                        marginBottom: '0.5rem',
                        color: '#94a3b8',
                        fontSize: '0.8rem',
                        textTransform: 'uppercase',
                        fontWeight: 600,
                      }}>
                        <i className="bi bi-link-45deg"></i>
                        URI
                      </div>
                      <code style={{
                        background: '#1e293b',
                        padding: '0.5rem 0.75rem',
                        borderRadius: '6px',
                        fontSize: '0.85rem',
                        color: '#a5f3fc',
                        display: 'block',
                      }}>{selectedResource.uri}</code>
                    </div>

                    <div style={{ marginBottom: '1.5rem' }}>
                      <div style={{ 
                        display: 'flex', 
                        alignItems: 'center', 
                        gap: '0.5rem',
                        marginBottom: '0.5rem',
                        color: '#94a3b8',
                        fontSize: '0.8rem',
                        textTransform: 'uppercase',
                        fontWeight: 600,
                      }}>
                        <i className="bi bi-file-earmark"></i>
                        MIME Type
                      </div>
                      <span style={{
                        background: '#334155',
                        padding: '0.375rem 0.75rem',
                        borderRadius: '6px',
                        fontSize: '0.85rem',
                        color: '#f1f5f9',
                      }}>{selectedResource.mime_type || 'text/plain'}</span>
                    </div>

                    {selectedResource.handler && Object.keys(selectedResource.handler).length > 0 && (
                      <div>
                        <div style={{ 
                          display: 'flex', 
                          alignItems: 'center', 
                          gap: '0.5rem',
                          marginBottom: '0.75rem',
                          color: '#94a3b8',
                          fontSize: '0.8rem',
                          textTransform: 'uppercase',
                          fontWeight: 600,
                        }}>
                          <i className="bi bi-code-slash"></i>
                          Handler Configuration
                        </div>
                        <pre style={{
                          background: '#1e293b',
                          padding: '1rem',
                          borderRadius: '8px',
                          fontSize: '0.8rem',
                          color: '#a5f3fc',
                          overflow: 'auto',
                          margin: 0,
                        }}>
                          {JSON.stringify(selectedResource.handler, null, 2)}
                        </pre>
                      </div>
                    )}
                  </div>
                )}

                {inspectorTab === 'prompts' && selectedPrompt && (
                  <div>
                    <h3 style={{ margin: '0 0 0.5rem', color: '#f1f5f9', fontSize: '1.25rem' }}>
                      {selectedPrompt.name}
                    </h3>
                    <p style={{ color: '#94a3b8', marginBottom: '1.5rem', fontSize: '0.9rem' }}>
                      {selectedPrompt.description || 'No description'}
                    </p>

                    <div style={{ marginBottom: '1.5rem' }}>
                      <div style={{ 
                        display: 'flex', 
                        alignItems: 'center', 
                        gap: '0.5rem',
                        marginBottom: '0.75rem',
                        color: '#94a3b8',
                        fontSize: '0.8rem',
                        textTransform: 'uppercase',
                        fontWeight: 600,
                      }}>
                        <i className="bi bi-chat-left-text"></i>
                        Template
                      </div>
                      <pre style={{
                        background: '#1e293b',
                        padding: '1rem',
                        borderRadius: '8px',
                        fontSize: '0.85rem',
                        color: '#fde68a',
                        overflow: 'auto',
                        margin: 0,
                        whiteSpace: 'pre-wrap',
                      }}>
                        {selectedPrompt.template}
                      </pre>
                    </div>

                    {selectedPrompt.arguments && Object.keys(selectedPrompt.arguments).length > 0 && (
                      <div>
                        <div style={{ 
                          display: 'flex', 
                          alignItems: 'center', 
                          gap: '0.5rem',
                          marginBottom: '0.75rem',
                          color: '#94a3b8',
                          fontSize: '0.8rem',
                          textTransform: 'uppercase',
                          fontWeight: 600,
                        }}>
                          <i className="bi bi-braces"></i>
                          Arguments Schema
                        </div>
                        {renderSchemaProperties(selectedPrompt.arguments)}
                      </div>
                    )}
                  </div>
                )}

                {inspectorTab === 'versions' && (
                  <div>
                    <h3 style={{ margin: '0 0 0.5rem', color: '#f1f5f9', fontSize: '1.25rem' }}>
                      Version History
                    </h3>
                    <p style={{ color: '#94a3b8', marginBottom: '1.5rem', fontSize: '0.9rem' }}>
                      {selectedServer.description}
                    </p>

                    <div style={{ 
                      display: 'grid', 
                      gridTemplateColumns: 'repeat(3, 1fr)', 
                      gap: '1rem',
                      marginBottom: '1.5rem',
                    }}>
                      <div style={{
                        background: '#1e293b',
                        padding: '1rem',
                        borderRadius: '8px',
                        textAlign: 'center',
                      }}>
                        <div style={{ fontSize: '1.5rem', fontWeight: '600', color: '#f1f5f9' }}>
                          {selectedServer.downloads || 0}
                        </div>
                        <div style={{ color: '#64748b', fontSize: '0.8rem' }}>Total Downloads</div>
                      </div>
                      <div style={{
                        background: '#1e293b',
                        padding: '1rem',
                        borderRadius: '8px',
                        textAlign: 'center',
                      }}>
                        <div style={{ fontSize: '1.5rem', fontWeight: '600', color: '#f1f5f9' }}>
                          {versions.length}
                        </div>
                        <div style={{ color: '#64748b', fontSize: '0.8rem' }}>Versions</div>
                      </div>
                      <div style={{
                        background: '#1e293b',
                        padding: '1rem',
                        borderRadius: '8px',
                        textAlign: 'center',
                      }}>
                        <div style={{ fontSize: '1.5rem', fontWeight: '600', color: '#22c55e' }}>
                          v{selectedServer.latest_version || selectedServer.version}
                        </div>
                        <div style={{ color: '#64748b', fontSize: '0.8rem' }}>Latest</div>
                      </div>
                    </div>

                    <div style={{ 
                      color: '#64748b', 
                      fontSize: '0.85rem',
                      display: 'flex',
                      alignItems: 'center',
                      gap: '0.5rem',
                    }}>
                      <i className="bi bi-calendar"></i>
                      First published {selectedServer.published_at ? formatDate(selectedServer.published_at) : 'N/A'}
                    </div>
                  </div>
                )}

                {inspectorTab === 'tools' && !selectedTool && selectedServer.tools?.length === 0 && (
                  <div style={{ textAlign: 'center', padding: '3rem', color: '#64748b' }}>
                    <i className="bi bi-tools" style={{ fontSize: '3rem', marginBottom: '1rem', display: 'block' }}></i>
                    <p>No tools available in this server</p>
                  </div>
                )}

                {inspectorTab === 'resources' && !selectedResource && selectedServer.resources?.length === 0 && (
                  <div style={{ textAlign: 'center', padding: '3rem', color: '#64748b' }}>
                    <i className="bi bi-folder" style={{ fontSize: '3rem', marginBottom: '1rem', display: 'block' }}></i>
                    <p>No resources available in this server</p>
                  </div>
                )}

                {inspectorTab === 'prompts' && !selectedPrompt && selectedServer.prompts?.length === 0 && (
                  <div style={{ textAlign: 'center', padding: '3rem', color: '#64748b' }}>
                    <i className="bi bi-chat-text" style={{ fontSize: '3rem', marginBottom: '1rem', display: 'block' }}></i>
                    <p>No prompts available in this server</p>
                  </div>
                )}
              </div>
            </div>
          </div>
        </div>
      )}
      <DeployOptionsModal
        open={showDeployModal && !!deployServer}
        title={deployServer?.name || 'Marketplace Server'}
        artifactLabel="marketplace server"
        downloading={downloading}
        onClose={() => setShowDeployModal(false)}
        onDownloadZip={() => handleDownload(deployServer!.id, deployServer!.name)}
        onHostedPublish={deployServer ? async () => {
          return marketplaceHostedDeploy(deployServer.id);
        } : undefined}
        onHostedStatus={deployServer ? async () => marketplaceHostedStatus(deployServer.id) : undefined}
      />
    </div>
  );
}
