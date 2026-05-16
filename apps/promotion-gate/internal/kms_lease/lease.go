// Package kms_lease mints single-use, time-boxed, action-scoped credential
// leases signed by AWS KMS / GCP Cloud HSM / YubiHSM.
//
// The whole point of the lease design is that the gate's agent process
// NEVER touches a long-lived production credential. The agent submits a
// PromotionBundle; the gate produces a Lease (signed token) whose envelope
// is scoped to one specific action (deploy artifact X to service Y, or
// run migration Z); the deploy pipeline consumes the lease, executes,
// returns. The lease expires automatically.
//
// Implementations:
//   - AwsKMS — AWS KMS asymmetric signing via SignWithGrant (SignaturePolicy).
//   - GcpHsm — GCP Cloud HSM AsymmetricSign.
//   - YubiHsm — PKCS#11 wrapper for FedRAMP-track customers.
//   - Dev    — local Ed25519 dev signer (always present; used in tests).
//
// Phase-6 ships:
//   - The full lease shape + idempotency tracker.
//   - The Dev signer (works without external dependencies).
//   - Adapter scaffolding for AWS / GCP / YubiHSM that wraps a Signer
//     interface; the SDK calls are documented in `aws.go`, `gcp.go`,
//     `yubi.go` and tested against MockSigner.
//
// CRITICAL guardrails:
//   - Lease keys are NEVER cached in the gate's process beyond a single
//     mint→use cycle. The Signer holds a handle, not the key.
//   - Idempotency keys are consumed on first use; replay returns ErrLeaseConsumed.
//   - Issued leases must outlast the deploy pipeline call (configurable;
//     default 5 minutes per promotion-contract.md).
package kms_lease

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
)

// DefaultLeaseTTL matches the promotion-contract.md default.
const DefaultLeaseTTL = 5 * time.Minute

// MaxLeaseTTL caps how long the gate will mint a lease for, even when the
// caller asks for longer. Defence against runaway approvals.
const MaxLeaseTTL = 30 * time.Minute

// Action scopes — narrow the lease to exactly one production action.
const (
	ActionDeployArtifact = "deploy_artifact"
	ActionRunMigration   = "run_migration"
	ActionRollbackDeploy = "rollback_deploy"
)

// Lease is the signed token the deploy pipeline consumes. It is a DSSE-shaped
// envelope but lives in its own type so the rest of the system doesn't have
// to import attestation when only the lease shape is needed.
type Lease struct {
	ID             string            `json:"id"`             // lease_<ulid>
	PromotionID    string            `json:"promotion_id"`
	BundleHash     string            `json:"bundle_hash"`
	Action         string            `json:"action"`
	ActionTarget   map[string]string `json:"action_target"`  // e.g. {"service":"api","cluster":"prod-eu"}
	IssuedAt       time.Time         `json:"issued_at"`
	ExpiresAt      time.Time         `json:"expires_at"`
	IssuerKeyARN   string            `json:"issuer_key_arn"`
	IssuerOidcSubj string            `json:"issuer_oidc_subject"`
	IdempotencyKey string            `json:"idempotency_key"`
	Sig            string            `json:"sig"`            // base64 detached signature over JSON minus this field
}

// LeaseRequest is the gate-internal call to mint a Lease.
type LeaseRequest struct {
	PromotionID  string
	BundleHash   string
	Action       string
	ActionTarget map[string]string
	TTL          time.Duration
	OidcSubject  string
}

// Signer is the abstraction over the underlying KMS / HSM. Implementations:
//   - Dev   — local Ed25519 keys
//   - AwsKMS — wraps an aws-sdk-go-v2 kms.Client
//   - GcpHsm — wraps kms.SignerClient
//   - YubiHsm — wraps a PKCS#11 session
type Signer interface {
	// Sign returns a detached signature over the given canonical bytes.
	Sign(ctx context.Context, payload []byte) ([]byte, error)
	// KeyARN identifies the signing key. Goes into Lease.IssuerKeyARN.
	KeyARN() string
	// Verify is used by tests + by the deploy pipeline's lease-validator.
	Verify(payload, sig []byte) error
}

// LeaseStore is the persistent idempotency-key tracker. The gate uses an
// in-memory store by default; production wires Redis.
type LeaseStore interface {
	Consume(ctx context.Context, leaseID, idem string) error
}

// Manager mints + tracks leases.
type Manager struct {
	signer Signer
	store  LeaseStore
	clock  func() time.Time
}

// New builds a Manager.
func New(signer Signer, store LeaseStore) *Manager {
	if store == nil {
		store = NewInMemoryStore()
	}
	return &Manager{signer: signer, store: store, clock: func() time.Time { return time.Now().UTC() }}
}

// MintLease returns a signed Lease. Idempotency: if the same (promotion_id,
// action, bundle_hash) tuple appears twice, the second call returns
// ErrLeaseConsumed.
func (m *Manager) MintLease(ctx context.Context, req LeaseRequest) (*Lease, error) {
	if req.PromotionID == "" || req.BundleHash == "" || req.Action == "" {
		return nil, errors.New("kms_lease: PromotionID, BundleHash, Action required")
	}
	ttl := req.TTL
	if ttl <= 0 {
		ttl = DefaultLeaseTTL
	}
	if ttl > MaxLeaseTTL {
		ttl = MaxLeaseTTL
	}
	now := m.clock()
	idem := computeIdempotency(req.PromotionID, req.Action, req.BundleHash)
	lease := &Lease{
		ID:             "lease_" + idem[:16],
		PromotionID:    req.PromotionID,
		BundleHash:     req.BundleHash,
		Action:         req.Action,
		ActionTarget:   req.ActionTarget,
		IssuedAt:       now,
		ExpiresAt:      now.Add(ttl),
		IssuerKeyARN:   m.signer.KeyARN(),
		IssuerOidcSubj: req.OidcSubject,
		IdempotencyKey: idem,
	}
	if err := m.store.Consume(ctx, lease.ID, idem); err != nil {
		return nil, err
	}
	canon, err := canonicalLeaseBytes(lease)
	if err != nil {
		return nil, fmt.Errorf("kms_lease: canonicalize: %w", err)
	}
	sig, err := m.signer.Sign(ctx, canon)
	if err != nil {
		return nil, fmt.Errorf("kms_lease: sign: %w", err)
	}
	lease.Sig = base64.StdEncoding.EncodeToString(sig)
	return lease, nil
}

// VerifyLease confirms the lease's signature and freshness. Returns nil
// when the lease is consumable. Called by the delivery_adapter and the
// deploy pipeline.
func (m *Manager) VerifyLease(lease *Lease) error {
	if lease == nil {
		return errors.New("kms_lease: nil lease")
	}
	if m.clock().After(lease.ExpiresAt) {
		return ErrLeaseExpired{ID: lease.ID, ExpiresAt: lease.ExpiresAt}
	}
	canon, err := canonicalLeaseBytes(lease)
	if err != nil {
		return err
	}
	sig, err := base64.StdEncoding.DecodeString(lease.Sig)
	if err != nil {
		return fmt.Errorf("kms_lease: decode sig: %w", err)
	}
	return m.signer.Verify(canon, sig)
}

// AssertScope ensures the caller's intended action is the one the lease
// was minted for. Used at the boundary between gate and pipeline.
func AssertScope(lease *Lease, action string, target map[string]string) error {
	if lease.Action != action {
		return fmt.Errorf("kms_lease: scope action mismatch: lease=%s requested=%s", lease.Action, action)
	}
	for k, v := range target {
		if lease.ActionTarget[k] != v {
			return fmt.Errorf("kms_lease: scope target mismatch on %s: lease=%s requested=%s", k, lease.ActionTarget[k], v)
		}
	}
	return nil
}

// canonicalLeaseBytes encodes a Lease without the Sig field for signing /
// verification.
func canonicalLeaseBytes(l *Lease) ([]byte, error) {
	clone := *l
	clone.Sig = ""
	return json.Marshal(clone)
}

func computeIdempotency(promotionID, action, bundleHash string) string {
	h := sha256.New()
	h.Write([]byte(promotionID))
	h.Write([]byte{0})
	h.Write([]byte(action))
	h.Write([]byte{0})
	h.Write([]byte(bundleHash))
	return hex.EncodeToString(h.Sum(nil))
}

// ── errors ─────────────────────────────────────────────────────────────────

// ErrLeaseConsumed is returned by LeaseStore.Consume on replay.
type ErrLeaseConsumed struct{ ID, Idem string }

func (e ErrLeaseConsumed) Error() string {
	return fmt.Sprintf("kms_lease: lease %s already consumed (idem=%s)", e.ID, e.Idem)
}

// ErrLeaseExpired indicates the lease has aged out.
type ErrLeaseExpired struct {
	ID        string
	ExpiresAt time.Time
}

func (e ErrLeaseExpired) Error() string {
	return fmt.Sprintf("kms_lease: lease %s expired at %v", e.ID, e.ExpiresAt)
}

// ── in-memory LeaseStore ───────────────────────────────────────────────────

// InMemoryStore tracks (idempotency_key → first_use_time). Concurrency-safe.
type InMemoryStore struct {
	mu   sync.Mutex
	seen map[string]time.Time
}

// NewInMemoryStore builds an InMemoryStore.
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{seen: map[string]time.Time{}}
}

// Consume implements LeaseStore.
func (s *InMemoryStore) Consume(_ context.Context, leaseID, idem string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if t, ok := s.seen[idem]; ok {
		return ErrLeaseConsumed{ID: leaseID, Idem: idem + " at " + t.Format(time.RFC3339Nano)}
	}
	s.seen[idem] = time.Now().UTC()
	return nil
}
