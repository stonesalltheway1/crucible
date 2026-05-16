// tier1Pbt.ts — fast-check / @fast-check/vitest driver.
//
// Pins (May 2026): fast-check 4.7.0, @fast-check/vitest 0.4.1.
//
// Tier 1 spec (docs/01-architecture/verifier-pipeline.md):
// - "Verifier requires authored property tests covering the changed
//   function's invariants. Runs them at >=10,000 iterations."
// - Wall-clock budget: 5 min default, 15 min max.
//
// How we run:
// 1. Discover *.test.ts / *.spec.ts files in the diff that import
//    fast-check.
// 2. Spawn `vitest run <files>` with VITEST_FAST_CHECK_NUM_RUNS=10000
//    in the environment. Tests written with @fast-check/vitest can
//    also pin numRuns inline via `it.prop([gen], { numRuns: 10000 })`
//    — we set the env so existing tests at the default numRuns are
//    automatically scaled up to the Crucible minimum.
// 3. Parse vitest's JSON reporter output to detect counterexamples
//    (fast-check prints "Counterexample:" in the failure message).
//
// If the diff has no property tests at all, we emit `failed` with a
// finding — Tier 1 is mandatory when the dispatcher routes us here, so
// "no PBTs" is a real failure, not a tool_unavailable.

import { spawn } from "node:child_process";
import { mkdtemp, readFile, rm, writeFile } from "node:fs/promises";
import path from "node:path";
import os from "node:os";

import { classifyDiff } from "../diff.js";
import {
  finalizeTestReport,
  newTestReport,
  type Counterexample,
  type Finding,
  type PBTStats,
  type TestReport,
  type VerificationRequest,
} from "../schema.js";

export const IT_PROP_MIN_RUNS = 10_000;

export interface Tier1Options {
  cwd?: string;
  vitestBin?: string;
  numRuns?: number;
  wallClockSeconds?: number;
  /**
   * Pre-supplied vitest JSON report content. Used by tests to bypass
   * the subprocess spawn. When set, `testFiles` is ignored.
   */
  preSuppliedReport?: string;
  /** Override the file-discovery step (test-only). */
  testFilesOverride?: readonly string[];
}

// ---- vitest JSON reporter shape (v2.x subset) ----------------------------

interface VitestAssertionResult {
  status: "passed" | "failed" | "skipped" | "pending" | "todo";
  fullName: string;
  failureMessages?: string[];
}

interface VitestTestResult {
  assertionResults: VitestAssertionResult[];
  status: "passed" | "failed";
}

interface VitestReport {
  numTotalTestSuites: number;
  numPassedTestSuites: number;
  numFailedTestSuites: number;
  numTotalTests: number;
  numPassedTests: number;
  numFailedTests: number;
  testResults: VitestTestResult[];
}

// ---- fast-check counterexample parsing -----------------------------------

const COUNTEREXAMPLE_RE =
  /Counterexample:\s*(?<shrunk>[\s\S]*?)(?:\nShrunk\s+(?<shrinks>\d+)\s+time\(s\))?\nGot(?:\s+error)?:\s*(?<got>[\s\S]*?)(?:\nseed:\s*(?<seed>-?\d+))?(?:\n|$)/i;

export function parseCounterexample(
  property: string,
  failureMessage: string,
): Counterexample | null {
  const m = COUNTEREXAMPLE_RE.exec(failureMessage);
  if (!m || !m.groups) {
    return null;
  }
  const shrunk = (m.groups["shrunk"] ?? "").trim();
  const got = (m.groups["got"] ?? "").trim();
  const seed = m.groups["seed"];
  const ce: Counterexample = {
    property,
    shrunk,
    stack_hint: got.slice(0, 500),
  };
  if (seed !== undefined && seed !== "") {
    ce.seed = seed;
  }
  return ce;
}

// ---- File discovery ------------------------------------------------------

async function findPbtFiles(
  cwd: string,
  candidates: readonly string[],
): Promise<string[]> {
  const out: string[] = [];
  for (const rel of candidates) {
    const abs = path.isAbsolute(rel) ? rel : path.join(cwd, rel);
    try {
      const src = await readFile(abs, "utf8");
      if (/from\s+["']fast-check["']/.test(src) ||
          /from\s+["']@fast-check\/vitest["']/.test(src) ||
          /require\(["']fast-check["']\)/.test(src)) {
        out.push(rel);
      }
    } catch {
      // missing on disk — skip
    }
  }
  return out;
}

// ---- Subprocess driver ---------------------------------------------------

function runVitest(
  bin: string,
  cwd: string,
  files: readonly string[],
  reportPath: string,
  numRuns: number,
  wallClockSeconds: number,
): Promise<{ exitCode: number; stderr: string }> {
  return new Promise((resolve, reject) => {
    const args = [
      "run",
      "--reporter=json",
      `--outputFile=${reportPath}`,
      ...files,
    ];
    const child = spawn(bin, args, {
      cwd,
      stdio: ["ignore", "pipe", "pipe"],
      env: {
        ...process.env,
        // @fast-check/vitest honours this env var for its `it.prop` runs.
        VITEST_FAST_CHECK_NUM_RUNS: String(numRuns),
        // fast-check's vanilla `fc.assert` also honours these defaults
        // when no inline numRuns is provided.
        FAST_CHECK_NUM_RUNS: String(numRuns),
        FORCE_COLOR: "0",
      },
    });
    let stderr = "";
    let killed = false;
    const timer = setTimeout(
      () => {
        killed = true;
        child.kill("SIGKILL");
      },
      Math.max(1, wallClockSeconds) * 1000,
    );
    child.stderr?.on("data", (b: Buffer) => {
      stderr += b.toString("utf8");
      process.stderr.write(b);
    });
    child.stdout?.on("data", (b: Buffer) => {
      // vitest JSON reporter writes to outputFile; anything on stdout
      // is incidental — forward to stderr to keep our protocol clean.
      process.stderr.write(b);
    });
    child.on("error", (err) => {
      clearTimeout(timer);
      reject(err);
    });
    child.on("exit", (code) => {
      clearTimeout(timer);
      if (killed) {
        resolve({
          exitCode: 124,
          stderr: stderr + "\n[crucible: killed by wall-clock budget]",
        });
        return;
      }
      resolve({ exitCode: code ?? 1, stderr });
    });
  });
}

// ---- Public entrypoint ---------------------------------------------------

export async function runTier1Pbt(
  req: VerificationRequest,
  opts: Tier1Options = {},
): Promise<TestReport> {
  const startedAt = new Date();
  const cwd = opts.cwd ?? process.cwd();
  const numRuns = opts.numRuns ?? IT_PROP_MIN_RUNS;
  const wallClockSeconds =
    opts.wallClockSeconds ??
    Math.max(300, Math.min(900, req.budget.wall_clock_cap_seconds || 300));

  const baseReport = newTestReport({
    task_id: req.task_id,
    diff_hash: req.diff.base_sha ?? req.base_sha,
    tier: "tier_1_pbt",
    framework: "fast-check-4.7.0+@fast-check/vitest-0.4.1",
    reporter_id: "crucible-verify-typescript",
    reporter_version: "2026.06.0-phase4",
    wall_clock_budget_seconds: wallClockSeconds,
    started_at: startedAt,
  });

  // Find candidate test files.
  const cls = classifyDiff(req.diff);
  const candidates =
    opts.testFilesOverride !== undefined
      ? Array.from(opts.testFilesOverride)
      : cls.tests.map((f) => f.path);

  const pbtFiles =
    opts.preSuppliedReport !== undefined
      ? candidates
      : await findPbtFiles(cwd, candidates);

  if (pbtFiles.length === 0 && opts.preSuppliedReport === undefined) {
    const finding: Finding = {
      category: "pbt_missing",
      severity: "error",
      detail:
        "No property-based test files found in diff. Tier 1 requires at least one *.test.ts that imports fast-check.",
    };
    return finalizeTestReport(
      {
        ...baseReport,
        verdict: "failed",
        passed: false,
        pbt: {
          iterations: 0,
          iterations_min: IT_PROP_MIN_RUNS,
        },
        findings: [finding],
      },
      new Date(),
    );
  }

  // Run vitest.
  let raw: string;
  if (opts.preSuppliedReport !== undefined) {
    raw = opts.preSuppliedReport;
  } else {
    const tmp = await mkdtemp(path.join(os.tmpdir(), "crucible-tier1-"));
    const reportPath = path.join(tmp, "vitest.json");
    try {
      const result = await runVitest(
        opts.vitestBin ?? "vitest",
        cwd,
        pbtFiles,
        reportPath,
        numRuns,
        wallClockSeconds,
      );
      try {
        raw = await readFile(reportPath, "utf8");
      } catch {
        return finalizeTestReport(
          {
            ...baseReport,
            verdict: "failed",
            passed: false,
            error: `vitest report missing (exit ${result.exitCode}). stderr tail: ${result.stderr.slice(-1500)}`,
          },
          new Date(),
        );
      }
    } catch (err) {
      return finalizeTestReport(
        {
          ...baseReport,
          verdict: "tool_unavailable",
          passed: false,
          error: `vitest spawn failed: ${(err as Error).message}`,
        },
        new Date(),
      );
    } finally {
      // Best-effort cleanup.
      void rm(tmp, { recursive: true, force: true }).catch(() => {});
    }
  }

  let report: VitestReport;
  try {
    report = JSON.parse(raw) as VitestReport;
  } catch (err) {
    return finalizeTestReport(
      {
        ...baseReport,
        verdict: "failed",
        passed: false,
        error: `vitest JSON report unparsable: ${(err as Error).message}`,
      },
      new Date(),
    );
  }

  // Collect counterexamples + property names.
  const counterexamples: Counterexample[] = [];
  const propertyNames: string[] = [];
  for (const tr of report.testResults ?? []) {
    for (const a of tr.assertionResults ?? []) {
      propertyNames.push(a.fullName);
      if (a.status === "failed") {
        const msg = (a.failureMessages ?? []).join("\n");
        const ce = parseCounterexample(a.fullName, msg);
        if (ce !== null) {
          counterexamples.push(ce);
        } else {
          counterexamples.push({
            property: a.fullName,
            shrunk: msg.slice(0, 500),
            stack_hint: msg.slice(0, 1000),
          });
        }
      }
    }
  }

  // Each test invocation runs `numRuns` iterations under @fast-check/vitest.
  const iterations = (report.numTotalTests || 0) * numRuns;
  const passed =
    report.numFailedTests === 0 &&
    iterations >= IT_PROP_MIN_RUNS &&
    counterexamples.length === 0;

  const pbt: PBTStats = {
    iterations,
    iterations_min: IT_PROP_MIN_RUNS,
    properties: propertyNames,
    ...(counterexamples.length > 0 ? { counterexamples } : {}),
  };

  const findings: Finding[] = counterexamples.slice(0, 50).map((c) => ({
    category: "property_failed",
    severity: "error" as const,
    detail: `Property "${c.property}" found counterexample: ${c.shrunk}`,
  }));

  return finalizeTestReport(
    {
      ...baseReport,
      pbt,
      findings,
      verdict: passed ? "passed" : "failed",
      passed,
    },
    new Date(),
  );
}

export const __testing__ = { parseCounterexample, findPbtFiles };
