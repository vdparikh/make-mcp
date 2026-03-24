package database

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vdparikh/make-mcp/backend/internal/models"
)

// CreateBlueprintServerForUser creates the "MCP Production Blueprint" server: diverse REST + webhook tools,
// discovery resources, orchestration prompts, multi-source context, and governance policies — aligned with
// how teams ship MCP servers in production (tools + resources + prompts + context + policy hints).
func (db *DB) CreateBlueprintServerForUser(ctx context.Context, ownerID string) (*models.Server, error) {
	serverID := uuid.New().String()
	now := time.Now()

	_, err := db.pool.Exec(ctx, `
		INSERT INTO servers (id, name, description, version, icon, status, owner_id, is_public, downloads, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, serverID,
		"MCP Production Blueprint",
		"Opinionated template for today’s MCP stack: HTTP tools (read + write paths), webhook egress, discovery resources, agent prompts, JWT/header context, and governance policies. Use it as a reference architecture before customizing.",
		"1.0.0",
		"bi-layers",
		"draft",
		ownerID,
		false,
		0,
		now, now)
	if err != nil {
		return nil, fmt.Errorf("creating blueprint server: %w", err)
	}

	if err := db.seedBlueprintTools(ctx, serverID, now); err != nil {
		return nil, fmt.Errorf("seeding blueprint tools: %w", err)
	}
	if err := db.seedBlueprintResources(ctx, serverID, now); err != nil {
		return nil, fmt.Errorf("seeding blueprint resources: %w", err)
	}
	if err := db.seedBlueprintPrompts(ctx, serverID, now); err != nil {
		return nil, fmt.Errorf("seeding blueprint prompts: %w", err)
	}
	if err := db.seedBlueprintContextConfigs(ctx, serverID, now); err != nil {
		return nil, fmt.Errorf("seeding blueprint context: %w", err)
	}

	return db.GetServer(ctx, serverID)
}

type blueprintToolDef struct {
	name            string
	description     string
	executionType   string
	inputSchema     map[string]interface{}
	outputSchema    map[string]interface{}
	executionConfig map[string]interface{}
	contextFields   []string
	outputDisplay   string
	readOnly        bool
	destructive     bool
	attachPolicy    bool
}

func (db *DB) seedBlueprintTools(ctx context.Context, serverID string, now time.Time) error {
	tools := []blueprintToolDef{
		{
			name:          "nasa_astronomy_picture",
			description:   "Fetch NASA’s Astronomy Picture of the Day (APOD) for today’s date. Uses the public DEMO_KEY tier.",
			executionType: string(models.ExecutionTypeRestAPI),
			inputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
			outputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"title":       map[string]interface{}{"type": "string"},
					"url":         map[string]interface{}{"type": "string"},
					"explanation": map[string]interface{}{"type": "string"},
					"date":        map[string]interface{}{"type": "string"},
					"media_type":  map[string]interface{}{"type": "string"},
				},
			},
			executionConfig: map[string]interface{}{
				"url":    "https://api.nasa.gov/planetary/apod?api_key=DEMO_KEY",
				"method": "GET",
				"headers": map[string]interface{}{
					"Accept": "application/json",
				},
			},
			outputDisplay: models.OutputDisplayCard,
			readOnly:      true,
			destructive:   false,
		},
		{
			name:          "open_meteo_current_weather",
			description:   "Current weather for latitude/longitude via Open-Meteo (no API key). Returns temperature, windspeed, and weather code.",
			executionType: string(models.ExecutionTypeRestAPI),
			inputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"latitude": map[string]interface{}{
						"type":        "number",
						"description": "WGS84 latitude (e.g. 37.77)",
					},
					"longitude": map[string]interface{}{
						"type":        "number",
						"description": "WGS84 longitude (e.g. -122.42)",
					},
				},
				"required": []string{"latitude", "longitude"},
			},
			outputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"latitude":          map[string]interface{}{"type": "number"},
					"longitude":         map[string]interface{}{"type": "number"},
					"generationtime_ms": map[string]interface{}{"type": "number"},
					"current_weather": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"temperature":   map[string]interface{}{"type": "number"},
							"windspeed":     map[string]interface{}{"type": "number"},
							"weathercode":   map[string]interface{}{"type": "number"},
							"time":          map[string]interface{}{"type": "string"},
							"winddirection": map[string]interface{}{"type": "number"},
						},
					},
				},
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
			name:          "random_inspirational_quote",
			description:   "Return a random quote (author + text) from Quotable — useful for UI copy or agent personality.",
			executionType: string(models.ExecutionTypeRestAPI),
			inputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
			outputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"_id":     map[string]interface{}{"type": "string"},
					"content": map[string]interface{}{"type": "string"},
					"author":  map[string]interface{}{"type": "string"},
					"tags":    map[string]interface{}{"type": "array"},
				},
			},
			executionConfig: map[string]interface{}{
				"url":    "https://api.quotable.io/random",
				"method": "GET",
				"headers": map[string]interface{}{
					"Accept": "application/json",
				},
			},
			outputDisplay: models.OutputDisplayJSON,
			readOnly:      true,
			destructive:   false,
		},
		{
			name:          "latest_exchange_rates",
			description:   "Latest FX rates for a base currency (USD, EUR, …) via Frankfurter (ECB-based, no API key).",
			executionType: string(models.ExecutionTypeRestAPI),
			inputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"base": map[string]interface{}{
						"type":        "string",
						"description": "ISO 4217 currency code (e.g. USD, GBP)",
					},
				},
				"required": []string{"base"},
			},
			outputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"amount": map[string]interface{}{"type": "number"},
					"base":   map[string]interface{}{"type": "string"},
					"date":   map[string]interface{}{"type": "string"},
					"rates": map[string]interface{}{
						"type": "object",
					},
				},
			},
			executionConfig: map[string]interface{}{
				"url":    "https://api.frankfurter.app/latest?from={{base}}",
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
			name:          "echo_httpbin_get",
			description:   "Read-only echo: GET request to httpbin returning args and headers — safe for connectivity checks.",
			executionType: string(models.ExecutionTypeRestAPI),
			inputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"ping": map[string]interface{}{
						"type":        "string",
						"description": "Arbitrary query marker",
					},
				},
			},
			outputSchema: map[string]interface{}{
				"type": "object",
			},
			executionConfig: map[string]interface{}{
				"url":    "https://httpbin.org/get?ping={{ping}}",
				"method": "GET",
				"headers": map[string]interface{}{
					"Accept": "application/json",
				},
			},
			outputDisplay: models.OutputDisplayJSON,
			readOnly:      true,
			destructive:   false,
		},
		{
			name:          "post_audit_event",
			description:   "POSTs a JSON payload to a request bin (httpbin) for audit / SIEM-style forwarding demos. Marked destructive: clients should confirm. Context (user_id, org) is injected when configured.",
			executionType: string(models.ExecutionTypeWebhook),
			inputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"event_type": map[string]interface{}{
						"type":        "string",
						"description": "e.g. tool_invocation, policy_violation",
					},
					"payload": map[string]interface{}{
						"type":        "object",
						"description": "Arbitrary JSON-serializable metadata",
					},
				},
				"required": []string{"event_type"},
			},
			outputSchema: map[string]interface{}{
				"type": "object",
			},
			executionConfig: map[string]interface{}{
				"url": "https://httpbin.org/post",
				"headers": map[string]interface{}{
					"Accept":       "application/json",
					"Content-Type": "application/json",
				},
			},
			contextFields: []string{"user_id", "organization_id"},
			outputDisplay: models.OutputDisplayJSON,
			readOnly:      false,
			destructive:   true,
			attachPolicy:  true,
		},
		{
			name:          "graphql_country_lookup",
			description:   "Learning sample (GraphQL): public Countries API — pass ISO 3166-1 alpha-2 code (e.g. US, DE). Test playground performs a real POST; generated server matches the same variables shape.",
			executionType: string(models.ExecutionTypeGraphQL),
			inputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"code": map[string]interface{}{
						"type":        "string",
						"description": "Two-letter country code",
					},
				},
				"required": []string{"code"},
			},
			outputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"country": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"name":     map[string]interface{}{"type": "string"},
							"capital":  map[string]interface{}{"type": "string"},
							"currency": map[string]interface{}{"type": "string"},
						},
					},
				},
			},
			executionConfig: map[string]interface{}{
				"url": "https://countries.trevorblades.com/graphql",
				"query": `query Country($code: ID!) {
  country(code: $code) {
    name
    capital
    currency
  }
}`,
				"headers": map[string]interface{}{
					"Accept": "application/json",
				},
			},
			outputDisplay: models.OutputDisplayCard,
			readOnly:      true,
			destructive:   false,
		},
		{
			name:          "sample_sql_inventory",
			description:   "Learning sample (Database): shows connection_string + parameterized query. Test playground returns fixed rows only (connection_string learning://blueprint-sql). Use a real DSN only in your own runtime.",
			executionType: string(models.ExecutionTypeDatabase),
			inputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"product_id": map[string]interface{}{
						"type":        "string",
						"description": "Example bind parameter (shown in resolved_query in test mode)",
					},
				},
			},
			outputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"columns": map[string]interface{}{"type": "array"},
					"rows":    map[string]interface{}{"type": "array"},
				},
			},
			executionConfig: map[string]interface{}{
				"connection_string": "learning://blueprint-sql",
				"query":             "SELECT product_id, name, stock FROM inventory WHERE product_id = {{product_id}}",
			},
			outputDisplay: models.OutputDisplayTable,
			readOnly:      true,
			destructive:   false,
		},
		{
			name:          "cli_echo_greeting",
			description:   "Learning sample (CLI): echo with {{message}} substitution and allowed_commands (matches generated Node allowlist). Test playground simulates resolution + policy only — no shell on the API host.",
			executionType: string(models.ExecutionTypeCLI),
			inputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"message": map[string]interface{}{
						"type":        "string",
						"description": "Printed by echo after download; preview shows resolved command only",
					},
				},
				"required": []string{"message"},
			},
			outputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"stdout": map[string]interface{}{"type": "string"},
				},
			},
			executionConfig: map[string]interface{}{
				"command":           "echo {{message}}",
				"timeout":           30000,
				"working_dir":       ".",
				"shell":             "/bin/bash",
				"allowed_commands":  []string{"echo"},
				"env":               map[string]interface{}{},
			},
			outputDisplay: models.OutputDisplayJSON,
			readOnly:      true,
			destructive:   false,
		},
		{
			name:          "sample_javascript_transform",
			description:   "Learning sample (JavaScript): documents an in-process transform pattern. Generated server uses a stub; implement in src/tools after download. Test playground returns snippet + input echo only.",
			executionType: string(models.ExecutionTypeJavaScript),
			inputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"n": map[string]interface{}{
						"type":        "number",
						"description": "Example numeric input",
					},
				},
				"required": []string{"n"},
			},
			outputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"doubled": map[string]interface{}{"type": "number"},
				},
			},
			executionConfig: map[string]interface{}{
				"runtime": "node",
				"snippet": "return { doubled: Number(input.n) * 2 };",
				"note":    "Replace the stub execute() body in the generated .ts tool file with your logic (or call fetch).",
			},
			outputDisplay: models.OutputDisplayJSON,
			readOnly:      true,
			destructive:   false,
		},
		{
			name:          "sample_python_string_stats",
			description:   "Learning sample (Python): same lifecycle as JavaScript — template stub in codegen; run Python only in an environment you control. Test playground is preview-only.",
			executionType: string(models.ExecutionTypePython),
			inputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"text": map[string]interface{}{
						"type":        "string",
						"description": "Sample string to analyze in your own worker",
					},
				},
				"required": []string{"text"},
			},
			outputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"length": map[string]interface{}{"type": "number"},
				},
			},
			executionConfig: map[string]interface{}{
				"runtime": "python3",
				"snippet": "return {\"length\": len(input[\"text\"]), \"words\": len(input[\"text\"].split())}",
				"note":    "Wire a subprocess or sidecar in your deployment; do not expose arbitrary python exec from shared APIs.",
			},
			outputDisplay: models.OutputDisplayJSON,
			readOnly:      true,
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

		_, err := db.pool.Exec(ctx, `
			INSERT INTO tools (id, server_id, name, description, input_schema, output_schema, execution_type, execution_config, context_fields, output_display, output_display_config, read_only_hint, destructive_hint, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		`, toolID, serverID, tool.name, tool.description, inputSchemaJSON, outputSchemaJSON, tool.executionType, executionConfigJSON, tool.contextFields, od, nil, tool.readOnly, tool.destructive, now, now)

		if err != nil {
			return fmt.Errorf("creating blueprint tool %s: %w", tool.name, err)
		}

		if tool.attachPolicy {
			if err := db.seedBlueprintAuditPolicies(ctx, toolID, now); err != nil {
				return fmt.Errorf("seeding blueprint policies: %w", err)
			}
		}
	}

	return nil
}

func (db *DB) seedBlueprintAuditPolicies(ctx context.Context, toolID string, now time.Time) error {
	policyID := uuid.New().String()
	_, err := db.pool.Exec(ctx, `
		INSERT INTO policies (id, tool_id, name, description, enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, policyID, toolID, "Audit egress governance", "Rate limits and role checks before POSTing audit-style events to external URLs.", true, now, now)
	if err != nil {
		return fmt.Errorf("creating blueprint policy: %w", err)
	}

	rules := []struct {
		ruleType   string
		config     map[string]interface{}
		priority   int
		failAction string
	}{
		{
			ruleType: string(models.PolicyRuleAllowedRoles),
			config: map[string]interface{}{
				"roles": []string{"admin", "auditor", "operator"},
			},
			priority:   1,
			failAction: "deny",
		},
		{
			ruleType: string(models.PolicyRuleRateLimit),
			config: map[string]interface{}{
				"max_calls":   30,
				"window_secs": 3600,
				"scope":       "user",
			},
			priority:   2,
			failAction: "deny",
		},
	}

	for _, rule := range rules {
		ruleID := uuid.New().String()
		configJSON, _ := json.Marshal(rule.config)
		_, err := db.pool.Exec(ctx, `
			INSERT INTO policy_rules (id, policy_id, type, config, priority, fail_action)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, ruleID, policyID, rule.ruleType, configJSON, rule.priority, rule.failAction)
		if err != nil {
			return fmt.Errorf("creating blueprint policy rule: %w", err)
		}
	}
	return nil
}

func (db *DB) seedBlueprintResources(ctx context.Context, serverID string, now time.Time) error {
	resources := []struct {
		name     string
		uri      string
		mimeType string
		handler  map[string]interface{}
	}{
		{
			name:     "server_discovery",
			uri:      "mcp://blueprint/discovery",
			mimeType: "text/markdown",
			handler: map[string]interface{}{
				"type": "static",
				"data": "# MCP Production Blueprint — Discovery\n\n" +
					"This server demonstrates **tools**, **resources**, **prompts**, and **context** the way production MCP deployments are structured in 2025–2026.\n\n" +
					"## Tools\n" +
					"- **Read path**: NASA APOD, Open-Meteo, quotes, FX rates, httpbin GET — use read_only_hint on safe tools.\n" +
					"- **Write / egress path**: post_audit_event uses a webhook and destructive_hint so clients can require confirmation.\n" +
					"- **Execution-type learning samples**: graphql_country_lookup (live GraphQL in test playground), sample_sql_inventory (fixed rows; `learning://blueprint-sql`), cli_echo_greeting (command resolution + allowlist check), sample_javascript_transform / sample_python_string_stats (preview-only; implement after download).\n" +
					"- **Presentation**: output_display uses JSON, table, and card shapes for MCP-capable UIs.\n\n" +
					"## Resources\n" +
					"Use resources for **stable URIs** (documentation, policy snippets, JSON manifests) that agents can subscribe to without invoking tools.\n\n" +
					"## Prompts\n" +
					"Prompts encode **orchestration** (planning, handoff, error recovery) — keep them versioned like APIs.\n\n" +
					"## Context\n" +
					"JWT + headers model how gateways attach **identity** before tools run; post_audit_event shows injected fields in payloads.\n\n" +
					"## Governance\n" +
					"Policies on egress tools illustrate **rate limits** and **roles**.",
			},
		},
		{
			name:     "tool_catalog_json",
			uri:      "mcp://blueprint/tools.json",
			mimeType: "application/json",
			handler: map[string]interface{}{
				"type": "static",
				"data": map[string]interface{}{
					"version":    "1.0",
					"categories": []string{"science", "weather", "content", "finance", "debug", "audit", "graphql", "database", "cli", "javascript", "python"},
					"notes":      "Replace this static manifest with your own generated catalog in production.",
				},
			},
		},
		{
			name:     "agent_safety_snippet",
			uri:      "mcp://blueprint/safety.md",
			mimeType: "text/markdown",
			handler: map[string]interface{}{
				"type": "static",
				"data": "## Agent safety checklist (template)\n\n" +
					"1. Prefer **read-only** tools until the user confirms a write.\n" +
					"2. Treat **destructive_hint: true** tools as requiring explicit user consent in the client.\n" +
					"3. Never send secrets in tool arguments; use server-side config or env.\n" +
					"4. Log **correlation IDs** (X-Request-ID header) when integrating with observability backends.",
			},
		},
	}

	for _, resource := range resources {
		resourceID := uuid.New().String()
		handlerJSON, _ := json.Marshal(resource.handler)
		_, err := db.pool.Exec(ctx, `
			INSERT INTO resources (id, server_id, name, uri, mime_type, handler, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`, resourceID, serverID, resource.name, resource.uri, resource.mimeType, handlerJSON, now, now)
		if err != nil {
			return fmt.Errorf("creating blueprint resource %s: %w", resource.name, err)
		}
	}
	return nil
}

func (db *DB) seedBlueprintPrompts(ctx context.Context, serverID string, now time.Time) error {
	prompts := []struct {
		name        string
		description string
		template    string
		arguments   map[string]interface{}
	}{
		{
			name:        "plan_tool_calls",
			description: "Before calling tools, produce a short numbered plan with assumptions and risks (modern agentic pattern).",
			template: `You are planning work for an MCP-connected assistant.

User goal: {{user_goal}}

Available context (optional): {{context_summary}}

Output:
1. **Objective** — one sentence.
2. **Tool strategy** — which tool categories you expect to use and why.
3. **Risks** — data privacy, cost, or irreversible side effects.
4. **First action** — the single next step.`,
			arguments: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"user_goal": map[string]interface{}{
						"type":        "string",
						"description": "What the user wants to achieve",
					},
					"context_summary": map[string]interface{}{
						"type":        "string",
						"description": "Optional pasted context or prior turns",
					},
				},
				"required": []string{"user_goal"},
			},
		},
		{
			name:        "session_handoff",
			description: "Summarize state for another agent or human operator (durable sessions / support handoff).",
			template: `Produce a concise handoff note.

## Facts
{{facts_json}}

## Completed
{{completed_steps}}

## Open questions
{{open_questions}}

## Recommended next owner action
(One bullet)`,
			arguments: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"facts_json": map[string]interface{}{
						"type":        "string",
						"description": "JSON or bullet text of known facts",
					},
					"completed_steps": map[string]interface{}{
						"type":        "string",
						"description": "What was already done",
					},
					"open_questions": map[string]interface{}{
						"type":        "string",
						"description": "Unresolved items",
					},
				},
				"required": []string{"facts_json", "completed_steps", "open_questions"},
			},
		},
		{
			name:        "recover_from_tool_error",
			description: "Turn a tool/API error into a user-safe explanation and retry strategy.",
			template: `Tool **{{tool_name}}** failed.

Raw error (for you, do not expose verbatim if it contains secrets):
{{error_text}}

Tool output snippet (if any):
{{output_snippet}}

Write:
1. **User-facing message** (no stack traces).
2. **Likely cause** (timeout, 4xx, validation, policy, etc.).
3. **Retry** — yes/no and what to change before retrying.`,
			arguments: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"tool_name": map[string]interface{}{
						"type": "string",
					},
					"error_text": map[string]interface{}{
						"type": "string",
					},
					"output_snippet": map[string]interface{}{
						"type": "string",
					},
				},
				"required": []string{"tool_name", "error_text"},
			},
		},
		{
			name:        "structured_answer",
			description: "Ask the model to answer with strict sections (good for RAG + tool grounding).",
			template: `Answer the question using the evidence below. If evidence is insufficient, say what is missing.

Question: {{question}}

Evidence:
{{evidence}}

Respond with:
### Answer
### Citations (quote short phrases from evidence)
### Confidence (high/medium/low)`,
			arguments: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"question": map[string]interface{}{
						"type": "string",
					},
					"evidence": map[string]interface{}{
						"type":        "string",
						"description": "Retrieved text or tool output",
					},
				},
				"required": []string{"question", "evidence"},
			},
		},
	}

	for _, prompt := range prompts {
		promptID := uuid.New().String()
		argumentsJSON, _ := json.Marshal(prompt.arguments)
		_, err := db.pool.Exec(ctx, `
			INSERT INTO prompts (id, server_id, name, description, template, arguments, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`, promptID, serverID, prompt.name, prompt.description, prompt.template, argumentsJSON, now, now)
		if err != nil {
			return fmt.Errorf("creating blueprint prompt %s: %w", prompt.name, err)
		}
	}
	return nil
}

func (db *DB) seedBlueprintContextConfigs(ctx context.Context, serverID string, now time.Time) error {
	configs := []struct {
		name       string
		sourceType string
		config     map[string]interface{}
	}{
		{
			name:       "JWT Claims",
			sourceType: "jwt",
			config: map[string]interface{}{
				"header_name": "Authorization",
				"claims_map": map[string]interface{}{
					"sub":         "user_id",
					"org_id":      "organization_id",
					"roles":       "roles",
					"permissions": "permissions",
				},
			},
		},
		{
			name:       "Session header",
			sourceType: "header",
			config: map[string]interface{}{
				"header_name":  "X-Session-ID",
				"target_field": "session_id",
			},
		},
		{
			name:       "Request correlation",
			sourceType: "header",
			config: map[string]interface{}{
				"header_name":  "X-Request-ID",
				"target_field": "request_id",
			},
		},
		{
			name:       "Workspace scope",
			sourceType: "header",
			config: map[string]interface{}{
				"header_name":  "X-Workspace-ID",
				"target_field": "workspace_id",
			},
		},
	}

	for _, cfg := range configs {
		configID := uuid.New().String()
		configJSON, _ := json.Marshal(cfg.config)
		_, err := db.pool.Exec(ctx, `
			INSERT INTO context_configs (id, server_id, name, source_type, config, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, configID, serverID, cfg.name, cfg.sourceType, configJSON, now, now)
		if err != nil {
			return fmt.Errorf("creating blueprint context %s: %w", cfg.name, err)
		}
	}
	return nil
}
