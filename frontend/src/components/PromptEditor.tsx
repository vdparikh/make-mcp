import { useState } from 'react';
import { toast } from 'react-toastify';
import Editor from '@monaco-editor/react';
import type { Prompt } from '../types';
import { createPrompt } from '../services/api';

interface Props {
  serverId: string;
  prompts: Prompt[];
  onPromptCreated: () => void;
  onPromptDeleted: (id: string) => void;
}

export default function PromptEditor({ serverId, prompts, onPromptCreated, onPromptDeleted }: Props) {
  const [showForm, setShowForm] = useState(false);
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [template, setTemplate] = useState('');
  const [arguments_, setArguments] = useState('{\n  "type": "object",\n  "properties": {\n    "input": {\n      "type": "string",\n      "description": "The input text"\n    }\n  }\n}');
  const [saving, setSaving] = useState(false);

  const resetForm = () => {
    setName('');
    setDescription('');
    setTemplate('');
    setArguments('{\n  "type": "object",\n  "properties": {\n    "input": {\n      "type": "string",\n      "description": "The input text"\n    }\n  }\n}');
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    try {
      setSaving(true);
      
      let parsedArgs;
      try {
        parsedArgs = JSON.parse(arguments_);
      } catch {
        toast.error('Invalid Arguments JSON');
        return;
      }

      await createPrompt({
        server_id: serverId,
        name,
        description,
        template,
        arguments: parsedArgs,
      });

      toast.success('Prompt created');
      setShowForm(false);
      resetForm();
      onPromptCreated();
    } catch (error) {
      toast.error('Failed to create prompt');
    } finally {
      setSaving(false);
    }
  };

  if (showForm) {
    return (
      <div className="card">
        <div className="card-header">
          <h3 className="card-title">Create New Prompt</h3>
          <button 
            className="btn btn-icon btn-secondary"
            onClick={() => { setShowForm(false); resetForm(); }}
          >
            <i className="bi bi-x-lg"></i>
          </button>
        </div>

        <form onSubmit={handleSubmit}>
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
          </div>

          <div className="form-group">
            <label className="form-label">Description</label>
            <input
              type="text"
              className="form-control"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Describe what this prompt does..."
            />
          </div>

          <div className="form-group">
            <label className="form-label">
              Template *
              <span style={{ fontWeight: 'normal', color: 'var(--text-muted)', marginLeft: '0.5rem' }}>
                (Use {'{{variable}}'} for placeholders)
              </span>
            </label>
            <textarea
              className="form-control"
              value={template}
              onChange={(e) => setTemplate(e.target.value)}
              placeholder="Summarize the following document:

{{document}}"
              rows={6}
              required
              style={{ fontFamily: 'monospace' }}
            />
          </div>

          <div className="form-group">
            <label className="form-label">Arguments Schema</label>
            <div className="editor-container">
              <Editor
                height="150px"
                language="json"
                theme="vs-dark"
                value={arguments_}
                onChange={(value) => setArguments(value || '')}
                options={{
                  minimap: { enabled: false },
                  fontSize: 13,
                  lineNumbers: 'off',
                  folding: false,
                }}
              />
            </div>
          </div>

          <div style={{ display: 'flex', gap: '0.75rem', marginTop: '1.5rem' }}>
            <button 
              type="button" 
              className="btn btn-secondary"
              onClick={() => { setShowForm(false); resetForm(); }}
            >
              Cancel
            </button>
            <button type="submit" className="btn btn-primary" disabled={saving}>
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
        <h3 style={{ margin: 0 }}>Prompts ({prompts.length})</h3>
        <button className="btn btn-primary" onClick={() => setShowForm(true)}>
          <i className="bi bi-plus-lg"></i>
          Add Prompt
        </button>
      </div>

      {prompts.length === 0 ? (
        <div className="empty-state">
          <i className="bi bi-chat-text"></i>
          <h3>No prompts yet</h3>
          <p>Prompts are templated instructions for AI</p>
          <button className="btn btn-primary" onClick={() => setShowForm(true)}>
            <i className="bi bi-plus-lg"></i>
            Create First Prompt
          </button>
        </div>
      ) : (
        <div>
          {prompts.map((prompt) => (
            <div key={prompt.id} className="tool-card">
              <div className="tool-icon" style={{ background: 'linear-gradient(135deg, #f59e0b, #d97706)' }}>
                <i className="bi bi-chat-text-fill"></i>
              </div>
              <div className="tool-info">
                <div className="tool-name">{prompt.name}</div>
                <div className="tool-description">
                  {prompt.description || 'No description'}
                </div>
                <div style={{ 
                  marginTop: '0.5rem', 
                  padding: '0.5rem', 
                  background: 'var(--dark-bg)', 
                  borderRadius: '4px',
                  fontSize: '0.8125rem',
                  fontFamily: 'monospace',
                  color: 'var(--text-muted)',
                  maxHeight: '60px',
                  overflow: 'hidden',
                  textOverflow: 'ellipsis'
                }}>
                  {prompt.template.substring(0, 150)}...
                </div>
              </div>
              <div>
                <button 
                  className="btn btn-icon btn-secondary btn-sm"
                  onClick={() => onPromptDeleted(prompt.id)}
                  title="Delete"
                >
                  <i className="bi bi-trash"></i>
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
