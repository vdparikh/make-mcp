import { useState } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { toast } from 'react-toastify';
import Editor from '@monaco-editor/react';
import { previewOpenAPIImport, importOpenAPI, fetchOpenAPISpecFromUrl, type OpenAPIPreview } from '../services/api';

const sampleOpenAPI = `openapi: "3.0.0"
info:
  title: Pet Store API
  description: A sample API for demonstrating OpenAPI import
  version: "1.0.0"
servers:
  - url: https://petstore.swagger.io/v2
paths:
  /pet/{petId}:
    get:
      operationId: getPetById
      summary: Find pet by ID
      description: Returns a single pet
      parameters:
        - name: petId
          in: path
          description: ID of pet to return
          required: true
          schema:
            type: integer
      responses:
        "200":
          description: Successful operation
          content:
            application/json:
              schema:
                type: object
                properties:
                  id:
                    type: integer
                  name:
                    type: string
                  status:
                    type: string
  /pet:
    post:
      operationId: addPet
      summary: Add a new pet to the store
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required:
                - name
              properties:
                name:
                  type: string
                status:
                  type: string
                  enum: [available, pending, sold]
      responses:
        "200":
          description: Successful operation
`;

export default function ImportOpenAPI() {
  const navigate = useNavigate();
  const [spec, setSpec] = useState('');
  const [preview, setPreview] = useState<OpenAPIPreview | null>(null);
  const [loading, setLoading] = useState(false);
  const [importing, setImporting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // URL import
  const [importUrl, setImportUrl] = useState('');
  const [fetchingUrl, setFetchingUrl] = useState(false);

  // Override fields
  const [serverName, setServerName] = useState('');
  const [description, setDescription] = useState('');
  const [baseUrl, setBaseUrl] = useState('');

  const handlePreview = async () => {
    if (!spec.trim()) {
      toast.error('Please paste an OpenAPI specification');
      return;
    }

    setLoading(true);
    setError(null);
    setPreview(null);

    try {
      const result = await previewOpenAPIImport(spec);
      setPreview(result);
      setServerName(result.server.name);
      setDescription(result.server.description || '');
      setBaseUrl(result.server.base_url || '');
      toast.success(`Found ${result.tools_count} tools`);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to parse OpenAPI spec';
      setError(message);
      toast.error(message);
    } finally {
      setLoading(false);
    }
  };

  const handleImport = async () => {
    if (!spec.trim()) {
      toast.error('Please paste an OpenAPI specification');
      return;
    }

    setImporting(true);
    try {
      const result = await importOpenAPI(spec, {
        server_name: serverName || undefined,
        description: description || undefined,
        base_url: baseUrl || undefined,
      });
      toast.success(`Created server with ${result.tools_created} tools`);
      navigate(`/servers/${result.server.id}`);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to import OpenAPI spec';
      toast.error(message);
    } finally {
      setImporting(false);
    }
  };

  const loadSample = () => {
    setSpec(sampleOpenAPI);
    setPreview(null);
    setError(null);
  };

  const handleFetchFromUrl = async () => {
    const url = importUrl.trim();
    if (!url) {
      toast.error('Please enter an API spec URL');
      return;
    }
    setFetchingUrl(true);
    setError(null);
    setPreview(null);
    try {
      const { spec: fetchedSpec } = await fetchOpenAPISpecFromUrl(url);
      setSpec(fetchedSpec);
      const result = await previewOpenAPIImport(fetchedSpec);
      setPreview(result);
      setServerName(result.server.name);
      setDescription(result.server.description || '');
      setBaseUrl(result.server.base_url || '');
      toast.success(`Preview ready: ${result.tools_count} tools. Click "Create server" to create.`);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to fetch or parse spec from URL';
      setError(message);
      toast.error(message);
    } finally {
      setFetchingUrl(false);
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
            <span className="page-breadcrumb-current">Import OpenAPI</span>
          </nav>
          <h1 className="page-title">
            <i className="bi bi-file-earmark-code page-title-icon"></i>
            Import from OpenAPI
          </h1>
          <p className="page-subtitle">
            Generate an MCP server from any OpenAPI 3.x specification
          </p>
        </div>
        <Link to="/" className="btn btn-secondary">
          <i className="bi bi-arrow-left"></i>
          Back to Dashboard
        </Link>
      </div>

      {/* URL — paste API URL → load and preview */}
      <div className="card" style={{ marginBottom: '1.5rem' }}>
        <div className="card-header">
          <h3 className="card-title">
            <i className="bi bi-link-45deg" style={{ marginRight: '0.5rem' }}></i>
            Load from URL
          </h3>
        </div>
        <div style={{ display: 'flex', gap: '0.75rem', flexWrap: 'wrap', alignItems: 'flex-start' }}>
          <input
            type="url"
            className="form-control"
            placeholder="Paste API URL (e.g. https://api.example.com/openapi.json)"
            value={importUrl}
            onChange={(e) => setImportUrl(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && handleFetchFromUrl()}
            style={{ flex: '1', minWidth: '280px' }}
          />
          <button
            type="button"
            className="btn btn-primary"
            onClick={handleFetchFromUrl}
            disabled={fetchingUrl || !importUrl.trim()}
          >
            {fetchingUrl ? (
              <>
                <i className="bi bi-hourglass-split"></i>
                Loading...
              </>
            ) : (
              <>
                <i className="bi bi-download"></i>
                Load
              </>
            )}
          </button>
        </div>
        <p style={{ color: 'var(--text-muted)', fontSize: '0.875rem', marginTop: '0.5rem', marginBottom: 0 }}>
          Paste a public OpenAPI 3.x URL (JSON or YAML). Load fetches the spec and shows the preview. Use <strong>Create server</strong> in the preview to create it.
        </p>
      </div>

      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '1.5rem' }}>
        {/* Left: Editor */}
        <div className="card">
          <div className="card-header">
            <h3 className="card-title">OpenAPI Specification</h3>
            <button className="btn btn-secondary btn-sm" onClick={loadSample}>
              <i className="bi bi-file-text"></i>
              Load Sample
            </button>
          </div>
          
          <div className="editor-container" style={{ height: '500px' }}>
            <Editor
              height="100%"
              language="yaml"
              theme="vs-dark"
              value={spec}
              onChange={(value) => setSpec(value || '')}
              options={{
                minimap: { enabled: false },
                fontSize: 13,
                wordWrap: 'on',
              }}
            />
          </div>

          <div style={{ display: 'flex', gap: '0.75rem', marginTop: '1rem' }}>
            <button 
              className="btn btn-primary" 
              onClick={handlePreview}
              disabled={loading || !spec.trim()}
            >
              {loading ? (
                <>
                  <i className="bi bi-hourglass-split"></i>
                  Loading...
                </>
              ) : (
                <>
                  <i className="bi bi-search"></i>
                  Preview
                </>
              )}
            </button>
          </div>

          {error && (
            <div style={{ 
              marginTop: '1rem', 
              padding: '1rem', 
              background: 'rgba(239, 68, 68, 0.1)', 
              borderRadius: '8px',
              border: '1px solid rgba(239, 68, 68, 0.3)',
              color: 'var(--error-color)'
            }}>
              <i className="bi bi-exclamation-triangle" style={{ marginRight: '0.5rem' }}></i>
              {error}
            </div>
          )}
        </div>

        {/* Right: Preview */}
        <div className="card">
          <div className="card-header">
            <h3 className="card-title">Preview</h3>
          </div>

          {!preview ? (
            <div className="empty-state">
              <i className="bi bi-arrow-left-circle"></i>
              <h3>Preview</h3>
              <p>Paste a spec or load from URL, then click Preview to see what will be created. Then click Create server.</p>
            </div>
          ) : (
            <div>
              {/* Server Info */}
              <div style={{ marginBottom: '1.5rem' }}>
                <h4 style={{ color: 'var(--text-primary)', marginBottom: '1rem' }}>
                  <i className="bi bi-server" style={{ marginRight: '0.5rem', color: 'var(--primary-color)' }}></i>
                  Server Details
                </h4>
                
                <div className="form-group">
                  <label className="form-label">Server Name</label>
                  <input
                    type="text"
                    className="form-control"
                    value={serverName}
                    onChange={(e) => setServerName(e.target.value)}
                  />
                </div>

                <div className="form-group">
                  <label className="form-label">Description</label>
                  <input
                    type="text"
                    className="form-control"
                    value={description}
                    onChange={(e) => setDescription(e.target.value)}
                  />
                </div>

                <div className="form-group">
                  <label className="form-label">Base URL</label>
                  <input
                    type="text"
                    className="form-control"
                    value={baseUrl}
                    onChange={(e) => setBaseUrl(e.target.value)}
                  />
                </div>
              </div>

              {/* Auth Config */}
              {preview.auth && (
                <div style={{ 
                  marginBottom: '1.5rem', 
                  padding: '1rem', 
                  background: 'rgba(129, 140, 248, 0.1)',
                  borderRadius: '8px',
                  border: '1px solid rgba(129, 140, 248, 0.2)'
                }}>
                  <h5 style={{ color: 'var(--text-primary)', marginBottom: '0.5rem' }}>
                    <i className="bi bi-shield-lock" style={{ marginRight: '0.5rem', color: 'var(--primary-color)' }}></i>
                    Authentication Detected
                  </h5>
                  <p style={{ color: 'var(--text-secondary)', margin: 0, fontSize: '0.875rem' }}>
                    Type: <strong style={{ color: 'var(--text-primary)' }}>{preview.auth.type}</strong>
                    {preview.auth.header_name && (
                      <> | Header: <strong style={{ color: 'var(--text-primary)' }}>{preview.auth.header_name}</strong></>
                    )}
                    {preview.auth.token_url && (
                      <> | Token URL: <strong style={{ color: 'var(--text-primary)' }}>{preview.auth.token_url}</strong></>
                    )}
                  </p>
                </div>
              )}

              {/* Tools List */}
              <div>
                <h4 style={{ color: 'var(--text-primary)', marginBottom: '1rem' }}>
                  <i className="bi bi-tools" style={{ marginRight: '0.5rem', color: 'var(--success-color)' }}></i>
                  Tools to Create ({preview.tools_count})
                </h4>
                
                <div style={{ maxHeight: '280px', overflowY: 'auto' }}>
                  {preview.tools.map((tool, index) => (
                    <div 
                      key={index}
                      style={{ 
                        padding: '0.75rem',
                        background: 'var(--dark-bg)',
                        borderRadius: '8px',
                        marginBottom: '0.5rem',
                        border: '1px solid var(--card-border)'
                      }}
                    >
                      <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', marginBottom: '0.25rem' }}>
                        <span className={`badge ${
                          tool.Method === 'GET' ? 'badge-success' :
                          tool.Method === 'POST' ? 'badge-primary' :
                          tool.Method === 'DELETE' ? 'badge-danger' :
                          'badge-warning'
                        }`} style={{ fontSize: '0.7rem' }}>
                          {tool.Method}
                        </span>
                        <span style={{ color: 'var(--text-primary)', fontWeight: 500 }}>
                          {tool.Name}
                        </span>
                      </div>
                      <p style={{ 
                        color: 'var(--text-secondary)', 
                        fontSize: '0.8125rem', 
                        margin: 0,
                        whiteSpace: 'nowrap',
                        overflow: 'hidden',
                        textOverflow: 'ellipsis'
                      }}>
                        {tool.Description || tool.Path}
                      </p>
                    </div>
                  ))}
                </div>
              </div>

              {/* Create server — only action that creates */}
              <div style={{ marginTop: '1.5rem', paddingTop: '1rem', borderTop: '1px solid var(--card-border)' }}>
                <button 
                  className="btn btn-primary  text-center fw-bold" 
                  onClick={handleImport}
                  disabled={importing}
                  style={{ width: '100%', textAlign: 'center' }}
                >
                  {importing ? (
                    <>
                      <i className="bi bi-hourglass-split"></i>
                      Creating server...
                    </>
                  ) : (
                    <>
                      <i className="bi bi-plus-circle"></i>
                      Create server
                    </>
                  )}
                </button>
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Info Section */}
      <div className="card" style={{ marginTop: '1.5rem' }}>
        <div className="card-header">
          <h3 className="card-title">Supported APIs</h3>
        </div>
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(200px, 1fr))', gap: '1rem' }}>
          {[
            { name: 'GitHub API', icon: 'bi-github', color: '#6e5494' },
            { name: 'Stripe API', icon: 'bi-credit-card', color: '#6772e5' },
            { name: 'Slack API', icon: 'bi-slack', color: '#4a154b' },
            { name: 'Salesforce', icon: 'bi-cloud', color: '#00a1e0' },
            { name: 'Twilio', icon: 'bi-telephone', color: '#f22f46' },
            { name: 'Any OpenAPI 3.x', icon: 'bi-file-code', color: '#85ea2d' },
          ].map((api) => (
            <div 
              key={api.name}
              style={{ 
                padding: '1rem',
                background: 'var(--dark-bg)',
                borderRadius: '8px',
                display: 'flex',
                alignItems: 'center',
                gap: '0.75rem',
                border: '1px solid var(--card-border)'
              }}
            >
              <i className={`bi ${api.icon}`} style={{ fontSize: '1.5rem', color: api.color }}></i>
              <span style={{ color: 'var(--text-primary)' }}>{api.name}</span>
            </div>
          ))}
        </div>
        <p style={{ color: 'var(--text-secondary)', marginTop: '1rem', marginBottom: 0 }}>
          <i className="bi bi-info-circle" style={{ marginRight: '0.5rem' }}></i>
          Works with any valid OpenAPI 3.x specification (YAML or JSON). Find OpenAPI specs on{' '}
          <a href="https://apis.guru/" target="_blank" rel="noopener noreferrer" style={{ color: 'var(--primary-color)' }}>
            APIs.guru
          </a>{' '}
          or check your API provider's documentation.
        </p>
      </div>
    </div>
  );
}
