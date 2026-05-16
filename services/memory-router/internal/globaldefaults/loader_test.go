package globaldefaults

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	memoryspec "github.com/crucible/memory-spec/go"
	cruciblev1 "github.com/crucible/sdk-go/crucible/v1"
)

func writeBundle(t *testing.T, dir string, stack memoryspec.Stack, b memoryspec.PerStackBundle) {
	t.Helper()
	path := filepath.Join(dir, string(stack)+".json")
	data, _ := json.MarshalIndent(b, "", "  ")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func mkBundle(stack memoryspec.Stack, ruleCount int) memoryspec.PerStackBundle {
	now := time.Now().UTC()
	b := memoryspec.PerStackBundle{
		BundleVersion: "1",
		Stack:         stack,
		GeneratedAt:   now,
		License:       memoryspec.BundleLicense{SafeForRedistribution: true, InputLicensesSeen: []string{"MIT", "Apache-2.0"}},
	}
	for i := 0; i < ruleCount; i++ {
		b.Conventions = append(b.Conventions, memoryspec.Convention{
			ID:         "conv_global_" + string(stack) + "_" + string(rune('A'+i)),
			TenantID:   "global",
			Layer:      memoryspec.LayerGlobalDefaults,
			Scope:      cruciblev1.ScopeFilter{Category: "Logging"},
			RuleNl:     "rule",
			Category:   memoryspec.CatLogging,
			Status:     memoryspec.StatusActive,
			Confidence: 0.6,
			ValidFrom:  now,
			WrittenAt:  now,
		})
	}
	return b
}

func TestLoadAll_LoadsKnownStacks(t *testing.T) {
	dir := t.TempDir()
	writeBundle(t, dir, memoryspec.StackNextJS, mkBundle(memoryspec.StackNextJS, 3))
	writeBundle(t, dir, memoryspec.StackFastAPI, mkBundle(memoryspec.StackFastAPI, 2))

	l := NewLoader()
	n, errs := l.LoadAll(dir)
	if n != 2 || len(errs) != 0 {
		t.Fatalf("expected 2 loaded, 0 errors; got %d / %v", n, errs)
	}
	all := l.ConventionsForStacks(memoryspec.StackNextJS, memoryspec.StackFastAPI)
	if len(all) != 5 {
		t.Fatalf("expected 5 active conventions; got %d", len(all))
	}
}

func TestLoadAll_RejectsBadLicense(t *testing.T) {
	dir := t.TempDir()
	b := mkBundle(memoryspec.StackNextJS, 1)
	b.License.SafeForRedistribution = false
	writeBundle(t, dir, memoryspec.StackNextJS, b)
	l := NewLoader()
	n, errs := l.LoadAll(dir)
	if n != 0 {
		t.Fatalf("license-unsafe bundle must NOT load; got n=%d", n)
	}
	if len(errs) == 0 {
		t.Fatal("expected error reporting unsafe bundle")
	}
}

func TestLoadAll_HandlesMissingDir(t *testing.T) {
	l := NewLoader()
	n, errs := l.LoadAll(filepath.Join(t.TempDir(), "nonexistent"))
	if n != 0 || len(errs) != 1 {
		t.Fatalf("missing dir → 0 loaded + 1 informational error; got %d / %d errs", n, len(errs))
	}
}

func TestConventionsForStacks_ForcesGlobalLayer(t *testing.T) {
	dir := t.TempDir()
	b := mkBundle(memoryspec.StackGoServices, 1)
	// Corrupt the layer to verify the loader forces it back.
	b.Conventions[0].Layer = memoryspec.LayerOrgOverrides
	writeBundle(t, dir, memoryspec.StackGoServices, b)

	// Manually load (skip Validate since the bundle would reject).
	l := NewLoader()
	l.bundles[memoryspec.StackGoServices] = b
	out := l.ConventionsForStacks(memoryspec.StackGoServices)
	if len(out) != 1 || out[0].Layer != memoryspec.LayerGlobalDefaults {
		t.Fatal("loader must force layer=global_defaults on emit")
	}
}
