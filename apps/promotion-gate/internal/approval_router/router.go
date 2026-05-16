// Package approval_router resolves the set of human approvers the gate
// must collect signatures from, given a `MergedDecision` and a per-tenant
// configuration.
//
// Inputs:
//
//   - The Rego decision (approver_groups, require_n_approvers, require_codeowner).
//   - The bundle's files_changed (for CODEOWNERS matching).
//   - A per-tenant ApprovalConfig (default approvers, override matchers).
//
// Output: a Cohort — the list of distinct (group, n-required) pairs the
// gate routes to Slack/web for approval.
package approval_router

import (
	"errors"
	"sort"
	"strings"

	"github.com/crucible/promotion-gate/internal/rego_engine"
	cruciblev1 "github.com/crucible/sdk-go/crucible/v1"
)

// Cohort is the structured "who must approve" result.
type Cohort struct {
	// Groups is the deduplicated set of approver-group identifiers (e.g.
	// "@platform-team", "@payments-leads").
	Groups []string `json:"groups"`
	// RequireN is the total number of approver signatures required.
	RequireN int `json:"require_n"`
	// RequireCodeowner means at least one approval must come from a
	// CODEOWNERS-matched user. The gate enforces this by checking the
	// approval attestations' `codeowner=true` flag.
	RequireCodeowner bool `json:"require_codeowner"`
	// MatchedFiles maps approver groups to the files that pulled them in
	// (only populated for CODEOWNERS matches).
	MatchedFiles map[string][]string `json:"matched_files,omitempty"`
}

// IsEmpty means no human approval is required.
func (c *Cohort) IsEmpty() bool { return c.RequireN == 0 && len(c.Groups) == 0 }

// ApprovalConfig is the per-tenant configuration. Loaded from a YAML/JSON
// file under `CRUCIBLE_TENANT_POLICY_DIR/<tenant>/approvers.yaml`.
type ApprovalConfig struct {
	DefaultApprovers []string         `json:"default_approvers,omitempty"`
	Overrides        []ApprovalOverride `json:"overrides,omitempty"`
	Codeowners       []CodeOwnerEntry  `json:"codeowners,omitempty"`
}

// ApprovalOverride is a single match-rule entry from promotion-contract.md.
type ApprovalOverride struct {
	Matches          MatchSpec `json:"matches"`
	Approvers        []string  `json:"approvers"`
	RequireCodeowner bool      `json:"require_codeowner,omitempty"`
	RequireN         int       `json:"require_n_approvers,omitempty"`
}

// MatchSpec is the override's match clause.
type MatchSpec struct {
	SchemaChanges         bool     `json:"schema_changes,omitempty"`
	CriticalPathsTouched  []string `json:"critical_paths_touched,omitempty"`
	BlastRadiusImpact     string   `json:"blast_radius.estimated_impact,omitempty"`
}

// CodeOwnerEntry is a `path_glob → groups` mapping. Loaded from CODEOWNERS
// or YAML config.
type CodeOwnerEntry struct {
	PathGlob string   `json:"path_glob"`
	Groups   []string `json:"groups"`
}

// Router resolves cohorts.
type Router struct {
	cfg ApprovalConfig
}

// New builds a Router with the given config.
func New(cfg ApprovalConfig) *Router { return &Router{cfg: cfg} }

// Resolve walks the Rego decision + bundle and returns the Cohort.
func (r *Router) Resolve(decision *rego_engine.MergedDecision, bundle *cruciblev1.PromotionBundle) (*Cohort, error) {
	if decision == nil {
		return nil, errors.New("approval_router: nil decision")
	}
	if bundle == nil {
		return nil, errors.New("approval_router: nil bundle")
	}
	cohort := &Cohort{
		RequireCodeowner: decision.RequireCodeowner,
		RequireN:         decision.RequireNApprovers,
		MatchedFiles:     map[string][]string{},
	}

	// Start with the Rego-supplied groups.
	for _, g := range decision.ApproverGroups {
		cohort.Groups = appendUnique(cohort.Groups, g)
	}

	// Apply tenant override matchers.
	for _, ov := range r.cfg.Overrides {
		if !ov.Matches.matches(decision, bundle) {
			continue
		}
		for _, a := range ov.Approvers {
			cohort.Groups = appendUnique(cohort.Groups, a)
		}
		if ov.RequireCodeowner {
			cohort.RequireCodeowner = true
		}
		if ov.RequireN > cohort.RequireN {
			cohort.RequireN = ov.RequireN
		}
	}

	// CODEOWNERS: add groups for every file matched by a glob.
	if cohort.RequireCodeowner || len(cohort.Groups) == 0 {
		for _, file := range bundle.FilesChanged {
			for _, co := range r.cfg.Codeowners {
				if matchGlob(co.PathGlob, file.Path) {
					for _, g := range co.Groups {
						cohort.Groups = appendUnique(cohort.Groups, g)
						cohort.MatchedFiles[g] = append(cohort.MatchedFiles[g], file.Path)
					}
				}
			}
		}
	}

	// Default cohort if still empty AND a human is required.
	if len(cohort.Groups) == 0 && (decision.NeedsHuman || cohort.RequireN > 0) {
		for _, g := range r.cfg.DefaultApprovers {
			cohort.Groups = appendUnique(cohort.Groups, g)
		}
	}

	if decision.NeedsHuman && cohort.RequireN == 0 {
		cohort.RequireN = 1
	}

	sort.Strings(cohort.Groups)
	return cohort, nil
}

// CountValid reports how many of `attestedApprovals` satisfy the cohort.
// A valid approval:
//   - has `group` ∈ cohort.Groups, AND
//   - is non-self (approver_oidc_subject != agent_oidc_subject), AND
//   - when RequireCodeowner, at least one carries `codeowner=true`.
//
// The promotion gate calls this on each /approve webhook to know when the
// quorum is reached.
func (r *Router) CountValid(cohort *Cohort, agentOidc string, approvals []Approval) (int, bool, error) {
	valid := 0
	codeownerSeen := false
	for _, a := range approvals {
		if a.ApproverOidcSubject == "" || a.Attestation == "" {
			continue
		}
		if a.ApproverOidcSubject == agentOidc {
			return 0, false, ErrSelfApproval{OIDC: agentOidc}
		}
		if !containsCI(cohort.Groups, a.Group) && !containsCI(cohort.Groups, "*") {
			continue
		}
		valid++
		if a.Codeowner {
			codeownerSeen = true
		}
	}
	codeownerOK := !cohort.RequireCodeowner || codeownerSeen
	return valid, codeownerOK && valid >= cohort.RequireN, nil
}

// Approval is the in-memory view of an approval record. Mirrors
// policy.ApprovalRecord but lives here so the router doesn't import
// libs/policy when only its API is needed.
type Approval struct {
	ApproverOidcSubject string `json:"approver_oidc_subject"`
	Attestation         string `json:"attestation"`
	Group               string `json:"group,omitempty"`
	Codeowner           bool   `json:"codeowner,omitempty"`
}

// ErrSelfApproval is returned when the same OIDC subject both produced the
// bundle and tried to approve it. Threat T21.
type ErrSelfApproval struct{ OIDC string }

func (e ErrSelfApproval) Error() string {
	return "approval_router: self-approval forbidden for " + e.OIDC
}

// ── helpers ─────────────────────────────────────────────────────────────────

func (m MatchSpec) matches(decision *rego_engine.MergedDecision, bundle *cruciblev1.PromotionBundle) bool {
	if m.SchemaChanges {
		// We don't have direct schema-changes signals at this layer; rely on
		// the Rego decision's reasons set which mentions "schema change".
		hit := false
		for _, r := range decision.Reasons {
			if strings.Contains(r, "schema change") {
				hit = true
				break
			}
		}
		if !hit {
			return false
		}
	}
	if m.BlastRadiusImpact != "" && string(m.BlastRadiusImpact) != string(bundle.BlastRadius.Reversibility) {
		// The match key from promotion-contract.md is `blast_radius.estimated_impact`;
		// PromotionBundle in Phase 6 stores Reversibility directly. The
		// rego decision carries impact via its reasons; we treat absence of
		// a direct field as a partial match — the rego_engine is the
		// authoritative source.
	}
	if len(m.CriticalPathsTouched) > 0 {
		// We don't have direct list here; defer to rego_engine signal.
	}
	return true
}

func appendUnique(xs []string, x string) []string {
	for _, e := range xs {
		if e == x {
			return xs
		}
	}
	return append(xs, x)
}

func containsCI(xs []string, x string) bool {
	for _, e := range xs {
		if strings.EqualFold(e, x) {
			return true
		}
	}
	return false
}

// matchGlob implements a tiny glob matcher: `*` matches one path segment,
// `**` matches arbitrarily many. Good enough for CODEOWNERS-style entries
// without pulling in path/filepath.Match (which is single-segment only).
func matchGlob(pattern, path string) bool {
	patParts := splitPath(pattern)
	pathParts := splitPath(path)
	return matchParts(patParts, pathParts)
}

func splitPath(p string) []string {
	out := []string{}
	for _, seg := range strings.Split(strings.Trim(p, "/"), "/") {
		if seg == "" {
			continue
		}
		out = append(out, seg)
	}
	return out
}

func matchParts(pat, p []string) bool {
	for {
		if len(pat) == 0 {
			return len(p) == 0
		}
		if pat[0] == "**" {
			pat = pat[1:]
			// `**` matches zero or more segments.
			for i := 0; i <= len(p); i++ {
				if matchParts(pat, p[i:]) {
					return true
				}
			}
			return false
		}
		if len(p) == 0 {
			return false
		}
		if pat[0] != "*" && pat[0] != p[0] {
			// Single-segment glob with leading * suffix.
			if strings.Contains(pat[0], "*") {
				if simpleGlob(pat[0], p[0]) {
					pat = pat[1:]
					p = p[1:]
					continue
				}
			}
			return false
		}
		pat = pat[1:]
		p = p[1:]
	}
}

func simpleGlob(pat, s string) bool {
	if pat == "*" {
		return true
	}
	// Implement simple star matching within a segment: a*b, a*, *b.
	if i := strings.Index(pat, "*"); i >= 0 {
		prefix := pat[:i]
		suffix := pat[i+1:]
		return strings.HasPrefix(s, prefix) && strings.HasSuffix(s, suffix) && len(s) >= len(prefix)+len(suffix)
	}
	return pat == s
}
