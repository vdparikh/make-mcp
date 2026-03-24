package api

import (
	"bytes"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vdparikh/make-mcp/backend/internal/database"
	"github.com/vdparikh/make-mcp/backend/internal/hosted"
	"github.com/vdparikh/make-mcp/backend/internal/hostedruntime"
	"github.com/vdparikh/make-mcp/backend/internal/hostedsecurity"
	"github.com/vdparikh/make-mcp/backend/internal/models"
)

// HostedPublishRequest is the body for hosted-publish.
type HostedPublishRequest struct {
	Version               string          `json:"version"` // deprecated; ignored for hosted runtime selection
	EnvProfile            string          `json:"env_profile,omitempty"`
	IdleTimeoutMinutes    *int            `json:"idle_timeout_minutes,omitempty"`
	HostedAuthMode        string          `json:"hosted_auth_mode,omitempty"` // no_auth | bearer_token | oidc | mtls
	RequireCallerIdentity *bool           `json:"require_caller_identity,omitempty"`
	HostedSecurityConfig  json.RawMessage `json:"hosted_security_config,omitempty"`
	// HostedRuntimeConfig is optional isolation tier, resource overrides, egress policy (see docs/hosted-runtime-isolation.md).
	HostedRuntimeConfig json.RawMessage `json:"hosted_runtime_config,omitempty"`
}

type HostedDeployConfig struct {
	EnvProfile            string `json:"env_profile,omitempty"`
	IdleTimeoutMinutes    *int   `json:"idle_timeout_minutes,omitempty"`
	HostedAuthMode        string `json:"hosted_auth_mode,omitempty"`
	RequireCallerIdentity *bool  `json:"require_caller_identity,omitempty"`
}

// HostedPublishResponse is returned after publishing to hosted.
type HostedPublishResponse struct {
	BaseURL    string `json:"base_url"`
	UserID     string `json:"user_id"`
	ServerSlug string `json:"server_slug"`
	Version    string `json:"version"`
	Endpoint   string `json:"endpoint"`
	MCPConfig  string `json:"mcp_config"`
}

type HostedStatusResponse struct {
	Running               bool            `json:"running"`
	UserID                string          `json:"user_id,omitempty"`
	ServerID              string          `json:"server_id,omitempty"`
	ServerSlug            string          `json:"server_slug,omitempty"`
	Version               string          `json:"version,omitempty"`
	SnapshotID            string          `json:"snapshot_id,omitempty"`
	SnapshotVersion       string          `json:"snapshot_version,omitempty"`
	StartedAt             string          `json:"started_at,omitempty"`
	LastEnsuredAt         string          `json:"last_ensured_at,omitempty"`
	Endpoint              string          `json:"endpoint,omitempty"`
	MCPConfig             string          `json:"mcp_config,omitempty"`
	Manifest              json.RawMessage `json:"manifest,omitempty"`
	ContainerID           string          `json:"container_id,omitempty"`
	HostPort              string          `json:"host_port,omitempty"`
	Runtime               string          `json:"runtime,omitempty"`
	Image                 string          `json:"image,omitempty"`
	MemoryMB              int64           `json:"memory_mb,omitempty"`
	NanoCPUs              int64           `json:"nano_cpus,omitempty"`
	PidsLimit             int64           `json:"pids_limit,omitempty"`
	IdleTimeoutMinutes    int             `json:"idle_timeout_minutes,omitempty"`
	NetworkScope          string          `json:"network_scope,omitempty"`
	HostedAuthMode        string          `json:"hosted_auth_mode,omitempty"`
	RequireCallerIdentity bool            `json:"require_caller_identity"`
}

type HostedSessionListItem struct {
	models.HostedSession
	ServerName string `json:"server_name,omitempty"`
}

type HostedCatalogItem struct {
	ServerID              string `json:"server_id"`
	ServerName            string `json:"server_name"`
	ServerSlug            string `json:"server_slug"`
	PublisherUserID       string `json:"publisher_user_id"`
	SnapshotVersion       string `json:"snapshot_version,omitempty"`
	Endpoint              string `json:"endpoint"`
	MCPConfig             string `json:"mcp_config"`
	HostedAuthMode        string `json:"hosted_auth_mode,omitempty"`
	RequireCallerIdentity bool   `json:"require_caller_identity"`
	LastEnsuredAt         string `json:"last_ensured_at,omitempty"`
}

type HostedCallerAPIKeyCreateRequest struct {
	CallerUserID string   `json:"caller_user_id,omitempty"` // defaults to current user id
	TenantID     string   `json:"tenant_id,omitempty"`
	Scopes       []string `json:"scopes,omitempty"`
	AllowAlias   bool     `json:"allow_alias"`
	ExpiresAt    string   `json:"expires_at,omitempty"` // RFC3339; optional
}

type HostedCallerAPIKeyCreateResponse struct {
	Key models.HostedCallerAPIKey `json:"key"`
	// api_key is only returned once at creation time.
	APIKey string `json:"api_key"`
}

func normalizeIdleTimeoutMinutes(value *int) (int, error) {
	if value == nil {
		return 0, nil
	}
	if *value < 0 || *value > 10080 {
		return 0, fmt.Errorf("idle_timeout_minutes must be between 0 and 10080")
	}
	return *value, nil
}

func normalizeHostedAuthMode(value string) (string, error) {
	mode := strings.TrimSpace(strings.ToLower(value))
	if mode == "" {
		return hostedAuthModeNoAuth, nil
	}
	// Map legacy values to the simplified model.
	switch mode {
	case hostedAuthModeBearerToken, "auto_flow":
		return hostedAuthModeBearerToken, nil
	case hostedAuthModeNoAuth, "caller_identity":
		return hostedAuthModeNoAuth, nil
	case hostedAuthModeOIDC:
		return hostedAuthModeOIDC, nil
	case hostedAuthModeMTLS:
		return hostedAuthModeMTLS, nil
	default:
		return "", fmt.Errorf("hosted_auth_mode must be one of: %s, %s, %s, %s", hostedAuthModeBearerToken, hostedAuthModeNoAuth, hostedAuthModeOIDC, hostedAuthModeMTLS)
	}
}

// HostedPublish publishes the latest hosted snapshot and makes it available at /api/users/:user_id/:server_slug.
func (h *Handler) HostedPublish(c *gin.Context) {
	id := c.Param("id")
	server := h.requireServerOwnership(c, id)
	if server == nil {
		return
	}

	userID := ""
	if uid, exists := c.Get("userID"); exists {
		userID = uid.(string)
	}
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	var req HostedPublishRequest
	_ = c.ShouldBindJSON(&req)
	idleTimeoutMinutes, idleErr := normalizeIdleTimeoutMinutes(req.IdleTimeoutMinutes)
	if idleErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": idleErr.Error()})
		return
	}
	hostedAuthMode, authModeErr := normalizeHostedAuthMode(req.HostedAuthMode)
	if authModeErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": authModeErr.Error()})
		return
	}
	if err := h.db.UpdateServerHostedAuthMode(c.Request.Context(), server.ID, hostedAuthMode); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to persist hosted auth mode"})
		return
	}
	server.HostedAuthMode = hostedAuthMode
	if req.RequireCallerIdentity != nil {
		if err := h.db.UpdateServerRequireCallerIdentity(c.Request.Context(), server.ID, *req.RequireCallerIdentity); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to persist caller identity setting"})
			return
		}
		server.RequireCallerIdentity = *req.RequireCallerIdentity
	}
	if len(req.HostedSecurityConfig) > 0 {
		if len(req.HostedSecurityConfig) > 256*1024 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "hosted_security_config exceeds maximum size"})
			return
		}
		if err := h.db.UpdateServerHostedSecurityConfig(c.Request.Context(), server.ID, req.HostedSecurityConfig); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to persist hosted security config"})
			return
		}
		server.HostedSecurityConfig = req.HostedSecurityConfig
	}
	if len(req.HostedRuntimeConfig) > 0 {
		if len(req.HostedRuntimeConfig) > 32*1024 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "hosted_runtime_config exceeds maximum size"})
			return
		}
		if err := h.db.UpdateServerHostedRuntimeConfig(c.Request.Context(), server.ID, req.HostedRuntimeConfig); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to persist hosted runtime config"})
			return
		}
		server.HostedRuntimeConfig = req.HostedRuntimeConfig
	}

	// Keep request backward-compatible, but hosted deployment no longer uses user-provided version.
	// Also repair older data where hosted snapshots polluted servers.latest_version.
	if strings.HasPrefix(strings.TrimSpace(server.LatestVersion), "hosted-") {
		if nonHosted, err := h.db.GetLatestNonHostedServerVersion(c.Request.Context(), id); err == nil {
			nextLatest := ""
			if nonHosted != nil {
				nextLatest = nonHosted.Version
				server.LatestVersion = nonHosted.Version
			}
			_ = h.db.UpdateServerLatestVersion(c.Request.Context(), id, nextLatest)
		}
	}

	snapshot, err := json.Marshal(server)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create snapshot"})
		return
	}
	sv, err := h.db.CreateHostedServerVersion(c.Request.Context(), id, userID, snapshot)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	hostedVersion := sv.Version

	// Ensure one running container for this (user, server), using the selected published version snapshot.
	snapshotServer := *server
	if sv != nil && len(sv.Snapshot) > 0 {
		var snap models.Server
		if err := json.Unmarshal(sv.Snapshot, &snap); err == nil {
			snapshotServer = snap
		}
	}
	observabilityEnv := h.hostedObservabilityEnv(c, server)
	rawRTC := req.HostedRuntimeConfig
	if len(rawRTC) == 0 {
		rawRTC = server.HostedRuntimeConfig
	}
	egressExtra, resolvedRuntime, mergeErr := h.mergeHostedRuntimeEnv(c, server, &snapshotServer, rawRTC, req.EnvProfile)
	if mergeErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": mergeErr.Error()})
		return
	}
	for k, v := range egressExtra {
		observabilityEnv[k] = v
	}
	if h.hostedMgr != nil {
		if cfg, err := h.hostedMgr.EnsureContainer(c.Request.Context(), userID, server.ID, hostedVersion, &snapshotServer, observabilityEnv, resolvedRuntime, idleTimeoutMinutes); err != nil {
			// Log but do not fail the publish.
			fmt.Printf("HostedPublish: failed to ensure container for user=%s server=%s: %v\n", userID, server.ID, err)
		} else {
			fmt.Printf("HostedPublish: ensured container for user=%s server=%s version=%s container=%s host_port=%s\n", userID, server.ID, hostedVersion, cfg.ContainerID, cfg.HostPort)
		}
	}

	slug := database.ServerSlug(server.Name)
	baseURL := strings.TrimSuffix(h.hostedBaseURL(c), "/")
	endpoint := baseURL + "/users/" + userID + "/" + slug

	hostedAccessKey, err := h.db.EnsureServerHostedAccessKey(c.Request.Context(), server.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to provision hosted access token"})
		return
	}
	mcpConfigBytes, err := h.buildHostedMCPConfig(slug, endpoint, server, hostedAccessKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to build mcp config"})
		return
	}
	mcpConfig := string(mcpConfigBytes)

	c.JSON(http.StatusOK, HostedPublishResponse{
		BaseURL:    baseURL,
		UserID:     userID,
		ServerSlug: slug,
		Version:    hostedVersion,
		Endpoint:   endpoint,
		MCPConfig:  mcpConfig,
	})
}

func (h *Handler) hostedBaseURL(c *gin.Context) string {
	baseURL := strings.TrimSpace(h.cfg.URLs.MCPHostedBaseURL)
	if baseURL != "" {
		return strings.TrimSuffix(baseURL, "/")
	}
	scheme := "https"
	if c.Request.TLS == nil && (c.Request.Header.Get("X-Forwarded-Proto") == "" || c.Request.Header.Get("X-Forwarded-Proto") == "http") {
		scheme = "http"
	}
	if proto := c.Request.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	}
	host := c.Request.Host
	if h := c.Request.Header.Get("X-Forwarded-Host"); h != "" {
		host = h
	}
	return scheme + "://" + host + "/api"
}

func (h *Handler) hostedObservabilityEndpoint(c *gin.Context) string {
	if base := strings.TrimSpace(h.cfg.URLs.MCPObservabilityIngestBaseURL); base != "" {
		base = strings.TrimSuffix(base, "/")
		if strings.HasSuffix(base, "/api") {
			return base + "/observability/events"
		}
		return base + "/api/observability/events"
	}
	base := h.hostedBaseURL(c)
	alias := strings.TrimSpace(h.cfg.Hosted.ObservabilityDockerHostAlias)
	if u, err := url.Parse(base); err == nil && alias != "" {
		host := strings.ToLower(u.Hostname())
		if h.cfg.ShouldRewriteObservabilityHost(host) {
			port := u.Port()
			if port != "" {
				u.Host = net.JoinHostPort(alias, port)
			} else {
				u.Host = alias
			}
			base = u.String()
		}
	}
	if strings.HasSuffix(base, "/api") {
		return base + "/observability/events"
	}
	return base + "/api/observability/events"
}

func (h *Handler) hostedObservabilityEnv(c *gin.Context, server *models.Server) map[string]string {
	env := map[string]string{}
	if server == nil {
		return env
	}
	key := strings.TrimSpace(server.ObservabilityReportingKey)
	if key == "" {
		if ensuredKey, err := h.db.EnsureServerObservabilityKey(c.Request.Context(), server.ID); err == nil {
			key = ensuredKey
		}
	}
	if key == "" {
		return env
	}
	env["MCP_OBSERVABILITY_ENDPOINT"] = h.hostedObservabilityEndpoint(c)
	env["MCP_OBSERVABILITY_KEY"] = key
	env["MCP_OBSERVABILITY_USER_ID"] = h.currentUserID(c)
	return env
}

func (h *Handler) hostedRuntimePlatformLimits() hostedruntime.PlatformLimits {
	return hostedruntime.PlatformLimitsFromYAML(h.cfg.Hosted.RuntimeIsolation)
}

// mergeHostedRuntimeEnv builds Docker env for MCP_EGRESS_* and resolves CPU/memory for the container.
func (h *Handler) mergeHostedRuntimeEnv(c *gin.Context, server *models.Server, snapshot *models.Server, raw json.RawMessage, envProfile string) (map[string]string, *hostedruntime.Resolved, error) {
	if server == nil || snapshot == nil {
		return nil, nil, fmt.Errorf("server snapshot required")
	}
	plat := h.hostedRuntimePlatformLimits()
	uc, err := hostedruntime.ParseUserConfig(raw)
	if err != nil {
		return nil, nil, err
	}
	resolved, err := hostedruntime.Resolve(uc, plat)
	if err != nil {
		return nil, nil, err
	}
	obs := h.hostedObservabilityEnv(c, server)
	var obsHosts []string
	if ep := strings.TrimSpace(obs["MCP_OBSERVABILITY_ENDPOINT"]); ep != "" {
		if oh, err := hostedruntime.HostFromURLOrHost(ep); err == nil {
			obsHosts = append(obsHosts, oh)
		}
	}
	merged := hostedruntime.MergeEgressAllowlist(
		resolved.EgressAllowlist,
		hostedruntime.CollectToolURLHosts(snapshot),
		hostedruntime.HostsFromEnvProfilesJSON(server.EnvProfiles, envProfile),
		obsHosts,
	)
	resolved.EgressAllowlist = merged
	out := map[string]string{}
	if resolved.EgressPolicy == hostedruntime.EgressDenyDefault {
		out["MCP_EGRESS_MODE"] = hostedruntime.EgressDenyDefault
		out["MCP_EGRESS_ALLOWLIST"] = strings.Join(merged, ",")
	} else {
		out["MCP_EGRESS_MODE"] = hostedruntime.EgressAllowAll
	}
	return out, resolved, nil
}

func (h *Handler) buildHostedMCPConfig(slug, endpoint string, server *models.Server, hostedAccessKey string) ([]byte, error) {
	serverConfig := map[string]interface{}{
		"url": endpoint,
	}
	headers := map[string]string{}
	authMode, _ := normalizeHostedAuthMode(server.HostedAuthMode)
	if authMode == hostedAuthModeBearerToken && strings.TrimSpace(hostedAccessKey) != "" {
		headers["Authorization"] = "Bearer " + hostedAccessKey
	}
	if server.RequireCallerIdentity {
		headers["X-Make-MCP-Caller-Id"] = "<caller-api-key>"
	}
	if len(headers) > 0 {
		serverConfig["headers"] = headers
	}

	// Cursor / MCP clients expect a static OAuth client id when DCR is unavailable (POST /register returns 501).
	// hosted_security_config.oauth_bff.client_id is the Keycloak (etc.) confidential client used by the BFF for code exchange.
	if prof, err := hostedsecurity.Resolve(server, server.HostedSecurityConfig, ""); err == nil && prof != nil {
		if prof.HostedAuthMode == hostedAuthModeOIDC && prof.OAuthBFF != nil && prof.OAuthBFF.Enabled {
			cid := strings.TrimSpace(prof.OAuthBFF.ClientID)
			if cid != "" {
				serverConfig["auth"] = map[string]interface{}{
					"CLIENT_ID": cid,
					"scopes": []string{
						"openid", "profile", "email",
						"mcp:tools", "mcp:resources",
					},
				}
			}
		}
	}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(map[string]interface{}{
		"mcpServers": map[string]interface{}{
			slug: serverConfig,
		},
	}); err != nil {
		return nil, err
	}
	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}

func (h *Handler) HostedStatus(c *gin.Context) {
	id := c.Param("id")
	server := h.requireServerOwnership(c, id)
	if server == nil {
		return
	}
	userID := h.currentUserID(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	if h.hostedMgr == nil {
		c.JSON(http.StatusOK, HostedStatusResponse{Running: false})
		return
	}

	cfg, err := h.hostedMgr.GetContainerForServer(c.Request.Context(), userID, server.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if cfg == nil {
		c.JSON(http.StatusOK, HostedStatusResponse{Running: false})
		return
	}
	resp, err := h.buildHostedStatusResponse(c, server, userID, cfg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) buildHostedStatusResponse(c *gin.Context, server *models.Server, userID string, cfg *hosted.ContainerConfig) (HostedStatusResponse, error) {
	if server == nil || cfg == nil || strings.TrimSpace(userID) == "" {
		return HostedStatusResponse{}, fmt.Errorf("server, cfg, and userID are required")
	}
	baseURL := strings.TrimSuffix(h.hostedBaseURL(c), "/")
	slug := database.ServerSlug(server.Name)
	hostedAuthMode, modeErr := normalizeHostedAuthMode(server.HostedAuthMode)
	if modeErr != nil {
		hostedAuthMode = hostedAuthModeNoAuth
	}
	hostedAccessKey, err := h.db.EnsureServerHostedAccessKey(c.Request.Context(), server.ID)
	if err != nil {
		return HostedStatusResponse{}, fmt.Errorf("failed to provision hosted access token: %w", err)
	}
	mcpConfigBytes, err := h.buildHostedMCPConfig(slug, baseURL+"/users/"+userID+"/"+slug, server, hostedAccessKey)
	if err != nil {
		return HostedStatusResponse{}, fmt.Errorf("failed to build mcp config: %w", err)
	}
	startedAt := ""
	if !cfg.StartedAt.IsZero() {
		startedAt = cfg.StartedAt.UTC().Format(time.RFC3339)
	}
	lastEnsuredAt := ""
	if !cfg.LastUsedAt.IsZero() {
		lastEnsuredAt = cfg.LastUsedAt.UTC().Format(time.RFC3339)
	}
	snapshotID := ""
	snapshotVersion := cfg.Version
	if sv, svErr := h.db.GetServerVersion(c.Request.Context(), server.ID, cfg.Version); svErr == nil && sv != nil {
		snapshotID = sv.ID
		snapshotVersion = sv.Version
	}
	manifest, _ := loadHostedManifest(userID, server.ID, cfg.Version)
	runtimeName := "docker"
	imageName := "node:20-alpine"
	var memoryMB int64 = 512
	var nanoCPUs int64 = 500_000_000
	var pidsLimit int64 = 128
	networkScope := fmt.Sprintf("%s:random-port -> 3000/tcp", strings.TrimSpace(h.cfg.Hosted.ContainerDialHost))
	idleTimeoutMinutes := cfg.IdleTimeoutMinutes
	if len(manifest) > 0 {
		var m map[string]interface{}
		if err := json.Unmarshal(manifest, &m); err == nil {
			if v, ok := m["runtime"].(string); ok && strings.TrimSpace(v) != "" {
				runtimeName = v
			}
			if v, ok := m["image"].(string); ok && strings.TrimSpace(v) != "" {
				imageName = v
			}
			if meta, ok := m["metadata"].(map[string]interface{}); ok {
				if idleTimeoutMinutes == 0 {
					switch t := meta["idle_timeout_minutes"].(type) {
					case float64:
						if t >= 0 {
							idleTimeoutMinutes = int(t)
						}
					case int:
						if t >= 0 {
							idleTimeoutMinutes = t
						}
					}
				}
				if resources, ok := meta["resources"].(map[string]interface{}); ok {
					if v, ok := resources["memory_mb"].(float64); ok && v > 0 {
						memoryMB = int64(v)
					}
					if v, ok := resources["nano_cpus"].(float64); ok && v > 0 {
						nanoCPUs = int64(v)
					}
					if v, ok := resources["pids_limit"].(float64); ok && v > 0 {
						pidsLimit = int64(v)
					}
				}
				if network, ok := meta["network"].(map[string]interface{}); ok {
					bindHost, _ := network["bind_host"].(string)
					containerPort, _ := network["container_port"].(float64)
					if bindHost != "" && containerPort > 0 {
						networkScope = fmt.Sprintf("%s:random-port -> %d/tcp", bindHost, int(containerPort))
					}
				}
			}
		}
	}
	return HostedStatusResponse{
		Running:               true,
		UserID:                userID,
		ServerID:              server.ID,
		ServerSlug:            slug,
		Version:               cfg.Version,
		SnapshotID:            snapshotID,
		SnapshotVersion:       snapshotVersion,
		StartedAt:             startedAt,
		LastEnsuredAt:         lastEnsuredAt,
		Endpoint:              baseURL + "/users/" + userID + "/" + slug,
		MCPConfig:             string(mcpConfigBytes),
		Manifest:              manifest,
		ContainerID:           cfg.ContainerID,
		HostPort:              cfg.HostPort,
		Runtime:               runtimeName,
		Image:                 imageName,
		MemoryMB:              memoryMB,
		NanoCPUs:              nanoCPUs,
		PidsLimit:             pidsLimit,
		IdleTimeoutMinutes:    idleTimeoutMinutes,
		NetworkScope:          networkScope,
		HostedAuthMode:        hostedAuthMode,
		RequireCallerIdentity: server.RequireCallerIdentity,
	}, nil
}

// ListHostedCatalog returns all running hosted endpoints that can be installed by users.
func (h *Handler) ListHostedCatalog(c *gin.Context) {
	if strings.TrimSpace(h.currentUserID(c)) == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	sessions, err := h.db.ListRunningHostedSessions(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	baseURL := strings.TrimSuffix(h.hostedBaseURL(c), "/")
	serverIDs := make([]string, 0, len(sessions))
	for _, s := range sessions {
		serverIDs = append(serverIDs, s.ServerID)
	}
	cores, cerr := h.db.GetServerCoresByIDs(c.Request.Context(), serverIDs)
	if cerr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": cerr.Error()})
		return
	}
	items := make([]HostedCatalogItem, 0, len(sessions))
	for _, s := range sessions {
		server := cores[s.ServerID]
		if server == nil {
			continue
		}
		slug := database.ServerSlug(server.Name)
		hostedAccessKey, keyErr := h.db.EnsureServerHostedAccessKey(c.Request.Context(), server.ID)
		if keyErr != nil {
			continue
		}
		mcpConfigBytes, cfgErr := h.buildHostedMCPConfig(slug, baseURL+"/users/"+s.UserID+"/"+slug, server, hostedAccessKey)
		if cfgErr != nil {
			continue
		}
		lastEnsuredAt := ""
		if !s.LastEnsuredAt.IsZero() {
			lastEnsuredAt = s.LastEnsuredAt.UTC().Format(time.RFC3339)
		}
		mode, modeErr := normalizeHostedAuthMode(server.HostedAuthMode)
		if modeErr != nil {
			mode = hostedAuthModeNoAuth
		}
		items = append(items, HostedCatalogItem{
			ServerID:              server.ID,
			ServerName:            server.Name,
			ServerSlug:            slug,
			PublisherUserID:       s.UserID,
			SnapshotVersion:       s.SnapshotVersion,
			Endpoint:              baseURL + "/users/" + s.UserID + "/" + slug,
			MCPConfig:             string(mcpConfigBytes),
			HostedAuthMode:        mode,
			RequireCallerIdentity: server.RequireCallerIdentity,
			LastEnsuredAt:         lastEnsuredAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *Handler) ListHostedCallerAPIKeys(c *gin.Context) {
	userID := strings.TrimSpace(h.currentUserID(c))
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	keys, err := h.db.ListHostedCallerAPIKeysByUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"keys": keys})
}

func (h *Handler) CreateHostedCallerAPIKey(c *gin.Context) {
	var req HostedCallerAPIKeyCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request payload"})
		return
	}
	userID := strings.TrimSpace(h.currentUserID(c))
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	req.CallerUserID = strings.TrimSpace(req.CallerUserID)
	req.TenantID = strings.TrimSpace(req.TenantID)
	// User-scoped key: caller identity defaults to current user and cannot impersonate other users.
	if req.CallerUserID == "" {
		req.CallerUserID = userID
	}
	if req.CallerUserID != userID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "caller_user_id must match current user"})
		return
	}
	var expiresAt *time.Time
	if strings.TrimSpace(req.ExpiresAt) != "" {
		t, err := time.Parse(time.RFC3339, req.ExpiresAt)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "expires_at must be RFC3339 format"})
			return
		}
		expiresUTC := t.UTC()
		if !expiresUTC.After(time.Now().UTC()) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "expires_at must be in the future"})
			return
		}
		expiresAt = &expiresUTC
	}
	createdBy := userID
	keyRecord, plainKey, err := h.db.CreateHostedCallerAPIKey(c.Request.Context(), userID, createdBy, req.CallerUserID, req.TenantID, req.Scopes, req.AllowAlias, expiresAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, HostedCallerAPIKeyCreateResponse{
		Key:    *keyRecord,
		APIKey: plainKey,
	})
}

func (h *Handler) RevokeHostedCallerAPIKey(c *gin.Context) {
	userID := strings.TrimSpace(h.currentUserID(c))
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	keyID := strings.TrimSpace(c.Param("key_id"))
	if keyID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "key_id is required"})
		return
	}
	if err := h.db.RevokeHostedCallerAPIKey(c.Request.Context(), userID, keyID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *Handler) ListHostedSessions(c *gin.Context) {
	userID := h.currentUserID(c)
	if strings.TrimSpace(userID) == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	if h.hostedMgr == nil {
		c.JSON(http.StatusOK, gin.H{"sessions": []HostedSessionListItem{}})
		return
	}
	sessions, err := h.hostedMgr.ListSessions(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	serverIDs := make([]string, 0, len(sessions))
	for _, s := range sessions {
		serverIDs = append(serverIDs, s.ServerID)
	}
	cores, cerr := h.db.GetServerCoresByIDs(c.Request.Context(), serverIDs)
	if cerr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": cerr.Error()})
		return
	}
	items := make([]HostedSessionListItem, 0, len(sessions))
	for _, s := range sessions {
		serverName := s.ServerID
		if srv := cores[s.ServerID]; srv != nil {
			serverName = srv.Name
		}
		items = append(items, HostedSessionListItem{
			HostedSession: s,
			ServerName:    serverName,
		})
	}
	c.JSON(http.StatusOK, gin.H{"sessions": items})
}

func (h *Handler) GetHostedSessionHealth(c *gin.Context) {
	serverID := c.Param("server_id")
	if h.requireServerOwnership(c, serverID) == nil {
		return
	}
	userID := h.currentUserID(c)
	if strings.TrimSpace(userID) == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	if h.hostedMgr == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "hosted manager unavailable"})
		return
	}
	s, err := h.hostedMgr.SessionHealth(c.Request.Context(), userID, serverID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if s == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}
	c.JSON(http.StatusOK, s)
}

func (h *Handler) RestartHostedSession(c *gin.Context) {
	serverID := c.Param("server_id")
	if h.requireServerOwnership(c, serverID) == nil {
		return
	}
	userID := h.currentUserID(c)
	if strings.TrimSpace(userID) == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	if h.hostedMgr == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "hosted manager unavailable"})
		return
	}
	s, err := h.hostedMgr.RestartSession(c.Request.Context(), userID, serverID)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	if s == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}
	c.JSON(http.StatusOK, s)
}

func (h *Handler) StopHostedSession(c *gin.Context) {
	serverID := c.Param("server_id")
	if h.requireServerOwnership(c, serverID) == nil {
		return
	}
	userID := h.currentUserID(c)
	if strings.TrimSpace(userID) == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	if h.hostedMgr == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "hosted manager unavailable"})
		return
	}
	s, err := h.hostedMgr.StopSession(c.Request.Context(), userID, serverID)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	if s == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}
	c.JSON(http.StatusOK, s)
}

func loadHostedManifest(userID, serverID, version string) (json.RawMessage, error) {
	if userID == "" || serverID == "" || version == "" {
		return nil, fmt.Errorf("userID, serverID, and version are required")
	}
	path := filepath.Join("generated-servers", userID, serverID, version, "manifest.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if !json.Valid(data) {
		return nil, fmt.Errorf("manifest is not valid JSON")
	}
	return json.RawMessage(data), nil
}

type hostedResolvedTarget struct {
	UserID     string
	ServerID   string
	ServerSlug string
	Version    string
	Server     *models.Server
	Snapshot   *models.Server
}

// resolveHostedTarget resolves user/slug to the latest hosted snapshot.
func (h *Handler) resolveHostedTarget(c *gin.Context) (*hostedResolvedTarget, error) {
	userID := c.Param("user_id")
	serverSlug := c.Param("server_slug")
	if userID == "" || serverSlug == "" {
		return nil, fmt.Errorf("missing hosted route params")
	}

	server, err := h.db.GetServerByOwnerAndSlug(c.Request.Context(), userID, serverSlug)
	if err != nil {
		return nil, err
	}
	if server == nil {
		return nil, nil
	}

	sv, err := h.db.GetLatestHostedServerVersion(c.Request.Context(), server.ID)
	if err != nil {
		return nil, err
	}
	if sv == nil {
		return nil, nil
	}
	var snap models.Server
	if err := json.Unmarshal(sv.Snapshot, &snap); err != nil {
		return nil, fmt.Errorf("unmarshal server snapshot: %w", err)
	}
	return &hostedResolvedTarget{
		UserID:     userID,
		ServerID:   server.ID,
		ServerSlug: serverSlug,
		Version:    sv.Version,
		Server:     server,
		Snapshot:   &snap,
	}, nil
}

func hostedAccessTokenFromRequest(c *gin.Context) string {
	if authz := strings.TrimSpace(c.GetHeader("Authorization")); authz != "" {
		parts := strings.Fields(authz)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			return strings.TrimSpace(parts[1])
		}
	}
	if key := strings.TrimSpace(c.GetHeader(hostedAccessHeader)); key != "" {
		return key
	}
	if key := strings.TrimSpace(c.GetHeader("X-MCP-API-Key")); key != "" {
		return key
	}
	return ""
}

func secureStringEquals(a, b string) bool {
	if len(a) == 0 || len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func hostedCallerIdentityFromRequest(c *gin.Context) (string, string) {
	callerID := strings.TrimSpace(c.GetHeader("X-Make-MCP-Caller-Id"))
	tenantID := strings.TrimSpace(c.GetHeader("X-Make-MCP-Tenant-Id"))
	return callerID, tenantID
}

func bearerTokenOnly(c *gin.Context) string {
	if authz := strings.TrimSpace(c.GetHeader("Authorization")); authz != "" {
		parts := strings.Fields(authz)
		if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
			return strings.TrimSpace(parts[1])
		}
	}
	return ""
}

func (h *Handler) requireMTLSPinIfNeeded(c *gin.Context, prof *hostedsecurity.ResolvedProfile) bool {
	need := prof.RequireMTLS || len(prof.TrustedCertFingerprints) > 0
	if !need {
		return true
	}
	if len(prof.TrustedCertFingerprints) == 0 {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "mTLS pin required but trusted_client_cert_sha256 is empty"})
		return false
	}
	hdr := strings.TrimSpace(c.GetHeader(hostedsecurity.HeaderClientCertSHA256))
	if !hostedsecurity.FingerprintAllowed(hdr, prof.TrustedCertFingerprints) {
		c.JSON(http.StatusForbidden, gin.H{
			"error":  "client certificate fingerprint required or not trusted",
			"header": hostedsecurity.HeaderClientCertSHA256,
		})
		return false
	}
	return true
}

func (h *Handler) requireHostedAccessBoundary(c *gin.Context, server *models.Server) bool {
	if server == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "hosted server not found"})
		return false
	}
	prof, err := hostedsecurity.Resolve(server, server.HostedSecurityConfig, c.GetHeader(hostedsecurity.HeaderEnv))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid hosted security configuration"})
		return false
	}
	if !hostedsecurity.IPAllowed(hostedsecurity.ClientIP(c.Request), prof.IPAllowlist) {
		c.JSON(http.StatusForbidden, gin.H{"error": "client IP not allowed for this hosted environment", "env": prof.EnvName})
		return false
	}
	if prof.RequireMTLS && len(prof.TrustedCertFingerprints) == 0 {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "mTLS required but no trusted certificate fingerprints configured"})
		return false
	}

	mode := prof.HostedAuthMode
	switch mode {
	case hostedAuthModeBearerToken:
		accessKey, err := h.db.EnsureServerHostedAccessKey(c.Request.Context(), server.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load hosted access policy"})
			return false
		}
		provided := hostedAccessTokenFromRequest(c)
		if provided == "" {
			c.Header("WWW-Authenticate", `Bearer realm="make-mcp", error="invalid_token", error_description="access token required for hosted endpoint", scope="mcp:invoke"`)
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":       "hosted access token required",
				"auth_header": fmt.Sprintf("Authorization: Bearer <token> or %s: <token>", hostedAccessHeader),
				"auth_mode":   hostedAuthModeBearerToken,
			})
			return false
		}
		if !secureStringEquals(accessKey, provided) {
			c.JSON(http.StatusForbidden, gin.H{"error": "invalid hosted access token"})
			return false
		}
		if !h.requireMTLSPinIfNeeded(c, prof) {
			return false
		}
	case hostedAuthModeOIDC:
		raw := bearerTokenOnly(c)
		if raw == "" {
			if prof.OAuthBFF != nil && prof.OAuthBFF.Enabled {
				u := h.oauthResourceMetadataURL(c, server.ID)
				c.Header("WWW-Authenticate", `Bearer realm="make-mcp", error="invalid_token", resource_metadata_uri="`+u+`"`)
			}
			c.JSON(http.StatusUnauthorized, gin.H{"error": "OIDC bearer JWT required", "auth_header": "Authorization: Bearer <jwt>"})
			return false
		}
		if prof.OIDC == nil || strings.TrimSpace(prof.OIDC.Issuer) == "" {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "OIDC issuer not configured in hosted_security_config"})
			return false
		}
		sub, email, err := hostedsecurity.VerifyOIDCJWT(c.Request.Context(), prof.OIDC.Issuer, prof.OIDC.Audience, prof.OIDC.JWKSURL, raw)
		if err != nil {
			hint := hostedsecurity.OIDCVerifyErrorHint(err)
			resp := gin.H{"error": "invalid OIDC token", "details": err.Error()}
			if hint != "" {
				resp["hint"] = hint
			}
			c.JSON(http.StatusUnauthorized, resp)
			return false
		}
		c.Set("verified_caller_id", sub)
		if email != "" {
			c.Set("verified_oidc_email", email)
		}
		c.Set("verified_auth_method", "oidc")
		if !h.requireMTLSPinIfNeeded(c, prof) {
			return false
		}
	case hostedAuthModeMTLS:
		if len(prof.TrustedCertFingerprints) == 0 {
			c.JSON(http.StatusForbidden, gin.H{"error": "mTLS mode requires trusted_client_cert_sha256 in hosted security config"})
			return false
		}
		if !h.requireMTLSPinIfNeeded(c, prof) {
			return false
		}
	default: // no_auth
		if !h.requireMTLSPinIfNeeded(c, prof) {
			return false
		}
	}

	if prof.RequireCallerIdentity {
		callerKey, _ := hostedCallerIdentityFromRequest(c)
		if callerKey == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":      "caller API key required",
				"hint":       "set X-Make-MCP-Caller-Id to a generated caller API key (mkc_...)",
				"header_key": "X-Make-MCP-Caller-Id",
			})
			return false
		}
		identity, err := h.db.ValidateHostedCallerAPIKey(c.Request.Context(), callerKey)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to validate caller identity"})
			return false
		}
		if identity == nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":      "invalid or expired caller API key",
				"header_key": "X-Make-MCP-Caller-Id",
			})
			return false
		}
		resolvedCallerID := identity.CallerUserID
		if identity.AllowAlias {
			alias := strings.TrimSpace(c.GetHeader("X-Make-MCP-Caller-Alias"))
			if alias != "" {
				resolvedCallerID = alias
			}
		}
		c.Set("verified_caller_id", resolvedCallerID)
		c.Set("verified_tenant_id", identity.TenantID)
		c.Set("verified_scopes", strings.Join(identity.Scopes, ","))
	}

	return true
}

func (h *Handler) proxyToHostedContainer(c *gin.Context, cfg *hosted.ContainerConfig) {
	dialHost := strings.TrimSpace(h.cfg.Hosted.ContainerDialHost)
	target, err := url.Parse("http://" + net.JoinHostPort(dialHost, cfg.HostPort))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid container target"})
		return
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.FlushInterval = -1 // immediate flush for SSE / streaming
	origDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		origDirector(req)
		req.URL.Path = "/"
		req.URL.RawPath = ""
		req.Host = target.Host
		req.Header.Set("X-Forwarded-Host", c.Request.Host)
		req.Header.Set("X-Forwarded-Uri", c.Request.URL.Path)
		if proto := c.GetHeader("X-Forwarded-Proto"); proto != "" {
			req.Header.Set("X-Forwarded-Proto", proto)
		} else if c.Request.TLS != nil {
			req.Header.Set("X-Forwarded-Proto", "https")
		} else {
			req.Header.Set("X-Forwarded-Proto", "http")
		}
		if v, ok := c.Get("verified_caller_id"); ok {
			if callerID, ok := v.(string); ok && strings.TrimSpace(callerID) != "" {
				req.Header.Set("X-Make-MCP-Caller-Id", callerID)
			}
		}
		if v, ok := c.Get("verified_tenant_id"); ok {
			if tenantID, ok := v.(string); ok && strings.TrimSpace(tenantID) != "" {
				req.Header.Set("X-Make-MCP-Tenant-Id", tenantID)
			}
		}
		if v, ok := c.Get("verified_scopes"); ok {
			if scopes, ok := v.(string); ok && strings.TrimSpace(scopes) != "" {
				req.Header.Set("X-Make-MCP-Scopes", scopes)
			}
		}
		if v, ok := c.Get("verified_oidc_email"); ok {
			if email, ok := v.(string); ok && strings.TrimSpace(email) != "" {
				req.Header.Set("X-Make-MCP-OIDC-Email", email)
			}
		}
		if v, ok := c.Get("verified_auth_method"); ok {
			if m, ok := v.(string); ok && m != "" {
				req.Header.Set("X-Make-MCP-Auth-Method", m)
			}
		}
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, req *http.Request, proxyErr error) {
		http.Error(w, "hosted proxy error: "+proxyErr.Error(), http.StatusBadGateway)
	}

	// Hint upstream proxies not to buffer SSE stream.
	c.Header("X-Accel-Buffering", "no")
	proxy.ServeHTTP(c.Writer, c.Request)
}

// HostedMCPGet proxies hosted MCP GET (JSON info or SSE stream) to the running container.
func (h *Handler) HostedMCPGet(c *gin.Context) {
	target, err := h.resolveHostedTarget(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if target == nil || h.hostedMgr == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "hosted server not found"})
		return
	}
	if !h.requireHostedAccessBoundary(c, target.Server) {
		return
	}
	observabilityEnv := h.hostedObservabilityEnv(c, target.Server)
	egressExtra, resolvedRuntime, mergeErr := h.mergeHostedRuntimeEnv(c, target.Server, target.Snapshot, target.Server.HostedRuntimeConfig, "")
	if mergeErr != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "invalid hosted runtime config", "details": mergeErr.Error()})
		return
	}
	for k, v := range egressExtra {
		observabilityEnv[k] = v
	}
	cfg, err := h.hostedMgr.EnsureContainer(c.Request.Context(), target.UserID, target.ServerID, target.Version, target.Snapshot, observabilityEnv, resolvedRuntime, -1)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to ensure hosted container", "details": err.Error()})
		return
	}
	h.proxyToHostedContainer(c, cfg)
}

// HostedMCPPost proxies hosted MCP JSON-RPC POST to the running container.
func (h *Handler) HostedMCPPost(c *gin.Context) {
	target, err := h.resolveHostedTarget(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if target == nil || h.hostedMgr == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "hosted server not found"})
		return
	}
	if !h.requireHostedAccessBoundary(c, target.Server) {
		return
	}
	observabilityEnv := h.hostedObservabilityEnv(c, target.Server)
	egressExtra, resolvedRuntime, mergeErr := h.mergeHostedRuntimeEnv(c, target.Server, target.Snapshot, target.Server.HostedRuntimeConfig, "")
	if mergeErr != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "invalid hosted runtime config", "details": mergeErr.Error()})
		return
	}
	for k, v := range egressExtra {
		observabilityEnv[k] = v
	}
	cfg, err := h.hostedMgr.EnsureContainer(c.Request.Context(), target.UserID, target.ServerID, target.Version, target.Snapshot, observabilityEnv, resolvedRuntime, -1)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to ensure hosted container", "details": err.Error()})
		return
	}
	h.proxyToHostedContainer(c, cfg)
}
