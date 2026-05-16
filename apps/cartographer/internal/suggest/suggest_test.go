package suggest

import (
	"strings"
	"testing"

	"github.com/crucible/apps/cartographer/internal/symbols"
	"github.com/crucible/apps/cartographer/internal/types"
)

func TestSuggestAlwaysIncludesReadmeRefresh(t *testing.T) {
	out := Suggest("", nil, nil, 5)
	if len(out) == 0 {
		t.Fatal("no suggestions")
	}
	found := false
	for _, s := range out {
		if strings.Contains(strings.ToLower(s.Title), "readme") {
			found = true
		}
	}
	if !found {
		t.Error("expected README suggestion")
	}
}

func TestSuggestPicksWebhookWhenSeen(t *testing.T) {
	idx := &symbols.Index{ByName: map[string][]int{"handle_webhook": {0}}}
	out := Suggest("nextjs", idx, nil, 5)
	found := false
	for _, s := range out {
		if strings.Contains(s.Title, "idempotency-key") {
			found = true
		}
	}
	if !found {
		t.Error("expected webhook suggestion when symbol present")
	}
}

func TestSuggestPicksCursorPaginationWhenConvSeen(t *testing.T) {
	cs := []types.ConventionCandidate{{Category: "Performance defaults", RuleNL: "Prefer cursor pagination over offset."}}
	out := Suggest("go-services", nil, cs, 5)
	found := false
	for _, s := range out {
		if strings.Contains(s.Title, "cursor pagination") {
			found = true
		}
	}
	if !found {
		t.Error("expected cursor-pagination suggestion when convention seen")
	}
}

func TestSuggestRespectsLimit(t *testing.T) {
	cs := []types.ConventionCandidate{{Category: "Performance defaults", RuleNL: "Prefer cursor pagination."}}
	idx := &symbols.Index{ByName: map[string][]int{"handle_webhook": {0}, "print": {1}}}
	out := Suggest("go-services", idx, cs, 2)
	if len(out) > 2 {
		t.Errorf("got %d, want ≤ 2", len(out))
	}
}
