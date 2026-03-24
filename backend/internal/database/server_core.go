package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/vdparikh/make-mcp/backend/internal/models"
)

// scanServerRow loads server columns from a row (tools/resources/prompts are left empty).
func scanServerRow(row pgx.Row) (*models.Server, error) {
	var server models.Server
	var authConfig []byte
	var icon, status, latestVersion, ownerID *string
	var publishedAt *time.Time
	var isPublic *bool
	var downloads *int
	var hostedVirtual *bool

	var obsKey *string
	var hostedAccessKey *string
	var hostedAuthMode *string
	var requireCallerIdentity *bool
	var envProfiles []byte
	var hostedSecurity []byte
	var hostedRuntime []byte

	if err := row.Scan(
		&server.ID, &server.Name, &server.Description, &server.Version, &icon, &status, &publishedAt, &latestVersion, &ownerID, &isPublic, &downloads, &hostedVirtual, &authConfig, &obsKey, &hostedAccessKey, &hostedAuthMode, &requireCallerIdentity, &envProfiles, &hostedSecurity, &hostedRuntime, &server.CreatedAt, &server.UpdatedAt,
	); err != nil {
		return nil, err
	}

	if icon != nil {
		server.Icon = *icon
	} else {
		server.Icon = "bi-server"
	}
	if status != nil {
		server.Status = models.ServerStatus(*status)
	} else {
		server.Status = models.ServerStatusDraft
	}
	if publishedAt != nil {
		server.PublishedAt = publishedAt
	}
	if latestVersion != nil {
		server.LatestVersion = *latestVersion
	}
	if ownerID != nil {
		server.OwnerID = *ownerID
	}
	if isPublic != nil {
		server.IsPublic = *isPublic
	}
	if downloads != nil {
		server.Downloads = *downloads
	}
	if hostedVirtual != nil {
		server.HostedVirtual = *hostedVirtual
	}

	if authConfig != nil {
		server.AuthConfig = authConfig
	}
	if obsKey != nil {
		server.ObservabilityReportingKey = *obsKey
	}
	if hostedAccessKey != nil {
		server.HostedAccessKey = *hostedAccessKey
	}
	if hostedAuthMode != nil {
		server.HostedAuthMode = *hostedAuthMode
	}
	if requireCallerIdentity != nil {
		server.RequireCallerIdentity = *requireCallerIdentity
	}
	if envProfiles != nil {
		server.EnvProfiles = envProfiles
	}
	if hostedSecurity != nil {
		server.HostedSecurityConfig = hostedSecurity
	}
	if hostedRuntime != nil {
		server.HostedRuntimeConfig = hostedRuntime
	}

	return &server, nil
}

func (db *DB) getServerCore(ctx context.Context, id string) (*models.Server, error) {
	row := db.pool.QueryRow(ctx,
		`SELECT id, name, description, version, icon, status, published_at, latest_version, owner_id, is_public, downloads, hosted_virtual, auth_config, observability_reporting_key, hosted_access_key, hosted_auth_mode, require_caller_identity, env_profiles, hosted_security_config, hosted_runtime_config, created_at, updated_at
		 FROM servers WHERE id = $1`, id,
	)
	server, scanErr := scanServerRow(row)
	if scanErr == pgx.ErrNoRows {
		return nil, nil
	}
	if scanErr != nil {
		return nil, fmt.Errorf("querying server: %w", scanErr)
	}
	return server, nil
}
