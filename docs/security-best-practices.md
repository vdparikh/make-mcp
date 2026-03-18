# MCP Security Best Practices in Make MCP

This document maps **MCP security best practices** (e.g. from vendor guides and OWASP) to what Make MCP supports and what you should do when building and deploying MCP servers.

## Security score (SlowMist checklist)

Make MCP computes a **security score** (0–100%, grade A–F) for each server based on the [SlowMist MCP Security Checklist](https://github.com/slowmist/MCP-Security-Checklist). Use it to see how your server measures up and what to improve.

- **While building:** Open your server → **Security** in the left nav. You’ll see the current score, grade, and a list of criteria (e.g. input validation, rate limiting, access control, CLI allowlist, tool hints). Address unmet items to raise the score.
- **In the marketplace:** Published servers show their security score and grade on the card and in the inspector. The **Security** tab in the inspector lists which checklist items the server satisfies.

The score is derived only from configuration we can see (schemas, policies, hints, resources, versioning). Runtime and deployment choices (e.g. running in a container, using TLS) are not measured.

---

## 1. Server and supply-chain security

| Practice | In Make MCP |
|----------|-------------|
| **Source servers responsibly** | Use the **Marketplace** for published servers. Prefer servers from trusted authors. When importing OpenAPI or code, review the generated tools before publishing. |
| **Internal allowlist** | Control which servers agents can use via your MCP client or gateway policy. Make MCP does not enforce allowlists; configure them in your client/gateway. |
| **Pin and verify versions** | Use **server versioning**: publish with a specific version (e.g. `1.0.0`). When deploying, pin to that version instead of "latest." In generated README we recommend pinning the server image/artifact by digest where possible. |

---

## 2. Least privilege for context and tooling

| Practice | In Make MCP |
|----------|-------------|
| **Scope credentials tightly** | In the **Tool Editor**, use the auth configuration to attach only the tokens/headers needed for that tool. Prefer short-lived tokens and avoid "god-mode" keys. |
| **Tool annotations in policy** | Use **Security hints** in the Tool Editor (Schema tab): **Read-only** and **Destructive**. These are emitted as `readOnlyHint` / `destructiveHint` in the generated server so gateways and clients can enforce policy (e.g. block writes, require confirmation for destructive tools). |
| **Policies** | Attach **Policies** to tools: rate limits, allowed roles, time windows, approval rules. The platform evaluates these when testing tools and can be mirrored in your gateway. |
| **Partition human vs. agent roles** | Use **Context** and **Policies** to scope by role. Give MCP agents dedicated service accounts; do not reuse human admin credentials. |

---

## 3. Human-in-the-loop and output validation

| Practice | In Make MCP |
|----------|-------------|
| **User confirmation for write/execute** | Mark tools that modify or delete data as **Destructive** in the Tool Editor. MCP clients can use `destructiveHint` to show a confirmation dialog before running. |
| **Sanitize untrusted content** | Validate and sanitize tool **inputs** in your backend or gateway. For CLI tools, use the **allowed_commands** list in execution config so only approved commands can run. |
| **Default to manual review** | Use **Policies** with "approval required" rules for sensitive tools. New or untrusted servers can be deployed with manual review until trust is established. |

---

## 4. Isolate and sandbox MCP environments

| Practice | In Make MCP |
|----------|-------------|
| **Non-root containers** | Generated **Dockerfile** runs as non-root user (`USER mcp`). Use it when building images for your servers. |
| **Egress allowlist** | Not enforced by Make MCP. Configure your orchestration (e.g. Kubernetes network policies) so MCP pods can only reach allowlisted APIs. |
| **Resource quotas** | Not enforced by Make MCP. Set CPU/memory limits in Kubernetes or your container runtime to limit abuse. |

---

## 5. Centralize control (gateway / proxy)

| Practice | In Make MCP |
|----------|-------------|
| **mTLS / audit logging** | Make MCP does not provide a gateway. Use your own MCP gateway or proxy to enforce mTLS and full-fidelity audit logs (prompt, tool call, args, latency). |
| **Inline guardrails** | Implement DLP, rate limiting, and prompt-injection detection in your gateway. Tool **security hints** (read-only, destructive) can inform gateway policy. |

---

## 6. Evaluate and select secure MCP servers

| Practice | In Make MCP |
|----------|-------------|
| **Secure development** | When publishing to the **Marketplace**, use clear versioning and release notes. Prefer servers that document their threat model and dependencies. |
| **Granular access** | Use **Policies** and **Security hints** so that deployed servers expose minimal privilege. Prefer short-lived, scoped credentials in tool auth config. |
| **Logging** | Generated servers support **MCP_LOG_FILE** for file-based logging. Use it (e.g. via `run-with-log.mjs`) and stream logs to your SIEM. |

---

## 7. Measure success

Track over time:

- **Incident frequency** for MCP-related issues.
- **Visibility**: % of MCP traffic logged with user ID and tool ID.
- **Supply-chain**: Prefer pinned versions and verified artifacts.
- **Human-in-the-loop**: Use of confirmation flows for destructive tools.

Make MCP gives you **versioning**, **policies**, **security hints**, and **logging hooks**; combine them with your gateway and deployment practices to align with these outcomes.

---

## Quick reference: Security features in the UI

| Feature | Where | Purpose |
|---------|--------|---------|
| **Security hints** | Tool Editor → Schema tab | Mark tool as **Read-only** or **Destructive**; emitted in generated server for gateways/clients. |
| **Policies** | Server → Policies | Rate limits, roles, time windows, approval rules per tool. |
| **Auth config** | Tool Editor → Config | Scoped API keys, bearer tokens, OAuth2 per tool. |
| **CLI allowlist** | Tool Editor → Config (CLI) | `allowed_commands` in execution config restricts which commands can run. |
| **Versioning** | Publish / Marketplace | Immutable versions; pin to a version when deploying. |
| **Non-root Docker** | Generated Dockerfile | Image runs as non-root user. |

For more detail on MCP security in the wild, see vendor guides (e.g. Wiz, Anthropic) and OWASP resources (e.g. LLM01 prompt injection).
