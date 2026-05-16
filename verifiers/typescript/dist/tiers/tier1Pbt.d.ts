import { type Counterexample, type TestReport, type VerificationRequest } from "../schema.js";
export declare const IT_PROP_MIN_RUNS = 10000;
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
export declare function parseCounterexample(property: string, failureMessage: string): Counterexample | null;
declare function findPbtFiles(cwd: string, candidates: readonly string[]): Promise<string[]>;
export declare function runTier1Pbt(req: VerificationRequest, opts?: Tier1Options): Promise<TestReport>;
export declare const __testing__: {
    parseCounterexample: typeof parseCounterexample;
    findPbtFiles: typeof findPbtFiles;
};
export {};
