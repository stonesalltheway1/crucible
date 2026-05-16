# Runbooks

Common operational scenarios with specific actions. Each runbook assumes the on-call has access to the Crucible web console, the Honeycomb/Grafana dashboards, and the internal CLI (`crucible-ops`).

## Index

- [RB-01: Cache hit rate dropped below 60%](#rb-01-cache-hit-rate-dropped)
- [RB-02: Median task cost > $5 sustained](#rb-02-median-task-cost-exceeded)
- [RB-03: Egress violation event](#rb-03-egress-violation)
- [RB-04: Sandbox escape attempt](#rb-04-sandbox-escape)
- [RB-05: Sigstore Rekor unreachable](#rb-05-rekor-unreachable)
- [RB-06: KMS signing failure](#rb-06-kms-signing-failure)
- [RB-07: Verifier disagreement > 25%](#rb-07-verifier-disagreement)
- [RB-08: Cross-tenant access attempt](#rb-08-cross-tenant-access)
- [RB-09: Twin spawn failure rate > 2%](#rb-09-twin-spawn-failure)
- [RB-10: Tier 3 proof timeout rate > 25%](#rb-10-tier3-timeout-rate)
- [RB-11: Customer reports false promotion approval](#rb-11-false-promotion-approval)
- [RB-12: Convention drift detected at scale](#rb-12-convention-drift)
- [RB-13: Vendor LLM model deprecation announced](#rb-13-vendor-deprecation)
- [RB-14: Frontier model API outage](#rb-14-model-api-outage)
- [RB-15: Customer requests emergency tenant freeze](#rb-15-emergency-tenant-freeze)

---

## RB-01: Cache hit rate dropped {#rb-01-cache-hit-rate-dropped}

**Severity:** P1 (margin-impacting; not user-facing-broken)

**Detection:** Honeycomb alert on cache hit rate metric < 60% sustained 30 min.

**Immediate actions:**

1. Check per-vendor cache status:
   ```
   crucible-ops cache stats --by vendor --window 1h
   ```
2. Identify which vendor's cache is missing — Anthropic 1h cache often misses if system prompt drifted.
3. Check recent deploys for system-prompt or tool-definition changes that could have invalidated cache keys.

**Root causes (most common):**

- **System prompt drift** — a small edit invalidated all caches. Roll back if customer-visible degradation; otherwise let it re-warm.
- **Tool definition changed** — same as above.
- **Vendor-side cache TTL changed** — check vendor announcement page.
- **Tenant load shift** — heavy new tenant onboarded with different traffic shape; expect rebuild within 24h.

**Fix:**

- If prompt drift: revert the prompt change, deploy. Cache rebuilds in minutes.
- If vendor issue: status page, internal incident open, customer comms not required unless degradation is user-visible.
- If load shift: monitor; warm-up converges within a day.

**Postmortem template:** required for any cache regression > 4h.

---

## RB-02: Median task cost exceeded {#rb-02-median-task-cost-exceeded}

**Severity:** P1

**Detection:** Median per-task cost > $2.50 sustained 1h.

**Investigation:**

1. Check routing distribution: is more traffic going to Tier 2 (Opus 4.7) than expected?
2. Check cache hit rate (RB-01).
3. Check verifier cost ratio — should be ≤ 10% of total; if higher, verifier is being invoked too aggressively.
4. Check task wall-clock — long tasks correlate with retry budget consumption.

**Common causes:**

- New tenant with unusually complex tasks (high Tier 2 mix).
- Cache regression (RB-01).
- Verifier disagreement loop (RB-07).
- Anti-loop protocol not firing — investigate Bounded Budget Enforcer logs.

**Fix:**

- Adjust routing threshold if a tenant's tasks are systematically misclassified.
- Tune verifier rubric_score threshold if rejection rate too high.
- Surface to product: a tenant with sustained > $5 median cost is at risk of churn — Customer Success should reach out before they see the bill.

---

## RB-03: Egress violation {#rb-03-egress-violation}

**Severity:** P0 (security event)

**Detection:** Any `security.egress_violation` event.

**Immediate actions:**

1. **Page on-call security lead.**
2. Tenant + task isolation — the offending sandbox is already SIGKILL'd by Tetragon.
3. Pull attestation chain for the task:
   ```
   crucible-ops attestation chain <task_id>
   ```
4. Identify what the agent was attempting to reach:
   ```
   crucible-ops egress incident <incident_id>
   ```
5. Determine whether this was:
   - **Benign misconfiguration** — agent legitimately needed an endpoint that wasn't in the manifest.
   - **Prompt-injection attempt** — task description or input data tried to exfiltrate.
   - **Compromised dependency** — a package the agent installed tried to phone home.
   - **Sandbox escape attempt** (escalate to RB-04).

**Disposition:**

- If benign: update the task's allowed_egress manifest; release the gate; customer notified.
- If injection: investigate the input source; tighten LLM-judge filter; alert customer.
- If dependency: investigate the package; report to OSS maintainers if needed; tenant + similar tenants notified.
- If escape: RB-04.

**Customer comms:** within 24h, regardless of disposition. Trust posture requires transparency.

---

## RB-04: Sandbox escape attempt {#rb-04-sandbox-escape}

**Severity:** P0 (critical security event)

**Detection:** Any syscall anomaly that crosses the Firecracker boundary, OR any successful access to host filesystem from a sandbox, OR any unexpected privilege escalation.

**Immediate actions:**

1. **Page on-call security lead + CTO.**
2. Quarantine the affected sandbox host machine. Drain all other sandboxes off it.
3. Snapshot the host's memory + disk for forensic analysis.
4. Pull the full attestation chain + OTel trace for the offending task.
5. Identify the attack vector (model output? injected dependency? CVE in Firecracker / kernel? misconfigured seccomp?).

**Within 1h:**
- All other sandboxes audited for the same attack pattern.
- Tenant of the offending task notified (the task may be legitimate red-teaming).
- Internal incident open with named owner.

**Within 24h:**
- Public security advisory if the vector is reproducible (we don't hide).
- Patch deployed with Tier 4 attestation.

**Within 72h:**
- Public postmortem.
- Adversarial test case added to the Crucible Test Harness.

---

## RB-05: Sigstore Rekor unreachable {#rb-05-rekor-unreachable}

**Severity:** P1 (audit-trail gap, but local journaling continues)

**Detection:** Sigstore Rekor publish failure rate > 1% for 5 min.

**Immediate actions:**

1. Confirm Rekor public log status at status.sigstore.dev.
2. Verify our local journaling is still operational — attestations queue locally until Rekor recovers.
3. Set the customer-visible status banner: "Attestation publishing temporarily delayed (no functional impact on tasks)."

**During the outage:**

- Tasks continue normally. The attestation socket buffers locally.
- Promotions that require Rekor verification of inbound attestations are gated — they wait for Rekor to recover OR for fallback to a customer's self-hosted Rekor (enterprise tier).

**After recovery:**

- Local journal back-fills to Rekor in priority order.
- Postmortem if outage > 30 min.

---

## RB-06: KMS signing failure {#rb-06-kms-signing-failure}

**Severity:** P1 (promotions blocked; verification continues)

**Detection:** KMS signing API error rate > 1% for 5 min.

**Immediate actions:**

1. Identify which KMS — AWS, GCP, customer's HSM (per deployment).
2. Check vendor status page.
3. Surface customer-visible status: "Promotion approvals temporarily delayed."
4. Promotions queue; verification work continues.

**Fallback:**

- If vendor KMS is the issue: no fallback. Wait for recovery.
- If customer's HSM is the issue: customer's IT team is engaged via their incident channel.

---

## RB-07: Verifier disagreement {#rb-07-verifier-disagreement}

**Severity:** P2 (quality signal)

**Detection:** Verifier disagrees with human reviewer's verdict > 25% over 24h (shadow-mode metric).

**Investigation:**

1. Sample 20 recent disagreements.
2. Classify:
   - **Verifier too strict:** verifier rejected, human merged anyway. Likely calibration drift.
   - **Verifier too lenient:** verifier approved, human rejected. More serious — adjust thresholds upward.
   - **Genuine style disagreements:** noise; expected.

**Fix:**

- If strict: adjust rubric_score threshold (currently 0.85; consider 0.80 if disagreements are style-only).
- If lenient: add per-category check (e.g., security-related diffs require human review regardless of verifier verdict). Tighten the cross-family pairing if one family is consistently more lenient.

**Communicate** to customer if their tenant shows the pattern: "We noticed our verifier is rejecting more than your reviewers — we're tuning."

---

## RB-08: Cross-tenant access attempt {#rb-08-cross-tenant-access}

**Severity:** P0 (existential isolation issue)

**Detection:** Any read or write to a tenant-scoped resource by a process bearing a different tenant's OIDC subject.

**Immediate actions:**

1. **Page CTO + security lead + CEO.**
2. Quarantine the offending process and any code path it executed.
3. Identify whether data was actually exfiltrated.

**Within 1h:**
- All affected tenants notified individually.
- Public status banner if multiple tenants involved.

**Within 24h:**
- Patch deployed.
- Full postmortem.
- External security firm engaged for verification.

**Within 30 days:**
- Customer-facing report.

This is the most severe event class. The architecture should make it vanishingly unlikely, but we treat any positive signal as code red.

---

## RB-09: Twin spawn failure {#rb-09-twin-spawn-failure}

**Severity:** P1

**Detection:** Twin-runtime spawn failure rate > 2% for 10 min.

**Investigation:**

1. Check sandbox-provider status (E2B / Daytona / self-hosted Firecracker pool).
2. Check Neon API status if DB branch provisioning is the bottleneck.
3. Check our own control-plane's manifest validation pipeline.

**Common causes:**

- E2B / Daytona throttling under load.
- Neon API timeout (rare; usually < 2s).
- Manifest validation regression after a control-plane deploy.

**Fix:**

- If provider throttling: scale capacity request, fall back to alternative provider.
- If Neon: pre-warm a pool of "twin-base" branches.
- If our code: rollback.

---

## RB-10: Tier 3 timeout rate {#rb-10-tier3-timeout-rate}

**Severity:** P2

**Detection:** Tier 3 proofs timing out > 25% over 24h.

**Investigation:**

1. Sample the failing proofs by tenant + prover (Dafny / Lean / TLA+).
2. Look for:
   - Misclassification (Tier 3 triggered on non-critical code).
   - Inadequate LLM-driven proof hints (the hint model is failing to converge).
   - Library-version regressions (Dafny / Lean updates).

**Fix:**

- If misclassified: tune the critical-path classifier. Lower escalation rate.
- If hint convergence: adjust the hint-generation prompt; refresh fine-tuned hint model if applicable.
- If library regression: pin to previous version; open issue upstream.

**Customer-facing:** the Tier 2.5 fallback (PBT + mutation + CODEOWNER review) means user impact is bounded — they still get verified PRs, just with a different proof chain.

---

## RB-11: False promotion approval {#rb-11-false-promotion-approval}

**Severity:** P0

**Scenario:** Customer reports an agent-merged PR was wrong and shouldn't have been approved.

**Investigation:**

1. Pull the full attestation chain.
2. Identify which gate let the change through:
   - Verifier approved when it shouldn't have? → RB-07 + adjust thresholds.
   - Rego policy auto-approved when human approval was required? → Policy bug; fix and redeploy.
   - Human approver clicked approve in error? → Procedural memory note: "human approved X; we maintain the audit trail."
3. Roll back the change via the customer's existing rollback infrastructure (we don't auto-undo merged PRs).

**Customer comms:** acknowledge within 1h; full incident report within 24h with the chain-of-evidence.

---

## RB-12: Convention drift {#rb-12-convention-drift}

**Severity:** P3 (informational)

**Detection:** > 10 conventions flagged drifting per tenant per week.

**Investigation:**

This is usually a *signal*, not a problem — the team's practices are evolving. The drift detector is doing its job.

**Action:**

- Surface the drift to the customer in the weekly digest.
- The customer reviews and either confirms (active → active), supersedes (active → superseded + new active), or archives (active → archived).
- We monitor for systemic drift (multiple tenants in the same stack drifting the same convention) — that suggests our OSS default is outdated.

---

## RB-13: Vendor deprecation {#rb-13-vendor-deprecation}

**Severity:** P2 (proactive)

**Scenario:** Anthropic / Google / OpenAI announces deprecation of a model in our routing table.

**Action:**

1. Add to the routing config: deprecated model → alternate model.
2. Test the alternate on the Crucible Test Harness; verify no regression.
3. Customer-visible changelog entry.
4. Update [01-architecture/model-routing.md](../01-architecture/model-routing.md).
5. Remove deprecated model from the routing table after the vendor's deprecation date.

For BYOK customers, they receive a notification but can override our default.

---

## RB-14: Frontier model API outage {#rb-14-model-api-outage}

**Severity:** P0 if primary executor; P1 if alternate.

**Detection:** Anthropic / Google / OpenAI API error rate > 5% for 5 min.

**Immediate actions:**

1. Auto-fail-over to alternate model in the routing table.
2. Surface customer-visible status: "Primary model degraded; routing to alternate. Quality may differ slightly."
3. Continue to monitor.

**During the outage:**

- Tasks continue with the alternate.
- Cache hit rate drops temporarily (different model = different cache).
- Cost may shift (alternate may be more expensive); cost-meter alerts as usual.

**After recovery:**

- Resume normal routing.
- Postmortem if outage > 1h.

---

## RB-15: Emergency tenant freeze {#rb-15-emergency-tenant-freeze}

**Scenario:** Customer requests an immediate halt of all agent activity (e.g., active incident, suspected compromise).

**Action:**

1. Authenticate the requester via established out-of-band channel.
2. Set tenant freeze:
   ```
   crucible-ops tenant freeze <tenant_id> --reason "..." --requested_by <oidc>
   ```
3. All in-flight tasks halt at next checkpoint.
4. All new task submissions return `TenantFrozen`.
5. Customer maintains access to web console, attestation log, and memory browser.

**Unfreeze:**

1. Customer requests unfreeze via the same channel.
2. We unfreeze with a signed attestation explaining the lifecycle.

---

## Runbook maintenance

- Every postmortem must update or create a runbook.
- Runbooks reviewed quarterly.
- The runbook list is itself versioned; major changes documented in CHANGELOG.
- On-call review covers the top 5 most-likely-to-fire runbooks each rotation.
