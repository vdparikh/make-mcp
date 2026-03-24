import { useApp } from "@modelcontextprotocol/ext-apps/react";
import { StrictMode, useCallback, useMemo, useState, type FormEvent } from "react";
import { createRoot } from "react-dom/client";

const IMPLEMENTATION = { name: "Make MCP App Viewer", version: "1.0.0" };

export type FormFieldDef = {
  name: string;
  label: string;
  type: string;
  default?: unknown;
  required?: boolean;
  placeholder?: string;
};

type ChartDataset = { label: string; data: number[] };

type McpAppPayload =
  | { widget: "table"; props: { columns: { key: string; label: string }[]; rows: Record<string, unknown>[] } }
  | { widget: "card"; props: { content: string; title?: string } }
  | { widget: "image"; props: { imageUrl: string; title?: string; alt: string } }
  | {
      widget: "form";
      props: {
        title?: string;
        submitLabel?: string;
        submitTool: string;
        fields: FormFieldDef[];
        initialValues?: Record<string, unknown>;
      };
    }
  | {
      widget: "chart";
      props: {
        chartType: string;
        title?: string;
        labels: string[];
        datasets: ChartDataset[];
      };
    }
  | {
      widget: "map";
      props: {
        embedUrl: string;
        title?: string;
      };
    };

/** Minimal typing for MCP App bridge (callServerTool). */
type McpAppBridge = {
  callServerTool: (req: { name: string; arguments?: Record<string, unknown> }) => Promise<unknown>;
};

function extractTextFromToolResult(notification: unknown): string | null {
  if (!notification || typeof notification !== "object") return null;
  const n = notification as {
    params?: {
      content?: Array<{ type?: string; text?: string }>;
    };
  };
  const parts = n.params?.content;
  if (!Array.isArray(parts)) return null;
  const textPart = parts.find((p) => p?.type === "text" && typeof p.text === "string");
  return textPart?.text ?? null;
}

const CHART_PALETTE = ["#2563eb", "#16a34a", "#d97706", "#dc2626", "#7c3aed", "#0891b2", "#db2777", "#4b5563"];

function ChartWidget({ props }: { props: Extract<McpAppPayload, { widget: "chart" }>["props"] }) {
  const { chartType, title, labels, datasets } = props;
  const isLine = chartType === "line";
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
    <div style={{ fontFamily: "system-ui, sans-serif", maxWidth: "100%" }}>
      {title ? <h2 style={{ fontSize: 18, marginBottom: 8 }}>{title}</h2> : null}
      <svg
        viewBox={`0 0 ${W} ${H}`}
        style={{ width: "100%", height: "auto", maxHeight: 360, background: "#fafafa", borderRadius: 8 }}
        role="img"
        aria-label={title || "Chart"}
      >
        <line x1={pad.l} y1={pad.t + innerH} x2={pad.l + innerW} y2={pad.t + innerH} stroke="#ccc" strokeWidth={1} />
        <line x1={pad.l} y1={pad.t} x2={pad.l} y2={pad.t + innerH} stroke="#ccc" strokeWidth={1} />
        {isLine ? (
          <>
            {datasets.map((ds, si) => {
              const pts = ds.data
                .map((v, i) => `${xCat(i)},${yScale(v)}`)
                .join(" ");
              const color = CHART_PALETTE[si % CHART_PALETTE.length];
              return (
                <polyline
                  key={si}
                  fill="none"
                  stroke={color}
                  strokeWidth={2}
                  points={pts}
                />
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
          <text
            key={i}
            x={xCat(i)}
            y={H - 12}
            textAnchor="middle"
            fontSize={11}
            fill="#444"
          >
            {lab.length > 12 ? lab.slice(0, 10) + "…" : lab}
          </text>
        ))}
      </svg>
      <div style={{ display: "flex", flexWrap: "wrap", gap: 12, marginTop: 8, fontSize: 12 }}>
        {datasets.map((ds, si) => (
          <span key={si} style={{ display: "inline-flex", alignItems: "center", gap: 6 }}>
            <span style={{ width: 12, height: 12, background: CHART_PALETTE[si % CHART_PALETTE.length], borderRadius: 2 }} />
            {ds.label || `Series ${si + 1}`}
          </span>
        ))}
      </div>
    </div>
  );
}

function MapWidget({ props }: { props: Extract<McpAppPayload, { widget: "map" }>["props"] }) {
  const { embedUrl, title } = props;
  return (
    <div style={{ fontFamily: "system-ui, sans-serif", maxWidth: "100%" }}>
      {title ? <h2 style={{ fontSize: 18, marginBottom: 8 }}>{title}</h2> : null}
      <iframe
        title={title || "Google Map"}
        src={embedUrl}
        loading="lazy"
        referrerPolicy="no-referrer-when-downgrade"
        sandbox="allow-scripts allow-same-origin allow-popups allow-forms allow-popups-to-escape-sandbox"
        style={{ width: "100%", height: 360, border: 0, borderRadius: 8, background: "#e5e7eb" }}
      />
    </div>
  );
}

function coerceFormValue(raw: string, type: string): unknown {
  switch (type) {
    case "boolean":
      return raw === "true" || raw === "on";
    case "number": {
      const n = Number(raw);
      return Number.isFinite(n) ? n : NaN;
    }
    default:
      return raw;
  }
}

function FormWidget({
  props,
  app,
}: {
  props: Extract<McpAppPayload, { widget: "form" }>["props"];
  app: McpAppBridge;
}) {
  const { title, submitLabel, submitTool, fields, initialValues } = props;
  const [values, setValues] = useState<Record<string, string>>(() => {
    const out: Record<string, string> = {};
    for (const f of fields) {
      const iv = initialValues?.[f.name];
      if (iv !== undefined && iv !== null) {
        out[f.name] = typeof iv === "boolean" ? (iv ? "true" : "false") : String(iv);
      } else if (f.default !== undefined && f.default !== null) {
        out[f.name] = typeof f.default === "boolean" ? (f.default ? "true" : "false") : String(f.default);
      } else {
        out[f.name] = f.type === "boolean" ? "false" : "";
      }
    }
    return out;
  });
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  const onSubmit = useCallback(
    async (e: FormEvent) => {
      e.preventDefault();
      setErr(null);
      const args: Record<string, unknown> = {};
      for (const f of fields) {
        const raw = values[f.name] ?? "";
        const v = coerceFormValue(raw, f.type);
        if (f.required && (raw === "" || (f.type === "number" && typeof v === "number" && Number.isNaN(v)))) {
          setErr(`Required: ${f.label}`);
          return;
        }
        if (f.type === "number" && typeof v === "number" && Number.isNaN(v)) {
          setErr(`Invalid number: ${f.label}`);
          return;
        }
        args[f.name] = v;
      }
      setBusy(true);
      try {
        await app.callServerTool({ name: submitTool, arguments: args });
      } catch (e) {
        setErr(e instanceof Error ? e.message : String(e));
      } finally {
        setBusy(false);
      }
    },
    [app, fields, submitTool, values]
  );

  return (
    <form
      onSubmit={onSubmit}
      style={{
        fontFamily: "system-ui, sans-serif",
        padding: 12,
        borderRadius: 8,
        border: "1px solid #ccc",
        maxWidth: 520,
      }}
    >
      {title ? <h2 style={{ fontSize: 18, marginBottom: 16 }}>{title}</h2> : null}
      {fields.map((f) => (
        <div key={f.name} style={{ marginBottom: 12 }}>
          <label style={{ display: "block", fontSize: 13, fontWeight: 600, marginBottom: 4 }}>
            {f.label}
            {f.required ? <span style={{ color: "#b91c1c" }}> *</span> : null}
          </label>
          {f.type === "textarea" ? (
            <textarea
              name={f.name}
              required={f.required}
              placeholder={f.placeholder}
              value={values[f.name] ?? ""}
              onChange={(e) => setValues((s) => ({ ...s, [f.name]: e.target.value }))}
              rows={4}
              style={{ width: "100%", boxSizing: "border-box", padding: 8, borderRadius: 6 }}
            />
          ) : f.type === "boolean" ? (
            <input
              type="checkbox"
              name={f.name}
              checked={values[f.name] === "true"}
              onChange={(e) => setValues((s) => ({ ...s, [f.name]: e.target.checked ? "true" : "false" }))}
            />
          ) : (
            <input
              type={f.type === "text" ? "text" : f.type}
              name={f.name}
              required={f.required}
              placeholder={f.placeholder}
              value={values[f.name] ?? ""}
              onChange={(e) => setValues((s) => ({ ...s, [f.name]: e.target.value }))}
              style={{ width: "100%", boxSizing: "border-box", padding: 8, borderRadius: 6 }}
            />
          )}
        </div>
      ))}
      {err ? (
        <div style={{ color: "#b91c1c", fontSize: 13, marginBottom: 8 }} role="alert">
          {err}
        </div>
      ) : null}
      <button type="submit" disabled={busy} style={{ padding: "8px 16px", borderRadius: 6, fontWeight: 600 }}>
        {busy ? "…" : submitLabel || "Submit"}
      </button>
    </form>
  );
}

function McpAppView({ text, app }: { text: string; app: McpAppBridge | null }) {
  const parsed = useMemo(() => {
    try {
      const j = JSON.parse(text) as { _mcp_app?: McpAppPayload; text?: string };
      return j;
    } catch {
      return null;
    }
  }, [text]);

  if (!parsed?._mcp_app) {
    return (
      <pre style={{ margin: 0, whiteSpace: "pre-wrap", fontFamily: "system-ui, sans-serif", fontSize: 14 }}>
        {text}
      </pre>
    );
  }

  const w = parsed._mcp_app;

  if (w.widget === "form") {
    if (!app) {
      return <div style={{ color: "#b91c1c" }}>Form widget requires MCP App bridge.</div>;
    }
    return <FormWidget props={w.props} app={app} />;
  }

  if (w.widget === "chart") {
    return <ChartWidget props={w.props} />;
  }

  if (w.widget === "map") {
    return <MapWidget props={w.props} />;
  }

  if (w.widget === "image") {
    const { imageUrl, title, alt } = w.props;
    return (
      <div style={{ fontFamily: "system-ui, sans-serif" }}>
        {title ? <h2 style={{ fontSize: 18, marginBottom: 12 }}>{title}</h2> : null}
        <img src={imageUrl} alt={alt || "result"} style={{ maxWidth: "100%", borderRadius: 8 }} />
      </div>
    );
  }

  if (w.widget === "card") {
    const { content, title } = w.props;
    return (
      <div style={{ fontFamily: "system-ui, sans-serif", padding: 16 }}>
        {title ? (
          <div style={{ fontSize: 12, fontWeight: 600, textTransform: "uppercase", opacity: 0.7, marginBottom: 12 }}>
            {title}
          </div>
        ) : null}
        <div style={{ fontSize: 22, lineHeight: 1.4, fontWeight: 500 }}>{content}</div>
      </div>
    );
  }

  if (w.widget === "table") {
    const { columns, rows } = w.props;
    return (
      <div style={{ overflow: "auto", fontFamily: "system-ui, sans-serif", fontSize: 13 }}>
        <table style={{ borderCollapse: "collapse", width: "100%" }}>
          <thead>
            <tr>
              {columns.map((c) => (
                <th key={c.key} style={{ borderBottom: "1px solid #ccc", textAlign: "left", padding: 8 }}>
                  {c.label}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {rows.map((row, ri) => (
              <tr key={ri}>
                {columns.map((c) => (
                  <td key={c.key} style={{ borderBottom: "1px solid #eee", padding: 8, verticalAlign: "top" }}>
                    {formatCell(row[c.key])}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    );
  }

  return <pre style={{ whiteSpace: "pre-wrap" }}>{text}</pre>;
}

function formatCell(v: unknown): string {
  if (v == null) return "—";
  if (typeof v === "object") {
    try {
      return JSON.stringify(v);
    } catch {
      return String(v);
    }
  }
  return String(v);
}

function App() {
  const [preview, setPreview] = useState<string>(
    "Connect this MCP server in a host that supports MCP Apps (e.g. MCP Jam, Claude). Run a tool with Card / Table / Image / Form / Chart / Map output to see the UI here."
  );

  const { app, error } = useApp({
    appInfo: IMPLEMENTATION,
    capabilities: {},
    onAppCreated: (appInstance) => {
      appInstance.ontoolresult = (params) => {
        const t = extractTextFromToolResult({ params });
        if (t) setPreview(t);
      };
    },
  });

  if (error) {
    return (
      <div style={{ padding: 16, color: "#b91c1c", fontFamily: "system-ui" }}>
        <strong>MCP App error:</strong> {error.message}
      </div>
    );
  }

  if (!app) {
    return (
      <div style={{ padding: 16, fontFamily: "system-ui" }}>
        <p>Loading MCP App bridge…</p>
      </div>
    );
  }

  return (
    <main style={{ padding: 16, maxWidth: 900, margin: "0 auto" }}>
      <McpAppView text={preview} app={app as McpAppBridge} />
    </main>
  );
}

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <App />
  </StrictMode>
);
