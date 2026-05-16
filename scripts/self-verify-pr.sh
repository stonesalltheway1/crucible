#!/usr/bin/env bash
# self-verify-pr.sh — submit the current PR diff to a deployed Crucible
# verifier instance. Used by .github/workflows/self-verify.yml.

set -euo pipefail

BASE=""
HEAD=""
OUT=""

while [[ $# -gt 0 ]]; do
    case "$1" in
        --base)   BASE="$2"; shift 2 ;;
        --head)   HEAD="$2"; shift 2 ;;
        --output) OUT="$2";  shift 2 ;;
        *) echo "unknown arg: $1" >&2; exit 2 ;;
    esac
done

[[ -z "$BASE" || -z "$HEAD" || -z "$OUT" ]] && { echo "all args required"; exit 2; }

diff_path=$(mktemp)
git diff "$BASE".."$HEAD" > "$diff_path"

response=$(curl --fail-with-body -sS \
    -H "Authorization: Bearer ${CRUCIBLE_API_TOKEN}" \
    -H "Content-Type: application/json" \
    -X POST "${CRUCIBLE_API_ADDR}/v1/tasks/verify-diff" \
    --data-binary "@$diff_path")

printf '%s' "$response" > "$OUT"
echo "Verifier verdict written to $OUT"
