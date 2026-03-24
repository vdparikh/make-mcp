package database

import (
	"context"
	"fmt"
	"strings"

	"github.com/vdparikh/make-mcp/backend/internal/models"
)

// RotateServerHostedAccessKey replaces the hosted access key and returns the new secret (show once to the user).
func (db *DB) RotateServerHostedAccessKey(ctx context.Context, serverID string) (string, error) {
	key, err := generateHostedAccessKey()
	if err != nil {
		return "", err
	}
	_, err = db.pool.Exec(ctx, `UPDATE servers SET hosted_access_key = $1, updated_at = NOW() WHERE id = $2`, key, serverID)
	if err != nil {
		return "", err
	}
	return key, nil
}

// UpdateServerHostedSecurityConfig stores JSON for per-environment IP/OIDC/mTLS (see docs/hosted-security.md).
func (db *DB) UpdateServerHostedSecurityConfig(ctx context.Context, serverID string, cfg []byte) error {
	_, err := db.pool.Exec(ctx, `UPDATE servers SET hosted_security_config = $2, updated_at = NOW() WHERE id = $1`, serverID, cfg)
	return err
}

// UpdateServerHostedRuntimeConfig stores JSON for isolation tier, resources, and egress policy.
func (db *DB) UpdateServerHostedRuntimeConfig(ctx context.Context, serverID string, cfg []byte) error {
	_, err := db.pool.Exec(ctx, `UPDATE servers SET hosted_runtime_config = $2, updated_at = NOW() WHERE id = $1`, serverID, cfg)
	return err
}

// InsertHostedSecurityAudit records a security-relevant action (rotation, config update).
func (db *DB) InsertHostedSecurityAudit(ctx context.Context, serverID, actorUserID, action, resourceType, resourceID string, metadata []byte) error {
	if len(metadata) == 0 {
		metadata = []byte("{}")
	}
	var actor any
	if strings.TrimSpace(actorUserID) != "" {
		actor = actorUserID
	} else {
		actor = nil
	}
	rt := strings.TrimSpace(resourceType)
	rid := strings.TrimSpace(resourceID)
	var rtArg any
	var ridArg any
	if rt != "" {
		rtArg = rt
	} else {
		rtArg = nil
	}
	if rid != "" {
		ridArg = rid
	} else {
		ridArg = nil
	}
	_, err := db.pool.Exec(ctx, `
		INSERT INTO hosted_security_audit (server_id, actor_user_id, action, resource_type, resource_id, metadata)
		VALUES ($1, $2, $3, $4, $5, $6::jsonb)`,
		serverID, actor, action, rtArg, ridArg, metadata)
	return err
}

// ListHostedSecurityAudit returns recent audit rows for a server (owner-scoped in query).
func (db *DB) ListHostedSecurityAudit(ctx context.Context, serverID, ownerUserID string, limit int) ([]models.HostedSecurityAuditEvent, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := db.pool.Query(ctx, `
		SELECT a.id, a.server_id, COALESCE(a.actor_user_id::text, ''), a.action, COALESCE(a.resource_type, ''), COALESCE(a.resource_id, ''), COALESCE(a.metadata::text, '{}'), a.created_at
		FROM hosted_security_audit a
		INNER JOIN servers s ON s.id = a.server_id
		WHERE a.server_id = $1 AND s.owner_id::text = $2
		ORDER BY a.created_at DESC
		LIMIT $3`, serverID, ownerUserID, limit)
	if err != nil {
		return nil, fmt.Errorf("hosted security audit: %w", err)
	}
	defer rows.Close()
	var out []models.HostedSecurityAuditEvent
	for rows.Next() {
		var e models.HostedSecurityAuditEvent
		var metaStr string
		if err := rows.Scan(&e.ID, &e.ServerID, &e.ActorUserID, &e.Action, &e.ResourceType, &e.ResourceID, &metaStr, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning audit: %w", err)
		}
		e.Metadata = []byte(metaStr)
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
