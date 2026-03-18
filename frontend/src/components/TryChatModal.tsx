import { useEffect, useMemo, useState } from 'react';
import { toast } from 'react-toastify';
import { getTryConfig, tryChat, type TryChatMessage, type TryProviderInfo } from '../services/api';
import { useTryChat } from '../contexts/TryChatContext';

export default function TryChatModal() {
  const { open, closeTryChat, target } = useTryChat();
  const [loadingConfig, setLoadingConfig] = useState(false);
  const [sending, setSending] = useState(false);
  const [providers, setProviders] = useState<TryProviderInfo[]>([]);
  const [provider, setProvider] = useState('');
  const [model, setModel] = useState('');
  const [input, setInput] = useState('');
  const [messages, setMessages] = useState<TryChatMessage[]>([]);
  const [toolEvents, setToolEvents] = useState<Array<{ name: string; success: boolean; duration_ms: number; error?: string }>>([]);
  const [endpoint, setEndpoint] = useState('');

  useEffect(() => {
    if (!open) return;
    setLoadingConfig(true);
    getTryConfig()
      .then((cfg) => {
        setProviders(cfg.providers || []);
        const nextProvider = cfg.default_provider || cfg.providers?.[0]?.name || '';
        setProvider(nextProvider);
        const defaultModel = cfg.providers?.find((p) => p.name === nextProvider)?.model || '';
        setModel(defaultModel);
      })
      .catch((err) => {
        toast.error(err.response?.data?.error || err.message || 'Failed to load Try Chat config');
      })
      .finally(() => setLoadingConfig(false));
  }, [open]);

  const selectedProvider = useMemo(() => providers.find((p) => p.name === provider) || null, [providers, provider]);

  const sendMessage = async () => {
    const trimmed = input.trim();
    if (!trimmed || sending) return;
    const nextMessages: TryChatMessage[] = [...messages, { role: 'user', content: trimmed }];
    setMessages(nextMessages);
    setInput('');
    setSending(true);
    try {
      const res = await tryChat({
        provider,
        model: model.trim() || undefined,
        messages: nextMessages,
        target: target || undefined,
      });
      setMessages((prev) => [...prev, { role: 'assistant', content: res.message || '(No response)' }]);
      if (res.model) setModel(res.model);
      if (res.provider) setProvider(res.provider);
      if (res.endpoint) setEndpoint(res.endpoint);
      const calls = res.tool_calls ?? [];
      if (calls.length > 0) {
        setToolEvents((prev) => [
          ...prev,
          ...calls.map((tc) => ({
            name: tc.name,
            success: tc.success,
            duration_ms: tc.duration_ms,
            error: tc.error,
          })),
        ]);
      }
    } catch (err: unknown) {
      const msg =
        typeof err === 'object' && err !== null && 'response' in err
          ? (err as { response?: { data?: { error?: string } } }).response?.data?.error
          : undefined;
      toast.error(msg || 'Try Chat request failed');
      setMessages((prev) => [...prev, { role: 'assistant', content: `Error: ${msg || 'Try Chat request failed'}` }]);
    } finally {
      setSending(false);
    }
  };

  if (!open) return null;

  return (
    <div className="modal-overlay" onClick={closeTryChat} style={{ zIndex: 2100 }}>
      <div
        className="modal-content"
        onClick={(e) => e.stopPropagation()}
        style={{ width: '92vw', maxWidth: '1400px', height: '90vh', maxHeight: '90vh', display: 'flex', flexDirection: 'column' }}
      >
        <div className="modal-header">
          <h2 className="modal-title">
            <i className="bi bi-stars" style={{ marginRight: '0.5rem' }} />
            Try Chat
          </h2>
          <button type="button" className="btn btn-secondary" onClick={closeTryChat}>
            Close
          </button>
        </div>
        <div className="modal-body" style={{ flex: 1, display: 'grid', gridTemplateColumns: 'minmax(0,1fr) 320px', gap: '1rem', minHeight: 0 }}>
          <div style={{ display: 'flex', flexDirection: 'column', minHeight: 0 }}>
            <div style={{ flex: 1, overflow: 'auto', border: '1px solid var(--card-border)', borderRadius: '8px', padding: '0.75rem', background: 'var(--dark-bg)' }}>
              {messages.length === 0 ? (
                <div style={{ color: 'var(--text-muted)' }}>
                  Ask a real-world question to preview agent behavior.
                </div>
              ) : (
                messages.map((m, idx) => (
                  <div key={idx} style={{ marginBottom: '0.75rem' }}>
                    <div style={{ fontSize: '0.75rem', color: 'var(--text-muted)', marginBottom: '0.25rem' }}>
                      {m.role === 'user' ? 'You' : 'Assistant'}
                    </div>
                    <div style={{ whiteSpace: 'pre-wrap', lineHeight: 1.5 }}>{m.content}</div>
                  </div>
                ))
              )}
              {sending && <div style={{ color: 'var(--text-muted)' }}>Thinking...</div>}
            </div>
            <div style={{ display: 'flex', gap: '0.5rem', marginTop: '0.75rem' }}>
              <textarea
                className="form-control"
                rows={3}
                placeholder="Ask the assistant..."
                value={input}
                onChange={(e) => setInput(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter' && !e.shiftKey) {
                    e.preventDefault();
                    void sendMessage();
                  }
                }}
              />
              <button className="btn btn-primary" onClick={() => void sendMessage()} disabled={sending || !input.trim()}>
                Send
              </button>
            </div>
          </div>
          <div style={{ border: '1px solid var(--card-border)', borderRadius: '8px', padding: '0.75rem', background: 'var(--hover-bg)' }}>
            <h4 style={{ marginTop: 0, marginBottom: '0.75rem', fontSize: '0.95rem' }}>Session Settings</h4>
            <div className="form-group" style={{ marginBottom: '0.5rem' }}>
              <label className="form-label">Provider</label>
              <select
                className="form-control"
                value={provider}
                onChange={(e) => {
                  const next = e.target.value;
                  setProvider(next);
                  const nextModel = providers.find((p) => p.name === next)?.model || '';
                  setModel(nextModel);
                }}
                disabled={loadingConfig}
              >
                {providers.map((p) => (
                  <option value={p.name} key={p.name}>
                    {p.name}
                  </option>
                ))}
              </select>
            </div>
            <div className="form-group" style={{ marginBottom: '0.5rem' }}>
              <label className="form-label">Model</label>
              <input className="form-control" value={model} onChange={(e) => setModel(e.target.value)} />
            </div>
            <div style={{ fontSize: '0.8rem', color: 'var(--text-muted)' }}>
              Default model: {selectedProvider?.model || '—'}
            </div>
            <hr style={{ borderColor: 'var(--card-border)', margin: '0.75rem 0' }} />
            <h4 style={{ marginTop: 0, marginBottom: '0.5rem', fontSize: '0.95rem' }}>Target Context</h4>
            <div style={{ fontSize: '0.85rem', color: 'var(--text-secondary)', lineHeight: 1.5 }}>
              <div><strong>Type:</strong> {target?.type || 'general'}</div>
              <div><strong>Name:</strong> {target?.name || '—'}</div>
              <div><strong>ID:</strong> {target?.id || '—'}</div>
              <div><strong>Endpoint:</strong> {endpoint || '—'}</div>
            </div>
            <hr style={{ borderColor: 'var(--card-border)', margin: '0.75rem 0' }} />
            <h4 style={{ marginTop: 0, marginBottom: '0.5rem', fontSize: '0.95rem' }}>Tool Calls</h4>
            {toolEvents.length === 0 ? (
              <div style={{ fontSize: '0.8rem', color: 'var(--text-muted)' }}>No tool calls yet.</div>
            ) : (
              <div style={{ maxHeight: '250px', overflow: 'auto', fontSize: '0.8rem' }}>
                {toolEvents.slice(-20).map((e, idx) => (
                  <div key={`${e.name}-${idx}`} style={{ marginBottom: '0.35rem', color: e.success ? 'var(--success-color)' : 'var(--danger)' }}>
                    {e.success ? '✓' : '✗'} {e.name} ({e.duration_ms} ms){e.error ? ` - ${e.error}` : ''}
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

