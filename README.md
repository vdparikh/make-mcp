# MCP Server Builder

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

<div align="center">
  <img src="/docs/image_1.png" width="30%" />
  <img src="/docs/image_2.png" width="30%" />
  <img src="/docs/image_3.png" width="30%" />
</div>

---

## Features

- **Visual Server Builder** — Create MCP servers through a drag-and-drop interface
- **Multiple Execution Types** — REST API, GraphQL, Webhooks, Database queries
- **Built-in Authentication** — API Key, Bearer Token, Basic Auth, OAuth 2.0 with visual configuration
- **Live Testing Playground** — Test tools before deployment
- **One-Click Export** — Download as Node.js project, ready to run
- **Context-Aware Execution** — Auto-inject user identity, permissions, org data
- **AI Governance Layer** — Policy engine to control tool access (rate limits, roles, approvals)
- **Self-Healing Tools** — Auto-detect failures and suggest fixes
- **Server Composition** — Combine multiple MCP servers into one

## Quick Start

### Prerequisites

- Go 1.22+
- Node.js 20+
- PostgreSQL 16+ (or Docker)

### Using Docker Compose (Recommended)

```bash
git clone https://github.com/vdparikh/make-mcp.git
cd make-mcp
docker-compose up --build
```

Open http://localhost:3000

### Manual Setup

```bash
# 1. Start PostgreSQL
docker run -d --name mcp-builder-db \
  -e POSTGRES_USER=postgres \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=mcp_builder \
  -p 5432:5432 \
  postgres:16-alpine

# 2. Start Backend
cd backend
go mod download
go run ./cmd/server

# 3. Start Frontend (new terminal)
cd frontend
npm install
npm run dev
```

Open http://localhost:3000

## Demo Server

On first startup, a **Demo API Toolkit** is automatically created with 8 working tools:

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

## Screenshots

### Dashboard
Create and manage your MCP servers in one place.

### Tool Builder
Visual schema editor with multiple execution types.

### Testing Playground
Test tools with mock input before deployment.

### Governance Policies
Define rules to control AI tool access.

## Architecture

```
┌─────────────────────┐     ┌─────────────────────┐
│   React Frontend    │────▶│    Go Backend API   │
│   (Port 3000)       │     │    (Port 8080)      │
└─────────────────────┘     └──────────┬──────────┘
                                       │
                            ┌──────────▼──────────┐
                            │     PostgreSQL      │
                            │    (Port 5432)      │
                            └─────────────────────┘
```

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

## Documentation

- [Getting Started Guide](./getting_started.md) — Full setup and usage guide

## Tech Stack

| Component | Technology |
|-----------|------------|
| Frontend | React, TypeScript, Vite, Bootstrap |
| Backend | Go, Gin, PostgreSQL |
| Code Editor | Monaco Editor |
| Generated Servers | Node.js, TypeScript, MCP SDK |


## License

MIT License - see [LICENSE](./LICENSE) for details.

