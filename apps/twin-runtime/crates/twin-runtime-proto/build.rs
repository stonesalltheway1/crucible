//! Build-time codegen for the Crucible v1 proto schema.
//!
//! Source-of-truth lives at `libs/twin-spec/proto/crucible/v1/*.proto`. We
//! compile the whole crucible.v1 package — twin-runtime-server implements
//! both [`TwinRuntimeService`] (control-plane → runtime) and
//! [`AgentSdkService`] (sandbox-agent → runtime).

use std::env;
use std::path::PathBuf;

fn main() -> Result<(), Box<dyn std::error::Error>> {
    let proto_root: PathBuf = workspace_root().join("libs/twin-spec/proto");
    let crucible_v1 = proto_root.join("crucible/v1");

    let protos = [
        crucible_v1.join("common.proto"),
        crucible_v1.join("task.proto"),
        crucible_v1.join("memory.proto"),
        crucible_v1.join("verification.proto"),
        crucible_v1.join("attestation.proto"),
        crucible_v1.join("control_plane.proto"),
        crucible_v1.join("agent_sdk.proto"),
        crucible_v1.join("sandbox.proto"),
    ];

    for p in &protos {
        if !p.exists() {
            panic!(
                "proto file missing: {} — did you forget to add it?",
                p.display()
            );
        }
        println!("cargo:rerun-if-changed={}", p.display());
    }
    println!("cargo:rerun-if-changed={}", proto_root.display());

    let out_dir: PathBuf = PathBuf::from(env::var("OUT_DIR")?);

    tonic_build::configure()
        .build_server(true)
        .build_client(true)
        .out_dir(&out_dir)
        // Serde derives on every message — useful for the local journal
        // attestation publisher and for logging.
        .type_attribute(".", "#[derive(serde::Serialize, serde::Deserialize)]")
        // Skip serde on byte fields where prost uses Vec<u8> — serde-of-bytes
        // is base64 by default, which is fine.
        .compile_protos(
            &protos.iter().map(PathBuf::as_path).collect::<Vec<_>>(),
            &[proto_root.as_path()],
        )?;

    Ok(())
}

fn workspace_root() -> PathBuf {
    // CARGO_MANIFEST_DIR = .../apps/twin-runtime/crates/twin-runtime-proto
    // workspace root      = .../
    let manifest_dir = PathBuf::from(env::var("CARGO_MANIFEST_DIR").unwrap());
    manifest_dir
        .ancestors()
        .nth(4)
        .expect("workspace root four levels up from twin-runtime-proto crate")
        .to_path_buf()
}
