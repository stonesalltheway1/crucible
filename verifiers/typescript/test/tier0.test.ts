// tier0.test.ts — Stryker driver, using pre-supplied mutation reports.
//
// We don't spawn stryker here (CI has no toolchain). Instead we feed the
// driver a hand-crafted mutation-testing-elements v3.7.x report and
// assert it parses + classifies + thresholds correctly.

import { describe, expect, it } from "vitest";

import { runTier0Mutation, __testing__ } from "../src/tiers/tier0Mutation.js";
import type { VerificationRequest } from "../src/schema.js";

function baseReq(): VerificationRequest {
  return {
    task_id: "t-tier0",
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
      wall_clock_cap_seconds: 30,
      wall_clock_spent_seconds: 0,
    },
    executor_sandbox_id: "sb-executor-1",
  };
}

// Strong suite: 9 killed, 1 survived → score 0.9 → passes (>=0.85).
const STRONG_REPORT = JSON.stringify({
  schemaVersion: "3.7.0",
  files: {
    "src/calc.ts": {
      language: "typescript",
      mutants: [
        ...Array.from({ length: 9 }, (_, i) => ({
          id: String(i),
          mutatorName: "ArithmeticOperator",
          status: "Killed",
          location: {
            start: { line: 10 + i, column: 1 },
            end: { line: 10 + i, column: 10 },
          },
        })),
        {
          id: "9",
          mutatorName: "BooleanLiteral",
          status: "Survived",
          location: { start: { line: 20, column: 1 }, end: { line: 20, column: 5 } },
          original: "true",
          replacement: "false",
        },
      ],
    },
  },
});

// Weak suite: 3 killed, 7 survived → score 0.3 → fails.
const WEAK_REPORT = JSON.stringify({
  schemaVersion: "3.7.0",
  files: {
    "src/calc.ts": {
      language: "typescript",
      mutants: [
        ...Array.from({ length: 3 }, (_, i) => ({
          id: String(i),
          mutatorName: "ArithmeticOperator",
          status: "Killed",
          location: { start: { line: 1, column: 1 }, end: { line: 1, column: 2 } },
        })),
        ...Array.from({ length: 7 }, (_, i) => ({
          id: String(i + 10),
          mutatorName: "BooleanLiteral",
          status: "Survived",
          location: {
            start: { line: 10 + i, column: 1 },
            end: { line: 10 + i, column: 5 },
          },
          original: "true",
          replacement: "false",
        })),
      ],
    },
  },
});

describe("tier 0 — stryker driver", () => {
  it("passes when score >= 0.85 (strong suite)", async () => {
    const report = await runTier0Mutation(baseReq(), {
      preSuppliedReport: STRONG_REPORT,
    });
    expect(report.tier).toBe("tier_0_mutation");
    expect(report.verdict).toBe("passed");
    expect(report.passed).toBe(true);
    expect(report.mutation).toBeDefined();
    expect(report.mutation?.killed).toBe(9);
    expect(report.mutation?.survived).toBe(1);
    expect(report.mutation?.total).toBe(10);
    expect(report.mutation?.score).toBeCloseTo(0.9, 5);
    expect(report.mutation?.diff_scoped).toBe(true);
    expect(report.mutation?.threshold).toBe(0.85);
  });

  it("fails when score < 0.85 (weak suite)", async () => {
    const report = await runTier0Mutation(baseReq(), {
      preSuppliedReport: WEAK_REPORT,
    });
    expect(report.verdict).toBe("failed");
    expect(report.passed).toBe(false);
    expect(report.mutation?.score).toBeCloseTo(0.3, 5);
    expect(report.findings?.length).toBeGreaterThan(0);
    expect(report.findings?.[0]?.category).toBe("mutation_survived");
  });

  it("skips when diff has no TS files", async () => {
    const req = baseReq();
    req.diff.files = [{ path: "README.md", action: "modify" }];
    const report = await runTier0Mutation(req);
    expect(report.verdict).toBe("skipped");
    expect(report.passed).toBe(true);
  });

  it("classifyMutants counts statuses correctly", () => {
    const parsed = JSON.parse(STRONG_REPORT);
    const stats = __testing__.classifyMutants(parsed);
    expect(stats.killed).toBe(9);
    expect(stats.survived).toBe(1);
    expect(stats.total).toBe(10);
    expect(stats.survivors).toHaveLength(1);
    expect(stats.survivors[0]?.mutator).toBe("BooleanLiteral");
  });

  it("makeStrykerConfig requires diff-scoped mutate list", () => {
    const cfg = __testing__.makeStrykerConfig(["src/a.ts", "src/b.ts"]);
    expect(cfg.mutate).toEqual(["src/a.ts", "src/b.ts"]);
    expect(cfg.reporters).toEqual(["json", "clear-text"]);
    expect(cfg.incremental).toBe(true);
    expect(cfg.testRunner).toBe("vitest");
    expect(cfg.thresholds?.break).toBe(
      Math.floor(__testing__.MUTATION_THRESHOLD * 100),
    );
  });

  it("emits TestReport with wire-format keys (no camelCase)", async () => {
    const report = await runTier0Mutation(baseReq(), {
      preSuppliedReport: STRONG_REPORT,
    });
    const j = JSON.stringify(report);
    expect(j).toContain('"schema_version":"1"');
    expect(j).toContain('"tier":"tier_0_mutation"');
    expect(j).toContain('"diff_scoped":true');
    expect(j).not.toContain('"diffScoped"');
  });
});
