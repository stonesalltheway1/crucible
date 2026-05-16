// tier2Contract.ts — schemathesis sidecar.
//
// Schemathesis is a Python tool. We shell out to `pipx run schemathesis`
// (same as the Python runner does) so this driver doesn't need to vendor
// the Python interpreter into the TS verifier image. Output is JUnit XML
// which we parse with a tiny regex-based extractor (real parsing is
// not worth a dep here — we only need <testsuite>/<testcase>/<failure>).
//
// The spec is detected from req.spec_changes first, then from the diff's
// `openapi*.{yaml,yml,json}` / `*.graphql` files. If no spec is present,
// the tier is skipped.

import { spawn } from "node:child_process";
import { mkdtemp, readFile, rm } from "node:fs/promises";
import os from "node:os";
import path from "node:path";

import { specPaths } from "../diff.js";
import {
  finalizeTestReport,
  newTestReport,
  type ContractStats,
  type ContractViolation,
  type Finding,
  type TestReport,
  type VerificationRequest,
} from "../schema.js";

export interface Tier2Options {
  cwd?: string;
  schemathesisCmd?: readonly string[]; // default: ["pipx", "run", "schemathesis"]
  /** Endpoint base URL — defaults to env CRUCIBLE_SCHEMATHESIS_BASE_URL. */
  baseUrl?: string;
  wallClockSeconds?: number;
  preSuppliedJunit?: string;
  /**
   * If true, do not invoke schemathesis at all (used when the runner image
   * is offline and pipx isn't reachable). Emits tool_unavailable.
   */
  dryRun?: boolean;
}

// ---- Tiny JUnit XML extractor -------------------------------------------
//
// We extract <testcase classname=".." name=".." (time=".." )?> elements and
// their nested <failure message=".." type="..">..</failure> entries. This
// is *not* a full XML parser — it intentionally trades robustness for zero
// deps. Schemathesis 4.x output is well-formed and stable in shape.

interface JUnitFailure {
  classname: string;
  name: string;
  message: string;
  body: string;
}

export function parseJUnitFailures(xml: string): JUnitFailure[] {
  const out: JUnitFailure[] = [];
  const caseRe =
    /<testcase\b([^>]*)>([\s\S]*?)<\/testcase>|<testcase\b([^/]*)\/>/g;
  let m: RegExpExecArray | null;
  while ((m = caseRe.exec(xml)) !== null) {
    const attrs = m[1] ?? m[3] ?? "";
    const inner = m[2] ?? "";
    const classname = /classname="([^"]*)"/.exec(attrs)?.[1] ?? "";
    const name = /name="([^"]*)"/.exec(attrs)?.[1] ?? "";
    if (!inner) continue;
    const failureRe =
      /<failure\b([^>]*)>([\s\S]*?)<\/failure>|<failure\b([^/]*)\/>/g;
    let f: RegExpExecArray | null;
    while ((f = failureRe.exec(inner)) !== null) {
      const fAttrs = f[1] ?? f[3] ?? "";
      const body = (f[2] ?? "").trim();
      const message = /message="([^"]*)"/.exec(fAttrs)?.[1] ?? "";
      out.push({
        classname,
        name,
        message: decodeXmlEntities(message),
        body: decodeXmlEntities(body),
      });
    }
  }
  return out;
}

function decodeXmlEntities(s: string): string {
  return s
    .replace(/&lt;/g, "<")
    .replace(/&gt;/g, ">")
    .replace(/&quot;/g, '"')
    .replace(/&apos;/g, "'")
    .replace(/&amp;/g, "&");
}

// ---- Endpoint extraction from JUnit testcase name -----------------------
//
// Schemathesis testcase names look like "GET /pets/{id}[200]" or
// "test_api[POST /users]". We probe a few patterns.
function extractEndpoint(name: string): { method: string; endpoint: string } {
  const m1 = /\b(GET|POST|PUT|PATCH|DELETE|HEAD|OPTIONS)\s+(\/[^\s\]]*)/i.exec(
    name,
  );
  if (m1) {
    return { method: m1[1]!.toUpperCase(), endpoint: m1[2]! };
  }
  return { method: "UNKNOWN", endpoint: name };
}

// ---- Subprocess driver --------------------------------------------------

function runSchemathesis(
  argv: readonly string[],
  cwd: string,
  wallClockSeconds: number,
): Promise<{ exitCode: number; stderr: string }> {
  return new Promise((resolve, reject) => {
    const [head, ...rest] = argv;
    if (head === undefined) {
      reject(new Error("schemathesis: empty argv"));
      return;
    }
    const child = spawn(head, rest, {
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

export async function runTier2Contract(
  req: VerificationRequest,
  opts: Tier2Options = {},
): Promise<TestReport> {
  const startedAt = new Date();
  const cwd = opts.cwd ?? process.cwd();
  const wallClockSeconds =
    opts.wallClockSeconds ??
    Math.max(900, Math.min(2700, req.budget.wall_clock_cap_seconds || 900));

  const baseReport = newTestReport({
    task_id: req.task_id,
    diff_hash: req.diff.base_sha ?? req.base_sha,
    tier: "tier_2_contract",
    framework: "schemathesis-sidecar",
    reporter_id: "crucible-verify-typescript",
    reporter_version: "2026.06.0-phase4",
    wall_clock_budget_seconds: wallClockSeconds,
    started_at: startedAt,
  });

  const specs = specPaths(req);
  if (specs.length === 0) {
    return finalizeTestReport(
      {
        ...baseReport,
        verdict: "skipped",
        passed: true,
        error: "no OpenAPI/GraphQL spec changes in diff — Tier 2 skipped",
      },
      new Date(),
    );
  }
  const spec = specs[0]!;
  const baseUrl =
    opts.baseUrl ??
    process.env["CRUCIBLE_SCHEMATHESIS_BASE_URL"] ??
    "http://localhost:8080";

  let raw: string;

  if (opts.preSuppliedJunit !== undefined) {
    raw = opts.preSuppliedJunit;
  } else if (opts.dryRun === true) {
    return finalizeTestReport(
      {
        ...baseReport,
        verdict: "tool_unavailable",
        passed: false,
        contract: { spec_path: spec },
        error: "schemathesis not invoked (dry-run)",
      },
      new Date(),
    );
  } else {
    const tmp = await mkdtemp(path.join(os.tmpdir(), "crucible-tier2-"));
    const junitPath = path.join(tmp, "schemathesis.xml");
    const cmd =
      opts.schemathesisCmd ??
      (["pipx", "run", "schemathesis"] as readonly string[]);
    const argv: string[] = [
      ...cmd,
      "run",
      spec,
      "--base-url",
      baseUrl,
      "--checks",
      "all",
      `--junit-xml=${junitPath}`,
      "--hypothesis-max-examples=200",
      "--hypothesis-deadline=2000",
    ];
    try {
      const result = await runSchemathesis(argv, cwd, wallClockSeconds);
      try {
        raw = await readFile(junitPath, "utf8");
      } catch {
        return finalizeTestReport(
          {
            ...baseReport,
            verdict: "tool_unavailable",
            passed: false,
            contract: { spec_path: spec },
            error: `schemathesis ran (exit ${result.exitCode}) but JUnit output missing. stderr tail: ${result.stderr.slice(-1500)}`,
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
          contract: { spec_path: spec },
          error: `schemathesis spawn failed: ${(err as Error).message}`,
        },
        new Date(),
      );
    } finally {
      void rm(tmp, { recursive: true, force: true }).catch(() => {});
    }
  }

  const failures = parseJUnitFailures(raw);
  const violations: ContractViolation[] = failures.map((f) => {
    const { method, endpoint } = extractEndpoint(f.name);
    return {
      endpoint,
      method,
      check: f.classname || "response_schema_conformance",
      detail: f.message || f.body.slice(0, 400),
      reproducer: f.body.slice(0, 800),
    };
  });

  const contract: ContractStats = {
    spec_path: spec,
    checks: ["response_schema_conformance", "status_code_conformance"],
    ...(violations.length > 0 ? { violations } : {}),
  };

  const findings: Finding[] = violations.slice(0, 50).map((v) => ({
    category: "contract_violation",
    severity: "error" as const,
    detail: `${v.method} ${v.endpoint}: ${v.check} — ${v.detail}`,
  }));

  const passed = violations.length === 0;
  return finalizeTestReport(
    {
      ...baseReport,
      contract,
      findings,
      verdict: passed ? "passed" : "failed",
      passed,
    },
    new Date(),
  );
}

export const __testing__ = { parseJUnitFailures, extractEndpoint };
