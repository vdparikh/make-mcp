# Documentation index

In-app: open **Documentation** in the UI (`/docs`) or deep-link to a page with `?doc=<id>` (e.g. `/docs?doc=hosted-security`).

| Doc ID | File | Topics |
|--------|------|--------|
| `getting-started` | [getting-started.md](./getting-started.md) | Quick start |
| `creating-servers` | [creating-servers.md](./creating-servers.md) | Tools, resources, prompts, context |
| `compositions` | [compositions.md](./compositions.md) | Combining MCP servers |
| `security-best-practices` | [security-best-practices.md](./security-best-practices.md) | Policies, SlowMist checklist, logging |
| `hosted-security` | [hosted-security.md](./hosted-security.md) | Hosted auth (Bearer, OIDC, mTLS), IP allowlists, caller identity, rotation, audit |
| `hosted-runtime-isolation` | [hosted-runtime-isolation.md](./hosted-runtime-isolation.md) | Docker CPU/memory tiers, optional deny-default HTTP egress for tools |
| `keycloak-local-oidc` | [keycloak-local-oidc.md](./keycloak-local-oidc.md) | Local Keycloak in Docker Compose, `hosted_security_config`, issuer notes |
| `hosted-oauth-bff` | [hosted-oauth-bff.md](./hosted-oauth-bff.md) | BFF flow, per-server issuer, token endpoint, Cursor `mcp.json`, troubleshooting |

See also [../CONFIGURATION.md](../CONFIGURATION.md) for `config.yaml` and environment variables.
