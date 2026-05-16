package kms_lease

import (
	"context"
	"errors"
)

// YubiHsmSigner wraps a PKCS#11 session against a YubiHSM 2 device, the
// FedRAMP-track on-prem signer.
//
// Configuration:
//
//   - `Slot` — PKCS#11 slot index (default 0).
//   - `KeyLabel` — CKA_LABEL on the signing key.
//   - `Algorithm` — typically CKM_ECDSA_SHA256 against a P-256 key.
//
// PKCS#11 plumbing isn't bound here so air-gap installs without the
// PKCS#11 module can still build the gate; the closures are provided
// by the daemon entrypoint.
type YubiHsmSigner struct {
	Slot      uint64
	KeyLabel  string
	Algorithm string
	SignFn    func(ctx context.Context, payload []byte) ([]byte, error)
	VerifyFn  func(payload, sig []byte) error
}

// NewYubiHsmSigner builds a YubiHsmSigner scaffold.
func NewYubiHsmSigner(slot uint64, keyLabel string, sign func(ctx context.Context, payload []byte) ([]byte, error), verify func(payload, sig []byte) error) (*YubiHsmSigner, error) {
	if keyLabel == "" {
		return nil, errors.New("kms_lease: YubiHSM keyLabel required")
	}
	if sign == nil || verify == nil {
		return nil, errors.New("kms_lease: YubiHSM sign/verify closures required")
	}
	return &YubiHsmSigner{Slot: slot, KeyLabel: keyLabel, Algorithm: "CKM_ECDSA_SHA256", SignFn: sign, VerifyFn: verify}, nil
}

// Sign implements Signer.Sign.
func (s *YubiHsmSigner) Sign(ctx context.Context, payload []byte) ([]byte, error) {
	return s.SignFn(ctx, payload)
}

// Verify implements Signer.Verify.
func (s *YubiHsmSigner) Verify(payload, sig []byte) error {
	return s.VerifyFn(payload, sig)
}

// KeyARN implements Signer.KeyARN. Synthesises a stable identifier from
// (slot, label).
func (s *YubiHsmSigner) KeyARN() string {
	return "arn:yubihsm:" + s.KeyLabel
}
