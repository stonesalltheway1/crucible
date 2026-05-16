package onboarding

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"
)

type stubLauncher struct{ id string; err error }

func (s stubLauncher) Launch(_ context.Context, _, _, _ string) (string, error) {
	return s.id, s.err
}

type stubSuggester struct{ list []Suggestion }

func (s stubSuggester) Suggest(_, _ string) ([]Suggestion, error) { return s.list, nil }

type stubDigest struct{ count int }

func (s *stubDigest) Send(_ context.Context, _ Tenant, _ string) error { s.count++; return nil }

func TestCreateTenantDuplicate(t *testing.T) {
	s := NewService(Config{})
	if _, err := s.CreateTenant("acme", "ada@x"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateTenant("acme", "x@y"); err == nil {
		t.Error("expected duplicate error")
	}
}

func TestGitHubAppInstallTransitionsStage(t *testing.T) {
	s := NewService(Config{})
	tn, _ := s.CreateTenant("acme", "ada@x")
	if tn.Stage != "install" {
		t.Errorf("stage=%q", tn.Stage)
	}
	if err := s.HandleGitHubAppInstall("acme", 1234); err != nil {
		t.Fatal(err)
	}
	got, _ := s.Tenant("acme")
	if got.Stage != "cartography" {
		t.Errorf("stage=%q want cartography", got.Stage)
	}
	if !got.GitHubInstalled {
		t.Error("install flag not set")
	}
	if !got.Sources.GitHubPRReviewComments {
		t.Error("source not enabled")
	}
}

func TestVerifyGitHubWebhookHMAC(t *testing.T) {
	s := NewService(Config{GitHubAppSecret: "sek"})
	body := []byte(`{"hello":"world"}`)
	mac := hmac.New(sha256.New, []byte("sek"))
	mac.Write(body)
	sig := []byte("sha256=" + hex.EncodeToString(mac.Sum(nil)))
	if !s.VerifyGitHubWebhook(sig, body) {
		t.Error("valid signature rejected")
	}
	if s.VerifyGitHubWebhook([]byte("sha256=wrong"), body) {
		t.Error("bad signature accepted")
	}
}

func TestVerifySlackSignature(t *testing.T) {
	s := NewService(Config{SlackSecret: "sek"})
	body := []byte("payload=hi")
	ts := "1700000000"
	mac := hmac.New(sha256.New, []byte("sek"))
	mac.Write([]byte("v0:" + ts + ":"))
	mac.Write(body)
	sig := "v0=" + hex.EncodeToString(mac.Sum(nil))
	if !s.VerifySlackSignature(ts, sig, body) {
		t.Error("valid signature rejected")
	}
	if s.VerifySlackSignature(ts, "v0=bad", body) {
		t.Error("bad signature accepted")
	}
}

func TestLaunchCartographerRequiresInstall(t *testing.T) {
	s := NewService(Config{Cartographer: stubLauncher{id: "j_x"}})
	_, _ = s.CreateTenant("acme", "ada@x")
	if _, err := s.LaunchCartographer(context.Background(), "acme", "acme/x", "/r"); err == nil {
		t.Error("expected install requirement")
	}
	_ = s.HandleGitHubAppInstall("acme", 1)
	id, err := s.LaunchCartographer(context.Background(), "acme", "acme/x", "/r")
	if err != nil {
		t.Fatal(err)
	}
	if id != "j_x" {
		t.Errorf("id=%q", id)
	}
}

func TestFirstTaskFlow(t *testing.T) {
	s := NewService(Config{Suggester: stubSuggester{list: []Suggestion{{Title: "x"}}}})
	_, _ = s.CreateTenant("acme", "a@x")
	_ = s.HandleGitHubAppInstall("acme", 1)
	if err := s.MarkFirstTaskSubmitted("acme"); err != nil {
		t.Fatal(err)
	}
	if err := s.MarkFirstVerifiedPR("acme"); err != nil {
		t.Fatal(err)
	}
	tn, _ := s.Tenant("acme")
	if tn.Stage != "bootstrap" {
		t.Errorf("stage=%q", tn.Stage)
	}
	if tn.FirstVerifiedPRAt.IsZero() {
		t.Error("first verified pr not stamped")
	}
	if _, err := s.FirstTaskSuggestions("acme", "acme/x"); err != nil {
		t.Fatal(err)
	}
}

func TestRunWeeklyDigestSendsForEachTenant(t *testing.T) {
	d := &stubDigest{}
	s := NewService(Config{Digest: d})
	_, _ = s.CreateTenant("acme", "x@y")
	_, _ = s.CreateTenant("beta", "y@z")
	n, err := s.RunWeeklyDigest(context.Background())
	if err != nil || n != 2 {
		t.Errorf("got %d, err=%v", n, err)
	}
}

func TestCSTouchpointDays(t *testing.T) {
	if TouchpointDue(1) != 1 || TouchpointDue(5) != 5 || TouchpointDue(30) != 30 {
		t.Error("expected 1/5/30 to fire")
	}
	if TouchpointDue(7) != 0 {
		t.Error("day 7 should not fire")
	}
}

func TestRunCustomerSuccessTouchpoints(t *testing.T) {
	hits := 0
	hk := func(_ context.Context, _ Tenant, day int) error {
		hits++
		if day != 1 {
			t.Errorf("day=%d", day)
		}
		return nil
	}
	now := time.Now().UTC()
	s := NewService(Config{CS: hk, Now: func() time.Time { return now.Add(24 * time.Hour) }})
	tn, _ := s.CreateTenant("acme", "x@y")
	tn.CreatedAt = now
	s.tenants["acme"] = tn
	n, err := s.RunCustomerSuccessTouchpoints(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 || hits != 1 {
		t.Errorf("n=%d hits=%d", n, hits)
	}
}

func TestHandleGitHubInstallationPayload(t *testing.T) {
	body := []byte(`{"action":"created","installation":{"id":42,"account":{"login":"acme"}}}`)
	slug, id, err := HandleGitHubInstallationPayload(body)
	if err != nil {
		t.Fatal(err)
	}
	if slug != "acme" || id != 42 {
		t.Errorf("slug=%q id=%d", slug, id)
	}
}

func TestHandleGitHubInstallationIgnoresOtherActions(t *testing.T) {
	body := []byte(`{"action":"deleted","installation":{"id":1,"account":{"login":"x"}}}`)
	_, _, err := HandleGitHubInstallationPayload(body)
	if err == nil || !strings.Contains(err.Error(), "ignored") {
		t.Errorf("err=%v", err)
	}
}

func TestWireSourceUnknown(t *testing.T) {
	s := NewService(Config{})
	_, _ = s.CreateTenant("acme", "x@y")
	err := s.WireSource("acme", "fictional", true)
	if err == nil {
		t.Error("expected unknown source error")
	}
}

var _ = errors.New // keep errors import referenced in case future tests want it.
