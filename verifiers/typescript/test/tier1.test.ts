// tier1.test.ts — fast-check counterexample parsing & report shape.

import { describe, expect, it } from "vitest";

import {
  IT_PROP_MIN_RUNS,
  runTier1Pbt,
  __testing__,
} from "../src/tiers/tier1Pbt.js";
import type { VerificationRequest } from "../src/schema.js";

function baseReq(): VerificationRequest {
  return {
    task_id: "t-tier1",
    tenant_id: "tenant-1",
    repo: "crucible/verifier-typescript",
    base_sha: "abc123",
    diff: {
      base_sha: "abc123",
      files: [
        { path: "src/calc.ts", action: "modify" },
        { path: "src/calc.test.ts", action: "modify" },
      ],
    },
    routing: {
      executor_model: "claude-opus-4-7",
      executor_vendor: "anthropic",
      verifier_model: "gemini-3.1-pro",
      verifier_vendor: "google",
    },
    languages: ["typescript"],
    budget: {
      verifier_cap_usd: 1,
      verifier_spent_usd: 0,
      wall_clock_cap_seconds: 300,
      wall_clock_spent_seconds: 0,
    },
    executor_sandbox_id: "sb-executor-1",
  };
}

const FC_FAILURE_MSG = `Error: Property failed after 17 tests
{ seed: 42, path: "9:1:0", endOnFailure: true }
Counterexample: [-1]
Shrunk 4 time(s)
Got error: Expected non-empty array but got empty
seed: 42`;

const VITEST_REPORT_WITH_COUNTEREXAMPLE = JSON.stringify({
  numTotalTestSuites: 1,
  numPassedTestSuites: 0,
  numFailedTestSuites: 1,
  numTotalTests: 1,
  numPassedTests: 0,
  numFailedTests: 1,
  testResults: [
    {
      status: "failed",
      assertionResults: [
        {
          status: "failed",
          fullName: "calc > sum is non-negative for positive inputs",
          failureMessages: [FC_FAILURE_MSG],
        },
      ],
    },
  ],
});

const VITEST_REPORT_PASSING = JSON.stringify({
  numTotalTestSuites: 1,
  numPassedTestSuites: 1,
  numFailedTestSuites: 0,
  numTotalTests: 2,
  numPassedTests: 2,
  numFailedTests: 0,
  testResults: [
    {
      status: "passed",
      assertionResults: [
        {
          status: "passed",
          fullName: "calc > sum is commutative",
        },
        {
          status: "passed",
          fullName: "calc > sum is associative",
        },
      ],
    },
  ],
});

describe("tier 1 — fast-check parsing", () => {
  it("parses counterexample from fast-check failure message", () => {
    const ce = __testing__.parseCounterexample("p1", FC_FAILURE_MSG);
    expect(ce).not.toBeNull();
    expect(ce?.property).toBe("p1");
    expect(ce?.shrunk).toContain("[-1]");
    expect(ce?.seed).toBe("42");
    expect(ce?.stack_hint).toContain("Expected non-empty array");
  });

  it("returns null on a non-fast-check failure message", () => {
    expect(__testing__.parseCounterexample("p1", "TypeError: nope")).toBeNull();
  });
});

describe("tier 1 — runner", () => {
  it("surfaces counterexamples from the vitest JSON report", async () => {
    const report = await runTier1Pbt(baseReq(), {
      preSuppliedReport: VITEST_REPORT_WITH_COUNTEREXAMPLE,
      testFilesOverride: ["src/calc.test.ts"],
    });
    expect(report.tier).toBe("tier_1_pbt");
    expect(report.verdict).toBe("failed");
    expect(report.passed).toBe(false);
    expect(report.pbt?.counterexamples).toBeDefined();
    expect(report.pbt?.counterexamples?.length).toBe(1);
    expect(report.pbt?.counterexamples?.[0]?.shrunk).toContain("[-1]");
    expect(report.pbt?.iterations_min).toBe(IT_PROP_MIN_RUNS);
    expect(report.findings?.length).toBeGreaterThan(0);
    expect(report.findings?.[0]?.category).toBe("property_failed");
  });

  it("passes when all properties pass at min runs", async () => {
    const report = await runTier1Pbt(baseReq(), {
      preSuppliedReport: VITEST_REPORT_PASSING,
      testFilesOverride: ["src/calc.test.ts"],
      numRuns: IT_PROP_MIN_RUNS,
    });
    expect(report.verdict).toBe("passed");
    expect(report.passed).toBe(true);
    expect(report.pbt?.iterations).toBe(2 * IT_PROP_MIN_RUNS);
    expect(report.pbt?.counterexamples).toBeUndefined();
  });

  it("fails when diff has no PBT files", async () => {
    const req = baseReq();
    req.diff.files = [{ path: "src/calc.ts", action: "modify" }];
    const report = await runTier1Pbt(req, { testFilesOverride: [] });
    expect(report.verdict).toBe("failed");
    expect(report.findings?.[0]?.category).toBe("pbt_missing");
  });

  it("emits snake_case wire keys", async () => {
    const report = await runTier1Pbt(baseReq(), {
      preSuppliedReport: VITEST_REPORT_PASSING,
      testFilesOverride: ["src/calc.test.ts"],
    });
    const j = JSON.stringify(report);
    expect(j).toContain('"iterations_min":10000');
    expect(j).not.toContain('"iterationsMin"');
    expect(j).toContain('"schema_version":"1"');
  });
});
