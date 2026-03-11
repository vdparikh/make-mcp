import { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { toast } from 'react-toastify';
import type { Server, ContextConfig } from '../types';
import { 
  getServer, 
  updateServer, 
  generateServer,
  deleteTool,
  deleteResource,
  deletePrompt,
  getContextConfigs,
} from '../services/api';
import ToolEditor from '../components/ToolEditor';
import ResourceEditor from '../components/ResourceEditor';
import PromptEditor from '../components/PromptEditor';
import ContextConfigEditor from '../components/ContextConfigEditor';
import PolicyEditor from '../components/PolicyEditor';
import TestPlayground from '../components/TestPlayground';
import HealingDashboard from '../components/HealingDashboard';

type TabType = 'general' | 'tools' | 'resources' | 'prompts' | 'context' | 'policies' | 'testing' | 'healing' | 'deploy';

export default function ServerEditor() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  
  const [server, setServer] = useState<Server | null>(null);
  const [loading, setLoading] = useState(true);
  const [activeTab, setActiveTab] = useState<TabType>('general');
  const [contextConfigs, setContextConfigs] = useState<ContextConfig[]>([]);
  const [generating, setGenerating] = useState(false);
  
  const [editName, setEditName] = useState('');
  const [editDescription, setEditDescription] = useState('');
  const [editVersion, setEditVersion] = useState('');

  useEffect(() => {
    if (id) {
      loadServer();
    }
  }, [id]);

  const loadServer = async () => {
    try {
      setLoading(true);
      const data = await getServer(id!);
      setServer(data);
      setEditName(data.name);
      setEditDescription(data.description);
      setEditVersion(data.version);
      
      const configs = await getContextConfigs(id!);
      setContextConfigs(configs);
    } catch (error) {
      toast.error('Failed to load server');
      navigate('/');
    } finally {
      setLoading(false);
    }
  };

  const handleUpdateServer = async () => {
    try {
      await updateServer(id!, {
        name: editName,
        description: editDescription,
        version: editVersion,
      });
      toast.success('Server updated');
      loadServer();
    } catch (error) {
      toast.error('Failed to update server');
    }
  };

  const handleGenerate = async () => {
    try {
      setGenerating(true);
      const blob = await generateServer(id!);
      
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `${server?.name}-mcp-server.zip`;
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      document.body.removeChild(a);
      
      toast.success('Server generated and downloaded');
    } catch (error) {
      toast.error('Failed to generate server');
    } finally {
      setGenerating(false);
    }
  };

  const handleDeleteTool = async (toolId: string) => {
    if (!confirm('Delete this tool?')) return;
    try {
      await deleteTool(toolId);
      toast.success('Tool deleted');
      loadServer();
    } catch (error) {
      toast.error('Failed to delete tool');
    }
  };

  const handleDeleteResource = async (resourceId: string) => {
    if (!confirm('Delete this resource?')) return;
    try {
      await deleteResource(resourceId);
      toast.success('Resource deleted');
      loadServer();
    } catch (error) {
      toast.error('Failed to delete resource');
    }
  };

  const handleDeletePrompt = async (promptId: string) => {
    if (!confirm('Delete this prompt?')) return;
    try {
      await deletePrompt(promptId);
      toast.success('Prompt deleted');
      loadServer();
    } catch (error) {
      toast.error('Failed to delete prompt');
    }
  };

  if (loading) {
    return (
      <div className="loading">
        <div className="spinner"></div>
      </div>
    );
  }

  if (!server) {
    return <div>Server not found</div>;
  }

  const tabs: { id: TabType; label: string; icon: string }[] = [
    { id: 'general', label: 'General', icon: 'bi-gear' },
    { id: 'tools', label: 'Tools', icon: 'bi-tools' },
    { id: 'resources', label: 'Resources', icon: 'bi-folder' },
    { id: 'prompts', label: 'Prompts', icon: 'bi-chat-text' },
    { id: 'context', label: 'Context', icon: 'bi-person-badge' },
    { id: 'policies', label: 'Policies', icon: 'bi-shield-check' },
    { id: 'testing', label: 'Testing', icon: 'bi-play-circle' },
    { id: 'healing', label: 'Healing', icon: 'bi-bandaid' },
    { id: 'deploy', label: 'Deploy', icon: 'bi-rocket-takeoff' },
  ];

  return (
    <div>
      <div className="page-header">
        <div>
          <div style={{ display: 'flex', alignItems: 'center', gap: '0.75rem' }}>
            <button 
              className="btn btn-icon btn-secondary"
              onClick={() => navigate('/')}
            >
              <i className="bi bi-arrow-left"></i>
            </button>
            <h1 className="page-title">{server.name}</h1>
            <span className="badge badge-primary">v{server.version}</span>
          </div>
          <p className="page-subtitle" style={{ marginLeft: '2.75rem' }}>
            {server.description || 'No description'}
          </p>
        </div>
        <button 
          className="btn btn-success" 
          onClick={handleGenerate}
          disabled={generating}
        >
          {generating ? (
            <>
              <span className="spinner" style={{ width: 16, height: 16, borderWidth: 2 }}></span>
              Generating...
            </>
          ) : (
            <>
              <i className="bi bi-download"></i>
              Generate & Download
            </>
          )}
        </button>
      </div>

      <div className="tabs">
        {tabs.map((tab) => (
          <button
            key={tab.id}
            className={`tab ${activeTab === tab.id ? 'active' : ''}`}
            onClick={() => setActiveTab(tab.id)}
          >
            <i className={`bi ${tab.icon}`} style={{ marginRight: '0.5rem' }}></i>
            {tab.label}
            {tab.id === 'tools' && server.tools && (
              <span className="badge badge-primary" style={{ marginLeft: '0.5rem' }}>
                {server.tools.length}
              </span>
            )}
            {tab.id === 'resources' && server.resources && (
              <span className="badge badge-primary" style={{ marginLeft: '0.5rem' }}>
                {server.resources.length}
              </span>
            )}
            {tab.id === 'prompts' && server.prompts && (
              <span className="badge badge-primary" style={{ marginLeft: '0.5rem' }}>
                {server.prompts.length}
              </span>
            )}
          </button>
        ))}
      </div>

      {activeTab === 'general' && (
        <div className="card">
          <h3 className="card-title" style={{ marginBottom: '1.5rem' }}>Server Configuration</h3>
          
          <div className="form-group">
            <label className="form-label">Server Name</label>
            <input
              type="text"
              className="form-control"
              value={editName}
              onChange={(e) => setEditName(e.target.value)}
            />
          </div>

          <div className="form-group">
            <label className="form-label">Description</label>
            <textarea
              className="form-control"
              value={editDescription}
              onChange={(e) => setEditDescription(e.target.value)}
              rows={3}
            />
          </div>

          <div className="form-group">
            <label className="form-label">Version</label>
            <input
              type="text"
              className="form-control"
              value={editVersion}
              onChange={(e) => setEditVersion(e.target.value)}
              placeholder="1.0.0"
            />
          </div>

          <button className="btn btn-primary" onClick={handleUpdateServer}>
            <i className="bi bi-check-lg"></i>
            Save Changes
          </button>
        </div>
      )}

      {activeTab === 'tools' && (
        <ToolEditor
          serverId={id!}
          tools={server.tools || []}
          onToolCreated={loadServer}
          onToolDeleted={handleDeleteTool}
        />
      )}

      {activeTab === 'resources' && (
        <ResourceEditor
          serverId={id!}
          resources={server.resources || []}
          onResourceCreated={loadServer}
          onResourceDeleted={handleDeleteResource}
        />
      )}

      {activeTab === 'prompts' && (
        <PromptEditor
          serverId={id!}
          prompts={server.prompts || []}
          onPromptCreated={loadServer}
          onPromptDeleted={handleDeletePrompt}
        />
      )}

      {activeTab === 'context' && (
        <ContextConfigEditor
          serverId={id!}
          configs={contextConfigs}
          onConfigCreated={loadServer}
        />
      )}

      {activeTab === 'policies' && (
        <PolicyEditor
          tools={server.tools || []}
          onPolicyUpdated={loadServer}
        />
      )}

      {activeTab === 'testing' && (
        <TestPlayground
          tools={server.tools || []}
        />
      )}

      {activeTab === 'healing' && (
        <HealingDashboard
          tools={server.tools || []}
        />
      )}

      {activeTab === 'deploy' && (
        <div className="card">
          <h3 className="card-title" style={{ marginBottom: '1.5rem' }}>
            <i className="bi bi-rocket-takeoff" style={{ marginRight: '0.75rem' }}></i>
            Deploy Options
          </h3>
          
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(250px, 1fr))', gap: '1rem' }}>
            <div className="card" style={{ background: 'var(--dark-bg)', border: '1px solid var(--primary-color)' }}>
              <div style={{ fontSize: '2rem', marginBottom: '0.75rem', color: 'var(--primary-color)' }}>
                <i className="bi bi-file-earmark-zip"></i>
              </div>
              <h4 style={{ marginBottom: '0.5rem', color: 'var(--text-primary)' }}>Download ZIP</h4>
              <p style={{ color: 'var(--text-secondary)', fontSize: '0.875rem', marginBottom: '1rem' }}>
                Download as a Node.js project ready to run
              </p>
              <button className="btn btn-primary btn-sm" onClick={handleGenerate}>
                <i className="bi bi-download"></i>
                Download
              </button>
            </div>

            <div className="card" style={{ background: 'var(--dark-bg)', border: '1px solid var(--card-border)' }}>
              <div style={{ fontSize: '2rem', marginBottom: '0.75rem', color: 'var(--text-muted)' }}>
                <i className="bi bi-box-seam"></i>
              </div>
              <h4 style={{ marginBottom: '0.5rem', color: 'var(--text-secondary)' }}>Docker Image</h4>
              <p style={{ color: 'var(--text-muted)', fontSize: '0.875rem', marginBottom: '1rem' }}>
                Build and push Docker image
              </p>
              <span style={{ fontSize: '0.8125rem', color: 'var(--text-muted)', fontStyle: 'italic' }}>
                Coming Soon
              </span>
            </div>

            <div className="card" style={{ background: 'var(--dark-bg)', border: '1px solid var(--card-border)' }}>
              <div style={{ fontSize: '2rem', marginBottom: '0.75rem', color: 'var(--text-muted)' }}>
                <i className="bi bi-cloud-upload"></i>
              </div>
              <h4 style={{ marginBottom: '0.5rem', color: 'var(--text-secondary)' }}>Cloud Deploy</h4>
              <p style={{ color: 'var(--text-muted)', fontSize: '0.875rem', marginBottom: '1rem' }}>
                Deploy to AWS, GCP, or Vercel
              </p>
              <span style={{ fontSize: '0.8125rem', color: 'var(--text-muted)', fontStyle: 'italic' }}>
                Coming Soon
              </span>
            </div>

            <div className="card" style={{ background: 'var(--dark-bg)', border: '1px solid var(--card-border)' }}>
              <div style={{ fontSize: '2rem', marginBottom: '0.75rem', color: 'var(--text-muted)' }}>
                <i className="bi bi-github"></i>
              </div>
              <h4 style={{ marginBottom: '0.5rem', color: 'var(--text-secondary)' }}>GitHub Export</h4>
              <p style={{ color: 'var(--text-muted)', fontSize: '0.875rem', marginBottom: '1rem' }}>
                Push to a GitHub repository
              </p>
              <span style={{ fontSize: '0.8125rem', color: 'var(--text-muted)', fontStyle: 'italic' }}>
                Coming Soon
              </span>
            </div>
          </div>

          <div style={{ marginTop: '2rem', padding: '1.25rem', background: 'linear-gradient(135deg, rgba(129, 140, 248, 0.1), rgba(56, 189, 248, 0.05))', border: '1px solid rgba(129, 140, 248, 0.2)', borderRadius: '12px' }}>
            <h4 style={{ marginBottom: '0.75rem', color: 'var(--text-primary)' }}>
              <i className="bi bi-play-circle" style={{ marginRight: '0.5rem', color: 'var(--success-color)' }}></i>
              How to Run Your MCP Server
            </h4>
            <div style={{ color: 'var(--text-secondary)', fontSize: '0.875rem', marginBottom: '1rem' }}>
              <p style={{ marginBottom: '0.5rem' }}>After downloading, run these commands:</p>
              <pre style={{ 
                background: 'rgba(0, 0, 0, 0.4)', 
                padding: '1rem', 
                borderRadius: '8px',
                overflow: 'auto',
                fontSize: '0.8125rem',
                color: '#a5f3fc',
                margin: '0.5rem 0'
              }}>
{`cd ${server.name.replace(/\s+/g, '-').toLowerCase()}-mcp-server
npm install
npm run build
npm start`}
              </pre>
            </div>
          </div>

          <div style={{ marginTop: '1.5rem', padding: '1.25rem', background: 'var(--dark-bg)', border: '1px solid var(--card-border)', borderRadius: '12px' }}>
            <h4 style={{ marginBottom: '0.75rem', color: 'var(--text-primary)' }}>
              <i className="bi bi-terminal" style={{ marginRight: '0.5rem', color: 'var(--secondary-color)' }}></i>
              MCP Client Configuration
            </h4>
            <p style={{ color: 'var(--text-secondary)', fontSize: '0.875rem', marginBottom: '1rem' }}>
              Add this to your MCP client (e.g., Claude Desktop, Cursor) config after building:
            </p>
            <pre style={{ 
              background: 'rgba(0, 0, 0, 0.5)', 
              padding: '1rem', 
              borderRadius: '8px',
              overflow: 'auto',
              fontSize: '0.8125rem',
              color: '#fde68a',
              border: '1px solid rgba(253, 230, 138, 0.2)'
            }}>
{`{
  "mcpServers": {
    "${server.name.replace(/\s+/g, '-').toLowerCase()}": {
      "command": "node",
      "args": ["/full/path/to/${server.name.replace(/\s+/g, '-').toLowerCase()}-mcp-server/dist/server.js"]
    }
  }
}`}
            </pre>
          </div>
        </div>
      )}
    </div>
  );
}
