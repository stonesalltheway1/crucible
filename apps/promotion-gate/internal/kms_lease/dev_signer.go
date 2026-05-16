package kms_lease

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// DevSigner is the local Ed25519 stand-in for AWS KMS / GCP HSM / YubiHSM.
// Used in tests + by `CRUCIBLE_GATE_DEV_MODE=1`.
//
// The keypair lives on disk so a restart preserves audit traceability. The
// "ARN" is synthesised: arn:crucible:kms:dev:<keyid>.
type DevSigner struct {
	priv ed25519.PrivateKey
	pub  ed25519.PublicKey
	arn  string
}

// NewDevSigner loads or generates a keypair under dir.
func NewDevSigner(dir string) (*DevSigner, error) {
	if dir == "" {
		dir = filepath.Join(os.TempDir(), "crucible-kms-dev")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("kms_lease: mkdir dev key dir: %w", err)
	}
	privPath := filepath.Join(dir, "kms.ed25519")
	pubPath := filepath.Join(dir, "kms.ed25519.pub")
	if b, err := os.ReadFile(privPath); err == nil {
		if len(b) != ed25519.PrivateKeySize {
			return nil, errors.New("kms_lease: corrupt dev private key")
		}
		priv := ed25519.PrivateKey(b)
		pub, ok := priv.Public().(ed25519.PublicKey)
		if !ok {
			return nil, errors.New("kms_lease: derive dev public key")
		}
		return &DevSigner{priv: priv, pub: pub, arn: arnFromPub(pub)}, nil
	}
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("kms_lease: gen dev key: %w", err)
	}
	if err := os.WriteFile(privPath, priv, 0o600); err != nil {
		return nil, err
	}
	if err := os.WriteFile(pubPath, pub, 0o644); err != nil {
		return nil, err
	}
	return &DevSigner{priv: priv, pub: pub, arn: arnFromPub(pub)}, nil
}

// Sign implements Signer.
func (s *DevSigner) Sign(_ context.Context, payload []byte) ([]byte, error) {
	return ed25519.Sign(s.priv, payload), nil
}

// KeyARN implements Signer.
func (s *DevSigner) KeyARN() string { return s.arn }

// Verify implements Signer.
func (s *DevSigner) Verify(payload, sig []byte) error {
	if !ed25519.Verify(s.pub, payload, sig) {
		return errors.New("kms_lease: dev signature did not verify")
	}
	return nil
}

func arnFromPub(pub ed25519.PublicKey) string {
	h := sha256.Sum256(pub)
	return "arn:crucible:kms:dev:" + hex.EncodeToString(h[:8])
}
