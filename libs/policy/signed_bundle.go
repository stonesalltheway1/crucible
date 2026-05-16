package policy

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// SignedTenantBundle wraps a TenantBundle plus a detached signature over the
// canonical JSON of the bundle. The gate's rego_engine refuses to compile a
// tenant bundle whose signature doesn't verify against the tenant's
// registered public key.
//
// We use a tiny SignerBytes / VerifierBytes interface here to avoid a
// circular import with libs/attestation (which depends on this via the
// promotion gate). The gate plumbs an attestation.Signer or
// attestation.LocalEd25519Signer in by adapting it to SignerBytes.
type SignedTenantBundle struct {
	Bundle     TenantBundle `json:"bundle"`
	BundleHash string       `json:"bundle_hash"`
	Signature  string       `json:"signature"` // base64
	KeyID      string       `json:"keyid"`
	OidcSubj   string       `json:"oidc_subject,omitempty"`
	SignedAt   time.Time    `json:"signed_at"`
}

// SignerBytes signs raw bytes. Implementations: LocalEd25519Signer in
// libs/attestation, plus a thin in-package SignerBytesFromEd25519 helper.
type SignerBytes interface {
	SignBytes(b []byte) ([]byte, error)
	OidcSubject() string
	KeyID() string
}

// VerifierBytes verifies a signature against raw bytes.
type VerifierBytes interface {
	VerifyBytes(b, sig []byte) error
	KeyID() string
}

// SignBundle hashes + signs a TenantBundle and returns the wrapped envelope.
func SignBundle(tb *TenantBundle, signer SignerBytes) (*SignedTenantBundle, error) {
	if tb == nil {
		return nil, errors.New("policy: nil tenant bundle")
	}
	if signer == nil {
		return nil, errors.New("policy: nil signer")
	}
	bodyHash, body, err := canonicalBundleBytes(tb)
	if err != nil {
		return nil, err
	}
	sig, err := signer.SignBytes(body)
	if err != nil {
		return nil, fmt.Errorf("policy: sign bundle: %w", err)
	}
	return &SignedTenantBundle{
		Bundle:     *tb,
		BundleHash: bodyHash,
		Signature:  base64.StdEncoding.EncodeToString(sig),
		KeyID:      signer.KeyID(),
		OidcSubj:   signer.OidcSubject(),
		SignedAt:   time.Now().UTC(),
	}, nil
}

// VerifyBundle checks the signature on a SignedTenantBundle. Returns the
// inner TenantBundle on success.
func VerifyBundle(env *SignedTenantBundle, verifier VerifierBytes) (*TenantBundle, error) {
	if env == nil {
		return nil, errors.New("policy: nil signed bundle")
	}
	if verifier == nil {
		return nil, errors.New("policy: nil verifier")
	}
	if env.KeyID != "" && verifier.KeyID() != "" && env.KeyID != verifier.KeyID() {
		return nil, fmt.Errorf("policy: signed bundle key id %q does not match verifier key id %q", env.KeyID, verifier.KeyID())
	}
	bodyHash, body, err := canonicalBundleBytes(&env.Bundle)
	if err != nil {
		return nil, err
	}
	if bodyHash != env.BundleHash {
		return nil, fmt.Errorf("policy: signed bundle hash mismatch: expected %s, got %s", env.BundleHash, bodyHash)
	}
	sig, err := base64.StdEncoding.DecodeString(env.Signature)
	if err != nil {
		return nil, fmt.Errorf("policy: decode signature: %w", err)
	}
	if err := verifier.VerifyBytes(body, sig); err != nil {
		return nil, fmt.Errorf("policy: signed bundle verify: %w", err)
	}
	return &env.Bundle, nil
}

// canonicalBundleBytes returns (hex sha256, canonical bytes). The canonical
// form is `json.Marshal` with the modules map's keys sorted ascending.
func canonicalBundleBytes(tb *TenantBundle) (string, []byte, error) {
	// Sort the modules map for canonical hashing.
	type canonical struct {
		TenantID    string      `json:"tenant_id"`
		Description string      `json:"description,omitempty"`
		Query       string      `json:"query,omitempty"`
		IssuedAt    time.Time   `json:"issued_at"`
		Version     int         `json:"version"`
		Modules     [][2]string `json:"modules"`
	}
	c := canonical{
		TenantID:    tb.TenantID,
		Description: tb.Description,
		Query:       tb.Query,
		IssuedAt:    tb.IssuedAt,
		Version:     tb.Version,
	}
	keys := make([]string, 0, len(tb.Modules))
	for k := range tb.Modules {
		keys = append(keys, k)
	}
	// Lexicographic.
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[j] < keys[i] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
	for _, k := range keys {
		c.Modules = append(c.Modules, [2]string{k, tb.Modules[k]})
	}
	body, err := json.Marshal(c)
	if err != nil {
		return "", nil, fmt.Errorf("policy: canonical marshal: %w", err)
	}
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:]), body, nil
}

// Ed25519Signer is a tiny SignerBytes / VerifierBytes pair so callers can
// avoid pulling in libs/attestation for the policy layer.
type Ed25519Signer struct {
	Priv ed25519.PrivateKey
	Pub  ed25519.PublicKey
	ID   string
	Subj string
}

// NewEd25519Signer generates a fresh ed25519 keypair. For local dev only.
func NewEd25519Signer(subject string) (*Ed25519Signer, error) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		return nil, fmt.Errorf("policy: gen key: %w", err)
	}
	id := hex.EncodeToString(pub[:8])
	if subject == "" {
		subject = "https://accounts.crucible.dev/policy/local/" + id
	}
	return &Ed25519Signer{Priv: priv, Pub: pub, ID: id, Subj: subject}, nil
}

func (s *Ed25519Signer) SignBytes(b []byte) ([]byte, error) {
	return ed25519.Sign(s.Priv, b), nil
}

func (s *Ed25519Signer) VerifyBytes(b, sig []byte) error {
	if !ed25519.Verify(s.Pub, b, sig) {
		return errors.New("policy: ed25519 signature did not verify")
	}
	return nil
}

func (s *Ed25519Signer) KeyID() string       { return s.ID }
func (s *Ed25519Signer) OidcSubject() string { return s.Subj }
