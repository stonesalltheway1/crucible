// Package coverage tracks per-endpoint last-recorded timestamps and
// per-host hit counts for the tape-population dashboard.
package coverage

import (
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/crucible/services/shadow-recorder/internal/types"
)

// Tracker tracks coverage stats.
type Tracker struct {
	mu     sync.Mutex
	byKey  map[string]*types.EndpointStat
}

// New returns a fresh Tracker.
func New() *Tracker {
	return &Tracker{byKey: map[string]*types.EndpointStat{}}
}

// Record updates the stat for the (tenant, host, method, path).
func (t *Tracker) Record(tenantID, host, method, path string, when time.Time, rerecordEvery time.Duration) {
	if when.IsZero() {
		when = time.Now().UTC()
	}
	pt := pathTemplate(path)
	key := tenantID + "|" + host + "|" + method + "|" + pt
	t.mu.Lock()
	defer t.mu.Unlock()
	st, ok := t.byKey[key]
	if !ok {
		st = &types.EndpointStat{
			TenantID: tenantID, Host: host, Method: method, PathTemplate: pt,
		}
		t.byKey[key] = st
	}
	st.HitCount++
	st.LastRecordedAt = when
	st.NextRecordDue = when.Add(rerecordEvery)
}

// HostCoverage returns the rollup for one host.
func (t *Tracker) HostCoverage(tenantID, host string) types.HostCoverage {
	t.mu.Lock()
	defer t.mu.Unlock()
	cov := types.HostCoverage{Host: host}
	for _, st := range t.byKey {
		if st.TenantID != tenantID || st.Host != host {
			continue
		}
		cov.Endpoints++
		cov.TotalHits += st.HitCount
		if cov.OldestRecord.IsZero() || st.LastRecordedAt.Before(cov.OldestRecord) {
			cov.OldestRecord = st.LastRecordedAt
		}
		if st.LastRecordedAt.After(cov.NewestRecord) {
			cov.NewestRecord = st.LastRecordedAt
		}
		stCopy := *st
		cov.Stats = append(cov.Stats, stCopy)
	}
	sort.Slice(cov.Stats, func(i, j int) bool { return cov.Stats[i].HitCount > cov.Stats[j].HitCount })
	return cov
}

// AllHosts returns the rollups for every host the tenant has covered.
func (t *Tracker) AllHosts(tenantID string) []types.HostCoverage {
	t.mu.Lock()
	hosts := map[string]bool{}
	for _, st := range t.byKey {
		if st.TenantID == tenantID {
			hosts[st.Host] = true
		}
	}
	t.mu.Unlock()
	var out []types.HostCoverage
	for h := range hosts {
		out = append(out, t.HostCoverage(tenantID, h))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].TotalHits > out[j].TotalHits })
	return out
}

// DueRerecords returns the (tenant, host, method, path) tuples whose
// next_record_due is in the past.
func (t *Tracker) DueRerecords(now time.Time) []types.EndpointStat {
	t.mu.Lock()
	defer t.mu.Unlock()
	var out []types.EndpointStat
	for _, st := range t.byKey {
		if !st.NextRecordDue.IsZero() && st.NextRecordDue.Before(now) {
			out = append(out, *st)
		}
	}
	return out
}

// pathTemplate normalises numeric / hex segments to placeholders so the
// per-endpoint stats don't explode for ID-bearing URLs.
//
// Examples:
//   /v1/customers/cus_abc123 → /v1/customers/{id}
//   /v1/charges/123/refunds   → /v1/charges/{id}/refunds
//   /api/orders/2024-05-15    → /api/orders/{id}
func pathTemplate(p string) string {
	parts := strings.Split(p, "/")
	for i, seg := range parts {
		if seg == "" {
			continue
		}
		if isIDLike(seg) {
			parts[i] = "{id}"
		}
	}
	return strings.Join(parts, "/")
}

func isIDLike(seg string) bool {
	// All-digit
	if isAllDigits(seg) && len(seg) >= 2 {
		return true
	}
	// UUID
	if len(seg) == 36 && countRune(seg, '-') == 4 {
		return true
	}
	// Stripe-style: <2-4 char prefix>_<≥4 alnum> (cus_abc123, ch_xyz789).
	if i := strings.IndexByte(seg, '_'); i >= 2 && i <= 4 && len(seg)-i-1 >= 4 {
		if isAlnum(seg[i+1:]) {
			return true
		}
	}
	// Hex token
	if len(seg) >= 16 && isHex(seg) {
		return true
	}
	return false
}

func isAlnum(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')) {
			return false
		}
	}
	return len(s) > 0
}

func isAllDigits(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return len(s) > 0
}

func isHex(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

func countRune(s string, r rune) int {
	n := 0
	for _, c := range s {
		if c == r {
			n++
		}
	}
	return n
}
