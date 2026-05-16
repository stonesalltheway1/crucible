# attestation

Crucible's in-toto / DSSE / Sigstore plumbing. Every meaningful action across the system emits one of the 14 predicate types defined in `libs/twin-spec/proto/crucible/v1/attestation.proto`; the library here handles the build → sign → publish chain.

## API

```go
import "github.com/crucible/attestation"

signer, _ := attestation.NewLocalEd25519Signer("")
journal, _ := attestation.NewLocalJournalPublisher("")
svc, _    := attestation.NewService(signer, journal)

receipt, _ := svc.Emit(ctx,
    cruciblev1.PredicatePlanProposal,
    "task_01H/plan",
    canonicalPlanBytes,
    planProposalPredicate,
)
fmt.Println(receipt.UUID, receipt.URL)
```

## Components

- `BuildStatement(predicateType, subjectName, subjectDigest, predicate)` → `InTotoStatement`
- `Signer` interface — implemented by:
  - `LocalEd25519Signer` — Phase-1 default, key on disk at `~/.crucible/dev-keys/`
  - `SigstoreKeylessSigner` — Phase-2 stub (OIDC via Fulcio)
- `Publisher` interface — implemented by:
  - `LocalJournalPublisher` — Phase-1 default, hash-chained append-only JSONL
  - `RekorV2Publisher` — Phase-2 stub (gated by `CRUCIBLE_REKOR_PUBLISH=1`)
- `Service` — the high-level `Emit(...)` facade combining signer + publisher.
- `Verify(envelope, pubKey)` — DSSE signature verification.
- `SchemaFor(predicateType)` + `ValidateRequired(predicateType, payload)` — minimal JSON Schema validation against the embedded `libs/twin-spec/schemas/*.json`.

## Phase 1 status

- **Local journal is the default** because Sigstore Rekor v2 has not GA'd as of May 2026. The journal is hash-chained — every entry's UUID = `sha256(prev_uuid || envelope_bytes)` — so attestations recorded today remain tamper-evident.
- **Local Ed25519 keys** rather than Fulcio-issued certs. The DSSE envelope shape is identical; Phase 2 swaps in `SigstoreKeylessSigner` with no caller changes.
- **`ValidateRequired` only checks `required` keys**; full Draft 2020-12 validation lands in Phase 2 once we add `github.com/santhosh-tekuri/jsonschema`.

## Tests

```bash
cd libs/attestation && go test ./...
```

Unit tests cover sign-and-verify round-trips, journal hash-chain integrity, and that every predicate type has both a Go type and a JSON Schema (via `AllSchemas()` + `cruciblev1.AllPredicateTypes`).
