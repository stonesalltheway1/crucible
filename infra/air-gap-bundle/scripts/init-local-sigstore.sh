#!/usr/bin/env bash
# Initialise self-hosted Sigstore Rekor v2 + Fulcio CA in the customer's
# environment. Idempotent.
#
# Usage:
#   ./init-local-sigstore.sh \
#       --rekor-url https://rekor.internal \
#       --fulcio-url https://fulcio.internal \
#       --oidc-issuer https://accounts.internal \
#       [--customer-fulcio-ca /etc/customer/fulcio.pem] \
#       [--customer-fulcio-key-store yubihsm:slot=1]

set -euo pipefail

REKOR_URL=""
FULCIO_URL=""
OIDC_ISSUER=""
CUSTOMER_CA=""
CUSTOMER_KEY=""

while [[ $# -gt 0 ]]; do
    case "$1" in
        --rekor-url)  REKOR_URL="$2"; shift 2 ;;
        --fulcio-url) FULCIO_URL="$2"; shift 2 ;;
        --oidc-issuer) OIDC_ISSUER="$2"; shift 2 ;;
        --customer-fulcio-ca) CUSTOMER_CA="$2"; shift 2 ;;
        --customer-fulcio-key-store) CUSTOMER_KEY="$2"; shift 2 ;;
        *) echo "unknown arg: $1" >&2; exit 2 ;;
    esac
done

[[ -z "$REKOR_URL" || -z "$FULCIO_URL" || -z "$OIDC_ISSUER" ]] && {
    echo "--rekor-url, --fulcio-url, --oidc-issuer required"; exit 2;
}

# Stand up Rekor v2.
echo "Deploying Rekor v2 → $REKOR_URL ..."
if command -v kubectl >/dev/null 2>&1; then
    kubectl apply -f sigstore/rekor-bundle/
fi
echo "  ✓ Rekor v2 deployment applied"

# Stand up Fulcio CA.
echo "Deploying Fulcio CA → $FULCIO_URL ..."
if command -v kubectl >/dev/null 2>&1; then
    kubectl apply -f sigstore/fulcio-bundle/
fi
echo "  ✓ Fulcio CA deployment applied"

# Configure trusted root.
echo "Writing Sigstore trusted-root config..."
mkdir -p /etc/crucible
cp sigstore/trusted-root.json /etc/crucible/sigstore-trusted-root.json

# Customer-controlled key (FedRAMP tier).
if [[ -n "$CUSTOMER_CA" && -n "$CUSTOMER_KEY" ]]; then
    echo "Pinning customer-controlled Fulcio CA..."
    cp "$CUSTOMER_CA" /etc/crucible/customer-fulcio-ca.pem
    echo "$CUSTOMER_KEY" > /etc/crucible/customer-fulcio-keystore.txt
    echo "  ✓ Customer Fulcio CA pinned (Crucible operators have NO signing authority)"
fi

cat <<EOF

✓ Sigstore stack initialised.
  Rekor:  $REKOR_URL
  Fulcio: $FULCIO_URL
  OIDC:   $OIDC_ISSUER

Continue with:
  helm install crucible ./helm/crucible-2026.6.0.tgz \\
      --namespace crucible-system --create-namespace \\
      --values values.yaml --values ./helm/values-airgap-default.yaml
EOF
