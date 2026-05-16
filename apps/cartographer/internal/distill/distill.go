// Package distill is the Haiku-4.5-driven LLM pass that turns raw
// natural-language sources (PR comments, ADRs, AGENTS.md bullets,
// CONTRIBUTING.md sections) into typed ConventionCandidate rows.
//
// The schema-constrained prompt is the AdaKGC SDD pattern from
// docs/06-research/memory-bootstrap.md §"Extraction model + prompt":
//
//   Given this excerpt from {source_type},
//   extract zero or more enforceable rules. Output JSON array of:
//     { category, rule, file_glob, rationale, evidence_quote }
//   Emit nothing if no enforceable convention is stated.
//
// We retry once on schema validation failure, then drop. Costs are
// bounded by the per-call token cap and the per-batch concurrency
// limit. The pricing assumption is Haiku 4.5 ($0.25/M input,
// $1.25/M output) per the Phase-5 memory-bootstrap docs; the
// caller's budget enforcer (control-plane budgetenforcer) is the
// source of truth for the running total.
package distill

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/crucible/apps/cartographer/internal/types"
)

// Provider selects which backend the Client calls.
type Provider string

const (
	// ProviderDistillerService routes through services/distiller (a
	// Crucible-managed Python sidecar). Default for production.
	ProviderDistillerService Provider = "distiller"
	// ProviderAnthropic calls the Anthropic Messages API directly.
	// Used when no Crucible distiller is deployed (single-service
	// rollouts like the Phase-8 v1 fly.io deployment).
	ProviderAnthropic Provider = "anthropic"
	// ProviderOffline returns no candidates. Used when no LLM is
	// reachable (air-gap dev, hermetic tests).
	ProviderOffline Provider = "offline"
)

// Client wraps the distiller HTTP endpoint OR a direct LLM endpoint.
type Client struct {
	Provider     Provider
	Endpoint     string
	Model        string
	APIKey       string // only used when Provider == ProviderAnthropic
	Timeout      time.Duration
	HTTP         *http.Client
	now          func() time.Time
	mu           sync.Mutex // serialises retry counters
	calls        int
}

// Config configures NewClient.
type Config struct {
	// Endpoint is the distiller service URL (provider=distiller) OR
	// the Anthropic API base URL (provider=anthropic, default
	// "https://api.anthropic.com"). Empty + no APIKey + no Provider
	// override = offline mode.
	Endpoint string
	Model    string
	Timeout  time.Duration
	// APIKey, when set, switches the client to ProviderAnthropic and
	// authenticates the Messages API call.
	APIKey   string
	Provider Provider
}

// NewClient builds a Client. The provider is auto-detected from the
// config: APIKey set → Anthropic; Endpoint set without APIKey →
// distiller service; neither → offline.
func NewClient(cfg Config) *Client {
	to := cfg.Timeout
	if to == 0 {
		to = 5 * time.Minute
	}
	provider := cfg.Provider
	if provider == "" {
		switch {
		case cfg.APIKey != "":
			provider = ProviderAnthropic
		case cfg.Endpoint != "":
			provider = ProviderDistillerService
		default:
			provider = ProviderOffline
		}
	}
	endpoint := cfg.Endpoint
	if provider == ProviderAnthropic && endpoint == "" {
		endpoint = "https://api.anthropic.com"
	}
	model := cfg.Model
	if provider == ProviderAnthropic && model == "" {
		model = "claude-haiku-4-5-20251001"
	}
	return &Client{
		Provider: provider,
		Endpoint: endpoint,
		Model:    model,
		APIKey:   cfg.APIKey,
		Timeout:  to,
		HTTP:     &http.Client{Timeout: to},
		now:      time.Now,
	}
}

// Excerpt is one piece of source text we ask the LLM to distill.
type Excerpt struct {
	Repo         string
	TenantID     string
	SourceChannel string // "pr_comment" | "adr_file" | "agents_md" | "contributing_md" | "incident"
	SourcePath   string
	Body         string
}

// rawRule is the schema the LLM produces.
type rawRule struct {
	Category      string `json:"category"`
	Rule          string `json:"rule"`
	FileGlob      string `json:"file_glob"`
	Rationale     string `json:"rationale"`
	EvidenceQuote string `json:"evidence_quote"`
}

// validCategories matches the 12-category taxonomy from
// docs/06-research/memory-bootstrap.md §"The 12-category taxonomy".
var validCategories = map[string]bool{
	"Naming":               true,
	"Layering":             true,
	"Library preferences":  true,
	"Test patterns":        true,
	"Error handling":       true,
	"Logging":              true,
	"Migration patterns":   true,
	"PR/commit hygiene":    true,
	"Security defaults":    true,
	"Performance defaults": true,
	"Concurrency":          true,
	"API shape":            true,
}

const distillPromptTemplate = `You are extracting enforceable engineering conventions from a source excerpt.

Given this excerpt from %s, extract zero or more enforceable rules.

Output a JSON array of objects with these fields:
  category   — one of: %s
  rule       — a concise imperative statement of the convention (max 200 chars)
  file_glob  — the glob the rule applies to (e.g. "**/*.go", "src/**/*.tsx", or "**/*" for repo-wide)
  rationale  — one sentence explaining why
  evidence_quote — the most-relevant verbatim phrase from the excerpt (max 240 chars)

Emit an empty array [] if no enforceable convention is stated.

Excerpt:
%s

Return ONLY the JSON array, no prose.`

// Distill runs one excerpt through the LLM and returns the parsed
// candidates. On schema-validation failure we retry once, then drop.
func (c *Client) Distill(ctx context.Context, ex Excerpt) ([]types.ConventionCandidate, error) {
	if c.Provider == ProviderOffline || (c.Provider == ProviderDistillerService && c.Endpoint == "") {
		return offlineDistill(ex), nil
	}
	body := buildPrompt(ex)
	rules, err := c.callOnce(ctx, body)
	if err != nil {
		// One retry with a schema-reminder suffix.
		body2 := body + "\n\nSCHEMA REMINDER: every object MUST include all five fields and `category` MUST be one of the listed values."
		rules, err = c.callOnce(ctx, body2)
		if err != nil {
			return nil, err
		}
	}
	out := make([]types.ConventionCandidate, 0, len(rules))
	for _, r := range rules {
		if !validCategories[r.Category] || strings.TrimSpace(r.Rule) == "" {
			continue
		}
		out = append(out, types.ConventionCandidate{
			ID:            "c_llm_" + simpleHash(ex.Repo+r.Rule+ex.SourcePath),
			Category:      r.Category,
			RuleNL:        clip(r.Rule, 200),
			FileGlob:      orDefault(r.FileGlob, "**/*"),
			Rationale:     clip(r.Rationale, 240),
			EvidenceQuote: clip(r.EvidenceQuote, 240),
			SourceChannel: ex.SourceChannel,
			SourcePath:    ex.SourcePath,
			Confidence:    confFor(ex.SourceChannel),
			Status:        "candidate",
			FirstSeen:     c.now().UTC(),
		})
	}
	return out, nil
}

// DistillBatch runs a batch in parallel with bounded concurrency.
func (c *Client) DistillBatch(ctx context.Context, exs []Excerpt, concurrency int) ([]types.ConventionCandidate, error) {
	if concurrency < 1 {
		concurrency = 4
	}
	if len(exs) == 0 {
		return nil, nil
	}
	jobs := make(chan Excerpt)
	results := make(chan []types.ConventionCandidate)
	errs := make(chan error, len(exs))
	wg := sync.WaitGroup{}
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for ex := range jobs {
				if ctx.Err() != nil {
					return
				}
				cands, err := c.Distill(ctx, ex)
				if err != nil {
					errs <- err
					continue
				}
				if len(cands) > 0 {
					results <- cands
				}
			}
		}()
	}
	go func() {
		for _, e := range exs {
			select {
			case <-ctx.Done():
				close(jobs)
				return
			case jobs <- e:
			}
		}
		close(jobs)
	}()
	doneCh := make(chan struct{})
	go func() {
		wg.Wait()
		close(results)
		close(doneCh)
	}()
	var out []types.ConventionCandidate
	for r := range results {
		out = append(out, r...)
	}
	<-doneCh
	close(errs)
	// Fail-soft: only one true error is fatal. Surface it but keep
	// candidates from successful calls.
	var firstErr error
	for e := range errs {
		if firstErr == nil {
			firstErr = e
		}
	}
	return out, firstErr
}

// CallCount returns the number of LLM round-trips made by this client.
func (c *Client) CallCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.calls
}

// --- Internal ---

func (c *Client) callOnce(ctx context.Context, prompt string) ([]rawRule, error) {
	c.mu.Lock()
	c.calls++
	c.mu.Unlock()
	switch c.Provider {
	case ProviderAnthropic:
		return c.callAnthropic(ctx, prompt)
	case ProviderDistillerService, "":
		return c.callDistillerService(ctx, prompt)
	}
	return nil, fmt.Errorf("distill: unknown provider %q", c.Provider)
}

func (c *Client) callDistillerService(ctx context.Context, prompt string) ([]rawRule, error) {
	req := map[string]any{
		"model":             c.Model,
		"prompt":            prompt,
		"max_output_tokens": 4096,
		"temperature":       0.0,
	}
	body, _ := json.Marshal(req)
	httpReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, c.Endpoint+"/v1/distill", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTP.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		buf, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("distiller HTTP %d: %s", resp.StatusCode, string(buf))
	}
	var envelope struct {
		Output string `json:"output"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return nil, err
	}
	return parseRules(envelope.Output)
}

// callAnthropic talks to the Messages API directly. We don't depend
// on the official SDK to keep the cartographer image at distroless
// + ~12 MB.
func (c *Client) callAnthropic(ctx context.Context, prompt string) ([]rawRule, error) {
	req := map[string]any{
		"model":      c.Model,
		"max_tokens": 4096,
		"system":     "You distill engineering conventions. Output ONLY the JSON array specified in the user prompt; no prose, no fenced-code wrapper.",
		"messages": []map[string]any{
			{"role": "user", "content": prompt},
		},
	}
	body, _ := json.Marshal(req)
	httpReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, c.Endpoint+"/v1/messages", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.APIKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	resp, err := c.HTTP.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		buf, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
		return nil, fmt.Errorf("anthropic HTTP %d: %s", resp.StatusCode, string(buf))
	}
	var envelope struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return nil, err
	}
	var combined strings.Builder
	for _, c := range envelope.Content {
		if c.Type == "text" {
			combined.WriteString(c.Text)
		}
	}
	return parseRules(combined.String())
}

func parseRules(raw string) ([]rawRule, error) {
	out := strings.TrimSpace(raw)
	// Strip a fenced-code wrapper if the model emitted one despite the
	// "JSON only" instruction.
	if strings.HasPrefix(out, "```") {
		out = strings.TrimPrefix(out, "```json")
		out = strings.TrimPrefix(out, "```")
		out = strings.TrimSuffix(out, "```")
		out = strings.TrimSpace(out)
	}
	// Some models prepend a sentence even when told not to. Find the
	// first '[' as a salvage path.
	if !strings.HasPrefix(out, "[") {
		if i := strings.IndexByte(out, '['); i >= 0 {
			out = out[i:]
		}
	}
	var rules []rawRule
	if err := json.Unmarshal([]byte(out), &rules); err != nil {
		return nil, fmt.Errorf("distiller: schema-validation failure: %w", err)
	}
	return rules, nil
}

func buildPrompt(ex Excerpt) string {
	cats := categoryList()
	body := clip(ex.Body, 8000)
	return fmt.Sprintf(distillPromptTemplate, ex.SourceChannel, cats, body)
}

func categoryList() string {
	cs := make([]string, 0, len(validCategories))
	for k := range validCategories {
		cs = append(cs, k)
	}
	// Stable order so prompts cache well.
	stableSortStrings(cs)
	return strings.Join(cs, " | ")
}

// confFor maps a source-channel to a base confidence. Final
// confidence is computed in the agreement pass; this is the prior.
func confFor(channel string) float64 {
	switch channel {
	case "adr_file":
		return 0.85
	case "agents_md":
		return 0.85
	case "contributing_md":
		return 0.7
	case "incident":
		return 0.8
	case "pr_comment":
		return 0.45
	}
	return 0.4
}

func clip(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func orDefault(s, d string) string {
	if strings.TrimSpace(s) == "" {
		return d
	}
	return s
}

func stableSortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j-1] > s[j]; j-- {
			s[j-1], s[j] = s[j], s[j-1]
		}
	}
}

// offlineDistill is the deterministic fallback. It produces nothing —
// the offline path relies on the deterministic lintconfig+agentsmd
// passes for production-quality output. We deliberately don't try to
// regex-extract conventions from PR comments here because the false
// positive rate would pollute the per-tenant memory.
func offlineDistill(ex Excerpt) []types.ConventionCandidate {
	return nil
}

func simpleHash(s string) string {
	const (
		offset uint32 = 2166136261
		prime  uint32 = 16777619
	)
	h := offset
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= prime
	}
	const hex = "0123456789abcdef"
	var b [8]byte
	for i := 7; i >= 0; i-- {
		b[i] = hex[h&0xf]
		h >>= 4
	}
	return string(b[:])
}

// Errors.
var (
	ErrNoEndpoint = errors.New("distill: no endpoint configured")
)
