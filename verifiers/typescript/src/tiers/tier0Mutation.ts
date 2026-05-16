// tier0Mutation.ts — Stryker driver.
//
// Pins (May 2026): @stryker-mutator/core 9.6.1, @stryker-mutator/vitest-runner 9.6.1.
// Stryker emits mutation reports in the mutation-testing-elements v3.7.x JSON
// schema (https://github.com/stryker-mutator/mutation-testing-elements). We
// shell out to stryker because the public API surface for embedding a
// stryker run is not stable across 9.x. The CLI is invoked with the
// generated `stryker.conf.json` we wrote into the workspace.
//
// Crucible mandate: mutation MUST be diff-scoped. We write the `mutate`
// glob list from the diff's TS/JS files only. The threshold (85% per
// docs/01-architecture/verifier-pipeline.md) is enforced AFTER stryker
// emits its report so we never accept a stryker pass that came from
// running on the whole repo.

import { spawn } from "node:child_process";
import { mkdir, readFile, writeFile } from "node:fs/promises";
import path from "node:path";

import { auditDiffPaths } from "../audit.js";
import { classifyDiff } from "../diff.js";
import {
  finalizeTestReport,
  newTestReport,
  type Finding,
  type MutationStats,
  type SurvivedMutant,
  type TestReport,
  type VerificationRequest,
} from "../schema.js";

const MUTATION_THRESHOLD = 0.85; // matches verifier-pipeline.md
const REPORT_PATH = "reports/mutation/mutation.json";

export interface Tier0Options {
  /** Working directory where stryker should run. Defaults to cwd. */
  cwd?: string;
  /** Stryker binary name on PATH. Defaults to "stryker". */
  strykerBin?: string;
  /** Override config-file write path. Defaults to `${cwd}/stryker.conf.json`. */
  configPath?: string;
  /** Wall-clock budget seconds (Crucible default: 30s, max 2 min). */
  wallClockSeconds?: number;
  /**
   * If true, skip actually invoking stryker and emit `tool_unavailable`.
   * The CLI sets this when the binary is missing or in dry-run mode.
   */
  dryRun?: boolean;
  /**
   * For tests: pre-supplied JSON content that bypasses the spawn. When
   * set, we skip the subprocess and parse this string instead.
   */
  preSuppliedReport?: string;
}

// ---- mutation-testing-elements v3.7.x schema shapes (subset) ------------

interface MTEMutant {
  id: string;
  mutatorName: string;
  status: string; // "Killed" | "Survived" | "NoCoverage" | "Timeout" | "CompileError" | "RuntimeError"
  location: {
    start: { line: number; column: number };
    end: { line: number; column: number };
  };
  replacement?: string;
  original?: string;
}

interface MTEFileResult {
  language: string;
  mutants: MTEMutant[];
  source?: string;
}

interface MTEReport {
  schemaVersion: string;
  files: Record<string, MTEFileResult>;
  thresholds?: { high: number; low: number };
  projectRoot?: string;
}

// ---- Stats computation --------------------------------------------------

function classifyMutants(report: MTEReport): {
  killed: number;
  survived: number;
  notCovered: number;
  timeout: number;
  total: number;
  survivors: SurvivedMutant[];
} {
  let killed = 0;
  let survived = 0;
  let notCovered = 0;
  let timeout = 0;
  let total = 0;
  const survivors: SurvivedMutant[] = [];

  for (const [file, fileResult] of Object.entries(report.files)) {
    for (const m of fileResult.mutants) {
      total++;
      switch (m.status) {
        case "Killed":
          killed++;
          break;
        case "Survived":
          survived++;
          survivors.push({
            file,
            line: m.location.start.line,
            mutator: m.mutatorName,
            ...(m.original !== undefined ? { original: m.original } : {}),
            ...(m.replacement !== undefined
              ? { replacement: m.replacement }
              : {}),
          });
          break;
        case "NoCoverage":
          notCovered++;
          break;
        case "Timeout":
          timeout++;
          break;
        // CompileError / RuntimeError are not counted as killed or
        // survived per mutation-testing-elements semantics; ignore.
        default:
          break;
      }
    }
  }

  return { killed, survived, notCovered, timeout, total, survivors };
}

// ---- Config generator ---------------------------------------------------

interface StrykerConfig {
  $schema?: string;
  packageManager?: string;
  reporters: string[];
  mutate: string[];
  testRunner: string;
  coverageAnalysis: string;
  incremental: boolean;
  incrementalFile?: string;
  thresholds?: { high: number; low: number; break: number };
  jsonReporter?: { fileName: string };
  htmlReporter?: { fileName: string };
}

function makeStrykerConfig(mutateFiles: readonly string[]): StrykerConfig {
  return {
    $schema:
      "./node_modules/@stryker-mutator/core/schema/stryker-schema.json",
    packageManager: "pnpm",
    reporters: ["json", "clear-text"],
    mutate: mutateFiles.length > 0 ? Array.from(mutateFiles) : ["src/**/*.ts"],
    testRunner: "vitest",
    coverageAnalysis: "perTest",
    incremental: true,
    incrementalFile: ".stryker-tmp/incremental.json",
    thresholds: {
      high: 90,
      low: Math.floor(MUTATION_THRESHOLD * 100),
      break: Math.floor(MUTATION_THRESHOLD * 100),
    },
    jsonReporter: { fileName: REPORT_PATH },
  };
}

// ---- Subprocess driver --------------------------------------------------

function runStryker(
  bin: string,
  cwd: string,
  configPath: string,
  wallClockSeconds: number,
): Promise<{ exitCode: number; stderr: string }> {
  return new Promise((resolve, reject) => {
    const child = spawn(bin, ["run", configPath], {
      cwd,
      stdio: ["ignore", "pipe", "pipe"],
      env: { ...process.env, FORCE_COLOR: "0" },
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
      // Stryker logs to stdout by default — forward to stderr so it
      // doesn't pollute the TestReport channel.
      process.stderr.write(b);
    });
    child.on("error", (err) => {
      clearTimeout(timer);
      reject(err);
    });
    child.on("exit", (code) => {
      clearTimeout(timer);
      if (killed) {
        resolve({ exitCode: 124, stderr: stderr + "\n[crucible: killed by wall-clock budget]" });
        return;
      }
      resolve({ exitCode: code ?? 1, stderr });
    });
  });
}

// ---- Public entrypoint --------------------------------------------------

export async function runTier0Mutation(
  req: VerificationRequest,
  opts: Tier0Options = {},
): Promise<TestReport> {
  const startedAt = new Date();
  const cls = classifyDiff(req.diff);
  // Path-pattern guard on the diff — fast-fail if the agent smuggled a
  // reasoning blob into the file path.
  auditDiffPaths(cls.source.map((f) => f.path));

  const wallClockSeconds =
    opts.wallClockSeconds ??
    Math.max(30, Math.min(120, req.budget.wall_clock_cap_seconds || 30));

  const baseReport = newTestReport({
    task_id: req.task_id,
    diff_hash: req.diff.base_sha ?? req.base_sha,
    tier: "tier_0_mutation",
    framework: "stryker-9.6.1",
    reporter_id: "crucible-verify-typescript",
    reporter_version: "2026.06.0-phase4",
    wall_clock_budget_seconds: wallClockSeconds,
    started_at: startedAt,
  });

  if (cls.production.length === 0) {
    return finalizeTestReport(
      {
        ...baseReport,
        verdict: "skipped",
        passed: true,
        error: "no TS/JS production files in diff — Tier 0 skipped",
      },
      new Date(),
    );
  }

  const cwd = opts.cwd ?? process.cwd();
  const configPath = opts.configPath ?? path.join(cwd, "stryker.conf.json");
  const config = makeStrykerConfig(cls.production.map((f) => f.path));

  let strykerStderr = "";
  let preSupplied = opts.preSuppliedReport;

  if (preSupplied === undefined) {
    if (opts.dryRun === true) {
      return finalizeTestReport(
        {
          ...baseReport,
          verdict: "tool_unavailable",
          passed: false,
          framework: "stryker-9.6.1",
          error: "stryker not invoked (dry-run)",
        },
        new Date(),
      );
    }

    try {
      await writeFile(configPath, JSON.stringify(config, null, 2), "utf8");
    } catch (err) {
      return finalizeTestReport(
        {
          ...baseReport,
          verdict: "tool_unavailable",
          passed: false,
          error: `cannot write stryker.conf.json: ${(err as Error).message}`,
        },
        new Date(),
      );
    }

    try {
      const result = await runStryker(
        opts.strykerBin ?? "stryker",
        cwd,
        configPath,
        wallClockSeconds,
      );
      strykerStderr = result.stderr;
      // Stryker exits non-zero when the break threshold is missed — we
      // still parse the report (the threshold check is our own).
    } catch (err) {
      return finalizeTestReport(
        {
          ...baseReport,
          verdict: "tool_unavailable",
          passed: false,
          error: `stryker spawn failed: ${(err as Error).message}`,
        },
        new Date(),
      );
    }
  }

  // Parse the report.
  let raw: string;
  if (preSupplied !== undefined) {
    raw = preSupplied;
  } else {
    const reportFile = path.join(cwd, REPORT_PATH);
    try {
      raw = await readFile(reportFile, "utf8");
    } catch (err) {
      return finalizeTestReport(
        {
          ...baseReport,
          verdict: "failed",
          passed: false,
          error: `cannot read mutation report at ${REPORT_PATH}: ${(err as Error).message}\nstryker stderr: ${strykerStderr.slice(-1000)}`,
        },
        new Date(),
      );
    }
  }

  let parsed: MTEReport;
  try {
    parsed = JSON.parse(raw) as MTEReport;
  } catch (err) {
    return finalizeTestReport(
      {
        ...baseReport,
        verdict: "failed",
        passed: false,
        error: `mutation.json not valid JSON: ${(err as Error).message}`,
      },
      new Date(),
    );
  }

  const stats = classifyMutants(parsed);
  const denom = stats.killed + stats.survived;
  const score = denom > 0 ? stats.killed / denom : 0;
  const passed = score >= MUTATION_THRESHOLD && stats.total > 0;

  const mutation: MutationStats = {
    killed: stats.killed,
    survived: stats.survived,
    total: stats.total,
    score,
    threshold: MUTATION_THRESHOLD,
    diff_scoped: true,
    mutated_files: Object.keys(parsed.files),
    ...(stats.notCovered > 0 ? { not_covered: stats.notCovered } : {}),
    ...(stats.timeout > 0 ? { timeout: stats.timeout } : {}),
    ...(stats.survivors.length > 0
      ? { survived_summary: stats.survivors }
      : {}),
  };

  const findings: Finding[] = stats.survivors.slice(0, 50).map((s) => ({
    category: "mutation_survived",
    severity: "error" as const,
    file: s.file,
    line: s.line,
    detail: `${s.mutator} mutant survived${s.replacement ? ` (replacement: ${s.replacement})` : ""}`,
  }));

  return finalizeTestReport(
    {
      ...baseReport,
      mutation,
      findings,
      verdict: passed ? "passed" : "failed",
      passed,
    },
    new Date(),
  );
}

// ---- Test-only exports --------------------------------------------------

export const __testing__ = {
  classifyMutants,
  makeStrykerConfig,
  MUTATION_THRESHOLD,
};
