//! Throughput bench for the Layer-1 parser. Run with `cargo bench`.
//!
//! Phase 2 has no perf SLO on Layer 1 yet — the bench exists so future
//! changes that regress parse throughput are visible without a separate
//! testing apparatus.

use std::time::Instant;
use twin_runtime_shim::Shim;
use crucible_sandbox_spec::SyscallShimPolicy;

fn main() {
    let shim = Shim::build(SyscallShimPolicy::default()).expect("default policy");
    let inputs = [
        "ls -la",
        "rm -rf /tmp/x",
        "echo $(rm -rf /etc); ls",
        "(cd / && rm -rf *) && git push --force",
        "FOO=bar baz; railway down; aws s3 rm s3://bucket/path",
        "git status && git log --oneline -20 | head -5",
    ];
    let iterations: u32 = 200_000;
    let start = Instant::now();
    for i in 0..iterations {
        let cmd = inputs[(i as usize) % inputs.len()];
        let _ = shim.evaluate_command(cmd, "task_bench");
    }
    let elapsed = start.elapsed();
    let ns_per = elapsed.as_nanos() / u128::from(iterations);
    println!(
        "cmd_parse_throughput: {iterations} iterations in {elapsed:?} ({ns_per} ns/op)"
    );
}
