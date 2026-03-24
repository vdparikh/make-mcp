# Hosted MCP OAuth BFF

Make MCP acts as an **OAuth BFF** in front of your IdP (e.g. Keycloak): MCP clients discover **`/.well-known/*`** on the API host, users sign in at the IdP, the **authorization code** returns to Make MCP, and the server exchanges it using the **client secret** (never sent to the client).

**Hosted MCP access** still uses **`hosted_auth_mode: oidc`**: the API validates **`Authorization: Bearer <JWT>`** on `GET/POST /api/users/...` (see [Hosted MCP security](./hosted-security.md)).

## End-to-end flow

1. Client loads **protected resource metadata (PRM)** → gets **`authorization_servers`** = per-server issuer `http://<api>/api/oauth/bff/<server_id>`.
2. Client loads **authorization server metadata** from **`/api/oauth/bff/<server_id>/.well-known/openid-configuration`** (includes `jwks_uri` from the upstream IdP for MCP client validation).
3. Browser **`/authorize`** → IdP login → **`/api/oauth/callback`** → Make MCP exchanges the code → redirects to the MCP app with **`?code=<one-time>&state=`**.
4. Client **`POST /api/oauth/token?server_id=...`** with **`grant_type=authorization_code`** (form or JSON) → receives **access_token** / **refresh_token**.
5. MCP calls use **`Authorization: Bearer <access_token>`**; JWT **`iss`** must match **`hosted_security_config.oidc.issuer`**.

## Requirements

| Item | Notes |
|------|--------|
| `hosted_auth_mode` | **`oidc`** |
| `hosted_security_config.oidc.issuer` | Same string as JWT **`iss`** (see [Keycloak local OIDC](./keycloak-local-oidc.md) for Docker vs hostnames). |
| `hosted_security_config.oauth_bff` | `enabled`, `upstream_issuer`, `client_id`, **`client_secret_env`** (env name for IdP client secret). |
| IdP redirect URI | **`{mcp_hosted_base}/oauth/callback`** e.g. `http://127.0.0.1:8080/api/oauth/callback` — **not** the desktop client port (`6274`). |
| `JWT_SECRET` | Signs internal OAuth **`state`** between authorize and callback. |
| `cors.allowed_origins` | Include **`http://127.0.0.1:6274`** and **`http://localhost:6274`** if you use **MCP Jam’s** in-browser OAuth debugger. |

### Example `hosted_security_config`

```json
{
  "default_env": "dev",
  "oidc": {
    "issuer": "http://localhost:8180/realms/mcp-dev",
    "audience": ""
  },
  "oauth_bff": {
    "enabled": true,
    "upstream_issuer": "http://localhost:8180/realms/mcp-dev",
    "client_id": "make-mcp-bff",
    "client_secret_env": "KEYCLOAK_OAUTH_CLIENT_SECRET"
  }
}
```

Leave **`oidc.audience` empty** unless you use an audience mapper at the IdP. Prefer **`urls.mcp_hosted_base_url`** in `config.yaml` (e.g. `http://127.0.0.1:8080/api`) so the Keycloak **Valid redirect URI** stays stable.

## HTTP endpoints (API host)

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/.well-known/oauth-protected-resource?server_id=` or `?resource=` | PRM; **`authorization_servers`** = `.../api/oauth/bff/<server_id>`. |
| GET | `/.well-known/oauth-protected-resource/...` | PRM when clients append the resource path. |
| GET | `/api/oauth/bff/<server_id>/.well-known/openid-configuration` | OIDC discovery (primary). |
| GET | `/api/oauth/bff/<server_id>/.well-known/oauth-authorization-server` | Same metadata (RFC 8414 name). |
| GET | `/.well-known/openid-configuration` | Alternate; `?server_id=` or `?resource=`. |
| GET | `/.well-known/oauth-authorization-server` | Same. |
| GET | `/authorize`, `/api/oauth/authorize` | Start BFF: **`server_id`** or **`resource`**, **`redirect_uri`**, **`state`**, **`scope`**. |
| GET | `/api/oauth/callback` | IdP return; exchange + redirect with one-time **`code`**. |
| POST | `/api/oauth/token?server_id=` | **`authorization_code`** (one-time code) or **`refresh_token`** (proxied to IdP). Body: **form or JSON**. |
| POST | `/register` | **501** — no dynamic client registration; use a pre-registered IdP client. |

**Per-server issuer:** clients call **`GET {issuer}/.well-known/openid-configuration`** without query params, so the issuer path **must** include **`server_id`** (`/api/oauth/bff/<uuid>`).

## Hosted MCP 401

Without a bearer JWT, responses include  
`WWW-Authenticate: Bearer ..., resource_metadata_uri="<PRM>?server_id=..."`  
so clients can start OAuth.

## Cursor / Claude / `mcp.json` (no DCR)

**`POST /register` returns 501.** Use the same **`client_id`** as **`oauth_bff.client_id`** (Keycloak confidential client). Deploy / hosted status **generates** this when OAuth BFF + OIDC are enabled:

```json
{
  "mcpServers": {
    "your-slug": {
      "url": "http://127.0.0.1:8080/api/users/<user_id>/<slug>",
      "auth": {
        "CLIENT_ID": "make-mcp-bff",
        "scopes": ["openid", "profile", "email", "mcp:tools", "mcp:resources"]
      }
    }
  }
}
```

## Troubleshooting

| Symptom | What to check |
|---------|----------------|
| OAuth debugger **`Last error: null`** | **CORS** for port **6274**; browser blocked `fetch` to the API. |
| Cannot load AS metadata | Use **issuer** from PRM (`.../api/oauth/bff/<id>`), not bare API origin. |
| Connected but no tools / **401** on MCP | **`oidc.issuer`** must match JWT **`iss`**; leave **`audience`** empty until you know **`aud`**; turn off **`require_caller_identity`** for Jam unless you use caller keys. |
| **502** / SSE errors after OAuth | **Docker** + hosted runtime cold start (`npm install`). Ensure **`hosted.container_dial_host`** / bind host match published ports; first deploy can take minutes. |
| PRM **`/api/api/users`** | Normalized to **`/api/users`**. |

## Security

- Do not put IdP **client_secret** in JSON; use **`client_secret_env`**.
- One-time **`code`** and tokens are sensitive; use HTTPS in production.

## Related

- [Hosted MCP security](./hosted-security.md)  
- [Keycloak & local OIDC](./keycloak-local-oidc.md)  
