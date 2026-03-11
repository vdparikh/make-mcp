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
8. [Example: Location Lookup Tool](#example-location-lookup-tool)
9. [Deployment](#deployment)

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

Open http://localhost:3000

**Note:** On first startup, a fully functional **Demo API Toolkit** server is automatically created with 8 working tools, sample resources, prompts, context configs, and policies. Use it as a model for building your own servers!

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
│   │   ├── database/database.go      # PostgreSQL + migrations
│   │   ├── models/models.go          # Data models
│   │   ├── generator/generator.go    # MCP server code generator
│   │   ├── context/engine.go         # Context Engine (Feature 1)
│   │   ├── governance/engine.go      # Policy Engine (Feature 2)
│   │   └── healing/engine.go         # Self-Healing Engine (Feature 3)
│   ├── go.mod
│   └── Dockerfile
│
├── frontend/                         # React UI
│   ├── src/
│   │   ├── pages/
│   │   │   ├── Dashboard.tsx         # Server list
│   │   │   ├── ServerEditor.tsx      # Server configuration
│   │   │   └── Compositions.tsx      # Server composition
│   │   ├── components/
│   │   │   ├── ToolEditor.tsx        # Tool builder
│   │   │   ├── ResourceEditor.tsx    # Resource builder
│   │   │   ├── PromptEditor.tsx      # Prompt builder
│   │   │   ├── ContextConfigEditor.tsx
│   │   │   ├── PolicyEditor.tsx      # Governance policies
│   │   │   ├── TestPlayground.tsx    # Live testing
│   │   │   └── HealingDashboard.tsx  # Self-healing monitoring
│   │   ├── services/api.ts           # API client
│   │   ├── types/index.ts            # TypeScript types
│   │   └── styles/App.css            # Styling
│   ├── package.json
│   └── Dockerfile
│
├── docker-compose.yml
├── Makefile
└── getting_started.md
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
| `javascript` | Run JavaScript code |
| `python` | Run Python scripts |
| `database` | Execute SQL queries |

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

### 4. Server Code Generation

Export MCP servers as:
- **ZIP package** - Node.js project ready to run
- Docker container (coming soon)
- Cloud deployment (coming soon)

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

### Servers

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/servers` | List all servers |
| POST | `/api/servers` | Create server |
| GET | `/api/servers/:id` | Get server with tools/resources/prompts |
| PUT | `/api/servers/:id` | Update server |
| DELETE | `/api/servers/:id` | Delete server |
| POST | `/api/servers/:id/generate` | Generate & download ZIP |

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
| GET | `/api/compositions` | List compositions |
| POST | `/api/compositions` | Create composition |

---

## Creating Your First MCP Server

### Step 1: Create a Server

1. Click **"New Server"** on the Dashboard
2. Enter:
   - **Name:** `weather-service`
   - **Description:** `Weather and location tools for AI agents`
3. Click **Create Server**

### Step 2: Add a Tool

1. Open the server and go to **Tools** tab
2. Click **"Add Tool"**
3. Configure:

**Basic Info:**
- **Name:** `get_location_by_zip`
- **Description:** `Get location details for a US zip code`
- **Execution Type:** `REST API`

**Input Schema:**
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

**Execution Config:**
```json
{
  "url": "https://api.zippopotam.us/us/{{zip_code}}",
  "method": "GET",
  "headers": {}
}
```

4. Click **Create Tool**

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

1. Go to **Deploy** tab
2. Click **"Download ZIP"**
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
      "args": ["/path/to/dist/server.js"]
    }
  }
}
```

---

## Example: Location Lookup Tool

Here's a complete example using the free [Zippopotam.us](https://api.zippopotam.us) API:

### Tool Configuration

| Field | Value |
|-------|-------|
| **Name** | `get_location_by_zip` |
| **Description** | `Get location details (city, state, coordinates) for a US zip code` |
| **Execution Type** | `REST API` |

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
-- Core tables
servers (id, name, description, version, auth_config, created_at, updated_at)
tools (id, server_id, name, description, input_schema, output_schema, execution_type, execution_config, context_fields)
resources (id, server_id, name, uri, mime_type, handler)
prompts (id, server_id, name, description, template, arguments)

-- Context & Governance
context_configs (id, server_id, name, source_type, config)
policies (id, tool_id, name, description, enabled)
policy_rules (id, policy_id, type, config, priority, fail_action)

-- Observability & Healing
tool_executions (id, tool_id, server_id, input, output, error, status_code, duration_ms, success, healing_applied)
healing_suggestions (id, tool_id, error_pattern, suggestion_type, suggestion, auto_apply, applied)

-- Composition
server_compositions (id, name, description, server_ids)
```

---

## Next Steps

1. **Create your first server** - Follow the guide above
2. **Add governance policies** - Protect sensitive tools
3. **Configure context** - Enable multi-tenant AI agents
4. **Monitor healing** - Watch for recurring errors
5. **Compose servers** - Build complex AI workflows

---

## Support

- **MCP Documentation:** https://modelcontextprotocol.io
- **Issues:** Open a GitHub issue

---

*Built with Go, React, and PostgreSQL*
