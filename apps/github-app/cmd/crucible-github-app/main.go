// Command crucible-github-app is the GitHub App backend for Phase 7.
//
// The app does three things:
//
//  1. Listens for `issue_comment` webhooks; if the comment starts with
//     `/crucible <description>`, submits a task to the control plane and
//     posts an acknowledgement comment with a link to the plan-approval UI.
//
//  2. Listens for the verifier's `task.completed` webhook; opens a PR on the
//     customer's repo with the diff, attaches the verifier report as a
//     review-style comment, and links the attestation chain in the PR body.
//
//  3. Mediates the promotion-approval button: an approval click in the
//     web console / Slack / GitHub-PR-comment routes through the gate and
//     the gate's response is posted back as a PR comment.
//
// Permissions requested (minimum viable):
//   • repo: read
//   • pull_requests: write
//   • issues: write     (to post on issue_comment threads)
//   • workflow: read    (verifier results)
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/crucible/github-app/internal/app"
)

const version = "2026.06.0-phase7"

func main() {
	addr := flag.String("addr", envOr("CRUCIBLE_GITHUB_APP_ADDR", ":9320"), "listen address")
	controlPlane := flag.String("control-plane", envOr("CRUCIBLE_API_ADDR", "http://localhost:8080"), "Crucible control plane endpoint")
	appID := flag.String("app-id", os.Getenv("GITHUB_APP_ID"), "GitHub App ID")
	privKeyPath := flag.String("private-key", os.Getenv("GITHUB_APP_PRIVATE_KEY_PATH"), "path to the GitHub App PEM private key")
	hookSecret := flag.String("webhook-secret", os.Getenv("GITHUB_WEBHOOK_SECRET"), "GitHub webhook shared secret")
	flag.Parse()

	if *hookSecret == "" {
		log.Println("WARNING: webhook secret unset — refusing to verify signatures in dev mode only")
	}

	a := app.New(app.Config{
		ControlPlaneAddr: *controlPlane,
		AppID:            *appID,
		PrivateKeyPath:   *privKeyPath,
		WebhookSecret:    *hookSecret,
		Version:          version,
	})

	srv := &http.Server{
		Addr:              *addr,
		Handler:           a.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      30 * time.Second,
	}

	fmt.Printf(`{"version":%q,"msg":"github-app listening","addr":%q}`+"\n", version, *addr)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
