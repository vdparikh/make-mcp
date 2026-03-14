# Server Compositions

Server Compositions allow you to combine multiple MCP servers into a single, unified server. This enables modular architecture where each server handles a specific domain, then compositions bring them together for AI agents.

## Table of Contents

1. [Overview](#overview)
2. [Why Use Compositions?](#why-use-compositions)
3. [Creating a Composition](#creating-a-composition)
4. [Composition Strategies](#composition-strategies)
5. [Conflict Resolution](#conflict-resolution)
6. [Examples](#examples)
7. [Best Practices](#best-practices)
8. [Exporting Compositions](#exporting-compositions)

---

## Overview

A composition is a meta-server that combines tools, resources, and prompts from multiple child servers.

```
┌─────────────────────────────────────────────────────────────┐
│                    Composed Server                          │
│                "Enterprise AI Toolkit"                      │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐ │
│  │  Weather    │  │   GitHub    │  │    Internal         │ │
│  │  Server     │  │   Server    │  │    APIs Server      │ │
│  │             │  │             │  │                     │ │
│  │ • get_weather│ │ • get_user  │  │ • get_employee      │ │
│  │ • forecast  │  │ • list_repos│  │ • submit_expense    │ │
│  │             │  │ • search    │  │ • book_meeting      │ │
│  └─────────────┘  └─────────────┘  └─────────────────────┘ │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

<!-- TODO: Add screenshot of Compositions page showing combined servers -->

---

## Why Use Compositions?

### Modular Architecture

| Benefit | Description |
|---------|-------------|
| **Separation of Concerns** | Each server handles one domain |
| **Reusability** | Share servers across multiple compositions |
| **Maintainability** | Update one server without affecting others |
| **Team Ownership** | Different teams own different servers |

### Use Cases

| Scenario | Composition |
|----------|-------------|
| **Enterprise Assistant** | HR tools + IT tools + Finance tools |
| **Developer Copilot** | GitHub + CI/CD + Documentation + Monitoring |
| **Sales AI** | CRM + Email + Calendar + Analytics |
| **Support Bot** | Ticketing + Knowledge Base + Customer Data |

---

## Creating a Composition

You must be **logged in** to create or view compositions; they are scoped to your user.

### Step 1: Navigate to Compositions

Click **"Compositions"** in the sidebar navigation.

<!-- TODO: Add screenshot of sidebar with Compositions highlighted -->

### Step 2: Click "Create Composition"

<!-- TODO: Add screenshot of Create Composition button -->

### Step 3: Configure Composition Details

| Field | Description | Example |
|-------|-------------|---------|
| **Name** | Composition identifier | `enterprise-toolkit` |
| **Description** | What this composition provides | `All-in-one toolkit for enterprise AI` |
| **Version** | Semantic version | `1.0.0` |

### Step 4: Select Servers to Include

Choose which servers to combine:

<!-- TODO: Add screenshot of server selection grid -->

**Selection Interface:**
- View all available servers
- See tool/resource/prompt counts for each
- Click to toggle selection
- Selected servers are highlighted

### Step 5: Configure Composition Options

| Option | Description | Default |
|--------|-------------|---------|
| **Prefix Tool Names** | Add server name prefix to avoid conflicts | `false` |
| **Merge Resources** | Combine all resources | `true` |
| **Merge Prompts** | Combine all prompts | `true` |

### Step 6: Save Composition

Click **"Create Composition"** to save.

---

## Composition Strategies

### Strategy 1: Domain Grouping

Group servers by business domain:

```
Sales Composition
├── crm-server (Salesforce tools)
├── email-server (Gmail/Outlook tools)
├── calendar-server (scheduling tools)
└── analytics-server (reporting tools)
```

### Strategy 2: Capability Layering

Layer servers by capability type:

```
Full-Stack Composition
├── data-layer (database tools)
├── api-layer (external API tools)
├── compute-layer (processing tools)
└── notification-layer (alerts/messaging)
```

### Strategy 3: Environment-Based

Different compositions for different environments:

```
Development Composition
├── github-server
├── local-db-server
└── mock-payment-server

Production Composition
├── github-server
├── production-db-server
└── stripe-server
```

---

## Conflict Resolution

When multiple servers have tools with the same name, you need a resolution strategy.

### Tool Name Conflicts

**Problem:** Two servers both have a `get_user` tool.

**Solutions:**

#### 1. Prefix with Server Name

Enable **"Prefix Tool Names"** option:

| Original | Prefixed |
|----------|----------|
| `github-server.get_user` | `github_get_user` |
| `crm-server.get_user` | `crm_get_user` |

#### 2. Rename Before Composing

Edit one server to use a different name:

| Server | Renamed Tool |
|--------|--------------|
| GitHub Server | `get_github_user` |
| CRM Server | `get_crm_contact` |

#### 3. Exclude Duplicate

Choose which server's tool to include and exclude the other.

### Resource URI Conflicts

Resources must have unique URIs. Use different URI schemes:

| Server | URI Pattern |
|--------|-------------|
| Weather | `weather://data/*` |
| GitHub | `github://repos/*` |
| CRM | `crm://contacts/*` |

---

## Examples

### Example 1: Developer Productivity Suite

Combine tools for software development:

**Included Servers:**

| Server | Tools |
|--------|-------|
| `github-server` | `get_user`, `list_repos`, `create_issue`, `search_code` |
| `jira-server` | `get_ticket`, `create_ticket`, `update_status` |
| `slack-server` | `send_message`, `create_channel`, `list_channels` |
| `docs-server` | `search_docs`, `get_page`, `update_page` |

**Composition Config:**

```json
{
  "name": "developer-suite",
  "description": "All-in-one toolkit for developers",
  "version": "1.0.0",
  "servers": [
    "github-server",
    "jira-server", 
    "slack-server",
    "docs-server"
  ],
  "options": {
    "prefix_tools": false,
    "merge_resources": true,
    "merge_prompts": true
  }
}
```

**Resulting Tools (12 total):**

```
get_user, list_repos, create_issue, search_code,
get_ticket, create_ticket, update_status,
send_message, create_channel, list_channels,
search_docs, get_page, update_page
```

<!-- TODO: Add screenshot of Developer Suite composition -->

---

### Example 2: Customer Support Platform

Combine tools for support agents:

**Included Servers:**

| Server | Purpose |
|--------|---------|
| `zendesk-server` | Ticket management |
| `customer-db-server` | Customer data lookup |
| `knowledge-base-server` | Article search |
| `slack-server` | Internal escalation |

**Composition Config:**

```json
{
  "name": "support-platform",
  "description": "Customer support AI toolkit",
  "version": "2.0.0",
  "servers": [
    "zendesk-server",
    "customer-db-server",
    "knowledge-base-server",
    "slack-server"
  ],
  "options": {
    "prefix_tools": true
  }
}
```

**Use Case Flow:**

```
1. Customer reports issue
2. AI uses zendesk_create_ticket
3. AI uses customer_get_profile to understand context
4. AI uses kb_search_articles to find solutions
5. If escalation needed: slack_send_message to #support-escalation
```

---

### Example 3: Data Analysis Workbench

Combine tools for data scientists:

**Included Servers:**

| Server | Tools |
|--------|-------|
| `sql-server` | `query_database`, `list_tables`, `describe_table` |
| `python-runner` | `execute_script`, `install_package` |
| `visualization-server` | `create_chart`, `export_image` |
| `storage-server` | `upload_file`, `download_file`, `list_files` |

**Composition Config:**

```json
{
  "name": "data-workbench",
  "description": "Data analysis and visualization toolkit",
  "version": "1.0.0",
  "servers": [
    "sql-server",
    "python-runner",
    "visualization-server",
    "storage-server"
  ]
}
```

**Combined Resources:**

| Resource | URI | Source Server |
|----------|-----|---------------|
| SQL Reference | `docs://sql/reference` | sql-server |
| Python Libraries | `docs://python/libraries` | python-runner |
| Chart Templates | `templates://charts/*` | visualization-server |
| Sample Datasets | `data://samples/*` | storage-server |

---

### Example 4: Enterprise AI Assistant

The ultimate composition combining everything:

```json
{
  "name": "enterprise-assistant",
  "description": "Complete enterprise AI assistant",
  "version": "3.0.0",
  "servers": [
    "hr-server",
    "it-helpdesk-server",
    "finance-server",
    "calendar-server",
    "email-server",
    "document-server",
    "analytics-server"
  ],
  "options": {
    "prefix_tools": true,
    "merge_resources": true,
    "merge_prompts": true
  }
}
```

**Tool Categories (with prefixes):**

| Category | Tools |
|----------|-------|
| HR | `hr_get_employee`, `hr_submit_pto`, `hr_get_benefits` |
| IT | `it_create_ticket`, `it_reset_password`, `it_request_access` |
| Finance | `finance_submit_expense`, `finance_get_budget`, `finance_approve_po` |
| Calendar | `calendar_schedule_meeting`, `calendar_find_availability` |
| Email | `email_send`, `email_search`, `email_get_thread` |
| Documents | `docs_search`, `docs_create`, `docs_share` |
| Analytics | `analytics_run_report`, `analytics_get_dashboard` |

---

## Best Practices

### 1. Plan Your Naming Convention

Before building servers, decide on a naming strategy:

| Pattern | Example | Use When |
|---------|---------|----------|
| `domain_action_target` | `github_get_user` | Many servers, explicit clarity |
| `action_target` | `get_user` | Few servers, no conflicts |
| `target_action` | `user_get` | Grouping by entity |

### 2. Design for Composition

When building individual servers:

- ✅ Use unique, descriptive tool names
- ✅ Use consistent input/output schemas
- ✅ Document tools thoroughly
- ✅ Use unique resource URI schemes
- ❌ Avoid generic names like `get_data`
- ❌ Avoid hardcoded configurations

### 3. Version Your Compositions

Track composition versions separately from server versions:

```
Composition v2.0.0
├── github-server v1.5.0
├── jira-server v2.1.0
└── slack-server v1.0.0
```

### 4. Test Composed Servers

After creating a composition:

1. Download the composed package
2. Test that all tools work together
3. Verify no naming conflicts
4. Check resource accessibility

### 5. Document the Composition

Include a prompt that describes all available tools:

```json
{
  "name": "available_tools",
  "description": "List all available tools in this composition",
  "template": "This AI assistant has access to tools from multiple domains:\n\n## GitHub\n- get_user: Get GitHub profile\n- list_repos: List repositories\n\n## Jira\n- get_ticket: Get Jira ticket details\n- create_ticket: Create new ticket\n\n..."
}
```

---

## Exporting Compositions

### Download Composed Server

1. Open the composition (from the Compositions page; you must be logged in—compositions are per-user).
2. Click **"Export"** to generate and download the combined server as a ZIP.
3. Extract and run the generated Node.js project as with any single server.

The exported package includes:
- All tools from all servers
- Merged resources
- Merged prompts
- Combined README

### Generated Structure

```
enterprise-assistant/
├── src/
│   ├── server.ts
│   └── tools/
│       ├── hr_get_employee.ts
│       ├── hr_submit_pto.ts
│       ├── it_create_ticket.ts
│       ├── finance_submit_expense.ts
│       └── index.ts
├── package.json
├── tsconfig.json
├── Dockerfile
└── README.md
```

### MCP Client Configuration

```json
{
  "mcpServers": {
    "enterprise-assistant": {
      "command": "node",
      "args": ["/path/to/enterprise-assistant/dist/server.js"],
      "env": {
        "GITHUB_TOKEN": "xxx",
        "JIRA_API_KEY": "xxx",
        "SLACK_TOKEN": "xxx"
      }
    }
  }
}
```

---

## Composition API

### Create Composition

```bash
POST /api/compositions
Content-Type: application/json

{
  "name": "my-composition",
  "description": "Combined server",
  "version": "1.0.0",
  "server_ids": [
    "uuid-server-1",
    "uuid-server-2",
    "uuid-server-3"
  ]
}
```

### List Compositions

```bash
GET /api/compositions
```

### Get Composition Details

```bash
GET /api/compositions/{id}
```

### Export Composition

```bash
POST /api/compositions/{id}/export
# Returns ZIP file (generated combined server)
# Requires authentication; composition must be owned by current user
```

### Delete Composition

```bash
DELETE /api/compositions/{id}
```

---

## Troubleshooting

### Common Issues

| Issue | Cause | Solution |
|-------|-------|----------|
| Tool not found | Naming conflict resolved wrong | Check prefix settings |
| Resource not accessible | URI conflict | Use unique URI schemes |
| Auth errors | Missing env vars | Set all required env vars |
| Large package size | Many servers | Consider splitting compositions |

### Debugging

1. **Check individual servers first** - Ensure each server works independently
2. **Review tool names** - Look for conflicts in the Tools tab
3. **Test in playground** - Use the test playground before exporting
4. **Check logs** - Review server logs for detailed errors

---

## Next Steps

- [Creating Servers](./creating-servers.md) - Build individual servers
- [Getting Started Guide](./getting-started.md) - Platform overview
- [API Reference](./getting-started.md#api-reference) - REST API documentation
