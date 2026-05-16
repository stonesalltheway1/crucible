package budget

import (
	"strings"
	"testing"

	memoryspec "github.com/crucible/memory-spec/go"
	cruciblev1 "github.com/crucible/sdk-go/crucible/v1"
)

func mem(content string, score float64) memoryspec.ScoredMemory {
	return memoryspec.ScoredMemory{
		Memory:        cruciblev1.Memory{Content: content},
		FinalScore:    score,
		TokenEstimate: Estimate(content),
	}
}

func TestEstimate_FourCharsPerToken(t *testing.T) {
	if got := Estimate(strings.Repeat("x", 100)); got != 25 {
		t.Fatalf("100/4 = 25, got %d", got)
	}
	if Estimate("") != 0 {
		t.Fatal("empty content = 0 tokens")
	}
}

func TestEnforce_DropsLowestScoreFirst(t *testing.T) {
	mems := []memoryspec.ScoredMemory{
		mem(strings.Repeat("a", 4000), 0.9),  // 1000 tokens
		mem(strings.Repeat("b", 12000), 0.5), // 3000 tokens
		mem(strings.Repeat("c", 12000), 0.8), // 3000 tokens
	}
	out, used := Enforce(mems, 5000) // budget = 5000 tokens
	if used > 5000 {
		t.Fatalf("used %d > 5000 budget", used)
	}
	// Score 0.9 fits (1000) + 0.8 fits (3000) → total 4000, under
	// budget. Score 0.5 (3000) would overflow → dropped.
	if len(out) != 2 {
		t.Fatalf("want 2 items kept, got %d", len(out))
	}
	if out[0].FinalScore < out[1].FinalScore {
		t.Fatal("ordering must be score-descending")
	}
}

func TestEnforce_TiesResolveBySmallerCheaperFirst(t *testing.T) {
	mems := []memoryspec.ScoredMemory{
		mem(strings.Repeat("x", 200), 0.7), // 50 tokens
		mem(strings.Repeat("y", 800), 0.7), // 200 tokens
	}
	out, _ := Enforce(mems, 1000)
	if len(out) != 2 {
		t.Fatalf("both fit; got %d", len(out))
	}
	if out[0].TokenEstimate != 50 {
		t.Fatal("cheaper item should sort first on tie")
	}
}

func TestEnforce_DefaultBudgetWhenZero(t *testing.T) {
	mems := []memoryspec.ScoredMemory{
		mem(strings.Repeat("a", 28000), 0.9), // 7000 tokens — exactly fits default
	}
	out, used := Enforce(mems, 0)
	if len(out) != 1 || used != 7000 {
		t.Fatalf("default budget should be 7000; got %d items, %d tokens", len(out), used)
	}
}

func TestEnforce_StableUnderEmpty(t *testing.T) {
	out, used := Enforce(nil, 1000)
	if len(out) != 0 || used != 0 {
		t.Fatal("empty input must return empty without panic")
	}
}
