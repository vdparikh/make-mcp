import { useState } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { toast } from 'react-toastify';
import Editor from '@monaco-editor/react';
import { previewOpenAPIImport, importOpenAPI, type OpenAPIPreview } from '../services/api';

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
      navigate(`/server/${result.server.id}`);
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

  return (
    <div className="page">
      <div className="page-header">
        <div>
          <nav style={{ marginBottom: '0.5rem' }}>
            <Link to="/" style={{ color: 'var(--text-muted)', textDecoration: 'none', fontSize: '0.875rem' }}>
              Dashboard
            </Link>
            <span style={{ color: 'var(--text-muted)', margin: '0 0.5rem' }}>/</span>
            <span style={{ color: 'var(--text-primary)', fontSize: '0.875rem' }}>Import OpenAPI</span>
          </nav>
          <h1 className="page-title">
            <i className="bi bi-file-earmark-code" style={{ marginRight: '0.75rem' }}></i>
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
                  Parsing...
                </>
              ) : (
                <>
                  <i className="bi bi-search"></i>
                  Preview Import
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
            <h3 className="card-title">Import Preview</h3>
          </div>

          {!preview ? (
            <div className="empty-state">
              <i className="bi bi-arrow-left-circle"></i>
              <h3>Paste & Preview</h3>
              <p>Paste an OpenAPI spec and click "Preview Import" to see what will be created</p>
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

              {/* Import Button */}
              <div style={{ marginTop: '1.5rem', paddingTop: '1rem', borderTop: '1px solid var(--card-border)' }}>
                <button 
                  className="btn btn-primary" 
                  onClick={handleImport}
                  disabled={importing}
                  style={{ width: '100%' }}
                >
                  {importing ? (
                    <>
                      <i className="bi bi-hourglass-split"></i>
                      Creating Server...
                    </>
                  ) : (
                    <>
                      <i className="bi bi-cloud-upload"></i>
                      Import & Create Server
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
