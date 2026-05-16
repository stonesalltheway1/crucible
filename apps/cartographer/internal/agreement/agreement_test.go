package agreement

import (
	"strings"
	"testing"
	"time"

	"github.com/crucible/apps/cartographer/internal/types"
)

func mk(category, rule, channel, src string, conf float64) types.ConventionCandidate {
	return types.ConventionCandidate{
		Category: category, RuleNL: rule, SourceChannel: channel,
		SourcePath: src, Confidence: conf,
		FirstSeen: time.Now(), Status: "candidate",
	}
}

func TestSignatureNormalizes(t *testing.T) {
	a := signature("Use cursor pagination.")
	b := signature("use Cursor pagination,")
	if a != b {
		t.Errorf("expected stable signature: %q vs %q", a, b)
	}
}

func TestScoreClusters(t *testing.T) {
	in := []types.ConventionCandidate{
		mk("Performance defaults", "Use cursor pagination.", "pr_comment", "pr/1", 0.45),
		mk("Performance defaults", "Use Cursor pagination,", "pr_comment", "pr/2", 0.45),
		mk("Performance defaults", "use cursor pagination — never offset", "adr_file", "adr/0007.md", 0.85),
		mk("Naming", "snake_case for db column names", "lint_config", ".editorconfig", 0.7),
	}
	b := Score(in, 5)
	hi, md, lo := b.Counts()
	total := hi + md + lo
	if total < 1 {
		t.Fatalf("no buckets populated")
	}
	// The cursor pagination cluster should land in HIGH due to multi-source agreement + tier-A.
	found := false
	for _, c := range b.High {
		if strings.Contains(strings.ToLower(c.RuleNL), "cursor pagination") {
			found = true
		}
	}
	if !found {
		t.Errorf("cursor-pagination cluster not in HIGH: %+v", b.High)
	}
}

func TestFilterContradictions(t *testing.T) {
	b := Bucket{
		High: []types.ConventionCandidate{
			mk("Performance defaults", "Do NOT use cursor pagination — use offset", "pr_comment", "pr/3", 0.7),
			mk("Naming", "Use snake_case", "lint_config", ".editorconfig", 0.7),
		},
	}
	b.FilterContradictions()
	if len(b.High) != 1 {
		t.Fatalf("got %d, want 1", len(b.High))
	}
	if !strings.Contains(b.High[0].RuleNL, "snake_case") {
		t.Errorf("wrong candidate kept: %v", b.High[0])
	}
}

func TestLog1Approximates(t *testing.T) {
	// Very loose check — Padé approximant for the small range we care about.
	for _, x := range []float64{1, 2, 5, 10, 100, 1000} {
		l := log1(x)
		if l < 0 {
			t.Errorf("log1(%v)=%v < 0", x, l)
		}
	}
}

func TestEmptyInput(t *testing.T) {
	b := Score(nil, 0)
	hi, md, lo := b.Counts()
	if hi != 0 || md != 0 || lo != 0 {
		t.Errorf("non-empty buckets on empty input")
	}
}
