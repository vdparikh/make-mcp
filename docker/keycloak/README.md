# Keycloak (local OIDC for hosted MCP)

Run Keycloak with **`docker`** on a user-defined network (the Make MCP repo does not ship root-level Compose). See **[docs/keycloak-local-oidc.md](../../docs/keycloak-local-oidc.md)** for **`hosted_security_config`** and issuer rules.

## Start Keycloak (example)

```bash
docker network create mcp-keycloak-dev 2>/dev/null || true
docker run -d --name keycloak --network mcp-keycloak-dev -p 8180:8080 \
  -e KEYCLOAK_ADMIN=admin -e KEYCLOAK_ADMIN_PASSWORD=admin \
  quay.io/keycloak/keycloak:latest start-dev
```

Admin UI: **http://localhost:8180** (admin / admin in dev only).

## First-time realm setup (Admin UI)

1. Open **http://localhost:8180** — login **admin** / **admin** (dev only; change for anything shared).
2. **Create realm** → name: **`mcp-dev`** → Create.
3. Realm **Settings → General**:
   - Set **Frontend URL** to match what the **Make MCP API** will use for OIDC discovery (see table below), e.g. **`http://host.docker.internal:8180`** if the API runs in Kubernetes on Docker Desktop and reaches Keycloak at that host, or **`http://localhost:8180`** if the API runs on the host.
4. **Clients → Create client**
   - **Client ID**: `mcp-hosted`
   - **Client authentication**: On  
   - **Service accounts roles**: On (for client-credentials)  
   - **Direct access grants**: On (optional, for username/password tests only)
   - Save, then **Credentials** tab → copy **Client secret**.
5. **Service account roles** (if client-credentials returns 403):  
   - Clients → `mcp-hosted` → **Service account roles** → assign **realm-management** view if needed, or use **Users** → create a test user for password grant instead.

## Get an access token (client credentials)

Call the token endpoint from a container on **`mcp-keycloak-dev`** so the hostname **`keycloak`** resolves:

```bash
docker run --rm --network mcp-keycloak-dev curlimages/curl:latest -s -X POST \
  "http://keycloak:8080/realms/mcp-dev/protocol/openid-connect/token" \
  -d grant_type=client_credentials \
  -d client_id=mcp-hosted \
  -d client_secret='YOUR_CLIENT_SECRET'
```

Parse `access_token` from the JSON response and send:

`Authorization: Bearer <access_token>`

to your hosted MCP URL.

## Issuer string checklist

| Where the Make MCP API runs | Set Keycloak Frontend URL + use `oidc.issuer` |
|-----------------------------|-----------------------------------------------|
| `go run` on host | e.g. **`http://localhost:8180`** |
| Kubernetes pod (typical Docker Desktop) | e.g. **`http://host.docker.internal:8180`** — must match token `iss` / discovery |

JWKS is discovered from `{issuer}/.well-known/openid-configuration` — that URL must be reachable from the process running the Make MCP API.

See **[docs/keycloak-local-oidc.md](../../docs/keycloak-local-oidc.md)** for full flow and `hosted_security_config` examples.
