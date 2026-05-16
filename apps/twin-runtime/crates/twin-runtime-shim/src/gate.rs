//! The policy gate.
//!
//! Owns the twin-vs-real scope resolution and the auto-approve / forward
//! decision for every intercepted destructive proposal. Per the threat
//! model, the gate is **fail-closed**: any classification ambiguity yields
//! [`crate::proposal::Scope::Real`], which routes through the Promotion
//! Contract for human approval.
//!
//! Architectural invariants enforced here:
//!
//! - Real-scoped destructive operations **never** auto-approve, regardless
//!   of [`crucible_sandbox_spec::SyscallShimPolicy::auto_approve_twin_scope`].
//! - The proposal's [`crate::proposal::DestructiveProposal::scope`] is the
//!   final answer; the corpus default is informational only.
//! - Every gate decision is observable — the runtime emits a
//!   `DestructiveProposal/v1` attestation when [`Outcome::Intercept`] or
//!   [`Outcome::ForwardToPromotion`] fires, and a `DestructiveApproval/v1`
//!   attestation when [`Outcome::AutoApprovedTwinScope`] fires.

use crucible_sandbox_spec::SyscallShimPolicy;

use crate::proposal::{DestructiveProposal, Scope, WireScope};
use crate::Outcome;

pub mod scope;

/// The policy gate. Constructed per-sandbox from the shim policy.
pub struct Gate {
    auto_approve_twin: bool,
    /// Hard gate: when false, even `Twin` scope still surfaces an
    /// `Intercept` outcome so the agent must explicitly approve. Used by
    /// adversarial-test runs.
    require_explicit_approval: bool,
}

impl Gate {
    /// Build a gate from a [`SyscallShimPolicy`].
    #[must_use]
    pub fn new(policy: &SyscallShimPolicy) -> Self {
        Self {
            auto_approve_twin: policy.auto_approve_twin_scope,
            require_explicit_approval: policy.gate_mode == "block",
        }
    }

    /// Evaluate a proposal. Returns the [`Outcome`] the runtime will return
    /// to `twin.shell.exec`'s caller.
    #[must_use]
    pub fn evaluate(&self, mut proposal: DestructiveProposal) -> Outcome {
        // Step 1: resolve the scope if path-dependent.
        let resolved = match proposal.corpus_default_scope {
            WireScope::AlwaysTwin => Scope::Twin,
            WireScope::AlwaysReal => Scope::Real,
            WireScope::PathDependent => scope::resolve(&proposal),
        };
        proposal.scope = resolved;
        proposal.refresh_hash();

        // Step 2: route by scope.
        match proposal.scope {
            Scope::Twin => {
                if self.auto_approve_twin && !self.require_explicit_approval {
                    Outcome::AutoApprovedTwinScope(proposal)
                } else {
                    Outcome::Intercept(proposal)
                }
            }
            Scope::Real => {
                // Real-scoped destructives are NEVER auto-approved. The
                // Promotion Contract is the only path. The runtime emits a
                // DestructiveProposal/v1 attestation; the user / approver
                // sees it; HSM-signed approval lands as a separate
                // attestation; only then does the runtime proceed.
                Outcome::ForwardToPromotion(proposal)
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::cmd_parse::{Command, CorpusHit};
    use crate::corpus::{PatternScope, Reversibility};
    use crate::proposal::DestructiveProposal;
    use crucible_sandbox_spec::SyscallShimPolicy;

    fn proposal(default: PatternScope, paths: &[&str]) -> DestructiveProposal {
        let argv = std::iter::once("rm".to_string())
            .chain(std::iter::once("-rf".to_string()))
            .chain(paths.iter().map(|s| s.to_string()))
            .collect();
        let hit = CorpusHit {
            pattern_id: "test-rm",
            reason: "test rm",
            command: Command {
                argv,
                source_offset: 0,
            },
            default_scope: default,
            reversibility: Reversibility::Lossy,
        };
        let cmd_str = format!("rm -rf {}", paths.join(" "));
        DestructiveProposal::from_match("task_x", &cmd_str, &hit)
    }

    #[test]
    fn always_twin_auto_approves_when_policy_allows() {
        let policy = SyscallShimPolicy::default(); // auto_approve_twin_scope=true
        let gate = Gate::new(&policy);
        let p = proposal(PatternScope::AlwaysTwin, &["/work/scratch/x"]);
        let outcome = gate.evaluate(p);
        assert!(matches!(outcome, Outcome::AutoApprovedTwinScope(_)));
        assert!(outcome.allows_execution());
    }

    #[test]
    fn always_real_forwards_to_promotion() {
        let policy = SyscallShimPolicy::default();
        let gate = Gate::new(&policy);
        let p = proposal(PatternScope::AlwaysReal, &["whatever"]);
        let outcome = gate.evaluate(p);
        assert!(matches!(outcome, Outcome::ForwardToPromotion(_)));
        assert!(!outcome.allows_execution());
    }

    #[test]
    fn auto_approve_disabled_still_intercepts_twin_scope() {
        let mut policy = SyscallShimPolicy::default();
        policy.auto_approve_twin_scope = false;
        let gate = Gate::new(&policy);
        let p = proposal(PatternScope::AlwaysTwin, &["/work/scratch/x"]);
        let outcome = gate.evaluate(p);
        assert!(matches!(outcome, Outcome::Intercept(_)));
        assert!(!outcome.allows_execution());
    }

    #[test]
    fn block_mode_forces_explicit_approval_even_for_twin() {
        let mut policy = SyscallShimPolicy::default();
        policy.gate_mode = "block".into();
        let gate = Gate::new(&policy);
        let p = proposal(PatternScope::AlwaysTwin, &["/work/scratch/x"]);
        let outcome = gate.evaluate(p);
        assert!(matches!(outcome, Outcome::Intercept(_)));
    }

    #[test]
    fn path_dependent_under_scratch_resolves_twin() {
        let policy = SyscallShimPolicy::default();
        let gate = Gate::new(&policy);
        let p = proposal(PatternScope::PathDependent, &["/work/scratch/build"]);
        let outcome = gate.evaluate(p);
        assert!(matches!(outcome, Outcome::AutoApprovedTwinScope(_)));
    }

    #[test]
    fn path_dependent_outside_scratch_resolves_real() {
        let policy = SyscallShimPolicy::default();
        let gate = Gate::new(&policy);
        let p = proposal(PatternScope::PathDependent, &["/etc/passwd"]);
        let outcome = gate.evaluate(p);
        assert!(matches!(outcome, Outcome::ForwardToPromotion(_)));
    }

    #[test]
    fn ambiguous_path_fails_closed_to_real() {
        // No path arg at all — gate cannot prove twin scope.
        let policy = SyscallShimPolicy::default();
        let gate = Gate::new(&policy);
        let p = proposal(PatternScope::PathDependent, &[]);
        let outcome = gate.evaluate(p);
        assert!(matches!(outcome, Outcome::ForwardToPromotion(_)));
    }
}
