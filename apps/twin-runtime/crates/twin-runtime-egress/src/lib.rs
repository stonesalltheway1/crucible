//! Egress allowlist enforcement.
//!
//! Two enforcement tiers per `docs/01-architecture/twin-runtime.md`
//! §"Layer 6":
//!
//! - **Production / self-hosted Firecracker**: Cilium / Tetragon
//!   TracingPolicy on `tcp_connect` with `Sigkill` on a non-allowlisted
//!   CIDR. Renderer in [`tetragon`].
//! - **E2B / solo-founder**: mitmproxy in transparent mode with a
//!   `tls_clienthello`-based allowlist addon. Per the May 2026 currency
//!   check, `allow_hosts` controls MITM-or-passthrough, not drop semantics;
//!   we use the addon for actual drops. Config renderer in [`mitmproxy`].
//!
//! This crate also exposes a [`ManifestValidator`] that the runtime calls
//! before forwarding a [`SandboxSpec`] to a provider; invalid manifests
//! fail-closed.

#![warn(missing_docs)]

use crucible_sandbox_spec::{EgressManifest, EgressRule, SandboxKind};
use thiserror::Error;

pub mod mitmproxy;
pub mod tetragon;

/// Errors raised by egress orchestration.
#[derive(Debug, Error)]
pub enum Error {
    /// Manifest violates an invariant.
    #[error("invalid egress manifest: {0}")]
    Invalid(String),
    /// Failed to render policy.
    #[error("render: {0}")]
    Render(String),
}

/// Result alias.
pub type Result<T> = std::result::Result<T, Error>;

/// Enforcement tier — corresponds to which renderer we use.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum EnforcementTier {
    /// Tetragon-on-host with SIGKILL.
    Tetragon,
    /// mitmproxy userspace allowlist.
    Mitmproxy,
}

impl EnforcementTier {
    /// Pick the right tier for a given sandbox kind.
    #[must_use]
    pub fn for_kind(kind: SandboxKind) -> Self {
        match kind {
            SandboxKind::RawFirecracker => Self::Tetragon,
            // E2B's guest cannot run eBPF; we lean on mitmproxy and E2B's
            // native SandboxNetworkOpts as the layered enforcement.
            SandboxKind::E2b | SandboxKind::Modal | SandboxKind::Daytona
            | SandboxKind::FlyMachines | SandboxKind::LocalDocker => Self::Mitmproxy,
        }
    }
}

/// Manifest validator.
pub struct ManifestValidator;

impl ManifestValidator {
    /// Validate the manifest. Fail-closed on any anomaly.
    ///
    /// # Errors
    /// Returns [`Error::Invalid`] on the first violation.
    pub fn validate(manifest: &EgressManifest) -> Result<()> {
        let mut seen_hosts = std::collections::HashSet::new();
        for rule in &manifest.rules {
            if rule.host.is_empty() {
                return Err(Error::Invalid("rule with empty host".into()));
            }
            if rule.host.contains(char::is_whitespace) {
                return Err(Error::Invalid(format!(
                    "rule host contains whitespace: {:?}",
                    rule.host
                )));
            }
            if rule.justification.is_empty() {
                return Err(Error::Invalid(format!(
                    "rule '{host}' missing justification (audit requirement)",
                    host = rule.host
                )));
            }
            if !seen_hosts.insert(rule.host.clone()) {
                return Err(Error::Invalid(format!("duplicate host: {}", rule.host)));
            }
            check_dangerous(rule)?;
        }
        Ok(())
    }
}

fn check_dangerous(rule: &EgressRule) -> Result<()> {
    let lower = rule.host.to_ascii_lowercase();
    if lower == "0.0.0.0"
        || lower == "0.0.0.0/0"
        || lower == "*"
        || lower == "::/0"
        || lower == "any"
    {
        return Err(Error::Invalid(format!(
            "rule host '{}' is a wildcard — not permitted in the egress allowlist",
            rule.host
        )));
    }
    if lower.ends_with(".internal")
        || lower.ends_with(".local")
        || lower.starts_with("169.254.")
    {
        return Err(Error::Invalid(format!(
            "rule host '{}' is a link-local / cloud-metadata address; \
             egress to these is structurally disallowed (see threat-model.md T14)",
            rule.host
        )));
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use crucible_sandbox_spec::{DefaultEgressAction, EgressDisposition};

    fn rule(host: &str) -> EgressRule {
        EgressRule {
            host: host.into(),
            ports: vec![443],
            disposition: EgressDisposition::Allow,
            tape_only: false,
            justification: "test".into(),
        }
    }

    #[test]
    fn valid_manifest_accepts() {
        let manifest = EgressManifest {
            rules: vec![rule("api.stripe.com"), rule("api.openai.com")],
            default_action: DefaultEgressAction::Deny,
        };
        ManifestValidator::validate(&manifest).expect("valid manifest");
    }

    #[test]
    fn wildcard_rejected() {
        let manifest = EgressManifest {
            rules: vec![rule("0.0.0.0/0")],
            default_action: DefaultEgressAction::Deny,
        };
        assert!(ManifestValidator::validate(&manifest).is_err());
    }

    #[test]
    fn link_local_rejected() {
        let manifest = EgressManifest {
            rules: vec![rule("169.254.169.254")],
            default_action: DefaultEgressAction::Deny,
        };
        assert!(ManifestValidator::validate(&manifest).is_err());
    }

    #[test]
    fn missing_justification_rejected() {
        let mut r = rule("api.example.com");
        r.justification = String::new();
        let manifest = EgressManifest {
            rules: vec![r],
            default_action: DefaultEgressAction::Deny,
        };
        assert!(ManifestValidator::validate(&manifest).is_err());
    }

    #[test]
    fn duplicate_host_rejected() {
        let manifest = EgressManifest {
            rules: vec![rule("api.example.com"), rule("api.example.com")],
            default_action: DefaultEgressAction::Deny,
        };
        assert!(ManifestValidator::validate(&manifest).is_err());
    }

    #[test]
    fn cloud_metadata_internal_dns_rejected() {
        let manifest = EgressManifest {
            rules: vec![rule("metadata.google.internal")],
            default_action: DefaultEgressAction::Deny,
        };
        assert!(ManifestValidator::validate(&manifest).is_err());
    }

    #[test]
    fn enforcement_tier_picks_correctly() {
        assert_eq!(
            EnforcementTier::for_kind(SandboxKind::RawFirecracker),
            EnforcementTier::Tetragon
        );
        assert_eq!(
            EnforcementTier::for_kind(SandboxKind::E2b),
            EnforcementTier::Mitmproxy
        );
    }
}
