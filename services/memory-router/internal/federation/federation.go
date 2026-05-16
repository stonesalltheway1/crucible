// Package federation implements the cross-tenant graduation candidate
// detector. Phase 5 wires the data model only — actual graduations fire
// in v2 Phase 10.
//
// Eligibility:
//   1. Rule appears in ≥ MinTenantsForGraduation distinct tenant graphs
//   2. Anonymized form (category + canonical rule_nl) is the union key
//   3. Every contributing tenant is opted-in (federation_optout = false)
//
// Anonymization is one-way: the canonical form preserves category +
// rule shape but strips tenant-specific identifiers (e.g. service
// names, library names that aren't in a public allowlist). The Phase 5
// helper does a conservative pass; the v2 engine adds adversarial
// review.
package federation

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"sort"
	"strings"
	"time"

	memoryspec "github.com/crucible/memory-spec/go"
)

// AnonymousForm returns the category-and-shape-only canonical form of a
// rule. Used as the dedup key when collapsing across tenants.
//
// Phase 5 implementation:
//   - Lowercase + collapse whitespace
//   - Strip leading determiners ("our ", "this team's ")
//   - Replace likely service / project names (a heuristic regex) with
//     <SERVICE> tokens
//   - Replace string literals in quotes with <LITERAL>
//   - Replace digits with <N>
//
// The output is a stable hash of the canonical text plus the category,
// so two paraphrases that share the canonical form get the same id.
func AnonymousForm(category memoryspec.ConventionCategory, ruleNl string) (canonical string, hashed string) {
	r := strings.ToLower(strings.TrimSpace(ruleNl))
	r = strings.Join(strings.Fields(r), " ")
	r = stripDeterminers(r)
	r = quotedLiteralPattern.ReplaceAllString(r, "<LITERAL>")
	r = numberPattern.ReplaceAllString(r, "<N>")
	r = identifierPattern.ReplaceAllStringFunc(r, redactNonAllowlisted)

	canonical = string(category) + "::" + r
	h := sha256.Sum256([]byte(canonical))
	hashed = "anon_" + hex.EncodeToString(h[:16])
	return canonical, hashed
}

// Detector iterates tenant graphs and yields graduation candidates.
type Detector struct {
	MinTenants uint32
}

// New returns a Detector with the Phase-5 5-tenant policy threshold.
func New() *Detector {
	return &Detector{MinTenants: uint32(memoryspec.MinTenantsForGraduation)}
}

// Scan groups conventions across tenants by anonymized form and returns
// records that meet the threshold. Tenants opted-out of federation are
// excluded from the contributing list.
func (d *Detector) Scan(byTenant map[string][]memoryspec.Convention, optedOut map[string]bool) []memoryspec.FederationGraduation {
	type bucket struct {
		category memoryspec.ConventionCategory
		canon    string
		tenants  map[string]struct{}
		convs    []string
	}
	buckets := map[string]*bucket{}
	for tenantID, convs := range byTenant {
		if optedOut[tenantID] {
			continue
		}
		for _, c := range convs {
			if c.Status != memoryspec.StatusActive {
				continue
			}
			canon, key := AnonymousForm(c.Category, c.RuleNl)
			b, ok := buckets[key]
			if !ok {
				b = &bucket{category: c.Category, canon: canon, tenants: map[string]struct{}{}, convs: nil}
				buckets[key] = b
			}
			b.tenants[tenantID] = struct{}{}
			b.convs = append(b.convs, c.ID)
		}
	}

	out := make([]memoryspec.FederationGraduation, 0, len(buckets))
	now := time.Now().UTC()
	for key, b := range buckets {
		if uint32(len(b.tenants)) < d.MinTenants {
			continue
		}
		sortedConvs := append([]string{}, b.convs...)
		sort.Strings(sortedConvs)
		out = append(out, memoryspec.FederationGraduation{
			AnonymizedRuleID:          key,
			Category:                  b.category,
			CanonicalFormNl:           b.canon,
			DistinctTenantCount:       uint32(len(b.tenants)),
			ContributingConventionIDs: sortedConvs,
			EligibleAt:                now,
			Fired:                     false, // Phase 5 records, never fires
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].AnonymizedRuleID < out[j].AnonymizedRuleID })
	return out
}

// ─── Anonymization helpers ─────────────────────────────────────────────────

var (
	quotedLiteralPattern = regexp.MustCompile(`"[^"]*"|'[^']*'|` + "`[^`]*`")
	numberPattern        = regexp.MustCompile(`\b\d+\b`)
	identifierPattern    = regexp.MustCompile(`\b[A-Za-z][A-Za-z0-9_]{2,}\b`)
)

// allowlist contains the lower-cased identifiers we keep verbatim in
// canonical form: common library and tool names that recur across teams
// without identifying a tenant.
var allowlist = map[string]struct{}{
	"slog": {}, "zap": {}, "pino": {}, "winston": {}, "logback": {},
	"date-fns": {}, "dayjs": {}, "moment": {}, "luxon": {},
	"zod": {}, "yup": {}, "joi": {}, "valibot": {},
	"vitest": {}, "jest": {}, "mocha": {}, "pytest": {}, "unittest": {},
	"context": {}, "promise": {}, "async": {}, "await": {},
	"react": {}, "vue": {}, "angular": {}, "svelte": {},
	"django": {}, "fastapi": {}, "flask": {}, "rails": {}, "spring": {},
	"node": {}, "express": {}, "next": {}, "nuxt": {}, "remix": {},
	"sql": {}, "postgres": {}, "mysql": {}, "mongo": {},
	"http": {}, "https": {}, "tcp": {}, "udp": {},
	"json": {}, "xml": {}, "yaml": {}, "toml": {},
	"the": {}, "and": {}, "for": {}, "with": {}, "use": {}, "do": {}, "not": {},
	"prefer": {}, "should": {}, "must": {}, "always": {}, "never": {},
	"all": {}, "any": {}, "every": {}, "some": {}, "none": {},
}

func redactNonAllowlisted(s string) string {
	low := strings.ToLower(s)
	if _, ok := allowlist[low]; ok {
		return low
	}
	if len(s) <= 3 {
		return low // 3-letter tokens are usually keywords ("get", "set", "let"); keep
	}
	// Replace with <ID> token preserving rough shape.
	return "<ID>"
}

var leadingDeterminers = []string{
	"our team's ", "our teams' ", "our team ", "our ",
	"this team's ", "this team ", "this project's ", "this project ",
	"the team's ", "the team ", "the project's ", "the project ",
}

func stripDeterminers(s string) string {
	for _, d := range leadingDeterminers {
		if strings.HasPrefix(s, d) {
			return s[len(d):]
		}
	}
	return s
}
