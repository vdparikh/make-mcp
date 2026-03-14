import { useState, type ReactNode } from 'react';
import { toast } from 'react-toastify';
import Editor from '@monaco-editor/react';
import type { Tool, TestToolResponse, MCPAppPayload } from '../types';
import { isMCPAppOutput } from '../types';
import { testTool } from '../services/api';

/** Format a table cell value so objects/arrays show as JSON instead of "[object Object]" */
function formatTableCellValue(value: unknown): ReactNode {
  if (value == null) return '—';
  if (typeof value === 'object') {
    try {
      const json = JSON.stringify(value, null, 2);
      return (
        <pre style={{
          margin: 0,
          fontSize: '0.75rem',
          whiteSpace: 'pre-wrap',
          wordBreak: 'break-word',
          maxHeight: '120px',
          overflow: 'auto',
          fontFamily: 'inherit',
        }}>
          {json}
        </pre>
      );
    } catch {
      return String(value);
    }
  }
  return String(value);
}

/** Renders MCP Apps card widget: main content in large type (e.g. joke, quote). */
function MCPAppCardWidget({ payload }: { payload: Extract<MCPAppPayload, { widget: 'card' }> }) {
  const { content, title } = payload.props;
  return (
    <div style={{
      background: 'var(--dark-bg)',
      borderRadius: '12px',
      overflow: 'hidden',
      border: '1px solid var(--card-border)',
      marginTop: '0.5rem',
      padding: '2rem 2.5rem',
      maxWidth: '100%',
    }}>
      {title && (
        <div style={{
          fontSize: '0.75rem',
          fontWeight: 600,
          textTransform: 'uppercase',
          letterSpacing: '0.08em',
          color: 'var(--text-muted)',
          marginBottom: '1rem',
        }}>
          {title}
        </div>
      )}
      <div style={{
        fontSize: 'clamp(1.25rem, 4vw, 2rem)',
        lineHeight: 1.5,
        fontWeight: 500,
        color: 'var(--text-primary)',
        wordBreak: 'break-word',
      }}>
        {content}
      </div>
      <div style={{
        marginTop: '1.25rem',
        paddingTop: '1rem',
        borderTop: '1px solid var(--card-border)',
        fontSize: '0.75rem',
        color: 'var(--text-muted)',
      }}>
        <i className="bi bi-card-text" style={{ marginRight: '0.5rem' }}></i>
        MCP App: Card
      </div>
    </div>
  );
}

/** Renders MCP Apps table widget. Single row → Key/Value layout; multiple rows → data table. */
function MCPAppTableWidget({ payload }: { payload: Extract<MCPAppPayload, { widget: 'table' }> }) {
  const { columns, rows } = payload.props;
  const isSingleRow = rows.length === 1;
  const singleRow = isSingleRow ? rows[0] : null;
  const keyValuePairs = singleRow
    ? columns.map((col) => ({ key: col.label, value: singleRow[col.key] }))
    : [];

  return (
    <div style={{
      background: 'var(--dark-bg)',
      borderRadius: '8px',
      overflow: 'hidden',
      border: '1px solid var(--card-border)',
      marginTop: '0.5rem',
      maxWidth: '100%',
    }}>
      <div style={{
        overflowX: 'auto',
        maxHeight: isSingleRow ? 'none' : '400px',
        overflowY: isSingleRow ? 'visible' : 'auto',
      }}>
        {isSingleRow && keyValuePairs.length > 0 ? (
          <table style={{
            width: '100%',
            borderCollapse: 'collapse',
            fontSize: '0.875rem',
            tableLayout: 'fixed',
          }}>
            <thead style={{
              background: '#1e293b',
              color: '#94a3b8',
              zIndex: 1,
            }}>
              <tr>
                <th style={{
                  padding: '0.5rem 1rem',
                  textAlign: 'left',
                  fontWeight: 600,
                  width: '140px',
                  borderBottom: '1px solid #334155',
                  fontSize: '0.75rem',
                  textTransform: 'uppercase',
                  letterSpacing: '0.05em',
                }}>Key</th>
                <th style={{
                  padding: '0.5rem 1rem',
                  textAlign: 'left',
                  fontWeight: 600,
                  borderBottom: '1px solid #334155',
                  fontSize: '0.75rem',
                  textTransform: 'uppercase',
                  letterSpacing: '0.05em',
                }}>Value</th>
              </tr>
            </thead>
            <tbody>
              {keyValuePairs.map(({ key, value }, i) => (
                <tr
                  key={key}
                  style={{
                    background: i % 2 === 0 ? '#1e293b' : '#0f172a',
                    color: '#e2e8f0',
                  }}
                >
                  <td style={{
                    padding: '0.5rem 1rem',
                    borderBottom: '1px solid #334155',
                    verticalAlign: 'top',
                    color: '#94a3b8',
                    fontWeight: 500,
                    width: '140px',
                  }}>{key}</td>
                  <td style={{
                    padding: '0.5rem 1rem',
                    borderBottom: '1px solid #334155',
                    verticalAlign: 'top',
                    wordBreak: 'break-word',
                  }}>{formatTableCellValue(value)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        ) : (
          <table style={{
            width: '100%',
            borderCollapse: 'collapse',
            fontSize: '0.8125rem',
            minWidth: 'min-content',
          }}>
            <thead style={{
              position: 'sticky',
              top: 0,
              background: '#1e293b',
              color: '#e2e8f0',
              zIndex: 1,
            }}>
              <tr>
                {columns.map((col) => (
                  <th
                    key={col.key}
                    style={{
                      padding: '0.5rem 0.75rem',
                      textAlign: 'left',
                      fontWeight: 600,
                      borderBottom: '1px solid #334155',
                      whiteSpace: 'nowrap',
                      fontSize: '0.75rem',
                    }}
                  >
                    {col.label}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {rows.map((row, i) => (
                <tr
                  key={i}
                  style={{
                    background: i % 2 === 0 ? '#1e293b' : '#0f172a',
                    color: '#e2e8f0',
                  }}
                >
                  {columns.map((col) => (
                    <td
                      key={col.key}
                      style={{
                        padding: '0.5rem 0.75rem',
                        borderBottom: '1px solid #334155',
                        verticalAlign: 'top',
                        maxWidth: '280px',
                        wordBreak: 'break-word',
                      }}
                    >
                      {formatTableCellValue(row[col.key])}
                    </td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
      <div style={{
        padding: '0.5rem 1rem',
        fontSize: '0.75rem',
        color: 'var(--text-muted)',
        background: '#0f172a',
        borderTop: '1px solid #334155',
      }}>
        <i className="bi bi-table" style={{ marginRight: '0.5rem' }}></i>
        {isSingleRow ? 'MCP App: Key-value' : `MCP App: Table · ${rows.length} rows`}
      </div>
    </div>
  );
}

interface Props {
  tools: Tool[];
}

export default function TestPlayground({ tools }: Props) {
  const [selectedTool, setSelectedTool] = useState<string>('');
  const [input, setInput] = useState('{\n  \n}');
  const [context, setContext] = useState('{\n  "user_id": "user-123",\n  "organization_id": "org-456",\n  "roles": ["user"]\n}');
  const [result, setResult] = useState<TestToolResponse | null>(null);
  const [testing, setTesting] = useState(false);

  const selectedToolData = tools.find(t => t.id === selectedTool);
  
  const isLiveExecutable = (type: string) => ['rest_api', 'webhook'].includes(type);
  const isMocked = selectedToolData && !isLiveExecutable(selectedToolData.execution_type);

  const handleTest = async () => {
    if (!selectedTool) {
      toast.error('Please select a tool');
      return;
    }

    try {
      setTesting(true);
      setResult(null);

      let parsedInput, parsedContext;
      try {
        parsedInput = JSON.parse(input);
      } catch {
        toast.error('Invalid Input JSON');
        return;
      }
      try {
        parsedContext = JSON.parse(context);
      } catch {
        toast.error('Invalid Context JSON');
        return;
      }

      const response = await testTool(selectedTool, parsedInput, parsedContext);
      setResult(response);

      if (response.success) {
        toast.success('Tool executed successfully');
      } else {
        toast.warning('Tool execution failed - check healing suggestions');
      }
    } catch (error: unknown) {
      const err = error as { response?: { data?: { error?: string; reason?: string; violated_rules?: string[] } } };
      if (err.response?.data?.error === 'Policy violation') {
        setResult({
          success: false,
          error: `Policy Violation: ${err.response.data.reason}`,
          output: { violated_rules: err.response.data.violated_rules },
          duration_ms: 0,
        });
        toast.error('Policy violation - action blocked');
      } else {
        toast.error('Failed to execute tool');
      }
    } finally {
      setTesting(false);
    }
  };

  const handleToolSelect = (toolId: string) => {
    setSelectedTool(toolId);
    const tool = tools.find(t => t.id === toolId);
    if (tool?.input_schema) {
      const schema = tool.input_schema as { properties?: Record<string, { type: string }> };
      const exampleInput: Record<string, unknown> = {};
      if (schema.properties) {
        Object.entries(schema.properties).forEach(([key, value]) => {
          switch (value.type) {
            case 'string': exampleInput[key] = ''; break;
            case 'number': exampleInput[key] = 0; break;
            case 'boolean': exampleInput[key] = false; break;
            default: exampleInput[key] = null;
          }
        });
      }
      setInput(JSON.stringify(exampleInput, null, 2));
    }
    setResult(null);
  };

  return (
    <div>
      <div className="card" style={{ marginBottom: '1.5rem' }}>
        <h3 className="card-title" style={{ marginBottom: '0.75rem' }}>
          <i className="bi bi-play-circle" style={{ marginRight: '0.75rem', color: 'var(--secondary-color)' }}></i>
          Live Testing Playground
        </h3>
        <p style={{ color: 'var(--text-secondary)' }}>
          Test your tools with mock input before deployment. The playground validates inputs, 
          applies governance policies, injects context, and shows healing suggestions on failure.
        </p>
      </div>

      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '1.5rem' }}>
        <div>
          <div className="form-group">
            <label className="form-label">Select Tool</label>
            <select
              className="form-control"
              value={selectedTool}
              onChange={(e) => handleToolSelect(e.target.value)}
            >
              <option value="">Select a tool to test...</option>
              {tools.map((tool) => (
                <option key={tool.id} value={tool.id}>
                  {tool.name} ({tool.execution_type})
                </option>
              ))}
            </select>
          </div>

          {selectedToolData && (
            <div style={{ 
              background: 'var(--dark-bg)', 
              borderRadius: '8px', 
              padding: '1rem',
              marginBottom: '1rem'
            }}>
              <h4 style={{ fontSize: '0.875rem', marginBottom: '0.5rem' }}>
                {selectedToolData.name}
              </h4>
              <p style={{ fontSize: '0.8125rem', color: 'var(--text-secondary)', margin: 0 }}>
                {selectedToolData.description || 'No description'}
              </p>
              {selectedToolData.context_fields && selectedToolData.context_fields.length > 0 && (
                <div style={{ marginTop: '0.5rem' }}>
                  <span className="badge badge-success">
                    <i className="bi bi-person-badge" style={{ marginRight: '0.25rem' }}></i>
                    Uses context: {selectedToolData.context_fields.join(', ')}
                  </span>
                </div>
              )}
              {isMocked && (
                <div style={{ 
                  marginTop: '0.75rem',
                  padding: '0.625rem 0.75rem',
                  background: 'rgba(245, 158, 11, 0.1)',
                  border: '1px solid rgba(245, 158, 11, 0.3)',
                  borderRadius: '6px',
                  fontSize: '0.8125rem',
                  color: 'var(--warning-color)'
                }}>
                  <i className="bi bi-info-circle" style={{ marginRight: '0.5rem' }}></i>
                  <strong>{selectedToolData.execution_type.toUpperCase()}</strong> tools are simulated in the playground. 
                  They will execute for real in the exported server.
                </div>
              )}
            </div>
          )}

          <div className="form-group">
            <label className="form-label">Input (JSON)</label>
            <div className="editor-container">
              <Editor
                height="200px"
                language="json"
                theme="vs-dark"
                value={input}
                onChange={(value) => setInput(value || '')}
                options={{
                  minimap: { enabled: false },
                  fontSize: 13,
                  lineNumbers: 'off',
                }}
              />
            </div>
          </div>

          <div className="form-group">
            <label className="form-label">
              Context (Simulated)
              <span style={{ fontWeight: 'normal', color: 'var(--text-muted)', marginLeft: '0.5rem' }}>
                (user_id, organization_id, roles, permissions)
              </span>
            </label>
            <div className="editor-container">
              <Editor
                height="150px"
                language="json"
                theme="vs-dark"
                value={context}
                onChange={(value) => setContext(value || '')}
                options={{
                  minimap: { enabled: false },
                  fontSize: 13,
                  lineNumbers: 'off',
                }}
              />
            </div>
          </div>

          <button 
            className="btn btn-success" 
            onClick={handleTest}
            disabled={!selectedTool || testing}
            style={{ width: '100%' }}
          >
            {testing ? (
              <>
                <span className="spinner" style={{ width: 16, height: 16, borderWidth: 2 }}></span>
                Executing...
              </>
            ) : (
              <>
                <i className="bi bi-play-fill"></i>
                Execute Tool
              </>
            )}
          </button>
        </div>

        <div>
          <label className="form-label">Result</label>
          
          {!result ? (
            <div style={{ 
              background: 'var(--dark-bg)', 
              borderRadius: '8px', 
              padding: '3rem',
              textAlign: 'center',
              color: 'var(--text-muted)',
              height: 'calc(100% - 28px)'
            }}>
              <i className="bi bi-terminal" style={{ fontSize: '2rem', marginBottom: '1rem', display: 'block' }}></i>
              <p>Execute a tool to see results here</p>
            </div>
          ) : (
            <div>
              <div style={{ 
                display: 'flex', 
                alignItems: 'center', 
                gap: '0.75rem',
                marginBottom: '1rem',
                padding: '1rem',
                background: result.success ? 'rgba(16, 185, 129, 0.1)' : 'rgba(239, 68, 68, 0.1)',
                border: `1px solid ${result.success ? 'rgba(16, 185, 129, 0.3)' : 'rgba(239, 68, 68, 0.3)'}`,
                borderRadius: '8px'
              }}>
                <i className={`bi ${result.success ? 'bi-check-circle-fill' : 'bi-x-circle-fill'}`} 
                   style={{ fontSize: '1.5rem', color: result.success ? 'var(--success-color)' : 'var(--danger-color)' }}></i>
                <div style={{ flex: 1 }}>
                  <div style={{ fontWeight: 600, color: result.success ? 'var(--success-color)' : 'var(--danger-color)' }}>
                    {result.success ? 'Success' : 'Failed'}
                    {isMocked && result.success && (
                      <span style={{ 
                        marginLeft: '0.5rem',
                        padding: '0.125rem 0.5rem',
                        background: 'rgba(245, 158, 11, 0.15)',
                        color: 'var(--warning-color)',
                        borderRadius: '4px',
                        fontSize: '0.6875rem',
                        fontWeight: 500,
                        textTransform: 'uppercase'
                      }}>
                        Simulated
                      </span>
                    )}
                  </div>
                  <div style={{ fontSize: '0.8125rem', color: 'var(--text-secondary)' }}>
                    Execution time: {result.duration_ms}ms
                  </div>
                </div>
              </div>

              {result.error && (
                <div style={{ 
                  background: 'rgba(239, 68, 68, 0.1)',
                  border: '1px solid rgba(239, 68, 68, 0.3)',
                  borderRadius: '8px',
                  padding: '1rem',
                  marginBottom: '1rem'
                }}>
                  <div style={{ fontWeight: 600, color: 'var(--danger-color)', marginBottom: '0.5rem' }}>
                    <i className="bi bi-exclamation-triangle" style={{ marginRight: '0.5rem' }}></i>
                    Error
                  </div>
                  <pre style={{ 
                    margin: 0, 
                    whiteSpace: 'pre-wrap', 
                    fontSize: '0.8125rem',
                    color: 'var(--text-secondary)'
                  }}>
                    {result.error}
                  </pre>
                </div>
              )}

              {result.output && (
                <div>
                  <label className="form-label" style={{ marginTop: '1rem' }}>Output</label>
                  {isMCPAppOutput(result.output) && result.output._mcp_app.widget === 'card' ? (
                    <MCPAppCardWidget payload={result.output._mcp_app} />
                  ) : isMCPAppOutput(result.output) && result.output._mcp_app.widget === 'table' ? (
                    <MCPAppTableWidget payload={result.output._mcp_app} />
                  ) : (
                    <div className="editor-container">
                      <Editor
                        height="300px"
                        language="json"
                        theme="vs-dark"
                        value={JSON.stringify(result.output, null, 2)}
                        options={{
                          minimap: { enabled: false },
                          fontSize: 13,
                          lineNumbers: 'off',
                          readOnly: true,
                        }}
                      />
                    </div>
                  )}
                </div>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
