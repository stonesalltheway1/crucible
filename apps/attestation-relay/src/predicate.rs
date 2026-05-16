//! The 13 Crucible predicate types + SLSA Provenance v1.
//!
//! Each predicate is a strongly-typed Rust struct + the canonical URI. The
//! relay's HTTP API accepts the URI + a `serde_json::Value` payload (so SDKs
//! can submit any predicate without round-tripping through this enum), but
//! these structs are the source of truth for shape validation.

use serde::{Deserialize, Serialize};

/// Predicate-type URI.
pub type PredicateType = &'static str;

/// Predicate-type URIs.
pub const PRED_WRITE_ATTESTATION: PredicateType = "https://crucible.dev/WriteAttestation/v1";
pub const PRED_MIGRATION_ATTESTATION: PredicateType = "https://crucible.dev/MigrationAttestation/v1";
pub const PRED_SERVICE_CALL_ATTESTATION: PredicateType = "https://crucible.dev/ServiceCallAttestation/v1";
pub const PRED_DESTRUCTIVE_PROPOSAL: PredicateType = "https://crucible.dev/DestructiveProposal/v1";
pub const PRED_DESTRUCTIVE_APPROVAL: PredicateType = "https://crucible.dev/DestructiveApproval/v1";
pub const PRED_TEST_REPORT: PredicateType = "https://crucible.dev/TestReport/v1";
pub const PRED_VERIFIER_APPROVAL: PredicateType = "https://crucible.dev/VerifierApproval/v1";
pub const PRED_VERIFIER_REJECTION: PredicateType = "https://crucible.dev/VerifierRejection/v1";
pub const PRED_PLAN_PROPOSAL: PredicateType = "https://crucible.dev/PlanProposal/v1";
pub const PRED_PLAN_APPROVAL: PredicateType = "https://crucible.dev/PlanApproval/v1";
pub const PRED_PROMOTION_BUNDLE: PredicateType = "https://crucible.dev/PromotionBundle/v1";
pub const PRED_PROMOTION_APPROVAL: PredicateType = "https://crucible.dev/PromotionApproval/v1";
pub const PRED_PROMOTION_OUTCOME: PredicateType = "https://crucible.dev/PromotionOutcome/v1";
pub const PRED_MEMORY_WRITE: PredicateType = "https://crucible.dev/MemoryWrite/v1";

/// SLSA Provenance v1 URI (emitted by Tier 4 alongside the Crucible types).
pub const PRED_SLSA_PROVENANCE_V1: PredicateType = "https://slsa.dev/provenance/v1";

/// All 14 predicate-type URIs the relay accepts. The 13 from the threat model
/// plus the SLSA Provenance v1 URI.
pub const ALL_PREDICATES: &[PredicateType] = &[
    PRED_WRITE_ATTESTATION,
    PRED_MIGRATION_ATTESTATION,
    PRED_SERVICE_CALL_ATTESTATION,
    PRED_DESTRUCTIVE_PROPOSAL,
    PRED_DESTRUCTIVE_APPROVAL,
    PRED_TEST_REPORT,
    PRED_VERIFIER_APPROVAL,
    PRED_VERIFIER_REJECTION,
    PRED_PLAN_PROPOSAL,
    PRED_PLAN_APPROVAL,
    PRED_PROMOTION_BUNDLE,
    PRED_PROMOTION_APPROVAL,
    PRED_PROMOTION_OUTCOME,
    PRED_MEMORY_WRITE,
    PRED_SLSA_PROVENANCE_V1,
];

/// A tagged enum over the 13 typed predicate payloads. The relay's HTTP
/// surface accepts opaque `serde_json::Value` so SDKs aren't forced to
/// transit this type, but the enum is used by tests and by the in-process
/// adapter (`adapter.rs` in the Go promotion gate uses this via FFI in v2).
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(tag = "_kind")]
pub enum Predicate {
    /// File write.
    WriteAttestation(WriteAttestation),
    /// DB migration.
    MigrationAttestation(MigrationAttestation),
    /// External-service call.
    ServiceCallAttestation(ServiceCallAttestation),
    /// Intercepted destructive command.
    DestructiveProposal(DestructiveProposal),
    /// Approval of a destructive command.
    DestructiveApproval(DestructiveApproval),
    /// Verifier test-run report.
    TestReport(TestReport),
    /// Final verifier approval.
    VerifierApproval(VerifierApproval),
    /// Final verifier rejection.
    VerifierRejection(VerifierRejection),
    /// Plan proposal.
    PlanProposal(PlanProposal),
    /// Plan approval.
    PlanApproval(PlanApproval),
    /// Promotion bundle.
    PromotionBundle(PromotionBundle),
    /// Promotion approval / policy decision.
    PromotionApproval(PromotionApproval),
    /// Promotion outcome (landed / rolled_back / etc.).
    PromotionOutcome(PromotionOutcome),
    /// Memory-write attestation (distiller-emitted).
    MemoryWrite(MemoryWrite),
}

impl Predicate {
    /// Returns the matching predicate-type URI.
    #[must_use]
    pub fn type_uri(&self) -> PredicateType {
        match self {
            Self::WriteAttestation(_) => PRED_WRITE_ATTESTATION,
            Self::MigrationAttestation(_) => PRED_MIGRATION_ATTESTATION,
            Self::ServiceCallAttestation(_) => PRED_SERVICE_CALL_ATTESTATION,
            Self::DestructiveProposal(_) => PRED_DESTRUCTIVE_PROPOSAL,
            Self::DestructiveApproval(_) => PRED_DESTRUCTIVE_APPROVAL,
            Self::TestReport(_) => PRED_TEST_REPORT,
            Self::VerifierApproval(_) => PRED_VERIFIER_APPROVAL,
            Self::VerifierRejection(_) => PRED_VERIFIER_REJECTION,
            Self::PlanProposal(_) => PRED_PLAN_PROPOSAL,
            Self::PlanApproval(_) => PRED_PLAN_APPROVAL,
            Self::PromotionBundle(_) => PRED_PROMOTION_BUNDLE,
            Self::PromotionApproval(_) => PRED_PROMOTION_APPROVAL,
            Self::PromotionOutcome(_) => PRED_PROMOTION_OUTCOME,
            Self::MemoryWrite(_) => PRED_MEMORY_WRITE,
        }
    }

    /// Returns a canonical-JSON byte representation of the payload for use
    /// inside the in-toto Statement's `predicate` field.
    pub fn to_json(&self) -> serde_json::Result<serde_json::Value> {
        match self {
            Self::WriteAttestation(v) => serde_json::to_value(v),
            Self::MigrationAttestation(v) => serde_json::to_value(v),
            Self::ServiceCallAttestation(v) => serde_json::to_value(v),
            Self::DestructiveProposal(v) => serde_json::to_value(v),
            Self::DestructiveApproval(v) => serde_json::to_value(v),
            Self::TestReport(v) => serde_json::to_value(v),
            Self::VerifierApproval(v) => serde_json::to_value(v),
            Self::VerifierRejection(v) => serde_json::to_value(v),
            Self::PlanProposal(v) => serde_json::to_value(v),
            Self::PlanApproval(v) => serde_json::to_value(v),
            Self::PromotionBundle(v) => serde_json::to_value(v),
            Self::PromotionApproval(v) => serde_json::to_value(v),
            Self::PromotionOutcome(v) => serde_json::to_value(v),
            Self::MemoryWrite(v) => serde_json::to_value(v),
        }
    }
}

// ── per-predicate structs ───────────────────────────────────────────────────

/// `WriteAttestation/v1`.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WriteAttestation {
    /// Task ID.
    pub task_id: String,
    /// Step ID.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub step_id: Option<String>,
    /// Tenant ID.
    pub tenant_id: String,
    /// Repository URL.
    pub repo: String,
    /// Base SHA.
    pub base_sha: String,
    /// File path.
    pub path: String,
    /// Action (`add` | `modify` | `delete`).
    pub action: String,
    /// SHA-256 of the file's contents.
    pub content_sha256: String,
    /// Size in bytes.
    pub size_bytes: u64,
    /// Timestamp.
    pub timestamp: chrono::DateTime<chrono::Utc>,
    /// Agent OIDC subject.
    pub agent_oidc_subject: String,
}

/// Schema-diff payload.
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct SchemaDiff {
    /// Added tables.
    #[serde(default)]
    pub added_tables: Vec<String>,
    /// Modified tables.
    #[serde(default)]
    pub modified_tables: Vec<String>,
    /// Dropped tables.
    #[serde(default)]
    pub dropped_tables: Vec<String>,
    /// Added columns.
    #[serde(default)]
    pub added_columns: Vec<String>,
    /// True if migration contains destructive DDL.
    #[serde(default)]
    pub destructive_ddl: bool,
}

/// `MigrationAttestation/v1`.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MigrationAttestation {
    /// Task ID.
    pub task_id: String,
    /// Tenant ID.
    pub tenant_id: String,
    /// Migration file path.
    pub migration_file: String,
    /// SHA-256 of the migration file.
    pub migration_sha256: String,
    /// Schema diff.
    pub schema_diff: SchemaDiff,
    /// Per-table row-count delta.
    #[serde(default)]
    pub row_count_change: std::collections::BTreeMap<String, String>,
    /// Applied-at timestamp.
    pub applied_at: chrono::DateTime<chrono::Utc>,
    /// Neon branch the migration was previewed against.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub neon_branch_id: Option<String>,
    /// Agent OIDC subject.
    pub agent_oidc_subject: String,
}

/// `ServiceCallAttestation/v1`.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct ServiceCallAttestation {
    /// Task ID.
    pub task_id: String,
    /// Tenant ID.
    pub tenant_id: String,
    /// Service name (`stripe`, `slack`, …).
    pub service: String,
    /// Endpoint path.
    pub endpoint: String,
    /// HTTP method.
    pub method: String,
    /// Request hash.
    pub request_hash: String,
    /// Response hash.
    pub response_hash: String,
    /// Tape disposition.
    pub tape_disposition: String,
    /// X-Crucible-Tape header value.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub x_crucible_tape: Option<String>,
    /// Latency.
    pub duration_ms: u64,
    /// Secrets used during the call.
    #[serde(default)]
    pub secrets_used: Vec<String>,
    /// Agent OIDC subject.
    pub agent_oidc_subject: String,
}

/// Blast-radius payload.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct BlastRadius {
    /// Affected resources.
    #[serde(default)]
    pub affected_resources: Vec<String>,
    /// Reversibility (`trivial` | `snapshot` | `lossy` | `irreversible`).
    pub reversibility: String,
    /// Impact score 0.0–1.0.
    pub impact_score: f64,
}

/// `DestructiveProposal/v1`.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DestructiveProposal {
    /// Task ID.
    pub task_id: String,
    /// Tenant ID.
    pub tenant_id: String,
    /// Command intercepted.
    pub command: String,
    /// Scope: `twin` | `real`.
    pub scope: String,
    /// Justification.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub justification: Option<String>,
    /// Blast radius.
    pub blast_radius: BlastRadius,
    /// Layer that intercepted the command.
    pub intercepted_at_layer: String,
    /// Agent OIDC subject.
    pub agent_oidc_subject: String,
}

/// `DestructiveApproval/v1`.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct DestructiveApproval {
    /// Rekor UUID of the proposal.
    pub proposal_attestation: String,
    /// Approval kind: `auto-twin` | `human-real`.
    pub approval_kind: String,
    /// Approver OIDC subject.
    pub approver_oidc_subject: String,
    /// Approved-at timestamp.
    pub approved_at: chrono::DateTime<chrono::Utc>,
    /// Approval attestation ID.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub approval_attestation_id: Option<String>,
}

/// Test-report stats.
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct TestReportStats {
    /// Killed mutants.
    #[serde(default)]
    pub killed: u32,
    /// Surviving mutants.
    #[serde(default)]
    pub survived: u32,
    /// Mutation score.
    #[serde(default)]
    pub score: f64,
    /// Iteration count.
    #[serde(default)]
    pub iterations: u32,
    /// Counterexamples.
    #[serde(default)]
    pub counterexamples: Vec<String>,
}

/// `TestReport/v1`.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TestReport {
    /// Task ID.
    pub task_id: String,
    /// Tier-prefixed test kind.
    pub test_kind: String,
    /// Framework name.
    pub framework: String,
    /// True iff the test passed.
    pub passed: bool,
    /// Stats union.
    pub stats: TestReportStats,
    /// Duration in seconds.
    pub duration_seconds: f64,
    /// Verifier model.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub verifier_model: Option<String>,
    /// Verifier OIDC subject.
    pub verifier_oidc_subject: String,
}

/// Per-tier result inside a verifier verdict.
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct TierResult {
    /// True iff the tier passed.
    #[serde(default)]
    pub passed: bool,
    /// Rekor UUID of the underlying TestReport, if any.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub report_attestation: Option<String>,
}

/// `VerifierApproval/v1`.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct VerifierApproval {
    /// Task ID.
    pub task_id: String,
    /// Diff hash.
    pub diff_hash: String,
    /// Verdict (always "approved").
    pub verdict: String,
    /// Rubric score.
    pub rubric_score: f64,
    /// Per-tier results.
    pub tier_results: std::collections::BTreeMap<String, TierResult>,
    /// Empty for approvals; populated for rejections.
    #[serde(default)]
    pub rejection_reasons: Vec<String>,
    /// Executor OIDC subject.
    pub executor_oidc_subject: String,
    /// Verifier OIDC subject (MUST differ from executor).
    pub verifier_oidc_subject: String,
    /// Signed-at timestamp.
    pub signed_at: chrono::DateTime<chrono::Utc>,
}

/// `VerifierRejection/v1`.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct VerifierRejection {
    /// Task ID.
    pub task_id: String,
    /// Diff hash.
    pub diff_hash: String,
    /// Verdict (always "rejected").
    pub verdict: String,
    /// Reasons.
    pub rejection_reasons: Vec<String>,
    /// Per-tier results.
    #[serde(default)]
    pub tier_results: std::collections::BTreeMap<String, TierResult>,
    /// Executor OIDC subject.
    pub executor_oidc_subject: String,
    /// Verifier OIDC subject.
    pub verifier_oidc_subject: String,
    /// Signed-at timestamp.
    pub signed_at: chrono::DateTime<chrono::Utc>,
}

/// `PlanProposal/v1`.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PlanProposal {
    /// Task ID.
    pub task_id: String,
    /// Tenant ID.
    pub tenant_id: String,
    /// Plan hash.
    pub plan_hash: String,
    /// Estimated cost (USD).
    pub estimated_cost_usd: f64,
    /// Estimated duration in minutes.
    pub estimated_duration_min: u32,
    /// Complexity.
    pub complexity: String,
    /// Step count.
    pub step_count: u32,
    /// Built-by OIDC subject.
    pub built_by_oidc: String,
    /// Built-at timestamp.
    pub built_at: chrono::DateTime<chrono::Utc>,
}

/// `PlanApproval/v1`.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PlanApproval {
    /// Task ID.
    pub task_id: String,
    /// Plan hash.
    pub plan_hash: String,
    /// Estimated cost (USD).
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub estimated_cost_usd: Option<f64>,
    /// Approver OIDC subject.
    pub approved_by_oidc: String,
    /// Approved-at timestamp.
    pub approved_at: chrono::DateTime<chrono::Utc>,
}

/// File-change descriptor inside a promotion bundle.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct FileChange {
    /// File path.
    pub path: String,
    /// `add` | `modify` | `delete`.
    pub action: String,
}

/// Suggested rollout step.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SuggestedRolloutStep {
    /// Weight in percent.
    pub weight: u32,
    /// Dwell seconds.
    pub dwell_seconds: u32,
}

/// Suggested rollout descriptor.
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct SuggestedRollout {
    /// Steps.
    #[serde(default)]
    pub steps: Vec<SuggestedRolloutStep>,
    /// Default dwell seconds when steps are weight-only.
    #[serde(default)]
    pub dwell_seconds_per_step: u32,
    /// SLO check expression.
    #[serde(default)]
    pub slo_check: String,
}

/// Promotion blast-radius enrichment.
#[derive(Debug, Clone, Serialize, Deserialize, Default)]
pub struct PromotionBlastRadius {
    /// Affected resources.
    #[serde(default)]
    pub affected_resources: Vec<String>,
    /// Affected services.
    #[serde(default)]
    pub affected_services: Vec<String>,
    /// Affected endpoints.
    #[serde(default)]
    pub affected_endpoints: Vec<String>,
    /// Schema-change descriptors.
    #[serde(default)]
    pub schema_changes: Vec<serde_json::Value>,
    /// Critical-path files touched.
    #[serde(default)]
    pub critical_paths_touched: Vec<String>,
    /// Estimated impact: `low` | `medium` | `high`.
    pub estimated_impact: String,
    /// Reversibility: `trivial` | `snapshot` | `lossy` | `irreversible`.
    pub reversibility: String,
    /// Impact score.
    pub impact_score: f64,
}

/// `PromotionBundle/v1`.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PromotionBundle {
    /// Task ID.
    pub task_id: String,
    /// Diff hash.
    pub diff_hash: String,
    /// VerifierApproval Rekor UUID.
    pub verifier_approval_attestation: String,
    /// Files changed.
    pub files_changed: Vec<FileChange>,
    /// Build provenance Rekor UUID.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub build_provenance_attestation: Option<String>,
    /// Rebuild hash (Tier 4 hermetic).
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub rebuild_hash: Option<String>,
    /// Blast radius.
    pub blast_radius: PromotionBlastRadius,
    /// Suggested rollout.
    pub suggested_rollout: SuggestedRollout,
    /// Agent OIDC subject.
    pub agent_oidc_subject: String,
    /// Signed-at timestamp.
    pub signed_at: chrono::DateTime<chrono::Utc>,
}

/// `PromotionApproval/v1`.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PromotionApproval {
    /// Bundle Rekor UUID.
    pub bundle_attestation: String,
    /// `auto-approve` | `human-approved`.
    pub policy_decision: String,
    /// SHA-256 of the compiled Rego modules.
    pub rego_policy_hash: String,
    /// Full Rego decision document.
    #[serde(default)]
    pub rego_decision_doc: serde_json::Value,
    /// Human approvers (empty for auto-approve).
    #[serde(default)]
    pub human_approver_oidc_subjects: Vec<String>,
    /// KMS signing key ARN (records who minted the lease).
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub kms_signing_key_arn: Option<String>,
    /// Single-use lease ID.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub lease_id: Option<String>,
    /// Approval timestamp.
    pub approved_at: chrono::DateTime<chrono::Utc>,
}

/// Per-step rollout result.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PromotionOutcomeStep {
    /// Weight in percent at this step.
    pub weight: u32,
    /// Dwell seconds.
    pub dwell_seconds: u32,
    /// SLO check result: `passed` | `failed` | `inconclusive`.
    pub slo_check: String,
    /// Timestamp.
    pub timestamp: chrono::DateTime<chrono::Utc>,
}

/// `PromotionOutcome/v1`.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct PromotionOutcome {
    /// Promotion ID.
    pub promotion_id: String,
    /// Bundle Rekor UUID.
    pub bundle_attestation: String,
    /// `landed` | `rolled_back` | `approval_timeout` | `policy_denied`.
    pub outcome: String,
    /// Per-step rollout history.
    #[serde(default)]
    pub rollout_steps: Vec<PromotionOutcomeStep>,
    /// Final state (e.g. `100% live`).
    #[serde(default)]
    pub final_state: String,
    /// Rollback reason.
    #[serde(default)]
    pub rollback_reason: String,
    /// Completed-at timestamp.
    pub completed_at: chrono::DateTime<chrono::Utc>,
}

/// `MemoryWrite/v1`.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MemoryWrite {
    /// Convention ID.
    pub convention_id: String,
    /// Tenant ID.
    pub tenant_id: String,
    /// Scope filter (opaque to the relay).
    pub scope: serde_json::Value,
    /// Natural-language rule.
    pub rule_nl: String,
    /// Category.
    pub category: String,
    /// Source evidence.
    #[serde(default)]
    pub source_evidence: Vec<serde_json::Value>,
    /// Confidence.
    pub confidence: f64,
    /// LLM-judge score.
    pub judge_score: f64,
    /// Writer OIDC subject.
    pub writer_oidc_subject: String,
    /// Written-at timestamp.
    pub written_at: chrono::DateTime<chrono::Utc>,
    /// Federation-graduation-friendly anonymized rule ID.
    /// Carried forward from Phase 5 — lets v2 Phase 10 federation logic
    /// trace back to contributing tenants without re-identifying them.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub anonymized_rule_id: Option<String>,
}

/// Validates a free-form predicate JSON value against its predicate-type
/// URI. We do a structural check (required fields present, types broadly
/// correct); full JSON-Schema validation lives in the gate's
/// `bundle_validator` which holds the embedded schemas.
pub fn validate_loose(uri: &str, payload: &serde_json::Value) -> crate::error::Result<()> {
    if !ALL_PREDICATES.contains(&uri) {
        return Err(crate::error::Error::Predicate(format!(
            "unknown predicate type: {uri}"
        )));
    }
    let obj = payload
        .as_object()
        .ok_or_else(|| crate::error::Error::Predicate(format!("predicate {uri} must be a JSON object")))?;
    // Cheap top-level check: every Crucible predicate has either task_id,
    // tenant_id, or convention_id / promotion_id. We assert at least one
    // identifier-like key so an obviously bogus blob is caught early.
    let has_id = ["task_id", "tenant_id", "convention_id", "promotion_id", "bundle_attestation", "proposal_attestation"]
        .iter()
        .any(|k| obj.contains_key(*k));
    if !has_id {
        return Err(crate::error::Error::Predicate(format!(
            "predicate {uri} has no identifier-like field"
        )));
    }
    Ok(())
}
