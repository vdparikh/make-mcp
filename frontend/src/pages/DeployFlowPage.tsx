import { useEffect, useMemo, useState } from 'react';
import { Link, useNavigate, useSearchParams } from 'react-router-dom';
import { toast } from 'react-toastify';
import {
  compositionHostedDeploy,
  compositionHostedStatus,
  downloadMarketplaceServer,
  exportComposition,
  generateServer,
  getComposition,
  githubExport,
  getMarketplaceServer,
  getServer,
  downloadHostedSecurityAuditExport,
  getHostedSecurity,
  hostedPublish,
  hostedStatus,
  listCompositions,
  listHostedSessions,
  listMarketplace,
  listServers,
  marketplaceHostedDeploy,
  marketplaceHostedStatus,
  rotateHostedAccessKey,
  type HostedAuthMode,
  type HostedEgressPolicy,
  type HostedIsolationTier,
  type HostedStatusResponse,
} from '../services/api';
import type { Server, ServerComposition } from '../types';

function formatHostedAuthModeLabel(mode?: string): string {
  switch (mode) {
    case 'bearer_token':
      return 'Bearer / API key';
    case 'oidc':
      return 'OIDC / SSO';
    case 'mtls':
      return 'mTLS';
    case 'no_auth':
      return 'No auth';
    default:
      return mode?.trim() ? mode : '—';
  }
}

function formatLiveIdleLabel(mins?: number): string {
  if (mins === undefined) return '—';
  if (mins === 0) return 'Never auto-shutdown';
  return `${mins} min`;
}

type TargetType = 'server' | 'marketplace' | 'composition';
type DeployMethod = 'hosted' | 'local' | 'cloud';
type LocalRuntime = 'nodejs' | 'docker';
type EnvProfile = '' | 'dev' | 'staging' | 'prod';
type DeployTargetOption = {
  id: string;
  type: TargetType;
  name: string;
  subtitle: string;
};

export default function DeployFlowPage() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const target = (searchParams.get('target') || '') as TargetType;
  const targetId = searchParams.get('id') || '';

  const [loadingTarget, setLoadingTarget] = useState(true);
  const [targetName, setTargetName] = useState('');
  const [method, setMethod] = useState<DeployMethod>('hosted');
  const [localRuntime, setLocalRuntime] = useState<LocalRuntime>('nodejs');
  const [envProfile, setEnvProfile] = useState<EnvProfile>('');
  const [idleTimeoutMinutes, setIdleTimeoutMinutes] = useState<number>(0);
  const [hostedAuthMode, setHostedAuthMode] = useState<HostedAuthMode>('no_auth');
  const [requireCallerIdentity, setRequireCallerIdentity] = useState(false);
  const [hostedSecurityJson, setHostedSecurityJson] = useState('{}');
  const [isolationTier, setIsolationTier] = useState<HostedIsolationTier>('standard');
  const [egressPolicy, setEgressPolicy] = useState<HostedEgressPolicy>('allow_all');
  const [egressAllowlistText, setEgressAllowlistText] = useState('');
  const [rotatingHostedKey, setRotatingHostedKey] = useState(false);
  const [auditExporting, setAuditExporting] = useState(false);
  const [downloading, setDownloading] = useState(false);

  const [publishingHosted, setPublishingHosted] = useState(false);
  const [hostedStatusLoading, setHostedStatusLoading] = useState(false);
  const [hostedResult, setHostedResult] = useState<HostedStatusResponse | null>(null);
  const [showMcpConfig, setShowMcpConfig] = useState(false);
  const [showHostedSecurityAdvanced, setShowHostedSecurityAdvanced] = useState(false);
  const [showRuntimeDetails, setShowRuntimeDetails] = useState(false);
  const [showManifest, setShowManifest] = useState(false);
  const [githubToken, setGithubToken] = useState('');
  const [githubOwner, setGithubOwner] = useState('');
  const [githubRepo, setGithubRepo] = useState('');
  const [githubBranch, setGithubBranch] = useState('main');
  const [githubCommitMsg, setGithubCommitMsg] = useState('');
  const [githubCreateRepo, setGithubCreateRepo] = useState(false);
  const [githubPrivate, setGithubPrivate] = useState(true);
  const [githubExporting, setGithubExporting] = useState(false);
  const [targetPickerLoading, setTargetPickerLoading] = useState(false);
  const [targetPickerOptions, setTargetPickerOptions] = useState<DeployTargetOption[]>([]);
  const [targetPickerType, setTargetPickerType] = useState<'all' | TargetType>('all');

  const validTarget = target === 'server' || target === 'marketplace' || target === 'composition';
  const hasSelectedTarget = validTarget && Boolean(targetId);
  const showTargetPicker = !hasSelectedTarget;
  const targetLabel = target === 'composition' ? 'composition' : target === 'marketplace' ? 'marketplace server' : 'MCP server';
  const breadcrumbLabel = target === 'composition' ? 'Compositions' : target === 'marketplace' ? 'Marketplace' : 'Deploy';
  const envLabel = envProfile ? envProfile.toUpperCase() : 'None (.env at runtime)';
  const idleLabel = idleTimeoutMinutes > 0 ? `${idleTimeoutMinutes} min` : 'Never auto-shutdown';
  const hostedAuthLabel = useMemo(() => {
    switch (hostedAuthMode) {
      case 'bearer_token':
        return 'Bearer / API key';
      case 'oidc':
        return 'OIDC / SSO';
      case 'mtls':
        return 'mTLS';
      default:
        return 'No auth';
    }
  }, [hostedAuthMode]);

  const endpointProtectionHelp = useMemo(() => {
    switch (hostedAuthMode) {
      case 'bearer_token':
        return 'Require a shared secret or bearer token on each request (headers X-Make-MCP-Key or X-MCP-API-Key). Rotate keys from Advanced JSON when this mode is on.';
      case 'oidc':
        return 'Clients must send a valid OIDC JWT in Authorization: Bearer. Set issuer and audience in Advanced JSON (see security guide).';
      case 'mtls':
        return 'Expect a client TLS certificate; ingress passes a cert fingerprint in X-Make-MCP-Client-Cert-SHA256. Optional IP allowlist in JSON.';
      default:
        return 'No bearer check at the edge. You can still lock things down with IP rules, mTLS, or caller rules in Advanced JSON.';
    }
  }, [hostedAuthMode]);

  const deployMethodHint = useMemo(() => {
    switch (method) {
      case 'hosted':
        return 'Runs on Make MCP’s infrastructure. You get a URL, status, and one-click install for Cursor / VS Code.';
      case 'local':
        return 'Download the generated package and run it with Node.js or Docker on your machine—ideal for local dev and full control.';
      case 'cloud':
        return 'Push to GitHub or plug the artifact into your own CI/CD without using the hosted runtime.';
      default:
        return '';
    }
  }, [method]);

  const publishDiffersFromLive = useMemo(() => {
    if (!hostedResult?.running || hostedStatusLoading) return false;
    const liveMode = hostedResult.hosted_auth_mode;
    if (liveMode && hostedAuthMode !== liveMode) return true;
    if (hostedResult.require_caller_identity !== requireCallerIdentity) return true;
    const liveIdle = hostedResult.idle_timeout_minutes ?? 0;
    if (liveIdle !== idleTimeoutMinutes) return true;
    return false;
  }, [
    hostedResult?.running,
    hostedResult?.hosted_auth_mode,
    hostedResult?.require_caller_identity,
    hostedResult?.idle_timeout_minutes,
    hostedStatusLoading,
    hostedAuthMode,
    requireCallerIdentity,
    idleTimeoutMinutes,
  ]);

  const buildHostedRuntimeConfig = (): Record<string, unknown> | undefined => {
    const extra = egressAllowlistText
      .split('\n')
      .map((s) => s.trim())
      .filter((s) => s.length > 0);
    const o: Record<string, unknown> = {
      isolation_tier: isolationTier,
      egress_policy: egressPolicy,
    };
    if (egressPolicy === 'deny_default' && extra.length > 0) {
      o.egress_allowlist = extra;
    }
    return o;
  };

  const parseHostedSecurityConfig = (): Record<string, unknown> | undefined => {
    const raw = hostedSecurityJson.trim();
    if (!raw || raw === '{}') return undefined;
    try {
      const parsed: unknown = JSON.parse(raw);
      if (typeof parsed !== 'object' || parsed === null || Array.isArray(parsed)) {
        toast.error('hosted_security_config must be a JSON object');
        return undefined;
      }
      return parsed as Record<string, unknown>;
    } catch {
      toast.error('Invalid JSON in hosted security profile');
      return undefined;
    }
  };

  useEffect(() => {
    if (!hasSelectedTarget) {
      setLoadingTarget(false);
      return;
    }
    let cancelled = false;
    setLoadingTarget(true);

    const load = async () => {
      try {
        if (target === 'server') {
          const s = await getServer(targetId);
          if (!cancelled) setTargetName(s.name);
        } else if (target === 'marketplace') {
          const m = await getMarketplaceServer(targetId);
          if (!cancelled) setTargetName(m.server.name);
        } else {
          const c = await getComposition(targetId);
          if (!cancelled) setTargetName(c.name);
        }
      } catch {
        if (!cancelled) toast.error('Failed to load deploy target');
      } finally {
        if (!cancelled) setLoadingTarget(false);
      }
    };
    load();
    return () => { cancelled = true; };
  }, [target, targetId, hasSelectedTarget]);

  useEffect(() => {
    if (!showTargetPicker) return;
    let cancelled = false;
    setTargetPickerLoading(true);
    const loadOptions = async () => {
      try {
        const [servers, marketplaceServers, compositions] = await Promise.all([
          listServers(),
          listMarketplace(),
          listCompositions(),
        ]);
        if (cancelled) return;
        const serverOptions: DeployTargetOption[] = (servers || []).map((s: Server) => ({
          id: s.id,
          type: 'server',
          name: s.name,
          subtitle: 'Your server',
        }));
        const marketplaceOptions: DeployTargetOption[] = (marketplaceServers || []).map((s: Server) => ({
          id: s.id,
          type: 'marketplace',
          name: s.name,
          subtitle: 'Marketplace server',
        }));
        const compositionOptions: DeployTargetOption[] = (compositions || []).map((c: ServerComposition) => ({
          id: c.id,
          type: 'composition',
          name: c.name,
          subtitle: 'Composition',
        }));
        setTargetPickerOptions([...serverOptions, ...marketplaceOptions, ...compositionOptions]);
      } catch {
        if (!cancelled) toast.error('Failed to load deploy targets');
      } finally {
        if (!cancelled) setTargetPickerLoading(false);
      }
    };
    loadOptions();
    return () => {
      cancelled = true;
    };
  }, [showTargetPicker]);

  const refreshHostedStatus = async () => {
    if (!hasSelectedTarget) return;
    setHostedStatusLoading(true);
    try {
      const normalizeStatus = (status: HostedStatusResponse | null): HostedStatusResponse | null => {
        if (!status) return null;
        const inferredRunning =
          status.running ||
          Boolean(status.endpoint || status.container_id || status.host_port || status.started_at || status.snapshot_version);
        return { ...status, running: inferredRunning };
      };

      const sleep = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms));

      let normalized: HostedStatusResponse | null = null;
      for (let attempt = 0; attempt < 3; attempt += 1) {
        let status: HostedStatusResponse;
        if (target === 'server') status = await hostedStatus(targetId);
        else if (target === 'marketplace') status = await marketplaceHostedStatus(targetId);
        else status = await compositionHostedStatus(targetId);
        normalized = normalizeStatus(status);
        if (normalized?.running) break;
        if (attempt < 2) await sleep(450 * (attempt + 1));
      }

      if (!normalized?.running && target === 'server') {
        // Fallback: session list can still show active runtime while status endpoint is catching up.
        const sessions = await listHostedSessions();
        const active = sessions.find((session) => session.server_id === targetId && session.status !== 'stopped');
        if (active) {
          normalized = {
            ...(normalized || { running: false }),
            running: true,
            container_id: active.container_id || normalized?.container_id,
            host_port: active.host_port || normalized?.host_port,
            started_at: active.started_at || normalized?.started_at,
            last_ensured_at: active.last_ensured_at || normalized?.last_ensured_at,
          };
        }
      }

      setHostedResult(normalized || { running: false });
      if (normalized?.hosted_auth_mode) {
        setHostedAuthMode(normalized.hosted_auth_mode as HostedAuthMode);
      }
      if (normalized?.require_caller_identity !== undefined) {
        setRequireCallerIdentity(normalized.require_caller_identity);
      }

      if (target === 'server') {
        try {
          const sec = await getHostedSecurity(targetId);
          if (sec.hosted_auth_mode) {
            setHostedAuthMode(sec.hosted_auth_mode);
          }
          if (sec.require_caller_identity !== undefined) {
            setRequireCallerIdentity(sec.require_caller_identity);
          }
          if (sec.hosted_security_config && Object.keys(sec.hosted_security_config).length > 0) {
            setHostedSecurityJson(JSON.stringify(sec.hosted_security_config, null, 2));
            setShowHostedSecurityAdvanced(true);
          } else {
            setHostedSecurityJson('{}');
            setShowHostedSecurityAdvanced(false);
          }
          if (sec.hosted_runtime_config && Object.keys(sec.hosted_runtime_config).length > 0) {
            const r = sec.hosted_runtime_config;
            const tier = r.isolation_tier;
            if (tier === 'standard' || tier === 'restricted' || tier === 'strict') {
              setIsolationTier(tier);
            }
            const ep = r.egress_policy;
            if (ep === 'allow_all' || ep === 'deny_default') {
              setEgressPolicy(ep);
            }
            const al = r.egress_allowlist;
            if (Array.isArray(al)) {
              setEgressAllowlistText(al.filter((x): x is string => typeof x === 'string').join('\n'));
            } else {
              setEgressAllowlistText('');
            }
          }
        } catch {
          /* keep form state if security endpoint fails */
        }
      }
    } catch {
      setHostedResult(null);
    } finally {
      setHostedStatusLoading(false);
    }
  };

  useEffect(() => {
    if (method !== 'hosted' || !hasSelectedTarget) return;
    refreshHostedStatus();
  }, [method, target, targetId, hasSelectedTarget]);

  const artifactSlug = useMemo(
    () =>
      (targetName || targetLabel)
        .toLowerCase()
        .replace(/\s+/g, '-')
        .replace(/[^a-z0-9-]/g, '')
        .replace(/-+/g, '-')
        .replace(/^-|-$/g, '') || 'mcp-server',
    [targetName, targetLabel]
  );

  useEffect(() => {
    if (target !== 'server' || !targetName) return;
    if (!githubRepo) {
      setGithubRepo(`${artifactSlug}-mcp-server`);
    }
    if (!githubCommitMsg) {
      setGithubCommitMsg(`Initial MCP server export: ${targetName}`);
    }
  }, [target, targetName, artifactSlug, githubRepo, githubCommitMsg]);

  const nodeClientConfig = JSON.stringify(
    {
      mcpServers: {
        [artifactSlug]: {
          command: 'node',
          args: ['/path/to/your-server/run-with-log.mjs'],
        },
      },
    },
    null,
    2
  );

  const dockerClientConfig = `{
  "mcpServers": {
    "${artifactSlug}": {
      "command": "docker",
      "args": ["run", "-i", "--rm", "${artifactSlug}-mcp"]
    }
  }
}`;

  const copyText = async (value: string, successMsg: string) => {
    try {
      await navigator.clipboard.writeText(value);
      toast.success(successMsg);
    } catch {
      toast.error('Could not copy');
    }
  };

  const handleHostedPublish = async () => {
    if (!hasSelectedTarget) return;
    setPublishingHosted(true);
    try {
      if (target === 'server') {
        const hostedSecurityConfig = parseHostedSecurityConfig();
        if (hostedSecurityConfig === undefined && hostedSecurityJson.trim() && hostedSecurityJson.trim() !== '{}') {
          return;
        }
        await hostedPublish(
          targetId,
          undefined,
          envProfile || undefined,
          idleTimeoutMinutes,
          hostedAuthMode,
          requireCallerIdentity,
          hostedSecurityConfig,
          buildHostedRuntimeConfig()
        );
      } else if (target === 'marketplace') {
        await marketplaceHostedDeploy(targetId, envProfile || undefined, idleTimeoutMinutes, hostedAuthMode, requireCallerIdentity);
      } else {
        await compositionHostedDeploy(targetId, envProfile || undefined, idleTimeoutMinutes, hostedAuthMode, requireCallerIdentity);
      }
      toast.success('Hosted MCP published');
      await refreshHostedStatus();
    } catch (err: unknown) {
      const message =
        typeof err === 'object' && err !== null && 'response' in err
          ? (err as { response?: { data?: { error?: string } } }).response?.data?.error
          : undefined;
      toast.error(message || 'Hosted publish failed');
    } finally {
      setPublishingHosted(false);
    }
  };

  const handleRotateHostedAccessKey = async () => {
    if (target !== 'server' || !targetId) return;
    setRotatingHostedKey(true);
    try {
      const res = await rotateHostedAccessKey(targetId);
      toast.success(res.warning || 'Hosted access key rotated. Update clients with the new secret.');
      await refreshHostedStatus();
    } catch (err: unknown) {
      const message =
        typeof err === 'object' && err !== null && 'response' in err
          ? (err as { response?: { data?: { error?: string } } }).response?.data?.error
          : undefined;
      toast.error(message || 'Key rotation failed');
    } finally {
      setRotatingHostedKey(false);
    }
  };

  const handleDownloadHostedSecurityAudit = async () => {
    if (target !== 'server' || !targetId) return;
    setAuditExporting(true);
    try {
      const blob = await downloadHostedSecurityAuditExport(targetId);
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `hosted-security-audit-${targetId}.csv`;
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      document.body.removeChild(a);
      toast.success('Audit export downloaded');
    } catch (err: unknown) {
      const message =
        typeof err === 'object' && err !== null && 'response' in err
          ? (err as { response?: { data?: { error?: string } } }).response?.data?.error
          : undefined;
      toast.error(message || 'Audit export failed');
    } finally {
      setAuditExporting(false);
    }
  };

  const handleDownload = async () => {
    if (!hasSelectedTarget) return;
    setDownloading(true);
    try {
      let blob: Blob;
      if (target === 'server') {
        blob = await generateServer(targetId, envProfile || undefined);
      } else if (target === 'marketplace') {
        blob = await downloadMarketplaceServer(targetId, envProfile || undefined);
      } else {
        blob = await exportComposition(targetId, {
          prefix_tool_names: false,
          merge_resources: true,
          merge_prompts: true,
          env_profile: envProfile || undefined,
        });
      }
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `${artifactSlug}-mcp-server.zip`;
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      document.body.removeChild(a);
      toast.success('ZIP downloaded');
    } catch {
      toast.error('Download failed');
    } finally {
      setDownloading(false);
    }
  };

  const handleGitHubExport = async () => {
    if (target !== 'server' || !targetId) return;
    if (!githubToken || !githubOwner || !githubRepo) {
      toast.error('GitHub token, owner, and repo are required');
      return;
    }
    setGithubExporting(true);
    try {
      const result = await githubExport(targetId, {
        token: githubToken,
        owner: githubOwner,
        repo: githubRepo,
        branch: githubBranch || 'main',
        commit_message: githubCommitMsg || `Export MCP server ${targetName || artifactSlug}`,
        create_repo: githubCreateRepo,
        private: githubPrivate,
        description: targetName ? `MCP Server: ${targetName}` : 'MCP Server export',
      });
      toast.success(result.message || 'Exported to GitHub');
      if (result.repo_url) window.open(result.repo_url, '_blank');
    } catch (err: unknown) {
      const message =
        typeof err === 'object' && err !== null && 'response' in err
          ? (err as { response?: { data?: { error?: string } } }).response?.data?.error
          : undefined;
      toast.error(message || 'Failed to export to GitHub');
    } finally {
      setGithubExporting(false);
    }
  };

  const formatRuntimeTime = (value?: string) => {
    if (!value) return '—';
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) return '—';
    return date.toLocaleString();
  };

  const hostedManifest = (hostedResult?.manifest ?? null) as Record<string, unknown> | null;

  const oneClickLinks = useMemo(() => {
    if (!hostedResult?.mcp_config) return null;
    try {
      const parsed = JSON.parse(hostedResult.mcp_config);
      const servers = parsed.mcpServers || {};
      const serverName = Object.keys(servers)[0];
      if (!serverName) return null;
      const serverConfig = servers[serverName];
      const configB64 = btoa(JSON.stringify(serverConfig));
      const cursorLink = `cursor://anysphere.cursor-deeplink/mcp/install?name=${encodeURIComponent(serverName)}&config=${configB64}`;
      const vscodeConfig = JSON.stringify({ name: serverName, type: 'sse', url: serverConfig.url, ...(serverConfig.headers ? { headers: serverConfig.headers } : {}) });
      const vscodeLink = `vscode:mcp/install?${encodeURIComponent(vscodeConfig)}`;
      const vscodeInsidersLink = `vscode-insiders:mcp/install?${encodeURIComponent(vscodeConfig)}`;
      return { serverName, serverConfig, cursorLink, vscodeLink, vscodeInsidersLink };
    } catch {
      return null;
    }
  }, [hostedResult?.mcp_config]);

  const filteredTargetOptions = targetPickerOptions.filter((option) => targetPickerType === 'all' || option.type === targetPickerType);

  if (showTargetPicker) {
    return (
      <div className="deploy-flow-page">
        <div className="page-header">
          <div>
            <nav className="page-breadcrumb">
              <Link to="/" className="page-breadcrumb-link">Dashboard</Link>
              <span className="page-breadcrumb-sep">/</span>
              <span className="page-breadcrumb-current">Deploy</span>
            </nav>
            <h1 className="page-title">
              <i className="bi bi-rocket-takeoff page-title-icon"></i>
              Deploy
            </h1>
            <p className="page-subtitle">Pick a server, marketplace item, or composition to continue deployment.</p>
          </div>
          <Link to="/" className="btn btn-secondary">
            <i className="bi bi-arrow-left"></i>
            Back
          </Link>
        </div>

        <div className="modal-overlay deploy-target-picker-overlay">
          <div className="modal-content deploy-target-picker-modal">
            <div className="modal-header">
              <h3 className="modal-title">Choose what to deploy</h3>
              <Link to="/" className="btn btn-secondary btn-sm">
                <i className="bi bi-x-lg"></i>
              </Link>
            </div>
            <div className="modal-body">
              <div className="deploy-target-picker-filters">
                <button type="button" className={`btn btn-sm ${targetPickerType === 'all' ? 'btn-primary' : 'btn-secondary'}`} onClick={() => setTargetPickerType('all')}>All</button>
                <button type="button" className={`btn btn-sm ${targetPickerType === 'server' ? 'btn-primary' : 'btn-secondary'}`} onClick={() => setTargetPickerType('server')}>Servers</button>
                <button type="button" className={`btn btn-sm ${targetPickerType === 'marketplace' ? 'btn-primary' : 'btn-secondary'}`} onClick={() => setTargetPickerType('marketplace')}>Marketplace</button>
                <button type="button" className={`btn btn-sm ${targetPickerType === 'composition' ? 'btn-primary' : 'btn-secondary'}`} onClick={() => setTargetPickerType('composition')}>Compositions</button>
              </div>

              {targetPickerLoading ? (
                <p className="deploy-flow-help">Loading deploy targets...</p>
              ) : filteredTargetOptions.length === 0 ? (
                <p className="deploy-flow-help">No deployable items found. Create a server or composition first.</p>
              ) : (
                <div className="deploy-target-picker-list">
                  {filteredTargetOptions.map((option) => (
                    <button
                      key={`${option.type}:${option.id}`}
                      type="button"
                      className="deploy-target-picker-item"
                      onClick={() => navigate(`/deploy?target=${option.type}&id=${encodeURIComponent(option.id)}`)}
                    >
                      <span>
                        <strong>{option.name}</strong>
                        <small>{option.subtitle}</small>
                      </span>
                      <i className="bi bi-chevron-right"></i>
                    </button>
                  ))}
                </div>
              )}
            </div>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="deploy-flow-page">
      <div className="page-header">
        <div>
          <nav className="page-breadcrumb">
            <Link to={target === 'marketplace' ? '/marketplace' : '/'} className="page-breadcrumb-link">
              {breadcrumbLabel}
            </Link>
            <span className="page-breadcrumb-sep">/</span>
            <span className="page-breadcrumb-current">Deploy</span>
          </nav>
          <h1 className="page-title">
            <i className="bi bi-rocket-takeoff page-title-icon"></i>
            Deploy {loadingTarget ? targetLabel : targetName || targetLabel}
          </h1>
          <p className="page-subtitle">Guided deploy flow with hosted runtime first and advanced details on demand.</p>
        </div>
        <button type="button" className="btn btn-secondary" onClick={() => navigate(-1)}>
          <i className="bi bi-arrow-left"></i>
          Back
        </button>
      </div>

      <div className="card deploy-flow-method-card">
        <div className="deploy-flow-section-intro">
          <h2 className="deploy-flow-section-title">Environment profile</h2>
          <p className="deploy-flow-help small mb-0">
            Picks which row from <strong>Server → Environments</strong> is baked into generated code when you download or publish (base URLs, database URL for DB tools).
          </p>
        </div>
        <div className="deploy-flow-setup-grid">
          <div className="deploy-flow-setting-card h-100">
            <label className="form-label" htmlFor="deploy-env-profile-select">Profile for build &amp; package</label>
            <select
              id="deploy-env-profile-select"
              className="form-control"
              value={envProfile}
              onChange={(e) => setEnvProfile(e.target.value as EnvProfile)}
              aria-describedby="deploy-env-profile-help"
            >
              <option value="">None — use .env / secrets at runtime</option>
              <option value="dev">Dev</option>
              <option value="staging">Staging</option>
              <option value="prod">Prod</option>
            </select>
            <p id="deploy-env-profile-help" className="deploy-flow-help small mt-2 mb-0">
              Leave on “None” if you inject URLs and credentials only when the process starts (typical for local runs).
            </p>
          </div>
          <div className="deploy-flow-summary-card h-100">
            <span className="deploy-flow-meta-label">Active for this flow</span>
            <strong>{envLabel}</strong>
          </div>
        </div>
        <details className="deploy-flow-env-details">
          <summary>
            <span className="deploy-flow-env-details-summary-text">Different from per-request env headers</span>
          </summary>
          <p className="deploy-flow-help small mb-0">
            This control is <strong>not</strong> the same as clients sending <code>X-Make-MCP-Env</code> or the <code>env_profiles</code> block in Advanced security JSON—those affect runtime routing for hosted calls.{' '}
            <Link to="/docs?doc=hosted-security">When to use which</Link>
          </p>
        </details>
      </div>

      <div className="card deploy-flow-mode-card">
        <div className="deploy-flow-mode-label">Deployment mode</div>
        <div className="deploy-flow-method-tabs">
          <button
            type="button"
            className={`deploy-flow-method-tab ${method === 'hosted' ? 'active' : ''}`}
            onClick={() => setMethod('hosted')}
            title="Hosted runtime on Make MCP"
          >
            <i className="bi bi-globe" aria-hidden="true"></i>
            Hosted (Recommended)
          </button>
          <button
            type="button"
            className={`deploy-flow-method-tab ${method === 'local' ? 'active' : ''}`}
            onClick={() => setMethod('local')}
            title="Run the generated server locally"
          >
            <i className="bi bi-laptop" aria-hidden="true"></i>
            Local Runtime
          </button>
          <button
            type="button"
            className={`deploy-flow-method-tab ${method === 'cloud' ? 'active' : ''}`}
            onClick={() => setMethod('cloud')}
            title="GitHub or custom pipeline"
          >
            <i className="bi bi-cloud-upload" aria-hidden="true"></i>
            Cloud / Repo
          </button>
        </div>
        <p className="deploy-flow-mode-hint">{deployMethodHint}</p>
      </div>

      {method === 'hosted' && (
        <div className="card deploy-flow-panel deploy-flow-panel-hosted">
          <div className="deploy-flow-panel-head">
            <div>
              <h3 className="card-title" style={{ marginBottom: '0.25rem' }}>Hosted MCP</h3>
              <p className="deploy-flow-help">
                <strong>Live deployment</strong> is what is running now. <strong>Next publish</strong> is what will apply when you click Publish — they can differ until you publish.
              </p>
            </div>
            <div className="deploy-flow-inline-actions">
              <button
                type="button"
                className="btn btn-secondary btn-sm me-2"
                onClick={refreshHostedStatus}
                disabled={hostedStatusLoading}
              >
                <i className="bi bi-arrow-clockwise"></i>
                Refresh status
              </button>
              <button
                className="btn btn-success"
                onClick={handleHostedPublish}
                disabled={publishingHosted}
              >
                {publishingHosted ? <><i className="bi bi-hourglass-split"></i> Publishing...</> : <><i className="bi bi-globe"></i>  {hostedResult?.running ? 'Re-publish MCP' : 'Publish MCP'}</>}
              </button>
            </div>
          </div>

          <div className="deploy-live-runtime">
            <div className="deploy-live-runtime-header">
              <h4 className="deploy-flow-section-title">Live deployment</h4>
              <p className="deploy-flow-help small mb-0">
                Read-only: the snapshot and URL clients use today. Use <strong>Refresh status</strong> to update.
              </p>
            </div>
            <div className={`hosted-status-banner ${hostedStatusLoading ? 'loading' : hostedResult?.running ? 'running' : 'idle'}`}>
              {hostedStatusLoading ? (
                <span><i className="bi bi-arrow-repeat"></i> Checking hosted runtime status...</span>
              ) : hostedResult?.running ? (
                <span><span className="hosted-running-dot" /><i className="bi bi-check-circle"></i> Already running</span>
              ) : (
                <span><i className="bi bi-info-circle"></i> No hosted runtime currently running</span>
              )}
            </div>
            {hostedResult?.running && !hostedStatusLoading && (
              <div className="deploy-live-meta">
                <div className="deploy-live-meta-row">
                  <span className="deploy-flow-meta-label">Published snapshot</span>
                  <span>{hostedResult.snapshot_version || hostedResult.version || '—'}</span>
                </div>
                <div className="deploy-live-meta-row">
                  <span className="deploy-flow-meta-label">Started</span>
                  <span>{formatRuntimeTime(hostedResult.started_at)}</span>
                </div>
                <div className="deploy-live-meta-row">
                  <span className="deploy-flow-meta-label">Last activity</span>
                  <span>{formatRuntimeTime(hostedResult.last_ensured_at)}</span>
                </div>
              </div>
            )}
            {hostedResult?.endpoint && (
              <div className="deploy-flow-endpoint-row">
                <div className="deploy-flow-endpoint-main">
                  <div className="deploy-flow-meta-label">Server URL</div>
                  <code>{hostedResult.endpoint}</code>
                </div>
                <button type="button" className="btn btn-secondary btn-sm" onClick={() => copyText(hostedResult.endpoint || '', 'Endpoint copied')}>
                  <i className="bi bi-clipboard"></i>
                  Copy
                </button>
              </div>
            )}
            {hostedResult?.running && !hostedStatusLoading && (
              <div className="deploy-publish-policy-strip deploy-live-policy-strip mb-2" aria-label="Live deployment policy">
                <span className="deploy-publish-policy-chip">
                  <i
                    className={`bi ${
                      hostedResult.hosted_auth_mode === 'no_auth' ? 'bi-unlock' : 'bi-shield-check'
                    }`}
                  ></i>
                  Live: {formatHostedAuthModeLabel(hostedResult.hosted_auth_mode)}
                </span>
                <span className="deploy-publish-policy-chip">
                  <i className={`bi ${hostedResult.require_caller_identity ? 'bi-person-check' : 'bi-person'}`}></i>
                  Caller ID: {hostedResult.require_caller_identity ? 'Required' : 'Optional'}
                </span>
                <span className="deploy-publish-policy-chip">
                  <i className="bi bi-clock"></i>
                  Idle: {formatLiveIdleLabel(hostedResult.idle_timeout_minutes)}
                </span>
              </div>
            )}
            {oneClickLinks && (
              <div className="deploy-oneclick-section">
                <div className="deploy-oneclick-header">
                  <i className="bi bi-lightning-charge-fill"></i>
                  <div>
                    <h4>One-click install</h4>
                    <p className="mb-0 small text-muted">Opens your IDE with the live server pre-filled.</p>
                  </div>
                </div>
                <div className="deploy-oneclick-buttons d-flex align-items-center mb-2">
                  <a href={oneClickLinks.cursorLink} className="me-2 gap-2 d-flex deploy-oneclick-btn deploy-oneclick-cursor" title="Install in Cursor">
                    <svg width="18" height="18" viewBox="0 0 100 100" fill="currentColor"><path d="M50 0 L95 25 L95 75 L50 100 L5 75 L5 25 Z" opacity="0.15"/><path d="M30 25 L75 50 L30 75 Z"/></svg>
                    Install in Cursor
                  </a>
                  <a href={oneClickLinks.vscodeLink} className="me-2 gap-2 d-flex deploy-oneclick-btn deploy-oneclick-vscode" title="Install in VS Code">
                    <svg width="18" height="18" viewBox="0 0 100 100" fill="currentColor"><path d="M71.5 99.1l23.4-11.6V13l-23.4-12L2.2 39.6 0 42.5v15.9l2.2 2.5 69.3 38.2zM29.5 70.2L17.7 60.4l11.8-9.5v19.3zM71.5 76L42.4 60l29.1-16V76z" opacity="0.9"/></svg>
                    Install in VS Code
                  </a>
                  <a href={oneClickLinks.vscodeInsidersLink} className="gap-2 d-flex deploy-oneclick-btn deploy-oneclick-vscode-insiders" title="Install in VS Code Insiders">
                    <svg width="18" height="18" viewBox="0 0 100 100" fill="currentColor"><path d="M71.5 99.1l23.4-11.6V13l-23.4-12L2.2 39.6 0 42.5v15.9l2.2 2.5 69.3 38.2zM29.5 70.2L17.7 60.4l11.8-9.5v19.3zM71.5 76L42.4 60l29.1-16V76z" opacity="0.9"/></svg>
                    VS Code Insiders
                  </a>
                </div>
                <p className="deploy-oneclick-hint d-flex align-items-start">
                  <i className="bi bi-info-circle"></i>
                  {hostedResult?.require_caller_identity ? (
                    <div>
                      One-click opens your IDE with this config. This server requires a verified caller key — go to{' '}
                      <Link to="/hosted/keys">Caller API Keys</Link> to generate one and replace{' '}
                      <code>&lt;caller-api-key&gt;</code> in the config (or in the install payload) with your <code>mkc_…</code> secret.
                    </div>
                  ) : (
                    <div>Click a button above — your IDE will open and prompt you to confirm the installation.</div>
                  )}
                </p>
              </div>
            )}
          </div>

          <div className="deploy-next-publish">
            <div className="deploy-next-publish-header">
              <h4 className="deploy-flow-section-title">Next publish</h4>
              <p className="deploy-flow-help small mb-2">
                Edit settings below, then click Publish or Re-publish. Nothing here changes the live deployment until publish completes.
              </p>
              {publishDiffersFromLive && (
                <div className="alert alert-warning py-2 px-3 mb-0" role="status" style={{ fontSize: '0.85rem' }}>
                  <i className="bi bi-exclamation-triangle me-1"></i>
                  You have unpublished changes — the live deployment above still reflects the last successful publish.
                </div>
              )}
            </div>
            <div className="deploy-publish-settings">
            <div className="deploy-publish-settings-header">
              <i className="bi bi-shield-lock"></i>
              <div>
                <h4>Access &amp; Security</h4>
                <p className="mb-0 text-muted small">Who may call the endpoint, optional caller attribution, and when the container may stop.</p>
              </div>
            </div>

            <div className="deploy-publish-grid">
              <div className="deploy-publish-card">
                <div className="deploy-publish-card-header">
                  <i className="bi bi-key"></i>
                  <span>Endpoint protection</span>
                </div>
                <p className="deploy-publish-card-desc">{endpointProtectionHelp}</p>
                <div className="deploy-publish-toggle-row" style={{ flexWrap: 'wrap', gap: '0.35rem' }}>
                  <button
                    type="button"
                    className={`deploy-publish-pill ${hostedAuthMode === 'no_auth' ? 'active' : ''}`}
                    onClick={() => setHostedAuthMode('no_auth')}
                    title="No bearer check; use only with other protections"
                  >
                    <i className="bi bi-unlock"></i> No auth
                  </button>
                  <button
                    type="button"
                    className={`deploy-publish-pill ${hostedAuthMode === 'bearer_token' ? 'active' : ''}`}
                    onClick={() => setHostedAuthMode('bearer_token')}
                    title="Require API key or bearer token on each request"
                  >
                    <i className="bi bi-lock"></i> Bearer / key
                  </button>
                  <button
                    type="button"
                    className={`deploy-publish-pill ${hostedAuthMode === 'oidc' ? 'active' : ''}`}
                    onClick={() => setHostedAuthMode('oidc')}
                    title="Validate OIDC JWTs (SSO / identity provider)"
                  >
                    <i className="bi bi-building"></i> OIDC
                  </button>
                  <button
                    type="button"
                    className={`deploy-publish-pill ${hostedAuthMode === 'mtls' ? 'active' : ''}`}
                    onClick={() => setHostedAuthMode('mtls')}
                    title="Client TLS certificates at the edge"
                  >
                    <i className="bi bi-shield-lock"></i> mTLS
                  </button>
                </div>
              </div>

              <div className="deploy-publish-card">
                <div className="deploy-publish-card-header">
                  <i className="bi bi-person-badge"></i>
                  <span>Caller identity</span>
                </div>
                <p className="deploy-publish-card-desc">
                  {requireCallerIdentity
                    ? 'Header X-Make-MCP-Caller-Id + caller key (mkc_…). Separate from hosted access key.'
                    : 'Optional attribution with caller keys (mkc_…). Not the hosted access key.'}
                </p>
                <label className="deploy-publish-switch">
                  <input
                    type="checkbox"
                    checked={requireCallerIdentity}
                    onChange={(e) => setRequireCallerIdentity(e.target.checked)}
                  />
                  <span className="deploy-publish-switch-slider" />
                  <span className="deploy-publish-switch-label">
                    {requireCallerIdentity ? 'Required' : 'Optional'}
                  </span>
                </label>
                <div className="deploy-flow-inline-actions" style={{ marginTop: '0.5rem' }}>
                  <Link className="btn btn-sm btn-outline-primary" to="/hosted/keys">
                    <i className="bi bi-key"></i>
                    Manage caller keys
                  </Link>
                </div>
              </div>

              <div className="deploy-publish-card">
                <div className="deploy-publish-card-header">
                  <i className="bi bi-clock-history"></i>
                  <span>Idle shutdown</span>
                </div>
                <p className="deploy-publish-card-desc">
                  {idleTimeoutMinutes > 0
                    ? `Container stops after ${idleTimeoutMinutes} min of inactivity and restarts on next request.`
                    : 'Container runs indefinitely. Stop manually from Observability > Sessions.'}
                </p>
                <select
                  className="form-control form-control-sm"
                  value={String(idleTimeoutMinutes)}
                  onChange={(e) => setIdleTimeoutMinutes(Number(e.target.value || 0))}
                >
                  <option value="0">Never (manual stop only)</option>
                  <option value="15">15 minutes</option>
                  <option value="30">30 minutes</option>
                  <option value="60">1 hour</option>
                  <option value="180">3 hours</option>
                  <option value="720">12 hours</option>
                  <option value="1440">24 hours</option>
                </select>
              </div>
            </div>

            {target === 'server' && (
              <div className="deploy-publish-card deploy-publish-card--wide mt-3">
                <div className="deploy-publish-card-header">
                  <i className="bi bi-box-seam"></i>
                  <span>Runtime isolation</span>
                </div>
                <p className="deploy-publish-card-desc small text-muted mb-2">
                  Docker CPU/memory tier. Optional <strong>deny-by-default</strong> outbound HTTP for generated tools
                  (cold-start <code>npm install</code> is unchanged). Tool URLs, Target env base URL, and observability
                  ingest are auto-allowed.{' '}
                  <Link to="/docs?doc=hosted-runtime-isolation">Docs</Link>
                </p>
                <div className="mb-2">
                  <div className="form-label text-muted mb-1">Isolation tier</div>
                  <div className="deploy-publish-toggle-row" style={{ flexWrap: 'wrap', gap: '0.35rem' }}>
                    <button
                      type="button"
                      className={`deploy-publish-pill ${isolationTier === 'standard' ? 'active' : ''}`}
                      onClick={() => setIsolationTier('standard')}
                    >
                      <i className="bi bi-speedometer2"></i> Standard
                    </button>
                    <button
                      type="button"
                      className={`deploy-publish-pill ${isolationTier === 'restricted' ? 'active' : ''}`}
                      onClick={() => setIsolationTier('restricted')}
                    >
                      <i className="bi bi-slash-circle"></i> Restricted
                    </button>
                    <button
                      type="button"
                      className={`deploy-publish-pill ${isolationTier === 'strict' ? 'active' : ''}`}
                      onClick={() => setIsolationTier('strict')}
                    >
                      <i className="bi bi-shield-fill"></i> Strict
                    </button>
                  </div>
                </div>
                <div className="mb-2">
                  <label className="form-label small text-muted mb-1">Tool HTTP egress</label>
                  <select
                    className="form-control form-control-sm"
                    value={egressPolicy}
                    onChange={(e) => setEgressPolicy(e.target.value as HostedEgressPolicy)}
                  >
                    <option value="allow_all">Allow all (default)</option>
                    <option value="deny_default">Deny by default + allowlist</option>
                  </select>
                </div>
                {egressPolicy === 'deny_default' && (
                  <div>
                    <label className="form-label small text-muted mb-1">Extra allowed hostnames (one per line)</label>
                    <textarea
                      className="form-control font-monospace"
                      rows={4}
                      spellCheck={false}
                      value={egressAllowlistText}
                      onChange={(e) => setEgressAllowlistText(e.target.value)}
                      placeholder={'partner-api.example.com\n*.vendor.io'}
                      style={{ fontSize: '0.8rem' }}
                    />
                  </div>
                )}
              </div>
            )}

            {target === 'server' && (
              <div className="deploy-flow-collapsible-list mb-2" style={{ marginTop: '1rem' }}>
                <button
                  type="button"
                  className="deploy-flow-collapse-trigger border-0"
                  onClick={() => setShowHostedSecurityAdvanced((v) => !v)}
                >
                  <span>
                    <i className="bi bi-file-earmark-code"></i> Advanced security (JSON)
                    {hostedSecurityJson.trim() !== '{}' && hostedSecurityJson.trim() !== '' ? (
                      <span className="badge bg-secondary ms-2" style={{ fontSize: '0.65rem', verticalAlign: 'middle' }}>
                        custom
                      </span>
                    ) : null}
                  </span>
                  <i className={`bi ${showHostedSecurityAdvanced ? 'bi-chevron-up' : 'bi-chevron-down'}`}></i>
                </button>
                {showHostedSecurityAdvanced && (
                  <div className="deploy-flow-collapse-body card deploy-publish-card border-0 shadow-none ps-3 pe-3">
                    <p className="deploy-flow-help small mb-2">
                      Power-user JSON: OIDC issuer/audience, OAuth BFF for browser login, IP allowlists, and{' '}
                      <code>env_profiles</code> for per-request routing (see <code>X-Make-MCP-Env</code>). Empty object uses defaults.{' '}
                      <Link to="/docs?doc=hosted-security">Security guide</Link>
                    </p>
                    <textarea
                      className="form-control font-monospace"
                      rows={10}
                      spellCheck={false}
                      value={hostedSecurityJson}
                      onChange={(e) => setHostedSecurityJson(e.target.value)}
                      style={{ fontSize: '0.8rem' }}
                    />
                    <div className="deploy-flow-inline-actions mt-2 d-flex flex-wrap gap-2">
                      {hostedAuthMode === 'bearer_token' && (
                        <button
                          type="button"
                          className="btn btn-outline-warning btn-sm"
                          disabled={rotatingHostedKey}
                          onClick={handleRotateHostedAccessKey}
                        >
                          {rotatingHostedKey ? (
                            <>
                              <i className="bi bi-hourglass-split"></i> Rotating…
                            </>
                          ) : (
                            <>
                              <i className="bi bi-arrow-repeat"></i> Rotate hosted access key
                            </>
                          )}
                        </button>
                      )}
                      <button
                        type="button"
                        className="btn btn-outline-secondary btn-sm"
                        disabled={auditExporting}
                        onClick={handleDownloadHostedSecurityAudit}
                      >
                        {auditExporting ? (
                          <>
                            <i className="bi bi-hourglass-split"></i> Exporting…
                          </>
                        ) : (
                          <>
                            <i className="bi bi-download"></i> Export security audit (CSV)
                          </>
                        )}
                      </button>
                    </div>
                  </div>
                )}
              </div>
            )}

            <p className="deploy-publish-preview-caption">Preview for next publish (auth, caller, idle — plus isolation when applicable)</p>
            <div className="deploy-publish-policy-strip" aria-label="Next publish policy preview">
              <span className="deploy-publish-policy-chip">
                <i
                  className={`bi ${
                    hostedAuthMode === 'no_auth' ? 'bi-unlock' : 'bi-shield-check'
                  }`}
                ></i>
                Next: {hostedAuthLabel}
              </span>
              <span className="deploy-publish-policy-chip">
                <i className={`bi ${requireCallerIdentity ? 'bi-person-check' : 'bi-person'}`}></i>
                Caller ID: {requireCallerIdentity ? 'Required' : 'Optional'}
              </span>
              <span className="deploy-publish-policy-chip">
                <i className="bi bi-clock"></i>
                {idleLabel}
              </span>
              {target === 'server' && (
                <>
                  <span className="deploy-publish-policy-chip">
                    <i className="bi bi-box-seam"></i>
                    Tier: {isolationTier}
                  </span>
                  <span className="deploy-publish-policy-chip">
                    <i className="bi bi-diagram-3"></i>
                    Egress: {egressPolicy === 'deny_default' ? 'Allowlist' : 'Open'}
                  </span>
                </>
              )}
            </div>
          </div>
          </div>

          <div className="deploy-flow-collapsible-list">
            <button type="button" className="deploy-flow-collapse-trigger" onClick={() => setShowMcpConfig((v) => !v)}>
              <span>
                <i className="bi bi-terminal"></i> Manual config (JSON)
                {oneClickLinks ? (
                  <span className="text-muted small ms-1 fw-normal">— optional if you use one-click above</span>
                ) : null}
              </span>
              <i className={`bi ${showMcpConfig ? 'bi-chevron-up' : 'bi-chevron-down'}`}></i>
            </button>
            {showMcpConfig && (
              <div className="deploy-flow-collapse-body">
                {hostedResult?.mcp_config ? (
                  <>
                    {hostedResult.require_caller_identity && (
                      <div className="alert alert-info py-2 px-3 mb-3" role="status" style={{ fontSize: '0.875rem' }}>
                        <strong>Before connecting:</strong> create a caller key on{' '}
                        <Link to="/hosted/keys">Caller API Keys</Link> and paste it in place of <code>&lt;caller-api-key&gt;</code>.
                      </div>
                    )}
                    <p className="deploy-flow-help mb-2">
                      Paste into your IDE's <code>mcp.json</code>, <code>mcp_config.json</code>, or <code>claude_desktop_config.json</code>.
                    </p>
                    <div className="deploy-flow-inline-actions">
                      <button type="button" className="btn btn-sm btn-outline-primary" onClick={() => copyText(hostedResult.mcp_config || '', 'MCP config copied')}>
                        <i className="bi bi-clipboard"></i>
                        Copy config
                      </button>
                    </div>
                    <pre>{hostedResult.mcp_config}</pre>
                  </>
                ) : (
                  <p className="deploy-flow-help">Publish first to generate this config.</p>
                )}
              </div>
            )}

            <button type="button" className="deploy-flow-collapse-trigger" onClick={() => setShowRuntimeDetails((v) => !v)}>
              <span><i className="bi bi-info-circle"></i> Runtime details</span>
              <i className={`bi ${showRuntimeDetails ? 'bi-chevron-up' : 'bi-chevron-down'}`}></i>
            </button>
            {showRuntimeDetails && (
              <div className="deploy-flow-collapse-body">
                <div className="deploy-flow-runtime-grid">
                  <span><strong>Snapshot:</strong> {hostedResult?.snapshot_version || hostedResult?.version || '—'}</span>
                  <span><strong>Started:</strong> {formatRuntimeTime(hostedResult?.started_at)}</span>
                  <span><strong>Last ensured:</strong> {formatRuntimeTime(hostedResult?.last_ensured_at)}</span>
                  <span><strong>Container:</strong> {hostedResult?.container_id ? hostedResult.container_id.slice(0, 12) : '—'}</span>
                  <span><strong>Port:</strong> {hostedResult?.host_port || '—'}</span>
                  <span><strong>Runtime:</strong> {hostedResult?.runtime || 'docker'}</span>
                  <span><strong>Image:</strong> {hostedResult?.image || 'node:20-alpine'}</span>
                  <span><strong>Memory limit:</strong> {hostedResult?.memory_mb || 512} MB</span>
                  <span><strong>CPU limit:</strong> {hostedResult?.nano_cpus ? `${(hostedResult.nano_cpus / 1_000_000_000).toFixed(2)} CPU` : '0.50 CPU'}</span>
                  <span><strong>PIDs limit:</strong> {hostedResult?.pids_limit || 128}</span>
                  <span><strong>Network:</strong> {hostedResult?.network_scope || '127.0.0.1:random-port -> 3000/tcp'}</span>
                  <span><strong>Idle shutdown:</strong> {hostedResult?.idle_timeout_minutes ? `${hostedResult.idle_timeout_minutes} min` : 'Disabled'}</span>
                  <span><strong>Auth:</strong> {formatHostedAuthModeLabel(hostedResult?.hosted_auth_mode)}</span>
                  <span><strong>Caller ID:</strong> {hostedResult?.require_caller_identity ? 'Required' : 'Optional'}</span>
                </div>
              </div>
            )}

            <button type="button" className="deploy-flow-collapse-trigger" onClick={() => setShowManifest((v) => !v)}>
              <span><i className="bi bi-file-earmark-code"></i> Manifest JSON</span>
              <i className={`bi ${showManifest ? 'bi-chevron-up' : 'bi-chevron-down'}`}></i>
            </button>
            {showManifest && (
              <div className="deploy-flow-collapse-body">
                {hostedManifest ? (
                  <pre>{JSON.stringify(hostedManifest, null, 2)}</pre>
                ) : (
                  <p className="deploy-flow-help">No manifest available yet.</p>
                )}
              </div>
            )}
          </div>
        </div>
      )}

      {method === 'local' && (
        <div className="card deploy-flow-panel">
          <div className="deploy-flow-panel-head">
            <div>
              <h3 className="card-title" style={{ marginBottom: '0.25rem' }}>Local runtime</h3>
              <p className="deploy-flow-help">Download package and run with Node.js or Docker.</p>
            </div>
            <button className="btn btn-primary" onClick={handleDownload} disabled={downloading}>
              <i className="bi bi-download"></i>
              {downloading ? 'Generating...' : 'Download ZIP'}
            </button>
          </div>

          <div className="deploy-flow-runtime-picker">
            <button
              type="button"
              className={`deploy-flow-runtime-card ${localRuntime === 'nodejs' ? 'active' : ''}`}
              onClick={() => setLocalRuntime('nodejs')}
            >
              <i className="bi bi-filetype-js"></i>
              <span>
                <strong>Node.js</strong>
                <small>Run directly with Node runtime</small>
              </span>
            </button>
            <button
              type="button"
              className={`deploy-flow-runtime-card ${localRuntime === 'docker' ? 'active' : ''}`}
              onClick={() => setLocalRuntime('docker')}
            >
              <i className="bi bi-box-seam"></i>
              <span>
                <strong>Docker</strong>
                <small>Run as an isolated container</small>
              </span>
            </button>
          </div>

          <div className="deploy-flow-step-list">
            <div className="deploy-flow-step-card">
              <h4 className="deploy-flow-mini-title mb-2">
                <span className="deploy-flow-step-badge">1</span>
                Setup and run
              </h4>
              <pre>
{localRuntime === 'docker'
  ? `cd ${artifactSlug}-mcp-server
cp .env.example .env
docker-compose up -d`
  : `cd ${artifactSlug}-mcp-server
npm install
npm run build`}
              </pre>
              <p className="deploy-flow-help deploy-flow-step-note">
                After you add the MCP client config below, your IDE launches the server command automatically.
              </p>
            </div>
            <div className="deploy-flow-step-card">
              <div className="deploy-flow-mini-title-row">
                <h4 className="deploy-flow-mini-title">
                  <span className="deploy-flow-step-badge">2</span>
                  Configure MCP client
                </h4>
                <button
                  type="button"
                  className="btn btn-sm btn-outline-primary"
                  onClick={() => copyText(localRuntime === 'docker' ? dockerClientConfig : nodeClientConfig, 'MCP config copied')}
                >
                  <i className="bi bi-clipboard"></i>
                  Copy
                </button>
              </div>
              <p className="deploy-flow-help" style={{ marginBottom: '0.45rem' }}>
                Add this to your IDE <code>mcp.json</code>:
              </p>
              <pre>{localRuntime === 'docker' ? dockerClientConfig : nodeClientConfig}</pre>
            </div>
          </div>
        </div>
      )}

      {method === 'cloud' && (
        <div className="card deploy-flow-panel">
          <div className="deploy-flow-panel-head">
            <div>
              <h3 className="card-title" style={{ marginBottom: '0.25rem' }}>Cloud / repository flow</h3>
              <p className="deploy-flow-help">Push directly to GitHub (server targets) or download package for your CI/CD stack.</p>
            </div>
            <button className="btn btn-primary" onClick={handleDownload} disabled={downloading}>
              <i className="bi bi-download"></i>
              {downloading ? 'Generating...' : 'Download ZIP'}
            </button>
          </div>
          {target === 'server' ? (
            <div className="deploy-flow-github-form">
              <div className="deploy-flow-github-grid">
                <div className="form-group">
                  <label className="form-label">GitHub token *</label>
                  <input
                    type="password"
                    className="form-control"
                    value={githubToken}
                    onChange={(e) => setGithubToken(e.target.value)}
                    placeholder="ghp_xxx..."
                  />
                </div>
                <div className="form-group">
                  <label className="form-label">Owner *</label>
                  <input
                    type="text"
                    className="form-control"
                    value={githubOwner}
                    onChange={(e) => setGithubOwner(e.target.value)}
                    placeholder="your-org-or-user"
                  />
                </div>
                <div className="form-group">
                  <label className="form-label">Repository *</label>
                  <input
                    type="text"
                    className="form-control"
                    value={githubRepo}
                    onChange={(e) => setGithubRepo(e.target.value)}
                    placeholder="my-mcp-server"
                  />
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
              </div>
              <div className="form-group">
                <label className="form-label">Commit message</label>
                <input
                  type="text"
                  className="form-control"
                  value={githubCommitMsg}
                  onChange={(e) => setGithubCommitMsg(e.target.value)}
                />
              </div>
              <div className="deploy-flow-github-options">
                <label><input type="checkbox" checked={githubCreateRepo} onChange={(e) => setGithubCreateRepo(e.target.checked)} /> Create repository if missing</label>
                <label><input type="checkbox" checked={githubPrivate} onChange={(e) => setGithubPrivate(e.target.checked)} /> Private repository</label>
              </div>
              <div className="deploy-flow-inline-actions">
                <button type="button" className="btn btn-primary" onClick={handleGitHubExport} disabled={githubExporting}>
                  <i className="bi bi-github"></i>
                  {githubExporting ? 'Pushing...' : 'Push to GitHub'}
                </button>
              </div>
            </div>
          ) : (
            <div className="deploy-flow-cloud-placeholder">
              <i className="bi bi-git"></i>
              <h4>Cloud deploy pipeline</h4>
              <p>Download ZIP and push from your local workflow. Direct GitHub push is currently available for server targets.</p>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

