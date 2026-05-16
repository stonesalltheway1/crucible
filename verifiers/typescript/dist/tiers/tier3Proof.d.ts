import { type TestReport, type VerificationRequest } from "../schema.js";
export interface Tier3Options {
    wallClockSeconds?: number;
}
export declare function runTier3Proof(req: VerificationRequest, opts?: Tier3Options): Promise<TestReport>;
