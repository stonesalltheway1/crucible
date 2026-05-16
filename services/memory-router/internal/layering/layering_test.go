package layering

import (
	"strings"
	"testing"
	"time"

	memoryspec "github.com/crucible/memory-spec/go"
	cruciblev1 "github.com/crucible/sdk-go/crucible/v1"
)

func conv(id, rule string, layer memoryspec.MemoryLayer) memoryspec.Convention {
	now := time.Now().UTC()
	return memoryspec.Convention{
		ID:         id,
		TenantID:   "ten_test",
		Layer:      layer,
		Scope:      cruciblev1.ScopeFilter{FileGlob: "api/**/*.ts"},
		RuleNl:     rule,
		Category:   memoryspec.CatLogging,
		Status:     memoryspec.StatusActive,
		Confidence: 0.7,
		ValidFrom:  now,
		WrittenAt:  now,
	}
}

func TestMerge_RepoOverridesBeatsGlobalDefaults(t *testing.T) {
	in := map[memoryspec.MemoryLayer][]memoryspec.Convention{
		memoryspec.LayerGlobalDefaults: {conv("conv_a", "use slog", memoryspec.LayerGlobalDefaults)},
		memoryspec.LayerRepoOverrides:  {conv("conv_a", "use pino", memoryspec.LayerRepoOverrides)},
	}
	out := Merge(in)
	if len(out) != 1 {
		t.Fatalf("want 1 winner; got %d", len(out))
	}
	if !strings.Contains(out[0].RuleNl, "pino") {
		t.Fatalf("repo override must win on same ID; got %q", out[0].RuleNl)
	}
	if out[0].Layer != memoryspec.LayerRepoOverrides {
		t.Fatalf("winner layer should be repo_overrides; got %q", out[0].Layer)
	}
}

func TestMerge_DistinctIdsAllKept(t *testing.T) {
	in := map[memoryspec.MemoryLayer][]memoryspec.Convention{
		memoryspec.LayerGlobalDefaults: {conv("conv_a", "rule a", memoryspec.LayerGlobalDefaults)},
		memoryspec.LayerOrgOverrides:   {conv("conv_b", "rule b", memoryspec.LayerOrgOverrides)},
	}
	out := Merge(in)
	if len(out) != 2 {
		t.Fatalf("two distinct ids; got %d", len(out))
	}
}

func TestMerge_SemanticParaphraseCollapsedAtSameCategory(t *testing.T) {
	in := map[memoryspec.MemoryLayer][]memoryspec.Convention{
		memoryspec.LayerGlobalDefaults: {conv("conv_default", "use slog calls", memoryspec.LayerGlobalDefaults)},
		// Same lowered-normalized rule wording at the repo layer with a
		// different ID — should still suppress the global default.
		memoryspec.LayerRepoOverrides: {conv("conv_repo", "Use Slog Calls", memoryspec.LayerRepoOverrides)},
	}
	out := Merge(in)
	if len(out) != 1 {
		t.Fatalf("semantic collapse: want 1, got %d", len(out))
	}
	if out[0].ID != "conv_repo" {
		t.Fatalf("higher-layer same-semantics rule must win; got id %q", out[0].ID)
	}
}

func TestMerge_OrderIsBottomUp(t *testing.T) {
	in := map[memoryspec.MemoryLayer][]memoryspec.Convention{
		memoryspec.LayerGlobalDefaults: {conv("conv_a", "rule a", memoryspec.LayerGlobalDefaults)},
		memoryspec.LayerOrgOverrides:   {conv("conv_b", "rule b", memoryspec.LayerOrgOverrides)},
		memoryspec.LayerRepoOverrides:  {conv("conv_c", "rule c", memoryspec.LayerRepoOverrides)},
	}
	out := Merge(in)
	if out[0].Layer != memoryspec.LayerGlobalDefaults {
		t.Fatal("first item should be lowest-priority layer")
	}
	if out[len(out)-1].Layer != memoryspec.LayerRepoOverrides {
		t.Fatal("last item should be highest-priority layer")
	}
}

func TestIsCustomerOverridePath_AcceptsCommonPaths(t *testing.T) {
	for _, p := range []string{"AGENTS.md", "agents.md", "CLAUDE.md", ".cursorrules"} {
		if !IsCustomerOverridePath(p) {
			t.Fatalf("path %q should be recognised as override", p)
		}
	}
	for _, p := range []string{"docs/agents.md", "src/foo.go"} {
		if IsCustomerOverridePath(p) {
			t.Fatalf("path %q must not be recognised as override", p)
		}
	}
}
