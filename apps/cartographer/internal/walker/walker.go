// Package walker walks a repo and classifies every file by language.
//
// The Phase-8 brief calls for "tree-sitter for the top stacks (Python,
// TS, Rust, Go, Java, Swift; minimum first 4)". We use file extension
// + shebang sniffing for the classification (the actual tree-sitter
// parsing happens in internal/symbols, where the symbol-index is
// built). Walking and classification are decoupled so the symbol
// builder can run in a separate goroutine pool.
package walker

import (
	"bufio"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Language identifiers — the same set the symbol indexer + verifier
// pipeline use.
const (
	LangPython     = "python"
	LangTypeScript = "typescript"
	LangJavaScript = "javascript"
	LangRust       = "rust"
	LangGo         = "go"
	LangJava       = "java"
	LangKotlin     = "kotlin"
	LangSwift      = "swift"
	LangRuby       = "ruby"
	LangPHP        = "php"
	LangCSharp     = "csharp"
	LangCPP        = "cpp"
	LangC          = "c"
	LangShell      = "shell"
	LangSQL        = "sql"
	LangProto      = "proto"
	LangMarkdown   = "markdown"
	LangYAML       = "yaml"
	LangTOML       = "toml"
	LangJSON       = "json"
	LangUnknown    = "unknown"
)

// File describes one file we visited.
type File struct {
	AbsPath  string
	RelPath  string
	Size     int64
	Language string
	IsTest   bool
	IsConfig bool
}

// Stats summarises a walk.
type Stats struct {
	Files       int
	Directories int
	ByLanguage  map[string]int
	BySize      int64
}

// Skip directories that vendor third-party code or pure build output.
var skipDirs = map[string]bool{
	".git": true, ".hg": true, ".svn": true,
	"node_modules": true, "vendor": true, "third_party": true,
	".venv": true, "venv": true, "env": true, "__pycache__": true,
	"target": true, "build": true, "dist": true, "out": true,
	".next": true, ".nuxt": true, ".cache": true, ".turbo": true,
	"bin": true, "obj": true,
	"DerivedData": true, // Xcode
}

// extLang maps file extensions → Language.
var extLang = map[string]string{
	".py":     LangPython,
	".pyi":    LangPython,
	".ts":     LangTypeScript,
	".tsx":    LangTypeScript,
	".js":     LangJavaScript,
	".jsx":    LangJavaScript,
	".mjs":    LangJavaScript,
	".cjs":    LangJavaScript,
	".rs":     LangRust,
	".go":     LangGo,
	".java":   LangJava,
	".kt":     LangKotlin,
	".kts":    LangKotlin,
	".swift":  LangSwift,
	".rb":     LangRuby,
	".php":    LangPHP,
	".cs":     LangCSharp,
	".cpp":    LangCPP,
	".cc":     LangCPP,
	".cxx":    LangCPP,
	".hpp":    LangCPP,
	".h":      LangC,
	".c":      LangC,
	".sh":     LangShell,
	".bash":   LangShell,
	".sql":    LangSQL,
	".proto":  LangProto,
	".md":     LangMarkdown,
	".yaml":   LangYAML,
	".yml":    LangYAML,
	".toml":   LangTOML,
	".json":   LangJSON,
}

// Test-file naming patterns per language. These are conservative — we
// recognise the patterns the language ecosystems agree on, and let the
// noise tag everything else as non-test.
var testPatterns = []func(rel string) bool{
	// Go: foo_test.go in any package.
	func(rel string) bool { return strings.HasSuffix(rel, "_test.go") },
	// JS/TS: .test.ts/.spec.tsx/.test.js, plus colocated __tests__.
	func(rel string) bool {
		base := filepath.Base(rel)
		return strings.Contains(base, ".test.") ||
			strings.Contains(base, ".spec.") ||
			strings.Contains(rel, "__tests__/")
	},
	// Python: test_*.py or *_test.py, plus tests/ dir.
	func(rel string) bool {
		base := filepath.Base(rel)
		if !strings.HasSuffix(base, ".py") {
			return false
		}
		if strings.HasPrefix(base, "test_") || strings.HasSuffix(base, "_test.py") {
			return true
		}
		return strings.HasPrefix(rel, "tests/") || strings.Contains(rel, "/tests/")
	},
	// Rust: integration tests under tests/, plus #[cfg(test)] modules
	// (we can't detect the latter without parsing — rely on tests/).
	func(rel string) bool {
		return strings.HasSuffix(rel, ".rs") &&
			(strings.HasPrefix(rel, "tests/") || strings.Contains(rel, "/tests/"))
	},
	// Java: src/test/java/**.
	func(rel string) bool {
		return strings.HasSuffix(rel, ".java") && strings.Contains(rel, "/test/")
	},
	// Swift: Tests/* directory convention.
	func(rel string) bool {
		return strings.HasSuffix(rel, ".swift") &&
			(strings.HasPrefix(rel, "Tests/") || strings.Contains(rel, "/Tests/"))
	},
}

// configFileNames is the set of files internal/lintconfig parses.
// Mirrored here so the walker can flag them up-front.
var configFileNames = map[string]bool{
	".editorconfig":      true,
	".prettierrc":        true,
	".prettierrc.json":   true,
	".prettierrc.yaml":   true,
	".prettierrc.yml":    true,
	".eslintrc":          true,
	".eslintrc.json":     true,
	".eslintrc.cjs":      true,
	".eslintrc.js":       true,
	".eslintrc.yaml":     true,
	"tsconfig.json":      true,
	".rubocop.yml":       true,
	".rubocop.yaml":      true,
	"pyproject.toml":     true,
	"setup.cfg":          true,
	"rustfmt.toml":       true,
	"clippy.toml":        true,
	".golangci.yml":      true,
	".golangci.yaml":     true,
	"phpcs.xml":          true,
	"checkstyle.xml":     true,
	".stylelintrc":       true,
	".stylelintrc.json":  true,
	".markdownlint.json": true,
	".markdownlint.yaml": true,
	"CODEOWNERS":         true,
	"commitlint.config.cjs": true,
	"commitlint.config.js":  true,
	"renovate.json":      true,
	".gitleaks.toml":     true,
}

// Walk walks the repo and returns the file list + stats.
//
// maxFiles caps the visit count; budgetBytes caps the cumulative file
// size we read. Either zero means no cap. We always skip vendored and
// build-output directories.
//
// The walker is single-threaded by design — its bottleneck is the
// filesystem, not CPU. The downstream symbol-builder pool fans out.
func Walk(root string, maxFiles int, budgetBytes int64) ([]File, Stats, error) {
	if root == "" {
		return nil, Stats{}, errors.New("walker: empty root")
	}
	info, err := os.Stat(root)
	if err != nil {
		return nil, Stats{}, err
	}
	if !info.IsDir() {
		return nil, Stats{}, errors.New("walker: root is not a directory")
	}

	stats := Stats{ByLanguage: map[string]int{}}
	var files []File

	walkFn := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// Ignore permission errors — they shouldn't crash the run.
			if errors.Is(err, fs.ErrPermission) {
				return nil
			}
			return err
		}
		if d.IsDir() {
			if path == root {
				return nil
			}
			if skipDirs[d.Name()] || strings.HasPrefix(d.Name(), ".") &&
				d.Name() != ".github" && d.Name() != ".cursor" {
				return fs.SkipDir
			}
			stats.Directories++
			return nil
		}

		if maxFiles > 0 && stats.Files >= maxFiles {
			return fs.SkipAll
		}

		fi, ferr := d.Info()
		if ferr != nil {
			return nil
		}
		size := fi.Size()
		if budgetBytes > 0 && stats.BySize+size > budgetBytes {
			return fs.SkipAll
		}

		rel, _ := filepath.Rel(root, path)
		rel = filepath.ToSlash(rel)
		lang := classify(rel, path)

		f := File{
			AbsPath:  path,
			RelPath:  rel,
			Size:     size,
			Language: lang,
			IsTest:   isTest(rel),
			IsConfig: configFileNames[d.Name()],
		}
		files = append(files, f)
		stats.Files++
		stats.BySize += size
		stats.ByLanguage[lang]++
		return nil
	}

	if err := filepath.WalkDir(root, walkFn); err != nil {
		return nil, Stats{}, err
	}
	return files, stats, nil
}

func classify(rel, abs string) string {
	ext := strings.ToLower(filepath.Ext(rel))
	if lang, ok := extLang[ext]; ok {
		return lang
	}
	if ext == "" {
		// Try shebang sniff for extension-less files in scripts/.
		if isLikelyShellScript(abs) {
			return LangShell
		}
	}
	return LangUnknown
}

func isTest(rel string) bool {
	for _, p := range testPatterns {
		if p(rel) {
			return true
		}
	}
	return false
}

// isLikelyShellScript reads up to 64 bytes and checks for #!/.
func isLikelyShellScript(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	buf := make([]byte, 64)
	n, _ := bufio.NewReader(f).Read(buf)
	head := strings.TrimSpace(string(buf[:n]))
	if !strings.HasPrefix(head, "#!") {
		return false
	}
	return strings.Contains(head, "sh") || strings.Contains(head, "bash") || strings.Contains(head, "zsh")
}

// FilterByLanguage returns the subset of files with the given language.
func FilterByLanguage(files []File, lang string) []File {
	var out []File
	for _, f := range files {
		if f.Language == lang {
			out = append(out, f)
		}
	}
	return out
}

// LanguageBreakdownText returns a human-readable language breakdown
// suitable for the web-console output.
func LanguageBreakdownText(stats Stats) string {
	type kv struct {
		Lang string
		N    int
	}
	var rows []kv
	for k, v := range stats.ByLanguage {
		if k == LangUnknown || v == 0 {
			continue
		}
		rows = append(rows, kv{k, v})
	}
	// Sort descending by count, but the alloc-light way: bubble-sort
	// the top few since len(rows) ≤ 20.
	for i := 0; i < len(rows); i++ {
		for j := i + 1; j < len(rows); j++ {
			if rows[j].N > rows[i].N {
				rows[i], rows[j] = rows[j], rows[i]
			}
		}
	}
	var sb strings.Builder
	for i, r := range rows {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(r.Lang)
		sb.WriteString("(")
		sb.WriteString(itoa(r.N))
		sb.WriteString(")")
	}
	return sb.String()
}

// SafeMutex protects shared state across the cartographer goroutines.
type SafeMutex = sync.Mutex

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
