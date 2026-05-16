import type { Diff, FileChange, VerificationRequest } from "./schema.js";
export interface DiffClassification {
    /** All TS/JS files touched by the diff (excluding deletions). */
    source: FileChange[];
    /** Subset of `source` that are test files (`*.test.ts`, etc.). */
    tests: FileChange[];
    /** Non-test TS/JS files — these are the mutation/PBT targets. */
    production: FileChange[];
    /** OpenAPI / GraphQL spec files touched by the diff. */
    specs: FileChange[];
    /** Files we don't grade in TS-land (e.g. *.md). */
    other: FileChange[];
}
/**
 * Classify the files in a Diff into source / test / spec / other buckets.
 * Deleted files are filtered out — there's nothing to mutate or test.
 */
export declare function classifyDiff(diff: Diff): DiffClassification;
/** Returns the file paths a tier should mutate / target. */
export declare function productionPaths(req: VerificationRequest): string[];
/** Returns the test-file paths the PBT tier should run. */
export declare function testPaths(req: VerificationRequest): string[];
/** Returns the spec-file paths the contract tier should pull from. */
export declare function specPaths(req: VerificationRequest): string[];
/** Strict diff hash — sha256-shaped placeholder when not provided. */
export declare function diffHash(req: VerificationRequest): string;
