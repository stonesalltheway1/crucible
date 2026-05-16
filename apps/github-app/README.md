# Crucible — GitHub App

Routes `/crucible <description>` PR/issue comments to the Crucible control
plane, posts plan-approval links back as comments, and surfaces the
verifier verdict + promotion lifecycle inline.

## Permissions requested (minimum-viable)

| Scope | Why |
|---|---|
| `repo: read` | enumerate the repo's PRs and files |
| `pull_requests: write` | open PRs for agent-authored verified bundles |
| `issues: write` | post `/crucible` acknowledgement and verifier-report comments |
| `workflow: read` | observe CI status / verifier results |

No `admin` or `org:*` permissions. The app is scoped to repository-level.

## Endpoints

| Path | Purpose |
|---|---|
| `POST /webhook` | GitHub webhooks (signed via `X-Hub-Signature-256`) |
| `POST /crucible/event` | Crucible-side events (signed via `X-Crucible-Signature`) |
| `GET /healthz` | health |

## Run (dev, via ngrok)

```bash
ngrok http 9320
# copy the ngrok URL into the App's webhook settings
go run ./cmd/crucible-github-app \
  --webhook-secret=$GITHUB_WEBHOOK_SECRET \
  --control-plane=http://localhost:8080
```

## Tests

```bash
go test ./...
```

Coverage: signature verification (valid + tampered + missing-prefix),
slash-command parsing, the `ping` smoke path.
