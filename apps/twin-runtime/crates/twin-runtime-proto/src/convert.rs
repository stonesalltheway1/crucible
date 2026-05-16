//! Conversions between proto wire types and [`crucible_sandbox_spec`]
//! domain types.
//!
//! Wire types are an external contract — they may evolve under buf
//! breaking-change tracking. The runtime works in the domain types so the
//! conversion boundary is the single place wire-format quirks need to be
//! handled.

// NOTE: this module is intentionally thin in Phase 2. Each conversion is a
// straightforward field-by-field copy. We keep them as explicit `impl
// From/TryFrom` blocks so that buf-induced renames produce a single compile
// error in this file rather than ripple through the runtime.

use crate::v1;

/// Marker error for proto→domain conversions that lose data.
#[derive(Debug, thiserror::Error)]
pub enum ConvertError {
    /// A proto enum variant has no domain counterpart. Typically because a
    /// newer client sent us an enum value our binary doesn't know.
    #[error("unknown enum variant {kind}::{value}")]
    UnknownEnum {
        /// Enum name.
        kind: &'static str,
        /// Wire value.
        value: i32,
    },
    /// A required field was missing.
    #[error("missing required field {0}")]
    Missing(&'static str),
}

mod kind {
    use super::*;

    impl TryFrom<v1::SandboxKind> for crucible_sandbox_spec::SandboxKind {
        type Error = ConvertError;
        fn try_from(value: v1::SandboxKind) -> Result<Self, Self::Error> {
            Ok(match value {
                v1::SandboxKind::E2b => Self::E2b,
                v1::SandboxKind::RawFirecracker => Self::RawFirecracker,
                v1::SandboxKind::Modal => Self::Modal,
                v1::SandboxKind::Daytona => Self::Daytona,
                v1::SandboxKind::FlyMachines => Self::FlyMachines,
                v1::SandboxKind::LocalDocker => Self::LocalDocker,
                v1::SandboxKind::Unspecified => {
                    return Err(ConvertError::UnknownEnum {
                        kind: "SandboxKind",
                        value: 0,
                    });
                }
            })
        }
    }

    impl From<crucible_sandbox_spec::SandboxKind> for v1::SandboxKind {
        fn from(value: crucible_sandbox_spec::SandboxKind) -> Self {
            match value {
                crucible_sandbox_spec::SandboxKind::E2b => Self::E2b,
                crucible_sandbox_spec::SandboxKind::RawFirecracker => Self::RawFirecracker,
                crucible_sandbox_spec::SandboxKind::Modal => Self::Modal,
                crucible_sandbox_spec::SandboxKind::Daytona => Self::Daytona,
                crucible_sandbox_spec::SandboxKind::FlyMachines => Self::FlyMachines,
                crucible_sandbox_spec::SandboxKind::LocalDocker => Self::LocalDocker,
            }
        }
    }
}

mod kill_reason {
    use super::*;
    use crucible_sandbox_spec::SandboxKillReason;

    impl TryFrom<v1::SandboxKillReason> for SandboxKillReason {
        type Error = ConvertError;
        fn try_from(value: v1::SandboxKillReason) -> Result<Self, Self::Error> {
            Ok(match value {
                v1::SandboxKillReason::Clean => Self::Clean,
                v1::SandboxKillReason::Ttl => Self::Ttl,
                v1::SandboxKillReason::EscapeAttempt => Self::EscapeAttempt,
                v1::SandboxKillReason::Budget => Self::Budget,
                v1::SandboxKillReason::Manual => Self::Manual,
                v1::SandboxKillReason::ProviderFailure => Self::ProviderFailure,
                v1::SandboxKillReason::HeartbeatLost => Self::HeartbeatLost,
                v1::SandboxKillReason::DestructiveDenied => Self::DestructiveDenied,
                v1::SandboxKillReason::Unspecified => {
                    return Err(ConvertError::UnknownEnum {
                        kind: "SandboxKillReason",
                        value: 0,
                    });
                }
            })
        }
    }

    impl From<SandboxKillReason> for v1::SandboxKillReason {
        fn from(value: SandboxKillReason) -> Self {
            match value {
                SandboxKillReason::Clean => Self::Clean,
                SandboxKillReason::Ttl => Self::Ttl,
                SandboxKillReason::EscapeAttempt => Self::EscapeAttempt,
                SandboxKillReason::Budget => Self::Budget,
                SandboxKillReason::Manual => Self::Manual,
                SandboxKillReason::ProviderFailure => Self::ProviderFailure,
                SandboxKillReason::HeartbeatLost => Self::HeartbeatLost,
                SandboxKillReason::DestructiveDenied => Self::DestructiveDenied,
            }
        }
    }
}
