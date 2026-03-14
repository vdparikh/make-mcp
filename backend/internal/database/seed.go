package database

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/vdparikh/make-mcp/backend/internal/auth"
)

// nullUUID returns nil for empty string so NULL can be stored in UUID columns.
func nullUUID(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// SeedDemoData creates initial demo data if the database is empty
func (db *DB) SeedDemoData(ctx context.Context) error {
	// Check if any servers exist
	var count int
	err := db.pool.QueryRow(ctx, "SELECT COUNT(*) FROM servers").Scan(&count)
	if err != nil {
		return fmt.Errorf("checking server count: %w", err)
	}

	if count > 0 {
		log.Println("Database already has data, skipping seed")
		return nil
	}

	log.Println("Seeding demo data...")
	now := time.Now()

	// Ensure a default user exists so the demo server can be owned
	var defaultUserID string
	var userCount int
	if err := db.pool.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&userCount); err != nil {
		return fmt.Errorf("checking users: %w", err)
	}
	if userCount == 0 {
		defaultUserID = uuid.New().String()
		hash, err := auth.HashPassword("demo123")
		if err != nil {
			return fmt.Errorf("hashing default password: %w", err)
		}
		_, err = db.pool.Exec(ctx, `
			INSERT INTO users (id, email, name, password_hash, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, defaultUserID, "demo@example.com", "Demo User", hash, now, now)
		if err != nil {
			return fmt.Errorf("creating default user: %w", err)
		}
		log.Println("Created default user: demo@example.com / demo123")
	}

	// Create demo server (owned by default user if we just created one)
	serverID := uuid.New().String()
	_, err = db.pool.Exec(ctx, `
		INSERT INTO servers (id, name, description, version, owner_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, serverID, "Demo API Toolkit", "A fully functional demo MCP server showcasing location lookup, weather info, and utility tools. Use this as a model for building your own servers.", "1.0.0", nullUUID(defaultUserID), now, now)

	if err != nil {
		return fmt.Errorf("creating demo server: %w", err)
	}

	// Create demo tools
	if err := db.seedDemoTools(ctx, serverID, now); err != nil {
		return fmt.Errorf("seeding tools: %w", err)
	}

	// Create demo resources
	if err := db.seedDemoResources(ctx, serverID, now); err != nil {
		return fmt.Errorf("seeding resources: %w", err)
	}

	// Create demo prompts
	if err := db.seedDemoPrompts(ctx, serverID, now); err != nil {
		return fmt.Errorf("seeding prompts: %w", err)
	}

	// Create demo context configs
	if err := db.seedDemoContextConfigs(ctx, serverID, now); err != nil {
		return fmt.Errorf("seeding context configs: %w", err)
	}

	log.Println("Demo data seeded successfully!")
	return nil
}

func (db *DB) seedDemoTools(ctx context.Context, serverID string, now time.Time) error {
	tools := []struct {
		name            string
		description     string
		executionType   string
		inputSchema     map[string]interface{}
		outputSchema    map[string]interface{}
		executionConfig map[string]interface{}
		contextFields   []string
	}{
		{
			name:          "get_location_by_zip",
			description:   "Get location details (city, state, coordinates) for a US zip code. Returns place name, state, latitude, and longitude.",
			executionType: "rest_api",
			inputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"zip_code": map[string]interface{}{
						"type":        "string",
						"description": "US ZIP code (e.g., 94538, 10001)",
					},
				},
				"required": []string{"zip_code"},
			},
			outputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"country":   map[string]interface{}{"type": "string"},
					"post code": map[string]interface{}{"type": "string"},
					"places": map[string]interface{}{
						"type": "array",
						"items": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"place name": map[string]interface{}{"type": "string"},
								"state":      map[string]interface{}{"type": "string"},
								"latitude":   map[string]interface{}{"type": "string"},
								"longitude":  map[string]interface{}{"type": "string"},
							},
						},
					},
				},
			},
			executionConfig: map[string]interface{}{
				"url":     "https://api.zippopotam.us/us/{{zip_code}}",
				"method":  "GET",
				"headers": map[string]interface{}{},
			},
			contextFields: []string{},
		},
		{
			name:          "get_random_user",
			description:   "Generate a random user profile with name, email, location, and profile picture. Useful for testing and demo purposes.",
			executionType: "rest_api",
			inputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"nationality": map[string]interface{}{
						"type":        "string",
						"description": "Two-letter country code (e.g., US, GB, FR). Leave empty for random.",
					},
				},
			},
			outputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"results": map[string]interface{}{
						"type": "array",
						"items": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"name":     map[string]interface{}{"type": "object"},
								"email":    map[string]interface{}{"type": "string"},
								"location": map[string]interface{}{"type": "object"},
								"picture":  map[string]interface{}{"type": "object"},
							},
						},
					},
				},
			},
			executionConfig: map[string]interface{}{
				"url":     "https://randomuser.me/api/?nat={{nationality}}",
				"method":  "GET",
				"headers": map[string]interface{}{},
			},
			contextFields: []string{},
		},
		{
			name:          "get_ip_info",
			description:   "Get geolocation and ISP information for an IP address. Returns country, city, ISP, and coordinates.",
			executionType: "rest_api",
			inputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"ip_address": map[string]interface{}{
						"type":        "string",
						"description": "IPv4 or IPv6 address to lookup (e.g., 8.8.8.8)",
					},
				},
				"required": []string{"ip_address"},
			},
			outputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"ip":       map[string]interface{}{"type": "string"},
					"city":     map[string]interface{}{"type": "string"},
					"region":   map[string]interface{}{"type": "string"},
					"country":  map[string]interface{}{"type": "string"},
					"loc":      map[string]interface{}{"type": "string"},
					"org":      map[string]interface{}{"type": "string"},
					"postal":   map[string]interface{}{"type": "string"},
					"timezone": map[string]interface{}{"type": "string"},
				},
			},
			executionConfig: map[string]interface{}{
				"url":     "https://ipinfo.io/{{ip_address}}/json",
				"method":  "GET",
				"headers": map[string]interface{}{},
			},
			contextFields: []string{},
		},
		{
			name:          "get_joke",
			description:   "Get a random programming joke or dad joke. Great for lightening the mood!",
			executionType: "rest_api",
			inputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
			outputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"id":     map[string]interface{}{"type": "string"},
					"joke":   map[string]interface{}{"type": "string"},
					"status": map[string]interface{}{"type": "integer"},
				},
			},
			executionConfig: map[string]interface{}{
				"url":    "https://icanhazdadjoke.com/",
				"method": "GET",
				"headers": map[string]interface{}{
					"Accept": "application/json",
				},
			},
			contextFields: []string{},
		},
		{
			name:          "get_github_user",
			description:   "Get public profile information for a GitHub user including repos, followers, and bio.",
			executionType: "rest_api",
			inputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"username": map[string]interface{}{
						"type":        "string",
						"description": "GitHub username (e.g., octocat)",
					},
				},
				"required": []string{"username"},
			},
			outputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"login":        map[string]interface{}{"type": "string"},
					"name":         map[string]interface{}{"type": "string"},
					"bio":          map[string]interface{}{"type": "string"},
					"public_repos": map[string]interface{}{"type": "integer"},
					"followers":    map[string]interface{}{"type": "integer"},
					"following":    map[string]interface{}{"type": "integer"},
					"avatar_url":   map[string]interface{}{"type": "string"},
					"html_url":     map[string]interface{}{"type": "string"},
				},
			},
			executionConfig: map[string]interface{}{
				"url":    "https://api.github.com/users/{{username}}",
				"method": "GET",
				"headers": map[string]interface{}{
					"Accept":     "application/vnd.github.v3+json",
					"User-Agent": "MCP-Server-Builder",
				},
			},
			contextFields: []string{},
		},
		{
			name:          "validate_email",
			description:   "Validate if an email address format is correct and check if the domain has valid MX records.",
			executionType: "rest_api",
			inputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"email": map[string]interface{}{
						"type":        "string",
						"description": "Email address to validate",
					},
				},
				"required": []string{"email"},
			},
			outputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"valid":      map[string]interface{}{"type": "boolean"},
					"validators": map[string]interface{}{"type": "object"},
				},
			},
			executionConfig: map[string]interface{}{
				"url":     "https://www.disify.com/api/email/{{email}}",
				"method":  "GET",
				"headers": map[string]interface{}{},
			},
			contextFields: []string{},
		},
		{
			name:          "get_country_info",
			description:   "Get detailed information about a country including capital, population, currencies, and languages.",
			executionType: "rest_api",
			inputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"country_name": map[string]interface{}{
						"type":        "string",
						"description": "Country name (e.g., France, United States, Japan)",
					},
				},
				"required": []string{"country_name"},
			},
			outputSchema: map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"name":       map[string]interface{}{"type": "object"},
						"capital":    map[string]interface{}{"type": "array"},
						"population": map[string]interface{}{"type": "integer"},
						"currencies": map[string]interface{}{"type": "object"},
						"languages":  map[string]interface{}{"type": "object"},
						"flag":       map[string]interface{}{"type": "string"},
					},
				},
			},
			executionConfig: map[string]interface{}{
				"url":     "https://restcountries.com/v3.1/name/{{country_name}}",
				"method":  "GET",
				"headers": map[string]interface{}{},
			},
			contextFields: []string{},
		},
		{
			name:          "get_secure_customer_data",
			description:   "Get customer data with automatic context injection. Demonstrates the Context Engine feature - user_id and org_id are automatically added from JWT/headers.",
			executionType: "rest_api",
			inputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"customer_id": map[string]interface{}{
						"type":        "string",
						"description": "Customer ID to lookup",
					},
				},
				"required": []string{"customer_id"},
			},
			outputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"message": map[string]interface{}{"type": "string"},
					"input":   map[string]interface{}{"type": "object"},
				},
			},
			executionConfig: map[string]interface{}{
				"url":     "https://httpbin.org/anything",
				"method":  "POST",
				"headers": map[string]interface{}{},
			},
			contextFields: []string{"user_id", "organization_id", "permissions"},
		},
	}

	for _, tool := range tools {
		toolID := uuid.New().String()

		inputSchemaJSON, _ := json.Marshal(tool.inputSchema)
		outputSchemaJSON, _ := json.Marshal(tool.outputSchema)
		executionConfigJSON, _ := json.Marshal(tool.executionConfig)

		_, err := db.pool.Exec(ctx, `
			INSERT INTO tools (id, server_id, name, description, input_schema, output_schema, execution_type, execution_config, context_fields, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		`, toolID, serverID, tool.name, tool.description, inputSchemaJSON, outputSchemaJSON, tool.executionType, executionConfigJSON, tool.contextFields, now, now)

		if err != nil {
			return fmt.Errorf("creating tool %s: %w", tool.name, err)
		}

		// Create a sample policy for the secure_customer_data tool
		if tool.name == "get_secure_customer_data" {
			if err := db.seedDemoPolicies(ctx, toolID, now); err != nil {
				return fmt.Errorf("seeding policies: %w", err)
			}
		}
	}

	return nil
}

func (db *DB) seedDemoResources(ctx context.Context, serverID string, now time.Time) error {
	resources := []struct {
		name     string
		uri      string
		mimeType string
		handler  map[string]interface{}
	}{
		{
			name:     "api_documentation",
			uri:      "mcp://docs/api",
			mimeType: "text/markdown",
			handler: map[string]interface{}{
				"type": "static",
				"data": "# Demo API Toolkit Documentation\n\nThis MCP server provides various utility tools:\n\n## Available Tools\n\n1. **get_location_by_zip** - Look up location by ZIP code\n2. **get_random_user** - Generate random user profiles\n3. **get_ip_info** - IP geolocation lookup\n4. **get_joke** - Get random dad jokes\n5. **get_github_user** - GitHub profile lookup\n6. **validate_email** - Email validation\n7. **get_country_info** - Country information\n8. **get_secure_customer_data** - Demo of context injection\n\n## Authentication\n\nMost tools use free public APIs and don't require authentication.",
			},
		},
		{
			name:     "sample_data",
			uri:      "mcp://data/samples",
			mimeType: "application/json",
			handler: map[string]interface{}{
				"type": "static",
				"data": map[string]interface{}{
					"sample_zip_codes":    []string{"94538", "10001", "90210", "60601", "02101"},
					"sample_github_users": []string{"octocat", "torvalds", "gaearon"},
					"sample_ip_addresses": []string{"8.8.8.8", "1.1.1.1", "208.67.222.222"},
					"sample_countries":    []string{"France", "Japan", "Brazil", "Australia"},
				},
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
			return fmt.Errorf("creating resource %s: %w", resource.name, err)
		}
	}

	return nil
}

func (db *DB) seedDemoPrompts(ctx context.Context, serverID string, now time.Time) error {
	prompts := []struct {
		name        string
		description string
		template    string
		arguments   map[string]interface{}
	}{
		{
			name:        "location_summary",
			description: "Generate a summary of a location based on ZIP code lookup",
			template:    "Based on the ZIP code {{zip_code}}, provide a brief summary of this location including:\n\n1. City and state name\n2. Geographic region of the US\n3. Any notable characteristics of this area\n\nLocation data: {{location_data}}",
			arguments: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"zip_code": map[string]interface{}{
						"type":        "string",
						"description": "The ZIP code that was looked up",
					},
					"location_data": map[string]interface{}{
						"type":        "string",
						"description": "The JSON response from the location lookup",
					},
				},
				"required": []string{"zip_code", "location_data"},
			},
		},
		{
			name:        "user_profile_analysis",
			description: "Analyze a user profile and generate insights",
			template:    "Analyze the following user profile and provide insights:\n\nProfile: {{profile_data}}\n\nPlease provide:\n1. A brief bio based on available information\n2. Demographic insights\n3. Any interesting observations",
			arguments: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"profile_data": map[string]interface{}{
						"type":        "string",
						"description": "JSON user profile data",
					},
				},
				"required": []string{"profile_data"},
			},
		},
		{
			name:        "country_comparison",
			description: "Compare two countries based on their data",
			template:    "Compare the following two countries:\n\nCountry 1: {{country1_name}}\nData: {{country1_data}}\n\nCountry 2: {{country2_name}}\nData: {{country2_data}}\n\nProvide a comparison including:\n1. Population difference\n2. Geographic size\n3. Cultural highlights\n4. Economic indicators if available",
			arguments: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"country1_name": map[string]interface{}{"type": "string"},
					"country1_data": map[string]interface{}{"type": "string"},
					"country2_name": map[string]interface{}{"type": "string"},
					"country2_data": map[string]interface{}{"type": "string"},
				},
				"required": []string{"country1_name", "country1_data", "country2_name", "country2_data"},
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
			return fmt.Errorf("creating prompt %s: %w", prompt.name, err)
		}
	}

	return nil
}

func (db *DB) seedDemoContextConfigs(ctx context.Context, serverID string, now time.Time) error {
	configs := []struct {
		name       string
		sourceType string
		config     map[string]interface{}
	}{
		{
			name:       "JWT User Context",
			sourceType: "jwt",
			config: map[string]interface{}{
				"header_name": "Authorization",
				"claims_map": map[string]interface{}{
					"sub":         "user_id",
					"org":         "organization_id",
					"roles":       "roles",
					"permissions": "permissions",
				},
			},
		},
		{
			name:       "Header User ID",
			sourceType: "header",
			config: map[string]interface{}{
				"header_name":  "X-User-ID",
				"target_field": "user_id",
			},
		},
		{
			name:       "Header Org ID",
			sourceType: "header",
			config: map[string]interface{}{
				"header_name":  "X-Organization-ID",
				"target_field": "organization_id",
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
			return fmt.Errorf("creating context config %s: %w", cfg.name, err)
		}
	}

	return nil
}

func (db *DB) seedDemoPolicies(ctx context.Context, toolID string, now time.Time) error {
	policyID := uuid.New().String()

	_, err := db.pool.Exec(ctx, `
		INSERT INTO policies (id, tool_id, name, description, enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, policyID, toolID, "Customer Data Access Policy", "Governance policy demonstrating role-based access and rate limiting for customer data.", true, now, now)

	if err != nil {
		return fmt.Errorf("creating policy: %w", err)
	}

	// Add policy rules
	rules := []struct {
		ruleType   string
		config     map[string]interface{}
		priority   int
		failAction string
	}{
		{
			ruleType: "allowed_roles",
			config: map[string]interface{}{
				"roles": []string{"admin", "support", "sales"},
			},
			priority:   1,
			failAction: "deny",
		},
		{
			ruleType: "rate_limit",
			config: map[string]interface{}{
				"max_calls":   100,
				"window_secs": 3600,
				"scope":       "user",
			},
			priority:   2,
			failAction: "deny",
		},
		{
			ruleType: "time_window",
			config: map[string]interface{}{
				"start_hour": 6,
				"end_hour":   22,
				"timezone":   "America/Los_Angeles",
				"weekdays":   []int{1, 2, 3, 4, 5},
			},
			priority:   3,
			failAction: "warn",
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
			return fmt.Errorf("creating policy rule: %w", err)
		}
	}

	return nil
}
