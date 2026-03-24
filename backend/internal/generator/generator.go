package generator

import (
	"archive/zip"
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"

	"github.com/vdparikh/make-mcp/backend/internal/models"
)

//go:embed embed/mcp-app-viewer.html
var mcpAppViewerHTML string

// Generator handles MCP server code generation
type Generator struct {
	templates map[string]*template.Template
	// PublicURLHostIP is the fallback host segment in generated server.ts when Host / X-Forwarded-Host are absent.
	PublicURLHostIP string
}

// NewGenerator creates a generator without a public URL host (invalid for server.ts emit — use NewGeneratorWithPublicHost).
func NewGenerator() *Generator {
	return NewGeneratorWithPublicHost("")
}

// NewGeneratorWithPublicHost creates a generator with the configured fallback host for generated HTTP servers.
func NewGeneratorWithPublicHost(publicURLHostIP string) *Generator {
	g := &Generator{
		templates:       make(map[string]*template.Template),
		PublicURLHostIP: publicURLHostIP,
	}
	g.initTemplates()
	return g
}

func (g *Generator) initTemplates() {
	funcMap := template.FuncMap{
		"toJSON": func(v interface{}) string {
			b, _ := json.MarshalIndent(v, "", "  ")
			return string(b)
		},
		"toCamelCase":  toCamelCase,
		"toPascalCase": toPascalCase,
		"toSnakeCase":  toSnakeCase,
	}

	g.templates["server"] = template.Must(template.New("server").Funcs(funcMap).Parse(serverTemplate))
	g.templates["tool"] = template.Must(template.New("tool").Funcs(funcMap).Parse(toolTemplate))
	g.templates["package"] = template.Must(template.New("package").Funcs(funcMap).Parse(packageJSONTemplate))
	g.templates["tsconfig"] = template.Must(template.New("tsconfig").Parse(tsconfigTemplate))
	g.templates["dockerfile"] = template.Must(template.New("dockerfile").Parse(dockerfileTemplate))
	g.templates["readme"] = template.Must(template.New("readme").Funcs(funcMap).Parse(readmeTemplate))
}

// GeneratedServer contains the generated server files
type GeneratedServer struct {
	Files map[string][]byte
}

// Generate generates an MCP server from a server configuration
func (g *Generator) Generate(server *models.Server) (*GeneratedServer, error) {
	gen := &GeneratedServer{
		Files: make(map[string][]byte),
	}

	serverCode, err := g.generateServerFile(server)
	if err != nil {
		return nil, fmt.Errorf("generating server file: %w", err)
	}
	gen.Files["src/server.ts"] = serverCode
	gen.Files["src/mcp-app-viewer.html"] = []byte(mcpAppViewerHTML)
	gen.Files["src/egress.ts"] = []byte(egressTSContent)

	for _, tool := range server.Tools {
		toolCode, err := g.generateToolFile(&tool)
		if err != nil {
			return nil, fmt.Errorf("generating tool %s: %w", tool.Name, err)
		}
		gen.Files[fmt.Sprintf("src/tools/%s.ts", toSnakeCase(tool.Name))] = toolCode
	}

	indexCode := g.generateToolsIndex(server.Tools)
	gen.Files["src/tools/index.ts"] = indexCode

	packageJSON, err := g.generatePackageJSON(server)
	if err != nil {
		return nil, fmt.Errorf("generating package.json: %w", err)
	}
	gen.Files["package.json"] = packageJSON

	gen.Files["tsconfig.json"] = []byte(tsconfigContent)
	gen.Files["src/types/mcp-sdk.d.ts"] = []byte(mcpSDKDeclarationsContent)

	gen.Files["Dockerfile"] = []byte(dockerfileContent)

	// Generate docker-compose.yml
	dockerCompose, err := g.generateDockerCompose(server)
	if err != nil {
		return nil, fmt.Errorf("generating docker-compose.yml: %w", err)
	}
	gen.Files["docker-compose.yml"] = dockerCompose

	// Generate .dockerignore
	gen.Files[".dockerignore"] = []byte(dockerignoreContent)

	// Generate .env.example
	envExample, err := g.generateEnvExample(server)
	if err != nil {
		return nil, fmt.Errorf("generating .env.example: %w", err)
	}
	gen.Files[".env.example"] = envExample

	readme, err := g.generateReadme(server)
	if err != nil {
		return nil, fmt.Errorf("generating README: %w", err)
	}
	gen.Files["README.md"] = readme

	// Node wrapper: Cursor and other clients use "node" as command; a .mjs script works with command "node" + args [path].
	gen.Files["run-with-log.mjs"] = []byte(`import { spawn } from "child_process";
import path from "path";
import { fileURLToPath } from "url";
const __dirname = path.dirname(fileURLToPath(import.meta.url));
const serverPath = path.join(__dirname, "dist", "server.js");
process.env.MCP_LOG_FILE = path.join(__dirname, "mcp.log");
const child = spawn(process.execPath, [serverPath, ...process.argv.slice(2)], {
  stdio: "inherit",
  env: process.env,
});
child.on("exit", (code) => process.exit(code ?? 0));
`)

	// Shell wrapper for users who prefer to run via bash (command: "bash", args: ["/path/to/run-with-log.sh"]).
	gen.Files["run-with-log.sh"] = []byte(`#!/bin/bash
cd "$(dirname "$0")"
export MCP_LOG_FILE="$(pwd)/mcp.log"
exec node dist/server.js "$@"
`)

	return gen, nil
}

// GenerateZip generates a zip file containing the MCP server.
// The zip has a root folder named {slug}-mcp-server so the name matches "Setup & Run" (cd {slug}-mcp-server).
func (g *Generator) GenerateZip(server *models.Server) ([]byte, error) {
	gen, err := g.Generate(server)
	if err != nil {
		return nil, err
	}

	rootDir := ServerSlug(server.Name) + "-mcp-server/"
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)

	for path, content := range gen.Files {
		zipPath := rootDir + path
		f, err := w.Create(zipPath)
		if err != nil {
			return nil, fmt.Errorf("creating zip entry %s: %w", path, err)
		}
		if _, err := f.Write(content); err != nil {
			return nil, fmt.Errorf("writing zip entry %s: %w", path, err)
		}
	}

	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("closing zip: %w", err)
	}

	return buf.Bytes(), nil
}

// CompositionOptions configures how servers are combined
type CompositionOptions struct {
	PrefixToolNames bool `json:"prefix_tool_names"`
	MergeResources  bool `json:"merge_resources"`
	MergePrompts    bool `json:"merge_prompts"`
}

// BuildCompositionServer builds a virtual combined server from multiple servers.
func (g *Generator) BuildCompositionServer(composition *models.ServerComposition, servers []*models.Server, options CompositionOptions) (*models.Server, error) {
	if composition == nil {
		return nil, fmt.Errorf("composition is required")
	}
	// Create a virtual combined server
	combined := &models.Server{
		ID:          composition.ID,
		Name:        composition.Name,
		Description: composition.Description,
		Version:     "1.0.0",
		Tools:       []models.Tool{},
		Resources:   []models.Resource{},
		Prompts:     []models.Prompt{},
	}

	// Track tool names to detect conflicts
	toolNames := make(map[string]string) // tool name -> source server

	// Merge tools from all servers
	for _, server := range servers {
		for _, tool := range server.Tools {
			originalName := tool.Name

			// Apply prefix if enabled or if there's a conflict
			if options.PrefixToolNames {
				tool.Name = toSnakeCase(server.Name) + "_" + tool.Name
			} else if existingServer, exists := toolNames[tool.Name]; exists {
				// Conflict detected - auto-prefix both
				// First, rename the existing tool
				for i, t := range combined.Tools {
					if t.Name == originalName {
						combined.Tools[i].Name = toSnakeCase(existingServer) + "_" + t.Name
						break
					}
				}
				// Then prefix the new tool
				tool.Name = toSnakeCase(server.Name) + "_" + tool.Name
			}

			toolNames[tool.Name] = server.Name

			// Update description to show source
			if tool.Description != "" {
				tool.Description = fmt.Sprintf("[%s] %s", server.Name, tool.Description)
			}

			combined.Tools = append(combined.Tools, tool)
		}

		// Merge resources if enabled
		if options.MergeResources {
			for _, resource := range server.Resources {
				// Prefix URI to avoid conflicts
				resource.URI = fmt.Sprintf("%s/%s", toSnakeCase(server.Name), resource.URI)
				combined.Resources = append(combined.Resources, resource)
			}
		}

		// Merge prompts if enabled
		if options.MergePrompts {
			for _, prompt := range server.Prompts {
				// Prefix prompt name to avoid conflicts
				prompt.Name = toSnakeCase(server.Name) + "_" + prompt.Name
				combined.Prompts = append(combined.Prompts, prompt)
			}
		}
	}

	return combined, nil
}

// GenerateComposition generates a combined MCP server from multiple servers
func (g *Generator) GenerateComposition(composition *models.ServerComposition, servers []*models.Server, options CompositionOptions) (*GeneratedServer, error) {
	combined, err := g.BuildCompositionServer(composition, servers, options)
	if err != nil {
		return nil, err
	}

	// Generate the combined server
	gen, err := g.Generate(combined)
	if err != nil {
		return nil, fmt.Errorf("generating combined server: %w", err)
	}

	// Generate a composition-specific README
	compositionReadme := g.generateCompositionReadme(composition, servers, combined, options)
	gen.Files["README.md"] = compositionReadme

	return gen, nil
}

// GenerateCompositionZip generates a zip file for a composition
func (g *Generator) GenerateCompositionZip(composition *models.ServerComposition, servers []*models.Server, options CompositionOptions) ([]byte, error) {
	gen, err := g.GenerateComposition(composition, servers, options)
	if err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)

	for path, content := range gen.Files {
		f, err := w.Create(path)
		if err != nil {
			return nil, fmt.Errorf("creating zip entry %s: %w", path, err)
		}
		if _, err := f.Write(content); err != nil {
			return nil, fmt.Errorf("writing zip entry %s: %w", path, err)
		}
	}

	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("closing zip: %w", err)
	}

	return buf.Bytes(), nil
}

func (g *Generator) generateCompositionReadme(composition *models.ServerComposition, servers []*models.Server, combined *models.Server, options CompositionOptions) []byte {
	var buf bytes.Buffer

	buf.WriteString(fmt.Sprintf("# %s\n\n", composition.Name))
	buf.WriteString(fmt.Sprintf("%s\n\n", composition.Description))
	buf.WriteString("## Composed Servers\n\n")
	buf.WriteString("This MCP server is a composition of the following servers:\n\n")

	for _, server := range servers {
		toolCount := len(server.Tools)
		buf.WriteString(fmt.Sprintf("- **%s** (%d tools)\n", server.Name, toolCount))
	}

	buf.WriteString(fmt.Sprintf("\n**Total Tools:** %d\n\n", len(combined.Tools)))

	buf.WriteString("## Available Tools\n\n")
	buf.WriteString("| Tool | Description | Source |\n")
	buf.WriteString("|------|-------------|--------|\n")

	for _, tool := range combined.Tools {
		// Extract source from description
		source := "Unknown"
		desc := tool.Description
		if len(desc) > 0 && desc[0] == '[' {
			end := strings.Index(desc, "]")
			if end > 0 {
				source = desc[1:end]
				desc = strings.TrimSpace(desc[end+1:])
			}
		}
		buf.WriteString(fmt.Sprintf("| `%s` | %s | %s |\n", tool.Name, desc, source))
	}

	buf.WriteString("\n## Installation\n\n")
	buf.WriteString("```bash\nnpm install\nnpm run build\n```\n\n")

	buf.WriteString("## Running\n\n")
	buf.WriteString("```bash\nnpm start\n```\n\n")

	buf.WriteString("## Docker\n\n")
	buf.WriteString("```bash\ndocker-compose up -d\n```\n\n")

	buf.WriteString("## MCP Client Configuration\n\n")
	buf.WriteString("```json\n")
	buf.WriteString(fmt.Sprintf(`{
  "mcpServers": {
    "%s": {
      "command": "node",
      "args": ["./dist/server.js"]
    }
  }
}`, toSnakeCase(composition.Name)))
	buf.WriteString("\n```\n")

	return buf.Bytes()
}

func (g *Generator) generateServerFile(server *models.Server) ([]byte, error) {
	var buf bytes.Buffer
	data := struct {
		*models.Server
		PublicURLHostIP string
	}{
		Server:          server,
		PublicURLHostIP: g.PublicURLHostIP,
	}
	if strings.TrimSpace(data.PublicURLHostIP) == "" {
		return nil, fmt.Errorf("generator: PublicURLHostIP is required (set hosted.generated_server_public_host_ip in config)")
	}
	if err := g.templates["server"].Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (g *Generator) generateToolFile(tool *models.Tool) ([]byte, error) {
	var buf bytes.Buffer

	data := struct {
		*models.Tool
		InputSchemaStr     string
		OutputSchemaStr    string
		ExecutionConfigStr string
	}{
		Tool: tool,
	}

	if tool.InputSchema != nil {
		data.InputSchemaStr = string(tool.InputSchema)
	} else {
		data.InputSchemaStr = "{}"
	}

	if tool.OutputSchema != nil {
		data.OutputSchemaStr = string(tool.OutputSchema)
	} else {
		data.OutputSchemaStr = "{}"
	}

	if tool.ExecutionConfig != nil {
		data.ExecutionConfigStr = string(tool.ExecutionConfig)
	} else {
		data.ExecutionConfigStr = "{}"
	}

	if err := g.templates["tool"].Execute(&buf, data); err != nil {
		return nil, err
	}
	out := buf.String()
	odc := "null"
	if len(tool.OutputDisplayConfig) > 0 {
		odc = strings.TrimSpace(string(tool.OutputDisplayConfig))
	}
	out = strings.Replace(out, "__MMC_OUTPUT_DISPLAY_CONFIG__", odc, 1)
	return []byte(out), nil
}

func (g *Generator) generateToolsIndex(tools []models.Tool) []byte {
	var buf bytes.Buffer

	for _, tool := range tools {
		buf.WriteString(fmt.Sprintf("export { %s } from './%s.js';\n", toPascalCase(tool.Name), toSnakeCase(tool.Name)))
	}

	return buf.Bytes()
}

func (g *Generator) generatePackageJSON(server *models.Server) ([]byte, error) {
	var buf bytes.Buffer
	if err := g.templates["package"].Execute(&buf, server); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (g *Generator) generateReadme(server *models.Server) ([]byte, error) {
	var buf bytes.Buffer
	if err := g.templates["readme"].Execute(&buf, server); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (g *Generator) generateDockerCompose(server *models.Server) ([]byte, error) {
	serverSlug := toSnakeCase(server.Name)

	compose := fmt.Sprintf(`version: '3.8'

services:
  %s:
    build:
      context: .
      dockerfile: Dockerfile
    container_name: %s-mcp
    restart: unless-stopped
    stdin_open: true
    tty: true
    environment:
      - NODE_ENV=production
    env_file:
      - .env
    volumes:
      - ./data:/app/data
    # Uncomment for network access
    # network_mode: host
    # Or expose specific ports
    # ports:
    #   - "3000:3000"

# Optional: Add more services
# volumes:
#   data:
`, serverSlug, serverSlug)

	return []byte(compose), nil
}

func (g *Generator) generateEnvExample(server *models.Server) ([]byte, error) {
	var envVars []string
	envVars = append(envVars, "# Environment variables for "+server.Name)
	envVars = append(envVars, "# Copy this file to .env and fill in your values")
	envVars = append(envVars, "")
	envVars = append(envVars, "NODE_ENV=production")
	envVars = append(envVars, "")

	// Extract env vars from tool configs
	envSet := make(map[string]bool)
	for _, tool := range server.Tools {
		var config map[string]interface{}
		if err := json.Unmarshal(tool.ExecutionConfig, &config); err != nil {
			continue
		}

		// Look for {{VAR}} patterns in the config
		configStr := string(tool.ExecutionConfig)
		extractEnvVars(configStr, envSet)
	}

	// Add common env vars based on auth patterns
	if len(envSet) > 0 {
		envVars = append(envVars, "# API Keys and Secrets")
		for env := range envSet {
			envVars = append(envVars, env+"=")
		}
	}

	envVars = append(envVars, "")
	envVars = append(envVars, "# Optional: CLI tool paths (for CLI execution type)")
	envVars = append(envVars, "# KUBECONFIG=/path/to/kubeconfig")
	envVars = append(envVars, "# AWS_PROFILE=default")
	envVars = append(envVars, "# DOCKER_HOST=unix:///var/run/docker.sock")
	envVars = append(envVars, "")
	envVars = append(envVars, "# Optional: MCP Apps (table/card/image) — extra CSP origins for embedded UI (comma-separated https:// origins)")
	envVars = append(envVars, "# MCP_APP_RESOURCE_DOMAINS=https://cdn.example.com,https://images.example.org")

	return []byte(strings.Join(envVars, "\n")), nil
}

func extractEnvVars(s string, envSet map[string]bool) {
	// Find patterns like {{VAR_NAME}} where VAR_NAME is uppercase with underscores
	start := 0
	for {
		idx := strings.Index(s[start:], "{{")
		if idx == -1 {
			break
		}
		idx += start
		end := strings.Index(s[idx:], "}}")
		if end == -1 {
			break
		}
		end += idx

		varName := s[idx+2 : end]
		// Check if it looks like an env var (all caps with underscores)
		if isEnvVarName(varName) {
			envSet[varName] = true
		}
		start = end + 2
	}
}

func isEnvVarName(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, r := range s {
		if !((r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_') {
			return false
		}
	}
	return true
}

func toCamelCase(s string) string {
	words := splitWords(s)
	for i := 1; i < len(words); i++ {
		words[i] = strings.Title(words[i])
	}
	out := strings.Join(words, "")
	if out == "" {
		return "item"
	}
	if out[0] >= '0' && out[0] <= '9' {
		return "n" + out
	}
	return out
}

func toPascalCase(s string) string {
	words := splitWords(s)
	for i := range words {
		words[i] = strings.Title(words[i])
	}
	out := strings.Join(words, "")
	if out == "" {
		return "Item"
	}
	if out[0] >= '0' && out[0] <= '9' {
		return "N" + out
	}
	return out
}

func toSnakeCase(s string) string {
	words := splitWords(s)
	out := strings.Join(words, "_")
	if out == "" {
		return "item"
	}
	return out
}

// ServerSlug returns a filesystem-safe kebab-case slug from a server name (e.g. "Demo API Toolkit" -> "demo-api-toolkit").
// Used for zip filename and root folder so they match the "Setup & Run" instructions.
func ServerSlug(name string) string {
	s := strings.ToLower(name)
	s = strings.ReplaceAll(s, " ", "-")
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func splitWords(s string) []string {
	var words []string
	var current strings.Builder
	prevLower := false
	flush := func() {
		if current.Len() == 0 {
			return
		}
		words = append(words, strings.ToLower(current.String()))
		current.Reset()
	}

	for _, r := range s {
		switch {
		case r >= 'A' && r <= 'Z':
			if prevLower {
				flush()
			}
			current.WriteRune(r + ('a' - 'A'))
			prevLower = false
		case r >= 'a' && r <= 'z':
			current.WriteRune(r)
			prevLower = true
		case r >= '0' && r <= '9':
			current.WriteRune(r)
			prevLower = false
		default:
			// Treat all non-ASCII-alnum runes as separators (including Unicode dashes).
			flush()
			prevLower = false
		}
	}
	flush()
	if len(words) == 0 {
		return []string{"item"}
	}
	return words
}

const serverTemplate = `import { createWriteStream, readFileSync } from "fs";
import path from "path";
import { fileURLToPath } from "url";
import { Server } from "@modelcontextprotocol/sdk/server/index.js";
import { StdioServerTransport } from "@modelcontextprotocol/sdk/server/stdio.js";
import {
  CallToolRequestSchema,
  ListToolsRequestSchema,
  ListResourcesRequestSchema,
  ReadResourceRequestSchema,
} from "@modelcontextprotocol/sdk/types.js";
import { RESOURCE_MIME_TYPE } from "@modelcontextprotocol/ext-apps/server";
{{range .Tools}}
import { {{toPascalCase .Name}} } from "./tools/{{toSnakeCase .Name}}.js";
{{end}}

const __dirname = path.dirname(fileURLToPath(import.meta.url));
let _mcpViewerCache: string | null = null;
function getMcpAppViewerHtml(): string {
  if (!_mcpViewerCache) {
    _mcpViewerCache = readFileSync(path.join(__dirname, "mcp-app-viewer.html"), "utf-8");
  }
  return _mcpViewerCache;
}

const MCP_APP_RESOURCE_URI = "ui://{{toSnakeCase .Name}}/mcp-app-viewer.html";

/**
 * MCP Apps hosts apply strict CSP to embedded UI. Declare origins for images (and other static
 * loads) via _meta.ui.csp.resourceDomains on resources/read. See:
 * https://apps.extensions.modelcontextprotocol.io/api/documents/csp-and-cors.html
 *
 * Extend with env MCP_APP_RESOURCE_DOMAINS="https://api.example.com,https://cdn.example.org"
 */
function getMcpAppResourceDomains(): string[] {
  const extra = (process.env.MCP_APP_RESOURCE_DOMAINS || "")
    .split(",")
    .map((s) => s.trim())
    .filter((s) => s.length > 0);
  const defaults = [
    "https://www.google.com",
    "https://*.nasa.gov",
    "https://images.unsplash.com",
    "https://*.githubusercontent.com",
    "https://*.amazonaws.com",
    "https://*.cloudfront.net",
    "https://*.googleusercontent.com",
    "https://*.gstatic.com",
    "https://www.google.com",
    "https://maps.google.com",
    "https://*.wikimedia.org",
    "https://*.imgur.com",
    "https://*.twimg.com",
    "https://*.cloudinary.com",
    "https://*.redditmedia.com",
    "https://*.discordapp.com",
    "https://*.slack-edge.com",
  ];
  return [...new Set([...defaults, ...extra])];
}

function mcpAppReadResourceContents(): Record<string, unknown> {
  return {
    uri: MCP_APP_RESOURCE_URI,
    mimeType: RESOURCE_MIME_TYPE,
    text: getMcpAppViewerHtml(),
    _meta: {
      ui: {
        csp: {
          resourceDomains: getMcpAppResourceDomains(),
        },
      },
    },
  };
}

const MCP_LOG_FILE = process.env.MCP_LOG_FILE;
const mcpLogStream = MCP_LOG_FILE ? createWriteStream(MCP_LOG_FILE, { flags: "a" }) : null;
function mcpLog(msg: string) {
  const line = new Date().toISOString() + " [MCP] " + msg + "\n";
  if (mcpLogStream) {
    mcpLogStream.write(line);
  } else {
    console.error("[MCP] " + msg);
  }
}

const MCP_OBSERVABILITY_ENDPOINT = process.env.MCP_OBSERVABILITY_ENDPOINT;
const MCP_OBSERVABILITY_KEY = process.env.MCP_OBSERVABILITY_KEY;
const MCP_OBSERVABILITY_USER_ID = process.env.MCP_OBSERVABILITY_USER_ID || "";
const MCP_OBSERVABILITY_CLIENT_AGENT = process.env.MCP_OBSERVABILITY_CLIENT_AGENT || "";
const MCP_OBSERVABILITY_USER_TOKEN = process.env.MCP_OBSERVABILITY_USER_TOKEN || "";
async function reportObservabilityEvent(
  toolName: string,
  durationMs: number,
  success: boolean,
  errorMessage: string,
  repairSuggestion: string,
  callerIdentity?: string,
  tenantIdentity?: string,
  clientAgent?: string
) {
  if (!MCP_OBSERVABILITY_ENDPOINT || !MCP_OBSERVABILITY_KEY) return;
  try {
    const ev: Record<string, unknown> = {
      tool_name: toolName,
      duration_ms: durationMs,
      success,
      error: errorMessage,
      repair_suggestion: repairSuggestion,
    };
    let resolvedCaller = (callerIdentity || "").trim();
    const resolvedTenant = (tenantIdentity || "").trim();
    if (!resolvedCaller) resolvedCaller = MCP_OBSERVABILITY_USER_ID;
    if (resolvedCaller && resolvedTenant) {
      resolvedCaller = resolvedTenant + "/" + resolvedCaller;
    }
    if (resolvedCaller) ev.client_user_id = resolvedCaller.slice(0, 200);
    const resolvedAgent = (clientAgent || "").trim() || MCP_OBSERVABILITY_CLIENT_AGENT;
    if (resolvedAgent) ev.client_agent = resolvedAgent.slice(0, 120);
    if (MCP_OBSERVABILITY_USER_TOKEN) ev.client_token = MCP_OBSERVABILITY_USER_TOKEN;
    const controller = new AbortController();
    const t = setTimeout(() => controller.abort(), 3000);
    await fetch(MCP_OBSERVABILITY_ENDPOINT, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ key: MCP_OBSERVABILITY_KEY, events: [ev] }),
      signal: controller.signal,
    });
    clearTimeout(t);
  } catch (_) {
    // best-effort; do not block or log to stderr
  }
}

function suggestRepair(errorMessage: string): string {
  const msg = (errorMessage || "").toLowerCase();
  if (/401|unauthorized|invalid.*token|token.*expired|jwt/.test(msg)) return "Check or refresh authentication token.";
  if (/404|not found/.test(msg)) return "Verify the resource path or identifier.";
  if (/429|rate limit|too many requests/.test(msg)) return "Rate limited; retry after a short delay.";
  if (/500|502|503|server error/.test(msg)) return "Remote server error; retry later.";
  if (/timeout|etimedout|econnrefused/.test(msg)) return "Network or timeout issue; check connectivity and retry.";
  return "";
}

function normalizeFormFieldType(t: unknown): string {
  const s = typeof t === "string" ? t.trim().toLowerCase() : "";
  const allowed = new Set([
    "text",
    "textarea",
    "boolean",
    "number",
    "date",
    "time",
    "datetime-local",
    "color",
  ]);
  return allowed.has(s) ? s : "text";
}

function getValueAtPath(obj: Record<string, unknown>, path: string): unknown {
  const parts = path.split(".");
  let cur: unknown = obj;
  for (const p of parts) {
    if (p === "") return undefined;
    if (/^[0-9]{1,3}$/.test(p)) {
      const idx = parseInt(p, 10);
      if (!Array.isArray(cur) || idx < 0 || idx >= cur.length) return undefined;
      cur = cur[idx];
      continue;
    }
    if (typeof cur !== "object" || cur === null || Array.isArray(cur)) return undefined;
    const m = cur as Record<string, unknown>;
    if (!Object.prototype.hasOwnProperty.call(m, p)) return undefined;
    cur = m[p];
  }
  return cur;
}

function readValueAtPath(obj: Record<string, unknown>, key: string): unknown {
  const k = key.trim();
  if (!k.includes(".")) return obj[k];
  return getValueAtPath(obj, k);
}

function buildChartWidgetPayload(
  obj: Record<string, unknown>,
  cfg: OutputDisplayConfig | null | undefined
): { chartType: string; title: string; labels: string[]; datasets: { label: string; data: number[] }[] } | null {
  const labelsKey =
    typeof cfg?.labels_key === "string" && cfg.labels_key.trim() ? cfg.labels_key.trim() : "labels";
  const datasetsKey =
    typeof cfg?.datasets_key === "string" && cfg.datasets_key.trim() ? cfg.datasets_key.trim() : "datasets";
  const chartType = cfg?.chart_type === "line" ? "line" : "bar";
  const rawLabels = readValueAtPath(obj, labelsKey);
  if (!Array.isArray(rawLabels) || rawLabels.length === 0) return null;
  const labels = rawLabels.slice(0, 64).map((v) => String(v ?? "").trim());
  if (labels.every((x) => !x)) return null;
  const rawDS = readValueAtPath(obj, datasetsKey);
  if (!Array.isArray(rawDS) || rawDS.length === 0) return null;
  const datasets: { label: string; data: number[] }[] = [];
  for (let i = 0; i < Math.min(rawDS.length, 8); i++) {
    const m = rawDS[i];
    if (!m || typeof m !== "object" || Array.isArray(m)) continue;
    const rec = m as Record<string, unknown>;
    const label = typeof rec.label === "string" ? rec.label.trim() : "";
    const rawData = rec.data;
    if (!Array.isArray(rawData)) continue;
    const data: number[] = [];
    for (let j = 0; j < Math.min(rawData.length, 256); j++) {
      const p = rawData[j];
      const n = typeof p === "number" && Number.isFinite(p) ? p : Number(p);
      if (Number.isFinite(n)) data.push(n);
    }
    if (data.length === 0) continue;
    datasets.push({ label, data });
  }
  if (datasets.length === 0) return null;
  let n = labels.length;
  for (const ds of datasets) {
    if (ds.data.length < n) n = ds.data.length;
  }
  if (n <= 0) return null;
  const labelsTrim = labels.slice(0, n);
  const datasetsTrim = datasets.map((ds) => ({ label: ds.label, data: ds.data.slice(0, n) }));
  let title = "";
  if (cfg?.title_key) {
    const tv = readValueAtPath(obj, cfg.title_key);
    if (tv !== undefined && tv !== null) title = String(tv).trim();
  }
  if (!title && typeof obj.title === "string") title = String(obj.title).trim();
  return { chartType, title, labels: labelsTrim, datasets: datasetsTrim };
}

function toFiniteNumber(v: unknown): number | null {
  if (typeof v === "number" && Number.isFinite(v)) return v;
  if (typeof v === "string" && v.trim()) {
    const n = Number(v.trim());
    if (Number.isFinite(n)) return n;
  }
  return null;
}

function allowedGoogleMapsEmbedURL(s: string): boolean {
  const t = s.trim();
  if (!t || t.length > 2048) return false;
  try {
    const u = new URL(t);
    if (u.protocol !== "https:") return false;
    const host = u.hostname.toLowerCase();
    if (host !== "www.google.com" && host !== "google.com" && host !== "maps.google.com") return false;
    let p = u.pathname || "/";
    if (p.startsWith("/maps")) return true;
    if (host === "maps.google.com" && (p === "/" || p === "")) {
      return Boolean(u.searchParams.get("q") || u.searchParams.get("pb"));
    }
    return false;
  } catch {
    return false;
  }
}

function parseMapZoomTS(obj: Record<string, unknown>, cfg: OutputDisplayConfig | null | undefined): number {
  let z = 14;
  if (cfg?.zoom !== undefined && typeof cfg.zoom === "number" && Number.isFinite(cfg.zoom)) {
    const zi = Math.floor(cfg.zoom);
    if (zi >= 1 && zi <= 20) z = zi;
  }
  if (cfg?.zoom_key && typeof cfg.zoom_key === "string" && cfg.zoom_key.trim()) {
    const k = cfg.zoom_key.trim();
    const zv = readValueAtPath(obj, k);
    const n = toFiniteNumber(zv);
    if (n !== null) {
      const zi = Math.floor(n);
      if (zi >= 1 && zi <= 20) return zi;
    }
  }
  for (const k of ["zoom", "z"]) {
    if (Object.prototype.hasOwnProperty.call(obj, k)) {
      const n = toFiniteNumber(obj[k]);
      if (n !== null) {
        const zi = Math.floor(n);
        if (zi >= 1 && zi <= 20) return zi;
      }
    }
  }
  return z;
}

function buildGoogleMapsEmbedURLFromCoords(lat: number, lng: number, zoom: number): string {
  const z = Math.max(1, Math.min(20, Math.floor(zoom)));
  const q = new URLSearchParams();
  q.set("q", lat + "," + lng);
  q.set("z", String(z));
  q.set("output", "embed");
  return "https://www.google.com/maps?" + q.toString();
}

function buildMapWidgetPayload(
  obj: Record<string, unknown>,
  cfg: OutputDisplayConfig | null | undefined
): { embedUrl: string; title?: string } | null {
  if (cfg?.embed_url_key && typeof cfg.embed_url_key === "string" && cfg.embed_url_key.trim()) {
    const k = cfg.embed_url_key.trim();
    const v = readValueAtPath(obj, k);
    if (typeof v === "string" && allowedGoogleMapsEmbedURL(v)) {
      const embedUrl = v.trim();
      let title = "";
      if (cfg.title_key) {
        const tv = readValueAtPath(obj, cfg.title_key);
        if (tv !== undefined && tv !== null) title = String(tv).trim();
      }
      if (!title && typeof obj.title === "string") title = String(obj.title).trim();
      const out: { embedUrl: string; title?: string } = { embedUrl };
      if (title) out.title = title;
      return out;
    }
  }
  const latKeys =
    cfg?.lat_key && typeof cfg.lat_key === "string" && cfg.lat_key.trim()
      ? [cfg.lat_key.trim()]
      : ["lat", "latitude"];
  const lngKeys =
    cfg?.lng_key && typeof cfg.lng_key === "string" && cfg.lng_key.trim()
      ? [cfg.lng_key.trim()]
      : ["lng", "lon", "longitude"];
  let lat: number | null = null;
  let lng: number | null = null;
  for (const k of latKeys) {
    const n = toFiniteNumber(readValueAtPath(obj, k));
    if (n !== null) {
      lat = n;
      break;
    }
  }
  for (const k of lngKeys) {
    const n = toFiniteNumber(readValueAtPath(obj, k));
    if (n !== null) {
      lng = n;
      break;
    }
  }
  if (lat === null || lng === null) return null;
  if (lat < -90 || lat > 90 || lng < -180 || lng > 180) return null;
  const zoom = parseMapZoomTS(obj, cfg);
  const embedUrl = buildGoogleMapsEmbedURLFromCoords(lat, lng, zoom);
  let title = "";
  if (cfg?.title_key) {
    const tv = readValueAtPath(obj, cfg.title_key);
    if (tv !== undefined && tv !== null) title = String(tv).trim();
  }
  if (!title && typeof obj.title === "string") title = String(obj.title).trim();
  const out: { embedUrl: string; title?: string } = { embedUrl };
  if (title) out.title = title;
  return out;
}

const server = new Server(
  {
    name: "{{.Name}}",
    version: "{{.Version}}",
  },
  {
    capabilities: {
      tools: {},
      resources: {},
    },
  }
);

type OutputDisplayConfig = {
  content_key?: string;
  title_key?: string;
  image_url_key?: string;
  submit_tool?: string;
  title?: string;
  submit_label?: string;
  fields?: Array<{
    name: string;
    label: string;
    type: string;
    default?: unknown;
    required?: boolean;
    placeholder?: string;
  }>;
  chart_type?: string;
  labels_key?: string;
  datasets_key?: string;
  lat_key?: string;
  lng_key?: string;
  zoom_key?: string;
  embed_url_key?: string;
  zoom?: number;
};

type GeneratedTool = {
  name: string;
  description?: string;
  inputSchema?: Record<string, unknown>;
  readOnlyHint?: boolean;
  destructiveHint?: boolean;
  outputDisplay?: string;
  outputDisplayConfig?: OutputDisplayConfig | null;
  execute: (args: Record<string, unknown>) => Promise<unknown> | unknown;
};

const tools: GeneratedTool[] = [
{{range .Tools}}  {{toPascalCase .Name}},
{{end}}];

type ToolResult = { content: Array<{ type: "text"; text: string }>; isError?: boolean };
type CallContext = { callerIdentity?: string; tenantIdentity?: string; clientAgent?: string };
const hostedSessions = new Map<string, import("http").ServerResponse>();

function listToolsResult() {
  return tools.map((t) => {
    const base: Record<string, unknown> = {
      name: t.name,
      description: t.description ?? "",
      inputSchema: t.inputSchema ?? {},
    };
    if (t.readOnlyHint) base.readOnlyHint = true;
    if (t.destructiveHint) base.destructiveHint = true;
    const hasUi =
      t.outputDisplay === "table" ||
      t.outputDisplay === "card" ||
      t.outputDisplay === "image" ||
      t.outputDisplay === "form" ||
      t.outputDisplay === "chart" ||
      t.outputDisplay === "map";
    if (hasUi) {
      base._meta = { ui: { resourceUri: MCP_APP_RESOURCE_URI } };
    }
    return base;
  });
}

async function executeToolCall(name: string, args: Record<string, unknown>, callCtx: CallContext = {}): Promise<ToolResult> {
  const argsStr = JSON.stringify(args || {});
  const argsPreview = argsStr.length > 200 ? argsStr.slice(0, 200) + "..." : argsStr;
  mcpLog("Tool called: " + name + " | args: " + argsPreview);

  const tool = tools.find((t) => t.name === name);
  if (!tool) {
    mcpLog("Tool not found: " + name);
    return {
      content: [{ type: "text", text: "Error executing " + name + ": Tool not found: " + name }],
      isError: true,
    };
  }

  const callStart = Date.now();
  try {
    const result = await tool.execute(args || {});
    mcpLog("Tool " + name + " completed in " + (Date.now() - callStart) + "ms");
    let text = JSON.stringify(result, null, 2);
    if (tool.outputDisplay === "table") {
      let rows: Record<string, unknown>[] = [];
      if (Array.isArray(result) && result.length > 0 && result.every((r) => typeof r === "object" && r !== null && !Array.isArray(r))) {
        rows = result as Record<string, unknown>[];
      } else if (typeof result === "object" && result !== null && !Array.isArray(result)) {
        rows = [result as Record<string, unknown>];
      }
      if (rows.length > 0) {
        const columns: { key: string; label: string }[] = [];
        const seen = new Set<string>();
        for (const row of rows) {
          for (const k of Object.keys(row)) {
            if (!seen.has(k)) {
              seen.add(k);
              columns.push({ key: k, label: k });
            }
          }
        }
        columns.sort((a, b) => a.key.localeCompare(b.key));
        const mcpApp = { text, _mcp_app: { widget: "table", props: { columns, rows } } };
        text = JSON.stringify(mcpApp, null, 2);
      }
    } else if (tool.outputDisplay === "card" && typeof result === "object" && result !== null && !Array.isArray(result)) {
      const obj = result as Record<string, unknown>;
      const cfg = tool.outputDisplayConfig;
      const contentKeys = ["joke", "text", "content", "message", "body", "description", "quote"];
      let content = "";
      if (cfg?.content_key && typeof obj[cfg.content_key] !== "undefined") {
        const v = obj[cfg.content_key];
        content = typeof v === "string" ? v : (v === null || v === undefined ? "" : JSON.stringify(v));
        content = content.trim();
      }
      if (!content) {
        for (const key of contentKeys) {
          if (typeof obj[key] === "string" && (obj[key] as string).length > 0) {
            content = obj[key] as string;
            break;
          }
        }
      }
      if (!content) {
        for (const v of Object.values(obj)) {
          if (typeof v === "string" && v.length > content.length) {
            content = v;
          }
        }
      }
      if (content) {
        let title = "Result";
        if (cfg?.title_key && typeof obj[cfg.title_key] === "string" && (obj[cfg.title_key] as string).trim()) {
          title = (obj[cfg.title_key] as string).trim();
        } else if (typeof obj.title === "string" && obj.title) {
          title = obj.title as string;
        } else if (typeof obj.name === "string" && obj.name) {
          title = obj.name as string;
        }
        const mcpApp = { text, _mcp_app: { widget: "card", props: { content, title } } };
        text = JSON.stringify(mcpApp, null, 2);
      }
    } else if (tool.outputDisplay === "image" && typeof result === "object" && result !== null && !Array.isArray(result)) {
      const obj = result as Record<string, unknown>;
      const cfg = tool.outputDisplayConfig;
      const fallbackUrlKeys = ["url", "image_url", "imageUrl", "image", "href", "link", "src"];
      let imageUrl = "";
      if (cfg?.image_url_key && typeof obj[cfg.image_url_key] !== "undefined") {
        const v = obj[cfg.image_url_key];
        imageUrl = typeof v === "string" ? v.trim() : "";
      }
      if (!imageUrl) {
        for (const k of fallbackUrlKeys) {
          const v = obj[k];
          if (typeof v === "string" && v.trim()) {
            imageUrl = v.trim();
            break;
          }
        }
      }
      let ok = false;
      try {
        const u = new URL(imageUrl);
        ok = (u.protocol === "http:" || u.protocol === "https:") && Boolean(u.host);
      } catch {
        ok = false;
      }
      if (ok) {
        let title = "";
        if (cfg?.title_key && typeof obj[cfg.title_key] === "string") {
          title = (obj[cfg.title_key] as string).trim();
        } else if (typeof obj.title === "string") {
          title = (obj.title as string).trim();
        } else if (typeof obj.name === "string") {
          title = (obj.name as string).trim();
        }
        const alt = title || "Image";
        const props: Record<string, unknown> = { imageUrl, alt };
        if (title) props.title = title;
        const mcpApp = { text, _mcp_app: { widget: "image", props } };
        text = JSON.stringify(mcpApp, null, 2);
      }
    } else if (tool.outputDisplay === "chart" && typeof result === "object" && result !== null && !Array.isArray(result)) {
      const payload = buildChartWidgetPayload(result as Record<string, unknown>, tool.outputDisplayConfig);
      if (payload) {
        const mcpApp = {
          text,
          _mcp_app: {
            widget: "chart",
            props: {
              chartType: payload.chartType,
              title: payload.title,
              labels: payload.labels,
              datasets: payload.datasets,
            },
          },
        };
        text = JSON.stringify(mcpApp, null, 2);
      }
    } else if (tool.outputDisplay === "form" && typeof result === "object" && result !== null && !Array.isArray(result)) {
      const cfg = tool.outputDisplayConfig;
      const submitTool = typeof cfg?.submit_tool === "string" ? cfg.submit_tool.trim() : "";
      const rawFields = Array.isArray(cfg?.fields) ? cfg.fields : [];
      const fields = rawFields
        .filter((f) => f && typeof f.name === "string" && f.name.length > 0)
        .map((f) => ({
          name: String(f.name).slice(0, 128),
          label: typeof f.label === "string" && f.label.trim() ? f.label.trim().slice(0, 200) : String(f.name),
          type: normalizeFormFieldType(f.type),
          default: f.default,
          required: Boolean(f.required),
          placeholder: typeof f.placeholder === "string" ? f.placeholder.slice(0, 500) : undefined,
        }));
      if (submitTool && fields.length > 0) {
        const obj = result as Record<string, unknown>;
        const initialValues: Record<string, unknown> = {};
        for (const f of fields) {
          if (Object.prototype.hasOwnProperty.call(obj, f.name)) {
            initialValues[f.name] = obj[f.name];
          } else if (f.default !== undefined) {
            initialValues[f.name] = f.default;
          }
        }
        const mcpApp = {
          text,
          _mcp_app: {
            widget: "form",
            props: {
              title: typeof cfg?.title === "string" ? cfg.title : "",
              submitLabel: typeof cfg?.submit_label === "string" && cfg.submit_label.trim() ? cfg.submit_label.trim() : "Submit",
              submitTool,
              fields,
              initialValues,
            },
          },
        };
        text = JSON.stringify(mcpApp, null, 2);
      }
    } else if (tool.outputDisplay === "map" && typeof result === "object" && result !== null && !Array.isArray(result)) {
      const payload = buildMapWidgetPayload(result as Record<string, unknown>, tool.outputDisplayConfig);
      if (payload) {
        const props: Record<string, unknown> = { embedUrl: payload.embedUrl };
        if (payload.title) props.title = payload.title;
        const mcpApp = { text, _mcp_app: { widget: "map", props } };
        text = JSON.stringify(mcpApp, null, 2);
      }
    }
    reportObservabilityEvent(name, Date.now() - callStart, true, "", "", callCtx.callerIdentity, callCtx.tenantIdentity, callCtx.clientAgent).catch(() => {});
    return { content: [{ type: "text", text }] };
  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : String(error);
    const durationMs = Date.now() - callStart;
    mcpLog("Tool " + name + " failed after " + durationMs + "ms: " + errorMessage);
    const repairSuggestion = suggestRepair(errorMessage);
    reportObservabilityEvent(name, durationMs, false, errorMessage, repairSuggestion, callCtx.callerIdentity, callCtx.tenantIdentity, callCtx.clientAgent).catch(() => {});
    return {
      content: [{ type: "text", text: "Error executing " + name + ": " + errorMessage }],
      isError: true,
    };
  }
}

server.setRequestHandler(ListToolsRequestSchema, async () => {
  mcpLog("ListTools requested (agent listing available tools)");
  return { tools: listToolsResult() };
});

server.setRequestHandler(ListResourcesRequestSchema, async () => {
  mcpLog("ListResources requested (MCP Apps UI)");
  return {
    resources: [
      {
        uri: MCP_APP_RESOURCE_URI,
        name: "Make MCP App UI",
        description: "Interactive UI for table, card, image, form, chart, and map tool outputs",
        mimeType: RESOURCE_MIME_TYPE,
      },
    ],
  };
});

server.setRequestHandler(ReadResourceRequestSchema, async (request) => {
  const uri = (request as { params: { uri: string } }).params.uri;
  mcpLog("ReadResource requested: " + uri);
  if (uri !== MCP_APP_RESOURCE_URI) {
    throw new Error("Unknown resource: " + uri);
  }
  return {
    contents: [mcpAppReadResourceContents()],
  };
});

server.setRequestHandler(CallToolRequestSchema, async (request) => {
  const { name, arguments: args } = (request as { params: { name: string; arguments?: unknown } }).params;
  return executeToolCall(name, (args || {}) as Record<string, unknown>, {});
});

function getHeader(req: import("http").IncomingMessage, key: string): string {
  const v = req.headers[key];
  if (Array.isArray(v)) return v[0] || "";
  return v || "";
}

function writeSSEEvent(res: import("http").ServerResponse, event: string, payload: unknown): void {
  res.write("event: " + event + "\n");
  res.write("data: " + JSON.stringify(payload) + "\n\n");
}

async function readJSONBody(req: import("http").IncomingMessage): Promise<Record<string, unknown>> {
  return await new Promise((resolve, reject) => {
    const chunks: Buffer[] = [];
    let size = 0;
    req.on("data", (chunk: Buffer) => {
      size += chunk.length;
      if (size > 1024*1024) {
        reject(new Error("request too large"));
        req.destroy();
        return;
      }
      chunks.push(chunk);
    });
    req.on("end", () => {
      const body = Buffer.concat(chunks).toString("utf8");
      if (!body.trim()) {
        resolve({});
        return;
      }
      try {
        const parsed = JSON.parse(body) as Record<string, unknown>;
        resolve(parsed);
      } catch (err) {
        reject(err);
      }
    });
    req.on("error", reject);
  });
}

function publicEndpointURL(req: import("http").IncomingMessage, fallbackPort: number): string {
  const proto = getHeader(req, "x-forwarded-proto") || "http";
  const host = getHeader(req, "x-forwarded-host") || getHeader(req, "host") || ("{{.PublicURLHostIP}}:" + String(fallbackPort));
  const uri = getHeader(req, "x-forwarded-uri") || req.url || "/";
  return proto + "://" + host + uri;
}

async function runHTTPServer(): Promise<void> {
  const port = Number(process.env.MCP_HTTP_PORT || "3000");
  const http = await import("http");
  const srv = http.createServer(async (req, res) => {
    try {
      if ((req.method || "GET").toUpperCase() === "GET") {
        // MCP streamable HTTP: SSE when client asks for event-stream, empty Accept (RFC default */*), or */*.
        // Only return JSON when Accept explicitly prefers application/json without */* or event-stream (e.g. curl probes).
        const accept = getHeader(req, "accept").toLowerCase();
        const prefersSSE =
          accept.includes("text/event-stream") ||
          accept.trim() === "" ||
          accept === "*/*" ||
          (accept.includes("*/*") && !/^application\/json\s*$/i.test(accept.trim()));
        const prefersJsonOnly =
          accept.includes("application/json") &&
          !accept.includes("text/event-stream") &&
          !accept.includes("*/*") &&
          accept.trim() !== "";
        if (prefersSSE && !prefersJsonOnly) {
          const sessionId = (await import("crypto")).randomUUID();
          hostedSessions.set(sessionId, res);
          res.statusCode = 200;
          res.setHeader("Content-Type", "text/event-stream");
          res.setHeader("Cache-Control", "no-cache");
          res.setHeader("Connection", "keep-alive");
          res.setHeader("X-Accel-Buffering", "no");
          writeSSEEvent(res, "endpoint", { url: publicEndpointURL(req, port), sessionId });
          const keepalive = setInterval(() => res.write(": ping\n\n"), 15000);
          req.on("close", () => {
            clearInterval(keepalive);
            hostedSessions.delete(sessionId);
          });
          return;
        }
        res.statusCode = 200;
        res.setHeader("Content-Type", "application/json");
        res.end(JSON.stringify({ name: "{{.Name}}", version: "{{.Version}}", tools: tools.length }));
        return;
      }

      if ((req.method || "").toUpperCase() !== "POST") {
        res.statusCode = 405;
        res.end("method not allowed");
        return;
      }

      const body = await readJSONBody(req);
      const method = typeof body.method === "string" ? body.method : "";
      const id = (body as { id?: unknown }).id ?? null;
      const sessionID = getHeader(req, "mcp-session-id");
      let responsePayload: Record<string, unknown> = { jsonrpc: "2.0", id, error: { code: -32600, message: "invalid request" } };

      if (method === "initialize") {
        responsePayload = {
          jsonrpc: "2.0",
          id,
          result: {
            protocolVersion: "2024-11-05",
            capabilities: {
              tools: { listChanged: true },
              resources: { subscribe: false, listChanged: false },
            },
            serverInfo: { name: "{{.Name}}", version: "{{.Version}}" },
          },
        };
      } else if (method === "notifications/initialized") {
        res.statusCode = 200;
        res.end();
        return;
      } else if (method === "tools/list") {
        responsePayload = { jsonrpc: "2.0", id, result: { tools: listToolsResult() } };
      } else if (method === "resources/list") {
        responsePayload = {
          jsonrpc: "2.0",
          id,
          result: {
            resources: [
              {
                uri: MCP_APP_RESOURCE_URI,
                name: "Make MCP App UI",
                description: "Interactive UI for table, card, image, form, chart, and map tool outputs",
                mimeType: RESOURCE_MIME_TYPE,
              },
            ],
          },
        };
      } else if (method === "resources/read") {
        const params = (body.params || {}) as { uri?: unknown };
        const uri = typeof params.uri === "string" ? params.uri : "";
        if (uri !== MCP_APP_RESOURCE_URI) {
          responsePayload = {
            jsonrpc: "2.0",
            id,
            error: { code: -32602, message: "Unknown resource: " + uri },
          };
        } else {
          responsePayload = {
            jsonrpc: "2.0",
            id,
            result: {
              contents: [mcpAppReadResourceContents()],
            },
          };
        }
  } else if (method === "tools/call") {
        const params = (body.params || {}) as { name?: unknown; arguments?: unknown };
        const toolName = typeof params.name === "string" ? params.name : "";
        const rawArgs = params.arguments;
        const toolArgs = (rawArgs && typeof rawArgs === "object" && !Array.isArray(rawArgs)) ? rawArgs as Record<string, unknown> : {};
        const callerIdentity = getHeader(req, "x-make-mcp-caller-id");
        const tenantIdentity = getHeader(req, "x-make-mcp-tenant-id");
        const clientAgent = getHeader(req, "user-agent");
        const toolResult = await executeToolCall(toolName, toolArgs, { callerIdentity, tenantIdentity, clientAgent });
        responsePayload = {
          jsonrpc: "2.0",
          id,
          result: {
            content: toolResult.content,
            ...(toolResult.isError ? { isError: true } : {}),
          },
        };
      } else {
        responsePayload = { jsonrpc: "2.0", id, error: { code: -32601, message: "method not found: " + method } };
      }

      if (sessionID && hostedSessions.has(sessionID)) {
        writeSSEEvent(hostedSessions.get(sessionID) as import("http").ServerResponse, "message", responsePayload);
        res.statusCode = 202;
        res.end();
        return;
      }

      res.statusCode = 200;
      res.setHeader("Content-Type", "application/json");
      res.end(JSON.stringify(responsePayload));
    } catch (err) {
      const msg = err instanceof Error ? err.message : String(err);
      res.statusCode = 500;
      res.setHeader("Content-Type", "application/json");
      res.end(JSON.stringify({ error: msg }));
    }
  });

  await new Promise<void>((resolve) => {
    srv.listen(port, "0.0.0.0", () => resolve());
  });
  mcpLog("Server running on HTTP on port " + String(port));
}

async function runStdioServer(): Promise<void> {
  const transport = new StdioServerTransport();
  await server.connect(transport);
  mcpLog("Server running on stdio");
}

async function main() {
  if ((process.env.MCP_TRANSPORT || "").toLowerCase() === "http") {
    await runHTTPServer();
    return;
  }
  await runStdioServer();
}

main().catch((error) => {
  console.error("Fatal error:", error);
  process.exit(1);
});
`

// egressTSContent enforces optional hosted runtime outbound URL policy (MCP_EGRESS_* env).
const egressTSContent = `const egressMode = (process.env.MCP_EGRESS_MODE || "allow_all").toLowerCase();

export function assertMcpEgressAllowed(rawUrl: string, context?: string): void {
  if (egressMode !== "deny_default") return;
  const list = (process.env.MCP_EGRESS_ALLOWLIST || "")
    .split(",")
    .map((s) => s.trim())
    .filter((s) => s.length > 0);
  if (list.length === 0) {
    throw new Error(
      "Egress denied: MCP_EGRESS_ALLOWLIST is empty (configure hosted runtime allowlist for this deployment)"
    );
  }
  let u: URL;
  try {
    u = new URL(rawUrl);
  } catch {
    throw new Error("Egress denied: invalid URL" + (context ? " (" + context + ")" : ""));
  }
  if (u.protocol !== "http:" && u.protocol !== "https:") {
    throw new Error("Egress denied: only http(s) URLs are allowed" + (context ? " (" + context + ")" : ""));
  }
  const host = u.hostname.toLowerCase();
  for (const rule of list) {
    const r = rule.toLowerCase().trim();
    if (r === host) return;
    if (r.startsWith("*.")) {
      const domain = r.slice(2);
      if (host.endsWith("." + domain)) return;
    }
  }
  throw new Error(
    "Egress denied: host " + host + " is not in MCP_EGRESS_ALLOWLIST" + (context ? " (" + context + ")" : "")
  );
}
`

const toolTemplate = `import { assertMcpEgressAllowed } from "../egress.js";

export interface {{toPascalCase .Name}}Input {
  [key: string]: unknown;
}

// OAuth2 token cache
let cachedToken: { token: string; expiresAt: number } | null = null;

async function getOAuth2Token(config: {
  tokenUrl: string;
  clientId: string;
  clientSecret: string;
  scope?: string;
}): Promise<string> {
  assertMcpEgressAllowed(config.tokenUrl, "oauth2 token");
  // Check cache
  if (cachedToken && Date.now() < cachedToken.expiresAt - 60000) {
    return cachedToken.token;
  }

  const params = new URLSearchParams();
  params.append("grant_type", "client_credentials");
  params.append("client_id", config.clientId);
  params.append("client_secret", config.clientSecret);
  if (config.scope) {
    params.append("scope", config.scope);
  }

  const response = await fetch(config.tokenUrl, {
    method: "POST",
    headers: {
      "Content-Type": "application/x-www-form-urlencoded",
    },
    body: params.toString(),
  });

  if (!response.ok) {
    throw new Error("OAuth2 token fetch failed: " + response.status);
  }

  const data = await response.json() as { access_token: string; expires_in?: number };
  const expiresIn = data.expires_in || 3600;
  
  cachedToken = {
    token: data.access_token,
    expiresAt: Date.now() + expiresIn * 1000,
  };

  return data.access_token;
}

interface RestApiConfig {
  url?: string;
  method?: string;
  headers?: Record<string, string>;
  auth?: {
    type?: string;
    oauth2?: {
      tokenUrl: string;
      clientId: string;
      clientSecret: string;
      scope?: string;
    };
  };
}

export const {{toPascalCase .Name}} = {
  name: "{{.Name}}",
  description: "{{.Description}}",
  inputSchema: {{.InputSchemaStr}},
  outputDisplay: "{{if eq .OutputDisplay "table"}}table{{else if eq .OutputDisplay "card"}}card{{else if eq .OutputDisplay "image"}}image{{else if eq .OutputDisplay "form"}}form{{else if eq .OutputDisplay "chart"}}chart{{else if eq .OutputDisplay "map"}}map{{else}}json{{end}}",
  outputDisplayConfig: __MMC_OUTPUT_DISPLAY_CONFIG__,
  readOnlyHint: {{if .ReadOnlyHint}}true{{else}}false{{end}},
  destructiveHint: {{if .DestructiveHint}}true{{else}}false{{end}},

  async execute(input: {{toPascalCase .Name}}Input): Promise<unknown> {
    {{if eq .ExecutionType "rest_api"}}
    const config = {{.ExecutionConfigStr}} as RestApiConfig;
    const url = config.url || "";
    const method = config.method || "GET";
    let headers: Record<string, string> = { ...(config.headers ?? {}) };
    
    // Handle OAuth2 authentication
    if (config.auth?.type === "oauth2" && config.auth?.oauth2) {
      const token = await getOAuth2Token(config.auth.oauth2);
      headers["Authorization"] = "Bearer " + token;
    }
    
    let finalUrl = url;
    for (const [key, value] of Object.entries(input)) {
      finalUrl = finalUrl.replace("{" + "{" + key + "}" + "}", String(value));
    }
    assertMcpEgressAllowed(finalUrl, "rest_api");
    
    const response = await fetch(finalUrl, {
      method,
      headers: {
        "Content-Type": "application/json",
        ...headers,
      },
      body: method !== "GET" ? JSON.stringify(input) : undefined,
    });
    
    if (!response.ok) {
      throw new Error("HTTP " + response.status + ": " + response.statusText);
    }
    
    return response.json();
    {{else if eq .ExecutionType "graphql"}}
    const config = {{.ExecutionConfigStr}} as RestApiConfig & { query?: string };
    const url = config.url || "";
    const query = config.query || "";
    let headers: Record<string, string> = { ...(config.headers ?? {}) };
    
    // Handle OAuth2 authentication
    if (config.auth?.type === "oauth2" && config.auth?.oauth2) {
      const token = await getOAuth2Token(config.auth.oauth2);
      headers["Authorization"] = "Bearer " + token;
    }
    assertMcpEgressAllowed(url, "graphql");
    
    const response = await fetch(url, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        ...headers,
      },
      body: JSON.stringify({
        query,
        variables: input,
      }),
    });
    
    if (!response.ok) {
      throw new Error("HTTP " + response.status + ": " + response.statusText);
    }
    
    const data = await response.json();
    if (data.errors) {
      throw new Error(data.errors[0].message);
    }
    
    return data.data;
    {{else if eq .ExecutionType "webhook"}}
    const config = {{.ExecutionConfigStr}} as RestApiConfig;
    const url = config.url || "";
    let headers: Record<string, string> = { ...(config.headers ?? {}) };
    
    // Handle OAuth2 authentication
    if (config.auth?.type === "oauth2" && config.auth?.oauth2) {
      const token = await getOAuth2Token(config.auth.oauth2);
      headers["Authorization"] = "Bearer " + token;
    }
    assertMcpEgressAllowed(url, "webhook");
    
    const response = await fetch(url, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        ...headers,
      },
      body: JSON.stringify(input),
    });
    
    if (!response.ok) {
      throw new Error("HTTP " + response.status + ": " + response.statusText);
    }
    
    return response.json();
    {{else if eq .ExecutionType "cli"}}
    const { exec } = await import("child_process");
    const { promisify } = await import("util");
    const execAsync = promisify(exec);
    
    const config = {{.ExecutionConfigStr}};
    let command = config.command || "";
    const timeout = config.timeout || 30000;
    const workingDir = config.working_dir || process.cwd();
    const shell = config.shell || "/bin/bash";
    
    // Substitute input variables into command
    for (const [key, value] of Object.entries(input)) {
      const placeholder = "{" + "{" + key + "}" + "}";
      command = command.split(placeholder).join(String(value));
    }
    
    // Security: validate command against allowlist if configured
    if (config.allowed_commands && Array.isArray(config.allowed_commands)) {
      const baseCommand = command.split(" ")[0];
      if (!config.allowed_commands.includes(baseCommand)) {
        throw new Error("Command not in allowlist: " + baseCommand);
      }
    }
    
    try {
      const { stdout, stderr } = await execAsync(command, {
        timeout,
        cwd: workingDir,
        shell,
        env: { ...process.env, ...config.env },
        maxBuffer: 10 * 1024 * 1024, // 10MB buffer
      });
      
      return {
        success: true,
        stdout: stdout.trim(),
        stderr: stderr.trim(),
        command,
      };
    } catch (error: unknown) {
      const execError = error as { stdout?: string; stderr?: string; code?: number; message?: string };
      return {
        success: false,
        stdout: execError.stdout || "",
        stderr: execError.stderr || "",
        exit_code: execError.code,
        error: execError.message,
        command,
      };
    }
    {{else if eq .ExecutionType "flow"}}
    // Flow execution - executes a pipeline of nodes
    const flowConfig = {{.ExecutionConfigStr}};
    const nodes = flowConfig.nodes || [];
    const edges = flowConfig.edges || [];
    
    // Build adjacency list for node traversal
    const outgoing: Record<string, string[]> = {};
    for (const edge of edges) {
      if (!outgoing[edge.source]) outgoing[edge.source] = [];
      outgoing[edge.source].push(edge.target);
    }
    
    // Create node map for quick lookup
    const nodeMap: Record<string, any> = {};
    for (const node of nodes) {
      nodeMap[node.id] = node;
    }
    
    // Find trigger node
    const triggerNode = nodes.find((n: any) => n.type === "trigger");
    if (!triggerNode) {
      return { success: false, error: "No trigger node found in flow" };
    }
    
    // Execute flow using BFS
    const nodeResults: any[] = [];
    let currentData = input;
    const queue = [triggerNode.id];
    const visited = new Set<string>();
    
    while (queue.length > 0) {
      const nodeId = queue.shift()!;
      if (visited.has(nodeId)) continue;
      visited.add(nodeId);
      
      const node = nodeMap[nodeId];
      if (!node) continue;
      
      const nodeStart = Date.now();
      let nodeOutput: any = currentData;
      let nodeError: string | null = null;
      let success = true;
      
      try {
        switch (node.type) {
          case "trigger":
            nodeOutput = currentData;
            break;
            
          case "api": {
            const config = node.data?.config || {};
            let url = config.url || "";
            const method = config.method || "GET";
            const headers = config.headers || {};
            
            // Variable substitution in URL
            for (const [key, value] of Object.entries(currentData)) {
              url = url.replace(new RegExp("{" + "{" + key + "}" + "}", "g"), String(value));
            }
            
            const fetchOptions: any = { method, headers };
            if (method !== "GET" && method !== "HEAD") {
              fetchOptions.body = JSON.stringify(currentData);
              fetchOptions.headers["Content-Type"] = "application/json";
            }
            assertMcpEgressAllowed(url, "flow api");
            
            const response = await fetch(url, fetchOptions);
            nodeOutput = await response.json();
            break;
          }
          
          case "cli": {
            const { exec } = await import("child_process");
            const { promisify } = await import("util");
            const execAsync = promisify(exec);
            
            const config = node.data?.config || {};
            let command = config.command || "";
            
            // Variable substitution
            for (const [key, value] of Object.entries(currentData)) {
              command = command.replace(new RegExp("{" + "{" + key + "}" + "}", "g"), String(value));
            }
            
            const { stdout, stderr } = await execAsync(command, {
              timeout: config.timeout || 30000,
              cwd: config.working_dir || process.cwd(),
              shell: config.shell || "/bin/bash",
            });
            nodeOutput = { stdout: stdout.trim(), stderr: stderr.trim(), command };
            break;
          }
          
          case "transform": {
            const config = node.data?.config || {};
            if (config.expression) {
              // Simple JavaScript expression evaluation (be careful with security!)
              const fn = new Function("input", "return " + config.expression);
              nodeOutput = fn(currentData);
            } else {
              nodeOutput = currentData;
            }
            break;
          }
          
          case "condition": {
            const config = node.data?.config || {};
            const condition = config.condition || "true";
            const fn = new Function("input", "return " + condition);
            const result = fn(currentData);
            nodeOutput = { ...currentData, __conditionResult: result };
            break;
          }
          
          case "output":
            nodeOutput = currentData;
            break;
            
          default:
            nodeOutput = currentData;
        }
      } catch (err: any) {
        success = false;
        nodeError = err.message || String(err);
      }
      
      nodeResults.push({
        node_id: nodeId,
        node_type: node.type,
        success,
        output: nodeOutput,
        error: nodeError,
        duration_ms: Date.now() - nodeStart,
      });
      
      if (!success) break;
      currentData = nodeOutput;
      
      // Add connected nodes to queue
      const nextNodes = outgoing[nodeId] || [];
      for (const nextId of nextNodes) {
        queue.push(nextId);
      }
    }
    
    const finalNode = nodeResults[nodeResults.length - 1];
    return {
      success: finalNode?.success ?? false,
      result: finalNode?.output,
      node_results: nodeResults,
      flow_id: flowConfig.flow_id,
    };
    {{else}}
    // Custom implementation placeholder
    return { input, message: "Tool executed successfully" };
    {{end}}
  },
};
`

const packageJSONTemplate = `{
  "name": "{{toSnakeCase .Name}}-mcp-server",
  "version": "{{.Version}}",
  "description": "{{.Description}}",
  "main": "dist/server.js",
  "type": "module",
  "scripts": {
    "build": "tsc && cp src/mcp-app-viewer.html dist/mcp-app-viewer.html",
    "start": "node dist/server.js",
    "dev": "tsc -w"
  },
  "dependencies": {
    "@modelcontextprotocol/ext-apps": "^1.2.2",
    "@modelcontextprotocol/sdk": "^1.25.0"
  },
  "devDependencies": {
    "@types/node": "^20.0.0",
    "typescript": "^5.0.0"
  }
}
`

const tsconfigContent = `{
  "compilerOptions": {
    "target": "ES2022",
    "module": "NodeNext",
    "moduleResolution": "NodeNext",
    "outDir": "./dist",
    "rootDir": "./src",
    "strict": true,
    "esModuleInterop": true,
    "skipLibCheck": true,
    "forceConsistentCasingInFileNames": true,
    "declaration": true
  },
  "include": ["src/**/*"],
  "exclude": ["node_modules", "dist"]
}
`

const mcpSDKDeclarationsContent = `declare module "@modelcontextprotocol/sdk/server/index.js" {
  export class Server {
    constructor(serverInfo: { name: string; version: string }, options: { capabilities?: { tools?: Record<string, unknown>; resources?: Record<string, unknown> } });
    setRequestHandler(schema: unknown, handler: (request: unknown) => Promise<unknown> | unknown): void;
    connect(transport: unknown): Promise<void>;
  }
}

declare module "@modelcontextprotocol/sdk/server/stdio.js" {
  export class StdioServerTransport {}
}

declare module "@modelcontextprotocol/sdk/types.js" {
  export const CallToolRequestSchema: unknown;
  export const ListToolsRequestSchema: unknown;
  export const ListResourcesRequestSchema: unknown;
  export const ReadResourceRequestSchema: unknown;
}

declare module "@modelcontextprotocol/ext-apps/server" {
  export const RESOURCE_MIME_TYPE: string;
}`

const tsconfigTemplate = tsconfigContent

const dockerfileContent = `# Build stage
FROM node:20-alpine AS builder

WORKDIR /app

# Copy package files
COPY package*.json ./

# Install dependencies
RUN npm ci --only=production=false

# Copy source code
COPY . .

# Build TypeScript
RUN npm run build

# Production stage
FROM node:20-alpine AS production

WORKDIR /app

# Create non-root user for security
RUN addgroup -g 1001 -S nodejs && \
    adduser -S mcp -u 1001

# Copy package files and install production deps only
COPY package*.json ./
RUN npm ci --only=production && npm cache clean --force

# Copy built files from builder
COPY --from=builder /app/dist ./dist

# Set ownership
RUN chown -R mcp:nodejs /app

# Switch to non-root user
USER mcp

# Environment
ENV NODE_ENV=production

# MCP servers use stdio, so we need interactive mode
CMD ["node", "dist/server.js"]
`

const dockerfileTemplate = dockerfileContent

const dockerignoreContent = `# Dependencies
node_modules/

# Build output (we build in Docker)
dist/

# Development files
*.md
.git/
.gitignore
.env
.env.*
!.env.example

# IDE
.idea/
.vscode/
*.swp
*.swo

# OS
.DS_Store
Thumbs.db

# Test files
*.test.ts
*.spec.ts
coverage/
__tests__/

# Docker files (don't need to copy these into image)
docker-compose*.yml
Dockerfile*
`

const readmeTemplate = `# {{.Name}}

{{.Description}}

## Installation

` + "```bash" + `
npm install
` + "```" + `

## Build

` + "```bash" + `
npm run build
` + "```" + `

## Run

` + "```bash" + `
npm start
` + "```" + `

## Tools

{{range .Tools}}
### {{.Name}}

{{.Description}}

**Input Schema:**
` + "```json" + `
{{toJSON .InputSchema}}
` + "```" + `

{{end}}

## Docker

Build:
` + "```bash" + `
docker build -t {{toSnakeCase .Name}}-mcp .
` + "```" + `

Run:
` + "```bash" + `
docker run -it {{toSnakeCase .Name}}-mcp
` + "```" + `

**Security (MCP best practices):** The image runs as a non-root user. Pin to a specific image digest in production (e.g. ` + "`docker pull my-server@sha256:...`" + `) instead of ` + "`latest`" + ` to avoid supply-chain surprises. For CLI tools, set ` + "`allowed_commands`" + ` in execution config to restrict which commands can run.

## MCP Configuration

Add to your MCP client configuration. In **Cursor**: Settings → MCP → Edit config (or edit ` + "`~/.cursor/mcp.json`" + ` directly).

**Basic (no observability):**

` + "```json" + `
{
  "mcpServers": {
    "{{toSnakeCase .Name}}": {
      "command": "node",
      "args": ["/absolute/path/to/this/folder/dist/server.js"]
    }
  }
}
` + "```" + `

**With observability reporting** (tool calls, latency, failures appear in Make MCP → Observability): get the endpoint URL and key from your server's **Observability** tab in Make MCP (Enable reporting), then add an ` + "`env`" + ` block:

` + "```json" + `
{
  "mcpServers": {
    "{{toSnakeCase .Name}}": {
      "command": "node",
      "args": ["/absolute/path/to/this/folder/dist/server.js"],
      "env": {
        "MCP_OBSERVABILITY_ENDPOINT": "https://your-make-mcp-host/api/observability/events",
        "MCP_OBSERVABILITY_KEY": "your-reporting-key-from-make-mcp"
      }
    }
  }
}
` + "```" + `

Replace ` + "`/absolute/path/to/this/folder`" + ` with the real path (e.g. ` + "`/Users/you/Downloads/{{toSnakeCase .Name}}-mcp-server`" + `). If you use ` + "`run-with-log.mjs`" + ` for logging, use its path in ` + "`args`" + ` and add the same ` + "`env`" + ` block; the server will receive these variables.

## Verifying that your client (e.g. Cursor) invokes the server

When your MCP client (Cursor, Claude, etc.) runs this server, it spawns it in the background, so you don't see console output and can't tell if tools are actually being called. Use file logging to confirm:

1. **Use the included Node wrapper** so the server writes every tool call to ` + "`mcp.log`" + `:
   - In your MCP client config, use **command** ` + "`node`" + ` and **args** ` + "`[\"/absolute/path/to/this/folder/run-with-log.mjs\"]`" + ` (replace with your actual path). Do not use the ` + "`.sh`" + ` script as the command—clients that invoke ` + "`node`" + ` will fail on ` + "`.sh`" + ` files.
   - Example (Cursor ` + "`mcp.json`" + `): ` + "`\"command\": \"node\"`" + `, ` + "`\"args\": [\"/Users/you/Downloads/demo-api-toolkit-mcp-server/run-with-log.mjs\"]`" + `.
2. **In another terminal**, from this project folder run:
   ` + "```" + `
   tail -f mcp.log
   ` + "```" + `
3. **In Cursor (or your client)**, ask the agent to use a tool (e.g. "Look up IP 8.8.8.8 using get_ip_info").
4. **Check ` + "`mcp.log`" + `**. You should see lines like:
   - ` + "`ListTools requested (agent listing available tools)`" + `
   - ` + "`Tool called: get_ip_info | args: {\"ip_address\":\"8.8.8.8\"}`" + `
   - ` + "`Tool get_ip_info completed in 85ms`" + `

If those lines appear when you ask the agent to use a tool, your platform is generating a valid MCP server and the client is invoking it correctly.

## Optional: Runtime observability

To send tool calls, latency, and failures to Make MCP (Observability page), set in your MCP client ` + "`env`" + `:

- ` + "`MCP_OBSERVABILITY_ENDPOINT`" + ` - e.g. ` + "`https://your-make-mcp-host/api/observability/events`" + `
- ` + "`MCP_OBSERVABILITY_KEY`" + ` - the reporting key from the server's Observability tab (Enable reporting)

Optional (when many users share the same MCP, so you can see who and which client had errors):

- ` + "`MCP_OBSERVABILITY_USER_ID`" + ` - end-user or tenant identifier (e.g. ` + "`user_123`" + `, email, or tenant id)
- ` + "`MCP_OBSERVABILITY_CLIENT_AGENT`" + ` - AI client name (e.g. ` + "`Cursor`" + `, ` + "`Claude Desktop`" + `, ` + "`VS Code`" + `)
- ` + "`MCP_OBSERVABILITY_USER_TOKEN`" + ` - optional API key or token for correlation (stored with the event, not validated)

The server will POST events after each tool call; failures will include repair suggestions when possible.

---
Generated by MCP Server Builder
`
