// Package verifygotest exercises the per-language Go runner end-to-end
// against the fixtures/good and fixtures/weak packages. The tests are
// hermetic: they DO NOT invoke `go build` / `go test` / `go-mutesting`
// against the host system (the verifier sandbox owns that). Instead
// they drive the runner's internal package boundaries directly and
// validate the emitted TestReport against the canonical schema.
//
// The full smoke test for the runner binary lives in the daemon's
// processpool_test.go — that test executes the compiled CLI in the
// FakeProvider sandbox and round-trips a TestReport.
package verifygotest

import (
	"context"
	"strings"
	"testing"
	"time"

	cruciblev1 "github.com/crucible/sdk-go/crucible/v1"
	"github.com/crucible/verifier/pkg/testreport"

	"github.com/crucible/verify-go/internal/audit"
	"github.com/crucible/verify-go/internal/diff"
	"github.com/crucible/verify-go/internal/tiers"
)

// TestAuditDenylistTrips ensures the leak-guard refuses requests that
// carry executor reasoning. Defence-in-depth duplicates the daemon's
// check; this test pins the runner-side behaviour.
func TestAuditDenylistTrips(t *testing.T) {
	bad := map[string]any{
		"task_id":   "t1",
		"reasoning": "the agent thinks foo",
	}
	err := audit.NoLeakage(bad)
	if err == nil {
		t.Fatal("expected LeakageError, got nil")
	}
	le, ok := err.(*audit.LeakageError)
	if !ok {
		t.Fatalf("expected *audit.LeakageError, got %T", err)
	}
	if le.OffendingField != "reasoning" {
		t.Fatalf("offending field = %q, want %q", le.OffendingField, "reasoning")
	}
	if le.Pattern != "reasoning" {
		t.Fatalf("pattern = %q, want %q", le.Pattern, "reasoning")
	}
}

func TestAuditDenylistRecurses(t *testing.T) {
	bad := map[string]any{
		"task_id": "t1",
		"nested": map[string]any{
			"thoughts": "hidden",
		},
	}
	err := audit.NoLeakage(bad)
	if err == nil {
		t.Fatal("expected nested leak detected")
	}
	if !strings.Contains(err.Error(), "nested.thoughts") {
		t.Fatalf("expected nested.thoughts path, got %v", err)
	}
}

func TestAuditDenylistArrayMaps(t *testing.T) {
	bad := map[string]any{
		"items": []any{
			map[string]any{"chain_of_thought": "x"},
		},
	}
	if err := audit.NoLeakage(bad); err == nil {
		t.Fatal("expected leak detected inside array element")
	}
}

func TestAuditCleanRequestPasses(t *testing.T) {
	ok := map[string]any{
		"task_id":   "t1",
		"tenant_id": "tenantA",
		"diff": map[string]any{
			"files": []any{
				map[string]any{"path": "main.go", "action": "modified"},
			},
		},
	}
	if err := audit.NoLeakage(ok); err != nil {
		t.Fatalf("unexpected leak error on clean request: %v", err)
	}
}

// TestFilterGoSplitsSourceAndTest pins the diff filter: source/test
// partition, deleted exclusion, vendor included (intentional).
func TestFilterGoSplitsSourceAndTest(t *testing.T) {
	d := cruciblev1.Diff{
		Files: []cruciblev1.FileChange{
			{Path: "fixtures/good/good.go", Action: "modified"},
			{Path: "fixtures/good/good_test.go", Action: "modified"},
			{Path: "fixtures/weak/weak.go", Action: "added"},
			{Path: "README.md", Action: "modified"},
			{Path: "removed.go", Action: "deleted"},
		},
	}
	fs := diff.FilterGo(d)
	if len(fs.Source) != 2 {
		t.Fatalf("source count = %d, want 2 (good.go, weak.go)", len(fs.Source))
	}
	if len(fs.Test) != 1 {
		t.Fatalf("test count = %d, want 1 (good_test.go)", len(fs.Test))
	}
	for _, f := range fs.Source {
		if strings.HasSuffix(f.Path, "_test.go") {
			t.Fatalf("test file %q leaked into source set", f.Path)
		}
	}
}

func TestFilterGoPackagesIsDistinct(t *testing.T) {
	d := cruciblev1.Diff{Files: []cruciblev1.FileChange{
		{Path: "fixtures/good/a.go", Action: "modified"},
		{Path: "fixtures/good/b.go", Action: "modified"},
		{Path: "fixtures/weak/c.go", Action: "modified"},
	}}
	pkgs := diff.FilterGo(d).Packages()
	if len(pkgs) != 2 {
		t.Fatalf("packages = %v, want 2 distinct", pkgs)
	}
}

// TestMutationParserHandlesInvertedSemantics is the keystone test for
// Tier 0. We feed a synthetic go-mutesting transcript and verify the
// runner counts FAIL as killed and PASS as survived, NOT the other way
// around. Mis-ordering this is a brand-existential bug.
func TestMutationParserHandlesInvertedSemantics(t *testing.T) {
	// Inline a representative go-mutesting transcript.
	transcript := []byte(strings.Join([]string{
		`FAIL "mutant/0" with checksum aaa  ./good.go:5:9: replaced + with -`,
		`PASS "mutant/1" with checksum bbb  ./weak.go:3:13: replaced == with !=`,
		`PASS "mutant/2" with checksum ccc  ./weak.go:7:9: replaced > with >=`,
		`FAIL "mutant/3" with checksum ddd  ./good.go:11:6: removed condition`,
		`The mutation score is 0.500 (2 passed, 2 failed, 0 duplicated, 0 skipped, total is 4)`,
	}, "\n"))
	stats := tiers.ParseMutestingOutputForTest(transcript)
	if stats.Killed != 2 {
		t.Fatalf("killed = %d, want 2 (the two FAIL lines)", stats.Killed)
	}
	if stats.Survived != 2 {
		t.Fatalf("survived = %d, want 2 (the two PASS lines)", stats.Survived)
	}
	if stats.Total != 4 {
		t.Fatalf("total = %d, want 4", stats.Total)
	}
	if stats.Score != 0.5 {
		t.Fatalf("score = %f, want 0.5", stats.Score)
	}
	if len(stats.SurvivedMutants) != 2 {
		t.Fatalf("survived_summary len = %d, want 2", len(stats.SurvivedMutants))
	}
	if stats.SurvivedMutants[0].File != "./weak.go" {
		t.Fatalf("first survived file = %q, want %q", stats.SurvivedMutants[0].File, "./weak.go")
	}
}

// TestMutationParserEmptyOutputZeroes verifies the parser is safe on
// an empty (tool-failed-immediately) transcript.
func TestMutationParserEmptyOutputZeroes(t *testing.T) {
	stats := tiers.ParseMutestingOutputForTest(nil)
	if stats.Total != 0 || stats.Score != 0 {
		t.Fatalf("expected zero counts, got %+v", stats)
	}
}

// TestMutationToolUnavailableProducesValidReport ensures that when
// go-mutesting is missing (the host case for the verifier test suite
// before the sandbox image is built), the runner emits a
// schema-valid TestReport with Verdict=tool_unavailable.
func TestMutationToolUnavailableProducesValidReport(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rep, err := tiers.RunMutation(ctx, tiers.MutationConfig{
		WorkDir:     "fixtures/weak",
		SourcePaths: []string{"weak.go"},
		Binary:      "go-mutesting-DEFINITELY-NOT-INSTALLED",
	})
	if err != nil {
		t.Fatalf("RunMutation returned err: %v", err)
	}
	if rep.Verdict != testreport.VerdictToolUnavailable {
		t.Fatalf("verdict = %q, want tool_unavailable", rep.Verdict)
	}
	if rep.Mutation == nil {
		t.Fatal("MutationStats nil")
	}
	if !rep.Mutation.DiffScoped {
		t.Fatal("DiffScoped must be true (Crucible mandate)")
	}
	stampForValidation(rep)
	if err := rep.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
}

// TestMutationEmptySourceListsSkips ensures a diff with no Go source
// files produces a Skipped report (e.g. doc-only change reached Go
// runner by accident).
func TestMutationEmptySourceListsSkips(t *testing.T) {
	rep, err := tiers.RunMutation(context.Background(), tiers.MutationConfig{
		WorkDir: "fixtures/good",
	})
	if err != nil {
		t.Fatal(err)
	}
	if rep.Verdict != testreport.VerdictSkipped {
		t.Fatalf("verdict = %q, want skipped", rep.Verdict)
	}
	if !rep.Passed {
		t.Fatal("skipped tier should pass (vacuous)")
	}
}

// TestPBTRapidPropertyDiscovery ensures the runner identifies the
// fixture's PropertyReverseIsInvolutive function by name. This is the
// scan that gates the `-run=^Property` selector.
func TestPBTRapidPropertyDiscovery(t *testing.T) {
	cfg := tiers.PBTConfig{
		WorkDir: "fixtures/good",
		TestFiles: []cruciblev1.FileChange{
			{Path: "good_test.go"},
		},
	}
	props := tiers.DiscoverPropertiesForTest(cfg)
	if len(props) == 0 {
		t.Fatal("expected at least one Property* function discovered")
	}
	var found bool
	for _, p := range props {
		if p == "PropertyReverseIsInvolutive" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected PropertyReverseIsInvolutive in %v", props)
	}
}

// TestPBTFuzzTargetDiscovery validates the regex catches the
// fixture's FuzzSum target with the exact `*testing.F` signature.
func TestPBTFuzzTargetDiscovery(t *testing.T) {
	cfg := tiers.PBTConfig{
		WorkDir: "fixtures/good",
		TestFiles: []cruciblev1.FileChange{
			{Path: "good_test.go"},
		},
	}
	targets := tiers.DiscoverFuzzTargetsForTest(cfg)
	if len(targets) != 1 {
		t.Fatalf("expected 1 fuzz target, got %d (%+v)", len(targets), targets)
	}
	if targets[0].Name != "FuzzSum" {
		t.Fatalf("target name = %q, want FuzzSum", targets[0].Name)
	}
}

// TestPBTReportPassesValidation drives RunPBT in the (very likely
// from this test harness) "go not on PATH" case and checks the
// emitted report still validates.
func TestPBTReportPassesValidation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rep, err := tiers.RunPBT(ctx, tiers.PBTConfig{
		WorkDir:  "fixtures/good",
		GoBinary: "go-DEFINITELY-NOT-INSTALLED",
		Timeout:  3 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	if rep.Verdict != testreport.VerdictToolUnavailable {
		t.Fatalf("verdict = %q, want tool_unavailable when go missing", rep.Verdict)
	}
	stampForValidation(rep)
	if err := rep.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
}

// TestPBTIterationsMinIsEnforced verifies the runner stamps the
// >=10_000 mandate.
func TestPBTIterationsMinIsEnforced(t *testing.T) {
	rep, err := tiers.RunPBT(context.Background(), tiers.PBTConfig{
		WorkDir:     "fixtures/good",
		GoBinary:    "go-NOT-INSTALLED",
		RapidChecks: 100, // under-mandate input
	})
	if err != nil {
		t.Fatal(err)
	}
	if rep.PBT.IterationsMin != tiers.IterationsMin {
		t.Fatalf("IterationsMin = %d, want %d", rep.PBT.IterationsMin, tiers.IterationsMin)
	}
	if rep.PBT.Iterations < tiers.IterationsMin {
		t.Fatalf("Iterations = %d should be coerced up to >= %d", rep.PBT.Iterations, tiers.IterationsMin)
	}
}

// TestContractSkippedWithoutSpecChanges verifies Tier 2 is correctly
// a no-op when the diff has no spec changes (typical Go business
// logic diff).
func TestContractSkippedWithoutSpecChanges(t *testing.T) {
	rep, err := tiers.RunContract(context.Background(), tiers.ContractConfig{
		WorkDir: "fixtures/good",
	})
	if err != nil {
		t.Fatal(err)
	}
	if rep.Verdict != testreport.VerdictSkipped {
		t.Fatalf("verdict = %q, want skipped", rep.Verdict)
	}
	if !rep.Passed {
		t.Fatal("skipped Tier 2 should pass")
	}
}

// TestProofAlwaysToolUnavailable pins the Tier 3 dispatch behaviour.
// Go has no first-class formal verifier in v1.
func TestProofAlwaysToolUnavailable(t *testing.T) {
	rep, err := tiers.RunProof(context.Background(), tiers.ProofConfig{})
	if err != nil {
		t.Fatal(err)
	}
	if rep.Verdict != testreport.VerdictToolUnavailable {
		t.Fatalf("verdict = %q, want tool_unavailable", rep.Verdict)
	}
	if rep.Passed {
		t.Fatal("Tier 3 must not fail open")
	}
	stampForValidation(rep)
	if err := rep.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
}

// TestHonestCIToolUnavailableWhenGoMissing ensures Tier 4 degrades
// gracefully.
func TestHonestCIToolUnavailableWhenGoMissing(t *testing.T) {
	rep, err := tiers.RunHonestCI(context.Background(), tiers.HonestCIConfig{
		WorkDir:  "fixtures/good",
		GoBinary: "go-NOT-INSTALLED",
	})
	if err != nil {
		t.Fatal(err)
	}
	if rep.Verdict != testreport.VerdictToolUnavailable {
		t.Fatalf("verdict = %q, want tool_unavailable", rep.Verdict)
	}
	if rep.HonestCI == nil || rep.HonestCI.BuilderID != tiers.BuilderID {
		t.Fatalf("BuilderID = %q, want %q", rep.HonestCI.BuilderID, tiers.BuilderID)
	}
}

// stampForValidation fills the identity fields main.go would
// otherwise stamp so we can call testreport.Validate() in unit tests
// without simulating the whole pipeline.
func stampForValidation(r *testreport.TestReport) {
	r.SchemaVersion = testreport.SchemaVersion
	r.TaskID = "task-test"
	if r.Language == "" {
		r.Language = testreport.LangGo
	}
	if r.ReporterID == "" {
		r.ReporterID = "crucible-verify-go-test"
	}
}
