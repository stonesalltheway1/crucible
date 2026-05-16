// diff.ts — pull the TS/JS files out of a VerificationRequest's diff.
//
// The verifier daemon hands us the cumulative agent-authored diff. Each
// tier scopes its work to the diff (Crucible mandate: mutation must be
// diff-scoped, otherwise wall-clock explodes). This module is the single
// source of truth for "which files does this tier care about?".
const TS_EXTENSIONS = [
    ".ts",
    ".tsx",
    ".mts",
    ".cts",
    ".js",
    ".jsx",
    ".mjs",
    ".cjs",
];
const TEST_PATTERNS = [
    /\.test\.[mc]?[jt]sx?$/i,
    /\.spec\.[mc]?[jt]sx?$/i,
    /__tests__\//i,
];
const SPEC_PATTERNS = [
    /openapi.*\.(ya?ml|json)$/i,
    /swagger.*\.(ya?ml|json)$/i,
    /\.graphql$/i,
    /\.gql$/i,
];
function hasTsExtension(path) {
    const lower = path.toLowerCase();
    return TS_EXTENSIONS.some((ext) => lower.endsWith(ext));
}
function isTestPath(path) {
    return TEST_PATTERNS.some((re) => re.test(path));
}
function isSpecPath(path) {
    return SPEC_PATTERNS.some((re) => re.test(path));
}
/**
 * Classify the files in a Diff into source / test / spec / other buckets.
 * Deleted files are filtered out — there's nothing to mutate or test.
 */
export function classifyDiff(diff) {
    const out = {
        source: [],
        tests: [],
        production: [],
        specs: [],
        other: [],
    };
    for (const f of diff.files) {
        if (f.action === "delete") {
            continue;
        }
        if (hasTsExtension(f.path)) {
            out.source.push(f);
            if (isTestPath(f.path)) {
                out.tests.push(f);
            }
            else {
                out.production.push(f);
            }
            continue;
        }
        if (isSpecPath(f.path)) {
            out.specs.push(f);
            continue;
        }
        out.other.push(f);
    }
    return out;
}
/** Returns the file paths a tier should mutate / target. */
export function productionPaths(req) {
    return classifyDiff(req.diff).production.map((f) => f.path);
}
/** Returns the test-file paths the PBT tier should run. */
export function testPaths(req) {
    return classifyDiff(req.diff).tests.map((f) => f.path);
}
/** Returns the spec-file paths the contract tier should pull from. */
export function specPaths(req) {
    // Prefer the explicit spec_changes if present — that's the executor's
    // hand-curated list. Fall back to diff inference.
    if (req.spec_changes && req.spec_changes.length > 0) {
        return req.spec_changes.map((s) => s.path);
    }
    return classifyDiff(req.diff).specs.map((f) => f.path);
}
/** Strict diff hash — sha256-shaped placeholder when not provided. */
export function diffHash(req) {
    // The daemon sets base_sha; for unit testing we accept either.
    return req.diff.base_sha ?? req.base_sha ?? "";
}
//# sourceMappingURL=diff.js.map