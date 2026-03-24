# Keycloak (local OIDC for hosted MCP)

Used with `docker compose --profile oidc` (see repo root `docker-compose.yml`).

## First-time realm setup (Admin UI)

1. Open **http://localhost:8180** — login **admin** / **admin** (dev only; change for anything shared).
2. **Create realm** → name: **`mcp-dev`** → Create.
3. Realm **Settings → General**:
   - Set **Frontend URL** to **`http://keycloak:8080`** (matches what we use in `hosted_security_config` so JWT `iss` matches JWKS discovery from the API container).
4. **Clients → Create client**
   - **Client ID**: `mcp-hosted`
   - **Client authentication**: On  
   - **Service accounts roles**: On (for client-credentials)  
   - **Direct access grants**: On (optional, for username/password tests only)
   - Save, then **Credentials** tab → copy **Client secret**.
5. **Service account roles** (if client-credentials returns 403):  
   - Clients → `mcp-hosted` → **Service account roles** → assign **realm-management** view if needed, or use **Users** → create a test user for password grant instead.

## Get an access token (client credentials)

From your **host** (with Keycloak on `localhost:8180`), the token `iss` may be `http://localhost:8180/...` unless Frontend URL is set as above. Prefer getting tokens **from the Docker network** so `iss` is `http://keycloak:8080/realms/mcp-dev`:

```bash
docker compose --profile oidc exec backend curl -s -X POST \
  "http://keycloak:8080/realms/mcp-dev/protocol/openid-connect/token" \
  -d grant_type=client_credentials \
  -d client_id=mcp-hosted \
  -d client_secret='YOUR_CLIENT_SECRET'
```

Parse `access_token` from the JSON response and send:

`Authorization: Bearer <access_token>`

to your hosted MCP URL.

## Issuer string checklist

| Where API runs | Use `oidc.issuer` |
|----------------|-------------------|
| `docker compose` backend container | `http://keycloak:8080/realms/mcp-dev` |
| `go run` on host (API on localhost:8080) | `http://localhost:8180/realms/mcp-dev` |

JWKS is discovered from `{issuer}/.well-known/openid-configuration` — that URL must be reachable from the process running the Make MCP API.

See **[docs/keycloak-local-oidc.md](../../docs/keycloak-local-oidc.md)** for full flow and `hosted_security_config` examples.
