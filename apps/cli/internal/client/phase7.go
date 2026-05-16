// Phase-7 client surface for the CLI.
//
// These methods map 1:1 to control-plane / promotion-gate / memory-router /
// attestation-relay routes added in Phases 5–6 and surfaced through the
// CLI in Phase 7. Existing Phase-1 client.go methods cover task/plan/budget;
// this file adds promote/memory/attestation/webhook/tenant/calibrate/release.

package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// ── promotions ─────────────────────────────────────────────────────────────

type PromotionDecision struct {
	Allow      bool     `json:"allow"`
	NeedsHuman bool     `json:"needs_human"`
	Reasons    []string `json:"reasons,omitempty"`
	Trace      struct {
		Path       string `json:"path"`
		PolicyHash string `json:"policy_hash"`
	} `json:"trace"`
}

type PromotionBundle struct {
	DiffHash         string `json:"diff_hash"`
	AgentOidcSubject string `json:"agent_oidc_subject"`
	FilesChanged     []struct {
		Path   string `json:"path"`
		Action string `json:"action"`
	} `json:"files_changed"`
}

type PromotionCanary struct {
	Adapter      string `json:"adapter"`
	CurrentStep  int    `json:"current_step"`
	Steps        []struct {
		Weight       int    `json:"weight"`
		DwellSeconds int    `json:"dwell_seconds"`
		SloCheck     string `json:"slo_check"`
	} `json:"steps"`
}

type Promotion struct {
	ID       string             `json:"id"`
	TaskID   string             `json:"task_id"`
	Status   string             `json:"status"`
	Decision *PromotionDecision `json:"decision,omitempty"`
	Bundle   *PromotionBundle   `json:"bundle,omitempty"`
	Canary   *PromotionCanary   `json:"canary,omitempty"`
}

func (c *Client) ListPromotions(ctx context.Context, status string) ([]*Promotion, error) {
	q := url.Values{}
	if status != "" {
		q.Set("status", status)
	}
	var resp struct{ Promotions []*Promotion `json:"promotions"` }
	if err := c.do(ctx, http.MethodGet, "/v1/tenants/"+c.tenant+"/promotions?"+q.Encode(), nil, &resp); err != nil {
		return nil, err
	}
	return resp.Promotions, nil
}

func (c *Client) GetPromotion(ctx context.Context, id string) (*Promotion, error) {
	var p Promotion
	if err := c.do(ctx, http.MethodGet, "/v1/tenants/"+c.tenant+"/promotions/"+url.PathEscape(id), nil, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func (c *Client) ApprovePromotion(ctx context.Context, id, group, bundleHash string) (*Promotion, error) {
	var resp struct{ Promotion *Promotion `json:"promotion"` }
	if err := c.do(ctx, http.MethodPost, "/v1/tenants/"+c.tenant+"/promotions/"+url.PathEscape(id)+"/approve",
		map[string]string{"group": group, "bundle_hash_bound": bundleHash}, &resp); err != nil {
		return nil, err
	}
	return resp.Promotion, nil
}

func (c *Client) RejectPromotion(ctx context.Context, id, reason string) error {
	return c.do(ctx, http.MethodPost, "/v1/tenants/"+c.tenant+"/promotions/"+url.PathEscape(id)+"/reject",
		map[string]string{"reason": reason}, nil)
}

func (c *Client) RollbackPromotion(ctx context.Context, id, reason string) error {
	return c.do(ctx, http.MethodPost, "/v1/tenants/"+c.tenant+"/promotions/"+url.PathEscape(id)+"/rollback",
		map[string]string{"reason": reason}, nil)
}

// ── memory ─────────────────────────────────────────────────────────────────

type RecalledMemory struct {
	ID     string  `json:"id"`
	RuleNl string  `json:"rule_nl"`
	Score  float64 `json:"score"`
}

type Convention struct {
	ID                  string  `json:"id"`
	Status              string  `json:"status"`
	Confidence          float64 `json:"confidence"`
	Category            string  `json:"category"`
	RuleNl              string  `json:"rule_nl"`
	PositiveExamples30d int     `json:"positive_examples_30d"`
	NegativeExamples30d int     `json:"negative_examples_30d"`
	LastViolatedAt      string  `json:"last_violated_at,omitempty"`
}

func (c *Client) MemoryRecall(ctx context.Context, query, scope string) ([]*RecalledMemory, error) {
	body := map[string]any{"query": query}
	if scope != "" {
		body["scope"] = parseScope(scope)
	}
	var resp struct{ Memories []*RecalledMemory `json:"memories"` }
	if err := c.do(ctx, http.MethodPost, "/v1/tenants/"+c.tenant+"/memory/recall", body, &resp); err != nil {
		return nil, err
	}
	return resp.Memories, nil
}

func (c *Client) MemoryNote(ctx context.Context, rule, category, scope string) error {
	body := map[string]any{"rule_nl": rule, "category": category}
	if scope != "" {
		body["scope"] = parseScope(scope)
	}
	return c.do(ctx, http.MethodPost, "/v1/tenants/"+c.tenant+"/memory/note", body, nil)
}

func (c *Client) ListConventions(ctx context.Context, status string) ([]*Convention, error) {
	q := url.Values{}
	if status != "" {
		q.Set("status", status)
	}
	var resp struct{ Conventions []*Convention `json:"conventions"` }
	if err := c.do(ctx, http.MethodGet, "/v1/tenants/"+c.tenant+"/memory/conventions?"+q.Encode(), nil, &resp); err != nil {
		return nil, err
	}
	return resp.Conventions, nil
}

func parseScope(s string) map[string]string {
	// Accepts "repo:foo, file:bar/**, category:security" → keys
	out := map[string]string{}
	for _, part := range splitCSV(s) {
		k, v, ok := cutColon(part)
		if !ok {
			continue
		}
		switch k {
		case "repo":
			out["repo"] = v
		case "file":
			out["file_glob"] = v
		case "category":
			out["category"] = v
		}
	}
	return out
}

func splitCSV(s string) []string {
	out := []string{}
	cur := ""
	for _, r := range s {
		if r == ',' {
			out = append(out, trimSpace(cur))
			cur = ""
		} else {
			cur += string(r)
		}
	}
	if cur != "" {
		out = append(out, trimSpace(cur))
	}
	return out
}

func cutColon(s string) (string, string, bool) {
	for i := 0; i < len(s); i++ {
		if s[i] == ':' {
			return s[:i], s[i+1:], true
		}
	}
	return s, "", false
}

func trimSpace(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	return s
}

// ── attestations ───────────────────────────────────────────────────────────

type Attestation struct {
	RekorUUID     string         `json:"rekor_uuid"`
	PredicateType string         `json:"predicate_type"`
	SignedAt      string         `json:"signed_at"`
	SignedByOIDC  string         `json:"signed_by_oidc"`
	Validation    string         `json:"validation"`
	SelfHosted    bool           `json:"self_hosted"`
	Subject       map[string]any `json:"subject"`
	Predicate     any            `json:"predicate"`
}

type VerifyResult struct {
	Verified bool `json:"verified"`
	Details  struct {
		InclusionProofValid  bool `json:"inclusion_proof_valid"`
		CertChainValid       bool `json:"cert_chain_valid"`
		SubjectDigestMatches bool `json:"subject_digest_matches"`
		SelfHosted           bool `json:"self_hosted"`
	} `json:"details"`
}

type AttestationChain struct {
	TaskID string `json:"task_id"`
	Nodes  []struct {
		RekorUUID     string `json:"rekor_uuid"`
		PredicateType string `json:"predicate_type"`
		SignedAt      string `json:"signed_at"`
		Label         string `json:"label"`
	} `json:"nodes"`
}

func (c *Client) GetAttestation(ctx context.Context, uuid string) (*Attestation, error) {
	var a Attestation
	if err := c.do(ctx, http.MethodGet, "/v1/attestations/"+url.PathEscape(uuid), nil, &a); err != nil {
		return nil, err
	}
	return &a, nil
}

func (c *Client) VerifyAttestation(ctx context.Context, uuid string) (*VerifyResult, error) {
	var r VerifyResult
	if err := c.do(ctx, http.MethodPost, "/v1/attestations/"+url.PathEscape(uuid)+"/verify", map[string]any{}, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

func (c *Client) GetAttestationChain(ctx context.Context, taskID string) (*AttestationChain, error) {
	var ch AttestationChain
	if err := c.do(ctx, http.MethodGet, "/v1/tenants/"+c.tenant+"/tasks/"+url.PathEscape(taskID)+"/attestation-chain", nil, &ch); err != nil {
		return nil, err
	}
	return &ch, nil
}

func (c *Client) ExportAttestationBundle(ctx context.Context, taskID string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		c.endpoint+"/v1/tenants/"+c.tenant+"/tasks/"+url.PathEscape(taskID)+"/attestation-bundle.tar", nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("export bundle: %d", resp.StatusCode)
	}
	buf := make([]byte, 0, 64*1024)
	tmp := make([]byte, 32*1024)
	for {
		n, err := resp.Body.Read(tmp)
		buf = append(buf, tmp[:n]...)
		if err != nil {
			break
		}
	}
	return buf, nil
}

// ── webhooks ───────────────────────────────────────────────────────────────

type WebhookSub struct {
	ID            string   `json:"id"`
	URL           string   `json:"url"`
	Events        []string `json:"events"`
	Active        bool     `json:"active"`
	SigningSecret string   `json:"signing_secret,omitempty"`
}

func (c *Client) CreateWebhook(ctx context.Context, urlStr string, events []string, description string) (*WebhookSub, error) {
	var s WebhookSub
	if err := c.do(ctx, http.MethodPost, "/v1/tenants/"+c.tenant+"/webhooks/subscriptions",
		map[string]any{"url": urlStr, "events": events, "description": description, "active": true}, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func (c *Client) ListWebhooks(ctx context.Context) ([]*WebhookSub, error) {
	var resp struct{ Subscriptions []*WebhookSub `json:"subscriptions"` }
	if err := c.do(ctx, http.MethodGet, "/v1/tenants/"+c.tenant+"/webhooks/subscriptions", nil, &resp); err != nil {
		return nil, err
	}
	return resp.Subscriptions, nil
}

func (c *Client) RedeliverWebhook(ctx context.Context, sub, event string) error {
	return c.do(ctx, http.MethodPost,
		"/v1/tenants/"+c.tenant+"/webhooks/subscriptions/"+url.PathEscape(sub)+"/redeliver",
		map[string]any{"event_ids": []string{event}}, nil)
}

// ── tenant config ──────────────────────────────────────────────────────────

func (c *Client) GetTenantConfig(ctx context.Context) (json.RawMessage, error) {
	var raw json.RawMessage
	if err := c.do(ctx, http.MethodGet, "/v1/tenants/"+c.tenant+"/config", nil, &raw); err != nil {
		return nil, err
	}
	return raw, nil
}

func (c *Client) SetTenantConfig(ctx context.Context, cfg any) error {
	return c.do(ctx, http.MethodPost, "/v1/tenants/"+c.tenant+"/config", cfg, nil)
}

// ── calibration ────────────────────────────────────────────────────────────

type CalibrationResult struct {
	Samples int     `json:"samples"`
	AucMean float64 `json:"auc_mean"`
	Weights []struct {
		Path   string  `json:"path"`
		Weight float64 `json:"weight"`
	} `json:"weights"`
}

func (c *Client) Calibrate(ctx context.Context, samples int, dryRun bool) (*CalibrationResult, error) {
	var r CalibrationResult
	if err := c.do(ctx, http.MethodPost, "/v1/tenants/"+c.tenant+"/calibrate",
		map[string]any{"samples": samples, "dry_run": dryRun}, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

// ── public verify-release (no tenant) ──────────────────────────────────────

type PublicClient struct {
	endpoint string
	http     *http.Client
}

func NewPublic(endpoint string) *PublicClient {
	if endpoint == "" {
		endpoint = "https://attest.crucible.dev"
	}
	return &PublicClient{endpoint: endpoint, http: &http.Client{Timeout: 5 * time.Minute}}
}

type ReleaseVerification struct {
	RekorUUID      string `json:"rekor_uuid"`
	ExpectedDigest string `json:"expected_sha256"`
	RebuildDigest  string `json:"rebuild_sha256"`
	SignerOIDC     string `json:"signer_oidc"`
	Match          bool   `json:"match"`
}

func (c *PublicClient) VerifyRelease(ctx context.Context, version string) (*ReleaseVerification, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.endpoint+"/v1/releases/"+url.PathEscape(version)+"/verify", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out ReleaseVerification
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}
