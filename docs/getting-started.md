# MCP Server Builder - Getting Started Guide

A UI-driven platform to create **Model Context Protocol (MCP) servers** without writing code.

---

## Table of Contents

1. [Quick Start](#quick-start)
2. [Project Structure](#project-structure)
3. [Architecture](#architecture)
4. [Core Features](#core-features)
5. [3 Powerful Features](#3-powerful-features)
6. [API Reference](#api-reference)
7. [Creating Your First MCP Server](#creating-your-first-mcp-server)
8. [Verifying that your client invokes the server](#verifying-that-your-client-eg-cursor-invokes-the-server)
9. [Example: Location Lookup Tool](#example-location-lookup-tool)
10. [Deployment](#deployment)
11. [Security Score](#security-score)

---

## Quick Start

### Prerequisites

- **Go 1.22+**
- **Node.js 20+**
- **PostgreSQL 16+** (or Docker)

### Option 1: Docker Compose (Recommended)

```bash
docker-compose up --build
```

Open http://localhost:3000

### Option 2: Manual Setup

```bash
# 1. Start PostgreSQL
docker run -d --name mcp-builder-db \
  -e POSTGRES_USER=postgres \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=mcp_builder \
  -p 5432:5432 \
  postgres:16-alpine

# 2. Start Backend (Terminal 1)
cd backend
go mod download
go run ./cmd/server

# 3. Start Frontend (Terminal 2)
cd frontend
npm install
npm run dev
```

Open http://localhost:3000. You will need to **log in** or **register**. On first run, a default demo user is created: **demo@example.com** / **demo123**. A **Demo API Toolkit** server is also seeded for that user with 8 tools, sample resources, prompts, context configs, and policies.

---

## Demo Server (Auto-Created)

When you first start the platform, a **Demo API Toolkit** server is automatically seeded with:

### 8 Working Tools (Free APIs, No Auth Required)

| Tool | Description | API |
|------|-------------|-----|
| `get_location_by_zip` | US ZIP code → city, state, coordinates | Zippopotam.us |
| `get_random_user` | Generate random user profiles | RandomUser.me |
| `get_ip_info` | IP address geolocation | IPInfo.io |
| `get_joke` | Random dad jokes | icanhazdadjoke.com |
| `get_github_user` | GitHub profile lookup | GitHub API |
| `validate_email` | Email format validation | Disify |
| `get_country_info` | Country details (capital, population) | RestCountries |
| `get_secure_customer_data` | Demo of context injection | HTTPBin |

### Sample Resources
- `api_documentation` - Markdown documentation
- `sample_data` - Test data (ZIP codes, usernames, IPs)

### Sample Prompts
- `location_summary` - Summarize location data
- `user_profile_analysis` - Analyze user profiles
- `country_comparison` - Compare two countries

### Context Configuration
- JWT claims extraction
- HTTP header extraction (X-User-ID, X-Organization-ID)

### Governance Policy
- Role-based access (admin, support, sales)
- Rate limiting (100 calls/hour)
- Business hours restriction

---

## Project Structure

```
make-mcp/
├── backend/                          # Go API Server
│   ├── cmd/server/main.go            # Entry point
│   ├── internal/
│   │   ├── api/handlers.go           # REST API handlers
│   │   ├── auth/                     # JWT auth, login/register
│   │   ├── database/                 # PostgreSQL + migrations + seed
│   │   ├── models/                   # Data models
│   │   ├── generator/                # MCP server code generator
│   │   ├── context/                  # Context Engine
│   │   ├── governance/               # Policy Engine
│   │   ├── healing/                  # Self-Healing Engine
│   │   ├── security/                 # Security score (SlowMist checklist)
│   │   └── openapi/                  # OpenAPI import
│   ├── go.mod
│   └── Dockerfile
│
├── frontend/                         # React UI
│   ├── src/
│   │   ├── pages/
│   │   │   ├── Dashboard.tsx         # Server list
│   │   │   ├── ServerEditor.tsx       # Server configuration
│   │   │   ├── Compositions.tsx      # Server composition
│   │   │   ├── Marketplace.tsx       # Browse published servers
│   │   │   ├── Login.tsx, Register   # Auth
│   │   │   └── ...
│   │   ├── components/
│   │   │   ├── ToolEditor.tsx        # Tool builder (REST, CLI, Flow, etc.)
│   │   │   ├── ResourceEditor.tsx, PromptEditor.tsx
│   │   │   ├── ContextConfigEditor.tsx, PolicyEditor.tsx
│   │   │   ├── TestPlayground.tsx    # Live testing (table/card output)
│   │   │   └── HealingDashboard.tsx
│   │   ├── services/api.ts
│   │   ├── types/index.ts
│   │   └── styles/App.css
│   ├── package.json
│   └── Dockerfile
│
├── docs/                             # Documentation
│   ├── getting-started.md
│   ├── creating-servers.md
│   ├── compositions.md
│   └── security-best-practices.md
├── docker-compose.yml
└── Makefile
```

---

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     React Frontend                          │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────────┐   │
│  │Dashboard │ │ Server   │ │  Test    │ │  Healing     │   │
│  │          │ │ Editor   │ │Playground│ │  Dashboard   │   │
│  └──────────┘ └──────────┘ └──────────┘ └──────────────┘   │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                      Go Backend API                         │
│  ┌──────────────┐ ┌──────────────┐ ┌──────────────────┐    │
│  │   REST API   │ │  Generator   │ │  Tool Executor   │    │
│  │   Handlers   │ │   Engine     │ │                  │    │
│  └──────────────┘ └──────────────┘ └──────────────────┘    │
│                                                             │
│  ┌──────────────┐ ┌──────────────┐ ┌──────────────────┐    │
│  │   Context    │ │  Governance  │ │   Self-Healing   │    │
│  │   Engine     │ │   Engine     │ │     Engine       │    │
│  └──────────────┘ └──────────────┘ └──────────────────┘    │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                      PostgreSQL                             │
│  servers │ tools │ resources │ prompts │ policies │ ...    │
└─────────────────────────────────────────────────────────────┘
```

---

## Core Features

### 1. Visual Server Builder

Create MCP servers through a graphical UI:
- Server name, description, version
- Tools (functions AI can call)
- Resources (data endpoints)
- Prompts (templated instructions)

### 2. Tool Builder

Create tools with multiple execution types:

| Execution Type | Description |
|----------------|-------------|
| `rest_api` | Call external REST APIs |
| `graphql` | Execute GraphQL queries |
| `webhook` | Send data to webhooks |
| `cli` | Execute shell commands (e.g. kubectl, docker, terraform, aws) with an optional `allowed_commands` allowlist |
| `database` | Execute SQL queries |
| `javascript` | Run JavaScript code |
| `python` | Run Python scripts |
| `flow` | Visual pipeline: chain nodes (API, transform, etc.) and convert to a single tool |

#### Authentication Configuration

The Tool Builder includes a dedicated **Authentication** tab with visual configuration for:

| Auth Type | Description | Configuration |
|-----------|-------------|---------------|
| **No Authentication** | Public APIs | None required |
| **API Key** | Header-based API key | Header name, prefix, value |
| **Bearer Token** | JWT/OAuth tokens | Token value (auto-adds `Authorization: Bearer`) |
| **Basic Auth** | Username + password | Credentials (auto-encodes Base64) |
| **OAuth 2.0** | Client credentials flow | Token URL, client ID, client secret, scope |

**How it works:**

1. Select an auth type from the visual picker
2. Fill in the required fields (secrets can use `{{ENV_VAR}}` placeholders)
3. Auth headers are automatically merged into the execution config
4. For OAuth 2.0, tokens are fetched and cached at runtime

**Example - API Key:**
```json
// Generated execution config
{
  "url": "https://api.example.com/data",
  "headers": {
    "X-API-Key": "your-api-key"
  },
  "auth": {
    "type": "api_key",
    "apiKey": {
      "headerName": "X-API-Key",
      "prefix": "",
      "value": "your-api-key"
    }
  }
}
```

**Example - OAuth 2.0:**
```json
{
  "url": "https://api.example.com/data",
  "auth": {
    "type": "oauth2",
    "oauth2": {
      "tokenUrl": "https://auth.example.com/oauth/token",
      "clientId": "client_123",
      "clientSecret": "{{OAUTH_CLIENT_SECRET}}",
      "scope": "read write"
    }
  }
}
```

The generated MCP server automatically handles OAuth2 token fetching and caching.

### 3. Live Testing Playground

Test tools before deployment:
- Provide mock input
- Simulate user context
- View responses and errors
- Get healing suggestions on failure

### 4. Server Code Generation & Deploy

From the **Deploy** tab you can:
- **Node.js** — Generate and download a ZIP (Node.js + TypeScript project). Includes `run-with-log.mjs` for verifying tool invocations.
- **Docker** — Instructions and generated Dockerfile; run as non-root.
- **GitHub** — Push the generated server to a GitHub repository (create or existing).
- **Deploy to Cloud** — Placeholder for future cloud deployment option.

### 5. Security Score

Make MCP computes a **security score** (0–100%, grade A–F) for each server based on the [SlowMist MCP Security Checklist](https://github.com/slowmist/MCP-Security-Checklist).

- **While building:** Open your server → **Security** in the left navigation. View the current score, grade, and a list of criteria (e.g. input validation, rate limiting, access control, CLI allowlist, tool hints). Address unmet items to improve the score.
- **In the marketplace:** Published servers display their security score and grade on the card and in the inspector; the **Security** tab shows which checklist items the server satisfies.

See [Security Best Practices](./security-best-practices.md) for the full mapping of practices to Make MCP features.

---

## 3 Powerful Features

### Feature 1: Context-Aware Tool Execution

Automatically inject user identity, permissions, and organization data into tool calls.

**Problem:** AI tools behave like dumb APIs without user context.

**Solution:** Configure context extraction from JWT, headers, or query params.

```json
// Tool definition with context
{
  "name": "get_customer_data",
  "context_fields": ["user_id", "organization_id", "permissions"]
}

// At runtime, tool receives:
{
  "customer_id": "123",
  "context": {
    "user_id": "abc",
    "organization_id": "org-42",
    "permissions": ["billing.read"]
  }
}
```

**Context Source Types:**
- `header` - Extract from HTTP headers
- `jwt` - Extract from JWT token claims
- `query` - Extract from URL query parameters
- `custom` - Custom extraction logic

### Feature 2: AI Governance Layer (Policy Engine)

Define rules that control when and how AI agents can call tools.

**Problem:** AI agents can accidentally call dangerous tools like `delete_all_users()` or `send_wire_transfer()`.

**Solution:** Define governance policies with rules.

**Available Rule Types:**

| Rule Type | Description | Example |
|-----------|-------------|---------|
| `approval_required` | Require human approval | Payments over $1000 |
| `max_value` | Limit field values | Max amount: $5000 |
| `allowed_roles` | Restrict to roles | Only `finance_agent` |
| `time_window` | Allow during hours | 9 AM - 5 PM only |
| `rate_limit` | Limit call frequency | 100 calls/hour |

**Example Policy:**
```yaml
tool: send_payment
rules:
  - type: max_value
    config:
      field: amount
      max_value: 5000
    fail_action: deny
  
  - type: allowed_roles
    config:
      roles: ["finance_agent", "admin"]
    fail_action: deny
  
  - type: time_window
    config:
      start_hour: 9
      end_hour: 17
      weekdays: [1, 2, 3, 4, 5]
    fail_action: deny
```

### Feature 3: Self-Healing Tools

Automatically detect failures and suggest fixes.

**Problem:** Tools fail due to expired tokens, schema changes, rate limits.

**Solution:** Analyze errors and provide repair suggestions.

**Auto-Detected Error Patterns:**

| Error | Detection | Suggestion |
|-------|-----------|------------|
| 401 Unauthorized | Token expired | Refresh OAuth token |
| 403 Forbidden | Permission denied | Request permissions |
| 429 Rate Limited | Too many requests | Retry with backoff |
| Schema mismatch | Field name changed | Update tool schema |
| Timeout | Request too slow | Extend timeout |
| 5xx Server Error | External service down | Retry with backoff |

---

## API Reference

All `/api/servers`, `/api/tools`, `/api/resources`, `/api/prompts`, `/api/policies`, `/api/compositions`, and `/api/import/openapi` endpoints require **authentication** (Bearer token). Public: `/api/health`, `/api/auth/login`, `/api/auth/register`, `/api/marketplace` (read).

### Auth

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/auth/register` | Register user |
| POST | `/api/auth/login` | Login (returns JWT) |
| GET | `/api/auth/me` | Current user (requires auth) |

### Servers

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/servers` | List current user's servers |
| POST | `/api/servers` | Create server |
| GET | `/api/servers/:id` | Get server with tools/resources/prompts |
| PUT | `/api/servers/:id` | Update server |
| DELETE | `/api/servers/:id` | Delete server |
| POST | `/api/servers/:id/generate` | Generate & download ZIP |
| POST | `/api/servers/:id/github-export` | Push to GitHub |
| POST | `/api/servers/:id/publish` | Publish version (marketplace) |
| GET | `/api/servers/:id/versions` | List published versions |
| GET | `/api/servers/:id/versions/:version` | Get version snapshot |
| GET | `/api/servers/:id/versions/:version/download` | Download version ZIP |
| GET | `/api/servers/:id/flows` | List flows (visual builder) |
| GET | `/api/servers/:id/security-score` | Security score (SlowMist) |
| GET | `/api/servers/:id/context-configs` | List context configs |
| POST | `/api/servers/:id/context-configs` | Create context config |

### Tools

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/tools` | Create tool |
| GET | `/api/tools/:id` | Get tool |
| PUT | `/api/tools/:id` | Update tool |
| DELETE | `/api/tools/:id` | Delete tool |
| POST | `/api/tools/:id/test` | Test tool execution |
| GET | `/api/tools/:id/executions` | Get execution history |
| GET | `/api/tools/:id/policies` | Get tool policies |
| GET | `/api/tools/:id/healing` | Get healing suggestions |

### Resources & Prompts

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/resources` | Create resource |
| DELETE | `/api/resources/:id` | Delete resource |
| POST | `/api/prompts` | Create prompt |
| DELETE | `/api/prompts/:id` | Delete prompt |

### Policies

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/policies` | Create policy |
| DELETE | `/api/policies/:id` | Delete policy |
| POST | `/api/policies/evaluate` | Evaluate policy |

### Compositions

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/compositions` | List current user's compositions |
| POST | `/api/compositions` | Create composition |
| GET | `/api/compositions/:id` | Get composition |
| PUT | `/api/compositions/:id` | Update composition |
| DELETE | `/api/compositions/:id` | Delete composition |
| POST | `/api/compositions/:id/export` | Export composition ZIP |

### Marketplace (public read)

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/marketplace` | List published public servers |
| GET | `/api/marketplace/:id` | Get server + versions + security score |
| GET | `/api/marketplace/:id/download` | Download latest version ZIP |

### Import

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/import/openapi/preview` | Preview OpenAPI → tools (no auth) |
| POST | `/api/import/openapi` | Import OpenAPI and create server (auth) |

---

## Creating Your First MCP Server

### Step 1: Create a Server

1. Click **"New Server"** on the Dashboard (after logging in).
2. In the modal, enter:
   - **Name:** `weather-service`
   - **Description:** `Weather and location tools for AI agents`
   - **Version:** `1.0.0` (optional)
3. Click **"Create Server"**. You are taken to the Server Editor.

### Step 2: Add a Tool

1. In the Server Editor left nav, open **Tools** and click **"Add Tool"**.
2. Configure:

**Basic Info:**
- **Name:** `get_location_by_zip`
- **Description:** `Get location details for a US zip code`
- **Execution Type:** `rest_api` (REST API)

**Input Schema** (Config tab): use the schema editor or paste:
```json
{
  "type": "object",
  "properties": {
    "zip_code": {
      "type": "string",
      "description": "US ZIP code"
    }
  },
  "required": ["zip_code"]
}
```

**Execution** (Config tab): set URL, method, and optional auth:
- **URL:** `https://api.zippopotam.us/us/{{zip_code}}`
- **Method:** `GET`
- **Headers:** leave empty or add as needed (no auth for this public API)

3. Click **Save** (or **Create Tool** when creating).

### Step 3: Test the Tool

1. Go to **Testing** tab
2. Select `get_location_by_zip`
3. Enter input:
```json
{
  "zip_code": "94538"
}
```
4. Click **Execute Tool**
5. View the response

### Step 4: Generate & Download

1. In the Server Editor left nav, open **Deploy**.
2. Select **Node.js** and follow the instructions: click **"Generate & Download"** to get a ZIP.
3. Extract and run:
```bash
cd weather-service-mcp-server
npm install
npm run build
npm start
```

### Step 5: Configure MCP Client

Add to your MCP client config (Claude Desktop, Cursor, etc.):

```json
{
  "mcpServers": {
    "weather-service": {
      "command": "node",
      "args": ["/path/to/weather-service-mcp-server/dist/server.js"]
    }
  }
}
```

To verify the client actually invokes your server, use **command** `node` and **args** `["/path/to/weather-service-mcp-server/run-with-log.mjs"]`, then run `tail -f mcp.log` in the server directory (see [Verifying that your client invokes the server](#verifying-that-your-client-eg-cursor-invokes-the-server)).

### Verifying that your client (e.g. Cursor) invokes the server

The client runs the server in the background, so you don't see console output and can't tell if tools are actually being called. Every **downloaded server** from Make MCP includes:

1. **`run-with-log.mjs`** – a Node script that runs the server and writes every MCP event to `mcp.log`. Use **command** `node` and **args** `["/full/path/to/run-with-log.mjs"]` in your MCP config (do not use the `.sh` script as the command—clients that run `node` will fail on `.sh`).
2. **README section** – "Verifying that your client (e.g. Cursor) invokes the server".

**Quick check:**

- In your MCP config set **command** to `node` and **args** to `["/full/path/to/your-server/run-with-log.mjs"]`.
- In another terminal: `cd /path/to/your-server && tail -f mcp.log`.
- In Cursor, ask the agent to use a tool (e.g. "Look up IP 8.8.8.8 using get_ip_info").
- If you see lines like `Tool called: get_ip_info | args: ...` and `Tool get_ip_info completed in ...ms` in `mcp.log`, the platform is generating a valid MCP server and your client is invoking it correctly.

**If Cursor’s AI says it “doesn’t have access” to your MCP tools:**  
That means Cursor’s model is not calling your server (or isn’t being given your tools). Try:

1. **Confirm the server is running** – In **Settings → MCP**, ensure your server (e.g. `demo-api-toolkit`) is **enabled** and shows no error. Restart Cursor after changing `mcp.json`.
2. **Use a context where tools are available** – In Cursor, MCP tools are often available in **Composer** (agent) or when using the right chat mode. Open Composer and ask: “Use the get_ip_info tool to look up IP 8.8.8.8.”
3. **Phrase the request so the model uses the tool** – Ask explicitly: “Call the get_ip_info MCP tool with argument ip_address 8.8.8.8” or “Use your get_ip_info tool to look up 8.8.8.8.”
4. **Confirm with mcp.log** – If you use `run-with-log.mjs` and run `tail -f mcp.log`, you’ll see whether the server received a request. No new lines when you send a message means Cursor didn’t call your server.

If the server is enabled and you’re in the right mode but the model still refuses to call it, that’s a Cursor product limitation. Your Make MCP–generated server is valid; the client just has to send requests to it.

---

## Example: Location Lookup Tool

Here's a complete example using the free [Zippopotam.us](https://api.zippopotam.us) API:

### Tool Configuration

| Field | Value |
|-------|-------|
| **Name** | `get_location_by_zip` |
| **Description** | `Get location details (city, state, coordinates) for a US zip code` |
| **Execution Type** | `rest_api` |

### Input Schema
```json
{
  "type": "object",
  "properties": {
    "zip_code": {
      "type": "string",
      "description": "US ZIP code (e.g., 94538)"
    }
  },
  "required": ["zip_code"]
}
```

### Output Schema
```json
{
  "type": "object",
  "properties": {
    "country": { "type": "string" },
    "post code": { "type": "string" },
    "places": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "place name": { "type": "string" },
          "state": { "type": "string" },
          "latitude": { "type": "string" },
          "longitude": { "type": "string" }
        }
      }
    }
  }
}
```

### Execution Configuration
```json
{
  "url": "https://api.zippopotam.us/us/{{zip_code}}",
  "method": "GET",
  "headers": {}
}
```

### Test Input
```json
{
  "zip_code": "94538"
}
```

### Expected Output
```json
{
  "country": "United States",
  "country abbreviation": "US",
  "post code": "94538",
  "places": [
    {
      "place name": "Fremont",
      "longitude": "-121.9712",
      "latitude": "37.5308",
      "state": "California",
      "state abbreviation": "CA"
    }
  ]
}
```

---

## Deployment

### Local Development

```bash
make dev
```

### Docker Compose

```bash
docker-compose up --build -d
```

### Production Considerations

1. **Database:** Use managed PostgreSQL (AWS RDS, Cloud SQL)
2. **Secrets:** Use environment variables for sensitive config
3. **HTTPS:** Put behind a reverse proxy with TLS
4. **Authentication:** Add auth middleware for production use

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `DATABASE_URL` | PostgreSQL connection string | `postgres://postgres:postgres@localhost:5432/mcp_builder?sslmode=disable` |
| `PORT` | API server port | `8080` |
| `DEBUG` | Enable debug mode | `false` |

---

## Database Schema

```sql
-- Users & Auth
users (id, email, name, password_hash, created_at, updated_at)

-- Core
servers (id, name, description, version, icon, status, published_at, latest_version, owner_id, is_public, downloads, auth_config, created_at, updated_at)
tools (id, server_id, name, description, input_schema, output_schema, execution_type, execution_config, context_fields, output_display, read_only_hint, destructive_hint, created_at, updated_at)
resources (id, server_id, name, uri, mime_type, handler, created_at, updated_at)
prompts (id, server_id, name, description, template, arguments, created_at, updated_at)

-- Context & Governance
context_configs (id, server_id, name, source_type, config, created_at, updated_at)
policies (id, tool_id, name, description, enabled, created_at, updated_at)
policy_rules (id, policy_id, type, config, priority, fail_action)

-- Observability & Healing
tool_executions (id, tool_id, server_id, input, output, error, status_code, duration_ms, success, healing_applied, created_at)
healing_suggestions (id, tool_id, error_pattern, suggestion_type, suggestion, auto_apply, applied, created_at)

-- Versioning & Marketplace
server_versions (id, server_id, version, release_notes, snapshot, published_by, published_at)

-- Visual flows
flows (id, server_id, name, description, nodes, edges, created_at, updated_at)

-- Composition
server_compositions (id, name, description, server_ids, owner_id, created_at, updated_at)
```

---

## Security Score

Make MCP computes a **security score** (0–100%, grade A–F) for every server using the [SlowMist MCP Security Checklist](https://github.com/slowmist/MCP-Security-Checklist).

| Where | What you see |
|-------|----------------|
| **Server Editor → Security** | Current score, grade, and a checklist of criteria (e.g. input validation, rate limiting, access control, CLI allowlist). Unmet items show a short reason so you can improve the score. |
| **Marketplace** | Each published server shows a score badge on the card. In the server inspector, the **Security** tab shows the full criteria list. |

The score is based only on configuration we can evaluate (schemas, policies, hints, resources, versioning). For a full mapping of MCP security practices to Make MCP features, see [Security Best Practices](./security-best-practices.md).

---

## Next Steps

1. **Create your first server** - Follow the guide above
2. **Add governance policies** - Protect sensitive tools
3. **Configure context** - Enable multi-tenant AI agents
4. **Check your security score** - Use the **Security** tab in the server editor and address unmet criteria
5. **Monitor healing** - Watch for recurring errors
6. **Compose servers** - Build complex AI workflows

---

## Support

- **MCP Documentation:** https://modelcontextprotocol.io
- **Issues:** Open a GitHub issue

---

*Built with Go, React, and PostgreSQL*
