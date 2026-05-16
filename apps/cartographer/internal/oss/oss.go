// Package oss loads the OSS-derived defaults the Phase-5
// `infra/oss-corpus-bootstrap` job emitted into
// services/memory-router/global_defaults/<stack>/conventions.json.
//
// Cartographer surfaces a stack-filtered subset of these as the
// "✓ Loaded 312 OSS-derived defaults for your stack." line. The full
// admission to the per-tenant memory happens in memory-router, not
// here.
package oss

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/crucible/apps/cartographer/internal/types"
)

// Loader knows where the per-stack bundles live on disk.
type Loader struct {
	Root string
}

// NewLoader constructs a loader pointing at the given root directory.
// If the directory doesn't exist, Loader still returns and operates as
// an empty source (zero defaults loaded). This keeps cartographer
// usable in dev / air-gap setups where the bundles haven't shipped.
func NewLoader(root string) (*Loader, error) {
	if root == "" {
		return nil, errors.New("oss: empty root")
	}
	abs, _ := filepath.Abs(root)
	return &Loader{Root: abs}, nil
}

// LoadStack loads the per-stack bundle for the given stack name.
// Stack names match the per-stack bundle directories (e.g. "nextjs",
// "django", "fastapi", "rails", "go-services", "rust-services",
// "spring-boot", "phoenix", "vue", "express", "laravel", "flask").
func (l *Loader) LoadStack(stack string) ([]types.ConventionCandidate, error) {
	if l == nil || l.Root == "" {
		return nil, nil
	}
	stack = normalizeStackName(stack)
	candidates := []string{
		filepath.Join(l.Root, stack, "conventions.json"),
		filepath.Join(l.Root, stack+".json"),
	}
	var path string
	for _, p := range candidates {
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			path = p
			break
		}
	}
	if path == "" {
		return nil, nil // stack unknown — caller reports zero loaded.
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var out []types.ConventionCandidate
	// Tolerant: try array first, then envelope.
	if err := json.Unmarshal(body, &out); err == nil && len(out) > 0 {
		annotateStack(out, stack)
		return out, nil
	}
	var env struct {
		Conventions []types.ConventionCandidate `json:"conventions"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, err
	}
	annotateStack(env.Conventions, stack)
	return env.Conventions, nil
}

// LoadStacks loads multiple stacks and returns the union, deduped by
// (category, signature).
func (l *Loader) LoadStacks(stacks ...string) ([]types.ConventionCandidate, error) {
	seen := map[string]bool{}
	var out []types.ConventionCandidate
	for _, s := range stacks {
		cs, err := l.LoadStack(s)
		if err != nil {
			return nil, err
		}
		for _, c := range cs {
			key := c.Category + "|" + strings.ToLower(strings.TrimSpace(c.RuleNL))
			if seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, c)
		}
	}
	return out, nil
}

func annotateStack(cs []types.ConventionCandidate, stack string) {
	for i := range cs {
		if cs[i].Stack == "" {
			cs[i].Stack = stack
		}
		if cs[i].SourceChannel == "" {
			cs[i].SourceChannel = "oss_default"
		}
		if cs[i].Status == "" {
			cs[i].Status = "active"
		}
	}
}

func normalizeStackName(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	switch s {
	case "next", "next.js", "nextjs":
		return "nextjs"
	case "fast-api", "fastapi":
		return "fastapi"
	case "django":
		return "django"
	case "rails", "ruby-on-rails":
		return "rails"
	case "go", "golang", "go-services":
		return "go-services"
	case "rust", "rust-services":
		return "rust-services"
	case "spring", "spring-boot":
		return "spring-boot"
	case "phoenix", "elixir":
		return "phoenix"
	}
	return s
}
