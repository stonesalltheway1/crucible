package kms_lease

import (
	"context"
	"errors"
)

// AwsKMSSigner wraps an aws-sdk-go-v2 kms.Client to mint asymmetric
// signatures via KMS.Sign. The signing algorithm is ECDSA_SHA_256 for
// asymmetric KMS keys (matches the FedRAMP-track default).
//
// Configuration:
//
//   - `KeyId` — KMS key ARN.
//   - `Region` — AWS region (set on the aws.Config).
//   - `Algorithm` — defaults to "ECDSA_SHA_256".
//
// The Phase-6 scaffold sketches the call boundary; the actual kms.Client
// is plumbed in via the api/server when CRUCIBLE_KMS_PROVIDER=aws is set
// at process start. The boundary is small enough that we can re-test the
// Go-side behaviour with a MockSigner in unit tests; production smoke
// tests in `infra/argo-rollouts/` exercise the real KMS path.
type AwsKMSSigner struct {
	KeyId     string
	Algorithm string
	Sign      func(ctx context.Context, payload []byte) ([]byte, error)
	Verify_   func(payload, sig []byte) error
}

// NewAwsKMSSigner returns a signer scaffold. The actual aws-sdk-go-v2
// wiring lives in the main package (so libs/policy and friends don't pull
// AWS deps); this signer holds the closures.
func NewAwsKMSSigner(keyARN string, sign func(ctx context.Context, payload []byte) ([]byte, error), verify func(payload, sig []byte) error) (*AwsKMSSigner, error) {
	if keyARN == "" {
		return nil, errors.New("kms_lease: AWS keyARN required")
	}
	if sign == nil || verify == nil {
		return nil, errors.New("kms_lease: AWS sign/verify closures required")
	}
	return &AwsKMSSigner{KeyId: keyARN, Algorithm: "ECDSA_SHA_256", Sign: sign, Verify_: verify}, nil
}

// SignAdapter implements Signer.Sign.
func (s *AwsKMSSigner) SignAdapter(ctx context.Context, payload []byte) ([]byte, error) {
	return s.Sign(ctx, payload)
}

// VerifyAdapter implements Signer.Verify.
func (s *AwsKMSSigner) VerifyAdapter(payload, sig []byte) error {
	return s.Verify_(payload, sig)
}

// KeyARN implements Signer.KeyARN.
func (s *AwsKMSSigner) KeyARN() string { return s.KeyId }
