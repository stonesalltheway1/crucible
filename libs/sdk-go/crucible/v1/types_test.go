package cruciblev1

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestAllPredicateTypes_AreCanonical(t *testing.T) {
	if len(AllPredicateTypes) != 14 {
		t.Fatalf("expected 14 predicate types, got %d", len(AllPredicateTypes))
	}
	seen := map[string]bool{}
	for _, p := range AllPredicateTypes {
		if !strings.HasPrefix(p, "https://crucible.dev/") {
			t.Errorf("unexpected predicate URI prefix: %s", p)
		}
		if !strings.HasSuffix(p, "/v1") {
			t.Errorf("predicate URI must be /v1-versioned: %s", p)
		}
		if seen[p] {
			t.Errorf("duplicate predicate type: %s", p)
		}
		seen[p] = true
	}
}

func TestScope_MarshalUnmarshal_All(t *testing.T) {
	s := Scope{All: true}
	b, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `"all"` {
		t.Fatalf("expected \"all\", got %s", string(b))
	}
	var back Scope
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	if !back.All || back.Filter != nil {
		t.Fatalf("round-trip mismatch: %+v", back)
	}
}

func TestScope_MarshalUnmarshal_Filter(t *testing.T) {
	s := Scope{Filter: &ScopeFilter{Repo: "github.com/a/b", FileGlob: "**/*.go"}}
	b, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"repo":"github.com/a/b"`) {
		t.Fatalf("filter not in output: %s", string(b))
	}
	var back Scope
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	if back.Filter == nil || back.Filter.Repo != "github.com/a/b" {
		t.Fatalf("round-trip mismatch: %+v", back)
	}
}

func TestScope_RejectsUnknownLiteral(t *testing.T) {
	var s Scope
	if err := json.Unmarshal([]byte(`"some"`), &s); err == nil {
		t.Fatal("expected error on unknown string literal")
	}
}

func TestScope_RejectsArray(t *testing.T) {
	var s Scope
	if err := json.Unmarshal([]byte(`[1,2]`), &s); err == nil {
		t.Fatal("expected error on array")
	}
}

func TestCrucibleError_FormatsAndCarriesMetadata(t *testing.T) {
	e := NewError(ErrBudgetExceeded, "budget hit", "re-plan", false)
	if got := e.Error(); !strings.Contains(got, "BudgetExceeded") || !strings.Contains(got, "re-plan") {
		t.Fatalf("Error() incomplete: %s", got)
	}
	if e.Retryable() {
		t.Fatal("expected non-retryable")
	}
}

func TestPlan_JSONFieldNames(t *testing.T) {
	p := Plan{
		TaskID:               "task_1",
		Description:          "x",
		EstimatedCostUsd:     1.0,
		EstimatedDurationMin: 10,
		Complexity:           ComplexityStandard,
		PlanHash:             "deadbeef",
	}
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`"task_id":"task_1"`,
		`"estimated_cost_usd":1`,
		`"complexity":"standard"`,
		`"plan_hash":"deadbeef"`,
	} {
		if !strings.Contains(string(b), want) {
			t.Errorf("missing field encoding %q in %s", want, string(b))
		}
	}
}

func TestFileChange_ActionEnumIsLowerSnake(t *testing.T) {
	c := FileChange{Path: "/x", Action: ActionAdd}
	b, _ := json.Marshal(c)
	if !strings.Contains(string(b), `"action":"add"`) {
		t.Fatalf("expected lower-case action; got %s", string(b))
	}
}

func TestComplexityValuesAreStable(t *testing.T) {
	want := []Complexity{
		ComplexityTrivial, ComplexityStandard, ComplexityComplex,
		ComplexityCritical, ComplexityModernization,
	}
	for _, c := range want {
		if c == "" {
			t.Fatalf("empty complexity constant")
		}
	}
}
