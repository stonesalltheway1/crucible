// schemaRoundtrip.test.ts — TestReport shape matches the Go canonical exactly.

import { describe, expect, it } from "vitest";

import {
  SCHEMA_VERSION,
  TIER_VALUES,
  finalizeTestReport,
  newTestReport,
  type TestReport,
} from "../src/schema.js";

describe("TestReport schema", () => {
  it("pins schema_version to '1'", () => {
    expect(SCHEMA_VERSION).toBe("1");
  });

  it("emits all snake_case keys the Go side expects", () => {
    const startedAt = new Date("2026-05-15T00:00:00Z");
    const report = finalizeTestReport(
      newTestReport({
        task_id: "task-42",
        diff_hash: "abc123",
        tier: "tier_0_mutation",
        framework: "stryker-9.6.1",
        reporter_id: "crucible-verify-typescript",
        reporter_version: "2026.06.0-phase4",
        wall_clock_budget_seconds: 30,
        started_at: startedAt,
      }),
      new Date("2026-05-15T00:00:05Z"),
    );
    const json = JSON.parse(JSON.stringify(report)) as Record<string, unknown>;

    // The canonical Go-required keys.
    expect(Object.keys(json).sort()).toEqual(
      expect.arrayContaining([
        "schema_version",
        "task_id",
        "diff_hash",
        "tier",
        "language",
        "framework",
        "verdict",
        "passed",
        "started_at",
        "finished_at",
        "duration_seconds",
        "wall_clock_budget_seconds",
        "reporter_id",
        "reporter_version",
      ]),
    );

    expect(json["schema_version"]).toBe("1");
    expect(json["task_id"]).toBe("task-42");
    expect(json["tier"]).toBe("tier_0_mutation");
    expect(json["language"]).toBe("typescript");
    expect(json["duration_seconds"]).toBe(5);
    expect(typeof json["started_at"]).toBe("string");
    expect((json["started_at"] as string).endsWith("Z")).toBe(true);
  });

  it("never emits camelCase variants of the wire keys", () => {
    const report = newTestReport({
      task_id: "t",
      diff_hash: "h",
      tier: "tier_1_pbt",
      framework: "fast-check",
      reporter_id: "crucible-verify-typescript",
      wall_clock_budget_seconds: 0,
      started_at: new Date(),
    });
    const j = JSON.stringify(report);
    for (const camel of [
      "schemaVersion",
      "taskId",
      "diffHash",
      "wallClockBudgetSeconds",
      "startedAt",
      "finishedAt",
      "reporterId",
    ]) {
      expect(j).not.toContain(`"${camel}"`);
    }
  });

  it("TIER_VALUES enumerates exactly the five Go-canonical tiers", () => {
    expect([...TIER_VALUES]).toEqual([
      "tier_0_mutation",
      "tier_1_pbt",
      "tier_2_contract",
      "tier_3_proof",
      "tier_4_honest_ci",
    ]);
  });

  it("includes tier-specific stats only when populated", () => {
    const base = finalizeTestReport(
      newTestReport({
        task_id: "t",
        diff_hash: "h",
        tier: "tier_0_mutation",
        framework: "stryker",
        reporter_id: "crucible-verify-typescript",
        wall_clock_budget_seconds: 30,
        started_at: new Date(),
      }),
      new Date(),
    );
    const withStats: TestReport = {
      ...base,
      mutation: {
        killed: 9,
        survived: 1,
        total: 10,
        score: 0.9,
        threshold: 0.85,
        diff_scoped: true,
      },
      verdict: "passed",
      passed: true,
    };
    const json = JSON.parse(JSON.stringify(withStats)) as Record<
      string,
      unknown
    >;
    expect(json["mutation"]).toBeDefined();
    expect(json["pbt"]).toBeUndefined();
    expect(json["contract"]).toBeUndefined();
    expect(json["proof"]).toBeUndefined();
    expect(json["honest_ci"]).toBeUndefined();
  });
});
