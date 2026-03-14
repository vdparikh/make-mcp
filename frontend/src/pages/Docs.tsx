import { useState } from 'react';
import { Link } from 'react-router-dom';

interface DocItem {
  id: string;
  title: string;
  description: string;
  icon: string;
}

const docs: DocItem[] = [
  {
    id: 'getting-started',
    title: 'Getting Started',
    description: 'Quick start guide for the MCP Server Builder platform',
    icon: 'bi-rocket-takeoff',
  },
  {
    id: 'creating-servers',
    title: 'Creating Servers',
    description: 'Complete guide to building MCP servers with tools, resources, and prompts',
    icon: 'bi-server',
  },
  {
    id: 'compositions',
    title: 'Server Compositions',
    description: 'How to combine multiple MCP servers into one unified interface',
    icon: 'bi-layers',
  },
  {
    id: 'security-best-practices',
    title: 'Security Best Practices',
    description: 'MCP security practices, in-app security score (SlowMist checklist), and how Make MCP supports them',
    icon: 'bi-shield-lock',
  },
];

const docContents: Record<string, string> = {
  'getting-started': `# Getting Started

Welcome to the MCP Server Builder! This guide will help you get up and running quickly.

## What is MCP?

The **Model Context Protocol (MCP)** allows AI systems to interact with external tools and data sources through standardized servers. MCP servers expose:

- **Tools** - Functions that AI can call
- **Resources** - Data endpoints
- **Prompts** - Templated instructions

## Quick Start

### 1. Create a Server

1. Log in, then click **"New Server"** on the Dashboard
2. Enter name, description, and optional version
3. Click **"Create Server"** in the modal

### 2. Add Tools

Tools are the core functionality of your MCP server. Each tool:
- Has a name and description
- Defines input/output schemas
- Specifies how to execute: **rest_api**, **graphql**, **cli** (e.g. kubectl, docker), **flow** (visual pipeline), **database**, **javascript**, **python**, or **webhook**

**Example Tool:**
\`\`\`json
{
  "name": "get_weather",
  "description": "Get weather for a city",
  "input_schema": {
    "type": "object",
    "properties": {
      "city": { "type": "string" }
    },
    "required": ["city"]
  }
}
\`\`\`

### 3. Test Your Tools

Use the **Testing** tab to test tools before deployment:
1. Select a tool
2. Enter test input
3. Click **"Run Test"**
4. View the response

### 4. Deploy

Open **Deploy** in the server left nav. Options:
- **Node.js** — Generate & download ZIP (includes \`run-with-log.mjs\` for verifying tool calls)
- **Docker** — Use generated Dockerfile (non-root)
- **GitHub** — Push generated server to a repo
- **Deploy to Cloud** — Placeholder for future cloud deployment

## Next Steps

- Learn about [Creating Servers](#creating-servers) in detail
- Explore [Server Compositions](#compositions) to combine servers
- Check your **Security** tab in the server editor for the security score (SlowMist checklist)
- Read [Security Best Practices](#security-best-practices) and advanced features like Context Engine and Governance`,

  'creating-servers': `# Creating Servers

This guide covers everything you need to know about creating MCP servers.

## Server Configuration

Every server has:
- **Name** - Identifier for your server
- **Description** - What your server does
- **Version** - Semantic version (e.g., 1.0.0)

## Tools

Tools are functions that AI agents can call.

### Execution Types

| Type | Use Case |
|------|----------|
| rest_api | Call external HTTP APIs |
| graphql | Query GraphQL endpoints |
| cli | Run shell commands (kubectl, docker, terraform; use \`allowed_commands\`) |
| database | Execute SQL queries |
| javascript | Custom JS logic |
| python | Custom Python scripts |
| webhook | Send data to webhooks |
| flow | Visual pipeline: build in Visual Builder, then Convert to Tool |

### Input/Output Schemas

Use JSON Schema to define inputs and outputs:

\`\`\`json
{
  "type": "object",
  "properties": {
    "query": {
      "type": "string",
      "description": "Search query"
    },
    "limit": {
      "type": "integer",
      "default": 10
    }
  },
  "required": ["query"]
}
\`\`\`

**Pro Tip:** Use the **"Generate from Sample"** button to auto-generate schemas from example JSON!

### Authentication

Tools can use various auth methods:
- **API Key** - Header-based API keys
- **Bearer Token** - JWT or OAuth tokens
- **Basic Auth** - Username/password
- **OAuth 2.0** - Client credentials flow

## Resources

Resources provide structured data endpoints:

\`\`\`
Resource: company_docs
URI: mcp://docs/company
MIME Type: application/json
\`\`\`

## Prompts

Prompts are templated instructions:

\`\`\`
Name: summarize_document

Template:
"Summarize the following document:
{{document}}"
\`\`\`

## Context Engine

Inject user context automatically:
- User ID
- Organization ID
- Roles & Permissions

## Policies

Apply governance rules to tools:
- Require approval for sensitive actions
- Set maximum values
- Restrict to specific roles
- Limit to business hours
- Rate limiting

## Visual Flow Builder

Build tool pipelines visually:
1. Go to **Tools** tab
2. Click **"Open Visual Builder"**
3. Drag and drop nodes
4. Connect them to create flows
5. Save and convert to a tool`,

  'compositions': `# Server Compositions

Compositions allow you to combine multiple MCP servers into one unified interface.

## Why Compositions?

| Benefit | Description |
|---------|-------------|
| **Modular Architecture** | Each server handles one domain |
| **Reusability** | Share servers across compositions |
| **Team Ownership** | Different teams own different servers |

## Creating a Composition

1. Go to **Compositions** in the sidebar
2. Click **"New Composition"**
3. Enter a name and description
4. Select 2+ servers to combine
5. Click **"Create Composition"**

## Export Options

When exporting a composition:

- **Prefix Tool Names** - Add server name prefix to avoid conflicts
- **Merge Resources** - Include resources from all servers
- **Merge Prompts** - Include prompts from all servers

## Example: Sales Agent

Combine:
- \`stripe-mcp\` - Payment tools
- \`salesforce-mcp\` - CRM tools  
- \`slack-mcp\` - Notification tools

Result: A unified Sales Agent MCP server with all capabilities.

## Conflict Resolution

If multiple servers have tools with the same name:

1. **Enable "Prefix Tool Names"** - Auto-prefixes with server name
2. **Rename before composing** - Edit one server's tool name
3. **Exclude duplicate** - Choose which server's tool to use

## Best Practices

1. **Use unique tool names** - Avoid conflicts
2. **Document your servers** - Clear descriptions
3. **Version your compositions** - Track changes
4. **Test after composing** - Verify all tools work`,

  'security-best-practices': `# Security Best Practices

Make MCP aligns with common MCP security guidance (supply-chain, least privilege, human-in-the-loop, sandboxing). This page summarizes what the platform supports.

## Security Score (SlowMist Checklist)

Every server has a **security score** (0–100%, grade A–F) based on the [SlowMist MCP Security Checklist](https://github.com/slowmist/MCP-Security-Checklist).

- **Server Editor → Security** – View your score, grade, and a checklist of criteria. Unmet items show a short reason so you can improve the score.
- **Marketplace** – Published servers show the score on the card and in the inspector’s **Security** tab.

The score is computed from configuration we can evaluate (schemas, policies, hints, resources, versioning).

## What Make MCP Supports

| Area | Feature | Where |
|------|---------|-------|
| **Tool annotations** | Read-only / Destructive hints | Tool Editor → Schema → Security hints |
| **Policies** | Rate limits, roles, approval rules | Server → Policies |
| **Scoped auth** | Per-tool API keys, OAuth2 | Tool Editor → Config → Auth |
| **CLI allowlist** | Restrict which commands run | Tool Editor → Config → \`allowed_commands\` |
| **Versioning** | Pin to a version, avoid "latest" | Publish → Versions; generated README |
| **Non-root Docker** | Image runs as non-root user | Generated Dockerfile |
| **Logging** | File-based audit trail | \`MCP_LOG_FILE\`, \`run-with-log.mjs\` |

## Security Hints (Read-only / Destructive)

In the **Tool Editor**, open the **Schema** tab. Under **Security hints**:

- **Read-only** – Mark tools that only read data. Gateways can use this to block write operations.
- **Destructive** – Mark tools that modify or delete data. MCP clients can require user confirmation before running.

These are emitted in the generated server so clients and gateways can enforce policy.

## Full Guide

For a full mapping of MCP security practices to Make MCP (supply-chain, least privilege, human-in-the-loop, sandboxing, gateway, metrics), see **docs/security-best-practices.md** in the repository.`,
};

export default function Docs() {
  const [selectedDoc, setSelectedDoc] = useState<string | null>(null);

  const renderMarkdown = (content: string) => {
    // Process tables first
    const tableRegex = /\|(.+)\|\n\|[-|\s]+\|\n((?:\|.+\|\n?)+)/g;
    let processed = content.replace(tableRegex, (_match, headerRow, bodyRows) => {
      const headers = headerRow.split('|').filter((c: string) => c.trim());
      const headerHtml = '<tr>' + headers.map((h: string) => `<th>${h.trim()}</th>`).join('') + '</tr>';
      
      const rows = bodyRows.trim().split('\n').map((row: string) => {
        const cells = row.split('|').filter((c: string) => c.trim());
        return '<tr>' + cells.map((c: string) => `<td>${c.trim()}</td>`).join('') + '</tr>';
      }).join('');
      
      return `<table><thead>${headerHtml}</thead><tbody>${rows}</tbody></table>`;
    });

    // Simple markdown-to-HTML conversion
    let html = processed
      // Headers
      .replace(/^### (.*$)/gim, '<h3>$1</h3>')
      .replace(/^## (.*$)/gim, '<h2>$1</h2>')
      .replace(/^# (.*$)/gim, '<h1>$1</h1>')
      // Bold
      .replace(/\*\*(.*?)\*\*/g, '<strong>$1</strong>')
      // Italic
      .replace(/\*(.*?)\*/g, '<em>$1</em>')
      // Code blocks
      .replace(/```(\w+)?\n([\s\S]*?)```/g, '<pre><code>$2</code></pre>')
      // Inline code
      .replace(/`([^`]+)`/g, '<code>$1</code>')
      // Links
      .replace(/\[([^\]]+)\]\(([^)]+)\)/g, '<a href="$2">$1</a>')
      // Line breaks
      .replace(/\n\n/g, '</p><p>')
      .replace(/\n/g, '<br/>');
    
    return `<p>${html}</p>`;
  };

  if (selectedDoc) {
    return (
      <div>
        <div className="page-header">
          <div>
            <nav style={{ marginBottom: '0.5rem' }}>
              <Link to="/" style={{ color: 'var(--text-muted)', textDecoration: 'none', fontSize: '0.875rem' }}>
                Dashboard
              </Link>
              <span style={{ color: 'var(--text-muted)', margin: '0 0.5rem' }}>/</span>
              <button 
                onClick={() => setSelectedDoc(null)}
                style={{ color: 'var(--text-muted)', textDecoration: 'none', fontSize: '0.875rem', background: 'none', border: 'none', cursor: 'pointer' }}
              >
                Documentation
              </button>
              <span style={{ color: 'var(--text-muted)', margin: '0 0.5rem' }}>/</span>
              <span style={{ color: 'var(--text-primary)', fontSize: '0.875rem' }}>
                {docs.find(d => d.id === selectedDoc)?.title}
              </span>
            </nav>
            <h1 className="page-title">Documentation</h1>
          </div>
          <button className="btn btn-secondary" onClick={() => setSelectedDoc(null)}>
            <i className="bi bi-arrow-left"></i>
            Back to Docs
          </button>
        </div>

        <div className="card">
          <div 
            className="doc-content"
            style={{ 
              lineHeight: 1.7,
              color: 'var(--text-primary)',
            }}
            dangerouslySetInnerHTML={{ __html: renderMarkdown(docContents[selectedDoc] || '') }}
          />
        </div>

        <style>{`
          .doc-content h1 { font-size: 1.75rem; margin: 0 0 1rem 0; color: var(--text-primary); }
          .doc-content h2 { font-size: 1.375rem; margin: 1.5rem 0 0.75rem 0; color: var(--text-primary); border-bottom: 1px solid var(--card-border); padding-bottom: 0.5rem; }
          .doc-content h3 { font-size: 1.125rem; margin: 1.25rem 0 0.5rem 0; color: var(--text-primary); }
          .doc-content p { margin: 0.75rem 0; }
          .doc-content code { background: #1a1a2e; color: #e5e7eb; padding: 0.125rem 0.375rem; border-radius: 4px; font-size: 0.875rem; }
          .doc-content pre { background: #1a1a2e; padding: 1rem; border-radius: 8px; overflow-x: auto; margin: 1rem 0; }
          .doc-content pre code { background: none; padding: 0; color: #a5f3fc; }
          .doc-content table { width: 100%; border-collapse: collapse; margin: 1rem 0; border: 1px solid var(--card-border); border-radius: 8px; overflow: hidden; }
          .doc-content th { padding: 0.75rem 1rem; text-align: left; background: var(--hover-bg); font-weight: 600; color: var(--text-primary); border-bottom: 2px solid var(--card-border); }
          .doc-content td { padding: 0.75rem 1rem; border-bottom: 1px solid var(--card-border); color: var(--text-secondary); }
          .doc-content tbody tr:last-child td { border-bottom: none; }
          .doc-content tbody tr:hover { background: var(--hover-bg); }
          .doc-content a { color: var(--primary-color); text-decoration: none; }
          .doc-content a:hover { text-decoration: underline; }
          .doc-content strong { color: var(--text-primary); }
        `}</style>
      </div>
    );
  }

  return (
    <div>
      <div className="page-header">
        <div>
          <nav style={{ marginBottom: '0.5rem' }}>
            <Link to="/" style={{ color: 'var(--text-muted)', textDecoration: 'none', fontSize: '0.875rem' }}>
              Dashboard
            </Link>
            <span style={{ color: 'var(--text-muted)', margin: '0 0.5rem' }}>/</span>
            <span style={{ color: 'var(--text-primary)', fontSize: '0.875rem' }}>Documentation</span>
          </nav>
          <h1 className="page-title">Documentation</h1>
          <p className="page-subtitle">Learn how to build and deploy MCP servers</p>
        </div>
        <Link to="/" className="btn btn-secondary">
          <i className="bi bi-arrow-left"></i>
          Back
        </Link>
      </div>

      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(300px, 1fr))', gap: '1rem' }}>
        {docs.map((doc) => (
          <div 
            key={doc.id} 
            className="card" 
            style={{ cursor: 'pointer', transition: 'all 0.2s' }}
            onClick={() => setSelectedDoc(doc.id)}
            onMouseOver={(e) => {
              e.currentTarget.style.borderColor = 'var(--primary-color)';
              e.currentTarget.style.transform = 'translateY(-2px)';
            }}
            onMouseOut={(e) => {
              e.currentTarget.style.borderColor = 'var(--card-border)';
              e.currentTarget.style.transform = 'translateY(0)';
            }}
          >
            <div style={{ fontSize: '2.5rem', marginBottom: '1rem', color: 'var(--primary-color)' }}>
              <i className={`bi ${doc.icon}`}></i>
            </div>
            <h3 style={{ marginBottom: '0.5rem', color: 'var(--text-primary)' }}>{doc.title}</h3>
            <p style={{ color: 'var(--text-secondary)', fontSize: '0.875rem', margin: 0 }}>
              {doc.description}
            </p>
          </div>
        ))}
      </div>

      <div className="card" style={{ marginTop: '2rem' }}>
        <h3 className="card-title" style={{ marginBottom: '1rem' }}>
          <i className="bi bi-link-45deg" style={{ marginRight: '0.5rem' }}></i>
          External Resources
        </h3>
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(250px, 1fr))', gap: '1rem' }}>
          <a 
            href="https://modelcontextprotocol.io" 
            target="_blank" 
            rel="noopener noreferrer"
            style={{ 
              display: 'flex', 
              alignItems: 'center', 
              gap: '0.75rem',
              padding: '1rem',
              background: 'var(--hover-bg)',
              borderRadius: '8px',
              textDecoration: 'none',
              color: 'var(--text-primary)',
              transition: 'all 0.2s'
            }}
          >
            <i className="bi bi-book" style={{ fontSize: '1.5rem', color: 'var(--primary-color)' }}></i>
            <div>
              <div style={{ fontWeight: 500 }}>MCP Documentation</div>
              <div style={{ fontSize: '0.75rem', color: 'var(--text-muted)' }}>Official protocol docs</div>
            </div>
            <i className="bi bi-box-arrow-up-right" style={{ marginLeft: 'auto', color: 'var(--text-muted)' }}></i>
          </a>

          <a 
            href="https://github.com/modelcontextprotocol" 
            target="_blank" 
            rel="noopener noreferrer"
            style={{ 
              display: 'flex', 
              alignItems: 'center', 
              gap: '0.75rem',
              padding: '1rem',
              background: 'var(--hover-bg)',
              borderRadius: '8px',
              textDecoration: 'none',
              color: 'var(--text-primary)',
              transition: 'all 0.2s'
            }}
          >
            <i className="bi bi-github" style={{ fontSize: '1.5rem' }}></i>
            <div>
              <div style={{ fontWeight: 500 }}>MCP GitHub</div>
              <div style={{ fontSize: '0.75rem', color: 'var(--text-muted)' }}>SDKs and examples</div>
            </div>
            <i className="bi bi-box-arrow-up-right" style={{ marginLeft: 'auto', color: 'var(--text-muted)' }}></i>
          </a>
        </div>
      </div>
    </div>
  );
}
