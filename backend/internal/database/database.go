package database

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
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
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,
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
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_tools_server_id ON tools(server_id)`,
		`CREATE INDEX IF NOT EXISTS idx_resources_server_id ON resources(server_id)`,
		`CREATE INDEX IF NOT EXISTS idx_prompts_server_id ON prompts(server_id)`,
		`CREATE INDEX IF NOT EXISTS idx_policies_tool_id ON policies(tool_id)`,
		`CREATE INDEX IF NOT EXISTS idx_tool_executions_tool_id ON tool_executions(tool_id)`,
		`CREATE INDEX IF NOT EXISTS idx_tool_executions_created_at ON tool_executions(created_at)`,
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
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if server.Version == "" {
		server.Version = "1.0.0"
	}

	_, err := db.pool.Exec(ctx,
		`INSERT INTO servers (id, name, description, version, created_at, updated_at) 
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		server.ID, server.Name, server.Description, server.Version, server.CreatedAt, server.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("inserting server: %w", err)
	}

	return server, nil
}

func (db *DB) GetServer(ctx context.Context, id string) (*models.Server, error) {
	var server models.Server
	var authConfig []byte

	err := db.pool.QueryRow(ctx,
		`SELECT id, name, description, version, auth_config, created_at, updated_at 
		 FROM servers WHERE id = $1`, id).
		Scan(&server.ID, &server.Name, &server.Description, &server.Version, &authConfig, &server.CreatedAt, &server.UpdatedAt)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("querying server: %w", err)
	}

	if authConfig != nil {
		server.AuthConfig = authConfig
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

	return &server, nil
}

func (db *DB) ListServers(ctx context.Context) ([]models.Server, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT id, name, description, version, created_at, updated_at 
		 FROM servers ORDER BY updated_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("querying servers: %w", err)
	}
	defer rows.Close()

	var servers []models.Server
	for rows.Next() {
		var s models.Server
		if err := rows.Scan(&s.ID, &s.Name, &s.Description, &s.Version, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning server: %w", err)
		}
		servers = append(servers, s)
	}

	// Fetch tools, resources, and prompts for each server
	for i := range servers {
		tools, err := db.GetToolsByServer(ctx, servers[i].ID)
		if err != nil {
			return nil, fmt.Errorf("getting tools for server %s: %w", servers[i].ID, err)
		}
		servers[i].Tools = tools

		resources, err := db.GetResourcesByServer(ctx, servers[i].ID)
		if err != nil {
			return nil, fmt.Errorf("getting resources for server %s: %w", servers[i].ID, err)
		}
		servers[i].Resources = resources

		prompts, err := db.GetPromptsByServer(ctx, servers[i].ID)
		if err != nil {
			return nil, fmt.Errorf("getting prompts for server %s: %w", servers[i].ID, err)
		}
		servers[i].Prompts = prompts
	}

	return servers, nil
}

func (db *DB) UpdateServer(ctx context.Context, id string, req models.CreateServerRequest) (*models.Server, error) {
	_, err := db.pool.Exec(ctx,
		`UPDATE servers SET name = $2, description = $3, version = $4, updated_at = NOW() WHERE id = $1`,
		id, req.Name, req.Description, req.Version)

	if err != nil {
		return nil, fmt.Errorf("updating server: %w", err)
	}

	return db.GetServer(ctx, id)
}

func (db *DB) DeleteServer(ctx context.Context, id string) error {
	_, err := db.pool.Exec(ctx, `DELETE FROM servers WHERE id = $1`, id)
	return err
}

// Tool operations
func (db *DB) CreateTool(ctx context.Context, req models.CreateToolRequest) (*models.Tool, error) {
	tool := &models.Tool{
		ID:              uuid.New().String(),
		ServerID:        req.ServerID,
		Name:            req.Name,
		Description:     req.Description,
		InputSchema:     req.InputSchema,
		OutputSchema:    req.OutputSchema,
		ExecutionType:   req.ExecutionType,
		ExecutionConfig: req.ExecutionConfig,
		ContextFields:   req.ContextFields,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	_, err := db.pool.Exec(ctx,
		`INSERT INTO tools (id, server_id, name, description, input_schema, output_schema, execution_type, execution_config, context_fields, created_at, updated_at) 
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		tool.ID, tool.ServerID, tool.Name, tool.Description, tool.InputSchema, tool.OutputSchema, tool.ExecutionType, tool.ExecutionConfig, tool.ContextFields, tool.CreatedAt, tool.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("inserting tool: %w", err)
	}

	return tool, nil
}

func (db *DB) GetTool(ctx context.Context, id string) (*models.Tool, error) {
	var tool models.Tool

	err := db.pool.QueryRow(ctx,
		`SELECT id, server_id, name, description, input_schema, output_schema, execution_type, execution_config, context_fields, created_at, updated_at 
		 FROM tools WHERE id = $1`, id).
		Scan(&tool.ID, &tool.ServerID, &tool.Name, &tool.Description, &tool.InputSchema, &tool.OutputSchema, &tool.ExecutionType, &tool.ExecutionConfig, &tool.ContextFields, &tool.CreatedAt, &tool.UpdatedAt)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("querying tool: %w", err)
	}

	return &tool, nil
}

func (db *DB) GetToolsByServer(ctx context.Context, serverID string) ([]models.Tool, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT id, server_id, name, description, input_schema, output_schema, execution_type, execution_config, context_fields, created_at, updated_at 
		 FROM tools WHERE server_id = $1 ORDER BY name`, serverID)
	if err != nil {
		return nil, fmt.Errorf("querying tools: %w", err)
	}
	defer rows.Close()

	var tools []models.Tool
	for rows.Next() {
		var t models.Tool
		if err := rows.Scan(&t.ID, &t.ServerID, &t.Name, &t.Description, &t.InputSchema, &t.OutputSchema, &t.ExecutionType, &t.ExecutionConfig, &t.ContextFields, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning tool: %w", err)
		}
		tools = append(tools, t)
	}

	return tools, nil
}

func (db *DB) UpdateTool(ctx context.Context, id string, req models.CreateToolRequest) (*models.Tool, error) {
	_, err := db.pool.Exec(ctx,
		`UPDATE tools SET name = $2, description = $3, input_schema = $4, output_schema = $5, execution_type = $6, execution_config = $7, context_fields = $8, updated_at = NOW() WHERE id = $1`,
		id, req.Name, req.Description, req.InputSchema, req.OutputSchema, req.ExecutionType, req.ExecutionConfig, req.ContextFields)

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
		 FROM policies WHERE tool_id = $1`, toolID)
	if err != nil {
		return nil, fmt.Errorf("querying policies: %w", err)
	}
	defer rows.Close()

	var policies []models.Policy
	for rows.Next() {
		var p models.Policy
		if err := rows.Scan(&p.ID, &p.ToolID, &p.Name, &p.Description, &p.Enabled, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning policy: %w", err)
		}

		ruleRows, err := db.pool.Query(ctx,
			`SELECT id, policy_id, type, config, priority, fail_action FROM policy_rules WHERE policy_id = $1 ORDER BY priority`, p.ID)
		if err != nil {
			return nil, fmt.Errorf("querying rules: %w", err)
		}

		for ruleRows.Next() {
			var r models.PolicyRule
			if err := ruleRows.Scan(&r.ID, &r.PolicyID, &r.Type, &r.Config, &r.Priority, &r.FailAction); err != nil {
				ruleRows.Close()
				return nil, fmt.Errorf("scanning rule: %w", err)
			}
			p.Rules = append(p.Rules, r)
		}
		ruleRows.Close()

		policies = append(policies, p)
	}

	return policies, nil
}

func (db *DB) DeletePolicy(ctx context.Context, id string) error {
	_, err := db.pool.Exec(ctx, `DELETE FROM policies WHERE id = $1`, id)
	return err
}

// Tool Execution logging
func (db *DB) LogToolExecution(ctx context.Context, exec *models.ToolExecution) error {
	exec.ID = uuid.New().String()
	exec.CreatedAt = time.Now()

	_, err := db.pool.Exec(ctx,
		`INSERT INTO tool_executions (id, tool_id, server_id, input, output, error, status_code, duration_ms, success, healing_applied, created_at) 
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		exec.ID, exec.ToolID, exec.ServerID, exec.Input, exec.Output, exec.Error, exec.StatusCode, exec.DurationMs, exec.Success, exec.HealingApplied, exec.CreatedAt)

	return err
}

func (db *DB) GetToolExecutions(ctx context.Context, toolID string, limit int) ([]models.ToolExecution, error) {
	if limit <= 0 {
		limit = 100
	}

	rows, err := db.pool.Query(ctx,
		`SELECT id, tool_id, server_id, input, output, error, status_code, duration_ms, success, healing_applied, created_at 
		 FROM tool_executions WHERE tool_id = $1 ORDER BY created_at DESC LIMIT $2`, toolID, limit)
	if err != nil {
		return nil, fmt.Errorf("querying executions: %w", err)
	}
	defer rows.Close()

	var executions []models.ToolExecution
	for rows.Next() {
		var e models.ToolExecution
		if err := rows.Scan(&e.ID, &e.ToolID, &e.ServerID, &e.Input, &e.Output, &e.Error, &e.StatusCode, &e.DurationMs, &e.Success, &e.HealingApplied, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning execution: %w", err)
		}
		executions = append(executions, e)
	}

	return executions, nil
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

// Server Composition
func (db *DB) CreateComposition(ctx context.Context, name, description string, serverIDs []string) (*models.ServerComposition, error) {
	comp := &models.ServerComposition{
		ID:          uuid.New().String(),
		Name:        name,
		Description: description,
		ServerIDs:   serverIDs,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	_, err := db.pool.Exec(ctx,
		`INSERT INTO server_compositions (id, name, description, server_ids, created_at, updated_at) 
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		comp.ID, comp.Name, comp.Description, comp.ServerIDs, comp.CreatedAt, comp.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("inserting composition: %w", err)
	}

	return comp, nil
}

func (db *DB) ListCompositions(ctx context.Context) ([]models.ServerComposition, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT id, name, description, server_ids, created_at, updated_at 
		 FROM server_compositions ORDER BY updated_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("querying compositions: %w", err)
	}
	defer rows.Close()

	var compositions []models.ServerComposition
	for rows.Next() {
		var c models.ServerComposition
		if err := rows.Scan(&c.ID, &c.Name, &c.Description, &c.ServerIDs, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning composition: %w", err)
		}
		compositions = append(compositions, c)
	}

	return compositions, nil
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
		 FROM context_configs WHERE server_id = $1`, serverID)
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
