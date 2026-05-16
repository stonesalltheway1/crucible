#!/usr/bin/env sh
# crucible-verify-swift — Phase-4 stub. See README for Phase-9 plan.
set -eu

cat > /dev/null || true

TIER="tier_0_mutation"
case "${1:-}" in
  --tier=*)
    TIER="${1#--tier=}"
    ;;
esac

NOW="$(date -u +%FT%TZ 2>/dev/null || echo "1970-01-01T00:00:00Z")"

cat >&1 <<EOF
===CRUCIBLE-TESTREPORT===
{
  "schema_version": "1",
  "task_id": "",
  "diff_hash": "",
  "tier": "${TIER}",
  "language": "swift",
  "framework": "muter-stub",
  "verdict": "tool_unavailable",
  "passed": false,
  "started_at": "${NOW}",
  "finished_at": "${NOW}",
  "duration_seconds": 0,
  "wall_clock_budget_seconds": 0,
  "reporter_id": "crucible-verify-swift",
  "reporter_version": "phase4-stub",
  "error": "Swift verifier is a Phase-4 stub. Phase 9+ wires muter + swift-testing."
}
EOF
