// Package planbuilder constructs a Plan from a task description by calling
// the executor model (default Sonnet 4.6) with the right system prompt, then
// emits a PlanProposal attestation.
//
// The Plan shape is the contract in docs/03-sdk/agent-sdk-reference.md →
// `type Plan`. The plan_hash field is sha256 of the canonical JSON encoding
// of the rest of the plan; the same plan submitted twice produces the same
// hash, enabling the optimistic-concurrency check on PlanApproval.
package planbuilder

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/crucible/attestation"
	"github.com/crucible/control-plane/internal/modelrouter"
	cruciblev1 "github.com/crucible/sdk-go/crucible/v1"
)

const plannerSystemPrompt = `You are Crucible's Plan Builder. Given a user's task description, you produce a structured Plan that the user will approve before any code is written.

Return ONLY a JSON object with this exact shape (no markdown, no prose):

{
  "description":            string,
  "steps":                  [{ "ordinal": int, "description": string, "retry_budget": int }],
  "estimated_cost_usd":     number,
  "estimated_duration_min": int,
  "files_to_touch":         [string],
  "db_migrations":          int,
  "external_effects":       [{ "service": string, "endpoints": [string], "live": bool }],
  "top_risks":              [{ "description": string, "impact": "low"|"medium"|"high" }],
  "retry_budget_per_step":  3,
  "wall_clock_budget_min":  int
}

Rules:
- Default retry_budget_per_step to 3 (per ADR-009).
- estimated_cost_usd should reflect the executor + verifier costs (Sonnet 4.6 + Gemini 3.1 Pro pairing); round to 2 decimals.
- estimated_duration_min should be conservative (overshoot slightly).
- Steps are atomic; each should be one logical action.
- top_risks should call out anything an experienced reviewer would flag.
- files_to_touch is best-effort; empty array if you can't tell.

Do not produce any text outside the JSON object.`

// PlanProposalAttestor is the interface Builder needs to emit attestations.
// It matches *attestation.Service so callers wire the real signer.
type PlanProposalAttestor interface {
	Emit(ctx context.Context, predicateType, subjectName string, subjectContent []byte, predicate any) (*cruciblev1.RekorEntry, error)
	Signer() attestation.Signer
}

// Builder is the plan-construction service.
type Builder struct {
	mr      *modelrouter.Router
	attest  PlanProposalAttestor
	model   string
}

// New constructs a Builder. model defaults to claude-sonnet-4-6.
func New(mr *modelrouter.Router, attest PlanProposalAttestor, model string) *Builder {
	if model == "" {
		model = "claude-sonnet-4-6"
	}
	return &Builder{mr: mr, attest: attest, model: model}
}

// Build calls the LLM, parses the plan JSON, computes plan_hash, and emits a
// PlanProposal attestation. Falls back to a deterministic skeleton plan if no
// LLM is available so the control plane is testable offline.
func (b *Builder) Build(ctx context.Context, task *cruciblev1.Task) (*cruciblev1.Plan, *cruciblev1.RekorEntry, error) {
	if task == nil {
		return nil, nil, errors.New("planbuilder: nil task")
	}

	plan, err := b.callOrFallback(ctx, task)
	if err != nil {
		return nil, nil, err
	}
	plan.TaskID = task.ID
	plan.BuiltAt = time.Now().UTC()
	if plan.RetryBudgetPerStep == 0 {
		plan.RetryBudgetPerStep = 3
	}
	if plan.WallClockBudgetMin == 0 {
		plan.WallClockBudgetMin = 60
	}
	plan.Complexity = inferComplexityFromTask(task)
	plan.PlanHash = computePlanHash(plan)

	// Emit a PlanProposal attestation.
	subject := []byte(plan.PlanHash)
	predicate := cruciblev1.PlanProposalAttestation{
		TaskID:               task.ID,
		TenantID:             task.TenantID,
		PlanHash:             plan.PlanHash,
		EstimatedCostUsd:     plan.EstimatedCostUsd,
		EstimatedDurationMin: plan.EstimatedDurationMin,
		Complexity:           plan.Complexity,
		StepCount:            uint32(len(plan.Steps)),
		BuiltByOidc:          b.attest.Signer().OidcSubject(),
		BuiltAt:              plan.BuiltAt,
	}
	entry, err := b.attest.Emit(ctx,
		cruciblev1.PredicatePlanProposal,
		fmt.Sprintf("task/%s/plan-proposal", task.ID),
		subject,
		predicate,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("planbuilder: emit attestation: %w", err)
	}
	return plan, entry, nil
}

func (b *Builder) callOrFallback(ctx context.Context, task *cruciblev1.Task) (*cruciblev1.Plan, error) {
	if b.mr == nil {
		return fallbackPlan(task), nil
	}
	spec, err := modelrouter.Lookup(b.model)
	if err != nil {
		return fallbackPlan(task), nil
	}
	if !modelrouter.HasEnv(spec.Vendor) {
		return fallbackPlan(task), nil
	}

	req := modelrouter.Request{
		Model:       b.model,
		System:      plannerSystemPrompt,
		CacheSystem: true, // 1h cache
		Messages: []modelrouter.Message{
			{Role: modelrouter.RoleUser, Content: "Task: " + task.Description + "\nRepo: " + task.Repo + "\nBase SHA: " + task.BaseSha},
		},
		MaxOutput: 2000,
		JSONMode:  true,
	}
	resp, err := b.mr.Call(ctx, req)
	if err != nil {
		return fallbackPlan(task), nil
	}
	plan, err := parsePlanJSON(resp.Content)
	if err != nil {
		return fallbackPlan(task), nil
	}
	if plan.EstimatedCostUsd == 0 {
		plan.EstimatedCostUsd = modelrouter.EstimateCostUSD(b.model, resp.Usage) * 8
	}
	return plan, nil
}

func parsePlanJSON(s string) (*cruciblev1.Plan, error) {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```json")
		s = strings.TrimPrefix(s, "```")
		s = strings.TrimSuffix(s, "```")
		s = strings.TrimSpace(s)
	}
	var p cruciblev1.Plan
	if err := json.Unmarshal([]byte(s), &p); err != nil {
		return nil, fmt.Errorf("planbuilder: parse plan: %w", err)
	}
	if p.Description == "" {
		return nil, errors.New("planbuilder: plan missing description")
	}
	return &p, nil
}

func fallbackPlan(task *cruciblev1.Task) *cruciblev1.Plan {
	return &cruciblev1.Plan{
		Description: task.Description,
		Steps: []cruciblev1.PlanStep{
			{Ordinal: 1, Description: "STUB: read existing relevant files", RetryBudget: 3},
			{Ordinal: 2, Description: "STUB: implement the change", RetryBudget: 3},
			{Ordinal: 3, Description: "STUB: author tests and run verifier", RetryBudget: 3},
		},
		EstimatedCostUsd:     0.50,
		EstimatedDurationMin: 10,
		FilesToTouch:         []string{},
		DbMigrations:         0,
		ExternalEffects:      []cruciblev1.ExternalEffect{},
		TopRisks: []cruciblev1.Risk{
			{Description: "STUB plan generated without LLM (no ANTHROPIC_API_KEY)", Impact: "low"},
		},
		RetryBudgetPerStep: 3,
		WallClockBudgetMin: 30,
	}
}

func inferComplexityFromTask(task *cruciblev1.Task) cruciblev1.Complexity {
	if task.Routing == nil {
		return cruciblev1.ComplexityStandard
	}
	// Go SDK ModelTier values are 0..4 (matching the SDK reference doc's
	// tier numbering). The proto enum prepends UNSPECIFIED so its int wire
	// values are 1..5; we use the Go alias on this side.
	switch task.Routing.ExecutorTier {
	case cruciblev1.ModelTier0:
		return cruciblev1.ComplexityTrivial
	case cruciblev1.ModelTier1:
		return cruciblev1.ComplexityStandard
	case cruciblev1.ModelTier2:
		if task.Routing.IsCritical {
			return cruciblev1.ComplexityCritical
		}
		return cruciblev1.ComplexityComplex
	default:
		return cruciblev1.ComplexityStandard
	}
}

// computePlanHash returns sha256(canonicalJSON(plan-minus-hash)).
// The hash field is omitted; otherwise we'd hash a hash.
func computePlanHash(p *cruciblev1.Plan) string {
	clone := *p
	clone.PlanHash = ""
	clone.BuiltAt = time.Time{} // exclude wall-clock from the hash so identical plans deterministically match
	b, _ := json.Marshal(&clone)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
