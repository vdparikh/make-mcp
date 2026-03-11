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

	readme, err := g.generateReadme(server)
	if err != nil {
		return nil, fmt.Errorf("generating README: %w", err)
	}
	gen.Files["README.md"] = readme

	return gen, nil
}

// GenerateZip generates a zip file containing the MCP server
func (g *Generator) GenerateZip(server *models.Server) ([]byte, error) {
	gen, err := g.Generate(server)
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

const serverTemplate = `import { Server } from "@modelcontextprotocol/sdk/server/index.js";
import { StdioServerTransport } from "@modelcontextprotocol/sdk/server/stdio.js";
import {
  CallToolRequestSchema,
  ListToolsRequestSchema,
} from "@modelcontextprotocol/sdk/types.js";
{{range .Tools}}
import { {{toPascalCase .Name}} } from "./tools/{{toSnakeCase .Name}}.js";
{{end}}

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
  return {
    tools: tools.map((t) => ({
      name: t.name,
      description: t.description,
      inputSchema: t.inputSchema,
    })),
  };
});

server.setRequestHandler(CallToolRequestSchema, async (request) => {
  const { name, arguments: args } = request.params;
  
  const tool = tools.find((t) => t.name === name);
  if (!tool) {
    throw new Error("Tool not found: " + name);
  }
  
  try {
    const result = await tool.execute(args || {});
    return {
      content: [
        {
          type: "text",
          text: JSON.stringify(result, null, 2),
        },
      ],
    };
  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : String(error);
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
  console.error("{{.Name}} MCP server running on stdio");
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

export const {{toPascalCase .Name}} = {
  name: "{{.Name}}",
  description: "{{.Description}}",
  inputSchema: {{.InputSchemaStr}},
  
  async execute(input: {{toPascalCase .Name}}Input): Promise<unknown> {
    {{if eq .ExecutionType "rest_api"}}
    const config = {{.ExecutionConfigStr}};
    const url = config.url || "";
    const method = config.method || "GET";
    let headers: Record<string, string> = { ...config.headers } || {};
    
    // Handle OAuth2 authentication
    if (config.auth?.type === "oauth2" && config.auth.oauth2) {
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
    const config = {{.ExecutionConfigStr}};
    const url = config.url || "";
    const query = config.query || "";
    let headers: Record<string, string> = { ...config.headers } || {};
    
    // Handle OAuth2 authentication
    if (config.auth?.type === "oauth2" && config.auth.oauth2) {
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
    const config = {{.ExecutionConfigStr}};
    const url = config.url || "";
    let headers: Record<string, string> = { ...config.headers } || {};
    
    // Handle OAuth2 authentication
    if (config.auth?.type === "oauth2" && config.auth.oauth2) {
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

const dockerfileContent = `FROM node:20-alpine

WORKDIR /app

COPY package*.json ./
RUN npm install

COPY . .
RUN npm run build

CMD ["npm", "start"]
`

const dockerfileTemplate = dockerfileContent

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

## MCP Configuration

Add to your MCP client configuration:

` + "```json" + `
{
  "mcpServers": {
    "{{toSnakeCase .Name}}": {
      "command": "node",
      "args": ["path/to/dist/server.js"]
    }
  }
}
` + "```" + `

---
Generated by MCP Server Builder
`
