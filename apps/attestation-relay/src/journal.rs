//! Hash-chained append-only JSONL journal.
//!
//! Format — one JSON object per line:
//!
//! ```text
//!   { "uuid":     "<sha256 of prev || envBytes>",
//!     "prev":     "<sha256 of prior entry or 64 zeros>",
//!     "ts":       "<RFC 3339Nano>",
//!     "index":    <u64>,
//!     "rekor_uuid": "<populated after back-fill>",
//!     "envelope": { ... } }
//! ```
//!
//! Schema-compatible with the Phase-1 Go `LocalJournalPublisher` — the
//! Rust relay reads and writes the same file format so the journal stays
//! coherent across phases.

use serde::{Deserialize, Serialize};
use std::io::{BufRead, BufReader, Write};
use std::path::{Path, PathBuf};
use std::sync::Mutex;
use tokio::sync::Mutex as AsyncMutex;

use crate::dsse::DsseEnvelope;
use crate::error::{Error, Result};
use crate::rekor::RekorEntry;

/// One JSONL line in the journal.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct JournalEntry {
    /// SHA-256 of `prev || canonical(envelope)` — content-addressed.
    pub uuid: String,
    /// Previous UUID (sixty-four zeros for the genesis entry).
    pub prev: String,
    /// RFC 3339 timestamp.
    pub ts: String,
    /// Monotonic index.
    pub index: u64,
    /// Rekor UUID once back-fill succeeds; empty otherwise.
    #[serde(default)]
    pub rekor_uuid: String,
    /// Embedded envelope.
    pub envelope: DsseEnvelope,
}

const ZERO_PREV: &str = "0000000000000000000000000000000000000000000000000000000000000000";

/// Hash-chained JSONL journal. Thread-safe + safe for concurrent
/// `append_envelope` calls.
#[derive(Debug)]
pub struct Journal {
    path: PathBuf,
    writer_lock: AsyncMutex<()>,
    inner: Mutex<JournalCache>,
}

#[derive(Debug, Default)]
struct JournalCache {
    last_uuid: Option<String>,
    last_index: u64,
}

impl Journal {
    /// Open or create the journal at `path`. The parent directory is
    /// created if missing.
    pub fn open(path: impl Into<PathBuf>) -> Result<Self> {
        let path = path.into();
        if let Some(parent) = path.parent() {
            std::fs::create_dir_all(parent)?;
        }
        // Touch the file so subsequent appends don't surprise us.
        if !path.exists() {
            std::fs::write(&path, b"")?;
        }
        let cache = Self::scan_to_tail(&path)?;
        Ok(Self {
            path,
            writer_lock: AsyncMutex::new(()),
            inner: Mutex::new(cache),
        })
    }

    /// Append a DSSE envelope. Returns a RekorEntry receipt flagged
    /// `local_journal_fallback = true`.
    pub async fn append_envelope(&self, env: &DsseEnvelope) -> Result<RekorEntry> {
        let _guard = self.writer_lock.lock().await;

        let env_bytes = serde_json::to_vec(env)?;

        // Resolve prev + new index.
        let (prev, next_idx) = {
            let cache = self.inner.lock().unwrap();
            (
                cache.last_uuid.clone().unwrap_or_else(|| ZERO_PREV.into()),
                cache.last_index + 1,
            )
        };

        // UUID = sha256(prev || canonical-envelope).
        let new_uuid = {
            use sha2::{Digest, Sha256};
            let mut h = Sha256::new();
            h.update(prev.as_bytes());
            h.update(&env_bytes);
            hex::encode(h.finalize())
        };

        let ts = chrono::Utc::now().to_rfc3339_opts(chrono::SecondsFormat::Nanos, true);

        let entry = JournalEntry {
            uuid: new_uuid.clone(),
            prev: prev.clone(),
            ts: ts.clone(),
            index: next_idx,
            rekor_uuid: String::new(),
            envelope: env.clone(),
        };
        let line = serde_json::to_vec(&entry)?;

        let mut f = std::fs::OpenOptions::new()
            .create(true)
            .append(true)
            .open(&self.path)
            .map_err(|e| Error::Journal(format!("open: {e}")))?;
        f.write_all(&line)?;
        f.write_all(b"\n")?;
        f.sync_data()?;

        {
            let mut cache = self.inner.lock().unwrap();
            cache.last_uuid = Some(new_uuid.clone());
            cache.last_index = next_idx;
        }

        Ok(RekorEntry {
            uuid: new_uuid,
            log_index: next_idx.to_string(),
            log_id: "local-journal".into(),
            integrated_time: ts,
            url: format!("file://{}#{}", self.path.display(), entry.uuid),
            local_journal_fallback: true,
            self_hosted: false,
        })
    }

    /// Read an entry by uuid.
    pub fn fetch(&self, uuid: &str) -> Result<DsseEnvelope> {
        let _g = self.writer_lock.try_lock();
        let f = std::fs::File::open(&self.path).map_err(|e| Error::Journal(format!("open: {e}")))?;
        let reader = BufReader::new(f);
        for line in reader.lines() {
            let line = line?;
            if line.is_empty() {
                continue;
            }
            let e: JournalEntry = match serde_json::from_str(&line) {
                Ok(v) => v,
                Err(err) => return Err(Error::Journal(format!("parse: {err}"))),
            };
            if e.uuid == uuid {
                return Ok(e.envelope);
            }
        }
        Err(Error::Journal(format!("uuid {uuid} not found")))
    }

    /// Validate the hash chain end-to-end. Returns the count of entries on
    /// success; returns Err on any chain break.
    pub fn validate_chain(&self) -> Result<usize> {
        let f = std::fs::File::open(&self.path).map_err(|e| Error::Journal(format!("open: {e}")))?;
        let reader = BufReader::new(f);
        let mut count = 0_usize;
        let mut prev: String = ZERO_PREV.into();
        for line in reader.lines() {
            let line = line?;
            if line.is_empty() {
                continue;
            }
            let e: JournalEntry = serde_json::from_str(&line)
                .map_err(|err| Error::Journal(format!("parse: {err}")))?;
            if e.prev != prev {
                return Err(Error::Journal(format!(
                    "chain break at index {}: expected prev={} got prev={}",
                    e.index, prev, e.prev
                )));
            }
            // Recompute UUID = sha256(prev || canonical(envelope)).
            let env_bytes = serde_json::to_vec(&e.envelope)?;
            use sha2::{Digest, Sha256};
            let mut h = Sha256::new();
            h.update(e.prev.as_bytes());
            h.update(&env_bytes);
            let expected = hex::encode(h.finalize());
            if expected != e.uuid {
                return Err(Error::Journal(format!(
                    "uuid mismatch at index {}: expected {} got {}",
                    e.index, expected, e.uuid
                )));
            }
            prev = e.uuid.clone();
            count += 1;
        }
        Ok(count)
    }

    /// List entries whose rekor_uuid is empty — candidates for back-fill.
    pub fn pending_backfill(&self, max: usize) -> Result<Vec<JournalEntry>> {
        let f = std::fs::File::open(&self.path).map_err(|e| Error::Journal(format!("open: {e}")))?;
        let reader = BufReader::new(f);
        let mut out = vec![];
        for line in reader.lines() {
            let line = line?;
            if line.is_empty() {
                continue;
            }
            let e: JournalEntry = serde_json::from_str(&line)
                .map_err(|err| Error::Journal(format!("parse: {err}")))?;
            if e.rekor_uuid.is_empty() {
                out.push(e);
                if out.len() >= max {
                    break;
                }
            }
        }
        Ok(out)
    }

    /// Mark a journal entry as having been back-filled. Rewrites the file
    /// in place. Phase-6 implementation is simple — small journals; v2
    /// switches to an index file.
    pub async fn mark_backfilled(&self, journal_uuid: &str, rekor_uuid: &str) -> Result<()> {
        let _guard = self.writer_lock.lock().await;
        let f = std::fs::File::open(&self.path).map_err(|e| Error::Journal(format!("open: {e}")))?;
        let reader = BufReader::new(f);
        let mut entries = Vec::<JournalEntry>::new();
        for line in reader.lines() {
            let line = line?;
            if line.is_empty() {
                continue;
            }
            let mut e: JournalEntry = serde_json::from_str(&line)
                .map_err(|err| Error::Journal(format!("parse: {err}")))?;
            if e.uuid == journal_uuid {
                e.rekor_uuid = rekor_uuid.into();
            }
            entries.push(e);
        }
        let tmp = self.path.with_extension("jsonl.tmp");
        {
            let mut out = std::fs::File::create(&tmp)
                .map_err(|e| Error::Journal(format!("tmp create: {e}")))?;
            for e in &entries {
                let line = serde_json::to_vec(e)?;
                out.write_all(&line)?;
                out.write_all(b"\n")?;
            }
            out.sync_data()?;
        }
        std::fs::rename(&tmp, &self.path)
            .map_err(|e| Error::Journal(format!("rename: {e}")))?;
        Ok(())
    }

    /// Number of entries.
    pub fn len(&self) -> usize {
        // Cheap counter — we cache last index.
        let cache = self.inner.lock().unwrap();
        cache.last_index as usize
    }

    /// Empty?
    pub fn is_empty(&self) -> bool {
        self.len() == 0
    }

    /// Path on disk.
    #[must_use]
    pub fn path(&self) -> &Path {
        &self.path
    }

    fn scan_to_tail(path: &Path) -> Result<JournalCache> {
        let f = std::fs::File::open(path).map_err(|e| Error::Journal(format!("open: {e}")))?;
        let reader = BufReader::new(f);
        let mut last_uuid: Option<String> = None;
        let mut last_index: u64 = 0;
        for line in reader.lines() {
            let line = line?;
            if line.is_empty() {
                continue;
            }
            let e: JournalEntry = serde_json::from_str(&line)
                .map_err(|err| Error::Journal(format!("parse: {err}")))?;
            last_uuid = Some(e.uuid);
            last_index = e.index;
        }
        Ok(JournalCache { last_uuid, last_index })
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::dsse::DsseSignature;

    fn synthetic_envelope(payload: &str) -> DsseEnvelope {
        let mut e = DsseEnvelope::new(payload.as_bytes());
        e.signatures.push(DsseSignature {
            keyid: "test".into(),
            sig: base64::engine::general_purpose::STANDARD.encode([0u8; 64]),
            cert: None,
        });
        e
    }

    #[tokio::test]
    async fn append_and_fetch() {
        let dir = tempfile::tempdir().unwrap();
        let j = Journal::open(dir.path().join("j.jsonl")).unwrap();
        let env = synthetic_envelope("hello");
        let r = j.append_envelope(&env).await.unwrap();
        assert!(r.local_journal_fallback);
        let got = j.fetch(&r.uuid).unwrap();
        assert_eq!(got.payload, env.payload);
    }

    #[tokio::test]
    async fn chain_validates() {
        let dir = tempfile::tempdir().unwrap();
        let j = Journal::open(dir.path().join("j.jsonl")).unwrap();
        for i in 0..5 {
            j.append_envelope(&synthetic_envelope(&format!("e-{i}"))).await.unwrap();
        }
        let count = j.validate_chain().unwrap();
        assert_eq!(count, 5);
    }

    #[tokio::test]
    async fn chain_detects_tamper() {
        let dir = tempfile::tempdir().unwrap();
        let path = dir.path().join("j.jsonl");
        let j = Journal::open(&path).unwrap();
        for i in 0..3 {
            j.append_envelope(&synthetic_envelope(&format!("e-{i}"))).await.unwrap();
        }
        // Tamper: replace an entry with a different payload, then validate.
        let s = std::fs::read_to_string(&path).unwrap();
        let mut lines: Vec<String> = s.lines().map(str::to_string).collect();
        // Tamper middle line's envelope but keep the uuid/prev.
        let mut e: JournalEntry = serde_json::from_str(&lines[1]).unwrap();
        e.envelope = synthetic_envelope("tampered");
        lines[1] = serde_json::to_string(&e).unwrap();
        std::fs::write(&path, lines.join("\n") + "\n").unwrap();
        let res = j.validate_chain();
        assert!(res.is_err(), "tamper must be detected, got {res:?}");
    }
}
