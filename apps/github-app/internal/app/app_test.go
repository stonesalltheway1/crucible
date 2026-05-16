package app

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParseCommand_CrucibleWithDescription(t *testing.T) {
	cmd, args, ok := parseCommand("/crucible add idempotency to /webhooks/stripe/refund\n\nplease and thanks")
	if !ok || cmd != "/crucible" || args != "add idempotency to /webhooks/stripe/refund" {
		t.Errorf("parseCommand returned (%q, %q, %v)", cmd, args, ok)
	}
}

func TestParseCommand_NoCommand(t *testing.T) {
	if _, _, ok := parseCommand("just a normal comment"); ok {
		t.Errorf("expected no command")
	}
}

func TestVerifySignature_ValidHMAC(t *testing.T) {
	body := []byte(`{"hello":"world"}`)
	secret := "shh"
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	sig := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	if !verifySignature(sig, body, secret) {
		t.Errorf("expected valid signature to pass")
	}
}

func TestVerifySignature_TamperedRejected(t *testing.T) {
	body := []byte(`{"hello":"world"}`)
	secret := "shh"
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(`{"hello":"different"}`))
	sig := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	if verifySignature(sig, body, secret) {
		t.Errorf("expected tampered signature to fail")
	}
}

func TestVerifySignature_MissingPrefix(t *testing.T) {
	if verifySignature("not-sha256", []byte("body"), "secret") {
		t.Errorf("expected non-sha256 prefix to fail")
	}
}

func TestHandler_Ping(t *testing.T) {
	a := New(Config{WebhookSecret: ""})
	srv := httptest.NewServer(a.Handler())
	defer srv.Close()
	req, _ := http.NewRequest("POST", srv.URL+"/webhook", strings.NewReader(`{}`))
	req.Header.Set("X-GitHub-Event", "ping")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestHandler_RejectsBadSignature(t *testing.T) {
	a := New(Config{WebhookSecret: "shh"})
	srv := httptest.NewServer(a.Handler())
	defer srv.Close()
	req, _ := http.NewRequest("POST", srv.URL+"/webhook", strings.NewReader(`{}`))
	req.Header.Set("X-GitHub-Event", "ping")
	req.Header.Set("X-Hub-Signature-256", "sha256=bogus")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 401 {
		t.Errorf("expected 401 on bad signature, got %d", resp.StatusCode)
	}
}
