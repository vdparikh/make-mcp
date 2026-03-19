# MCP Security Best Practices in Make MCP

This document maps **MCP security best practices** (e.g. from vendor guides and OWASP) to what Make MCP supports and what you should do when building and deploying MCP servers.

## Security score (SlowMist checklist)

Make MCP computes a **security score** (0–100%, grade A–F) for each server based on the [SlowMist MCP Security Checklist](https://github.com/slowmist/MCP-Security-Checklist). Use it to see how your server measures up and what to improve.

- **While building:** Open your server → **Security** in the left nav. You’ll see the current score, grade, and a list of criteria (e.g. input validation, rate limiting, access control, CLI allowlist, tool hints). Address unmet items to raise the score.
- **In the marketplace:** Published servers show their security score and grade on the card and in the inspector. The **Security** tab in the inspector lists which checklist items the server satisfies.

The score is derived only from configuration we can see (schemas, policies, hints, resources, versioning). Runtime and deployment choices (e.g. running in a container, using TLS) are not measured.

---

## Runtime Security Model (Hosted MCP)

`Context + Policies` decides *what should happen*.  
`Runtime security model` defines *how that decision is enforced safely at execution time*.

For Make MCP hosted deployments, the runtime model is:

1. **Authenticate caller at hosted boundary** (`bearer_token` or `no_auth`)
2. **Optionally require caller identity** (`require_caller_identity` with `X-Make-MCP-Caller-Id`)
3. **Enforce policy and execution limits** before/while tool code runs
4. **Run in isolated, constrained container runtime**
5. **Record auditable observability events** with caller attribution when present

### Trust boundaries

| Boundary | Risk | Runtime control in Make MCP |
|----------|------|-----------------------------|
| Client -> Hosted endpoint | Unauthenticated or anonymous calls | `hosted_auth_mode` + optional `require_caller_identity` |
| Hosted proxy -> Tool execution | Over-broad tool execution / unsafe calls | Policies, tool schema validation, CLI allowlist, timeouts |
| Container -> Host/network | Breakout, lateral movement, abuse | Non-root runtime, resource limits, restricted bind scope |
| Runtime -> Observability | Missing attribution / weak audit trail | Caller and tenant headers propagated into runtime observability events |

### Controls available today (and how to use them)

| Control | Where to configure | Why it matters |
|---------|--------------------|----------------|
| **Endpoint auth mode** (`bearer_token` / `no_auth`) | Deploy -> Hosted -> Publish MCP -> Access & Security | Prevents unauthorized use of hosted URL (when Bearer is used) |
| **Require caller identity** toggle | Deploy -> Hosted -> Publish MCP -> Access & Security | Enables per-user attribution; enforces `X-Make-MCP-Caller-Id` |
| **Idle shutdown** | Deploy -> Hosted -> Publish MCP -> Access & Security | Reduces attack window and cost for inactive workloads |
| **Policy rules** (rate, roles, approval, time-window) | Server Editor -> Policies | Runtime guardrails for dangerous/high-cost actions |
| **Destructive / read-only hints** | Tool Editor -> Schema | Lets clients/gateways enforce confirmation and safer execution paths |
| **CLI command allowlist** | Tool Editor -> Config (CLI tools) | Reduces command-injection blast radius |
| **Observability reporting** | Server -> Observability | Auditable runtime history (caller, tool, status, latency) |

### How to apply this model in Make MCP (recommended flow)

1. **Start with endpoint protection**
   - Set `hosted_auth_mode` to `bearer_token` unless you intentionally need open access.
2. **Turn on caller attribution**
   - Enable **Require caller identity** to force `X-Make-MCP-Caller-Id`.
   - Optionally pass `X-Make-MCP-Tenant-Id` for multi-tenant reporting.
3. **Constrain tool behavior**
   - Add policy rules for rate limits, role checks, approval gates, and business-hour windows.
   - For CLI tools, configure `allowed_commands`.
4. **Mark tool risk explicitly**
   - Set read-only/destructive hints in tool schema.
5. **Validate with observability**
   - Confirm events show caller identity, tenant (if provided), and policy outcomes.
6. **Operate through hosted sessions**
   - Use hosted session controls (health/restart/stop) for runtime hygiene and incident response.

### Minimum production baseline

Use this baseline for most hosted deployments:

- `hosted_auth_mode = bearer_token`
- `require_caller_identity = true`
- At least one policy per destructive tool
- `allowed_commands` for every CLI tool
- Observability enabled and reviewed regularly

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
