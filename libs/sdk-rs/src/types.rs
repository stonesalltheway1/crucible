use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum Action {
    Add,
    Modify,
    Delete,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum Complexity {
    Trivial,
    Standard,
    Complex,
    Critical,
    Modernization,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum Reversibility {
    Trivial,
    Snapshot,
    Lossy,
    Irreversible,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum TaskStatus {
    Received,
    Planning,
    AwaitingApproval,
    Approved,
    Rejected,
    Executing,
    Verifying,
    Promoting,
    Landed,
    RolledBack,
    BudgetExceeded,
    RetryLimitExceeded,
    WallClockExceeded,
    Failed,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct Glob {
    pub pattern: String,
}

#[derive(Debug, Clone, PartialEq, Default, Serialize, Deserialize)]
pub struct ScopeFilter {
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub repo: Option<String>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub file_glob: Option<String>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub category: Option<String>,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
#[serde(untagged)]
pub enum Scope {
    All(StringAll),
    Filter(ScopeFilter),
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct StringAll(#[serde(deserialize_with = "deser_all_literal")] pub ());

fn deser_all_literal<'de, D>(d: D) -> Result<(), D::Error>
where
    D: serde::Deserializer<'de>,
{
    let s = String::deserialize(d)?;
    if s != "all" {
        return Err(serde::de::Error::custom(format!(
            "expected \"all\", got {:?}",
            s
        )));
    }
    Ok(())
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct FileChange {
    pub path: String,
    pub action: Action,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub content: Option<String>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub content_sha256: Option<String>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub size_bytes: Option<u64>,
}

#[derive(Debug, Clone, PartialEq, Default, Serialize, Deserialize)]
pub struct Diff {
    #[serde(default)]
    pub files: Vec<FileChange>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub base_sha: Option<String>,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct BlastRadius {
    pub affected_resources: Vec<String>,
    pub reversibility: Reversibility,
    pub impact_score: f64,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct PlanStep {
    pub ordinal: u32,
    pub description: String,
    #[serde(default = "default_retry_budget")]
    pub retry_budget: u32,
    #[serde(default)]
    pub retries_used: u32,
}

fn default_retry_budget() -> u32 {
    3
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct Plan {
    pub task_id: String,
    pub description: String,
    pub steps: Vec<PlanStep>,
    pub estimated_cost_usd: f64,
    pub estimated_duration_min: u32,
    pub files_to_touch: Vec<String>,
    pub db_migrations: u32,
    pub external_effects: Vec<ExternalEffect>,
    pub top_risks: Vec<Risk>,
    pub retry_budget_per_step: u32,
    pub wall_clock_budget_min: u32,
    pub complexity: Complexity,
    pub plan_hash: String,
    pub built_at: DateTime<Utc>,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct ExternalEffect {
    pub service: String,
    pub endpoints: Vec<String>,
    pub live: bool,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct Risk {
    pub description: String,
    pub impact: String,
}

#[derive(Debug, Clone, PartialEq, Default, Serialize, Deserialize)]
pub struct Budget {
    #[serde(default)]
    pub spent_usd: f64,
    pub cap_usd: f64,
    #[serde(default)]
    pub steps_used: u32,
    #[serde(default)]
    pub steps_cap: u32,
    #[serde(default)]
    pub wall_clock_used_seconds: u64,
    #[serde(default)]
    pub wall_clock_cap_seconds: u64,
    #[serde(default)]
    pub retries_used: u32,
    #[serde(default)]
    pub retry_cap: u32,
}

pub mod predicates {
    pub const WRITE: &str = "https://crucible.dev/WriteAttestation/v1";
    pub const MIGRATION: &str = "https://crucible.dev/MigrationAttestation/v1";
    pub const SERVICE_CALL: &str = "https://crucible.dev/ServiceCallAttestation/v1";
    pub const DESTRUCTIVE_PROPOSAL: &str = "https://crucible.dev/DestructiveProposal/v1";
    pub const DESTRUCTIVE_APPROVAL: &str = "https://crucible.dev/DestructiveApproval/v1";
    pub const TEST_REPORT: &str = "https://crucible.dev/TestReport/v1";
    pub const VERIFIER_APPROVAL: &str = "https://crucible.dev/VerifierApproval/v1";
    pub const VERIFIER_REJECTION: &str = "https://crucible.dev/VerifierRejection/v1";
    pub const PLAN_PROPOSAL: &str = "https://crucible.dev/PlanProposal/v1";
    pub const PLAN_APPROVAL: &str = "https://crucible.dev/PlanApproval/v1";
    pub const PROMOTION_BUNDLE: &str = "https://crucible.dev/PromotionBundle/v1";
    pub const PROMOTION_APPROVAL: &str = "https://crucible.dev/PromotionApproval/v1";
    pub const PROMOTION_OUTCOME: &str = "https://crucible.dev/PromotionOutcome/v1";
    pub const MEMORY_WRITE: &str = "https://crucible.dev/MemoryWrite/v1";

    pub const ALL: [&str; 14] = [
        WRITE,
        MIGRATION,
        SERVICE_CALL,
        DESTRUCTIVE_PROPOSAL,
        DESTRUCTIVE_APPROVAL,
        TEST_REPORT,
        VERIFIER_APPROVAL,
        VERIFIER_REJECTION,
        PLAN_PROPOSAL,
        PLAN_APPROVAL,
        PROMOTION_BUNDLE,
        PROMOTION_APPROVAL,
        PROMOTION_OUTCOME,
        MEMORY_WRITE,
    ];
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn predicate_list_has_14_entries() {
        assert_eq!(predicates::ALL.len(), 14);
        for uri in predicates::ALL {
            assert!(uri.starts_with("https://crucible.dev/"));
            assert!(uri.ends_with("/v1"));
        }
    }

    #[test]
    fn budget_default_is_zero() {
        let b = Budget::default();
        assert_eq!(b.spent_usd, 0.0);
        assert_eq!(b.cap_usd, 0.0);
        assert_eq!(b.retry_cap, 0);
    }

    #[test]
    fn plan_serializes_round_trip() {
        let p = Plan {
            task_id: "task_01H".into(),
            description: "test".into(),
            steps: vec![],
            estimated_cost_usd: 1.0,
            estimated_duration_min: 10,
            files_to_touch: vec![],
            db_migrations: 0,
            external_effects: vec![],
            top_risks: vec![],
            retry_budget_per_step: 3,
            wall_clock_budget_min: 60,
            complexity: Complexity::Standard,
            plan_hash: "0".repeat(64),
            built_at: Utc::now(),
        };
        let s = serde_json::to_string(&p).unwrap();
        let p2: Plan = serde_json::from_str(&s).unwrap();
        assert_eq!(p.task_id, p2.task_id);
        assert_eq!(p.complexity, p2.complexity);
    }
}
