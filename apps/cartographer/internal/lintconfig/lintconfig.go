// Package lintconfig parses lint configs into ConventionCandidates.
//
// This is the "Tier-A deterministic config pass" from
// docs/06-research/memory-bootstrap.md §B — ~30% of conventions for
// free, no LLM required. Matches the Phase-5 LintConfigAdapter
// behaviour but written in Go for the production cartographer.
//
// Each parser reads a config file and emits one or more typed
// ConventionCandidate rows. We DON'T try to convert every option in
// every config — only the rules that map cleanly to a Crucible
// convention category. Everything else is dropped.
package lintconfig

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/crucible/apps/cartographer/internal/types"
)

// Parser parses one config file and returns the conventions it implies.
type Parser interface {
	// Filename matches: returns true if this parser handles a file with
	// the given basename.
	Matches(basename string) bool
	// Parse returns the conventions derived from the file body.
	Parse(repo, tenantID, relPath string, body []byte) ([]types.ConventionCandidate, error)
}

// All returns every parser the cartographer knows about.
func All() []Parser {
	return []Parser{
		EditorconfigParser{},
		PrettierParser{},
		EslintParser{},
		TsconfigParser{},
		PyprojectParser{},
		RuffParser{},
		GolangciParser{},
		RustfmtParser{},
		ClippyParser{},
		RubocopParser{},
		StylelintParser{},
		MarkdownlintParser{},
		CodeownersParser{},
		CommitlintParser{},
		RenovateParser{},
		GitleaksParser{},
		PhpcsParser{},
		CheckstyleParser{},
	}
}

// Run runs every parser on every file in root that matches it. The
// pass is deterministic and single-threaded — config parsing is fast.
func Run(root, repo, tenantID string) []types.ConventionCandidate {
	var out []types.ConventionCandidate
	parsers := All()
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		base := filepath.Base(path)
		for _, p := range parsers {
			if !p.Matches(base) {
				continue
			}
			body, rerr := os.ReadFile(path)
			if rerr != nil {
				continue
			}
			rel, _ := filepath.Rel(root, path)
			rel = filepath.ToSlash(rel)
			cands, perr := p.Parse(repo, tenantID, rel, body)
			if perr != nil {
				continue
			}
			out = append(out, cands...)
		}
		return nil
	})
	return out
}

// --- helpers ---

func mkCandidate(repo, tenantID, relPath, channel, category, ruleNL, fileGlob, evidence string, conf float64) types.ConventionCandidate {
	id := "c_" + strings.ReplaceAll(strings.ReplaceAll(category, " ", "_"), "/", "_") +
		"_" + simpleHash(repo+ruleNL+fileGlob)
	return types.ConventionCandidate{
		ID:            id,
		Category:      category,
		RuleNL:        ruleNL,
		FileGlob:      fileGlob,
		Rationale:     "Derived from " + relPath + ".",
		EvidenceQuote: trim(evidence, 240),
		SourceChannel: channel,
		SourcePath:    relPath,
		Confidence:    conf,
		Status:        "candidate",
		FirstSeen:     time.Now().UTC(),
	}
}

func trim(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// simpleHash is a stable, dependency-free 32-bit FNV-1a — fine for
// generating stable convention IDs that remain stable across cartographer
// runs on the same repo.
func simpleHash(s string) string {
	const (
		offset uint32 = 2166136261
		prime  uint32 = 16777619
	)
	h := offset
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= prime
	}
	const hex = "0123456789abcdef"
	var b [8]byte
	for i := 7; i >= 0; i-- {
		b[i] = hex[h&0xf]
		h >>= 4
	}
	return string(b[:])
}

// --- parsers ---

type EditorconfigParser struct{}

func (EditorconfigParser) Matches(b string) bool { return b == ".editorconfig" }
func (EditorconfigParser) Parse(repo, tenantID, rel string, body []byte) ([]types.ConventionCandidate, error) {
	var out []types.ConventionCandidate
	for _, line := range strings.Split(string(body), "\n") {
		l := strings.TrimSpace(line)
		if l == "" || strings.HasPrefix(l, "#") || strings.HasPrefix(l, ";") || strings.HasPrefix(l, "[") {
			continue
		}
		k, v, ok := splitKV(l)
		if !ok {
			continue
		}
		switch strings.ToLower(k) {
		case "indent_style":
			out = append(out, mkCandidate(repo, tenantID, rel, "lint_config", "Naming",
				"Use "+v+" for indentation in this repo.", "**/*", l, 0.85))
		case "end_of_line":
			out = append(out, mkCandidate(repo, tenantID, rel, "lint_config", "Naming",
				"Use "+v+" line endings.", "**/*", l, 0.7))
		case "max_line_length":
			out = append(out, mkCandidate(repo, tenantID, rel, "lint_config", "Naming",
				"Wrap lines at column "+v+".", "**/*", l, 0.6))
		case "insert_final_newline":
			if v == "true" {
				out = append(out, mkCandidate(repo, tenantID, rel, "lint_config", "Naming",
					"Files end with a single newline.", "**/*", l, 0.7))
			}
		}
	}
	return out, nil
}

type PrettierParser struct{}

func (PrettierParser) Matches(b string) bool {
	return b == ".prettierrc" || b == ".prettierrc.json" || b == ".prettierrc.yaml" || b == ".prettierrc.yml"
}
func (PrettierParser) Parse(repo, tenantID, rel string, body []byte) ([]types.ConventionCandidate, error) {
	cfg := map[string]any{}
	_ = json.Unmarshal(body, &cfg)
	var out []types.ConventionCandidate
	if v, ok := cfg["semi"].(bool); ok {
		rule := "Statements end with a semicolon."
		if !v {
			rule = "Statements omit trailing semicolons."
		}
		out = append(out, mkCandidate(repo, tenantID, rel, "lint_config", "Naming", rule, "**/*.{ts,tsx,js,jsx}", "semi", 0.85))
	}
	if v, ok := cfg["singleQuote"].(bool); ok && v {
		out = append(out, mkCandidate(repo, tenantID, rel, "lint_config", "Naming",
			"Use single-quoted strings in JS/TS source.", "**/*.{ts,tsx,js,jsx}", "singleQuote", 0.8))
	}
	if v, ok := cfg["printWidth"].(float64); ok {
		out = append(out, mkCandidate(repo, tenantID, rel, "lint_config", "Naming",
			"Wrap source lines at column "+ftoa(v)+".", "**/*.{ts,tsx,js,jsx}", "printWidth", 0.6))
	}
	if v, ok := cfg["trailingComma"].(string); ok {
		out = append(out, mkCandidate(repo, tenantID, rel, "lint_config", "Naming",
			"Trailing-comma policy is `"+v+"`.", "**/*.{ts,tsx,js,jsx}", "trailingComma", 0.6))
	}
	return out, nil
}

type EslintParser struct{}

func (EslintParser) Matches(b string) bool {
	return strings.HasPrefix(b, ".eslintrc")
}
func (EslintParser) Parse(repo, tenantID, rel string, body []byte) ([]types.ConventionCandidate, error) {
	cfg := map[string]any{}
	_ = json.Unmarshal(body, &cfg) // best-effort; .cjs / .js paths just no-op.
	var out []types.ConventionCandidate
	rules, _ := cfg["rules"].(map[string]any)
	for k, v := range rules {
		level := normalizeESLintLevel(v)
		if level == "off" {
			continue
		}
		cat := categorizeESLintRule(k)
		if cat == "" {
			continue
		}
		out = append(out, mkCandidate(repo, tenantID, rel, "lint_config", cat,
			"ESLint rule `"+k+"` is "+level+" — adhere to it.", "**/*.{ts,tsx,js,jsx}", k+":"+level, 0.7))
	}
	return out, nil
}

type TsconfigParser struct{}

func (TsconfigParser) Matches(b string) bool { return b == "tsconfig.json" }
func (TsconfigParser) Parse(repo, tenantID, rel string, body []byte) ([]types.ConventionCandidate, error) {
	cfg := map[string]any{}
	_ = json.Unmarshal(body, &cfg)
	var out []types.ConventionCandidate
	co, _ := cfg["compilerOptions"].(map[string]any)
	if v, ok := co["strict"].(bool); ok && v {
		out = append(out, mkCandidate(repo, tenantID, rel, "lint_config", "Error handling",
			"TypeScript strict mode is on; never disable per-file or per-symbol unless documented.",
			"**/*.{ts,tsx}", "strict:true", 0.9))
	}
	if v, ok := co["noImplicitAny"].(bool); ok && v {
		out = append(out, mkCandidate(repo, tenantID, rel, "lint_config", "Error handling",
			"Implicit `any` is forbidden — annotate parameters.", "**/*.{ts,tsx}", "noImplicitAny:true", 0.85))
	}
	if v, ok := co["target"].(string); ok {
		out = append(out, mkCandidate(repo, tenantID, rel, "lint_config", "Library preferences",
			"Compile target is `"+v+"`; do not lower it.", "**/*.{ts,tsx}", "target:"+v, 0.55))
	}
	return out, nil
}

type PyprojectParser struct{}

func (PyprojectParser) Matches(b string) bool { return b == "pyproject.toml" }
func (PyprojectParser) Parse(repo, tenantID, rel string, body []byte) ([]types.ConventionCandidate, error) {
	// Surface the most-asked rules as text scans: tool sections imply
	// "this tool is in use".
	text := string(body)
	var out []types.ConventionCandidate
	if strings.Contains(text, "[tool.ruff") {
		out = append(out, mkCandidate(repo, tenantID, rel, "lint_config", "Naming",
			"Ruff is the lint+format authority for this repo.", "**/*.py", "[tool.ruff]", 0.8))
	}
	if strings.Contains(text, "[tool.black") {
		out = append(out, mkCandidate(repo, tenantID, rel, "lint_config", "Naming",
			"Black is the formatter — keep its diff at zero.", "**/*.py", "[tool.black]", 0.8))
	}
	if strings.Contains(text, "[tool.isort") {
		out = append(out, mkCandidate(repo, tenantID, rel, "lint_config", "Naming",
			"Imports are sorted with isort.", "**/*.py", "[tool.isort]", 0.7))
	}
	if strings.Contains(text, "[tool.mypy") {
		out = append(out, mkCandidate(repo, tenantID, rel, "lint_config", "Error handling",
			"mypy enforces typing — preserve existing annotations.", "**/*.py", "[tool.mypy]", 0.85))
	}
	if strings.Contains(text, "[tool.pytest") {
		out = append(out, mkCandidate(repo, tenantID, rel, "lint_config", "Test patterns",
			"Tests are discovered via pytest configuration in pyproject.toml.", "tests/**/*.py", "[tool.pytest]", 0.7))
	}
	return out, nil
}

type RuffParser struct{}

func (RuffParser) Matches(b string) bool { return b == "ruff.toml" || b == ".ruff.toml" }
func (RuffParser) Parse(repo, tenantID, rel string, body []byte) ([]types.ConventionCandidate, error) {
	return []types.ConventionCandidate{
		mkCandidate(repo, tenantID, rel, "lint_config", "Naming",
			"Ruff configuration is the authority for Python lint+format.", "**/*.py", string(body[:min(len(body), 80)]), 0.8),
	}, nil
}

type GolangciParser struct{}

func (GolangciParser) Matches(b string) bool { return b == ".golangci.yml" || b == ".golangci.yaml" }
func (GolangciParser) Parse(repo, tenantID, rel string, body []byte) ([]types.ConventionCandidate, error) {
	text := string(body)
	var out []types.ConventionCandidate
	out = append(out, mkCandidate(repo, tenantID, rel, "lint_config", "Naming",
		"golangci-lint is the gate — every linter listed in this config must pass on diff.",
		"**/*.go", "linters:", 0.8))
	if strings.Contains(text, "errcheck") {
		out = append(out, mkCandidate(repo, tenantID, rel, "lint_config", "Error handling",
			"Errors must be checked — `_ = err` is forbidden by errcheck.", "**/*.go", "errcheck", 0.9))
	}
	if strings.Contains(text, "govet") {
		out = append(out, mkCandidate(repo, tenantID, rel, "lint_config", "Error handling",
			"`go vet` clean is required.", "**/*.go", "govet", 0.85))
	}
	return out, nil
}

type RustfmtParser struct{}

func (RustfmtParser) Matches(b string) bool { return b == "rustfmt.toml" || b == ".rustfmt.toml" }
func (RustfmtParser) Parse(repo, tenantID, rel string, body []byte) ([]types.ConventionCandidate, error) {
	return []types.ConventionCandidate{
		mkCandidate(repo, tenantID, rel, "lint_config", "Naming",
			"rustfmt configuration governs all .rs formatting.", "**/*.rs", "rustfmt", 0.85),
	}, nil
}

type ClippyParser struct{}

func (ClippyParser) Matches(b string) bool { return b == "clippy.toml" || b == ".clippy.toml" }
func (ClippyParser) Parse(repo, tenantID, rel string, body []byte) ([]types.ConventionCandidate, error) {
	return []types.ConventionCandidate{
		mkCandidate(repo, tenantID, rel, "lint_config", "Error handling",
			"clippy lints are gating — keep `cargo clippy --all-targets -- -D warnings` clean.", "**/*.rs", "clippy", 0.85),
	}, nil
}

type RubocopParser struct{}

func (RubocopParser) Matches(b string) bool { return b == ".rubocop.yml" || b == ".rubocop.yaml" }
func (RubocopParser) Parse(repo, tenantID, rel string, body []byte) ([]types.ConventionCandidate, error) {
	return []types.ConventionCandidate{
		mkCandidate(repo, tenantID, rel, "lint_config", "Naming",
			"Rubocop configuration is the Ruby style authority.", "**/*.rb", "rubocop", 0.8),
	}, nil
}

type StylelintParser struct{}

func (StylelintParser) Matches(b string) bool {
	return b == ".stylelintrc" || b == ".stylelintrc.json" || b == ".stylelintrc.yaml"
}
func (StylelintParser) Parse(repo, tenantID, rel string, body []byte) ([]types.ConventionCandidate, error) {
	return []types.ConventionCandidate{
		mkCandidate(repo, tenantID, rel, "lint_config", "Naming",
			"Stylelint config governs CSS/SCSS conventions.", "**/*.{css,scss}", "stylelint", 0.7),
	}, nil
}

type MarkdownlintParser struct{}

func (MarkdownlintParser) Matches(b string) bool {
	return b == ".markdownlint.json" || b == ".markdownlint.yaml" || b == ".markdownlint.yml"
}
func (MarkdownlintParser) Parse(repo, tenantID, rel string, body []byte) ([]types.ConventionCandidate, error) {
	return []types.ConventionCandidate{
		mkCandidate(repo, tenantID, rel, "lint_config", "Naming",
			"Markdownlint config is enforced — keep docs clean.", "**/*.md", "markdownlint", 0.5),
	}, nil
}

type CodeownersParser struct{}

func (CodeownersParser) Matches(b string) bool { return b == "CODEOWNERS" }
func (CodeownersParser) Parse(repo, tenantID, rel string, body []byte) ([]types.ConventionCandidate, error) {
	var out []types.ConventionCandidate
	for _, line := range strings.Split(string(body), "\n") {
		l := strings.TrimSpace(line)
		if l == "" || strings.HasPrefix(l, "#") {
			continue
		}
		fields := strings.Fields(l)
		if len(fields) < 2 {
			continue
		}
		out = append(out, mkCandidate(repo, tenantID, rel, "lint_config", "PR/commit hygiene",
			"Changes to `"+fields[0]+"` require review by "+strings.Join(fields[1:], ", ")+".",
			fields[0], l, 0.95))
	}
	return out, nil
}

type CommitlintParser struct{}

func (CommitlintParser) Matches(b string) bool {
	return b == "commitlint.config.cjs" || b == "commitlint.config.js" || b == "commitlint.config.mjs"
}
func (CommitlintParser) Parse(repo, tenantID, rel string, body []byte) ([]types.ConventionCandidate, error) {
	rule := "This repo enforces Conventional Commits — commit subjects use the form `<type>(<scope>): <subject>`."
	return []types.ConventionCandidate{
		mkCandidate(repo, tenantID, rel, "lint_config", "PR/commit hygiene", rule, "**/*", "commitlint", 0.9),
	}, nil
}

type RenovateParser struct{}

func (RenovateParser) Matches(b string) bool { return b == "renovate.json" }
func (RenovateParser) Parse(repo, tenantID, rel string, body []byte) ([]types.ConventionCandidate, error) {
	return []types.ConventionCandidate{
		mkCandidate(repo, tenantID, rel, "lint_config", "Library preferences",
			"Dependency updates land via Renovate; keep manifests as-is unless intentional.", "**/*", "renovate", 0.7),
	}, nil
}

type GitleaksParser struct{}

func (GitleaksParser) Matches(b string) bool { return b == ".gitleaks.toml" }
func (GitleaksParser) Parse(repo, tenantID, rel string, body []byte) ([]types.ConventionCandidate, error) {
	return []types.ConventionCandidate{
		mkCandidate(repo, tenantID, rel, "lint_config", "Security defaults",
			"gitleaks scans this repo for secrets — never commit any literal credential.", "**/*", "gitleaks", 0.95),
	}, nil
}

type PhpcsParser struct{}

func (PhpcsParser) Matches(b string) bool { return b == "phpcs.xml" || b == "phpcs.xml.dist" }
func (PhpcsParser) Parse(repo, tenantID, rel string, body []byte) ([]types.ConventionCandidate, error) {
	return []types.ConventionCandidate{
		mkCandidate(repo, tenantID, rel, "lint_config", "Naming",
			"PHP_CodeSniffer is the PHP style authority.", "**/*.php", "phpcs", 0.7),
	}, nil
}

type CheckstyleParser struct{}

func (CheckstyleParser) Matches(b string) bool { return b == "checkstyle.xml" }
func (CheckstyleParser) Parse(repo, tenantID, rel string, body []byte) ([]types.ConventionCandidate, error) {
	return []types.ConventionCandidate{
		mkCandidate(repo, tenantID, rel, "lint_config", "Naming",
			"Checkstyle config is the Java style authority.", "**/*.java", "checkstyle", 0.7),
	}, nil
}

// --- helpers ---

func splitKV(line string) (string, string, bool) {
	eq := strings.IndexByte(line, '=')
	if eq < 0 {
		return "", "", false
	}
	return strings.TrimSpace(line[:eq]), strings.TrimSpace(line[eq+1:]), true
}

func ftoa(v float64) string {
	// Integer-only print for the printWidth-style integers we expect.
	n := int(v)
	return itoa(n)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func normalizeESLintLevel(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case float64:
		switch t {
		case 0:
			return "off"
		case 1:
			return "warn"
		default:
			return "error"
		}
	case []any:
		if len(t) == 0 {
			return "off"
		}
		return normalizeESLintLevel(t[0])
	}
	return "off"
}

func categorizeESLintRule(k string) string {
	switch {
	case strings.Contains(k, "no-unused"), strings.Contains(k, "no-undef"):
		return "Error handling"
	case strings.Contains(k, "import"), strings.HasPrefix(k, "import/"):
		return "Layering"
	case strings.Contains(k, "naming"), strings.Contains(k, "camelcase"):
		return "Naming"
	case strings.Contains(k, "console"), strings.Contains(k, "logger"):
		return "Logging"
	case strings.Contains(k, "react"):
		return "Library preferences"
	case strings.Contains(k, "security"):
		return "Security defaults"
	}
	return "Naming"
}

// Errors common to every parser.
var (
	ErrEmptyConfig = errors.New("lintconfig: empty config body")
)
