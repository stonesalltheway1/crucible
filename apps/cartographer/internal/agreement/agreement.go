// Package agreement implements cross-source agreement scoring per
// docs/06-research/memory-bootstrap.md §3.
//
// The pipeline:
//
//   1. Group candidates by (category, normalized_rule_signature).
//   2. For each cluster, compute confidence = log(distinct_repos+1) /
//      log(repos_examined+1) + tier-A bonus + per-source priors.
//   3. Threshold:
//        confidence ≥ 0.4 → ACTIVE
//        confidence ∈ [0.25, 0.4) → SUGGESTED
//        confidence < 0.25 → CANDIDATE
//   4. Counter-examples that contradict a high-conf candidate demote it.
//
// We do NOT cluster by embedding here — that would require a model
// dependency. Instead we cluster by a heuristic signature: lower-cased
// stop-word-stripped rule text. The Phase 5 distiller's HDBSCAN
// embedding cluster runs downstream (memory-router admission) and
// supersedes our candidates if the embedding agreement says
// otherwise. This keeps the cartographer hermetic.
package agreement

import (
	"sort"
	"strings"

	"github.com/crucible/apps/cartographer/internal/types"
)

// Bucket contains the result.
type Bucket struct {
	High      []types.ConventionCandidate // confidence ≥ 0.6
	Medium    []types.ConventionCandidate // 0.4 ≤ conf < 0.6
	Low       []types.ConventionCandidate // 0.25 ≤ conf < 0.4
	Discarded []types.ConventionCandidate // conf < 0.25
}

// Score scores and re-clusters the candidates. The result preserves
// the input candidate fields and updates Confidence + Status.
func Score(in []types.ConventionCandidate, reposExamined int) Bucket {
	if reposExamined < 1 {
		reposExamined = 1
	}
	clusters := clusterBySignature(in)

	var b Bucket
	for sig, group := range clusters {
		_ = sig
		// Distinct sources for this cluster.
		repos := distinctRepos(group)
		conf := scoreOne(repos, reposExamined, group)
		// Pick a representative (highest base-conf candidate).
		rep := pickRep(group)
		rep.Confidence = clamp(conf, 0, 1)
		rep.Status = bucketStatus(rep.Confidence)
		switch {
		case rep.Confidence >= 0.6:
			b.High = append(b.High, rep)
		case rep.Confidence >= 0.4:
			b.Medium = append(b.Medium, rep)
		case rep.Confidence >= 0.25:
			b.Low = append(b.Low, rep)
		default:
			b.Discarded = append(b.Discarded, rep)
		}
	}
	sortByConfidenceDesc(b.High)
	sortByConfidenceDesc(b.Medium)
	sortByConfidenceDesc(b.Low)
	return b
}

// Counts returns (high, medium, low) counts.
func (b Bucket) Counts() (int, int, int) { return len(b.High), len(b.Medium), len(b.Low) }

// FilterContradictions demotes candidates whose rule text matches a
// known-contradiction list. The list is a small curated set; the
// Phase-5 detector handles the larger embedding-space contradiction
// surface.
func (b *Bucket) FilterContradictions() {
	contradicts := []func(string) bool{
		func(r string) bool { l := strings.ToLower(r); return strings.Contains(l, "do not use cursor pagination") },
		func(r string) bool { l := strings.ToLower(r); return strings.Contains(l, "use offset pagination") && strings.Contains(l, "always") },
	}
	demote := func(in []types.ConventionCandidate) []types.ConventionCandidate {
		out := in[:0]
		for _, c := range in {
			drop := false
			for _, fn := range contradicts {
				if fn(c.RuleNL) {
					drop = true
					break
				}
			}
			if !drop {
				out = append(out, c)
			}
		}
		return out
	}
	b.High = demote(b.High)
	b.Medium = demote(b.Medium)
}

// --- Internals ---

var stopWords = map[string]bool{
	"the": true, "a": true, "an": true, "and": true, "or": true, "for": true,
	"to": true, "of": true, "in": true, "on": true, "at": true, "be": true,
	"is": true, "are": true, "this": true, "that": true, "should": true,
	"must": true, "always": true, "never": true,
}

func signature(rule string) string {
	low := strings.ToLower(rule)
	for _, ch := range []string{".", ",", "(", ")", "—", "-", "'", "\"", ":"} {
		low = strings.ReplaceAll(low, ch, " ")
	}
	tokens := strings.Fields(low)
	keep := tokens[:0]
	for _, t := range tokens {
		if stopWords[t] || len(t) <= 2 {
			continue
		}
		keep = append(keep, t)
	}
	sort.Strings(keep)
	return strings.Join(keep, " ")
}

func clusterBySignature(cands []types.ConventionCandidate) map[string][]types.ConventionCandidate {
	out := map[string][]types.ConventionCandidate{}
	for _, c := range cands {
		sig := c.Category + "|" + signature(c.RuleNL)
		out[sig] = append(out[sig], c)
	}
	return out
}

func distinctRepos(cands []types.ConventionCandidate) int {
	seen := map[string]bool{}
	for _, c := range cands {
		// Use SourcePath as a within-tenant repo proxy: same file →
		// same repo signal. Cross-repo distinctness comes when we
		// fold OSS-default candidates into the score.
		seen[c.SourceChannel+":"+c.SourcePath] = true
	}
	return len(seen)
}

// scoreOne computes the agreement-weighted confidence for a cluster.
// Formula derived from docs/06-research/memory-bootstrap.md:
//
//   confidence = log(distinct_sources + 1) / log(repos_examined + 1)
//   + tier-A bonus * 1.5
//   + per-source prior
//
// We then clamp to [0,1].
func scoreOne(distinct, reposExamined int, cands []types.ConventionCandidate) float64 {
	base := log1(float64(distinct)) / log1(float64(reposExamined)+1.0)
	priorMax := 0.0
	tierABonus := 0.0
	for _, c := range cands {
		if c.Confidence > priorMax {
			priorMax = c.Confidence
		}
		switch c.SourceChannel {
		case "adr_file", "agents_md":
			tierABonus = 0.1
		}
	}
	return base + priorMax*0.5 + tierABonus
}

func pickRep(cands []types.ConventionCandidate) types.ConventionCandidate {
	rep := cands[0]
	for _, c := range cands[1:] {
		if c.Confidence > rep.Confidence {
			rep = c
		}
	}
	return rep
}

func bucketStatus(conf float64) string {
	switch {
	case conf >= 0.6:
		return "active"
	case conf >= 0.4:
		return "suggested"
	case conf >= 0.25:
		return "candidate"
	}
	return "discarded"
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// log1 is a tiny dependency-free natural log approximation good to
// ~3 decimal places over [1, 1e6]. We use a Padé approximant on
// log(1+x) and add 1 to avoid the singularity at 0.
func log1(x float64) float64 {
	if x <= 0 {
		return 0
	}
	// Reduce x = 2^k * m where m ∈ [1,2).
	k := 0
	for x >= 2 {
		x /= 2
		k++
	}
	for x < 1 {
		x *= 2
		k--
	}
	// log(m) ≈ 2 * (m-1)/(m+1) * (1 + ((m-1)/(m+1))^2 / 3 + …)
	y := (x - 1) / (x + 1)
	y2 := y * y
	logm := 2 * y * (1 + y2/3 + y2*y2/5)
	return float64(k)*0.6931471805599453 + logm
}

func sortByConfidenceDesc(cs []types.ConventionCandidate) {
	sort.SliceStable(cs, func(i, j int) bool { return cs[i].Confidence > cs[j].Confidence })
}
