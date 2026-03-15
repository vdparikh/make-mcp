package generator

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"

	"github.com/vdparikh/make-mcp/backend/internal/models"
)

// Generator handles MCP server code generation
type Generator struct {
	templates map[string]*template.Template
}

// NewGenerator creates a new generator
func NewGenerator() *Generator {
	g := &Generator{
		templates: make(map[string]*template.Template),
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
		"toCamelCase": toCamelCase,
		"toPascalCase": toPascalCase,
		"toSnakeCase": toSnakeCase,
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

// GenerateComposition generates a combined MCP server from multiple servers
func (g *Generator) GenerateComposition(composition *models.ServerComposition, servers []*models.Server, options CompositionOptions) (*GeneratedServer, error) {
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
	if err := g.templates["server"].Execute(&buf, server); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (g *Generator) generateToolFile(tool *models.Tool) ([]byte, error) {
	var buf bytes.Buffer
	
	data := struct {
		*models.Tool
		InputSchemaStr  string
		OutputSchemaStr string
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
	return buf.Bytes(), nil
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
	return strings.Join(words, "")
}

func toPascalCase(s string) string {
	words := splitWords(s)
	for i := range words {
		words[i] = strings.Title(words[i])
	}
	return strings.Join(words, "")
}

func toSnakeCase(s string) string {
	words := splitWords(s)
	return strings.Join(words, "_")
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
	s = strings.ReplaceAll(s, "-", "_")
	s = strings.ReplaceAll(s, " ", "_")
	
	var words []string
	current := ""
	
	for i, r := range s {
		if r == '_' {
			if current != "" {
				words = append(words, strings.ToLower(current))
				current = ""
			}
		} else if i > 0 && r >= 'A' && r <= 'Z' && s[i-1] >= 'a' && s[i-1] <= 'z' {
			if current != "" {
				words = append(words, strings.ToLower(current))
			}
			current = string(r)
		} else {
			current += string(r)
		}
	}
	
	if current != "" {
		words = append(words, strings.ToLower(current))
	}
	
	return words
}

const serverTemplate = `import { createWriteStream } from "fs";
import { Server } from "@modelcontextprotocol/sdk/server/index.js";
import { StdioServerTransport } from "@modelcontextprotocol/sdk/server/stdio.js";
import {
  CallToolRequestSchema,
  ListToolsRequestSchema,
} from "@modelcontextprotocol/sdk/types.js";
{{range .Tools}}
import { {{toPascalCase .Name}} } from "./tools/{{toSnakeCase .Name}}.js";
{{end}}

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
  repairSuggestion: string
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
    if (MCP_OBSERVABILITY_USER_ID) ev.client_user_id = MCP_OBSERVABILITY_USER_ID;
    if (MCP_OBSERVABILITY_CLIENT_AGENT) ev.client_agent = MCP_OBSERVABILITY_CLIENT_AGENT;
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

const server = new Server(
  {
    name: "{{.Name}}",
    version: "{{.Version}}",
  },
  {
    capabilities: {
      tools: {},
    },
  }
);

const tools = [
{{range .Tools}}  {{toPascalCase .Name}},
{{end}}];

server.setRequestHandler(ListToolsRequestSchema, async () => {
  mcpLog("ListTools requested (agent listing available tools)");
  return {
    tools: tools.map((t) => ({
      name: t.name,
      description: t.description ?? "",
      inputSchema: t.inputSchema ?? {},
      ...(t.readOnlyHint && { readOnlyHint: true }),
      ...(t.destructiveHint && { destructiveHint: true }),
    })),
  };
});

server.setRequestHandler(CallToolRequestSchema, async (request) => {
  const { name, arguments: args } = request.params;
  const argsStr = JSON.stringify(args || {});
  const argsPreview = argsStr.length > 200 ? argsStr.slice(0, 200) + "..." : argsStr;
  mcpLog("Tool called: " + name + " | args: " + argsPreview);

  const tool = tools.find((t) => t.name === name);
  if (!tool) {
    mcpLog("Tool not found: " + name);
    throw new Error("Tool not found: " + name);
  }

  const callStart = Date.now();
  try {
    let result = await tool.execute(args || {});
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
            if (!seen.has(k)) { seen.add(k); columns.push({ key: k, label: k }); }
          }
        }
        columns.sort((a, b) => a.key.localeCompare(b.key));
        const mcpApp = { text, _mcp_app: { widget: "table", props: { columns, rows } } };
        text = JSON.stringify(mcpApp, null, 2);
      }
    } else if (tool.outputDisplay === "card" && typeof result === "object" && result !== null && !Array.isArray(result)) {
      const obj = result as Record<string, unknown>;
      const contentKeys = ["joke", "text", "content", "message", "body", "description", "quote"];
      let content = "";
      for (const key of contentKeys) {
        if (typeof obj[key] === "string" && (obj[key] as string).length > 0) {
          content = obj[key] as string;
          break;
        }
      }
      if (!content) {
        for (const v of Object.values(obj)) {
          if (typeof v === "string" && v.length > content.length) content = v;
        }
      }
      if (content) {
        const title = (typeof obj.title === "string" && obj.title) ? obj.title : (typeof obj.name === "string" && obj.name) ? obj.name : "Result";
        const mcpApp = { text, _mcp_app: { widget: "card", props: { content, title } } };
        text = JSON.stringify(mcpApp, null, 2);
      }
    }
    reportObservabilityEvent(name, Date.now() - callStart, true, "", "").catch(() => {});
    return {
      content: [
        {
          type: "text",
          text,
        },
      ],
    };
  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : String(error);
    const durationMs = Date.now() - callStart;
    mcpLog("Tool " + name + " failed after " + durationMs + "ms: " + errorMessage);
    const repairSuggestion = suggestRepair(errorMessage);
    reportObservabilityEvent(name, durationMs, false, errorMessage, repairSuggestion).catch(() => {});
    return {
      content: [
        {
          type: "text",
          text: "Error executing " + name + ": " + errorMessage,
        },
      ],
      isError: true,
    };
  }
});

async function main() {
  const transport = new StdioServerTransport();
  await server.connect(transport);
  mcpLog("Server running on stdio");
}

main().catch((error) => {
  console.error("Fatal error:", error);
  process.exit(1);
});
`

const toolTemplate = `export interface {{toPascalCase .Name}}Input {
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
  outputDisplay: "{{if eq .OutputDisplay "table"}}table{{else if eq .OutputDisplay "card"}}card{{else}}json{{end}}",
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
    "build": "tsc",
    "start": "node dist/server.js",
    "dev": "tsc -w"
  },
  "dependencies": {
    "@modelcontextprotocol/sdk": "^1.0.0"
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
