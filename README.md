# Make MCP

> **Status:** Work in progress. This project is a **technology demonstrator** and **proof-of-concept** for building MCP servers through a UI. It is **not** recommended for production use: security, scalability, operational hardening, and support are not at production grade. Use it for experimentation, learning, and demos only.

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.22+-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go">
  <img src="https://img.shields.io/badge/React-18+-61DAFB?style=for-the-badge&logo=react&logoColor=black" alt="React">
  <img src="https://img.shields.io/badge/PostgreSQL-16+-4169E1?style=for-the-badge&logo=postgresql&logoColor=white" alt="PostgreSQL">
  <img src="https://img.shields.io/badge/MCP-Compatible-green?style=for-the-badge" alt="MCP">
</p>

<p align="center">
  <strong>Build Model Context Protocol (MCP) servers visually — no code required.</strong>
</p>

<p align="center">
  Create plug-and-play MCP servers for AI agents, IDEs like Cursor, and platforms like Claude Desktop through an intuitive UI.
</p>

<p align="center">
  <a href="https://vdparikh.github.io/make-mcp/">Project website</a> · <a href="https://github.com/vdparikh/make-mcp">GitHub</a>
</p>

---

## Features

- **Visual Server Builder** — Create MCP servers through a drag-and-drop interface
- **Multiple Execution Types** — REST API, GraphQL, Webhooks, CLI (kubectl, docker, terraform), Database, JavaScript, Python, and Visual Flow (pipeline → tool)
- **Built-in Authentication** — API Key, Bearer Token, Basic Auth, OAuth 2.0 with visual configuration
- **Live Testing Playground** — Test tools before deployment
- **One-Click Export** — Download as Node.js project, ready to run
- **Context-Aware Execution** — Auto-inject user identity, permissions, org data
- **AI Governance Layer** — Policy engine to control tool access (rate limits, roles, approvals)
- **Self-Healing Tools** — Auto-detect failures and suggest fixes
- **Server Composition** — Combine multiple MCP servers into one
- **Security Score** — In-app score (0–100%, grade A–F) based on the [SlowMist MCP Security Checklist](https://github.com/slowmist/MCP-Security-Checklist); shown while building and in the marketplace
- **OpenAPI Import** — Turn an OpenAPI spec into an MCP server in one step; paste or upload, and each path becomes a tool

## Quick Start

**Requirements for the supported path:** a **local Kubernetes** cluster (Docker Desktop, Rancher Desktop, kind, k3d, minikube, …), **`kubectl`** configured for it, **[Skaffold](https://skaffold.dev/)**, and **Docker** (or another builder Skaffold can use for images). The repo treats **Kubernetes + Skaffold** as the single documented way to run the full stack; there is no root-level Docker Compose anymore.

### Run with Kubernetes (recommended)

```bash
git clone https://github.com/vdparikh/make-mcp.git
cd make-mcp
./scripts/skaffold-dev.sh
```

Open http://localhost:3000. **Sign up** with your email and name, then create a **passkey** (no password). Use the same passkey to sign in next time.

Use **`./scripts/skaffold-dev.sh`** (or `skaffold dev --cleanup=false`) so Skaffold does **not** delete the `make-mcp` namespace on exit—plain `skaffold dev` tears down resources and **can wipe the Postgres PVC**. **Exiting Skaffold still leaves app pods running** (only the dev process stops); run **`./scripts/skaffold-stop.sh`** when you want those pods scaled down while keeping Postgres data. See **[CONFIGURATION.md](./CONFIGURATION.md)** for details, secrets, and Keycloak notes.

### Manual Setup (Go + Postgres on the host)

See **[CONFIGURATION.md](./CONFIGURATION.md)** for `config/config.yaml` and **`env.example.sh`** / **`env.sh`** (secrets).

```bash
# 1. Start PostgreSQL
docker run -d --name mcp-builder-db \
  -e POSTGRES_USER=postgres \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=mcp_builder \
  -p 5432:5432 \
  postgres:16-alpine

# 2. Configure secrets and dev env (from repo root)
cp env.example.sh env.sh
# Edit env.sh: set JWT_SECRET (e.g. openssl rand -hex 32) and DATABASE_URL if needed

set -a && source ./env.sh && set +a

# 3. Start Backend (from repo root so config/config.yaml is found, or set MAKE_MCP_CONFIG)
cd backend
go mod download
go run ./cmd/server

# 4. Start Frontend (new terminal; same env.sh so DEV_API_PROXY_TARGET is set)
cd frontend
npm install
npm run dev
```

Open the app at the URL Vite prints (e.g. http://localhost:3000). **Register** with email and name, then add a **passkey** to sign in (passwordless).

## Authentication

Sign-in is **passkey-only** (WebAuthn). There are no passwords. Register with email and name, then create a passkey; use that passkey to sign in on this and supported devices.

## Creating Your First Server

After signing in, create a new server from the dashboard, or use **Start with Template**: **MCP Apps Lab** (one opinionated tool per MCP App widget: table, card, image, chart, map, form — ideal for MCP Jam / Claude), **MCP Production Blueprint** (opinionated full stack: REST + webhook tools, discovery resources, agent prompts, JWT/header context, governance policies), **Demo Template** (lightweight APIs), or **REST Template**. You can add tools (REST, CLI, Flow, etc.), resources, and prompts, then generate and download the MCP server. Example tools you can add:

| Tool | Description |
|------|-------------|
| `get_location_by_zip` | US ZIP code lookup |
| `get_random_user` | Generate random user profiles |
| `get_ip_info` | IP geolocation |
| `get_joke` | Random dad jokes |
| `get_github_user` | GitHub profile lookup |
| `validate_email` | Email validation |
| `get_country_info` | Country details |
| `get_secure_customer_data` | Context injection demo |

Screenshots and feature walkthroughs are on the [project website](https://vdparikh.github.io/make-mcp/).

## 3 Powerful Features

### 1. Context-Aware Tool Execution

Automatically inject user context into tool calls:

```json
{
  "name": "get_customer_data",
  "context_fields": ["user_id", "organization_id", "permissions"]
}
```

AI asks "Show me my invoices" → Tool automatically knows `customer_id = current_user`

### 2. AI Governance Layer

Define policies to control tool access:

```yaml
tool: send_payment
rules:
  - type: max_value
    field: amount
    max_value: 5000
  - type: allowed_roles
    roles: [finance_agent, admin]
  - type: time_window
    hours: 9-17
    weekdays: [Mon-Fri]
```

### 3. Self-Healing Tools

Auto-detect and fix common failures:

| Error | Auto-Suggestion |
|-------|-----------------|
| 401 Unauthorized | Refresh OAuth token |
| 429 Rate Limited | Retry with backoff |
| Schema mismatch | Update tool schema |

## Using Generated Servers

After downloading your MCP server:

```bash
cd my-server-mcp-server
npm install
npm run build
npm start
```

Add to your MCP client config:

```json
{
  "mcpServers": {
    "my-server": {
      "command": "node",
      "args": ["/path/to/my-server-mcp-server/dist/server.js"]
    }
  }
}
```

## Hosted Deploy Architecture

Make MCP supports hosted MCP deployment from the Deploy page via **Publish MCP**.

- **Hosted endpoint**: `http(s)://<host>/api/users/<user_id>/<server_slug>` (versionless URL)
- **Remote MCP testers** (e.g. [Cloudflare Workers AI Playground](https://playground.ai.cloudflare.com/), [MCP Inspector](https://developers.cloudflare.com/agents/guides/test-remote-mcp-server/)):
  - **Direct to your API** (browser, tunnel to the **root** of the API): use the **full path** above — `https://<host>/api/users/<user_id>/<server_slug>`. Do **not** use only `https://<host>/`: `GET /` returns JSON (`application/json`), while these clients expect an SSE stream (`text/event-stream`) and may show `SSE error: Invalid content type`.
  - **Cloudflare quick tunnel** (`cloudflared tunnel --url ...`): if you point `--url` at the **full** hosted path, e.g. `http://127.0.0.1:8080/api/users/<user_id>/<server_slug>`, the printed URL is only `https://<random>.trycloudflare.com` — **use that hostname alone** in the playground (do **not** append `/api/users/...` again, or the path can be wrong and you’ll get JSON or 404 instead of SSE). If you instead use `--url http://127.0.0.1:8080`, then paste **`https://<random>.trycloudflare.com/api/users/<user_id>/<server_slug>`** (path appended by the tunnel).
  - If hosted auth is **`bearer_token`**, the playground must send the same **Bearer** token as in Deploy’s MCP config.
- **Request path**: client `GET/POST` -> backend hosted route -> reverse proxy -> managed container
- **Transport**: hosted runtime uses MCP over HTTP + SSE; backend streams SSE without buffering
- **Container model**: one active container per `user + server` (older/stale hosted containers are reconciled/cleaned)
- **Snapshot model**: each hosted publish creates a hosted-only snapshot record (separate from normal semantic release versions)
- **Runtime source**: generated server files are written under `backend/generated-servers/<user>/<server>/<hosted-snapshot>/` and mounted into the container
- **Observability**: backend injects runtime env (`MCP_OBSERVABILITY_*`) into hosted containers; for URL-based hosted servers, container env is the source of truth
- **Manifest**: each hosted snapshot includes a formal `manifest.json` in its generated folder (runtime/image/tools/auth/resources/prompts/observability)
- **Access model**: hosted publish uses two independent controls:
  - `hosted_auth_mode`: `bearer_token` or `no_auth`
  - `require_caller_identity`: boolean toggle requiring `X-Make-MCP-Caller-Id` (and optional tenant id) for per-caller attribution

In the Deploy UI, Hosted status shows explicit runtime metadata:

- deployed snapshot id/version
- container started-at time
- last ensured time
- hosted URL and MCP config

### One-Click MCP Client Install

For hosted runtimes, Deploy now offers one-click installation for supported IDE clients:

- **Cursor** via deep link (`cursor://anysphere.cursor-deeplink/mcp/install?...`)
- **VS Code** via protocol activation (`vscode:mcp/install?...`)
- **VS Code Insiders** via protocol activation (`vscode-insiders:mcp/install?...`)

Manual JSON config copy remains available for clients that do not support deep-link install flows.

Example hosted manifest:

```json
{
  "name": "Demo API Toolkit",
  "snapshot_version": "hosted-1773848702093159000",
  "server_version": "1.0.0",
  "runtime": "docker",
  "image": "node:20-alpine",
  "endpoint": "/api/users/<user_id>/<server_slug>",
  "tools": [
    { "name": "get_country_info", "execution_type": "rest_api" }
  ],
  "auth": { "type": "none" },
  "observability": true
}
```

## Documentation

- [Getting Started Guide](./docs/getting-started.md) — Full setup and usage guide
- [Creating Servers](./docs/creating-servers.md) — Detailed guide to building MCP servers
- [Server Compositions](./docs/compositions.md) — Combining multiple servers into one
- [Security Best Practices](./docs/security-best-practices.md) — MCP security practices, runtime security model, and the in-app security score

## Tech Stack

| Component | Technology |
|-----------|------------|
| Frontend | React, TypeScript, Vite, Bootstrap |
| Backend | Go, Gin, PostgreSQL |
| Code Editor | Monaco Editor |
| Generated Servers | Node.js, TypeScript, MCP SDK |


## License

MIT License - see [LICENSE](./LICENSE) for details.

