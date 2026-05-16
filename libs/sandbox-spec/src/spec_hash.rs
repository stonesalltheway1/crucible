//! Canonical JSON + SHA-256 hashing for [`super::SandboxSpec`].
//!
//! Two specs that differ only in serialisation (key order, whitespace) must
//! produce the same hash. We achieve this by re-serialising through
//! [`serde_json::Value`] with [`BTreeMap`] semantics: any [`serde_json::Map`]
//! is converted to a sorted variant before serialisation.

use serde::Serialize;
use sha2::{Digest, Sha256};

/// Serialises `value` to canonical JSON: keys are recursively sorted, no
/// trailing whitespace, no leading/trailing whitespace. The output is
/// stable across runs.
///
/// # Panics
///
/// Panics if `value` cannot be serialised to JSON. Every type in this crate
/// derives `Serialize` and is `serde_json`-clean by construction.
#[must_use]
pub fn canonical_json<T: Serialize>(value: &T) -> String {
    let raw = serde_json::to_value(value).expect("sandbox-spec types are serde_json-clean");
    let sorted = sort_keys(raw);
    serde_json::to_string(&sorted).expect("sorted Value is serde_json-clean")
}

fn sort_keys(v: serde_json::Value) -> serde_json::Value {
    use serde_json::Value;
    match v {
        Value::Object(map) => {
            let mut sorted = std::collections::BTreeMap::new();
            for (k, val) in map {
                sorted.insert(k, sort_keys(val));
            }
            Value::Object(sorted.into_iter().collect())
        }
        Value::Array(items) => Value::Array(items.into_iter().map(sort_keys).collect()),
        other => other,
    }
}

/// Returns the hex-encoded SHA-256 of [`canonical_json`].
#[must_use]
pub fn sha256_canonical<T: Serialize>(value: &T) -> String {
    let bytes = canonical_json(value);
    let mut hasher = Sha256::new();
    hasher.update(bytes.as_bytes());
    hex::encode(hasher.finalize())
}

#[cfg(test)]
mod tests {
    use super::*;
    use serde_json::json;

    #[test]
    fn canonical_json_sorts_keys() {
        let v = json!({"z": 1, "a": 2, "m": {"y": 3, "x": 4}});
        let canonical = canonical_json(&v);
        assert_eq!(canonical, r#"{"a":2,"m":{"x":4,"y":3},"z":1}"#);
    }

    #[test]
    fn canonical_json_preserves_array_order() {
        // Arrays are *ordered* data — sorting them would change semantics.
        let v = json!([3, 1, 2]);
        assert_eq!(canonical_json(&v), "[3,1,2]");
    }

    #[test]
    fn sha256_canonical_is_deterministic() {
        let a = json!({"x": 1, "y": [2, 3], "z": {"nested": true}});
        let b = json!({"z": {"nested": true}, "y": [2, 3], "x": 1});
        assert_eq!(sha256_canonical(&a), sha256_canonical(&b));
    }

    #[test]
    fn sha256_canonical_changes_with_data() {
        let a = json!({"x": 1});
        let b = json!({"x": 2});
        assert_ne!(sha256_canonical(&a), sha256_canonical(&b));
    }
}
