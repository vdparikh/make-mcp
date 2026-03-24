import { useEffect, useMemo, useState } from 'react';
import { toast } from 'react-toastify';
import { getTryConfig, getServer, tryChat, type TryChatMessage, type TryProviderInfo } from '../services/api';
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
  const [catalogTools, setCatalogTools] = useState<string[]>([]);
  const [toolsLoading, setToolsLoading] = useState(false);

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

  useEffect(() => {
    if (!open) {
      setMessages([]);
      setToolEvents([]);
      setEndpoint('');
      setCatalogTools([]);
      setToolsLoading(false);
      setInput('');
      return;
    }
    setEndpoint(target?.endpoint || '');
    if (target?.toolNames?.length) {
      setCatalogTools(target.toolNames);
      setToolsLoading(false);
      return;
    }
    if (target?.id && target.type === 'server') {
      setToolsLoading(true);
      let cancelled = false;
      getServer(target.id)
        .then((s) => {
          if (cancelled) return;
          const names = (s.tools ?? []).map((t) => t.name).filter(Boolean) as string[];
          setCatalogTools(names);
        })
        .catch(() => {
          if (!cancelled) setCatalogTools([]);
        })
        .finally(() => {
          if (!cancelled) setToolsLoading(false);
        });
      return () => {
        cancelled = true;
      };
    }
    setCatalogTools([]);
    setToolsLoading(false);
  }, [open, target]);

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
    <div className="modal-overlay try-chat-overlay" onClick={closeTryChat}>
      <div className="modal-content try-chat-modal" onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <h2 className="modal-title">
            <i className="bi bi-stars" style={{ marginRight: '0.5rem' }} />
            Try Chat
          </h2>
          <button type="button" className="btn btn-secondary" onClick={closeTryChat}>
            Close
          </button>
        </div>
        <div className="modal-body try-chat-body">
          <div className="try-chat-main">
            <div className="try-chat-thread">
              {messages.length === 0 ? (
                <div className="try-chat-empty">
                  Ask a real-world question to preview agent behavior.
                </div>
              ) : (
                messages.map((m, idx) => (
                  <div key={idx} className={`try-chat-message-row ${m.role === 'user' ? 'user' : 'assistant'}`}>
                    <div className="try-chat-message-label">{m.role === 'user' ? 'You' : 'Assistant'}</div>
                    <div className={`try-chat-bubble ${m.role === 'user' ? 'user' : 'assistant'}`}>
                      {m.content}
                    </div>
                  </div>
                ))
              )}
              {sending && <div className="try-chat-thinking">Thinking...</div>}
            </div>
            <div className="try-chat-composer">
              <textarea
                className="form-control try-chat-input"
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
              <button className="btn btn-primary try-chat-send" onClick={() => void sendMessage()} disabled={sending || !input.trim()}>
                Send
              </button>
            </div>
          </div>
          <div className="try-chat-sidebar">
            <h4 className="try-chat-sidebar-title">Target Context</h4>
            <div className="try-chat-context">
              <div><strong>Type:</strong> {target?.type || 'general'}</div>
              <div><strong>Name:</strong> {target?.name || '—'}</div>
              <div><strong>ID:</strong> {target?.id || '—'}</div>
              <div>
                <strong>Endpoint:</strong>{' '}
                <span className="try-chat-endpoint" title={endpoint || undefined}>
                  {endpoint || '—'}
                </span>
              </div>
            </div>
            <hr className="try-chat-divider" />
            <h4 className="try-chat-sidebar-title">Tools ({catalogTools.length})</h4>
            {toolsLoading ? (
              <div className="try-chat-muted-note">Loading tools…</div>
            ) : catalogTools.length === 0 ? (
              <div className="try-chat-muted-note">No tools listed for this server.</div>
            ) : (
              <div className="try-chat-tools-list">
                {catalogTools.map((name) => (
                  <div key={name} className="try-chat-tool-row">
                    {name}
                  </div>
                ))}
              </div>
            )}
            <hr className="try-chat-divider" />
            <h4 className="try-chat-sidebar-title">Session Settings</h4>
            <div className="form-group try-chat-field">
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
            <div className="form-group try-chat-field">
              <label className="form-label">Model</label>
              <input className="form-control" value={model} onChange={(e) => setModel(e.target.value)} />
            </div>
            <div className="try-chat-muted-note">
              Default model: {selectedProvider?.model || '—'}
            </div>
            <hr className="try-chat-divider" />
            <h4 className="try-chat-sidebar-title">Tool Calls</h4>
            {toolEvents.length === 0 ? (
              <div className="try-chat-muted-note">No tool calls yet.</div>
            ) : (
              <div className="try-chat-tool-events">
                {toolEvents.slice(-20).map((e, idx) => (
                  <div key={`${e.name}-${idx}`} className={`try-chat-tool-event ${e.success ? 'success' : 'error'}`}>
                    <span className="marker">{e.success ? '✓' : '✗'}</span> {e.name} ({e.duration_ms} ms){e.error ? ` - ${e.error}` : ''}
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

