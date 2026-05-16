package inferred

import (
	"strings"
	"testing"

	"github.com/crucible/apps/cartographer/internal/types"
)

func TestGenerateEmpty(t *testing.T) {
	got := Generate("acme/x", "nextjs", nil, nil)
	if !strings.Contains(got, "No high-confidence conventions") {
		t.Errorf("missing empty marker: %q", got)
	}
}

func TestGenerateOrdersByCategoryAndConfidence(t *testing.T) {
	high := []types.ConventionCandidate{
		{Category: "Naming", RuleNL: "Use snake_case", SourceChannel: "lint_config", SourcePath: ".editorconfig", Confidence: 0.85},
		{Category: "Performance defaults", RuleNL: "Cursor pagination", SourceChannel: "adr_file", SourcePath: "adr/0007.md", Confidence: 0.9},
		{Category: "Naming", RuleNL: "kebab-case for URLs", SourceChannel: "agents_md", SourcePath: "AGENTS.md", Confidence: 0.7},
	}
	got := Generate("acme/x", "go-services", high, nil)
	if !strings.Contains(got, "## Naming") {
		t.Error("missing Naming heading")
	}
	if !strings.Contains(got, "Use snake_case") {
		t.Error("missing first naming rule")
	}
	if !strings.Contains(got, "Cursor pagination") {
		t.Error("missing performance rule")
	}
	if !strings.Contains(got, "(stack: go-services)") {
		t.Error("missing stack annotation")
	}
	if !strings.Contains(got, "Next steps") {
		t.Error("missing next-steps trailer")
	}
}

func TestGenerateCapsPerCategory(t *testing.T) {
	var high []types.ConventionCandidate
	for i := 0; i < 20; i++ {
		high = append(high, types.ConventionCandidate{
			Category: "Naming", RuleNL: "rule x", SourceChannel: "lint_config", SourcePath: ".editorconfig", Confidence: 0.5,
		})
	}
	got := Generate("repo", "stack", high, nil)
	count := strings.Count(got, "rule x")
	if count > 8 {
		t.Errorf("too many rules in category: got %d", count)
	}
}
