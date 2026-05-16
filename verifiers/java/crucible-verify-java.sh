#!/usr/bin/env sh
# crucible-verify-java — Phase-4 stub. Emits a TestReport with
# Verdict=tool_unavailable so the verifier daemon's process pool
# surfaces the gap honestly rather than silently skipping.
#
# Real Java verifier ships in Phase 9 if a design partner asks.
set -eu

# Read stdin to avoid SIGPIPE when the daemon writes the request.
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
  "language": "java",
  "framework": "pitest-stub",
  "verdict": "tool_unavailable",
  "passed": false,
  "started_at": "${NOW}",
  "finished_at": "${NOW}",
  "duration_seconds": 0,
  "wall_clock_budget_seconds": 0,
  "reporter_id": "crucible-verify-java",
  "reporter_version": "phase4-stub",
  "error": "Java verifier is a Phase-4 stub. Phase 9+ wires Pitest + jqwik + JQF."
}
EOF
