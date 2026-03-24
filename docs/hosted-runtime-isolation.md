# Hosted runtime isolation

Hosted MCP servers run in **Docker** with configurable **CPU / memory / process limits** and optional **application-layer egress control** for generated tool code.

## Isolation tiers

| Tier | Typical use |
|------|----------------|
| `standard` | Default; balanced limits. |
| `restricted` | Tighter CPU/memory for noisy or untrusted workloads. |
| `strict` | Smallest footprint; use with explicit egress allowlists. |

Tier presets and hard **caps** are defined in **`config.yaml`** under `hosted.runtime_isolation` (see `config.example.yaml`). Users can override `memory_mb`, `nano_cpus`, or `pids_limit` per server in **`hosted_runtime_config`** up to the operator caps.

## Egress policy (`egress_policy`)

- **`allow_all`** (default): Generated tools may call any `http:`/`https:` URL (same as before isolation).
- **`deny_default`**: Only hosts listed in **`egress_allowlist`** plus **automatic** entries are allowed:
  - Hosts inferred from **tool** `execution_config` URLs (REST/GraphQL/webhook/OAuth token URL, flow API nodes).
  - Host from **Server → Environments** for the **publish** `env_profile` (base URL).
  - Host of **observability ingest** (`MCP_OBSERVABILITY_ENDPOINT`) so telemetry still works.

**Important:** `npm install` during container **cold start** still uses the network normally; egress policy applies to the **running Node MCP server** (`fetch` in generated tools). OS-level firewall / `iptables` in the container is **not** implemented here.

### Allowlist format

Comma-separated in the container env `MCP_EGRESS_ALLOWLIST`. Each entry is a **hostname** or wildcard:

- `api.example.com` — exact host.
- `*.cdn.example.net` — subdomains of `cdn.example.net` (not the apex; add apex explicitly if needed).

Configure via Deploy → **hosted_runtime_config** JSON or the Deploy UI fields.

## Example `hosted_runtime_config`

```json
{
  "isolation_tier": "strict",
  "egress_policy": "deny_default",
  "egress_allowlist": ["extra-api.partner.com"]
}
```

## API

- Persisted on the server row as **`hosted_runtime_config`** (JSONB).
- Sent on **`POST /api/servers/:id/hosted-publish`** as `hosted_runtime_config`.
- Returned from **`GET /api/servers/:id/hosted-security`** as `hosted_runtime_config`.

## Manifest

Published **`manifest.json`** under the generated server includes:

- `metadata.isolation` — tier, egress policy, merged egress host list.
- `metadata.resources` — effective memory / nano_cpus / pids.
