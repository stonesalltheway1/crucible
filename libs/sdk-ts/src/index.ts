// Crucible SDK — TypeScript entry point.
//
// Phase 1 ships types only. The runtime `twin.*` surface lives in Phase 2 once
// the AgentSdkService gRPC handlers in apps/twin-runtime exist.

export * from "./types.js";

export const SDK_VERSION = "2026.6.0-phase1";

export interface ClientOptions {
  endpoint: string;     // e.g. "http://localhost:8080"
  tenantId?: string;
  apiKey?: string;
}

// CrucibleClient is the Phase-2 entry point. Phase-1 stub returns a structured
// "not yet wired" error from every call so consumers can integrate against the
// shape today and swap to the real implementation later.
export class CrucibleClient {
  constructor(private opts: ClientOptions) {}

  notWired(method: string): never {
    throw new Error(
      `STUB: CrucibleClient.${method}() — Phase 1 stub. ` +
        `The twin.* surface ships with Phase 2; see docs/PHASE-1-REPORT.md.`,
    );
  }

  get endpoint(): string {
    return this.opts.endpoint;
  }
}
