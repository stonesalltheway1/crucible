package oss

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/crucible/apps/cartographer/internal/types"
)

func TestLoadStackArray(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "nextjs")
	_ = os.MkdirAll(dir, 0o755)
	body, _ := json.Marshal([]types.ConventionCandidate{{
		Category: "Library preferences", RuleNL: "Prefer date-fns over moment.",
	}})
	if err := os.WriteFile(filepath.Join(dir, "conventions.json"), body, 0o644); err != nil {
		t.Fatal(err)
	}
	l, _ := NewLoader(root)
	cs, err := l.LoadStack("Next.js")
	if err != nil {
		t.Fatal(err)
	}
	if len(cs) != 1 {
		t.Fatalf("got %d", len(cs))
	}
	if cs[0].Stack != "nextjs" {
		t.Errorf("stack=%q", cs[0].Stack)
	}
}

func TestLoadStackEnvelope(t *testing.T) {
	root := t.TempDir()
	body, _ := json.Marshal(map[string]any{
		"conventions": []types.ConventionCandidate{{
			Category: "Naming", RuleNL: "snake_case for column names",
		}},
	})
	if err := os.WriteFile(filepath.Join(root, "django.json"), body, 0o644); err != nil {
		t.Fatal(err)
	}
	l, _ := NewLoader(root)
	cs, _ := l.LoadStack("django")
	if len(cs) != 1 {
		t.Errorf("got %d", len(cs))
	}
}

func TestLoadStackMissingReturnsEmpty(t *testing.T) {
	root := t.TempDir()
	l, _ := NewLoader(root)
	cs, err := l.LoadStack("unknown")
	if err != nil {
		t.Fatal(err)
	}
	if len(cs) != 0 {
		t.Errorf("expected empty, got %d", len(cs))
	}
}

func TestLoadStacksDedupes(t *testing.T) {
	root := t.TempDir()
	body, _ := json.Marshal([]types.ConventionCandidate{{Category: "Naming", RuleNL: "X"}})
	_ = os.MkdirAll(filepath.Join(root, "a"), 0o755)
	_ = os.MkdirAll(filepath.Join(root, "b"), 0o755)
	_ = os.WriteFile(filepath.Join(root, "a", "conventions.json"), body, 0o644)
	_ = os.WriteFile(filepath.Join(root, "b", "conventions.json"), body, 0o644)
	l, _ := NewLoader(root)
	cs, _ := l.LoadStacks("a", "b")
	if len(cs) != 1 {
		t.Errorf("dedup failed: got %d", len(cs))
	}
}
