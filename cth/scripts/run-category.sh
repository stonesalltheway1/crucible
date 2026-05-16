#!/usr/bin/env bash
# Run a single CTH category. Used by .github/workflows/cth.yml.
set -euo pipefail
CATEGORY="${1:?category required}"
CTH_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT_DIR="cth-results/$CATEGORY"
mkdir -p "$OUT_DIR"
( cd "$CTH_ROOT/grading" && \
  go run ./cmd/cth-grade \
      -root "$CTH_ROOT" \
      -category "$CATEGORY" \
      -out "$OUT_DIR/results.json" )
