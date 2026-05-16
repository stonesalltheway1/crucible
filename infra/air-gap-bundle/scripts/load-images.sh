#!/usr/bin/env bash
# Load all OCI images from the bundle into a customer's local OCI registry.
#
# Usage:
#   ./load-images.sh --registry registry.internal.acme.com [--prefix crucible/]
#
# Prerequisites: skopeo OR crane on $PATH (skopeo preferred).
#
# Exit 0 on success.

set -euo pipefail

REGISTRY=""
PREFIX="crucible/"
while [[ $# -gt 0 ]]; do
    case "$1" in
        --registry) REGISTRY="$2"; shift 2 ;;
        --prefix)   PREFIX="$2"; shift 2 ;;
        *) echo "unknown arg: $1" >&2; exit 2 ;;
    esac
done
[[ -z "$REGISTRY" ]] && { echo "--registry required"; exit 2; }

PUSHER=""
if command -v skopeo >/dev/null 2>&1; then
    PUSHER="skopeo"
elif command -v crane >/dev/null 2>&1; then
    PUSHER="crane"
else
    echo "✗ neither skopeo nor crane on PATH" >&2
    exit 1
fi

count=0
for img in images/*.oci; do
    [[ -e "$img" ]] || continue
    name=$(basename "$img" .oci)
    target="$REGISTRY/${PREFIX}${name}:2026.06.0"
    echo "Pushing $name → $target"
    if [[ "$PUSHER" == "skopeo" ]]; then
        skopeo copy --preserve-digests --all "oci-archive:$img" "docker://$target"
    else
        crane push "$img" "$target"
    fi
    count=$((count + 1))
done

echo "✓ Loaded $count images into $REGISTRY"
