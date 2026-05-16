# sdk-go

The Crucible Agent SDK for Go. Phase 1 ships types only; the runtime `twin.*` surface ships with Phase 2.

## Layout

```
crucible/v1/
  types.go              Plan, Task, Budget, Routing, Diff, Convention, ...
  attestation_types.go  WriteAttestation, MigrationAttestation, ... (14 predicate payloads)
  json.go               Scope union codec + CrucibleError
```

These types are hand-rolled to match `libs/twin-spec/proto/crucible/v1/*.proto`. Phase 2 will regenerate them via `buf generate` — see `scripts/regen-proto.sh`. The hand-rolled vs buf-generated drift is asserted in `libs/sdk-go/crucible/v1/sync_test.go` (Phase 2).
