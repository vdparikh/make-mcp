import { useEffect, useMemo, useState } from 'react';
import { toast } from 'react-toastify';
import type { HostedStatusResponse } from '../services/api';

type DeployType = 'hosted' | 'nodejs' | 'docker' | 'github' | 'azure' | null;

interface DeployOptionsModalProps {
  open: boolean;
  title: string;
  artifactLabel?: string;
  downloading?: boolean;
  onClose: () => void;
  onDownloadZip: () => Promise<void>;
  onHostedPublish?: () => Promise<HostedStatusResponse>;
  onHostedStatus?: () => Promise<HostedStatusResponse>;
}

export default function DeployOptionsModal({
  open,
  title,
  artifactLabel = 'MCP server',
  downloading = false,
  onClose,
  onDownloadZip,
  onHostedPublish,
  onHostedStatus,
}: DeployOptionsModalProps) {
  const [selectedDeploy, setSelectedDeploy] = useState<DeployType>('hosted');
  const [deployTargetEnv, setDeployTargetEnv] = useState('');
  const [publishingHosted, setPublishingHosted] = useState(false);
  const [hostedStatusLoading, setHostedStatusLoading] = useState(false);
  const [hostedResult, setHostedResult] = useState<HostedStatusResponse | null>(null);
  const [showManifest, setShowManifest] = useState(false);

  const hostedSupported = useMemo(() => typeof onHostedPublish === 'function', [onHostedPublish]);

  useEffect(() => {
    if (!open) return;
    setSelectedDeploy('hosted');
    setShowManifest(false);
  }, [open, title]);

  useEffect(() => {
    if (!open || typeof onHostedStatus !== 'function') return;
    let cancelled = false;
    setHostedStatusLoading(true);
    onHostedStatus()
      .then((status) => {
        if (cancelled) return;
        setHostedResult(status || null);
      })
      .catch(() => {
        if (!cancelled) setHostedResult(null);
      })
      .finally(() => {
        if (!cancelled) setHostedStatusLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [open, onHostedStatus, title]);

  if (!open) return null;

  const handleHostedPublish = async () => {
    if (!onHostedPublish) return;
    try {
      setPublishingHosted(true);
      const result = await onHostedPublish();
      setHostedResult(result || {});
      toast.success('Hosted MCP published');
    } catch (err: unknown) {
      const message =
        typeof err === 'object' && err !== null && 'response' in err
          ? (err as { response?: { data?: { error?: string } } }).response?.data?.error
          : undefined;
      toast.error(message || 'Hosted publish failed');
    } finally {
      setPublishingHosted(false);
    }
  };

  const handleDownload = async () => {
    try {
      await onDownloadZip();
    } catch {
      // Page handlers already surface meaningful toast messages.
    }
  };

  const formatRuntimeTime = (value?: string) => {
    if (!value) return '—';
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) return '—';
    return date.toLocaleString();
  };
  const artifactSlug = title
    .toLowerCase()
    .replace(/\s+/g, '-')
    .replace(/[^a-z0-9-]/g, '')
    .replace(/-+/g, '-')
    .replace(/^-|-$/g, '') || 'mcp-server';
  const nodeClientConfig = JSON.stringify(
    {
      mcpServers: {
        [artifactSlug]: {
          command: 'node',
          args: ['/path/to/your-server/run-with-log.mjs'],
        },
      },
    },
    null,
    2
  );
  const dockerClientConfig = `{
  "mcpServers": {
    "${artifactSlug}": {
      "command": "docker",
      "args": ["run", "-i", "--rm", "${artifactSlug}-mcp"]
    }
  }
}`;
  const copyText = async (value: string, successMsg: string) => {
    try {
      await navigator.clipboard.writeText(value);
      toast.success(successMsg);
    } catch {
      toast.error('Could not copy');
    }
  };

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal-content deploy-options-modal" style={{ maxWidth: '80%', maxHeight: '90vh' }} onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <h3 className="modal-title">
            <i className="bi bi-rocket-takeoff" style={{ marginRight: '0.5rem' }} />
            Deploy {title}
          </h3>
          <button type="button" className="btn btn-sm btn-outline-secondary" onClick={onClose}>
            <i className="bi bi-x-lg" />
          </button>
        </div>
        <div className="modal-body">
          <div className="card" style={{ marginBottom: '1.5rem' }}>
            <p style={{ color: 'var(--text-secondary)', marginBottom: '1rem', marginTop: 0 }}>
              Choose how you want to deploy this {artifactLabel}. Options match the main server deployment flow.
            </p>
            <div className="form-group" style={{ marginBottom: '1.5rem' }}>
              <label className="form-label" style={{ fontWeight: 600 }}>
                <i className="bi bi-cloud" style={{ marginRight: '0.5rem', color: 'var(--primary-color)' }} />
                Target environment
              </label>
              <select
                className="form-control w-100"
                value={deployTargetEnv}
                onChange={(e) => setDeployTargetEnv(e.target.value)}
              >
                <option value="">Use .env at runtime (no profile baked in)</option>
                <option value="dev">Dev</option>
                <option value="staging">Staging</option>
                <option value="prod">Prod</option>
              </select>
              <p style={{ fontSize: '0.75rem', color: 'var(--text-muted)', marginTop: '0.25rem', marginBottom: 0 }}>
                For marketplace and compositions, hosted deploy uses managed runtime settings; ZIP deploy keeps runtime configurable via env.
              </p>
            </div>
            <div className="deploy-option-grid">
            {[
              { id: 'hosted' as const, title: 'Publish MCP', subtitle: 'Hosted at URL', icon: 'bi-globe', color: '#8b5cf6' },
              { id: 'nodejs' as const, title: 'Node.js', subtitle: 'Download & run locally', icon: 'bi-filetype-js', color: 'var(--primary-color)' },
              { id: 'docker' as const, title: 'Docker', subtitle: 'Containerized deployment', icon: 'bi-box-seam', color: 'var(--success-color)' },
              { id: 'github' as const, title: 'GitHub', subtitle: 'Push to repository', icon: 'bi-github', color: '#24292f' },
              { id: 'azure' as const, title: 'Deploy to Cloud', subtitle: 'Deploy to cloud', icon: 'bi-cloud-upload', color: '#0078d4' },
            ].map((opt) => (
              <button
                key={opt.id}
                type="button"
                onClick={() => setSelectedDeploy(opt.id)}
                className={`deploy-option-card ${selectedDeploy === opt.id ? 'selected' : ''}`}
                style={{
                  ['--deploy-color' as string]: opt.color,
                  padding: '1.5rem',
                  background:
                    selectedDeploy === opt.id
                      ? opt.id === 'nodejs'
                        ? 'var(--primary-light)'
                        : opt.id === 'docker'
                          ? 'rgba(16, 185, 129, 0.1)'
                          : opt.id === 'github'
                            ? 'rgba(36, 41, 47, 0.1)'
                            : opt.id === 'azure'
                              ? 'rgba(0, 120, 212, 0.1)'
                              : 'rgba(139, 92, 246, 0.1)'
                      : 'var(--dark-bg)',
                  border: `2px solid ${selectedDeploy === opt.id ? opt.color : 'var(--card-border)'}`,
                  borderRadius: '12px',
                  cursor: 'pointer',
                  textAlign: 'center',
                }}
              >
                <div className="deploy-option-icon" style={{ color: selectedDeploy === opt.id ? opt.color : 'var(--text-secondary)' }}>
                  <i className={`bi ${opt.icon}`} />
                </div>
                <h4 style={{ marginBottom: '0.25rem', color: 'var(--text-primary)', fontSize: '1rem' }}>{opt.title}</h4>
                <p style={{ color: 'var(--text-muted)', fontSize: '0.8125rem', margin: 0 }}>{opt.subtitle}</p>
              </button>
            ))}
            </div>
          </div>

          {selectedDeploy === 'hosted' && (
            <div className="card">
              <h3 className="card-title" style={{ marginBottom: '0.25rem' }}>
                <i className="bi bi-globe" style={{ marginRight: '0.75rem', color: '#8b5cf6' }} />
                Publish MCP
              </h3>
              {hostedSupported ? (
                <>
                  <p style={{ color: 'var(--text-secondary)', marginTop: 0, marginBottom: '0.75rem' }}>
                    Publish this {artifactLabel} to the platform. It will be available at a URL you can add to Cursor/Claude Desktop.
                  </p>
                  <div className={`hosted-status-banner ${hostedStatusLoading ? 'loading' : hostedResult?.running ? 'running' : 'idle'}`}>
                    {hostedStatusLoading ? (
                      <span>
                        <i className="bi bi-arrow-repeat" style={{ marginRight: '0.5rem' }} />
                        Checking hosted runtime status...
                      </span>
                    ) : hostedResult?.running ? (
                      <span>
                        <span className="hosted-running-dot" />
                        <i className="bi bi-check-circle" style={{ marginRight: '0.5rem' }} />
                        Already running.
                      </span>
                    ) : (
                      <span>
                        <i className="bi bi-info-circle" style={{ marginRight: '0.5rem' }} />
                        No hosted runtime currently running.
                      </span>
                    )}
                  </div>
                  <button className="btn btn-primary" onClick={handleHostedPublish} disabled={publishingHosted} style={{ marginBottom: '0.75rem' }}>
                    {publishingHosted ? (
                      <>
                        <i className="bi bi-hourglass-split" /> Publishing...
                      </>
                    ) : (
                      <>
                        <i className="bi bi-globe" /> Publish MCP
                      </>
                    )}
                  </button>
                  {hostedResult?.endpoint && (
                    <div style={{ marginTop: '0.25rem', fontSize: '0.88rem' }}>
                      <div style={{ color: 'var(--text-muted)', marginBottom: '0.25rem' }}>Endpoint</div>
                      <code style={{ whiteSpace: 'pre-wrap' }}>{hostedResult.endpoint}</code>
                    </div>
                  )}
                  {hostedResult?.running && (
                    <div style={{ marginTop: '0.75rem', fontSize: '0.88rem', color: 'var(--text-secondary)' }}>
                      <div style={{ display: 'flex', gap: '1rem', flexWrap: 'wrap' }}>
                        <span><strong>Snapshot:</strong> {hostedResult.snapshot_version || hostedResult.version || '—'}</span>
                        <span><strong>Started:</strong> {formatRuntimeTime(hostedResult.started_at)}</span>
                        <span><strong>Last ensured:</strong> {formatRuntimeTime(hostedResult.last_ensured_at)}</span>
                        <span><strong>Container:</strong> {hostedResult.container_id ? hostedResult.container_id.slice(0, 12) : '—'}</span>
                        <span><strong>Port:</strong> {hostedResult.host_port || '—'}</span>
                      </div>
                    </div>
                  )}
                  {hostedResult?.mcp_config && (
                    <div style={{ marginTop: '0.75rem', fontSize: '0.88rem' }}>
                      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '0.25rem' }}>
                        <div style={{ color: 'var(--text-muted)' }}>MCP config (for your IDE)</div>
                        <button type="button" className="btn btn-sm btn-outline-primary" onClick={() => copyText(hostedResult.mcp_config || '', 'MCP config copied')}>
                          <i className="bi bi-clipboard" /> Copy config
                        </button>
                      </div>
                      <pre style={{ margin: 0, maxHeight: '200px', overflow: 'auto' }}>{hostedResult.mcp_config}</pre>
                    </div>
                  )}
                  {hostedResult?.manifest && (
                    <div style={{ marginTop: '0.75rem', fontSize: '0.88rem' }}>
                      {typeof hostedResult.manifest === 'object' && hostedResult.manifest !== null && (
                        <div style={{ color: 'var(--text-secondary)', marginBottom: '0.5rem' }}>
                          <strong>Build details:</strong>{' '}
                          Runtime {String((hostedResult.manifest as Record<string, unknown>).runtime || 'docker')}
                          {' · '}
                          Image {String((hostedResult.manifest as Record<string, unknown>).image || 'node:20-alpine')}
                          {' · '}
                          Snapshot {String((hostedResult.manifest as Record<string, unknown>).snapshot_version || hostedResult.snapshot_version || hostedResult.version || '—')}
                        </div>
                      )}
                      <button
                        type="button"
                        className="btn btn-sm btn-outline-secondary"
                        style={{ padding: '0.3rem 0.55rem' }}
                        onClick={() => setShowManifest((v) => !v)}
                      >
                        {showManifest ? 'Hide Manifest' : 'Show Manifest'}
                      </button>
                      {typeof hostedResult.manifest === 'object' && hostedResult.manifest !== null && (
                        <div style={{ marginTop: '0.5rem', fontSize: '0.8125rem', color: 'var(--text-secondary)' }}>
                          {'composition_options' in hostedResult.manifest && (
                            <span>
                              <strong>Composition options:</strong>{' '}
                              {JSON.stringify((hostedResult.manifest as Record<string, unknown>).composition_options)}
                            </span>
                          )}
                        </div>
                      )}
                      {showManifest && (
                        <pre style={{ margin: '0.5rem 0 0', maxHeight: '260px', overflow: 'auto' }}>
                          {JSON.stringify(hostedResult.manifest, null, 2)}
                        </pre>
                      )}
                    </div>
                  )}
                </>
              ) : (
                <p style={{ color: 'var(--text-muted)' }}>
                  Hosted publish is not available for this item yet. Use Node.js/Docker deployment packages below.
                </p>
              )}
            </div>
          )}

          {(selectedDeploy === 'nodejs' || selectedDeploy === 'docker') && (
            <div className="card">
              <h4 className="card-title" style={{ marginBottom: '0.5rem' }}>
                {selectedDeploy === 'nodejs' ? 'Node.js Deployment' : 'Docker Deployment'}
              </h4>
              <p style={{ color: 'var(--text-secondary)', marginTop: 0, marginBottom: '1rem' }}>
                Download and run your {artifactLabel} package.
              </p>
              <div style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: '1rem' }}>
                <button className="btn btn-primary" onClick={handleDownload} disabled={downloading}>
                  <i className="bi bi-download" />
                  {downloading ? ' Generating...' : ' Download ZIP'}
                </button>
              </div>
              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '1.5rem' }}>
                <div>
                  <h4 style={{ fontSize: '0.875rem', fontWeight: 600, marginBottom: '0.75rem' }}>
                    <span style={{ display: 'inline-flex', alignItems: 'center', justifyContent: 'center', width: 24, height: 24, borderRadius: '50%', background: selectedDeploy === 'docker' ? 'var(--success-color)' : 'var(--primary-color)', color: '#fff', fontSize: '0.75rem', marginRight: '0.5rem' }}>1</span>
                    Setup &amp; Run
                  </h4>
                  <pre style={{ background: '#1a1a2e', padding: '1rem', borderRadius: '8px', fontSize: '0.8125rem', color: '#e5e7eb', margin: 0 }}>
{selectedDeploy === 'docker'
  ? `cd ${artifactSlug}-mcp-server

# Copy env template
cp .env.example .env

# Start with compose
docker-compose up -d`
  : `cd ${artifactSlug}-mcp-server
npm install
npm run build
npm start`}
                  </pre>
                </div>
                <div>
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '0.5rem' }}>
                    <h4 style={{ fontSize: '0.875rem', fontWeight: 600, margin: 0 }}>
                      <span style={{ display: 'inline-flex', alignItems: 'center', justifyContent: 'center', width: 24, height: 24, borderRadius: '50%', background: selectedDeploy === 'docker' ? 'var(--success-color)' : 'var(--primary-color)', color: '#fff', fontSize: '0.75rem', marginRight: '0.5rem' }}>2</span>
                      Configure MCP Client
                    </h4>
                    <button
                      type="button"
                      className="btn btn-sm btn-outline-primary"
                      onClick={() => copyText(selectedDeploy === 'docker' ? dockerClientConfig : nodeClientConfig, 'MCP config copied')}
                    >
                      <i className="bi bi-clipboard" /> Copy
                    </button>
                  </div>
                  <p style={{ fontSize: '0.8125rem', color: 'var(--text-secondary)', marginBottom: '0.5rem' }}>
                    Add to your Cursor or Claude Desktop <code>mcp.json</code>:
                  </p>
                  <pre style={{ background: '#1a1a2e', padding: '1rem', borderRadius: '8px', fontSize: '0.8125rem', color: '#fde68a', margin: 0 }}>
{selectedDeploy === 'docker' ? dockerClientConfig : nodeClientConfig}
                  </pre>
                </div>
              </div>
            </div>
          )}

          {selectedDeploy === 'github' && (
            <div className="card">
              <h4 className="card-title" style={{ marginBottom: '0.5rem' }}>GitHub Deployment</h4>
              <p style={{ color: 'var(--text-secondary)' }}>
                Download the ZIP, push to a GitHub repo, then set up CI/CD (GitHub Actions, Render, Railway, Fly.io, etc.).
              </p>
              <button className="btn btn-primary" onClick={handleDownload} disabled={downloading}>
                <i className="bi bi-download" />
                {downloading ? ' Generating...' : ' Download ZIP'}
              </button>
            </div>
          )}

          {selectedDeploy === 'azure' && (
            <div className="card">
              <h4 className="card-title" style={{ marginBottom: '0.5rem' }}>Cloud Deployment</h4>
              <p style={{ color: 'var(--text-secondary)', marginTop: 0 }}>
                Build and deploy to cloud via CI/CD pipeline.
              </p>
              <div style={{ textAlign: 'center', padding: '2rem', background: 'var(--hover-bg)', borderRadius: '12px', border: '2px dashed var(--card-border)' }}>
                <i className="bi bi-gear" style={{ fontSize: '2.5rem', color: 'var(--text-muted)', display: 'block', marginBottom: '0.75rem' }} />
                <h4 style={{ marginBottom: '0.35rem' }}>Not Yet Implemented</h4>
                <p style={{ color: 'var(--text-muted)', margin: 0 }}>Use Download ZIP and deploy via your preferred cloud tooling.</p>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
