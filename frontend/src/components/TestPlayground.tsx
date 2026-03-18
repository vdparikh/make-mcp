import { useState, useEffect, type ReactNode } from 'react';
import { toast } from 'react-toastify';
import Editor from '@monaco-editor/react';
import type { Tool, TestToolResponse, MCPAppPayload, ToolTestPreset, PolicyEvaluationResult, PolicyEvaluationResultDetailed } from '../types';
import { isMCPAppOutput } from '../types';
import type { EnvProfileKey } from '../types';
import { testTool, listToolTestPresets, createToolTestPreset, deleteToolTestPreset, evaluatePolicy, evaluatePolicyDetailed } from '../services/api';

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
  serverId?: string;
  tools: Tool[];
  initialToolId?: string;
  /** When set, show "Edit environment profiles" link that switches to the Environments tab */
  onOpenEnvironments?: () => void;
}

export default function TestPlayground({ serverId, tools, initialToolId, onOpenEnvironments }: Props) {
  const [selectedTool, setSelectedTool] = useState<string>('');
  const [selectedEnvProfile, setSelectedEnvProfile] = useState<EnvProfileKey>('dev');
  const [input, setInput] = useState('{\n  \n}');
  const [context, setContext] = useState('{\n  "user_id": "user-123",\n  "organization_id": "org-456",\n  "roles": ["user"]\n}');
  const [result, setResult] = useState<TestToolResponse | null>(null);
  const [testing, setTesting] = useState(false);
  const [dryRun, setDryRun] = useState(false);
  const [presets, setPresets] = useState<ToolTestPreset[]>([]);
  const [selectedPresetId, setSelectedPresetId] = useState<string>('');
  const [presetsLoading, setPresetsLoading] = useState(false);
  const [policyEval, setPolicyEval] = useState<PolicyEvaluationResult | null>(null);
  const [whatIfOpen, setWhatIfOpen] = useState(false);
  const [whatIfInput, setWhatIfInput] = useState('');
  const [whatIfContext, setWhatIfContext] = useState('');
  const [whatIfResult, setWhatIfResult] = useState<PolicyEvaluationResultDetailed | null>(null);
  const [whatIfLoading, setWhatIfLoading] = useState(false);
  const [showContext, setShowContext] = useState(false);

  useEffect(() => {
    if (initialToolId && tools.some((t) => t.id === initialToolId)) {
      setSelectedTool(initialToolId);
      const tool = tools.find((t) => t.id === initialToolId);
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
    }
  }, [initialToolId, tools]);

  useEffect(() => {
    if (!selectedTool) {
      setPresets([]);
      return;
    }
    let cancelled = false;
    setPresetsLoading(true);
    listToolTestPresets(selectedTool)
      .then((list) => {
        if (!cancelled) setPresets(list);
      })
      .catch(() => {
        if (!cancelled) setPresets([]);
      })
      .finally(() => {
        if (!cancelled) setPresetsLoading(false);
      });
    return () => { cancelled = true; };
  }, [selectedTool]);

  const selectedToolData = tools.find(t => t.id === selectedTool);

  const hasEnvProfiles = Boolean(serverId);
  
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

      if (dryRun && selectedToolData?.destructive_hint) {
        const previewOutput = {
          dry_run: true,
          tool_id: selectedTool,
          execution_type: selectedToolData.execution_type,
          input: parsedInput,
          context: parsedContext,
        };
        setResult({
          success: true,
          output: previewOutput,
          duration_ms: 0,
        });
        try {
          const evalRes = await evaluatePolicy(selectedTool, parsedInput as Record<string, unknown>, parsedContext);
          setPolicyEval(evalRes);
        } catch {
          setPolicyEval(null);
        }
        toast.info('Dry-run preview only. External tool execution was skipped.');
        return;
      }

      const response = await testTool(selectedTool, parsedInput, parsedContext, hasEnvProfiles ? selectedEnvProfile : undefined);
      setResult(response);
      try {
        const evalRes = await evaluatePolicy(selectedTool, parsedInput as Record<string, unknown>, parsedContext);
        setPolicyEval(evalRes);
      } catch {
        setPolicyEval(null);
      }
      if (response.success) {
        toast.success('Tool executed successfully');
      } else {
        toast.warning('Tool execution failed - check healing suggestions');
      }
    } catch (error: unknown) {
      const err = error as { response?: { data?: { error?: string; reason?: string; violated_rules?: string[]; injected_context?: Record<string, unknown> } } };
      if (err.response?.data?.error === 'Policy violation') {
        setResult({
          success: false,
          error: `Policy Violation: ${err.response.data.reason}`,
          output: { violated_rules: err.response.data.violated_rules },
          duration_ms: 0,
          injected_context: err.response.data.injected_context,
        });
        try {
          let parsedInput: Record<string, unknown>, parsedContext: Record<string, unknown>;
          try { parsedInput = JSON.parse(input); } catch { parsedInput = {}; }
          try { parsedContext = JSON.parse(context); } catch { parsedContext = {}; }
          const evalRes = await evaluatePolicy(selectedTool, parsedInput, parsedContext);
          setPolicyEval(evalRes);
        } catch {
          setPolicyEval(null);
        }
        toast.error('Policy violation - action blocked');
      } else {
        setPolicyEval(null);
        toast.error('Failed to execute tool');
      }
    } finally {
      setTesting(false);
    }
  };

  const handleToolSelect = (toolId: string) => {
    setSelectedTool(toolId);
    setDryRun(false);
    setSelectedPresetId('');
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
    setPolicyEval(null);
    setWhatIfResult(null);
  };

  const fetchPresets = () => {
    if (!selectedTool) return;
    listToolTestPresets(selectedTool).then(setPresets).catch(() => setPresets([]));
  };

  const handleSavePreset = async () => {
    if (!selectedTool) {
      toast.error('Select a tool before saving a preset');
      return;
    }
    const name = window.prompt('Preset name', '');
    if (!name?.trim()) return;
    let parsedInput: Record<string, unknown>;
    let parsedContext: Record<string, unknown>;
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
    try {
      const created = await createToolTestPreset(selectedTool, {
        name: name.trim(),
        input: parsedInput,
        context: parsedContext,
      });
      fetchPresets();
      setSelectedPresetId(created.id);
      toast.success('Preset saved');
    } catch (err: unknown) {
      const e = err as { response?: { status: number; data?: { error?: string } } };
      if (e.response?.status === 401) {
        toast.error('Sign in to save presets');
      } else {
        toast.error(e.response?.data?.error ?? 'Failed to save preset');
      }
    }
  };

  const handleApplyPreset = (presetId: string) => {
    const preset = presets.find((p) => p.id === presetId);
    if (!preset) return;
    setSelectedPresetId(presetId);
    setInput(JSON.stringify(preset.input_json ?? {}, null, 2));
    setContext(JSON.stringify(preset.context_json ?? {}, null, 2));
    setResult(null);
    toast.info(`Loaded preset "${preset.name}"`);
  };

  const runWhatIf = async () => {
    if (!selectedTool) return;
    let parsedInput: Record<string, unknown>, parsedContext: Record<string, unknown>;
    try {
      parsedInput = JSON.parse(whatIfInput || '{}');
    } catch {
      toast.error('Invalid What-if Input JSON');
      return;
    }
    try {
      parsedContext = JSON.parse(whatIfContext || '{}');
    } catch {
      toast.error('Invalid What-if Context JSON');
      return;
    }
    setWhatIfLoading(true);
    setWhatIfResult(null);
    try {
      const detailed = await evaluatePolicyDetailed(selectedTool, parsedInput, parsedContext);
      setWhatIfResult(detailed);
    } catch {
      toast.error('Policy simulation failed');
      setWhatIfResult(null);
    } finally {
      setWhatIfLoading(false);
    }
  };

  const openWhatIf = () => {
    setWhatIfInput(input);
    setWhatIfContext(context);
    setWhatIfResult(null);
    setWhatIfOpen(true);
  };

  const handleDeletePreset = async () => {
    if (!selectedTool || !selectedPresetId) return;
    const preset = presets.find((p) => p.id === selectedPresetId);
    if (preset && !window.confirm(`Delete preset "${preset.name}"?`)) return;
    try {
      await deleteToolTestPreset(selectedTool, selectedPresetId);
      fetchPresets();
      setSelectedPresetId('');
      toast.success('Preset deleted');
    } catch (err: unknown) {
      const e = err as { response?: { status: number; data?: { error?: string } } };
      if (e.response?.status === 401) {
        toast.error('Sign in to delete presets');
      } else {
        toast.error(e.response?.data?.error ?? 'Failed to delete preset');
      }
    }
  };

  return (
    <div>
      <div style={{ marginBottom: '1.25rem', display: 'flex', alignItems: 'center', justifyContent: 'space-between', flexWrap: 'wrap', gap: '0.5rem' }}>
        <h3 className="card-title" style={{ margin: 0 }}>
          <i className="bi bi-play-circle" style={{ marginRight: '0.5rem', color: 'var(--secondary-color)' }}></i>
          Testing
        </h3>
        <a
          href="https://github.com/vdparikh/make-mcp/blob/main/docs/creating-servers.md#3-test-your-tools"
          target="_blank"
          rel="noopener noreferrer"
          style={{ fontSize: '0.8125rem', color: 'var(--text-muted)', textDecoration: 'none' }}
        >
          Best practices
        </a>
      </div>

      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '1.5rem' }}>
        <div>
          <div style={{ display: 'flex', gap: '0.75rem', alignItems: 'flex-end', flexWrap: 'wrap', marginBottom: '1rem' }}>
            <div className="form-group" style={{ flex: '1 1 200px', marginBottom: 0 }}>
              <label className="form-label">Tool</label>
              <select
                className="form-control"
                value={selectedTool}
                onChange={(e) => handleToolSelect(e.target.value)}
              >
                <option value="">Select a tool...</option>
                {tools.map((tool) => (
                  <option key={tool.id} value={tool.id}>
                    {tool.name} ({tool.execution_type})
                  </option>
                ))}
              </select>
            </div>
            {hasEnvProfiles && (
              <div className="form-group" style={{ marginBottom: 0, minWidth: '120px' }}>
                <label className="form-label">Environment</label>
                <select
                  className="form-control"
                  value={selectedEnvProfile}
                  onChange={(e) => setSelectedEnvProfile(e.target.value as EnvProfileKey)}
                >
                  <option value="dev">Dev</option>
                  <option value="staging">Staging</option>
                  <option value="prod">Prod</option>
                </select>
              </div>
            )}
            {hasEnvProfiles && onOpenEnvironments && (
              <button type="button" className="btn btn-link btn-sm" style={{ padding: 0, fontSize: '0.8125rem', marginBottom: '0.25rem' }} onClick={onOpenEnvironments}>
                Edit profiles
              </button>
            )}
          </div>

          {selectedToolData && (
            <div style={{ marginBottom: '1rem', padding: '0.75rem 1rem', background: 'var(--dark-bg)', borderRadius: '8px', border: '1px solid var(--card-border)' }}>
              <p style={{ fontSize: '0.8125rem', color: 'var(--text-secondary)', margin: 0 }}>
                {selectedToolData.description || 'No description'}
                {selectedToolData.context_fields && selectedToolData.context_fields.length > 0 && (
                  <span style={{ marginLeft: '0.5rem', color: 'var(--text-muted)' }}>
                    · Context: {selectedToolData.context_fields.join(', ')}
                  </span>
                )}
              </p>
              {isMocked && (
                <span style={{ fontSize: '0.75rem', color: 'var(--warning-color)', marginTop: '0.25rem', display: 'inline-block' }}>
                  Simulated here; runs live in exported server.
                </span>
              )}
            </div>
          )}

          {selectedTool && (
            <div style={{ display: 'flex', gap: '0.5rem', alignItems: 'center', flexWrap: 'wrap', marginBottom: '1rem' }}>
              <select
                className="form-control"
                style={{ width: 'auto', minWidth: '140px' }}
                value={selectedPresetId}
                onChange={(e) => handleApplyPreset(e.target.value)}
                disabled={presetsLoading}
              >
                <option value="">{presetsLoading ? '…' : 'Preset…'}</option>
                {presets.map((preset) => (
                  <option key={preset.id} value={preset.id}>{preset.name}</option>
                ))}
              </select>
              <button type="button" className="btn btn-outline-primary btn-sm" onClick={handleSavePreset} title="Save current input + context as preset">
                <i className="bi bi-bookmark-plus"></i>
              </button>
              <button type="button" className="btn btn-outline-secondary btn-sm" onClick={handleDeletePreset} disabled={!selectedPresetId} title="Delete preset">
                <i className="bi bi-trash"></i>
              </button>
            </div>
          )}

          <div className="form-group" style={{ marginBottom: '0.75rem' }}>
            <label className="form-label">Input</label>
            <div className="editor-container">
              <Editor
                height="180px"
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

          <div style={{ marginBottom: '1rem' }}>
            <button
              type="button"
              className="btn btn-link btn-sm"
              style={{ padding: 0, fontSize: '0.8125rem', marginBottom: '0.25rem' }}
              onClick={() => setShowContext(!showContext)}
            >
              {showContext ? 'Hide' : 'Show'} context (simulated)
            </button>
            {showContext && (
              <div className="editor-container">
                <Editor
                  height="120px"
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
            )}
          </div>

          {selectedToolData?.destructive_hint && (
            <label style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', fontSize: '0.8125rem', cursor: 'pointer', marginBottom: '1rem' }}>
              <input type="checkbox" checked={dryRun} onChange={(e) => setDryRun(e.target.checked)} style={{ margin: 0 }} />
              <span>Dry-run (preview only, no execution)</span>
            </label>
          )}

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

              {policyEval && (
                <div style={{
                  background: policyEval.allowed && !policyEval.requires_approval ? 'rgba(16, 185, 129, 0.08)' : policyEval.requires_approval ? 'rgba(245, 158, 11, 0.1)' : 'rgba(239, 68, 68, 0.08)',
                  border: `1px solid ${policyEval.allowed && !policyEval.requires_approval ? 'rgba(16, 185, 129, 0.25)' : policyEval.requires_approval ? 'rgba(245, 158, 11, 0.3)' : 'rgba(239, 68, 68, 0.25)'}`,
                  borderRadius: '8px',
                  padding: '1rem',
                  marginBottom: '1rem',
                }}>
                  <div style={{ fontWeight: 600, fontSize: '0.875rem', marginBottom: '0.5rem', display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
                    <i className={`bi ${policyEval.allowed && !policyEval.requires_approval ? 'bi-shield-check' : policyEval.requires_approval ? 'bi-shield-exclamation' : 'bi-shield-x'}`}
                       style={{ color: policyEval.allowed && !policyEval.requires_approval ? 'var(--success-color)' : policyEval.requires_approval ? 'var(--warning-color)' : 'var(--danger-color)' }}></i>
                    Policy decision: {policyEval.allowed && !policyEval.requires_approval ? 'Allowed' : policyEval.requires_approval ? 'Approval required' : 'Denied'}
                  </div>
                  {policyEval.reason && <div style={{ fontSize: '0.8125rem', color: 'var(--text-secondary)', marginBottom: '0.375rem' }}>{policyEval.reason}</div>}
                  {policyEval.approval_reason && <div style={{ fontSize: '0.8125rem', color: 'var(--warning-color)' }}>{policyEval.approval_reason}</div>}
                  {policyEval.violated_rules && policyEval.violated_rules.length > 0 && (
                    <ul style={{ margin: '0.5rem 0 0 1rem', padding: 0, fontSize: '0.8125rem', color: 'var(--text-secondary)' }}>
                      {policyEval.violated_rules.map((r, i) => <li key={i}>{r}</li>)}
                    </ul>
                  )}
                  {policyEval.allowed && !policyEval.requires_approval && (!policyEval.violated_rules || policyEval.violated_rules.length === 0) && (
                    <div style={{ fontSize: '0.8125rem', color: 'var(--text-muted)' }}>All rules passed.</div>
                  )}
                </div>
              )}

              {result.injected_context && Object.keys(result.injected_context).length > 0 && (
                <div style={{ 
                  background: 'var(--hover-bg)', 
                  border: '1px solid var(--card-border)', 
                  borderRadius: '8px', 
                  padding: '0.75rem 1rem', 
                  marginBottom: '1rem' 
                }}>
                  <div style={{ fontWeight: 600, fontSize: '0.8125rem', color: 'var(--text-secondary)', marginBottom: '0.375rem' }}>
                    <i className="bi bi-person-badge" style={{ marginRight: '0.5rem', color: 'var(--primary-color)' }}></i>
                    Context passed to tool
                  </div>
                  <pre style={{ margin: 0, whiteSpace: 'pre-wrap', fontSize: '0.75rem' }}>
                    {JSON.stringify(result.injected_context, null, 2)}
                  </pre>
                </div>
              )}

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

          {selectedTool && (
            <div style={{ marginTop: '1.5rem', borderTop: '1px solid var(--card-border)', paddingTop: '1rem' }}>
              {!whatIfOpen ? (
                <button type="button" className="btn btn-outline-secondary btn-sm" onClick={openWhatIf}>
                  <i className="bi bi-question-circle" style={{ marginRight: '0.25rem' }}></i>
                  What if? Simulate policy with different input/context
                </button>
              ) : (
                <div style={{ background: 'var(--dark-bg)', borderRadius: '8px', padding: '1rem', border: '1px solid var(--card-border)' }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '0.75rem' }}>
                    <span style={{ fontWeight: 600, fontSize: '0.875rem' }}>What if? — Policy simulation</span>
                    <button type="button" className="btn btn-icon btn-secondary btn-sm" onClick={() => { setWhatIfOpen(false); setWhatIfResult(null); }}>
                      <i className="bi bi-x-lg"></i>
                    </button>
                  </div>
                  <p style={{ fontSize: '0.8125rem', color: 'var(--text-muted)', marginBottom: '0.75rem' }}>
                    Change input or context below and click Simulate to see which rules would pass or fail (no tool execution).
                  </p>
                  <div className="form-group" style={{ marginBottom: '0.5rem' }}>
                    <label className="form-label" style={{ fontSize: '0.75rem' }}>Input (JSON)</label>
                    <Editor height="80px" language="json" theme="vs-dark" value={whatIfInput} onChange={(v) => setWhatIfInput(v || '')}
                      options={{ minimap: { enabled: false }, fontSize: 12, lineNumbers: 'off' }} />
                  </div>
                  <div className="form-group" style={{ marginBottom: '0.75rem' }}>
                    <label className="form-label" style={{ fontSize: '0.75rem' }}>Context (JSON)</label>
                    <Editor height="80px" language="json" theme="vs-dark" value={whatIfContext} onChange={(v) => setWhatIfContext(v || '')}
                      options={{ minimap: { enabled: false }, fontSize: 12, lineNumbers: 'off' }} />
                  </div>
                  <button type="button" className="btn btn-primary btn-sm" onClick={runWhatIf} disabled={whatIfLoading}>
                    {whatIfLoading ? 'Simulating...' : 'Simulate'}
                  </button>
                  {whatIfResult && (
                    <div style={{ marginTop: '1rem' }}>
                      <div style={{ fontSize: '0.8125rem', fontWeight: 600, marginBottom: '0.5rem', color: whatIfResult.allowed && !whatIfResult.requires_approval ? 'var(--success-color)' : 'var(--danger-color)' }}>
                        Result: {whatIfResult.allowed && !whatIfResult.requires_approval ? 'Allowed' : whatIfResult.requires_approval ? 'Approval required' : 'Denied'}
                        {whatIfResult.reason && ` — ${whatIfResult.reason}`}
                      </div>
                      {(whatIfResult.rule_results?.length ?? 0) > 0 && (
                        <ul style={{ margin: 0, paddingLeft: '1.25rem', fontSize: '0.8125rem' }}>
                          {whatIfResult.rule_results.map((r, i) => (
                            <li key={i} style={{ marginBottom: '0.25rem', color: r.passed ? 'var(--text-secondary)' : 'var(--danger-color)' }}>
                              <strong>{r.policy_name}</strong> · {r.rule_type}: {r.passed ? 'Passed' : r.message || 'Failed'}
                            </li>
                          ))}
                        </ul>
                      )}
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
