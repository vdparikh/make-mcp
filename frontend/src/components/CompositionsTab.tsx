import { useState, useEffect } from 'react';
import { toast } from 'react-toastify';
import type { Server, ServerComposition } from '../types';
import DeployOptionsModal from './DeployOptionsModal';
import {
  createComposition,
  updateComposition,
  deleteComposition,
  compositionHostedDeploy,
  compositionHostedStatus,
  exportComposition,
} from '../services/api';

interface CompositionsTabProps {
  servers: Server[];
  compositions: ServerComposition[];
  loading: boolean;
  onRefresh: () => Promise<void>;
  /** When true, open the create form and then clear this (via onFormOpened) */
  openFormRequested?: boolean;
  onFormOpened?: () => void;
}

export default function CompositionsTab({
  servers,
  compositions,
  loading,
  onRefresh,
  openFormRequested,
  onFormOpened,
}: CompositionsTabProps) {
  const [showFormModal, setShowFormModal] = useState(false);
  useEffect(() => {
    if (openFormRequested) {
      setShowFormModal(true);
      onFormOpened?.();
    }
  }, [openFormRequested, onFormOpened]);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [selectedServers, setSelectedServers] = useState<string[]>([]);
  const [saving, setSaving] = useState(false);

  const [showDeployModal, setShowDeployModal] = useState(false);
  const [deployingComposition, setDeployingComposition] = useState<ServerComposition | null>(null);
  const [exporting, setExporting] = useState(false);

  const resetForm = () => {
    setName('');
    setDescription('');
    setSelectedServers([]);
    setEditingId(null);
  };

  const handleEdit = (comp: ServerComposition) => {
    setName(comp.name);
    setDescription(comp.description);
    setSelectedServers(comp.server_ids);
    setEditingId(comp.id);
    setShowFormModal(true);
  };

  const handleDelete = async (id: string) => {
    if (!confirm('Delete this composition?')) return;
    try {
      await deleteComposition(id);
      toast.success('Composition deleted');
      onRefresh();
    } catch {
      toast.error('Failed to delete composition');
    }
  };

  const openDeployModal = (composition: ServerComposition) => {
    setDeployingComposition(composition);
    setShowDeployModal(true);
  };

  const downloadCompositionZip = async (composition: ServerComposition) => {
    setExporting(true);
    try {
      const blob = await exportComposition(composition.id, {
        prefix_tool_names: false,
        merge_resources: true,
        merge_prompts: true,
      });
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `${composition.name?.toLowerCase().replace(/\s+/g, '-') || 'composition'}-mcp-server.zip`;
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      document.body.removeChild(a);
      toast.success('Composition exported successfully');
    } catch {
      toast.error('Failed to export composition');
    } finally {
      setExporting(false);
    }
  };

  const toggleServer = (serverId: string) => {
    if (selectedServers.includes(serverId)) {
      setSelectedServers(selectedServers.filter((id) => id !== serverId));
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
      if (editingId) {
        await updateComposition(editingId, { name, description, server_ids: selectedServers });
        toast.success('Composition updated');
      } else {
        await createComposition({ name, description, server_ids: selectedServers });
        toast.success('Composition created');
      }
      setShowFormModal(false);
      resetForm();
      onRefresh();
    } catch {
      toast.error(editingId ? 'Failed to update composition' : 'Failed to create composition');
    } finally {
      setSaving(false);
    }
  };

  const getServerName = (serverId: string) => servers.find((s) => s.id === serverId)?.name || serverId;

  if (loading) {
    return (
      <div className="loading" style={{ minHeight: '200px' }}>
        <div className="spinner"></div>
      </div>
    );
  }

  return (
    <>
      <div className="card" style={{ marginBottom: '1.5rem' }}>
        <h3 className="card-title" style={{ marginBottom: '0.75rem' }}>
          <i className="bi bi-layers" style={{ marginRight: '0.75rem', color: 'var(--primary-color)' }}></i>
          MCP Server Composition
        </h3>
        <p style={{ color: 'var(--text-secondary)', marginBottom: '1rem' }}>
          Compose multiple MCP servers into a single unified interface. This enables complex AI workflows that span multiple services.
        </p>
        <div
          style={{
            background: 'linear-gradient(135deg, rgba(129, 140, 248, 0.15), rgba(56, 189, 248, 0.08))',
            border: '1px solid rgba(129, 140, 248, 0.3)',
            borderRadius: '8px',
            padding: '1rem',
          }}
        >
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
          <p style={{ fontSize: '0.8125rem', color: 'var(--text-muted)', margin: 0 }}>
            AI workflow: Find lead (Salesforce) → Create invoice (Stripe) → Notify team (Slack)
          </p>
        </div>
      </div>

      {compositions.length === 0 && !showFormModal ? (
        <div className="empty-state">
          <i className="bi bi-layers"></i>
          <h3>No compositions yet</h3>
          <p>Combine multiple MCP servers into one unified interface</p>
          <button type="button" className="btn btn-primary" onClick={() => setShowFormModal(true)}>
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
                  <p className="card-description">{composition.description || 'No description'}</p>
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
                    {composition.server_ids.reduce(
                      (acc, id) => acc + (servers.find((s) => s.id === id)?.tools?.length || 0),
                      0
                    )}{' '}
                    Total Tools
                  </span>
                </div>
              </div>
              <div style={{ display: 'flex', gap: '0.5rem', marginTop: '1rem', paddingTop: '1rem', borderTop: '1px solid var(--card-border)' }}>
                <button type="button" className="btn btn-primary btn-sm" style={{ flex: 1 }} onClick={() => openDeployModal(composition)}>
                  <i className="bi bi-cloud-arrow-up"></i>
                  Deploy
                </button>
                <button type="button" className="btn btn-secondary btn-sm" onClick={() => handleEdit(composition)}>
                  <i className="bi bi-pencil"></i>
                </button>
                <button type="button" className="btn btn-secondary btn-sm" onClick={() => handleDelete(composition.id)}>
                  <i className="bi bi-trash"></i>
                </button>
              </div>
            </div>
          ))}
        </div>
      )}

      <DeployOptionsModal
        open={showDeployModal && !!deployingComposition}
        title={deployingComposition?.name || 'Composition'}
        artifactLabel="composition"
        downloading={exporting}
        onClose={() => setShowDeployModal(false)}
        onDownloadZip={() => downloadCompositionZip(deployingComposition!)}
        onHostedPublish={deployingComposition ? async () => compositionHostedDeploy(deployingComposition.id) : undefined}
        onHostedStatus={deployingComposition ? async () => compositionHostedStatus(deployingComposition.id) : undefined}
      />

      {showFormModal && (
        <div className="modal-overlay" onClick={() => { setShowFormModal(false); resetForm(); }}>
          <div className="modal-content" style={{ maxWidth: '900px' }} onClick={(e) => e.stopPropagation()}>
            <div className="modal-header">
              <h2 className="modal-title">{editingId ? 'Edit Composition' : 'Create New Composition'}</h2>
              <button
                type="button"
                className="btn btn-icon btn-secondary"
                onClick={() => { setShowFormModal(false); resetForm(); }}
              >
                <i className="bi bi-x-lg"></i>
              </button>
            </div>
            <form onSubmit={handleSubmit}>
              <div className="modal-body">
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
                      autoFocus
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
                  <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(220px, 1fr))', gap: '0.75rem' }}>
                    {servers.map((server) => (
                      <div
                        key={server.id}
                        role="button"
                        tabIndex={0}
                        onClick={() => toggleServer(server.id)}
                        onKeyDown={(e) => e.key === 'Enter' && toggleServer(server.id)}
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
                          <div
                            style={{
                              width: 20,
                              height: 20,
                              borderRadius: '4px',
                              border: `2px solid ${selectedServers.includes(server.id) ? 'var(--primary-color)' : 'var(--card-border)'}`,
                              background: selectedServers.includes(server.id) ? 'var(--primary-color)' : 'transparent',
                              display: 'flex',
                              alignItems: 'center',
                              justifyContent: 'center',
                            }}
                          >
                            {selectedServers.includes(server.id) && (
                              <i className="bi bi-check" style={{ color: 'white', fontSize: '0.75rem' }}></i>
                            )}
                          </div>
                          <div>
                            <div style={{ fontWeight: 500, color: 'var(--text-primary)' }}>{server.name}</div>
                            <div style={{ fontSize: '0.75rem', color: 'var(--text-secondary)' }}>{server.tools?.length || 0} tools</div>
                          </div>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              </div>
              <div className="modal-footer">
                <button type="button" className="btn btn-secondary" onClick={() => { setShowFormModal(false); resetForm(); }}>
                  Cancel
                </button>
                <button type="submit" className="btn btn-primary" disabled={saving || selectedServers.length < 2}>
                  {saving ? (editingId ? 'Updating...' : 'Creating...') : (editingId ? 'Update Composition' : 'Create Composition')}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </>
  );
}
