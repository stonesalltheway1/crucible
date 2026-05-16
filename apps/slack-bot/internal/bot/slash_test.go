package bot

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleSlash_NoText_RespondsWithUsage(t *testing.T) {
	b := New(Config{SlackSigningSecret: "test"})
	srv := httptest.NewServer(http.HandlerFunc(b.HandleSlash))
	defer srv.Close()
	body := "command=%2Fcrucible&text=&user_id=U1&user_name=alice"
	resp, err := http.Post(srv.URL, "application/x-www-form-urlencoded", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestHandleSlash_RejectsBadSignature(t *testing.T) {
	b := New(Config{SlackSigningSecret: "real-secret"})
	srv := httptest.NewServer(http.HandlerFunc(b.HandleSlash))
	defer srv.Close()
	body := "command=%2Fcrucible&text=add+idempotency&user_id=U1&user_name=alice"
	// No X-Slack-Signature → 401
	resp, err := http.Post(srv.URL, "application/x-www-form-urlencoded", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 401 {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}

func TestEscapeSlack(t *testing.T) {
	got := escapeSlack("<script>alert('x')</script>")
	want := "&lt;script&gt;alert('x')&lt;/script&gt;"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestWebConsoleURL(t *testing.T) {
	cases := map[string]string{
		"https://api.crucible.dev":      "https://app.crucible.dev",
		"http://localhost:8080":         "http://localhost:8080", // no api. prefix → unchanged
		"https://api.staging.crucible.dev": "https://app.staging.crucible.dev",
	}
	for in, want := range cases {
		if got := webConsoleURL(in); got != want {
			t.Errorf("webConsoleURL(%q) = %q, want %q", in, got, want)
		}
	}
}
