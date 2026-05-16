// Package memoryspec consolidates Phase-5 memory-layer types. Agent-visible
// types (Convention, Memory, ComplianceReport) are re-exported from the
// sdk-go cruciblev1 package; this package adds the server-internal types
// the memory-router, distiller, and cartographer use.
//
// Disk-serialization JSON tags are snake_case and match the JSON Schemas
// in libs/memory-spec/schemas/. The hand-rolled Go is in lock-step with
// libs/memory-spec/proto/crucible/v1/*.proto; the proto files are the
// long-term source of truth once `buf generate` is wired into CI.
package memoryspec

import (
	"errors"
	"fmt"
	"time"

	cruciblev1 "github.com/crucible/sdk-go/crucible/v1"
)

// ─── Layering ───────────────────────────────────────────────────────────────

// MemoryLayer is the three-tier layering used by the retrieval router.
type MemoryLayer string

const (
	// LayerGlobalDefaults is the OSS-derived bundle layer. Read-only,
	// shared across all tenants, never carries customer-private data.
	LayerGlobalDefaults MemoryLayer = "global_defaults"
	// LayerOrgOverrides is tenant-private convention storage.
	LayerOrgOverrides MemoryLayer = "org_overrides"
	// LayerRepoOverrides is per-repo, lowest priority on writes but
	// highest priority on reads (bottom-up retrieval).
	LayerRepoOverrides MemoryLayer = "repo_overrides"
)

// ReadOrder returns the layers in the order the retrieval router reads
// them (lowest priority first; later layers override earlier).
func ReadOrder() []MemoryLayer {
	return []MemoryLayer{LayerGlobalDefaults, LayerOrgOverrides, LayerRepoOverrides}
}

// Priority returns a numeric layer priority where higher beats lower.
// Used by the merge step that resolves duplicate convention ids across
// layers.
func (l MemoryLayer) Priority() int {
	switch l {
	case LayerGlobalDefaults:
		return 1
	case LayerOrgOverrides:
		return 2
	case LayerRepoOverrides:
		return 3
	}
	return 0
}

// ─── Convention taxonomy ────────────────────────────────────────────────────

// ConventionCategory is the 12-bucket taxonomy from
// docs/01-architecture/memory-layer.md §"Convention taxonomy".
type ConventionCategory string

const (
	CatNaming               ConventionCategory = "Naming"
	CatLayering             ConventionCategory = "Layering"
	CatLibraryPreferences   ConventionCategory = "LibraryPreferences"
	CatTestPatterns         ConventionCategory = "TestPatterns"
	CatErrorHandling        ConventionCategory = "ErrorHandling"
	CatLogging              ConventionCategory = "Logging"
	CatMigrationPatterns    ConventionCategory = "MigrationPatterns"
	CatPrCommitHygiene      ConventionCategory = "PrCommitHygiene"
	CatSecurityDefaults     ConventionCategory = "SecurityDefaults"
	CatPerformanceDefaults  ConventionCategory = "PerformanceDefaults"
	CatConcurrency          ConventionCategory = "Concurrency"
	CatApiShape             ConventionCategory = "ApiShape"
)

// AllCategories is the canonical iteration order.
func AllCategories() []ConventionCategory {
	return []ConventionCategory{
		CatNaming, CatLayering, CatLibraryPreferences, CatTestPatterns,
		CatErrorHandling, CatLogging, CatMigrationPatterns, CatPrCommitHygiene,
		CatSecurityDefaults, CatPerformanceDefaults, CatConcurrency, CatApiShape,
	}
}

// ValidCategory reports whether a string is a known taxonomy bucket.
// Admission rejects candidates with an unknown category — this is the
// "ban category=other" gate from the brief.
func ValidCategory(c string) bool {
	for _, k := range AllCategories() {
		if string(k) == c {
			return true
		}
	}
	return false
}

// ConventionStatus mirrors the lifecycle states a convention progresses
// through. Mapped 1:1 with cruciblev1.ConventionStatus but with the
// additional pre-admission states (candidate, suggested) the distiller
// uses internally.
type ConventionStatus string

const (
	StatusActive     ConventionStatus = "active"
	StatusDrifting   ConventionStatus = "drifting"
	StatusSuperseded ConventionStatus = "superseded"
	StatusRejected   ConventionStatus = "rejected"
	// StatusCandidate is the invisible bucket: judge passed, agreement
	// below surface threshold (0.25..0.4). Persisted; not surfaced.
	StatusCandidate ConventionStatus = "candidate"
	// StatusSuggested is the medium-confidence bucket (0.4..0.7).
	// Surfaced as "suggestion" in the web console.
	StatusSuggested ConventionStatus = "suggested"
)

// ─── Stacks ─────────────────────────────────────────────────────────────────

// Stack is the per-stack bundle identifier.
type Stack string

const (
	StackNextJS      Stack = "nextjs"
	StackDjango      Stack = "django"
	StackFastAPI     Stack = "fastapi"
	StackFlask       Stack = "flask"
	StackRails       Stack = "rails"
	StackSpringBoot  Stack = "spring_boot"
	StackGoServices  Stack = "go_services"
	StackRustServices Stack = "rust_services"
	StackPhoenix     Stack = "phoenix_elixir"
	StackVue         Stack = "vue"
	StackExpress     Stack = "express"
	StackLaravel     Stack = "laravel"
)

// AllStacks is the canonical 12-stack list.
func AllStacks() []Stack {
	return []Stack{
		StackNextJS, StackDjango, StackFastAPI, StackFlask, StackRails,
		StackSpringBoot, StackGoServices, StackRustServices, StackPhoenix,
		StackVue, StackExpress, StackLaravel,
	}
}

// ─── Source channels ────────────────────────────────────────────────────────

// SourceChannel is the upstream system a distiller adapter is connected
// to.
type SourceChannel string

const (
	ChannelGitHubPRReview    SourceChannel = "github_pr_review"
	ChannelGitHubSquashMerge SourceChannel = "github_squash_merge"
	ChannelIncidentExport    SourceChannel = "incident_export"
	ChannelSlackIncidents    SourceChannel = "slack_incidents"
	ChannelConfluencePage    SourceChannel = "confluence_page"
	ChannelNotionPage        SourceChannel = "notion_page"
	ChannelAdrFile           SourceChannel = "adr_file"
	ChannelLintConfig        SourceChannel = "lint_config"
)

// ─── Convention (disk shape) ────────────────────────────────────────────────

// SourceRef re-exports the SDK type so memoryspec callers don't need to
// import cruciblev1 separately.
type SourceRef = cruciblev1.SourceRef

// ScopeFilter re-exports the SDK type.
type ScopeFilter = cruciblev1.ScopeFilter

// Convention is the disk-serialized shape (matches schemas/convention_v1.json).
// The wire-format equivalent for SDK callers is cruciblev1.Convention; the
// difference is this type carries layer + lifecycle metadata the router
// uses internally.
type Convention struct {
	ID               string             `json:"id"`
	TenantID         string             `json:"tenant_id"`
	Layer            MemoryLayer        `json:"layer,omitempty"`
	Scope            ScopeFilter        `json:"scope"`
	RuleNl           string             `json:"rule_nl"`
	RuleMachine      string             `json:"rule_machine,omitempty"`
	Category         ConventionCategory `json:"category"`
	Status           ConventionStatus   `json:"status"`
	Confidence       float64            `json:"confidence"`
	JudgeScore       float64            `json:"judge_score,omitempty"`
	JudgeRationale   string             `json:"judge_rationale,omitempty"`
	PositiveExamples []SourceRef        `json:"positive_examples,omitempty"`
	NegativeExamples []SourceRef        `json:"negative_examples,omitempty"`
	SourceEvidence   []SourceRef        `json:"source_evidence,omitempty"`
	FirstSeen        time.Time          `json:"first_seen"`
	LastReinforced   time.Time          `json:"last_reinforced,omitempty"`
	LastViolated     *time.Time         `json:"last_violated,omitempty"`
	ValidFrom        time.Time          `json:"valid_from"`
	ValidTo          *time.Time         `json:"valid_to,omitempty"`
	Supersedes       []string           `json:"supersedes,omitempty"`
	WriterSubject    string             `json:"writer_oidc_subject,omitempty"`
	WrittenAt        time.Time          `json:"written_at"`
	StackTag         string             `json:"stack_tag,omitempty"`
	// AnonymizedForm is set when this convention is eligible for
	// cross-tenant federation graduation (≥5 tenants in same categorical
	// form). Phase 5 records but does not act on this field — actual
	// graduation fires in v2 Phase 10.
	AnonymizedForm string `json:"anonymized_form,omitempty"`
}

// Validate enforces the brief's "ban category=other" admission rule plus
// basic structural checks.
func (c *Convention) Validate() error {
	if c.ID == "" {
		return errors.New("convention: id required")
	}
	if c.TenantID == "" {
		return errors.New("convention: tenant_id required")
	}
	if c.RuleNl == "" || len(c.RuleNl) > 1024 {
		return fmt.Errorf("convention %s: rule_nl must be 1..1024 chars", c.ID)
	}
	if !ValidCategory(string(c.Category)) {
		return fmt.Errorf("convention %s: invalid category %q (must be one of the 12 taxonomy buckets)", c.ID, c.Category)
	}
	if c.Confidence < 0 || c.Confidence > 1 {
		return fmt.Errorf("convention %s: confidence %.3f out of [0,1]", c.ID, c.Confidence)
	}
	if c.JudgeScore < 0 || c.JudgeScore > 1 {
		return fmt.Errorf("convention %s: judge_score %.3f out of [0,1]", c.ID, c.JudgeScore)
	}
	switch c.Status {
	case StatusActive, StatusDrifting, StatusSuperseded, StatusRejected,
		StatusCandidate, StatusSuggested:
	default:
		return fmt.Errorf("convention %s: invalid status %q", c.ID, c.Status)
	}
	if c.Layer != "" {
		switch c.Layer {
		case LayerGlobalDefaults, LayerOrgOverrides, LayerRepoOverrides:
		default:
			return fmt.Errorf("convention %s: invalid layer %q", c.ID, c.Layer)
		}
	}
	return nil
}

// ToWire converts to the cruciblev1 SDK-visible type. Loses the layer +
// candidate/suggested status info; those are server-internal.
func (c *Convention) ToWire() cruciblev1.Convention {
	out := cruciblev1.Convention{
		ID:                c.ID,
		TenantID:          c.TenantID,
		Scope:             c.Scope,
		RuleNl:            c.RuleNl,
		Category:          string(c.Category),
		Confidence:        c.Confidence,
		JudgeScore:        c.JudgeScore,
		SourceEvidence:    c.SourceEvidence,
		ValidFrom:         c.ValidFrom,
		ValidTo:           c.ValidTo,
		WriterOidcSubject: c.WriterSubject,
		WrittenAt:         c.WrittenAt,
	}
	// SDK only knows the 4 SDK-visible status values; collapse internal
	// ones onto "active" with confidence carrying the calibration signal.
	switch c.Status {
	case StatusActive, StatusSuggested, StatusCandidate:
		out.Status = cruciblev1.ConvActive
	case StatusDrifting:
		out.Status = cruciblev1.ConvDrifting
	case StatusSuperseded:
		out.Status = cruciblev1.ConvSuperseded
	case StatusRejected:
		out.Status = cruciblev1.ConvRejected
	}
	if len(c.Supersedes) > 0 {
		out.Supersedes = c.Supersedes[0]
	}
	return out
}

// ─── Per-stack bundle ───────────────────────────────────────────────────────

// BundleLicense captures the license-filter audit of the inputs that
// produced a bundle. Refuses to ship a bundle whose inputs included any
// GPL/AGPL/SSPL/BUSL repos.
type BundleLicense struct {
	SafeForRedistribution bool     `json:"safe_for_redistribution"`
	InputLicensesSeen     []string `json:"input_licenses_seen,omitempty"`
	ExcludedLicenses      []string `json:"excluded_licenses,omitempty"`
	AttributionFile       string   `json:"attribution_file,omitempty"`
}

// BundleStats reports counts at each pipeline stage. Used by the
// PHASE-5-REPORT per-stack rule count table.
type BundleStats struct {
	ReposExamined    int `json:"repos_examined,omitempty"`
	ConfigsParsed    int `json:"configs_parsed,omitempty"`
	AgentsMdParsed   int `json:"agents_md_parsed,omitempty"`
	PrCommentsMined  int `json:"pr_comments_mined,omitempty"`
	AdrsParsed       int `json:"adrs_parsed,omitempty"`
	RawCandidates    int `json:"raw_candidates,omitempty"`
	PostJudge        int `json:"post_judge,omitempty"`
	PostAgreement    int `json:"post_agreement,omitempty"`
	ActiveRules      int `json:"active_rules,omitempty"`
	SuggestedRules   int `json:"suggested_rules,omitempty"`
	CandidateRules   int `json:"candidate_rules,omitempty"`
}

// PerStackBundle is the disk shape of a default-rules file shipped at
// services/memory-router/global_defaults/<stack>.json.
type PerStackBundle struct {
	BundleVersion   string         `json:"bundle_version"`
	Stack           Stack          `json:"stack"`
	GeneratedAt     time.Time      `json:"generated_at"`
	GeneratorCommit string         `json:"generator_commit,omitempty"`
	License         BundleLicense  `json:"license"`
	Stats           BundleStats    `json:"stats"`
	Conventions     []Convention   `json:"conventions"`
}

// Validate checks the bundle is shippable: license-safe, version 1, and
// all conventions pass individual Validate.
func (b *PerStackBundle) Validate() error {
	if b.BundleVersion != "1" {
		return fmt.Errorf("bundle: only bundle_version=1 supported, got %q", b.BundleVersion)
	}
	if !b.License.SafeForRedistribution {
		return fmt.Errorf("bundle %s: license.safe_for_redistribution must be true to ship", b.Stack)
	}
	for i := range b.Conventions {
		if err := b.Conventions[i].Validate(); err != nil {
			return fmt.Errorf("bundle %s convention[%d]: %w", b.Stack, i, err)
		}
		if b.Conventions[i].Layer != "" && b.Conventions[i].Layer != LayerGlobalDefaults {
			return fmt.Errorf("bundle %s convention[%d]: must be layer=global_defaults, got %q", b.Stack, i, b.Conventions[i].Layer)
		}
	}
	return nil
}

// ─── Retrieval ──────────────────────────────────────────────────────────────

// RetrievalQuery is the memory-router input.
type RetrievalQuery struct {
	TenantID          string
	TaskID            string
	Query             string
	Scope             ScopeFilter
	MaxTokens         uint32
	MaxItems          uint32
	IncludeHot        bool
	IncludeEpisodic   bool
	IncludeSemantic   bool
	IncludeProcedural bool
}

// ScoredMemory wraps a Memory with provenance + score breakdown.
type ScoredMemory struct {
	Memory           cruciblev1.Memory `json:"memory"`
	Layer            MemoryLayer       `json:"layer"`
	SemanticScore    float64           `json:"semantic_score"`
	ImportanceScore  float64           `json:"importance_score"`
	FinalScore       float64           `json:"final_score"`
	TokenEstimate    uint32            `json:"token_estimate"`
}

// RetrievalResult is the memory-router output.
type RetrievalResult struct {
	Memories         []ScoredMemory `json:"memories"`
	TokensUsed       uint32         `json:"tokens_used"`
	BudgetTokens     uint32         `json:"budget_tokens"`
	ItemsConsidered  uint32         `json:"items_considered"`
	ItemsReturned    uint32         `json:"items_returned"`
	LatencyMs        uint32         `json:"latency_ms"`
}

// ─── Admission + drift ─────────────────────────────────────────────────────

// AdmissionScore is the A-MAC composite. composite = utility * confidence *
// novelty * recency * content_prior, clamped to [0, 1].
type AdmissionScore struct {
	Utility       float64 `json:"utility"`
	Confidence    float64 `json:"confidence"`
	Novelty       float64 `json:"novelty"`
	Recency       float64 `json:"recency"`
	ContentPrior  float64 `json:"content_prior"`
	Composite     float64 `json:"composite"`
	Admitted      bool    `json:"admitted"`
	Threshold     string  `json:"admission_threshold_label"`
}

// ConventionDrift is what the 30-day drift detector emits.
type ConventionDrift struct {
	ConventionID  string    `json:"convention_id"`
	TenantID      string    `json:"tenant_id"`
	Positives30d  uint32    `json:"positives_30d"`
	Negatives30d  uint32    `json:"negatives_30d"`
	Ratio         float64   `json:"ratio"`
	Threshold     float64   `json:"threshold"`
	DetectedAt    time.Time `json:"detected_at"`
	SuggestedAction string  `json:"suggested_action"`
}

// FederationGraduation records when a rule satisfies the ≥5-tenant
// categorical-form policy. Phase 5 wires the data model only; the
// graduation engine fires in v2 Phase 10.
type FederationGraduation struct {
	AnonymizedRuleID         string             `json:"anonymized_rule_id"`
	Category                 ConventionCategory `json:"category"`
	CanonicalFormNl          string             `json:"canonical_form_nl"`
	DistinctTenantCount      uint32             `json:"distinct_tenant_count"`
	ContributingConventionIDs []string          `json:"contributing_convention_ids"`
	EligibleAt               time.Time          `json:"eligible_at"`
	Fired                    bool               `json:"fired"`
	PromotedToLayer          string             `json:"promoted_to_layer,omitempty"`
}

// MinTenantsForGraduation is the policy threshold. Documented in
// docs/01-architecture/memory-layer.md §"Cross-tenant federation".
const MinTenantsForGraduation = 5
