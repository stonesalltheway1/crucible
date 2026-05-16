package cruciblev1

import (
	"encoding/json"
	"time"
)

// Predicate-type URIs. Every Crucible in-toto attestation carries one of these
// in the InTotoStatement.PredicateType field. JSON Schemas for each live under
// libs/twin-spec/schemas/.
const (
	PredicateWriteAttestation             = "https://crucible.dev/WriteAttestation/v1"
	PredicateMigrationAttestation         = "https://crucible.dev/MigrationAttestation/v1"
	PredicateServiceCallAttestation       = "https://crucible.dev/ServiceCallAttestation/v1"
	PredicateDestructiveProposal          = "https://crucible.dev/DestructiveProposal/v1"
	PredicateDestructiveApproval          = "https://crucible.dev/DestructiveApproval/v1"
	PredicateTestReport                   = "https://crucible.dev/TestReport/v1"
	PredicateVerifierApproval             = "https://crucible.dev/VerifierApproval/v1"
	PredicateVerifierRejection            = "https://crucible.dev/VerifierRejection/v1"
	PredicatePlanProposal                 = "https://crucible.dev/PlanProposal/v1"
	PredicatePlanApproval                 = "https://crucible.dev/PlanApproval/v1"
	PredicatePromotionBundle              = "https://crucible.dev/PromotionBundle/v1"
	PredicatePromotionApproval            = "https://crucible.dev/PromotionApproval/v1"
	PredicatePromotionOutcome             = "https://crucible.dev/PromotionOutcome/v1"
	PredicateMemoryWrite                  = "https://crucible.dev/MemoryWrite/v1"
	PredicateInTotoStatementType          = "https://in-toto.io/Statement/v1"
	PredicateDsseEnvelopePayloadType      = "application/vnd.in-toto+json"
)

// AllPredicateTypes is the complete set of predicate-type URIs Crucible emits.
// Used by tests and the schema-validator to assert no drift between proto,
// hand-rolled types, and JSON Schemas.
var AllPredicateTypes = []string{
	PredicateWriteAttestation,
	PredicateMigrationAttestation,
	PredicateServiceCallAttestation,
	PredicateDestructiveProposal,
	PredicateDestructiveApproval,
	PredicateTestReport,
	PredicateVerifierApproval,
	PredicateVerifierRejection,
	PredicatePlanProposal,
	PredicatePlanApproval,
	PredicatePromotionBundle,
	PredicatePromotionApproval,
	PredicatePromotionOutcome,
	PredicateMemoryWrite,
}

// ── in-toto envelope shapes ────────────────────────────────────────────────

type StatementSubject struct {
	Name   string            `json:"name"`
	Digest map[string]string `json:"digest"`
}

type InTotoStatement struct {
	Type          string             `json:"_type"`
	Subject       []StatementSubject `json:"subject"`
	PredicateType string             `json:"predicateType"`
	Predicate     json.RawMessage    `json:"predicate"`
}

type DsseSignature struct {
	KeyID string `json:"keyid"`
	Sig   string `json:"sig"`
	Cert  string `json:"cert,omitempty"`
}

type DsseEnvelope struct {
	PayloadType string          `json:"payloadType"`
	Payload     string          `json:"payload"`
	Signatures  []DsseSignature `json:"signatures"`
}

// RekorEntry is the receipt returned by the publisher (real Rekor or local
// journal). When LocalJournalFallback is true, UUID + URL refer to the local
// hash-chained journal entry, not a Sigstore Rekor entry.
type RekorEntry struct {
	UUID                  string `json:"uuid"`
	LogIndex              string `json:"log_index"`
	LogID                 string `json:"log_id"`
	IntegratedTime        string `json:"integrated_time"`
	URL                   string `json:"url"`
	LocalJournalFallback  bool   `json:"local_journal_fallback"`
}

// ── 14 predicate payloads ──────────────────────────────────────────────────

type WriteAttestation struct {
	TaskID            string    `json:"task_id"`
	StepID            string    `json:"step_id,omitempty"`
	TenantID          string    `json:"tenant_id"`
	Repo              string    `json:"repo"`
	BaseSha           string    `json:"base_sha"`
	Path              string    `json:"path"`
	Action            Action    `json:"action"`
	ContentSha256     string    `json:"content_sha256"`
	SizeBytes         uint64    `json:"size_bytes"`
	Timestamp         time.Time `json:"timestamp"`
	AgentOidcSubject  string    `json:"agent_oidc_subject"`
}

type SchemaDiff struct {
	AddedTables    []string `json:"added_tables"`
	ModifiedTables []string `json:"modified_tables"`
	DroppedTables  []string `json:"dropped_tables"`
	AddedColumns   []string `json:"added_columns"`
	DestructiveDDL bool     `json:"destructive_ddl"`
}

type MigrationAttestation struct {
	TaskID            string            `json:"task_id"`
	TenantID          string            `json:"tenant_id"`
	MigrationFile     string            `json:"migration_file"`
	MigrationSha256   string            `json:"migration_sha256"`
	SchemaDiff        SchemaDiff        `json:"schema_diff"`
	RowCountChange    map[string]string `json:"row_count_change,omitempty"`
	AppliedAt         time.Time         `json:"applied_at"`
	NeonBranchID      string            `json:"neon_branch_id,omitempty"`
	AgentOidcSubject  string            `json:"agent_oidc_subject"`
}

type ServiceCallAttestation struct {
	TaskID            string   `json:"task_id"`
	TenantID          string   `json:"tenant_id"`
	Service           string   `json:"service"`
	Endpoint          string   `json:"endpoint"`
	Method            string   `json:"method"`
	RequestHash       string   `json:"request_hash"`
	ResponseHash      string   `json:"response_hash"`
	TapeDisposition   string   `json:"tape_disposition"`
	XCrucibleTape     string   `json:"x_crucible_tape,omitempty"`
	DurationMs        uint64   `json:"duration_ms"`
	SecretsUsed       []string `json:"secrets_used,omitempty"`
	AgentOidcSubject  string   `json:"agent_oidc_subject"`
}

type DestructiveProposalAttestation struct {
	TaskID             string      `json:"task_id"`
	TenantID           string      `json:"tenant_id"`
	Command            string      `json:"command"`
	Scope              string      `json:"scope"` // "twin" | "real"
	Justification      string      `json:"justification,omitempty"`
	BlastRadius        BlastRadius `json:"blast_radius"`
	InterceptedAtLayer string      `json:"intercepted_at_layer"`
	AgentOidcSubject   string      `json:"agent_oidc_subject"`
}

type DestructiveApprovalAttestation struct {
	ProposalAttestation      string    `json:"proposal_attestation"`
	ApprovalKind             string    `json:"approval_kind"` // "auto-twin" | "human-real"
	ApproverOidcSubject      string    `json:"approver_oidc_subject"`
	ApprovedAt               time.Time `json:"approved_at"`
	ApprovalAttestationID    string    `json:"approval_attestation_id,omitempty"`
}

type TestReportStats struct {
	Killed          uint32   `json:"killed,omitempty"`
	Survived        uint32   `json:"survived,omitempty"`
	Score           float64  `json:"score,omitempty"`
	Iterations      uint32   `json:"iterations,omitempty"`
	Counterexamples []string `json:"counterexamples,omitempty"`
}

type TestKind string

const (
	TestKindTier0Mutation   TestKind = "tier_0_mutation"
	TestKindTier1PBT        TestKind = "tier_1_pbt"
	TestKindTier2Contract   TestKind = "tier_2_contract"
	TestKindTier3Proof      TestKind = "tier_3_proof"
	TestKindTier4HonestCI   TestKind = "tier_4_honest_ci"
	TestKindProjectNative   TestKind = "project_native"
)

type TestReportAttestation struct {
	TaskID              string          `json:"task_id"`
	TestKind            TestKind        `json:"test_kind"`
	Framework           string          `json:"framework"`
	Passed              bool            `json:"passed"`
	Stats               TestReportStats `json:"stats"`
	DurationSeconds     float64         `json:"duration_seconds"`
	VerifierModel       string          `json:"verifier_model,omitempty"`
	VerifierOidcSubject string          `json:"verifier_oidc_subject"`
}

type PlanProposalAttestation struct {
	TaskID                string     `json:"task_id"`
	TenantID              string     `json:"tenant_id"`
	PlanHash              string     `json:"plan_hash"`
	EstimatedCostUsd      float64    `json:"estimated_cost_usd"`
	EstimatedDurationMin  uint32     `json:"estimated_duration_min"`
	Complexity            Complexity `json:"complexity"`
	StepCount             uint32     `json:"step_count"`
	BuiltByOidc           string     `json:"built_by_oidc"`
	BuiltAt               time.Time  `json:"built_at"`
}

type PlanApprovalAttestation struct {
	TaskID            string    `json:"task_id"`
	PlanHash          string    `json:"plan_hash"`
	EstimatedCostUsd  float64   `json:"estimated_cost_usd,omitempty"`
	ApprovedByOidc    string    `json:"approved_by_oidc"`
	ApprovedAt        time.Time `json:"approved_at"`
}

type PromotionBundleAttestation struct {
	TaskID                       string           `json:"task_id"`
	DiffHash                     string           `json:"diff_hash"`
	VerifierApprovalAttestation  string           `json:"verifier_approval_attestation"`
	FilesChanged                 []FileChange     `json:"files_changed"`
	BuildProvenanceAttestation   string           `json:"build_provenance_attestation,omitempty"`
	RebuildHash                  string           `json:"rebuild_hash,omitempty"`
	BlastRadius                  BlastRadius      `json:"blast_radius"`
	SuggestedRollout             SuggestedRollout `json:"suggested_rollout"`
	AgentOidcSubject             string           `json:"agent_oidc_subject"`
	SignedAt                     time.Time        `json:"signed_at"`
}

type PromotionApprovalAttestation struct {
	BundleAttestation           string          `json:"bundle_attestation"`
	PolicyDecision              string          `json:"policy_decision"` // "auto-approve" | "human-approved"
	RegoPolicyHash              string          `json:"rego_policy_hash"`
	RegoDecisionDoc             json.RawMessage `json:"rego_decision_doc,omitempty"`
	HumanApproverOidcSubjects   []string        `json:"human_approver_oidc_subjects,omitempty"`
	KmsSigningKeyArn            string          `json:"kms_signing_key_arn,omitempty"`
	LeaseID                     string          `json:"lease_id,omitempty"`
	ApprovedAt                  time.Time       `json:"approved_at"`
}

type PromotionOutcomeStep struct {
	Weight        uint32    `json:"weight"`
	DwellSeconds  uint32    `json:"dwell_seconds"`
	SloCheck      string    `json:"slo_check,omitempty"`
	Timestamp     time.Time `json:"timestamp,omitempty"`
}

type PromotionOutcomeAttestation struct {
	PromotionID         string                  `json:"promotion_id"`
	BundleAttestation   string                  `json:"bundle_attestation"`
	Outcome             string                  `json:"outcome"` // "landed" | "rolled_back" | "approval_timeout" | "policy_denied"
	RolloutSteps        []PromotionOutcomeStep  `json:"rollout_steps,omitempty"`
	FinalState          string                  `json:"final_state,omitempty"`
	RollbackReason      string                  `json:"rollback_reason,omitempty"`
	CompletedAt         time.Time               `json:"completed_at"`
}

type MemoryWriteAttestation struct {
	ConventionID         string      `json:"convention_id"`
	TenantID             string      `json:"tenant_id"`
	Scope                ScopeFilter `json:"scope"`
	RuleNl               string      `json:"rule_nl"`
	Category             string      `json:"category"`
	SourceEvidence       []SourceRef `json:"source_evidence,omitempty"`
	Confidence           float64     `json:"confidence"`
	JudgeScore           float64     `json:"judge_score"`
	WriterOidcSubject    string      `json:"writer_oidc_subject"`
	WrittenAt            time.Time   `json:"written_at"`
}

// VerifierApprovalAttestation is the JSON-serializable form of the verifier's
// signed approval (see VerifierApproval above, which is the in-memory shape).
type VerifierApprovalAttestation = VerifierApproval

// VerifierRejectionAttestation likewise.
type VerifierRejectionAttestation = VerifierRejection
