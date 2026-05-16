// Package orchestrator runs the full Phase-8 cartographer pipeline:
// walk → symbol-index → lint-config → AGENTS.md/ADR → PR comments →
// incidents → distill → cross-source agreement → OSS-defaults filter
// → inferred AGENTS.md → console output → first-task suggestions.
package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/crucible/apps/cartographer/internal/agentsmd"
	"github.com/crucible/apps/cartographer/internal/agreement"
	"github.com/crucible/apps/cartographer/internal/console"
	"github.com/crucible/apps/cartographer/internal/distill"
	"github.com/crucible/apps/cartographer/internal/incidents"
	"github.com/crucible/apps/cartographer/internal/inferred"
	"github.com/crucible/apps/cartographer/internal/lintconfig"
	"github.com/crucible/apps/cartographer/internal/oss"
	"github.com/crucible/apps/cartographer/internal/prcomments"
	"github.com/crucible/apps/cartographer/internal/stackdetect"
	"github.com/crucible/apps/cartographer/internal/suggest"
	"github.com/crucible/apps/cartographer/internal/symbols"
	"github.com/crucible/apps/cartographer/internal/types"
	"github.com/crucible/apps/cartographer/internal/walker"
)

// Deps wires the external collaborators.
type Deps struct {
	OSS         *oss.Loader
	LLM         *distill.Client
	GitHubToken string // optional — when empty the PR-comments stage is skipped
	Now         func() time.Time
}

// Run executes the full pipeline.
func Run(ctx context.Context, job types.CartographyJob, d Deps, progress func(stage string, frac float64)) (*types.CartographyResult, error) {
	if job.RepoLocalPath == "" {
		return nil, errors.New("orchestrator: repo_local_path required")
	}
	now := d.Now
	if now == nil {
		now = time.Now
	}
	emit := func(stage string, frac float64) {
		if progress != nil {
			progress(stage, frac)
		}
	}
	r := &types.CartographyResult{
		JobID: job.JobID, TenantID: job.TenantID, Repo: job.Repo,
		StartedAt: now().UTC(),
	}

	// Stage 1: walk + classify.
	emit("walking", 0.05)
	files, stats, err := walker.Walk(job.RepoLocalPath, 0, 0)
	if err != nil {
		return nil, err
	}
	r.FilesIndexed = stats.Files
	r.Directories = stats.Directories

	// Stage 2: symbol index.
	emit("indexing-symbols", 0.15)
	idx, err := symbols.Build(ctx, files)
	if err != nil {
		return nil, err
	}
	r.SymbolCount = len(idx.Entries)

	// Stage 3: detect stack.
	emit("detecting-stack", 0.20)
	st := stackdetect.Detect(job.RepoLocalPath)
	if job.StackHint != "" {
		st.Primary = job.StackHint
	}
	r.StackPrimary = st.Primary
	r.StackSecondary = st.Secondary

	// Stage 4: lint configs.
	emit("parsing-lint-configs", 0.30)
	configCands := lintconfig.Run(job.RepoLocalPath, job.Repo, job.TenantID)
	r.ConventionsFromConfigs = len(configCands)

	// Stage 5: AGENTS.md / CLAUDE.md / .cursorrules / CONTRIBUTING.md.
	emit("reading-agents-md", 0.40)
	override, ok := agentsmd.FindCustomerOverride(job.RepoLocalPath)
	r.HasCustomerOverride = ok
	r.CustomerOverridePath = override.Path
	var agentsCands []types.ConventionCandidate
	if ok {
		agentsCands = agentsmd.ExtractFromAgentsMD(job.Repo, job.TenantID, override.Path, override.Body)
	}
	r.ConventionsFromAgentsMD = len(agentsCands)

	var contribCands []types.ConventionCandidate
	for _, name := range agentsmd.ContributingFiles {
		full := filepath.Join(job.RepoLocalPath, name)
		body, rerr := readFile(full)
		if rerr == nil && len(body) > 0 {
			contribCands = append(contribCands, agentsmd.ExtractFromContributing(job.Repo, job.TenantID, name, body)...)
		}
	}
	r.ConventionsFromContributing = len(contribCands)

	// Stage 6: ADRs.
	emit("scanning-adrs", 0.50)
	adrCands := agentsmd.ScanADRDirs(job.Repo, job.TenantID, job.RepoLocalPath)
	r.ConventionsFromADRs = len(adrCands)

	// Stage 7: PR review comments.
	var prCands []types.ConventionCandidate
	var prFetched []prcomments.Comment
	if job.IncludePRHistory && d.GitHubToken != "" {
		emit("scanning-pr-comments", 0.60)
		client := prcomments.NewClient(d.GitHubToken)
		opts := prcomments.DefaultOptions()
		if job.PRHistoryMaxComments > 0 {
			opts.MaxComments = job.PRHistoryMaxComments
		}
		if job.PRHistoryMonths > 0 {
			opts.WindowDays = job.PRHistoryMonths * 30
		}
		fetched, ferr := client.Fetch(ctx, job.Repo, opts)
		if ferr == nil {
			prFetched = fetched
		}
	}
	// Distill PR comments → conventions.
	if len(prFetched) > 0 {
		emit("distilling-pr-comments", 0.70)
		var exs []distill.Excerpt
		for _, c := range prFetched {
			exs = append(exs, distill.Excerpt{
				Repo: job.Repo, TenantID: job.TenantID,
				SourceChannel: "pr_comment",
				SourcePath:    c.URL,
				Body:          c.Body,
			})
		}
		llm, _ := d.LLM.DistillBatch(ctx, exs, 4)
		prCands = append(prCands, llm...)
	}
	r.ConventionsFromPRReview = len(prCands)

	// Stage 8: incidents.
	var incCands []types.ConventionCandidate
	emit("detecting-incidents", 0.80)
	for _, c := range prFetched {
		incCands = append(incCands, incidents.Detect(job.Repo, job.TenantID, c.URL, c.Body)...)
	}
	if ok {
		incCands = append(incCands, incidents.Detect(job.Repo, job.TenantID, override.Path, string(override.Body))...)
	}
	r.ConventionsFromIncidents = len(incCands)

	// Stage 9: cross-source agreement scoring.
	emit("scoring-agreement", 0.85)
	// Aggregate everything except OSS defaults; OSS defaults aren't
	// scored — they're loaded as-is and shown for review.
	all := append([]types.ConventionCandidate{}, configCands...)
	all = append(all, agentsCands...)
	all = append(all, contribCands...)
	all = append(all, adrCands...)
	all = append(all, prCands...)
	all = append(all, incCands...)
	bucket := agreement.Score(all, len(prFetched)+len(adrCands)+1)
	bucket.FilterContradictions()

	// Stage 10: OSS defaults filtered by stack.
	emit("loading-oss-defaults", 0.90)
	var ossCands []types.ConventionCandidate
	if d.OSS != nil && r.StackPrimary != "" {
		ossCands, _ = d.OSS.LoadStack(r.StackPrimary)
	}
	r.ConventionsFromOSSDefaults = len(ossCands)

	hi, md, lo := bucket.Counts()
	r.HighConfidenceCount = hi
	r.MediumConfidenceCount = md
	r.LowConfidenceCount = lo

	r.Sample = sampleN(bucket.High, 10)

	// Stage 11: inferred AGENTS.md (only if no customer override).
	if !ok {
		emit("generating-inferred-agents-md", 0.93)
		r.InferredAgentsMDMarkdown = inferred.Generate(job.Repo, r.StackPrimary, bucket.High, bucket.Medium)
	}

	// Stage 12: first-task suggestions.
	emit("computing-first-tasks", 0.97)
	r.FirstTaskSuggestions = suggest.Suggest(r.StackPrimary, idx, append(bucket.High, ossCands...), 3)

	// Stage 13: web-console line set.
	r.CompletedAt = now().UTC()
	r.WallClockSeconds = r.CompletedAt.Sub(r.StartedAt).Seconds()
	r.UsdSpent = estUSD(r, d.LLM)
	r.TokensSpent = estTokens(d.LLM)
	r.ConsoleOutputLines = console.Lines(r)

	emit("done", 1.0)
	return r, nil
}

func sampleN(in []types.ConventionCandidate, n int) []types.ConventionCandidate {
	if len(in) <= n {
		return append([]types.ConventionCandidate(nil), in...)
	}
	return append([]types.ConventionCandidate(nil), in[:n]...)
}

func estUSD(r *types.CartographyResult, c *distill.Client) float64 {
	calls := 0
	if c != nil {
		calls = c.CallCount()
	}
	// Phase-5 unit-economics estimate: ~$0.003 per Haiku 4.5 call at
	// the cartographer's typical 2-3K-token excerpt size, ignoring
	// cache. Honest under/over: bounds within ±50%.
	return float64(calls) * 0.003
}

func estTokens(c *distill.Client) int {
	if c == nil {
		return 0
	}
	return c.CallCount() * 2400
}

// readFile is a fail-soft wrapper.
func readFile(p string) ([]byte, error) {
	body, err := readFileImpl(p)
	if err != nil {
		return nil, err
	}
	if len(body) == 0 {
		return nil, fmt.Errorf("empty: %s", p)
	}
	return body, nil
}

func readFileImpl(p string) ([]byte, error) {
	// Local file read — kept tiny; the orchestrator imports os in
	// other paths via the agentsmd package, so we don't add another
	// dep here.
	return _readFile(p)
}

// _readFile is defined in a small helper file so this package stays
// imports-clean.
var _ = strings.TrimSpace // keep `strings` referenced in go vet.
