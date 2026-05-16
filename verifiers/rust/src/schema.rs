//! Schema mirror for `apps/verifier/pkg/testreport/testreport.go`.
//!
//! Field naming, ordering, and `omitempty` semantics are reproduced
//! exactly: all JSON keys are `snake_case` and every optional field that
//! Go marks with `,omitempty` is modelled as `Option<T>` plus
//! `#[serde(skip_serializing_if = "Option::is_none")]` (or, for
//! collections, `Vec::is_empty` / `default+skip_serializing_if`). The
//! dispatcher unmarshals the resulting JSON straight into the Go struct,
//! so any drift here will surface as a hard schema error in
//! `processpool.parseRunnerOutput`.

#![allow(missing_docs)]

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};

/// Schema contract version. Bumped only on breaking change (90-day
/// deprecation per `twin-spec/schemas/README.md`).
pub const SCHEMA_VERSION: &str = "1";

/// In-toto `predicateType` URI for every TestReport.
pub const PREDICATE_TYPE: &str = "https://crucible.dev/TestReport/v1";

/// Reporter identifier baked into emitted reports.
pub const REPORTER_ID: &str = "crucible-verify-rust";
/// Reporter version baked into emitted reports.
pub const REPORTER_VERSION: &str = env!("CARGO_PKG_VERSION");

/// Per-language runner identity (mirrors `testreport.Language`).
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum Language {
    Python,
    Typescript,
    Rust,
    Go,
    Java,
    Swift,
    Polyglot,
}

impl Default for Language {
    fn default() -> Self {
        Self::Rust
    }
}

/// Tier identifier (mirrors `testreport.Tier`).
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum Tier {
    #[serde(rename = "tier_0_mutation")]
    Mutation,
    #[serde(rename = "tier_1_pbt")]
    Pbt,
    #[serde(rename = "tier_2_contract")]
    Contract,
    #[serde(rename = "tier_3_proof")]
    Proof,
    #[serde(rename = "tier_4_honest_ci")]
    HonestCi,
}

impl Tier {
    /// Parse the CLI tier flag (e.g. `"tier_0_mutation"`).
    pub fn parse(s: &str) -> anyhow::Result<Self> {
        match s {
            "tier_0_mutation" => Ok(Self::Mutation),
            "tier_1_pbt" => Ok(Self::Pbt),
            "tier_2_contract" => Ok(Self::Contract),
            "tier_3_proof" => Ok(Self::Proof),
            "tier_4_honest_ci" => Ok(Self::HonestCi),
            other => anyhow::bail!("unknown tier {other:?}"),
        }
    }

    /// Wire string identical to the Go constant.
    pub fn as_str(self) -> &'static str {
        match self {
            Self::Mutation => "tier_0_mutation",
            Self::Pbt => "tier_1_pbt",
            Self::Contract => "tier_2_contract",
            Self::Proof => "tier_3_proof",
            Self::HonestCi => "tier_4_honest_ci",
        }
    }
}

/// Runner-local pass/fail signal (mirrors `testreport.Verdict`).
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum Verdict {
    Passed,
    Failed,
    TimedOut,
    ToolUnavailable,
    Skipped,
}

impl Default for Verdict {
    fn default() -> Self {
        Self::Skipped
    }
}

/// Single runner+tier+language report — the wire shape the Go dispatcher
/// unmarshals. Order is best-effort identical to the Go struct (serde
/// preserves declaration order for struct serialisation).
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TestReport {
    pub schema_version: String,
    pub task_id: String,
    pub diff_hash: String,
    pub tier: Tier,
    pub language: Language,
    pub framework: String,
    pub verdict: Verdict,
    pub passed: bool,

    pub started_at: DateTime<Utc>,
    pub finished_at: DateTime<Utc>,
    pub duration_seconds: f64,
    pub wall_clock_budget_seconds: f64,

    #[serde(skip_serializing_if = "Option::is_none")]
    pub mutation: Option<MutationStats>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub pbt: Option<PbtStats>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub contract: Option<ContractStats>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub proof: Option<ProofStats>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub honest_ci: Option<HonestCiStats>,

    #[serde(default, skip_serializing_if = "Vec::is_empty")]
    pub findings: Vec<Finding>,

    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub tool_digest: String,

    pub reporter_id: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub reporter_version: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub reporter_oidc_subject: String,

    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub error: String,
}

impl TestReport {
    /// Build a freshly-stamped report for `(tier, language)` with the
    /// reporter identity already baked in.
    pub fn new(tier: Tier, language: Language, task_id: &str, diff_hash: &str) -> Self {
        let now = Utc::now();
        Self {
            schema_version: SCHEMA_VERSION.to_string(),
            task_id: task_id.to_string(),
            diff_hash: diff_hash.to_string(),
            tier,
            language,
            framework: String::new(),
            verdict: Verdict::Skipped,
            passed: false,
            started_at: now,
            finished_at: now,
            duration_seconds: 0.0,
            wall_clock_budget_seconds: 0.0,
            mutation: None,
            pbt: None,
            contract: None,
            proof: None,
            honest_ci: None,
            findings: Vec::new(),
            tool_digest: String::new(),
            reporter_id: REPORTER_ID.to_string(),
            reporter_version: REPORTER_VERSION.to_string(),
            reporter_oidc_subject: String::new(),
            error: String::new(),
        }
    }

    /// Mark the report as `tool_unavailable` with a human-readable detail.
    pub fn tool_unavailable(mut self, detail: impl Into<String>) -> Self {
        self.verdict = Verdict::ToolUnavailable;
        self.passed = false;
        self.error = detail.into();
        self
    }

    /// Stamp `finished_at` and `duration_seconds` from the supplied start.
    pub fn stamp_finished(&mut self) {
        let now = Utc::now();
        self.finished_at = now;
        let dur = now
            .signed_duration_since(self.started_at)
            .num_milliseconds()
            .max(0) as f64
            / 1000.0;
        self.duration_seconds = dur;
    }
}

/// Tier-0 mutation statistics (cargo-mutants).
#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct MutationStats {
    pub killed: u32,
    pub survived: u32,
    #[serde(default, skip_serializing_if = "is_zero_u32")]
    pub not_covered: u32,
    #[serde(default, skip_serializing_if = "is_zero_u32")]
    pub timeout: u32,
    pub total: u32,
    pub score: f64,
    pub threshold: f64,
    pub diff_scoped: bool,
    #[serde(default, skip_serializing_if = "Vec::is_empty")]
    pub mutated_files: Vec<String>,
    #[serde(default, skip_serializing_if = "Vec::is_empty")]
    pub survived_summary: Vec<SurvivedMutant>,
}

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct SurvivedMutant {
    pub file: String,
    pub line: u32,
    pub mutator: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub original: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub replacement: String,
}

/// Tier-1 property test statistics (proptest, cargo-fuzz).
#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct PbtStats {
    pub iterations: u64,
    pub iterations_min: u64,
    #[serde(default, skip_serializing_if = "Vec::is_empty")]
    pub properties: Vec<String>,
    #[serde(default, skip_serializing_if = "Vec::is_empty")]
    pub counterexamples: Vec<Counterexample>,
    #[serde(default, skip_serializing_if = "is_zero_u64")]
    pub fuzz_corpus_size: u64,
    #[serde(default, skip_serializing_if = "is_zero_u64")]
    pub fuzz_new_seeds: u64,
    #[serde(default, skip_serializing_if = "is_zero_u64")]
    pub fuzz_crashes: u64,
}

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct Counterexample {
    pub property: String,
    pub shrunk: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub seed: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub stack_hint: String,
}

/// Tier-2 contract statistics (schemathesis).
#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct ContractStats {
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub spec_path: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub spec_hash: String,
    #[serde(default, skip_serializing_if = "is_zero_u32")]
    pub stateful_workflows: u32,
    #[serde(default, skip_serializing_if = "Vec::is_empty")]
    pub checks: Vec<String>,
    #[serde(default, skip_serializing_if = "Vec::is_empty")]
    pub violations: Vec<ContractViolation>,
    #[serde(default, skip_serializing_if = "is_zero_u64")]
    pub dst_iterations: u64,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub dst_replay_id: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub dst_failing_schedule: String,
}

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct ContractViolation {
    pub endpoint: String,
    pub method: String,
    pub check: String,
    pub detail: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub reproducer: String,
}

/// Tier-3 proof statistics (Kani).
#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct ProofStats {
    pub prover: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub proof_artifact: String,
    #[serde(default, skip_serializing_if = "is_zero_u32")]
    pub obligations: u32,
    #[serde(default, skip_serializing_if = "is_zero_u32")]
    pub discharged: u32,
    pub timed_out: bool,
    #[serde(default, skip_serializing_if = "is_zero_f64")]
    pub wall_clock_seconds: f64,
    #[serde(default, skip_serializing_if = "std::ops::Not::not")]
    pub cached_partial: bool,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub fallback_tier: String,
    #[serde(default, skip_serializing_if = "std::ops::Not::not")]
    pub codeowner_review_required: bool,
    #[serde(default, skip_serializing_if = "Vec::is_empty")]
    pub unsoundness_hints: Vec<String>,
}

/// Tier-4 honest-CI statistics.
#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct HonestCiStats {
    pub builder_id: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub nix_flake_hash: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub nix_lock_hash: String,
    pub executor_rebuild_hash: String,
    pub verifier_rebuild_hash: String,
    pub bit_identical: bool,
    pub slsa_level: u32,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub in_toto_statement_hash: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub fulcio_cert_hash: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub rekor_uuid: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub witness_attestation: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub tekton_chains_ref: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub diffoscope_report: String,
    pub scrubber_audit_ok: bool,
    #[serde(default, skip_serializing_if = "is_zero_u32")]
    pub scrubber_audit_entries: u32,
}

/// Structured finding — mirrors `testreport.Finding`. The rubric LLM-judge
/// folds these into `VerifierRejection.RejectionReasons`.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Finding {
    pub category: String,
    pub severity: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub file: String,
    #[serde(default, skip_serializing_if = "is_zero_u32")]
    pub line: u32,
    pub detail: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub suggested_fix: String,
}

/// Subset of the `verification.VerificationRequest` Go struct relevant to
/// the per-language Rust runner. We only deserialise the fields the
/// runner consumes; unknown fields are tolerated by `serde(default)` and
/// `serde_json::Value` for the diff body.
#[derive(Debug, Clone, Default, Deserialize)]
pub struct VerificationRequest {
    #[serde(default)]
    pub task_id: String,
    #[serde(default)]
    pub tenant_id: String,
    #[serde(default)]
    pub repo: String,
    #[serde(default)]
    pub base_sha: String,
    #[serde(default)]
    pub diff: Diff,
    #[serde(default)]
    pub test_files: Vec<FileChange>,
    #[serde(default)]
    pub spec_changes: Vec<SpecChange>,
    #[serde(default)]
    pub languages: Vec<String>,
    #[serde(default)]
    pub executor_sandbox_id: String,
    #[serde(default)]
    pub budget: BudgetEnvelope,
}

#[derive(Debug, Clone, Default, Deserialize)]
pub struct Diff {
    #[serde(default)]
    pub files: Vec<FileChange>,
}

#[derive(Debug, Clone, Default, Deserialize)]
pub struct FileChange {
    #[serde(default)]
    pub path: String,
    #[serde(default)]
    pub status: String,
    #[serde(default)]
    pub old_path: String,
    #[serde(default)]
    pub unified_diff: String,
    #[serde(default)]
    pub before_blob_sha: String,
    #[serde(default)]
    pub after_blob_sha: String,
}

#[derive(Debug, Clone, Default, Deserialize)]
pub struct SpecChange {
    #[serde(default)]
    pub path: String,
    #[serde(default)]
    pub kind: String,
    #[serde(default)]
    pub previous_hash: String,
    #[serde(default)]
    pub current_hash: String,
    #[serde(default)]
    pub delta: String,
}

#[derive(Debug, Clone, Default, Deserialize)]
pub struct BudgetEnvelope {
    #[serde(default)]
    pub verifier_cap_usd: f64,
    #[serde(default)]
    pub verifier_spent_usd: f64,
    #[serde(default)]
    pub wall_clock_cap_seconds: u64,
    #[serde(default)]
    pub wall_clock_spent_seconds: u64,
}

// ----- serde helpers -------------------------------------------------------

#[allow(clippy::trivially_copy_pass_by_ref)]
fn is_zero_u32(v: &u32) -> bool {
    *v == 0
}

#[allow(clippy::trivially_copy_pass_by_ref)]
fn is_zero_u64(v: &u64) -> bool {
    *v == 0
}

#[allow(clippy::trivially_copy_pass_by_ref)]
fn is_zero_f64(v: &f64) -> bool {
    *v == 0.0
}
