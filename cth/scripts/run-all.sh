#!/usr/bin/env bash
# Run every CTH category and emit a single results.json.
set -euo pipefail
CTH_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT_DIR="${1:-cth-results}"
mkdir -p "$OUT_DIR"
( cd "$CTH_ROOT/grading" && \
  go run ./cmd/cth-grade \
      -root "$CTH_ROOT" \
      -out "$OUT_DIR/results.json" )
