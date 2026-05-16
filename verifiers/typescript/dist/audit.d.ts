export declare const REASONING_DENYLIST: readonly string[];
export declare class LeakageError extends Error {
    readonly offendingField: string;
    readonly pattern: string;
    constructor(field: string, pattern: string);
}
/**
 * Walk an arbitrary JSON tree and throw LeakageError on the first key
 * that matches the denylist. Arrays of objects are recursed; arrays of
 * primitives are skipped (only keys are policy-relevant).
 */
export declare function auditNoLeakage(value: unknown, prefix?: string): void;
/**
 * Path-pattern guard for diff entries. The Go side scans diff file paths
 * for reasoning-like fragments; we replicate that so a malicious agent
 * cannot smuggle a chain-of-thought through the diff.
 */
export declare function auditDiffPaths(paths: readonly string[]): void;
