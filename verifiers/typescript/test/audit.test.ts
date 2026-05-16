// audit.test.ts — reasoning-leak audit refuses suspicious requests.

import { describe, expect, it } from "vitest";

import {
  LeakageError,
  REASONING_DENYLIST,
  auditDiffPaths,
  auditNoLeakage,
} from "../src/audit.js";

describe("auditNoLeakage", () => {
  it("passes a clean request", () => {
    expect(() =>
      auditNoLeakage({
        task_id: "t1",
        diff: { files: [{ path: "src/foo.ts", action: "modify" }] },
        routing: { executor_vendor: "anthropic", verifier_vendor: "google" },
      }),
    ).not.toThrow();
  });

  it("rejects top-level reasoning key", () => {
    expect(() => auditNoLeakage({ reasoning: "I considered..." })).toThrow(
      LeakageError,
    );
  });

  it.each([
    "chain_of_thought",
    "scratchpad",
    "agent_trace",
    "executor_trace",
    "thinking_trace",
    "cot",
    "reflection",
  ])("rejects %s anywhere in the tree", (key) => {
    const payload = { task_id: "t1", per_task_signals: { [key]: "secret" } };
    expect(() => auditNoLeakage(payload)).toThrow(LeakageError);
    try {
      auditNoLeakage(payload);
    } catch (err) {
      expect(err).toBeInstanceOf(LeakageError);
      const e = err as LeakageError;
      expect(e.offendingField).toBe(`per_task_signals.${key}`);
    }
  });

  it("rejects keys that *contain* a deny-token (substring match)", () => {
    // "my_reasoning_trace" contains "reasoning"
    expect(() =>
      auditNoLeakage({ task_id: "t1", my_reasoning_trace: "x" }),
    ).toThrow(LeakageError);
  });

  it("recurses into nested objects", () => {
    expect(() =>
      auditNoLeakage({
        outer: { middle: { reasoning_summary: "leak" } },
      }),
    ).toThrow(LeakageError);
  });

  it("recurses into arrays of objects", () => {
    expect(() =>
      auditNoLeakage({
        items: [{ id: 1 }, { id: 2, chain_of_thought: "..." }],
      }),
    ).toThrow(LeakageError);
  });

  it("ignores arrays of primitives", () => {
    expect(() =>
      auditNoLeakage({ tags: ["a", "b", "reasoning"] }),
    ).not.toThrow();
  });

  it("REASONING_DENYLIST covers the brief tokens", () => {
    for (const token of [
      "reasoning",
      "chain_of_thought",
      "scratchpad",
      "agent_trace",
      "executor_trace",
      "thinking_trace",
      "cot",
      "reflection",
    ]) {
      expect(REASONING_DENYLIST.includes(token)).toBe(true);
    }
  });
});

describe("auditDiffPaths", () => {
  it("passes clean paths", () => {
    expect(() =>
      auditDiffPaths(["src/foo.ts", "test/bar.test.ts", "openapi.yaml"]),
    ).not.toThrow();
  });

  it.each([
    "src/reasoning/index.ts",
    "agent_trace/log.txt",
    "executor_trace.json",
    "scratchpad/_scratchpad_.md", // matches "_scratchpad_"
  ])("refuses suspicious path %s", (p) => {
    expect(() => auditDiffPaths([p])).toThrow(LeakageError);
  });
});
