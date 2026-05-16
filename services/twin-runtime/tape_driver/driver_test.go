package tapedriver

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPrepareTapeRequiresTapeSet(t *testing.T) {
	d := New()
	_, err := d.PrepareTape(context.Background(), TapeSpec{})
	require.Error(t, err)
}

func TestEvaluateRequestFailsClosedOnMiss(t *testing.T) {
	d := New()
	resp, err := d.EvaluateRequest(context.Background(), Request{
		Service:   "stripe",
		Endpoint:  "/v1/charges",
		Method:    "POST",
		RequestID: "req_test",
	})
	require.NoError(t, err)
	require.Equal(t, 599, resp.Status)
	require.Equal(t, string(DispoMissBlocked), resp.X_Crucible_Tape)
	require.Equal(t, string(DispoMissBlocked), resp.Headers["X-Crucible-Tape"])
}

func TestRegexScrubberRedactsEmail(t *testing.T) {
	s := NewRegexScrubber()
	out, report := s.Scrub([]byte(`{"user":"alice@example.com"}`))
	require.NotContains(t, string(out), "alice@example.com")
	require.Contains(t, string(out), "redacted@example.com")
	require.Len(t, report.Rewrites, 1)
	require.Equal(t, "email", report.Rewrites[0].Scrubber)
}

func TestRegexScrubberRedactsSSN(t *testing.T) {
	s := NewRegexScrubber()
	out, _ := s.Scrub([]byte(`SSN: 123-45-6789`))
	require.Contains(t, string(out), "XXX-XX-XXXX")
	require.NotContains(t, string(out), "123-45-6789")
}

func TestRegexScrubberRedactsCreditCard(t *testing.T) {
	s := NewRegexScrubber()
	out, _ := s.Scrub([]byte(`card: 4242 4242 4242 4242`))
	require.NotContains(t, string(out), "4242 4242 4242 4242")
}

func TestRegexScrubberRedactsJWT(t *testing.T) {
	s := NewRegexScrubber()
	jwt := "eyJhbGciOiJIUzI1NiIs.eyJzdWIiOiIxMjM0NTYiLCJuYW1lIjoiSm9objsImlhdCI6MTUxNjIzOTAyMn0.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"
	out, report := s.Scrub([]byte("Authorization: Bearer " + jwt))
	require.NotContains(t, string(out), jwt)
	found := false
	for _, r := range report.Rewrites {
		if r.Scrubber == "jwt" {
			found = true
		}
	}
	require.True(t, found, "jwt scrubber should fire")
}

func TestRegexScrubberRedactsCloudKeys(t *testing.T) {
	s := NewRegexScrubber()
	cases := []struct {
		input    string
		scrubber string
	}{
		{`AWS=AKIAIOSFODNN7EXAMPLE`, "aws-access-key"},
		{`PAT=ghp_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa`, "github-pat"},
		{`KEY=sk-ant-api03-` + strings.Repeat("a", 60), "anthropic-key"},
	}
	for _, tt := range cases {
		_, report := s.Scrub([]byte(tt.input))
		found := false
		for _, r := range report.Rewrites {
			if r.Scrubber == tt.scrubber {
				found = true
			}
		}
		require.True(t, found, "scrubber %s should fire on %s", tt.scrubber, tt.input)
	}
}

func TestHoverflyCmdRender(t *testing.T) {
	cmd := HoverflyCmd{
		BinPath:    "/usr/bin/hoverfly",
		ProxyPort:  8500,
		AdminPort:  8888,
		Middleware: "/opt/scrub",
		AuthEnable: true,
		Username:   "admin",
		Password:   "p",
	}
	argv := cmd.Render()
	require.Contains(t, argv, "-pp")
	require.Contains(t, argv, "8500")
	require.Contains(t, argv, "-middleware")
	require.Contains(t, argv, "-auth")
}

func TestScrubReportEnumeratesRewrites(t *testing.T) {
	s := NewRegexScrubber()
	_, report := s.Scrub([]byte("email a@b.com phone 555-555-5555 ssn 111-22-3333"))
	require.GreaterOrEqual(t, len(report.Rewrites), 3)
}
