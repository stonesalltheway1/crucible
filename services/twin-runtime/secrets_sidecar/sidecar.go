// Package secretssidecar implements the Crucible secrets-twin layer.
//
// Per the May 2026 currency check on Infisical:
//
//   - Dynamic-secret TTL floor is 5 seconds (NOT arbitrary sub-second).
//   - The `infisical agent` sidecar does NOT do request mutation. We layer
//     our own egress-proxy placeholder substitution on top.
//   - There is no parent-mints-child token API; we use per-sandbox
//     Universal Auth identities with restrictive Client Secret constraints.
//   - Audit-log streaming is Enterprise-only.
//
// The Sidecar runs alongside the sandbox process. The agent calls
// twin.secret.get(name) which the runtime SDK forwards to the sidecar's
// IssueLease endpoint. The agent receives a SecretHandle (NOT the value).
// At egress time the egress proxy substitutes the placeholder
// `$secret(name)$` for the actual lease via the InjectionDirective
// returned alongside the handle.
//
// All API calls are env-gated; if CRUCIBLE_INFISICAL_API_URL +
// CRUCIBLE_INFISICAL_CLIENT_ID + CRUCIBLE_INFISICAL_CLIENT_SECRET are unset
// the sidecar runs in stub mode (typed Phase-2 errors, no live calls).
package secretssidecar

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	EnvAPIURL       = "CRUCIBLE_INFISICAL_API_URL"
	EnvClientID     = "CRUCIBLE_INFISICAL_CLIENT_ID"
	EnvClientSecret = "CRUCIBLE_INFISICAL_CLIENT_SECRET"
	EnvProjectID    = "CRUCIBLE_INFISICAL_PROJECT_ID"

	DefaultAPIURL = "https://app.infisical.com/api"

	// MinTTL — Infisical's documented floor for dynamic-secret TTL.
	MinTTL = 5 * time.Second
)

// Sidecar is the secrets-twin daemon.
type Sidecar interface {
	// IssueLease mints a dynamic credential and returns a typed handle.
	// The raw credential value NEVER appears in the returned struct —
	// only the InjectionDirective the egress proxy uses to substitute
	// at request time.
	IssueLease(ctx context.Context, req LeaseRequest) (Lease, error)

	// RevokeLease tears down the credential before its TTL expires.
	RevokeLease(ctx context.Context, leaseID string) error

	// Resolve is invoked by the egress proxy at request time to obtain
	// the raw value for a SecretRef. The proxy strictly enforces that
	// Resolve is only called from inside the egress process, never the
	// agent process.
	Resolve(ctx context.Context, ref SecretRef) (string, error)
}

// LeaseRequest declares what kind of dynamic credential to mint.
type LeaseRequest struct {
	SandboxID string
	TaskID    string
	TenantID  string
	Name      string        // logical name the agent uses
	VaultPath string        // Infisical path
	Scope     ScopeKind     // Static / DynamicPG / DynamicMySQL / DynamicMongo / DynamicAWS
	TTL       time.Duration // ≥5s
}

// Lease is the typed handle returned to the agent.
type Lease struct {
	ID         string
	Name       string
	SandboxID  string
	IssuedAt   time.Time
	ExpiresAt  time.Time
	// Directive describes how the egress proxy should inject the value
	// without ever returning it to the agent process.
	Directive InjectionDirective
}

// SecretRef is the opaque reference the egress proxy resolves at request time.
type SecretRef struct {
	LeaseID string
	Name    string
}

// ScopeKind enumerates supported scope types.
type ScopeKind string

const (
	ScopeStatic       ScopeKind = "static"
	ScopeDynamicPG    ScopeKind = "dynamic-pg"
	ScopeDynamicMySQL ScopeKind = "dynamic-mysql"
	ScopeDynamicMongo ScopeKind = "dynamic-mongo"
	ScopeDynamicAWS   ScopeKind = "dynamic-aws-iam"
)

// InjectionDirective tells the egress proxy how to inject the value.
type InjectionDirective struct {
	// HeaderName: the request header to set, e.g. "Authorization".
	HeaderName string
	// HeaderFormat: a sprintf template with one `%s` for the value, e.g.
	// `Bearer %s`.
	HeaderFormat string
	// QueryParam: optionally a query parameter to add (mutually exclusive
	// with HeaderName).
	QueryParam string
	// BodyJSONPath: optionally a JSONPath to write the value into the
	// request body (mutually exclusive with header / query).
	BodyJSONPath string
}

// DefaultSidecar is the production implementation.
type DefaultSidecar struct {
	apiURL       string
	clientID     string
	clientSecret string
	projectID    string
	client       *http.Client

	mu     sync.Mutex
	cache  map[string]cachedLease
	bearer cachedBearer
}

type cachedLease struct {
	lease    Lease
	rawValue string
}

type cachedBearer struct {
	token     string
	expiresAt time.Time
}

// New constructs from env. If the required vars are missing it returns a
// stub sidecar — typed Phase-2 error on every method.
func New() Sidecar {
	apiURL := os.Getenv(EnvAPIURL)
	if apiURL == "" {
		apiURL = DefaultAPIURL
	}
	clientID := os.Getenv(EnvClientID)
	clientSecret := os.Getenv(EnvClientSecret)
	if clientID == "" || clientSecret == "" {
		return &stubSidecar{
			msg: fmt.Sprintf(
				"%s or %s unset — sidecar in stub mode. Set both env vars pointing at a Crucible-dedicated Infisical machine identity.",
				EnvClientID, EnvClientSecret,
			),
		}
	}
	return &DefaultSidecar{
		apiURL:       strings.TrimRight(apiURL, "/"),
		clientID:     clientID,
		clientSecret: clientSecret,
		projectID:    os.Getenv(EnvProjectID),
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
		cache: make(map[string]cachedLease),
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// DefaultSidecar implementation
// ─────────────────────────────────────────────────────────────────────────────

func (s *DefaultSidecar) IssueLease(ctx context.Context, req LeaseRequest) (Lease, error) {
	if req.TTL < MinTTL {
		return Lease{}, fmt.Errorf("ttl=%s below Infisical floor of %s", req.TTL, MinTTL)
	}
	if req.Name == "" {
		return Lease{}, errors.New("LeaseRequest.Name empty")
	}
	if req.VaultPath == "" {
		return Lease{}, errors.New("LeaseRequest.VaultPath empty")
	}

	bearer, err := s.acquireBearer(ctx)
	if err != nil {
		return Lease{}, fmt.Errorf("auth: %w", err)
	}

	// Issue a dynamic credential via Infisical's REST API. The endpoint
	// depends on Scope.
	endpoint := fmt.Sprintf("%s/v1/dynamic-secrets/leases", s.apiURL)
	body := map[string]any{
		"projectId":       s.projectID,
		"name":            req.VaultPath,
		"ttl":             req.TTL.Seconds(),
		"environmentSlug": "production",
	}
	var resp struct {
		Lease struct {
			ID        string    `json:"id"`
			ExpiresAt time.Time `json:"expiresAt"`
			// data carries the dynamic credential (username, password, etc.).
			// We never expose this map to the agent — only the egress proxy
			// reads it via Resolve.
			Data map[string]string `json:"data"`
		} `json:"lease"`
	}
	if err := s.do(ctx, "POST", endpoint, bearer, body, &resp); err != nil {
		return Lease{}, fmt.Errorf("issue lease: %w", err)
	}

	leaseID := resp.Lease.ID
	if leaseID == "" {
		leaseID = uuid.NewString()
	}
	expires := resp.Lease.ExpiresAt
	if expires.IsZero() {
		expires = time.Now().Add(req.TTL)
	}

	// We assume the dynamic credential is delivered as `{username, password}`
	// — the agent gets only the InjectionDirective; the raw value lives
	// inside this process's cache and is reachable only via Resolve.
	rawValue := combineCredential(resp.Lease.Data)

	lease := Lease{
		ID:        leaseID,
		Name:      req.Name,
		SandboxID: req.SandboxID,
		IssuedAt:  time.Now(),
		ExpiresAt: expires,
		Directive: directiveFor(req.Scope, req.Name),
	}

	s.mu.Lock()
	s.cache[leaseID] = cachedLease{lease: lease, rawValue: rawValue}
	s.mu.Unlock()

	return lease, nil
}

func (s *DefaultSidecar) RevokeLease(ctx context.Context, leaseID string) error {
	bearer, err := s.acquireBearer(ctx)
	if err != nil {
		return err
	}
	endpoint := fmt.Sprintf("%s/v1/dynamic-secrets/leases/%s", s.apiURL, url.PathEscape(leaseID))
	if err := s.do(ctx, "DELETE", endpoint, bearer, nil, nil); err != nil {
		// 404 ⇒ already gone, treat as idempotent.
		var httpErr *HTTPError
		if errors.As(err, &httpErr) && httpErr.Status == http.StatusNotFound {
			s.mu.Lock()
			delete(s.cache, leaseID)
			s.mu.Unlock()
			return nil
		}
		return err
	}
	s.mu.Lock()
	delete(s.cache, leaseID)
	s.mu.Unlock()
	return nil
}

func (s *DefaultSidecar) Resolve(ctx context.Context, ref SecretRef) (string, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.cache[ref.LeaseID]
	if !ok {
		return "", fmt.Errorf("lease %s not in cache", ref.LeaseID)
	}
	if time.Now().After(entry.lease.ExpiresAt) {
		delete(s.cache, ref.LeaseID)
		return "", fmt.Errorf("lease %s expired", ref.LeaseID)
	}
	return entry.rawValue, nil
}

// acquireBearer mints a Universal-Auth access token. Cached until expiry-30s.
func (s *DefaultSidecar) acquireBearer(ctx context.Context) (string, error) {
	s.mu.Lock()
	if time.Now().Before(s.bearer.expiresAt.Add(-30 * time.Second)) && s.bearer.token != "" {
		token := s.bearer.token
		s.mu.Unlock()
		return token, nil
	}
	s.mu.Unlock()

	endpoint := fmt.Sprintf("%s/v1/auth/universal-auth/login", s.apiURL)
	body := map[string]string{
		"clientId":     s.clientID,
		"clientSecret": s.clientSecret,
	}
	var resp struct {
		AccessToken string `json:"accessToken"`
		ExpiresIn   int    `json:"expiresIn"`
	}
	if err := s.do(ctx, "POST", endpoint, "", body, &resp); err != nil {
		return "", err
	}
	if resp.AccessToken == "" {
		return "", errors.New("universal-auth: empty accessToken")
	}
	exp := time.Now().Add(time.Duration(resp.ExpiresIn) * time.Second)
	s.mu.Lock()
	s.bearer = cachedBearer{token: resp.AccessToken, expiresAt: exp}
	s.mu.Unlock()
	return resp.AccessToken, nil
}

func (s *DefaultSidecar) do(
	ctx context.Context,
	method, urlStr, bearer string,
	body any,
	out any,
) error {
	var reqBody io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal: %w", err)
		}
		reqBody = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, urlStr, reqBody)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return &HTTPError{Status: resp.StatusCode, Body: string(raw), URL: urlStr}
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

func combineCredential(data map[string]string) string {
	if v, ok := data["value"]; ok {
		return v
	}
	if u, p := data["username"], data["password"]; u != "" && p != "" {
		return u + ":" + p
	}
	parts := make([]string, 0, len(data))
	for k, v := range data {
		parts = append(parts, k+"="+v)
	}
	return strings.Join(parts, ";")
}

func directiveFor(scope ScopeKind, name string) InjectionDirective {
	_ = name
	switch scope {
	case ScopeDynamicPG, ScopeDynamicMySQL, ScopeDynamicMongo:
		// DB credentials substitute into the connection URI; the egress
		// proxy uses the BodyJSONPath form when forwarding via libpq, or
		// the header when forwarding via HTTP.
		return InjectionDirective{BodyJSONPath: "$.credentials"}
	case ScopeDynamicAWS:
		return InjectionDirective{
			HeaderName:   "Authorization",
			HeaderFormat: "AWS4-HMAC-SHA256 %s",
		}
	default:
		return InjectionDirective{
			HeaderName:   "Authorization",
			HeaderFormat: "Bearer %s",
		}
	}
}

// HTTPError mirrors the Neon driver's typed HTTP error.
type HTTPError struct {
	Status int
	Body   string
	URL    string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("infisical http %d on %s: %s", e.Status, e.URL, e.Body)
}

// ─────────────────────────────────────────────────────────────────────────────
// Stub for missing-key mode
// ─────────────────────────────────────────────────────────────────────────────

type stubSidecar struct {
	msg string
}

func (s *stubSidecar) IssueLease(ctx context.Context, req LeaseRequest) (Lease, error) {
	_ = ctx
	_ = req
	return Lease{}, &StubError{Msg: s.msg}
}

func (s *stubSidecar) RevokeLease(ctx context.Context, leaseID string) error {
	_ = ctx
	_ = leaseID
	return &StubError{Msg: s.msg}
}

func (s *stubSidecar) Resolve(ctx context.Context, ref SecretRef) (string, error) {
	_ = ctx
	_ = ref
	return "", &StubError{Msg: s.msg}
}

// StubError is the Phase-2 stub-mode error.
type StubError struct {
	Msg string
}

func (e *StubError) Error() string {
	return "STUB: " + e.Msg
}

// IsStub returns true if err is a *StubError.
func IsStub(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(*StubError)
	return ok
}
