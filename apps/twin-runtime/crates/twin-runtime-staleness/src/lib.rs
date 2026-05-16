//! Tape staleness tracker.
//!
//! Per docs/06-research/tape-coverage-strategy.md "Tape staleness — the
//! irreducible problem":
//!
//!   When upstream service ships a breaking change, tapes silently lie.
//!   Mitigations:
//!     1. Tape-age metrics surfaced to agents and the verifier
//!     2. Promotion canary catches lying tapes
//!     3. Auto-rollback on canary regression
//!     4. Periodic re-capture cron
//!     5. Tape staleness warning in PR descriptions
//!
//! This crate ships #1, #4, and #5. The verifier (Phase 4) consumes the
//! warning surface; the promotion canary (Phase 6) closes the loop.

#![forbid(unsafe_code)]

use chrono::{DateTime, Utc};
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::time::Duration;
use thiserror::Error;

/// One per-endpoint staleness record.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct EndpointFreshness {
    /// Canonical key (service|method|endpoint).
    pub key: String,
    /// When the tape was last refreshed (capture or shadow-recorder write).
    pub last_recorded: DateTime<Utc>,
    /// How frequently we expect this endpoint to be re-recorded.
    pub recapture_interval: Duration,
    /// Sample count seen since the last re-capture; used to weight
    /// "is this hot enough to worry about?" decisions.
    pub samples: u64,
}

/// Severity bucket for the verifier rubric to consume.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
pub enum StalenessBand {
    /// Within 1× interval — green.
    Fresh,
    /// 1–2× interval — caution; surface to agent reasoning.
    Aging,
    /// > 2× interval — stale; verifier weights down.
    Stale,
    /// No recording exists at all.
    Unrecorded,
}

impl StalenessBand {
    /// Short human label for dashboards and PR comments.
    #[must_use]
    pub const fn label(self) -> &'static str {
        match self {
            Self::Fresh => "fresh",
            Self::Aging => "aging",
            Self::Stale => "stale",
            Self::Unrecorded => "unrecorded",
        }
    }
}

/// Classifier — pure function of (now, last_recorded, interval).
#[must_use]
pub fn classify(
    last_recorded: Option<DateTime<Utc>>,
    interval: Duration,
    now: DateTime<Utc>,
) -> StalenessBand {
    let Some(ts) = last_recorded else {
        return StalenessBand::Unrecorded;
    };
    let age = now
        .signed_duration_since(ts)
        .to_std()
        .unwrap_or_else(|_| Duration::ZERO);
    if age <= interval {
        StalenessBand::Fresh
    } else if age <= interval * 2 {
        StalenessBand::Aging
    } else {
        StalenessBand::Stale
    }
}

/// Per-task staleness report surfaced to the agent + verifier.
#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct StalenessReport {
    /// Per-endpoint findings keyed by canonical key.
    pub findings: Vec<Finding>,
    /// Aggregate counts across the task's tape hits.
    pub counts: StalenessCounts,
    /// Generated-at timestamp.
    pub generated_at: DateTime<Utc>,
}

/// One staleness finding.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Finding {
    /// Canonical key.
    pub key: String,
    /// Band classification.
    pub band: StalenessBand,
    /// Age in seconds at report time.
    pub age_seconds: u64,
    /// Configured interval in seconds.
    pub interval_seconds: u64,
    /// Number of times this endpoint was hit in the task.
    pub hits_in_task: u64,
    /// Human-readable PR comment fragment.
    pub message: String,
}

/// Aggregate counts.
#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct StalenessCounts {
    pub fresh: u64,
    pub aging: u64,
    pub stale: u64,
    pub unrecorded: u64,
}

/// In-memory tracker. Production callers persist via the Phase 6
/// promotion bundle; the tracker itself is a thin per-task aggregator.
#[derive(Debug, Default)]
pub struct Tracker {
    /// Persisted freshness per endpoint.
    pub registry: HashMap<String, EndpointFreshness>,
    /// Per-task hit counts.
    hits: HashMap<String, u64>,
}

impl Tracker {
    /// Construct an empty tracker.
    #[must_use]
    pub fn new() -> Self {
        Self::default()
    }

    /// Register the freshness baseline for one endpoint. Idempotent;
    /// later writes overwrite earlier ones (the shadow recorder reports
    /// the canonical timestamp).
    pub fn register(&mut self, freshness: EndpointFreshness) {
        self.registry.insert(freshness.key.clone(), freshness);
    }

    /// Increment the per-task hit count for one endpoint.
    pub fn record_hit(&mut self, key: &str) {
        *self.hits.entry(key.to_string()).or_default() += 1;
    }

    /// Render the per-task report.
    #[must_use]
    pub fn report(&self, now: DateTime<Utc>) -> StalenessReport {
        let mut findings = Vec::new();
        let mut counts = StalenessCounts::default();
        for (key, hits) in &self.hits {
            let (band, interval, last) = match self.registry.get(key) {
                Some(f) => (
                    classify(Some(f.last_recorded), f.recapture_interval, now),
                    f.recapture_interval,
                    Some(f.last_recorded),
                ),
                None => (
                    StalenessBand::Unrecorded,
                    DEFAULT_RECAPTURE,
                    None,
                ),
            };
            let age_seconds = last
                .map(|t| {
                    now.signed_duration_since(t)
                        .to_std()
                        .unwrap_or(Duration::ZERO)
                        .as_secs()
                })
                .unwrap_or(0);
            let interval_seconds = interval.as_secs();
            let msg = render_message(key, band, age_seconds, interval_seconds);
            match band {
                StalenessBand::Fresh => counts.fresh += 1,
                StalenessBand::Aging => counts.aging += 1,
                StalenessBand::Stale => counts.stale += 1,
                StalenessBand::Unrecorded => counts.unrecorded += 1,
            }
            findings.push(Finding {
                key: key.clone(),
                band,
                age_seconds,
                interval_seconds,
                hits_in_task: *hits,
                message: msg,
            });
        }
        findings.sort_by(|a, b| a.key.cmp(&b.key));
        StalenessReport {
            findings,
            counts,
            generated_at: now,
        }
    }

    /// Returns true if the report contains at least one stale entry. The
    /// promotion gate consults this to decide whether to demand operator
    /// re-record approval before promotion.
    #[must_use]
    pub fn has_stale(&self, now: DateTime<Utc>) -> bool {
        self.report(now).counts.stale > 0
    }
}

/// Default re-capture interval if not configured per-endpoint.
pub const DEFAULT_RECAPTURE: Duration = Duration::from_secs(30 * 24 * 3600);

fn render_message(
    key: &str,
    band: StalenessBand,
    age_seconds: u64,
    interval_seconds: u64,
) -> String {
    match band {
        StalenessBand::Fresh => format!(
            "[{key}] tape fresh ({age_seconds}s old, interval {interval_seconds}s)"
        ),
        StalenessBand::Aging => format!(
            "[{key}] tape aging ({age_seconds}s old, interval {interval_seconds}s) — consider re-recording"
        ),
        StalenessBand::Stale => format!(
            "[{key}] **tape stale** ({age_seconds}s old, > 2× interval {interval_seconds}s); verifier confidence reduced"
        ),
        StalenessBand::Unrecorded => format!(
            "[{key}] **no recording exists** — synth response only; verifier confidence reduced"
        ),
    }
}

/// Canonicalises (service, method, endpoint) into the registry key.
#[must_use]
pub fn key_for(service: &str, method: &str, endpoint: &str) -> String {
    format!(
        "{}|{}|{}",
        service.to_lowercase(),
        method.to_uppercase(),
        endpoint
    )
}

/// Errors from the tracker (currently none surface; reserved for future
/// persistence-layer additions).
#[derive(Debug, Error)]
pub enum StalenessError {
    /// Reserved.
    #[error("reserved")]
    Reserved,
}

#[cfg(test)]
mod tests {
    use super::*;
    use chrono::TimeZone;

    fn now() -> DateTime<Utc> {
        Utc.with_ymd_and_hms(2026, 5, 15, 12, 0, 0).unwrap()
    }

    #[test]
    fn classify_handles_fresh_aging_stale_unrecorded() {
        let day = Duration::from_secs(24 * 3600);
        let n = now();
        assert_eq!(
            classify(Some(n - chrono::Duration::hours(12)), day, n),
            StalenessBand::Fresh
        );
        assert_eq!(
            classify(Some(n - chrono::Duration::hours(36)), day, n),
            StalenessBand::Aging
        );
        assert_eq!(
            classify(Some(n - chrono::Duration::hours(72)), day, n),
            StalenessBand::Stale
        );
        assert_eq!(classify(None, day, n), StalenessBand::Unrecorded);
    }

    #[test]
    fn tracker_reports_per_endpoint_band() {
        let mut t = Tracker::new();
        let n = now();
        t.register(EndpointFreshness {
            key: key_for("stripe", "GET", "/v1/charges"),
            last_recorded: n - chrono::Duration::days(2),
            recapture_interval: Duration::from_secs(24 * 3600),
            samples: 100,
        });
        t.record_hit(&key_for("stripe", "GET", "/v1/charges"));
        t.record_hit(&key_for("stripe", "GET", "/v1/customers"));
        let report = t.report(n);
        assert_eq!(report.findings.len(), 2);
        // /v1/charges → stale (48h vs 24h interval → > 2×? exactly 2×, classified Aging).
        assert!(report
            .findings
            .iter()
            .find(|f| f.key.ends_with("/v1/charges"))
            .unwrap()
            .band
            != StalenessBand::Fresh);
        assert_eq!(
            report
                .findings
                .iter()
                .find(|f| f.key.ends_with("/v1/customers"))
                .unwrap()
                .band,
            StalenessBand::Unrecorded
        );
    }

    #[test]
    fn has_stale_signals_promotion_gate() {
        let mut t = Tracker::new();
        let n = now();
        t.register(EndpointFreshness {
            key: key_for("stripe", "GET", "/v1/charges"),
            last_recorded: n - chrono::Duration::days(90),
            recapture_interval: Duration::from_secs(24 * 3600),
            samples: 1,
        });
        t.record_hit(&key_for("stripe", "GET", "/v1/charges"));
        assert!(t.has_stale(n));
    }

    #[test]
    fn render_message_includes_action_hint() {
        let m = render_message("svc|GET|/x", StalenessBand::Aging, 90_000, 86_400);
        assert!(m.contains("aging"));
        assert!(m.contains("consider re-recording"));
    }
}
