# Configuration

## Files

| File | Purpose |
|------|---------|
| `config/config.yaml` | Non-secret settings: listen address, CORS, WebAuthn RP ID/origins, hosted/Docker networking, optional URL overrides, and **`llm:`** (multiple providers: `type: openai_compatible` for Groq/OpenAI-style APIs, or `type: anthropic` for Claude Messages API; API keys from env vars named in `api_key_env`, e.g. `GROQ_API_KEY`, `ANTHROPIC_API_KEY`). |
| `config/config.example.yaml` | Copy to `config.yaml` as a starting point. |
| `config/docker.config.yaml` | Used by `docker-compose` (`MAKE_MCP_CONFIG=/config/docker.config.yaml`). |
| `env.example.sh` | **Secrets and dev tooling env vars.** Copy to `env.sh`, fill in values, then `set -a && source ./env.sh && set +a`. |
| `env.sh` | Gitignored local file — your real secrets (create from `env.example.sh`). |

Application hosts and ports come from `config.yaml` (and optional env overrides below).

## Required environment variables (secrets)

Set via `env.sh` or your process manager:

- **`DATABASE_URL`** — PostgreSQL connection string.
- **`JWT_SECRET`** — Strong random string for signing session JWTs (e.g. `openssl rand -hex 32`).

## Optional environment overrides

| Variable | Effect |
|----------|--------|
| `MAKE_MCP_CONFIG` | Path to YAML config file (default: finds `config/config.yaml` relative to cwd). |
| `PORT` | Overrides `server.listen_port` in YAML. |
| `DEBUG` | `true` / `1` / `yes` enables Gin debug mode (also configurable in YAML). |
| `WEBAUTHN_RP_ID` | Overrides `webauthn.rp_id`. |
| `WEBAUTHN_RP_ORIGINS` | Comma-separated list; overrides `webauthn.rp_origins`. |
| `MCP_HOSTED_BASE_URL` | Public API base for hosted MCP URLs (trimmed, no trailing slash). |
| `MCP_OBSERVABILITY_INGEST_BASE_URL` | Base URL for observability ingestion from hosted runtimes. |
| `DOCS_DIR` | Directory for markdown docs served by the API. |

Hosted MCP endpoint security (Bearer / OIDC / mTLS, env profiles, rotation, audit) is documented in **[docs/hosted-security.md](./docs/hosted-security.md)** and in-app under **Documentation → Hosted MCP Security**. **Runtime isolation** (Docker tiers, optional tool HTTP egress allowlist) is in **[docs/hosted-runtime-isolation.md](./docs/hosted-runtime-isolation.md)** — operator caps live under **`hosted.runtime_isolation`** in `config.yaml`. Optional **Keycloak** for local OIDC testing: **`docker compose --profile oidc up -d keycloak`** — see **[docs/keycloak-local-oidc.md](./docs/keycloak-local-oidc.md)**. **OAuth BFF** (browser login via Make MCP, `/.well-known/...`, `/api/oauth/*`) is documented in **[docs/hosted-oauth-bff.md](./docs/hosted-oauth-bff.md)**; optional env **`KEYCLOAK_OAUTH_CLIENT_SECRET`** for the confidential IdP client.

## Local development

From the **repository root**:

```bash
cp env.example.sh env.sh
# Edit env.sh: set JWT_SECRET and DATABASE_URL

set -a && source ./env.sh && set +a
cd backend && go run ./cmd/server
```

In another terminal (with the same `env.sh` sourced for `DEV_API_PROXY_TARGET` and `VITE_EXAMPLE_DB_HOST`):

```bash
cd frontend && npm install && npm run dev
```

Align `DEV_API_PROXY_TARGET` with `server.listen_host` / `server.listen_port` in `config/config.yaml` (e.g. `http://127.0.0.1:8080`).

## WebAuthn / passkeys

`webauthn.rp_id` must be a **valid RP ID** for the host users type in the browser. `localhost` and `127.0.0.1` are different hosts: if `rp_id` is `localhost`, open the app as `http://localhost:3000` (not `http://127.0.0.1:3000`), or change `rp_id` to `127.0.0.1` and use that host consistently. Adjust `webauthn.rp_origins` to list every origin (scheme + host + port) you use.

## Docker Compose

Compose mounts `./config` read-only and sets `MAKE_MCP_CONFIG=/config/docker.config.yaml`. Override the default weak JWT:

```bash
JWT_SECRET=$(openssl rand -hex 32) docker-compose up --build
```
