# Predicate JSON Schemas

Source-of-truth JSON Schemas for every `https://crucible.dev/*/v1` in-toto predicate type Crucible emits. The proto definitions in `../proto/crucible/v1/attestation.proto` are the wire schema; the files here are the *signed-content* schema that consumers (the verifier, the promotion gate, customer auditors) validate against when reading attestations off Sigstore Rekor (or the local journal fallback).

Versioning rule: a new predicate version bumps the URI path (`/v1` → `/v2`) and lands as a new file here, with a 90-day deprecation window for the old one. See `docs/03-sdk/attestation-formats.md` §"Schema source of truth".
