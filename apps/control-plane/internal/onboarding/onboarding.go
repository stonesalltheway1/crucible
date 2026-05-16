// Package onboarding implements the 4-stage flow per
// docs/04-operations/onboarding.md.
//
//   1. Install: GitHub App handler → tenant provisioning;
//      Slack workspace OAuth handler.
//   2. Cartography: kick the apps/cartographer pipeline.
//   3. First verified PR: surface first-task suggestions; track time-
//      to-first-PR.
//   4. Convention bootstrap: rolling distillation; weekly digest;
//      day-1 / day-2 / day-5 / day-30 customer-success outreach
//      hooks.
//
// All persistence here is in-memory at first; the production wiring
// substitutes the existing control-plane store. Webhook signature
// verification follows the Phase-7 GitHub-App and Slack-bot patterns.
package onboarding

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Tenant tracks the per-tenant onboarding state.
type Tenant struct {
	ID            string    `json:"id"`
	Slug          string    `json:"slug"`
	Email         string    `json:"primary_contact_email"`
	GitHubInstalled bool    `json:"github_installed"`
	GitHubInstallationID int64 `json:"github_installation_id,omitempty"`
	SlackWorkspace string  `json:"slack_workspace,omitempty"`
	Sources        Sources `json:"sources"`
	CreatedAt      time.Time `json:"created_at"`
	Stage          string    `json:"stage"` // install | cartography | first_pr | bootstrap
	FirstTaskSubmittedAt time.Time `json:"first_task_submitted_at,omitempty"`
	FirstVerifiedPRAt    time.Time `json:"first_verified_pr_at,omitempty"`
}

// Sources tracks data-source adapter wiring per docs/onboarding §1.
type Sources struct {
	GitHubPRReviewComments bool `json:"github_pr_review_comments"`
	LinearIncidents        bool `json:"linear_incidents"`
	JiraIncidents          bool `json:"jira_incidents"`
	SlackIncidents         bool `json:"slack_incidents"`
	Confluence             bool `json:"confluence"`
	Notion                 bool `json:"notion"`
}

// CartographerLauncher is the contract the onboarding flow uses to
// kick the cartographer. Apps wire it via an HTTP client.
type CartographerLauncher interface {
	Launch(ctx context.Context, tenantID, repo, repoLocalPath string) (jobID string, err error)
}

// SuggestionEngine returns first-task suggestions for the tenant.
type SuggestionEngine interface {
	Suggest(tenantID, repo string) ([]Suggestion, error)
}

// Suggestion is a "good first task" recommendation.
type Suggestion struct {
	Title        string  `json:"title"`
	Rationale    string  `json:"rationale"`
	EstUSD       float64 `json:"est_usd"`
	EstWallMin   int     `json:"est_wall_min"`
	WhySafeFirst string  `json:"why_safe_first"`
}

// DigestSender sends weekly Friday digests.
type DigestSender interface {
	Send(ctx context.Context, tenant Tenant, body string) error
}

// CSOutreachHook is the customer-success notification hook.
type CSOutreachHook func(ctx context.Context, tenant Tenant, day int) error

// Service is the onboarding service.
type Service struct {
	mu         sync.Mutex
	tenants    map[string]*Tenant
	cartog     CartographerLauncher
	suggester  SuggestionEngine
	digest     DigestSender
	cs         CSOutreachHook
	clock      func() time.Time
	ghSecret   []byte
	slackSecret []byte
}

// Config wires the service.
type Config struct {
	Cartographer    CartographerLauncher
	Suggester       SuggestionEngine
	Digest          DigestSender
	CS              CSOutreachHook
	GitHubAppSecret string
	SlackSecret     string
	Now             func() time.Time
}

// NewService returns a Service.
func NewService(cfg Config) *Service {
	now := cfg.Now
	if now == nil {
		now = time.Now
	}
	return &Service{
		tenants:    map[string]*Tenant{},
		cartog:     cfg.Cartographer,
		suggester:  cfg.Suggester,
		digest:     cfg.Digest,
		cs:         cfg.CS,
		clock:      now,
		ghSecret:   []byte(cfg.GitHubAppSecret),
		slackSecret: []byte(cfg.SlackSecret),
	}
}

// CreateTenant provisions a tenant. Stage is "install" until both the
// GitHub App is installed and the Cartographer has run.
func (s *Service) CreateTenant(slug, email string) (*Tenant, error) {
	if slug == "" || email == "" {
		return nil, errors.New("onboarding: slug + email required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.tenants[slug]; exists {
		return nil, fmt.Errorf("onboarding: tenant %q already exists", slug)
	}
	t := &Tenant{
		ID: "ten_" + simpleHash(slug+email),
		Slug:    slug,
		Email:   email,
		CreatedAt: s.clock().UTC(),
		Stage:   "install",
	}
	s.tenants[slug] = t
	return t, nil
}

// HandleGitHubAppInstall is invoked by the github-app webhook on the
// `installation.created` event.
func (s *Service) HandleGitHubAppInstall(slug string, installationID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tenants[slug]
	if !ok {
		return fmt.Errorf("onboarding: tenant %q not found", slug)
	}
	t.GitHubInstalled = true
	t.GitHubInstallationID = installationID
	t.Sources.GitHubPRReviewComments = true
	if t.Stage == "install" {
		t.Stage = "cartography"
	}
	return nil
}

// VerifyGitHubWebhook validates the X-Hub-Signature-256 header.
func (s *Service) VerifyGitHubWebhook(signature, body []byte) bool {
	if len(s.ghSecret) == 0 {
		return false
	}
	mac := hmac.New(sha256.New, s.ghSecret)
	mac.Write(body)
	want := append([]byte("sha256="), []byte(hex.EncodeToString(mac.Sum(nil)))...)
	return hmac.Equal(signature, want)
}

// HandleSlackOAuth wires a Slack workspace.
func (s *Service) HandleSlackOAuth(slug, workspaceName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tenants[slug]
	if !ok {
		return fmt.Errorf("onboarding: tenant %q not found", slug)
	}
	t.SlackWorkspace = workspaceName
	t.Sources.SlackIncidents = true
	return nil
}

// VerifySlackSignature validates the X-Slack-Signature header.
func (s *Service) VerifySlackSignature(timestamp, signature string, body []byte) bool {
	if len(s.slackSecret) == 0 || timestamp == "" || signature == "" {
		return false
	}
	mac := hmac.New(sha256.New, s.slackSecret)
	mac.Write([]byte("v0:" + timestamp + ":"))
	mac.Write(body)
	want := "v0=" + hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(signature), []byte(want))
}

// WireSource opt-in flips a source flag.
func (s *Service) WireSource(slug, source string, on bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tenants[slug]
	if !ok {
		return fmt.Errorf("onboarding: tenant %q not found", slug)
	}
	switch source {
	case "github_pr_review_comments":
		t.Sources.GitHubPRReviewComments = on
	case "linear_incidents":
		t.Sources.LinearIncidents = on
	case "jira_incidents":
		t.Sources.JiraIncidents = on
	case "slack_incidents":
		t.Sources.SlackIncidents = on
	case "confluence":
		t.Sources.Confluence = on
	case "notion":
		t.Sources.Notion = on
	default:
		return fmt.Errorf("onboarding: unknown source %q", source)
	}
	return nil
}

// LaunchCartographer kicks the cartographer for a repo.
func (s *Service) LaunchCartographer(ctx context.Context, slug, repo, repoLocalPath string) (string, error) {
	s.mu.Lock()
	t, ok := s.tenants[slug]
	s.mu.Unlock()
	if !ok {
		return "", fmt.Errorf("onboarding: tenant %q not found", slug)
	}
	if !t.GitHubInstalled {
		return "", errors.New("onboarding: install the GitHub App before kicking cartography")
	}
	if s.cartog == nil {
		return "", errors.New("onboarding: cartographer launcher not wired")
	}
	return s.cartog.Launch(ctx, t.ID, repo, repoLocalPath)
}

// FirstTaskSuggestions is what the dashboard surfaces post-Cartography.
func (s *Service) FirstTaskSuggestions(slug, repo string) ([]Suggestion, error) {
	if s.suggester == nil {
		return nil, errors.New("onboarding: suggester not wired")
	}
	s.mu.Lock()
	t, ok := s.tenants[slug]
	s.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("onboarding: tenant %q not found", slug)
	}
	return s.suggester.Suggest(t.ID, repo)
}

// MarkFirstTaskSubmitted records the day the first task was submitted.
func (s *Service) MarkFirstTaskSubmitted(slug string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tenants[slug]
	if !ok {
		return fmt.Errorf("onboarding: tenant %q not found", slug)
	}
	if t.FirstTaskSubmittedAt.IsZero() {
		t.FirstTaskSubmittedAt = s.clock().UTC()
	}
	return nil
}

// MarkFirstVerifiedPR records the day the first verified PR landed.
func (s *Service) MarkFirstVerifiedPR(slug string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tenants[slug]
	if !ok {
		return fmt.Errorf("onboarding: tenant %q not found", slug)
	}
	if t.FirstVerifiedPRAt.IsZero() {
		t.FirstVerifiedPRAt = s.clock().UTC()
	}
	t.Stage = "bootstrap"
	return nil
}

// Tenant returns a snapshot of the tenant.
func (s *Service) Tenant(slug string) (*Tenant, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tenants[slug]
	if !ok {
		return nil, false
	}
	cp := *t
	return &cp, true
}

// TouchpointDue returns the days-since-create cohort the touchpoint
// scheduler should fire for. Days 1, 2, 5, 30 per docs/onboarding §
// "Onboarding touchpoints".
func TouchpointDue(daysSinceCreate int) int {
	for _, d := range []int{1, 2, 5, 30} {
		if daysSinceCreate == d {
			return d
		}
	}
	return 0
}

// RunWeeklyDigest sends Friday digests for every tenant.
func (s *Service) RunWeeklyDigest(ctx context.Context) (int, error) {
	if s.digest == nil {
		return 0, nil
	}
	s.mu.Lock()
	tenants := make([]Tenant, 0, len(s.tenants))
	for _, t := range s.tenants {
		tenants = append(tenants, *t)
	}
	s.mu.Unlock()
	count := 0
	for _, t := range tenants {
		body := digestBodyFor(t)
		if err := s.digest.Send(ctx, t, body); err == nil {
			count++
		}
	}
	return count, nil
}

// RunCustomerSuccessTouchpoints fires any due CS hooks.
func (s *Service) RunCustomerSuccessTouchpoints(ctx context.Context) (int, error) {
	if s.cs == nil {
		return 0, nil
	}
	s.mu.Lock()
	tenants := make([]Tenant, 0, len(s.tenants))
	for _, t := range s.tenants {
		tenants = append(tenants, *t)
	}
	s.mu.Unlock()
	count := 0
	now := s.clock().UTC()
	for _, t := range tenants {
		days := int(now.Sub(t.CreatedAt).Hours() / 24)
		if d := TouchpointDue(days); d > 0 {
			if err := s.cs(ctx, t, d); err == nil {
				count++
			}
		}
	}
	return count, nil
}

func digestBodyFor(t Tenant) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Crucible weekly digest — %s\n\n", t.Slug)
	if t.FirstVerifiedPRAt.IsZero() {
		fmt.Fprintf(&sb, "No verified PRs landed yet. Stage: %s.\n", t.Stage)
		fmt.Fprintf(&sb, "Suggested next: review the inferred AGENTS.md at https://app.crucible.dev/memory and pick one of the first-task suggestions.\n")
	} else {
		fmt.Fprintf(&sb, "Crucible has been operating in your codebase since %s.\n", t.FirstVerifiedPRAt.Format("2006-01-02"))
		fmt.Fprintf(&sb, "Memory growth, conventions confirmed, and convention drift will be summarised here.\n")
	}
	return sb.String()
}

// HandleGitHubInstallationPayload parses an installation.created payload.
func HandleGitHubInstallationPayload(body []byte) (slug string, installationID int64, err error) {
	var p struct {
		Action       string `json:"action"`
		Installation struct {
			ID      int64 `json:"id"`
			Account struct {
				Login string `json:"login"`
			} `json:"account"`
		} `json:"installation"`
	}
	if e := json.Unmarshal(body, &p); e != nil {
		return "", 0, e
	}
	if p.Action != "created" {
		return "", 0, fmt.Errorf("onboarding: ignored github action %q", p.Action)
	}
	return p.Installation.Account.Login, p.Installation.ID, nil
}

// simpleHash is a stable, dependency-free 32-bit hash for tenant IDs.
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
