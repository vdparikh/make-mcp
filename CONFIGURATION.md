# Configuration

## Files

| File | Purpose |
|------|---------|
| `config/config.yaml` | Non-secret settings for **local** `go run` / tests: listen address, CORS, WebAuthn RP ID/origins, hosted/Docker networking, optional URL overrides, and **`llm:`** (multiple providers: `type: openai_compatible` for Groq/OpenAI-style APIs, or `type: anthropic` for Claude Messages API; API keys from env vars named in `api_key_env`, e.g. `GROQ_API_KEY`, `ANTHROPIC_API_KEY`). |
| `config/config.example.yaml` | Copy to `config.yaml` as a starting point. |
| `k8s/*.yaml` | Kubernetes manifests for the main app (namespace **`make-mcp`**). `k8s/01-secrets.yaml` ships **dev-only** defaults; replace for any shared or production cluster. **`k8s/02-configmap.yaml`** holds the in-cluster copy of non-secret YAML as **`app.config.yaml`** (including **`llm:`**); the backend **Docker image does not embed** `config/config.yaml`. |
| `skaffold.yaml` | [Skaffold](https://skaffold.dev/) — build images and deploy manifests (e.g. Rancher Desktop / local K8s). Run **`skaffold dev`** from the repo root. |
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

**Named localhost URL:** `http://make-mcp.localhost:3000` resolves like `localhost` (most systems treat `*.localhost` as 127.0.0.1). It is listed in CORS / `rp_origins` in `config/config.yaml` (when you run locally) and in **`k8s/02-configmap.yaml`** (in-cluster). For **passkeys**, the WebAuthn **RP ID must match the hostname** in the address bar: if you sign in at `make-mcp.localhost`, set **`WEBAUTHN_RP_ID=make-mcp.localhost`** for the backend; if you use plain `http://localhost:3000`, keep the default **`rp_id: localhost`** in config.

Hosted MCP endpoint security (Bearer / OIDC / mTLS, env profiles, rotation, audit) is documented in **[docs/hosted-security.md](./docs/hosted-security.md)** and in-app under **Documentation → Hosted MCP Security**. **Runtime isolation** (Docker tiers, optional tool HTTP egress allowlist) is in **[docs/hosted-runtime-isolation.md](./docs/hosted-runtime-isolation.md)** — operator caps live under **`hosted.runtime_isolation`** in `config.yaml`. Optional **Keycloak** for local OIDC testing (standalone **`docker run`**, no repo Compose) — see **[docs/keycloak-local-oidc.md](./docs/keycloak-local-oidc.md)**. **OAuth BFF** (browser login via Make MCP, `/.well-known/...`, `/api/oauth/*`) is documented in **[docs/hosted-oauth-bff.md](./docs/hosted-oauth-bff.md)**; optional env **`KEYCLOAK_OAUTH_CLIENT_SECRET`** for the confidential IdP client.

## Kubernetes (Skaffold)

With `kubectl` pointed at your cluster and Skaffold installed, run **`./scripts/skaffold-dev.sh`** (recommended) or **`skaffold dev --cleanup=false`** from the **repository root** so Postgres **PVCs are not deleted** when you stop Skaffold. **Stopping Skaffold only ends its process (and the port-forward); Deployments and StatefulSet pods keep running** by design. To scale **backend**, **frontend**, and **postgres** to zero but **keep** the namespace and volumes, run **`./scripts/skaffold-stop.sh`** (or `make skaffold-stop`). To remove everything Skaffold applied (including the PVC), run **`skaffold delete`**. Plain **`skaffold dev`** (without `--cleanup=false`) tears down deployed resources on exit and **wipes the database volume** in the `make-mcp` namespace. You can also run from **`backend/`** — a small `backend/skaffold.yaml` delegates to the root config.

This builds the backend and frontend Docker images, applies `k8s/*.yaml` into namespace `make-mcp`, and port-forwards the **frontend** service to **localhost:3000** (UI proxies `/api` to the backend in-cluster). Replace the values in **`k8s/01-secrets.yaml`** (or inject secrets via your platform) before production; keep `database-url` and `postgres-password` consistent.

**Config file in cluster:** The backend container sets **`MAKE_MCP_CONFIG=/config/app.config.yaml`**, mounted from the **`app.config.yaml`** key in **`k8s/02-configmap.yaml`**. Edits to repo **`config/config.yaml` do not affect the cluster** until you copy the relevant blocks into that ConfigMap (or change the deployment to mount a different ConfigMap) and restart the backend.

**Try Chat / LLM:** The **`llm:`** block in the ConfigMap only describes providers. **Chat requires API keys in the pod environment** (e.g. **`GROQ_API_KEY`** from Secret key **`groq-api-key`** — see comments in **`k8s/04-backend.yaml`** / **`k8s/01-secrets.yaml`**). Without the key, startup logs `LLM features disabled` and **`POST /api/try/chat`** returns 503; the Session Settings UI still lists providers from YAML after a recent fix.

**Hosted MCP:** The sample manifests use a **Kubernetes** hosted runtime (Pods/Services) plus **`hostPath`** for generated code under **`/var/lib/make-mcp/generated-servers`** on the node (single-node dev). Older Docker-socket-based setups are not what the current **`k8s/04-backend.yaml`** deploys by default.

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

