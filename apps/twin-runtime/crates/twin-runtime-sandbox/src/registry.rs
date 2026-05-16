//! Provider registry — picks the right [`SandboxProvider`] for a given
//! [`crucible_sandbox_spec::SandboxKind`].
//!
//! The runtime instantiates one registry at startup and routes each spawn
//! request through it. This is the seam where new providers (Modal, Daytona,
//! Fly Machines) land.

use crucible_sandbox_spec::{Error, Result, SandboxKind, SandboxProvider};
use std::collections::HashMap;
use std::sync::Arc;

use crate::e2b::E2bProvider;
use crate::raw_firecracker::RawFirecrackerProvider;

/// A registry mapping [`SandboxKind`] to a provider instance.
pub struct ProviderRegistry {
    providers: HashMap<SandboxKind, Arc<dyn SandboxProvider>>,
}

impl ProviderRegistry {
    /// Build a registry populated with the providers available in this
    /// build of the runtime: E2B (real or stub depending on env) and
    /// raw-Firecracker (typed stub).
    #[must_use]
    pub fn with_defaults() -> Self {
        let mut providers: HashMap<SandboxKind, Arc<dyn SandboxProvider>> = HashMap::new();
        providers.insert(SandboxKind::E2b, Arc::new(E2bProvider::from_env()));
        providers.insert(
            SandboxKind::RawFirecracker,
            Arc::new(RawFirecrackerProvider::new()),
        );
        Self { providers }
    }

    /// Build an empty registry. Used by tests to inject mock providers.
    #[must_use]
    pub fn empty() -> Self {
        Self {
            providers: HashMap::new(),
        }
    }

    /// Register a provider for a given kind. Overrides any previous entry.
    pub fn register(&mut self, kind: SandboxKind, provider: Arc<dyn SandboxProvider>) {
        self.providers.insert(kind, provider);
    }

    /// Look up the provider for `kind`.
    ///
    /// # Errors
    /// Returns [`Error::Other`] when no provider is registered for the kind.
    pub fn get(&self, kind: SandboxKind) -> Result<Arc<dyn SandboxProvider>> {
        self.providers.get(&kind).cloned().ok_or_else(|| {
            Error::Other(format!("no provider registered for kind {kind:?}"))
        })
    }
}

impl Default for ProviderRegistry {
    fn default() -> Self {
        Self::with_defaults()
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crucible_sandbox_spec::conformance::mock::MockProvider;

    #[test]
    fn defaults_register_e2b_and_raw_firecracker() {
        let r = ProviderRegistry::with_defaults();
        assert!(r.get(SandboxKind::E2b).is_ok());
        assert!(r.get(SandboxKind::RawFirecracker).is_ok());
        assert!(r.get(SandboxKind::Daytona).is_err());
    }

    #[test]
    fn register_supports_mock_for_tests() {
        let mut r = ProviderRegistry::empty();
        r.register(
            SandboxKind::LocalDocker,
            Arc::new(MockProvider::new(SandboxKind::LocalDocker)),
        );
        assert!(r.get(SandboxKind::LocalDocker).is_ok());
    }
}
