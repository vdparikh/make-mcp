import { useState, useEffect } from 'react';
import { useParams, useNavigate, Link } from 'react-router-dom';
import { toast } from 'react-toastify';
import type { Server, ContextConfig, ServerVersion, SecurityScoreResult, ObservabilitySummaryResponse, EnvProfile, EnvProfileKey, EnvProfilesMap } from '../types';
import { 
  getServer, 
  updateServer, 
  generateServer,
  deleteTool,
  deleteResource,
  deletePrompt,
  getContextConfigs,
  deleteContextConfig,
  githubExport,
  publishServer,
  hostedPublish,
  hostedStatus,
  getServerVersions,
  downloadServerVersion,
  getSecurityScore,
  getServerObservability,
  enableServerObservability,
  getEnvProfiles,
  updateEnvProfiles,
} from '../services/api';
import type { HostedPublishResponse, HostedStatusResponse } from '../services/api';
import ToolEditor, { type ToolSection } from '../components/ToolEditor';
import ResourceEditor from '../components/ResourceEditor';
import PromptEditor from '../components/PromptEditor';
import ContextConfigEditor from '../components/ContextConfigEditor';
import PolicyEditor from '../components/PolicyEditor';
import TestPlayground from '../components/TestPlayground';
import HealingDashboard from '../components/HealingDashboard';

type TabType = 'general' | 'environments' | 'tools' | 'resources' | 'prompts' | 'context' | 'policies' | 'security' | 'testing' | 'healing' | 'observability' | 'deploy' | 'versions';

function serverSlug(name: string): string {
  return name.replace(/\s+/g, '-').toLowerCase().replace(/[^a-z0-9-]/g, '');
}

function SecurityScorePanel({ serverId }: { serverId: string }) {
  const [result, setResult] = useState<SecurityScoreResult | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError(null);
    getSecurityScore(serverId)
      .then((data) => {
        if (!cancelled) setResult(data);
      })
      .catch((err) => {
        if (!cancelled) setError(err.response?.data?.error || err.message || 'Failed to load security score');
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => { cancelled = true; };
  }, [serverId]);

  if (loading) {
    return (
      <div className="card">
        <div style={{ padding: '2rem', textAlign: 'center', color: 'var(--text-muted)' }}>
          <span className="spinner" style={{ width: 24, height: 24, borderWidth: 2 }}></span>
          <p style={{ marginTop: '0.75rem' }}>Calculating security score…</p>
        </div>
      </div>
    );
  }
  if (error || !result) {
    return (
      <div className="card">
        <div style={{ padding: '2rem', textAlign: 'center', color: 'var(--danger)' }}>
          <i className="bi bi-exclamation-triangle" style={{ fontSize: '2rem' }}></i>
          <p style={{ marginTop: '0.75rem' }}>{error || 'Unable to load security score'}</p>
        </div>
      </div>
    );
  }

  const gradeColor =
    result.grade === 'A' ? '#16a34a' :
    result.grade === 'B' ? '#2563eb' :
    result.grade === 'C' ? '#ca8a04' :
    result.grade === 'D' ? '#ea580c' : '#dc2626';

  return (
    <div className="card">
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', flexWrap: 'wrap', gap: '1rem', marginBottom: '1.5rem' }}>
        <div>
          <h3 className="card-title" style={{ marginBottom: '0.25rem' }}>
            <i className="bi bi-shield-lock" style={{ marginRight: '0.75rem' }}></i>
            Security Score
          </h3>
          <p style={{ color: 'var(--text-secondary)', margin: 0, fontSize: '0.9rem' }}>
            Based on the <a href={result.checklist_url} target="_blank" rel="noopener noreferrer">SlowMist MCP Security Checklist</a>
          </p>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: '1rem' }}>
          <div style={{
            width: 72,
            height: 72,
            borderRadius: '50%',
            background: `linear-gradient(135deg, ${gradeColor}22, ${gradeColor}44)`,
            border: `3px solid ${gradeColor}`,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            fontSize: '1.75rem',
            fontWeight: 700,
            color: gradeColor,
          }}>
            {result.grade}
          </div>
          <div>
            <div style={{ fontSize: '1.5rem', fontWeight: 700, color: 'var(--text-primary)' }}>{result.score}%</div>
            <div style={{ fontSize: '0.8rem', color: 'var(--text-muted)' }}>{result.earned} / {result.max_points} points</div>
          </div>
        </div>
      </div>
      <div style={{ borderTop: '1px solid var(--card-border)', paddingTop: '1rem' }}>
        <h4 style={{ marginBottom: '0.75rem', fontSize: '0.95rem' }}>Checklist criteria</h4>
        <ul style={{ listStyle: 'none', padding: 0, margin: 0 }}>
          {result.criteria.map((c) => (
            <li
              key={c.id}
              style={{
                display: 'flex',
                alignItems: 'flex-start',
                gap: '0.5rem',
                padding: '0.5rem 0',
                borderBottom: '1px solid var(--card-border)',
              }}
            >
              <i
                className={c.met ? 'bi bi-check-circle-fill' : 'bi bi-x-circle'}
                style={{ color: c.met ? '#16a34a' : 'var(--text-muted)', marginTop: '2px', flexShrink: 0 }}
              />
              <div>
                <span style={{ fontWeight: 500 }}>{c.name}</span>
                <span style={{ marginLeft: '0.5rem', fontSize: '0.75rem', color: 'var(--text-muted)' }}>
                  ({c.priority})
                </span>
                {c.reason && !c.met && (
                  <div style={{ fontSize: '0.8rem', color: 'var(--text-muted)', marginTop: '0.25rem' }}>{c.reason}</div>
                )}
              </div>
            </li>
          ))}
        </ul>
      </div>
      <p style={{ marginTop: '1rem', fontSize: '0.85rem', color: 'var(--text-muted)' }}>
        Improve your score by addressing the unmet criteria above. When you publish, this score is shown in the marketplace.
      </p>
    </div>
  );
}

function ObservabilityPanel({ serverId }: { serverId: string }) {
  const [data, setData] = useState<ObservabilitySummaryResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [enabling, setEnabling] = useState(false);

  const fetchObservability = () => {
    setLoading(true);
    setError(null);
    getServerObservability(serverId)
      .then(setData)
      .catch((err) => setError(err.response?.data?.error || err.message || 'Failed to load observability'))
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    fetchObservability();
  }, [serverId]);

  const handleEnable = () => {
    setEnabling(true);
    enableServerObservability(serverId)
      .then(() => {
        toast.success('Observability reporting enabled. Set the env vars in your deployed server.');
        fetchObservability();
      })
      .catch((err) => toast.error(err.response?.data?.error || err.message || 'Failed to enable'))
      .finally(() => setEnabling(false));
  };

  if (loading && !data) {
    return (
      <div className="card">
        <div style={{ padding: '2rem', textAlign: 'center', color: 'var(--text-muted)' }}>
          <span className="spinner" style={{ width: 24, height: 24, borderWidth: 2 }}></span>
          <p style={{ marginTop: '0.75rem' }}>Loading observability…</p>
        </div>
      </div>
    );
  }
  if (error && !data) {
    return (
      <div className="card">
        <div style={{ padding: '2rem', textAlign: 'center', color: 'var(--danger)' }}>
          <i className="bi bi-exclamation-triangle" style={{ fontSize: '2rem' }}></i>
          <p style={{ marginTop: '0.75rem' }}>{error}</p>
        </div>
      </div>
    );
  }

  const hasKey = data?.reporting_key;

  return (
    <div>
      <div className="card" style={{ marginBottom: '1.5rem' }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', flexWrap: 'wrap', gap: '1rem' }}>
          <div>
            <h3 className="card-title" style={{ marginBottom: '0.25rem' }}>
              <i className="bi bi-graph-up" style={{ marginRight: '0.75rem' }}></i>
              Enable runtime observability
            </h3>
            <p style={{ color: 'var(--text-secondary)', margin: 0, fontSize: '0.9rem' }}>
              Send tool calls, latency, and failures from this server to the Observability dashboard when it runs in Cursor, Claude, or any MCP client.
            </p>
          </div>
          {!hasKey && (
            <button className="btn btn-primary" onClick={handleEnable} disabled={enabling}>
              {enabling ? <span className="spinner" style={{ width: 16, height: 16, borderWidth: 2 }}></span> : <i className="bi bi-broadcast" />}
              {' '}Enable reporting
            </button>
          )}
        </div>
        {hasKey && data?.endpoint_url && (
          <div style={{ marginTop: '1rem', padding: '1rem', background: 'var(--background-secondary)', borderRadius: '8px', fontSize: '0.85rem' }}>
            <div style={{ marginBottom: '0.5rem', fontWeight: 600 }}>Environment variables for your deployed server</div>
            <pre style={{ margin: 0, whiteSpace: 'pre-wrap', wordBreak: 'break-all' }}>
              MCP_OBSERVABILITY_ENDPOINT={data.endpoint_url}{'\n'}
              MCP_OBSERVABILITY_KEY={data.reporting_key}
            </pre>
            <p style={{ marginTop: '0.75rem', color: 'var(--text-muted)', fontSize: '0.8rem' }}>
              After setting these and restarting the server, tool calls will appear in the Observability dashboard. Optionally set <code>MCP_OBSERVABILITY_USER_ID</code> and <code>MCP_OBSERVABILITY_CLIENT_AGENT</code> (e.g. Cursor, Claude Desktop) so you can see who and which client each call came from when many users share the same MCP.
            </p>
            <p style={{ marginTop: '0.5rem' }}>
              <Link to={`/observability?server_id=${serverId}`} className="btn btn-secondary btn-sm">
                <i className="bi bi-graph-up" style={{ marginRight: '0.5rem' }}></i>
                View observability dashboard
              </Link>
            </p>
          </div>
        )}
      </div>

      {!hasKey && !loading && (
        <div className="card">
          <div style={{ textAlign: 'center', padding: '2rem', color: 'var(--text-muted)' }}>
            <i className="bi bi-graph-up" style={{ fontSize: '2.5rem', marginBottom: '0.5rem' }}></i>
            <p>Enable reporting above, then set the env vars in your deployed MCP server. View all tool calls, latency, failures, and repair suggestions on the <Link to="/observability" style={{ color: 'var(--primary-color)' }}>Observability</Link> page.</p>
          </div>
        </div>
      )}
    </div>
  );
}

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
  const [editIcon, setEditIcon] = useState('bi-server');
  
  // Environments tab state (Dev / Staging / Prod profiles)
  const [envProfiles, setEnvProfiles] = useState<EnvProfilesMap>({});
  const [envProfilesLoading, setEnvProfilesLoading] = useState(false);
  const [envProfilesSaving, setEnvProfilesSaving] = useState(false);

  // Deploy state
  type DeployType = 'nodejs' | 'docker' | 'github' | 'azure' | 'hosted' | null;
  const [selectedDeploy, setSelectedDeploy] = useState<DeployType>(null);
  const [showDeployModal, setShowDeployModal] = useState(false);
  const [deployTargetEnv, setDeployTargetEnv] = useState<EnvProfileKey | ''>('');
  // Hosted (Publish MCP) state
  const [hostedPublishVersion, setHostedPublishVersion] = useState('');
  const [hostedPublishing, setHostedPublishing] = useState(false);
  const [hostedResult, setHostedResult] = useState<HostedPublishResponse | null>(null);
  const [hostedRuntime, setHostedRuntime] = useState<HostedStatusResponse | null>(null);
  const [hostedStatusLoading, setHostedStatusLoading] = useState(false);
  
  // GitHub export state
  const [showGitHubModal, setShowGitHubModal] = useState(false);
  const [githubToken, setGithubToken] = useState('');
  const [githubOwner, setGithubOwner] = useState('');
  const [githubRepo, setGithubRepo] = useState('');
  const [githubBranch, setGithubBranch] = useState('main');
  const [githubCommitMsg, setGithubCommitMsg] = useState('');
  const [githubCreateRepo, setGithubCreateRepo] = useState(false);
  const [githubPrivate, setGithubPrivate] = useState(true);
  const [githubExporting, setGithubExporting] = useState(false);
  
  // Publish & Versioning state
  const [showPublishModal, setShowPublishModal] = useState(false);
  const [publishVersion, setPublishVersion] = useState('');
  const [publishNotes, setPublishNotes] = useState('');
  const [publishPublic, setPublishPublic] = useState(false);
  const [publishing, setPublishing] = useState(false);
  const [versions, setVersions] = useState<ServerVersion[]>([]);
  const [showToolsTree, setShowToolsTree] = useState(true);
  const [showResourcesTree, setShowResourcesTree] = useState(true);
  const [showPromptsTree, setShowPromptsTree] = useState(true);
  const [focusedToolId, setFocusedToolId] = useState<string | null>(null);
  const [selectedResourceId, setSelectedResourceId] = useState<string | null>(null);
  const [selectedPromptId, setSelectedPromptId] = useState<string | null>(null);
  const [preselectedToolId, setPreselectedToolId] = useState<string | null>(null);

  // Clear tree selection when switching away from that section
  useEffect(() => {
    if (activeTab !== 'tools' && focusedToolId) {
      setFocusedToolId(null);
    }
    if (activeTab !== 'resources' && selectedResourceId) {
      setSelectedResourceId(null);
    }
    if (activeTab !== 'prompts' && selectedPromptId) {
      setSelectedPromptId(null);
    }
    setPreselectedToolId(null);
  }, [activeTab]);

  useEffect(() => {
    if (id) {
      loadServer();
    }
  }, [id]);

  useEffect(() => {
    if (activeTab === 'environments' && id) {
      let cancelled = false;
      setEnvProfilesLoading(true);
      getEnvProfiles(id)
        .then((data) => {
          if (!cancelled) setEnvProfiles(data);
        })
        .catch(() => {
          if (!cancelled) setEnvProfiles({});
        })
        .finally(() => {
          if (!cancelled) setEnvProfilesLoading(false);
        });
      return () => { cancelled = true; };
    }
  }, [activeTab, id]);

  useEffect(() => {
    if (!id || !showDeployModal || selectedDeploy !== 'hosted') return;
    let cancelled = false;
    setHostedStatusLoading(true);
    hostedStatus(id)
      .then((status) => {
        if (!cancelled) setHostedRuntime(status);
      })
      .catch(() => {
        if (!cancelled) setHostedRuntime({ running: false });
      })
      .finally(() => {
        if (!cancelled) setHostedStatusLoading(false);
      });
    return () => { cancelled = true; };
  }, [id, showDeployModal, selectedDeploy]);

  const loadServer = async () => {
    try {
      setLoading(true);
      const data = await getServer(id!);
      setServer(data);
      setEditName(data.name);
      setEditDescription(data.description);
      setEditVersion(data.version);
      setEditIcon(data.icon || 'bi-server');
      
      const configs = await getContextConfigs(id!);
      setContextConfigs(configs);
      
      const versionList = await getServerVersions(id!);
      setVersions(versionList);
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
        icon: editIcon,
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
      const blob = await generateServer(id!, deployTargetEnv || undefined);
      
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `${server ? serverSlug(server.name) : 'mcp-server'}-mcp-server.zip`;
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

  const setEnvProfileField = (key: EnvProfileKey, field: 'base_url' | 'db_url', value: string) => {
    setEnvProfiles((prev) => {
      const next = { ...prev };
      const cur = (next[key] ?? {}) as EnvProfile;
      next[key] = { ...cur, [field]: value || undefined };
      return next;
    });
  };

  const handleSaveEnvProfiles = async () => {
    if (!id) return;
    try {
      setEnvProfilesSaving(true);
      await updateEnvProfiles(id, envProfiles);
      toast.success('Environment profiles saved');
    } catch (err) {
      toast.error('Failed to save environment profiles');
    } finally {
      setEnvProfilesSaving(false);
    }
  };

  const openPublishModal = () => {
    const currentVersion = server?.latest_version || server?.version || '1.0.0';
    const parts = currentVersion.split('.');
    const minor = parseInt(parts[1] || '0', 10);
    const suggestedVersion = `${parts[0]}.${minor + 1}.0`;
    setPublishVersion(suggestedVersion);
    setPublishNotes('');
    setPublishPublic(server?.is_public || false);
    setShowPublishModal(true);
  };

  const handlePublish = async () => {
    if (!publishVersion.trim()) {
      toast.error('Version is required');
      return;
    }

    try {
      setPublishing(true);
      await publishServer(id!, {
        version: publishVersion,
        release_notes: publishNotes,
        is_public: publishPublic,
      });
      toast.success(`Published version ${publishVersion}`);
      setShowPublishModal(false);
      loadServer();
    } catch (error: unknown) {
      const err = error as { response?: { data?: { error?: string } } };
      toast.error(err.response?.data?.error || 'Failed to publish');
    } finally {
      setPublishing(false);
    }
  };

  const handleDownloadVersion = async (version: string) => {
    try {
      const blob = await downloadServerVersion(id!, version);
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `${server ? serverSlug(server.name) : 'mcp-server'}-v${version}-mcp-server.zip`;
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      document.body.removeChild(a);
      toast.success(`Downloaded version ${version}`);
    } catch (error) {
      toast.error('Failed to download version');
    }
  };

  const openGitHubModal = () => {
    // Pre-fill repo name from server name
    const repoName = server?.name.toLowerCase().replace(/\s+/g, '-').replace(/[^a-z0-9-]/g, '') + '-mcp-server';
    setGithubRepo(repoName);
    setGithubCommitMsg(`Initial MCP server export: ${server?.name}`);
    setShowGitHubModal(true);
  };

  const handleGitHubExport = async () => {
    if (!githubToken || !githubOwner || !githubRepo) {
      toast.error('Please fill in all required fields');
      return;
    }

    try {
      setGithubExporting(true);
      const result = await githubExport(id!, {
        token: githubToken,
        owner: githubOwner,
        repo: githubRepo,
        branch: githubBranch,
        commit_message: githubCommitMsg,
        create_repo: githubCreateRepo,
        private: githubPrivate,
        description: server?.description || `MCP Server: ${server?.name}`,
      });

      toast.success(result.message);
      setShowGitHubModal(false);
      
      // Open the repo in a new tab
      window.open(result.repo_url, '_blank');
    } catch (error: unknown) {
      const err = error as { response?: { data?: { error?: string } } };
      toast.error(err.response?.data?.error || 'Failed to export to GitHub');
    } finally {
      setGithubExporting(false);
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

  const handleDeleteContextConfig = async (configId: string) => {
    if (!confirm('Delete this context configuration?')) return;
    try {
      await deleteContextConfig(configId);
      toast.success('Context configuration deleted');
      loadServer();
    } catch (error) {
      toast.error('Failed to delete context configuration');
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
    { id: 'environments', label: 'Environments', icon: 'bi-cloud' },
    { id: 'tools', label: 'Tools', icon: 'bi-tools' },
    { id: 'resources', label: 'Resources', icon: 'bi-folder' },
    { id: 'prompts', label: 'Prompts', icon: 'bi-chat-text' },
    { id: 'context', label: 'Context', icon: 'bi-person-badge' },
    { id: 'policies', label: 'Policies', icon: 'bi-shield-check' },
    { id: 'security', label: 'Security', icon: 'bi-shield-lock' },
    { id: 'testing', label: 'Testing', icon: 'bi-play-circle' },
    { id: 'healing', label: 'Healing', icon: 'bi-bandaid' },
    { id: 'observability', label: 'Observability', icon: 'bi-graph-up' },
    { id: 'versions', label: 'Versions', icon: 'bi-clock-history' },
  ];

  return (
    <div>
      <div className="page-header">
        <div>
          <nav style={{ marginBottom: '0.5rem' }}>
            <Link to="/" style={{ color: 'var(--text-muted)', textDecoration: 'none', fontSize: '0.875rem' }}>
              Dashboard
            </Link>
            <span style={{ color: 'var(--text-muted)', margin: '0 0.5rem' }}>/</span>
            <span style={{ color: 'var(--text-primary)', fontSize: '0.875rem' }}>{server.name}</span>
          </nav>
          <div style={{ display: 'flex', alignItems: 'center', gap: '0.75rem' }}>
            <h1 className="page-title">{server.name}</h1>
            <span className="badge badge-primary">v{server.latest_version || server.version}</span>
            <span className={`status-badge ${server.status || 'draft'}`}>
              <i className={`bi ${server.status === 'published' ? 'bi-check-circle-fill' : server.status === 'archived' ? 'bi-archive-fill' : 'bi-pencil-fill'}`}></i>
              {server.status === 'published' ? 'Published' : server.status === 'archived' ? 'Archived' : 'Draft'}
            </span>
            {server.is_public && (
              <span className="badge" style={{ background: '#dcfce7', color: '#15803d' }}>
                <i className="bi bi-globe" style={{ marginRight: '4px' }}></i>
                Public
              </span>
            )}
          </div>
          <p className="page-subtitle">
            {server.description || 'No description'}
            {server.downloads > 0 && (
              <span style={{ marginLeft: '0.75rem', color: 'var(--text-muted)' }}>
                <i className="bi bi-download" style={{ marginRight: '4px' }}></i>
                {server.downloads} downloads
              </span>
            )}
          </p>
        </div>
        <div style={{ display: 'flex', gap: '0.75rem' }}>
          <button 
            className="btn btn-primary" 
            onClick={openPublishModal}
          >
            <i className="bi bi-upload"></i>
            Publish
          </button>
          <button 
            className="btn btn-success" 
            onClick={() => setShowDeployModal(true)}
          >
            <i className="bi bi-rocket-takeoff"></i>
            Deploy
          </button>
        </div>
      </div>

      <div style={{ display: 'grid', gridTemplateColumns: '260px minmax(0,1fr)', gap: '1.5rem', alignItems: 'flex-start' }}>
        <div className="card" style={{ padding: '1rem', position: 'sticky', top: 0 }}>
          <h3 className="card-title" style={{ marginBottom: '0.75rem', fontSize: '0.95rem' }}>
            <i className="bi bi-diagram-3" style={{ marginRight: '0.5rem', color: 'var(--primary-color)' }}></i>
            Server navigation
          </h3>
          <p style={{ fontSize: '0.8rem', color: 'var(--text-secondary)', marginBottom: '0.75rem' }}>
            Focused view for <strong>{server.name}</strong>. Pick a section to configure this MCP server.
          </p>
          <div style={{ borderTop: '1px solid var(--card-border)', margin: '0.5rem 0 0.75rem 0' }} />
          <div style={{ display: 'flex', flexDirection: 'column', gap: '0.25rem' }}>
            {tabs.map((tab) => {
              const isTreeSection = tab.id === 'tools' || tab.id === 'resources' || tab.id === 'prompts';
              const isActive = activeTab === tab.id;
              const count =
                tab.id === 'tools'
                  ? server.tools?.length || 0
                  : tab.id === 'resources'
                  ? server.resources?.length || 0
                  : tab.id === 'prompts'
                  ? server.prompts?.length || 0
                  : 0;
              const expanded =
                tab.id === 'tools'
                  ? showToolsTree
                  : tab.id === 'resources'
                  ? showResourcesTree
                  : tab.id === 'prompts'
                  ? showPromptsTree
                  : false;

              const toggleExpanded = () => {
                if (tab.id === 'tools') setShowToolsTree(!showToolsTree);
                if (tab.id === 'resources') setShowResourcesTree(!showResourcesTree);
                if (tab.id === 'prompts') setShowPromptsTree(!showPromptsTree);
              };

              return (
                <div key={tab.id}>
                  <button
                    type="button"
                    onClick={() => setActiveTab(tab.id)}
                    style={{
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'space-between',
                      width: '100%',
                      borderRadius: '8px',
                      border: 'none',
                      padding: '0.45rem 0.6rem',
                      background: isActive ? 'var(--primary-light)' : 'transparent',
                      color: 'var(--text-primary)',
                      cursor: 'pointer',
                      fontSize: '0.82rem',
                      textAlign: 'left',
                      transition: 'background 0.15s',
                    }}
                  >
                    <span style={{ display: 'inline-flex', alignItems: 'center', gap: '0.45rem' }}>
                      {isTreeSection && count > 0 && (
                        <i
                          className={`bi ${expanded ? 'bi-caret-down-fill' : 'bi-caret-right-fill'}`}
                          style={{ fontSize: '0.6rem' }}
                          onClick={(e) => {
                            e.stopPropagation();
                            toggleExpanded();
                          }}
                        ></i>
                      )}
                      {!isTreeSection && (
                        <i
                          className={`bi ${tab.icon}`}
                          style={{ color: isActive ? 'var(--primary-color)' : 'var(--text-secondary)' }}
                        ></i>
                      )}
                      {isTreeSection && (
                        <i
                          className={`bi ${tab.icon}`}
                          style={{ color: isActive ? 'var(--primary-color)' : 'var(--text-secondary)' }}
                        ></i>
                      )}
                      {tab.label}
                    </span>
                    {isTreeSection && count > 0 && (
                      <span className="badge badge-primary">
                        {count}
                      </span>
                    )}
                  </button>

                  {isTreeSection && expanded && count > 0 && (
                    <div style={{ marginTop: '0.1rem', marginLeft: '1.5rem' }}>
                      {tab.id === 'tools' &&
                        server.tools?.map((tool) => (
                          <button
                            key={tool.id}
                            type="button"
                            onClick={() => {
                              setFocusedToolId(tool.id);
                              setActiveTab('tools');
                            }}
                            style={{
                              display: 'block',
                              width: '100%',
                              textAlign: 'left',
                              border: 'none',
                              background: focusedToolId === tool.id ? 'var(--primary-light)' : 'transparent',
                              padding: '0.15rem 0.25rem',
                              borderRadius: '4px',
                              fontSize: '0.78rem',
                              color: focusedToolId === tool.id ? 'var(--primary-color)' : 'var(--text-secondary)',
                              cursor: 'pointer',
                            }}
                          >
                            {tool.name}
                          </button>
                        ))}
                      {tab.id === 'resources' &&
                        server.resources?.map((r) => (
                          <button
                            key={r.id}
                            type="button"
                            onClick={() => {
                              setSelectedResourceId(r.id);
                              setActiveTab('resources');
                            }}
                            style={{
                              display: 'block',
                              width: '100%',
                              textAlign: 'left',
                              border: 'none',
                              background: selectedResourceId === r.id ? 'var(--primary-light)' : 'transparent',
                              padding: '0.15rem 0.25rem',
                              borderRadius: '4px',
                              fontSize: '0.78rem',
                              color: selectedResourceId === r.id ? 'var(--primary-color)' : 'var(--text-secondary)',
                              cursor: 'pointer',
                            }}
                          >
                            {r.name}
                          </button>
                        ))}
                      {tab.id === 'prompts' &&
                        server.prompts?.map((p) => (
                          <button
                            key={p.id}
                            type="button"
                            onClick={() => {
                              setSelectedPromptId(p.id);
                              setActiveTab('prompts');
                            }}
                            style={{
                              display: 'block',
                              width: '100%',
                              textAlign: 'left',
                              border: 'none',
                              background: selectedPromptId === p.id ? 'var(--primary-light)' : 'transparent',
                              padding: '0.15rem 0.25rem',
                              borderRadius: '4px',
                              fontSize: '0.78rem',
                              color: selectedPromptId === p.id ? 'var(--primary-color)' : 'var(--text-secondary)',
                              cursor: 'pointer',
                            }}
                          >
                            {p.name}
                          </button>
                        ))}
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        </div>

        <div>
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

              <div className="form-group">
                <label className="form-label">Server Icon</label>
                <div style={{ 
                  display: 'grid', 
                  gridTemplateColumns: 'repeat(12, 1fr)', 
                  gap: '0.375rem',
                  padding: '0.75rem',
                  background: 'var(--dark-bg)',
                  borderRadius: '8px'
                }}>
                  {[
                    'bi-server', 'bi-cloud', 'bi-database', 'bi-globe', 'bi-robot', 'bi-cpu',
                    'bi-terminal', 'bi-code-slash', 'bi-gear', 'bi-lightning', 'bi-shield-check', 'bi-graph-up',
                    'bi-chat-dots', 'bi-envelope', 'bi-calendar', 'bi-file-text', 'bi-currency-dollar', 'bi-cart',
                    'bi-person', 'bi-building', 'bi-box', 'bi-palette', 'bi-music-note', 'bi-camera'
                  ].map((iconClass) => (
                    <button
                      key={iconClass}
                      type="button"
                      onClick={() => setEditIcon(iconClass)}
                      style={{
                        width: '36px',
                        height: '36px',
                        borderRadius: '6px',
                        border: editIcon === iconClass ? '2px solid var(--primary-color)' : '2px solid transparent',
                        background: editIcon === iconClass ? 'var(--primary-light)' : 'var(--card-bg)',
                        cursor: 'pointer',
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                        fontSize: '1rem',
                        color: editIcon === iconClass ? 'var(--primary-color)' : 'var(--text-secondary)',
                        transition: 'all 0.15s'
                      }}
                    >
                      <i className={`bi ${iconClass}`}></i>
                    </button>
                  ))}
                </div>
              </div>

              <button className="btn btn-primary" onClick={handleUpdateServer}>
                <i className="bi bi-check-lg"></i>
                Save Changes
              </button>
            </div>
          )}

          {activeTab === 'environments' && (
            <div className="card">
              <h3 className="card-title" style={{ marginBottom: '0.5rem' }}>
                <i className="bi bi-cloud" style={{ marginRight: '0.5rem', color: 'var(--primary-color)' }}></i>
                Environment profiles
              </h3>
              <p style={{ color: 'var(--text-secondary)', fontSize: '0.875rem', marginBottom: '1rem', marginTop: 0 }}>
                Set base URLs and database URLs for Dev, Staging, and Prod. The <strong>Testing</strong> tab uses the selected profile when running tools; <strong>Deploy</strong> uses this configuration when generating the server.
              </p>
              {envProfilesLoading ? (
                <p style={{ color: 'var(--text-muted)' }}>Loading…</p>
              ) : (
                <>
                  {(['dev', 'staging', 'prod'] as EnvProfileKey[]).map((key) => {
                    const p = (envProfiles[key] ?? {}) as EnvProfile;
                    return (
                      <div key={key} style={{ marginBottom: '1rem', padding: '1rem', background: 'var(--dark-bg)', borderRadius: '8px', border: '1px solid var(--card-border)' }}>
                        <div style={{ fontWeight: 600, textTransform: 'capitalize', marginBottom: '0.75rem', color: 'var(--text-primary)' }}>{key}</div>
                        <div className="form-group" style={{ marginBottom: '0.5rem' }}>
                          <label className="form-label" style={{ fontSize: '0.8125rem' }}>Base URL (REST / GraphQL / Webhook)</label>
                          <input
                            type="text"
                            className="form-control"
                            placeholder={key === 'dev' ? 'https://dev-api.example.com' : key === 'staging' ? 'https://staging-api.example.com' : 'https://api.example.com'}
                            value={p.base_url ?? ''}
                            onChange={(e) => setEnvProfileField(key, 'base_url', e.target.value)}
                          />
                        </div>
                        <div className="form-group" style={{ marginBottom: 0 }}>
                          <label className="form-label" style={{ fontSize: '0.8125rem' }}>Database URL (database tools)</label>
                          <input
                            type="text"
                            className="form-control"
                            placeholder="postgres://user:pass@host:5432/db"
                            value={p.db_url ?? ''}
                            onChange={(e) => setEnvProfileField(key, 'db_url', e.target.value)}
                          />
                        </div>
                      </div>
                    );
                  })}
                  <button type="button" className="btn btn-primary btn-sm" onClick={handleSaveEnvProfiles} disabled={envProfilesSaving}>
                    {envProfilesSaving ? 'Saving…' : 'Save environment profiles'}
                  </button>
                </>
              )}
            </div>
          )}

          {activeTab === 'tools' && (
            <div>
              {!focusedToolId && (
                <div style={{
                  marginBottom: '1rem',
                  padding: '1rem',
                  background: 'linear-gradient(135deg, rgba(129, 140, 248, 0.15), rgba(56, 189, 248, 0.1))',
                  borderRadius: '12px',
                  border: '1px solid rgba(129, 140, 248, 0.3)',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'space-between'
                }}>
                  <div>
                    <h4 style={{ margin: 0, color: 'var(--text-primary)', fontSize: '0.9375rem' }}>
                      <i className="bi bi-diagram-3" style={{ marginRight: '0.5rem', color: 'var(--primary-color)' }}></i>
                      Visual Builder (pipelines only)
                    </h4>
                    <p style={{ margin: '0.25rem 0 0 0', color: 'var(--text-secondary)', fontSize: '0.8125rem' }}>
                      Build multi-tool pipelines with drag-and-drop — not for editing a single tool.
                    </p>
                  </div>
                  <button 
                    className="btn btn-primary btn-sm"
                    onClick={() => navigate(`/servers/${id}/flow`)}
                  >
                    <i className="bi bi-box-arrow-up-right"></i>
                    Open Visual Builder
                  </button>
                </div>
              )}
              <ToolEditor
                serverId={id!}
                tools={server.tools || []}
                focusToolId={focusedToolId}
                onCloseEdit={() => setFocusedToolId(null)}
                onNavigateToSection={(section: ToolSection, toolId: string) => {
                  setPreselectedToolId(toolId);
                  setActiveTab(section);
                }}
                onToolCreated={loadServer}
                onToolDeleted={handleDeleteTool}
              />
            </div>
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
              tools={server.tools || []}
              onPromptCreated={loadServer}
              onPromptDeleted={handleDeletePrompt}
            />
          )}

          {activeTab === 'context' && (
            <ContextConfigEditor
              serverId={id!}
              configs={contextConfigs}
              onConfigCreated={loadServer}
              onConfigDeleted={handleDeleteContextConfig}
            />
          )}

          {activeTab === 'policies' && (
            <PolicyEditor
              tools={server.tools || []}
              initialToolId={preselectedToolId ?? undefined}
              onPolicyUpdated={loadServer}
            />
          )}

          {activeTab === 'security' && (
            <SecurityScorePanel serverId={id!} />
          )}

          {activeTab === 'testing' && (
            <TestPlayground
              serverId={id!}
              tools={server.tools || []}
              initialToolId={preselectedToolId ?? undefined}
              onOpenEnvironments={() => setActiveTab('environments')}
            />
          )}

          {activeTab === 'healing' && (
            <HealingDashboard
              tools={server.tools || []}
              initialToolId={preselectedToolId ?? undefined}
            />
          )}

          {activeTab === 'observability' && (
            <ObservabilityPanel serverId={server.id} />
          )}

          {activeTab === 'versions' && (
            <div className="card">
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '1.5rem' }}>
                <div>
                  <h3 className="card-title" style={{ marginBottom: '0.25rem' }}>
                    <i className="bi bi-clock-history" style={{ marginRight: '0.75rem' }}></i>
                    Version History
                  </h3>
                  <p style={{ color: 'var(--text-secondary)', margin: 0 }}>
                    Published versions of this server
                  </p>
                </div>
                <button className="btn btn-primary" onClick={openPublishModal}>
                  <i className="bi bi-plus-lg"></i>
                  Publish New Version
                </button>
              </div>

              {versions.length === 0 ? (
                <div style={{ 
                  textAlign: 'center', 
                  padding: '3rem 1rem', 
                  background: 'var(--background-secondary)',
                  borderRadius: '8px',
                }}>
                  <i className="bi bi-archive" style={{ fontSize: '2.5rem', color: 'var(--text-muted)', marginBottom: '1rem', display: 'block' }}></i>
                  <h4 style={{ marginBottom: '0.5rem' }}>No Published Versions</h4>
                  <p style={{ color: 'var(--text-muted)', marginBottom: '1rem' }}>
                    Publish your first version to create a snapshot that can be downloaded from the marketplace.
                  </p>
                  <button className="btn btn-primary" onClick={openPublishModal}>
                    <i className="bi bi-upload"></i>
                    Publish First Version
                  </button>
                </div>
              ) : (
                <div className="version-list">
                  {versions.map((version) => (
                    <div key={version.id} className="version-item">
                      <div className="version-info">
                        <div style={{ display: 'flex', alignItems: 'center', gap: '0.75rem' }}>
                          <span className="version-tag">v{version.version}</span>
                          {version.version === server.latest_version && (
                            <span className="badge badge-success" style={{ fontSize: '0.7rem' }}>Latest</span>
                          )}
                        </div>
                        {version.release_notes && (
                          <p className="version-notes">{version.release_notes}</p>
                        )}
                        <span className="version-date">
                          Published {new Date(version.published_at).toLocaleDateString('en-US', {
                            year: 'numeric',
                            month: 'long',
                            day: 'numeric',
                          })}
                        </span>
                      </div>
                      <div className="version-actions">
                        <button 
                          className="btn btn-secondary btn-sm"
                          onClick={() => handleDownloadVersion(version.version)}
                          data-tooltip="Download this version"
                        >
                          <i className="bi bi-download"></i>
                        </button>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          )}

      {/* Deploy Modal */}
      {showDeployModal && (
        <div
          style={{
            position: 'fixed',
            top: 0,
            left: 0,
            right: 0,
            bottom: 0,
            background: 'rgba(0, 0, 0, 0.5)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            zIndex: 1000,
          }}
          onClick={() => setShowDeployModal(false)}
        >
          <div
            style={{
              background: 'var(--card-bg)',
              borderRadius: '12px',
              width: '100%',
              maxWidth: '80%',
              maxHeight: '90vh',
              overflow: 'auto',
              boxShadow: '0 20px 40px rgba(0, 0, 0, 0.2)',
              position: 'relative',
            }}
            onClick={(e) => e.stopPropagation()}
          >
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '1rem 1.25rem', borderBottom: '1px solid var(--card-border)' }}>
              <h2 style={{ margin: 0, fontSize: '1.25rem' }}>
                <i className="bi bi-rocket-takeoff" style={{ marginRight: '0.5rem' }}></i>
                Deploy Your Server
              </h2>
              <button type="button" className="btn btn-sm btn-outline-secondary" onClick={() => setShowDeployModal(false)} aria-label="Close">
                <i className="bi bi-x-lg"></i>
              </button>
            </div>
            <div style={{ padding: '1.25rem' }}>
          {/* Step 1: Target environment + deployment type */}
          <div className="card" style={{ marginBottom: '1.5rem' }}>
            <p style={{ color: 'var(--text-secondary)', marginBottom: '1rem', marginTop: 0 }}>
              Choose how you want to deploy your MCP server. Environment profiles (Dev / Staging / Prod) are configured in the <strong>Environments</strong> tab in the left menu.
            </p>
            <div className="form-group" style={{ marginBottom: '1.5rem' }}>
              <label className="form-label" style={{ fontWeight: 600 }}>
                <i className="bi bi-cloud" style={{ marginRight: '0.5rem', color: 'var(--primary-color)' }}></i>
                Target environment
              </label>
              <select
                className="form-control w-100"
                value={deployTargetEnv}
                onChange={(e) => setDeployTargetEnv(e.target.value as EnvProfileKey | '')}
              >
                <option value="">Use .env at runtime (no profile baked in)</option>
                <option value="dev">Dev</option>
                <option value="staging">Staging</option>
                <option value="prod">Prod</option>
              </select>
              <p style={{ fontSize: '0.75rem', color: 'var(--text-muted)', marginTop: '0.25rem', marginBottom: 0 }}>
                When you generate or download, the server will be built with this environment’s base URL and database URL. Choose “Use .env at runtime” to keep URLs in .env instead.
              </p>
            </div>

            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(5, 1fr)', gap: '1rem' }}>
              <button
                onClick={() => setSelectedDeploy('nodejs')}
                style={{
                  padding: '1.5rem',
                  background: selectedDeploy === 'nodejs' ? 'var(--primary-light)' : 'var(--dark-bg)',
                  border: `2px solid ${selectedDeploy === 'nodejs' ? 'var(--primary-color)' : 'var(--card-border)'}`,
                  borderRadius: '12px',
                  cursor: 'pointer',
                  textAlign: 'center',
                  transition: 'all 0.2s',
                }}
              >
                <div style={{ fontSize: '2.5rem', marginBottom: '0.75rem', color: selectedDeploy === 'nodejs' ? 'var(--primary-color)' : 'var(--text-secondary)' }}>
                  <i className="bi bi-filetype-js"></i>
                </div>
                <h4 style={{ marginBottom: '0.25rem', color: 'var(--text-primary)', fontSize: '1rem' }}>Node.js</h4>
                <p style={{ color: 'var(--text-muted)', fontSize: '0.8125rem', margin: 0 }}>
                  Download & run locally
                </p>
              </button>

              <button
                onClick={() => setSelectedDeploy('docker')}
                style={{
                  padding: '1.5rem',
                  background: selectedDeploy === 'docker' ? 'rgba(16, 185, 129, 0.1)' : 'var(--dark-bg)',
                  border: `2px solid ${selectedDeploy === 'docker' ? 'var(--success-color)' : 'var(--card-border)'}`,
                  borderRadius: '12px',
                  cursor: 'pointer',
                  textAlign: 'center',
                  transition: 'all 0.2s',
                }}
              >
                <div style={{ fontSize: '2.5rem', marginBottom: '0.75rem', color: selectedDeploy === 'docker' ? 'var(--success-color)' : 'var(--text-secondary)' }}>
                  <i className="bi bi-box-seam"></i>
                </div>
                <h4 style={{ marginBottom: '0.25rem', color: 'var(--text-primary)', fontSize: '1rem' }}>Docker</h4>
                <p style={{ color: 'var(--text-muted)', fontSize: '0.8125rem', margin: 0 }}>
                  Containerized deployment
                </p>
              </button>

              <button
                onClick={() => setSelectedDeploy('github')}
                style={{
                  padding: '1.5rem',
                  background: selectedDeploy === 'github' ? 'rgba(36, 41, 47, 0.1)' : 'var(--dark-bg)',
                  border: `2px solid ${selectedDeploy === 'github' ? '#24292f' : 'var(--card-border)'}`,
                  borderRadius: '12px',
                  cursor: 'pointer',
                  textAlign: 'center',
                  transition: 'all 0.2s',
                }}
              >
                <div style={{ fontSize: '2.5rem', marginBottom: '0.75rem', color: selectedDeploy === 'github' ? '#24292f' : 'var(--text-secondary)' }}>
                  <i className="bi bi-github"></i>
                </div>
                <h4 style={{ marginBottom: '0.25rem', color: 'var(--text-primary)', fontSize: '1rem' }}>GitHub</h4>
                <p style={{ color: 'var(--text-muted)', fontSize: '0.8125rem', margin: 0 }}>
                  Push to repository
                </p>
              </button>

              <button
                onClick={() => setSelectedDeploy('azure')}
                style={{
                  padding: '1.5rem',
                  background: selectedDeploy === 'azure' ? 'rgba(0, 120, 212, 0.1)' : 'var(--dark-bg)',
                  border: `2px solid ${selectedDeploy === 'azure' ? '#0078d4' : 'var(--card-border)'}`,
                  borderRadius: '12px',
                  cursor: 'pointer',
                  textAlign: 'center',
                  transition: 'all 0.2s',
                }}
              >
                <div style={{ fontSize: '2.5rem', marginBottom: '0.75rem', color: selectedDeploy === 'azure' ? '#0078d4' : 'var(--text-secondary)' }}>
                  <i className="bi bi-cloud-upload"></i>
                </div>
                <h4 style={{ marginBottom: '0.25rem', color: 'var(--text-primary)', fontSize: '1rem' }}>Deploy to Cloud</h4>
                <p style={{ color: 'var(--text-muted)', fontSize: '0.8125rem', margin: 0 }}>
                  Deploy to cloud
                </p>
              </button>

              <button
                onClick={() => { setSelectedDeploy('hosted'); }}
                style={{
                  padding: '1.5rem',
                  background: selectedDeploy === 'hosted' ? 'rgba(139, 92, 246, 0.1)' : 'var(--dark-bg)',
                  border: `2px solid ${selectedDeploy === 'hosted' ? '#8b5cf6' : 'var(--card-border)'}`,
                  borderRadius: '12px',
                  cursor: 'pointer',
                  textAlign: 'center',
                  transition: 'all 0.2s',
                }}
              >
                <div style={{ fontSize: '2.5rem', marginBottom: '0.75rem', color: selectedDeploy === 'hosted' ? '#8b5cf6' : 'var(--text-secondary)' }}>
                  <i className="bi bi-globe"></i>
                </div>
                <h4 style={{ marginBottom: '0.25rem', color: 'var(--text-primary)', fontSize: '1rem' }}>Publish MCP</h4>
                <p style={{ color: 'var(--text-muted)', fontSize: '0.8125rem', margin: 0 }}>
                  Hosted at URL
                </p>
              </button>
            </div>
          </div>

          {/* Step 2: Show instructions based on selection */}
          {selectedDeploy === 'nodejs' && (
            <div className="card">
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: '1.5rem' }}>
                <div>
                  <h3 className="card-title" style={{ marginBottom: '0.25rem' }}>
                    <i className="bi bi-filetype-js" style={{ marginRight: '0.75rem', color: 'var(--primary-color)' }}></i>
                    Node.js Deployment
                  </h3>
                  <p style={{ color: 'var(--text-secondary)', margin: 0 }}>
                    Download and run your server with Node.js
                  </p>
                </div>
                <button className="btn btn-primary" onClick={handleGenerate} disabled={generating}>
                  <i className="bi bi-download"></i>
                  {generating ? 'Generating...' : 'Download ZIP'}
                </button>
              </div>

              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '1.5rem' }}>
                <div>
                  <h4 style={{ fontSize: '0.875rem', fontWeight: 600, marginBottom: '0.75rem', color: 'var(--text-primary)' }}>
                    <span style={{ 
                      display: 'inline-flex', 
                      alignItems: 'center', 
                      justifyContent: 'center',
                      width: '24px', 
                      height: '24px', 
                      background: 'var(--primary-color)', 
                      color: 'white', 
                      borderRadius: '50%', 
                      fontSize: '0.75rem',
                      marginRight: '0.5rem'
                    }}>1</span>
                    Setup & Run
                  </h4>
                  <pre style={{ background: '#1a1a2e', padding: '1rem', borderRadius: '8px', fontSize: '0.8125rem', color: '#e5e7eb', margin: 0 }}>
{`cd ${serverSlug(server.name)}-mcp-server
npm install
npm run build
npm start`}
                  </pre>
                </div>
                
                <div>
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '0.5rem' }}>
                    <h4 style={{ fontSize: '0.875rem', fontWeight: 600, margin: 0, color: 'var(--text-primary)' }}>
                      <span style={{ 
                        display: 'inline-flex', 
                        alignItems: 'center', 
                        justifyContent: 'center',
                        width: '24px', 
                        height: '24px', 
                        background: 'var(--primary-color)', 
                        color: 'white', 
                        borderRadius: '50%', 
                        fontSize: '0.75rem',
                        marginRight: '0.5rem'
                      }}>2</span>
                      Configure MCP Client
                    </h4>
                    <button
                      type="button"
                      className="btn btn-sm btn-outline-primary"
                      onClick={() => {
                        const slug = serverSlug(server.name);
                        const serverEntry: Record<string, unknown> = {
                          command: 'node',
                          args: ['/path/to/your-server/run-with-log.mjs'],
                        };
                        if (server.observability_reporting_key) {
                          serverEntry.env = {
                            MCP_OBSERVABILITY_ENDPOINT: `${window.location.origin}/api/observability/events`,
                            MCP_OBSERVABILITY_KEY: server.observability_reporting_key,
                            MCP_OBSERVABILITY_USER_ID: '',
                            MCP_OBSERVABILITY_CLIENT_AGENT: 'Cursor',
                            MCP_OBSERVABILITY_USER_TOKEN: '',
                          };
                        }
                        const config = JSON.stringify({ mcpServers: { [slug]: serverEntry } }, null, 2);
                        navigator.clipboard.writeText(config).then(
                          () => toast.success('MCP config copied to clipboard'),
                          () => toast.error('Could not copy')
                        );
                      }}
                    >
                      <i className="bi bi-clipboard"></i> Copy
                    </button>
                  </div>
                  <p style={{ fontSize: '0.8125rem', color: 'var(--text-secondary)', marginBottom: '0.5rem' }}>
                    Add to your Cursor or Claude Desktop <code>mcp.json</code> (replace path with your server folder):
                  </p>
                  <pre style={{ background: '#1a1a2e', padding: '1rem', borderRadius: '8px', fontSize: '0.8125rem', color: '#fde68a', margin: 0 }}>
                    {(() => {
                      const slug = serverSlug(server.name);
                      const serverEntry: Record<string, unknown> = {
                        command: 'node',
                        args: ['/path/to/your-server/run-with-log.mjs'],
                      };
                      if (server.observability_reporting_key) {
                        serverEntry.env = {
                          MCP_OBSERVABILITY_ENDPOINT: `${window.location.origin}/api/observability/events`,
                          MCP_OBSERVABILITY_KEY: server.observability_reporting_key,
                          MCP_OBSERVABILITY_USER_ID: '',
                          MCP_OBSERVABILITY_CLIENT_AGENT: 'Cursor',
                          MCP_OBSERVABILITY_USER_TOKEN: '',
                        };
                      }
                      return JSON.stringify({ mcpServers: { [slug]: serverEntry } }, null, 2);
                    })()}
                  </pre>
                </div>
              </div>
            </div>
          )}

          {selectedDeploy === 'docker' && (
            <div className="card">
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: '1.5rem' }}>
                <div>
                  <h3 className="card-title" style={{ marginBottom: '0.25rem' }}>
                    <i className="bi bi-box-seam" style={{ marginRight: '0.75rem', color: 'var(--success-color)' }}></i>
                    Docker Deployment
                  </h3>
                  <p style={{ color: 'var(--text-secondary)', margin: 0 }}>
                    Run in a container with Docker or Docker Compose
                  </p>
                </div>
                <button className="btn btn-success" onClick={handleGenerate} disabled={generating}>
                  <i className="bi bi-download"></i>
                  {generating ? 'Generating...' : 'Download ZIP'}
                </button>
              </div>

              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '1.5rem' }}>
                <div>
                  <h4 style={{ fontSize: '0.875rem', fontWeight: 600, marginBottom: '0.75rem', color: 'var(--text-primary)' }}>
                    <span style={{ 
                      display: 'inline-flex', 
                      alignItems: 'center', 
                      justifyContent: 'center',
                      width: '24px', 
                      height: '24px', 
                      background: 'var(--success-color)', 
                      color: 'white', 
                      borderRadius: '50%', 
                      fontSize: '0.75rem',
                      marginRight: '0.5rem'
                    }}>1</span>
                    Docker Compose (Recommended)
                  </h4>
                  <pre style={{ background: '#1a1a2e', padding: '1rem', borderRadius: '8px', fontSize: '0.8125rem', color: '#e5e7eb', margin: 0 }}>
{`cd ${serverSlug(server.name)}-mcp-server

# Copy environment template
cp .env.example .env

# Edit .env with your API keys
nano .env

# Start the container
docker-compose up -d`}
                  </pre>
                </div>
                
                <div>
                  <h4 style={{ fontSize: '0.875rem', fontWeight: 600, marginBottom: '0.75rem', color: 'var(--text-primary)' }}>
                    <span style={{ 
                      display: 'inline-flex', 
                      alignItems: 'center', 
                      justifyContent: 'center',
                      width: '24px', 
                      height: '24px', 
                      background: 'var(--success-color)', 
                      color: 'white', 
                      borderRadius: '50%', 
                      fontSize: '0.75rem',
                      marginRight: '0.5rem'
                    }}>2</span>
                    Build & Run Directly
                  </h4>
                  <pre style={{ background: '#1a1a2e', padding: '1rem', borderRadius: '8px', fontSize: '0.8125rem', color: '#e5e7eb', margin: 0 }}>
{`cd ${serverSlug(server.name)}-mcp-server

# Build the image
docker build -t ${serverSlug(server.name)}-mcp .

# Run the container
docker run -it --rm \\
  -e API_KEY=your_key \\
  ${serverSlug(server.name)}-mcp`}
                  </pre>
                </div>
              </div>

              <div style={{ marginTop: '1.5rem', padding: '1rem', background: 'var(--hover-bg)', borderRadius: '8px' }}>
                <h4 style={{ fontSize: '0.875rem', fontWeight: 600, marginBottom: '0.5rem', color: 'var(--text-primary)' }}>
                  <i className="bi bi-info-circle" style={{ marginRight: '0.5rem', color: 'var(--secondary-color)' }}></i>
                  What's Included
                </h4>
                <div style={{ display: 'flex', gap: '2rem', fontSize: '0.8125rem', color: 'var(--text-secondary)' }}>
                  <span><i className="bi bi-check2" style={{ marginRight: '0.375rem', color: 'var(--success-color)' }}></i>Dockerfile</span>
                  <span><i className="bi bi-check2" style={{ marginRight: '0.375rem', color: 'var(--success-color)' }}></i>docker-compose.yml</span>
                  <span><i className="bi bi-check2" style={{ marginRight: '0.375rem', color: 'var(--success-color)' }}></i>.env.example</span>
                  <span><i className="bi bi-check2" style={{ marginRight: '0.375rem', color: 'var(--success-color)' }}></i>.dockerignore</span>
                </div>
              </div>

              <div style={{ marginTop: '1.5rem' }}>
                <h4 style={{ fontSize: '0.875rem', fontWeight: 600, marginBottom: '0.75rem', color: 'var(--text-primary)' }}>
                  <span style={{ 
                    display: 'inline-flex', 
                    alignItems: 'center', 
                    justifyContent: 'center',
                    width: '24px', 
                    height: '24px', 
                    background: 'var(--success-color)', 
                    color: 'white', 
                    borderRadius: '50%', 
                    fontSize: '0.75rem',
                    marginRight: '0.5rem'
                  }}>3</span>
                  Configure MCP Client
                </h4>
                <p style={{ fontSize: '0.8125rem', color: 'var(--text-secondary)', marginBottom: '0.5rem' }}>
                  Add to your Claude Desktop or Cursor config:
                </p>
                <pre style={{ background: '#1a1a2e', padding: '1rem', borderRadius: '8px', fontSize: '0.8125rem', color: '#fde68a', margin: 0 }}>
{`{
  "mcpServers": {
    "${serverSlug(server.name)}": {
      "command": "docker",
      "args": ["run", "-i", "--rm", "${serverSlug(server.name)}-mcp"]
    }
  }
}`}
                </pre>
              </div>
            </div>
          )}

          {selectedDeploy === 'github' && (
            <div className="card">
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: '1.5rem' }}>
                <div>
                  <h3 className="card-title" style={{ marginBottom: '0.25rem' }}>
                    <i className="bi bi-github" style={{ marginRight: '0.75rem' }}></i>
                    GitHub Deployment
                  </h3>
                  <p style={{ color: 'var(--text-secondary)', margin: 0 }}>
                    Push your server code directly to a GitHub repository
                  </p>
                </div>
                <button className="btn btn-primary" onClick={openGitHubModal}>
                  <i className="bi bi-github"></i>
                  Push to GitHub
                </button>
              </div>

              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '1.5rem' }}>
                <div>
                  <h4 style={{ fontSize: '0.875rem', fontWeight: 600, marginBottom: '0.75rem', color: 'var(--text-primary)' }}>
                    What Happens
                  </h4>
                  <ul style={{ margin: 0, paddingLeft: '1.25rem', color: 'var(--text-secondary)', fontSize: '0.875rem', lineHeight: 1.8 }}>
                    <li>Generates complete MCP server code</li>
                    <li>Creates repository (if needed)</li>
                    <li>Pushes all files to your branch</li>
                    <li>Includes README with setup instructions</li>
                  </ul>
                </div>
                
                <div>
                  <h4 style={{ fontSize: '0.875rem', fontWeight: 600, marginBottom: '0.75rem', color: 'var(--text-primary)' }}>
                    Requirements
                  </h4>
                  <ul style={{ margin: 0, paddingLeft: '1.25rem', color: 'var(--text-secondary)', fontSize: '0.875rem', lineHeight: 1.8 }}>
                    <li>GitHub Personal Access Token</li>
                    <li>Token needs <code style={{ background: 'var(--hover-bg)', padding: '0.125rem 0.375rem', borderRadius: '4px' }}>repo</code> scope</li>
                    <li>For new repos: <code style={{ background: 'var(--hover-bg)', padding: '0.125rem 0.375rem', borderRadius: '4px' }}>public_repo</code> or <code style={{ background: 'var(--hover-bg)', padding: '0.125rem 0.375rem', borderRadius: '4px' }}>repo</code></li>
                  </ul>
                </div>
              </div>

              <div style={{ marginTop: '1.5rem', padding: '1rem', background: 'var(--hover-bg)', borderRadius: '8px' }}>
                <h4 style={{ fontSize: '0.875rem', fontWeight: 600, marginBottom: '0.5rem', color: 'var(--text-primary)' }}>
                  <i className="bi bi-lightbulb" style={{ marginRight: '0.5rem', color: 'var(--warning-color)' }}></i>
                  Pro Tip
                </h4>
                <p style={{ margin: 0, fontSize: '0.8125rem', color: 'var(--text-secondary)' }}>
                  After pushing, you can set up GitHub Actions for CI/CD, or deploy directly to services like Railway, Render, or Fly.io.
                </p>
              </div>
            </div>
          )}

          {selectedDeploy === 'azure' && (
            <div className="card">
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: '1.5rem' }}>
                <div>
                  <h3 className="card-title" style={{ marginBottom: '0.25rem' }}>
                    <i className="bi bi-cloud-upload" style={{ marginRight: '0.75rem', color: '#0078d4' }}></i>
                    Deploy to Cloud
                  </h3>
                  <p style={{ color: 'var(--text-secondary)', margin: 0 }}>
                    Build and deploy to cloud via CI/CD pipeline
                  </p>
                </div>
                <span style={{ 
                  padding: '0.375rem 0.75rem',
                  background: 'rgba(245, 158, 11, 0.1)',
                  color: 'var(--warning-color)',
                  borderRadius: '6px',
                  fontSize: '0.75rem',
                  fontWeight: 600
                }}>
                  Coming Soon
                </span>
              </div>

              <div style={{ 
                textAlign: 'center', 
                padding: '3rem', 
                background: 'var(--hover-bg)', 
                borderRadius: '12px',
                border: '2px dashed var(--card-border)'
              }}>
                <i className="bi bi-gear" style={{ fontSize: '3rem', color: 'var(--text-muted)', marginBottom: '1rem', display: 'block' }}></i>
                <h4 style={{ color: 'var(--text-primary)', marginBottom: '0.5rem' }}>Not Yet Implemented</h4>
                <p style={{ color: 'var(--text-muted)', margin: 0, maxWidth: '400px', marginLeft: 'auto', marginRight: 'auto' }}>
                  This feature will enable one-click deployment to cloud with automatic CI/CD pipeline setup.
                </p>
              </div>

              <div style={{ marginTop: '1.5rem', padding: '1rem', background: 'var(--hover-bg)', borderRadius: '8px' }}>
                <h4 style={{ fontSize: '0.875rem', fontWeight: 600, marginBottom: '0.5rem', color: 'var(--text-primary)' }}>
                  <i className="bi bi-lightbulb" style={{ marginRight: '0.5rem', color: 'var(--warning-color)' }}></i>
                  Planned Features
                </h4>
                <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '0.5rem', fontSize: '0.8125rem', color: 'var(--text-secondary)' }}>
                  <span><i className="bi bi-circle" style={{ marginRight: '0.375rem', fontSize: '0.5rem' }}></i>Container registry integration</span>
                  <span><i className="bi bi-circle" style={{ marginRight: '0.375rem', fontSize: '0.5rem' }}></i>Automated pipeline generation</span>
                  <span><i className="bi bi-circle" style={{ marginRight: '0.375rem', fontSize: '0.5rem' }}></i>Kubernetes deployment support</span>
                  <span><i className="bi bi-circle" style={{ marginRight: '0.375rem', fontSize: '0.5rem' }}></i>Environment configuration</span>
                </div>
              </div>
            </div>
          )}

          {selectedDeploy === 'hosted' && (
            <div className="card">
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: '1.5rem' }}>
                <div>
                  <h3 className="card-title" style={{ marginBottom: '0.25rem' }}>
                    <i className="bi bi-globe" style={{ marginRight: '0.75rem', color: '#8b5cf6' }}></i>
                    Publish MCP
                  </h3>
                  <p style={{ color: 'var(--text-secondary)', margin: 0 }}>
                    Publish this server to the platform. It will be available at a URL you can add to your IDE (Cursor, Claude Desktop, etc.).
                  </p>
                </div>
              </div>
              <div style={{ marginBottom: '1rem', padding: '0.75rem', borderRadius: '8px', background: 'var(--hover-bg)' }}>
                {hostedStatusLoading ? (
                  <span style={{ color: 'var(--text-muted)' }}><i className="bi bi-arrow-repeat" style={{ marginRight: '0.5rem' }}></i>Checking hosted runtime status...</span>
                ) : hostedRuntime?.running ? (
                  <span style={{ color: 'var(--success-color)' }}>
                    <i className="bi bi-check-circle" style={{ marginRight: '0.5rem' }}></i>
                    Already running{hostedRuntime.version ? ` (version ${hostedRuntime.version})` : ''}.
                  </span>
                ) : (
                  <span style={{ color: 'var(--text-muted)' }}>
                    <i className="bi bi-info-circle" style={{ marginRight: '0.5rem' }}></i>
                    No hosted runtime currently running.
                  </span>
                )}
              </div>

              {!hostedResult ? (
                <>
                  <div className="form-group" style={{ marginBottom: '1rem' }}>
                    <label className="form-label" style={{ fontWeight: 600 }}>Version (optional)</label>
                    <input
                      type="text"
                      className="form-control"
                      placeholder={server?.latest_version || server?.version || '1.0.0'}
                      value={hostedPublishVersion}
                      onChange={(e) => setHostedPublishVersion(e.target.value)}
                    />
                    <p style={{ fontSize: '0.75rem', color: 'var(--text-muted)', marginTop: '0.25rem', marginBottom: 0 }}>
                      Leave empty to use latest published version or current version. If the version does not exist, it will be published now.
                    </p>
                  </div>
                  <button
                    type="button"
                    className="btn btn-primary"
                    onClick={async () => {
                      if (!id) return;
                      setHostedPublishing(true);
                      try {
                        const v = hostedPublishVersion.trim() || undefined;
                        const result = await hostedPublish(id, v);
                        setHostedResult(result);
                        setHostedRuntime({
                          running: true,
                          user_id: result.user_id,
                          server_slug: result.server_slug,
                          version: result.version,
                          endpoint: result.endpoint,
                        });
                        toast.success('Published. Your MCP server is available at the URL below.');
                      } catch (err: unknown) {
                        const msg = err && typeof err === 'object' && 'response' in err && err.response && typeof (err.response as { data?: { error?: string } }).data?.error === 'string'
                          ? (err.response as { data: { error: string } }).data.error
                          : 'Failed to publish';
                        toast.error(msg);
                      } finally {
                        setHostedPublishing(false);
                      }
                    }}
                    disabled={hostedPublishing}
                  >
                    {hostedPublishing ? (
                      <>
                        <i className="bi bi-hourglass-split"></i> Publishing...
                      </>
                    ) : (
                      <>
                        <i className="bi bi-globe"></i> {hostedRuntime?.running ? 'Re-publish MCP' : 'Publish MCP'}
                      </>
                    )}
                  </button>
                </>
              ) : (
                <div>
                  <div className="form-group" style={{ marginBottom: '1rem' }}>
                    <label className="form-label" style={{ fontWeight: 600 }}>Server URL</label>
                    <div style={{ display: 'flex', gap: '0.5rem', alignItems: 'center' }}>
                      <input
                        type="text"
                        className="form-control"
                        readOnly
                        value={hostedResult.endpoint}
                        style={{ fontFamily: 'monospace', fontSize: '0.875rem' }}
                      />
                      <button
                        type="button"
                        className="btn btn-secondary"
                        onClick={() => { navigator.clipboard.writeText(hostedResult!.endpoint); toast.success('URL copied'); }}
                      >
                        <i className="bi bi-clipboard"></i> Copy
                      </button>
                    </div>
                    <p style={{ fontSize: '0.75rem', color: 'var(--text-muted)', marginTop: '0.25rem', marginBottom: 0 }}>
                      Your app is available at: <strong>/users/{hostedResult.user_id}/{hostedResult.server_slug}</strong>
                    </p>
                  </div>
                  <div className="form-group" style={{ marginBottom: 0 }}>
                    <label className="form-label" style={{ fontWeight: 600 }}>MCP config (for your IDE)</label>
                    <p style={{ fontSize: '0.75rem', color: 'var(--text-muted)', marginBottom: '0.5rem' }}>
                      Add this to your MCP client config (e.g. Cursor <code>mcp.json</code>, Claude Desktop settings). You can edit the server name key if you like.
                    </p>
                    <div style={{ position: 'relative' }}>
                      <pre style={{ padding: '1rem', borderRadius: '8px', fontSize: '0.8125rem', overflow: 'auto', margin: 0, whiteSpace: 'pre-wrap', wordBreak: 'break-all' }}>
                        {hostedResult.mcp_config}
                      </pre>
                      <button
                        type="button"
                        className="btn btn-primary"
                        style={{ position: 'absolute', top: '0.5rem', right: '0.5rem' }}
                        onClick={() => { navigator.clipboard.writeText(hostedResult.mcp_config); toast.success('MCP config copied'); }}
                      >
                        <i className="bi bi-clipboard"></i> Copy config
                      </button>
                    </div>
                  </div>
                </div>
              )}
            </div>
          )}

          {!selectedDeploy && (
            <div style={{ 
              textAlign: 'center', 
              padding: '3rem', 
              background: 'var(--dark-bg)', 
              borderRadius: '12px',
              border: '2px dashed var(--card-border)'
            }}>
              <i className="bi bi-arrow-up-circle" style={{ fontSize: '2.5rem', color: 'var(--text-muted)', marginBottom: '1rem', display: 'block' }}></i>
              <p style={{ color: 'var(--text-muted)', margin: 0 }}>
                Select a deployment method above to see instructions
              </p>
            </div>
          )}
            </div>
          </div>
        </div>
      )}

        </div>
      </div>

      {/* GitHub Export Modal */}
      {showGitHubModal && (
        <div 
          style={{
            position: 'fixed',
            top: 0,
            left: 0,
            right: 0,
            bottom: 0,
            background: 'rgba(0, 0, 0, 0.5)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            zIndex: 1000,
          }}
          onClick={() => setShowGitHubModal(false)}
        >
          <div 
            style={{
              background: 'var(--card-bg)',
              borderRadius: '12px',
              width: '100%',
              maxWidth: '550px',
              maxHeight: '90vh',
              overflow: 'auto',
              boxShadow: '0 20px 40px rgba(0, 0, 0, 0.2)',
            }}
            onClick={(e) => e.stopPropagation()}
          >
            <div style={{ 
              display: 'flex', 
              justifyContent: 'space-between', 
              alignItems: 'center',
              padding: '1rem 1.25rem',
              borderBottom: '1px solid var(--card-border)'
            }}>
              <h3 style={{ margin: 0, fontSize: '1.125rem' }}>
                <i className="bi bi-github" style={{ marginRight: '0.5rem' }}></i>
                Push to GitHub
              </h3>
              <button 
                className="btn btn-icon btn-secondary"
                onClick={() => setShowGitHubModal(false)}
              >
                <i className="bi bi-x-lg"></i>
              </button>
            </div>
            <div style={{ padding: '1.25rem' }}>
              <div className="form-group">
                <label className="form-label">
                  GitHub Personal Access Token *
                  <a 
                    href="https://github.com/settings/tokens/new?scopes=repo" 
                    target="_blank" 
                    rel="noopener noreferrer"
                    style={{ marginLeft: '0.5rem', fontSize: '0.75rem' }}
                  >
                    Create token
                  </a>
                </label>
                <input
                  type="password"
                  className="form-control"
                  value={githubToken}
                  onChange={(e) => setGithubToken(e.target.value)}
                  placeholder="ghp_xxxxxxxxxxxx"
                />
                <small style={{ color: 'var(--text-muted)', marginTop: '0.25rem', display: 'block' }}>
                  Token needs <code>repo</code> scope for private repos
                </small>
              </div>

              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '1rem' }}>
                <div className="form-group">
                  <label className="form-label">Owner / Organization *</label>
                  <input
                    type="text"
                    className="form-control"
                    value={githubOwner}
                    onChange={(e) => setGithubOwner(e.target.value)}
                    placeholder="username or org"
                  />
                </div>
                <div className="form-group">
                  <label className="form-label">Repository Name *</label>
                  <input
                    type="text"
                    className="form-control"
                    value={githubRepo}
                    onChange={(e) => setGithubRepo(e.target.value)}
                    placeholder="my-mcp-server"
                  />
                </div>
              </div>

              <div className="form-group">
                <label className="form-label">Branch</label>
                <input
                  type="text"
                  className="form-control"
                  value={githubBranch}
                  onChange={(e) => setGithubBranch(e.target.value)}
                  placeholder="main"
                />
              </div>

              <div className="form-group">
                <label className="form-label">Commit Message</label>
                <input
                  type="text"
                  className="form-control"
                  value={githubCommitMsg}
                  onChange={(e) => setGithubCommitMsg(e.target.value)}
                  placeholder="Initial MCP server export"
                />
              </div>

              <div style={{ 
                background: 'var(--hover-bg)', 
                borderRadius: '8px', 
                padding: '1rem',
                marginTop: '1rem'
              }}>
                <div className="form-group" style={{ marginBottom: '0.75rem' }}>
                  <label style={{ display: 'flex', alignItems: 'center', gap: '0.75rem', cursor: 'pointer' }}>
                    <input
                      type="checkbox"
                      checked={githubCreateRepo}
                      onChange={(e) => setGithubCreateRepo(e.target.checked)}
                      style={{ width: 18, height: 18 }}
                    />
                    <div>
                      <div style={{ fontWeight: 500 }}>Create repository if it doesn't exist</div>
                      <div style={{ fontSize: '0.75rem', color: 'var(--text-muted)' }}>
                        Will create a new repo under your account
                      </div>
                    </div>
                  </label>
                </div>

                {githubCreateRepo && (
                  <div className="form-group" style={{ marginBottom: 0, marginLeft: '2rem' }}>
                    <label style={{ display: 'flex', alignItems: 'center', gap: '0.75rem', cursor: 'pointer' }}>
                      <input
                        type="checkbox"
                        checked={githubPrivate}
                        onChange={(e) => setGithubPrivate(e.target.checked)}
                        style={{ width: 18, height: 18 }}
                      />
                      <span style={{ fontWeight: 500 }}>Make repository private</span>
                    </label>
                  </div>
                )}
              </div>
            </div>
            <div style={{ 
              display: 'flex', 
              justifyContent: 'flex-end', 
              gap: '0.75rem',
              padding: '1rem 1.25rem',
              borderTop: '1px solid var(--card-border)'
            }}>
              <button 
                className="btn btn-secondary"
                onClick={() => setShowGitHubModal(false)}
              >
                Cancel
              </button>
              <button 
                className="btn btn-primary"
                onClick={handleGitHubExport}
                disabled={githubExporting || !githubToken || !githubOwner || !githubRepo}
              >
                {githubExporting ? (
                  <>
                    <span className="spinner" style={{ width: 16, height: 16, marginRight: '0.5rem' }}></span>
                    Pushing...
                  </>
                ) : (
                  <>
                    <i className="bi bi-github"></i>
                    Push to GitHub
                  </>
                )}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Publish Modal */}
      {showPublishModal && (
        <div 
          style={{
            position: 'fixed',
            top: 0,
            left: 0,
            right: 0,
            bottom: 0,
            backgroundColor: 'rgba(0,0,0,0.5)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            zIndex: 10000,
          }}
          onClick={() => setShowPublishModal(false)}
        >
          <div 
            style={{
              backgroundColor: 'white',
              borderRadius: '12px',
              width: '90%',
              maxWidth: '500px',
              overflow: 'hidden',
            }}
            onClick={(e) => e.stopPropagation()}
          >
            <div style={{ 
              padding: '1.25rem', 
              borderBottom: '1px solid var(--card-border)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
            }}>
              <h3 style={{ margin: 0 }}>
                <i className="bi bi-upload" style={{ marginRight: '0.75rem' }}></i>
                Publish Version
              </h3>
              <button 
                onClick={() => setShowPublishModal(false)}
                style={{
                  background: 'none',
                  border: 'none',
                  fontSize: '1.25rem',
                  cursor: 'pointer',
                  color: 'var(--text-muted)',
                }}
              >
                <i className="bi bi-x"></i>
              </button>
            </div>

            <div style={{ padding: '1.25rem' }}>
              <p style={{ color: 'var(--text-secondary)', marginTop: 0, marginBottom: '1.5rem' }}>
                Publishing creates a snapshot of your server that can be downloaded from the marketplace.
                Changes after publishing won't affect published versions.
              </p>

              <div style={{ marginBottom: '1rem' }}>
                <label style={{ display: 'block', marginBottom: '0.5rem', fontWeight: 500 }}>Version *</label>
                <input
                  type="text"
                  style={{
                    width: '100%',
                    padding: '0.75rem',
                    border: '1px solid var(--card-border)',
                    borderRadius: '8px',
                    fontSize: '1rem',
                    boxSizing: 'border-box',
                  }}
                  value={publishVersion}
                  onChange={(e) => setPublishVersion(e.target.value)}
                  placeholder="e.g., 1.0.0"
                />
                <small style={{ color: 'var(--text-muted)', display: 'block', marginTop: '0.25rem' }}>
                  Use semantic versioning (major.minor.patch)
                </small>
              </div>

              <div style={{ marginBottom: '1rem' }}>
                <label style={{ display: 'block', marginBottom: '0.5rem', fontWeight: 500 }}>Release Notes</label>
                <textarea
                  style={{
                    width: '100%',
                    padding: '0.75rem',
                    border: '1px solid var(--card-border)',
                    borderRadius: '8px',
                    fontSize: '1rem',
                    resize: 'vertical',
                    minHeight: '80px',
                    boxSizing: 'border-box',
                  }}
                  rows={3}
                  value={publishNotes}
                  onChange={(e) => setPublishNotes(e.target.value)}
                  placeholder="What's new in this version..."
                />
              </div>

              <div style={{ 
                display: 'flex', 
                alignItems: 'flex-start', 
                gap: '0.75rem', 
                padding: '1rem',
                background: 'var(--background-secondary)',
                borderRadius: '8px',
                cursor: 'pointer',
              }}
              onClick={() => setPublishPublic(!publishPublic)}
              >
                <input
                  type="checkbox"
                  checked={publishPublic}
                  onChange={(e) => setPublishPublic(e.target.checked)}
                  style={{ width: 20, height: 20, marginTop: '2px', cursor: 'pointer' }}
                />
                <div>
                  <div style={{ fontWeight: 500 }}>Make public in marketplace</div>
                  <div style={{ fontSize: '0.8rem', color: 'var(--text-muted)' }}>
                    Anyone can discover and download this server
                  </div>
                </div>
              </div>
            </div>

            <div style={{ 
              display: 'flex', 
              justifyContent: 'flex-end', 
              gap: '0.75rem',
              padding: '1rem 1.25rem',
              borderTop: '1px solid var(--card-border)',
              background: 'var(--background-secondary)',
            }}>
              <button 
                className="btn btn-secondary"
                onClick={() => setShowPublishModal(false)}
              >
                Cancel
              </button>
              <button 
                className="btn btn-primary"
                onClick={handlePublish}
                disabled={publishing || !publishVersion.trim()}
              >
                {publishing ? (
                  <>
                    <span className="spinner" style={{ width: 16, height: 16, marginRight: '0.5rem' }}></span>
                    Publishing...
                  </>
                ) : (
                  <>
                    <i className="bi bi-upload"></i>
                    Publish v{publishVersion}
                  </>
                )}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
