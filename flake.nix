{
  description = "Crucible — the AI engineer that tests every change in a digital twin";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
    rust-overlay = {
      url = "github:oxalica/rust-overlay";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs = { self, nixpkgs, flake-utils, rust-overlay, ... }:
    flake-utils.lib.eachSystem [
      "x86_64-linux"
      "aarch64-linux"
      "x86_64-darwin"
      "aarch64-darwin"
    ] (system:
      let
        overlays = [ (import rust-overlay) ];
        pkgs = import nixpkgs { inherit system overlays; };

        rustToolchain = pkgs.rust-bin.stable.latest.default.override {
          extensions = [ "rust-src" "rust-analyzer" "clippy" "rustfmt" ];
          targets = [ "x86_64-unknown-linux-gnu" ];
        };

        commonTools = with pkgs; [
          git
          jq
          curl
          gnumake
          buf
          protobuf
          cosign
          opa
        ];

        goTools = with pkgs; [
          go_1_23
          gopls
          golangci-lint
          gotools
          delve
        ];

        nodeTools = with pkgs; [
          nodejs_22
          pnpm
          biome
        ];

        pythonTools = with pkgs; [
          python312
          python312Packages.pip
          ruff
          uv
        ];

        rustTools = [
          rustToolchain
          pkgs.cargo-mutants
          pkgs.cargo-nextest
        ];
      in {
        devShells = {
          default = pkgs.mkShell {
            name = "crucible-dev";
            packages = commonTools ++ goTools ++ nodeTools ++ pythonTools ++ rustTools;

            shellHook = ''
              echo "Crucible dev shell ($(${pkgs.go_1_23}/bin/go version | awk '{print $3}'))"
              echo "Tools: go, node, python, rust, buf, cosign, opa"
              export CRUCIBLE_DEV_SHELL=1
              export CGO_ENABLED=0
              export GO111MODULE=on
            '';
          };

          go-only = pkgs.mkShell {
            name = "crucible-go";
            packages = commonTools ++ goTools;
          };

          rust-only = pkgs.mkShell {
            name = "crucible-rust";
            packages = commonTools ++ rustTools;
          };

          node-only = pkgs.mkShell {
            name = "crucible-node";
            packages = commonTools ++ nodeTools;
          };

          python-only = pkgs.mkShell {
            name = "crucible-python";
            packages = commonTools ++ pythonTools;
          };
        };

        packages = {
          control-plane = pkgs.buildGoModule {
            pname = "crucible-control-plane";
            version = "2026.06.0-phase1";
            src = ./.;
            modRoot = "apps/control-plane";
            vendorHash = null;
            CGO_ENABLED = "0";
            ldflags = [ "-s" "-w" "-trimpath" ];
            doCheck = false;
            meta.description = "Crucible Agent Control Plane";
          };

          cli = pkgs.buildGoModule {
            pname = "crucible-cli";
            version = "2026.06.0-phase1";
            src = ./.;
            modRoot = "apps/cli";
            vendorHash = null;
            CGO_ENABLED = "0";
            ldflags = [ "-s" "-w" "-trimpath" ];
            doCheck = false;
            meta.description = "Crucible CLI";
          };

          # Phase 2 — Twin Runtime (Rust). Hermetic build per ADR-013.
          twin-runtime = pkgs.rustPlatform.buildRustPackage {
            pname = "crucible-twin-runtime";
            version = "2026.06.0-phase2";
            src = ./.;
            sourceRoot = "source/apps/twin-runtime";
            cargoLock = {
              lockFile = ./apps/twin-runtime/Cargo.lock;
              # outputHashes covers any git deps; Phase 2 has none.
              outputHashes = { };
            };
            buildPhase = ''
              cargo build --release --locked -p twin-runtime-server
            '';
            doCheck = false;
            cargoBuildFlags = [ "-p" "twin-runtime-server" ];
            nativeBuildInputs = [ pkgs.protobuf ];
            meta.description = "Crucible Twin Runtime gRPC server";
          };
        };

        checks = {
          format-check = pkgs.runCommand "format-check" { buildInputs = [ pkgs.go_1_23 ]; } ''
            cd ${./.}
            test -z "$(gofmt -l apps/ libs/ services/ 2>/dev/null || true)"
            touch $out
          '';
        };

        formatter = pkgs.nixpkgs-fmt;
      });
}
