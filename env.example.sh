#!/usr/bin/env bash
# Copy to env.sh and fill in real values:  cp env.example.sh env.sh
# Load before running backend or frontend dev:
#   set -a && source ./env.sh && set +a
#
# Keep URLs aligned with config/config.yaml (hostname, ports).

# --- Required secrets ---
export DATABASE_URL="postgres://postgres:postgres@127.0.0.1:5432/mcp_builder?sslmode=disable"
export JWT_SECRET="CHANGE_ME_use_openssl_rand_hex_32"

# --- Optional: override config file path (default finds config/config.yaml) ---
# export MAKE_MCP_CONFIG="/absolute/path/to/config.yaml"

# --- Optional: WebAuthn overrides (otherwise use config.yaml) ---
# export WEBAUTHN_RP_ID="your.hostname"
# export WEBAUTHN_RP_ORIGINS="https://app.example.com,https://app.example.com:443"

# --- Optional: hosted OAuth BFF (Keycloak confidential client secret; see docs/hosted-oauth-bff.md) ---
# export KEYCLOAK_OAUTH_CLIENT_SECRET="..."

# --- Optional: hosted / observability public URLs (otherwise derived from incoming requests) ---
# export MCP_HOSTED_BASE_URL="https://api.example.com/api"
# export MCP_OBSERVABILITY_INGEST_BASE_URL="https://api.example.com"

# --- Frontend dev: Vite proxy target (must match server.listen_* in config.yaml) ---
export DEV_API_PROXY_TARGET="http://127.0.0.1:8080"

# --- Frontend: example DB host shown in tool templates (no secrets) ---
export VITE_EXAMPLE_DB_HOST="127.0.0.1"

# --- Optional: Docker ---
# export DOCKER_HOST="unix:///path/to/docker.sock"
