package lintconfig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestRunExtractsConventionsFromConfigs(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, ".editorconfig"), "root=true\n[*]\nindent_style=space\nmax_line_length=120\ninsert_final_newline=true\n")
	write(t, filepath.Join(root, ".prettierrc.json"), `{"semi": false, "singleQuote": true, "printWidth": 100}`)
	write(t, filepath.Join(root, "tsconfig.json"), `{"compilerOptions": {"strict": true, "noImplicitAny": true, "target": "ES2022"}}`)
	write(t, filepath.Join(root, "pyproject.toml"), "[tool.ruff]\nline-length = 100\n[tool.mypy]\nstrict = true\n[tool.pytest.ini_options]\n")
	write(t, filepath.Join(root, "CODEOWNERS"), "/api/ @api-team\n*.tf @platform\n")
	write(t, filepath.Join(root, ".golangci.yml"), "linters:\n  enable:\n    - errcheck\n    - govet\n")

	cands := Run(root, "acme/payments", "ten_x")
	if len(cands) < 8 {
		t.Fatalf("expected ≥8 conventions, got %d", len(cands))
	}
	// Spot-check categories present.
	cats := map[string]int{}
	for _, c := range cands {
		cats[c.Category]++
	}
	for _, want := range []string{"Naming", "Error handling", "PR/commit hygiene"} {
		if cats[want] == 0 {
			t.Errorf("expected category %q present", want)
		}
	}
	// Strict-mode conventions should be high-confidence (>=0.8).
	var sawStrict bool
	for _, c := range cands {
		if strings.Contains(c.RuleNL, "strict mode is on") && c.Confidence >= 0.8 {
			sawStrict = true
		}
	}
	if !sawStrict {
		t.Error("strict mode convention missing or low-confidence")
	}
}

func TestPrettierParserParsesBooleans(t *testing.T) {
	cands, err := PrettierParser{}.Parse("repo", "ten", ".prettierrc.json", []byte(`{"semi": true, "singleQuote": true, "printWidth": 80}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(cands) < 3 {
		t.Errorf("expected ≥3 candidates, got %d", len(cands))
	}
}

func TestCodeownersParserExtractsLines(t *testing.T) {
	cands, _ := CodeownersParser{}.Parse("repo", "ten", "CODEOWNERS", []byte("/api/ @owner1 @owner2\n# comment\n*.tf @ops\n"))
	if len(cands) != 2 {
		t.Errorf("expected 2 candidates, got %d", len(cands))
	}
}
