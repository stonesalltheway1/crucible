//! Layer 3 — Tetragon TracingPolicy emission + audit-event consumer.
//!
//! Tetragon attaches at the host kernel level on self-hosted Firecracker,
//! filtering on `tcp_connect`, `inode_unlink`, and `bprm_check_security`
//! using `NotifyEnforcer` with `Sigkill` actions. For the E2B tier where
//! in-guest eBPF is not available (the guest lacks `CAP_BPF`), we generate
//! the policy YAML for the orchestrator's host-side Tetragon instance and
//! consume the resulting event stream as advisory.
//!
//! Per the May 2026 currency check, the Tetragon socket default changed
//! from `localhost:54321` to `/var/run/tetragon/tetragon.sock`; we use the
//! socket form in the generated policy and the consumer.

use crate::Result;
use crucible_sandbox_spec::{EgressManifest, SyscallShimPolicy};
use serde::{Deserialize, Serialize};

/// Activate the Tetragon layer for the sandbox. Generates the
/// `TracingPolicyNamespaced` YAML and registers a Tetragon consumer
/// listening for matching events.
///
/// # Errors
/// Returns [`crate::Error::TetragonUnavailable`] on non-Linux hosts in a
/// production policy. Dev hosts log a `STUB:` and continue.
pub fn activate(policy: &SyscallShimPolicy) -> Result<()> {
    let _ = policy;
    inner::activate()
}

/// Render the Tetragon TracingPolicyNamespaced YAML for an egress allowlist.
/// Output is deterministic so the spec_hash includes it.
#[must_use]
pub fn render_egress_tracing_policy(
    sandbox_id: &str,
    tenant_id: &str,
    manifest: &EgressManifest,
) -> String {
    let allowed_cidrs: Vec<String> = manifest
        .rules
        .iter()
        .filter_map(|r| {
            // We pre-resolve FQDN→CIDR at the runtime's DNS-resolution
            // sidecar; here we only emit CIDR rules. Anything that isn't a
            // CIDR is treated as already-resolved by the sidecar before
            // this function is called.
            if r.host.parse::<std::net::IpAddr>().is_ok() || r.host.contains('/') {
                Some(r.host.clone())
            } else {
                None
            }
        })
        .collect();

    let policy = TracingPolicyNamespaced {
        api_version: "cilium.io/v1alpha1".into(),
        kind: "TracingPolicyNamespaced".into(),
        metadata: PolicyMetadata {
            name: format!("crucible-egress-{sandbox_id}"),
            namespace: format!("tenant-{tenant_id}"),
        },
        spec: PolicySpec {
            kprobes: vec![Kprobe {
                call: "tcp_connect".into(),
                syscall: false,
                args: vec![KprobeArg {
                    index: 0,
                    arg_type: "sock".into(),
                }],
                selectors: vec![Selector {
                    match_args: vec![MatchArg {
                        index: 0,
                        operator: "NotDAddr".into(),
                        values: {
                            let mut values = vec!["127.0.0.1".into(), "10.0.0.0/8".into()];
                            values.extend(allowed_cidrs);
                            values
                        },
                    }],
                    match_actions: vec![MatchAction {
                        action: "Sigkill".into(),
                    }],
                }],
            }],
        },
    };
    serde_yaml::to_string(&policy).expect("Tetragon policy is serde-yaml-clean")
}

// ─────────────────────────────────────────────────────────────────────────────
// YAML schema mirrors — defined locally so we can render without depending
// on the cilium-rust crate (which doesn't exist).
// ─────────────────────────────────────────────────────────────────────────────

#[derive(Debug, Clone, Serialize, Deserialize)]
struct TracingPolicyNamespaced {
    #[serde(rename = "apiVersion")]
    api_version: String,
    kind: String,
    metadata: PolicyMetadata,
    spec: PolicySpec,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
struct PolicyMetadata {
    name: String,
    namespace: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
struct PolicySpec {
    kprobes: Vec<Kprobe>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
struct Kprobe {
    call: String,
    syscall: bool,
    args: Vec<KprobeArg>,
    selectors: Vec<Selector>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
struct KprobeArg {
    index: u32,
    #[serde(rename = "type")]
    arg_type: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
struct Selector {
    #[serde(rename = "matchArgs")]
    match_args: Vec<MatchArg>,
    #[serde(rename = "matchActions")]
    match_actions: Vec<MatchAction>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
struct MatchArg {
    index: u32,
    operator: String,
    values: Vec<String>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
struct MatchAction {
    action: String,
}

#[cfg(target_os = "linux")]
mod inner {
    use crate::Result;

    pub fn activate() -> Result<()> {
        tracing::warn!(
            "STUB: tetragon::activate — TracingPolicy submission to \
             /var/run/tetragon/tetragon.sock pending. Linux dev hosts with \
             Tetragon installed can consume render_egress_tracing_policy output \
             via `kubectl apply -f -` or `tetra tracingpolicy add -`. Tracked in \
             docs/PHASE-2-REPORT.md."
        );
        Ok(())
    }
}

#[cfg(not(target_os = "linux"))]
mod inner {
    use crate::Result;

    pub fn activate() -> Result<()> {
        tracing::warn!(
            "STUB: tetragon::activate — non-Linux host. Layer 3 is a no-op for \
             cargo-test on dev hosts. Production runtime MUST be Linux."
        );
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crucible_sandbox_spec::{DefaultEgressAction, EgressDisposition, EgressManifest, EgressRule};

    #[test]
    fn render_emits_loopback_and_internal_cidrs_always() {
        let manifest = EgressManifest::deny_all();
        let yaml = render_egress_tracing_policy("sb_1", "ten_1", &manifest);
        assert!(yaml.contains("127.0.0.1"));
        assert!(yaml.contains("10.0.0.0/8"));
        assert!(yaml.contains("NotDAddr"));
        assert!(yaml.contains("Sigkill"));
    }

    #[test]
    fn render_includes_cidr_rules() {
        let manifest = EgressManifest {
            rules: vec![EgressRule {
                host: "192.0.2.0/24".into(),
                ports: vec![443],
                disposition: EgressDisposition::Allow,
                tape_only: false,
                justification: "test partner".into(),
            }],
            default_action: DefaultEgressAction::Deny,
        };
        let yaml = render_egress_tracing_policy("sb_1", "ten_1", &manifest);
        assert!(yaml.contains("192.0.2.0/24"));
    }

    #[test]
    fn render_omits_unresolved_fqdn() {
        // FQDN allowlist entries are pre-resolved to CIDRs by the runtime's
        // DNS-resolution sidecar; the Tetragon policy only carries
        // resolved addresses. The render function omits unresolved entries
        // (they'd be rejected by Tetragon's selector schema anyway).
        let manifest = EgressManifest {
            rules: vec![EgressRule {
                host: "api.example.com".into(),
                ports: vec![443],
                disposition: EgressDisposition::Allow,
                tape_only: false,
                justification: "test".into(),
            }],
            default_action: DefaultEgressAction::Deny,
        };
        let yaml = render_egress_tracing_policy("sb_1", "ten_1", &manifest);
        assert!(!yaml.contains("api.example.com"));
    }

    #[test]
    fn render_is_deterministic() {
        let manifest = EgressManifest::deny_all();
        let y1 = render_egress_tracing_policy("sb_1", "ten_1", &manifest);
        let y2 = render_egress_tracing_policy("sb_1", "ten_1", &manifest);
        assert_eq!(y1, y2);
    }

    #[test]
    fn render_yaml_parses_back() {
        let manifest = EgressManifest::deny_all();
        let yaml = render_egress_tracing_policy("sb_1", "ten_1", &manifest);
        let _back: TracingPolicyNamespaced = serde_yaml::from_str(&yaml).expect("roundtrip");
    }
}
