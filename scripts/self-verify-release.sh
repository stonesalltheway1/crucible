#!/usr/bin/env bash
# self-verify-release.sh — run Crucible's verifier on the bundled release
# artifacts before publishing them. Non-optional per Phase-8 GUARDRAILS.

set -euo pipefail

VERSION=""
BUNDLE=""

while [[ $# -gt 0 ]]; do
    case "$1" in
        --version) VERSION="$2"; shift 2 ;;
        --bundle)  BUNDLE="$2";  shift 2 ;;
        *) echo "unknown arg: $1" >&2; exit 2 ;;
    esac
done

[[ -z "$VERSION" || -z "$BUNDLE" ]] && { echo "--version + --bundle required"; exit 2; }
[[ -z "${CRUCIBLE_API_ADDR:-}" || -z "${CRUCIBLE_API_TOKEN:-}" ]] && {
    echo "CRUCIBLE_API_ADDR + CRUCIBLE_API_TOKEN required"; exit 2;
}

bundle_sha=$(sha256sum "$BUNDLE" | awk '{print $1}')

response=$(curl --fail-with-body -sS \
    -H "Authorization: Bearer $CRUCIBLE_API_TOKEN" \
    -H "Content-Type: application/json" \
    -X POST "$CRUCIBLE_API_ADDR/v1/release/verify" \
    -d "$(printf '{"version":"%s","bundle_sha256":"%s"}' "$VERSION" "$bundle_sha")")

echo "$response"
verdict=$(printf '%s' "$response" | jq -r '.verdict // "unknown"')
rekor_uuid=$(printf '%s' "$response" | jq -r '.rekor_uuid // ""')

if [[ "$verdict" != "approved" ]]; then
    echo "✗ Self-verification failed: $verdict" >&2
    exit 1
fi
echo "✓ Self-verification approved. Rekor UUID: $rekor_uuid"
