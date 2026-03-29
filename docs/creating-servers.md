# Creating MCP Servers

This guide walks you through creating a complete MCP (Model Context Protocol) server using the visual builder.

## Table of Contents

1. [Overview](#overview)
2. [Creating a New Server](#creating-a-new-server)
3. [Adding Tools](#adding-tools)
4. [Configuring Authentication](#configuring-authentication)
5. [Adding Resources](#adding-resources)
6. [Adding Prompts](#adding-prompts)
7. [Context Configuration](#context-configuration)
8. [Governance Policies](#governance-policies)
9. [Testing Your Server](#testing-your-server)
10. [Security Score](#security-score)
11. [Exporting & Deployment](#exporting--deployment)

---

## Overview

An MCP server consists of several components:

### In-app help and guided walkthroughs

While editing a server, the UI links to this guide where it matters:

- **Context** — "Read full Context Engine guide" in the Context tab.
- **Policies** — "Read full governance & policies guide" in the Policies tab.
- **Testing** — "Read testing best practices" in the Testing tab.
- **Healing** — Link to logging and healing in the Healing tab.

Use the in-app **Docs** (from the main nav or dashboard) to browse **Creating Servers** and **Security Best Practices** without leaving the app. These form the guided walkthrough for building and securing a server.

| Component | Description | Required |
|-----------|-------------|----------|
| **Server** | Container with name, description, version | Yes |
| **Tools** | Functions that AI agents can call | Yes (at least 1) |
| **Resources** | Data endpoints AI can read | No |
| **Prompts** | Templated instructions for AI | No |
| **Context Config** | User identity injection rules | No |
| **Policies** | Access control and governance | No |

<!-- TODO: Add image showing server architecture diagram -->

---

## Creating a New Server

### Step 1: Access the Dashboard

Navigate to the Dashboard and click the **"Create Server"** button.

<!-- TODO: Add screenshot of dashboard with Create Server button highlighted -->

### Step 2: Fill in Server Details

| Field | Description | Example |
|-------|-------------|---------|
| **Name** | Unique identifier for your server | `weather-api` |
| **Description** | What this server does | `Weather data and forecasting tools` |
| **Version** | Semantic version | `1.0.0` |

```json
{
  "name": "weather-api",
  "description": "Weather data and forecasting tools for AI agents",
  "version": "1.0.0"
}
```

### Step 3: Click Create

Your server is created and you're taken to the Server Editor where you can add tools, resources, and more.

<!-- TODO: Add screenshot of the Server Editor main view -->

---

## Adding Tools

Tools are the core functionality of your MCP server. They define what actions AI agents can perform.

### Step 1: Navigate to Tools Tab

In the Server Editor, click the **"Tools"** tab.

### Step 2: Click "Add Tool"

<!-- TODO: Add screenshot of empty tools view with Add Tool button -->

### Tool template gallery

When adding or editing a tool, the **Execution** (Basic) tab offers **quick templates** so you don’t start from scratch:

- **REST** — GET/POST/PUT/DELETE examples (e.g. GET request, Create resource, Update resource). Pick one to prefill name, description, input schema, and execution config (URL, method).
- **GraphQL** — GraphQL query/mutation templates with a sample endpoint and variables.
- **Database** — SQL query templates (e.g. Select by ID, Insert row) with a placeholder connection string.
- **CLI** — One-click templates for **Kubernetes** (`kubectl`), **Docker**, **Terraform**, **AWS CLI**, **Git**, **npm**, with suggested commands and input schema.

Choose an execution type, then use the **Quick … templates** row to apply a template; adjust name, URL, and schema as needed.

### Step 3: Configure Basic Info

| Field | Description | Example |
|-------|-------------|---------|
| **Tool Name** | Function name (see [tool naming rules](#tool-naming-rules) below) | `get_weather` |
| **Description** | What this tool does | `Get current weather for a location` |
| **Execution Type** | How the tool runs | `rest_api` |

#### Tool naming rules

Tool names follow MCP-style conventions (enforced in the API and UI):

- **Length:** 1–128 characters.
- **Charset:** ASCII letters (`A–Z`, `a–z`), digits (`0–9`), underscore (`_`), hyphen (`-`), and dot (`.`).
- **Uniqueness:** Names must be **unique per server** and are **case-sensitive** (`getUser` ≠ `getuser`).
- **Invalid:** Spaces, commas, and other special characters are not allowed.

Examples: `getUser`, `DATA_EXPORT_v2`, `admin.tools.list`.

#### Execution Types

| Type | Use Case | Example |
|------|----------|---------|
| `rest_api` | Call external REST APIs | Weather API, GitHub API |
| `graphql` | Execute GraphQL queries | Shopify, GitHub GraphQL |
| `webhook` | Send data to webhooks | Slack, Discord |
| `cli` | Execute shell commands (use `allowed_commands` for safety) | kubectl, docker, terraform, aws |
| `database` | Execute SQL queries | Internal databases |
| `javascript` | Run custom JS code | Data transformation |
| `python` | Run Python scripts | ML inference |
| `flow` | Visual pipeline: chain API, transform, and other nodes; convert to one tool | Multi-step workflows |

#### CLI Tools (DevOps)

The `cli` execution type enables powerful DevOps integrations. Quick templates are available for:

| Tool | Commands |
|------|----------|
| **Kubernetes** | `kubectl get pods`, `kubectl describe`, `kubectl logs` |
| **Docker** | `docker ps`, `docker logs`, `docker exec` |
| **Terraform** | `terraform plan`, `terraform apply`, `terraform state` |
| **AWS CLI** | `aws s3 ls`, `aws ec2 describe-instances` |
| **Git** | `git status`, `git log`, `git diff` |
| **npm** | `npm run`, `npm install`, `npm test` |

**CLI Configuration:**

```json
{
  "command": "kubectl get pods -n {{namespace}} -o json",
  "timeout": 30000,
  "working_dir": ".",
  "shell": "/bin/bash",
  "allowed_commands": ["kubectl", "docker", "terraform"],
  "env": {
    "KUBECONFIG": "/path/to/kubeconfig"
  }
}
```

| Field | Description |
|-------|-------------|
| `command` | Shell command with `{{variable}}` placeholders |
| `timeout` | Max execution time in milliseconds |
| `working_dir` | Working directory for command |
| `shell` | Shell to use (default: `/bin/bash`) |
| `allowed_commands` | Whitelist of allowed base commands (security) |
| `env` | Additional environment variables |

#### Reviewing tool changes (diff view)

When you **edit** an existing tool and click **Save**, the app shows a **Review tool changes** modal before applying. It lists each changed field with **Before** and **After** (name, description, execution type, context fields, security hints, input/output schema, execution config). Use it to confirm edits, then **Apply** or **Cancel**. No changes are written until you confirm.

#### Visual Flow Builder

The **flow** execution type lets you build a pipeline visually (nodes and edges), then **Convert to Tool** to expose it as a single MCP tool.

1. In the **Tools** tab, click **"Open Visual Builder"** (or add a tool and choose execution type **Flow**).
2. Add nodes (e.g. HTTP request, transform, merge) and connect them with edges.
3. Set the flow input and output. Save the flow.
4. Use **Test** to run the flow in the UI: **REST API** nodes perform real `http`/`https` requests (same URL and method as in the node config). **Transform** uses a **dot path** into the previous JSON (e.g. `origin` after [httpbin.org/get](https://httpbin.org/get), or `headers.Host` for nested keys). Leave transform empty to pass the payload through.
5. Use **"Convert to Tool"** to create a tool from the flow; the generated Node tool runs a richer pipeline (including JS-style expressions where supported).

### Step 4: Define Input Schema

The input schema defines what parameters the tool accepts. Use JSON Schema format:

```json
{
  "type": "object",
  "properties": {
    "location": {
      "type": "string",
      "description": "City name or ZIP code"
    },
    "units": {
      "type": "string",
      "enum": ["celsius", "fahrenheit"],
      "default": "celsius"
    }
  },
  "required": ["location"]
}
```

### Step 5: Define Output Schema

The output schema documents what the tool returns:

```json
{
  "type": "object",
  "properties": {
    "temperature": {
      "type": "number",
      "description": "Current temperature"
    },
    "conditions": {
      "type": "string",
      "description": "Weather conditions (sunny, cloudy, etc.)"
    },
    "humidity": {
      "type": "number",
      "description": "Humidity percentage"
    }
  }
}
```

### Step 6: Configure Execution

The execution config tells the tool how to run. For REST APIs:

```json
{
  "url": "https://api.weatherapi.com/v1/current.json?q={{location}}",
  "method": "GET",
  "headers": {
    "Content-Type": "application/json"
  }
}
```

#### Variable Substitution

Use `{{variable_name}}` to inject input parameters:

| Pattern | Description |
|---------|-------------|
| `{{location}}` | Replaced with the `location` input value |
| `{{user_id}}` | Replaced with context-injected user ID |
| `{{ENV_VAR}}` | Replaced with environment variable at runtime |

---

## Configuring Authentication

Most APIs require authentication. The Tool Builder provides a visual Authentication tab.

### Supported Auth Types

#### 1. API Key

For APIs that use header-based API keys:

| Field | Example |
|-------|---------|
| Header Name | `X-API-Key` |
| Prefix | (optional) `Api-Key` |
| Value | `your-api-key-here` |

**Generated Header:**
```
X-API-Key: your-api-key-here
```

<!-- TODO: Add screenshot of API Key configuration form -->

#### 2. Bearer Token

For JWT or OAuth access tokens:

| Field | Example |
|-------|---------|
| Token | `eyJhbGciOiJIUzI1NiIs...` |

**Generated Header:**
```
Authorization: Bearer eyJhbGciOiJIUzI1NiIs...
```

#### 3. Basic Auth

For username/password authentication:

| Field | Example |
|-------|---------|
| Username | `api_user` |
| Password | `secret123` |

**Generated Header:**
```
Authorization: Basic YXBpX3VzZXI6c2VjcmV0MTIz
```

#### 4. OAuth 2.0 (Client Credentials)

For OAuth 2.0 client credentials flow:

| Field | Example |
|-------|---------|
| Token URL | `https://auth.example.com/oauth/token` |
| Client ID | `client_abc123` |
| Client Secret | `secret_xyz789` |
| Scope | `read write` |

The generated server will automatically:
- Fetch tokens at runtime
- Cache tokens until expiry
- Refresh tokens before they expire

<!-- TODO: Add screenshot of OAuth 2.0 configuration form -->

### Using Environment Variables

For security, use environment variable placeholders instead of hardcoding secrets:

```
{{WEATHER_API_KEY}}
{{OAUTH_CLIENT_SECRET}}
```

At runtime, these are replaced with actual environment variables.

---

## Adding Resources

Resources are read-only data endpoints that AI agents can access.

### Step 1: Navigate to Resources Tab

### Step 2: Click "Add Resource"

### Step 3: Configure Resource

| Field | Description | Example |
|-------|-------------|---------|
| **Name** | Resource identifier | `api_documentation` |
| **URI** | Resource path | `docs://api/reference` |
| **Description** | What data this provides | `API reference documentation` |
| **MIME Type** | Content type | `text/markdown` |

### Example: Documentation Resource

```json
{
  "name": "api_documentation",
  "uri": "docs://weather-api/reference",
  "description": "Complete API reference for the weather service",
  "mime_type": "text/markdown",
  "content": "# Weather API Reference\n\n## Endpoints\n\n### GET /current\n..."
}
```

### Example: Sample Data Resource

```json
{
  "name": "sample_locations",
  "uri": "data://weather-api/sample-locations",
  "description": "Sample locations for testing",
  "mime_type": "application/json",
  "content": "{\"locations\": [\"New York\", \"London\", \"Tokyo\"]}"
}
```

---

## Adding Prompts

Prompts are templated instructions that help AI agents use your tools effectively.

### Step 1: Navigate to Prompts Tab

### Step 2: Click "Add Prompt"

### Step 3: Configure Prompt

| Field | Description |
|-------|-------------|
| **Name** | Prompt identifier |
| **Description** | When to use this prompt |
| **Arguments** | Required parameters |
| **Template** | The actual prompt text |

### Example: Weather Summary Prompt

```json
{
  "name": "weather_summary",
  "description": "Generate a weather summary for a location",
  "arguments": [
    {
      "name": "location",
      "description": "The location to summarize weather for",
      "required": true
    },
    {
      "name": "timeframe",
      "description": "today, week, or month",
      "required": false
    }
  ]
}
```

**Template:**
```
Provide a comprehensive weather summary for {{location}}.

Include:
1. Current conditions
2. Temperature (high/low)
3. Precipitation chance
4. Recommendations for outdoor activities

Timeframe: {{timeframe | default: "today"}}
```

### Example: Comparison Prompt

```json
{
  "name": "compare_weather",
  "description": "Compare weather between two locations",
  "arguments": [
    {
      "name": "location1",
      "required": true
    },
    {
      "name": "location2", 
      "required": true
    }
  ]
}
```

**Template:**
```
Compare the weather between {{location1}} and {{location2}}.

Create a table showing:
- Current temperature
- Humidity
- Conditions
- Best location for outdoor activities today
```

---

## Context Configuration

Context injection automatically adds user identity, permissions, and organization data to tool calls.

### Step 1: Navigate to Context Tab

### Step 2: Add Context Configuration

<!-- TODO: Add screenshot of Context Configuration form -->

### Extraction Sources

| Source | Description | Example |
|--------|-------------|---------|
| `jwt_claim` | Extract from JWT token | `sub`, `email`, `org_id` |
| `header` | Extract from HTTP header | `X-User-ID`, `X-Org-ID` |
| `query_param` | Extract from URL query | `?user_id=123` |

### Example: JWT Claims Extraction

```json
{
  "name": "jwt_user_context",
  "source_type": "jwt_claim",
  "source_key": "Authorization",
  "extractions": {
    "user_id": "sub",
    "email": "email",
    "organization_id": "org_id",
    "roles": "roles"
  }
}
```

### Example: Header Extraction

```json
{
  "name": "header_context",
  "source_type": "header",
  "extractions": {
    "user_id": "X-User-ID",
    "organization_id": "X-Organization-ID",
    "trace_id": "X-Trace-ID"
  }
}
```

### Using Context in Tools

Add context fields to your tool:

```json
{
  "name": "get_user_data",
  "context_fields": ["user_id", "organization_id", "permissions"]
}
```

At runtime, the tool receives:

```json
{
  "customer_id": "123",
  "context": {
    "user_id": "user_abc",
    "organization_id": "org_xyz",
    "permissions": ["read", "write"]
  }
}
```

---

## Governance Policies

Policies control who can use tools and how.

### Step 1: Navigate to Policies Tab

### Step 2: Create Policy

<!-- TODO: Add screenshot of Policy Editor form -->

### Policy templates and wizard

You can start from a **template** instead of editing JSON by hand:

1. Select the tool, then in **Start from template** choose a recipe:
   - **Payment limits** — Cap a numeric input field (e.g. `amount`) at a maximum value.
   - **Approval for production writes** — Require human or manager approval before the tool runs.
   - **Weekend freeze** — Block tool calls on Saturday and Sunday.
   - **Allowed roles only** — Restrict access to specific user roles from context.
   - **Business hours only** — Allow calls only during configured hours (e.g. 9–17).

2. Click the recipe; a short **wizard** asks a few questions (e.g. field name, max amount, roles, timezone).

3. Click **Build policy** — the policy form is prefilled. Review, add more rules if needed, then **Create Policy**.

### Policy Rule Types

| Rule Type | Description | Example |
|-----------|-------------|---------|
| `max_value` | Limit numeric parameter values | Max transfer: $10,000 |
| `allowed_roles` | Require specific roles | Only `admin`, `manager` |
| `time_window` | Restrict to business hours | 9am-5pm weekdays |
| `approval_required` | Require human approval | For amounts > $5,000 |
| `rate_limit` | Limit calls per time period | 100 calls/hour |

### Example: Payment Protection Policy

```json
{
  "name": "payment_protection",
  "description": "Protect high-value transactions",
  "tool_name": "process_payment",
  "enabled": true,
  "rules": [
    {
      "type": "max_value",
      "field": "amount",
      "value": 10000,
      "message": "Payments over $10,000 require manual approval"
    },
    {
      "type": "allowed_roles",
      "roles": ["admin", "finance"],
      "message": "Only admin and finance can process payments"
    },
    {
      "type": "time_window",
      "start_hour": 9,
      "end_hour": 17,
      "timezone": "America/New_York",
      "message": "Payments only allowed during business hours"
    }
  ]
}
```

### Example: API Rate Limiting

```json
{
  "name": "api_rate_limit",
  "description": "Prevent API abuse",
  "tool_name": "*",
  "enabled": true,
  "rules": [
    {
      "type": "rate_limit",
      "max_calls": 100,
      "window_seconds": 3600,
      "message": "Rate limit exceeded (100 calls/hour)"
    }
  ]
}
```

---

## Testing Your Server

Before deployment, test your tools in the built-in playground.

### Step 1: Navigate to Test Tab

<!-- TODO: Add screenshot of Test Playground -->

### Step 2: Select a Tool

Choose the tool you want to test from the dropdown.

### Step 3: Provide Test Input

Enter sample input in JSON format:

```json
{
  "location": "94539"
}
```

### Step 4: Simulate Context (Optional)

Add mock context to test context-aware tools:

```json
{
  "user_id": "test_user_123",
  "organization_id": "test_org",
  "roles": ["admin"]
}
```

### Environment profile (Dev / Staging / Prod)

When testing, you can choose an **Environment** (Dev, Staging, or Prod). The selected profile overrides **base URL** for REST/GraphQL/Webhook tools and **database URL** for database tools, so you can run the same tool against different backends without changing the tool config. Configure the URLs for each profile in the **Environments** tab in the left menu (below General). See [Environment profiles](#environment-profiles-dev--staging--prod).

### Per-tool test presets

You can **save** the current Input + Context as a named **preset** for the selected tool (e.g. "High-amount payment", "Admin role"). Presets are stored **per user** in the database (sign-in required). Use the **Presets** dropdown to **Select preset…**, **Save current as preset**, or **Delete preset**. Handy for quickly switching between test scenarios (e.g. different roles or amounts) without re-typing JSON.

### Step 5: Execute

Click **"Execute Tool"** to test. View:
- Response data
- Execution time
- Any errors

### Dry-run mode for destructive tools

For tools marked **Destructive** (Tool Editor → Schema → Security hints), the Testing tab shows a **Dry-run** checkbox: *"Dry-run (skip real execution; preview input + context only)"*. When checked, clicking **Execute Tool** does **not** call the real tool; the result shows a preview payload (tool id, execution type, input, context) and the policy decision, so you can verify policies and context without performing the destructive action.

### Policy decision (why allowed or denied)

After each run, the **Policy decision** panel shows whether the call was **Allowed**, **Denied**, or **Approval required**, with a short reason and the list of rules that were violated (if any). If all rules passed, it shows “All rules passed.” This helps you see why a call succeeded or was blocked.

### What-if? Policy simulation

You can simulate policy evaluation **without executing the tool**:

1. Select a tool, then click **What if? Simulate policy with different input/context**.
2. Edit the **Input** and **Context** JSON (e.g. change `roles` or `amount`).
3. Click **Simulate**.

The result shows the overall decision (Allowed / Denied / Approval required) and **per-rule results**: each policy and rule is listed as Passed or Failed with a message. Use this to test different contexts (e.g. different roles or amounts) and see which rules would fire.

### Step 6: Review Healing Suggestions

If the tool fails, check the **Healing** tab for auto-detected issues and suggested fixes.

<!-- TODO: Add screenshot of Healing Dashboard with error analysis -->

---

## Security Score

Before publishing, check how your server scores against the [SlowMist MCP Security Checklist](https://github.com/slowmist/MCP-Security-Checklist).

### Where to find it

- In the Server Editor, open **Security** in the left navigation (shield icon).

### What you see

- **Score (0–100%)** and **grade (A–F)** based on criteria we can evaluate from your configuration.
- A list of **checklist criteria** with ✓ (met) or ✗ (not met). Unmet items include a short reason (e.g. “Define input_schema with type and properties for every tool”).
- A link to the full SlowMist checklist for reference.

### How to improve your score

| Criterion | What to do |
|-----------|------------|
| Input validation | Ensure every tool has an `input_schema` with `type: "object"` and `properties` (or `required`). |
| Rate limiting | Add a **Policy** with a **Rate limit** rule to tools that call external APIs. |
| Access control | Attach at least one **Policy** to every tool marked **Destructive**. |
| Tool hints | In the Tool Editor → **Schema** tab, use **Security hints** to mark tools as **Read-only** or **Destructive**. |
| CLI allowlist | For CLI tools, set **allowed_commands** in the execution config so only approved commands can run. |
| Version / pinning | Set a server **Version** and publish versions so consumers can pin to a specific release. |

When you **publish** a server, the same score is shown in the **Marketplace** on the server card and in the inspector’s **Security** tab. For more detail, see [Security Best Practices](./security-best-practices.md).

---

## Exporting & Deployment

From the Server Editor, open **Deploy** in the left nav.

### Environment profiles (Dev / Staging / Prod)

In the Server Editor left menu, open the **Environments** tab (below General). There you define per-environment settings:

- **Dev**, **Staging**, **Prod** — each has:
  - **Base URL** — Used for REST API, GraphQL, and Webhook tools. If the tool’s execution config uses `{{BASE_URL}}` or a path starting with `/`, this URL is applied when you run tests or (in generated code) at runtime via env.
  - **Database URL** — Used for Database tools. Overrides the connection string for the selected profile.

Click **Save environment profiles** to store them. The **Testing** tab uses the selected profile when you run tools; **Deploy** uses this same configuration when generating the server.

### Node.js (Download ZIP)

1. Select **Node.js** and click **"Generate & Download"**.
2. Extract the ZIP (folder name is based on server name, e.g. `my-server-mcp-server`).
3. Run:

```bash
cd my-server-mcp-server
npm install
npm run build
npm start
```

### Docker

1. Select **Docker** in the Deploy tab. The generated server includes a Dockerfile (non-root user).
2. Build and run:

```bash
docker build -t my-mcp-server .
docker run -p 3000:3000 my-mcp-server
```

Configure your MCP client to connect to the server (stdio or the port your server uses).

### Push to GitHub

1. Select **GitHub** in the Deploy tab.
2. Provide a GitHub token (with `repo` scope), owner, repo name, and optionally create the repo.
3. The generated server is pushed to the repository.

### Publish MCP (Hosted URL)

1. Select **Publish MCP** in the Deploy tab.
2. Configure **Access & Security**:
   - **Endpoint protection**: `No auth` or `Bearer token`
   - **Caller identity**: optional/required toggle (`X-Make-MCP-Caller-Id`)
   - **Idle shutdown**: stop container after inactivity window
3. Click **Publish MCP** (or **Re-publish MCP** if already running).
4. Use one-click install buttons (Cursor / VS Code / VS Code Insiders), or copy manual MCP JSON config.

Hosted URL format:

```text
http(s)://<host>/api/users/<user_id>/<server_slug>
```

Notes:

- Hosted routing is **versionless** in the URL.
- Each publish stores a **hosted-only snapshot** internally and starts/reuses a managed container.
- Hosted snapshots are tracked separately from normal semantic release versions so your `Version` field in server configuration remains stable.
- Runtime metadata in Deploy shows the active hosted snapshot id/version, started-at, and last-ensured timestamps.

### MCP Client Configuration

Manual config (for clients without deep-link install):

```json
{
  "mcpServers": {
    "weather-api": {
      "command": "node",
      "args": ["/path/to/my-server-mcp-server/dist/server.js"]
    }
  }
}
```

To verify the client invokes your server, use `run-with-log.mjs` as the entry point and run `tail -f mcp.log` (see [Getting Started - Verifying](../getting-started.md#verifying-that-your-client-eg-cursor-invokes-the-server)).

For **hosted URL** servers, the process runs remotely inside a managed container. Any runtime environment variables (including observability vars) must be configured by the hosting platform; local `mcp.json` env is not the source of truth for hosted runtime configuration.

For supported IDEs, prefer **one-click install** from Deploy:

- Cursor deep link install
- VS Code protocol install
- VS Code Insiders protocol install

### Environment Variables

Set required environment variables before running:

```bash
export WEATHER_API_KEY="your-api-key"
export OAUTH_CLIENT_SECRET="your-secret"
npm start
```

---

## Complete Example: GitHub API Server

Here's a complete example of building a GitHub API server:

### Server Config

| Field | Value |
|-------|-------|
| Name | `github-api` |
| Description | `GitHub repository and user tools` |
| Version | `1.0.0` |

### Tool 1: Get User

```json
{
  "name": "get_github_user",
  "description": "Get GitHub user profile information",
  "execution_type": "rest_api",
  "input_schema": {
    "type": "object",
    "properties": {
      "username": {
        "type": "string",
        "description": "GitHub username"
      }
    },
    "required": ["username"]
  },
  "execution_config": {
    "url": "https://api.github.com/users/{{username}}",
    "method": "GET",
    "headers": {
      "Accept": "application/vnd.github.v3+json",
      "User-Agent": "MCP-Server"
    }
  }
}
```

### Tool 2: List Repositories

```json
{
  "name": "list_repos",
  "description": "List repositories for a user",
  "execution_type": "rest_api",
  "input_schema": {
    "type": "object",
    "properties": {
      "username": {
        "type": "string"
      },
      "sort": {
        "type": "string",
        "enum": ["created", "updated", "pushed", "full_name"],
        "default": "updated"
      }
    },
    "required": ["username"]
  },
  "execution_config": {
    "url": "https://api.github.com/users/{{username}}/repos?sort={{sort}}",
    "method": "GET",
    "headers": {
      "Accept": "application/vnd.github.v3+json"
    }
  }
}
```

### Tool 3: Search Code (Authenticated)

```json
{
  "name": "search_code",
  "description": "Search code across GitHub",
  "execution_type": "rest_api",
  "input_schema": {
    "type": "object",
    "properties": {
      "query": {
        "type": "string",
        "description": "Search query"
      },
      "language": {
        "type": "string",
        "description": "Filter by programming language"
      }
    },
    "required": ["query"]
  },
  "execution_config": {
    "url": "https://api.github.com/search/code?q={{query}}+language:{{language}}",
    "method": "GET",
    "headers": {
      "Accept": "application/vnd.github.v3+json"
    },
    "auth": {
      "type": "bearer_token",
      "bearerToken": {
        "token": "{{GITHUB_TOKEN}}"
      }
    }
  }
}
```

### Resource: API Documentation

```json
{
  "name": "github_api_docs",
  "uri": "docs://github-api/reference",
  "description": "Quick reference for available GitHub API tools",
  "mime_type": "text/markdown",
  "content": "# GitHub API Tools\n\n- `get_github_user`: Get user profile\n- `list_repos`: List user repositories\n- `search_code`: Search code (requires auth)"
}
```

### Prompt: Repository Analysis

```json
{
  "name": "analyze_repos",
  "description": "Analyze a user's repositories",
  "arguments": [
    {
      "name": "username",
      "required": true
    }
  ]
}
```

**Template:**
```
Analyze the GitHub repositories for user {{username}}.

1. First, get the user profile to understand their background
2. List their repositories sorted by recent activity
3. Identify:
   - Primary programming languages
   - Most popular repositories (by stars)
   - Recent activity patterns
   - Notable projects

Provide a summary of their GitHub presence and expertise.
```

---

## Complete Example: DevOps CLI Server

Here's a complete example of building a DevOps server with CLI tools for Kubernetes and Docker:

### Server Config

| Field | Value |
|-------|-------|
| Name | `devops-toolkit` |
| Description | `Kubernetes and Docker management tools` |
| Version | `1.0.0` |

### Tool 1: List Kubernetes Pods

```json
{
  "name": "kubectl_get_pods",
  "description": "List pods in a Kubernetes namespace",
  "execution_type": "cli",
  "input_schema": {
    "type": "object",
    "properties": {
      "namespace": {
        "type": "string",
        "description": "Kubernetes namespace",
        "default": "default"
      },
      "label_selector": {
        "type": "string",
        "description": "Label selector (e.g., app=nginx)"
      }
    },
    "required": ["namespace"]
  },
  "execution_config": {
    "command": "kubectl get pods -n {{namespace}} -l {{label_selector}} -o json",
    "timeout": 30000,
    "shell": "/bin/bash",
    "allowed_commands": ["kubectl"]
  }
}
```

### Tool 2: Get Pod Logs

```json
{
  "name": "kubectl_logs",
  "description": "Get logs from a Kubernetes pod",
  "execution_type": "cli",
  "input_schema": {
    "type": "object",
    "properties": {
      "pod_name": {
        "type": "string",
        "description": "Name of the pod"
      },
      "namespace": {
        "type": "string",
        "default": "default"
      },
      "tail_lines": {
        "type": "number",
        "default": 100
      },
      "container": {
        "type": "string",
        "description": "Container name (for multi-container pods)"
      }
    },
    "required": ["pod_name"]
  },
  "execution_config": {
    "command": "kubectl logs {{pod_name}} -n {{namespace}} --tail={{tail_lines}}",
    "timeout": 30000,
    "allowed_commands": ["kubectl"]
  }
}
```

### Tool 3: Docker Container Status

```json
{
  "name": "docker_ps",
  "description": "List running Docker containers",
  "execution_type": "cli",
  "input_schema": {
    "type": "object",
    "properties": {
      "all": {
        "type": "boolean",
        "description": "Show all containers (including stopped)",
        "default": false
      },
      "filter": {
        "type": "string",
        "description": "Filter (e.g., name=myapp)"
      }
    }
  },
  "execution_config": {
    "command": "docker ps --format 'table {{.Names}}\\t{{.Status}}\\t{{.Ports}}'",
    "timeout": 10000,
    "allowed_commands": ["docker"]
  }
}
```

### Tool 4: Terraform Plan

```json
{
  "name": "terraform_plan",
  "description": "Preview Terraform infrastructure changes",
  "execution_type": "cli",
  "input_schema": {
    "type": "object",
    "properties": {
      "working_dir": {
        "type": "string",
        "description": "Path to Terraform configuration"
      },
      "var_file": {
        "type": "string",
        "description": "Path to variables file"
      }
    },
    "required": ["working_dir"]
  },
  "execution_config": {
    "command": "terraform plan -var-file={{var_file}} -no-color",
    "working_dir": "{{working_dir}}",
    "timeout": 120000,
    "allowed_commands": ["terraform"],
    "env": {
      "TF_IN_AUTOMATION": "true"
    }
  }
}
```

### Prompt: Kubernetes Troubleshooting

```json
{
  "name": "k8s_troubleshoot",
  "description": "Troubleshoot Kubernetes issues",
  "arguments": [
    {
      "name": "namespace",
      "required": true
    },
    {
      "name": "issue_description",
      "required": true
    }
  ]
}
```

**Template:**
```
Troubleshoot the following Kubernetes issue in namespace {{namespace}}:

Issue: {{issue_description}}

Steps:
1. List all pods in the namespace to check their status
2. For any pods not in Running state, get their logs
3. Check for common issues:
   - CrashLoopBackOff: Check logs for startup errors
   - ImagePullBackOff: Verify image name and registry access
   - Pending: Check node resources and scheduling constraints
4. Provide specific remediation steps
```

---

## Next Steps

- [Server Compositions](./compositions.md) - Combine multiple servers
- [Getting Started Guide](./getting-started.md) - Full platform overview
- [API Reference](./getting-started.md#api-reference) - REST API documentation
