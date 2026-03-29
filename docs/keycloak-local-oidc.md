# Local OIDC testing with Keycloak

This complements [Hosted MCP security](./hosted-security.md). It walks through **`hosted_security_config`** and a **Keycloak** container run with **`docker`** (see **`docker/keycloak/README.md`**).

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

## Issuer, JWKS, and networking

The API loads **`{issuer}/.well-known/openid-configuration`** and then the IdP JWKS. The **`issuer`** you configure must **exactly match** the JWT `iss` claim.

| Where the Make MCP API runs | Typical `oidc.issuer` |
|-----------------------------|------------------------|
| `go run` on the host | `http://localhost:8180/realms/mcp-dev` (if Keycloak maps **8180→8080**) |
| In a **Kubernetes** pod (e.g. Skaffold on Docker Desktop) | Base URL **reachable from the pod**, often **`http://host.docker.internal:8180/realms/mcp-dev`**; set Keycloak **Frontend URL** to the **same** origin the API will use for discovery |
| Ephemeral `curl` / tooling on the same Docker **network** as Keycloak | `http://keycloak:8080/realms/mcp-dev` when the container is named **`keycloak`** and listens on **8080** inside the network |

JWTs issued with **`iss: http://localhost:8180/realms/...`** will **not** validate if you configured **`issuer`** as `http://keycloak:8080/realms/...` — the strings must match.

**Practical fix:** In Keycloak **Realm → Settings**, set **Frontend URL** to the **same origin** you use in **`oidc.issuer`** (the one the API process can reach). Then obtain test tokens with an **`iss`** that matches (see **`docker/keycloak/README.md`**).

## Where to paste the JSON

1. **UI**: **Deploy** → Hosted → **Advanced security profile** textarea (MCP server target), then **Publish**.
2. **API**: `PUT /api/servers/:id/hosted-security` with `hosted_auth_mode: "oidc"` and `hosted_security_config: { ... }`.

## Call the hosted MCP endpoint

OIDC mode requires:

`Authorization: Bearer <access_token_or_id_token_jwt>`

The static hosted key headers (`X-Make-MCP-Key`) do **not** apply in OIDC mode — only `Authorization: Bearer` (see [hosted-security.md](./hosted-security.md)).

## Start Keycloak

Use a user-defined Docker network and publish the admin port (details and token examples are in **`docker/keycloak/README.md`**):

```bash
docker network create mcp-keycloak-dev 2>/dev/null || true
docker run -d --name keycloak --network mcp-keycloak-dev -p 8180:8080 \
  -e KEYCLOAK_ADMIN=admin -e KEYCLOAK_ADMIN_PASSWORD=admin \
  quay.io/keycloak/keycloak:latest start-dev
```

Admin console: **http://localhost:8180** (admin / admin in dev). If the container already exists, remove it or pick another name.

Then follow **`docker/keycloak/README.md`** to create realm **`mcp-dev`** and client **`mcp-hosted`**.

## OAuth BFF (MCP Jam, Cursor, `/.well-known`)

For **browser-based OAuth** where Keycloak only trusts Make MCP’s callback (not each laptop), add **`oauth_bff`** next to **`oidc`** and register **`http://<api>:8080/api/oauth/callback`** (or your **`urls.mcp_hosted_base_url` + `/oauth/callback`**) in Keycloak. Full flow, endpoints, and **`mcp.json`** are in **[Hosted OAuth BFF](./hosted-oauth-bff.md)**.

## Related

- [Hosted MCP security](./hosted-security.md) — full schema and modes  
- [Hosted OAuth BFF](./hosted-oauth-bff.md) — BFF endpoints, `mcp.json`, troubleshooting  
- [CONFIGURATION.md](../CONFIGURATION.md) — app config and env vars  
