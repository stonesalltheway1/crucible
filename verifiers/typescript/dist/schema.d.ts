export declare const SCHEMA_VERSION: "1";
export declare const PREDICATE_TYPE: "https://crucible.dev/TestReport/v1";
export type Tier = "tier_0_mutation" | "tier_1_pbt" | "tier_2_contract" | "tier_3_proof" | "tier_4_honest_ci";
export declare const TIER_VALUES: readonly Tier[];
export type Language = "python" | "typescript" | "rust" | "go" | "java" | "swift" | "polyglot";
export type Verdict = "passed" | "failed" | "timed_out" | "tool_unavailable" | "skipped";
export interface SurvivedMutant {
    file: string;
    line: number;
    mutator: string;
    original?: string;
    replacement?: string;
}
export interface MutationStats {
    killed: number;
    survived: number;
    not_covered?: number;
    timeout?: number;
    total: number;
    score: number;
    threshold: number;
    diff_scoped: boolean;
    mutated_files?: string[];
    survived_summary?: SurvivedMutant[];
}
export interface Counterexample {
    property: string;
    shrunk: string;
    seed?: string;
    stack_hint?: string;
}
export interface PBTStats {
    iterations: number;
    iterations_min: number;
    properties?: string[];
    counterexamples?: Counterexample[];
    fuzz_corpus_size?: number;
    fuzz_new_seeds?: number;
    fuzz_crashes?: number;
}
export interface ContractViolation {
    endpoint: string;
    method: string;
    check: string;
    detail: string;
    reproducer?: string;
}
export interface ContractStats {
    spec_path?: string;
    spec_hash?: string;
    stateful_workflows?: number;
    checks?: string[];
    violations?: ContractViolation[];
    dst_iterations?: number;
    dst_replay_id?: string;
    dst_failing_schedule?: string;
}
export interface ProofStats {
    prover: string;
    proof_artifact?: string;
    obligations?: number;
    discharged?: number;
    timed_out: boolean;
    wall_clock_seconds?: number;
    cached_partial?: boolean;
    fallback_tier?: string;
    codeowner_review_required?: boolean;
    unsoundness_hints?: string[];
}
export interface HonestCIStats {
    builder_id: string;
    nix_flake_hash?: string;
    nix_lock_hash?: string;
    executor_rebuild_hash: string;
    verifier_rebuild_hash: string;
    bit_identical: boolean;
    slsa_level: number;
    in_toto_statement_hash?: string;
    fulcio_cert_hash?: string;
    rekor_uuid?: string;
    witness_attestation?: string;
    tekton_chains_ref?: string;
    diffoscope_report?: string;
    scrubber_audit_ok: boolean;
    scrubber_audit_entries?: number;
}
export interface Finding {
    category: string;
    severity: "info" | "warn" | "error";
    file?: string;
    line?: number;
    detail: string;
    suggested_fix?: string;
}
export interface TestReport {
    schema_version: typeof SCHEMA_VERSION;
    task_id: string;
    diff_hash: string;
    tier: Tier;
    language: Language;
    framework: string;
    verdict: Verdict;
    passed: boolean;
    started_at: string;
    finished_at: string;
    duration_seconds: number;
    wall_clock_budget_seconds: number;
    mutation?: MutationStats;
    pbt?: PBTStats;
    contract?: ContractStats;
    proof?: ProofStats;
    honest_ci?: HonestCIStats;
    findings?: Finding[];
    tool_digest?: string;
    reporter_id: string;
    reporter_version?: string;
    reporter_oidc_subject?: string;
    error?: string;
}
export type Action = "create" | "modify" | "delete" | "rename";
export interface FileChange {
    path: string;
    action: Action;
    content?: string;
    content_sha256?: string;
    size_bytes?: number;
}
export interface Diff {
    files: FileChange[];
    base_sha?: string;
}
export interface Routing {
    executor_model: string;
    executor_vendor: string;
    executor_tier?: string;
    verifier_model: string;
    verifier_vendor: string;
    verifier_tier?: string;
    critical_score?: number;
    is_critical?: boolean;
    decided_at?: string;
    classifier_attestation_id?: string;
}
export interface SpecChange {
    path: string;
    kind: "openapi" | "graphql" | "avro" | "proto" | string;
    previous_hash: string;
    current_hash: string;
    delta?: string;
}
export interface CriticalPathScore {
    file: string;
    score: number;
    band: "cold" | "warm" | "hot" | "molten" | string;
    reason?: string;
}
export interface BudgetEnvelope {
    verifier_cap_usd: number;
    verifier_spent_usd: number;
    wall_clock_cap_seconds: number;
    wall_clock_spent_seconds: number;
}
export interface VerificationRequest {
    task_id: string;
    tenant_id: string;
    repo: string;
    base_sha: string;
    diff: Diff;
    test_files?: FileChange[];
    spec_changes?: SpecChange[];
    routing: Routing;
    languages: string[];
    critical_path_scores?: CriticalPathScore[];
    per_task_signals?: Record<string, unknown>;
    budget: BudgetEnvelope;
    attestation_chain?: string[];
    executor_sandbox_id: string;
}
/**
 * Construct a TestReport with the schema-version-locked fields filled in.
 * Callers populate the tier-specific stats and findings.
 */
export declare function newTestReport(args: {
    task_id: string;
    diff_hash: string;
    tier: Tier;
    framework: string;
    reporter_id: string;
    reporter_version?: string;
    wall_clock_budget_seconds: number;
    started_at: Date;
}): TestReport;
export declare function finalizeTestReport(report: TestReport, finished_at: Date): TestReport;
