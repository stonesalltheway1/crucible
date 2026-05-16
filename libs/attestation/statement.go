// Package attestation builds, signs, and publishes Crucible in-toto attestations.
//
// Architecture (per docs/03-sdk/attestation-formats.md):
//
//   payload → InTotoStatement → DSSE envelope → Publisher
//                                   │
//                       Sigstore Rekor v2 (when CRUCIBLE_REKOR_PUBLISH=1)
//                                   │
//                       Local hash-chained journal (default)
//
// Phase 1 ships the local journal as the default publisher because Sigstore
// Rekor v2 has not yet GA'd as of May 2026 (see Phase 1 report). The local
// journal is hash-chained and content-addressed so attestations produced today
// remain verifiable after Phase-2 wires the real Rekor client.
package attestation

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	cruciblev1 "github.com/crucible/sdk-go/crucible/v1"
)

// BuildStatement assembles an in-toto Statement v1 given a predicate-type URI,
// a single subject (name + sha256 digest), and a typed predicate payload that
// will be JSON-serialized.
//
// The returned Statement is the canonical signed payload — the same bytes go
// into the DSSE envelope's payload field after base64 encoding.
func BuildStatement(predicateType, subjectName string, subjectDigest [32]byte, predicate any) (*cruciblev1.InTotoStatement, error) {
	if predicateType == "" {
		return nil, fmt.Errorf("attestation: predicate type cannot be empty")
	}
	if subjectName == "" {
		return nil, fmt.Errorf("attestation: subject name cannot be empty")
	}
	raw, err := canonicalJSON(predicate)
	if err != nil {
		return nil, fmt.Errorf("attestation: marshal predicate: %w", err)
	}
	return &cruciblev1.InTotoStatement{
		Type: cruciblev1.PredicateInTotoStatementType,
		Subject: []cruciblev1.StatementSubject{{
			Name:   subjectName,
			Digest: map[string]string{"sha256": hex.EncodeToString(subjectDigest[:])},
		}},
		PredicateType: predicateType,
		Predicate:     json.RawMessage(raw),
	}, nil
}

// SubjectDigest hashes content with SHA-256 for use as the in-toto subject.digest.
func SubjectDigest(content []byte) [32]byte {
	return sha256.Sum256(content)
}

// canonicalJSON marshals v with sorted keys and no map iteration ordering so
// the byte representation is stable across signer / verifier runs.
//
// We use encoding/json with stable sort via json.Marshal — encoding/json
// already sorts map keys alphabetically. For nested map[string]any this gives
// us deterministic bytes.
func canonicalJSON(v any) ([]byte, error) {
	if raw, ok := v.(json.RawMessage); ok {
		return raw, nil
	}
	return json.Marshal(v)
}

// Now is overridable for testing.
var Now = func() time.Time { return time.Now().UTC() }
