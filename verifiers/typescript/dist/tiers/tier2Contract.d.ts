import { type TestReport, type VerificationRequest } from "../schema.js";
export interface Tier2Options {
    cwd?: string;
    schemathesisCmd?: readonly string[];
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
interface JUnitFailure {
    classname: string;
    name: string;
    message: string;
    body: string;
}
export declare function parseJUnitFailures(xml: string): JUnitFailure[];
declare function extractEndpoint(name: string): {
    method: string;
    endpoint: string;
};
export declare function runTier2Contract(req: VerificationRequest, opts?: Tier2Options): Promise<TestReport>;
export declare const __testing__: {
    parseJUnitFailures: typeof parseJUnitFailures;
    extractEndpoint: typeof extractEndpoint;
};
export {};
