import { useState, useMemo, useEffect, useRef } from 'react';
import { toast } from 'react-toastify';
import type { Prompt, Tool } from '../types';
import { createPrompt } from '../services/api';

interface Props {
  serverId: string;
  prompts: Prompt[];
  tools?: Tool[];
  onPromptCreated: () => void;
  onPromptDeleted: (id: string) => void;
  focusPromptId?: string | null;
  onCloseEdit?: () => void;
}

export default function PromptEditor({
  serverId,
  prompts,
  tools = [],
  onPromptCreated,
  onPromptDeleted,
  focusPromptId,
  onCloseEdit,
}: Props) {
  const openedViaSidebarRef = useRef(false);
  const [showForm, setShowForm] = useState(false);
  const [showHelp, setShowHelp] = useState(false);
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [template, setTemplate] = useState('');
  const [saving, setSaving] = useState(false);
  const [previewValues, setPreviewValues] = useState<Record<string, string>>({});
  const [selectedTools, setSelectedTools] = useState<string[]>([]);

  // Extract variables from template ({{variable}} pattern)
  const detectedVariables = useMemo(() => {
    const matches = template.match(/\{\{(\w+)\}\}/g) || [];
    const vars = [...new Set(matches.map(m => m.replace(/\{\{|\}\}/g, '')))];
    return vars;
  }, [template]);

  // Generate preview with substituted values
  const previewTemplate = useMemo(() => {
    let preview = template;
    detectedVariables.forEach(v => {
      const value = previewValues[v] || `[${v}]`;
      preview = preview.replace(new RegExp(`\\{\\{${v}\\}\\}`, 'g'), value);
    });
    return preview;
  }, [template, detectedVariables, previewValues]);

  // Build arguments schema from detected variables
  const buildArgumentsSchema = () => {
    const properties: Record<string, { type: string; description: string }> = {};
    detectedVariables.forEach(v => {
      properties[v] = {
        type: 'string',
        description: `Value for ${v}`
      };
    });
    return {
      type: 'object',
      properties,
      required: detectedVariables
    };
  };

  const resetForm = () => {
    setName('');
    setDescription('');
    setTemplate('');
    setPreviewValues({});
    setSelectedTools([]);
  };

  const loadPromptIntoForm = (prompt: Prompt) => {
    setName(prompt.name);
    setDescription(prompt.description);
    setTemplate(prompt.template);
    setPreviewValues({});
    setSelectedTools([]);
  };

  useEffect(() => {
    if (!focusPromptId) {
      if (openedViaSidebarRef.current) {
        setShowForm(false);
        resetForm();
        openedViaSidebarRef.current = false;
      }
      return;
    }
    if (prompts.length === 0) return;
    const prompt = prompts.find((p) => p.id === focusPromptId);
    if (prompt) {
      openedViaSidebarRef.current = true;
      loadPromptIntoForm(prompt);
      setShowForm(true);
    }
  }, [focusPromptId, prompts]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    try {
      setSaving(true);
      
      const argumentsSchema = buildArgumentsSchema();

      await createPrompt({
        server_id: serverId,
        name,
        description,
        template,
        arguments: argumentsSchema,
      });

      toast.success('Prompt created');
      setShowForm(false);
      resetForm();
      onCloseEdit?.();
      onPromptCreated();
    } catch (error) {
      toast.error('Failed to create prompt');
    } finally {
      setSaving(false);
    }
  };

  const handleEditPrompt = (prompt: Prompt) => {
    openedViaSidebarRef.current = false;
    onCloseEdit?.();
    loadPromptIntoForm(prompt);
    setShowForm(true);
  };

  // Example templates for quick start
  const exampleTemplates = [
    {
      name: 'Summarize',
      template: 'Please summarize the following content in a clear and concise way:\n\n{{content}}',
      description: 'Summarize any text content'
    },
    {
      name: 'Code Review',
      template: 'Review the following {{language}} code for bugs, security issues, and improvements:\n\n```{{language}}\n{{code}}\n```',
      description: 'Review code for issues'
    },
    {
      name: 'Explain',
      template: 'Explain {{topic}} in simple terms that a {{audience}} would understand.',
      description: 'Explain a topic for a specific audience'
    }
  ];

  if (showForm) {
    return (
      <div className="card">
        <div className="card-header">
          <h3 className="card-title">
            <i className="bi bi-chat-text" style={{ marginRight: '0.5rem', color: '#f59e0b' }}></i>
            Create Prompt Template
          </h3>
          <button 
            type="button"
            className="btn btn-outline-primary btn-sm"
            onClick={() => {
              setShowForm(false);
              resetForm();
              openedViaSidebarRef.current = false;
              onCloseEdit?.();
            }}
            title="Return to prompts list"
            style={{ fontWeight: 600 }}
          >
            <i className="bi bi-arrow-left" style={{ marginRight: '0.35rem' }}></i>
            Back to list
          </button>
        </div>

        {/* Quick Start Templates */}
        <div style={{ marginBottom: '1.5rem' }}>
          <label className="form-label" style={{ marginBottom: '0.5rem' }}>Quick Start</label>
          <div style={{ display: 'flex', gap: '0.5rem', flexWrap: 'wrap' }}>
            {exampleTemplates.map((ex, i) => (
              <button
                key={i}
                type="button"
                className="btn btn-secondary btn-sm"
                onClick={() => {
                  setName(ex.name.toLowerCase().replace(/\s+/g, '_'));
                  setDescription(ex.description);
                  setTemplate(ex.template);
                }}
              >
                {ex.name}
              </button>
            ))}
          </div>
        </div>

        <form onSubmit={handleSubmit}>
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '1.5rem' }}>
            {/* Left Column - Template Definition */}
            <div>
              <div className="form-group">
                <label className="form-label">Prompt Name *</label>
                <input
                  type="text"
                  className="form-control"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  placeholder="e.g., summarize_document"
                  required
                />
                <small style={{ color: 'var(--text-muted)', fontSize: '0.75rem' }}>
                  This is how AI will invoke this prompt
                </small>
              </div>

              <div className="form-group">
                <label className="form-label">Description</label>
                <input
                  type="text"
                  className="form-control"
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  placeholder="Describe what this prompt helps accomplish..."
                />
              </div>

              <div className="form-group">
                <label className="form-label">
                  Template *
                  <span style={{ fontWeight: 'normal', color: 'var(--text-muted)', marginLeft: '0.5rem' }}>
                    Use <code style={{ background: 'var(--hover-bg)', padding: '0.125rem 0.375rem', borderRadius: '4px' }}>{'{{variable}}'}</code> for inputs
                  </span>
                </label>
                <textarea
                  className="form-control"
                  value={template}
                  onChange={(e) => setTemplate(e.target.value)}
                  placeholder="Write your prompt template here...

Example:
Please analyze the following {{document_type}} and provide a {{output_format}}:

{{content}}"
                  rows={10}
                  required
                  style={{ fontFamily: 'monospace', fontSize: '0.875rem' }}
                />
              </div>

              {/* Detected Variables */}
              {detectedVariables.length > 0 && (
                <div style={{ 
                  padding: '1rem', 
                  background: 'rgba(16, 185, 129, 0.1)', 
                  border: '1px solid rgba(16, 185, 129, 0.3)',
                  borderRadius: '8px',
                  marginTop: '1rem'
                }}>
                  <div style={{ fontSize: '0.8125rem', fontWeight: 600, color: 'var(--success-color)', marginBottom: '0.5rem' }}>
                    <i className="bi bi-check-circle" style={{ marginRight: '0.375rem' }}></i>
                    {detectedVariables.length} Variable{detectedVariables.length > 1 ? 's' : ''} Detected
                  </div>
                  <div style={{ display: 'flex', gap: '0.5rem', flexWrap: 'wrap' }}>
                    {detectedVariables.map(v => (
                      <span key={v} style={{ 
                        padding: '0.25rem 0.5rem',
                        background: 'var(--success-color)',
                        color: 'white',
                        borderRadius: '4px',
                        fontSize: '0.75rem',
                        fontFamily: 'monospace'
                      }}>
                        {`{{${v}}}`}
                      </span>
                    ))}
                  </div>
                </div>
              )}

              {/* Suggested Tools */}
              {tools.length > 0 && (
                <div className="form-group" style={{ marginTop: '1rem' }}>
                  <label className="form-label">
                    Suggest Tools
                    <span style={{ fontWeight: 'normal', color: 'var(--text-muted)', marginLeft: '0.5rem' }}>
                      (optional)
                    </span>
                  </label>
                  <div style={{ display: 'flex', gap: '0.5rem', flexWrap: 'wrap' }}>
                    {tools.slice(0, 6).map(tool => (
                      <button
                        key={tool.id}
                        type="button"
                        onClick={() => {
                          if (selectedTools.includes(tool.id)) {
                            setSelectedTools(selectedTools.filter(t => t !== tool.id));
                          } else {
                            setSelectedTools([...selectedTools, tool.id]);
                          }
                        }}
                        style={{
                          padding: '0.375rem 0.75rem',
                          background: selectedTools.includes(tool.id) ? 'var(--primary-light)' : 'var(--dark-bg)',
                          border: `1px solid ${selectedTools.includes(tool.id) ? 'var(--primary-color)' : 'var(--card-border)'}`,
                          borderRadius: '6px',
                          fontSize: '0.8125rem',
                          cursor: 'pointer',
                          color: selectedTools.includes(tool.id) ? 'var(--primary-color)' : 'var(--text-secondary)'
                        }}
                      >
                        <i className="bi bi-tools" style={{ marginRight: '0.375rem' }}></i>
                        {tool.name}
                      </button>
                    ))}
                  </div>
                  <small style={{ color: 'var(--text-muted)', fontSize: '0.75rem' }}>
                    Suggest tools that work well with this prompt
                  </small>
                </div>
              )}
            </div>

            {/* Right Column - Preview */}
            <div>
              <div style={{ 
                padding: '1rem', 
                background: 'var(--dark-bg)', 
                borderRadius: '8px',
                border: '1px solid var(--card-border)',
                height: '100%'
              }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '1rem' }}>
                  <label className="form-label" style={{ margin: 0 }}>
                    <i className="bi bi-eye" style={{ marginRight: '0.375rem' }}></i>
                    Live Preview
                  </label>
                </div>

                {/* Variable Inputs for Preview */}
                {detectedVariables.length > 0 && (
                  <div style={{ marginBottom: '1rem' }}>
                    <div style={{ fontSize: '0.75rem', color: 'var(--text-muted)', marginBottom: '0.5rem' }}>
                      Enter sample values to preview:
                    </div>
                    {detectedVariables.map(v => (
                      <div key={v} style={{ marginBottom: '0.5rem' }}>
                        <label style={{ 
                          fontSize: '0.75rem', 
                          color: 'var(--text-secondary)',
                          display: 'block',
                          marginBottom: '0.25rem'
                        }}>
                          {v}
                        </label>
                        <input
                          type="text"
                          className="form-control"
                          placeholder={`Enter ${v}...`}
                          value={previewValues[v] || ''}
                          onChange={(e) => setPreviewValues({ ...previewValues, [v]: e.target.value })}
                          style={{ fontSize: '0.8125rem', padding: '0.375rem 0.5rem' }}
                        />
                      </div>
                    ))}
                  </div>
                )}

                {/* Rendered Preview */}
                <div style={{ 
                  padding: '1rem',
                  background: '#1a1a2e',
                  borderRadius: '8px',
                  minHeight: '200px'
                }}>
                  <div style={{ fontSize: '0.6875rem', color: 'var(--text-muted)', marginBottom: '0.5rem', textTransform: 'uppercase' }}>
                    Rendered Output
                  </div>
                  <pre style={{ 
                    margin: 0, 
                    whiteSpace: 'pre-wrap', 
                    fontSize: '0.8125rem',
                    color: '#e5e7eb',
                    fontFamily: 'inherit',
                    lineHeight: 1.6
                  }}>
                    {previewTemplate || 'Start typing your template to see preview...'}
                  </pre>
                </div>

                {/* How it Works */}
                <div style={{ marginTop: '1rem', padding: '0.75rem', background: 'var(--hover-bg)', borderRadius: '6px' }}>
                  <div style={{ fontSize: '0.75rem', fontWeight: 600, color: 'var(--text-primary)', marginBottom: '0.375rem' }}>
                    <i className="bi bi-info-circle" style={{ marginRight: '0.375rem', color: 'var(--secondary-color)' }}></i>
                    How Prompts Work
                  </div>
                  <div style={{ fontSize: '0.75rem', color: 'var(--text-secondary)', lineHeight: 1.5 }}>
                    When an AI invokes this prompt, it will substitute the variables with actual values. 
                    The rendered text guides the AI's response.
                  </div>
                </div>
              </div>
            </div>
          </div>

          <div style={{ display: 'flex', gap: '0.75rem', marginTop: '1.5rem' }}>
            <button 
              type="button" 
              className="btn btn-secondary"
              onClick={() => {
                setShowForm(false);
                resetForm();
                openedViaSidebarRef.current = false;
                onCloseEdit?.();
              }}
            >
              Cancel
            </button>
            <button type="submit" className="btn btn-primary" disabled={saving || !name || !template}>
              {saving ? 'Creating...' : 'Create Prompt'}
            </button>
          </div>
        </form>
      </div>
    );
  }

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '1rem' }}>
        <div>
          <h3 style={{ margin: 0 }}>Prompts ({prompts.length})</h3>
        </div>
        <div style={{ display: 'flex', gap: '0.5rem' }}>
          <button 
            className="btn btn-secondary btn-sm"
            onClick={() => setShowHelp(!showHelp)}
          >
            <i className="bi bi-question-circle"></i>
            {showHelp ? 'Hide Help' : 'What are Prompts?'}
          </button>
          <button
            className="btn btn-primary"
            onClick={() => {
              openedViaSidebarRef.current = false;
              onCloseEdit?.();
              resetForm();
              setShowForm(true);
            }}
          >
            <i className="bi bi-plus-lg"></i>
            Add Prompt
          </button>
        </div>
      </div>

      {/* Help Section */}
      {showHelp && (
        <div style={{ 
          marginBottom: '1.5rem', 
          padding: '1.25rem', 
          background: 'var(--primary-light)', 
          border: '1px solid rgba(99, 102, 241, 0.2)',
          borderRadius: '12px'
        }}>
          <h4 style={{ margin: '0 0 0.75rem 0', color: 'var(--primary-color)', fontSize: '1rem' }}>
            <i className="bi bi-lightbulb" style={{ marginRight: '0.5rem' }}></i>
            Understanding MCP Prompts
          </h4>
          <div style={{ fontSize: '0.875rem', color: 'var(--text-secondary)', lineHeight: 1.7 }}>
            <p style={{ margin: '0 0 0.75rem 0' }}>
              <strong>Prompts</strong> are reusable message templates that AI clients can invoke. 
              They help standardize how users interact with your server's capabilities.
            </p>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '1rem' }}>
              <div>
                <strong style={{ color: 'var(--text-primary)' }}>Use Cases:</strong>
                <ul style={{ margin: '0.5rem 0', paddingLeft: '1.25rem' }}>
                  <li>Pre-defined analysis templates</li>
                  <li>Standardized code review prompts</li>
                  <li>Data transformation instructions</li>
                  <li>Domain-specific queries</li>
                </ul>
              </div>
              <div>
                <strong style={{ color: 'var(--text-primary)' }}>How They Work:</strong>
                <ul style={{ margin: '0.5rem 0', paddingLeft: '1.25rem' }}>
                  <li>Define template with <code style={{ background: 'white', padding: '0.125rem 0.25rem', borderRadius: '3px' }}>{'{{variables}}'}</code></li>
                  <li>AI fills in the variables when invoking</li>
                  <li>Rendered text guides AI response</li>
                  <li>Can suggest relevant tools to use</li>
                </ul>
              </div>
            </div>
          </div>
        </div>
      )}

      {prompts.length === 0 ? (
        <div className="empty-state">
          <i className="bi bi-chat-text"></i>
          <h3>No prompts yet</h3>
          <p>Prompts are reusable templates that help standardize AI interactions</p>
          <button
            className="btn btn-primary"
            onClick={() => {
              openedViaSidebarRef.current = false;
              onCloseEdit?.();
              resetForm();
              setShowForm(true);
            }}
          >
            <i className="bi bi-plus-lg"></i>
            Create First Prompt
          </button>
        </div>
      ) : (
        <div>
          {prompts.map((prompt) => {
            const vars = (prompt.template.match(/\{\{(\w+)\}\}/g) || [])
              .map(m => m.replace(/\{\{|\}\}/g, ''));
            
            return (
              <div key={prompt.id} className="tool-card">
                <div className="tool-icon" style={{ background: '#f59e0b', color: 'white' }}>
                  <i className="bi bi-chat-text-fill"></i>
                </div>
                <div className="tool-info">
                  <div className="tool-name">{prompt.name}</div>
                  <div className="tool-description">
                    {prompt.description || 'No description'}
                  </div>
                  
                  {/* Variables */}
                  {vars.length > 0 && (
                    <div style={{ marginTop: '0.5rem', display: 'flex', gap: '0.375rem', flexWrap: 'wrap' }}>
                      {vars.map(v => (
                        <span key={v} style={{ 
                          padding: '0.125rem 0.375rem',
                          background: 'rgba(245, 158, 11, 0.15)',
                          color: '#b45309',
                          borderRadius: '4px',
                          fontSize: '0.6875rem',
                          fontFamily: 'monospace'
                        }}>
                          {`{{${v}}}`}
                        </span>
                      ))}
                    </div>
                  )}

                  {/* Template Preview */}
                  <div style={{ 
                    marginTop: '0.5rem', 
                    padding: '0.5rem', 
                    background: '#1a1a2e', 
                    borderRadius: '4px',
                    fontSize: '0.75rem',
                    fontFamily: 'monospace',
                    color: '#e5e7eb',
                    maxHeight: '60px',
                    overflow: 'hidden',
                    textOverflow: 'ellipsis'
                  }}>
                    {prompt.template.substring(0, 150)}{prompt.template.length > 150 ? '...' : ''}
                  </div>
                </div>
                <div className="tool-actions">
                  <button 
                    className="btn btn-icon btn-secondary btn-sm"
                    onClick={() => handleEditPrompt(prompt)}
                    data-tooltip="Edit"
                  >
                    <i className="bi bi-pencil"></i>
                  </button>
                  <button 
                    className="btn btn-icon btn-secondary btn-sm"
                    onClick={() => onPromptDeleted(prompt.id)}
                    data-tooltip="Delete"
                  >
                    <i className="bi bi-trash"></i>
                  </button>
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
