import { useState } from 'react';
import { toast } from 'react-toastify';
import Editor from '@monaco-editor/react';
import type { Tool, TestToolResponse } from '../types';
import { testTool } from '../services/api';

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
                <div>
                  <div style={{ fontWeight: 600, color: result.success ? 'var(--success-color)' : 'var(--danger-color)' }}>
                    {result.success ? 'Success' : 'Failed'}
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
                </div>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
