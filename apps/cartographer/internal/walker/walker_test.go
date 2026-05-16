package walker

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, p, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestWalkClassifiesFiles(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "main.go"), "package main\nfunc main() {}")
	writeFile(t, filepath.Join(root, "app.py"), "print('hi')")
	writeFile(t, filepath.Join(root, "ui.tsx"), "export {}")
	writeFile(t, filepath.Join(root, "lib.rs"), "fn main() {}")
	writeFile(t, filepath.Join(root, "Foo.swift"), "")
	writeFile(t, filepath.Join(root, "App.java"), "")
	writeFile(t, filepath.Join(root, "node_modules", "ignored.js"), "should be skipped")
	writeFile(t, filepath.Join(root, "tests", "test_app.py"), "def test_x(): pass")
	writeFile(t, filepath.Join(root, "internal", "main_test.go"), "package internal")
	writeFile(t, filepath.Join(root, "pyproject.toml"), "[tool.ruff]")
	writeFile(t, filepath.Join(root, "tsconfig.json"), "{}")

	files, stats, err := Walk(root, 0, 0)
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}
	if stats.Files == 0 {
		t.Fatal("no files indexed")
	}

	want := map[string]string{
		"main.go":                 LangGo,
		"app.py":                  LangPython,
		"ui.tsx":                  LangTypeScript,
		"lib.rs":                  LangRust,
		"Foo.swift":               LangSwift,
		"App.java":                LangJava,
		"tests/test_app.py":       LangPython,
		"internal/main_test.go":   LangGo,
		"pyproject.toml":          LangTOML,
		"tsconfig.json":           LangJSON,
	}
	got := map[string]string{}
	tests := map[string]bool{}
	configs := map[string]bool{}
	for _, f := range files {
		got[f.RelPath] = f.Language
		if f.IsTest {
			tests[f.RelPath] = true
		}
		if f.IsConfig {
			configs[f.RelPath] = true
		}
		if f.RelPath == "node_modules/ignored.js" {
			t.Fatalf("walker did not skip node_modules: %v", f.RelPath)
		}
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("file %q: language=%q want=%q", k, got[k], v)
		}
	}
	if !tests["tests/test_app.py"] {
		t.Errorf("tests/test_app.py not flagged as test")
	}
	if !tests["internal/main_test.go"] {
		t.Errorf("internal/main_test.go not flagged as test")
	}
	if !configs["pyproject.toml"] {
		t.Errorf("pyproject.toml not flagged as config")
	}
	if !configs["tsconfig.json"] {
		t.Errorf("tsconfig.json not flagged as config")
	}
}

func TestWalkBudgetEnforced(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < 50; i++ {
		writeFile(t, filepath.Join(root, "f", "x"+string(rune('a'+i%26))+".go"), "package x")
	}
	files, stats, err := Walk(root, 10, 0)
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}
	if stats.Files > 10 {
		t.Errorf("budget: indexed %d > 10", stats.Files)
	}
	if len(files) > 10 {
		t.Errorf("len(files)=%d > 10", len(files))
	}
}

func TestLanguageBreakdownText(t *testing.T) {
	stats := Stats{ByLanguage: map[string]int{
		LangGo:     5,
		LangPython: 12,
		LangRust:   1,
	}}
	got := LanguageBreakdownText(stats)
	if got == "" {
		t.Fatal("empty breakdown")
	}
	// Python should appear before Go because count is higher.
	pyIdx := indexOf(got, "python")
	goIdx := indexOf(got, "go(")
	if pyIdx < 0 || goIdx < 0 || pyIdx > goIdx {
		t.Errorf("ordering wrong: %q", got)
	}
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
