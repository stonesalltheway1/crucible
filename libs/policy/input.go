package policy

import (
	"time"

	cruciblev1 "github.com/crucible/sdk-go/crucible/v1"
)

// PromotionInput is the canonical document the default and tenant bundles
// evaluate against. The promotion gate builds this from a PromotionBundle +
// enrichment fields and serializes it to map[string]any before
// Engine.Evaluate.
//
// We export the struct so the gate, the slack-bot, and the SDK can build
// inputs without sprinkling string keys across the codebase.
type PromotionInput struct {
	TaskID                      string                              `json:"task_id"`
	TenantID                    string                              `json:"tenant_id"`
	DiffHash                    string                              `json:"diff_hash"`
	FilesChanged                []cruciblev1.FileChange             `json:"files_changed,omitempty"`
	VerifierApprovalAttestation string                              `json:"verifier_approval_attestation,omitempty"`
	BuildProvenanceAttestation  string                              `json:"build_provenance_attestation,omitempty"`
	RebuildHash                 string                              `json:"rebuild_hash,omitempty"`
	BlastRadius                 PromotionBlastRadius                `json:"blast_radius"`
	SuggestedRollout            cruciblev1.SuggestedRollout         `json:"suggested_rollout,omitempty"`
	TierResults                 PromotionTierResults                `json:"tier_results"`
	AgentOidcSubject            string                              `json:"agent_oidc_subject"`
	Approvals                   []ApprovalRecord                    `json:"approvals,omitempty"`
	Context                     PromotionContext                    `json:"context"`
	CodeOwners                  CodeOwnerMatch                      `json:"codeowners,omitempty"`
	TenantOverrides             map[string]any                      `json:"tenant_overrides,omitempty"`
}

// PromotionBlastRadius is the policy-input shape; it extends BlastRadius with
// the schema-change + critical-path enrichment fields the gate computes.
type PromotionBlastRadius struct {
	AffectedResources     []string                  `json:"affected_resources,omitempty"`
	AffectedServices      []string                  `json:"affected_services,omitempty"`
	AffectedEndpoints     []string                  `json:"affected_endpoints,omitempty"`
	SchemaChanges         []SchemaChangeEntry       `json:"schema_changes,omitempty"`
	CriticalPathsTouched  []string                  `json:"critical_paths_touched,omitempty"`
	EstimatedImpact       string                    `json:"estimated_impact"`
	Reversibility         cruciblev1.Reversibility  `json:"reversibility"`
	ImpactScore           float64                   `json:"impact_score"`
}

// SchemaChangeEntry is a single migration descriptor pulled from the
// MigrationAttestation chain.
type SchemaChangeEntry struct {
	File           string `json:"file"`
	DestructiveDDL bool   `json:"destructive_ddl"`
	AddedTables    int    `json:"added_tables"`
	DroppedTables  int    `json:"dropped_tables"`
}

type PromotionTierResults struct {
	Tier0 *TierEntry `json:"tier_0,omitempty"`
	Tier1 *TierEntry `json:"tier_1,omitempty"`
	Tier2 *TierEntry `json:"tier_2,omitempty"`
	Tier3 *TierEntry `json:"tier_3,omitempty"`
	Tier4 *TierEntry `json:"tier_4,omitempty"`
}

type TierEntry struct {
	Passed             bool   `json:"passed"`
	ReportAttestation  string `json:"report_attestation,omitempty"`
}

type ApprovalRecord struct {
	ApproverOidcSubject string    `json:"approver_oidc_subject"`
	Attestation         string    `json:"attestation"`
	ApprovedAt          time.Time `json:"approved_at"`
	Codeowner           bool      `json:"codeowner,omitempty"`
	Group               string    `json:"group,omitempty"`
}

type PromotionContext struct {
	MergeFreeze       bool      `json:"merge_freeze"`
	MergeFreezeUntil  time.Time `json:"merge_freeze_until,omitempty"`
	MergeFreezeReason string    `json:"merge_freeze_reason,omitempty"`
	Geo               string    `json:"geo,omitempty"`
	IsTestPromotion   bool      `json:"is_test_promotion"`
}

type CodeOwnerMatch struct {
	Matched []CodeOwnerRule `json:"matched,omitempty"`
}

type CodeOwnerRule struct {
	PathGlob string   `json:"path_glob"`
	Groups   []string `json:"groups"`
}
