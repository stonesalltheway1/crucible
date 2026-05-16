You are starting Phase 11 of building Crucible — v2 twin runtime deepening.

v1 (Phases 2-3) shipped Postgres-centric twins via E2B and raw Firecracker.
Phase 11 expands the twin model to GPU workloads, mobile platforms, embedded
systems, and multi-region orchestration.

Pillar C from docs/07-roadmap/v2-vision.md. This phase is the most "ICP
expansion" of v2 — it opens Crucible to entirely new buyer segments (ML
engineering, mobile dev, firmware, hardware-adjacent).

CALIBRATION
===========
Phase 11 targets ~25K LoC. Each vertical (GPU / mobile / embedded /
multi-region) is largely an integration project against mature primitives.
Quality bar emphasizes correctness on the new platforms because customers
in these verticals are unforgiving of platform-specific bugs.

READ FIRST
==========
1. docs/PHASE-10-REPORT.md
2. memory/project_crucible_phase10.md
3. docs/07-roadmap/v2-vision.md (Pillar C)
4. docs/01-architecture/twin-runtime.md (SandboxProvider abstraction)
5. docs/05-decisions/ADR-015-firecracker-via-e2b.md
6. Customer signal from v1 + Phase 10 — which vertical is most validated?
7. Vertical-specific competitive research:
   - Embedder.com (firmware vertical)
   - Callstack agent-device (mobile)
   - Unity AI (game-dev)
   - JetBrains Koog + Mellum (Kotlin/Android)

If customer signal heavily favors ONE vertical (mobile vs GPU vs embedded),
SHIP THAT FIRST and defer the others to subsequent v2 phases.

RESEARCH BEFORE CODING (parallel)
=================================
1. Modal Sandbox GPU offering — pricing, GPU types (A100, H100), cold-start
   latency, container-runtime; alternative GPU-capable sandbox providers
   (Lambda Labs, RunPod) for comparison.

2. NVIDIA container runtime + CUDA-in-Firecracker (Kata-runtime variant) for
   self-hosted GPU twins.

3. MacStadium / Mac-Cloud APIs — iOS twins require macOS hosts; provisioning
   APIs; cold-start latency; pricing.

4. Android emulator-in-Firecracker — KVM-accelerated Android emulator
   patterns; Google Cloud's Android Emulator API; AWS Device Farm.

5. QEMU + ESP32 / STM32 firmware simulation — Renode (Antmicro) current
   state for multi-platform hardware emulation; QEMU board-support coverage.

6. AWS Local Zones / GCP regional sandboxes — multi-region orchestration
   primitives; data-residency enforcement.

7. iOS Xcode toolchain — what's needed for hermetic CI builds; SwiftPM vs
   Bazel for hermetic; Xcode-cloud APIs.

8. Embedder.com architecture references (their public materials) — how they
   ground firmware agents in hardware catalogs.

PHASE 11 SCOPE
==============

ICP-DRIVEN PRIORITIZATION
-------------------------
Pick the vertical with strongest v1 customer signal as the primary deliverable
for THIS session; stub the others. Default order if signal is balanced:

1. GPU twins (broadest applicability — ML workloads are everywhere)
2. Mobile twins (clearest competitive wedge — incumbents are weak here)
3. Multi-region (compliance / latency-driven; smaller scope)
4. Embedded / firmware (highest WTP but smallest market)

EXPLICITLY IN SCOPE (per vertical; ship 1-2 fully, stub the rest)
----------------------------------------------------------------

1. apps/twin-runtime/sandbox/modal/ — GPU sandbox driver (C1):
   - Modal SandboxProvider implementation
   - GPU types as task-manifest parameter (a100 / h100 / l4)
   - Cost accounting per GPU-hour (different from CPU-hour)
   - Per-tenant GPU quota
   - PyTorch / CUDA / cuDNN pre-loaded base images
   - ML-specific twin: same architectural invariants (twin DB, twin services,
     etc.) just with GPU-attached compute

2. apps/twin-runtime/sandbox/mobile-ios/ — iOS twin driver (C2):
   - MacStadium API integration for macOS host provisioning
   - Xcode + iOS Simulator setup per twin
   - Per-task simulator instance (iPhone 16 / iPad / specific OS version
     per manifest)
   - Hermetic build via SwiftPM (or Bazel if customer uses it)
   - Twin includes: filesystem, simulator state, mock services (StoreKit,
     APNs, CloudKit, etc.)
   - Tape replay extended for native iOS HTTP/URLSession patterns

3. apps/twin-runtime/sandbox/mobile-android/ — Android twin driver (C2):
   - Android Emulator in KVM-accelerated Firecracker
   - Per-task emulator instance (Pixel 8 / specific Android version)
   - Gradle hermetic build
   - Twin includes: app data, emulator state, mock services (Play Billing,
     FCM, Google Sign-In)

4. apps/twin-runtime/sandbox/embedded/ — firmware twin driver (C3):
   - QEMU + Renode multi-platform hardware emulator
   - Hardware-catalog grounding (ESP32, STM32, Nordic SDK device profiles)
   - Per-MCU peripheral simulation
   - Twin includes: flash memory, peripheral state, simulated hardware
     events
   - Use case: senior firmware engineers in safety-critical contexts

5. apps/twin-runtime/multi-region/ — C4 (multi-region orchestration):
   - Tenant-region affinity (config in tenant settings)
   - Per-region sandbox-provider routing
   - Cross-region attestation chain (Rekor instances per region OR shared
     global)
   - Data-residency enforcement: twin runs in customer's specified region
     ONLY; egress allowlist enforces

6. Per-vertical verifier extensions:
   - GPU: numerical-accuracy property tests (proptest for tensor ops)
   - iOS: XCTest + xcuitest integration; snapshot regression for UI
   - Android: Espresso / Compose UI Testing integration
   - Embedded: hardware-in-the-loop tests via Renode; cycle-accurate fuzzing
   - Tier 3 hot for embedded: ASIL/SIL annotation handling (path-pattern
     match per docs/06-research/tier3-trigger-automation.md)

7. SDK extensions for new platform primitives:
   - twin.gpu.* — query GPU state, run inference, etc.
   - twin.mobile.simulator.* — interact with simulator
   - twin.firmware.* — flash, run, inspect peripheral state

8. Tests:
   - Per-vertical: a fixture project per platform; agent builds + verifies
     end-to-end.
   - GPU: inference correctness against a known reference output.
   - iOS: simulator screenshot regression on a sample app.
   - Android: emulator runs the agent's build successfully.
   - Embedded: ESP32 firmware boot + peripheral interaction simulated.
   - Multi-region: data-residency test (egress to wrong region blocked).

9. Docs updates:
   - docs/05-decisions/ADR-020-multi-vertical-sandbox-providers.md
   - Per-vertical docs/01-architecture/twin-runtime-{gpu,mobile,embedded}.md
   - Pricing tier updates if vertical-specific pricing emerges (e.g., GPU-hours)
   - docs/04-operations/runbooks.md additions

EXPLICITLY OUT OF SCOPE
-----------------------
- Console games (Unity/Unreal twin support) — different vertical; v3+
- Smart contracts / blockchain twins — different vertical
- VR/AR (Vision Pro, Meta Quest) — too early
- Web3 / decentralized infra targets — not aligned with ICP

WORKING AGREEMENTS
==================
- All new platforms implement the SandboxProvider interface. The dispatcher
  doesn't grow per-platform branching beyond per-platform manifest fields.
- GPU twins maintain the same trust invariants: twin DB, twin services,
  twin secrets, syscall shim — none of these are relaxed for "ML workloads."
- Mobile twins maintain attestation parity: every fs.write, every simulator
  interaction emits attestations.
- Multi-region: data-residency is enforced at the egress proxy layer + the
  sandbox-provider selection layer. Two independent enforcement points.

QUALITY BAR
===========
- Per-vertical first-task time: ≤ 5 minutes for GPU; ≤ 10 minutes for iOS /
  Android (simulator boot is slower); ≤ 8 minutes for embedded (QEMU is
  fast).
- All threat-model invariants from Phase 2 hold across new platforms. Audit
  per-platform.
- Multi-region: data-residency violation detection has zero false negatives
  in adversarial test.
- Mutation score ≥ 85% on diff.

PROGRESS TRACKING
=================
  1. Read docs + customer signal
  2. Currency-check research (8 streams)
  3. PRIORITIZE: pick 1-2 verticals based on customer signal
  4. Implement primary vertical's sandbox provider
  5. Per-vertical verifier extensions
  6. SDK extensions
  7. Multi-region orchestration (if prioritized)
  8. Per-vertical tests
  9. Docs + report

END-OF-SESSION REPORT
=====================
docs/PHASE-11-REPORT.md:

1. Which verticals shipped, which stubbed
2. Per-vertical demo (commands + output)
3. Per-vertical pricing implications (GPU-hour cost, etc.)
4. Threat-model invariant audit across new platforms
5. The Phase 12 prompt (pricing + specialization wedge — template at
   docs/08-phase-prompts/phase-12-pricing-and-specialization.md)

Update memory: project_crucible_phase11.md.

GUARDRAILS
==========
- Do NOT relax safety invariants for any vertical. ML, mobile, firmware
  customers expect the same trust posture as web-backend customers.
- Do NOT skip attestation emission on new platforms. Every twin action
  attests; this is the brand.
- Do NOT default GPU twins to high-tier expensive GPUs. Quota + manifest-
  declared GPU type is the customer-control surface.
- Do NOT cross data-residency boundaries even for fallback routing. If the
  customer's region's provider is unavailable, halt the task; don't fail
  over to another region.

Each vertical opens a new ICP. Phase 11 is where Crucible stops being a
"web-backend tool" and becomes a "production engineering platform."

Begin.
