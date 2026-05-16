package processpool

import (
	"context"
	"encoding/json"
	"testing"

	cruciblev1 "github.com/crucible/sdk-go/crucible/v1"
	"github.com/crucible/verifier/internal/dispatcher"
	"github.com/crucible/verifier/internal/verification"
	"github.com/crucible/verifier/pkg/testreport"
)

func newReq() *verification.VerificationRequest {
	return &verification.VerificationRequest{
		TaskID:   "t1",
		TenantID: "ten_1",
		BaseSHA:  "abc",
		Diff: cruciblev1.Diff{Files: []cruciblev1.FileChange{
			{Path: "a.py", Action: cruciblev1.ActionModify, ContentSha256: "0xa"},
		}},
		Routing: cruciblev1.Routing{
			ExecutorVendor: "anthropic",
			VerifierVendor: "google",
		},
		ExecutorSandboxID: "sb_executor",
	}
}

func TestPool_submitWritesReportThroughFakeProvider(t *testing.T) {
	want := &testreport.TestReport{
		SchemaVersion: testreport.SchemaVersion,
		TaskID:        "t1",
		Tier:          testreport.TierMutation,
		Language:      testreport.LangPython,
		Framework:     "mutmut",
		Verdict:       testreport.VerdictPassed,
		Passed:        true,
		Mutation: &testreport.MutationStats{
			Killed: 9, Survived: 1, Total: 10, Score: 0.9, Threshold: 0.85, DiffScoped: true,
		},
		DurationSeconds: 1,
	}
	provider := &FakeProvider{Reports: map[string]*testreport.TestReport{
		"python/tier_0_mutation": want,
	}}
	pool := NewPool(provider)
	got, err := pool.Submit(context.Background(), dispatcher.RunnerKind{
		Language: testreport.LangPython, Tier: testreport.TierMutation,
	}, newReq())
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if got.Mutation == nil || got.Mutation.Killed != 9 {
		t.Fatalf("Submit did not round-trip mutation stats; got %+v", got)
	}
}

func TestPool_refusesSpawnIntoExecutorSandbox(t *testing.T) {
	provider := &collidingSandboxProvider{collidingID: "sb_executor"}
	pool := NewPool(provider)
	_, err := pool.Submit(context.Background(), dispatcher.RunnerKind{
		Language: testreport.LangPython, Tier: testreport.TierMutation,
	}, newReq())
	if err == nil {
		t.Fatalf("expected refusal when spawning into executor sandbox")
	}
}

type collidingSandboxProvider struct{ collidingID string }

func (p *collidingSandboxProvider) Spawn(_ context.Context, spec SandboxSpec) (Sandbox, error) {
	return &collidingSandbox{id: p.collidingID, spec: spec}, nil
}

type collidingSandbox struct {
	id   string
	spec SandboxSpec
}

func (s *collidingSandbox) ID() string { return s.id }
func (s *collidingSandbox) Kill(_ context.Context) error { return nil }
func (s *collidingSandbox) Exec(_ context.Context, _ []string, _ []byte) (Output, error) {
	r := &testreport.TestReport{
		SchemaVersion: testreport.SchemaVersion, TaskID: s.spec.TaskID,
		Tier: s.spec.Tier, Language: s.spec.Language, Verdict: testreport.VerdictPassed, Passed: true,
		Mutation: &testreport.MutationStats{Killed: 1, Survived: 0, Total: 1, Score: 1.0, Threshold: 0.85, DiffScoped: true},
	}
	b, _ := json.Marshal(r)
	return Output{Stdout: b}, nil
}

func TestTrimPrelude(t *testing.T) {
	in := []byte("pytest preamble\nsome lines\n===CRUCIBLE-TESTREPORT===\n{\"task_id\":\"t1\"}\n")
	got := trimPrelude(in)
	if string(got) != `{"task_id":"t1"}` {
		t.Fatalf("trimPrelude: got %q", got)
	}
}
