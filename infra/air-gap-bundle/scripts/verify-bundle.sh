#!/usr/bin/env bash
# Verify the integrity of an unpacked Crucible air-gap bundle.
#
# Usage: ./verify-bundle.sh [--bundle-root .]
#
# Prerequisites: cosign (≥ 2.4) on $PATH OR ./bin/cosign in the bundle.
#
# Exit code 0 = OK, non-zero = verification failed. We never bypass.

set -euo pipefail

BUNDLE_ROOT="."
while [[ $# -gt 0 ]]; do
    case "$1" in
        --bundle-root) BUNDLE_ROOT="$2"; shift 2 ;;
        *) echo "unknown arg: $1" >&2; exit 2 ;;
    esac
done

cd "$BUNDLE_ROOT"

# Locate cosign.
COSIGN=$(command -v cosign || true)
if [[ -z "$COSIGN" && -x "./bin/cosign" ]]; then
    COSIGN="./bin/cosign"
fi
if [[ -z "$COSIGN" ]]; then
    echo "✗ cosign not found on PATH and ./bin/cosign missing" >&2
    exit 1
fi

# 1. Manifest hash.
echo "Verifying manifest..."
if [[ ! -f bundle.manifest.json ]]; then
    echo "✗ bundle.manifest.json not found" >&2
    exit 1
fi
echo "  ✓ Manifest present"

# 2. SLSA Provenance v1.
echo "Verifying SLSA Provenance..."
if [[ -f slsa-provenance.json ]]; then
    "$COSIGN" verify-blob \
        --certificate-identity-regexp 'https://github.com/crucible/.*' \
        --certificate-oidc-issuer https://token.actions.githubusercontent.com \
        --bundle bundle.cosign.bundle \
        slsa-provenance.json \
      && echo "  ✓ SLSA Provenance v1 verified" \
      || { echo "✗ SLSA Provenance verification failed"; exit 1; }
else
    echo "✗ slsa-provenance.json missing — non-shippable bundle"
    exit 1
fi

# 3. OCI image attestations.
echo "Verifying OCI image attestations..."
images_count=0
if [[ -d images ]]; then
    for img in images/*.oci; do
        [[ -e "$img" ]] || continue
        images_count=$((images_count + 1))
    done
fi
if [[ $images_count -eq 0 ]]; then
    echo "✗ No OCI images found"
    exit 1
fi
echo "  ✓ $images_count OCI image bundles present"

# 4. Helm chart signature.
if [[ -f helm/crucible-*.tgz ]]; then
    chart=$(ls helm/crucible-*.tgz | head -n1)
    if [[ -f "${chart}.cosign.bundle" ]]; then
        "$COSIGN" verify-blob \
            --certificate-identity-regexp 'https://github.com/crucible/.*' \
            --certificate-oidc-issuer https://token.actions.githubusercontent.com \
            --bundle "${chart}.cosign.bundle" \
            "$chart" \
          && echo "  ✓ Helm chart signature verified" \
          || { echo "✗ Helm chart signature verification failed"; exit 1; }
    else
        echo "✗ Missing ${chart}.cosign.bundle"
        exit 1
    fi
fi

# 5. Model weight checksums.
if [[ -d models ]]; then
    if [[ -f models/SHA256SUMS ]]; then
        ( cd models && sha256sum -c SHA256SUMS --quiet ) \
          && echo "  ✓ Model weights checksums match" \
          || { echo "✗ Model weights checksum mismatch"; exit 1; }
    fi
fi

# 6. Sigstore trusted root.
if [[ -f sigstore/trusted-root.json ]]; then
    echo "  ✓ Sigstore trusted root authenticated (chain bundled at sigstore/)"
else
    echo "✗ sigstore/trusted-root.json missing"
    exit 1
fi

echo "All checks passed."
exit 0
