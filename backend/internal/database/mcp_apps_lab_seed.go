package database

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vdparikh/make-mcp/backend/internal/models"
)

// CreateMCPAppsLabServerForUser creates a focused template for exploring MCP Apps widgets
// (table, card, image, chart, map, form) in the test playground and in MCP Jam / Claude.
func (db *DB) CreateMCPAppsLabServerForUser(ctx context.Context, ownerID string) (*models.Server, error) {
	serverID := uuid.New().String()
	now := time.Now()

	_, err := db.pool.Exec(ctx, `
		INSERT INTO servers (id, name, description, version, icon, status, owner_id, is_public, downloads, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, serverID,
		"MCP Apps Lab",
		"Opinionated starter to explore MCP Apps UIs: table, card, image, bar chart, Google Maps embed, and interactive form + submit. Use the test playground or an MCP Apps host (e.g. MCP Jam). Chart sample JSON is loaded from this repo’s docs/samples via GitHub raw.",
		"1.0.0",
		"bi-grid-3x3-gap",
		"draft",
		ownerID,
		false,
		0,
		now, now)
	if err != nil {
		return nil, fmt.Errorf("creating MCP Apps Lab server: %w", err)
	}

	if err := db.seedMCPAppsLabTools(ctx, serverID, now); err != nil {
		return nil, fmt.Errorf("seeding MCP Apps Lab tools: %w", err)
	}
	if err := db.seedMCPAppsLabResources(ctx, serverID, now); err != nil {
		return nil, fmt.Errorf("seeding MCP Apps Lab resources: %w", err)
	}
	if err := db.seedMCPAppsLabPrompts(ctx, serverID, now); err != nil {
		return nil, fmt.Errorf("seeding MCP Apps Lab prompts: %w", err)
	}

	return db.GetServer(ctx, serverID)
}

type mcpAppsLabToolDef struct {
	name                string
	description         string
	executionType       string
	inputSchema         map[string]interface{}
	outputSchema        map[string]interface{}
	executionConfig     map[string]interface{}
	contextFields       []string
	outputDisplay       string
	outputDisplayConfig map[string]interface{}
	readOnly            bool
	destructive         bool
}

func normalizeToolODC(m map[string]interface{}) (json.RawMessage, error) {
	if len(m) == 0 {
		return nil, nil
	}
	b, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return models.NormalizeOutputDisplayConfigRaw(b)
}

func (db *DB) seedMCPAppsLabTools(ctx context.Context, serverID string, now time.Time) error {
	tools := []mcpAppsLabToolDef{
		{
			name:          "mcp_apps_card_quote",
			description:   "Random quote (Quotable) — MCP App **card** widget (content + author).",
			executionType: string(models.ExecutionTypeRestAPI),
			inputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
			outputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"content": map[string]interface{}{"type": "string"},
					"author":  map[string]interface{}{"type": "string"},
				},
			},
			executionConfig: map[string]interface{}{
				"url":    "https://api.quotable.io/random",
				"method": "GET",
				"headers": map[string]interface{}{
					"Accept": "application/json",
				},
			},
			outputDisplay: models.OutputDisplayCard,
			outputDisplayConfig: map[string]interface{}{
				"content_key": "content",
				"title_key":   "author",
			},
			readOnly:    true,
			destructive: false,
		},
		{
			name:          "mcp_apps_table_weather",
			description:   "Current weather (Open-Meteo) — MCP App **table** widget.",
			executionType: string(models.ExecutionTypeRestAPI),
			inputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"latitude":  map[string]interface{}{"type": "number", "description": "WGS84 latitude"},
					"longitude": map[string]interface{}{"type": "number", "description": "WGS84 longitude"},
				},
				"required": []string{"latitude", "longitude"},
			},
			outputSchema: map[string]interface{}{
				"type": "object",
			},
			executionConfig: map[string]interface{}{
				"url":    "https://api.open-meteo.com/v1/forecast?latitude={{latitude}}&longitude={{longitude}}&current_weather=true",
				"method": "GET",
				"headers": map[string]interface{}{
					"Accept": "application/json",
				},
			},
			outputDisplay: models.OutputDisplayTable,
			readOnly:      true,
			destructive:   false,
		},
		{
			name:          "mcp_apps_image_apod",
			description:   "NASA Astronomy Picture of the Day — MCP App **image** widget (uses APOD url + title).",
			executionType: string(models.ExecutionTypeRestAPI),
			inputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
			outputSchema: map[string]interface{}{
				"type": "object",
			},
			executionConfig: map[string]interface{}{
				"url":    "https://api.nasa.gov/planetary/apod?api_key=DEMO_KEY",
				"method": "GET",
				"headers": map[string]interface{}{
					"Accept": "application/json",
				},
			},
			outputDisplay: models.OutputDisplayImage,
			outputDisplayConfig: map[string]interface{}{
				"image_url_key": "url",
				"title_key":     "title",
			},
			readOnly:    true,
			destructive: false,
		},
		{
			name:          "mcp_apps_chart_sample",
			description:   "Loads a static JSON with labels + datasets (bar chart). Uses GitHub raw for this repo’s docs/samples/mcp-apps-chart-sample.json — MCP App **chart** widget.",
			executionType: string(models.ExecutionTypeRestAPI),
			inputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
			outputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"labels":   map[string]interface{}{"type": "array"},
					"datasets": map[string]interface{}{"type": "array"},
					"title":    map[string]interface{}{"type": "string"},
				},
			},
			executionConfig: map[string]interface{}{
				"url":    "https://raw.githubusercontent.com/vdparikh/make-mcp/main/docs/samples/mcp-apps-chart-sample.json",
				"method": "GET",
				"headers": map[string]interface{}{
					"Accept": "application/json",
				},
			},
			outputDisplay: models.OutputDisplayChart,
			outputDisplayConfig: map[string]interface{}{
				"chart_type": "bar",
			},
			readOnly:    true,
			destructive: false,
		},
		{
			name:          "mcp_apps_map_from_zip",
			description:   "US ZIP → latitude/longitude (Zippopotam) — MCP App **map** widget (uses places.0.latitude / places.0.longitude).",
			executionType: string(models.ExecutionTypeRestAPI),
			inputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"zip_code": map[string]interface{}{
						"type":        "string",
						"description": "US ZIP (e.g. 94102)",
					},
				},
				"required": []string{"zip_code"},
			},
			outputSchema: map[string]interface{}{
				"type": "object",
			},
			executionConfig: map[string]interface{}{
				"url":     "https://api.zippopotam.us/us/{{zip_code}}",
				"method":  "GET",
				"headers": map[string]interface{}{},
			},
			outputDisplay: models.OutputDisplayMap,
			outputDisplayConfig: map[string]interface{}{
				"lat_key": "places.0.latitude",
				"lng_key": "places.0.longitude",
				"zoom":    14,
			},
			readOnly:    true,
			destructive: false,
		},
		{
			name:          "mcp_apps_open_feedback_form",
			description:   "Loads sample user JSON — pairs with **mcp_apps_submit_feedback** for MCP App **form** widget (prefills name/email when present).",
			executionType: string(models.ExecutionTypeRestAPI),
			inputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
			outputSchema: map[string]interface{}{
				"type": "object",
			},
			executionConfig: map[string]interface{}{
				"url":     "https://jsonplaceholder.typicode.com/users/1",
				"method":  "GET",
				"headers": map[string]interface{}{"Accept": "application/json"},
			},
			outputDisplay: models.OutputDisplayForm,
			outputDisplayConfig: map[string]interface{}{
				"submit_tool":  "mcp_apps_submit_feedback",
				"title":        "Feedback",
				"submit_label": "Send",
				"fields": []map[string]interface{}{
					{"name": "name", "label": "Name", "type": "text", "required": true},
					{"name": "email", "label": "Email", "type": "text", "required": true},
					{"name": "message", "label": "Message", "type": "textarea", "required": false},
				},
			},
			readOnly:    true,
			destructive: false,
		},
		{
			name:          "mcp_apps_submit_feedback",
			description:   "POSTs form payload to httpbin — target of the form widget from **mcp_apps_open_feedback_form**.",
			executionType: string(models.ExecutionTypeRestAPI),
			inputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name":    map[string]interface{}{"type": "string"},
					"email":   map[string]interface{}{"type": "string"},
					"message": map[string]interface{}{"type": "string"},
				},
				"required": []string{"name", "email"},
			},
			outputSchema: map[string]interface{}{
				"type": "object",
			},
			executionConfig: map[string]interface{}{
				"url":    "https://httpbin.org/post",
				"method": "POST",
				"headers": map[string]interface{}{
					"Accept":       "application/json",
					"Content-Type": "application/json",
				},
			},
			outputDisplay: models.OutputDisplayJSON,
			readOnly:      false,
			destructive:   false,
		},
	}

	for _, tool := range tools {
		toolID := uuid.New().String()
		od := tool.outputDisplay
		if od == "" {
			od = models.OutputDisplayJSON
		}

		inputSchemaJSON, _ := json.Marshal(tool.inputSchema)
		outputSchemaJSON, _ := json.Marshal(tool.outputSchema)
		executionConfigJSON, _ := json.Marshal(tool.executionConfig)

		odcRaw, err := normalizeToolODC(tool.outputDisplayConfig)
		if err != nil {
			return fmt.Errorf("normalize output_display_config for %s: %w", tool.name, err)
		}

		_, err = db.pool.Exec(ctx, `
			INSERT INTO tools (id, server_id, name, description, input_schema, output_schema, execution_type, execution_config, context_fields, output_display, output_display_config, read_only_hint, destructive_hint, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		`, toolID, serverID, tool.name, tool.description, inputSchemaJSON, outputSchemaJSON, tool.executionType, executionConfigJSON, tool.contextFields, od, nullableJSON(odcRaw), tool.readOnly, tool.destructive, now, now)
		if err != nil {
			return fmt.Errorf("creating MCP Apps Lab tool %s: %w", tool.name, err)
		}
	}

	return nil
}

func (db *DB) seedMCPAppsLabResources(ctx context.Context, serverID string, now time.Time) error {
	resourceID := uuid.New().String()
	handler := map[string]interface{}{
		"type": "static",
		"data": "# MCP Apps Lab\n\n" +
			"This server showcases **MCP Apps** output modes supported by Make MCP:\n\n" +
			"| Widget | Tool |\n" +
			"|--------|------|\n" +
			"| Card | `mcp_apps_card_quote` |\n" +
			"| Table | `mcp_apps_table_weather` |\n" +
			"| Image | `mcp_apps_image_apod` |\n" +
			"| Chart | `mcp_apps_chart_sample` |\n" +
			"| Map | `mcp_apps_map_from_zip` |\n" +
			"| Form | `mcp_apps_open_feedback_form` → `mcp_apps_submit_feedback` |\n\n" +
			"**Chart** loads JSON from `docs/samples/mcp-apps-chart-sample.json` (GitHub raw). If that URL 404s (fork/branch), update the tool URL or host the file yourself.\n\n" +
			"Try tools in the **Test** tab or connect the generated server in **MCP Jam** / Claude with MCP Apps enabled.",
	}
	handlerJSON, _ := json.Marshal(handler)

	_, err := db.pool.Exec(ctx, `
		INSERT INTO resources (id, server_id, name, uri, mime_type, handler, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, resourceID, serverID, "mcp_apps_lab_readme", "mcp://mcp-apps-lab/readme", "text/markdown", handlerJSON, now, now)
	if err != nil {
		return fmt.Errorf("creating MCP Apps Lab resource: %w", err)
	}
	return nil
}

func (db *DB) seedMCPAppsLabPrompts(ctx context.Context, serverID string, now time.Time) error {
	promptID := uuid.New().String()
	args := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"focus": map[string]interface{}{
				"type":        "string",
				"description": "Optional: card, table, image, chart, map, or form",
			},
		},
	}
	argsJSON, _ := json.Marshal(args)

	_, err := db.pool.Exec(ctx, `
		INSERT INTO prompts (id, server_id, name, description, template, arguments, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, promptID, serverID,
		"explore_mcp_apps",
		"Suggests trying each MCP Apps widget in this lab server.",
		"Summarize what MCP Apps are and list the tools in this server that demonstrate each widget type (card, table, image, chart, map, form). If the user asked about a specific focus ({{focus}}), emphasize that widget.",
		argsJSON,
		now, now)
	if err != nil {
		return fmt.Errorf("creating MCP Apps Lab prompt: %w", err)
	}
	return nil
}
