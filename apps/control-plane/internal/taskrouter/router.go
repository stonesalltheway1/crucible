// Package taskrouter classifies an incoming Task description into a Routing
// decision (complexity, executor model, verifier model, critical-path score).
//
// Classification is a cheap Haiku-4.5 call. We prompt-cache the system block
// at the 1h slot so per-task classification cost stays near zero across a
// session.
package taskrouter

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/crucible/control-plane/internal/modelrouter"
	cruciblev1 "github.com/crucible/sdk-go/crucible/v1"
)

const classifierSystemPrompt = `You are Crucible's Task Router. You classify incoming coding-task descriptions into a JSON object with these fields:

  complexity:      "trivial" | "standard" | "complex" | "critical" | "modernization"
  critical_score:  integer 0..100  (how likely this touches a @critical path)
  rationale:       one short sentence explaining the call
  suggested_files: array of strings (empty if you can't tell)

Definitions:
  - "trivial"        → typo / rename / one-line change
  - "standard"       → multi-file feature edit; new endpoint; test author
  - "complex"        → refactor across modules; new architecture; invariant authoring
  - "critical"       → touches auth / billing / migrations / crypto / consensus paths
  - "modernization"  → multi-week rewrites (rare)

Respond ONLY with the JSON object. No prose, no markdown fences.`

// Classification is the structured output the classifier returns.
type Classification struct {
	Complexity     string   `json:"complexity"`
	CriticalScore  int      `json:"critical_score"`
	Rationale      string   `json:"rationale"`
	SuggestedFiles []string `json:"suggested_files"`
}

// Router classifies tasks and decides executor/verifier pairing.
type Router struct {
	mr             *modelrouter.Router
	classifierModel string
}

// New constructs a Router. classifierModel defaults to claude-haiku-4-5.
func New(mr *modelrouter.Router, classifierModel string) *Router {
	if classifierModel == "" {
		classifierModel = "claude-haiku-4-5"
	}
	return &Router{mr: mr, classifierModel: classifierModel}
}

// Classify calls the classifier LLM and returns a Classification.
// Falls back to a heuristic if no Anthropic vendor is registered.
func (r *Router) Classify(ctx context.Context, description string) (Classification, error) {
	if r.mr == nil {
		return heuristicClassify(description), nil
	}
	if _, err := modelrouter.Lookup(r.classifierModel); err != nil {
		return heuristicClassify(description), nil
	}
	// Skip the LLM call when no vendor is wired (e.g. test runs without keys).
	spec, _ := modelrouter.Lookup(r.classifierModel)
	if !modelrouter.HasEnv(spec.Vendor) {
		return heuristicClassify(description), nil
	}

	req := modelrouter.Request{
		Model:       r.classifierModel,
		System:      classifierSystemPrompt,
		CacheSystem: true, // 1h cache slot
		Messages: []modelrouter.Message{
			{Role: modelrouter.RoleUser, Content: "Task description: " + description},
		},
		MaxOutput: 300,
		JSONMode:  true,
	}
	resp, err := r.mr.Call(ctx, req)
	if err != nil {
		// On API errors, fall back to heuristic so the control plane still works.
		return heuristicClassify(description), nil
	}

	cls, err := parseClassifierResponse(resp.Content)
	if err != nil {
		return heuristicClassify(description), nil
	}
	return cls, nil
}

// Route picks executor + verifier given a Classification and returns a Routing.
func (r *Router) Route(cls Classification) (*cruciblev1.Routing, error) {
	tier := tierForComplexity(cls.Complexity)
	if cls.CriticalScore >= 80 && tier < modelrouter.Tier2 {
		tier = modelrouter.Tier2
	}
	executor, err := modelrouter.PrimaryForTier(tier)
	if err != nil {
		return nil, err
	}
	verifier, err := modelrouter.CrossFamilyVerifier(executor)
	if err != nil {
		return nil, err
	}
	return &cruciblev1.Routing{
		ExecutorModel:  executor.ID,
		ExecutorVendor: string(executor.Vendor),
		ExecutorTier:   cruciblev1.ModelTier(int(executor.Tier)),
		VerifierModel:  verifier.ID,
		VerifierVendor: string(verifier.Vendor),
		VerifierTier:   cruciblev1.ModelTier(int(verifier.Tier)),
		CriticalScore:  float64(cls.CriticalScore),
		IsCritical:     cls.CriticalScore >= 80,
		DecidedAt:      time.Now().UTC(),
	}, nil
}

func tierForComplexity(c string) modelrouter.ModelTier {
	switch strings.ToLower(c) {
	case "trivial":
		return modelrouter.Tier0
	case "standard":
		return modelrouter.Tier1
	case "complex", "critical", "modernization":
		return modelrouter.Tier2
	default:
		return modelrouter.Tier1
	}
}

// parseClassifierResponse strips ```json fences if the model added them, then
// json.Unmarshal into Classification.
func parseClassifierResponse(s string) (Classification, error) {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		// Strip the fence; the model occasionally adds it despite the
		// "no markdown" instruction.
		s = strings.TrimPrefix(s, "```json")
		s = strings.TrimPrefix(s, "```")
		s = strings.TrimSuffix(s, "```")
		s = strings.TrimSpace(s)
	}
	var c Classification
	if err := json.Unmarshal([]byte(s), &c); err != nil {
		return Classification{}, fmt.Errorf("taskrouter: parse classifier response: %w", err)
	}
	if c.Complexity == "" {
		return Classification{}, fmt.Errorf("taskrouter: classifier returned empty complexity")
	}
	return c, nil
}

// heuristicClassify is the fallback when no LLM is reachable. Keyword-based,
// deliberately conservative: when in doubt, escalate.
func heuristicClassify(description string) Classification {
	d := strings.ToLower(description)
	c := Classification{
		Complexity:    "standard",
		CriticalScore: 10,
		Rationale:     "heuristic fallback (no Anthropic API key set)",
	}
	switch {
	case anyOf(d, "typo", "rename", "format ", "fix typo"):
		c.Complexity = "trivial"
		c.CriticalScore = 0
	case anyOf(d, "migration", "schema change", "drop ", "alter table"):
		c.Complexity = "critical"
		c.CriticalScore = 85
	case anyOf(d, "auth", "billing", "refund", "payment", "kms", "secret", "crypto", "consensus"):
		c.Complexity = "critical"
		c.CriticalScore = 80
	case anyOf(d, "refactor", "extract", "modulariz", "decompose"):
		c.Complexity = "complex"
		c.CriticalScore = 30
	case anyOf(d, "modernize", "upgrade ", "rewrite"):
		c.Complexity = "modernization"
		c.CriticalScore = 40
	}
	return c
}

func anyOf(s string, needles ...string) bool {
	for _, n := range needles {
		if strings.Contains(s, n) {
			return true
		}
	}
	return false
}
