import { useState, useEffect } from 'react';
import { toast } from 'react-toastify';
import Editor from '@monaco-editor/react';
import type { Tool, Policy, PolicyRule, PolicyRuleType } from '../types';
import { createPolicy, getToolPolicies, deletePolicy } from '../services/api';

interface Props {
  tools: Tool[];
  initialToolId?: string;
  onPolicyUpdated: () => void;
}

const ruleTypes: { value: PolicyRuleType; label: string; icon: string; description: string }[] = [
  { value: 'approval_required', label: 'Require Approval', icon: 'bi-check-circle', description: 'Require human approval before execution' },
  { value: 'max_value', label: 'Max Value', icon: 'bi-arrow-up-circle', description: 'Limit maximum value for a field' },
  { value: 'allowed_roles', label: 'Allowed Roles', icon: 'bi-people', description: 'Restrict to specific user roles' },
  { value: 'time_window', label: 'Time Window', icon: 'bi-clock', description: 'Allow only during specific hours' },
  { value: 'rate_limit', label: 'Rate Limit', icon: 'bi-speedometer2', description: 'Limit number of calls' },
  { value: 'custom', label: 'Custom', icon: 'bi-code-slash', description: 'Advanced JSON config' },
];

const ruleTemplates: Record<PolicyRuleType, string> = {
  approval_required: '{\n  "approval_type": "human",\n  "message": "This action requires manager approval"\n}',
  max_value: '{\n  "field": "amount",\n  "max_value": 5000\n}',
  allowed_roles: '{\n  "roles": ["admin", "finance_agent"]\n}',
  time_window: '{\n  "start_hour": 9,\n  "end_hour": 17,\n  "timezone": "America/New_York",\n  "weekdays": [1, 2, 3, 4, 5]\n}',
  rate_limit: '{\n  "max_calls": 100,\n  "window_secs": 3600,\n  "scope": "user"\n}',
  custom: '{}',
};

export default function PolicyEditor({ tools, initialToolId, onPolicyUpdated }: Props) {
  const [selectedTool, setSelectedTool] = useState<string>('');
  const [policies, setPolicies] = useState<Policy[]>([]);
  const [showForm, setShowForm] = useState(false);
  const [loading, setLoading] = useState(false);
  
  const [policyName, setPolicyName] = useState('');
  const [policyDescription, setPolicyDescription] = useState('');
  const [rules, setRules] = useState<{ type: PolicyRuleType; config: string; failAction: string }[]>([]);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (initialToolId && tools.some((t) => t.id === initialToolId)) {
      setSelectedTool(initialToolId);
    }
  }, [initialToolId, tools]);

  useEffect(() => {
    if (selectedTool) {
      loadPolicies();
    }
  }, [selectedTool]);

  const loadPolicies = async () => {
    try {
      setLoading(true);
      const data = await getToolPolicies(selectedTool);
      setPolicies(data);
    } catch (error) {
      toast.error('Failed to load policies');
    } finally {
      setLoading(false);
    }
  };

  const resetForm = () => {
    setPolicyName('');
    setPolicyDescription('');
    setRules([]);
  };

  const addRule = (type: PolicyRuleType) => {
    setRules([...rules, { type, config: ruleTemplates[type], failAction: 'deny' }]);
  };

  const removeRule = (index: number) => {
    setRules(rules.filter((_, i) => i !== index));
  };

  const updateRule = (index: number, field: string, value: string) => {
    const updated = [...rules];
    updated[index] = { ...updated[index], [field]: value };
    setRules(updated);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    try {
      setSaving(true);
      
      const parsedRules: PolicyRule[] = rules.map((rule, i) => {
        let config;
        try {
          config = JSON.parse(rule.config);
        } catch {
          throw new Error(`Invalid JSON in rule ${i + 1}`);
        }
        return {
          id: '',
          policy_id: '',
          type: rule.type,
          config,
          priority: i,
          fail_action: rule.failAction as PolicyRule['fail_action'],
        };
      });

      await createPolicy({
        tool_id: selectedTool,
        name: policyName,
        description: policyDescription,
        rules: parsedRules,
        enabled: true,
      });

      toast.success('Policy created');
      setShowForm(false);
      resetForm();
      loadPolicies();
      onPolicyUpdated();
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Failed to create policy');
    } finally {
      setSaving(false);
    }
  };

  const handleDeletePolicy = async (policyId: string) => {
    if (!confirm('Delete this policy?')) return;
    try {
      await deletePolicy(policyId);
      toast.success('Policy deleted');
      loadPolicies();
    } catch (error) {
      toast.error('Failed to delete policy');
    }
  };

  return (
    <div>
      <div className="card" style={{ marginBottom: '1.5rem' }}>
        <h3 className="card-title" style={{ marginBottom: '0.75rem' }}>
          <i className="bi bi-shield-check" style={{ marginRight: '0.75rem', color: 'var(--success-color)' }}></i>
          AI Governance Layer
        </h3>
        <p style={{ color: 'var(--text-secondary)', marginBottom: '1rem' }}>
          Define policies that control when and how AI agents can call tools. 
          Prevent unauthorized actions, enforce limits, and require approvals for sensitive operations.
        </p>
        
        <div style={{ 
          background: 'linear-gradient(135deg, rgba(52, 211, 153, 0.15), rgba(52, 211, 153, 0.05))',
          border: '1px solid rgba(52, 211, 153, 0.3)',
          borderRadius: '8px',
          padding: '1rem',
        }}>
          <h4 style={{ fontSize: '0.875rem', marginBottom: '0.5rem', color: 'var(--text-primary)' }}>
            <i className="bi bi-shield-exclamation" style={{ marginRight: '0.5rem', color: 'var(--warning-color)' }}></i>
            Example: Payment Protection
          </h4>
          <code style={{ fontSize: '0.8125rem', color: '#a5f3fc', display: 'block', marginBottom: '0.5rem', background: 'rgb(0,0,0)', padding: '0.5rem', borderRadius: '4px' }}>
            AI requests: send_payment($50,000)
          </code>
          <code style={{ fontSize: '0.8125rem', color: '#fca5a5', display: 'block', background: 'rgb(0,0,0)', padding: '0.5rem', borderRadius: '4px' }}>
            Policy Engine: DENIED - Exceeds max_amount of $5,000
          </code>
        </div>
      </div>

      <div className="form-group" style={{ marginBottom: '1.5rem' }}>
        <label className="form-label">Select Tool to Configure Policies</label>
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

      {selectedTool && !showForm && (
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '1rem' }}>
          <h3 style={{ margin: 0 }}>
            Policies for {tools.find(t => t.id === selectedTool)?.name}
          </h3>
          <button className="btn btn-primary" onClick={() => setShowForm(true)}>
            <i className="bi bi-plus-lg"></i>
            Add Policy
          </button>
        </div>
      )}

      {showForm && (
        <div className="card">
          <div className="card-header">
            <h3 className="card-title">Create New Policy</h3>
            <button 
              className="btn btn-icon btn-secondary"
              onClick={() => { setShowForm(false); resetForm(); }}
            >
              <i className="bi bi-x-lg"></i>
            </button>
          </div>

          <form onSubmit={handleSubmit}>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '1rem' }}>
              <div className="form-group">
                <label className="form-label">Policy Name *</label>
                <input
                  type="text"
                  className="form-control"
                  value={policyName}
                  onChange={(e) => setPolicyName(e.target.value)}
                  placeholder="e.g., Payment Limits"
                  required
                />
              </div>
              <div className="form-group">
                <label className="form-label">Description</label>
                <input
                  type="text"
                  className="form-control"
                  value={policyDescription}
                  onChange={(e) => setPolicyDescription(e.target.value)}
                  placeholder="Describe this policy..."
                />
              </div>
            </div>

            <div className="form-group">
              <label className="form-label">Add Rules</label>
              <p style={{ fontSize: '0.8125rem', color: 'var(--text-muted)', marginBottom: '0.5rem' }}>
                Allowed rule types: Require Approval, Max Value, Allowed Roles, Time Window, Rate Limit. Custom is available for advanced JSON config.
              </p>
              <div style={{ display: 'flex', flexWrap: 'wrap', gap: '0.5rem', marginBottom: '1rem' }}>
                {ruleTypes.map((type) => (
                  <button
                    key={type.value}
                    type="button"
                    className="btn btn-secondary btn-sm"
                    onClick={() => addRule(type.value)}
                  >
                    <i className={`bi ${type.icon}`}></i>
                    {type.label}
                  </button>
                ))}
              </div>

              {rules.length === 0 ? (
                <div style={{ 
                  padding: '2rem', 
                  textAlign: 'center', 
                  background: 'var(--dark-bg)', 
                  borderRadius: '8px',
                  color: 'var(--text-muted)'
                }}>
                  Click a rule type above to add it
                </div>
              ) : (
                <div>
                  {rules.map((rule, index) => (
                    <div key={index} className="policy-card">
                      <div className="policy-header">
                        <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
                          <i className={`bi ${ruleTypes.find(t => t.value === rule.type)?.icon}`} style={{ color: 'var(--primary-color)' }}></i>
                          <span style={{ fontWeight: 500 }}>
                            {ruleTypes.find(t => t.value === rule.type)?.label}
                          </span>
                        </div>
                        <div style={{ display: 'flex', gap: '0.5rem', alignItems: 'center' }}>
                          <select
                            className="form-control"
                            style={{ width: 'auto', padding: '0.25rem 0.5rem', fontSize: '0.8125rem' }}
                            value={rule.failAction}
                            onChange={(e) => updateRule(index, 'failAction', e.target.value)}
                          >
                            <option value="deny">Deny</option>
                            <option value="warn">Warn</option>
                            <option value="approve">Request Approval</option>
                          </select>
                          <button
                            type="button"
                            className="btn btn-icon btn-secondary btn-sm"
                            onClick={() => removeRule(index)}
                          >
                            <i className="bi bi-trash"></i>
                          </button>
                        </div>
                      </div>
                      <div className="editor-container" style={{ marginTop: '0.5rem' }}>
                        <Editor
                          height="100px"
                          language="json"
                          theme="vs-dark"
                          value={rule.config}
                          onChange={(value) => updateRule(index, 'config', value || '')}
                          options={{
                            minimap: { enabled: false },
                            fontSize: 12,
                            lineNumbers: 'off',
                            folding: false,
                          }}
                        />
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>

            <div style={{ display: 'flex', gap: '0.75rem', marginTop: '1.5rem' }}>
              <button 
                type="button" 
                className="btn btn-secondary"
                onClick={() => { setShowForm(false); resetForm(); }}
              >
                Cancel
              </button>
              <button type="submit" className="btn btn-primary" disabled={saving || rules.length === 0}>
                {saving ? 'Creating...' : 'Create Policy'}
              </button>
            </div>
          </form>
        </div>
      )}

      {selectedTool && !showForm && !loading && (
        policies.length === 0 ? (
          <div className="empty-state">
            <i className="bi bi-shield"></i>
            <h3>No policies configured</h3>
            <p>Add governance policies to control tool access</p>
            <button className="btn btn-primary" onClick={() => setShowForm(true)}>
              <i className="bi bi-plus-lg"></i>
              Create First Policy
            </button>
          </div>
        ) : (
          <div>
            {policies.map((policy) => (
              <div key={policy.id} className="tool-card">
                <div className="tool-icon" style={{ background: '#6366f1', color: 'white' }}>
                  <i className="bi bi-shield-fill-check"></i>
                </div>
                <div className="tool-info">
                  <div className="tool-name">{policy.name}</div>
                  <div className="tool-description">
                    {policy.description || 'No description'}
                  </div>
                  <div style={{ marginTop: '0.5rem', display: 'flex', gap: '0.5rem', flexWrap: 'wrap', alignItems: 'center' }}>
                    <span className={`badge ${policy.enabled ? 'badge-success' : 'badge-warning'}`}>
                      {policy.enabled ? 'Enabled' : 'Disabled'}
                    </span>
                    {policy.rules?.map((rule, i) => (
                      <span key={i} className="badge badge-primary">
                        <i className={`bi ${ruleTypes.find(t => t.value === rule.type)?.icon}`} style={{ marginRight: '0.25rem' }}></i>
                        {ruleTypes.find(t => t.value === rule.type)?.label}
                      </span>
                    ))}
                  </div>
                </div>
                <div className="tool-actions">
                  <button 
                    className="btn btn-icon btn-secondary btn-sm"
                    onClick={() => {
                      setPolicyName(policy.name);
                      setPolicyDescription(policy.description || '');
                      setRules(policy.rules?.map(r => ({
                        type: r.type,
                        config: JSON.stringify(r.config || {}, null, 2),
                        failAction: r.fail_action || 'deny'
                      })) || []);
                      setShowForm(true);
                    }}
                    data-tooltip="Edit"
                  >
                    <i className="bi bi-pencil"></i>
                  </button>
                  <button 
                    className="btn btn-icon btn-secondary btn-sm"
                    onClick={() => handleDeletePolicy(policy.id)}
                    data-tooltip="Delete"
                  >
                    <i className="bi bi-trash"></i>
                  </button>
                </div>
              </div>
            ))}
          </div>
        )
      )}
    </div>
  );
}
