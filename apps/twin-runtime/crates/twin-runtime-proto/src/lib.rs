//! Proto-generated types for the Crucible Twin Runtime.
//!
//! All messages live under the [`crucible::v1`] module which mirrors the
//! proto package. The `FILE_DESCRIPTOR_SET` byte constant is exposed for
//! gRPC-reflection consumers.

#![allow(clippy::doc_markdown)]
#![allow(clippy::default_trait_access)]
#![allow(clippy::too_many_lines)]
#![allow(clippy::derive_partial_eq_without_eq)]
#![allow(clippy::large_enum_variant)]
#![allow(clippy::module_inception)]

/// Crucible v1 proto types.
#[allow(missing_docs)]
pub mod crucible {
    /// `crucible.v1` package.
    pub mod v1 {
        include!(concat!(env!("OUT_DIR"), "/crucible.v1.rs"));
    }
}

pub use crucible::v1 as v1;

/// Conversion helpers between the proto types and the
/// [`crucible_sandbox_spec`] domain types. The runtime uses the domain
/// types internally and converts at the gRPC boundary.
pub mod convert;
