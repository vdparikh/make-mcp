# Make MCP — Reference

## Generator (backend/internal/generator/generator.go)

- **Generate(server)** produces a `GeneratedServer` with `Files map[string][]byte` (e.g. `src/server.ts`, `src/tools/<name>.ts`, `package.json`, `README.md`, `run-with-log.mjs`, `run-with-log.sh`).
- **Templates**: `serverTemplate`, `toolTemplate`, `readmeTemplate` are backtick raw strings; template funcs include `toSnakeCase`, `toPascalCase`, `toJSON`.
- **Tool execution types**: `rest_api`, `graphql`, `webhook`, `database`, `javascript`, `python`, `cli`, `flow` — each has a branch in `toolTemplate` for `execute()` and optional config interfaces (e.g. `RestApiConfig`).
- **Output display**: `outputDisplay` is `json` | `table` | `card`; in `CallTool` handler, table/card wrap result in `_mcp_app` for MCP Apps; ListTools uses `description ?? ""`, `inputSchema ?? {}` to avoid `undefined` in JSON.
- **Logging**: When `MCP_LOG_FILE` is set, `mcpLog()` writes only to the file (no `console.error`) to avoid Cursor showing "undefined" after each line.

## API (backend/internal/api/handlers.go)

- Auth: JWT middleware; many routes require auth.
- Servers: CRUD, publish, versions, download; marketplace (list public, download by id).
- Tools, resources, prompts, context, policies: CRUD under `/api/servers/:id/...`.
- Test tool: `POST /api/servers/:id/tools/:toolId/test` with JSON body; returns result (optionally wrapped for MCP Apps table/card).
- OpenAPI import: endpoint to create tools from OpenAPI spec.

## Database (backend/internal/database/database.go)

- Migrations: SQL in comments or inline; run on connect (e.g. `CREATE TABLE IF NOT EXISTS`, `ALTER TABLE ... ADD COLUMN IF NOT EXISTS`).
- Tables: `users`, `servers`, `server_versions`, `tools`, `resources`, `prompts`, `context_configs`, `policies`, etc.
- Use `context.Context` on all DB calls; scan into structs or pointers for nullable fields (e.g. `*string` for nullable UUIDs).

## Frontend Key Files

- **api.ts**: Axios-based client; all backend routes called from here.
- **types/index.ts**: Server, Tool, Resource, Prompt, ServerVersion, PublishRequest, OutputDisplay, MCPAppPayload, etc.
- **TestPlayground**: Renders tool result; if `isMCPAppOutput(result.output)` then table or card widget.
- **ToolEditor**: Schema, execution type, execution config, output display (json/table/card).
- **ServerEditor**: Tabs for details, tools, resources, prompts, context, policies, versions; publish modal.
