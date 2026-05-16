// Package main is the cth-grade CLI: drive a Crucible instance through
// every case in cth/, collect grading metrics, emit a per-release
// report.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/crucible/cth/grading/internal/grade"
	"github.com/crucible/cth/grading/internal/runner"
	"github.com/crucible/cth/grading/internal/spec"
)

func main() {
	root := flag.String("root", ".", "CTH root directory")
	category := flag.String("category", "", "limit to one category (greenfield|feature-add|refactor|critical-path|adversarial|regression)")
	out := flag.String("out", "cth-results/results.json", "output path")
	apiAddr := flag.String("api-addr", os.Getenv("CRUCIBLE_API_ADDR"), "Crucible control-plane address")
	apiToken := flag.String("api-token", os.Getenv("CRUCIBLE_API_TOKEN"), "Crucible API token")
	failFast := flag.Bool("fail-fast", false, "stop at the first non-passing case")
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	slog.SetDefault(logger)

	cases, err := loadCases(*root, *category)
	if err != nil {
		slog.Error("loadCases", "err", err)
		os.Exit(1)
	}
	r := runner.New(runner.Config{Addr: *apiAddr, Token: *apiToken})
	results := make([]grade.CaseResult, 0, len(cases))
	for _, c := range cases {
		slog.Info("running case", "id", c.ID, "category", c.Category)
		res := r.Run(context.Background(), c)
		results = append(results, res)
		if !res.Passed && *failFast {
			slog.Error("fail-fast triggered", "case", c.ID, "reason", res.Reason)
			break
		}
	}
	report := grade.Aggregate(results)
	report.GeneratedAt = time.Now().UTC()

	_ = os.MkdirAll(filepath.Dir(*out), 0o755)
	body, _ := json.MarshalIndent(report, "", "  ")
	if err := os.WriteFile(*out, body, 0o644); err != nil {
		slog.Error("write", "err", err)
		os.Exit(1)
	}
	fmt.Println(string(body))
	if !report.AllPassed {
		os.Exit(2)
	}
}

func loadCases(root, category string) ([]spec.Case, error) {
	var out []spec.Case
	categoryDirs := []string{"greenfield", "feature-add", "refactor", "critical-path", "adversarial", "regression"}
	if category != "" {
		categoryDirs = []string{category}
	}
	for _, cat := range categoryDirs {
		dir := filepath.Join(root, cat)
		_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil || !d.IsDir() {
				return err
			}
			specPath := filepath.Join(path, "spec.json")
			if _, err := os.Stat(specPath); err != nil {
				return nil
			}
			body, err := os.ReadFile(specPath)
			if err != nil {
				return err
			}
			var c spec.Case
			if err := json.Unmarshal(body, &c); err != nil {
				return err
			}
			c.Dir = path
			c.ID = strings.TrimPrefix(strings.TrimPrefix(path, root), "/")
			c.Category = cat
			out = append(out, c)
			return nil
		})
	}
	return out, nil
}
