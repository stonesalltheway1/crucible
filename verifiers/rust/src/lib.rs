//! Library surface for the `crucible-verify-rust` binary. We expose the
//! schema, audit, diff, and tier modules so integration tests in
//! `tests/` can poke at them without re-running the CLI; the binary in
//! `src/main.rs` re-uses these modules verbatim.

#![forbid(unsafe_code)]
#![warn(missing_docs)]

pub mod audit;
pub mod diff;
pub mod schema;
pub mod tiers;
