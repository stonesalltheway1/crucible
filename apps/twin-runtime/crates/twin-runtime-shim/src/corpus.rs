//! The destructive-pattern corpus.
//!
//! Each entry describes one class of destructive operation observable at the
//! shell layer. The matcher in [`crate::cmd_parse`] walks every extracted
//! [`crate::cmd_parse::Command`] against this corpus.
//!
//! Adding a new pattern is a one-line append here — the property test in
//! `tests/property_50k.rs` will exercise it automatically because every
//! corpus entry is sampled in the adversarial-input generator.
//!
//! The corpus is intentionally **liberal**: false-positive interceptions
//! cost a Promotion-Contract round-trip; false-negative non-interceptions
//! cost a security incident. We err toward false positive every time.

use once_cell::sync::Lazy;

/// One destructive-pattern entry.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct DestructivePattern {
    /// Stable semantic identifier (kebab-case). Used in attestations.
    pub id: &'static str,
    /// Binary name the pattern matches on (first argv element).
    /// Some patterns match many binaries (e.g. cloud CLIs) — use
    /// [`Self::extra_binaries`] for additional names; the primary name
    /// is the most common.
    pub binary: &'static str,
    /// Other binary names that share this pattern.
    pub extra_binaries: &'static [&'static str],
    /// Argument predicate. Returns true if `argv[1..]` is destructive.
    pub arg_predicate: fn(&[String]) -> bool,
    /// Human-readable reason recorded in the proposal.
    pub reason: &'static str,
    /// Default scope classifier — `"twin"` or `"real"`. Many patterns are
    /// always real-scoped (`railway down`, `terraform destroy`); `rm`-family
    /// patterns are classified at gate-time based on the affected paths.
    pub default_scope: PatternScope,
    /// Reversibility hint — feeds [`crate::proposal::DestructiveProposal::blast_radius`].
    pub reversibility: Reversibility,
}

/// Reversibility classification — mirrors the proto enum but kept private
/// to the shim so the corpus doesn't depend on proto types.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Reversibility {
    /// `git revert`-style; trivial.
    Trivial,
    /// Restorable from snapshot.
    Snapshot,
    /// Partial recovery possible.
    Lossy,
    /// Gone forever.
    Irreversible,
}

/// Default-scope hint for a corpus entry.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum PatternScope {
    /// Always twin-scoped — the syscall affects only the sandbox.
    AlwaysTwin,
    /// Always real-scoped — the syscall affects production resources.
    AlwaysReal,
    /// Path-dependent — the gate classifies at evaluation time.
    PathDependent,
}

// ─────────────────────────────────────────────────────────────────────────────
// Predicate helpers
// ─────────────────────────────────────────────────────────────────────────────

fn has_any(argv: &[String], needles: &[&str]) -> bool {
    argv.iter().any(|a| needles.iter().any(|n| a == n))
}

fn starts_with_any(argv: &[String], prefixes: &[&str]) -> bool {
    argv.iter()
        .any(|a| prefixes.iter().any(|p| a.starts_with(p)))
}

fn rm_recursive(argv: &[String]) -> bool {
    // rm -r / rm -R / rm -rf / rm --recursive / rm -fr / rm -Rf etc.
    // Plus the flagless `rm` with shell wildcards is still destructive on the
    // shell side; we treat any `rm` as a destructive proposal candidate.
    // The matcher only fires when at least one argument names a path.
    let recursive = argv.iter().any(|a| {
        matches!(
            a.as_str(),
            "-r" | "-R" | "-rf" | "-Rf" | "-fr" | "-fR" | "--recursive"
        ) || (a.starts_with('-') && (a.contains('r') || a.contains('R')))
    });
    let force = argv.iter().any(|a| {
        matches!(a.as_str(), "-f" | "-rf" | "-fr" | "-Rf" | "-fR" | "--force")
            || (a.starts_with('-') && a.contains('f'))
    });
    // Bare `rm path` is also destructive; flag it.
    recursive || force || !argv.is_empty()
}

fn find_delete(argv: &[String]) -> bool {
    has_any(argv, &["-delete", "-exec", "-execdir"])
        && (has_any(argv, &["-delete"]) || argv.windows(2).any(|w| {
            (w[0] == "-exec" || w[0] == "-execdir")
                && (w[1].contains("rm") || w[1].contains("rmdir"))
        }))
}

fn git_destructive(argv: &[String]) -> bool {
    if argv.is_empty() {
        return false;
    }
    match argv[0].as_str() {
        "push" => argv.iter().any(|a| {
            matches!(a.as_str(), "--force" | "-f" | "--force-with-lease")
                || a.starts_with("+")
        }),
        "reset" => argv.iter().any(|a| a == "--hard"),
        "clean" => has_any(argv, &["-f", "-fd", "-fdx", "-x", "-X"]),
        "branch" => has_any(argv, &["-D"]),
        "tag" => has_any(argv, &["-d"]),
        "filter-branch" | "filter-repo" => true,
        _ => false,
    }
}

fn kubectl_destructive(argv: &[String]) -> bool {
    if argv.is_empty() {
        return false;
    }
    matches!(argv[0].as_str(), "delete" | "drain" | "uncordon")
        || (argv[0] == "scale" && has_any(argv, &["--replicas=0"]))
}

fn helm_destructive(argv: &[String]) -> bool {
    matches!(
        argv.first().map(String::as_str),
        Some("uninstall" | "delete" | "rollback")
    )
}

fn terraform_destructive(argv: &[String]) -> bool {
    matches!(
        argv.first().map(String::as_str),
        Some("destroy" | "taint" | "state")
    ) && !matches!(
        argv.get(1).map(String::as_str),
        Some("show" | "list" | "pull")
    )
}

fn cloud_delete(argv: &[String]) -> bool {
    // aws s3 rm, aws s3api delete-*, aws ec2 terminate-instances, etc.
    // We match any token containing "delete" or "terminate" or "destroy".
    argv.iter().any(|a| {
        let lower = a.to_ascii_lowercase();
        lower.contains("delete") || lower.contains("terminate") || lower.contains("destroy") || lower == "rm"
    })
}

fn paas_destructive(argv: &[String]) -> bool {
    // railway down / fly destroy / vercel rm / heroku apps:destroy
    if argv.is_empty() {
        return false;
    }
    let first = argv[0].as_str();
    matches!(
        first,
        "down" | "destroy" | "rm" | "delete" | "remove" | "apps:destroy" | "apps:delete"
    ) || starts_with_any(argv, &["apps:destroy", "apps:delete"])
}

fn powershell_destructive(argv: &[String]) -> bool {
    let recursive = has_any(argv, &["-Recurse", "/s"]);
    let force = has_any(argv, &["-Force"]);
    recursive || force || (!argv.is_empty() && (argv[0].starts_with('-') || argv[0].starts_with('C') || argv[0].starts_with('D')))
}

fn redis_destructive(argv: &[String]) -> bool {
    argv.iter().any(|a| {
        matches!(
            a.to_ascii_uppercase().as_str(),
            "FLUSHALL" | "FLUSHDB" | "DEBUG" | "SHUTDOWN" | "CONFIG"
        )
    })
}

fn mongo_destructive(argv: &[String]) -> bool {
    has_any(argv, &["--drop", "--dropTarget"])
        || argv.iter().any(|a| a.contains("dropDatabase"))
}

fn sql_destructive(argv: &[String]) -> bool {
    // For shell-style invocation: `psql -c 'DROP TABLE foo'` etc.
    let blob = argv.join(" ").to_ascii_uppercase();
    blob.contains("DROP TABLE")
        || blob.contains("DROP DATABASE")
        || blob.contains("TRUNCATE")
        || blob.contains("DELETE FROM ") && !blob.contains("WHERE ")
        || blob.contains("ALTER TABLE") && blob.contains("DROP COLUMN")
        || blob.contains("DROP SCHEMA")
}

fn always_true(_: &[String]) -> bool {
    true
}

// ─────────────────────────────────────────────────────────────────────────────
// The corpus itself
// ─────────────────────────────────────────────────────────────────────────────

/// The active destructive-pattern corpus. Loaded once at process start.
pub static CORPUS: Lazy<Vec<DestructivePattern>> = Lazy::new(build_corpus);

fn build_corpus() -> Vec<DestructivePattern> {
    vec![
        DestructivePattern {
            id: "rm-recursive",
            binary: "rm",
            extra_binaries: &[],
            arg_predicate: rm_recursive,
            reason: "rm with recursive/force flag",
            default_scope: PatternScope::PathDependent,
            reversibility: Reversibility::Lossy,
        },
        DestructivePattern {
            id: "rmdir",
            binary: "rmdir",
            extra_binaries: &[],
            arg_predicate: always_true,
            reason: "directory removal",
            default_scope: PatternScope::PathDependent,
            reversibility: Reversibility::Lossy,
        },
        DestructivePattern {
            id: "find-delete",
            binary: "find",
            extra_binaries: &[],
            arg_predicate: find_delete,
            reason: "find with -delete or -exec rm",
            default_scope: PatternScope::PathDependent,
            reversibility: Reversibility::Lossy,
        },
        DestructivePattern {
            id: "git-destructive",
            binary: "git",
            extra_binaries: &[],
            arg_predicate: git_destructive,
            reason: "git push --force / reset --hard / clean -f / branch -D / tag -d / filter-branch",
            default_scope: PatternScope::PathDependent,
            reversibility: Reversibility::Lossy,
        },
        DestructivePattern {
            id: "kubectl-destructive",
            binary: "kubectl",
            extra_binaries: &[],
            arg_predicate: kubectl_destructive,
            reason: "kubectl delete / drain / scale --replicas=0",
            default_scope: PatternScope::AlwaysReal,
            reversibility: Reversibility::Lossy,
        },
        DestructivePattern {
            id: "helm-destructive",
            binary: "helm",
            extra_binaries: &[],
            arg_predicate: helm_destructive,
            reason: "helm uninstall / delete / rollback",
            default_scope: PatternScope::AlwaysReal,
            reversibility: Reversibility::Snapshot,
        },
        DestructivePattern {
            id: "terraform-destroy",
            binary: "terraform",
            extra_binaries: &["tofu"],
            arg_predicate: terraform_destructive,
            reason: "terraform destroy / taint / state mutation",
            default_scope: PatternScope::AlwaysReal,
            reversibility: Reversibility::Irreversible,
        },
        DestructivePattern {
            id: "aws-delete",
            binary: "aws",
            extra_binaries: &[],
            arg_predicate: cloud_delete,
            reason: "aws delete-* / terminate-* / rm",
            default_scope: PatternScope::AlwaysReal,
            reversibility: Reversibility::Irreversible,
        },
        DestructivePattern {
            id: "gcloud-delete",
            binary: "gcloud",
            extra_binaries: &[],
            arg_predicate: cloud_delete,
            reason: "gcloud delete-* / instances delete",
            default_scope: PatternScope::AlwaysReal,
            reversibility: Reversibility::Irreversible,
        },
        DestructivePattern {
            id: "az-delete",
            binary: "az",
            extra_binaries: &[],
            arg_predicate: cloud_delete,
            reason: "az delete-* / vm delete / group delete",
            default_scope: PatternScope::AlwaysReal,
            reversibility: Reversibility::Irreversible,
        },
        DestructivePattern {
            id: "paas-destructive",
            binary: "railway",
            extra_binaries: &["fly", "vercel", "heroku", "render", "northflank", "koyeb"],
            arg_predicate: paas_destructive,
            reason: "PaaS destructive: railway down / fly destroy / vercel rm / heroku apps:destroy",
            default_scope: PatternScope::AlwaysReal,
            reversibility: Reversibility::Irreversible,
        },
        DestructivePattern {
            id: "powershell-remove-item",
            binary: "Remove-Item",
            extra_binaries: &["rm", "del", "ri"],
            arg_predicate: powershell_destructive,
            reason: "Remove-Item -Force -Recurse (PowerShell)",
            default_scope: PatternScope::PathDependent,
            reversibility: Reversibility::Lossy,
        },
        DestructivePattern {
            id: "redis-destructive",
            binary: "redis-cli",
            extra_binaries: &[],
            arg_predicate: redis_destructive,
            reason: "redis-cli FLUSHALL / FLUSHDB / SHUTDOWN / DEBUG",
            default_scope: PatternScope::AlwaysReal,
            reversibility: Reversibility::Irreversible,
        },
        DestructivePattern {
            id: "mongo-destructive",
            binary: "mongorestore",
            extra_binaries: &["mongo", "mongosh", "mongodump"],
            arg_predicate: mongo_destructive,
            reason: "mongorestore --drop / dropDatabase",
            default_scope: PatternScope::AlwaysReal,
            reversibility: Reversibility::Lossy,
        },
        DestructivePattern {
            id: "sql-destructive",
            binary: "psql",
            extra_binaries: &["mysql", "sqlite3", "cockroach"],
            arg_predicate: sql_destructive,
            reason: "SQL DROP/TRUNCATE/DELETE-without-WHERE/DROP-COLUMN/DROP-SCHEMA",
            default_scope: PatternScope::PathDependent,
            reversibility: Reversibility::Lossy,
        },
        DestructivePattern {
            id: "docker-destructive",
            binary: "docker",
            extra_binaries: &["podman", "ctr", "crictl"],
            arg_predicate: |argv| {
                argv.iter().any(|a| {
                    matches!(
                        a.as_str(),
                        "rm" | "rmi" | "system" | "image" | "volume" | "network" | "container"
                    )
                }) && argv.iter().any(|a| {
                    matches!(a.as_str(), "prune" | "rm" | "rmi" | "remove")
                })
            },
            reason: "docker rm / rmi / system prune / volume rm",
            default_scope: PatternScope::PathDependent,
            reversibility: Reversibility::Lossy,
        },
        DestructivePattern {
            id: "dd-of-device",
            binary: "dd",
            extra_binaries: &[],
            arg_predicate: |argv| {
                argv.iter().any(|a| a.starts_with("of=") && {
                    let target = &a[3..];
                    target.starts_with("/dev/")
                        || target == "/dev/sda"
                        || target == "/dev/nvme0n1"
                        || target.starts_with("/dev/xvd")
                })
            },
            reason: "dd of=/dev/* overwrites a block device",
            default_scope: PatternScope::AlwaysReal,
            reversibility: Reversibility::Irreversible,
        },
        DestructivePattern {
            id: "shred",
            binary: "shred",
            extra_binaries: &["wipe", "srm"],
            arg_predicate: always_true,
            reason: "secure-erase utility",
            default_scope: PatternScope::PathDependent,
            reversibility: Reversibility::Irreversible,
        },
        DestructivePattern {
            id: "mkfs",
            binary: "mkfs",
            extra_binaries: &[
                "mkfs.ext4",
                "mkfs.ext3",
                "mkfs.xfs",
                "mkfs.btrfs",
                "mkfs.f2fs",
                "mkfs.vfat",
                "mkfs.fat",
                "mke2fs",
                "newfs",
            ],
            arg_predicate: always_true,
            reason: "filesystem creation reformats a block device",
            default_scope: PatternScope::AlwaysReal,
            reversibility: Reversibility::Irreversible,
        },
        DestructivePattern {
            id: "systemctl-destructive",
            binary: "systemctl",
            extra_binaries: &["service"],
            arg_predicate: |argv| {
                matches!(
                    argv.first().map(String::as_str),
                    Some("stop" | "disable" | "mask" | "kill" | "reset-failed" | "poweroff" | "reboot" | "halt")
                )
            },
            reason: "systemctl stop / disable / kill / poweroff",
            default_scope: PatternScope::AlwaysReal,
            reversibility: Reversibility::Snapshot,
        },
        DestructivePattern {
            id: "process-tree-kill",
            binary: "kill",
            extra_binaries: &["pkill", "killall"],
            arg_predicate: |argv| {
                argv.iter().any(|a| {
                    matches!(a.as_str(), "-9" | "-KILL" | "-15" | "-TERM" | "-1" | "-HUP")
                }) || argv.iter().any(|a| a == "init" || a == "systemd" || a == "1")
            },
            reason: "kill -9 init / kill -HUP / pkill broad pattern",
            default_scope: PatternScope::PathDependent,
            reversibility: Reversibility::Snapshot,
        },
        DestructivePattern {
            id: "chown-recursive-root",
            binary: "chown",
            extra_binaries: &["chmod"],
            arg_predicate: |argv| {
                argv.iter().any(|a| a == "-R" || a == "--recursive")
                    && argv.iter().any(|a| a == "/" || a == "/etc" || a == "/usr")
            },
            reason: "recursive ownership/permission change rooted at / or /etc or /usr",
            default_scope: PatternScope::AlwaysReal,
            reversibility: Reversibility::Lossy,
        },
        DestructivePattern {
            id: "iptables-flush",
            binary: "iptables",
            extra_binaries: &["nft", "ip6tables"],
            arg_predicate: |argv| {
                argv.iter().any(|a| matches!(a.as_str(), "-F" | "--flush" | "-X" | "-Z"))
            },
            reason: "iptables flush / delete chain",
            default_scope: PatternScope::AlwaysReal,
            reversibility: Reversibility::Snapshot,
        },
    ]
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn corpus_is_non_empty() {
        assert!(!CORPUS.is_empty());
    }

    #[test]
    fn corpus_ids_are_unique() {
        let mut seen = std::collections::HashSet::new();
        for p in CORPUS.iter() {
            assert!(seen.insert(p.id), "duplicate id: {}", p.id);
        }
    }

    #[test]
    fn rm_recursive_predicate() {
        assert!(rm_recursive(&["-rf".into(), "foo".into()]));
        assert!(rm_recursive(&["-r".into(), "foo".into()]));
        assert!(rm_recursive(&["--recursive".into(), "foo".into()]));
        // Bare `rm foo` is also flagged — destructive proposal at the gate
        // decides whether to auto-approve based on scope.
        assert!(rm_recursive(&["foo".into()]));
        // Empty argv is not destructive (no path target).
        assert!(!rm_recursive(&[]));
    }

    #[test]
    fn git_destructive_predicate() {
        assert!(git_destructive(&["push".into(), "--force".into()]));
        assert!(git_destructive(&["push".into(), "-f".into()]));
        assert!(git_destructive(&["push".into(), "origin".into(), "+main".into()]));
        assert!(git_destructive(&["reset".into(), "--hard".into()]));
        assert!(git_destructive(&["clean".into(), "-fd".into()]));
        assert!(git_destructive(&["branch".into(), "-D".into(), "feature".into()]));
        assert!(git_destructive(&["filter-repo".into()]));
        // Benign git commands.
        assert!(!git_destructive(&["status".into()]));
        assert!(!git_destructive(&["push".into(), "origin".into(), "main".into()]));
    }

    #[test]
    fn kubectl_destructive_predicate() {
        assert!(kubectl_destructive(&["delete".into(), "pod".into(), "foo".into()]));
        assert!(kubectl_destructive(&["drain".into(), "node1".into()]));
        assert!(kubectl_destructive(&["scale".into(), "deployment/foo".into(), "--replicas=0".into()]));
        assert!(!kubectl_destructive(&["get".into(), "pods".into()]));
    }

    #[test]
    fn paas_destructive_predicate() {
        assert!(paas_destructive(&["down".into()]));
        assert!(paas_destructive(&["destroy".into()]));
        assert!(paas_destructive(&["rm".into(), "deploy".into()]));
        assert!(paas_destructive(&["apps:destroy".into()]));
        assert!(!paas_destructive(&["status".into()]));
    }

    #[test]
    fn cloud_delete_predicate() {
        assert!(cloud_delete(&["s3".into(), "rm".into(), "s3://bucket".into()]));
        assert!(cloud_delete(&["ec2".into(), "terminate-instances".into()]));
        assert!(cloud_delete(&["iam".into(), "delete-user".into()]));
        assert!(!cloud_delete(&["ec2".into(), "describe-instances".into()]));
    }

    #[test]
    fn sql_destructive_predicate() {
        assert!(sql_destructive(&["-c".into(), "DROP TABLE users".into()]));
        assert!(sql_destructive(&["-c".into(), "TRUNCATE charges".into()]));
        assert!(sql_destructive(&["-c".into(), "DELETE FROM users".into()]));
        // DELETE with WHERE clause is allowed — it's an op-by-op decision, not destructive on its face.
        assert!(!sql_destructive(&["-c".into(), "DELETE FROM users WHERE id = 1".into()]));
        assert!(!sql_destructive(&["-c".into(), "SELECT * FROM users".into()]));
    }

    #[test]
    fn dd_predicate() {
        let p = CORPUS.iter().find(|p| p.id == "dd-of-device").unwrap();
        assert!((p.arg_predicate)(&["if=/dev/zero".into(), "of=/dev/sda".into()]));
        assert!((p.arg_predicate)(&["of=/dev/nvme0n1".into()]));
        // Writing to a regular file is allowed.
        assert!(!(p.arg_predicate)(&["if=foo".into(), "of=bar".into()]));
    }

    #[test]
    fn redis_destructive_predicate() {
        assert!(redis_destructive(&["FLUSHALL".into()]));
        assert!(redis_destructive(&["-p".into(), "6379".into(), "FLUSHDB".into()]));
        assert!(redis_destructive(&["shutdown".into()])); // case-insensitive
        assert!(!redis_destructive(&["GET".into(), "foo".into()]));
    }

    #[test]
    fn iptables_flush_predicate() {
        let p = CORPUS.iter().find(|p| p.id == "iptables-flush").unwrap();
        assert!((p.arg_predicate)(&["-F".into()]));
        assert!((p.arg_predicate)(&["--flush".into(), "INPUT".into()]));
        assert!(!(p.arg_predicate)(&["-L".into()])); // listing is fine
    }
}
