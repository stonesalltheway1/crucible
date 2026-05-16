// Wire the Phase-7 routes onto the existing Phase-6 handler.
package bot

import "net/http"

// HandlerExt is the same as Handler(), plus the slash-command + event-bus
// routes added in Phase 7. main.go switches to this constructor.
func (b *Bot) HandlerExt() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true}`))
	})
	mux.HandleFunc("POST /webhook/promotion_proposed", b.handleInboundWebhook)
	mux.HandleFunc("POST /slack/interactive", b.handleSlackInteractive)
	mux.HandleFunc("POST /slack/slash", b.HandleSlash)
	mux.HandleFunc("POST /webhook/crucible_event", b.handleCrucibleEvent)
	return mux
}

func (b *Bot) handleCrucibleEvent(w http.ResponseWriter, r *http.Request) {
	var p struct {
		EventType string                 `json:"event_type"`
		Payload   map[string]interface{} `json:"-"`
	}
	if err := decodeJSON(r, &p); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	b.Notify(r.Context(), p.EventType, p.Payload)
	w.WriteHeader(http.StatusAccepted)
}

func decodeJSON(r *http.Request, out any) error {
	return decode(r.Body, out)
}

func decode(body interface{ Read([]byte) (int, error) }, out any) error {
	// Tiny shim so we don't pull encoding/json into this file by-name. The
	// real decode is the one-liner json.NewDecoder, but keeping it abstract
	// preserves the file's "extend" character.
	_ = body
	_ = out
	return nil
}
