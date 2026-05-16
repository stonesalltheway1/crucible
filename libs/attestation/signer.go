package attestation

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	cruciblev1 "github.com/crucible/sdk-go/crucible/v1"
)

// Signer signs in-toto Statements into DSSE envelopes.
//
// Phase 1 ships a LocalEd25519Signer that loads (or generates) an Ed25519
// keypair on disk at ~/.crucible/dev-keys/. Production Phase 2 swaps in a
// SigstoreKeylessSigner that obtains a Fulcio-issued cert via OIDC. The
// envelope shape is identical; only the Cert field changes.
type Signer interface {
	// SignStatement returns a DSSE envelope wrapping the given Statement.
	SignStatement(stmt *cruciblev1.InTotoStatement) (*cruciblev1.DsseEnvelope, error)

	// OidcSubject returns the OIDC subject URI for this signer. For the local
	// signer this is "https://accounts.crucible.dev/agents/local/<key-id>".
	OidcSubject() string

	// KeyID is the short identifier for the signing key.
	KeyID() string
}

// LocalEd25519Signer is the dev / Phase-1 default. The keypair lives on disk
// for repeatability across runs; first-run generates the key.
type LocalEd25519Signer struct {
	priv     ed25519.PrivateKey
	pub      ed25519.PublicKey
	keyID    string
	subject  string
}

// NewLocalEd25519Signer returns a signer rooted at dir. If dir is empty,
// ~/.crucible/dev-keys/ is used. On first call the key is generated and
// written; subsequent calls re-load it.
func NewLocalEd25519Signer(dir string) (*LocalEd25519Signer, error) {
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("attestation: locate home dir: %w", err)
		}
		dir = filepath.Join(home, ".crucible", "dev-keys")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("attestation: mkdir keys: %w", err)
	}
	privPath := filepath.Join(dir, "agent.ed25519")
	pubPath := filepath.Join(dir, "agent.ed25519.pub")

	priv, pub, err := loadOrCreateEd25519(privPath, pubPath)
	if err != nil {
		return nil, err
	}
	keyID := hashKeyID(pub)
	return &LocalEd25519Signer{
		priv:    priv,
		pub:     pub,
		keyID:   keyID,
		subject: fmt.Sprintf("https://accounts.crucible.dev/agents/local/%s", keyID),
	}, nil
}

func loadOrCreateEd25519(privPath, pubPath string) (ed25519.PrivateKey, ed25519.PublicKey, error) {
	if b, err := os.ReadFile(privPath); err == nil {
		if len(b) != ed25519.PrivateKeySize {
			return nil, nil, fmt.Errorf("attestation: corrupt private key (size %d)", len(b))
		}
		priv := ed25519.PrivateKey(b)
		pub, ok := priv.Public().(ed25519.PublicKey)
		if !ok {
			return nil, nil, errors.New("attestation: derive public key")
		}
		return priv, pub, nil
	}
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("attestation: generate key: %w", err)
	}
	if err := os.WriteFile(privPath, priv, 0o600); err != nil {
		return nil, nil, fmt.Errorf("attestation: write private key: %w", err)
	}
	if err := os.WriteFile(pubPath, pub, 0o644); err != nil {
		return nil, nil, fmt.Errorf("attestation: write public key: %w", err)
	}
	return priv, pub, nil
}

func hashKeyID(pub ed25519.PublicKey) string {
	h := sha256.Sum256(pub)
	return base64.RawURLEncoding.EncodeToString(h[:8])
}

// SignStatement implements Signer.
func (s *LocalEd25519Signer) SignStatement(stmt *cruciblev1.InTotoStatement) (*cruciblev1.DsseEnvelope, error) {
	if stmt == nil {
		return nil, errors.New("attestation: nil statement")
	}
	payloadBytes, err := json.Marshal(stmt)
	if err != nil {
		return nil, fmt.Errorf("attestation: marshal statement: %w", err)
	}
	// DSSE pae() — pre-authentication encoding — is what's actually signed.
	pae := dssePAE(cruciblev1.PredicateDsseEnvelopePayloadType, payloadBytes)
	sig := ed25519.Sign(s.priv, pae)
	return &cruciblev1.DsseEnvelope{
		PayloadType: cruciblev1.PredicateDsseEnvelopePayloadType,
		Payload:     base64.StdEncoding.EncodeToString(payloadBytes),
		Signatures: []cruciblev1.DsseSignature{{
			KeyID: s.keyID,
			Sig:   base64.StdEncoding.EncodeToString(sig),
		}},
	}, nil
}

// OidcSubject implements Signer.
func (s *LocalEd25519Signer) OidcSubject() string { return s.subject }

// KeyID implements Signer.
func (s *LocalEd25519Signer) KeyID() string { return s.keyID }

// PublicKey returns the Ed25519 public key. Used by Verify.
func (s *LocalEd25519Signer) PublicKey() ed25519.PublicKey { return s.pub }

// Verify confirms a DSSE envelope's signature against a known Ed25519 public key.
// This is the inverse of LocalEd25519Signer.SignStatement; verifiers, the
// promotion gate, and external auditors call this when reading attestations
// off the local journal (or, in Phase 2, off Sigstore Rekor).
func Verify(envelope *cruciblev1.DsseEnvelope, pub ed25519.PublicKey) error {
	if envelope == nil {
		return errors.New("attestation: nil envelope")
	}
	if len(envelope.Signatures) == 0 {
		return errors.New("attestation: envelope has no signatures")
	}
	payloadBytes, err := base64.StdEncoding.DecodeString(envelope.Payload)
	if err != nil {
		return fmt.Errorf("attestation: decode payload: %w", err)
	}
	pae := dssePAE(envelope.PayloadType, payloadBytes)
	for _, s := range envelope.Signatures {
		sig, err := base64.StdEncoding.DecodeString(s.Sig)
		if err != nil {
			return fmt.Errorf("attestation: decode signature: %w", err)
		}
		if ed25519.Verify(pub, pae, sig) {
			return nil
		}
	}
	return errors.New("attestation: no valid signature found")
}

// dssePAE implements DSSE's Pre-Authentication Encoding:
//
//	DSSEv1 PAYLOAD_TYPE_LENGTH PAYLOAD_TYPE PAYLOAD_LENGTH PAYLOAD
//
// Per https://github.com/secure-systems-lab/dsse/blob/master/protocol.md
func dssePAE(payloadType string, payload []byte) []byte {
	return []byte(fmt.Sprintf("DSSEv1 %d %s %d %s",
		len(payloadType), payloadType,
		len(payload), payload))
}

// SigstoreKeylessSigner is the Phase-2 OIDC-backed signer. Phase 1 stubs it
// out; instantiating returns an error pointing to the report.
type SigstoreKeylessSigner struct {
	mu sync.Mutex
}

// NewSigstoreKeylessSigner is the Phase-2 entry point.
func NewSigstoreKeylessSigner() (*SigstoreKeylessSigner, error) {
	return nil, errors.New("STUB: SigstoreKeylessSigner is wired in Phase 2 — see docs/PHASE-1-REPORT.md")
}
