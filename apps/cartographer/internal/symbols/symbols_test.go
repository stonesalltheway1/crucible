package symbols

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/crucible/apps/cartographer/internal/walker"
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

func TestBuildExtractsSymbolsAcrossLanguages(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "main.go"), "package main\n\nfunc Helper(x int) int { return x }\n\ntype User struct{ Name string }\n")
	writeFile(t, filepath.Join(root, "app.py"), "def handle_webhook(req):\n    return 200\n\nclass Config:\n    pass\n")
	writeFile(t, filepath.Join(root, "ui.tsx"), "export function Button() { return null }\nexport const TOKEN = 'k'\nexport class Page {}\n")
	writeFile(t, filepath.Join(root, "lib.rs"), "pub fn parse(s: &str) -> u64 { 0 }\npub struct Conn;\n")
	// Real-world Java + Swift code is rarely all on one line. The scanner
	// is line-anchored (cheap; matches what 99% of source looks like in
	// production codebases). Multi-line layout is what we test on.
	writeFile(t, filepath.Join(root, "App.java"), "public class App {\n    public int compute(int x) {\n        return x;\n    }\n}\n")
	writeFile(t, filepath.Join(root, "Foo.swift"), "public struct Foo {\n    public func bar() {}\n}\n")

	files, _, err := walker.Walk(root, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	idx, err := Build(context.Background(), files)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"Helper", "User", "handle_webhook", "Config", "Button", "TOKEN", "Page", "parse", "Conn", "App", "compute", "Foo", "bar"}
	for _, w := range want {
		if len(idx.ByName[w]) == 0 {
			t.Errorf("symbol %q not found in index", w)
		}
	}
	if idx.ByLang[walker.LangGo] != 2 {
		t.Errorf("go symbols=%d want 2", idx.ByLang[walker.LangGo])
	}
}

func TestBuildSkipsTestFiles(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "main_test.go"), "package x\nfunc TestX(t *testing.T) {}\n")
	writeFile(t, filepath.Join(root, "main.go"), "package x\nfunc Real() {}\n")
	files, _, err := walker.Walk(root, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	idx, _ := Build(context.Background(), files)
	if len(idx.ByName["TestX"]) != 0 {
		t.Error("TestX should not be indexed (test file)")
	}
	if len(idx.ByName["Real"]) == 0 {
		t.Error("Real should be indexed")
	}
}

func TestTopReturnsMostFrequent(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "a.go"), "package x\nfunc handler() {}\nfunc handler2() {}\n")
	writeFile(t, filepath.Join(root, "b.go"), "package x\nfunc handler() {}\nfunc helper() {}\n")
	files, _, _ := walker.Walk(root, 0, 0)
	idx, _ := Build(context.Background(), files)
	top := idx.Top(2)
	if len(top) == 0 {
		t.Fatal("empty top")
	}
	// The most frequent name should be handler (appears 2x); top[0] = "handler".
	if top[0] != "handler" {
		t.Errorf("top[0]=%q want handler", top[0])
	}
}
