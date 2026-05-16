#!/usr/bin/env bash
#
# run.sh — Phase-5 migration runner (twin-first promotion flow)
#
# Eating our own dogfood: this script doesn't directly hit prod. It
# submits each migration to Crucible's own control plane against the
# Crucible-owned twin DB, gets a VerifierApproval, then promotes via
# the Promotion Contract (Phase 6, stubbed). For local dev, the
# --direct flag bypasses the promotion contract.
#
# Usage:
#   run.sh                       — applies pending migrations via twin flow
#   run.sh --direct              — local dev only; applies directly via psql
#   run.sh --plan                — dry-run; emits schema diff only
#
# Required env:
#   CRUCIBLE_MEMORY_PG_DSN       — Postgres DSN (router service role for direct)
#   CRUCIBLE_FALKOR_ADDR         — FalkorDB host:port
#   CRUCIBLE_REDIS_ADDR          — Redis host:port
#   CRUCIBLE_TWIN_RUNTIME_ADDR   — required unless --direct

set -euo pipefail

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MIG_DIR="${DIR}/postgres/migrations"
FALKOR_DIR="${DIR}/falkordb"

MODE="twin"
for arg in "$@"; do
    case "$arg" in
        --direct) MODE="direct" ;;
        --plan)   MODE="plan" ;;
        *) echo "unknown arg: $arg" >&2; exit 64 ;;
    esac
done

if [[ "$MODE" == "twin" && -z "${CRUCIBLE_TWIN_RUNTIME_ADDR:-}" ]]; then
    echo "twin mode requires CRUCIBLE_TWIN_RUNTIME_ADDR — pass --direct for local dev" >&2
    exit 64
fi

echo "==> Running Phase-5 migrations (mode=$MODE)"

case "$MODE" in
    direct)
        for f in "$MIG_DIR"/*.sql; do
            echo "  apply: $(basename "$f")"
            psql "$CRUCIBLE_MEMORY_PG_DSN" -f "$f" -v ON_ERROR_STOP=1
        done
        for f in "$FALKOR_DIR"/*.cypher; do
            echo "  falkor: $(basename "$f")"
            # FalkorDB ships a `falkor-cli`; we shell out via redis-cli for the
            # bootstrap because the cypher files are per-graph and the loader
            # iterates per tenant. graph-level apply happens in the Go runner.
        done
        ;;
    twin)
        # Submit migrations via twin.db.migrate → verifier tier 4 → promote.
        echo "  twin promotion flow not yet wired (Phase 6 hook); see run.sh"
        echo "  use --direct for local dev."
        exit 1
        ;;
    plan)
        for f in "$MIG_DIR"/*.sql; do
            echo "  would apply: $(basename "$f")"
        done
        ;;
esac

echo "==> done"
