---
name: make-mcp
description: >-
  Provides project context for Make MCP, a UI-driven MCP server builder (Go backend,
  React frontend, PostgreSQL). Use when editing Make MCP code, adding features to
  the server builder, generator, API, or frontend, or when working with MCP tools,
  resources, prompts, versioning, or marketplace.
---

# Make MCP — Project Skill

## What This Project Is

**Make MCP** lets users build **Model Context Protocol (MCP) servers** via a visual UI. No hand-written server code. Users define tools, resources, prompts, auth, and policies; the platform generates a runnable Node.js/TypeScript MCP server.

- **Backend**: Go (Gin), PostgreSQL (pgx/v5), JWT auth
- **Frontend**: React, TypeScript, Vite, Bootstrap
- **Generated output**: Node.js + TypeScript, `@modelcontextprotocol/sdk`, stdio transport
- **Docs**: `PROJECT.md` (product/spec), `README.md` (user-facing), `docs/` (guides)

## Project Layout

| Path | Purpose |
|------|---------|
| `backend/cmd/server/main.go` | API entrypoint |
| `backend/internal/api/handlers.go` | HTTP routes and handlers |
| `backend/internal/database/database.go` | DB access, migrations in SQL comments |
| `backend/internal/models/models.go` | Server, Tool, Resource, Prompt, ServerVersion, etc. |
| `backend/internal/generator/generator.go` | Codegen: server + tool templates → Node/TS zip |
| `backend/internal/openapi/parser.go` | OpenAPI → tool definitions |
| `backend/internal/auth/auth.go` | JWT middleware |
| `frontend/src/` | React app (pages, components, services, styles) |
| `frontend/src/services/api.ts` | Backend API client |
| `docs/` | getting-started, creating-servers, compositions |

## Conventions

### Go (backend)

- Idiomatic Go: stdlib first, explicit errors, no ignored returns
- Use `context.Context` in DB and API layers
- Validate and sanitize all external input; treat as untrusted
- DB: migrations as SQL in `database.go`; run on startup
- New API: add route in `handlers.go`, optional DB helper in `database.go`, model in `models.go` if needed

### Generator

- `backend/internal/generator/generator.go`: `serverTemplate`, `toolTemplate`, `readmeTemplate` (raw strings with `text/template`)
- Templates emit TypeScript/JSON; use `{{.Field}}`, `{{range .Tools}}`, `toSnakeCase`, `toPascalCase`, `toJSON`
- In raw strings, **backticks** must not appear literally (they close the Go string). Use ` + "`" + ` or ` + "```" + ` for backticks in generated output
- New tool execution type: extend `toolTemplate` (and models) for the new type; keep generated TS valid (types, no `undefined` in responses)

### Frontend

- React + TypeScript; Bootstrap for layout and components
- Centralize styles in `frontend/src/styles/App.css` where possible
- API: `frontend/src/services/api.ts`; types in `frontend/src/types/index.ts`
- New pages: add route in router and nav in `Sidebar.tsx`

### MCP and generated server

- Generated server: stdio transport, `ListTools` / `CallTool` handlers, optional `MCP_LOG_FILE` file logging (no stderr when set to avoid Cursor logging "undefined")
- Tools expose `name`, `description`, `inputSchema`; list response uses `?? ""` and `?? {}` so nothing is `undefined`
- Output display: `json` (default), `table`, `card` for MCP Apps–style widgets in Test Playground and generated server

## Common Tasks

- **New tool execution type**: Add constant and handling in `models.go`, DB if needed, `generator.go` (tool template branch), and frontend tool editor options.
- **New API endpoint**: Handler in `handlers.go`, route registration, optional `database.go` and `models.go` changes.
- **UI change**: Relevant component under `frontend/src/`; reuse Bootstrap and existing patterns.
- **Fix generated server**: Change `serverTemplate` or `toolTemplate` in `generator.go`; re-download from UI to get new code.
- **Doc update**: `docs/getting-started.md`, `docs/creating-servers.md`, or `docs/compositions.md`; keep README and PROJECT.md in sync if behavior changes.

## Testing and Running

- **Backend**: `cd backend && go run ./cmd/server` (expects PostgreSQL; see README or PROJECT.md for env).
- **Frontend**: `cd frontend && npm install && npm run dev`.
- **Full stack**: `docker-compose up --build` (backend, frontend, DB).
- **Generated server**: User downloads zip, `npm install && npm run build`, then `node dist/server.js` or `run-with-log.mjs` for file logging; verify in Cursor in a **new window** (project chat often does not receive MCP tools).

## References

- Detailed architecture, APIs, and feature list: [PROJECT.md](../../PROJECT.md) (repo root)
- User setup and verification: [docs/getting-started.md](../../docs/getting-started.md)
- Deeper generator/API notes: [reference.md](reference.md)
