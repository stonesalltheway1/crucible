package testreport

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func newSampleMutationReport() *TestReport {
	now := time.Date(2026, 5, 15, 12, 0, 0, 0, time.UTC)
	return &TestReport{
		SchemaVersion: SchemaVersion,
		TaskID:        "task_01",
		DiffHash:      "deadbeef",
		Tier:          TierMutation,
		Language:      LangPython,
		Framework:     "mutmut",
		Verdict:       VerdictPassed,
		Passed:        true,
		StartedAt:     now,
		FinishedAt:    now.Add(30 * time.Second),
		DurationSeconds: 30,
		WallClockBudgetSeconds: 120,
		Mutation: &MutationStats{
			Killed: 17, Survived: 2, Total: 19,
			Score: 17.0 / 19.0, Threshold: 0.85,
			DiffScoped: true,
		},
		ReporterID:      "crucible-python-runner",
		ReporterVersion: "phase4-dev",
	}
}

func TestValidate_ok(t *testing.T) {
	r := newSampleMutationReport()
	if err := r.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
}

func TestValidate_rejectsNonDiffScopedMutation(t *testing.T) {
	r := newSampleMutationReport()
	r.Mutation.DiffScoped = false
	if err := r.Validate(); err == nil {
		t.Fatalf("expected error for non-diff-scoped mutation")
	} else if !strings.Contains(err.Error(), "diff-scoped") {
		t.Fatalf("error %q does not mention diff-scoped invariant", err)
	}
}

func TestValidate_rejectsLowPBTIterations(t *testing.T) {
	r := newSampleMutationReport()
	r.Tier = TierPBT
	r.Mutation = nil
	r.PBT = &PBTStats{
		Iterations:    100,
		IterationsMin: 10_000,
	}
	r.Framework = "hypothesis"
	if err := r.Validate(); err == nil {
		t.Fatalf("expected error for PBT iterations < required")
	}
}

func TestValidate_rejectsUnknownTier(t *testing.T) {
	r := newSampleMutationReport()
	r.Tier = "tier_99_bogus"
	if err := r.Validate(); err == nil {
		t.Fatalf("expected error for unknown tier")
	}
}

func TestValidate_rejectsMissingTaskID(t *testing.T) {
	r := newSampleMutationReport()
	r.TaskID = ""
	if err := r.Validate(); err == nil {
		t.Fatalf("expected error for missing task_id")
	}
}

func TestCanonicalJSON_isStable(t *testing.T) {
	r := newSampleMutationReport()
	a, err := r.CanonicalJSON()
	if err != nil {
		t.Fatal(err)
	}
	b, err := r.CanonicalJSON()
	if err != nil {
		t.Fatal(err)
	}
	if string(a) != string(b) {
		t.Fatalf("canonical JSON not stable across calls")
	}
}

func TestContentHash_changes_on_score(t *testing.T) {
	r := newSampleMutationReport()
	h1, _ := r.ContentHash()
	r.Mutation.Score = r.Mutation.Score - 0.01
	h2, _ := r.ContentHash()
	if h1 == h2 {
		t.Fatalf("ContentHash should change when Score changes")
	}
}

func TestAsTierResult_capturesScore(t *testing.T) {
	r := newSampleMutationReport()
	tr := r.AsTierResult()
	if tr.Score < 0.89 || tr.Score > 0.90 {
		t.Fatalf("AsTierResult score = %v, want ~0.894", tr.Score)
	}
	if tr.Framework != "mutmut" {
		t.Fatalf("framework not preserved")
	}
}

func TestMergeFindings_dedupes(t *testing.T) {
	a := newSampleMutationReport()
	a.Findings = []Finding{
		{Category: "mutation_survived", File: "f.py", Line: 12, Detail: "X"},
		{Category: "mutation_survived", File: "f.py", Line: 12, Detail: "X"}, // dup
		{Category: "mutation_survived", File: "f.py", Line: 13, Detail: "Y"},
	}
	b := newSampleMutationReport()
	b.Findings = []Finding{
		{Category: "mutation_survived", File: "f.py", Line: 13, Detail: "Y"}, // dup across reports
		{Category: "mutation_survived", File: "f.py", Line: 14, Detail: "Z"},
	}
	got := MergeFindings(a, b)
	if len(got) != 3 {
		t.Fatalf("MergeFindings: got %d findings, want 3", len(got))
	}
}

func TestJSONRoundTrip(t *testing.T) {
	r := newSampleMutationReport()
	raw, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var back TestReport
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatal(err)
	}
	if back.TaskID != r.TaskID || back.Tier != r.Tier {
		t.Fatalf("round-trip mismatch")
	}
	if back.Mutation == nil || back.Mutation.Killed != r.Mutation.Killed {
		t.Fatalf("mutation stats round-trip mismatch")
	}
}
