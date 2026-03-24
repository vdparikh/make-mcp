import { useMemo, useState } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { toast } from 'react-toastify';
import { importServerJSON, type ServerJSONExportPayload } from '../services/api';

export default function ImportServerJSON() {
  const navigate = useNavigate();
  const [raw, setRaw] = useState<string>('');
  const [loading, setLoading] = useState(false);
  const [importing, setImporting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const [serverNameOverride, setServerNameOverride] = useState('');
  const [descriptionOverride, setDescriptionOverride] = useState('');
  const sample = useMemo(
    () =>
      `// Paste a server export payload here.\n// Tip: it should include schema_version, server, tools, resources, prompts, context_configs, and policies.\n`,
    []
  );

  const handleImport = async () => {
    const trimmed = raw.trim();
    if (!trimmed) {
      toast.error('Please paste a server JSON export payload');
      return;
    }

    let payload: ServerJSONExportPayload;
    try {
      payload = JSON.parse(trimmed) as ServerJSONExportPayload;
    } catch (e) {
      setError('Invalid JSON');
      toast.error('Invalid JSON');
      return;
    }

    setImporting(true);
    setError(null);
    setLoading(true);
    try {
      const result = await importServerJSON(payload, {
        server_name_override: serverNameOverride || undefined,
        description_override: descriptionOverride || undefined,
      });
      toast.success(`Imported server: ${result.server.name}`);
      navigate(`/servers/${result.server.id}`);
    } catch (err: any) {
      const msg = err?.response?.data?.error || err?.message || 'Failed to import server JSON';
      setError(msg);
      toast.error(msg);
    } finally {
      setLoading(false);
      setImporting(false);
    }
  };

  return (
    <div className="page">
      <div className="page-header">
        <div>
          <nav className="page-breadcrumb">
            <Link to="/" className="page-breadcrumb-link">
              Dashboard
            </Link>
            <span className="page-breadcrumb-sep">/</span>
            <span className="page-breadcrumb-current">Import Server JSON</span>
          </nav>
          <h1 className="page-title">
            <i className="bi bi-file-earmark-code page-title-icon"></i>
            Import from Server JSON
          </h1>
          <p className="page-subtitle">Clone a server configuration between environments or orgs.</p>
        </div>
        <Link to="/" className="btn btn-secondary">
          <i className="bi bi-arrow-left"></i>
          Back to Dashboard
        </Link>
      </div>

      <div className="card" style={{ marginBottom: '1.5rem' }}>
        <div className="card-header">
          <h3 className="card-title">
            <i className="bi bi-upload" style={{ marginRight: '0.5rem' }}></i>
            Paste Export JSON
          </h3>
          <button type="button" className="btn btn-secondary btn-sm" onClick={() => setRaw(sample)}>
            <i className="bi bi-file-text"></i> Insert Tip
          </button>
        </div>

        <div className="editor-container" style={{ height: 420, padding: '1rem' }}>
          <textarea
            className="form-control"
            style={{ height: '100%', fontFamily: 'monospace', fontSize: 13, resize: 'vertical' }}
            value={raw}
            onChange={(e) => setRaw(e.target.value)}
            placeholder="Paste JSON export payload"
          />
        </div>

        <div style={{ padding: '0 1rem 1rem 1rem', display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '1rem' }}>
          <div>
            <label className="form-label">Server name override (optional)</label>
            <input
              className="form-control"
              value={serverNameOverride}
              onChange={(e) => setServerNameOverride(e.target.value)}
              placeholder="Use exported name if empty"
            />
          </div>
          <div>
            <label className="form-label">Description override (optional)</label>
            <input
              className="form-control"
              value={descriptionOverride}
              onChange={(e) => setDescriptionOverride(e.target.value)}
              placeholder="Use exported description if empty"
            />
          </div>
        </div>

        {error && (
          <div style={{ padding: '0 1rem 1rem 1rem', color: 'var(--danger)' }}>
            <i className="bi bi-exclamation-triangle" style={{ marginRight: '0.5rem' }} />
            {error}
          </div>
        )}

        <div style={{ padding: '0 1rem 1rem 1rem' }}>
          <button
            type="button"
            className="btn btn-primary"
            onClick={handleImport}
            disabled={importing || loading}
          >
            {importing ? (
              <>
                <i className="bi bi-hourglass-split" /> Importing...
              </>
            ) : (
              <>
                <i className="bi bi-file-earmark-arrow-up" /> Import server
              </>
            )}
          </button>
        </div>
      </div>
    </div>
  );
}

