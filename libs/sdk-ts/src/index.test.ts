import { test } from "node:test";
import assert from "node:assert/strict";
import { CrucibleClient, Predicates, SDK_VERSION } from "./index.js";

test("SDK_VERSION is set to the Phase-1 release identifier", () => {
  assert.equal(SDK_VERSION, "2026.6.0-phase1");
});

test("Predicates includes all 14 predicate-type URIs", () => {
  const uris = Object.values(Predicates);
  assert.equal(uris.length, 14, "expected 14 predicate types");
  for (const uri of uris) {
    assert.match(uri, /^https:\/\/crucible\.dev\/[A-Z][A-Za-z]+\/v1$/);
  }
});

test("CrucibleClient stub surfaces a STUB error on use", () => {
  const c = new CrucibleClient({ endpoint: "http://localhost:8080" });
  assert.equal(c.endpoint, "http://localhost:8080");
  assert.throws(
    () => c.notWired("fs.read"),
    /STUB: CrucibleClient\.fs\.read/,
  );
});
