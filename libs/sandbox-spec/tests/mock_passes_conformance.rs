#![cfg(feature = "conformance")]

//! Integration test: the in-tree [`MockProvider`] satisfies the conformance
//! corpus. Concrete providers replicate this test in their own crate.

use crucible_sandbox_spec::conformance::{mock::MockProvider, run_conformance};
use crucible_sandbox_spec::SandboxKind;

#[tokio::test]
async fn mock_provider_passes_full_conformance() {
    let provider = MockProvider::new(SandboxKind::LocalDocker);
    run_conformance(&provider)
        .await
        .expect("mock provider must pass conformance");
}
