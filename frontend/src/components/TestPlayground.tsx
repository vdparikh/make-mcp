import { useState, useEffect, useCallback, type ReactNode, type FormEvent } from 'react';
import { toast } from 'react-toastify';
import Editor from '@monaco-editor/react';
import type { Tool, TestToolResponse, MCPAppPayload, ToolTestPreset, PolicyEvaluationResult, PolicyEvaluationResultDetailed } from '../types';
import { isMCPAppOutput } from '../types';
import type { EnvProfileKey } from '../types';
import { testTool, listToolTestPresets, createToolTestPreset, deleteToolTestPreset, evaluatePolicy, evaluatePolicyDetailed } from '../services/api';
import ConfirmModal from './ConfirmModal';

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

/** Renders MCP Apps image widget (remote http/https URL). */
function MCPAppImageWidget({ payload }: { payload: Extract<MCPAppPayload, { widget: 'image' }> }) {
  const { imageUrl, title, alt } = payload.props;
  return (
    <div style={{
      background: 'var(--dark-bg)',
      borderRadius: '12px',
      overflow: 'hidden',
      border: '1px solid var(--card-border)',
      marginTop: '0.5rem',
      maxWidth: '100%',
    }}>
      {title && (
        <div style={{
          padding: '0.75rem 1rem',
          fontSize: '0.875rem',
          fontWeight: 600,
          borderBottom: '1px solid var(--card-border)',
        }}>
          {title}
        </div>
      )}
      <div style={{ padding: '1rem', textAlign: 'center' }}>
        {/* eslint-disable-next-line jsx-a11y/img-redundant-alt -- alt from tool payload */}
        <img
          src={imageUrl}
          alt={alt}
          style={{
            maxWidth: '100%',
            maxHeight: '420px',
            height: 'auto',
            borderRadius: 8,
            objectFit: 'contain',
          }}
        />
      </div>
      <div style={{
        padding: '0.5rem 1rem',
        fontSize: '0.75rem',
        color: 'var(--text-muted)',
        borderTop: '1px solid var(--card-border)',
      }}>
        <i className="bi bi-image" style={{ marginRight: '0.5rem' }}></i>
        MCP App: Image
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

const CHART_PALETTE = ['#2563eb', '#16a34a', '#d97706', '#dc2626', '#7c3aed', '#0891b2', '#db2777', '#4b5563'];

/** MCP Apps chart widget (SVG bar / line). */
function MCPAppChartWidget({ payload }: { payload: Extract<MCPAppPayload, { widget: 'chart' }> }) {
  const { chartType, title, labels, datasets } = payload.props;
  const isLine = chartType === 'line';
  const W = 640;
  const H = 320;
  const pad = { t: title ? 44 : 28, r: 20, b: 56, l: 52 };
  const innerW = W - pad.l - pad.r;
  const innerH = H - pad.t - pad.b;
  let maxV = 0;
  for (const ds of datasets) {
    for (const v of ds.data) {
      if (Number.isFinite(v) && v > maxV) maxV = v;
    }
  }
  if (maxV <= 0) maxV = 1;
  const n = Math.max(1, labels.length);
  const m = Math.max(1, datasets.length);
  const xCat = (i: number) => pad.l + (innerW * (i + 0.5)) / n;
  const yScale = (v: number) => pad.t + innerH - (innerH * Math.max(0, v)) / maxV;

  return (
    <div style={{
      background: 'var(--dark-bg)',
      borderRadius: '8px',
      overflow: 'hidden',
      border: '1px solid var(--card-border)',
      marginTop: '0.5rem',
      maxWidth: '100%',
      padding: '1rem',
    }}>
      {title ? <div style={{ fontWeight: 600, marginBottom: '0.75rem' }}>{title}</div> : null}
      <svg
        viewBox={`0 0 ${W} ${H}`}
        style={{ width: '100%', height: 'auto', maxHeight: 360, background: '#0f172a', borderRadius: 8 }}
        role="img"
        aria-label={title || 'Chart'}
      >
        <line x1={pad.l} y1={pad.t + innerH} x2={pad.l + innerW} y2={pad.t + innerH} stroke="#334155" strokeWidth={1} />
        <line x1={pad.l} y1={pad.t} x2={pad.l} y2={pad.t + innerH} stroke="#334155" strokeWidth={1} />
        {isLine ? (
          <>
            {datasets.map((ds, si) => {
              const pts = ds.data
                .map((v, i) => `${xCat(i)},${yScale(v)}`)
                .join(' ');
              const color = CHART_PALETTE[si % CHART_PALETTE.length];
              return (
                <polyline key={si} fill="none" stroke={color} strokeWidth={2} points={pts} />
              );
            })}
            {datasets.map((ds, si) =>
              ds.data.map((v, i) => (
                <circle
                  key={`${si}-${i}`}
                  cx={xCat(i)}
                  cy={yScale(v)}
                  r={4}
                  fill={CHART_PALETTE[si % CHART_PALETTE.length]}
                />
              )),
            )}
          </>
        ) : (
          <>
            {labels.flatMap((_, i) => {
              const gw = innerW / n;
              const bw = (gw * 0.7) / m;
              return datasets.map((ds, si) => {
                const v = ds.data[i] ?? 0;
                const x0 = pad.l + i * gw + gw * 0.15 + si * bw;
                const y0 = yScale(v);
                const h = pad.t + innerH - y0;
                return (
                  <rect
                    key={`${i}-${si}`}
                    x={x0}
                    y={y0}
                    width={bw * 0.92}
                    height={h}
                    fill={CHART_PALETTE[si % CHART_PALETTE.length]}
                    opacity={0.85}
                  />
                );
              });
            })}
          </>
        )}
        {labels.map((lab, i) => (
          <text key={i} x={xCat(i)} y={H - 12} textAnchor="middle" fontSize={11} fill="#94a3b8">
            {lab.length > 12 ? `${lab.slice(0, 10)}…` : lab}
          </text>
        ))}
      </svg>
      <div style={{ display: 'flex', flexWrap: 'wrap', gap: 12, marginTop: 8, fontSize: 12, color: 'var(--text-muted)' }}>
        {datasets.map((ds, si) => (
          <span key={si} style={{ display: 'inline-flex', alignItems: 'center', gap: 6 }}>
            <span style={{ width: 12, height: 12, background: CHART_PALETTE[si % CHART_PALETTE.length], borderRadius: 2 }} />
            {ds.label || `Series ${si + 1}`}
          </span>
        ))}
      </div>
      <div style={{ marginTop: '0.75rem', fontSize: '0.75rem', color: 'var(--text-muted)' }}>
        <i className="bi bi-bar-chart-line" style={{ marginRight: '0.5rem' }}></i>
        MCP App: Chart ({isLine ? 'line' : 'bar'})
      </div>
    </div>
  );
}

/** Google Maps embed (iframe); URL is built server-side from lat/lng or allowlisted embed URL. */
function MCPAppMapWidget({ payload }: { payload: Extract<MCPAppPayload, { widget: 'map' }> }) {
  const { embedUrl, title } = payload.props;
  return (
    <div style={{
      background: 'var(--dark-bg)',
      borderRadius: '8px',
      overflow: 'hidden',
      border: '1px solid var(--card-border)',
      marginTop: '0.5rem',
      maxWidth: '100%',
      padding: '1rem',
    }}>
      {title ? <div style={{ fontWeight: 600, marginBottom: '0.75rem' }}>{title}</div> : null}
      <iframe
        title={title || 'Google Map'}
        src={embedUrl}
        loading="lazy"
        referrerPolicy="no-referrer-when-downgrade"
        sandbox="allow-scripts allow-same-origin allow-popups allow-forms allow-popups-to-escape-sandbox"
        style={{ width: '100%', height: 360, border: 0, borderRadius: 8, background: '#0f172a' }}
      />
      <div style={{ marginTop: '0.75rem', fontSize: '0.75rem', color: 'var(--text-muted)' }}>
        <i className="bi bi-geo-alt" style={{ marginRight: '0.5rem' }}></i>
        MCP App: Map (Google Maps embed)
      </div>
    </div>
  );
}

function coerceFormValue(raw: string, type: string): unknown {
  switch (type) {
    case 'boolean':
      return raw === 'true' || raw === 'on';
    case 'number': {
      const n = Number(raw);
      return Number.isFinite(n) ? n : NaN;
    }
    default:
      return raw;
  }
}

/** MCP Apps form: submits to another tool on the same server (test harness). */
function MCPAppFormWidget({
  payload,
  tools,
  runSubmit,
}: {
  payload: Extract<MCPAppPayload, { widget: 'form' }>;
  tools: Tool[];
  runSubmit: (submitToolName: string, args: Record<string, unknown>) => Promise<void>;
}) {
  const { title, submitLabel, submitTool, fields, initialValues } = payload.props;
  const [values, setValues] = useState<Record<string, string>>(() => {
    const out: Record<string, string> = {};
    for (const f of fields) {
      const iv = initialValues?.[f.name];
      if (iv !== undefined && iv !== null) {
        out[f.name] = typeof iv === 'boolean' ? (iv ? 'true' : 'false') : String(iv);
      } else if (f.default !== undefined && f.default !== null) {
        out[f.name] = typeof f.default === 'boolean' ? (f.default ? 'true' : 'false') : String(f.default);
      } else {
        out[f.name] = f.type === 'boolean' ? 'false' : '';
      }
    }
    return out;
  });
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);
  const missingTool = !tools.some((t) => t.name === submitTool);

  const onSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setErr(null);
    const args: Record<string, unknown> = {};
    for (const f of fields) {
      const raw = values[f.name] ?? '';
      const v = coerceFormValue(raw, f.type);
      if (f.required && (raw === '' || (f.type === 'number' && typeof v === 'number' && Number.isNaN(v)))) {
        setErr(`Required: ${f.label}`);
        return;
      }
      if (f.type === 'number' && typeof v === 'number' && Number.isNaN(v)) {
        setErr(`Invalid number: ${f.label}`);
        return;
      }
      args[f.name] = v;
    }
    setBusy(true);
    try {
      await runSubmit(submitTool, args);
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  };

  return (
    <div style={{
      background: 'var(--dark-bg)',
      borderRadius: '8px',
      overflow: 'hidden',
      border: '1px solid var(--card-border)',
      marginTop: '0.5rem',
      maxWidth: '100%',
      padding: '1rem',
    }}>
      {missingTool && (
        <div style={{ color: 'var(--danger-color)', fontSize: '0.8125rem', marginBottom: '0.75rem' }}>
          Submit tool <code>{submitTool}</code> is not on this server — add it or fix Output display config.
        </div>
      )}
      <form onSubmit={onSubmit}>
        {title ? <div style={{ fontWeight: 600, marginBottom: '1rem' }}>{title}</div> : null}
        {fields.map((f) => (
          <div key={f.name} style={{ marginBottom: '0.75rem' }}>
            <label className="form-label small mb-1" style={{ display: 'block' }}>
              {f.label}
              {f.required ? <span style={{ color: 'var(--danger-color)' }}> *</span> : null}
            </label>
            {f.type === 'textarea' ? (
              <textarea
                className="form-control form-control-sm"
                required={f.required}
                placeholder={f.placeholder}
                value={values[f.name] ?? ''}
                onChange={(e) => setValues((s) => ({ ...s, [f.name]: e.target.value }))}
                rows={4}
              />
            ) : f.type === 'boolean' ? (
              <input
                type="checkbox"
                checked={values[f.name] === 'true'}
                onChange={(e) => setValues((s) => ({ ...s, [f.name]: e.target.checked ? 'true' : 'false' }))}
              />
            ) : (
              <input
                type={f.type === 'text' ? 'text' : f.type}
                className="form-control form-control-sm"
                required={f.required}
                placeholder={f.placeholder}
                value={values[f.name] ?? ''}
                onChange={(e) => setValues((s) => ({ ...s, [f.name]: e.target.value }))}
              />
            )}
          </div>
        ))}
        {err ? <div style={{ color: 'var(--danger-color)', fontSize: '0.8125rem', marginBottom: '0.5rem' }}>{err}</div> : null}
        <button type="submit" className="btn btn-primary btn-sm" disabled={busy || missingTool}>
          {busy ? '…' : submitLabel || 'Submit'}
        </button>
      </form>
      <div style={{ marginTop: '0.75rem', fontSize: '0.75rem', color: 'var(--text-muted)' }}>
        <i className="bi bi-ui-checks" style={{ marginRight: '0.5rem' }}></i>
        MCP App: Form → <code>{submitTool}</code>
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
  /** When true (e.g. tool workbench), the tool dropdown cannot be changed */
  lockToolSelection?: boolean;
}

export default function TestPlayground({ serverId, tools, initialToolId, onOpenEnvironments, lockToolSelection }: Props) {
  const [selectedTool, setSelectedTool] = useState<string>('');
  const [selectedEnvProfile, setSelectedEnvProfile] = useState<EnvProfileKey>('dev');
  const [input, setInput] = useState('{\n  \n}');
  const [context, setContext] = useState('{\n  "user_id": "user-123",\n  "organization_id": "org-456",\n  "roles": ["user"]\n}');
  const [result, setResult] = useState<TestToolResponse | null>(null);
  const [testing, setTesting] = useState(false);
  const [dryRun, setDryRun] = useState(false);
  const [presets, setPresets] = useState<ToolTestPreset[]>([]);
  const [selectedPresetId, setSelectedPresetId] = useState<string>('');
  const [showDeletePresetConfirm, setShowDeletePresetConfirm] = useState(false);
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

  const runFormSubmit = useCallback(
    async (submitToolName: string, args: Record<string, unknown>) => {
      const tid = tools.find((t) => t.name === submitToolName)?.id;
      if (!tid) {
        toast.error(`Tool "${submitToolName}" not found on this server`);
        return;
      }
      let parsedContext: Record<string, unknown>;
      try {
        parsedContext = JSON.parse(context);
      } catch {
        toast.error('Invalid context JSON');
        return;
      }
      setTesting(true);
      try {
        const response = await testTool(tid, args, parsedContext, hasEnvProfiles ? selectedEnvProfile : undefined);
        setResult(response);
        try {
          const evalRes = await evaluatePolicy(tid, args, parsedContext);
          setPolicyEval(evalRes);
        } catch {
          setPolicyEval(null);
        }
        if (response.success) {
          toast.success('Tool executed successfully');
        } else {
          toast.warning('Tool execution failed');
        }
      } finally {
        setTesting(false);
      }
    },
    [tools, context, hasEnvProfiles, selectedEnvProfile],
  );
  
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
    <div className="test-playground">
      <div className="card" style={{ marginBottom: '1rem', border: '1px solid var(--card-border)' }}>
        <div className="card-body py-3 px-3">
          <div style={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', flexWrap: 'wrap', gap: '0.35rem', marginBottom: '0.5rem' }}>
            <h3 className="card-title" style={{ margin: 0, fontSize: '1.1rem' }}>
              <i className="bi bi-play-circle" style={{ marginRight: '0.35rem', color: 'var(--secondary-color)' }}></i>
              Testing
            </h3>
            <a
              href="https://github.com/vdparikh/make-mcp/blob/main/docs/creating-servers.md#3-test-your-tools"
              target="_blank"
              rel="noopener noreferrer"
              style={{ fontSize: '0.75rem', color: 'var(--text-muted)', textDecoration: 'none', whiteSpace: 'nowrap' }}
            >
              Best practices
            </a>
          </div>
          <p style={{ color: 'var(--text-secondary)', margin: '0 0 0.65rem 0', fontSize: '0.8125rem', lineHeight: 1.35 }}>
            Run against an environment profile; presets remember input + context.
          </p>

          <div className="row g-2 align-items-end">
            <div className={hasEnvProfiles ? 'col-12 col-lg-4' : 'col-12 col-md-6'}>
              <div className="form-group mb-0">
                <label className="form-label">Tool</label>
                <select
                  className="form-control form-control-sm"
                  value={selectedTool}
                  onChange={(e) => handleToolSelect(e.target.value)}
                  disabled={lockToolSelection}
                >
                  <option value="">Select a tool...</option>
                  {tools.map((tool) => (
                    <option key={tool.id} value={tool.id}>
                      {tool.name} ({tool.execution_type})
                    </option>
                  ))}
                </select>
              </div>
            </div>
            <div className={hasEnvProfiles ? 'col-12 col-lg-4' : 'col-12 col-md-6'}>
              <div className="form-group mb-0">
                <label className="form-label">Preset</label>
                <div className="d-flex gap-1 align-items-stretch">
                  <select
                    className="form-control form-control-sm flex-grow-1"
                    style={{ minWidth: 0 }}
                    value={selectedPresetId}
                    onChange={(e) => handleApplyPreset(e.target.value)}
                    disabled={!selectedTool || presetsLoading}
                  >
                    <option value="">{presetsLoading ? '…' : selectedTool ? 'Preset…' : 'Select a tool first'}</option>
                    {presets.map((preset) => (
                      <option key={preset.id} value={preset.id}>{preset.name}</option>
                    ))}
                  </select>
                  <button
                    type="button"
                    className="btn btn-outline-primary btn-sm flex-shrink-0"
                    onClick={handleSavePreset}
                    disabled={!selectedTool}
                    title="Save current input + context as preset"
                  >
                    <i className="bi bi-bookmark-plus"></i>
                  </button>
                  <button
                    type="button"
                    className="btn btn-outline-secondary btn-sm flex-shrink-0"
                    onClick={() => setShowDeletePresetConfirm(true)}
                    disabled={!selectedTool || !selectedPresetId}
                    title="Delete preset"
                  >
                    <i className="bi bi-trash"></i>
                  </button>
                </div>
              </div>
            </div>
            {hasEnvProfiles && (
              <div className="col-12 col-lg-4">
                <div className="form-group mb-0">
                  <div className="d-flex align-items-center justify-content-between gap-1" style={{ minHeight: '1.125rem', marginBottom: '0.25rem' }}>
                    <label className="form-label mb-0">Environment</label>
                    {onOpenEnvironments && (
                      <button
                        type="button"
                        className="btn btn-link p-0 lh-1"
                        title="Edit environment profiles"
                        aria-label="Edit environment profiles"
                        onClick={onOpenEnvironments}
                      >
                        <i className="bi bi-pencil-square" style={{ fontSize: '0.95rem', color: 'var(--primary-color)' }} />
                      </button>
                    )}
                  </div>
                  <select
                    className="form-control form-control-sm"
                    value={selectedEnvProfile}
                    onChange={(e) => setSelectedEnvProfile(e.target.value as EnvProfileKey)}
                  >
                    <option value="dev">Dev</option>
                    <option value="staging">Staging</option>
                    <option value="prod">Prod</option>
                  </select>
                </div>
              </div>
            )}
          </div>

          {selectedToolData && (
            <div
              style={{
                marginTop: '0.65rem',
                padding: '0.45rem 0.65rem',
                background: 'var(--dark-bg)',
                borderRadius: '6px',
                border: '1px solid var(--card-border)',
              }}
            >
              <p style={{ fontSize: '0.78rem', color: 'var(--text-secondary)', margin: 0, lineHeight: 1.4 }}>
                {selectedToolData.description || 'No description'}
                {selectedToolData.context_fields && selectedToolData.context_fields.length > 0 && (
                  <span style={{ marginLeft: '0.35rem', color: 'var(--text-muted)' }}>
                    · Context: {selectedToolData.context_fields.join(', ')}
                  </span>
                )}
              </p>
              {isMocked && (
                <span style={{ fontSize: '0.7rem', color: 'var(--warning-color)', marginTop: '0.2rem', display: 'inline-block' }}>
                  Simulated here; runs live in exported server.
                </span>
              )}
            </div>
          )}
        </div>
      </div>

      <div className="test-playground-grid">
        <div>
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
            className="btn btn-success test-execute-btn" 
            onClick={handleTest}
            disabled={!selectedTool || testing}
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
              <div className={`test-result-banner ${result.success ? 'success' : 'error'}`}>
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
                <div className={`test-policy-card ${policyEval.allowed && !policyEval.requires_approval ? 'allowed' : policyEval.requires_approval ? 'approval' : 'denied'}`}>
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
                <div className="test-context-card">
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
                <div className="test-error-card">
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
                  ) : isMCPAppOutput(result.output) && result.output._mcp_app.widget === 'image' ? (
                    <MCPAppImageWidget payload={result.output._mcp_app} />
                  ) : isMCPAppOutput(result.output) && result.output._mcp_app.widget === 'table' ? (
                    <MCPAppTableWidget payload={result.output._mcp_app} />
                  ) : isMCPAppOutput(result.output) && result.output._mcp_app.widget === 'chart' ? (
                    <MCPAppChartWidget payload={result.output._mcp_app} />
                  ) : isMCPAppOutput(result.output) && result.output._mcp_app.widget === 'map' ? (
                    <MCPAppMapWidget payload={result.output._mcp_app} />
                  ) : isMCPAppOutput(result.output) && result.output._mcp_app.widget === 'form' ? (
                    <MCPAppFormWidget
                      payload={result.output._mcp_app}
                      tools={tools}
                      runSubmit={runFormSubmit}
                    />
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
      <ConfirmModal
        open={showDeletePresetConfirm}
        title="Delete preset?"
        message={`Delete preset "${presets.find((p) => p.id === selectedPresetId)?.name || 'selected preset'}"?`}
        confirmLabel="Delete"
        danger
        onCancel={() => setShowDeletePresetConfirm(false)}
        onConfirm={async () => {
          setShowDeletePresetConfirm(false);
          await handleDeletePreset();
        }}
      />
    </div>
  );
}
