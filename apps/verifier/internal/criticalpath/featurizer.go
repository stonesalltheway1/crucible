package criticalpath

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"math"
	"regexp"
	"sort"
	"strings"
)

// FileFeatures is the raw [0..1]-bounded signal vector for one file. Each
// field maps directly to a coefficient in the weighted-sum score equation.
type FileFeatures struct {
	Path             string
	PathPatternAxes  map[Axis]bool // which axis regexes matched
	PathPatternScore float64        // 0..1, max(axis_match)

	LLMCategoryScore  float64 // 0..1, weighted by LLM confidence
	LLMCategory       string  // "security" | "money" | "data-integrity" | "safety" | "performance" | "infrastructure" | "ui" | "plumbing" | "test" | "dead"

	FanInCentrality  float64 // log-normalized
	IncidentMention  float64 // 0..1, postmortem hits decayed
	SLOBacking       float64 // 1.0 if backs ≥99.9 SLO else 0
	ReviewIntensity  float64 // 0..1, normalized over reviewer count + comments/PR
	CVEHistory       float64 // 1.0 if CVE-touched in 24mo else 0
	TestCovGradient  float64 // z-score within repo, [0..1]
	CommentMarker    float64 // density score
	CodeOwnersScore  float64 // 1.0 if owned by sec/payments/sre teams
	UIOrTestPenalty  float64 // 0..1 — penalty applied with negative weight
}

// Featurizer assembles features for one file. Production featurizers
// call OSV-DB, post-mortem stores, Datadog catalogs; tests inject
// deterministic implementations.
type Featurizer interface {
	Featurize(ctx context.Context, path string, content []byte) (FileFeatures, error)
}

// PathPatternFeaturizer is the offline-only featurizer: only the regex set,
// comment markers, and UI/test penalty. Production layers add the
// other signals via FeaturizerStack.
type PathPatternFeaturizer struct {
	Patterns           []AxisRegex
	MarkerPatterns     []*regexp.Regexp // optional override
	UIOrTestPatterns   []*regexp.Regexp
	SimulatorPatterns  []*regexp.Regexp
}

// NewPathPatternFeaturizer returns the default canonical featurizer.
func NewPathPatternFeaturizer() *PathPatternFeaturizer {
	return &PathPatternFeaturizer{
		Patterns:          DefaultAxisPatterns,
		MarkerPatterns:    CommentMarkers,
		UIOrTestPatterns:  UIOrTestPathPatterns,
		SimulatorPatterns: SimulatorOrSandboxPatterns,
	}
}

// Featurize computes path-pattern, comment-marker, and ui-or-test signals.
// LLM, centrality, incident, SLO, CVE, review-intensity, codeowners are
// returned zero and meant to be merged in by a FeaturizerStack.
func (p *PathPatternFeaturizer) Featurize(_ context.Context, path string, content []byte) (FileFeatures, error) {
	f := FileFeatures{
		Path:            path,
		PathPatternAxes: map[Axis]bool{},
	}

	// 1. Path-pattern axes — match against path, not content. (Content
	// scans land in the comment-marker score, not the axis score.)
	maxAxisHit := 0.0
	for _, ar := range p.Patterns {
		if ar.Pattern.MatchString(path) {
			f.PathPatternAxes[ar.Axis] = true
			// every axis match contributes 1.0; we take max for the
			// per-file score.
			maxAxisHit = 1.0
		}
	}
	f.PathPatternScore = maxAxisHit

	// 2. Comment-marker score — density per kloc.
	if len(content) > 0 {
		hits := 0
		for _, m := range p.MarkerPatterns {
			hits += len(m.FindAllIndex(content, -1))
		}
		kloc := math.Max(1.0, float64(len(content))/1024.0)
		dens := float64(hits) / kloc
		// Saturate density at 5 hits / kloc → 1.0.
		f.CommentMarker = math.Min(1.0, dens/5.0)
	}

	// 3. UI-or-test penalty
	for _, m := range p.UIOrTestPatterns {
		if m.MatchString(path) {
			f.UIOrTestPenalty = 1.0
			break
		}
	}
	// 3b. Simulator/fixture/mock files — additional UI-or-test-like
	// down-weight even when path doesn't say "test" (catches
	// `tools/payment_simulator_for_demos.py`).
	if f.UIOrTestPenalty == 0 {
		for _, m := range p.SimulatorPatterns {
			if m.MatchString(path) {
				f.UIOrTestPenalty = 0.6
				break
			}
		}
	}

	return f, nil
}

// FeaturizerStack composes multiple Featurizers; later layers' non-zero
// fields override earlier layers'. This is how production layers in the
// LLM-judge / OSV-DB / Datadog signals on top of the offline baseline.
type FeaturizerStack []Featurizer

// Featurize folds each layer's output by overriding non-zero fields.
func (s FeaturizerStack) Featurize(ctx context.Context, path string, content []byte) (FileFeatures, error) {
	var cur FileFeatures
	for i, layer := range s {
		next, err := layer.Featurize(ctx, path, content)
		if err != nil {
			return cur, err
		}
		if i == 0 {
			cur = next
			continue
		}
		cur = mergeFeatures(cur, next)
	}
	return cur, nil
}

func mergeFeatures(a, b FileFeatures) FileFeatures {
	out := a
	out.Path = a.Path
	if b.Path != "" {
		out.Path = b.Path
	}
	// PathPatternAxes union; PathPatternScore = max
	if out.PathPatternAxes == nil {
		out.PathPatternAxes = map[Axis]bool{}
	}
	for k, v := range b.PathPatternAxes {
		out.PathPatternAxes[k] = out.PathPatternAxes[k] || v
	}
	out.PathPatternScore = math.Max(a.PathPatternScore, b.PathPatternScore)
	out.LLMCategoryScore = pickNonzero(a.LLMCategoryScore, b.LLMCategoryScore)
	if b.LLMCategory != "" {
		out.LLMCategory = b.LLMCategory
	}
	out.FanInCentrality = pickNonzero(a.FanInCentrality, b.FanInCentrality)
	out.IncidentMention = pickNonzero(a.IncidentMention, b.IncidentMention)
	out.SLOBacking = pickNonzero(a.SLOBacking, b.SLOBacking)
	out.ReviewIntensity = pickNonzero(a.ReviewIntensity, b.ReviewIntensity)
	out.CVEHistory = pickNonzero(a.CVEHistory, b.CVEHistory)
	out.TestCovGradient = pickNonzero(a.TestCovGradient, b.TestCovGradient)
	out.CommentMarker = math.Max(a.CommentMarker, b.CommentMarker)
	out.CodeOwnersScore = pickNonzero(a.CodeOwnersScore, b.CodeOwnersScore)
	out.UIOrTestPenalty = math.Max(a.UIOrTestPenalty, b.UIOrTestPenalty)
	return out
}

func pickNonzero(a, b float64) float64 {
	if b > 0 {
		return b
	}
	return a
}

// LLMJudgeFeaturizer is the Featurizer that calls Haiku-4.5 (or the
// configured small model) to categorise the file.
type LLMJudgeFeaturizer struct {
	Judge LLMJudge
	// Cache holds content-hash → CategoryAndConfidence. The Phase-4
	// implementation uses an in-memory LRU; production uses Redis +
	// per-tenant scope.
	Cache LLMJudgeCache
}

// LLMJudge issues one categorisation call.
type LLMJudge interface {
	Categorise(ctx context.Context, path string, content []byte) (CategoryAndConfidence, error)
}

// CategoryAndConfidence is the structured return from the LLM judge.
type CategoryAndConfidence struct {
	Category   string  `json:"category"`
	Confidence float64 `json:"confidence"` // 0..1
	Reasoning  string  `json:"reasoning"`
}

// LLMJudgeCache is a content-addressed cache keyed by sha256(path+content).
type LLMJudgeCache interface {
	Get(key string) (CategoryAndConfidence, bool)
	Put(key string, v CategoryAndConfidence)
}

// Featurize calls the LLM judge (or returns cached) and maps the
// category to a LLMCategoryScore in [0..1]. Categories below "plumbing"
// in importance return 0; "security"/"money"/"data-integrity"/"safety"
// return 1.0 scaled by confidence.
func (l *LLMJudgeFeaturizer) Featurize(ctx context.Context, path string, content []byte) (FileFeatures, error) {
	if l.Judge == nil {
		return FileFeatures{Path: path}, nil
	}
	key := contentKey(path, content)
	var cat CategoryAndConfidence
	hit := false
	if l.Cache != nil {
		cat, hit = l.Cache.Get(key)
	}
	if !hit {
		c, err := l.Judge.Categorise(ctx, path, content)
		if err != nil {
			// Soft failure — feature is "unknown".
			return FileFeatures{Path: path}, nil
		}
		cat = c
		if l.Cache != nil {
			l.Cache.Put(key, cat)
		}
	}
	score := categoryToScore(cat.Category) * cat.Confidence
	out := FileFeatures{
		Path:             path,
		LLMCategoryScore: score,
		LLMCategory:      cat.Category,
	}
	// "ui" / "test" / "dead" categorisation contributes to the UI/test
	// penalty even if the path didn't trigger it.
	if cat.Category == "ui" || cat.Category == "test" || cat.Category == "dead" {
		out.UIOrTestPenalty = math.Max(0.8*cat.Confidence, 0.0)
	}
	return out, nil
}

func categoryToScore(c string) float64 {
	switch strings.ToLower(c) {
	case "security", "money", "data-integrity", "safety":
		return 1.0
	case "performance":
		return 0.8
	case "infrastructure":
		return 0.5
	case "plumbing":
		return 0.25
	case "ui", "test", "dead":
		return 0.0
	default:
		return 0.3
	}
}

// FanInCentralityFeaturizer reads a precomputed import-graph centrality
// map (path → log-normalised PageRank/fan-in). Production wires this
// from the tree-sitter + per-language symbol-resolver job that runs
// periodically.
type FanInCentralityFeaturizer struct {
	Centrality map[string]float64 // path → 0..1
	Top5pcCut  float64            // files at/above this are auto-critical
}

func (f *FanInCentralityFeaturizer) Featurize(_ context.Context, path string, _ []byte) (FileFeatures, error) {
	score := f.Centrality[path]
	out := FileFeatures{Path: path, FanInCentrality: score}
	if f.Top5pcCut > 0 && score >= f.Top5pcCut {
		// Override LLMCategoryScore upward when fan-in is extreme. This
		// is the load-bearing utils/retry.ts case in the doc.
		out.LLMCategoryScore = math.Max(out.LLMCategoryScore, 0.8)
	}
	return out, nil
}

// CodeOwnersFeaturizer reads CODEOWNERS and sets a 1.0 signal when the
// file's owner team matches sec/payments/sre/oncall lists.
type CodeOwnersFeaturizer struct {
	Owners map[string][]string // path glob → owner teams
}

var criticalTeamSuffix = []string{
	"-sec", "-security", "-payments", "-billing",
	"-sre", "-oncall", "-platform-security",
	"-finance", "-fraud", "-compliance",
}

func (c *CodeOwnersFeaturizer) Featurize(_ context.Context, path string, _ []byte) (FileFeatures, error) {
	teams := c.lookup(path)
	if len(teams) == 0 {
		return FileFeatures{Path: path}, nil
	}
	for _, t := range teams {
		lt := strings.ToLower(t)
		for _, suf := range criticalTeamSuffix {
			if strings.HasSuffix(lt, suf) || strings.Contains(lt, suf+"-") {
				return FileFeatures{Path: path, CodeOwnersScore: 1.0}, nil
			}
		}
	}
	return FileFeatures{Path: path, CodeOwnersScore: 0.3}, nil
}

func (c *CodeOwnersFeaturizer) lookup(path string) []string {
	if c.Owners == nil {
		return nil
	}
	// Phase 4 takes the longest-matching glob (CODEOWNERS semantics).
	bestLen := -1
	var best []string
	for glob, teams := range c.Owners {
		if matchGlob(glob, path) && len(glob) > bestLen {
			bestLen = len(glob)
			best = teams
		}
	}
	return best
}

// CVEFeaturizer reads a precomputed "this path was touched by a CVE in
// the last 24mo" set, populated from `git log --grep='CVE-'` + OSV-DB.
type CVEFeaturizer struct {
	Touched map[string]bool
}

func (c *CVEFeaturizer) Featurize(_ context.Context, path string, _ []byte) (FileFeatures, error) {
	if c.Touched[path] {
		return FileFeatures{Path: path, CVEHistory: 1.0}, nil
	}
	return FileFeatures{Path: path}, nil
}

// ProductionSignalsFeaturizer takes precomputed (postmortem-mention,
// SLO-backing, review-intensity) maps. These come from Rootly /
// FireHydrant / Datadog / GitHub adapters; Phase 4 keeps the adapter
// integration stubbed but the data shape is stable.
type ProductionSignalsFeaturizer struct {
	IncidentMention  map[string]float64
	SLOBacking       map[string]float64
	ReviewIntensity  map[string]float64
	TestCovGradient  map[string]float64
}

func (p *ProductionSignalsFeaturizer) Featurize(_ context.Context, path string, _ []byte) (FileFeatures, error) {
	return FileFeatures{
		Path:            path,
		IncidentMention: p.IncidentMention[path],
		SLOBacking:      p.SLOBacking[path],
		ReviewIntensity: p.ReviewIntensity[path],
		TestCovGradient: p.TestCovGradient[path],
	}, nil
}

// Internal helpers — small and dependency-free so the
// criticalpath package is hermetic.

// matchGlob is a small subset of fnmatch: supports `*` and `**`. CODEOWNERS
// uses gitignore-style globs; for Phase 4 we accept the common subset.
func matchGlob(glob, path string) bool {
	if glob == "*" || glob == "**" {
		return true
	}
	if !strings.ContainsAny(glob, "*?") {
		return glob == path || strings.HasSuffix(path, glob)
	}
	// Translate `**` → `.*`, `*` → `[^/]*` (no slash crossing), `?` → `.`.
	var b strings.Builder
	b.WriteString(`^`)
	for i := 0; i < len(glob); i++ {
		c := glob[i]
		switch c {
		case '*':
			if i+1 < len(glob) && glob[i+1] == '*' {
				b.WriteString(`.*`)
				i++
			} else {
				b.WriteString(`[^/]*`)
			}
		case '?':
			b.WriteString(`.`)
		case '.', '+', '(', ')', '|', '^', '$', '{', '}', '[', ']', '\\':
			b.WriteByte('\\')
			b.WriteByte(c)
		default:
			b.WriteByte(c)
		}
	}
	b.WriteString(`$`)
	re, err := regexp.Compile(b.String())
	if err != nil {
		return false
	}
	return re.MatchString(path)
}

// contentKey is the cache key sha256(path||content) — hex-encoded.
func contentKey(path string, content []byte) string {
	h := sha256.New()
	_, _ = h.Write([]byte(path))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write(content)
	return hex.EncodeToString(h.Sum(nil))
}

// SortFeaturesByPath returns a stable copy of features sorted by Path.
// Used by callers that need deterministic enumeration.
func SortFeaturesByPath(in []FileFeatures) []FileFeatures {
	out := make([]FileFeatures, len(in))
	copy(out, in)
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}
