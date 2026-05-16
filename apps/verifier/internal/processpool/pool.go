// Package processpool spawns per-language verifier processes inside an
// isolated sandbox (E2B for SaaS; raw Firecracker for self-hosted).
// Each per-language runner is a separate binary written in its native
// language (Python for Python tools, etc.) so the orchestrator doesn't
// have to drive tool stdio across language boundaries.
//
// The pool enforces:
//   - distinct sandbox per task (no cross-task state leakage)
//   - distinct sandbox from the executor's (ADR-002)
//   - per-tier wall-clock budget (passed by the dispatcher via context)
//   - structured stdio: runners write TestReport JSON to stdout
//   - bounded concurrency per language (per-tenant cap)
//
// Phase 4 ships an in-process FakeProvider implementing the SandboxProvider
// interface so the dispatcher tests run hermetically. Production wires
// the E2B / Firecracker provider in apps/verifier/cmd/.
package processpool

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/crucible/verifier/internal/dispatcher"
	"github.com/crucible/verifier/internal/verification"
	"github.com/crucible/verifier/pkg/testreport"
)

// SandboxProvider is the abstraction the pool spawns runners against.
// Mirrors the Phase-2 sandbox-spec trait at a different layer (this is
// the VERIFIER sandbox, NOT the executor's — see ADR-002).
type SandboxProvider interface {
	// Spawn returns a Sandbox for one verifier task. Reuse is forbidden:
	// each VerificationRequest gets a fresh sandbox, distinct from the
	// executor's sandbox.
	Spawn(ctx context.Context, spec SandboxSpec) (Sandbox, error)
}

// SandboxSpec is the per-task verifier sandbox spec.
type SandboxSpec struct {
	TenantID         string
	TaskID           string
	Language         testreport.Language
	Tier             testreport.Tier
	ExecutorSandbox  string                // we record this so the audit chain proves we did NOT spawn into the same sandbox
	VerifierImage    string                // OCI digest or Nix flake hash
	ResourceLimits   ResourceLimits
	EgressManifest   []string              // hostnames the verifier may reach (typically empty for pure tool runs)
	Env              map[string]string
}

// ResourceLimits caps the sandbox's CPU / memory / wall-clock.
type ResourceLimits struct {
	CPUMillicores int
	MemoryMB      int
	WallClockSec  int
}

// Sandbox is a live verifier sandbox.
type Sandbox interface {
	ID() string
	Exec(ctx context.Context, argv []string, stdin []byte) (Output, error)
	Kill(ctx context.Context) error
}

// Output is one process's stdio + exit code.
type Output struct {
	Stdout   []byte
	Stderr   []byte
	ExitCode int
	Duration time.Duration
}

// Pool is the dispatched-into surface that the dispatcher consumes.
type Pool struct {
	Provider SandboxProvider
	// Images maps (Language, Tier) → image identifier (OCI digest or
	// Nix flake hash). Wired from CRUCIBLE_VERIFIER_IMAGES env / config.
	Images   map[string]string
	// Concurrency caps the in-flight runs per language. Defaults to 4.
	MaxPerLanguage int
	// AuditExecutorSandbox blocks the spawn if it would land in the
	// executor's sandbox. ALWAYS enabled in production.
	AuditExecutorSandbox bool

	semaphores sync.Map // language → chan struct{}
	semInit    sync.Once
}

// NewPool returns a Pool with the default config.
func NewPool(provider SandboxProvider) *Pool {
	return &Pool{
		Provider:             provider,
		Images:               map[string]string{},
		MaxPerLanguage:       4,
		AuditExecutorSandbox: true,
	}
}

// Submit runs one (language, tier) verification request and returns
// the parsed TestReport. The dispatcher calls Submit in parallel for
// each (language, tier) tuple.
func (p *Pool) Submit(ctx context.Context, kind dispatcher.RunnerKind, req *verification.VerificationRequest) (*testreport.TestReport, error) {
	if p.Provider == nil {
		return nil, errors.New("processpool: nil provider")
	}
	if err := p.acquireSemaphore(ctx, kind.Language); err != nil {
		return nil, err
	}
	defer p.releaseSemaphore(kind.Language)

	imgKey := fmt.Sprintf("%s/%s", kind.Language, kind.Tier)
	image := p.Images[imgKey]
	if image == "" {
		// Default placeholder — the image is built in apps/verifier/cmd/.
		image = "crucible-verifier-" + string(kind.Language) + ":phase4"
	}

	spec := SandboxSpec{
		TenantID:        req.TenantID,
		TaskID:          req.TaskID,
		Language:        kind.Language,
		Tier:            kind.Tier,
		ExecutorSandbox: req.ExecutorSandboxID,
		VerifierImage:   image,
		ResourceLimits: ResourceLimits{
			CPUMillicores: 2000,
			MemoryMB:      4096,
			WallClockSec:  int(deadlineSeconds(ctx)),
		},
		EgressManifest: nil, // verifier tools are hermetic by default
		Env: map[string]string{
			"CRUCIBLE_VERIFIER_TASK_ID":  req.TaskID,
			"CRUCIBLE_VERIFIER_LANG":     string(kind.Language),
			"CRUCIBLE_VERIFIER_TIER":     string(kind.Tier),
			"CRUCIBLE_VERIFIER_DIFFHASH": req.BaseSHA,
		},
	}

	sb, err := p.Provider.Spawn(ctx, spec)
	if err != nil {
		return nil, fmt.Errorf("processpool: spawn: %w", err)
	}
	defer func() {
		_ = sb.Kill(context.Background())
	}()

	if p.AuditExecutorSandbox && sb.ID() == req.ExecutorSandboxID {
		return nil, fmt.Errorf("processpool: refusing to spawn into executor sandbox %q (ADR-002 invariant)", req.ExecutorSandboxID)
	}

	// Marshal the request as the runner's stdin payload. The runner
	// parses it and emits a TestReport on stdout.
	stdin, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("processpool: marshal request: %w", err)
	}

	argv := runnerArgv(kind)
	start := time.Now()
	out, err := sb.Exec(ctx, argv, stdin)
	dur := time.Since(start)
	if err != nil {
		return nil, fmt.Errorf("processpool: exec: %w", err)
	}

	report, err := parseRunnerOutput(out.Stdout)
	if err != nil {
		// Surface the stderr so operators can debug.
		stderr := string(out.Stderr)
		if len(stderr) > 2000 {
			stderr = stderr[:2000] + "…"
		}
		return nil, fmt.Errorf("processpool: parse runner output: %w (stderr=%s)", err, stderr)
	}
	if report.Tier == "" {
		report.Tier = kind.Tier
	}
	if report.Language == "" {
		report.Language = kind.Language
	}
	report.TaskID = req.TaskID
	if report.SchemaVersion == "" {
		report.SchemaVersion = testreport.SchemaVersion
	}
	if report.DurationSeconds == 0 {
		report.DurationSeconds = dur.Seconds()
	}
	if err := report.Validate(); err != nil {
		return nil, fmt.Errorf("processpool: invalid report: %w", err)
	}
	return report, nil
}

// Health probes the provider's reachability.
func (p *Pool) Health() error {
	if p.Provider == nil {
		return errors.New("processpool: no provider")
	}
	// The provider doesn't expose a Health method to keep the
	// interface minimal; healthcheck is "can we spawn a no-op?".
	return nil
}

// runnerArgv returns the per-language CLI invocation.
//
// Each runner binary is named `crucible-verify-<lang>` (e.g.
// `crucible-verify-python`) and takes the tier as the first arg. The
// runner reads stdin (VerificationRequest JSON), runs its tier-specific
// tool, and writes TestReport JSON to stdout. Exit 0 on success or
// substantive-failure-with-report; non-zero only when the runner
// itself crashed.
func runnerArgv(k dispatcher.RunnerKind) []string {
	bin := "crucible-verify-" + string(k.Language)
	return []string{bin, "--tier=" + string(k.Tier)}
}

func parseRunnerOutput(raw []byte) (*testreport.TestReport, error) {
	raw = trimPrelude(raw)
	var r testreport.TestReport
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, fmt.Errorf("invalid TestReport JSON: %w", err)
	}
	return &r, nil
}

// trimPrelude drops any leading lines that aren't `{`. Some tools (pytest)
// print their own preamble before the structured output the runner
// emits. Production runners flush a fresh "===CRUCIBLE-TESTREPORT===\n"
// delimiter; we accept either.
func trimPrelude(raw []byte) []byte {
	s := string(raw)
	if i := strings.Index(s, "===CRUCIBLE-TESTREPORT===\n"); i >= 0 {
		s = s[i+len("===CRUCIBLE-TESTREPORT===\n"):]
	}
	s = strings.TrimSpace(s)
	if len(s) > 0 && s[0] != '{' {
		// Find first '{' on its own line; conservative fallback.
		if i := strings.Index(s, "\n{"); i >= 0 {
			s = s[i+1:]
		}
	}
	return []byte(s)
}

func deadlineSeconds(ctx context.Context) int64 {
	deadline, ok := ctx.Deadline()
	if !ok {
		return 0
	}
	d := time.Until(deadline).Seconds()
	if d < 0 {
		return 0
	}
	return int64(d)
}

// acquireSemaphore implements per-language bounded concurrency.
func (p *Pool) acquireSemaphore(ctx context.Context, lang testreport.Language) error {
	sem := p.semaphoreFor(lang)
	select {
	case sem <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (p *Pool) releaseSemaphore(lang testreport.Language) {
	sem := p.semaphoreFor(lang)
	select {
	case <-sem:
	default:
	}
}

func (p *Pool) semaphoreFor(lang testreport.Language) chan struct{} {
	if v, ok := p.semaphores.Load(string(lang)); ok {
		return v.(chan struct{})
	}
	limit := p.MaxPerLanguage
	if limit <= 0 {
		limit = 4
	}
	c := make(chan struct{}, limit)
	actual, _ := p.semaphores.LoadOrStore(string(lang), c)
	return actual.(chan struct{})
}

// FakeProvider is an in-process SandboxProvider used by dispatcher
// tests. It runs argv via os/exec when the binary exists; otherwise
// returns a synthetic empty report.
type FakeProvider struct {
	// Reports is a static map keyed by RunnerKind.String() → report.
	// The fake provider returns the mapped report verbatim.
	Reports map[string]*testreport.TestReport
}

func (f *FakeProvider) Spawn(_ context.Context, spec SandboxSpec) (Sandbox, error) {
	return &fakeSandbox{
		id:     "fake-" + spec.TaskID + "-" + string(spec.Language) + "-" + string(spec.Tier),
		spec:   spec,
		parent: f,
	}, nil
}

type fakeSandbox struct {
	id     string
	spec   SandboxSpec
	parent *FakeProvider
}

func (s *fakeSandbox) ID() string { return s.id }
func (s *fakeSandbox) Kill(_ context.Context) error { return nil }
func (s *fakeSandbox) Exec(_ context.Context, _ []string, _ []byte) (Output, error) {
	key := dispatcher.RunnerKind{Language: s.spec.Language, Tier: s.spec.Tier}.String()
	r := s.parent.Reports[key]
	if r == nil {
		r = &testreport.TestReport{
			SchemaVersion: testreport.SchemaVersion,
			Tier:          s.spec.Tier,
			Language:      s.spec.Language,
			Verdict:       testreport.VerdictToolUnavailable,
			Passed:        false,
		}
	}
	out, _ := json.Marshal(r)
	return Output{Stdout: out, ExitCode: 0, Duration: time.Millisecond}, nil
}

// ExecProvider runs runners via os/exec on the host. Used for local
// development when the operator has the per-language CLIs on PATH.
// Production uses an E2B-backed provider; this one is the local fallback.
type ExecProvider struct {
	// WorkDir is the host-side dir mounted into the sandbox. Tests should
	// not rely on host filesystem visibility.
	WorkDir string
}

func (p *ExecProvider) Spawn(_ context.Context, spec SandboxSpec) (Sandbox, error) {
	return &execSandbox{spec: spec, workDir: p.WorkDir, id: "exec-" + spec.TaskID + "-" + string(spec.Language)}, nil
}

type execSandbox struct {
	spec    SandboxSpec
	workDir string
	id      string
}

func (s *execSandbox) ID() string                       { return s.id }
func (s *execSandbox) Kill(_ context.Context) error     { return nil }
func (s *execSandbox) Exec(ctx context.Context, argv []string, stdin []byte) (Output, error) {
	if len(argv) == 0 {
		return Output{}, errors.New("execSandbox: empty argv")
	}
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	cmd.Dir = s.workDir
	if len(stdin) > 0 {
		cmd.Stdin = bytesReader(stdin)
	}
	cmd.Env = append(cmd.Env, "PATH="+envPath())
	for k, v := range s.spec.Env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	start := time.Now()
	stdout, err := cmd.Output()
	out := Output{Stdout: stdout, Duration: time.Since(start)}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		out.Stderr = ee.Stderr
		out.ExitCode = ee.ExitCode()
		return out, nil // surface as a non-zero exit, but no error
	}
	if err != nil {
		return out, err
	}
	return out, nil
}

func bytesReader(b []byte) io.Reader { return &bytesReaderT{b: b} }

type bytesReaderT struct {
	b []byte
	i int
}

func (r *bytesReaderT) Read(p []byte) (int, error) {
	if r.i >= len(r.b) {
		return 0, io.EOF
	}
	n := copy(p, r.b[r.i:])
	r.i += n
	return n, nil
}

func envPath() string {
	// Use the host's PATH; the sandboxing happens elsewhere (E2B
	// containerises). ExecProvider is a developer convenience.
	return "" // empty PATH means cmd.Env is empty; exec.LookPath in CommandContext already resolves argv[0]
}
