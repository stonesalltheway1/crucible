# Twin Runtime

The execution surface for every agent action. The core innovation: **the agent never touches real systems directly**. Everything happens in a per-task ephemeral mirror — filesystem, database, services, secrets — and changes are promoted to real systems only via the signed Promotion Contract.

## Composition

A twin is composed of six layers, all spun up at task start and destroyed at task end:

1. **Sandbox** — Firecracker microVM, ~110ms cold start.
2. **Filesystem twin** — git worktree + overlayfs.
3. **Database twin** — Neon CoW branch (or per-engine equivalent).
4. **Service twin** — Hoverfly replay tapes.
5. **Secrets twin** — Infisical scoped dynamic tokens.
6. **Network policy** — Cilium/Tetragon eBPF allowlist.

Each is independently destructible; together they form a faithful enough mirror that agent changes which "work" in the twin will work after promotion in ~99% of cases (see [06-research/tape-coverage-strategy.md](../06-research/tape-coverage-strategy.md) for the residual failure modes).

## Layer 1: Sandbox

**Default pick:** E2B (Firecracker-based, hosted). $0.0504/vCPU-hr; ~150ms cold start; mature Python/TS SDK; 24-hour max session.

**Self-hosted pick:** Firecracker + containerd + ZFS. Marginal cost ≈ $0/twin after initial orchestrator build. Snapshot-restore 3–10ms once warm.

**Solo-founder pick:** Daytona (~90ms creation, $200 free credits) or Fly Machines (scale-to-zero, $0.07/CPU-hr).

**Filesystem layout inside the sandbox:**

```
/work/repo            git worktree (depth 1, base_sha pinned)
/work/scratch         overlayfs upper, agent's mutation surface
/work/tapes           mounted Hoverfly tapes (read-only)
/work/secrets         tmpfs, ephemeral Infisical tokens (mode 0400)
/work/.crucible       attestation socket + control fds
/work/.crucible/log   per-action append-only event journal
```

**Concurrency:** N concurrent twins per task (default 5) for fan-out exploration. Each gets its own Neon branch + Hoverfly tape ref. Snapshots are taken at checkpoint boundaries (post-plan, post-tier-0-verify, post-tier-1-verify) so the agent can fork-and-explore alternative approaches without re-running setup.

**Lifecycle:** sandbox is created with the task, killed on completion or 1-hour absolute TTL. `sandbox.kill()` is unconditional and recursive — there is no "save state for later"; that's what the attestation log + Promotion Bundle are for.

## Layer 2: Filesystem Twin

**Git worktree** at `base_sha`, depth 1 (just the working tree, not full history). For huge repos use `git clone --filter=blob:none --depth=1` to defer blob fetches; the cartographer ([memory-layer.md](memory-layer.md)) only needs file paths and symbols.

**OverlayFS upper** on top of the worktree. The agent writes here; the lower layer (the actual repo) is read-only. This is the cheapest possible per-task isolation — no per-file copy until first write. Discarding the twin = `umount overlay`.

**Build cache** lives in a separate ZFS dataset cloned per task; nuked at task end unless explicitly persisted to the per-tenant cache layer.

**Why not git worktrees alone?** Worktrees give isolation but not COW. A 50MB `node_modules` install slows every twin without overlayfs.

## Layer 3: Database Twin

**Default pick: Neon Postgres branching.** `POST /projects/{id}/branches` returns a connection string in 1–2 seconds. Branch cost $0.002/hr; storage shared with parent so marginal cost is near-zero for typical task duration. Cold-start 400–750ms is fine for ephemeral.

**Twin-base branch:** every project has a "twin-base" branch which is itself a daily snapshot of production with PII scrubbed (see Layer 4 scrub pipeline applied to dumps too). Per-task branches are children of twin-base, not children of `main`.

**Schema migration verification:** migrations run against the twin branch first. The verifier diffs the resulting schema against expected and checks for destructive DDL on critical tables (which must be in an explicit allowlist for the task to proceed).

**Per-engine equivalents:**

| Engine | Mechanism | Latency |
|---|---|---|
| Postgres | Neon CoW branch | 1–2s |
| MySQL | PlanetScale branch | seconds |
| SQLite/libSQL | Turso branch | instant |
| MongoDB | Atlas snapshot-restore-to-new-cluster | minutes |
| Redis/KV | Fresh `redis-server` inside sandbox | <1s |
| S3 | MinIO inside sandbox + rclone mirror prefix | seconds |
| ClickHouse | Table-level `CREATE TABLE … CLONE AS` | seconds |

For Postgres-shaped customers Neon is the obvious answer. Other engines either work or we explicitly do not support them in v1 (e.g., Cassandra, Aurora-only stacks).

## Layer 4: Service Twin (Tapes)

**Default pick: Hoverfly OSS + custom PII scrubber.**

### Recording

A Crucible-installed agent runs in **shadow mode** against the customer's staging (or a sanctioned subset of production traffic). eBPF or Envoy taps egress HTTP/gRPC and records to content-addressed tape files keyed by `(service, endpoint, request_hash)`.

### Scrub pipeline (at capture, before persistence)

1. **Presidio Analyzer + Anonymizer** — names, SSN, credit cards, phones, addresses, emails, MRNs.
2. **spaCy NER** as backbone + a separate pass for free-text fields.
3. **FF3-1 format-preserving encryption** for structure-bearing fields (BINs, phone formats, account-number checksums).
4. **Deterministic pseudonymization** keyed per-tape-set so referential integrity is preserved.
5. **Audit log** — every tape entry records which scrubbers fired and which fields were rewritten.

PII scrubbing **must** run at capture, before bytes hit disk. Scrubbing on replay is too late.

### Replay decision tree

On every outgoing request from the twin:

```
1. Match tape entry exactly (path + method + sig)        → REPLAY, tag hit-exact
2. Match by template (path pattern + method, ID diffs)   → REPLAY with param rewrite, tag hit-template
3. Miss but endpoint in OpenAPI spec, READ-ONLY method   → SYNTHESIZE from schema (Prism/Microcks + Faker + optional LLM), tag synth-readonly
4. Miss in spec, MUTATING method                         → DETERMINISTIC STUB + journal write-side mutation, tag synth-mutation
5. Miss, NOT in spec, live-call allowed in manifest      → PASSTHROUGH via scrubbing proxy, persist for future, tag live-passthrough
6. Miss, NOT in spec, live NOT allowed, auth required    → FAIL CLOSED 599, tag miss-blocked
7. Miss, NOT in spec, live NOT allowed, no auth          → Policy-driven; default 599
```

Every replayed response carries `X-Crucible-Tape: hit-exact | hit-template | synth-readonly | synth-mutation | live-passthrough | miss-blocked` so the agent and the verifier both *see* whether the response is trustworthy.

### Policy knobs (surfaced to users)

- `tape.mode = strict | hybrid | adaptive`
- `tape.synth_engine = none | schema | schema+llm`
- `tape.allow_live = [host_allowlist]`
- `tape.mutation_policy = journal | block`

Defaults: `hybrid + schema+llm + [] + journal`.

Full reasoning in [06-research/tape-coverage-strategy.md](../06-research/tape-coverage-strategy.md).

## Layer 5: Secrets Twin

**Default pick: Infisical OSS + AWS KMS** for the production-promotion signing key.

**Architecture:**

- The sandbox gets an Infisical sidecar that holds the only long-lived credential.
- On request from the agent (via `twin.secret.get(name)`), the sidecar issues a dynamic, twin-scoped token with sub-minute TTL.
- The token value is never returned to the agent's process memory; it's only injected into the outgoing HTTP request at the egress proxy via header rewrite.
- **Real production credentials are physically unreachable.** They live in an HSM-backed vault on a separate VPC. The only code path to use them is the Promotion Contract's KMS-signed credential lease.

**Per-environment alternatives:**

| Vault | Use when |
|---|---|
| Infisical Cloud | Default for hosted tier |
| Infisical OSS self-host | Self-hosted enterprise |
| HashiCorp Vault Community | If customer already uses Vault |
| AWS Secrets Manager + STS | All-in-AWS customers |
| GCP Secret Manager + Workload Identity | All-in-GCP |
| 1Password Connect | Small-team, dev-friendly |

**Unseal ceremony for destructive prod actions:** a Slack-button approval triggers AWS KMS / GCP Cloud HSM / YubiHSM (per deployment) to sign a single-use, time-boxed, action-scoped credential lease. The signed lease is consumed by the deploy pipeline. Never returned to the agent.

## Layer 6: Network Egress + Destructive-Op Gate

### Egress allowlist

**Cilium + Tetragon TracingPolicy** drops any TCP connection outside the per-task manifest allowlist with `SIGKILL`. Each task manifest declares `allowed_egress: [...]`; everything else is dropped at the kernel.

For solo-founder tier, skip the eBPF complexity and use a userspace egress proxy (mitmproxy with an allowlist) until you have multiple concurrent agents.

### Syscall shim

The agent's shell (`twin.shell.exec`) is wrapped in a shim that intercepts destructive operations both statically (command-line pattern match) and at exec-time (syscall filter). Patterns matched:

```
rm -rf, rm -fr, find … -delete
DROP, TRUNCATE, DELETE … WHERE 1=1
git push --force, git push -f, git reset --hard origin
kubectl delete, helm uninstall, terraform destroy
aws … delete-*, gcloud … delete, az … delete
railway down, fly destroy, vercel rm
rmdir /s, Remove-Item -Force -Recurse
mongorestore --drop, redis-cli FLUSHALL
```

Matched commands are intercepted before exec and converted to typed `DestructiveProposal`:

```typescript
{
  command: string,
  justification: string,        // agent must explain
  blast_radius_estimate: {      // computed by shim
    affected_resources: string[],
    reversibility: "none" | "snapshot" | "soft-delete",
    impact_score: number
  },
  signed_by_agent_oidc: string  // Sigstore keyless cert
}
```

Routed to the gate. Default policy:

- **Twin-scoped destructives** (operating on Neon branch, sandbox FS, Hoverfly tape): gate auto-approves.
- **Real-scoped destructives** (operating on production via promoted credential lease): gate requires human signature via Sigstore keyless OIDC.

Every proposal — approved or denied — is logged to Sigstore Rekor.

## The agent SDK surface

The agent's only access to the twin is through a small set of typed primitives. There is no "raw exec" path; even shell commands go through the syscall shim.

```typescript
// Filesystem (overlayfs upper)
twin.fs.read(path: string): FileContent
twin.fs.write(path: string, content: string): WriteAttestation
twin.fs.delete(path: string): DestructiveProposal | DeleteAttestation
twin.fs.list(glob: string): Path[]

// DB (Neon branch)
twin.db.query(sql: string): QueryResult
twin.db.migrate(file: string): MigrationProposal   // verifier-gated

// Services (Hoverfly tape or live)
twin.svc.call(service: string, endpoint: string, payload: any): Response

// Secrets (vault, ephemeral, twin-scoped)
twin.secret.get(name: string): SecretRef           // value injected at egress, never returned

// Shell (syscall-shim wrapped)
twin.shell.exec(cmd: string): ExecResult | DestructiveProposal

// Tests + verifiers
twin.test.run(suite?: string): TestReport
twin.verify.tier0(diff: Diff): MutationReport
twin.verify.tier1(spec: PBTSpec): PBTReport
twin.verify.tier2(spec: ContractSpec): ContractReport
twin.verify.tier3(spec: FormalSpec): ProofReport
twin.verify.tier4(): HonestCIReport

// Memory
twin.memory.recall(query: string, scope: Scope): Memory[]
twin.memory.note(fact: string, source: SourceRef): MemoryId

// Plan + budget
twin.plan.propose(plan: Plan): PlanApproval
twin.plan.checkBudget(): Budget
twin.plan.checkpoint(name: string): Snapshot

// Promotion
twin.promote(bundle: PromotionBundle): PromotionId
```

Every call emits an in-toto attestation; the SDK auto-signs via the agent's keyless OIDC. Full reference in [03-sdk/agent-sdk-reference.md](../03-sdk/agent-sdk-reference.md).

## Lifecycle in detail

```
t=0     Control Plane validates task manifest.
t+10ms  Mint Infisical scoped token (TTL = task duration estimate).
t+20ms  POST /projects/{id}/branches → Neon branch DSN.
t+30ms  Reserve E2B sandbox slot.
t+150ms Sandbox running. Git worktree on base_sha checked out.
        Overlayfs mounted. Hoverfly tapes mounted RO.
        Egress proxy started with manifest allowlist.
        Infisical sidecar injected.
        Syscall shim active.
t+200ms Agent receives twin.* SDK handle.
t+200ms — Agent executes plan steps. Every action emits in-toto attestation
        via the /work/.crucible attestation socket.
t=N     Agent calls twin.verify.* (Tier 0/1/2/3) and twin.test.run().
        Verifier process spins up in separate sandbox with different model.
t=N+M   Verifier emits VerifierApproval or VerifierRejection.
        If approved, agent calls twin.promote(bundle).
        Control Plane runs Promotion Contract (see promotion-contract.md).
t=N+M+P sandbox.kill(). Neon branch DELETE. Infisical token revoked.
        Final attestation published.
```

## Failure modes and recovery

### Sandbox crash mid-task

Detected via heartbeat. Last attestation timestamp identifies progress. Control plane offers user three options: resume from last checkpoint, fork from a named checkpoint, or abandon. No automatic resume — the user must approve.

### Neon branch creation timeout

After 10s, fall through to a smaller "lite" twin without a DB branch. Verifier downgrades any DB-touching changes to "unverified" and requires explicit human approval at promotion.

### Hoverfly tape corruption

Detected at mount via checksum. Task fails with `TwinSetupError: tape integrity`. Fallback: retry with synth-engine-only (no recorded tape).

### Egress policy violation

Hard kill of the offending process. Logged as security event. Agent receives a clean error in the SDK call. Task continues if recoverable.

### Destructive-op shim bypass attempt

Shim is enforced at multiple layers: command-line parse, ptrace syscall filter, eBPF post-exec. Bypassing one is hard; bypassing all three within sandbox is the threat model boundary — if achieved, that's a red-team finding, not a normal failure mode.

## What's deliberately not in scope for v1

- **GPU-accelerated twins.** ML workload customers are not v1 ICP.
- **Multi-region twin orchestration.** Single-region per task.
- **State snapshot/restore across sessions.** Twins are ephemeral; persistence is the Memory Layer's job.
- **Long-running daemon emulation.** Tasks complete in minutes; the twin is not a persistent dev env.

See [06-research/tape-coverage-strategy.md](../06-research/tape-coverage-strategy.md) for the service-replay deep dive and [threat-model.md](threat-model.md) for the security architecture.
