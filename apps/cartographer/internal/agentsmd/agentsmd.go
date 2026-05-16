// Package agentsmd reads the customer-facing convention statements:
// AGENTS.md / CLAUDE.md / .cursorrules / CONTRIBUTING.md / docs/adr/*.
//
// These are the highest-confidence sources because they are explicit
// statements of intent. The Cartographer treats them as authoritative
// overrides over OSS defaults; the inferred-AGENTS.md generator
// (internal/inferred) only fires when none of these exist.
package agentsmd

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/crucible/apps/cartographer/internal/types"
)

// CustomerOverrideFiles is the set of files we treat as authoritative
// agent-instruction files at the repo root.
var CustomerOverrideFiles = []string{"AGENTS.md", "CLAUDE.md", ".cursorrules", ".cursor/rules.md"}

// ContributingFiles match the common variants.
var ContributingFiles = []string{"CONTRIBUTING.md", "CONTRIBUTING", "docs/CONTRIBUTING.md"}

// ADRDirs is the set of conventional ADR directories we scan.
var ADRDirs = []string{"docs/adr", "docs/adrs", "docs/architecture", "adr", "adrs", "architecture-decisions"}

// FoundOverride is the result of FindCustomerOverride.
type FoundOverride struct {
	Path string
	Body []byte
}

// FindCustomerOverride scans the repo root for AGENTS.md / CLAUDE.md /
// .cursorrules. Returns the first match (in priority order). Empty
// path means "none found" — the inferred generator should fire.
func FindCustomerOverride(repoRoot string) (FoundOverride, bool) {
	for _, name := range CustomerOverrideFiles {
		full := filepath.Join(repoRoot, name)
		if body, err := os.ReadFile(full); err == nil {
			return FoundOverride{Path: name, Body: body}, true
		}
	}
	return FoundOverride{}, false
}

// ExtractFromAgentsMD parses an AGENTS.md-style file into typed
// ConventionCandidates. The format we recognise is loose because the
// 60K-AGENTS.md ecosystem doesn't agree on a strict schema; we look
// for headings + bulleted items and treat each bullet as a
// candidate.
func ExtractFromAgentsMD(repo, tenantID, relPath string, body []byte) []types.ConventionCandidate {
	var out []types.ConventionCandidate
	currentCategory := "Naming"
	br := bufio.NewScanner(strings.NewReader(string(body)))
	br.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for br.Scan() {
		line := br.Text()
		stripped := strings.TrimSpace(line)
		if stripped == "" {
			continue
		}

		if strings.HasPrefix(stripped, "#") {
			currentCategory = headingToCategory(strings.Trim(stripped, "# "))
			continue
		}

		// Bulleted lines are candidates.
		if strings.HasPrefix(stripped, "-") || strings.HasPrefix(stripped, "*") {
			rule := strings.TrimSpace(strings.TrimLeft(stripped, "-* "))
			if rule == "" {
				continue
			}
			out = append(out, types.ConventionCandidate{
				ID:            "c_agents_" + simpleHash(repo+rule+relPath),
				Category:      currentCategory,
				RuleNL:        rule,
				FileGlob:      "**/*",
				Rationale:     "Stated in " + relPath + " (customer override).",
				EvidenceQuote: trim(stripped, 240),
				SourceChannel: "adr_file",
				SourcePath:    relPath,
				Confidence:    0.9, // customer-stated rules are high-conf.
				Status:        "active",
				FirstSeen:     time.Now().UTC(),
			})
		}
	}
	return out
}

// ExtractFromContributing parses CONTRIBUTING.md into candidates.
// We use the same bulleted-item heuristic but a lower base confidence
// (CONTRIBUTING.md targets contributors-at-large; team-internal
// conventions live in AGENTS.md).
func ExtractFromContributing(repo, tenantID, relPath string, body []byte) []types.ConventionCandidate {
	cands := ExtractFromAgentsMD(repo, tenantID, relPath, body)
	for i := range cands {
		cands[i].Confidence = 0.75
		cands[i].SourceChannel = "contributing_md"
	}
	return cands
}

// ExtractFromADRs walks one ADR directory and emits one candidate per
// ADR file (the ADR title is treated as the rule statement;
// rationale is the file body).
func ExtractFromADRs(repo, tenantID, adrDirAbs string) []types.ConventionCandidate {
	var out []types.ConventionCandidate
	_ = filepath.Walk(adrDirAbs, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".md") {
			return nil
		}
		body, rerr := os.ReadFile(path)
		if rerr != nil {
			return nil
		}
		title := firstHeading(body)
		if title == "" {
			title = filepath.Base(path)
		}
		base, _ := filepath.Rel(filepath.Dir(adrDirAbs), path)
		out = append(out, types.ConventionCandidate{
			ID:            "c_adr_" + simpleHash(repo+base),
			Category:      headingToCategory(title),
			RuleNL:        "Apply the decision recorded in " + base + ": " + title + ".",
			FileGlob:      "**/*",
			Rationale:     "ADR captured at " + base + ".",
			EvidenceQuote: trim(string(body), 240),
			SourceChannel: "adr_file",
			SourcePath:    base,
			Confidence:    0.85,
			Status:        "active",
			FirstSeen:     time.Now().UTC(),
		})
		return nil
	})
	return out
}

// ScanADRDirs runs ExtractFromADRs over the conventional ADR
// directories and returns the merged result.
func ScanADRDirs(repo, tenantID, repoRoot string) []types.ConventionCandidate {
	var out []types.ConventionCandidate
	for _, d := range ADRDirs {
		full := filepath.Join(repoRoot, d)
		if info, err := os.Stat(full); err == nil && info.IsDir() {
			out = append(out, ExtractFromADRs(repo, tenantID, full)...)
		}
	}
	return out
}

// Helpers.

func headingToCategory(h string) string {
	low := strings.ToLower(h)
	switch {
	case strings.Contains(low, "naming"):
		return "Naming"
	case strings.Contains(low, "test"):
		return "Test patterns"
	case strings.Contains(low, "log"):
		return "Logging"
	case strings.Contains(low, "error"):
		return "Error handling"
	case strings.Contains(low, "secur"):
		return "Security defaults"
	case strings.Contains(low, "migrat"):
		return "Migration patterns"
	case strings.Contains(low, "library"), strings.Contains(low, "depend"):
		return "Library preferences"
	case strings.Contains(low, "layer"), strings.Contains(low, "module"):
		return "Layering"
	case strings.Contains(low, "perform"):
		return "Performance defaults"
	case strings.Contains(low, "concurr"):
		return "Concurrency"
	case strings.Contains(low, "api"):
		return "API shape"
	case strings.Contains(low, "commit"), strings.Contains(low, "pr "):
		return "PR/commit hygiene"
	}
	return "Naming"
}

func firstHeading(body []byte) string {
	br := bufio.NewScanner(strings.NewReader(string(body)))
	for br.Scan() {
		line := strings.TrimSpace(br.Text())
		if strings.HasPrefix(line, "#") {
			return strings.Trim(line, "# ")
		}
	}
	return ""
}

func trim(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
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

// Errors.
var (
	ErrNoOverride = errors.New("agentsmd: no customer override found")
)
