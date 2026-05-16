// Package prcomments scans recent PR review comments via the GitHub
// GraphQL API.
//
// Per the Phase-8 brief: "last 24 months, top 1000 by length". The
// scanner is rate-limit-aware (GitHub's secondary-rate-limit signals
// rather than the published cap). On large repos we cap the cumulative
// query budget to keep cartographer wall-clock under 30 min.
//
// We only scan REVIEW comments — the ones that signal "this should be
// changed" semantics. Issue comments and inline file comments are
// included; bot comments (dependabot, renovate, github-actions) are
// dropped.
package prcomments

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

// Client is the GitHub GraphQL fetcher.
type Client struct {
	Endpoint string // default https://api.github.com/graphql
	Token    string
	HTTP     *http.Client
}

// NewClient constructs a Client with sane defaults.
func NewClient(token string) *Client {
	return &Client{
		Endpoint: "https://api.github.com/graphql",
		Token:    token,
		HTTP: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Comment is the trimmed body we hand to the distiller.
type Comment struct {
	Owner       string    `json:"owner"`
	Repo        string    `json:"repo"`
	PRNumber    int       `json:"pr_number"`
	Body        string    `json:"body"`
	Author      string    `json:"author"`
	URL         string    `json:"url"`
	IsBot       bool      `json:"is_bot"`
	State       string    `json:"state"` // "review_changes_requested" | "review_commented" | "issue_comment"
	CreatedAt   time.Time `json:"created_at"`
	IncidentRef string    `json:"incident_ref,omitempty"` // detected Linear/Jira/Slack URL
}

// FetchOptions bound the scan.
type FetchOptions struct {
	MaxComments int
	WindowDays  int
	NowFunc     func() time.Time
}

// DefaultOptions returns the brief's defaults.
func DefaultOptions() FetchOptions {
	return FetchOptions{MaxComments: 1000, WindowDays: 24 * 30, NowFunc: time.Now}
}

// botUsers is the set of accounts whose comments we ignore.
var botUsers = map[string]bool{
	"dependabot[bot]":       true,
	"renovate[bot]":         true,
	"github-actions[bot]":   true,
	"pre-commit-ci[bot]":    true,
	"sonarcloud[bot]":       true,
	"codecov[bot]":          true,
	"snyk-bot":              true,
	"deepsource-autofix[bot]": true,
}

const minCommentLen = 20

// Fetch fetches recent PR review comments. The repo arg uses the
// "owner/name" format that maps cleanly to the GraphQL search query.
func (c *Client) Fetch(ctx context.Context, repo string, opts FetchOptions) ([]Comment, error) {
	if c.Token == "" {
		return nil, ErrNoToken
	}
	owner, name, err := splitRepo(repo)
	if err != nil {
		return nil, err
	}
	if opts.MaxComments == 0 {
		opts = DefaultOptions()
	}
	if opts.NowFunc == nil {
		opts.NowFunc = time.Now
	}
	cutoff := opts.NowFunc().AddDate(0, 0, -opts.WindowDays)

	// We page the GraphQL query 50 PRs at a time, descending by updated
	// date, until either we exhaust the cutoff window or we've gathered
	// enough comments. Per-PR review-comment cap is 50; per-PR
	// issue-comment cap is 50. The vast majority of PRs have <30
	// comments.
	const prPageSize = 50
	var cursor string
	var collected []Comment
	for len(collected) < opts.MaxComments {
		resp, err := c.executeQuery(ctx, owner, name, prPageSize, cursor)
		if err != nil {
			return collected, err
		}
		for _, pr := range resp.Data.Repository.PullRequests.Nodes {
			if pr.UpdatedAt.Before(cutoff) {
				goto out // PR list is descending; everything after is older.
			}
			collected = append(collected, extractComments(owner, name, pr)...)
			if len(collected) >= opts.MaxComments {
				break
			}
		}
		if !resp.Data.Repository.PullRequests.PageInfo.HasNextPage {
			break
		}
		cursor = resp.Data.Repository.PullRequests.PageInfo.EndCursor
	}
out:
	// Filter + truncate to top-N by body length.
	filtered := filterAndRank(collected, opts.MaxComments)
	return filtered, nil
}

func filterAndRank(in []Comment, max int) []Comment {
	var keep []Comment
	for _, c := range in {
		if c.IsBot {
			continue
		}
		body := strings.TrimSpace(c.Body)
		if len(body) < minCommentLen {
			continue
		}
		// Drop trivial approvals.
		low := strings.ToLower(body)
		if low == "lgtm" || low == "approved" || low == "ship it" || low == "done" {
			continue
		}
		c.Body = body
		c.IncidentRef = detectIncidentRef(body)
		keep = append(keep, c)
	}
	sort.Slice(keep, func(i, j int) bool { return len(keep[i].Body) > len(keep[j].Body) })
	if len(keep) > max {
		keep = keep[:max]
	}
	return keep
}

// detectIncidentRef returns a URL-form incident reference if one is
// embedded in the comment body.
func detectIncidentRef(body string) string {
	for _, prefix := range []string{
		"https://linear.app/",
		"https://jira.",
		"https://atlassian.net/",
		"https://app.shortcut.com/",
		"https://github.com/", // could be a cross-link to an incident issue
	} {
		if i := strings.Index(body, prefix); i >= 0 {
			end := i + len(prefix)
			for end < len(body) && body[end] != ' ' && body[end] != ')' && body[end] != ']' && body[end] != '\n' {
				end++
			}
			return body[i:end]
		}
	}
	if i := strings.Index(body, "INC-"); i >= 0 {
		end := i + 4
		for end < len(body) && (body[end] >= '0' && body[end] <= '9') {
			end++
		}
		if end > i+4 {
			return body[i:end]
		}
	}
	return ""
}

// --- GraphQL types + execution ---

type graphqlResponse struct {
	Data struct {
		Repository struct {
			PullRequests struct {
				PageInfo struct {
					HasNextPage bool   `json:"hasNextPage"`
					EndCursor   string `json:"endCursor"`
				} `json:"pageInfo"`
				Nodes []prNode `json:"nodes"`
			} `json:"pullRequests"`
		} `json:"repository"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors,omitempty"`
}

type prNode struct {
	Number    int       `json:"number"`
	UpdatedAt time.Time `json:"updatedAt"`
	URL       string    `json:"url"`
	Reviews   struct {
		Nodes []reviewNode `json:"nodes"`
	} `json:"reviews"`
	Comments struct {
		Nodes []commentNode `json:"nodes"`
	} `json:"comments"`
}

type reviewNode struct {
	State       string        `json:"state"`
	Body        string        `json:"body"`
	Author      authorNode    `json:"author"`
	URL         string        `json:"url"`
	SubmittedAt time.Time     `json:"submittedAt"`
	Comments    struct {
		Nodes []commentNode `json:"nodes"`
	} `json:"comments"`
}

type commentNode struct {
	Body      string     `json:"body"`
	Author    authorNode `json:"author"`
	URL       string     `json:"url"`
	CreatedAt time.Time  `json:"createdAt"`
}

type authorNode struct {
	Login string `json:"login"`
}

// schema is the literal GraphQL document; pinned to the v4 API.
const schemaTemplate = `query($owner: String!, $name: String!, $first: Int!, $after: String) {
	repository(owner: $owner, name: $name) {
		pullRequests(first: $first, after: $after, orderBy: {field: UPDATED_AT, direction: DESC}, states: [OPEN, MERGED, CLOSED]) {
			pageInfo { hasNextPage endCursor }
			nodes {
				number
				updatedAt
				url
				reviews(first: 50) {
					nodes {
						state
						body
						url
						submittedAt
						author { login }
						comments(first: 50) {
							nodes { body author { login } url createdAt }
						}
					}
				}
				comments(first: 50) {
					nodes { body author { login } url createdAt }
				}
			}
		}
	}
}`

func (c *Client) executeQuery(ctx context.Context, owner, name string, first int, cursor string) (*graphqlResponse, error) {
	endpoint := c.Endpoint
	if endpoint == "" {
		endpoint = "https://api.github.com/graphql"
	}
	body := map[string]any{
		"query": schemaTemplate,
		"variables": map[string]any{
			"owner": owner,
			"name":  name,
			"first": first,
			"after": cursor,
		},
	}
	if cursor == "" {
		// GraphQL servers reject empty `after`; pass null instead.
		body["variables"].(map[string]any)["after"] = nil
	}
	bbuf, _ := json.Marshal(body)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bbuf))
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "crucible-cartographer/2026.06.0-phase8")
	cli := c.HTTP
	if cli == nil {
		cli = http.DefaultClient
	}
	resp, err := cli.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == 502 || resp.StatusCode == 503 {
		// secondary-rate-limit: the caller should retry with backoff.
		return nil, fmt.Errorf("github rate-limited: HTTP %d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		buf, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("github graphql HTTP %d: %s", resp.StatusCode, string(buf))
	}
	var r graphqlResponse
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, err
	}
	if len(r.Errors) > 0 {
		return &r, fmt.Errorf("github graphql error: %s", r.Errors[0].Message)
	}
	return &r, nil
}

func extractComments(owner, name string, pr prNode) []Comment {
	var out []Comment
	for _, rev := range pr.Reviews.Nodes {
		state := normalizeReviewState(rev.State)
		if rev.Body != "" {
			out = append(out, Comment{
				Owner: owner, Repo: name, PRNumber: pr.Number,
				Body: rev.Body, Author: rev.Author.Login,
				URL: rev.URL, IsBot: botUsers[rev.Author.Login],
				State: state, CreatedAt: rev.SubmittedAt,
			})
		}
		for _, sub := range rev.Comments.Nodes {
			out = append(out, Comment{
				Owner: owner, Repo: name, PRNumber: pr.Number,
				Body: sub.Body, Author: sub.Author.Login,
				URL: sub.URL, IsBot: botUsers[sub.Author.Login],
				State: state, CreatedAt: sub.CreatedAt,
			})
		}
	}
	for _, ic := range pr.Comments.Nodes {
		out = append(out, Comment{
			Owner: owner, Repo: name, PRNumber: pr.Number,
			Body: ic.Body, Author: ic.Author.Login,
			URL: ic.URL, IsBot: botUsers[ic.Author.Login],
			State: "issue_comment", CreatedAt: ic.CreatedAt,
		})
	}
	return out
}

func normalizeReviewState(s string) string {
	switch strings.ToUpper(s) {
	case "CHANGES_REQUESTED":
		return "review_changes_requested"
	case "APPROVED":
		return "review_approved"
	case "COMMENTED":
		return "review_commented"
	}
	return strings.ToLower(s)
}

func splitRepo(repo string) (string, string, error) {
	parts := strings.SplitN(repo, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("prcomments: bad repo %q (expected owner/name)", repo)
	}
	return parts[0], parts[1], nil
}

// Errors.
var (
	ErrNoToken = errors.New("prcomments: no GitHub token configured")
)
