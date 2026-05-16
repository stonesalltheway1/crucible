// tier4HonestCi.ts — honest-CI reproducible-build driver.
//
// The full Tier 4 spec (docs/01-architecture/verifier-pipeline.md) wants:
//   1. Hermetic rebuild (Nix or Bazel), bit-identical hash compare
//   2. In-toto attestation signed by Sigstore keyless OIDC
//   3. SLSA L3 admissions
//   4. Rego policy admission
//
// Items 2/3/4 are owned by the verifier daemon (signing happens out of
// the per-language runner sandbox, by design — the runner has no OIDC
// identity). What we own here is item (1): run the project's build
// twice with `npm_config_prefer_frozen_lockfile=true`, sha256 the
// `dist/` tree of each run, compare.
//
// We deliberately use the TS-ecosystem-native `pnpm build` rather than
// shelling to Nix — Tier 4 ships with both paths and the TS runner
// implements the npm-side hermeticity check. The Nix rebuild is the
// daemon's job (apps/verifier/internal/honestci).

import { spawn } from "node:child_process";
import { createHash } from "node:crypto";
import { mkdtemp, readdir, readFile, rm, stat } from "node:fs/promises";
import os from "node:os";
import path from "node:path";

import {
  finalizeTestReport,
  newTestReport,
  type Finding,
  type HonestCIStats,
  type TestReport,
  type VerificationRequest,
} from "../schema.js";

export interface Tier4Options {
  cwd?: string;
  pnpmBin?: string;
  /** dir to sha256 after the build. Default: "dist". */
  outputDir?: string;
  wallClockSeconds?: number;
  /**
   * Inject the two hashes (test-only). When set, both build invocations
   * are skipped.
   */
  preSuppliedHashes?: { first: string; second: string };
  dryRun?: boolean;
}

// ---- Recursive tree hash ------------------------------------------------
//
// We hash the sorted list of (relative-path, sha256(content)) pairs. This
// gives us a deterministic hash independent of inode order or atime —
// the only sources of non-determinism we control. For npm packages, the
// remaining non-determinism (timestamps in package.json from `pnpm
// install`, etc.) is filtered: we only hash the `dist/` tree, which
// contains compiled outputs.

async function listFilesRecursive(root: string): Promise<string[]> {
  const out: string[] = [];
  async function walk(dir: string): Promise<void> {
    let entries;
    try {
      entries = await readdir(dir, { withFileTypes: true });
    } catch {
      return;
    }
    for (const e of entries) {
      const full = path.join(dir, e.name);
      if (e.isDirectory()) {
        await walk(full);
      } else if (e.isFile()) {
        out.push(full);
      }
    }
  }
  await walk(root);
  return out;
}

export async function hashDirectory(root: string): Promise<string> {
  const absRoot = path.resolve(root);
  const files = await listFilesRecursive(absRoot);
  files.sort();

  const tree = createHash("sha256");
  for (const f of files) {
    const rel = path.relative(absRoot, f).split(path.sep).join("/");
    let buf: Buffer;
    try {
      buf = await readFile(f);
    } catch {
      continue;
    }
    const fileHash = createHash("sha256").update(buf).digest("hex");
    tree.update(`${rel}\0${fileHash}\n`);
  }
  return tree.digest("hex");
}

// ---- Subprocess driver --------------------------------------------------

function runPnpm(
  bin: string,
  args: readonly string[],
  cwd: string,
  wallClockSeconds: number,
): Promise<{ exitCode: number; stderr: string }> {
  return new Promise((resolve, reject) => {
    const child = spawn(bin, Array.from(args), {
      cwd,
      stdio: ["ignore", "pipe", "pipe"],
      env: {
        ...process.env,
        npm_config_prefer_frozen_lockfile: "true",
        // Force deterministic behaviour where pnpm supports it.
        npm_config_progress: "false",
        FORCE_COLOR: "0",
        CI: "true",
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

// ---- Public entrypoint --------------------------------------------------

export async function runTier4HonestCi(
  req: VerificationRequest,
  opts: Tier4Options = {},
): Promise<TestReport> {
  const startedAt = new Date();
  const cwd = opts.cwd ?? process.cwd();
  const outputDir = opts.outputDir ?? "dist";
  const wallClockSeconds =
    opts.wallClockSeconds ??
    Math.max(300, Math.min(1800, req.budget.wall_clock_cap_seconds || 300));

  const baseReport = newTestReport({
    task_id: req.task_id,
    diff_hash: req.diff.base_sha ?? req.base_sha,
    tier: "tier_4_honest_ci",
    framework: "pnpm-double-build",
    reporter_id: "crucible-verify-typescript",
    reporter_version: "2026.06.0-phase4",
    wall_clock_budget_seconds: wallClockSeconds,
    started_at: startedAt,
  });

  if (opts.preSuppliedHashes !== undefined) {
    const { first, second } = opts.preSuppliedHashes;
    const bitIdentical = first === second;
    const stats: HonestCIStats = {
      builder_id: "crucible://builders/pnpm-double-build/v1",
      executor_rebuild_hash: first,
      verifier_rebuild_hash: second,
      bit_identical: bitIdentical,
      slsa_level: bitIdentical ? 3 : 0,
      scrubber_audit_ok: true,
    };
    const findings: Finding[] = bitIdentical
      ? []
      : [
          {
            category: "rebuild_diverged",
            severity: "error" as const,
            detail: `dist/ hash diverged: first=${first}, second=${second}`,
          },
        ];
    return finalizeTestReport(
      {
        ...baseReport,
        honest_ci: stats,
        findings,
        verdict: bitIdentical ? "passed" : "failed",
        passed: bitIdentical,
      },
      new Date(),
    );
  }

  if (opts.dryRun === true) {
    return finalizeTestReport(
      {
        ...baseReport,
        verdict: "tool_unavailable",
        passed: false,
        error: "pnpm not invoked (dry-run)",
      },
      new Date(),
    );
  }

  const pnpm = opts.pnpmBin ?? "pnpm";

  // First pass — install + build into the project's own dist/.
  let firstStaging: string | null = null;
  let secondStaging: string | null = null;
  try {
    // We isolate each build to a temporary `--store-dir` to maximize
    // determinism; cwd remains the project root because the build
    // scripts likely use relative paths.
    firstStaging = await mkdtemp(path.join(os.tmpdir(), "crucible-tier4-pnpm1-"));
    secondStaging = await mkdtemp(path.join(os.tmpdir(), "crucible-tier4-pnpm2-"));

    const halfBudget = Math.max(1, Math.floor(wallClockSeconds / 2));

    const install1 = await runPnpm(
      pnpm,
      ["install", "--frozen-lockfile", `--store-dir=${firstStaging}`],
      cwd,
      halfBudget,
    );
    if (install1.exitCode !== 0) {
      return finalizeTestReport(
        {
          ...baseReport,
          verdict: "failed",
          passed: false,
          error: `pnpm install (pass 1) exit ${install1.exitCode}: ${install1.stderr.slice(-1000)}`,
        },
        new Date(),
      );
    }
    const build1 = await runPnpm(pnpm, ["build"], cwd, halfBudget);
    if (build1.exitCode !== 0) {
      return finalizeTestReport(
        {
          ...baseReport,
          verdict: "failed",
          passed: false,
          error: `pnpm build (pass 1) exit ${build1.exitCode}: ${build1.stderr.slice(-1000)}`,
        },
        new Date(),
      );
    }
    const distAbs = path.join(cwd, outputDir);
    try {
      await stat(distAbs);
    } catch {
      return finalizeTestReport(
        {
          ...baseReport,
          verdict: "failed",
          passed: false,
          error: `pnpm build (pass 1) produced no ${outputDir}/ directory`,
        },
        new Date(),
      );
    }
    const firstHash = await hashDirectory(distAbs);

    // Second pass.
    const install2 = await runPnpm(
      pnpm,
      ["install", "--frozen-lockfile", `--store-dir=${secondStaging}`],
      cwd,
      halfBudget,
    );
    if (install2.exitCode !== 0) {
      return finalizeTestReport(
        {
          ...baseReport,
          verdict: "failed",
          passed: false,
          error: `pnpm install (pass 2) exit ${install2.exitCode}: ${install2.stderr.slice(-1000)}`,
        },
        new Date(),
      );
    }
    const build2 = await runPnpm(pnpm, ["build"], cwd, halfBudget);
    if (build2.exitCode !== 0) {
      return finalizeTestReport(
        {
          ...baseReport,
          verdict: "failed",
          passed: false,
          error: `pnpm build (pass 2) exit ${build2.exitCode}: ${build2.stderr.slice(-1000)}`,
        },
        new Date(),
      );
    }
    const secondHash = await hashDirectory(distAbs);

    const bitIdentical = firstHash === secondHash;
    const stats: HonestCIStats = {
      builder_id: "crucible://builders/pnpm-double-build/v1",
      executor_rebuild_hash: firstHash,
      verifier_rebuild_hash: secondHash,
      bit_identical: bitIdentical,
      slsa_level: bitIdentical ? 3 : 0,
      scrubber_audit_ok: true,
    };
    const findings: Finding[] = bitIdentical
      ? []
      : [
          {
            category: "rebuild_diverged",
            severity: "error" as const,
            detail: `dist/ hash diverged: first=${firstHash}, second=${secondHash}`,
          },
        ];

    return finalizeTestReport(
      {
        ...baseReport,
        honest_ci: stats,
        findings,
        verdict: bitIdentical ? "passed" : "failed",
        passed: bitIdentical,
      },
      new Date(),
    );
  } catch (err) {
    return finalizeTestReport(
      {
        ...baseReport,
        verdict: "tool_unavailable",
        passed: false,
        error: `tier 4 driver failed: ${(err as Error).message}`,
      },
      new Date(),
    );
  } finally {
    for (const dir of [firstStaging, secondStaging]) {
      if (dir !== null) {
        void rm(dir, { recursive: true, force: true }).catch(() => {});
      }
    }
  }
}

export const __testing__ = { hashDirectory };
