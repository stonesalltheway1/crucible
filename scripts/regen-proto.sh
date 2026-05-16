#!/usr/bin/env bash
# Regenerate SDK stubs from libs/twin-spec/proto.
#
# Phase 1 note: Phase 1 ships hand-rolled Go types under libs/sdk-go/crucible/v1/
# so the control plane builds without `buf generate`. CI runs this script to
# verify that the hand-rolled types stay aligned with the proto source-of-truth
# once `buf` is wired in.

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SPEC="$ROOT/libs/twin-spec"

if ! command -v buf >/dev/null 2>&1; then
  echo "buf not found in PATH. Run 'nix develop' or install from https://buf.build" >&2
  exit 1
fi

cd "$SPEC"

buf lint
buf format -d
buf generate

echo
echo "Regenerated SDK stubs:"
echo "  ../sdk-go/gen"
echo "  ../sdk-ts/src/gen"
echo "  ../sdk-py/crucible_sdk/gen"
echo "  ../sdk-rs/src/gen"
