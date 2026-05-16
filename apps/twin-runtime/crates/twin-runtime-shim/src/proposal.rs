//! The typed [`DestructiveProposal`] — the result of intercepting a
//! destructive operation. Mirrors the `DestructiveProposal/v1` predicate
//! schema from `docs/03-sdk/attestation-formats.md`.

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};

use crate::corpus::{PatternScope, Reversibility};
use crate::cmd_parse::CorpusHit;

/// Layer at which the destructive intent was caught.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "kebab-case")]
pub enum InterceptLayer {
    /// Layer 1 — userspace command-line parse.
    CmdLineParse,
    /// Layer 2a — seccomp-unotify supervisor decision.
    SeccompUnotify,
    /// Layer 2b — BPF LSM hook returned `-EPERM`.
    BpfLsm,
    /// Layer 3 — Tetragon TracingPolicy `NotifyEnforcer`.
    Tetragon,
}

/// Scope as resolved by [`crate::gate::scope`] (not the corpus default).
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "kebab-case")]
pub enum Scope {
    /// Effects contained inside the twin (overlayfs upper, Neon branch,
    /// Hoverfly tape). Gate auto-approves; effects are reversible via
    /// `umount` / `DELETE /branches`.
    Twin,
    /// Effects reach real systems. Gate forwards to the Promotion Contract
    /// for HSM-signed approval. NEVER auto-approve.
    Real,
}

/// Blast radius estimate. Mirrors the proto type with serde for the
/// attestation pipeline.
#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub struct BlastRadius {
    /// Resources the operation affects (best-effort).
    pub affected_resources: Vec<String>,
    /// How reversible the operation is.
    pub reversibility: ReversibilityWire,
    /// Impact score in `[0.0, 1.0]`. Used for prioritisation in dashboards.
    pub impact_score: f64,
}

/// Wire-format reversibility — kebab-cased so the attestation matches the
/// proto enum's stringified form. We don't reuse [`Reversibility`] directly
/// because we don't want serde rendering Rust-y `"Trivial"` casing in the
/// attestation envelope.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "kebab-case")]
pub enum ReversibilityWire {
    /// Trivial revert (e.g. git revert).
    Trivial,
    /// Restorable from snapshot.
    Snapshot,
    /// Partial recovery possible.
    Lossy,
    /// Gone forever.
    Irreversible,
}

impl From<Reversibility> for ReversibilityWire {
    fn from(r: Reversibility) -> Self {
        match r {
            Reversibility::Trivial => Self::Trivial,
            Reversibility::Snapshot => Self::Snapshot,
            Reversibility::Lossy => Self::Lossy,
            Reversibility::Irreversible => Self::Irreversible,
        }
    }
}

/// The full typed proposal. Serialised to the in-toto predicate when
/// emitted to the attestation pipeline.
#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct DestructiveProposal {
    /// Task the proposal belongs to.
    pub task_id: String,
    /// Original full input to `twin.shell.exec`.
    pub command: String,
    /// Pattern semantic id from the corpus.
    pub pattern_id: String,
    /// Human-readable reason from the corpus.
    pub reason: String,
    /// Layer at which the intent was caught.
    pub intercepted_at_layer: InterceptLayer,
    /// Scope resolved by the gate. May be `Twin` or `Real`; the corpus's
    /// `default_scope` is the input to the resolver but not the answer.
    pub scope: Scope,
    /// Default corpus scope (informational; not authoritative).
    pub corpus_default_scope: WireScope,
    /// Blast-radius estimate.
    pub blast_radius: BlastRadius,
    /// Wall-clock timestamp.
    pub proposed_at: DateTime<Utc>,
    /// SHA-256 of the canonical proposal JSON (excluding the hash itself).
    /// Used to bind the attestation to the proposal content.
    pub content_hash: String,
}

/// Wire-format scope hint — kebab-cased like the corpus default-scope field.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "kebab-case")]
pub enum WireScope {
    /// Always twin.
    AlwaysTwin,
    /// Always real.
    AlwaysReal,
    /// Path-dependent.
    PathDependent,
}

impl From<PatternScope> for WireScope {
    fn from(s: PatternScope) -> Self {
        match s {
            PatternScope::AlwaysTwin => Self::AlwaysTwin,
            PatternScope::AlwaysReal => Self::AlwaysReal,
            PatternScope::PathDependent => Self::PathDependent,
        }
    }
}

impl DestructiveProposal {
    /// Build a proposal from a Layer-1 corpus hit. The scope is provisionally
    /// set to the corpus default; the gate may upgrade it before the
    /// proposal is finalised.
    #[must_use]
    pub fn from_match(task_id: &str, command: &str, hit: &CorpusHit) -> Self {
        let provisional_scope = match hit.default_scope {
            PatternScope::AlwaysTwin => Scope::Twin,
            PatternScope::AlwaysReal | PatternScope::PathDependent => Scope::Real,
            // `PathDependent` defaults to Real for fail-closed behaviour; the
            // gate downgrades to Twin only when scope::resolve produces a
            // confident twin classification.
        };
        let affected: Vec<String> = hit
            .command
            .args()
            .iter()
            .filter(|a| !a.starts_with('-'))
            .cloned()
            .collect();
        let impact = match hit.reversibility {
            Reversibility::Trivial => 0.1,
            Reversibility::Snapshot => 0.3,
            Reversibility::Lossy => 0.7,
            Reversibility::Irreversible => 1.0,
        };
        let mut proposal = Self {
            task_id: task_id.to_string(),
            command: command.to_string(),
            pattern_id: hit.pattern_id.to_string(),
            reason: hit.reason.to_string(),
            intercepted_at_layer: InterceptLayer::CmdLineParse,
            scope: provisional_scope,
            corpus_default_scope: hit.default_scope.into(),
            blast_radius: BlastRadius {
                affected_resources: affected,
                reversibility: hit.reversibility.into(),
                impact_score: impact,
            },
            proposed_at: Utc::now(),
            content_hash: String::new(),
        };
        proposal.content_hash = proposal.compute_hash();
        proposal
    }

    /// Recompute the content-hash. Used by the gate after a scope upgrade.
    pub fn refresh_hash(&mut self) {
        self.content_hash = self.compute_hash();
    }

    fn compute_hash(&self) -> String {
        let mut clone = self.clone();
        clone.content_hash.clear();
        let bytes = serde_json::to_vec(&clone).expect("DestructiveProposal is serde_json-clean");
        let mut hasher = Sha256::new();
        hasher.update(&bytes);
        hex::encode(hasher.finalize())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::cmd_parse::{Command, CorpusHit};
    use crate::corpus::{PatternScope, Reversibility};

    fn fake_hit() -> CorpusHit {
        CorpusHit {
            pattern_id: "rm-recursive",
            reason: "rm with recursive/force flag",
            command: Command {
                argv: vec!["rm".into(), "-rf".into(), "/tmp/x".into()],
                source_offset: 0,
            },
            default_scope: PatternScope::PathDependent,
            reversibility: Reversibility::Lossy,
        }
    }

    #[test]
    fn proposal_from_match_records_args_excluding_flags() {
        let hit = fake_hit();
        let p = DestructiveProposal::from_match("task_x", "rm -rf /tmp/x", &hit);
        assert_eq!(p.task_id, "task_x");
        assert_eq!(p.pattern_id, "rm-recursive");
        assert_eq!(p.blast_radius.affected_resources, vec!["/tmp/x"]);
        assert_eq!(p.intercepted_at_layer, InterceptLayer::CmdLineParse);
        // PathDependent defaults to Real (fail-closed).
        assert_eq!(p.scope, Scope::Real);
        assert!(!p.content_hash.is_empty());
    }

    #[test]
    fn proposal_hash_changes_with_scope_change() {
        let hit = fake_hit();
        let mut p = DestructiveProposal::from_match("task_x", "rm -rf /tmp/x", &hit);
        let h1 = p.content_hash.clone();
        p.scope = Scope::Twin;
        p.refresh_hash();
        assert_ne!(h1, p.content_hash);
    }

    #[test]
    fn impact_scales_with_reversibility() {
        let mut hit = fake_hit();
        hit.reversibility = Reversibility::Trivial;
        let p = DestructiveProposal::from_match("t", "x", &hit);
        assert!((p.blast_radius.impact_score - 0.1).abs() < f64::EPSILON);

        hit.reversibility = Reversibility::Irreversible;
        let p = DestructiveProposal::from_match("t", "x", &hit);
        assert!((p.blast_radius.impact_score - 1.0).abs() < f64::EPSILON);
    }
}
