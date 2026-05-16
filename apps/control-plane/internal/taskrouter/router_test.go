package taskrouter

import (
	"context"
	"strings"
	"testing"

	"github.com/crucible/control-plane/internal/modelrouter"
	cruciblev1 "github.com/crucible/sdk-go/crucible/v1"
)

func TestHeuristic_Trivial(t *testing.T) {
	c := heuristicClassify("Fix typo in README.md")
	if c.Complexity != "trivial" {
		t.Fatalf("expected trivial, got %s", c.Complexity)
	}
	if c.CriticalScore != 0 {
		t.Fatalf("expected score 0, got %d", c.CriticalScore)
	}
}

func TestHeuristic_Migration(t *testing.T) {
	c := heuristicClassify("Add an additive migration to the users table")
	if c.Complexity != "critical" {
		t.Fatalf("expected critical for migration, got %s", c.Complexity)
	}
	if c.CriticalScore < 80 {
		t.Fatalf("expected critical_score >= 80, got %d", c.CriticalScore)
	}
}

func TestHeuristic_Auth(t *testing.T) {
	c := heuristicClassify("Implement auth middleware for the billing service")
	if c.Complexity != "critical" {
		t.Fatalf("expected critical, got %s", c.Complexity)
	}
}

func TestHeuristic_DefaultStandard(t *testing.T) {
	c := heuristicClassify("Add Stripe webhook handler for refund events")
	if c.Complexity != "standard" {
		t.Fatalf("expected standard for default case, got %s", c.Complexity)
	}
}

func TestParseClassifierResponse_StripsFences(t *testing.T) {
	in := "```json\n{\"complexity\":\"complex\",\"critical_score\":40,\"rationale\":\"why\",\"suggested_files\":[]}\n```"
	c, err := parseClassifierResponse(in)
	if err != nil {
		t.Fatal(err)
	}
	if c.Complexity != "complex" {
		t.Fatalf("expected complex, got %s", c.Complexity)
	}
}

func TestParseClassifierResponse_RejectsEmptyComplexity(t *testing.T) {
	_, err := parseClassifierResponse(`{"critical_score":1}`)
	if err == nil {
		t.Fatal("expected error on missing complexity")
	}
}

func TestRoute_PicksCrossFamilyVerifier(t *testing.T) {
	r := New(nil, "")
	routing, err := r.Route(Classification{Complexity: "complex", CriticalScore: 10})
	if err != nil {
		t.Fatal(err)
	}
	if routing.ExecutorVendor == routing.VerifierVendor {
		t.Fatalf("verifier must be cross-family: exec=%s ver=%s",
			routing.ExecutorVendor, routing.VerifierVendor)
	}
	if routing.IsCritical {
		t.Fatal("expected non-critical for complex/critical-score=10")
	}
}

func TestRoute_PromotesToTier2WhenCritical(t *testing.T) {
	r := New(nil, "")
	routing, err := r.Route(Classification{Complexity: "trivial", CriticalScore: 90})
	if err != nil {
		t.Fatal(err)
	}
	if routing.ExecutorTier != cruciblev1.ModelTier(int(modelrouter.Tier2)) {
		t.Fatalf("expected Tier 2 due to critical promotion, got %d", routing.ExecutorTier)
	}
	if !routing.IsCritical {
		t.Fatal("expected IsCritical=true")
	}
}

func TestClassify_FallsBackWhenNoVendor(t *testing.T) {
	// No vendor wired → fall through to heuristic.
	r := New(modelrouter.NewRouter(), "")
	c, err := r.Classify(context.Background(), "Rename a variable")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(c.Rationale, "fallback") {
		t.Fatalf("expected heuristic fallback rationale, got %q", c.Rationale)
	}
}
