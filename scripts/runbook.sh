#!/usr/bin/env sh
# scripts/runbook.sh — local runbook lookup tool.
#
# Usage:
#   ./scripts/runbook.sh                     # list all RB-* entries
#   ./scripts/runbook.sh RB-07               # show runbook RB-07 (verifier disagreement)
#   ./scripts/runbook.sh tier3               # full-text search
#
# Phase 4: surfaces RB-07 (verifier disagreement) and RB-10 (Tier 3
# timeout rate) so operators investigating verifier-pipeline issues
# can pull the procedure quickly without leaving the terminal.
set -eu

RB_FILE="${RB_FILE:-docs/04-operations/runbooks.md}"

if [ ! -f "$RB_FILE" ]; then
    echo "runbook file not found: $RB_FILE" >&2
    exit 1
fi

if [ "$#" -eq 0 ]; then
    echo "Available runbooks (from $RB_FILE):"
    echo
    grep -E '^## RB-[0-9]+:' "$RB_FILE" | sed 's/^## /  /;s/{#.*}$//'
    echo
    echo "Usage:  $0 RB-07  |  $0 tier3  |  $0 verifier"
    exit 0
fi

key="$1"

case "$key" in
    RB-*|rb-*)
        # Exact ID lookup.
        id="$(printf '%s' "$key" | tr '[:lower:]' '[:upper:]')"
        echo "═══ $id ═══"
        # Extract the section: from "## $id:" until the next "## RB-" or EOF.
        awk -v id="$id" '
            BEGIN { p=0 }
            /^## RB-/ {
                if (p) exit
                if ($0 ~ "^## " id ":") { p=1; print; next }
            }
            p { print }
        ' "$RB_FILE"
        ;;
    *)
        # Free-text search.
        echo "Searching for: $key"
        echo
        grep -n -E "$key" "$RB_FILE" | head -40
        echo
        echo "(Pull a specific runbook with $0 RB-NN)"
        ;;
esac
