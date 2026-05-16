# Tier 3 Trigger Automation

Resolves open question #3 from the original architecture: how do we auto-classify which code paths are `@critical` so users don't have to annotate everything by hand?

## What "critical" means (six orthogonal axes)

Critical is a multi-dimensional predicate, not binary. Different axes imply different Tier 3 tools:

| Axis | Examples | Tier 3 tool |
|---|---|---|
| Performance-critical (hot path) | p99-dominating code, inner loops | TLA+ for concurrency; Z3 for invariants |
| Security-sensitive | authn/authz, crypto, deserialization, input validation | CBMC for memory safety; SAW for crypto; Dafny for state machines |
| Money paths | billing, refunds, ledgers, currency conversion, idempotency | Dafny or Lean (clean math invariants) |
| Data-integrity | migrations, replication, leader election, audit logs | TLA+ (AWS DynamoDB precedent) |
| Safety-of-life / regulatory | medical (IEC 62304), automotive (ISO 26262 ASIL-D), aviation (DO-178C-A) | Tool dictated by certification |
| Blast radius (centrality) | shared utilities imported by 100+ modules | Tool depends on actual function |

The sixth axis — blast radius — is critical for non-obviously-critical code. A bug in `utils/retry.ts` with fan-in 230 is more dangerous than a bug in `billing/edge_case_handler.ts` that's only called from one place.

## Existing tools we draw signal from

| Family | Tools | What we use |
|---|---|---|
| SAST severity | Semgrep p/security-audit, CodeQL severity scores, SonarQube hotspots, Snyk Code priorityScore | Severity tag → criticality input |
| Dependency scanners | Trivy, Grype, OWASP Dependency Check, Dependabot | CVE-touched files signal |
| Ownership / tier metadata | Datadog Service Catalog tier, Backstage criticality, CODEOWNERS, PagerDuty service-criticality | Direct critical-path signal |
| Hotspot detection | CodeScene, Bridgecrew, GitGuardian | Churn × complexity hotspots |

Crucible's classifier *consumes* these tools' outputs rather than re-implementing them.

## Production-signal mining

Higher fidelity than static heuristics because observed:

- **Incident post-mortems.** Parse Rootly / FireHydrant / Jeli exports, Confluence / Notion postmortem pages. Run NER + file-path regex. Files mentioned in 3+ Sev1/Sev2 in last year are unambiguously critical.
- **SLO / error-budget data.** Map endpoints to source functions via OpenTelemetry semantic conventions (`code.filepath`, `code.function` span attributes). Endpoints attached to SLOs ≥99.9% promote their backing functions.
- **Pager frequency.** PagerDuty → JIRA → git-blame chain. "Files blamed by alerts that paged ≥2 engineers in 90 days."
- **PR review intensity.** Files attracting ≥3 distinct reviewers, PRs with ≥20 comments, PRs blocked (`REQUEST_CHANGES`) >30% of the time.
- **Test coverage gradient.** Files in 95th percentile of `coverage_lines / sloc` — engineers wrote disproportionate tests for a reason.
- **Churn-vs-review ratio.** High churn + low review = risky but underwatched. High churn + high review = critical and watched.

## Path-pattern heuristics

The cheap baseline. Crucible ships default regex sets, namespaced by axis:

```regex
SECURITY:   \b(auth[nz]?|oauth|saml|jwt|session|login|signin|password|
            secret|token|cred|kms|kdf|crypto|cipher|sign|verify|hash|
            mtls|tls|x509|csrf|cors|sanitiz|escape|validate|permit|
            rbac|acl|policy|capabilit|sandbox)\b/i

MONEY:      \b(billing|invoice|payment|payout|refund|charge|subscri|
            ledger|account(ing)?|balance|currency|fx|tax|vat|gst|
            stripe|adyen|braintree|paypal|wallet|escrow|settle)\b/i

DATA:       \b(migrat|schema|replicat|snapshot|backup|restore|
            audit_?log|gdpr|pii|consensus|raft|paxos|leader|quorum|
            checksum|wal|journal|fsync)\b/i

SAFETY:     \b(asil|sil[1-4]|do178|iec6\d{3}|hipaa|hitrust|fda|
            iso26262|misra|safety|interlock|estop|failsafe)\b/i

HOTPATH:    \b(hot|fast_?path|inner_?loop|simd|vectoriz|kernel)\b/i
```

## Comment-marker mining

Grep code + docstrings for risk markers:

```
DANGER, DO NOT TOUCH, HERE BE DRAGONS,
// HACK, FIXME critical, XXX security,
@critical, @dangerous, WARNING:, TODO(security),
SECURITY:, THREAD-SAFETY:, INVARIANT:
```

Files with ≥2 markers are candidates.

## Import-graph centrality

Build static call/import graph via tree-sitter + language-specific resolvers (pyan, jdeps, go-callvis, ts-morph). Compute:

- **Fan-in:** distinct modules importing this one.
- **PageRank** on the call graph.
- **Articulation-point status:** does removing this node disconnect a subgraph?

Top 5% by PageRank or fan-in ≥ 50 → critical regardless of subject matter. This is what catches `utils/retry.ts`.

## CVE recency

Files touched by any CVE patch in last 24 months (extractable from `git log --grep='CVE-'` + OSV-DB) get a permanent boost.

## Test-name harvesting

Functions referenced by tests named `test_security_*`, `test_critical_*`, `test_*_invariant`, `test_*_property`, or wrapped in `@pytest.mark.critical` / `[Trait("Category", "Critical")]` inherit the tag.

## LLM-based classification

A small LLM judge (Haiku 4.5, cached by content hash) categorizes each candidate file:

```
Classify this code into one of:
  {security, money, data-integrity, safety, performance,
   infrastructure, ui, plumbing, test, dead}.
Return JSON:
  {category, confidence (0..1), reasoning (one sentence)}.
```

Cached aggressively. Re-runs free. Temperature 0; ensemble 3 calls when confidence < 0.7.

LLM judges catch context heuristics miss: `validator.py` could be input validation (critical) or UI-form schema validation (non-critical). Only semantic reading distinguishes.

## The weighted multi-signal score

```
S(file) = 100 * sigmoid(
    1.5 * path_pattern_score        // 0..1, max of axis regex matches
  + 1.2 * llm_category_score        // 0..1, weighted by confidence
  + 1.0 * fanin_centrality          // log-normalized PageRank
  + 0.9 * incident_mention_score    // postmortem hits, decayed
  + 0.8 * slo_backing_score         // 1.0 if backs ≥99.9 SLO
  + 0.7 * review_intensity_score    // reviewers + comments/PR
  + 0.7 * cve_history_score         // 1.0 if CVE-touched in 24mo
  + 0.6 * test_coverage_gradient    // z-score within repo
  + 0.5 * comment_marker_score      // DANGER/HACK density
  + 0.4 * codeowners_team_score     // owned by sec/payments/sre
  - 0.5 * ui_or_test_penalty        // pure UI/test files lose points
)
```

### Threshold bands (defaults; per-tenant tunable)

| Band | Score | Behavior |
|---|---|---|
| Cold | 0–39 | Tier 1 only (lint + type-check + diff-scoped mutation) |
| Warm | 40–59 | Tier 2 (property tests + mutation) |
| Hot | 60–79 | Suggest Tier 3 (one-click confirm in PR comment) |
| Molten | 80–100 | Auto Tier 3 + block merge until proof discharged or waived |

## Calibration

The default weights above are starting points. The actual weights are tuned per-tenant.

`crucible calibrate` command:

1. Cartographer samples 200 files stratified across the score distribution (50 obvious-critical, 50 obvious-non-critical, 100 ambiguous).
2. A team engineer labels each: `critical | warm | cold | not-applicable`.
3. Logistic regression fits the weight vector against labels.
4. Defaults from the general OSS-trained model serve as priors.
5. Online learning thereafter: every override (in production usage) updates weights.

Calibration takes ~1 hour of human time; pays for itself in reduced false-positive Tier 3 escalation within a week.

## Asymmetric cost

False-positive cost ≈ 20 min of CI + engineer annoyance.
False-negative cost ≈ a Sev1 in production.

Ratio: ~1:1000. The scorer is **biased toward over-escalation**. Combined with a cheap override path, over-escalation doesn't poison adoption.

## PR-level trigger

A PR auto-escalates to Tier 3 when ANY of:

1. Touches any file with `S ≥ 80`.
2. Touches ≥3 files with `S ≥ 60`.
3. Modifies a function annotated `@critical` (explicit or inherited).
4. Diff contains security/money regex tokens AND is ≥40 lines.

## Override flow

Three mechanisms, all recorded for learning:

1. **Inline comment:** `// crucible: not-critical` (or `# crucible: not-critical` per language).
2. **PR command:** `/crucible skip-tier3 reason:"..."` — requires reason string, logged to procedural memory.
3. **CODEOWNERS designated approver:** `@security-team` or `@principal-eng` approval of the skip carries weight 2× a normal override.

Every override becomes a training example: `(file_features, true_label=non_critical, overridder, reason)`.

Conversely: a non-escalated PR followed by an incident touching its files within 30 days is a hard negative — boost weights that *would have* caught it.

## Confidence-driven UX

Platt-scaled probability determines the UI:

| P(critical) | UI |
|---|---|
| ≥ 0.9 | Silent auto-escalate; surfaced in PR as a checkbox pre-ticked |
| 0.6–0.9 | PR comment: "I think this needs Tier 3 because: [top-3 signals]. Confirm?" |
| 0.3–0.6 | Foldable suggestion; no friction |
| < 0.3 | Silent |

## Auto-annotation of functions

A function gets `@critical` auto-annotated when:

1. Lives in a file with `S ≥ 70`.
2. AND any of:
   - Is exported/public.
   - Called by ≥2 files outside its module.
   - Handles untrusted input (parameter type matches `Request`, `bytes`, `str` from network sources).

The annotation persists in `.crucible/annotations.toml` (sidecar file), surviving refactors that move the function.

## Tier 3 timeout fallback

Proofs are slow; Tier 3 timeouts happen. Crucible does **NOT fail open**:

1. **Wall-clock budget:** Dafny 10 min, Lean 30 min, TLA+ model-check 20 min.
2. **On timeout, degrade to Tier 2.5:**
   - Exhaustive PBT (≥10,000 cases)
   - Mutation testing on the diff
   - **Mandatory CODEOWNER human review.**
3. **Cache partial proofs.** Incremental verification on next PR resumes where it left off.
4. **Surface to dashboard.** Chronic Tier 3 timeouts on the same code path are a signal to invest in proof engineering.

## Examples

**Obvious critical:** `src/auth/oauth_callback.py` (path match: auth, LLM category: security, fan-in 12, CVE history 2 in 18mo, owned by `@security-team`) → `S ≈ 92` → Molten.

**Obvious critical:** `services/billing/refund_engine.go` (path: billing + refund, SLO-backing 99.95% revenue endpoint, postmortem mentions 4, review intensity 3.2 reviewers/PR) → `S ≈ 88` → Molten.

**Obvious non-critical:** `web/components/MarketingHeroBanner.tsx` (UI penalty, LLM category: ui, fan-in 1, no security keywords) → `S ≈ 8` → Cold.

**Genuinely ambiguous (the load-bearing case):** `lib/utils/retry.ts` — small, plumbing-looking, but fan-in 230. Path heuristics say low; centrality says very high. Score lands `S ≈ 64` → Hot (suggest Tier 3). A bug in retry.ts that double-charges on retry is a money path even though nothing in its filename says "money."

**Adversarial mislabel:** `tools/payment_simulator_for_demos.py` — heuristic says money, LLM judge correctly flags as demo simulator, dropping score < 40.

This last example is the load-bearing argument for the ensemble — no single signal layer is sufficient.

## Open issues

- **Per-monorepo subdirectory tuning.** A monorepo may have wildly different criticality between `marketing/` and `payments/`. Handled by file_glob scope; v2 may add explicit subdirectory weight overrides.
- **Cross-language Tier 3 tool gaps.** Elixir, Crystal, etc. have weak Tier 3 tooling. Fallback to Tier 2.5 with explicit warning.
- **Calibration data freshness.** Weight vectors age as codebase evolves. Quarterly auto-recalibration scheduled.

## References

- [01-architecture/verifier-pipeline.md](../01-architecture/verifier-pipeline.md)
- [ADR-008: Tier 3 annotation default-off](../05-decisions/ADR-008-tier3-annotation-default-off.md)
