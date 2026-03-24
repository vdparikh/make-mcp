import { useEffect, useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import { toast } from 'react-toastify';
import { listHostedCatalog, type HostedCatalogItem } from '../services/api';
import { useTryChat } from '../contexts/TryChatContext';

type InstallLinkSet = {
  cursorLink: string;
  vscodeLink: string;
  vscodeInsidersLink: string;
};

function formatDate(value?: string): string {
  if (!value) return '—';
  const d = new Date(value);
  if (Number.isNaN(d.getTime())) return '—';
  return d.toLocaleString();
}

function buildInstallLinks(item: HostedCatalogItem): InstallLinkSet | null {
  try {
    const parsed = JSON.parse(item.mcp_config || '{}');
    const servers = parsed.mcpServers || {};
    const serverName = Object.keys(servers)[0];
    if (!serverName) return null;
    const serverConfig = servers[serverName];
    const cursorPayload = btoa(JSON.stringify(serverConfig));
    const cursorLink = `cursor://anysphere.cursor-deeplink/mcp/install?name=${encodeURIComponent(serverName)}&config=${cursorPayload}`;

    const vscodeConfig = JSON.stringify({
      name: serverName,
      type: 'sse',
      url: serverConfig.url,
      ...(serverConfig.headers ? { headers: serverConfig.headers } : {}),
    });
    const vscodeLink = `vscode:mcp/install?${encodeURIComponent(vscodeConfig)}`;
    const vscodeInsidersLink = `vscode-insiders:mcp/install?${encodeURIComponent(vscodeConfig)}`;
    return { cursorLink, vscodeLink, vscodeInsidersLink };
  } catch {
    return null;
  }
}

export default function HostedCatalogPage() {
  const { openTryChat } = useTryChat();
  const [loading, setLoading] = useState(true);
  const [items, setItems] = useState<HostedCatalogItem[]>([]);
  const [query, setQuery] = useState('');
  const [showAllConfigs, setShowAllConfigs] = useState<Record<string, boolean>>({});

  const fetchItems = async () => {
    setLoading(true);
    try {
      const data = await listHostedCatalog();
      setItems(data);
    } catch (err: unknown) {
      const message =
        typeof err === 'object' && err !== null && 'response' in err
          ? (err as { response?: { data?: { error?: string } } }).response?.data?.error
          : undefined;
      toast.error(message || 'Failed to load hosted runtime catalog');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchItems();
  }, []);

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return items;
    return items.filter((item) => {
      const haystack = `${item.server_name} ${item.server_slug} ${item.publisher_user_id}`.toLowerCase();
      return haystack.includes(q);
    });
  }, [items, query]);

  const copyText = async (value: string, successMsg: string) => {
    try {
      await navigator.clipboard.writeText(value);
      toast.success(successMsg);
    } catch {
      toast.error('Could not copy');
    }
  };

  return (
    <div className="dashboard-page">
<div className="page-header" style={{ alignItems: 'flex-start' }}>
        <div>
          <nav className="page-breadcrumb">
            <Link to="/" className="page-breadcrumb-link">Dashboard</Link>
            <span className="page-breadcrumb-sep">/</span>
            <span className="page-breadcrumb-current">Hosted Catalog</span>
          </nav>
          <h1 className="page-title" style={{ marginBottom: 4 }}>
            <i className="bi bi-hdd-network page-title-icon"></i>
            Running Hosted Servers
          </h1>
          <p className="page-subtitle" style={{ marginBottom: 0 }}>
            Install any currently running hosted MCP server in one click.
          </p>
        </div>
        <div className="deploy-flow-inline-actions">
          <button type="button" className="btn btn-secondary" onClick={fetchItems} disabled={loading}>
            <i className="bi bi-arrow-clockwise"></i>
            Refresh
          </button>
        </div>
      </div>

      <div className="card dashboard-toolbar-card" style={{ marginBottom: '1rem' }}>
        <div className="dashboard-toolbar-row">
          <div className="dashboard-search d-flex align-items-center mb-2">
            <i className="bi bi-search"></i>
            <input
              type="text"
              className="form-control"
              placeholder="Search by server name, slug, or publisher..."
              value={query}
              onChange={(e) => setQuery(e.target.value)}
            />
          </div>
          <span className="table-count-chip ms-3  dashboard-cc-pill ">
            {filtered.length} running endpoint{filtered.length === 1 ? '' : 's'}
          </span>
        </div>
      </div>

      {loading ? (
        <div className="card">
          <div className="card-body text-center text-muted py-5">
            <i className="bi bi-arrow-repeat" style={{ marginRight: 8 }}></i>
            Loading hosted catalog...
          </div>
        </div>
      ) : filtered.length === 0 ? (
        <div className="card">
          <div className="card-body text-center py-5">
            <i className="bi bi-inboxes empty-state-icon"></i>
            <h3 style={{ marginTop: '0.75rem' }}>No running hosted servers found</h3>
            <p className="text-muted">Publish a server from Deploy to make it installable here.</p>
            <Link to="/deploy" className="btn btn-primary">
              <i className="bi bi-rocket-takeoff"></i>
              Open Deploy
            </Link>
          </div>
        </div>
      ) : (
        <div className="hosted-catalog-grid">
          {filtered.map((item) => {
            const links = buildInstallLinks(item);
            const openConfig = Boolean(showAllConfigs[item.server_id]);
            return (
              <article key={`${item.publisher_user_id}:${item.server_id}`} className="card hosted-catalog-card">
                <div className="card-body">
                  <div className="hosted-catalog-head">
                    <div>
                      <h3 className="hosted-catalog-title">{item.server_name}</h3>
                      <div className="hosted-catalog-meta">
                        <span><i className="bi bi-link-45deg"></i> {item.server_slug}</span>
                        <span><i className="bi bi-person"></i> {item.publisher_user_id}</span>
                        <span><i className="bi bi-clock-history"></i> {formatDate(item.last_ensured_at)}</span>
                      </div>
                    </div>
                    <span className="hosted-catalog-badge">
                      {item.hosted_auth_mode === 'bearer_token' ? 'Bearer protected' : 'No auth'}
                      {item.require_caller_identity ? ' + Caller key' : ''}
                    </span>
                  </div>

                  <div className="deploy-flow-endpoint-row" style={{ marginTop: '0.8rem' }}>
                    <div className="deploy-flow-endpoint-main">
                      <div className="deploy-flow-meta-label">Endpoint</div>
                      <code>{item.endpoint}</code>
                    </div>
                    <button type="button" className="btn btn-secondary btn-sm" onClick={() => copyText(item.endpoint, 'Endpoint copied')}>
                      <i className="bi bi-clipboard"></i>
                      Copy
                    </button>
                  </div>

                  {links && (
                    <div className="deploy-oneclick-buttons gap-2 d-flex align-items-center" style={{ marginTop: '0.8rem' }}>
                      <a href={links.cursorLink} className="w-100 text-center deploy-oneclick-btn deploy-oneclick-cursor">Install in Cursor</a>
                      <a href={links.vscodeLink} className="w-100 text-center deploy-oneclick-btn deploy-oneclick-vscode">Install in VS Code</a>
                      <a href={links.vscodeInsidersLink} className="w-100 text-center deploy-oneclick-btn deploy-oneclick-vscode-insiders">VS Code Insiders</a>
                    </div>
                  )}

                  {item.require_caller_identity && (
                    <div className="alert alert-info py-2 px-3 mb-0 mt-2" role="status" style={{ fontSize: '0.875rem' }}>
                      <strong>Caller API key required.</strong>{' '}
                      Open{' '}
                      <Link to="/hosted/keys">Caller API Keys</Link>
                      {' '}to generate a key, then replace <code>&lt;caller-api-key&gt;</code> in the JSON below with your <code>mkc_…</code> secret before the client connects.
                    </div>
                  )}

                  </div>
                  <div className='card-footer border-0 bg-transparent'>

                  <div className="deploy-flow-inline-actions d-flex align-items-center" style={{ marginTop: '0.75rem' }}>
                    <div>
                      <button
                        type="button"
                        className="btn btn-sm btn-warning me-2"
                        onClick={() =>
                          openTryChat({
                            type: 'server',
                            id: item.server_id,
                            name: item.server_name,
                            endpoint: item.endpoint,
                          })
                        }
                      >
                        <i className="bi bi-stars"></i>
                        Try Chat
                      </button>
                    </div>
                    <div>
                      <button type="button" className="btn btn-sm btn-outline-primary me-2" onClick={() => copyText(item.mcp_config, 'MCP config copied')}>
                        <i className="bi bi-clipboard"></i>
                        Copy config
                      </button>
                      <button
                        type="button"
                        className="btn btn-sm btn-secondary"
                        onClick={() => setShowAllConfigs((prev) => ({ ...prev, [item.server_id]: !openConfig }))}
                      >
                        <i className={`bi ${openConfig ? 'bi-chevron-up' : 'bi-chevron-down'}`}></i>
                        {openConfig ? 'Hide JSON' : 'Show JSON'}
                      </button>
                    </div>
                  </div>

                  {openConfig && (
                    <pre className="hosted-catalog-config-pre">{item.mcp_config}</pre>
                  )}
                </div>
              </article>
            );
          })}
        </div>
      )}
    </div>
  );
}
