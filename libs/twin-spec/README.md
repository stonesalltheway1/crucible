# twin-spec

Source-of-truth schemas for the entire Crucible system. Every protobuf message in `proto/crucible/v1/` is consumed by:

- `libs/sdk-go/` — agent-side Go SDK
- `libs/sdk-ts/` — TypeScript SDK
- `libs/sdk-py/` — Python SDK
- `libs/sdk-rs/` — Rust SDK
- `apps/control-plane/` — server-side Go service
- The verifier, promotion-gate, distiller, and every other component that touches a `Plan`, `PromotionBundle`, `VerifierApproval`, or attestation.

## Files

```
proto/crucible/v1/
  common.proto           Glob, Scope, FileChange, Diff, SourceRef, SecretRef,
                         ExecResult, BlastRadius, DestructiveProposal, Risk,
                         Complexity, ModelTier, ErrorCode, CrucibleError
  task.proto             Task, TaskStatus, Plan, PlanStep, PlanApproval,
                         PlanRejection, Routing, Budget
  memory.proto           Convention, Memory, ComplianceReport
  verification.proto     TierResult, TierResults, VerifierApproval,
                         VerifierRejection, PromotionBundle, PromotionId,
                         PromotionStatus
  attestation.proto      InTotoStatement, DsseEnvelope, RekorEntry,
                         all 13 predicate types
  control_plane.proto    ControlPlaneService (Health/SubmitTask/GetTask/
                         ListTasks/ApprovePlan/RejectPlan/ReplanTask/
                         GetBudget)
  agent_sdk.proto        AgentSdkService (twin.fs.*, twin.shell.*,
                         twin.memory.*, twin.plan.*)

schemas/
  *.json                 JSON Schemas for each https://crucible.dev/*/v1
                         predicate, used by the verifier and external auditors
                         to validate signed payloads pulled from Rekor or the
                         local journal.
```

## Regenerating SDK stubs

```bash
nix develop
./scripts/regen-proto.sh
```

This runs `buf generate` against `buf.gen.yaml`, producing output under each `libs/sdk-*/.../gen/` directory.

## Phase 1 status

Phase 1 ships hand-rolled Go types under `libs/sdk-go/crucible/v1/` so the control plane builds without `buf generate`. The proto source-of-truth is authoritative; CI (once wired) will diff buf-generated code against the hand-rolled equivalents and fail on drift.

## Versioning

All packages are `crucible.v1.*`. Breaking changes bump to `crucible.v2.*` and stay backward-compatible for one major version, per `docs/02-engineering/repo-structure.md`. Old predicate JSON Schemas remain readable indefinitely — Sigstore Rekor is append-only.
