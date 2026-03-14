import { useState, useEffect } from 'react';
import { toast } from 'react-toastify';
import type { Tool, ToolExecution, HealingSuggestion } from '../types';
import { getToolExecutions, getHealingSuggestions } from '../services/api';

interface Props {
  tools: Tool[];
}

export default function HealingDashboard({ tools }: Props) {
  const [selectedTool, setSelectedTool] = useState<string>('');
  const [executions, setExecutions] = useState<ToolExecution[]>([]);
  const [suggestions, setSuggestions] = useState<HealingSuggestion[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (selectedTool) {
      loadData();
    }
  }, [selectedTool]);

  const loadData = async () => {
    try {
      setLoading(true);
      const [execs, suggs] = await Promise.all([
        getToolExecutions(selectedTool),
        getHealingSuggestions(selectedTool),
      ]);
      setExecutions(execs);
      setSuggestions(suggs);
    } catch (error) {
      toast.error('Failed to load healing data');
    } finally {
      setLoading(false);
    }
  };

  const formatTime = (dateString: string) => {
    return new Date(dateString).toLocaleString('en-US', {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  };

  const failureRate = executions.length > 0 
    ? ((executions.filter(e => !e.success).length / executions.length) * 100).toFixed(1)
    : '0';

  const avgDuration = executions.length > 0
    ? (executions.reduce((acc, e) => acc + e.duration_ms, 0) / executions.length).toFixed(0)
    : '0';

  return (
    <div>
      <div className="card" style={{ marginBottom: '1.5rem' }}>
        <h3 className="card-title" style={{ marginBottom: '0.75rem' }}>
          <i className="bi bi-bandaid" style={{ marginRight: '0.75rem', color: 'var(--warning-color)' }}></i>
          Self-Healing Dashboard
        </h3>
        <p style={{ color: 'var(--text-secondary)', marginBottom: '1rem' }}>
          Monitor tool executions and get automatic suggestions for fixing failures.
          The healing engine detects patterns like expired tokens, schema mismatches, 
          rate limits, and suggests fixes.
        </p>
        
        <div style={{ 
          background: 'linear-gradient(135deg, rgba(251, 191, 36, 0.15), rgba(251, 191, 36, 0.05))',
          border: '1px solid rgba(251, 191, 36, 0.3)',
          borderRadius: '8px',
          padding: '1rem',
        }}>
          <h4 style={{ fontSize: '0.875rem', marginBottom: '0.5rem', color: 'var(--text-primary)' }}>
            <i className="bi bi-magic" style={{ marginRight: '0.5rem', color: 'var(--primary-color)' }}></i>
            Auto-Detection Examples
          </h4>
          <ul style={{ color: '#e2e8f0', fontSize: '0.8125rem', margin: 0, paddingLeft: '1.25rem' }}>
            <li><strong style={{ color: '#fca5a5' }}>401 Unauthorized</strong> → Suggests: <span style={{ color: '#86efac' }}>Refresh OAuth token</span></li>
            <li><strong style={{ color: '#fcd34d' }}>429 Rate Limited</strong> → Suggests: <span style={{ color: '#86efac' }}>Retry with exponential backoff</span></li>
            <li><strong style={{ color: '#c4b5fd' }}>Schema mismatch</strong> → Suggests: <span style={{ color: '#86efac' }}>Update tool schema</span></li>
            <li><strong style={{ color: '#a5f3fc' }}>Timeout</strong> → Suggests: <span style={{ color: '#86efac' }}>Extend timeout, optimize query</span></li>
          </ul>
        </div>
      </div>

      <div className="form-group" style={{ marginBottom: '1.5rem' }}>
        <label className="form-label">Select Tool</label>
        <select
          className="form-control"
          value={selectedTool}
          onChange={(e) => setSelectedTool(e.target.value)}
        >
          <option value="">Select a tool...</option>
          {tools.map((tool) => (
            <option key={tool.id} value={tool.id}>
              {tool.name}
            </option>
          ))}
        </select>
      </div>

      {selectedTool && (
        <>
          <div className="stats-grid" style={{ marginBottom: '1.5rem' }}>
            <div className="stat-card">
              <div className="stat-value">{executions.length}</div>
              <div className="stat-label">Total Executions</div>
            </div>
            <div className="stat-card">
              <div className="stat-value" style={{ 
                background: parseFloat(failureRate) > 10 
                  ? 'linear-gradient(135deg, var(--danger-color), #dc2626)' 
                  : undefined 
              }}>
                {failureRate}%
              </div>
              <div className="stat-label">Failure Rate</div>
            </div>
            <div className="stat-card">
              <div className="stat-value">{avgDuration}ms</div>
              <div className="stat-label">Avg Duration</div>
            </div>
            <div className="stat-card">
              <div className="stat-value">{suggestions.length}</div>
              <div className="stat-label">Healing Suggestions</div>
            </div>
          </div>

          {suggestions.length > 0 && (
            <div style={{ marginBottom: '1.5rem' }}>
              <h3 style={{ marginBottom: '1rem' }}>
                <i className="bi bi-lightbulb" style={{ marginRight: '0.5rem', color: 'var(--warning-color)' }}></i>
                Healing Suggestions
              </h3>
              {suggestions.map((suggestion) => (
                <div key={suggestion.id} className="healing-card">
                  <div className="healing-header">
                    <i className="bi bi-bandaid healing-icon"></i>
                    <div>
                      <div style={{ fontWeight: 600 }}>{suggestion.suggestion.message}</div>
                      <div style={{ fontSize: '0.8125rem', color: 'var(--text-secondary)' }}>
                        Type: {suggestion.suggestion_type}
                      </div>
                    </div>
                    <div className="healing-confidence">
                      Confidence: {(suggestion.suggestion.confidence * 100).toFixed(0)}%
                    </div>
                  </div>
                  <p style={{ fontSize: '0.875rem', color: 'var(--text-secondary)', margin: '0.75rem 0' }}>
                    {suggestion.suggestion.description}
                  </p>
                  <div style={{ display: 'flex', gap: '0.5rem', alignItems: 'center' }}>
                    {suggestion.suggestion.auto_fix && (
                      <span className="badge badge-success">
                        <i className="bi bi-check" style={{ marginRight: '0.25rem' }}></i>
                        Can Auto-Fix
                      </span>
                    )}
                    <span className="badge badge-primary">
                      Action: {suggestion.suggestion.fix_action}
                    </span>
                    {suggestion.applied && (
                      <span className="badge badge-success">Applied</span>
                    )}
                  </div>
                </div>
              ))}
            </div>
          )}

          <div>
            <h3 style={{ marginBottom: '1rem' }}>Recent Executions</h3>
            {loading ? (
              <div className="loading">
                <div className="spinner"></div>
              </div>
            ) : executions.length === 0 ? (
              <div className="empty-state">
                <i className="bi bi-clock-history"></i>
                <h3>No executions yet</h3>
                <p>Test the tool to see execution history</p>
              </div>
            ) : (
              <div className="execution-log">
                {executions.slice(0, 20).map((exec) => (
                  <div key={exec.id} className="execution-item">
                    <div className={`execution-status ${exec.success ? 'success' : 'error'}`}></div>
                    <div className="execution-details">
                      <div style={{ fontWeight: 500, marginBottom: '0.25rem' }}>
                        {exec.success ? 'Success' : 'Failed'}
                        {exec.status_code > 0 && (
                          <span style={{ fontWeight: 'normal', color: 'var(--text-muted)', marginLeft: '0.5rem' }}>
                            HTTP {exec.status_code}
                          </span>
                        )}
                      </div>
                      {exec.error && (
                        <div style={{ 
                          fontSize: '0.8125rem', 
                          color: 'var(--danger-color)',
                          background: 'rgba(239, 68, 68, 0.1)',
                          padding: '0.5rem',
                          borderRadius: '4px',
                          marginTop: '0.5rem'
                        }}>
                          {exec.error}
                        </div>
                      )}
                      <div className="execution-time">{formatTime(exec.created_at)}</div>
                    </div>
                    <div className="execution-duration">
                      {exec.duration_ms}ms
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        </>
      )}
    </div>
  );
}
