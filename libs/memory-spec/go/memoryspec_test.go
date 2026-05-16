package memoryspec

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	cruciblev1 "github.com/crucible/sdk-go/crucible/v1"
)

func validConvention(t *testing.T) Convention {
	t.Helper()
	now := time.Now().UTC()
	return Convention{
		ID:         "conv_01HQABCDEFGHJKLMNPQRSTVWX",
		TenantID:   "ten_test",
		Layer:      LayerOrgOverrides,
		Scope:      cruciblev1.ScopeFilter{FileGlob: "api/**/*.ts"},
		RuleNl:     "Use date-fns; don't introduce moment.js.",
		Category:   CatLibraryPreferences,
		Status:     StatusActive,
		Confidence: 0.74,
		JudgeScore: 0.91,
		FirstSeen:  now.Add(-30 * 24 * time.Hour),
		ValidFrom:  now.Add(-30 * 24 * time.Hour),
		WrittenAt:  now,
	}
}

func TestConventionValidate_OK(t *testing.T) {
	c := validConvention(t)
	if err := c.Validate(); err != nil {
		t.Fatalf("want ok, got %v", err)
	}
}

func TestConventionValidate_RejectsUnknownCategory(t *testing.T) {
	c := validConvention(t)
	c.Category = "Other"
	err := c.Validate()
	if err == nil || !strings.Contains(err.Error(), "invalid category") {
		t.Fatalf("want invalid-category error, got %v", err)
	}
}

func TestConventionValidate_RejectsBadConfidence(t *testing.T) {
	c := validConvention(t)
	c.Confidence = 1.2
	err := c.Validate()
	if err == nil || !strings.Contains(err.Error(), "confidence") {
		t.Fatalf("want confidence error, got %v", err)
	}
}

func TestConventionValidate_RejectsBadLayer(t *testing.T) {
	c := validConvention(t)
	c.Layer = "weird"
	err := c.Validate()
	if err == nil || !strings.Contains(err.Error(), "invalid layer") {
		t.Fatalf("want invalid-layer error, got %v", err)
	}
}

func TestConventionValidate_RejectsTooLongRule(t *testing.T) {
	c := validConvention(t)
	c.RuleNl = strings.Repeat("x", 1025)
	if err := c.Validate(); err == nil {
		t.Fatal("want length error")
	}
}

func TestLayerPriority_OrdersGlobalLowest(t *testing.T) {
	if LayerGlobalDefaults.Priority() >= LayerOrgOverrides.Priority() {
		t.Fatal("global_defaults must be lowest priority")
	}
	if LayerOrgOverrides.Priority() >= LayerRepoOverrides.Priority() {
		t.Fatal("org_overrides must be lower priority than repo_overrides")
	}
}

func TestReadOrder_BottomUp(t *testing.T) {
	order := ReadOrder()
	if order[0] != LayerGlobalDefaults || order[1] != LayerOrgOverrides || order[2] != LayerRepoOverrides {
		t.Fatalf("read order must be global, org, repo (bottom-up); got %v", order)
	}
}

func TestAllCategories_Count(t *testing.T) {
	if got := len(AllCategories()); got != 12 {
		t.Fatalf("taxonomy must have 12 buckets, got %d", got)
	}
}

func TestAllStacks_Count(t *testing.T) {
	if got := len(AllStacks()); got != 12 {
		t.Fatalf("must have 12 stacks, got %d", got)
	}
}

func TestPerStackBundle_RefusesGPLInputs(t *testing.T) {
	b := PerStackBundle{
		BundleVersion: "1",
		Stack:         StackNextJS,
		GeneratedAt:   time.Now().UTC(),
		License: BundleLicense{
			SafeForRedistribution: false, // GPL inputs flipped this off
			ExcludedLicenses:      []string{"GPL-3.0"},
		},
		Conventions: nil,
	}
	if err := b.Validate(); err == nil {
		t.Fatal("must refuse to validate license-unsafe bundle")
	}
}

func TestPerStackBundle_RequiresGlobalLayer(t *testing.T) {
	c := validConvention(t)
	c.Layer = LayerOrgOverrides
	b := PerStackBundle{
		BundleVersion: "1",
		Stack:         StackNextJS,
		GeneratedAt:   time.Now().UTC(),
		License:       BundleLicense{SafeForRedistribution: true},
		Conventions:   []Convention{c},
	}
	if err := b.Validate(); err == nil {
		t.Fatal("bundle must reject non-global_defaults convention")
	}
}

func TestPerStackBundle_HappyPath(t *testing.T) {
	c := validConvention(t)
	c.Layer = LayerGlobalDefaults
	c.TenantID = "global"
	b := PerStackBundle{
		BundleVersion: "1",
		Stack:         StackFastAPI,
		GeneratedAt:   time.Now().UTC(),
		License:       BundleLicense{SafeForRedistribution: true},
		Conventions:   []Convention{c},
	}
	if err := b.Validate(); err != nil {
		t.Fatalf("happy bundle validate: %v", err)
	}
	// JSON round-trip preserves layer.
	raw, _ := json.Marshal(b)
	var got PerStackBundle
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Conventions[0].Layer != LayerGlobalDefaults {
		t.Fatalf("layer lost: %q", got.Conventions[0].Layer)
	}
}

func TestToWire_MapsCandidateToActive(t *testing.T) {
	c := validConvention(t)
	c.Status = StatusCandidate
	w := c.ToWire()
	if w.Status != cruciblev1.ConvActive {
		t.Fatalf("candidate must map to active on wire (server-internal status hidden), got %q", w.Status)
	}
}

func TestValidCategory_AllTwelve(t *testing.T) {
	for _, c := range AllCategories() {
		if !ValidCategory(string(c)) {
			t.Fatalf("category %q rejected by ValidCategory", c)
		}
	}
	if ValidCategory("Other") {
		t.Fatal("category=Other must be rejected (admission gate)")
	}
}
