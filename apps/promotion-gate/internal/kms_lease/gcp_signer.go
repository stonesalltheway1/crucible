package kms_lease

import (
	"context"
	"errors"
)

// GcpHsmSigner wraps a Google Cloud HSM key (cloudkms.SignerClient).
//
// Configuration:
//
//   - `KeyName` — cryptoKeyVersions resource name.
//   - `Algorithm` — typically ASYMMETRIC_SIGN_ECDSA_P256_SHA256.
//
// Same scaffold pattern as AwsKMSSigner: the cloud-kms client lives in
// the main package, the signer holds Sign/Verify closures so tests can
// swap a MockSigner without pulling GCP deps into the test runner.
type GcpHsmSigner struct {
	KeyName   string
	Algorithm string
	SignFn    func(ctx context.Context, payload []byte) ([]byte, error)
	VerifyFn  func(payload, sig []byte) error
}

// NewGcpHsmSigner builds a GcpHsmSigner scaffold.
func NewGcpHsmSigner(keyName string, sign func(ctx context.Context, payload []byte) ([]byte, error), verify func(payload, sig []byte) error) (*GcpHsmSigner, error) {
	if keyName == "" {
		return nil, errors.New("kms_lease: GCP keyName required")
	}
	if sign == nil || verify == nil {
		return nil, errors.New("kms_lease: GCP sign/verify closures required")
	}
	return &GcpHsmSigner{KeyName: keyName, Algorithm: "ASYMMETRIC_SIGN_ECDSA_P256_SHA256", SignFn: sign, VerifyFn: verify}, nil
}

// Sign implements Signer.Sign.
func (s *GcpHsmSigner) Sign(ctx context.Context, payload []byte) ([]byte, error) {
	return s.SignFn(ctx, payload)
}

// Verify implements Signer.Verify.
func (s *GcpHsmSigner) Verify(payload, sig []byte) error {
	return s.VerifyFn(payload, sig)
}

// KeyARN implements Signer.KeyARN. Returns the GCP keyName.
func (s *GcpHsmSigner) KeyARN() string { return s.KeyName }
