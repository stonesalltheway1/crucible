// Package spec defines the on-disk schema for cth case specs.
package spec

// Case is one CTH test case.
type Case struct {
	ID                  string             `json:"id,omitempty"`
	Dir                 string             `json:"dir,omitempty"`
	Category            string             `json:"category,omitempty"`
	Description         string             `json:"description"`
	Stack               string             `json:"stack"`
	TaskDescription     string             `json:"task_description"`
	BudgetUSDMax        float64            `json:"budget_usd_max"`
	WallClockMinMax     int                `json:"wall_clock_min_max"`
	ExpectVerifierVerdict string           `json:"expect_verifier_verdict"`           // "approved" | "rejected" | "either"
	ExpectTier3Triggered bool              `json:"expect_tier3_triggered,omitempty"`
	ExpectDestructiveOpGate bool           `json:"expect_destructive_op_gate,omitempty"`
	ExpectCacheHitRateMin  float64         `json:"expect_cache_hit_rate_min,omitempty"`
	AssertGitDiff       []GitDiffAssertion `json:"assert_git_diff,omitempty"`
	AssertNoSecurityIncidents bool          `json:"assert_no_security_incidents"`
	Adversarial         AdversarialSpec    `json:"adversarial,omitempty"`
}

// GitDiffAssertion is a single property the post-agent diff must satisfy.
type GitDiffAssertion struct {
	Path      string   `json:"path"`
	Touched   bool     `json:"touched"`           // file MUST be touched
	NotTouched bool    `json:"not_touched"`       // file MUST NOT be touched
	MustContain []string `json:"must_contain"`
	MustNotContain []string `json:"must_not_contain"`
}

// AdversarialSpec captures adversarial-only fields.
type AdversarialSpec struct {
	AttackVector string `json:"attack_vector"` // free-text categorisation
	ExpectAgent  string `json:"expect_agent"`  // "refuse" | "halt" | "report-to-operator"
}
