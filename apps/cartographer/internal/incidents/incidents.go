// Package incidents collects incident references from PR descriptions
// and AGENTS.md / CONTRIBUTING.md text.
//
// We don't fetch the incident bodies themselves (Linear / Jira /
// Slack #incidents access requires per-customer auth handled by the
// onboarding flow). We surface the references as candidates the
// distillation pass turns into typed conventions when the customer
// later wires the incident-source adapters.
package incidents

import (
	"strings"
	"time"

	"github.com/crucible/apps/cartographer/internal/types"
)

var prefixes = []string{
	"https://linear.app/",
	"https://jira.",
	"https://atlassian.net/",
	"https://app.shortcut.com/",
	"https://app.clickup.com/",
}

// Detect finds incident references in a body and emits typed
// candidates with channel="incident".
func Detect(repo, tenantID, sourcePath, body string) []types.ConventionCandidate {
	var out []types.ConventionCandidate
	for _, p := range prefixes {
		idx := 0
		for {
			i := strings.Index(body[idx:], p)
			if i < 0 {
				break
			}
			start := idx + i
			end := start + len(p)
			for end < len(body) && body[end] != ' ' && body[end] != ')' && body[end] != ']' && body[end] != '\n' {
				end++
			}
			ref := body[start:end]
			out = append(out, types.ConventionCandidate{
				ID:            "c_inc_" + simpleHash(repo+ref),
				Category:      "Security defaults",
				RuleNL:        "Apply learnings from " + ref + " — context required from the linked incident.",
				FileGlob:      "**/*",
				Rationale:     "Incident referenced from " + sourcePath,
				EvidenceQuote: ref,
				SourceChannel: "incident",
				SourcePath:    sourcePath,
				Confidence:    0.4,
				Status:        "candidate",
				FirstSeen:     time.Now().UTC(),
			})
			idx = end
		}
	}
	// Bare INC-#### tokens.
	idx := 0
	for {
		i := strings.Index(body[idx:], "INC-")
		if i < 0 {
			break
		}
		start := idx + i
		end := start + 4
		for end < len(body) && body[end] >= '0' && body[end] <= '9' {
			end++
		}
		if end > start+4 {
			ref := body[start:end]
			out = append(out, types.ConventionCandidate{
				ID:            "c_inc_" + simpleHash(repo+ref),
				Category:      "Security defaults",
				RuleNL:        "Apply learnings from " + ref + ".",
				FileGlob:      "**/*",
				Rationale:     "Incident token referenced from " + sourcePath,
				EvidenceQuote: ref,
				SourceChannel: "incident",
				SourcePath:    sourcePath,
				Confidence:    0.4,
				Status:        "candidate",
				FirstSeen:     time.Now().UTC(),
			})
		}
		idx = end
	}
	return out
}

func simpleHash(s string) string {
	const (
		offset uint32 = 2166136261
		prime  uint32 = 16777619
	)
	h := offset
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= prime
	}
	const hex = "0123456789abcdef"
	var b [8]byte
	for i := 7; i >= 0; i-- {
		b[i] = hex[h&0xf]
		h >>= 4
	}
	return string(b[:])
}
