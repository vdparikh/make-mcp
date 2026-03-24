package database

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/vdparikh/make-mcp/backend/internal/models"
)

// uniqueValidServerUUIDs returns deduplicated server IDs that parse as UUIDs.
// Invalid or empty strings are skipped (caller treats them as missing servers).
func uniqueValidServerUUIDs(ids []string) []string {
	seen := make(map[string]struct{}, len(ids))
	var out []string
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, err := uuid.Parse(id); err != nil {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

// GetServerCoresByIDs loads server row metadata only (no tools, resources, or prompts).
// Missing IDs are omitted from the map. Invalid UUID strings are ignored.
func (db *DB) GetServerCoresByIDs(ctx context.Context, ids []string) (map[string]*models.Server, error) {
	unique := uniqueValidServerUUIDs(ids)
	if len(unique) == 0 {
		return map[string]*models.Server{}, nil
	}
	return db.loadServerCoresByIDs(ctx, unique)
}

// GetServersFullOrdered loads full server records (tools, resources, prompts) for the given IDs.
// Results are returned in the same order as orderedIDs; duplicate IDs produce independent copies
// (matching repeated GetServer calls). Uses a bounded number of queries instead of one GetServer per ID.
func (db *DB) GetServersFullOrdered(ctx context.Context, orderedIDs []string) ([]*models.Server, error) {
	if len(orderedIDs) == 0 {
		return nil, nil
	}
	for _, id := range orderedIDs {
		if _, err := uuid.Parse(id); err != nil {
			return nil, fmt.Errorf("invalid server id %q: %w", id, err)
		}
	}

	seen := make(map[string]struct{}, len(orderedIDs))
	var unique []string
	for _, id := range orderedIDs {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}

	byID, err := db.loadServerCoresByIDs(ctx, unique)
	if err != nil {
		return nil, err
	}
	for _, id := range unique {
		if _, ok := byID[id]; !ok {
			return nil, fmt.Errorf("server %s not found", id)
		}
	}

	toolsByServer, err := db.loadToolsByServerIDs(ctx, unique)
	if err != nil {
		return nil, err
	}
	resByServer, err := db.loadResourcesByServerIDs(ctx, unique)
	if err != nil {
		return nil, err
	}
	promptByServer, err := db.loadPromptsByServerIDs(ctx, unique)
	if err != nil {
		return nil, err
	}

	for _, id := range unique {
		s := byID[id]
		s.Tools = toolsByServer[id]
		if s.Tools == nil {
			s.Tools = []models.Tool{}
		}
		s.Resources = resByServer[id]
		if s.Resources == nil {
			s.Resources = []models.Resource{}
		}
		s.Prompts = promptByServer[id]
		if s.Prompts == nil {
			s.Prompts = []models.Prompt{}
		}
	}

	out := make([]*models.Server, 0, len(orderedIDs))
	for _, id := range orderedIDs {
		s, ok := byID[id]
		if !ok {
			return nil, fmt.Errorf("server %s not found", id)
		}
		out = append(out, cloneServerShallow(s))
	}
	return out, nil
}

func cloneServerShallow(s *models.Server) *models.Server {
	if s == nil {
		return nil
	}
	c := *s
	if len(s.Tools) > 0 {
		c.Tools = append([]models.Tool(nil), s.Tools...)
	} else {
		c.Tools = []models.Tool{}
	}
	if len(s.Resources) > 0 {
		c.Resources = append([]models.Resource(nil), s.Resources...)
	} else {
		c.Resources = []models.Resource{}
	}
	if len(s.Prompts) > 0 {
		c.Prompts = append([]models.Prompt(nil), s.Prompts...)
	} else {
		c.Prompts = []models.Prompt{}
	}
	return &c
}

func (db *DB) loadServerCoresByIDs(ctx context.Context, ids []string) (map[string]*models.Server, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT id, name, description, version, icon, status, published_at, latest_version, owner_id, is_public, downloads, hosted_virtual, auth_config, observability_reporting_key, hosted_access_key, hosted_auth_mode, require_caller_identity, env_profiles, hosted_security_config, hosted_runtime_config, created_at, updated_at
		 FROM servers WHERE id::text = ANY($1)`, ids)
	if err != nil {
		return nil, fmt.Errorf("querying servers by id: %w", err)
	}
	defer rows.Close()

	byID := make(map[string]*models.Server, len(ids))
	for rows.Next() {
		s, err := scanServerRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning server: %w", err)
		}
		byID[s.ID] = s
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating servers: %w", err)
	}
	return byID, nil
}

func (db *DB) loadToolsByServerIDs(ctx context.Context, serverIDs []string) (map[string][]models.Tool, error) {
	if len(serverIDs) == 0 {
		return map[string][]models.Tool{}, nil
	}
	rows, err := db.pool.Query(ctx,
		`SELECT id, server_id, name, description, input_schema, output_schema, execution_type, execution_config, context_fields, output_display, output_display_config, read_only_hint, destructive_hint, created_at, updated_at
		 FROM tools WHERE server_id::text = ANY($1) ORDER BY server_id, name`, serverIDs)
	if err != nil {
		return nil, fmt.Errorf("querying tools by server ids: %w", err)
	}
	defer rows.Close()

	out := make(map[string][]models.Tool)
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
		out[t.ServerID] = append(out[t.ServerID], t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating tools: %w", err)
	}
	return out, nil
}

func (db *DB) loadResourcesByServerIDs(ctx context.Context, serverIDs []string) (map[string][]models.Resource, error) {
	if len(serverIDs) == 0 {
		return map[string][]models.Resource{}, nil
	}
	rows, err := db.pool.Query(ctx,
		`SELECT id, server_id, name, uri, mime_type, handler, created_at, updated_at
		 FROM resources WHERE server_id::text = ANY($1) ORDER BY server_id, name`, serverIDs)
	if err != nil {
		return nil, fmt.Errorf("querying resources by server ids: %w", err)
	}
	defer rows.Close()

	out := make(map[string][]models.Resource)
	for rows.Next() {
		var r models.Resource
		if err := rows.Scan(&r.ID, &r.ServerID, &r.Name, &r.URI, &r.MimeType, &r.Handler, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning resource: %w", err)
		}
		out[r.ServerID] = append(out[r.ServerID], r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating resources: %w", err)
	}
	return out, nil
}

func (db *DB) loadPromptsByServerIDs(ctx context.Context, serverIDs []string) (map[string][]models.Prompt, error) {
	if len(serverIDs) == 0 {
		return map[string][]models.Prompt{}, nil
	}
	rows, err := db.pool.Query(ctx,
		`SELECT id, server_id, name, description, template, arguments, created_at, updated_at
		 FROM prompts WHERE server_id::text = ANY($1) ORDER BY server_id, name`, serverIDs)
	if err != nil {
		return nil, fmt.Errorf("querying prompts by server ids: %w", err)
	}
	defer rows.Close()

	out := make(map[string][]models.Prompt)
	for rows.Next() {
		var p models.Prompt
		if err := rows.Scan(&p.ID, &p.ServerID, &p.Name, &p.Description, &p.Template, &p.Arguments, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning prompt: %w", err)
		}
		out[p.ServerID] = append(out[p.ServerID], p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating prompts: %w", err)
	}
	return out, nil
}
