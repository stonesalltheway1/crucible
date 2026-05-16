#!/usr/bin/env bash
# Emit a values.yaml snippet for an air-gap install based on flags.

set -euo pipefail

KMS=""
HSM_PKCS11=""
HSM_SLOT="0"
REKOR_MODE="self-hosted"
LLM_PROVIDER="vllm"
LLM_MODELS=""
GPU_NS="gpu-workloads"

while [[ $# -gt 0 ]]; do
    case "$1" in
        --kms) KMS="$2"; shift 2 ;;
        --hsm-pkcs11-lib) HSM_PKCS11="$2"; shift 2 ;;
        --hsm-slot) HSM_SLOT="$2"; shift 2 ;;
        --rekor-mode) REKOR_MODE="$2"; shift 2 ;;
        --llm-provider) LLM_PROVIDER="$2"; shift 2 ;;
        --llm-models) LLM_MODELS="$2"; shift 2 ;;
        --gpu-pool-namespace) GPU_NS="$2"; shift 2 ;;
        *) echo "unknown arg: $1" >&2; exit 2 ;;
    esac
done

cat <<EOF
crucible:
  topology: airgap
  kms:
    provider: $KMS
EOF
if [[ -n "$HSM_PKCS11" ]]; then
    echo "    keyRef: pkcs11://$HSM_SLOT/crucible-prod"
fi
cat <<EOF
  sigstore:
    mode: $REKOR_MODE
  llmRouting:
    tier0Provider: $LLM_PROVIDER
    tier1Provider: $LLM_PROVIDER
    tier2Provider: $LLM_PROVIDER
    verifierProvider: $LLM_PROVIDER
EOF
if [[ -n "$LLM_MODELS" ]]; then
    echo "  airgap:"
    echo "    models:"
    IFS=',' read -ra MODELS <<< "$LLM_MODELS"
    for m in "${MODELS[@]}"; do
        echo "      - name: $m"
    done
fi
cat <<EOF
    gpuPool:
      namespace: $GPU_NS
EOF
