# Crucible default promotion policy.
#
# This is the canonical Phase-6 policy bundle. The shape comes directly from
# docs/01-architecture/promotion-contract.md §"Rego policy structure".
#
# Decision document shape:
#
#   {
#     "allow":        bool,
#     "needs_human":  bool,
#     "reasons":      [string],
#     "require_codeowner": bool,
#     "approver_groups":   [string],     # the cohort the approval_router must
#                                        # collect signatures from
#     "require_n_approvers": number,     # for N-of-M flows; 0 means "any 1"
#     "auto_approve":      bool,         # true when no human is required and
#                                        # allow is true
#     "trace": object                    # explainability for the audit log
#   }
#
# Input shape (PromotionBundle as JSON, plus a few enrichment fields the
# rego_engine adds before evaluation):
#
#   {
#     "task_id":     "task_...",
#     "tenant_id":   "ten_...",
#     "diff_hash":   "0x...",
#     "files_changed": [{"path":"...","action":"add|modify|delete"}],
#     "verifier_approval_attestation": "rekor:...",
#     "build_provenance_attestation":  "rekor:...",
#     "rebuild_hash": "0x...",
#     "blast_radius": {
#       "affected_resources": [...],
#       "affected_services":  [...],     # enrichment
#       "affected_endpoints": [...],     # enrichment
#       "schema_changes":     [...],     # enrichment from MigrationAttestation
#       "critical_paths_touched": [...], # enrichment from critical-path
#                                        # classifier
#       "estimated_impact":   "low|medium|high",
#       "reversibility":      "trivial|snapshot|lossy|irreversible",
#       "impact_score":       number
#     },
#     "suggested_rollout": {...},
#     "tier_results": {
#       "tier_0": {"passed": bool, ...},
#       "tier_1": {"passed": bool, ...},
#       "tier_2": {"passed": bool, ...},
#       "tier_3": {"passed": bool, ...},
#       "tier_4": {"passed": bool, ...}
#     },
#     "agent_oidc_subject": "...",
#     "approvals":   [{"approver_oidc_subject":"...","attestation":"rekor:..."}],
#     "context": {
#       "merge_freeze": bool,
#       "merge_freeze_until": "RFC3339",
#       "merge_freeze_reason": "string",
#       "geo": "us|eu|apac",
#       "is_test_promotion": bool        # never bypasses, only relaxes
#     },
#     "codeowners": {
#       "matched": [{"path_glob":"...","groups":["@team-a"]}]
#     },
#     "tenant_overrides": {
#       "default_approver_groups": ["@platform-team"],
#       "require_n_approvers_for_high_impact": 2,
#       "deny_geo": ["..."],
#       "rules": []                       # tenant rego layered on top
#     }
#   }

package crucible.promotion

import rego.v1

# ─────────────────────────────────────────────────────────────────────────────
# Top-level decision — always populated, never undefined.
# ─────────────────────────────────────────────────────────────────────────────

default decision := {
	"allow": false,
	"needs_human": true,
	"reasons": ["policy: default-deny — no matching allow rule"],
	"require_codeowner": false,
	"approver_groups": ["@platform-team"],
	"require_n_approvers": 1,
	"auto_approve": false,
	"trace": {"path": "default"},
}

# ── HARD DENIES ──────────────────────────────────────────────────────────────
# These run before any allow rule and are non-overridable by tenant policy.

decision := d if {
	some d in deny_decisions
}

deny_decisions contains d if {
	not has_verifier_approval
	d := {
		"allow": false,
		"needs_human": false,
		"reasons": ["policy: missing verifier approval attestation"],
		"require_codeowner": false,
		"approver_groups": [],
		"require_n_approvers": 0,
		"auto_approve": false,
		"trace": {"path": "deny.missing_verifier"},
	}
}

deny_decisions contains d if {
	self_approval_present
	d := {
		"allow": false,
		"needs_human": true,
		"reasons": ["policy: self-approval forbidden — approver_oidc_subject must differ from agent_oidc_subject"],
		"require_codeowner": false,
		"approver_groups": [],
		"require_n_approvers": 0,
		"auto_approve": false,
		"trace": {"path": "deny.self_approval"},
	}
}

deny_decisions contains d if {
	input.context.merge_freeze == true
	not input.context.is_test_promotion
	d := {
		"allow": false,
		"needs_human": false,
		"reasons": [sprintf("policy: merge freeze active until %v (%v)", [object.get(input.context, "merge_freeze_until", "unknown"), object.get(input.context, "merge_freeze_reason", "no reason supplied")])],
		"require_codeowner": false,
		"approver_groups": [],
		"require_n_approvers": 0,
		"auto_approve": false,
		"trace": {"path": "deny.merge_freeze"},
	}
}

deny_decisions contains d if {
	input.blast_radius.estimated_impact != "low"
	not tier4_passed
	d := {
		"allow": false,
		"needs_human": false,
		"reasons": ["policy: Tier 4 reproducible-build attestation required for non-trivial promotions"],
		"require_codeowner": false,
		"approver_groups": [],
		"require_n_approvers": 0,
		"auto_approve": false,
		"trace": {"path": "deny.tier4_missing"},
	}
}

deny_decisions contains d if {
	input.blast_radius.reversibility == "irreversible"
	not has_human_approval
	d := {
		"allow": false,
		"needs_human": true,
		"reasons": ["policy: irreversible change without recorded human approval"],
		"require_codeowner": true,
		"approver_groups": approver_groups_for_irreversible,
		"require_n_approvers": 2,
		"auto_approve": false,
		"trace": {"path": "deny.irreversible_without_human"},
	}
}

deny_decisions contains d if {
	some bad_geo in object.get(input.tenant_overrides, "deny_geo", [])
	bad_geo == input.context.geo
	d := {
		"allow": false,
		"needs_human": false,
		"reasons": [sprintf("policy: tenant denies promotions originating from geo=%v", [bad_geo])],
		"require_codeowner": false,
		"approver_groups": [],
		"require_n_approvers": 0,
		"auto_approve": false,
		"trace": {"path": "deny.geo"},
	}
}

# ── REQUIRE-HUMAN CONDITIONS ─────────────────────────────────────────────────
# A non-empty set means human approval is required to allow.

require_human if has_schema_change
require_human if has_critical_path
require_human if input.blast_radius.estimated_impact == "high"
require_human if input.blast_radius.reversibility == "lossy"
require_human if input.blast_radius.reversibility == "snapshot"
require_human if not tier3_passed_when_required

# ── AUTO-APPROVE — trivial diffs only ───────────────────────────────────────

decision := d if {
	# No hard deny applies (else deny_decisions would have populated).
	count(deny_decisions) == 0
	not require_human
	has_verifier_approval
	tier_zero_passed
	tier_one_passed
	input.blast_radius.estimated_impact == "low"
	input.blast_radius.reversibility == "trivial"
	not has_schema_change
	not has_critical_path
	d := {
		"allow": true,
		"needs_human": false,
		"reasons": [],
		"require_codeowner": false,
		"approver_groups": [],
		"require_n_approvers": 0,
		"auto_approve": true,
		"trace": {"path": "allow.trivial_auto"},
	}
}

# ── REQUIRES HUMAN — schema-change path ─────────────────────────────────────

decision := d if {
	count(deny_decisions) == 0
	has_schema_change
	has_verifier_approval
	tier_four_passed_if_required
	d := {
		"allow": human_approved_for("schema_change"),
		"needs_human": not human_approved_for("schema_change"),
		"reasons": [sprintf("policy: schema change requires %v from %v", [n_approvers("schema_change"), approver_groups_for("schema_change")])],
		"require_codeowner": false,
		"approver_groups": approver_groups_for("schema_change"),
		"require_n_approvers": n_approvers("schema_change"),
		"auto_approve": false,
		"trace": {"path": "human.schema_change"},
	}
}

# ── REQUIRES HUMAN — critical-path path ─────────────────────────────────────

decision := d if {
	count(deny_decisions) == 0
	not has_schema_change
	has_critical_path
	has_verifier_approval
	tier_four_passed_if_required
	d := {
		"allow": human_approved_for("critical_path"),
		"needs_human": not human_approved_for("critical_path"),
		"reasons": [sprintf("policy: critical-path touched %v — CODEOWNER + %v required", [input.blast_radius.critical_paths_touched, approver_groups_for("critical_path")])],
		"require_codeowner": true,
		"approver_groups": approver_groups_for("critical_path"),
		"require_n_approvers": n_approvers("critical_path"),
		"auto_approve": false,
		"trace": {"path": "human.critical_path"},
	}
}

# ── REQUIRES HUMAN — high-impact path (catch-all for non-trivial) ───────────

decision := d if {
	count(deny_decisions) == 0
	not has_schema_change
	not has_critical_path
	require_human
	has_verifier_approval
	tier_four_passed_if_required
	d := {
		"allow": human_approved_for("high_impact"),
		"needs_human": not human_approved_for("high_impact"),
		"reasons": [sprintf("policy: %v impact / %v reversibility requires human approval", [input.blast_radius.estimated_impact, input.blast_radius.reversibility])],
		"require_codeowner": false,
		"approver_groups": approver_groups_for("high_impact"),
		"require_n_approvers": n_approvers("high_impact"),
		"auto_approve": false,
		"trace": {"path": "human.high_impact"},
	}
}

# ─────────────────────────────────────────────────────────────────────────────
# Predicates.
# ─────────────────────────────────────────────────────────────────────────────

has_verifier_approval if {
	input.verifier_approval_attestation
	input.verifier_approval_attestation != ""
}

has_schema_change if count(object.get(input.blast_radius, "schema_changes", [])) > 0

has_critical_path if count(object.get(input.blast_radius, "critical_paths_touched", [])) > 0

tier_zero_passed if input.tier_results.tier_0.passed == true

tier_one_passed if input.tier_results.tier_1.passed == true

tier3_passed_when_required if {
	not has_critical_path
}

tier3_passed_when_required if {
	has_critical_path
	input.tier_results.tier_3.passed == true
}

tier4_passed if input.tier_results.tier_4.passed == true

tier_four_passed_if_required if input.blast_radius.estimated_impact == "low"

tier_four_passed_if_required if tier4_passed

# ── self-approval — agent OIDC must differ from every approver OIDC ─────────

self_approval_present if {
	some a in object.get(input, "approvals", [])
	a.approver_oidc_subject == input.agent_oidc_subject
}

# ── human approval bookkeeping ──────────────────────────────────────────────

human_approved_for(category) if {
	count(valid_approvals) >= n_approvers(category)
}

valid_approvals contains a if {
	some a in object.get(input, "approvals", [])
	a.approver_oidc_subject != input.agent_oidc_subject
	a.attestation
	a.attestation != ""
}

has_human_approval if count(valid_approvals) > 0

# ── approver group selection ────────────────────────────────────────────────

approver_groups_for(category) := groups if {
	category == "schema_change"
	groups := object.get(input.tenant_overrides, "schema_approver_groups", ["@dba-team", "@platform-team"])
}

approver_groups_for(category) := groups if {
	category == "critical_path"
	groups := codeowner_groups
}

approver_groups_for(category) := groups if {
	category == "high_impact"
	groups := object.get(input.tenant_overrides, "high_impact_approver_groups", ["@on-call", "@eng-leadership"])
}

approver_groups_for_irreversible := groups if {
	groups := object.get(input.tenant_overrides, "irreversible_approver_groups", ["@cto", "@eng-leadership"])
}

codeowner_groups := groups if {
	matched := object.get(input.codeowners, "matched", [])
	count(matched) > 0
	groups := union_of_groups(matched)
}

codeowner_groups := groups if {
	matched := object.get(input.codeowners, "matched", [])
	count(matched) == 0
	groups := object.get(input.tenant_overrides, "default_approver_groups", ["@platform-team"])
}

union_of_groups(matched) := groups if {
	groups := [g | some m in matched; some g in m.groups]
}

# ── N-of-M ──────────────────────────────────────────────────────────────────

n_approvers(category) := n if {
	category == "schema_change"
	n := object.get(input.tenant_overrides, "require_n_approvers_for_schema", 1)
}

n_approvers(category) := n if {
	category == "critical_path"
	n := object.get(input.tenant_overrides, "require_n_approvers_for_critical", 2)
}

n_approvers(category) := n if {
	category == "high_impact"
	n := object.get(input.tenant_overrides, "require_n_approvers_for_high_impact", 2)
}
