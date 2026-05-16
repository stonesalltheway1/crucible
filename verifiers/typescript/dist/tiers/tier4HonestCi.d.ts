import { type TestReport, type VerificationRequest } from "../schema.js";
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
    preSuppliedHashes?: {
        first: string;
        second: string;
    };
    dryRun?: boolean;
}
export declare function hashDirectory(root: string): Promise<string>;
export declare function runTier4HonestCi(req: VerificationRequest, opts?: Tier4Options): Promise<TestReport>;
export declare const __testing__: {
    hashDirectory: typeof hashDirectory;
};
