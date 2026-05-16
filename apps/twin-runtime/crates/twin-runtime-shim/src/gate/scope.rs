//! Scope resolution — twin-vs-real classification for path-dependent
//! destructive proposals.
//!
//! Rules:
//!
//! - A proposal whose **every** affected resource resides inside the twin
//!   filesystem mount (`/work/scratch`, `/work/repo`, `/work/.crucible/...`)
//!   resolves to [`crate::proposal::Scope::Twin`]. The overlayfs upper
//!   means `rm` here is just `umount overlay` away from full reversal.
//!
//! - Any path that escapes the twin tree, names a real cloud resource (an
//!   `s3://` URI, a `gs://` URI, an SSH host), or that the resolver can't
//!   classify confidently resolves to [`crate::proposal::Scope::Real`].
//!
//! - Commands with **no** affected-resource argument (e.g., `redis-cli
//!   FLUSHALL` against a configured server) are unresolvable from the
//!   command alone and fail-closed to `Real`.
//!
//! This is intentionally simple and conservative. Over-classifying as Real
//! costs a Promotion-Contract round-trip; under-classifying costs a
//! security incident.

use crate::proposal::{DestructiveProposal, Scope, WireScope};

/// Paths recognised as belonging to the twin filesystem mount layout
/// (`docs/01-architecture/twin-runtime.md` §"Filesystem layout inside the
/// sandbox").
const TWIN_PATH_PREFIXES: &[&str] = &[
    "/work/scratch",
    "/work/repo",
    "/work/.crucible",
    "/work/tapes",
    "/tmp/crucible-twin",
];

/// URI schemes that always denote real cloud resources.
const REAL_URI_SCHEMES: &[&str] = &[
    "s3://",
    "gs://",
    "az://",
    "abfs://",
    "wasbs://",
    "https://",
    "http://",
    "ssh://",
    "ftp://",
    "scp://",
];

/// Resolve the scope of `proposal`. The corpus default informs the prior;
/// for [`WireScope::PathDependent`] we look at the affected resources.
#[must_use]
pub fn resolve(proposal: &DestructiveProposal) -> Scope {
    if proposal.blast_radius.affected_resources.is_empty() {
        // Can't classify confidently — fail-closed.
        return Scope::Real;
    }

    let mut all_twin = true;
    for resource in &proposal.blast_radius.affected_resources {
        if is_definitely_real(resource) {
            return Scope::Real;
        }
        if !is_inside_twin(resource) {
            all_twin = false;
        }
    }
    if all_twin {
        Scope::Twin
    } else {
        Scope::Real
    }
}

fn is_definitely_real(resource: &str) -> bool {
    REAL_URI_SCHEMES.iter().any(|s| resource.starts_with(s))
        || resource.contains("@") && resource.contains(":")  // user@host:path
        || resource.starts_with("/etc/")
        || resource.starts_with("/var/")
        || resource.starts_with("/usr/")
        || resource.starts_with("/opt/")
        || resource.starts_with("/boot/")
        || resource.starts_with("/dev/")
        || resource.starts_with("/proc/")
        || resource.starts_with("/sys/")
}

fn is_inside_twin(resource: &str) -> bool {
    // Canonicalisation: trim leading `./`, collapse `//`.
    let normalised = canonicalise(resource);
    TWIN_PATH_PREFIXES.iter().any(|p| {
        let pn = canonicalise(p);
        normalised == pn || normalised.starts_with(&format!("{pn}/"))
    })
}

fn canonicalise(p: &str) -> String {
    // We deliberately do not call std::fs::canonicalize — the path may not
    // exist (yet), and we want pure-string semantics so the resolver is
    // testable. We also intentionally do NOT resolve `..` segments: a path
    // containing `..` is suspicious and treated as outside the twin.
    if p.contains("..") {
        return format!("[contains-dotdot:{p}]");
    }
    let mut s: String = p.to_string();
    while s.contains("//") {
        s = s.replace("//", "/");
    }
    if let Some(rest) = s.strip_prefix("./") {
        s = rest.to_string();
    }
    s
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::cmd_parse::{Command, CorpusHit};
    use crate::corpus::{PatternScope, Reversibility};
    use crate::proposal::DestructiveProposal;

    fn proposal(paths: &[&str]) -> DestructiveProposal {
        let argv = std::iter::once("rm".to_string())
            .chain(paths.iter().map(|s| s.to_string()))
            .collect();
        let hit = CorpusHit {
            pattern_id: "test",
            reason: "test",
            command: Command {
                argv,
                source_offset: 0,
            },
            default_scope: PatternScope::PathDependent,
            reversibility: Reversibility::Lossy,
        };
        DestructiveProposal::from_match("t", "test", &hit)
    }

    #[test]
    fn empty_paths_fail_closed_to_real() {
        let p = proposal(&[]);
        assert_eq!(resolve(&p), Scope::Real);
    }

    #[test]
    fn single_twin_path_resolves_twin() {
        let p = proposal(&["/work/scratch/x"]);
        assert_eq!(resolve(&p), Scope::Twin);
    }

    #[test]
    fn etc_path_resolves_real() {
        let p = proposal(&["/etc/passwd"]);
        assert_eq!(resolve(&p), Scope::Real);
    }

    #[test]
    fn s3_uri_resolves_real() {
        let p = proposal(&["s3://prod-bucket/data"]);
        assert_eq!(resolve(&p), Scope::Real);
    }

    #[test]
    fn ssh_remote_resolves_real() {
        let p = proposal(&["user@prod-host:/var/data"]);
        assert_eq!(resolve(&p), Scope::Real);
    }

    #[test]
    fn dotdot_path_treated_as_outside_twin() {
        let p = proposal(&["/work/scratch/../etc/passwd"]);
        assert_eq!(resolve(&p), Scope::Real);
    }

    #[test]
    fn double_slash_canonicalises() {
        let p = proposal(&["/work//scratch//build/x"]);
        assert_eq!(resolve(&p), Scope::Twin);
    }

    #[test]
    fn mixed_paths_resolve_real() {
        let p = proposal(&["/work/scratch/x", "/etc/passwd"]);
        assert_eq!(resolve(&p), Scope::Real);
    }

    #[test]
    fn unknown_path_resolves_real() {
        // Not inside twin, not definitely real → still Real (fail-closed).
        let p = proposal(&["foo/bar"]);
        assert_eq!(resolve(&p), Scope::Real);
    }

    #[test]
    fn tmp_crucible_twin_resolves_twin() {
        let p = proposal(&["/tmp/crucible-twin/working"]);
        assert_eq!(resolve(&p), Scope::Twin);
    }
}
