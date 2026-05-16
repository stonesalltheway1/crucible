#!/usr/bin/env node
import { type TestReport, type Tier, type VerificationRequest } from "./schema.js";
export declare function dispatch(tier: Tier, req: VerificationRequest): Promise<TestReport>;
export declare function main(argv: readonly string[]): Promise<number>;
