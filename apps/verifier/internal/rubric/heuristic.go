package rubric

import (
	"context"
	"encoding/json"
	"time"

	"github.com/crucible/verifier/internal/verification"
	"github.com/crucible/verifier/pkg/testreport"
)

// HeuristicClient is a deterministic LLMClient that computes a rubric
// score from the per-tier reports + trust signals WITHOUT calling a
// model. Used in CI when no API key is set, AND as a sanity-check
// fallback the daemon can pivot to when the cross-family verifier API
// is unreachable.
//
// The heuristic identity is "crucible-heuristic"; the cross-family
// guard treats it as a separate vendor lineage so it can pair with any
// executor without violating ADR-002. (This is correct: a deterministic
// rule engine has no model lineage at all.)
type HeuristicClient struct {
	VendorName string
	ModelName  string
}

// NewHeuristicClient returns a deterministic non-LLM judge identity.
func NewHeuristicClient() *HeuristicClient {
	return &HeuristicClient{
		VendorName: "crucible-heuristic",
		ModelName:  "rubric-heuristic-v1",
	}
}

func (h *HeuristicClient) Vendor() string { return h.VendorName }
func (h *HeuristicClient) Model() string  { return h.ModelName }

// Call ignores System/User and instead parses out the embedded tier-summary
// section, scoring each tier independently. We pull the heuristic
// straight from the prompt body so the same code-path is exercised by
// CI and the daemon. (The audit guards in RenderPrompt have already
// guaranteed no executor-reasoning leak in the body we're inspecting.)
func (h *HeuristicClient) Call(_ context.Context, req CallRequest) (CallResponse, error) {
	score := HeuristicScore(req.User)
	resp := Score{
		Score:      score,
		Threshold:  DefaultThreshold,
		Passed:     score >= DefaultThreshold,
		Confidence: 0.6, // heuristic — calibrated lower than an LLM
		Subscores: map[string]float64{
			"diff_correctness":       score,
			"test_adequacy":          score,
			"spec_consistency":       score,
			"robustness":             score,
			"security_posture":       score,
			"trust_signal_alignment": score,
		},
	}
	body, _ := json.Marshal(resp)
	return CallResponse{
		Body:         body,
		ModelLatency: 0,
		CostUSD:      0,
	}, nil
}

// HeuristicScore inspects the user-prompt body and assigns a score.
// Implementation: every "passed=true" in a tier block contributes
// +0.18; "passed=false" contributes -0.5 (caps at 0).
func HeuristicScore(userBody string) float64 {
	score := 0.1 // small baseline so a no-tier run is bound at 0.1
	for _, line := range splitLines(userBody) {
		switch {
		case containsSubstring(line, "passed=true"):
			score += 0.18
		case containsSubstring(line, "passed=false"):
			score -= 0.5
		case containsSubstring(line, "bit_identical=true"):
			score += 0.05
		case containsSubstring(line, "bit_identical=false"):
			score -= 0.3
		}
	}
	if score < 0 {
		return 0
	}
	if score > 1 {
		return 1
	}
	return score
}

// MakeHeuristicJudge wires a HeuristicClient into a Judge with the
// usual cross-family Blocklist — caller can extend the blocklist by
// passing the executor vendor.
func MakeHeuristicJudge(executorVendor string) *Judge {
	return &Judge{
		Client:    NewHeuristicClient(),
		Criteria:  DefaultCriteria,
		Threshold: DefaultThreshold,
		// VendorBlocklist intentionally empty — heuristic is vendor-neutral.
	}
}

// Convenience for tests that want a Score directly without going through
// LLM marshalling.
func HeuristicScoreFor(req *verification.VerificationRequest, reports []*testreport.TestReport) Score {
	body, err := RenderPrompt(PromptInput{Request: req, TierReports: reports, Criteria: DefaultCriteria})
	if err != nil {
		return Score{Score: 0, Passed: false, Confidence: 0, RejectionReasons: []RejectionReason{
			{Category: "render_failed", Severity: "error", Detail: err.Error()},
		}}
	}
	s := HeuristicScore(body.User)
	score := Score{
		Score:      s,
		Threshold:  DefaultThreshold,
		Passed:     s >= DefaultThreshold,
		Confidence: 0.6,
		Subscores: map[string]float64{
			"diff_correctness":       s,
			"test_adequacy":          s,
			"spec_consistency":       s,
			"robustness":             s,
			"security_posture":       s,
			"trust_signal_alignment": s,
		},
		PromptHash:  body.Hash,
		JudgeVendor: "crucible-heuristic",
		JudgeModel:  "rubric-heuristic-v1",
	}
	score.RejectionReasons = append(score.RejectionReasons, trustSignalWarnings(req)...)
	score.RejectionReasons = append(score.RejectionReasons, hardRejections(req, reports)...)
	if countErrors(score.RejectionReasons) > 0 {
		score.Passed = false
	}
	_ = time.Now()
	return score
}

func splitLines(s string) []string {
	out := make([]string, 0, 32)
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		out = append(out, s[start:])
	}
	return out
}

func containsSubstring(s, sub string) bool {
	if len(sub) == 0 {
		return true
	}
	if len(sub) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
