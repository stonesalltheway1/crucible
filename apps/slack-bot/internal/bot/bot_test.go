package bot

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestBot(gateURL string) *Bot {
	return New(Config{
		BindAddr: ":0", GateAddr: gateURL, RelayAddr: "http://relay",
		SlackBotToken: "xoxb-test", SlackSigningSecret: "test",
		ApproversChannel: "#crucible-test",
	})
}

func TestWebhookCreatesPending(t *testing.T) {
	gate := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"ok":true}`)
	}))
	defer gate.Close()
	b := newTestBot(gate.URL)
	handler := b.Handler()

	body := `{
		"event_type": "task.promotion_proposed",
		"promotion_id": "prom_test",
		"task_id": "task_x",
		"tenant_id": "ten_demo",
		"bundle": {"diff_hash":"0xabc","agent_oidc_subject":"agent@acme"},
		"cohort": {"groups":["@platform-team"],"require_n":1}
	}`
	req := httptest.NewRequest("POST", "/webhook/promotion_proposed", strings.NewReader(body))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", w.Code)
	}
	if _, ok := b.pending["prom_test"]; !ok {
		t.Fatal("expected pending record")
	}
}

func TestApproveCallback_RejectsSelfApproval(t *testing.T) {
	gate := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer gate.Close()
	b := newTestBot(gate.URL)
	// Seed a pending whose agent_oidc_subject matches the user's email.
	b.pending["prom_self"] = &PendingApproval{
		PromotionID:      "prom_self",
		AgentOidcSubject: "agent@acme",
		BundleDiffHash:   "0xabc",
	}
	val, _ := json.Marshal(map[string]any{
		"promotion_id":     "prom_self",
		"bundle_diff_hash": "0xabc",
	})
	payload, _ := json.Marshal(slackInteractivePayload{
		Type: "block_actions",
		User: struct {
			ID    string `json:"id"`
			Email string `json:"email,omitempty"`
		}{ID: "U1", Email: "agent@acme"},
		Actions: []struct {
			ActionID string `json:"action_id"`
			Value    string `json:"value"`
		}{{ActionID: "crucible_approve", Value: string(val)}},
	})
	form := "payload=" + string(payload)
	req := httptest.NewRequest("POST", "/slack/interactive", strings.NewReader(form))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	b.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 on self-approval, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestApproveCallback_RoutesToGate(t *testing.T) {
	called := false
	gate := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/approve") {
			called = true
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer gate.Close()
	b := newTestBot(gate.URL)
	b.pending["prom_x"] = &PendingApproval{
		PromotionID:      "prom_x",
		AgentOidcSubject: "agent@acme",
		BundleDiffHash:   "0xabc",
	}
	val, _ := json.Marshal(map[string]any{
		"promotion_id":     "prom_x",
		"bundle_diff_hash": "0xabc",
	})
	payload, _ := json.Marshal(slackInteractivePayload{
		Type: "block_actions",
		User: struct {
			ID    string `json:"id"`
			Email string `json:"email,omitempty"`
		}{ID: "U2", Email: "approver@acme"},
		Actions: []struct {
			ActionID string `json:"action_id"`
			Value    string `json:"value"`
		}{{ActionID: "crucible_approve", Value: string(val)}},
	})
	form := "payload=" + string(payload)
	req := httptest.NewRequest("POST", "/slack/interactive", strings.NewReader(form))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	b.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	if !called {
		t.Fatal("expected gate /approve hit")
	}
}
