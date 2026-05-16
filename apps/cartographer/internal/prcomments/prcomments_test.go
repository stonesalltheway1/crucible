package prcomments

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestFilterAndRankFiltersBots(t *testing.T) {
	in := []Comment{
		{Body: "lgtm", IsBot: false, Author: "human"},
		{Body: "Please use cursor pagination here — offset is O(N) on the page.", Author: "human"},
		{Body: "automatic dependency update", IsBot: true, Author: "dependabot[bot]"},
		{Body: "approved", Author: "human"},
		{Body: "x", Author: "human"},
	}
	out := filterAndRank(in, 100)
	if len(out) != 1 {
		t.Fatalf("got %d, want 1", len(out))
	}
	if !strings.Contains(out[0].Body, "cursor pagination") {
		t.Errorf("wrong comment kept: %q", out[0].Body)
	}
}

func TestFilterAndRankSortsByLength(t *testing.T) {
	// All bodies must be > the 20-char min comment length AFTER trimming
	// trailing whitespace, so all three survive filtering and we can
	// assert the sort order.
	in := []Comment{
		{Body: strings.Repeat("a ", 20), Author: "x"},  // 39 chars after trim
		{Body: strings.Repeat("b ", 50), Author: "x"},  // 99 chars after trim
		{Body: strings.Repeat("c ", 15), Author: "x"},  // 29 chars after trim
	}
	out := filterAndRank(in, 100)
	if len(out) != 3 {
		t.Fatalf("got %d", len(out))
	}
	if len(out[0].Body) < len(out[1].Body) || len(out[1].Body) < len(out[2].Body) {
		t.Error("not sorted descending")
	}
}

func TestDetectIncidentRefFindsLinear(t *testing.T) {
	body := "this caused the outage in https://linear.app/acme/issue/PAY-123 yesterday"
	got := detectIncidentRef(body)
	if !strings.Contains(got, "PAY-123") {
		t.Errorf("got %q want PAY-123", got)
	}
}

func TestDetectIncidentRefFindsINC(t *testing.T) {
	body := "Followup to INC-4221 — see comment thread."
	got := detectIncidentRef(body)
	if got != "INC-4221" {
		t.Errorf("got %q", got)
	}
}

func TestFetchHitsGraphQLAndDecodes(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer t" {
			t.Errorf("missing auth header: %v", r.Header)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"repository": map[string]any{
					"pullRequests": map[string]any{
						"pageInfo": map[string]any{"hasNextPage": false, "endCursor": ""},
						"nodes": []any{
							map[string]any{
								"number":    1,
								"updatedAt": time.Now().Format(time.RFC3339),
								"url":       "https://github.com/x/y/pull/1",
								"reviews": map[string]any{"nodes": []any{
									map[string]any{
										"state": "CHANGES_REQUESTED",
										"body":  "Please use cursor pagination here — offset is O(N) on the page.",
										"author": map[string]any{"login": "ada"},
										"url":    "https://github.com/x/y/pull/1#review-1",
										"submittedAt": time.Now().Format(time.RFC3339),
										"comments":    map[string]any{"nodes": []any{}},
									},
								}},
								"comments": map[string]any{"nodes": []any{}},
							},
						},
					},
				},
			},
		})
	}))
	defer ts.Close()

	c := &Client{Endpoint: ts.URL, Token: "t", HTTP: ts.Client()}
	got, err := c.Fetch(context.Background(), "x/y", DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d comments", len(got))
	}
	if !strings.Contains(got[0].Body, "cursor pagination") {
		t.Errorf("wrong body: %q", got[0].Body)
	}
}

func TestFetchRejectsMissingToken(t *testing.T) {
	c := &Client{}
	_, err := c.Fetch(context.Background(), "x/y", DefaultOptions())
	if err == nil {
		t.Fatal("expected ErrNoToken")
	}
}
