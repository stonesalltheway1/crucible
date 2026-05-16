import { type SurvivedMutant, type TestReport, type VerificationRequest } from "../schema.js";
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
interface MTEMutant {
    id: string;
    mutatorName: string;
    status: string;
    location: {
        start: {
            line: number;
            column: number;
        };
        end: {
            line: number;
            column: number;
        };
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
    thresholds?: {
        high: number;
        low: number;
    };
    projectRoot?: string;
}
declare function classifyMutants(report: MTEReport): {
    killed: number;
    survived: number;
    notCovered: number;
    timeout: number;
    total: number;
    survivors: SurvivedMutant[];
};
interface StrykerConfig {
    $schema?: string;
    packageManager?: string;
    reporters: string[];
    mutate: string[];
    testRunner: string;
    coverageAnalysis: string;
    incremental: boolean;
    incrementalFile?: string;
    thresholds?: {
        high: number;
        low: number;
        break: number;
    };
    jsonReporter?: {
        fileName: string;
    };
    htmlReporter?: {
        fileName: string;
    };
}
declare function makeStrykerConfig(mutateFiles: readonly string[]): StrykerConfig;
export declare function runTier0Mutation(req: VerificationRequest, opts?: Tier0Options): Promise<TestReport>;
export declare const __testing__: {
    classifyMutants: typeof classifyMutants;
    makeStrykerConfig: typeof makeStrykerConfig;
    MUTATION_THRESHOLD: number;
};
export {};
