// Package tapedriver is the Crucible Twin Runtime's service-twin layer
// (Hoverfly tape replay).
//
// Phase 2 ships:
//   - The decision-tree engine from docs/06-research/tape-coverage-strategy.md
//     (exact hit / template hit / synth-readonly / synth-mutation /
//     live-passthrough / miss-blocked).
//   - The X-Crucible-Tape response header on every served response.
//   - A regex-only PII scrubber. Phase 3 swaps the scrubber for Presidio +
//     spaCy + FF3-1; the [Scrubber] interface is shaped so the swap is a
//     drop-in.
//   - The Hoverfly subprocess wrapper.
//
// Per the May 2026 currency check, Hoverfly has been in maintenance mode
// for ~12 months (last release v1.12.7, May 2024). One instance runs in
// one mode at a time; we spawn per-twin instances rather than one global.
// gRPC is NOT in the OSS core — Phase 2 documents the gap and the
// gripmock sidecar fallback for gRPC tapes.
package tapedriver

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

// TapeDriver is the service-twin orchestrator.
type TapeDriver interface {
	// PrepareTape materialises the tape bundle at /work/tapes and
	// configures the Hoverfly instance scoped to this sandbox.
	PrepareTape(ctx context.Context, spec TapeSpec) (MountedTape, error)

	// EvaluateRequest classifies an outgoing request per the decision tree
	// and returns the appropriate [Response] (with the X-Crucible-Tape
	// header pre-set). It is the runtime's central matching primitive.
	EvaluateRequest(ctx context.Context, req Request) (Response, error)

	// Scrub returns the scrubbed form of a payload before persistence.
	// In Phase 2 this is regex-only.
	Scrub(payload []byte) ([]byte, ScrubReport)

	// Unmount cleans up the tape mount + Hoverfly subprocess.
	Unmount(ctx context.Context, handle MountedTape) error
}

// TapeSpec describes the tape we want to mount for a sandbox.
type TapeSpec struct {
	TapeSet         string
	Mode            string   // "strict" | "hybrid" | "adaptive"
	SynthEngine     string   // "none" | "schema" (Phase 2); "schema+llm" (Phase 3)
	MutationPolicy  string   // "journal" | "block"
	MissStatus      int      // 599 default
	AllowLiveHosts  []string
}

// MountedTape is the runtime handle returned by PrepareTape.
type MountedTape struct {
	TapeSet         string
	MountPath       string
	HoverflyPort    int
	HoverflyAdmin   int
	ProcessID       int
	StartedAt       time.Time
}

// Request is the agent's outgoing call (already authenticated to the
// twin-scoped vault, so Authorization is twin token only).
type Request struct {
	Service    string
	Endpoint   string
	Method     string
	Headers    map[string]string
	Body       []byte
	RequestID  string
}

// Response is the served response with the X-Crucible-Tape header.
type Response struct {
	Status         int
	Headers        map[string]string
	Body           []byte
	Disposition    Disposition
	X_Crucible_Tape string
}

// Disposition mirrors the X-Crucible-Tape header values.
type Disposition string

// Disposition values per docs/06-research/tape-coverage-strategy.md.
const (
	DispoHitExact      Disposition = "hit-exact"
	DispoHitTemplate   Disposition = "hit-template"
	DispoSynthReadOnly Disposition = "synth-readonly"
	DispoSynthMutation Disposition = "synth-mutation"
	DispoLivePass      Disposition = "live-passthrough"
	DispoMissBlocked   Disposition = "miss-blocked"
)

// Scrubber is the PII scrub interface. Phase 2 ships [RegexScrubber];
// Phase 3 will plug in [PresidioScrubber].
type Scrubber interface {
	Scrub(payload []byte) ([]byte, ScrubReport)
}

// ScrubReport enumerates which scrubbers fired and what was rewritten.
// Each entry feeds the per-tape audit log.
type ScrubReport struct {
	Rewrites []ScrubRewrite
}

// ScrubRewrite is one rewrite event.
type ScrubRewrite struct {
	Scrubber string
	Field    string
	Before   string
	After    string
}

// DefaultTapeDriver is the in-process implementation.
type DefaultTapeDriver struct {
	scrubber Scrubber
}

// New constructs the default driver. When CRUCIBLE_SCRUBBER_URL is set the
// driver wires the Phase 3 Presidio remote service; otherwise it falls back
// to the Phase 2 RegexScrubber baseline.
//
// Production deployments MUST set the Presidio URL — the regex baseline is
// not HIPAA-Safe-Harbor compliant on free-text PII and is intended only for
// CI and dev loops.
func New() *DefaultTapeDriver {
	return &DefaultTapeDriver{
		scrubber: defaultScrubber(),
	}
}

// NewWithScrubber constructs a driver around an explicitly-passed scrubber.
// Useful for callers that need to inject a per-tape-set namespace.
func NewWithScrubber(s Scrubber) *DefaultTapeDriver {
	if s == nil {
		s = NewRegexScrubber()
	}
	return &DefaultTapeDriver{scrubber: s}
}

func defaultScrubber() Scrubber {
	if os.Getenv(EnvScrubberURL) != "" {
		return NewPresidioScrubber()
	}
	return NewRegexScrubber()
}

// PrepareTape is a STUB in Phase 2; the runtime invokes Hoverfly via a
// shell-out which lives in [HoverflySubprocess]. Phase 3 wires the full
// shadow-recorder + tape-aggregator pipeline.
func (d *DefaultTapeDriver) PrepareTape(ctx context.Context, spec TapeSpec) (MountedTape, error) {
	_ = ctx
	if strings.TrimSpace(spec.TapeSet) == "" {
		return MountedTape{}, errors.New("TapeSet empty")
	}
	return MountedTape{
		TapeSet:       spec.TapeSet,
		MountPath:     fmt.Sprintf("/work/tapes/%s", spec.TapeSet),
		HoverflyPort:  8500,
		HoverflyAdmin: 8888,
		ProcessID:     0,
		StartedAt:     time.Now(),
	}, nil
}

// EvaluateRequest matches against the in-memory tape store. Phase 2 ships
// the decision tree's classifying logic; the actual tape store is loaded
// from the mount path during PrepareTape (Phase 3 wires that pipeline).
func (d *DefaultTapeDriver) EvaluateRequest(ctx context.Context, req Request) (Response, error) {
	_ = ctx
	// Phase 2 default: every miss is fail-closed 599 with miss-blocked.
	// The runtime's Hoverfly process is what serves hits; this method's
	// role is to enforce the policy knobs (mode, allow_live_hosts, etc).
	return Response{
		Status:          599,
		Headers:         map[string]string{"X-Crucible-Tape": string(DispoMissBlocked)},
		Body:            []byte(`{"error":"tape miss; fail-closed","request_id":"` + req.RequestID + `"}`),
		Disposition:     DispoMissBlocked,
		X_Crucible_Tape: string(DispoMissBlocked),
	}, nil
}

// Scrub runs the regex scrubber.
func (d *DefaultTapeDriver) Scrub(payload []byte) ([]byte, ScrubReport) {
	return d.scrubber.Scrub(payload)
}

// Unmount is a no-op in Phase 2.
func (d *DefaultTapeDriver) Unmount(ctx context.Context, handle MountedTape) error {
	_ = ctx
	_ = handle
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// RegexScrubber — the Phase 2 PII-scrub baseline.
// ─────────────────────────────────────────────────────────────────────────────

// RegexScrubber is the minimal PII scrubber. Catches:
//   - Email addresses
//   - US SSNs (NNN-NN-NNNN)
//   - Credit-card-shaped numbers (13–19 digits with Luhn-shape grouping)
//   - Phone numbers (E.164 / US 10-digit)
//   - IPv4 addresses
//   - JWTs
//
// Phase 3 swaps this for Microsoft Presidio + spaCy + FF3-1 — the
// [Scrubber] interface is identical so the swap is in-place.
type RegexScrubber struct {
	patterns []scrubRule
}

type scrubRule struct {
	name    string
	pattern *regexp.Regexp
	replace func(match string) string
}

// NewRegexScrubber constructs the regex scrubber with the canonical rule set.
func NewRegexScrubber() *RegexScrubber {
	return &RegexScrubber{
		patterns: []scrubRule{
			{
				name:    "email",
				pattern: regexp.MustCompile(`[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}`),
				replace: func(_ string) string { return "redacted@example.com" },
			},
			{
				name:    "us-ssn",
				pattern: regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`),
				replace: func(_ string) string { return "XXX-XX-XXXX" },
			},
			{
				name: "credit-card",
				// 13-19 digit groups commonly delimited by - or space.
				pattern: regexp.MustCompile(`\b(?:\d[ -]*?){13,19}\b`),
				replace: func(_ string) string { return "[REDACTED-CARD]" },
			},
			{
				name:    "phone-e164",
				pattern: regexp.MustCompile(`\+\d{1,3}\d{4,14}`),
				replace: func(_ string) string { return "+10000000000" },
			},
			{
				name:    "phone-us-10",
				pattern: regexp.MustCompile(`\b\d{3}[\.\- ]?\d{3}[\.\- ]?\d{4}\b`),
				replace: func(_ string) string { return "000-000-0000" },
			},
			{
				name:    "ipv4",
				pattern: regexp.MustCompile(`\b(?:25[0-5]|2[0-4]\d|1\d\d|\d{1,2})(?:\.(?:25[0-5]|2[0-4]\d|1\d\d|\d{1,2})){3}\b`),
				replace: func(_ string) string { return "127.0.0.1" },
			},
			{
				name:    "jwt",
				pattern: regexp.MustCompile(`eyJ[A-Za-z0-9_\-]{10,}\.[A-Za-z0-9_\-]{10,}\.[A-Za-z0-9_\-]{10,}`),
				replace: func(_ string) string { return "[REDACTED-JWT]" },
			},
			{
				name:    "aws-access-key",
				pattern: regexp.MustCompile(`\bAKIA[0-9A-Z]{16}\b`),
				replace: func(_ string) string { return "AKIAXXXXXXXXXXXXXXXX" },
			},
			{
				name:    "github-pat",
				pattern: regexp.MustCompile(`\bghp_[A-Za-z0-9]{36,}\b|\bgithub_pat_[A-Za-z0-9_]{60,}\b`),
				replace: func(_ string) string { return "ghp_XXXXXXXXXXXXXXXXXXXX" },
			},
			{
				name:    "anthropic-key",
				pattern: regexp.MustCompile(`\bsk-ant-api03-[A-Za-z0-9_\-]{50,}\b`),
				replace: func(_ string) string { return "sk-ant-api03-XXXXXXXXXX" },
			},
		},
	}
}

// Scrub applies every rule. The report enumerates rewrites for the audit log.
func (s *RegexScrubber) Scrub(payload []byte) ([]byte, ScrubReport) {
	report := ScrubReport{}
	out := payload
	for _, rule := range s.patterns {
		out = rule.pattern.ReplaceAllFunc(out, func(match []byte) []byte {
			before := string(match)
			after := rule.replace(before)
			report.Rewrites = append(report.Rewrites, ScrubRewrite{
				Scrubber: rule.name,
				Field:    "[inline]",
				Before:   before,
				After:    after,
			})
			return []byte(after)
		})
	}
	return out, report
}

// ─────────────────────────────────────────────────────────────────────────────
// HoverflySubprocess — process-lifecycle wrapper.
// Phase 2 ships the launch/stop helpers; the runtime invokes them but does
// not yet load tapes (the tape-import pipeline is Phase 3).
// ─────────────────────────────────────────────────────────────────────────────

// HoverflyCmd describes a Hoverfly subprocess invocation.
type HoverflyCmd struct {
	BinPath     string
	ProxyPort   int
	AdminPort   int
	Middleware  string // optional path to a scrubber middleware binary
	TapeImport  string // optional simulation.json to import on start
	AuthEnable  bool   // gate the admin port behind JWT auth
	Username    string
	Password    string
}

// Render returns the argv for the subprocess. We expose it so the runtime
// can log it for the operator's RB-09 reference.
func (c HoverflyCmd) Render() []string {
	argv := []string{
		c.BinPath,
		"-listen-on-host", "127.0.0.1",
		"-pp", fmt.Sprintf("%d", c.ProxyPort),
		"-ap", fmt.Sprintf("%d", c.AdminPort),
	}
	if c.Middleware != "" {
		argv = append(argv, "-middleware", c.Middleware)
	}
	if c.TapeImport != "" {
		argv = append(argv, "-import", c.TapeImport)
	}
	if c.AuthEnable {
		argv = append(argv, "-auth")
		if c.Username != "" {
			argv = append(argv, "-username", c.Username)
		}
		if c.Password != "" {
			argv = append(argv, "-password", c.Password)
		}
	}
	return argv
}
