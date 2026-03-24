package database

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vdparikh/make-mcp/backend/internal/mcpvalidate"
	"github.com/vdparikh/make-mcp/backend/internal/models"
)

type DB struct {
	pool *pgxpool.Pool
}

func New(connString string) (*DB, error) {
	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("parsing connection string: %w", err)
	}

	config.MaxConns = 25
	config.MinConns = 5
	config.MaxConnLifetime = time.Hour
	config.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("creating pool: %w", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	return &DB{pool: pool}, nil
}

func (db *DB) Close() {
	db.pool.Close()
}

// nullIfEmpty returns nil for empty string so NULL can be stored in nullable columns.
func nullIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// nullableJSON returns nil for empty RawMessage so PostgreSQL stores NULL.
func nullableJSON(raw json.RawMessage) interface{} {
	if len(raw) == 0 {
		return nil
	}
	return raw
}

func (db *DB) RunMigrations(ctx context.Context) error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS servers (
			id UUID PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			description TEXT,
			version VARCHAR(50) DEFAULT '1.0.0',
			auth_config JSONB,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS tools (
			id UUID PRIMARY KEY,
			server_id UUID REFERENCES servers(id) ON DELETE CASCADE,
			name VARCHAR(255) NOT NULL,
			description TEXT,
			input_schema JSONB,
			output_schema JSONB,
			execution_type VARCHAR(50) NOT NULL,
			execution_config JSONB,
			context_fields TEXT[],
			output_display VARCHAR(20) DEFAULT 'json',
			read_only_hint BOOLEAN DEFAULT false,
			destructive_hint BOOLEAN DEFAULT false,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,
		`ALTER TABLE tools ADD COLUMN IF NOT EXISTS output_display VARCHAR(20) DEFAULT 'json'`,
		`ALTER TABLE tools ADD COLUMN IF NOT EXISTS read_only_hint BOOLEAN DEFAULT false`,
		`ALTER TABLE tools ADD COLUMN IF NOT EXISTS destructive_hint BOOLEAN DEFAULT false`,
		`ALTER TABLE tools ADD COLUMN IF NOT EXISTS output_display_config JSONB`,
		`CREATE TABLE IF NOT EXISTS resources (
			id UUID PRIMARY KEY,
			server_id UUID REFERENCES servers(id) ON DELETE CASCADE,
			name VARCHAR(255) NOT NULL,
			uri VARCHAR(512) NOT NULL,
			mime_type VARCHAR(100),
			handler JSONB,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS prompts (
			id UUID PRIMARY KEY,
			server_id UUID REFERENCES servers(id) ON DELETE CASCADE,
			name VARCHAR(255) NOT NULL,
			description TEXT,
			template TEXT NOT NULL,
			arguments JSONB,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS context_configs (
			id UUID PRIMARY KEY,
			server_id UUID REFERENCES servers(id) ON DELETE CASCADE,
			name VARCHAR(255) NOT NULL,
			source_type VARCHAR(50) NOT NULL,
			config JSONB,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS policies (
			id UUID PRIMARY KEY,
			tool_id UUID REFERENCES tools(id) ON DELETE CASCADE,
			name VARCHAR(255) NOT NULL,
			description TEXT,
			enabled BOOLEAN DEFAULT true,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS policy_rules (
			id UUID PRIMARY KEY,
			policy_id UUID REFERENCES policies(id) ON DELETE CASCADE,
			type VARCHAR(50) NOT NULL,
			config JSONB,
			priority INTEGER DEFAULT 0,
			fail_action VARCHAR(50) DEFAULT 'deny'
		)`,
		`CREATE TABLE IF NOT EXISTS tool_executions (
			id UUID PRIMARY KEY,
			tool_id UUID REFERENCES tools(id) ON DELETE CASCADE,
			server_id UUID REFERENCES servers(id) ON DELETE CASCADE,
			input JSONB,
			output JSONB,
			error TEXT,
			status_code INTEGER,
			duration_ms BIGINT,
			success BOOLEAN,
			healing_applied BOOLEAN DEFAULT false,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS healing_suggestions (
			id UUID PRIMARY KEY,
			tool_id UUID REFERENCES tools(id) ON DELETE CASCADE,
			error_pattern TEXT,
			suggestion_type VARCHAR(100),
			suggestion JSONB,
			auto_apply BOOLEAN DEFAULT false,
			applied BOOLEAN DEFAULT false,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS server_compositions (
			id UUID PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			description TEXT,
			server_ids UUID[],
			owner_id UUID,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,
		`ALTER TABLE server_compositions ADD COLUMN IF NOT EXISTS owner_id UUID`,
		`CREATE TABLE IF NOT EXISTS flows (
			id UUID PRIMARY KEY,
			server_id UUID REFERENCES servers(id) ON DELETE CASCADE,
			name VARCHAR(255) NOT NULL,
			description TEXT,
			nodes JSONB NOT NULL,
			edges JSONB NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_tools_server_id ON tools(server_id)`,
		`CREATE INDEX IF NOT EXISTS idx_tools_server_id_name ON tools(server_id, name)`,
		`CREATE INDEX IF NOT EXISTS idx_resources_server_id ON resources(server_id)`,
		`CREATE INDEX IF NOT EXISTS idx_prompts_server_id ON prompts(server_id)`,
		`CREATE INDEX IF NOT EXISTS idx_policies_tool_id ON policies(tool_id)`,
		`CREATE INDEX IF NOT EXISTS idx_policy_rules_policy_id_priority ON policy_rules(policy_id, priority)`,
		`CREATE INDEX IF NOT EXISTS idx_tool_executions_tool_id ON tool_executions(tool_id)`,
		`CREATE INDEX IF NOT EXISTS idx_tool_executions_created_at ON tool_executions(created_at)`,
		`ALTER TABLE tool_executions ADD COLUMN IF NOT EXISTS source VARCHAR(20) DEFAULT 'playground'`,
		`ALTER TABLE tool_executions ADD COLUMN IF NOT EXISTS tool_name VARCHAR(255)`,
		`ALTER TABLE tool_executions ADD COLUMN IF NOT EXISTS repair_suggestion TEXT`,
		`CREATE INDEX IF NOT EXISTS idx_tool_executions_server_source ON tool_executions(server_id, source)`,
		`ALTER TABLE servers ADD COLUMN IF NOT EXISTS observability_reporting_key VARCHAR(64) UNIQUE`,
		`ALTER TABLE tool_executions ADD COLUMN IF NOT EXISTS client_user_id VARCHAR(255)`,
		`ALTER TABLE tool_executions ADD COLUMN IF NOT EXISTS client_agent VARCHAR(100)`,
		`ALTER TABLE tool_executions ADD COLUMN IF NOT EXISTS client_token VARCHAR(512)`,
		`CREATE INDEX IF NOT EXISTS idx_flows_server_id ON flows(server_id)`,
		`CREATE INDEX IF NOT EXISTS idx_context_configs_server_id ON context_configs(server_id)`,
		`ALTER TABLE servers ADD COLUMN IF NOT EXISTS icon VARCHAR(100)`,
		`CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY,
			email VARCHAR(255) UNIQUE NOT NULL,
			name VARCHAR(255) NOT NULL,
			password_hash VARCHAR(255),
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_users_email ON users(email)`,
		`DO $$ BEGIN IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = 'users' AND column_name = 'password_hash' AND is_nullable = 'NO') THEN ALTER TABLE users ALTER COLUMN password_hash DROP NOT NULL; END IF; END $$`,
		`CREATE TABLE IF NOT EXISTS webauthn_credentials (
			id UUID PRIMARY KEY,
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			credential_id BYTEA NOT NULL,
			data JSONB NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			UNIQUE(credential_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_webauthn_credentials_user_id ON webauthn_credentials(user_id)`,
		`ALTER TABLE servers ADD COLUMN IF NOT EXISTS status VARCHAR(50) DEFAULT 'draft'`,
		`ALTER TABLE servers ADD COLUMN IF NOT EXISTS published_at TIMESTAMP WITH TIME ZONE`,
		`ALTER TABLE servers ADD COLUMN IF NOT EXISTS latest_version VARCHAR(50)`,
		`ALTER TABLE servers ADD COLUMN IF NOT EXISTS owner_id UUID`,
		`ALTER TABLE servers ADD COLUMN IF NOT EXISTS is_public BOOLEAN DEFAULT false`,
		`ALTER TABLE servers ADD COLUMN IF NOT EXISTS downloads INTEGER DEFAULT 0`,
		`ALTER TABLE servers ADD COLUMN IF NOT EXISTS hosted_virtual BOOLEAN NOT NULL DEFAULT false`,
		`UPDATE servers
		 SET hosted_virtual = true
		 WHERE hosted_virtual = false
		   AND owner_id IS NOT NULL
		   AND is_public = false
		   AND status = 'draft'
		   AND (name LIKE '% (Marketplace)' OR name LIKE '% (Composition)')`,
		`CREATE TABLE IF NOT EXISTS server_versions (
			id UUID PRIMARY KEY,
			server_id UUID REFERENCES servers(id) ON DELETE CASCADE,
			version VARCHAR(50) NOT NULL,
			release_notes TEXT,
			snapshot JSONB NOT NULL,
			published_by UUID,
			published_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			UNIQUE(server_id, version)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_server_versions_server_id ON server_versions(server_id)`,
		`CREATE INDEX IF NOT EXISTS idx_servers_status ON servers(status)`,
		`CREATE INDEX IF NOT EXISTS idx_servers_is_public ON servers(is_public)`,
		`CREATE INDEX IF NOT EXISTS idx_servers_owner_non_virtual ON servers(owner_id) WHERE hosted_virtual = false`,
		`CREATE TABLE IF NOT EXISTS tool_test_presets (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tool_id UUID NOT NULL REFERENCES tools(id) ON DELETE CASCADE,
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			name TEXT NOT NULL,
			input_json JSONB NOT NULL,
			context_json JSONB NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_tool_test_presets_tool_user ON tool_test_presets(tool_id, user_id)`,
		`ALTER TABLE servers ADD COLUMN IF NOT EXISTS env_profiles JSONB`,
		`ALTER TABLE servers ADD COLUMN IF NOT EXISTS hosted_access_key VARCHAR(128)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_servers_hosted_access_key ON servers(hosted_access_key) WHERE hosted_access_key IS NOT NULL`,
		`ALTER TABLE servers ADD COLUMN IF NOT EXISTS hosted_auth_mode VARCHAR(32)`,
		`UPDATE servers SET hosted_auth_mode = 'no_auth' WHERE hosted_auth_mode IS NULL OR TRIM(hosted_auth_mode) = '' OR hosted_auth_mode = 'caller_identity' OR hosted_auth_mode = 'auto_flow'`,
		`ALTER TABLE servers ADD COLUMN IF NOT EXISTS require_caller_identity BOOLEAN NOT NULL DEFAULT false`,
		`CREATE TABLE IF NOT EXISTS hosted_sessions (
			id UUID PRIMARY KEY,
			user_id UUID NOT NULL,
			server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
			snapshot_version VARCHAR(128),
			container_id VARCHAR(128),
			host_port VARCHAR(20),
			status VARCHAR(32) NOT NULL DEFAULT 'starting',
			health VARCHAR(32) NOT NULL DEFAULT 'unknown',
			last_used_at TIMESTAMPTZ,
			last_ensured_at TIMESTAMPTZ,
			started_at TIMESTAMPTZ,
			stopped_at TIMESTAMPTZ,
			last_error TEXT,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE(user_id, server_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_hosted_sessions_user_id ON hosted_sessions(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_hosted_sessions_status ON hosted_sessions(status)`,
		`CREATE TABLE IF NOT EXISTS hosted_user_caller_api_keys (
			id UUID PRIMARY KEY,
			owner_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			key_id VARCHAR(32) NOT NULL UNIQUE,
			key_hash VARCHAR(128) NOT NULL,
			caller_user_id VARCHAR(255) NOT NULL,
			tenant_id VARCHAR(255),
			scopes TEXT[] NOT NULL DEFAULT '{}',
			allow_alias BOOLEAN NOT NULL DEFAULT false,
			expires_at TIMESTAMPTZ,
			revoked_at TIMESTAMPTZ,
			created_by UUID,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_hosted_user_caller_api_keys_owner_user_id ON hosted_user_caller_api_keys(owner_user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_hosted_user_caller_api_keys_key_id ON hosted_user_caller_api_keys(key_id)`,
		`ALTER TABLE servers ADD COLUMN IF NOT EXISTS hosted_security_config JSONB`,
		`CREATE TABLE IF NOT EXISTS hosted_security_audit (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			server_id UUID NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
			actor_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
			action VARCHAR(64) NOT NULL,
			resource_type VARCHAR(32),
			resource_id VARCHAR(128),
			metadata JSONB,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_hosted_security_audit_server_created ON hosted_security_audit(server_id, created_at DESC)`,
		`ALTER TABLE servers ADD COLUMN IF NOT EXISTS hosted_runtime_config JSONB`,
	}

	for _, m := range migrations {
		if _, err := db.pool.Exec(ctx, m); err != nil {
			return fmt.Errorf("running migration: %w", err)
		}
	}

	return nil
}

// Server operations
func (db *DB) CreateServer(ctx context.Context, req models.CreateServerRequest) (*models.Server, error) {
	server := &models.Server{
		ID:          uuid.New().String(),
		Name:        req.Name,
		Description: req.Description,
		Version:     req.Version,
		Icon:        req.Icon,
		Status:      models.ServerStatusDraft,
		OwnerID:     req.OwnerID,
		IsPublic:    false,
		Downloads:   0,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if server.Version == "" {
		server.Version = "1.0.0"
	}

	if server.Icon == "" {
		server.Icon = "bi-server"
	}

	_, err := db.pool.Exec(ctx,
		`INSERT INTO servers (id, name, description, version, icon, status, owner_id, is_public, downloads, created_at, updated_at) 
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		server.ID, server.Name, server.Description, server.Version, server.Icon, server.Status, nullIfEmpty(server.OwnerID), server.IsPublic, server.Downloads, server.CreatedAt, server.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("inserting server: %w", err)
	}

	return server, nil
}

// EnsureHostedVirtualServer creates a minimal server row with a fixed ID when missing.
// Used for hosted deployments of assets like marketplace items/compositions.
func (db *DB) EnsureHostedVirtualServer(ctx context.Context, id, ownerID, name, description string) (*models.Server, error) {
	if strings.TrimSpace(id) == "" || strings.TrimSpace(ownerID) == "" || strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("id, ownerID, and name are required")
	}
	existing, err := db.GetServer(ctx, id)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		if !existing.HostedVirtual {
			if _, err := db.pool.Exec(ctx, `UPDATE servers SET hosted_virtual = true, updated_at = NOW() WHERE id = $1`, id); err != nil {
				return nil, fmt.Errorf("marking hosted virtual server: %w", err)
			}
			existing.HostedVirtual = true
		}
		return existing, nil
	}

	server := &models.Server{
		ID:            id,
		Name:          name,
		Description:   description,
		Version:       "1.0.0",
		Icon:          "bi-server",
		Status:        models.ServerStatusDraft,
		OwnerID:       ownerID,
		IsPublic:      false,
		Downloads:     0,
		HostedVirtual: true,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	_, err = db.pool.Exec(ctx,
		`INSERT INTO servers (id, name, description, version, icon, status, owner_id, is_public, downloads, hosted_virtual, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		server.ID, server.Name, server.Description, server.Version, server.Icon, server.Status, nullIfEmpty(server.OwnerID), server.IsPublic, server.Downloads, server.HostedVirtual, server.CreatedAt, server.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("inserting hosted virtual server: %w", err)
	}
	return server, nil
}

func (db *DB) GetServer(ctx context.Context, id string) (*models.Server, error) {
	server, err := db.getServerCore(ctx, id)
	if err != nil {
		return nil, err
	}
	if server == nil {
		return nil, nil
	}

	tools, err := db.GetToolsByServer(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("getting tools: %w", err)
	}
	server.Tools = tools

	resources, err := db.GetResourcesByServer(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("getting resources: %w", err)
	}
	server.Resources = resources

	prompts, err := db.GetPromptsByServer(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("getting prompts: %w", err)
	}
	server.Prompts = prompts

	return server, nil
}

func (db *DB) ListServers(ctx context.Context, ownerID string) ([]models.Server, error) {
	if ownerID == "" {
		return nil, nil
	}
	// Compare as text so owner_id filter is exact; only return rows owned by this user
	rows, err := db.pool.Query(ctx,
		`SELECT id, name, description, version, icon, status, published_at, latest_version, owner_id, is_public, downloads, hosted_virtual, created_at, updated_at
		 FROM servers
		 WHERE owner_id IS NOT NULL
		   AND owner_id::text = $1
		   AND hosted_virtual = false
		 ORDER BY updated_at DESC`,
		ownerID)
	if err != nil {
		return nil, fmt.Errorf("querying servers: %w", err)
	}
	defer rows.Close()

	var servers []models.Server
	for rows.Next() {
		var s models.Server
		var icon, status, latestVersion, oid *string
		var publishedAt *time.Time
		var isPublic *bool
		var downloads *int
		var hostedVirtual *bool
		if err := rows.Scan(&s.ID, &s.Name, &s.Description, &s.Version, &icon, &status, &publishedAt, &latestVersion, &oid, &isPublic, &downloads, &hostedVirtual, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning server: %w", err)
		}
		if icon != nil {
			s.Icon = *icon
		} else {
			s.Icon = "bi-server"
		}
		if status != nil {
			s.Status = models.ServerStatus(*status)
		} else {
			s.Status = models.ServerStatusDraft
		}
		if publishedAt != nil {
			s.PublishedAt = publishedAt
		}
		if latestVersion != nil {
			s.LatestVersion = *latestVersion
		}
		if oid != nil {
			s.OwnerID = *oid
		}
		if isPublic != nil {
			s.IsPublic = *isPublic
		}
		if downloads != nil {
			s.Downloads = *downloads
		}
		if hostedVirtual != nil {
			s.HostedVirtual = *hostedVirtual
		}
		servers = append(servers, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating servers: %w", err)
	}

	// Annotate hosted running state in one query to avoid per-server calls.
	runningByServer := make(map[string]bool)
	sessionRows, err := db.pool.Query(ctx,
		`SELECT server_id FROM hosted_sessions
		 WHERE user_id::text = $1 AND status = 'running'`,
		ownerID)
	if err != nil {
		return nil, fmt.Errorf("querying hosted sessions: %w", err)
	}
	for sessionRows.Next() {
		var sid string
		if err := sessionRows.Scan(&sid); err != nil {
			sessionRows.Close()
			return nil, fmt.Errorf("scanning hosted session: %w", err)
		}
		runningByServer[sid] = true
	}
	if err := sessionRows.Err(); err != nil {
		sessionRows.Close()
		return nil, fmt.Errorf("iterating hosted sessions: %w", err)
	}
	sessionRows.Close()

	if len(servers) == 0 {
		return servers, nil
	}

	ids := make([]string, len(servers))
	for i := range servers {
		ids[i] = servers[i].ID
	}
	toolsBy, err := db.loadToolsByServerIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("batch loading tools for server list: %w", err)
	}
	resBy, err := db.loadResourcesByServerIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("batch loading resources for server list: %w", err)
	}
	promptBy, err := db.loadPromptsByServerIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("batch loading prompts for server list: %w", err)
	}

	for i := range servers {
		id := servers[i].ID
		servers[i].HostedRunning = runningByServer[id]
		servers[i].Tools = toolsBy[id]
		servers[i].Resources = resBy[id]
		servers[i].Prompts = promptBy[id]
	}

	return servers, nil
}

func (db *DB) UpdateServer(ctx context.Context, id string, req models.CreateServerRequest) (*models.Server, error) {
	icon := req.Icon
	if icon == "" {
		icon = "bi-server"
	}

	_, err := db.pool.Exec(ctx,
		`UPDATE servers SET name = $2, description = $3, version = $4, icon = $5, updated_at = NOW() WHERE id = $1`,
		id, req.Name, req.Description, req.Version, icon)

	if err != nil {
		return nil, fmt.Errorf("updating server: %w", err)
	}

	return db.GetServer(ctx, id)
}

func (db *DB) UpdateEnvProfiles(ctx context.Context, serverID string, profiles json.RawMessage) error {
	_, err := db.pool.Exec(ctx,
		`UPDATE servers SET env_profiles = $2, updated_at = NOW() WHERE id = $1`,
		serverID, profiles)
	return err
}

func (db *DB) DeleteServer(ctx context.Context, id string) error {
	_, err := db.pool.Exec(ctx, `DELETE FROM servers WHERE id = $1`, id)
	return err
}

// Tool operations
func (db *DB) CreateTool(ctx context.Context, req models.CreateToolRequest) (*models.Tool, error) {
	req.Name = strings.TrimSpace(req.Name)
	if err := mcpvalidate.ValidateToolName(req.Name); err != nil {
		return nil, err
	}
	var nameTaken bool
	if err := db.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM tools WHERE server_id = $1 AND name = $2)`,
		req.ServerID, req.Name).Scan(&nameTaken); err != nil {
		return nil, fmt.Errorf("check tool name: %w", err)
	}
	if nameTaken {
		return nil, fmt.Errorf("%w: %q already exists on this server", mcpvalidate.ErrDuplicateToolName, req.Name)
	}

	outputDisplay := models.NormalizeOutputDisplay(req.OutputDisplay)
	odc, err := models.NormalizeOutputDisplayConfigRaw(req.OutputDisplayConfig)
	if err != nil {
		return nil, fmt.Errorf("output_display_config: %w", err)
	}

	tool := &models.Tool{
		ID:                  uuid.New().String(),
		ServerID:            req.ServerID,
		Name:                req.Name,
		Description:         req.Description,
		InputSchema:         req.InputSchema,
		OutputSchema:        req.OutputSchema,
		ExecutionType:       req.ExecutionType,
		ExecutionConfig:     req.ExecutionConfig,
		ContextFields:       req.ContextFields,
		OutputDisplay:       outputDisplay,
		OutputDisplayConfig: odc,
		ReadOnlyHint:        req.ReadOnlyHint,
		DestructiveHint:     req.DestructiveHint,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}

	_, err = db.pool.Exec(ctx,
		`INSERT INTO tools (id, server_id, name, description, input_schema, output_schema, execution_type, execution_config, context_fields, output_display, output_display_config, read_only_hint, destructive_hint, created_at, updated_at) 
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`,
		tool.ID, tool.ServerID, tool.Name, tool.Description, tool.InputSchema, tool.OutputSchema, tool.ExecutionType, tool.ExecutionConfig, tool.ContextFields, tool.OutputDisplay, nullableJSON(tool.OutputDisplayConfig), tool.ReadOnlyHint, tool.DestructiveHint, tool.CreatedAt, tool.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("inserting tool: %w", err)
	}

	return tool, nil
}

func (db *DB) GetTool(ctx context.Context, id string) (*models.Tool, error) {
	var tool models.Tool
	var outputDisplay *string

	var odc []byte
	err := db.pool.QueryRow(ctx,
		`SELECT id, server_id, name, description, input_schema, output_schema, execution_type, execution_config, context_fields, output_display, output_display_config, read_only_hint, destructive_hint, created_at, updated_at 
		 FROM tools WHERE id = $1`, id).
		Scan(&tool.ID, &tool.ServerID, &tool.Name, &tool.Description, &tool.InputSchema, &tool.OutputSchema, &tool.ExecutionType, &tool.ExecutionConfig, &tool.ContextFields, &outputDisplay, &odc, &tool.ReadOnlyHint, &tool.DestructiveHint, &tool.CreatedAt, &tool.UpdatedAt)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("querying tool: %w", err)
	}
	if outputDisplay != nil {
		tool.OutputDisplay = *outputDisplay
	} else {
		tool.OutputDisplay = models.OutputDisplayJSON
	}
	if len(odc) > 0 {
		tool.OutputDisplayConfig = odc
	}

	return &tool, nil
}

func (db *DB) GetToolsByServer(ctx context.Context, serverID string) ([]models.Tool, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT id, server_id, name, description, input_schema, output_schema, execution_type, execution_config, context_fields, output_display, output_display_config, read_only_hint, destructive_hint, created_at, updated_at 
		 FROM tools WHERE server_id = $1 ORDER BY name`, serverID)
	if err != nil {
		return nil, fmt.Errorf("querying tools: %w", err)
	}
	defer rows.Close()

	var tools []models.Tool
	for rows.Next() {
		var t models.Tool
		var outputDisplay *string
		var odc []byte
		if err := rows.Scan(&t.ID, &t.ServerID, &t.Name, &t.Description, &t.InputSchema, &t.OutputSchema, &t.ExecutionType, &t.ExecutionConfig, &t.ContextFields, &outputDisplay, &odc, &t.ReadOnlyHint, &t.DestructiveHint, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning tool: %w", err)
		}
		if outputDisplay != nil {
			t.OutputDisplay = *outputDisplay
		} else {
			t.OutputDisplay = models.OutputDisplayJSON
		}
		if len(odc) > 0 {
			t.OutputDisplayConfig = odc
		}
		tools = append(tools, t)
	}

	return tools, nil
}

// GetToolIDByServerAndName returns the tool ID for a tool in the server with the given name (for observability event resolution).
func (db *DB) GetToolIDByServerAndName(ctx context.Context, serverID, toolName string) (string, error) {
	var id string
	err := db.pool.QueryRow(ctx, `SELECT id FROM tools WHERE server_id = $1 AND name = $2`, serverID, toolName).Scan(&id)
	if err == pgx.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return id, nil
}

// maxObsIngestToolNames caps unique tool names resolved per observability ingest batch (bounds query size).
const maxObsIngestToolNames = 512

// GetToolIDsByServerAndNames returns tool IDs keyed by exact tool name for tools on serverID.
// names is deduplicated; empty strings skipped. If more than maxObsIngestToolNames unique names are
// provided, only the first maxObsIngestToolNames (in order) are queried.
func (db *DB) GetToolIDsByServerAndNames(ctx context.Context, serverID string, names []string) (map[string]string, error) {
	if strings.TrimSpace(serverID) == "" {
		return map[string]string{}, nil
	}
	seen := make(map[string]struct{})
	var unique []string
	for _, n := range names {
		n = strings.TrimSpace(n)
		if n == "" {
			continue
		}
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		unique = append(unique, n)
		if len(unique) >= maxObsIngestToolNames {
			break
		}
	}
	if len(unique) == 0 {
		return map[string]string{}, nil
	}
	rows, err := db.pool.Query(ctx,
		`SELECT id, name FROM tools WHERE server_id = $1 AND name::text = ANY($2)`,
		serverID, unique)
	if err != nil {
		return nil, fmt.Errorf("querying tool ids by names: %w", err)
	}
	defer rows.Close()
	out := make(map[string]string, len(unique))
	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, fmt.Errorf("scanning tool id: %w", err)
		}
		out[name] = id
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating tool ids: %w", err)
	}
	return out, nil
}

// ListServerSummariesByOwner returns id and name for non-virtual servers owned by ownerID (for lightweight UI lists).
func (db *DB) ListServerSummariesByOwner(ctx context.Context, ownerID string) ([]models.ServerSummary, error) {
	if strings.TrimSpace(ownerID) == "" {
		return nil, nil
	}
	rows, err := db.pool.Query(ctx,
		`SELECT id, name FROM servers
		 WHERE owner_id IS NOT NULL AND owner_id::text = $1 AND hosted_virtual = false
		 ORDER BY updated_at DESC`,
		ownerID)
	if err != nil {
		return nil, fmt.Errorf("querying server summaries: %w", err)
	}
	defer rows.Close()
	var out []models.ServerSummary
	for rows.Next() {
		var s models.ServerSummary
		if err := rows.Scan(&s.ID, &s.Name); err != nil {
			return nil, fmt.Errorf("scanning server summary: %w", err)
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating server summaries: %w", err)
	}
	if out == nil {
		return []models.ServerSummary{}, nil
	}
	return out, nil
}

func (db *DB) UpdateTool(ctx context.Context, id string, req models.CreateToolRequest) (*models.Tool, error) {
	req.Name = strings.TrimSpace(req.Name)
	if err := mcpvalidate.ValidateToolName(req.Name); err != nil {
		return nil, err
	}
	current, err := db.GetTool(ctx, id)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, fmt.Errorf("tool not found")
	}
	if current.Name != req.Name {
		var nameTaken bool
		if err := db.pool.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM tools WHERE server_id = $1 AND name = $2 AND id <> $3)`,
			current.ServerID, req.Name, id).Scan(&nameTaken); err != nil {
			return nil, fmt.Errorf("check tool name: %w", err)
		}
		if nameTaken {
			return nil, fmt.Errorf("%w: %q already exists on this server", mcpvalidate.ErrDuplicateToolName, req.Name)
		}
	}

	outputDisplay := models.NormalizeOutputDisplay(req.OutputDisplay)
	odc, err := models.NormalizeOutputDisplayConfigRaw(req.OutputDisplayConfig)
	if err != nil {
		return nil, fmt.Errorf("output_display_config: %w", err)
	}
	_, err = db.pool.Exec(ctx,
		`UPDATE tools SET name = $2, description = $3, input_schema = $4, output_schema = $5, execution_type = $6, execution_config = $7, context_fields = $8, output_display = $9, output_display_config = $10, read_only_hint = $11, destructive_hint = $12, updated_at = NOW() WHERE id = $1`,
		id, req.Name, req.Description, req.InputSchema, req.OutputSchema, req.ExecutionType, req.ExecutionConfig, req.ContextFields, outputDisplay, nullableJSON(odc), req.ReadOnlyHint, req.DestructiveHint)

	if err != nil {
		return nil, fmt.Errorf("updating tool: %w", err)
	}

	return db.GetTool(ctx, id)
}

func (db *DB) DeleteTool(ctx context.Context, id string) error {
	_, err := db.pool.Exec(ctx, `DELETE FROM tools WHERE id = $1`, id)
	return err
}

// Resource operations
func (db *DB) CreateResource(ctx context.Context, req models.CreateResourceRequest) (*models.Resource, error) {
	resource := &models.Resource{
		ID:        uuid.New().String(),
		ServerID:  req.ServerID,
		Name:      req.Name,
		URI:       req.URI,
		MimeType:  req.MimeType,
		Handler:   req.Handler,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	_, err := db.pool.Exec(ctx,
		`INSERT INTO resources (id, server_id, name, uri, mime_type, handler, created_at, updated_at) 
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		resource.ID, resource.ServerID, resource.Name, resource.URI, resource.MimeType, resource.Handler, resource.CreatedAt, resource.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("inserting resource: %w", err)
	}

	return resource, nil
}

func (db *DB) GetResourcesByServer(ctx context.Context, serverID string) ([]models.Resource, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT id, server_id, name, uri, mime_type, handler, created_at, updated_at 
		 FROM resources WHERE server_id = $1 ORDER BY name`, serverID)
	if err != nil {
		return nil, fmt.Errorf("querying resources: %w", err)
	}
	defer rows.Close()

	var resources []models.Resource
	for rows.Next() {
		var r models.Resource
		if err := rows.Scan(&r.ID, &r.ServerID, &r.Name, &r.URI, &r.MimeType, &r.Handler, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning resource: %w", err)
		}
		resources = append(resources, r)
	}

	return resources, nil
}

func (db *DB) DeleteResource(ctx context.Context, id string) error {
	_, err := db.pool.Exec(ctx, `DELETE FROM resources WHERE id = $1`, id)
	return err
}

// Prompt operations
func (db *DB) CreatePrompt(ctx context.Context, req models.CreatePromptRequest) (*models.Prompt, error) {
	prompt := &models.Prompt{
		ID:          uuid.New().String(),
		ServerID:    req.ServerID,
		Name:        req.Name,
		Description: req.Description,
		Template:    req.Template,
		Arguments:   req.Arguments,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	_, err := db.pool.Exec(ctx,
		`INSERT INTO prompts (id, server_id, name, description, template, arguments, created_at, updated_at) 
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		prompt.ID, prompt.ServerID, prompt.Name, prompt.Description, prompt.Template, prompt.Arguments, prompt.CreatedAt, prompt.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("inserting prompt: %w", err)
	}

	return prompt, nil
}

func (db *DB) GetPromptsByServer(ctx context.Context, serverID string) ([]models.Prompt, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT id, server_id, name, description, template, arguments, created_at, updated_at 
		 FROM prompts WHERE server_id = $1 ORDER BY name`, serverID)
	if err != nil {
		return nil, fmt.Errorf("querying prompts: %w", err)
	}
	defer rows.Close()

	var prompts []models.Prompt
	for rows.Next() {
		var p models.Prompt
		if err := rows.Scan(&p.ID, &p.ServerID, &p.Name, &p.Description, &p.Template, &p.Arguments, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning prompt: %w", err)
		}
		prompts = append(prompts, p)
	}

	return prompts, nil
}

func (db *DB) DeletePrompt(ctx context.Context, id string) error {
	_, err := db.pool.Exec(ctx, `DELETE FROM prompts WHERE id = $1`, id)
	return err
}

// Policy operations
func (db *DB) CreatePolicy(ctx context.Context, req models.CreatePolicyRequest) (*models.Policy, error) {
	policy := &models.Policy{
		ID:          uuid.New().String(),
		ToolID:      req.ToolID,
		Name:        req.Name,
		Description: req.Description,
		Enabled:     req.Enabled,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx,
		`INSERT INTO policies (id, tool_id, name, description, enabled, created_at, updated_at) 
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		policy.ID, policy.ToolID, policy.Name, policy.Description, policy.Enabled, policy.CreatedAt, policy.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("inserting policy: %w", err)
	}

	for _, rule := range req.Rules {
		rule.ID = uuid.New().String()
		rule.PolicyID = policy.ID

		_, err = tx.Exec(ctx,
			`INSERT INTO policy_rules (id, policy_id, type, config, priority, fail_action) 
			 VALUES ($1, $2, $3, $4, $5, $6)`,
			rule.ID, rule.PolicyID, rule.Type, rule.Config, rule.Priority, rule.FailAction)

		if err != nil {
			return nil, fmt.Errorf("inserting policy rule: %w", err)
		}
		policy.Rules = append(policy.Rules, rule)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing transaction: %w", err)
	}

	return policy, nil
}

func (db *DB) GetPoliciesByTool(ctx context.Context, toolID string) ([]models.Policy, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT id, tool_id, name, description, enabled, created_at, updated_at 
		 FROM policies WHERE tool_id = $1 ORDER BY created_at`, toolID)
	if err != nil {
		return nil, fmt.Errorf("querying policies: %w", err)
	}
	defer rows.Close()

	var policies []models.Policy
	policyByID := make(map[string]*models.Policy)
	policyIDs := make([]string, 0)
	for rows.Next() {
		var p models.Policy
		if err := rows.Scan(&p.ID, &p.ToolID, &p.Name, &p.Description, &p.Enabled, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning policy: %w", err)
		}
		policies = append(policies, p)
		policyIDs = append(policyIDs, p.ID)
	}
	for i := range policies {
		policyByID[policies[i].ID] = &policies[i]
	}
	if err := db.attachPolicyRules(ctx, policyIDs, policyByID); err != nil {
		return nil, err
	}

	return policies, nil
}

// GetPoliciesByServer returns policies grouped by tool_id for all tools on a server.
// This avoids N+1 policy queries when server-level operations need all policy data.
func (db *DB) GetPoliciesByServer(ctx context.Context, serverID string) (map[string][]models.Policy, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT p.id, p.tool_id, p.name, p.description, p.enabled, p.created_at, p.updated_at
		 FROM policies p
		 INNER JOIN tools t ON t.id = p.tool_id
		 WHERE t.server_id = $1
		 ORDER BY p.tool_id, p.created_at`, serverID)
	if err != nil {
		return nil, fmt.Errorf("querying server policies: %w", err)
	}
	defer rows.Close()

	byTool := make(map[string][]models.Policy)
	policyByID := make(map[string]*models.Policy)
	policyIDs := make([]string, 0)
	for rows.Next() {
		var p models.Policy
		if err := rows.Scan(&p.ID, &p.ToolID, &p.Name, &p.Description, &p.Enabled, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning server policy: %w", err)
		}
		byTool[p.ToolID] = append(byTool[p.ToolID], p)
		policyIDs = append(policyIDs, p.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating server policies: %w", err)
	}

	if len(policyIDs) == 0 {
		return byTool, nil
	}

	for toolID := range byTool {
		policies := byTool[toolID]
		for i := range policies {
			policyByID[policies[i].ID] = &policies[i]
		}
		byTool[toolID] = policies
	}

	if err := db.attachPolicyRules(ctx, policyIDs, policyByID); err != nil {
		return nil, err
	}

	return byTool, nil
}

func (db *DB) attachPolicyRules(ctx context.Context, policyIDs []string, policyByID map[string]*models.Policy) error {
	if len(policyIDs) == 0 {
		return nil
	}
	ruleRows, err := db.pool.Query(ctx,
		`SELECT id, policy_id, type, config, priority, fail_action
		 FROM policy_rules
		 WHERE policy_id::text = ANY($1)
		 ORDER BY policy_id, priority`, policyIDs)
	if err != nil {
		return fmt.Errorf("querying policy rules: %w", err)
	}
	defer ruleRows.Close()

	for ruleRows.Next() {
		var r models.PolicyRule
		if err := ruleRows.Scan(&r.ID, &r.PolicyID, &r.Type, &r.Config, &r.Priority, &r.FailAction); err != nil {
			return fmt.Errorf("scanning policy rule: %w", err)
		}
		if p := policyByID[r.PolicyID]; p != nil {
			p.Rules = append(p.Rules, r)
		}
	}
	if err := ruleRows.Err(); err != nil {
		return fmt.Errorf("iterating policy rules: %w", err)
	}
	return nil
}

func (db *DB) DeletePolicy(ctx context.Context, id string) error {
	_, err := db.pool.Exec(ctx, `DELETE FROM policies WHERE id = $1`, id)
	return err
}

// Tool Execution logging
func (db *DB) LogToolExecution(ctx context.Context, exec *models.ToolExecution) error {
	exec.ID = uuid.New().String()
	exec.CreatedAt = time.Now()
	source := exec.Source
	if source == "" {
		source = "playground"
	}

	_, err := db.pool.Exec(ctx,
		`INSERT INTO tool_executions (id, tool_id, server_id, tool_name, source, client_user_id, client_agent, client_token, input, output, error, status_code, duration_ms, success, healing_applied, repair_suggestion, created_at) 
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)`,
		exec.ID, exec.ToolID, exec.ServerID, nullIfEmpty(exec.ToolName), source,
		nullIfEmpty(exec.ClientUserID), nullIfEmpty(exec.ClientAgent), nullIfEmpty(exec.ClientToken),
		exec.Input, exec.Output, exec.Error, exec.StatusCode, exec.DurationMs, exec.Success, exec.HealingApplied, nullIfEmpty(exec.RepairSuggestion), exec.CreatedAt)

	return err
}

func (db *DB) GetToolExecutions(ctx context.Context, toolID string, limit int) ([]models.ToolExecution, error) {
	if limit <= 0 {
		limit = 100
	}

	rows, err := db.pool.Query(ctx,
		`SELECT id, tool_id, server_id, COALESCE(tool_name, ''), COALESCE(source, 'playground'), COALESCE(client_user_id, ''), COALESCE(client_agent, ''), COALESCE(client_token, ''), input, output, error, status_code, duration_ms, success, healing_applied, COALESCE(repair_suggestion, ''), created_at 
		 FROM tool_executions WHERE tool_id = $1 ORDER BY created_at DESC LIMIT $2`, toolID, limit)
	if err != nil {
		return nil, fmt.Errorf("querying executions: %w", err)
	}
	defer rows.Close()

	var executions []models.ToolExecution
	for rows.Next() {
		var e models.ToolExecution
		if err := rows.Scan(&e.ID, &e.ToolID, &e.ServerID, &e.ToolName, &e.Source, &e.ClientUserID, &e.ClientAgent, &e.ClientToken, &e.Input, &e.Output, &e.Error, &e.StatusCode, &e.DurationMs, &e.Success, &e.HealingApplied, &e.RepairSuggestion, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning execution: %w", err)
		}
		executions = append(executions, e)
	}

	return executions, nil
}

// GetServerByObservabilityKey returns the server that has the given reporting key (for runtime event ingestion).
func (db *DB) GetServerByObservabilityKey(ctx context.Context, key string) (*models.Server, error) {
	if key == "" {
		return nil, nil
	}
	var id string
	err := db.pool.QueryRow(ctx, `SELECT id FROM servers WHERE observability_reporting_key = $1`, key).Scan(&id)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	// Ingest only needs server metadata (id); avoid loading tools/resources/prompts.
	return db.getServerCore(ctx, id)
}

// EnsureServerObservabilityKey sets observability_reporting_key for the server if not set, and returns it.
func (db *DB) EnsureServerObservabilityKey(ctx context.Context, serverID string) (string, error) {
	var existing *string
	err := db.pool.QueryRow(ctx, `SELECT observability_reporting_key FROM servers WHERE id = $1`, serverID).Scan(&existing)
	if err != nil {
		return "", err
	}
	if existing != nil && *existing != "" {
		return *existing, nil
	}
	key := uuid.New().String() // 36 chars, fits VARCHAR(64)
	_, err = db.pool.Exec(ctx, `UPDATE servers SET observability_reporting_key = $1 WHERE id = $2`, key, serverID)
	if err != nil {
		return "", err
	}
	return key, nil
}

func generateHostedAccessKey() (string, error) {
	// 32 random bytes => 43-char URL-safe token.
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// EnsureServerHostedAccessKey sets hosted_access_key for the server if missing, and returns it.
func (db *DB) EnsureServerHostedAccessKey(ctx context.Context, serverID string) (string, error) {
	var existing *string
	err := db.pool.QueryRow(ctx, `SELECT hosted_access_key FROM servers WHERE id = $1`, serverID).Scan(&existing)
	if err != nil {
		return "", err
	}
	if existing != nil && *existing != "" {
		return *existing, nil
	}
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

func (db *DB) UpdateServerHostedAuthMode(ctx context.Context, serverID, hostedAuthMode string) error {
	_, err := db.pool.Exec(ctx, `UPDATE servers SET hosted_auth_mode = $2, updated_at = NOW() WHERE id = $1`, serverID, hostedAuthMode)
	return err
}

func (db *DB) UpdateServerRequireCallerIdentity(ctx context.Context, serverID string, required bool) error {
	_, err := db.pool.Exec(ctx, `UPDATE servers SET require_caller_identity = $2, updated_at = NOW() WHERE id = $1`, serverID, required)
	return err
}

// ListRuntimeExecutionsForUser returns runtime tool executions for all servers owned by the user, with optional server_id, tool_name, client_user_id, client_agent filters.
func (db *DB) ListRuntimeExecutionsForUser(ctx context.Context, userID, serverID, toolName, clientUserID, clientAgent string, limit int) ([]models.ToolExecution, error) {
	if userID == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = 200
	}
	query := `SELECT e.id, e.tool_id, e.server_id, COALESCE(e.tool_name, ''), COALESCE(e.source, 'playground'), COALESCE(e.client_user_id, ''), COALESCE(e.client_agent, ''), COALESCE(e.client_token, ''), e.input, e.output, e.error, e.status_code, e.duration_ms, e.success, e.healing_applied, COALESCE(e.repair_suggestion, ''), e.created_at
		FROM tool_executions e
		INNER JOIN servers s ON e.server_id = s.id
		WHERE s.owner_id::text = $1 AND e.source = 'runtime'`
	args := []interface{}{userID}
	argNum := 2
	if serverID != "" {
		query += fmt.Sprintf(" AND e.server_id = $%d", argNum)
		args = append(args, serverID)
		argNum++
	}
	if toolName != "" {
		query += fmt.Sprintf(" AND e.tool_name = $%d", argNum)
		args = append(args, toolName)
		argNum++
	}
	if clientUserID != "" {
		query += fmt.Sprintf(" AND e.client_user_id = $%d", argNum)
		args = append(args, clientUserID)
		argNum++
	}
	if clientAgent != "" {
		query += fmt.Sprintf(" AND e.client_agent = $%d", argNum)
		args = append(args, clientAgent)
		argNum++
	}
	query += fmt.Sprintf(" ORDER BY e.created_at DESC LIMIT $%d", argNum)
	args = append(args, limit)
	rows, err := db.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying runtime executions for user: %w", err)
	}
	defer rows.Close()
	var list []models.ToolExecution
	for rows.Next() {
		var e models.ToolExecution
		if err := rows.Scan(&e.ID, &e.ToolID, &e.ServerID, &e.ToolName, &e.Source, &e.ClientUserID, &e.ClientAgent, &e.ClientToken, &e.Input, &e.Output, &e.Error, &e.StatusCode, &e.DurationMs, &e.Success, &e.HealingApplied, &e.RepairSuggestion, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning execution: %w", err)
		}
		list = append(list, e)
	}
	return list, nil
}

// ListRuntimeExecutionsByServer returns recent tool executions with source='runtime' for the observability tab.
func (db *DB) ListRuntimeExecutionsByServer(ctx context.Context, serverID string, limit int) ([]models.ToolExecution, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := db.pool.Query(ctx,
		`SELECT id, tool_id, server_id, COALESCE(tool_name, ''), COALESCE(source, 'playground'), COALESCE(client_user_id, ''), COALESCE(client_agent, ''), COALESCE(client_token, ''), input, output, error, status_code, duration_ms, success, healing_applied, COALESCE(repair_suggestion, ''), created_at 
		 FROM tool_executions WHERE server_id = $1 AND source = 'runtime' ORDER BY created_at DESC LIMIT $2`, serverID, limit)
	if err != nil {
		return nil, fmt.Errorf("querying runtime executions: %w", err)
	}
	defer rows.Close()

	var list []models.ToolExecution
	for rows.Next() {
		var e models.ToolExecution
		if err := rows.Scan(&e.ID, &e.ToolID, &e.ServerID, &e.ToolName, &e.Source, &e.ClientUserID, &e.ClientAgent, &e.ClientToken, &e.Input, &e.Output, &e.Error, &e.StatusCode, &e.DurationMs, &e.Success, &e.HealingApplied, &e.RepairSuggestion, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning execution: %w", err)
		}
		list = append(list, e)
	}
	return list, nil
}

// Healing suggestions
func (db *DB) CreateHealingSuggestion(ctx context.Context, suggestion *models.HealingSuggestion) error {
	suggestion.ID = uuid.New().String()
	suggestion.CreatedAt = time.Now()

	_, err := db.pool.Exec(ctx,
		`INSERT INTO healing_suggestions (id, tool_id, error_pattern, suggestion_type, suggestion, auto_apply, applied, created_at) 
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		suggestion.ID, suggestion.ToolID, suggestion.ErrorPattern, suggestion.SuggestionType, suggestion.Suggestion, suggestion.AutoApply, suggestion.Applied, suggestion.CreatedAt)

	return err
}

func (db *DB) GetHealingSuggestions(ctx context.Context, toolID string) ([]models.HealingSuggestion, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT id, tool_id, error_pattern, suggestion_type, suggestion, auto_apply, applied, created_at 
		 FROM healing_suggestions WHERE tool_id = $1 ORDER BY created_at DESC`, toolID)
	if err != nil {
		return nil, fmt.Errorf("querying suggestions: %w", err)
	}
	defer rows.Close()

	var suggestions []models.HealingSuggestion
	for rows.Next() {
		var s models.HealingSuggestion
		if err := rows.Scan(&s.ID, &s.ToolID, &s.ErrorPattern, &s.SuggestionType, &s.Suggestion, &s.AutoApply, &s.Applied, &s.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning suggestion: %w", err)
		}
		suggestions = append(suggestions, s)
	}

	return suggestions, nil
}

// ListToolTestPresets returns all test presets for a tool for the given user.
func (db *DB) ListToolTestPresets(ctx context.Context, toolID, userID string) ([]models.ToolTestPreset, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT id, tool_id, user_id, name, COALESCE(input_json, '{}'), COALESCE(context_json, '{}'), created_at, updated_at
		 FROM tool_test_presets WHERE tool_id = $1 AND user_id = $2 ORDER BY updated_at DESC`,
		toolID, userID)
	if err != nil {
		return nil, fmt.Errorf("listing tool test presets: %w", err)
	}
	defer rows.Close()

	var presets []models.ToolTestPreset
	for rows.Next() {
		var p models.ToolTestPreset
		if err := rows.Scan(&p.ID, &p.ToolID, &p.UserID, &p.Name, &p.InputJSON, &p.ContextJSON, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning preset: %w", err)
		}
		presets = append(presets, p)
	}
	return presets, nil
}

// CreateToolTestPreset inserts a new test preset. ID is generated if not set.
func (db *DB) CreateToolTestPreset(ctx context.Context, p *models.ToolTestPreset) error {
	if p.ID == "" {
		p.ID = uuid.New().String()
	}
	now := time.Now()
	p.CreatedAt = now
	p.UpdatedAt = now
	if p.InputJSON == nil {
		p.InputJSON = json.RawMessage("{}")
	}
	if p.ContextJSON == nil {
		p.ContextJSON = json.RawMessage("{}")
	}
	_, err := db.pool.Exec(ctx,
		`INSERT INTO tool_test_presets (id, tool_id, user_id, name, input_json, context_json, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		p.ID, p.ToolID, p.UserID, p.Name, p.InputJSON, p.ContextJSON, p.CreatedAt, p.UpdatedAt)
	if err != nil {
		return fmt.Errorf("inserting tool test preset: %w", err)
	}
	return nil
}

// DeleteToolTestPreset deletes a preset only if it belongs to the given user.
func (db *DB) DeleteToolTestPreset(ctx context.Context, presetID, userID string) error {
	res, err := db.pool.Exec(ctx,
		`DELETE FROM tool_test_presets WHERE id = $1 AND user_id = $2`, presetID, userID)
	if err != nil {
		return fmt.Errorf("deleting tool test preset: %w", err)
	}
	if res.RowsAffected() == 0 {
		return nil // idempotent: no row or wrong user
	}
	return nil
}

// Server Composition
func (db *DB) CreateComposition(ctx context.Context, name, description string, serverIDs []string, ownerID string) (*models.ServerComposition, error) {
	comp := &models.ServerComposition{
		ID:          uuid.New().String(),
		Name:        name,
		Description: description,
		ServerIDs:   serverIDs,
		OwnerID:     ownerID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	_, err := db.pool.Exec(ctx,
		`INSERT INTO server_compositions (id, name, description, server_ids, owner_id, created_at, updated_at) 
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		comp.ID, comp.Name, comp.Description, comp.ServerIDs, nullIfEmpty(comp.OwnerID), comp.CreatedAt, comp.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("inserting composition: %w", err)
	}

	return comp, nil
}

func (db *DB) ListCompositions(ctx context.Context, ownerID string) ([]models.ServerComposition, error) {
	if ownerID == "" {
		return nil, nil
	}
	rows, err := db.pool.Query(ctx,
		`SELECT id, name, description, server_ids, owner_id, created_at, updated_at 
		 FROM server_compositions WHERE owner_id IS NOT NULL AND owner_id = $1 ORDER BY updated_at DESC`,
		ownerID)
	if err != nil {
		return nil, fmt.Errorf("querying compositions: %w", err)
	}
	defer rows.Close()

	var compositions []models.ServerComposition
	for rows.Next() {
		var c models.ServerComposition
		var oid *string
		if err := rows.Scan(&c.ID, &c.Name, &c.Description, &c.ServerIDs, &oid, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning composition: %w", err)
		}
		if oid != nil {
			c.OwnerID = *oid
		}
		compositions = append(compositions, c)
	}

	return compositions, nil
}

func (db *DB) GetComposition(ctx context.Context, id string) (*models.ServerComposition, error) {
	var c models.ServerComposition
	var oid *string
	err := db.pool.QueryRow(ctx,
		`SELECT id, name, description, server_ids, owner_id, created_at, updated_at 
		 FROM server_compositions WHERE id = $1`, id).Scan(
		&c.ID, &c.Name, &c.Description, &c.ServerIDs, &oid, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("getting composition: %w", err)
	}
	if oid != nil {
		c.OwnerID = *oid
	}
	return &c, nil
}

func (db *DB) UpdateComposition(ctx context.Context, id, name, description string, serverIDs []string) (*models.ServerComposition, error) {
	now := time.Now()
	_, err := db.pool.Exec(ctx,
		`UPDATE server_compositions SET name = $1, description = $2, server_ids = $3, updated_at = $4 WHERE id = $5`,
		name, description, serverIDs, now, id)
	if err != nil {
		return nil, fmt.Errorf("updating composition: %w", err)
	}
	return db.GetComposition(ctx, id)
}

func (db *DB) DeleteComposition(ctx context.Context, id string) error {
	_, err := db.pool.Exec(ctx, `DELETE FROM server_compositions WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("deleting composition: %w", err)
	}
	return nil
}

// RemoveServerFromAllCompositions removes serverID from every composition's server_ids.
// Compositions that would have fewer than two servers afterward are deleted.
func (db *DB) RemoveServerFromAllCompositions(ctx context.Context, serverID string) error {
	if serverID == "" {
		return nil
	}
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin composition prune tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	rows, err := tx.Query(ctx,
		`SELECT id, name, description, server_ids FROM server_compositions WHERE $1::uuid = ANY(server_ids)`,
		serverID)
	if err != nil {
		return fmt.Errorf("query compositions containing server: %w", err)
	}
	defer rows.Close()

	type compRow struct {
		id, name, description string
		ids                   []string
	}
	var affected []compRow
	for rows.Next() {
		var r compRow
		if err := rows.Scan(&r.id, &r.name, &r.description, &r.ids); err != nil {
			return fmt.Errorf("scan composition for prune: %w", err)
		}
		affected = append(affected, r)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iter compositions for prune: %w", err)
	}

	now := time.Now()
	for _, r := range affected {
		next := make([]string, 0, len(r.ids))
		for _, id := range r.ids {
			if id != serverID {
				next = append(next, id)
			}
		}
		if len(next) < 2 {
			if _, err := tx.Exec(ctx, `DELETE FROM server_compositions WHERE id = $1`, r.id); err != nil {
				return fmt.Errorf("delete composition %s after server removal: %w", r.id, err)
			}
			continue
		}
		if _, err := tx.Exec(ctx,
			`UPDATE server_compositions SET server_ids = $1, updated_at = $2 WHERE id = $3`,
			next, now, r.id); err != nil {
			return fmt.Errorf("update composition %s after server removal: %w", r.id, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit composition prune: %w", err)
	}
	return nil
}

// Context Config operations
func (db *DB) CreateContextConfig(ctx context.Context, serverID, name, sourceType string, config json.RawMessage) (*models.ContextConfig, error) {
	cc := &models.ContextConfig{
		ID:         uuid.New().String(),
		ServerID:   serverID,
		Name:       name,
		SourceType: sourceType,
		Config:     config,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	_, err := db.pool.Exec(ctx,
		`INSERT INTO context_configs (id, server_id, name, source_type, config, created_at, updated_at) 
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		cc.ID, cc.ServerID, cc.Name, cc.SourceType, cc.Config, cc.CreatedAt, cc.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("inserting context config: %w", err)
	}

	return cc, nil
}

func (db *DB) GetContextConfigs(ctx context.Context, serverID string) ([]models.ContextConfig, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT id, server_id, name, source_type, config, created_at, updated_at 
		 FROM context_configs WHERE server_id = $1 ORDER BY name`, serverID)
	if err != nil {
		return nil, fmt.Errorf("querying context configs: %w", err)
	}
	defer rows.Close()

	var configs []models.ContextConfig
	for rows.Next() {
		var c models.ContextConfig
		if err := rows.Scan(&c.ID, &c.ServerID, &c.Name, &c.SourceType, &c.Config, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning context config: %w", err)
		}
		configs = append(configs, c)
	}

	return configs, nil
}

func (db *DB) DeleteContextConfig(ctx context.Context, id string) error {
	_, err := db.pool.Exec(ctx, `DELETE FROM context_configs WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("deleting context config: %w", err)
	}
	return nil
}

// Flow operations

func (db *DB) CreateFlow(ctx context.Context, req models.CreateFlowRequest) (*models.Flow, error) {
	flow := &models.Flow{
		ID:          uuid.New().String(),
		ServerID:    req.ServerID,
		Name:        req.Name,
		Description: req.Description,
		Nodes:       req.Nodes,
		Edges:       req.Edges,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	_, err := db.pool.Exec(ctx,
		`INSERT INTO flows (id, server_id, name, description, nodes, edges, created_at, updated_at) 
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		flow.ID, flow.ServerID, flow.Name, flow.Description, flow.Nodes, flow.Edges, flow.CreatedAt, flow.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("inserting flow: %w", err)
	}

	return flow, nil
}

func (db *DB) GetFlow(ctx context.Context, id string) (*models.Flow, error) {
	var flow models.Flow
	err := db.pool.QueryRow(ctx,
		`SELECT id, server_id, name, description, nodes, edges, created_at, updated_at 
		 FROM flows WHERE id = $1`, id).
		Scan(&flow.ID, &flow.ServerID, &flow.Name, &flow.Description, &flow.Nodes, &flow.Edges, &flow.CreatedAt, &flow.UpdatedAt)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("flow not found")
	}
	if err != nil {
		return nil, fmt.Errorf("querying flow: %w", err)
	}

	return &flow, nil
}

func (db *DB) UpdateFlow(ctx context.Context, id string, req models.UpdateFlowRequest) (*models.Flow, error) {
	flow, err := db.GetFlow(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != "" {
		flow.Name = req.Name
	}
	if req.Description != "" {
		flow.Description = req.Description
	}
	if req.Nodes != nil {
		flow.Nodes = req.Nodes
	}
	if req.Edges != nil {
		flow.Edges = req.Edges
	}
	flow.UpdatedAt = time.Now()

	_, err = db.pool.Exec(ctx,
		`UPDATE flows SET name = $1, description = $2, nodes = $3, edges = $4, updated_at = $5 WHERE id = $6`,
		flow.Name, flow.Description, flow.Nodes, flow.Edges, flow.UpdatedAt, id)

	if err != nil {
		return nil, fmt.Errorf("updating flow: %w", err)
	}

	return flow, nil
}

func (db *DB) DeleteFlow(ctx context.Context, id string) error {
	_, err := db.pool.Exec(ctx, `DELETE FROM flows WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("deleting flow: %w", err)
	}
	return nil
}

func (db *DB) ListFlows(ctx context.Context) ([]models.Flow, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT id, server_id, name, description, nodes, edges, created_at, updated_at 
		 FROM flows ORDER BY updated_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("querying flows: %w", err)
	}
	defer rows.Close()

	var flows []models.Flow
	for rows.Next() {
		var f models.Flow
		if err := rows.Scan(&f.ID, &f.ServerID, &f.Name, &f.Description, &f.Nodes, &f.Edges, &f.CreatedAt, &f.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning flow: %w", err)
		}
		flows = append(flows, f)
	}

	return flows, nil
}

func (db *DB) GetFlowsByServer(ctx context.Context, serverID string) ([]models.Flow, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT id, server_id, name, description, nodes, edges, created_at, updated_at 
		 FROM flows WHERE server_id = $1 ORDER BY updated_at DESC`, serverID)
	if err != nil {
		return nil, fmt.Errorf("querying flows: %w", err)
	}
	defer rows.Close()

	var flows []models.Flow
	for rows.Next() {
		var f models.Flow
		if err := rows.Scan(&f.ID, &f.ServerID, &f.Name, &f.Description, &f.Nodes, &f.Edges, &f.CreatedAt, &f.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning flow: %w", err)
		}
		flows = append(flows, f)
	}

	return flows, nil
}

// User operations
func (db *DB) CreateUser(ctx context.Context, email, name, passwordHash string) (*models.User, error) {
	user := &models.User{
		ID:           uuid.New().String(),
		Email:        email,
		Name:         name,
		PasswordHash: passwordHash,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	_, err := db.pool.Exec(ctx,
		`INSERT INTO users (id, email, name, password_hash, created_at, updated_at) 
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		user.ID, user.Email, user.Name, nullIfEmpty(passwordHash), user.CreatedAt, user.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("inserting user: %w", err)
	}

	return user, nil
}

func (db *DB) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	var passwordHash *string

	err := db.pool.QueryRow(ctx,
		`SELECT id, email, name, password_hash, created_at, updated_at 
		 FROM users WHERE email = $1`, email).
		Scan(&user.ID, &user.Email, &user.Name, &passwordHash, &user.CreatedAt, &user.UpdatedAt)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("querying user: %w", err)
	}
	if passwordHash != nil {
		user.PasswordHash = *passwordHash
	}
	return &user, nil
}

func (db *DB) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	var user models.User
	var passwordHash *string

	err := db.pool.QueryRow(ctx,
		`SELECT id, email, name, password_hash, created_at, updated_at 
		 FROM users WHERE id = $1`, id).
		Scan(&user.ID, &user.Email, &user.Name, &passwordHash, &user.CreatedAt, &user.UpdatedAt)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("querying user: %w", err)
	}
	if passwordHash != nil {
		user.PasswordHash = *passwordHash
	}
	return &user, nil
}

// WebAuthnCredentialRow is a row from webauthn_credentials for loading.
type WebAuthnCredentialRow struct {
	CredentialID []byte
	Data         []byte
}

func (db *DB) SaveWebAuthnCredential(ctx context.Context, userID string, credentialID []byte, data []byte) error {
	_, err := db.pool.Exec(ctx,
		`INSERT INTO webauthn_credentials (id, user_id, credential_id, data) VALUES ($1, $2, $3, $4)`,
		uuid.New().String(), userID, credentialID, data)
	return err
}

func (db *DB) GetWebAuthnCredentials(ctx context.Context, userID string) ([]WebAuthnCredentialRow, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT credential_id, data FROM webauthn_credentials WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []WebAuthnCredentialRow
	for rows.Next() {
		var row WebAuthnCredentialRow
		if err := rows.Scan(&row.CredentialID, &row.Data); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

// Server Version operations
func (db *DB) PublishServerVersion(ctx context.Context, serverID, version, releaseNotes, publishedBy string, snapshot []byte) (*models.ServerVersion, error) {
	sv := &models.ServerVersion{
		ID:           uuid.New().String(),
		ServerID:     serverID,
		Version:      version,
		ReleaseNotes: releaseNotes,
		Snapshot:     snapshot,
		PublishedBy:  publishedBy,
		PublishedAt:  time.Now(),
	}

	// Handle empty publishedBy - use NULL instead of empty string for UUID column
	var publishedByParam interface{}
	if publishedBy == "" {
		publishedByParam = nil
	} else {
		publishedByParam = publishedBy
	}

	_, err := db.pool.Exec(ctx,
		`INSERT INTO server_versions (id, server_id, version, release_notes, snapshot, published_by, published_at) 
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		sv.ID, sv.ServerID, sv.Version, sv.ReleaseNotes, sv.Snapshot, publishedByParam, sv.PublishedAt)

	if err != nil {
		return nil, fmt.Errorf("inserting server version: %w", err)
	}

	// Update server status and latest_version
	_, err = db.pool.Exec(ctx,
		`UPDATE servers SET status = $2, latest_version = $3, published_at = NOW(), updated_at = NOW() WHERE id = $1`,
		serverID, models.ServerStatusPublished, version)

	if err != nil {
		return nil, fmt.Errorf("updating server status: %w", err)
	}

	return sv, nil
}

// CreateHostedServerVersion stores a hosted-only snapshot without mutating servers.latest_version.
func (db *DB) CreateHostedServerVersion(ctx context.Context, serverID, publishedBy string, snapshot []byte) (*models.ServerVersion, error) {
	version := fmt.Sprintf("hosted-%d", time.Now().UTC().UnixNano())
	sv := &models.ServerVersion{
		ID:           uuid.New().String(),
		ServerID:     serverID,
		Version:      version,
		ReleaseNotes: "Hosted deployment",
		Snapshot:     snapshot,
		PublishedBy:  publishedBy,
		PublishedAt:  time.Now(),
	}

	var publishedByParam interface{}
	if publishedBy == "" {
		publishedByParam = nil
	} else {
		publishedByParam = publishedBy
	}

	_, err := db.pool.Exec(ctx,
		`INSERT INTO server_versions (id, server_id, version, release_notes, snapshot, published_by, published_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		sv.ID, sv.ServerID, sv.Version, sv.ReleaseNotes, sv.Snapshot, publishedByParam, sv.PublishedAt)
	if err != nil {
		return nil, fmt.Errorf("inserting hosted server version: %w", err)
	}
	return sv, nil
}

// GetLatestHostedServerVersion returns the latest hosted-only snapshot for a server.
func (db *DB) GetLatestHostedServerVersion(ctx context.Context, serverID string) (*models.ServerVersion, error) {
	var v models.ServerVersion
	var publishedBy *string
	err := db.pool.QueryRow(ctx,
		`SELECT id, server_id, version, COALESCE(release_notes, ''), snapshot, published_by, published_at
		 FROM server_versions
		 WHERE server_id = $1 AND release_notes = 'Hosted deployment'
		 ORDER BY published_at DESC
		 LIMIT 1`, serverID).
		Scan(&v.ID, &v.ServerID, &v.Version, &v.ReleaseNotes, &v.Snapshot, &publishedBy, &v.PublishedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("querying latest hosted server version: %w", err)
	}
	if publishedBy != nil {
		v.PublishedBy = *publishedBy
	}
	return &v, nil
}

// GetLatestNonHostedServerVersion returns latest non-hosted snapshot for a server.
func (db *DB) GetLatestNonHostedServerVersion(ctx context.Context, serverID string) (*models.ServerVersion, error) {
	var v models.ServerVersion
	var publishedBy *string
	err := db.pool.QueryRow(ctx,
		`SELECT id, server_id, version, COALESCE(release_notes, ''), snapshot, published_by, published_at
		 FROM server_versions
		 WHERE server_id = $1 AND COALESCE(release_notes, '') <> 'Hosted deployment'
		 ORDER BY published_at DESC
		 LIMIT 1`, serverID).
		Scan(&v.ID, &v.ServerID, &v.Version, &v.ReleaseNotes, &v.Snapshot, &publishedBy, &v.PublishedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("querying latest non-hosted server version: %w", err)
	}
	if publishedBy != nil {
		v.PublishedBy = *publishedBy
	}
	return &v, nil
}

func (db *DB) UpdateServerLatestVersion(ctx context.Context, serverID, latestVersion string) error {
	_, err := db.pool.Exec(ctx,
		`UPDATE servers SET latest_version = $2, updated_at = NOW() WHERE id = $1`,
		serverID, latestVersion)
	if err != nil {
		return fmt.Errorf("updating server latest_version: %w", err)
	}
	return nil
}

func timePtrOrNil(t *time.Time) interface{} {
	if t == nil {
		return nil
	}
	return *t
}

func stringPtrOrNil(s string) interface{} {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}

// UpsertHostedSession creates or updates durable hosted runtime session state.
func (db *DB) UpsertHostedSession(ctx context.Context, s models.HostedSession) (*models.HostedSession, error) {
	now := time.Now().UTC()
	if s.ID == "" {
		s.ID = uuid.New().String()
	}
	if strings.TrimSpace(s.Status) == "" {
		s.Status = "starting"
	}
	if strings.TrimSpace(s.Health) == "" {
		s.Health = "unknown"
	}

	var out models.HostedSession
	err := db.pool.QueryRow(ctx, `
		INSERT INTO hosted_sessions (
			id, user_id, server_id, snapshot_version, container_id, host_port, status, health,
			last_used_at, last_ensured_at, started_at, stopped_at, last_error, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8,
			$9, $10, $11, $12, $13, $14, $14
		)
		ON CONFLICT (user_id, server_id) DO UPDATE SET
			snapshot_version = EXCLUDED.snapshot_version,
			container_id = EXCLUDED.container_id,
			host_port = EXCLUDED.host_port,
			status = EXCLUDED.status,
			health = EXCLUDED.health,
			last_used_at = EXCLUDED.last_used_at,
			last_ensured_at = EXCLUDED.last_ensured_at,
			started_at = EXCLUDED.started_at,
			stopped_at = EXCLUDED.stopped_at,
			last_error = EXCLUDED.last_error,
			updated_at = EXCLUDED.updated_at
		RETURNING id, user_id, server_id, COALESCE(snapshot_version, ''), COALESCE(container_id, ''),
			COALESCE(host_port, ''), status, health, last_used_at, last_ensured_at, started_at, stopped_at,
			COALESCE(last_error, ''), created_at, updated_at
	`,
		s.ID, s.UserID, s.ServerID,
		stringPtrOrNil(s.SnapshotVersion),
		stringPtrOrNil(s.ContainerID),
		stringPtrOrNil(s.HostPort),
		s.Status, s.Health,
		timePtrOrNil(s.LastUsedAt),
		timePtrOrNil(s.LastEnsuredAt),
		timePtrOrNil(s.StartedAt),
		timePtrOrNil(s.StoppedAt),
		stringPtrOrNil(s.LastError),
		now,
	).Scan(
		&out.ID, &out.UserID, &out.ServerID, &out.SnapshotVersion, &out.ContainerID,
		&out.HostPort, &out.Status, &out.Health, &out.LastUsedAt, &out.LastEnsuredAt, &out.StartedAt,
		&out.StoppedAt, &out.LastError, &out.CreatedAt, &out.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("upserting hosted session: %w", err)
	}
	return &out, nil
}

func (db *DB) GetHostedSession(ctx context.Context, userID, serverID string) (*models.HostedSession, error) {
	var out models.HostedSession
	err := db.pool.QueryRow(ctx, `
		SELECT id, user_id, server_id, COALESCE(snapshot_version, ''), COALESCE(container_id, ''),
			COALESCE(host_port, ''), status, health, last_used_at, last_ensured_at, started_at, stopped_at,
			COALESCE(last_error, ''), created_at, updated_at
		FROM hosted_sessions
		WHERE user_id = $1 AND server_id = $2
	`, userID, serverID).Scan(
		&out.ID, &out.UserID, &out.ServerID, &out.SnapshotVersion, &out.ContainerID,
		&out.HostPort, &out.Status, &out.Health, &out.LastUsedAt, &out.LastEnsuredAt, &out.StartedAt,
		&out.StoppedAt, &out.LastError, &out.CreatedAt, &out.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("querying hosted session: %w", err)
	}
	return &out, nil
}

func (db *DB) ListHostedSessions(ctx context.Context, userID string) ([]models.HostedSession, error) {
	rows, err := db.pool.Query(ctx, `
		SELECT id, user_id, server_id, COALESCE(snapshot_version, ''), COALESCE(container_id, ''),
			COALESCE(host_port, ''), status, health, last_used_at, last_ensured_at, started_at, stopped_at,
			COALESCE(last_error, ''), created_at, updated_at
		FROM hosted_sessions
		WHERE user_id = $1
		ORDER BY COALESCE(last_used_at, updated_at) DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("listing hosted sessions: %w", err)
	}
	defer rows.Close()

	sessions := make([]models.HostedSession, 0)
	for rows.Next() {
		var s models.HostedSession
		if err := rows.Scan(
			&s.ID, &s.UserID, &s.ServerID, &s.SnapshotVersion, &s.ContainerID,
			&s.HostPort, &s.Status, &s.Health, &s.LastUsedAt, &s.LastEnsuredAt, &s.StartedAt,
			&s.StoppedAt, &s.LastError, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning hosted session: %w", err)
		}
		sessions = append(sessions, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating hosted sessions: %w", err)
	}
	return sessions, nil
}

// ListRunningHostedSessions returns all currently running hosted sessions across users.
func (db *DB) ListRunningHostedSessions(ctx context.Context) ([]models.HostedSession, error) {
	rows, err := db.pool.Query(ctx, `
		SELECT id, user_id, server_id, COALESCE(snapshot_version, ''), COALESCE(container_id, ''),
			COALESCE(host_port, ''), status, health, last_used_at, last_ensured_at, started_at, stopped_at,
			COALESCE(last_error, ''), created_at, updated_at
		FROM hosted_sessions
		WHERE status = 'running'
		ORDER BY COALESCE(last_used_at, updated_at) DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("listing running hosted sessions: %w", err)
	}
	defer rows.Close()

	sessions := make([]models.HostedSession, 0)
	for rows.Next() {
		var s models.HostedSession
		if err := rows.Scan(
			&s.ID, &s.UserID, &s.ServerID, &s.SnapshotVersion, &s.ContainerID,
			&s.HostPort, &s.Status, &s.Health, &s.LastUsedAt, &s.LastEnsuredAt, &s.StartedAt,
			&s.StoppedAt, &s.LastError, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning running hosted session: %w", err)
		}
		sessions = append(sessions, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating running hosted sessions: %w", err)
	}
	return sessions, nil
}

func hashHostedCallerAPIKey(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func generateHostedCallerAPIKeyMaterial() (keyID, secret, fullKey string, err error) {
	keyIDBytes := make([]byte, 8)
	if _, err = rand.Read(keyIDBytes); err != nil {
		return "", "", "", err
	}
	secretBytes := make([]byte, 16)
	if _, err = rand.Read(secretBytes); err != nil {
		return "", "", "", err
	}
	keyID = hex.EncodeToString(keyIDBytes)
	secret = hex.EncodeToString(secretBytes)
	fullKey = "mkc_" + keyID + "_" + secret
	return keyID, secret, fullKey, nil
}

func normalizeScopes(scopes []string) []string {
	if len(scopes) == 0 {
		return []string{}
	}
	seen := make(map[string]struct{}, len(scopes))
	out := make([]string, 0, len(scopes))
	for _, s := range scopes {
		v := strings.TrimSpace(s)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

func (db *DB) CreateHostedCallerAPIKey(ctx context.Context, ownerUserID, createdBy, callerUserID, tenantID string, scopes []string, allowAlias bool, expiresAt *time.Time) (*models.HostedCallerAPIKey, string, error) {
	if strings.TrimSpace(ownerUserID) == "" || strings.TrimSpace(callerUserID) == "" {
		return nil, "", fmt.Errorf("owner_user_id and caller_user_id are required")
	}
	keyID, _, fullKey, err := generateHostedCallerAPIKeyMaterial()
	if err != nil {
		return nil, "", fmt.Errorf("generating caller key: %w", err)
	}
	keyHash := hashHostedCallerAPIKey(fullKey)
	now := time.Now().UTC()
	record := &models.HostedCallerAPIKey{
		ID:           uuid.New().String(),
		OwnerUserID:  ownerUserID,
		KeyID:        keyID,
		CallerUserID: strings.TrimSpace(callerUserID),
		TenantID:     strings.TrimSpace(tenantID),
		Scopes:       normalizeScopes(scopes),
		AllowAlias:   allowAlias,
		ExpiresAt:    expiresAt,
		CreatedBy:    strings.TrimSpace(createdBy),
		CreatedAt:    now,
	}
	var createdByParam interface{}
	if strings.TrimSpace(record.CreatedBy) == "" {
		createdByParam = nil
	} else {
		createdByParam = record.CreatedBy
	}
	_, err = db.pool.Exec(ctx, `
		INSERT INTO hosted_user_caller_api_keys (
			id, owner_user_id, key_id, key_hash, caller_user_id, tenant_id, scopes, allow_alias, expires_at, created_by, created_at
		) VALUES (
			$1, $2, $3, $4, $5, NULLIF($6, ''), $7, $8, $9, $10, $11
		)
	`,
		record.ID, record.OwnerUserID, record.KeyID, keyHash, record.CallerUserID, record.TenantID, record.Scopes, record.AllowAlias, record.ExpiresAt, createdByParam, record.CreatedAt,
	)
	if err != nil {
		return nil, "", fmt.Errorf("inserting hosted caller key: %w", err)
	}
	return record, fullKey, nil
}

func (db *DB) ListHostedCallerAPIKeysByUser(ctx context.Context, ownerUserID string) ([]models.HostedCallerAPIKey, error) {
	rows, err := db.pool.Query(ctx, `
		SELECT id, owner_user_id::text, key_id, caller_user_id, COALESCE(tenant_id, ''), scopes, allow_alias, expires_at, revoked_at, COALESCE(created_by::text, ''), created_at
		FROM hosted_user_caller_api_keys
		WHERE owner_user_id::text = $1
		ORDER BY created_at DESC
	`, ownerUserID)
	if err != nil {
		return nil, fmt.Errorf("listing hosted caller keys: %w", err)
	}
	defer rows.Close()
	out := make([]models.HostedCallerAPIKey, 0)
	for rows.Next() {
		var k models.HostedCallerAPIKey
		if err := rows.Scan(&k.ID, &k.OwnerUserID, &k.KeyID, &k.CallerUserID, &k.TenantID, &k.Scopes, &k.AllowAlias, &k.ExpiresAt, &k.RevokedAt, &k.CreatedBy, &k.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning hosted caller key: %w", err)
		}
		out = append(out, k)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating hosted caller keys: %w", err)
	}
	return out, nil
}

func (db *DB) RevokeHostedCallerAPIKey(ctx context.Context, ownerUserID, keyRecordID string) error {
	_, err := db.pool.Exec(ctx, `
		UPDATE hosted_user_caller_api_keys
		SET revoked_at = NOW()
		WHERE id = $1 AND owner_user_id::text = $2 AND revoked_at IS NULL
	`, keyRecordID, ownerUserID)
	if err != nil {
		return fmt.Errorf("revoking hosted caller key: %w", err)
	}
	return nil
}

func parseHostedCallerAPIKey(raw string) (keyID string, ok bool) {
	value := strings.TrimSpace(raw)
	if !strings.HasPrefix(value, "mkc_") {
		return "", false
	}
	parts := strings.Split(value, "_")
	if len(parts) != 3 || parts[1] == "" || parts[2] == "" {
		return "", false
	}
	return parts[1], true
}

func (db *DB) ValidateHostedCallerAPIKey(ctx context.Context, rawKey string) (*models.HostedCallerIdentity, error) {
	keyID, ok := parseHostedCallerAPIKey(rawKey)
	if !ok {
		return nil, nil
	}
	var storedHash string
	var callerUserID string
	var tenantID string
	var scopes []string
	var allowAlias bool
	err := db.pool.QueryRow(ctx, `
		SELECT key_hash, caller_user_id, COALESCE(tenant_id, ''), scopes, allow_alias
		FROM hosted_user_caller_api_keys
		WHERE key_id = $1
		  AND revoked_at IS NULL
		  AND (expires_at IS NULL OR expires_at > NOW())
	`, keyID).Scan(&storedHash, &callerUserID, &tenantID, &scopes, &allowAlias)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("validating hosted caller key: %w", err)
	}
	providedHash := hashHostedCallerAPIKey(strings.TrimSpace(rawKey))
	if len(storedHash) == 0 || len(storedHash) != len(providedHash) || subtle.ConstantTimeCompare([]byte(storedHash), []byte(providedHash)) != 1 {
		return nil, nil
	}
	return &models.HostedCallerIdentity{
		CallerUserID: callerUserID,
		TenantID:     tenantID,
		Scopes:       normalizeScopes(scopes),
		AllowAlias:   allowAlias,
	}, nil
}

func (db *DB) GetServerVersions(ctx context.Context, serverID string) ([]models.ServerVersion, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT id, server_id, version, COALESCE(release_notes, ''), snapshot, published_by, published_at 
		 FROM server_versions WHERE server_id = $1 ORDER BY published_at DESC`, serverID)
	if err != nil {
		return nil, fmt.Errorf("querying server versions: %w", err)
	}
	defer rows.Close()

	var versions []models.ServerVersion
	for rows.Next() {
		var v models.ServerVersion
		var publishedBy *string
		if err := rows.Scan(&v.ID, &v.ServerID, &v.Version, &v.ReleaseNotes, &v.Snapshot, &publishedBy, &v.PublishedAt); err != nil {
			return nil, fmt.Errorf("scanning server version: %w", err)
		}
		if publishedBy != nil {
			v.PublishedBy = *publishedBy
		}
		versions = append(versions, v)
	}

	return versions, nil
}

func (db *DB) GetServerVersion(ctx context.Context, serverID, version string) (*models.ServerVersion, error) {
	var v models.ServerVersion
	var publishedBy *string

	err := db.pool.QueryRow(ctx,
		`SELECT id, server_id, version, COALESCE(release_notes, ''), snapshot, published_by, published_at 
		 FROM server_versions WHERE server_id = $1 AND version = $2`, serverID, version).
		Scan(&v.ID, &v.ServerID, &v.Version, &v.ReleaseNotes, &v.Snapshot, &publishedBy, &v.PublishedAt)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("querying server version: %w", err)
	}

	if publishedBy != nil {
		v.PublishedBy = *publishedBy
	}

	return &v, nil
}

// ServerSlug returns a URL-safe slug from a server name (lowercase, spaces to hyphens, non-alphanumeric removed).
// Matches frontend serverSlug: replace spaces with '-', lowercase, remove non [a-z0-9-].
func ServerSlug(name string) string {
	s := strings.TrimSpace(name)
	s = strings.ToLower(s)
	var b strings.Builder
	lastHyphen := false
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastHyphen = false
		} else if (r == ' ' || r == '_' || r == '-') && !lastHyphen {
			b.WriteRune('-')
			lastHyphen = true
		}
	}
	return strings.Trim(b.String(), "-")
}

// GetServerByOwnerAndSlug returns the server owned by ownerID whose name slug matches serverSlug.
// Uses a lightweight id/name query plus a single GetServer for the match (avoids loading every server's tools/resources).
func (db *DB) GetServerByOwnerAndSlug(ctx context.Context, ownerID, serverSlug string) (*models.Server, error) {
	if strings.TrimSpace(ownerID) == "" {
		return nil, nil
	}
	slugWant := strings.ToLower(strings.TrimSpace(serverSlug))
	rows, err := db.pool.Query(ctx,
		`SELECT id, name FROM servers
		 WHERE owner_id IS NOT NULL AND owner_id::text = $1 AND hosted_virtual = false`,
		ownerID)
	if err != nil {
		return nil, fmt.Errorf("querying servers by owner for slug: %w", err)
	}
	defer rows.Close()
	var matchID string
	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, fmt.Errorf("scanning server row: %w", err)
		}
		if ServerSlug(name) == slugWant {
			matchID = id
			break
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating servers: %w", err)
	}
	if matchID == "" {
		return nil, nil
	}
	return db.GetServer(ctx, matchID)
}

func (db *DB) ListPublishedServers(ctx context.Context) ([]models.Server, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT id, name, description, version, icon, status, published_at, latest_version, owner_id, is_public, downloads, created_at, updated_at 
		 FROM servers WHERE status = 'published' AND is_public = true ORDER BY downloads DESC, updated_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("querying published servers: %w", err)
	}
	defer rows.Close()

	var servers []models.Server
	for rows.Next() {
		var s models.Server
		var icon, status, latestVersion *string
		var ownerID *string
		var publishedAt *time.Time
		var isPublic *bool
		var downloads *int
		if err := rows.Scan(&s.ID, &s.Name, &s.Description, &s.Version, &icon, &status, &publishedAt, &latestVersion, &ownerID, &isPublic, &downloads, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning server: %w", err)
		}
		if icon != nil {
			s.Icon = *icon
		} else {
			s.Icon = "bi-server"
		}
		if status != nil {
			s.Status = models.ServerStatus(*status)
		}
		if publishedAt != nil {
			s.PublishedAt = publishedAt
		}
		if latestVersion != nil {
			s.LatestVersion = *latestVersion
		}
		if ownerID != nil {
			s.OwnerID = *ownerID
		}
		if isPublic != nil {
			s.IsPublic = *isPublic
		}
		if downloads != nil {
			s.Downloads = *downloads
		}
		servers = append(servers, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating published servers: %w", err)
	}

	// Fetch tools and resources for all servers (for security score and display)
	if len(servers) > 0 {
		ids := make([]string, len(servers))
		for i := range servers {
			ids[i] = servers[i].ID
		}
		toolsBy, err := db.loadToolsByServerIDs(ctx, ids)
		if err != nil {
			return nil, fmt.Errorf("batch loading tools for published servers: %w", err)
		}
		resBy, err := db.loadResourcesByServerIDs(ctx, ids)
		if err != nil {
			return nil, fmt.Errorf("batch loading resources for published servers: %w", err)
		}
		for i := range servers {
			id := servers[i].ID
			servers[i].Tools = toolsBy[id]
			servers[i].Resources = resBy[id]
		}
	}

	return servers, nil
}

func (db *DB) UpdateServerPublicStatus(ctx context.Context, serverID string, isPublic bool) error {
	_, err := db.pool.Exec(ctx,
		`UPDATE servers SET is_public = $2, updated_at = NOW() WHERE id = $1`,
		serverID, isPublic)
	return err
}

func (db *DB) IncrementServerDownloads(ctx context.Context, serverID string) error {
	_, err := db.pool.Exec(ctx,
		`UPDATE servers SET downloads = downloads + 1 WHERE id = $1`,
		serverID)
	return err
}
