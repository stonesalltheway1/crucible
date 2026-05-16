//! Network namespace + Tetragon policy rendering.
//!
//! Tetragon attaches at the HOST, not in-guest (per the Phase 2 finding).
//! For the self-host tier we render the per-sandbox TracingPolicyNamespaced
//! YAML to the watch directory; the host's Tetragon daemon picks it up.
//!
//! Cilium handles the L3/L4 allowlist via the same kernel-resolved
//! `struct path` primitives the shim's Layer 2 uses. Per the May 2026
//! check, Cilium still has no native FQDN allowlist (IP/CIDR only) so
//! we render CIDRs; DNS resolution sits in a sidecar.

use serde::{Deserialize, Serialize};
use std::path::Path;

use crate::provider::Error;

/// Per-sandbox network policy.
#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct NetworkPolicy {
    /// Egress allowlist.
    pub egress: Vec<EgressRule>,
}

/// One CIDR + port range allowance.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EgressRule {
    /// CIDR block (IPv4 or IPv6).
    pub cidr: String,
    /// Allowed destination ports.
    pub ports: Vec<u16>,
}

/// Render the Tetragon policy to disk in the watch directory.
pub fn write_tetragon_policy(
    dir: &Path,
    policy: &NetworkPolicy,
    namespace: &str,
) -> Result<(), Error> {
    let yaml = render_yaml(policy, namespace);
    std::fs::create_dir_all(dir)
        .map_err(|e| Error::Io(format!("mkdir {}: {e}", dir.display())))?;
    let path = dir.join(format!("{namespace}.yaml"));
    fs_err::write(&path, yaml.as_bytes())
        .map_err(|e| Error::Io(format!("write policy {}: {e}", path.display())))?;
    Ok(())
}

fn render_yaml(policy: &NetworkPolicy, namespace: &str) -> String {
    let mut sb = String::new();
    sb.push_str("apiVersion: cilium.io/v1alpha1\n");
    sb.push_str("kind: TracingPolicyNamespaced\n");
    sb.push_str("metadata:\n");
    sb.push_str(&format!("  name: crucible-egress-{namespace}\n"));
    sb.push_str(&format!("  namespace: {namespace}\n"));
    sb.push_str("spec:\n");
    sb.push_str("  kprobes:\n");
    sb.push_str("    - call: tcp_connect\n");
    sb.push_str("      syscall: false\n");
    sb.push_str("      args:\n");
    sb.push_str("        - index: 0\n");
    sb.push_str("          type: sock\n");
    sb.push_str("      selectors:\n");
    sb.push_str("        - matchArgs:\n");
    sb.push_str("            - index: 0\n");
    sb.push_str("              operator: NotDAddr\n");
    sb.push_str("              values:\n");
    for rule in &policy.egress {
        sb.push_str("                - ");
        sb.push_str(&rule.cidr);
        sb.push('\n');
    }
    sb.push_str("          matchActions:\n");
    sb.push_str("            - action: Sigkill\n");
    sb
}

#[cfg(test)]
mod tests {
    use super::*;
    use tempfile::tempdir;

    #[test]
    fn render_yaml_includes_sigkill_action() {
        let p = NetworkPolicy {
            egress: vec![EgressRule {
                cidr: "10.0.0.0/8".into(),
                ports: vec![443],
            }],
        };
        let y = render_yaml(&p, "ns_test");
        assert!(y.contains("TracingPolicyNamespaced"));
        assert!(y.contains("Sigkill"));
        assert!(y.contains("10.0.0.0/8"));
        assert!(y.contains("ns_test"));
    }

    #[test]
    fn write_policy_creates_file_in_watch_dir() {
        let dir = tempdir().unwrap();
        let p = NetworkPolicy {
            egress: vec![EgressRule {
                cidr: "127.0.0.0/8".into(),
                ports: vec![80],
            }],
        };
        write_tetragon_policy(dir.path(), &p, "ns").unwrap();
        let f = dir.path().join("ns.yaml");
        assert!(f.exists());
        let content = std::fs::read_to_string(f).unwrap();
        assert!(content.contains("crucible-egress-ns"));
    }
}
