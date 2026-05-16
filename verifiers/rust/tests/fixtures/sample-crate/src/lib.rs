//! Tiny library exercising the cargo-mutants 27.x mutator catalogue.
//!
//! - `add` exercises a `BinaryOperator` mutation (`+` → `-`).
//! - `is_positive` exercises a `BooleanLiteral` mutation (`true` → `false`).
//! - `double` exercises another `BinaryOperator` (`*` → `/`).
//!
//! The bundled tests kill the first two mutants; the third intentionally
//! survives to produce a representative `survived_summary` entry in the
//! fixture outcomes file.

#![forbid(unsafe_code)]

/// Adds two integers. Tested by `tests::add_returns_sum`.
pub fn add(a: i32, b: i32) -> i32 {
    a + b
}

/// Returns true when `n` is strictly positive. Tested by
/// `tests::is_positive_signals`.
pub fn is_positive(n: i32) -> bool {
    if n > 0 {
        true
    } else {
        false
    }
}

/// Doubles the supplied integer. The bundled `double_works` test only
/// checks `double(0)`, so a `*`-to-`/` mutant survives.
pub fn double(n: i32) -> i32 {
    n * 2
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn add_returns_sum() {
        assert_eq!(add(2, 3), 5);
        assert_eq!(add(-1, 1), 0);
    }

    #[test]
    fn is_positive_signals() {
        assert!(is_positive(1));
        assert!(!is_positive(0));
        assert!(!is_positive(-1));
    }

    #[test]
    fn double_works() {
        // Intentionally weak test: only covers n = 0 so `*` → `/`
        // survives. Mirrors the fixture's "Missed" outcome.
        assert_eq!(double(0), 0);
    }
}
