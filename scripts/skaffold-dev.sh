#!/usr/bin/env bash
# Run Skaffold without deleting the make-mcp namespace when you Ctrl+C — keeps Postgres PVCs and data.
#
# By design, workloads (pods) stay Running after you exit; only Skaffold's process (and port-forward)
# stops. To scale app/postgres down but keep PVCs: ./scripts/skaffold-stop.sh
set -euo pipefail
cd "$(dirname "$0")/.."

on_exit() {
	local code=$?
	echo "" >&2
	echo "Skaffold exited (exit ${code}). Namespace ${SKAFFOLD_NAMESPACE:-make-mcp} was left intact (--cleanup=false)." >&2
	echo "  Pods may still be running. To stop them but keep Postgres data: ./scripts/skaffold-stop.sh" >&2
	echo "  To remove all applied resources (including PVC): skaffold delete" >&2
}
trap on_exit EXIT

skaffold dev --cleanup=false "$@"
