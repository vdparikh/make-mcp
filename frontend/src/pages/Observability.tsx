import { useState, useEffect } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import type { ObservabilityDashboardResponse, ToolExecution } from '../types';
import {
  checkHostedSessionHealth,
  getObservabilityDashboard,
  listHostedSessions,
  restartHostedSession,
  stopHostedSession,
  type HostedSession,
} from '../services/api';

export default function Observability() {
  const [searchParams, setSearchParams] = useSearchParams();
  const serverIdFromUrl = searchParams.get('server_id') ?? '';
  const toolNameFromUrl = searchParams.get('tool_name') ?? '';
  const clientUserIdFromUrl = searchParams.get('client_user_id') ?? '';
  const clientAgentFromUrl = searchParams.get('client_agent') ?? '';

  const [data, setData] = useState<ObservabilityDashboardResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [serverFilter, setServerFilter] = useState(serverIdFromUrl);
  const [toolFilter, setToolFilter] = useState(toolNameFromUrl);
  const [clientUserFilter, setClientUserFilter] = useState(clientUserIdFromUrl);
  const [clientAgentFilter, setClientAgentFilter] = useState(clientAgentFromUrl);
  const [showSessionsModal, setShowSessionsModal] = useState(false);
  const [sessionsLoading, setSessionsLoading] = useState(false);
  const [sessionsError, setSessionsError] = useState<string | null>(null);
  const [sessions, setSessions] = useState<HostedSession[]>([]);
  const [sessionActionBusy, setSessionActionBusy] = useState<Record<string, string>>({});

  const fetchData = () => {
    setLoading(true);
    setError(null);
    const params: { server_id?: string; tool_name?: string; client_user_id?: string; client_agent?: string; limit?: number } = { limit: 200 };
    if (serverFilter) params.server_id = serverFilter;
    if (toolFilter) params.tool_name = toolFilter;
    if (clientUserFilter) params.client_user_id = clientUserFilter;
    if (clientAgentFilter) params.client_agent = clientAgentFilter;
    getObservabilityDashboard(params)
      .then(setData)
      .catch((err) => setError(err.response?.data?.error || err.message || 'Failed to load observability'))
      .finally(() => setLoading(false));
  };

  const fetchSessions = () => {
    setSessionsLoading(true);
    setSessionsError(null);
    listHostedSessions()
      .then((items) => setSessions(items))
      .catch((err) => setSessionsError(err.response?.data?.error || err.message || 'Failed to load hosted sessions'))
      .finally(() => setSessionsLoading(false));
  };

  useEffect(() => {
    fetchData();
  }, [serverFilter, toolFilter, clientUserFilter, clientAgentFilter]);

  useEffect(() => {
    setServerFilter(serverIdFromUrl);
    setToolFilter(toolNameFromUrl);
    setClientUserFilter(clientUserIdFromUrl);
    setClientAgentFilter(clientAgentFromUrl);
  }, [serverIdFromUrl, toolNameFromUrl, clientUserIdFromUrl, clientAgentFromUrl]);

  useEffect(() => {
    if (!showSessionsModal) return;
    fetchSessions();
  }, [showSessionsModal]);

  const applyFilters = () => {
    const next = new URLSearchParams(searchParams);
    if (serverFilter) next.set('server_id', serverFilter); else next.delete('server_id');
    if (toolFilter) next.set('tool_name', toolFilter); else next.delete('tool_name');
    if (clientUserFilter) next.set('client_user_id', clientUserFilter); else next.delete('client_user_id');
    if (clientAgentFilter) next.set('client_agent', clientAgentFilter); else next.delete('client_agent');
    setSearchParams(next);
  };

  const events = data?.recent_events ?? [];
  const servers = data?.servers ?? [];
  const uniqueToolNames = Array.from(new Set(events.map((e) => e.tool_name || e.tool_id).filter(Boolean))).sort();
  const uniqueClientUserIds = Array.from(new Set(events.map((e) => e.client_user_id).filter(Boolean))).sort();
  const uniqueClientAgents = Array.from(new Set(events.map((e) => e.client_agent).filter(Boolean))).sort();

  const serverName = (id: string) => servers.find((s) => s.id === id)?.name ?? id;
  const formatTime = (value?: string) => {
    if (!value) return '—';
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) return '—';
    return date.toLocaleString();
  };
  const withSessionAction = async (session: HostedSession, action: 'health' | 'restart' | 'stop') => {
    const key = session.server_id;
    setSessionActionBusy((prev) => ({ ...prev, [key]: action }));
    try {
      const updated =
        action === 'health'
          ? await checkHostedSessionHealth(key)
          : action === 'restart'
            ? await restartHostedSession(key)
            : await stopHostedSession(key);
      setSessions((prev) => prev.map((item) => (item.server_id === key ? { ...item, ...updated } : item)));
    } catch (err: unknown) {
      const message =
        typeof err === 'object' && err !== null && 'response' in err
          ? (err as { response?: { data?: { error?: string } } }).response?.data?.error
          : undefined;
      setSessionsError(message || 'Session action failed');
    } finally {
      setSessionActionBusy((prev) => {
        const next = { ...prev };
        delete next[key];
        return next;
      });
    }
  };

  return (
    <div>
      <nav className="page-breadcrumb">
        <Link to="/" className="page-breadcrumb-link">
          Dashboard
        </Link>
        <span className="page-breadcrumb-sep">/</span>
        <span className="page-breadcrumb-current">Observability</span>
      </nav>
      <div className="page-header" style={{ marginBottom: '1.5rem' }}>
        <div>
          <h1 className="page-title">
            <i className="bi bi-graph-up page-title-icon"></i>
            Observability
          </h1>
          <p className="page-subtitle">
            Tool calls, latency, failures, and repair suggestions from your MCP servers across all clients.
          </p>
        </div>
        <button className="btn btn-secondary" onClick={() => setShowSessionsModal(true)}>
          <i className="bi bi-hdd-rack" style={{ marginRight: '0.4rem' }} />
          Hosted Runtime Sessions
        </button>
      </div>

      <div className="card observability-filters-card" style={{ marginBottom: '1.5rem' }}>
        <h3 className="card-title" style={{ marginBottom: '0.75rem' }}>Filters</h3>
        <div className="observability-filters-row">
          <div>
            <label style={{ display: 'block', fontSize: '0.8rem', color: 'var(--text-muted)', marginBottom: '0.25rem' }}>Server</label>
            <select
              className="form-select"
              value={serverFilter}
              onChange={(e) => setServerFilter(e.target.value)}
              style={{ minWidth: '200px' }}
            >
              <option value="">All servers</option>
              {servers.map((s) => (
                <option key={s.id} value={s.id}>{s.name}</option>
              ))}
            </select>
          </div>
          <div>
            <label style={{ display: 'block', fontSize: '0.8rem', color: 'var(--text-muted)', marginBottom: '0.25rem' }}>Tool</label>
            <select
              className="form-select"
              value={toolFilter}
              onChange={(e) => setToolFilter(e.target.value)}
              style={{ minWidth: '180px' }}
            >
              <option value="">All tools</option>
              {uniqueToolNames.map((name) => (
                <option key={name} value={name}>{name}</option>
              ))}
            </select>
          </div>
          <div>
            <label style={{ display: 'block', fontSize: '0.8rem', color: 'var(--text-muted)', marginBottom: '0.25rem' }}>User / Tenant</label>
            <select
              className="form-select"
              value={clientUserFilter}
              onChange={(e) => setClientUserFilter(e.target.value)}
              style={{ minWidth: '160px' }}
            >
              <option value="">All</option>
              {uniqueClientUserIds.map((id) => (
                <option key={id} value={id}>{id}</option>
              ))}
            </select>
          </div>
          <div>
            <label style={{ display: 'block', fontSize: '0.8rem', color: 'var(--text-muted)', marginBottom: '0.25rem' }}>Client (Agent)</label>
            <select
              className="form-select"
              value={clientAgentFilter}
              onChange={(e) => setClientAgentFilter(e.target.value)}
              style={{ minWidth: '160px' }}
            >
              <option value="">All</option>
              {uniqueClientAgents.map((agent) => (
                <option key={agent} value={agent}>{agent}</option>
              ))}
            </select>
          </div>
          <button className="btn btn-primary" onClick={applyFilters}>
            Apply
          </button>
          <button className="btn btn-secondary" onClick={fetchData} title="Refresh">
            <i className="bi bi-arrow-clockwise" />
          </button>
        </div>
      </div>

      {loading && !data && (
        <div className="card">
          <div style={{ padding: '2rem', textAlign: 'center', color: 'var(--text-muted)' }}>
            <span className="spinner" style={{ width: 24, height: 24, borderWidth: 2 }}></span>
            <p style={{ marginTop: '0.75rem' }}>Loading observability…</p>
          </div>
        </div>
      )}

      {error && !data && (
        <div className="card">
          <div style={{ padding: '2rem', textAlign: 'center', color: 'var(--danger)' }}>
            <i className="bi bi-exclamation-triangle" style={{ fontSize: '2rem' }}></i>
            <p style={{ marginTop: '0.75rem' }}>{error}</p>
          </div>
        </div>
      )}

      {data && !loading && (
        <div className="observability-layout">
          <div className="observability-main">
            {events.length > 0 && (
              <div className="card observability-main-card" style={{ marginBottom: '1.5rem' }}>
                <div className="observability-main-card-head">
                  <h4 className="card-title" style={{ marginBottom: '0.75rem' }}>
                    <i className="bi bi-list-ul" style={{ marginRight: '0.5rem' }}></i>
                    Recent tool calls
                  </h4>
                  <span className="observability-row-count">{events.length} events</span>
                </div>
                <div style={{ overflowX: 'auto' }}>
                  <table className="table">
                    <thead>
                      <tr>
                        <th>Server</th>
                        <th>Tool</th>
                        <th>User / Tenant</th>
                        <th>Client (Agent)</th>
                        <th>Time</th>
                        <th>Latency</th>
                        <th>Status</th>
                        <th>Error / Suggestion</th>
                      </tr>
                    </thead>
                    <tbody>
                      {events.slice(0, 100).map((e: ToolExecution) => (
                        <tr key={e.id}>
                          <td>
                            <Link to={`/servers/${e.server_id}`} style={{ color: 'var(--primary-color)' }}>
                              {serverName(e.server_id)}
                            </Link>
                          </td>
                          <td>{e.tool_name || e.tool_id}</td>
                          <td style={{ fontSize: '0.85rem' }}>{e.client_user_id || '—'}</td>
                          <td style={{ fontSize: '0.85rem' }}>{e.client_agent || '—'}</td>
                          <td>{new Date(e.created_at).toLocaleString()}</td>
                          <td>{e.duration_ms} ms</td>
                          <td>
                            {e.success ? (
                              <span className="observability-status ok"><i className="bi bi-check-circle" /> OK</span>
                            ) : (
                              <span className="observability-status failed"><i className="bi bi-x-circle" /> Failed</span>
                            )}
                          </td>
                          <td style={{ fontSize: '0.85rem' }}>
                            {e.error && <span style={{ color: 'var(--danger-color)' }}>{e.error}</span>}
                            {e.repair_suggestion && (
                              <div style={{ marginTop: '0.25rem', color: 'var(--text-secondary)' }}>
                                <i className="bi bi-lightbulb" /> {e.repair_suggestion}
                              </div>
                            )}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            )}

            {events.length === 0 && !loading && (
              <div className="card">
                <div style={{ textAlign: 'center', padding: '2rem', color: 'var(--text-muted)' }}>
                  <i className="bi bi-graph-up" style={{ fontSize: '2.5rem', marginBottom: '0.5rem' }}></i>
                  <p>No runtime events yet. Enable observability reporting in a server’s Observability tab and set the env vars in your deployed MCP server.</p>
                  <p style={{ marginTop: '0.5rem', fontSize: '0.9rem' }}>
                    <Link to="/" style={{ color: 'var(--primary-color)' }}>Go to Dashboard</Link> to open a server and enable reporting.
                  </p>
                </div>
              </div>
            )}
          </div>

          <aside className="observability-side">
            {data.latency_by_tool?.length > 0 && (
              <div className="card observability-side-card">
                <h4 className="card-title" style={{ marginBottom: '0.75rem' }}>
                  <i className="bi bi-speedometer2" style={{ marginRight: '0.5rem' }}></i>
                  Latency by tool
                </h4>
                <ul style={{ listStyle: 'none', padding: 0, margin: 0 }}>
                  {data.latency_by_tool.map((s) => (
                    <li key={s.tool_id || s.tool_name} style={{ padding: '0.5rem 0', borderBottom: '1px solid var(--card-border)' }}>
                      <div style={{ fontWeight: 500 }}>{s.tool_name || s.tool_id}</div>
                      <div style={{ fontSize: '0.85rem', color: 'var(--text-muted)' }}>
                        {s.count} calls · avg {Math.round(s.avg_ms)} ms · max {s.p95_ms} ms
                      </div>
                    </li>
                  ))}
                </ul>
              </div>
            )}
            {data.failures_by_tool?.length > 0 && (
              <div className="card observability-side-card">
                <h4 className="card-title" style={{ marginBottom: '0.75rem' }}>
                  <i className="bi bi-exclamation-triangle" style={{ marginRight: '0.5rem' }}></i>
                  Failures by tool
                </h4>
                <ul style={{ listStyle: 'none', padding: 0, margin: 0 }}>
                  {data.failures_by_tool.map((f) => (
                    <li key={f.tool_id || f.tool_name} style={{ padding: '0.5rem 0', borderBottom: '1px solid var(--card-border)' }}>
                      <div style={{ fontWeight: 500 }}>{f.tool_name || f.tool_id}</div>
                      <div style={{ fontSize: '0.85rem', color: 'var(--danger)' }}>{f.count} failure(s)</div>
                      {f.last_error && (
                        <div style={{ fontSize: '0.8rem', color: 'var(--text-muted)', marginTop: '0.25rem' }}>{f.last_error}</div>
                      )}
                    </li>
                  ))}
                </ul>
              </div>
            )}
            {data.repair_suggestions?.length > 0 && (
              <div className="card observability-side-card observability-sessions-modal-card">
                <h4 className="card-title" style={{ marginBottom: '0.75rem' }}>
                  <i className="bi bi-lightbulb" style={{ marginRight: '0.5rem' }}></i>
                  Repair suggestions
                </h4>
                <ul style={{ listStyle: 'none', padding: 0, margin: 0 }}>
                  {data.repair_suggestions.map((s, i) => (
                    <li key={i} style={{ padding: '0.75rem 0', borderBottom: '1px solid var(--card-border)' }}>
                      <div style={{ fontWeight: 500 }}>{s.tool_name || s.tool_id}</div>
                      <div style={{ fontSize: '0.9rem', color: 'var(--text-secondary)' }}>{s.suggestion}</div>
                      <div style={{ fontSize: '0.8rem', color: 'var(--text-muted)', marginTop: '0.25rem' }}>
                        {new Date(s.created_at).toLocaleString()}
                      </div>
                    </li>
                  ))}
                </ul>
              </div>
            )}
          </aside>
        </div>
      )}

      {showSessionsModal && (
        <div className="modal-overlay" onClick={() => setShowSessionsModal(false)}>
          <div className="modal-content" style={{ maxWidth: '1000px' }} onClick={(e) => e.stopPropagation()}>
            <div className="modal-header">
              <h3 className="modal-title">Hosted Runtime Sessions</h3>
              <button className="btn btn-secondary" onClick={() => setShowSessionsModal(false)}>
                Close
              </button>
            </div>
            <div className="modal-body">
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '0.75rem' }}>
                <p style={{ margin: 0, color: 'var(--text-muted)', fontSize: '0.9rem' }}>
                  Durable runtime sessions tracked in DB. Use actions to check health, restart, or stop.
                </p>
                <button className="btn btn-secondary" onClick={fetchSessions} disabled={sessionsLoading}>
                  <i className="bi bi-arrow-clockwise" />
                </button>
              </div>
              {sessionsError && (
                <div style={{ marginBottom: '0.75rem', color: 'var(--danger)' }}>{sessionsError}</div>
              )}
              {sessionsLoading && (
                <div style={{ textAlign: 'center', color: 'var(--text-muted)', padding: '1rem 0' }}>
                  <span className="spinner" style={{ width: 20, height: 20, borderWidth: 2 }} />
                </div>
              )}
              {!sessionsLoading && sessions.length === 0 && (
                <div style={{ color: 'var(--text-muted)', padding: '1rem 0' }}>No hosted sessions found.</div>
              )}
              {sessions.length > 0 && (
                <div style={{ overflowX: 'auto' }}>
                  <table className="table">
                    <thead>
                      <tr>
                        <th>Server</th>
                        <th>Snapshot</th>
                        <th>Status</th>
                        <th>Health</th>
                        <th>Container</th>
                        <th>Started</th>
                        <th>Last Used</th>
                        <th>Actions</th>
                      </tr>
                    </thead>
                    <tbody>
                      {sessions.map((s) => {
                        const busy = sessionActionBusy[s.server_id];
                        return (
                          <tr key={s.server_id}>
                            <td>
                              <Link to={`/servers/${s.server_id}`} style={{ color: 'var(--primary-color)' }}>
                                {s.server_name || s.server_id}
                              </Link>
                            </td>
                            <td>{s.snapshot_version || '—'}</td>
                            <td>{s.status || 'unknown'}</td>
                            <td>{s.health || 'unknown'}</td>
                            <td style={{ fontSize: '0.8rem' }}>{s.container_id ? s.container_id.slice(0, 12) : '—'}</td>
                            <td>{formatTime(s.started_at)}</td>
                            <td>{formatTime(s.last_used_at)}</td>
                            <td>
                              <div className="observability-session-actions">
                                <button
                                  className="btn btn-secondary"
                                  disabled={!!busy}
                                  onClick={() => withSessionAction(s, 'health')}
                                  title="Check health"
                                >
                                  {busy === 'health' ? '...' : 'Health'}
                                </button>
                                <button
                                  className="btn btn-secondary"
                                  disabled={!!busy}
                                  onClick={() => withSessionAction(s, 'restart')}
                                  title="Restart runtime"
                                >
                                  {busy === 'restart' ? '...' : 'Restart'}
                                </button>
                                <button
                                  className="btn btn-secondary"
                                  disabled={!!busy || s.status === 'stopped'}
                                  onClick={() => withSessionAction(s, 'stop')}
                                  title="Stop runtime"
                                >
                                  {busy === 'stop' ? '...' : 'Stop'}
                                </button>
                              </div>
                            </td>
                          </tr>
                        );
                      })}
                    </tbody>
                  </table>
                </div>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
