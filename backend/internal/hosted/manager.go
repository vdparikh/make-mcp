package hosted

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	types "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/google/uuid"
	"github.com/vdparikh/make-mcp/backend/internal/database"
	"github.com/vdparikh/make-mcp/backend/internal/generator"
	"github.com/vdparikh/make-mcp/backend/internal/models"
)

// ContainerConfig holds runtime info for a hosted MCP container.
type ContainerConfig struct {
	ContainerID string
	HostPort    string
	Version     string
	StartedAt   time.Time
	LastUsedAt  time.Time
}

// HostedManifest captures explicit runtime/deployment metadata for a hosted snapshot.
type HostedManifest struct {
	Name          string                 `json:"name"`
	Snapshot      string                 `json:"snapshot_version"`
	ServerVersion string                 `json:"server_version"`
	Runtime       string                 `json:"runtime"`
	Image         string                 `json:"image"`
	Endpoint      string                 `json:"endpoint"`
	Tools         []HostedManifestTool   `json:"tools,omitempty"`
	Resources     []HostedManifestRef    `json:"resources,omitempty"`
	Prompts       []HostedManifestRef    `json:"prompts,omitempty"`
	Auth          json.RawMessage        `json:"auth,omitempty"`
	Observability bool                   `json:"observability"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

type HostedManifestTool struct {
	Name            string          `json:"name"`
	Description     string          `json:"description,omitempty"`
	ExecutionType   string          `json:"execution_type,omitempty"`
	InputSchema     json.RawMessage `json:"input_schema,omitempty"`
	OutputSchema    json.RawMessage `json:"output_schema,omitempty"`
	ReadOnlyHint    bool            `json:"read_only_hint,omitempty"`
	DestructiveHint bool            `json:"destructive_hint,omitempty"`
}

type HostedManifestRef struct {
	Name string `json:"name"`
}

const (
	labelManaged = "make-mcp.managed"
	labelUserID  = "make-mcp.user-id"
	labelServer  = "make-mcp.server-id"
	labelVersion = "make-mcp.version"
)

// Manager tracks one container per (user, server).
type Manager struct {
	mu        sync.Mutex
	cli       *client.Client
	containers map[string]*ContainerConfig // key: userID + ":" + serverID
	gen       *generator.Generator
	db        *database.DB
}

// NewManager creates a new hosted container manager using environment-based Docker config.
// If DOCKER_HOST is not set and we're on macOS, uses Rancher Desktop (~/.rd/docker.sock)
// or Docker Desktop (~/.docker/run/docker.sock) so the backend can connect.
func NewManager(db *database.DB) (*Manager, error) {
	opts := []client.Opt{client.WithAPIVersionNegotiation()}
	dockerHost := resolveDockerHost()
	if dockerHost != "" {
		opts = append(opts, client.WithHost(dockerHost))
	} else {
		opts = append(opts, client.FromEnv)
	}
	cli, err := client.NewClientWithOpts(opts...)
	if err != nil {
		return nil, fmt.Errorf("creating docker client: %w", err)
	}
	return &Manager{
		cli:        cli,
		containers: make(map[string]*ContainerConfig),
		gen:        generator.NewGenerator(),
		db:         db,
	}, nil
}

func resolveDockerHost() string {
	envHost := strings.TrimSpace(os.Getenv("DOCKER_HOST"))
	if runtime.GOOS != "darwin" {
		return envHost
	}

	// On macOS, /var/run/docker.sock is often wrong for Rancher Desktop.
	// If env points there (or is empty), prefer user-scoped socket paths.
	if envHost != "" && envHost != "unix:///var/run/docker.sock" {
		return envHost
	}

	home := os.Getenv("HOME")
	if home == "" {
		if u, err := user.Current(); err == nil {
			home = u.HomeDir
		}
	}
	if home == "" {
		return envHost
	}

	rdSock := filepath.Join(home, ".rd", "docker.sock")
	if _, err := os.Stat(rdSock); err == nil {
		return "unix://" + rdSock
	}

	dockerDesktopSock := filepath.Join(home, ".docker", "run", "docker.sock")
	if _, err := os.Stat(dockerDesktopSock); err == nil {
		return "unix://" + dockerDesktopSock
	}

	// Default to Rancher Desktop socket path even if it is not created yet.
	return "unix://" + rdSock
}

// key builds the in-memory key for a (user, server) pair.
func key(userID, serverID string) string {
	return userID + ":" + serverID
}

// EnsureContainer ensures there is a running container for the given user/server.
// It returns the ContainerConfig for the running container.
func (m *Manager) EnsureContainer(ctx context.Context, userID string, serverID string, version string, snapshot *models.Server, envVars map[string]string) (*ContainerConfig, error) {
	if userID == "" || snapshot == nil {
		return nil, fmt.Errorf("userID and snapshot are required")
	}
	if serverID == "" || version == "" {
		return nil, fmt.Errorf("serverID and version are required")
	}

	k := key(userID, serverID)
	if cfg, err := m.reconcileContainers(ctx, userID, serverID, version, envVars); err == nil && cfg != nil {
		cfg.LastUsedAt = time.Now()
		m.mu.Lock()
		m.containers[k] = cfg
		m.mu.Unlock()
		_ = m.upsertSession(ctx, userID, serverID, *cfg, "running", "unknown", "", nil)
		return cfg, nil
	} else if err != nil {
		return nil, err
	}

	// Generate server code for the published snapshot version.
	gen, err := m.gen.Generate(snapshot)
	if err != nil {
		return nil, fmt.Errorf("generate server: %w", err)
	}

	rootDir := filepath.Join("generated-servers", userID, serverID, version)
	if err := writeGeneratedServer(rootDir, gen); err != nil {
		return nil, fmt.Errorf("write generated server: %w", err)
	}
	if err := writeHostedManifest(rootDir, snapshot, version, userID, serverID, envVars); err != nil {
		return nil, fmt.Errorf("write hosted manifest: %w", err)
	}

	// Start container for this generated server.
	cfg, err := m.startContainer(ctx, rootDir, userID, serverID, version, envVars)
	if err != nil {
		return nil, err
	}

	m.mu.Lock()
	m.containers[k] = cfg
	m.mu.Unlock()
	_ = m.upsertSession(ctx, userID, serverID, *cfg, "running", "unknown", "", nil)
	return cfg, nil
}

// startContainer creates and starts a Docker container for the generated server folder.
func (m *Manager) startContainer(ctx context.Context, rootDir string, userID string, serverID string, version string, envVars map[string]string) (*ContainerConfig, error) {
	// Run generated server directly in official Node image so hosted publish
	// does not depend on a prebuilt local runner image.
	image := "node:20-alpine"
	absRootDir, err := filepath.Abs(rootDir)
	if err != nil {
		return nil, fmt.Errorf("resolve generated server path: %w", err)
	}

	hostPort := randomPort()
	pidsLimit := int64(128)

	env := []string{}
	for k, v := range envVars {
		if k == "" {
			continue
		}
		env = append(env, k+"="+v)
	}

	config := &container.Config{
		Image:      image,
		WorkingDir: "/app",
		User:       "node",
		OpenStdin:  true, // Keep stdio MCP process alive in detached containers.
		StdinOnce:  false,
		Env:        env,
		Labels: map[string]string{
			labelManaged: "true",
			labelUserID:  userID,
			labelServer:  serverID,
			labelVersion: version,
		},
		Cmd: []string{
			"sh",
			"-lc",
			"npm install && npm run build && MCP_TRANSPORT=http MCP_HTTP_PORT=3000 node dist/server.js",
		},
		ExposedPorts: nat.PortSet{
			"3000/tcp": struct{}{},
		},
	}

	hostConfig := &container.HostConfig{
		Privileged: false,
		CapDrop:    []string{"ALL"},
		SecurityOpt: []string{
			"no-new-privileges:true",
		},
		Resources: container.Resources{
			// Conservative defaults; can be tuned.
			Memory:    512 * 1024 * 1024, // 512MB
			NanoCPUs:  500_000_000,       // 0.5 CPU
			PidsLimit: &pidsLimit,
		},
		PortBindings: nat.PortMap{
			"3000/tcp": {{
				HostIP:   "127.0.0.1",
				HostPort: hostPort,
			}},
		},
		Binds: []string{
			filepath.Clean(absRootDir) + ":/app",
		},
	}

	networkingConfig := &network.NetworkingConfig{}

	name := "mcp-hosted-" + uuid.New().String()
	resp, err := m.cli.ContainerCreate(ctx, config, hostConfig, networkingConfig, nil, name)
	if err != nil {
		return nil, fmt.Errorf("container create (docker host %s): %w", m.cli.DaemonHost(), err)
	}

	if err := m.cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return nil, fmt.Errorf("container start: %w", err)
	}

	return &ContainerConfig{
		ContainerID: resp.ID,
		HostPort:    hostPort,
		Version:     version,
		StartedAt:   time.Now(),
		LastUsedAt:  time.Now(),
	}, nil
}

func (m *Manager) GetContainerForServer(ctx context.Context, userID string, serverID string) (*ContainerConfig, error) {
	args := filters.NewArgs()
	args.Add("label", labelManaged+"=true")
	args.Add("label", labelUserID+"="+userID)
	args.Add("label", labelServer+"="+serverID)
	args.Add("status", "running")
	list, err := m.cli.ContainerList(ctx, container.ListOptions{All: false, Filters: args})
	if err != nil {
		return nil, fmt.Errorf("list hosted containers: %w", err)
	}
	if len(list) == 0 {
		return nil, nil
	}
	c := list[0]
	startedAt, _ := m.containerStartedAt(ctx, c.ID)
	lastEnsured := time.Time{}
	m.mu.Lock()
	if cfg, ok := m.containers[key(userID, serverID)]; ok && cfg.ContainerID == c.ID {
		lastEnsured = cfg.LastUsedAt
	}
	m.mu.Unlock()
	return &ContainerConfig{
		ContainerID: c.ID,
		HostPort:    hostPortFromContainer(c),
		Version:     c.Labels[labelVersion],
		StartedAt:   startedAt,
		LastUsedAt:  lastEnsured,
	}, nil
}

func (m *Manager) ListSessions(ctx context.Context, userID string) ([]models.HostedSession, error) {
	if m.db == nil {
		return nil, fmt.Errorf("database is required")
	}
	sessions, err := m.db.ListHostedSessions(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]models.HostedSession, 0, len(sessions))
	for _, s := range sessions {
		updated := s
		if strings.TrimSpace(s.ContainerID) == "" {
			out = append(out, updated)
			continue
		}
		inspect, inspectErr := m.cli.ContainerInspect(ctx, s.ContainerID)
		now := time.Now().UTC()
		if inspectErr != nil {
			updated.Status = "stopped"
			updated.Health = "unknown"
			updated.StoppedAt = &now
			updated.LastError = inspectErr.Error()
		} else {
			updated.Status = "stopped"
			if inspect.State != nil {
				if inspect.State.Running {
					updated.Status = "running"
				}
				if strings.TrimSpace(inspect.State.StartedAt) != "" {
					if t, err := time.Parse(time.RFC3339Nano, inspect.State.StartedAt); err == nil {
						updated.StartedAt = &t
					}
				}
				if inspect.State.Health != nil && strings.TrimSpace(inspect.State.Health.Status) != "" {
					updated.Health = inspect.State.Health.Status
				} else {
					updated.Health = "unknown"
				}
			}
			updated.LastError = ""
			if updated.Status == "stopped" {
				updated.StoppedAt = &now
			}
		}
		if _, err := m.db.UpsertHostedSession(ctx, updated); err != nil {
			return nil, err
		}
		out = append(out, updated)
	}
	return out, nil
}

func (m *Manager) StopSession(ctx context.Context, userID, serverID string) (*models.HostedSession, error) {
	if m.db == nil {
		return nil, fmt.Errorf("database is required")
	}
	s, err := m.db.GetHostedSession(ctx, userID, serverID)
	if err != nil {
		return nil, err
	}
	if s == nil {
		return nil, nil
	}
	now := time.Now().UTC()
	if strings.TrimSpace(s.ContainerID) != "" {
		timeout := 10
		if err := m.cli.ContainerStop(ctx, s.ContainerID, container.StopOptions{Timeout: &timeout}); err != nil {
			s.Status = "error"
			s.LastError = fmt.Sprintf("stop container: %v", err)
			s.StoppedAt = &now
			updated, upsertErr := m.db.UpsertHostedSession(ctx, *s)
			if upsertErr != nil {
				return nil, upsertErr
			}
			return updated, nil
		}
	}
	s.Status = "stopped"
	s.Health = "unknown"
	s.LastError = ""
	s.StoppedAt = &now
	updated, err := m.db.UpsertHostedSession(ctx, *s)
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func (m *Manager) RestartSession(ctx context.Context, userID, serverID string) (*models.HostedSession, error) {
	if m.db == nil {
		return nil, fmt.Errorf("database is required")
	}
	s, err := m.db.GetHostedSession(ctx, userID, serverID)
	if err != nil {
		return nil, err
	}
	if s == nil {
		return nil, nil
	}
	if strings.TrimSpace(s.ContainerID) == "" {
		return nil, fmt.Errorf("session has no container id")
	}
	timeout := 10
	now := time.Now().UTC()
	if err := m.cli.ContainerRestart(ctx, s.ContainerID, container.StopOptions{Timeout: &timeout}); err != nil {
		s.Status = "error"
		s.LastError = fmt.Sprintf("restart container: %v", err)
		if _, upsertErr := m.db.UpsertHostedSession(ctx, *s); upsertErr != nil {
			return nil, upsertErr
		}
		return s, nil
	}
	s.Status = "running"
	s.Health = "unknown"
	s.LastError = ""
	s.LastEnsuredAt = &now
	s.LastUsedAt = &now
	s.StartedAt = &now
	s.StoppedAt = nil
	updated, err := m.db.UpsertHostedSession(ctx, *s)
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func (m *Manager) SessionHealth(ctx context.Context, userID, serverID string) (*models.HostedSession, error) {
	if m.db == nil {
		return nil, fmt.Errorf("database is required")
	}
	s, err := m.db.GetHostedSession(ctx, userID, serverID)
	if err != nil {
		return nil, err
	}
	if s == nil {
		return nil, nil
	}
	if strings.TrimSpace(s.ContainerID) == "" {
		s.Health = "unknown"
		s.Status = "stopped"
		return m.db.UpsertHostedSession(ctx, *s)
	}
	inspect, err := m.cli.ContainerInspect(ctx, s.ContainerID)
	now := time.Now().UTC()
	if err != nil {
		s.Status = "stopped"
		s.Health = "unknown"
		s.StoppedAt = &now
		s.LastError = err.Error()
		return m.db.UpsertHostedSession(ctx, *s)
	}
	s.LastError = ""
	if inspect.State != nil && inspect.State.Running {
		s.Status = "running"
		s.StoppedAt = nil
	} else {
		s.Status = "stopped"
		s.StoppedAt = &now
	}
	if inspect.State != nil && inspect.State.Health != nil && strings.TrimSpace(inspect.State.Health.Status) != "" {
		s.Health = inspect.State.Health.Status
	} else {
		s.Health = "unknown"
	}
	return m.db.UpsertHostedSession(ctx, *s)
}

func (m *Manager) reconcileContainers(ctx context.Context, userID string, serverID string, version string, envVars map[string]string) (*ContainerConfig, error) {
	args := filters.NewArgs()
	args.Add("label", labelManaged+"=true")
	args.Add("label", labelUserID+"="+userID)
	args.Add("label", labelServer+"="+serverID)

	list, err := m.cli.ContainerList(ctx, container.ListOptions{All: true, Filters: args})
	if err != nil {
		return nil, fmt.Errorf("list existing hosted containers: %w", err)
	}

	var keep *ContainerConfig
	for _, c := range list {
		isRunning := strings.EqualFold(c.State, "running")
		ver := c.Labels[labelVersion]
		if keep == nil && isRunning && ver == version {
			cfg := &ContainerConfig{
				ContainerID: c.ID,
				HostPort:    hostPortFromContainer(c),
				Version:     ver,
				StartedAt:   time.Now(),
				LastUsedAt:  time.Now(),
			}
			matches, matchErr := m.containerEnvMatches(ctx, c.ID, envVars)
			if matchErr == nil && matches {
				keep = cfg
				continue
			}
			_ = m.stopAndRemoveContainer(ctx, c.ID)
			continue
		}
		_ = m.stopAndRemoveContainer(ctx, c.ID)
	}

	// Cleanup legacy hosted containers created before labels, matching this user/server mount path.
	legacy, err := m.cli.ContainerList(ctx, container.ListOptions{All: true, Filters: filters.NewArgs()})
	if err == nil {
		marker := string(filepath.Separator) + filepath.Join("generated-servers", userID, serverID) + string(filepath.Separator)
		for _, c := range legacy {
			if c.Labels[labelManaged] == "true" {
				continue
			}
			if !hasHostedName(c.Names) {
				continue
			}
			inspect, inspectErr := m.cli.ContainerInspect(ctx, c.ID)
			if inspectErr != nil {
				continue
			}
			shouldRemove := false
			for _, mount := range inspect.Mounts {
				source := filepath.Clean(mount.Source)
				if strings.Contains(source, marker) || strings.HasSuffix(source, filepath.Join("generated-servers", userID, serverID)) {
					shouldRemove = true
					break
				}
			}
			if shouldRemove {
				_ = m.stopAndRemoveContainer(ctx, c.ID)
			}
		}
	}

	return keep, nil
}

func (m *Manager) containerEnvMatches(ctx context.Context, containerID string, expected map[string]string) (bool, error) {
	if len(expected) == 0 {
		return true, nil
	}
	inspect, err := m.cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return false, err
	}
	envMap := make(map[string]string, len(inspect.Config.Env))
	for _, item := range inspect.Config.Env {
		parts := strings.SplitN(item, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}
	for k, v := range expected {
		if envMap[k] != v {
			return false, nil
		}
	}
	return true, nil
}

func (m *Manager) containerStartedAt(ctx context.Context, containerID string) (time.Time, error) {
	inspect, err := m.cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return time.Time{}, err
	}
	if inspect.State == nil || strings.TrimSpace(inspect.State.StartedAt) == "" {
		return time.Time{}, nil
	}
	t, err := time.Parse(time.RFC3339Nano, inspect.State.StartedAt)
	if err != nil {
		return time.Time{}, err
	}
	return t, nil
}

func (m *Manager) isContainerRunning(ctx context.Context, containerID string) (bool, error) {
	inspect, err := m.cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return false, err
	}
	return inspect.State != nil && inspect.State.Running, nil
}

func (m *Manager) stopAndRemoveContainer(ctx context.Context, containerID string) error {
	timeout := 5
	_ = m.cli.ContainerStop(ctx, containerID, container.StopOptions{Timeout: &timeout})
	if err := m.cli.ContainerRemove(ctx, containerID, container.RemoveOptions{Force: true}); err != nil {
		return fmt.Errorf("remove container %s: %w", containerID, err)
	}
	return nil
}

func (m *Manager) upsertSession(ctx context.Context, userID, serverID string, cfg ContainerConfig, status, health, lastError string, stoppedAt *time.Time) error {
	if m.db == nil {
		return nil
	}
	now := time.Now().UTC()
	s := models.HostedSession{
		UserID:          userID,
		ServerID:        serverID,
		SnapshotVersion: cfg.Version,
		ContainerID:     cfg.ContainerID,
		HostPort:        cfg.HostPort,
		Status:          status,
		Health:          health,
		LastUsedAt:      &cfg.LastUsedAt,
		LastEnsuredAt:   &now,
		StartedAt:       &cfg.StartedAt,
		StoppedAt:       stoppedAt,
		LastError:       lastError,
	}
	_, err := m.db.UpsertHostedSession(ctx, s)
	return err
}

// writeGeneratedServer writes generated files to disk.
func writeGeneratedServer(rootDir string, gen *generator.GeneratedServer) error {
	for path, data := range gen.Files {
		full := filepath.Join(rootDir, filepath.FromSlash(path))
		if err := ensureDir(filepath.Dir(full)); err != nil {
			return err
		}
		if err := osWriteFile(full, data); err != nil {
			return err
		}
	}
	return nil
}

func writeHostedManifest(rootDir string, snapshot *models.Server, snapshotVersion, userID, serverID string, envVars map[string]string) error {
	if snapshot == nil {
		return fmt.Errorf("snapshot is required")
	}
	tools := make([]HostedManifestTool, 0, len(snapshot.Tools))
	for _, t := range snapshot.Tools {
		tools = append(tools, HostedManifestTool{
			Name:            t.Name,
			Description:     t.Description,
			ExecutionType:   string(t.ExecutionType),
			InputSchema:     t.InputSchema,
			OutputSchema:    t.OutputSchema,
			ReadOnlyHint:    t.ReadOnlyHint,
			DestructiveHint: t.DestructiveHint,
		})
	}
	resources := make([]HostedManifestRef, 0, len(snapshot.Resources))
	for _, r := range snapshot.Resources {
		resources = append(resources, HostedManifestRef{Name: r.Name})
	}
	prompts := make([]HostedManifestRef, 0, len(snapshot.Prompts))
	for _, p := range snapshot.Prompts {
		prompts = append(prompts, HostedManifestRef{Name: p.Name})
	}
	manifest := HostedManifest{
		Name:          snapshot.Name,
		Snapshot:      snapshotVersion,
		ServerVersion: snapshot.Version,
		Runtime:       "docker",
		Image:         "node:20-alpine",
		Endpoint:      fmt.Sprintf("/api/users/%s/%s", userID, databaseSafeSlug(snapshot.Name)),
		Tools:         tools,
		Resources:     resources,
		Prompts:       prompts,
		Auth:          snapshot.AuthConfig,
		Observability: strings.TrimSpace(envVars["MCP_OBSERVABILITY_KEY"]) != "",
		Metadata: map[string]interface{}{
			"user_id":   userID,
			"server_id": serverID,
		},
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}
	return osWriteFile(filepath.Join(rootDir, "manifest.json"), data)
}

func databaseSafeSlug(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = strings.ReplaceAll(s, " ", "-")
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		}
	}
	return strings.Trim(b.String(), "-")
}

func ensureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}

func osWriteFile(name string, data []byte) error {
	return os.WriteFile(name, data, 0o644)
}

// randomPort returns a string port in the 40000-50000 range.
func randomPort() string {
	return fmt.Sprintf("%d", 40000+time.Now().UnixNano()%10000)
}

func hostPortFromContainer(c types.Container) string {
	for _, p := range c.Ports {
		if p.PrivatePort == 3000 && p.PublicPort > 0 {
			return strconv.Itoa(int(p.PublicPort))
		}
	}
	return ""
}

func hasHostedName(names []string) bool {
	for _, n := range names {
		if strings.Contains(n, "mcp-hosted-") {
			return true
		}
	}
	return false
}

