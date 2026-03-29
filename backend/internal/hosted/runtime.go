package hosted

import (
	"context"
	"strings"

	"github.com/vdparikh/make-mcp/backend/internal/hostedruntime"
	"github.com/vdparikh/make-mcp/backend/internal/models"
)

// Runtime abstracts Docker- or Kubernetes-backed hosted MCP processes.
type Runtime interface {
	EnsureContainer(ctx context.Context, userID string, serverID string, version string, snapshot *models.Server, envVars map[string]string, rt *hostedruntime.Resolved, idleTimeoutMinutes int) (*ContainerConfig, error)
	GetContainerForServer(ctx context.Context, userID string, serverID string) (*ContainerConfig, error)
	ListSessions(ctx context.Context, userID string) ([]models.HostedSession, error)
	StopSession(ctx context.Context, userID string, serverID string) (*models.HostedSession, error)
	RestartSession(ctx context.Context, userID string, serverID string) (*models.HostedSession, error)
	SessionHealth(ctx context.Context, userID string, serverID string) (*models.HostedSession, error)
}

// DialHost returns the host to use when the API calls the hosted runtime (overrides config when set).
func DialHost(cfg *ContainerConfig, fallback string) string {
	if cfg == nil {
		return fallback
	}
	if h := strings.TrimSpace(cfg.DialHost); h != "" {
		return h
	}
	return fallback
}
