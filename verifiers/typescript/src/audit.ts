// audit.ts — executor-reasoning leak guard.
//
// Mirrors apps/verifier/internal/verification/verification.go:reasoningDenylist
// but operates on the parsed JSON request before any tool work. If a key
// anywhere in the request tree matches the denylist, we refuse with exit
// code 2 (ADR-002 invariant — the verifier MUST NOT see executor reasoning).

export const REASONING_DENYLIST: readonly string[] = [
  "reasoning",
  "chain_of_thought",
  "chain-of-thought",
  "cot",
  "thinking_trace",
  "thinking-trace",
  "thoughts",
  "scratchpad",
  "internal_monologue",
  "hidden_state",
  "agent_trace",
  "executor_trace",
  "trajectory",
  "plan_critique",
  "reflection",
] as const;

// Task-spec subset: the brief enumerates these specific tokens as the
// minimum allowed-list for v1. We OR them with the broader Go list so
// nothing on either side is missed.
const BRIEF_DENYLIST: readonly string[] = [
  "reasoning",
  "chain_of_thought",
  "scratchpad",
  "agent_trace",
  "executor_trace",
  "thinking_trace",
  "cot",
  "reflection",
] as const;

export class LeakageError extends Error {
  public readonly offendingField: string;
  public readonly pattern: string;
  constructor(field: string, pattern: string) {
    super(
      `executor-reasoning leak detected — field ${JSON.stringify(field)} matched pattern ${JSON.stringify(pattern)} (ADR-002 invariant)`,
    );
    this.name = "LeakageError";
    this.offendingField = field;
    this.pattern = pattern;
  }
}

function matchesDenylist(lowerKey: string): string | null {
  // BRIEF list first — exact substring match — then the broader list.
  for (const term of BRIEF_DENYLIST) {
    if (lowerKey.includes(term)) {
      return term;
    }
  }
  for (const term of REASONING_DENYLIST) {
    if (lowerKey.includes(term)) {
      return term;
    }
  }
  return null;
}

/**
 * Walk an arbitrary JSON tree and throw LeakageError on the first key
 * that matches the denylist. Arrays of objects are recursed; arrays of
 * primitives are skipped (only keys are policy-relevant).
 */
export function auditNoLeakage(value: unknown, prefix = ""): void {
  if (value === null || typeof value !== "object") {
    return;
  }
  if (Array.isArray(value)) {
    for (let i = 0; i < value.length; i++) {
      auditNoLeakage(value[i], `${prefix}[${i}]`);
    }
    return;
  }
  const obj = value as Record<string, unknown>;
  // Sort keys for deterministic error reporting — matches Go behaviour.
  const keys = Object.keys(obj).sort();
  for (const k of keys) {
    const full = prefix === "" ? k : `${prefix}.${k}`;
    const hit = matchesDenylist(k.toLowerCase());
    if (hit !== null) {
      throw new LeakageError(full, hit);
    }
    auditNoLeakage(obj[k], full);
  }
}

/**
 * Path-pattern guard for diff entries. The Go side scans diff file paths
 * for reasoning-like fragments; we replicate that so a malicious agent
 * cannot smuggle a chain-of-thought through the diff.
 */
export function auditDiffPaths(paths: readonly string[]): void {
  const denyFragments = [
    ".reasoning.",
    "/reasoning/",
    ".cot.",
    "/cot/",
    "_thinking_",
    "_scratchpad_",
    "agent_trace",
    "executor_trace",
  ];
  for (const p of paths) {
    const pl = p.toLowerCase();
    for (const frag of denyFragments) {
      if (pl.includes(frag)) {
        throw new LeakageError(`diff.files.${p}`, "path-pattern");
      }
    }
  }
}
