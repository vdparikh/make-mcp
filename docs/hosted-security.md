# Hosted MCP security

This guide describes **identity, authentication, and trust boundaries** for MCP servers **hosted** on the platform (HTTP/SSE endpoint behind the API reverse proxy).

## Concepts

| Layer | Purpose |
|--------|---------|
| **Transport** | TLS terminates at your ingress; optional **mTLS** is enforced by matching a **client certificate SHA-256** forwarded to the API. |
| **Hosted access** | Who may call the hosted URL: `no_auth`, static **bearer/API key**, **OIDC JWT**, or **mTLS-only**. |
| **Caller identity** | Optional second factor: `X-Make-MCP-Caller-Id` with a **caller API key** (`mkc_...`) so every tool run is attributed to a **tenant / user / scope** (see [Caller API Keys](./security-best-practices.md)). |
| **Environment (two meanings)** | **Deploy target env** = which **Server → Environments** profile is baked into publish. **`X-Make-MCP-Env`** + **`hosted_security_config.env_profiles`** = per-request **security** policy; see [Two kinds of “environment”](#two-kinds-of-environment). |

## Configuration locations

1. **Server row** — `hosted_auth_mode`, `require_caller_identity`, `hosted_access_key` (secret, rotation via API).
2. **`hosted_security_config` (JSON)** — Per-environment **IP allowlists**, **OIDC** issuer/audience/JWKS, optional **OAuth BFF** (`oauth_bff`) for browser login via Make MCP (see [Hosted OAuth BFF](./hosted-oauth-bff.md)), **mTLS** trusted cert fingerprints, optional **mode overrides**.

Set via:

- **Deploy** → Hosted publish request (`hosted_security_config` in JSON body), or  
- **`PUT /api/servers/:id/hosted-security`** (authenticated owner).

### How to set `hosted_security_config` (step by step)

1. **Pick `hosted_auth_mode`** on the server (e.g. `oidc`) — same value can be mirrored inside `env_profiles` for per-environment overrides.
2. **Build JSON** using the schema below. At minimum for OIDC you need **`oidc.issuer`** (your realm issuer URL). Optional: **`oidc.audience`**, **`oidc.jwks_url`** (if discovery is blocked; otherwise leave empty and JWKS is discovered).
3. **Paste** into **Deploy → Hosted → Advanced security profile** (or send in the publish / `PUT` body as `hosted_security_config`).
4. **Publish** (or `PUT`) so the server row is updated.
5. **Match runtime network**: `{issuer}/.well-known/...` must resolve from the **process running the Make MCP API** (host, Kubernetes pod, or a container network). The JWT **`iss`** claim must **exactly equal** `oidc.issuer`. See **[Keycloak local OIDC](./keycloak-local-oidc.md)** for a standalone Keycloak container and issuer pitfalls.

## Headers (client → API)

| Header | When |
|--------|------|
| `Authorization: Bearer <token>` | **Bearer / key mode:** static **hosted access key** (rotate from Deploy when that mode is selected). **OIDC mode:** IdP **JWT** (not the hosted key). |
| `X-Make-MCP-Key` | Same secret as the hosted access key; alternative to `Authorization` for clients that prefer a custom header. |
| `X-MCP-API-Key` | Alias of `X-Make-MCP-Key` (same value). |
| `X-Make-MCP-Env` | Selects a row in **`hosted_security_config.env_profiles`** (falls back to **`default_env`** when omitted). See [Two kinds of “environment”](#two-kinds-of-environment). |
| `X-Make-MCP-Client-Cert-SHA256` | Hex SHA-256 of client cert (64 hex chars); set by ingress from mTLS handshake. |
| `X-Make-MCP-Caller-Id` | **Caller API key** (`mkc_…`) when **Require caller identity** is on — **attribution** and tenancy, not the same as the hosted access key. Create keys under **Hosted → Caller keys**. |
| `X-Make-MCP-Tenant-Id` | Optional tenant hint (validated with caller key). |
| `X-Make-MCP-Caller-Alias` | Optional alias when caller key allows it. |

## Two kinds of “environment”

| Concept | Where it is set | What it does |
|--------|------------------|--------------|
| **Deploy “Target environment”** (Dev / Staging / Prod) | **Deploy** flow UI | Picks which **server environment profile** to use — the **base URLs** and DB URLs from **Server → Environments** in the editor. Used when **generating**, **downloading**, and **publishing** the hosted container so tools call the right backends. |
| **`hosted_security_config.default_env` + `env_profiles`** | **Advanced security profile** JSON (or API) | Chooses **security policy** (IP allowlist, OIDC overrides, optional auth mode overrides) per **incoming request**, keyed by the **`X-Make-MCP-Env`** header (e.g. `dev`, `staging`, `prod`). If the header is missing, **`default_env`** applies. |

They often use the same names (dev/staging/prod) but serve **different** purposes: one configures **what the MCP server calls**, the other configures **how the API authorizes the caller**. They are **not** overridden from `mcp.json` — clients may send `X-Make-MCP-Env` if you use security `env_profiles`; the Deploy target env is fixed at **publish** time.

Forwarded to the **runtime container** (for tools / observability):

- `X-Make-MCP-Caller-Id`, `X-Make-MCP-Tenant-Id`, `X-Make-MCP-Scopes`  
- `X-Make-MCP-OIDC-Email` (OIDC path)  
- `X-Make-MCP-Auth-Method` (`oidc`, etc.)

## `hosted_security_config` schema (reference)

```json
{
  "default_env": "prod",
  "oidc": {
    "issuer": "https://your-idp.example.com",
    "audience": "your-api-audience",
    "jwks_url": ""
  },
  "mtls": {
    "trusted_client_cert_sha256": ["abcdef0123...64 hex..."]
  },
  "env_profiles": {
    "dev": {
      "ip_allowlist": ["127.0.0.1/32", "::1/128"],
      "hosted_auth_mode": "no_auth",
      "require_caller_identity": false,
      "require_mtls": false
    },
    "prod": {
      "ip_allowlist": ["203.0.113.0/24"],
      "hosted_auth_mode": "bearer_token",
      "require_caller_identity": true,
      "require_mtls": true,
      "oidc": {
        "issuer": "https://login.example.com",
        "audience": "mcp-hosted"
      },
      "mtls": {
        "trusted_client_cert_sha256": []
      }
    }
  }
}
```

- **`oidc` / `mtls` at root** apply when an env profile does not override them.
- **`require_mtls`**: after bearer or OIDC succeeds, the client must also present a **trusted** cert fingerprint.
- **IP allowlist**: evaluated using **`X-Forwarded-For`** first hop (configure trusted proxies).

### Hosted auth modes

| `hosted_auth_mode` | Behavior |
|--------------------|----------|
| `no_auth` | No bearer secret; optional IP / mTLS / caller layers still apply. |
| `bearer_token` | Compare `Authorization` / `X-Make-MCP-Key` to **rotated** hosted access key. |
| `oidc` | Validate **JWT** with OIDC discovery and **JWKS** (RS256 / ES256 P-256); `sub` becomes the verified subject. Set **`oidc.issuer`**; leave **`audience`** empty unless your IdP sets a matching **`aud`**. For browser OAuth via Make MCP, add **`oauth_bff`** — see [Hosted OAuth BFF](./hosted-oauth-bff.md). |
| `mtls` | Only **client certificate** fingerprints (no bearer); use with ingress mTLS. |

## Key rotation and audit

| Action | API |
|--------|-----|
| Rotate static hosted key | `POST /api/servers/:id/hosted-security/rotate-access-key` |
| Read settings (no secret) | `GET /api/servers/:id/hosted-security` |
| Update modes + JSON | `PUT /api/servers/:id/hosted-security` |
| Audit log (JSON) | `GET /api/servers/:id/hosted-security/audit?limit=100` |
| Audit export (CSV) | `GET /api/servers/:id/hosted-security/audit/export?limit=500` |

Events include **config saves** and **hosted access key rotation** with **actor user id** and timestamp metadata.

**Short-lived tokens:** For enterprise SSO, prefer **OIDC access tokens** from your IdP (standard exp/iss/aud). Static hosted keys are long-lived; rotate them on schedule or after personnel changes.

## Operational notes

- **mTLS** at the app layer trusts **only** the fingerprint header; terminate TLS and validate client certs at **Envoy/nginx** and forward the fingerprint.
- **OIDC** validation uses **RS256** and **ES256** (realm JWKS) with a short-lived cache.
- **Secrets** never appear in audit metadata; store keys in a vault and rotate using the API above.

## Related

- [Hosted OAuth BFF](./hosted-oauth-bff.md) — browser OAuth via Make MCP (`/.well-known/...`, `/api/oauth/*`)  
- [Keycloak local OIDC](./keycloak-local-oidc.md) — local Keycloak (`docker run`) for testing  
- [Security best practices](./security-best-practices.md)  
- [Configuration](../CONFIGURATION.md)  
- [Getting started](./getting-started.md) (Try Chat, observability)
