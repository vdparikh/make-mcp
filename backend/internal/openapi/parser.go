package openapi

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/vdparikh/make-mcp/backend/internal/mcpvalidate"
	"github.com/vdparikh/make-mcp/backend/internal/models"
	"gopkg.in/yaml.v3"
)

// OpenAPISpec represents a simplified OpenAPI 3.0 specification
type OpenAPISpec struct {
	OpenAPI string         `json:"openapi" yaml:"openapi"`
	Info    OpenAPIInfo    `json:"info" yaml:"info"`
	Servers []OpenAPIServer `json:"servers" yaml:"servers"`
	Paths   map[string]PathItem `json:"paths" yaml:"paths"`
	Components *Components `json:"components,omitempty" yaml:"components,omitempty"`
}

type OpenAPIInfo struct {
	Title       string `json:"title" yaml:"title"`
	Description string `json:"description" yaml:"description"`
	Version     string `json:"version" yaml:"version"`
}

type OpenAPIServer struct {
	URL         string `json:"url" yaml:"url"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}

type PathItem struct {
	Get     *Operation `json:"get,omitempty" yaml:"get,omitempty"`
	Post    *Operation `json:"post,omitempty" yaml:"post,omitempty"`
	Put     *Operation `json:"put,omitempty" yaml:"put,omitempty"`
	Delete  *Operation `json:"delete,omitempty" yaml:"delete,omitempty"`
	Patch   *Operation `json:"patch,omitempty" yaml:"patch,omitempty"`
}

type Operation struct {
	OperationID string       `json:"operationId,omitempty" yaml:"operationId,omitempty"`
	Summary     string       `json:"summary,omitempty" yaml:"summary,omitempty"`
	Description string       `json:"description,omitempty" yaml:"description,omitempty"`
	Tags        []string     `json:"tags,omitempty" yaml:"tags,omitempty"`
	Parameters  []Parameter  `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	RequestBody *RequestBody `json:"requestBody,omitempty" yaml:"requestBody,omitempty"`
	Responses   map[string]Response `json:"responses,omitempty" yaml:"responses,omitempty"`
	Security    []map[string][]string `json:"security,omitempty" yaml:"security,omitempty"`
}

type Parameter struct {
	Name        string      `json:"name" yaml:"name"`
	In          string      `json:"in" yaml:"in"` // path, query, header, cookie
	Description string      `json:"description,omitempty" yaml:"description,omitempty"`
	Required    bool        `json:"required,omitempty" yaml:"required,omitempty"`
	Schema      *Schema     `json:"schema,omitempty" yaml:"schema,omitempty"`
	Ref         string      `json:"$ref,omitempty" yaml:"$ref,omitempty"`
}

type RequestBody struct {
	Description string                `json:"description,omitempty" yaml:"description,omitempty"`
	Required    bool                  `json:"required,omitempty" yaml:"required,omitempty"`
	Content     map[string]MediaType  `json:"content,omitempty" yaml:"content,omitempty"`
}

type MediaType struct {
	Schema *Schema `json:"schema,omitempty" yaml:"schema,omitempty"`
}

type Response struct {
	Description string               `json:"description,omitempty" yaml:"description,omitempty"`
	Content     map[string]MediaType `json:"content,omitempty" yaml:"content,omitempty"`
}

type Schema struct {
	Ref         string             `json:"$ref,omitempty" yaml:"$ref,omitempty"`
	Type        string             `json:"type,omitempty" yaml:"type,omitempty"`
	Format      string             `json:"format,omitempty" yaml:"format,omitempty"`
	Description string             `json:"description,omitempty" yaml:"description,omitempty"`
	Properties  map[string]*Schema `json:"properties,omitempty" yaml:"properties,omitempty"`
	Items       *Schema            `json:"items,omitempty" yaml:"items,omitempty"`
	Required    []string           `json:"required,omitempty" yaml:"required,omitempty"`
	Enum        []interface{}      `json:"enum,omitempty" yaml:"enum,omitempty"`
	Default     interface{}        `json:"default,omitempty" yaml:"default,omitempty"`
}

type Components struct {
	Schemas         map[string]*Schema         `json:"schemas,omitempty" yaml:"schemas,omitempty"`
	SecuritySchemes map[string]*SecurityScheme `json:"securitySchemes,omitempty" yaml:"securitySchemes,omitempty"`
	Parameters      map[string]*Parameter      `json:"parameters,omitempty" yaml:"parameters,omitempty"`
}

type SecurityScheme struct {
	Type         string `json:"type" yaml:"type"` // apiKey, http, oauth2, openIdConnect
	Description  string `json:"description,omitempty" yaml:"description,omitempty"`
	Name         string `json:"name,omitempty" yaml:"name,omitempty"` // for apiKey
	In           string `json:"in,omitempty" yaml:"in,omitempty"`     // for apiKey: header, query, cookie
	Scheme       string `json:"scheme,omitempty" yaml:"scheme,omitempty"` // for http: bearer, basic
	BearerFormat string `json:"bearerFormat,omitempty" yaml:"bearerFormat,omitempty"`
	Flows        *OAuthFlows `json:"flows,omitempty" yaml:"flows,omitempty"`
}

type OAuthFlows struct {
	ClientCredentials *OAuthFlow `json:"clientCredentials,omitempty" yaml:"clientCredentials,omitempty"`
	AuthorizationCode *OAuthFlow `json:"authorizationCode,omitempty" yaml:"authorizationCode,omitempty"`
}

type OAuthFlow struct {
	TokenURL          string            `json:"tokenUrl,omitempty" yaml:"tokenUrl,omitempty"`
	AuthorizationURL  string            `json:"authorizationUrl,omitempty" yaml:"authorizationUrl,omitempty"`
	Scopes            map[string]string `json:"scopes,omitempty" yaml:"scopes,omitempty"`
}

// Parser handles OpenAPI spec parsing
type Parser struct{}

// NewParser creates a new OpenAPI parser
func NewParser() *Parser {
	return &Parser{}
}

// ParseResult contains the parsed server and tools
type ParseResult struct {
	ServerName    string
	ServerDesc    string
	ServerVersion string
	BaseURL       string
	Tools         []ToolDefinition
	AuthConfig    *AuthConfig
}

type ToolDefinition struct {
	Name            string
	Description     string
	Method          string
	Path            string
	InputSchema     map[string]interface{}
	OutputSchema    map[string]interface{}
	ExecutionConfig map[string]interface{}
	PathParams      []string
	QueryParams     []string
	HeaderParams    []string
}

type AuthConfig struct {
	Type         string // api_key, bearer_token, basic_auth, oauth2
	HeaderName   string
	Prefix       string
	TokenURL     string
	Scopes       []string
}

// Parse parses an OpenAPI spec from YAML or JSON
func (p *Parser) Parse(data []byte) (*ParseResult, error) {
	var spec OpenAPISpec
	
	// Try YAML first, then JSON
	if err := yaml.Unmarshal(data, &spec); err != nil {
		if err := json.Unmarshal(data, &spec); err != nil {
			return nil, fmt.Errorf("failed to parse OpenAPI spec: %w", err)
		}
	}

	// Validate it's OpenAPI 3.x
	if !strings.HasPrefix(spec.OpenAPI, "3.") {
		return nil, fmt.Errorf("only OpenAPI 3.x is supported, got: %s", spec.OpenAPI)
	}

	result := &ParseResult{
		ServerName:    toSnakeCase(spec.Info.Title),
		ServerDesc:    spec.Info.Description,
		ServerVersion: spec.Info.Version,
		Tools:         []ToolDefinition{},
	}

	// Get base URL
	if len(spec.Servers) > 0 {
		result.BaseURL = spec.Servers[0].URL
	}

	// Extract auth config
	if spec.Components != nil && spec.Components.SecuritySchemes != nil {
		result.AuthConfig = p.extractAuthConfig(spec.Components.SecuritySchemes)
	}

	// Parse paths into tools
	for path, pathItem := range spec.Paths {
		if pathItem.Get != nil {
			tool := p.operationToTool("GET", path, pathItem.Get, &spec)
			result.Tools = append(result.Tools, tool)
		}
		if pathItem.Post != nil {
			tool := p.operationToTool("POST", path, pathItem.Post, &spec)
			result.Tools = append(result.Tools, tool)
		}
		if pathItem.Put != nil {
			tool := p.operationToTool("PUT", path, pathItem.Put, &spec)
			result.Tools = append(result.Tools, tool)
		}
		if pathItem.Delete != nil {
			tool := p.operationToTool("DELETE", path, pathItem.Delete, &spec)
			result.Tools = append(result.Tools, tool)
		}
		if pathItem.Patch != nil {
			tool := p.operationToTool("PATCH", path, pathItem.Patch, &spec)
			result.Tools = append(result.Tools, tool)
		}
	}

	// MCP tool names: allowed charset + unique per server (sanitized from OpenAPI operation IDs / paths).
	used := make(map[string]struct{})
	for i := range result.Tools {
		base := mcpvalidate.SanitizeToolName(result.Tools[i].Name)
		if base == "" {
			base = "tool"
		}
		result.Tools[i].Name = mcpvalidate.EnsureUniqueToolName(base, used)
	}

	return result, nil
}

func (p *Parser) extractAuthConfig(schemes map[string]*SecurityScheme) *AuthConfig {
	for _, scheme := range schemes {
		switch scheme.Type {
		case "apiKey":
			return &AuthConfig{
				Type:       "api_key",
				HeaderName: scheme.Name,
			}
		case "http":
			if scheme.Scheme == "bearer" {
				return &AuthConfig{
					Type: "bearer_token",
				}
			} else if scheme.Scheme == "basic" {
				return &AuthConfig{
					Type: "basic_auth",
				}
			}
		case "oauth2":
			config := &AuthConfig{
				Type: "oauth2",
			}
			if scheme.Flows != nil && scheme.Flows.ClientCredentials != nil {
				config.TokenURL = scheme.Flows.ClientCredentials.TokenURL
				for scope := range scheme.Flows.ClientCredentials.Scopes {
					config.Scopes = append(config.Scopes, scope)
				}
			}
			return config
		}
	}
	return nil
}

func (p *Parser) operationToTool(method, path string, op *Operation, spec *OpenAPISpec) ToolDefinition {
	// Generate tool name
	name := op.OperationID
	if name == "" {
		name = generateToolName(method, path)
	}
	name = toSnakeCase(name)

	// Description
	desc := op.Summary
	if desc == "" {
		desc = op.Description
	}
	if desc == "" {
		desc = fmt.Sprintf("%s %s", method, path)
	}

	tool := ToolDefinition{
		Name:        name,
		Description: desc,
		Method:      method,
		Path:        path,
		PathParams:  []string{},
		QueryParams: []string{},
		HeaderParams: []string{},
	}

	// Build input schema from parameters
	inputSchema := map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
		"required":   []string{},
	}
	properties := inputSchema["properties"].(map[string]interface{})
	required := []string{}

	// Process parameters
	for _, param := range op.Parameters {
		// Resolve parameter reference if needed
		resolvedParam := p.resolveParameter(param, spec)
		
		propSchema := p.parameterToSchema(resolvedParam)
		properties[resolvedParam.Name] = propSchema
		
		if resolvedParam.Required {
			required = append(required, resolvedParam.Name)
		}

		switch resolvedParam.In {
		case "path":
			tool.PathParams = append(tool.PathParams, resolvedParam.Name)
		case "query":
			tool.QueryParams = append(tool.QueryParams, resolvedParam.Name)
		case "header":
			tool.HeaderParams = append(tool.HeaderParams, resolvedParam.Name)
		}
	}

	// Process request body
	if op.RequestBody != nil {
		if content, ok := op.RequestBody.Content["application/json"]; ok && content.Schema != nil {
			bodySchema := p.resolveSchema(content.Schema, spec)
			if bodySchema.Properties != nil {
				for propName, propSchema := range bodySchema.Properties {
					properties[propName] = p.schemaToJSONSchema(propSchema, spec)
				}
				for _, req := range bodySchema.Required {
					required = append(required, req)
				}
			}
		}
	}

	inputSchema["required"] = required
	tool.InputSchema = inputSchema

	// Build output schema from response
	tool.OutputSchema = p.extractOutputSchema(op.Responses, spec)

	// Build execution config
	tool.ExecutionConfig = p.buildExecutionConfig(method, path, tool)

	return tool
}

func (p *Parser) resolveParameter(param Parameter, spec *OpenAPISpec) Parameter {
	if param.Ref != "" && spec.Components != nil && spec.Components.Parameters != nil {
		refName := strings.TrimPrefix(param.Ref, "#/components/parameters/")
		if resolved, ok := spec.Components.Parameters[refName]; ok {
			return *resolved
		}
	}
	return param
}

func (p *Parser) resolveSchema(schema *Schema, spec *OpenAPISpec) *Schema {
	if schema == nil {
		return nil
	}
	if schema.Ref != "" && spec.Components != nil && spec.Components.Schemas != nil {
		refName := strings.TrimPrefix(schema.Ref, "#/components/schemas/")
		if resolved, ok := spec.Components.Schemas[refName]; ok {
			return resolved
		}
	}
	return schema
}

func (p *Parser) parameterToSchema(param Parameter) map[string]interface{} {
	schema := map[string]interface{}{}
	
	if param.Schema != nil {
		schema["type"] = param.Schema.Type
		if param.Schema.Format != "" {
			schema["format"] = param.Schema.Format
		}
		if len(param.Schema.Enum) > 0 {
			schema["enum"] = param.Schema.Enum
		}
		if param.Schema.Default != nil {
			schema["default"] = param.Schema.Default
		}
	} else {
		schema["type"] = "string"
	}
	
	if param.Description != "" {
		schema["description"] = param.Description
	}
	
	return schema
}

func (p *Parser) schemaToJSONSchema(schema *Schema, spec *OpenAPISpec) map[string]interface{} {
	if schema == nil {
		return map[string]interface{}{"type": "string"}
	}

	resolved := p.resolveSchema(schema, spec)
	if resolved == nil {
		return map[string]interface{}{"type": "string"}
	}

	result := map[string]interface{}{}
	
	if resolved.Type != "" {
		result["type"] = resolved.Type
	}
	if resolved.Description != "" {
		result["description"] = resolved.Description
	}
	if resolved.Format != "" {
		result["format"] = resolved.Format
	}
	if len(resolved.Enum) > 0 {
		result["enum"] = resolved.Enum
	}
	if resolved.Default != nil {
		result["default"] = resolved.Default
	}
	
	if resolved.Properties != nil {
		props := map[string]interface{}{}
		for name, prop := range resolved.Properties {
			props[name] = p.schemaToJSONSchema(prop, spec)
		}
		result["properties"] = props
	}
	
	if resolved.Items != nil {
		result["items"] = p.schemaToJSONSchema(resolved.Items, spec)
	}
	
	if len(resolved.Required) > 0 {
		result["required"] = resolved.Required
	}
	
	return result
}

func (p *Parser) extractOutputSchema(responses map[string]Response, spec *OpenAPISpec) map[string]interface{} {
	// Try to get 200 or 201 response schema
	for _, code := range []string{"200", "201", "default"} {
		if resp, ok := responses[code]; ok {
			if content, ok := resp.Content["application/json"]; ok && content.Schema != nil {
				return p.schemaToJSONSchema(content.Schema, spec)
			}
		}
	}
	
	return map[string]interface{}{
		"type": "object",
		"description": "API response",
	}
}

func (p *Parser) buildExecutionConfig(method, path string, tool ToolDefinition) map[string]interface{} {
	// Convert OpenAPI path params {param} to template {{param}}
	urlPath := path
	for _, param := range tool.PathParams {
		urlPath = strings.ReplaceAll(urlPath, "{"+param+"}", "{{"+param+"}}")
	}

	// Build query string template
	if len(tool.QueryParams) > 0 {
		queryParts := []string{}
		for _, param := range tool.QueryParams {
			queryParts = append(queryParts, param+"={{"+param+"}}")
		}
		urlPath += "?" + strings.Join(queryParts, "&")
	}

	config := map[string]interface{}{
		"url":     "{{BASE_URL}}" + urlPath,
		"method":  method,
		"headers": map[string]interface{}{
			"Content-Type": "application/json",
		},
	}

	return config
}

func generateToolName(method, path string) string {
	// Remove leading slash and split
	path = strings.TrimPrefix(path, "/")
	parts := strings.Split(path, "/")
	
	// Filter out path parameters
	filtered := []string{}
	for _, part := range parts {
		if !strings.HasPrefix(part, "{") {
			filtered = append(filtered, part)
		}
	}
	
	// Build name
	methodPrefix := strings.ToLower(method)
	switch method {
	case "GET":
		methodPrefix = "get"
	case "POST":
		methodPrefix = "create"
	case "PUT":
		methodPrefix = "update"
	case "DELETE":
		methodPrefix = "delete"
	case "PATCH":
		methodPrefix = "patch"
	}
	
	if len(filtered) > 0 {
		return methodPrefix + "_" + strings.Join(filtered, "_")
	}
	return methodPrefix + "_resource"
}

func toSnakeCase(s string) string {
	// Handle common separators
	s = strings.ReplaceAll(s, "-", "_")
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, ".", "_")
	
	// Handle camelCase
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			prev := rune(s[i-1])
			if prev >= 'a' && prev <= 'z' {
				result.WriteRune('_')
			}
		}
		result.WriteRune(r)
	}
	
	return strings.ToLower(result.String())
}

// ToServerAndTools converts ParseResult to models for database storage
func (r *ParseResult) ToServerAndTools(serverID string) (*models.Server, []models.Tool) {
	server := &models.Server{
		ID:          serverID,
		Name:        r.ServerName,
		Description: r.ServerDesc,
		Version:     r.ServerVersion,
	}
	if server.Version == "" {
		server.Version = "1.0.0"
	}

	tools := make([]models.Tool, 0, len(r.Tools))
	for _, t := range r.Tools {
		// Update URL with actual base URL
		if execConfig, ok := t.ExecutionConfig["url"].(string); ok {
			t.ExecutionConfig["url"] = strings.ReplaceAll(execConfig, "{{BASE_URL}}", r.BaseURL)
		}

		// Add auth config if present
		if r.AuthConfig != nil {
			authMap := map[string]interface{}{
				"type": r.AuthConfig.Type,
			}
			switch r.AuthConfig.Type {
			case "api_key":
				authMap["apiKey"] = map[string]interface{}{
					"headerName": r.AuthConfig.HeaderName,
					"prefix":     r.AuthConfig.Prefix,
					"value":      "{{API_KEY}}",
				}
			case "bearer_token":
				authMap["bearerToken"] = map[string]interface{}{
					"token": "{{BEARER_TOKEN}}",
				}
			case "oauth2":
				authMap["oauth2"] = map[string]interface{}{
					"tokenUrl":     r.AuthConfig.TokenURL,
					"clientId":     "{{OAUTH_CLIENT_ID}}",
					"clientSecret": "{{OAUTH_CLIENT_SECRET}}",
					"scope":        strings.Join(r.AuthConfig.Scopes, " "),
				}
			}
			t.ExecutionConfig["auth"] = authMap
		}

		inputSchemaJSON, _ := json.Marshal(t.InputSchema)
		outputSchemaJSON, _ := json.Marshal(t.OutputSchema)
		execConfigJSON, _ := json.Marshal(t.ExecutionConfig)

		tool := models.Tool{
			ServerID:        serverID,
			Name:            t.Name,
			Description:     t.Description,
			ExecutionType:   "rest_api",
			InputSchema:     inputSchemaJSON,
			OutputSchema:    outputSchemaJSON,
			ExecutionConfig: execConfigJSON,
		}
		tools = append(tools, tool)
	}

	return server, tools
}
