# Local OIDC testing with Keycloak

This complements [Hosted MCP security](./hosted-security.md). It walks through **`hosted_security_config`** and a **Keycloak** container added via Docker Compose (`--profile oidc`).

## What `hosted_security_config` is

It is **JSON** stored on the server row (not secret by itself; secrets are IdP client secrets and hosted keys). It can include:

| Field | Purpose |
|-------|---------|
| `default_env` | Default when `X-Make-MCP-Env` is missing. |
| `oidc` | **Root** defaults: `issuer`, optional `audience`, optional `jwks_url` (otherwise discovered from issuer). |
| `mtls` | Trusted client cert SHA-256s (see main doc). |
| `env_profiles` | Per-environment overrides (IP allowlist, `hosted_auth_mode`, nested `oidc`, etc.). |

You set **`hosted_auth_mode`** to **`oidc`** (Deploy UI, publish payload, or `PUT /api/servers/:id/hosted-security`) **and** point **`oidc.issuer`** at your IdP’s realm issuer URL.

## Minimal example (OIDC only)

```json
{
  "default_env": "dev",
  "oidc": {
    "issuer": "http://keycloak:8080/realms/mcp-dev",
    "audience": ""
  }
}
```

- Leave **`audience` empty** (or omit) for first tests — Keycloak client-credentials tokens may not set `aud` the way you expect; the API only enforces audience when non-empty.
- **`issuer` must exactly match** the JWT `iss` claim (see below).

## Issuer, JWKS, and Docker networking

The API loads **`{issuer}/.well-known/openid-configuration`** and then the IdP JWKS. So:

- If the **Make MCP API runs inside Docker** (default `docker-compose`), it must reach Keycloak at a hostname on the Compose network, e.g. **`http://keycloak:8080/realms/mcp-dev`**.
- JWTs issued with **`iss: http://localhost:8180/realms/...`** will **not** validate if you configured **`issuer`** as `http://keycloak:8080/realms/...` — the strings must match.

**Practical fix for local dev**

1. In Keycloak **Realm → Settings**, set **Frontend URL** to **`http://keycloak:8080`** so access tokens use that issuer even when you hit the admin UI via `localhost:8180`.
2. Use **`oidc.issuer`: `http://keycloak:8080/realms/mcp-dev`** in Make MCP when the API runs in Docker.
3. Obtain tokens by calling the token endpoint **from a container on the same network** (see `docker/keycloak/README.md`), or any method that yields a JWT whose `iss` is exactly that URL.

If you run **`go run ./cmd/server` on the host** (not in Docker), use **`http://localhost:8180/realms/mcp-dev`** as issuer and ensure discovery is reachable from the host.

## Where to paste the JSON

1. **UI**: **Deploy** → Hosted → **Advanced security profile** textarea (MCP server target), then **Publish**.
2. **API**: `PUT /api/servers/:id/hosted-security` with `hosted_auth_mode: "oidc"` and `hosted_security_config: { ... }`.

## Call the hosted MCP endpoint

OIDC mode requires:

`Authorization: Bearer <access_token_or_id_token_jwt>`

The static hosted key headers (`X-Make-MCP-Key`) do **not** apply in OIDC mode — only `Authorization: Bearer` (see [hosted-security.md](./hosted-security.md)).

## Start Keycloak

From the repo root:

```bash
docker compose --profile oidc up -d keycloak
```

Admin console: **http://localhost:8180** (admin / admin in dev).

Then follow **`docker/keycloak/README.md`** to create realm **`mcp-dev`** and client **`mcp-hosted`**.

## OAuth BFF (MCP Jam, Cursor, `/.well-known`)

For **browser-based OAuth** where Keycloak only trusts Make MCP’s callback (not each laptop), add **`oauth_bff`** next to **`oidc`** and register **`http://<api>:8080/api/oauth/callback`** (or your **`urls.mcp_hosted_base_url` + `/oauth/callback`**) in Keycloak. Full flow, endpoints, and **`mcp.json`** are in **[Hosted OAuth BFF](./hosted-oauth-bff.md)**.

## Related

- [Hosted MCP security](./hosted-security.md) — full schema and modes  
- [Hosted OAuth BFF](./hosted-oauth-bff.md) — BFF endpoints, `mcp.json`, troubleshooting  
- [CONFIGURATION.md](../CONFIGURATION.md) — app config and env vars  
