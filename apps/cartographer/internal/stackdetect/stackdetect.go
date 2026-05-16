// Package stackdetect classifies the primary + secondary stacks of a
// repo from its file mix and manifest files.
package stackdetect

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// Stack is the result.
type Stack struct {
	Primary   string
	Secondary []string
}

// Detect inspects manifests and file extensions.
func Detect(root string) Stack {
	hits := map[string]int{}
	tally := func(name string, n int) { hits[name] += n }

	if exists(filepath.Join(root, "package.json")) {
		// Distinguish Next/Vue/Express/etc by reading dependencies.
		body, _ := os.ReadFile(filepath.Join(root, "package.json"))
		var pj struct {
			Dependencies    map[string]string `json:"dependencies"`
			DevDependencies map[string]string `json:"devDependencies"`
		}
		_ = json.Unmarshal(body, &pj)
		merged := map[string]string{}
		for k, v := range pj.Dependencies {
			merged[k] = v
		}
		for k, v := range pj.DevDependencies {
			merged[k] = v
		}
		switch {
		case has(merged, "next"):
			tally("nextjs", 5)
		case has(merged, "nuxt"):
			tally("nuxt", 5)
		case has(merged, "vue"):
			tally("vue", 5)
		case has(merged, "@angular/core"):
			tally("angular", 5)
		case has(merged, "express"), has(merged, "fastify"), has(merged, "koa"):
			tally("express", 4)
		default:
			tally("javascript", 2)
		}
	}
	if exists(filepath.Join(root, "pyproject.toml")) || exists(filepath.Join(root, "requirements.txt")) {
		py, _ := os.ReadFile(filepath.Join(root, "pyproject.toml"))
		req, _ := os.ReadFile(filepath.Join(root, "requirements.txt"))
		all := strings.ToLower(string(py) + "\n" + string(req))
		switch {
		case strings.Contains(all, "django"):
			tally("django", 5)
		case strings.Contains(all, "fastapi"):
			tally("fastapi", 5)
		case strings.Contains(all, "flask"):
			tally("flask", 5)
		default:
			tally("python", 2)
		}
	}
	if exists(filepath.Join(root, "Gemfile")) {
		body, _ := os.ReadFile(filepath.Join(root, "Gemfile"))
		if strings.Contains(strings.ToLower(string(body)), "rails") {
			tally("rails", 5)
		} else {
			tally("ruby", 2)
		}
	}
	if exists(filepath.Join(root, "go.mod")) {
		tally("go-services", 4)
	}
	if exists(filepath.Join(root, "Cargo.toml")) {
		tally("rust-services", 4)
	}
	if exists(filepath.Join(root, "build.gradle.kts")) || exists(filepath.Join(root, "build.gradle")) ||
		exists(filepath.Join(root, "pom.xml")) {
		tally("spring-boot", 4)
	}
	if exists(filepath.Join(root, "mix.exs")) {
		tally("phoenix", 4)
	}
	if exists(filepath.Join(root, "Package.swift")) {
		tally("swift", 4)
	}
	if exists(filepath.Join(root, "composer.json")) {
		body, _ := os.ReadFile(filepath.Join(root, "composer.json"))
		if strings.Contains(strings.ToLower(string(body)), "laravel") {
			tally("laravel", 5)
		} else {
			tally("php", 2)
		}
	}

	// Pick top.
	type kv struct {
		k string
		v int
	}
	var rows []kv
	for k, v := range hits {
		rows = append(rows, kv{k, v})
	}
	for i := 0; i < len(rows); i++ {
		for j := i + 1; j < len(rows); j++ {
			if rows[j].v > rows[i].v {
				rows[i], rows[j] = rows[j], rows[i]
			}
		}
	}
	out := Stack{}
	if len(rows) > 0 {
		out.Primary = rows[0].k
		for _, r := range rows[1:] {
			out.Secondary = append(out.Secondary, r.k)
		}
	}
	return out
}

func exists(p string) bool {
	if _, err := os.Stat(p); err == nil {
		return true
	}
	return false
}

func has(m map[string]string, key string) bool {
	_, ok := m[key]
	return ok
}
