package approval_router

import (
	"strings"
	"testing"

	"github.com/crucible/policy"
	"github.com/crucible/promotion-gate/internal/rego_engine"
	cruciblev1 "github.com/crucible/sdk-go/crucible/v1"
)

func mkDecision(groups []string, needsHuman, requireCodeowner bool, n int, reasons ...string) *rego_engine.MergedDecision {
	return &rego_engine.MergedDecision{
		ApproverGroups:    groups,
		NeedsHuman:        needsHuman,
		RequireCodeowner:  requireCodeowner,
		RequireNApprovers: n,
		Reasons:           reasons,
		DefaultDecision: policy.Decision{
			ApproverGroups:    groups,
			NeedsHuman:        needsHuman,
			RequireCodeowner:  requireCodeowner,
			RequireNApprovers: n,
		},
	}
}

func mkBundle(files ...string) *cruciblev1.PromotionBundle {
	fc := make([]cruciblev1.FileChange, len(files))
	for i, f := range files {
		fc[i] = cruciblev1.FileChange{Path: f, Action: "modify", ContentSha256: "abc"}
	}
	return &cruciblev1.PromotionBundle{
		TaskID: "t", DiffHash: "0x", VerifierApprovalAttestation: "rekor:x",
		AgentOidcSubject: "agent_oidc",
		FilesChanged:     fc,
	}
}

func TestResolve_RegoGroupsCarry(t *testing.T) {
	r := New(ApprovalConfig{DefaultApprovers: []string{"@platform-team"}})
	c, err := r.Resolve(mkDecision([]string{"@dba-team"}, true, false, 1), mkBundle("api/x.go"))
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Groups) != 1 || c.Groups[0] != "@dba-team" {
		t.Fatalf("expected @dba-team, got %v", c.Groups)
	}
}

func TestResolve_DefaultsWhenEmpty(t *testing.T) {
	r := New(ApprovalConfig{DefaultApprovers: []string{"@platform-team"}})
	c, err := r.Resolve(mkDecision(nil, true, false, 0), mkBundle("api/x.go"))
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Groups) != 1 || c.Groups[0] != "@platform-team" {
		t.Fatalf("expected default, got %v", c.Groups)
	}
	if c.RequireN != 1 {
		t.Fatalf("expected RequireN=1 when NeedsHuman, got %d", c.RequireN)
	}
}

func TestResolve_CodeownersGlob(t *testing.T) {
	cfg := ApprovalConfig{
		Codeowners: []CodeOwnerEntry{
			{PathGlob: "src/billing/**", Groups: []string{"@payments-leads"}},
		},
	}
	r := New(cfg)
	c, err := r.Resolve(mkDecision([]string{"@platform-team"}, true, true, 2), mkBundle("src/billing/refunds.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !contains(c.Groups, "@payments-leads") {
		t.Fatalf("expected codeowners @payments-leads, got %v", c.Groups)
	}
	if len(c.MatchedFiles["@payments-leads"]) == 0 {
		t.Fatal("expected matched files recorded")
	}
}

func TestCountValid_RejectsSelf(t *testing.T) {
	r := New(ApprovalConfig{})
	cohort := &Cohort{Groups: []string{"@p"}, RequireN: 1}
	_, _, err := r.CountValid(cohort, "agent_oidc", []Approval{
		{ApproverOidcSubject: "agent_oidc", Attestation: "rekor:1", Group: "@p"},
	})
	if err == nil || !strings.Contains(err.Error(), "self-approval") {
		t.Fatalf("expected self-approval rejection, got %v", err)
	}
}

func TestCountValid_RequiresCodeowner(t *testing.T) {
	r := New(ApprovalConfig{})
	cohort := &Cohort{Groups: []string{"@p"}, RequireN: 1, RequireCodeowner: true}
	// Non-codeowner approval doesn't satisfy.
	_, ok, _ := r.CountValid(cohort, "agent", []Approval{
		{ApproverOidcSubject: "u1", Attestation: "rekor:1", Group: "@p", Codeowner: false},
	})
	if ok {
		t.Fatal("expected NOT ok when codeowner required but not present")
	}
	_, ok, _ = r.CountValid(cohort, "agent", []Approval{
		{ApproverOidcSubject: "u1", Attestation: "rekor:1", Group: "@p", Codeowner: true},
	})
	if !ok {
		t.Fatal("expected ok with codeowner approval")
	}
}

func TestCountValid_NofM(t *testing.T) {
	r := New(ApprovalConfig{})
	cohort := &Cohort{Groups: []string{"@p"}, RequireN: 2}
	_, ok, _ := r.CountValid(cohort, "agent", []Approval{
		{ApproverOidcSubject: "u1", Attestation: "rekor:1", Group: "@p"},
	})
	if ok {
		t.Fatal("expected not ok with 1/2 approvals")
	}
	_, ok, _ = r.CountValid(cohort, "agent", []Approval{
		{ApproverOidcSubject: "u1", Attestation: "rekor:1", Group: "@p"},
		{ApproverOidcSubject: "u2", Attestation: "rekor:2", Group: "@p"},
	})
	if !ok {
		t.Fatal("expected ok at 2/2")
	}
}

func TestGlobMatching(t *testing.T) {
	cases := []struct {
		pat, path string
		want      bool
	}{
		{"src/billing/**", "src/billing/refunds.go", true},
		{"src/billing/**", "src/billing/sub/refunds.go", true},
		{"src/billing/*.go", "src/billing/refunds.go", true},
		{"src/billing/*.go", "src/billing/sub/refunds.go", false},
		{"src/billing/*", "src/billing/refunds.go", true},
		{"src/api/**", "src/billing/refunds.go", false},
	}
	for _, c := range cases {
		got := matchGlob(c.pat, c.path)
		if got != c.want {
			t.Errorf("matchGlob(%q, %q)=%v want %v", c.pat, c.path, got, c.want)
		}
	}
}

func contains(xs []string, x string) bool {
	for _, e := range xs {
		if e == x {
			return true
		}
	}
	return false
}
