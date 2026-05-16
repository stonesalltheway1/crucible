// Phase 7 expansion: slash-command task submission + DM-based status
// notifications. The Phase-6 surface (channel-level promotion approvals
// with Block Kit + signature verification) remains in bot.go.
package bot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// SlashRequest mirrors Slack's slash-command POST body.
type SlashRequest struct {
	Token       string
	TeamID      string
	ChannelID   string
	UserID      string
	UserName    string
	Command     string
	Text        string
	ResponseURL string
	TriggerID   string
}

// HandleSlash is wired by extendHandler in extend.go.
func (b *Bot) HandleSlash(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if !b.verifySlackSignature(r.Header, body) {
		http.Error(w, "bad signature", http.StatusUnauthorized)
		return
	}
	form, err := url.ParseQuery(string(body))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	req := SlashRequest{
		TeamID:      form.Get("team_id"),
		ChannelID:   form.Get("channel_id"),
		UserID:      form.Get("user_id"),
		UserName:    form.Get("user_name"),
		Command:     form.Get("command"),
		Text:        strings.TrimSpace(form.Get("text")),
		ResponseURL: form.Get("response_url"),
		TriggerID:   form.Get("trigger_id"),
	}

	switch req.Command {
	case "/crucible":
		if req.Text == "" {
			respondSlash(w, "ephemeral", "Usage: `/crucible <task description>`")
			return
		}
		// Submit asynchronously; Slack expects a response within 3s.
		go b.submitTaskFromSlash(context.Background(), req)
		respondSlash(w, "ephemeral", fmt.Sprintf("Submitting task: _%s_… you'll get a DM with the plan-approval link.", escapeSlack(req.Text)))
	case "/crucible-status":
		go b.dmTaskStatus(context.Background(), req)
		respondSlash(w, "ephemeral", "Pulling your active tasks…")
	default:
		respondSlash(w, "ephemeral", "Unknown command.")
	}
}

func (b *Bot) submitTaskFromSlash(ctx context.Context, req SlashRequest) {
	// Resolve the user's email → control-plane identity. For dev mode we
	// synthesise a stable email per user_id; production deploys plug into
	// the workspace SAML binding via auth.users.lookupByEmail.
	email := req.UserName + "@workspace.test"

	body, _ := json.Marshal(map[string]any{
		"description":    req.Text,
		"submitted_by":   email,
		"submitted_from": "slack-slash",
	})
	httpReq, _ := http.NewRequestWithContext(ctx, "POST", b.cfg.GateAddr+"/v1/tasks", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := b.http.Do(httpReq)
	if err != nil {
		b.dmUser(ctx, req.UserID, fmt.Sprintf("Crucible: failed to submit — `%s`", err.Error()))
		return
	}
	defer resp.Body.Close()
	var ack struct{ Task struct{ ID string `json:"id"` } `json:"task"` }
	_ = json.NewDecoder(resp.Body).Decode(&ack)

	url := webConsoleURL(b.cfg.GateAddr) + "/tasks/" + ack.Task.ID + "/approve"
	b.dmUser(ctx, req.UserID, fmt.Sprintf(
		"Crucible task *%s* queued. Review the plan: %s",
		ack.Task.ID, url,
	))
}

func (b *Bot) dmTaskStatus(_ context.Context, req SlashRequest) {
	// Production: query the control plane for the user's active tasks and
	// post a Block Kit summary in DM. Stubbed at the boundary; the DM path
	// itself is exercised by dmUser.
	b.dmUser(context.Background(), req.UserID, "Your active Crucible tasks will appear here once the control plane returns them.")
}

func (b *Bot) dmUser(ctx context.Context, userID, text string) {
	if b.cfg.SlackBotToken == "" || strings.HasPrefix(b.cfg.SlackBotToken, "xoxb-test") {
		// Dev mode — no-op
		return
	}
	body, _ := json.Marshal(map[string]any{"channel": userID, "text": text})
	req, _ := http.NewRequestWithContext(ctx, "POST", "https://slack.com/api/chat.postMessage", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+b.cfg.SlackBotToken)
	resp, err := b.http.Do(req)
	if err == nil {
		_ = resp.Body.Close()
	}
}

// Notify routes Crucible events into Slack DMs/channels.
//
// task.plan_proposed   → DM the submitter with the approval link.
// task.completed       → DM the submitter with the PR URL.
// task.budget_exceeded → DM + alert channel.
func (b *Bot) Notify(ctx context.Context, eventType string, payload map[string]any) {
	switch eventType {
	case "task.plan_proposed":
		user := stringFromMap(payload, "submitted_by")
		taskID := stringFromMap(payload, "task_id")
		if user == "" || taskID == "" {
			return
		}
		b.dmByEmail(ctx, user, fmt.Sprintf("Plan ready: %s/tasks/%s/approve", webConsoleURL(b.cfg.GateAddr), taskID))
	case "task.completed":
		user := stringFromMap(payload, "submitted_by")
		pr := stringFromMap(payload, "pr_url")
		if user == "" || pr == "" {
			return
		}
		b.dmByEmail(ctx, user, fmt.Sprintf("Crucible task complete — PR: %s", pr))
	case "task.budget_exceeded":
		user := stringFromMap(payload, "submitted_by")
		spent := payload["spent_usd"]
		cap := payload["cap_usd"]
		if user != "" {
			b.dmByEmail(ctx, user, fmt.Sprintf("⚠ Crucible halted — budget exceeded ($%v / $%v).", spent, cap))
		}
	}
}

func (b *Bot) dmByEmail(ctx context.Context, email, text string) {
	// Production: look up the Slack userID by email via users.lookupByEmail.
	// Dev: just channel DMs to the approvers channel.
	b.dmUser(ctx, b.cfg.ApproversChannel, "@"+email+" "+text)
}

// ── helpers ────────────────────────────────────────────────────────────────

func respondSlash(w http.ResponseWriter, responseType, text string) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"response_type": responseType, "text": text})
}

func stringFromMap(m map[string]any, k string) string {
	if v, ok := m[k]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func escapeSlack(s string) string {
	r := strings.NewReplacer("<", "&lt;", ">", "&gt;", "&", "&amp;")
	return r.Replace(s)
}

func webConsoleURL(gateAddr string) string {
	return strings.Replace(gateAddr, "api.", "app.", 1)
}
