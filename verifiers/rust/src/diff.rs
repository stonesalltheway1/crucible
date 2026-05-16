//! Helpers for extracting Rust-relevant information from a
//! `VerificationRequest.diff`.
//!
//! The dispatcher hands the runner a `Diff` carrying per-file unified
//! patches. The runner needs three views of that data:
//!
//! 1. A unified-diff blob it can write to disk and feed to
//!    `cargo mutants --in-diff <path>`.
//! 2. The list of touched `.rs` paths so it can scope greps for things
//!    like `#[kani::proof]` harnesses.
//! 3. The contents of each touched file's `unified_diff` so it can
//!    string-scan for tier-3 annotations even when the rest of the file
//!    is unavailable.

use crate::schema::{Diff, FileChange};

/// One Rust file touched by the diff.
#[derive(Debug, Clone)]
pub struct RustFile<'a> {
    /// Repo-relative path (forward slashes, as written in the request).
    pub path: &'a str,
    /// Unified diff body — may be empty if the dispatcher omitted it.
    pub unified_diff: &'a str,
    /// Coarse status (`added`, `modified`, `deleted`, …).
    pub status: &'a str,
}

/// Iterator over Rust files in the diff. Filters by `.rs` extension and
/// strips files whose path was rewritten to a non-Rust file.
pub fn rust_files(diff: &Diff) -> impl Iterator<Item = RustFile<'_>> {
    diff.files.iter().filter_map(file_change_as_rust)
}

fn file_change_as_rust(f: &FileChange) -> Option<RustFile<'_>> {
    if !is_rust_path(&f.path) {
        return None;
    }
    Some(RustFile {
        path: &f.path,
        unified_diff: &f.unified_diff,
        status: &f.status,
    })
}

/// True for any `.rs` source — we keep the check simple on purpose so
/// generated files (`build.rs`) are also covered.
pub fn is_rust_path(path: &str) -> bool {
    path.ends_with(".rs")
}

/// Build a single concatenated unified-diff document for cargo-mutants
/// `--in-diff`. The file's `unified_diff` field is concatenated in the
/// dispatcher-provided order; if no per-file diffs are present we emit
/// a synthetic header so cargo-mutants sees at least the touched paths.
pub fn build_unified_diff(diff: &Diff) -> String {
    let mut out = String::new();
    for f in rust_files(diff) {
        if !f.unified_diff.is_empty() {
            // Trust the dispatcher's already-formatted hunk.
            if !out.is_empty() && !out.ends_with('\n') {
                out.push('\n');
            }
            out.push_str(f.unified_diff);
            if !out.ends_with('\n') {
                out.push('\n');
            }
        } else {
            // Synthetic "touched but empty" header. cargo-mutants will
            // treat the whole file as in-scope on a path match.
            use std::fmt::Write as _;
            let _ = write!(out, "--- a/{path}\n+++ b/{path}\n", path = f.path);
        }
    }
    out
}

/// Rust paths only — useful for greps and harness discovery.
pub fn rust_paths(diff: &Diff) -> Vec<String> {
    rust_files(diff).map(|f| f.path.to_string()).collect()
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::schema::FileChange;

    fn change(path: &str, body: &str) -> FileChange {
        FileChange {
            path: path.to_string(),
            unified_diff: body.to_string(),
            status: "modified".to_string(),
            ..Default::default()
        }
    }

    #[test]
    fn filters_non_rust_files() {
        let d = Diff {
            files: vec![
                change("src/lib.rs", "--- a/src/lib.rs\n+++ b/src/lib.rs\n"),
                change("docs/foo.md", "ignored"),
            ],
        };
        let paths: Vec<_> = rust_paths(&d);
        assert_eq!(paths, vec!["src/lib.rs".to_string()]);
    }

    #[test]
    fn synthesises_header_when_diff_missing() {
        let d = Diff {
            files: vec![change("src/a.rs", "")],
        };
        let unified = build_unified_diff(&d);
        assert!(unified.contains("--- a/src/a.rs"));
        assert!(unified.contains("+++ b/src/a.rs"));
    }

    #[test]
    fn preserves_provided_diff_bodies() {
        let body = "--- a/src/x.rs\n+++ b/src/x.rs\n@@ -1 +1 @@\n-old\n+new\n";
        let d = Diff {
            files: vec![change("src/x.rs", body)],
        };
        let unified = build_unified_diff(&d);
        assert!(unified.contains("@@ -1 +1 @@"));
    }
}
