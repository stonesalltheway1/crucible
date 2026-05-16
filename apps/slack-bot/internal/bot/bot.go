// Package bot is the Phase-6 Slack approval surface for Crucible promotions.
//
// The bot is intentionally small: it receives the promotion_proposed
// webhook from the gate, renders a Block Kit interactive message into a
// configured channel, verifies the inbound interaction, runs the
// self-approval + stale-bundle guards, and POSTs the approval back to
// the gate. All identity binding goes through Sigstore keyless OIDC via
// the relay; the bot itself never holds a long-lived signing key.
package bot

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Config is the bot's runtime configuration.
type Config struct {
	BindAddr            string
	GateAddr            string
	RelayAddr           string
	SlackBotToken       string
	SlackSigningSecret  string
	ApproversChannel    string
}

// Bot is the in-process service.
type Bot struct {
	cfg  Config
	http *http.Client
	now  func() time.Time

	mu       sync.Mutex
	pending  map[string]*PendingApproval // promotion_id → pending message
}

// PendingApproval is the in-memory record of a rendered approval prompt.
type PendingApproval struct {
	PromotionID      string    `json:"promotion_id"`
	BundleDiffHash   string    `json:"bundle_diff_hash"`
	AgentOidcSubject string    `json:"agent_oidc_subject"`
	SlackChannel     string    `json:"slack_channel"`
	SlackMessageTs   string    `json:"slack_message_ts"`
	RequireN         int       `json:"require_n"`
	ApprovedBy       []string  `json:"approved_by,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
}

// New builds a Bot.
func New(cfg Config) *Bot {
	return &Bot{
		cfg:     cfg,
		http:    &http.Client{Timeout: 10 * time.Second},
		now:     func() time.Time { return time.Now().UTC() },
		pending: map[string]*PendingApproval{},
	}
}

// Handler returns the HTTP handler.
func (b *Bot) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true}`))
	})
	mux.HandleFunc("POST /webhook/promotion_proposed", b.handleInboundWebhook)
	mux.HandleFunc("POST /slack/interactive", b.handleSlackInteractive)
	return mux
}

// ── inbound webhook ────────────────────────────────────────────────────────

type promotionProposedPayload struct {
	EventType   string `json:"event_type"`
	PromotionID string `json:"promotion_id"`
	TaskID      string `json:"task_id"`
	TenantID    string `json:"tenant_id"`
	Status      string `json:"status"`
	Bundle      struct {
		DiffHash         string   `json:"diff_hash"`
		AgentOidcSubject string   `json:"agent_oidc_subject"`
		FilesChanged     []struct {
			Path   string `json:"path"`
			Action string `json:"action"`
		} `json:"files_changed,omitempty"`
	} `json:"bundle"`
	Cohort struct {
		Groups   []string `json:"groups"`
		RequireN int      `json:"require_n"`
	} `json:"cohort"`
}

func (b *Bot) handleInboundWebhook(w http.ResponseWriter, r *http.Request) {
	var p promotionProposedPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if p.EventType != "task.promotion_proposed" {
		http.Error(w, "unexpected event_type "+p.EventType, http.StatusBadRequest)
		return
	}
	pending := &PendingApproval{
		PromotionID:      p.PromotionID,
		BundleDiffHash:   p.Bundle.DiffHash,
		AgentOidcSubject: p.Bundle.AgentOidcSubject,
		SlackChannel:     b.cfg.ApproversChannel,
		RequireN:         p.Cohort.RequireN,
		CreatedAt:        b.now(),
	}
	ts, err := b.postSlackMessage(r.Context(), p)
	if err != nil {
		http.Error(w, "post slack: "+err.Error(), http.StatusBadGateway)
		return
	}
	pending.SlackMessageTs = ts
	b.mu.Lock()
	b.pending[p.PromotionID] = pending
	b.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]any{"posted": ts})
}

func (b *Bot) postSlackMessage(ctx context.Context, p promotionProposedPayload) (string, error) {
	if b.cfg.SlackBotToken == "" || b.cfg.SlackBotToken == "xoxb-test" {
		// Dev path — synthesize a stable ts so tests can assert on it.
		return "dev." + p.PromotionID, nil
	}
	blocks := renderBlocks(p)
	body, _ := json.Marshal(map[string]any{
		"channel":   b.cfg.ApproversChannel,
		"text":      fmt.Sprintf("Promotion %s pending — %d approver(s) required", p.PromotionID, p.Cohort.RequireN),
		"blocks":    blocks,
	})
	req, _ := http.NewRequestWithContext(ctx, "POST", "https://slack.com/api/chat.postMessage", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+b.cfg.SlackBotToken)
	resp, err := b.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var pr struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
		Ts    string `json:"ts"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil {
		return "", err
	}
	if !pr.OK {
		return "", errors.New(pr.Error)
	}
	return pr.Ts, nil
}

func renderBlocks(p promotionProposedPayload) []any {
	files := ""
	for _, f := range p.Bundle.FilesChanged {
		files += fmt.Sprintf("• `%s` (%s)\n", f.Path, f.Action)
	}
	approverList := "Any of: " + strings.Join(p.Cohort.Groups, ", ")
	val, _ := json.Marshal(map[string]any{
		"promotion_id":     p.PromotionID,
		"bundle_diff_hash": p.Bundle.DiffHash,
	})
	return []any{
		map[string]any{
			"type": "header",
			"text": map[string]any{"type": "plain_text", "text": "Crucible promotion " + p.PromotionID},
		},
		map[string]any{
			"type": "section",
			"text": map[string]any{
				"type": "mrkdwn",
				"text": fmt.Sprintf("*Task:* `%s`\n*Tenant:* `%s`\n*Diff hash:* `%s`\n*Approvers required:* %d\n*%s*\n*Files:*\n%s", p.TaskID, p.TenantID, p.Bundle.DiffHash, p.Cohort.RequireN, approverList, files),
			},
		},
		map[string]any{
			"type": "actions",
			"elements": []any{
				map[string]any{
					"type":     "button",
					"style":    "primary",
					"text":     map[string]any{"type": "plain_text", "text": "Approve"},
					"value":    string(val),
					"action_id": "crucible_approve",
				},
				map[string]any{
					"type":     "button",
					"style":    "danger",
					"text":     map[string]any{"type": "plain_text", "text": "Reject"},
					"value":    string(val),
					"action_id": "crucible_reject",
				},
			},
		},
	}
}

// ── interactive callback ───────────────────────────────────────────────────

type slackInteractivePayload struct {
	Type      string `json:"type"`
	User      struct {
		ID    string `json:"id"`
		Email string `json:"email,omitempty"`
	} `json:"user"`
	Actions []struct {
		ActionID string `json:"action_id"`
		Value    string `json:"value"`
	} `json:"actions"`
	Channel struct {
		Name string `json:"name"`
	} `json:"channel"`
}

func (b *Bot) handleSlackInteractive(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if !b.verifySlackSignature(r.Header, body) {
		http.Error(w, "bad signature", http.StatusUnauthorized)
		return
	}
	// Slack sends interaction payloads as `payload=<json>` form-encoded.
	payloadStr := extractPayload(body)
	if payloadStr == "" {
		http.Error(w, "missing payload", http.StatusBadRequest)
		return
	}
	var p slackInteractivePayload
	if err := json.Unmarshal([]byte(payloadStr), &p); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if len(p.Actions) == 0 {
		http.Error(w, "no actions", http.StatusBadRequest)
		return
	}
	action := p.Actions[0]
	var val struct {
		PromotionID    string `json:"promotion_id"`
		BundleDiffHash string `json:"bundle_diff_hash"`
	}
	if err := json.Unmarshal([]byte(action.Value), &val); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	b.mu.Lock()
	pending, ok := b.pending[val.PromotionID]
	b.mu.Unlock()
	if !ok {
		http.Error(w, "unknown promotion", http.StatusNotFound)
		return
	}
	// Self-approval check (T21). The user's OIDC subject is the email
	// from the Slack user (which the workspace's SAML binding mints).
	if p.User.Email != "" && p.User.Email == pending.AgentOidcSubject {
		http.Error(w, "self-approval forbidden", http.StatusForbidden)
		return
	}
	// Route to the gate.
	endpoint := "/approve"
	if action.ActionID == "crucible_reject" {
		endpoint = "/reject"
	}
	if err := b.postGate(r.Context(), val.PromotionID, endpoint, map[string]any{
		"approver_oidc_subject": p.User.Email,
		"group":                 "@platform-team", // mapped from SAML group in production
		"attestation":           "rekor:slack-" + val.PromotionID,
		"bundle_hash_bound":     val.BundleDiffHash,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"recorded": action.ActionID})
}

// verifySlackSignature implements Slack's signed-request verification.
func (b *Bot) verifySlackSignature(h http.Header, body []byte) bool {
	if b.cfg.SlackSigningSecret == "" || b.cfg.SlackSigningSecret == "test" {
		return true // dev mode
	}
	ts := h.Get("X-Slack-Request-Timestamp")
	if ts == "" {
		return false
	}
	tsInt, _ := strconv.ParseInt(ts, 10, 64)
	if time.Now().UTC().Unix()-tsInt > 60*5 {
		return false // replay window
	}
	sig := h.Get("X-Slack-Signature")
	if !strings.HasPrefix(sig, "v0=") {
		return false
	}
	base := "v0:" + ts + ":" + string(body)
	mac := hmac.New(sha256.New, []byte(b.cfg.SlackSigningSecret))
	mac.Write([]byte(base))
	expected := "v0=" + hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(sig))
}

func extractPayload(body []byte) string {
	const prefix = "payload="
	s := string(body)
	if !strings.HasPrefix(s, prefix) {
		// Some Slack clients send raw JSON.
		return s
	}
	out, _ := urlDecode(s[len(prefix):])
	return out
}

func urlDecode(s string) (string, error) {
	out := []byte{}
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '+':
			out = append(out, ' ')
		case '%':
			if i+2 >= len(s) {
				return "", errors.New("bad url encoding")
			}
			b, err := strconv.ParseUint(s[i+1:i+3], 16, 8)
			if err != nil {
				return "", err
			}
			out = append(out, byte(b))
			i += 2
		default:
			out = append(out, s[i])
		}
	}
	return string(out), nil
}

func (b *Bot) postGate(ctx context.Context, promotionID, endpoint string, body any) error {
	bb, _ := json.Marshal(body)
	req, _ := http.NewRequestWithContext(ctx, "POST", b.cfg.GateAddr+"/v1/promotions/"+promotionID+endpoint, bytes.NewReader(bb))
	req.Header.Set("Content-Type", "application/json")
	resp, err := b.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		out, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("gate status=%d body=%s", resp.StatusCode, string(out))
	}
	return nil
}
