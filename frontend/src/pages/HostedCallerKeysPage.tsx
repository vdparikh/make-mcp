import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { toast } from 'react-toastify';
import {
  createHostedCallerAPIKey,
  listHostedCallerAPIKeys,
  revokeHostedCallerAPIKey,
  type HostedCallerAPIKey,
} from '../services/api';
import { useAuth } from '../contexts/AuthContext';

function toRFC3339Local(value: string): string | undefined {
  if (!value) return undefined;
  const d = new Date(value);
  if (Number.isNaN(d.getTime())) return undefined;
  return d.toISOString();
}

export default function HostedCallerKeysPage() {
  const { user } = useAuth();
  const [keys, setKeys] = useState<HostedCallerAPIKey[]>([]);
  const [loading, setLoading] = useState(false);
  const [creating, setCreating] = useState(false);
  const [newCallerUserID, setNewCallerUserID] = useState('');
  const [newScopes, setNewScopes] = useState('');
  const [allowAlias, setAllowAlias] = useState(false);
  const [expiresAtLocal, setExpiresAtLocal] = useState('');
  const [newPlainKey, setNewPlainKey] = useState('');
  const [revokingKeyID, setRevokingKeyID] = useState('');

  const loadKeys = async () => {
    setLoading(true);
    try {
      const list = await listHostedCallerAPIKeys();
      setKeys(list);
    } catch {
      toast.error('Failed to load caller keys');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (!user?.id) return;
    setNewCallerUserID(user.id);
    void loadKeys();
  }, [user?.id]);

  const onCreate = async () => {
    if (!newCallerUserID.trim()) {
      toast.error('Caller user ID is required');
      return;
    }
    setCreating(true);
    try {
      const scopes = newScopes
        .split(',')
        .map((v) => v.trim())
        .filter(Boolean);
      const response = await createHostedCallerAPIKey({
        caller_user_id: newCallerUserID.trim(),
        scopes,
        allow_alias: allowAlias,
        expires_at: toRFC3339Local(expiresAtLocal),
      });
      setNewPlainKey(response.api_key);
      setNewCallerUserID(user?.id || '');
      setNewScopes('');
      setAllowAlias(false);
      setExpiresAtLocal('');
      toast.success('Caller API key created');
      await loadKeys();
    } catch (err: unknown) {
      const message =
        typeof err === 'object' && err !== null && 'response' in err
          ? (err as { response?: { data?: { error?: string } } }).response?.data?.error
          : undefined;
      toast.error(message || 'Failed to create caller API key');
    } finally {
      setCreating(false);
    }
  };

  const onRevoke = async (keyId: string) => {
    if (!keyId) return;
    setRevokingKeyID(keyId);
    try {
      await revokeHostedCallerAPIKey(keyId);
      toast.success('Caller API key revoked');
      await loadKeys();
    } catch {
      toast.error('Failed to revoke key');
    } finally {
      setRevokingKeyID('');
    }
  };

  return (
    <div className="dashboard-page">
      <div className="page-header" style={{ alignItems: 'flex-start' }}>
        <div>
          <nav className="page-breadcrumb">
            <Link to="/" className="page-breadcrumb-link">Dashboard</Link>
            <span className="page-breadcrumb-sep">/</span>
            <span className="page-breadcrumb-current">Account</span>
            <span className="page-breadcrumb-sep">/</span>
            <span className="page-breadcrumb-current">Caller API Keys</span>
          </nav>
          <h1 className="page-title" style={{ marginBottom: 4 }}>
            <i className="bi bi-key page-title-icon"></i>
            Caller API Keys
          </h1>
          <p className="page-subtitle" style={{ marginBottom: 0 }}>
            Generate per-caller keys used in <code>X-Make-MCP-Caller-Id</code>. Runtime verifies key, resolves identity, and overrides caller headers.
          </p>
        </div>
      </div>

      <div className="card" style={{ marginBottom: '1rem' }}>
        <div className="card-header">
          <h3 style={{ margin: 0, fontSize: '1rem' }}>Create Caller Key</h3>
        </div>
        <div className="card-body">
          <div className="deploy-flow-github-grid">
            <div className="form-group">
              <label className="form-label">Caller user ID</label>
              <input className="form-control" value={newCallerUserID} onChange={(e) => setNewCallerUserID(e.target.value)} readOnly />
            </div>
            <div className="form-group">
              <label className="form-label">Scopes (comma separated)</label>
              <input className="form-control" value={newScopes} onChange={(e) => setNewScopes(e.target.value)} placeholder="tools:read, tools:write" />
            </div>
            <div className="form-group">
              <label className="form-label">Expires at (optional)</label>
              <input type="datetime-local" className="form-control" value={expiresAtLocal} onChange={(e) => setExpiresAtLocal(e.target.value)} />
            </div>
          </div>
          <div className="form-check mt-2">
            <input id="allowAlias" type="checkbox" className="form-check-input" checked={allowAlias} onChange={(e) => setAllowAlias(e.target.checked)} />
            <label htmlFor="allowAlias" className="form-check-label">
              Allow caller alias override with <code>X-Make-MCP-Caller-Alias</code>
            </label>
          </div>
          <div className="deploy-flow-inline-actions mt-3">
            <button type="button" className="btn btn-primary" disabled={creating || !user?.id} onClick={onCreate}>
              {creating ? <><i className="bi bi-hourglass-split"></i> Creating...</> : <><i className="bi bi-key-fill"></i> Generate API Key</>}
            </button>
          </div>
          {newPlainKey && (
            <div className="alert alert-warning mt-3" role="alert">
              <strong>Copy now:</strong> this key is shown once.<br />
              <code>{newPlainKey}</code>
            </div>
          )}
        </div>
      </div>

      <div className="card">
        <div className="card-header">
          <h3 style={{ margin: 0, fontSize: '1rem' }}>Existing Caller Keys</h3>
        </div>
        <div className="card-body">
          {loading ? (
            <p className="text-muted">Loading...</p>
          ) : keys.length === 0 ? (
            <p className="text-muted">No caller keys created yet.</p>
          ) : (
            <div className="table-responsive">
              <table className="table">
                <thead>
                  <tr>
                    <th>Key ID</th>
                    <th>Caller User</th>
                    <th>Tenant</th>
                    <th>Scopes</th>
                    <th>Expires</th>
                    <th>Status</th>
                    <th></th>
                  </tr>
                </thead>
                <tbody>
                  {keys.map((k) => (
                    <tr key={k.id}>
                      <td><code>{k.key_id}</code></td>
                      <td>{k.caller_user_id}</td>
                      <td>{k.tenant_id || '—'}</td>
                      <td>{k.scopes?.length ? k.scopes.join(', ') : '—'}</td>
                      <td>{k.expires_at ? new Date(k.expires_at).toLocaleString() : 'Never'}</td>
                      <td>{k.revoked_at ? 'Revoked' : 'Active'}</td>
                      <td className="text-end">
                        {!k.revoked_at && (
                          <button
                            type="button"
                            className="btn btn-sm btn-outline-danger"
                            disabled={revokingKeyID === k.id}
                            onClick={() => onRevoke(k.id)}
                          >
                            {revokingKeyID === k.id ? 'Revoking...' : 'Revoke'}
                          </button>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
