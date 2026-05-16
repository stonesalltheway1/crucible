#!/usr/bin/env node
// cli.ts — `crucible-verify-typescript` entrypoint.
//
// Protocol (matches apps/verifier/internal/processpool/pool.go):
//   stdin: VerificationRequest as JSON
//   args:  --tier=<tier_0_mutation|...>
//   stdout: "===CRUCIBLE-TESTREPORT===\n" + TestReport JSON
//   stderr: human-readable logs (NEVER mix with the stdout protocol)
//   exit:  0 on substantive result OR substantive failure (with report)
//          2 on executor-reasoning leak (ADR-002 invariant)
//          1 on procedural crash (no report)

import path from "node:path";
import { stdin } from "node:process";
import { fileURLToPath } from "node:url";

import { LeakageError, auditNoLeakage, auditDiffPaths } from "./audit.js";
import {
  finalizeTestReport,
  newTestReport,
  TIER_VALUES,
  type TestReport,
  type Tier,
  type VerificationRequest,
} from "./schema.js";
import { runTier0Mutation } from "./tiers/tier0Mutation.js";
import { runTier1Pbt } from "./tiers/tier1Pbt.js";
import { runTier2Contract } from "./tiers/tier2Contract.js";
import { runTier3Proof } from "./tiers/tier3Proof.js";
import { runTier4HonestCi } from "./tiers/tier4HonestCi.js";

const PROTOCOL_DELIM = "===CRUCIBLE-TESTREPORT===\n";
const REPORTER_ID = "crucible-verify-typescript";
const REPORTER_VERSION = "2026.06.0-phase4";

interface ParsedArgs {
  tier: Tier;
}

function parseArgs(argv: readonly string[]): ParsedArgs {
  let tier: Tier | null = null;
  for (const a of argv.slice(2)) {
    if (a.startsWith("--tier=")) {
      const v = a.slice("--tier=".length);
      if ((TIER_VALUES as readonly string[]).includes(v)) {
        tier = v as Tier;
      } else {
        throw new Error(`unknown tier "${v}"`);
      }
    } else if (a === "--help" || a === "-h") {
      printHelp();
      process.exit(0);
    } else if (a === "--version") {
      process.stdout.write(`${REPORTER_ID} ${REPORTER_VERSION}\n`);
      process.exit(0);
    }
  }
  if (tier === null) {
    throw new Error(
      "missing --tier=<tier_0_mutation|tier_1_pbt|tier_2_contract|tier_3_proof|tier_4_honest_ci>",
    );
  }
  return { tier };
}

function printHelp(): void {
  process.stderr.write(`crucible-verify-typescript — per-language verifier runner

Usage:
  crucible-verify-typescript --tier=<tier> < request.json > report.json

Tiers:
  tier_0_mutation    Stryker mutation testing (diff-scoped, threshold 0.85)
  tier_1_pbt         fast-check property tests at >=10,000 iterations
  tier_2_contract    schemathesis sidecar against OpenAPI/GraphQL spec
  tier_3_proof       no-op placeholder — emits tool_unavailable
  tier_4_honest_ci   double pnpm build + sha256(dist/) comparison

Exit codes:
  0  TestReport emitted (verdict carries pass/fail)
  1  procedural crash (no report)
  2  executor-reasoning leak (ADR-002 invariant — refused before any work)
`);
}

async function readStdin(): Promise<string> {
  return new Promise((resolve, reject) => {
    let buf = "";
    stdin.setEncoding("utf8");
    stdin.on("data", (chunk: string) => {
      buf += chunk;
    });
    stdin.on("end", () => resolve(buf));
    stdin.on("error", (err) => reject(err));
  });
}

function emitReport(report: TestReport): void {
  // Always coerce reporter fields — they're load-bearing for the
  // attestation trail and must never drift from the binary identity.
  const finalized: TestReport = {
    ...report,
    reporter_id: REPORTER_ID,
    reporter_version: REPORTER_VERSION,
    language: "typescript",
  };
  process.stdout.write(PROTOCOL_DELIM);
  process.stdout.write(JSON.stringify(finalized));
  process.stdout.write("\n");
}

function emitError(tier: Tier | null, msg: string): void {
  // Even on internal error we emit a structured report so the daemon
  // gets a parseable response. The daemon distinguishes by `verdict`.
  const startedAt = new Date();
  const report = finalizeTestReport(
    newTestReport({
      task_id: "",
      diff_hash: "",
      tier: tier ?? "tier_0_mutation",
      framework: "crucible-verify-typescript",
      reporter_id: REPORTER_ID,
      reporter_version: REPORTER_VERSION,
      wall_clock_budget_seconds: 0,
      started_at: startedAt,
    }),
    new Date(),
  );
  emitReport({
    ...report,
    verdict: "failed",
    passed: false,
    error: msg,
  });
}

export async function dispatch(
  tier: Tier,
  req: VerificationRequest,
): Promise<TestReport> {
  switch (tier) {
    case "tier_0_mutation":
      return runTier0Mutation(req);
    case "tier_1_pbt":
      return runTier1Pbt(req);
    case "tier_2_contract":
      return runTier2Contract(req);
    case "tier_3_proof":
      return runTier3Proof(req);
    case "tier_4_honest_ci":
      return runTier4HonestCi(req);
  }
}

export async function main(argv: readonly string[]): Promise<number> {
  let parsed: ParsedArgs;
  try {
    parsed = parseArgs(argv);
  } catch (err) {
    process.stderr.write(`crucible-verify-typescript: ${(err as Error).message}\n`);
    return 1;
  }

  let raw: string;
  try {
    raw = await readStdin();
  } catch (err) {
    process.stderr.write(
      `crucible-verify-typescript: stdin read failed: ${(err as Error).message}\n`,
    );
    return 1;
  }
  if (raw.trim() === "") {
    process.stderr.write(
      "crucible-verify-typescript: empty stdin — expected VerificationRequest JSON\n",
    );
    return 1;
  }

  let req: VerificationRequest;
  try {
    const parsedJson = JSON.parse(raw) as unknown;
    if (parsedJson === null || typeof parsedJson !== "object") {
      throw new Error("request must be a JSON object");
    }
    // Run the leak audit on the raw parsed tree BEFORE casting — we
    // refuse on a key match even if the key isn't part of our typed
    // schema (defense against future shape extensions).
    auditNoLeakage(parsedJson);
    req = parsedJson as VerificationRequest;
    // Diff-path audit — refuses paths like `src/reasoning/...`.
    if (req.diff?.files !== undefined) {
      auditDiffPaths(req.diff.files.map((f) => f.path));
    }
  } catch (err) {
    if (err instanceof LeakageError) {
      process.stderr.write(
        `crucible-verify-typescript: REFUSING — ${err.message}\n`,
      );
      return 2;
    }
    process.stderr.write(
      `crucible-verify-typescript: invalid request JSON: ${(err as Error).message}\n`,
    );
    return 1;
  }

  let report: TestReport;
  try {
    report = await dispatch(parsed.tier, req);
  } catch (err) {
    emitError(parsed.tier, `dispatch crashed: ${(err as Error).message}`);
    return 0; // we still emitted a report — daemon will see verdict=failed
  }

  emitReport(report);
  return 0;
}

// Detect whether we're being executed directly vs imported by a test.
// On Windows, comparing url.pathname against argv[1] requires URL-decoding
// and slash normalisation — `fileURLToPath` + `path.resolve` does both
// for us. Linux/macOS behave identically under this routine.
const isMain = (() => {
  try {
    const argv1 = process.argv[1];
    if (argv1 === undefined) return false;
    const selfPath = fileURLToPath(import.meta.url);
    return path.resolve(argv1) === selfPath;
  } catch {
    return false;
  }
})();

if (isMain) {
  void main(process.argv).then(
    (code) => {
      process.exit(code);
    },
    (err: unknown) => {
      process.stderr.write(
        `crucible-verify-typescript: fatal: ${(err as Error)?.stack ?? String(err)}\n`,
      );
      process.exit(1);
    },
  );
}
