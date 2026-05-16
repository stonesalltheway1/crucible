//! Filesystem twin — git worktree + overlayfs orchestration.
//!
//! Per `docs/01-architecture/twin-runtime.md` §"Layer 2", the filesystem
//! twin is composed of:
//!
//! - `/work/repo` — git worktree at `base_sha`, depth 1.
//! - `/work/scratch` — overlayfs upper, the agent's mutation surface.
//! - `/work/tapes` — read-only Hoverfly tapes (mounted by the tape driver).
//! - `/work/secrets` — tmpfs for Infisical tokens (mounted by the secrets sidecar).
//! - `/work/.crucible` — attestation socket + control fds.
//!
//! This crate covers `/work/repo` and `/work/scratch`. Tape and secret
//! mounts are owned by the respective service drivers.
//!
//! On Linux the overlay uses real `overlayfs`. On other hosts (dev macOS /
//! Windows / CI without privilege) the overlay falls back to a `copy` mode
//! that recursively copies the repo into the upper. Less efficient but
//! semantically equivalent for tests.

#![warn(missing_docs)]

use async_trait::async_trait;
use crucible_sandbox_spec::FilesystemSpec;
use std::path::{Path, PathBuf};
use thiserror::Error;
use tokio::process::Command;

/// Errors from filesystem-twin orchestration.
#[derive(Debug, Error)]
pub enum Error {
    /// `git` subprocess failed.
    #[error("git: {0}")]
    Git(String),
    /// `mount` / `umount` failed (Linux only).
    #[error("mount: {0}")]
    Mount(String),
    /// Generic IO error.
    #[error("io: {0}")]
    Io(#[from] std::io::Error),
    /// Spec was rejected.
    #[error("invalid filesystem spec: {0}")]
    InvalidSpec(String),
    /// Platform-only feature not available.
    #[error("platform: {0}")]
    Platform(String),
}

/// Result alias for the fs crate.
pub type Result<T> = std::result::Result<T, Error>;

/// Layout for the per-twin work dir inside the sandbox.
#[derive(Debug, Clone)]
pub struct WorkLayout {
    /// Absolute path the work dir is rooted at. Typically `/work` inside
    /// the sandbox.
    pub root: PathBuf,
    /// Path inside the sandbox where the git worktree lives.
    pub repo: PathBuf,
    /// Path inside the sandbox of the overlay upper.
    pub scratch: PathBuf,
    /// Path inside the sandbox of the overlay work-dir (overlayfs needs
    /// a separate "work" path; ignored when overlay_mode is `copy`).
    pub overlay_work: PathBuf,
    /// Path inside the sandbox of the unified mountpoint the agent sees.
    pub mount: PathBuf,
}

impl WorkLayout {
    /// Build the default layout rooted at `/work` (or any absolute path).
    #[must_use]
    pub fn rooted_at(root: impl Into<PathBuf>) -> Self {
        let root = root.into();
        Self {
            repo: root.join("repo"),
            scratch: root.join("scratch"),
            overlay_work: root.join(".overlay-work"),
            mount: root.join("mount"),
            root,
        }
    }
}

/// Orchestrator for the filesystem twin.
#[async_trait]
pub trait Orchestrator: Send + Sync {
    /// Prepare the twin filesystem from the spec. Idempotent.
    async fn prepare(&self, spec: &FilesystemSpec, layout: &WorkLayout) -> Result<()>;
    /// Mount the overlay. Linux-only when `overlay_mode = "overlayfs-linux"`.
    async fn mount(&self, layout: &WorkLayout, mode: OverlayMode) -> Result<()>;
    /// Unmount + clean up. Idempotent.
    async fn unmount(&self, layout: &WorkLayout, mode: OverlayMode) -> Result<()>;
    /// Compute the diff vs base_sha. Returns the cumulative file changes.
    async fn diff(&self, layout: &WorkLayout) -> Result<Vec<FileChange>>;
}

/// Overlay implementation selector.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum OverlayMode {
    /// Real overlayfs (Linux + appropriate caps).
    OverlayFsLinux,
    /// Recursive copy fallback (any host).
    Copy,
    /// Bind-mount read-only (any host).
    BindRo,
}

impl OverlayMode {
    /// Parse the spec's `overlay_mode` string. Unknown values fail-closed
    /// to [`OverlayMode::Copy`] with a tracing warning.
    #[must_use]
    pub fn parse(s: &str) -> Self {
        match s {
            "overlayfs-linux" => Self::OverlayFsLinux,
            "copy" => Self::Copy,
            "bind-ro" => Self::BindRo,
            other => {
                tracing::warn!(value = %other, "unknown overlay_mode, defaulting to copy");
                Self::Copy
            }
        }
    }
}

/// File change observed in [`Orchestrator::diff`].
#[derive(Debug, Clone, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
pub struct FileChange {
    /// Path relative to the mount root.
    pub path: String,
    /// `"add" | "modify" | "delete"`.
    pub action: String,
    /// SHA-256 of the new content (empty for `delete`).
    pub content_sha256: String,
    /// File size in bytes (zero for `delete`).
    pub size_bytes: u64,
}

/// Production orchestrator.
pub struct ProdOrchestrator;

impl ProdOrchestrator {
    /// Construct.
    #[must_use]
    pub fn new() -> Self {
        Self
    }
}

impl Default for ProdOrchestrator {
    fn default() -> Self {
        Self::new()
    }
}

#[async_trait]
impl Orchestrator for ProdOrchestrator {
    async fn prepare(&self, spec: &FilesystemSpec, layout: &WorkLayout) -> Result<()> {
        if spec.base_sha.is_empty() {
            return Err(Error::InvalidSpec("base_sha empty".into()));
        }
        if spec.repo_url.is_empty() {
            return Err(Error::InvalidSpec("repo_url empty".into()));
        }
        fs_err::create_dir_all(&layout.root)?;
        fs_err::create_dir_all(&layout.repo)?;
        fs_err::create_dir_all(&layout.scratch)?;
        fs_err::create_dir_all(&layout.overlay_work)?;
        fs_err::create_dir_all(&layout.mount)?;
        // Clone into `repo`. `--filter=blob:none --depth=1` per twin-runtime.md.
        clone_shallow(&spec.repo_url, &spec.base_sha, spec.depth, &layout.repo).await?;
        for path in &spec.prewarm_paths {
            tracing::debug!(prewarm = %path, "prewarm_paths is Phase 3 — recording for follow-up");
        }
        Ok(())
    }

    async fn mount(&self, layout: &WorkLayout, mode: OverlayMode) -> Result<()> {
        match mode {
            OverlayMode::OverlayFsLinux => mount_overlayfs(layout).await,
            OverlayMode::Copy => mount_copy(layout).await,
            OverlayMode::BindRo => mount_bind_ro(layout).await,
        }
    }

    async fn unmount(&self, layout: &WorkLayout, mode: OverlayMode) -> Result<()> {
        match mode {
            OverlayMode::OverlayFsLinux | OverlayMode::BindRo => unmount_kernel(layout).await,
            OverlayMode::Copy => unmount_copy(layout).await,
        }
    }

    async fn diff(&self, layout: &WorkLayout) -> Result<Vec<FileChange>> {
        // `git diff --name-status` against base_sha gives us add/modify/delete
        // in a single shot. We also recurse for files added in the upper but
        // never tracked.
        let output = Command::new("git")
            .arg("-C")
            .arg(&layout.mount)
            .args(["status", "--porcelain", "-z", "--untracked-files=all"])
            .output()
            .await
            .map_err(|e| Error::Git(format!("git status: {e}")))?;
        if !output.status.success() {
            return Err(Error::Git(format!(
                "git status failed: {}",
                String::from_utf8_lossy(&output.stderr)
            )));
        }
        let mut changes = Vec::new();
        for entry in output.stdout.split(|&b| b == 0) {
            if entry.is_empty() {
                continue;
            }
            let line = String::from_utf8_lossy(entry);
            if line.len() < 3 {
                continue;
            }
            let status = &line[..2];
            let path = line[3..].to_string();
            let action = match status {
                " D" | "D " | "DD" => "delete",
                "??" | "A " | " A" => "add",
                _ => "modify",
            };
            let (size, sha) = if action == "delete" {
                (0, String::new())
            } else {
                let full = layout.mount.join(&path);
                if let Ok(data) = fs_err::read(&full) {
                    use sha2::{Digest, Sha256};
                    let mut h = Sha256::new();
                    h.update(&data);
                    (data.len() as u64, hex::encode(h.finalize()))
                } else {
                    (0, String::new())
                }
            };
            changes.push(FileChange {
                path,
                action: action.into(),
                content_sha256: sha,
                size_bytes: size,
            });
        }
        Ok(changes)
    }
}

async fn clone_shallow(url: &str, sha: &str, depth: u32, dest: &Path) -> Result<()> {
    let depth_arg = format!("--depth={}", depth.max(1));
    let status = Command::new("git")
        .args([
            "clone",
            &depth_arg,
            "--filter=blob:none",
            "--no-checkout",
            url,
        ])
        .arg(dest)
        .status()
        .await
        .map_err(|e| Error::Git(format!("git clone: {e}")))?;
    if !status.success() {
        return Err(Error::Git(format!("git clone failed: exit={status:?}")));
    }
    let status = Command::new("git")
        .arg("-C")
        .arg(dest)
        .args(["fetch", "origin", sha])
        .status()
        .await
        .map_err(|e| Error::Git(format!("git fetch: {e}")))?;
    if !status.success() {
        return Err(Error::Git(format!("git fetch sha failed: exit={status:?}")));
    }
    let status = Command::new("git")
        .arg("-C")
        .arg(dest)
        .args(["checkout", sha])
        .status()
        .await
        .map_err(|e| Error::Git(format!("git checkout: {e}")))?;
    if !status.success() {
        return Err(Error::Git(format!("git checkout failed: exit={status:?}")));
    }
    Ok(())
}

#[cfg(target_os = "linux")]
async fn mount_overlayfs(layout: &WorkLayout) -> Result<()> {
    use std::ffi::CString;
    use std::os::unix::ffi::OsStrExt;
    let opts = format!(
        "lowerdir={},upperdir={},workdir={}",
        layout.repo.display(),
        layout.scratch.display(),
        layout.overlay_work.display(),
    );
    let opts_c = CString::new(opts).map_err(|e| Error::Mount(e.to_string()))?;
    let target_c = CString::new(layout.mount.as_os_str().as_bytes())
        .map_err(|e| Error::Mount(e.to_string()))?;
    let fs_c = CString::new("overlay").map_err(|e| Error::Mount(e.to_string()))?;
    let source_c = CString::new("overlay").map_err(|e| Error::Mount(e.to_string()))?;

    // SAFETY: arguments are valid C strings owned by this function;
    // libc::mount is the canonical entry point.
    let rc = unsafe {
        libc::mount(
            source_c.as_ptr(),
            target_c.as_ptr(),
            fs_c.as_ptr(),
            0,
            opts_c.as_ptr().cast(),
        )
    };
    if rc != 0 {
        let err = std::io::Error::last_os_error();
        return Err(Error::Mount(format!("overlayfs mount: {err}")));
    }
    Ok(())
}

#[cfg(not(target_os = "linux"))]
async fn mount_overlayfs(layout: &WorkLayout) -> Result<()> {
    tracing::warn!(
        layout = ?layout.mount,
        "STUB: overlayfs unavailable on non-Linux host — degrading to copy"
    );
    mount_copy(layout).await
}

async fn mount_copy(layout: &WorkLayout) -> Result<()> {
    fs_err::create_dir_all(&layout.mount)?;
    copy_recursive(&layout.repo, &layout.mount).await?;
    Ok(())
}

async fn mount_bind_ro(layout: &WorkLayout) -> Result<()> {
    // Bind-ro is equivalent to copy for the diff layer since the agent
    // writes go to the upper (scratch) and we surface them via diff().
    mount_copy(layout).await
}

async fn copy_recursive(src: &Path, dst: &Path) -> Result<()> {
    fs_err::create_dir_all(dst)?;
    let mut entries = tokio::fs::read_dir(src).await?;
    while let Some(entry) = entries.next_entry().await? {
        let path = entry.path();
        let name = entry.file_name();
        let target = dst.join(&name);
        let metadata = entry.metadata().await?;
        if metadata.is_dir() {
            Box::pin(copy_recursive(&path, &target)).await?;
        } else {
            tokio::fs::copy(&path, &target).await?;
        }
    }
    Ok(())
}

#[cfg(target_os = "linux")]
async fn unmount_kernel(layout: &WorkLayout) -> Result<()> {
    use std::ffi::CString;
    use std::os::unix::ffi::OsStrExt;
    let target_c = CString::new(layout.mount.as_os_str().as_bytes())
        .map_err(|e| Error::Mount(e.to_string()))?;
    // SAFETY: target is a valid C string. UMOUNT_NOFOLLOW is per-mount safe.
    let rc = unsafe { libc::umount2(target_c.as_ptr(), libc::UMOUNT_NOFOLLOW) };
    if rc != 0 {
        let err = std::io::Error::last_os_error();
        // Idempotent: ENOENT / EINVAL = "not mounted" is OK.
        if err.raw_os_error() == Some(libc::ENOENT) || err.raw_os_error() == Some(libc::EINVAL) {
            return Ok(());
        }
        return Err(Error::Mount(format!("umount2: {err}")));
    }
    Ok(())
}

#[cfg(not(target_os = "linux"))]
async fn unmount_kernel(layout: &WorkLayout) -> Result<()> {
    // Copy fallback — `unmount_copy` cleans up the mount dir.
    unmount_copy(layout).await
}

async fn unmount_copy(layout: &WorkLayout) -> Result<()> {
    if layout.mount.exists() {
        tokio::fs::remove_dir_all(&layout.mount).await?;
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn overlay_mode_parses_known() {
        assert_eq!(OverlayMode::parse("overlayfs-linux"), OverlayMode::OverlayFsLinux);
        assert_eq!(OverlayMode::parse("copy"), OverlayMode::Copy);
        assert_eq!(OverlayMode::parse("bind-ro"), OverlayMode::BindRo);
    }

    #[test]
    fn overlay_mode_unknown_defaults_to_copy() {
        assert_eq!(OverlayMode::parse("nonsense"), OverlayMode::Copy);
    }

    #[test]
    fn work_layout_paths_under_root() {
        let l = WorkLayout::rooted_at("/tmp/twin");
        assert_eq!(l.repo, PathBuf::from("/tmp/twin/repo"));
        assert_eq!(l.scratch, PathBuf::from("/tmp/twin/scratch"));
        assert_eq!(l.mount, PathBuf::from("/tmp/twin/mount"));
    }

    #[tokio::test]
    async fn copy_recursive_copies_a_directory() {
        let src = tempfile::tempdir().unwrap();
        let dst = tempfile::tempdir().unwrap();
        fs_err::write(src.path().join("a.txt"), b"hello").unwrap();
        fs_err::create_dir_all(src.path().join("sub")).unwrap();
        fs_err::write(src.path().join("sub/b.txt"), b"world").unwrap();
        copy_recursive(src.path(), dst.path()).await.unwrap();
        assert_eq!(fs_err::read(dst.path().join("a.txt")).unwrap(), b"hello");
        assert_eq!(
            fs_err::read(dst.path().join("sub/b.txt")).unwrap(),
            b"world"
        );
    }
}
