#!/usr/bin/env bash
# Scale main workloads to zero in make-mcp. Keeps namespace + PVCs (Postgres data).
# Use after exiting ./scripts/skaffold-dev.sh when you want pods to stop but data preserved.
set -euo pipefail
NS="${SKAFFOLD_NAMESPACE:-make-mcp}"

if ! kubectl get ns "$NS" >/dev/null 2>&1; then
  echo "Namespace $NS not found; nothing to scale." >&2
  exit 0
fi

echo "Scaling deployments and postgres to 0 in namespace $NS..."
kubectl scale deployment backend frontend --replicas=0 -n "$NS"
kubectl scale statefulset postgres --replicas=0 -n "$NS"
echo "Done. Port-forward from Skaffold is already gone; cluster pods for app/postgres are stopped."
echo "Hosted MCP user pods (label make-mcp.managed=true), if any, are unchanged — delete them in the UI or with kubectl if you want them gone."
echo "Start again: ./scripts/skaffold-dev.sh"
echo "Delete everything Skaffold applied (including PVC): skaffold delete"
