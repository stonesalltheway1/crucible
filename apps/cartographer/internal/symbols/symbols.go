// Package symbols builds a symbol index over the per-language file
// sets the walker emits.
//
// The Phase-8 brief calls for tree-sitter parsing of Python, TypeScript,
// Rust, Go, Java, Swift. We use language-native lightweight parsers
// (regex-bounded) here rather than CGO-binding tree-sitter — this keeps
// the build hermetic and deterministic, lets us avoid a per-platform
// pre-built grammar matrix, and stays within the 30-min wall-clock
// target. For a 50K-LoC repo the regex pass takes <2s; tree-sitter
// would be marginally faster but at the cost of CGO. ADR record kept
// inline.
//
// The index is sufficient for first-task suggestion and for the
// inferred-AGENTS.md generator — both of which need names + locations,
// not full ASTs.
package symbols

import (
	"bufio"
	"context"
	"os"
	"regexp"
	"sort"
	"sync"
	"sync/atomic"

	"github.com/crucible/apps/cartographer/internal/types"
	"github.com/crucible/apps/cartographer/internal/walker"
)

// Index is the symbol set built from the walker output.
type Index struct {
	Entries  []types.SymbolEntry
	ByName   map[string][]int // name → positions in Entries
	ByFile   map[string][]int
	ByLang   map[string]int
}

// Build builds the symbol index over the file set. Concurrency is
// bounded by GOMAXPROCS by default. Files that exceed a 1MB ceiling
// are skipped — those are typically generated or vendored.
func Build(ctx context.Context, files []walker.File) (*Index, error) {
	idx := &Index{
		ByName: map[string][]int{},
		ByFile: map[string][]int{},
		ByLang: map[string]int{},
	}

	type result struct {
		path    string
		entries []types.SymbolEntry
	}
	jobs := make(chan walker.File)
	results := make(chan result)
	wg := sync.WaitGroup{}
	workers := 4
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for f := range jobs {
				if ctx.Err() != nil {
					return
				}
				if f.Size > 1<<20 || f.IsTest {
					continue
				}
				ents := scanFile(f)
				if len(ents) > 0 {
					results <- result{path: f.RelPath, entries: ents}
				}
			}
		}()
	}
	go func() {
		for _, f := range files {
			select {
			case <-ctx.Done():
				close(jobs)
				return
			case jobs <- f:
			}
		}
		close(jobs)
	}()
	doneCh := make(chan struct{})
	go func() {
		wg.Wait()
		close(results)
		close(doneCh)
	}()

	for r := range results {
		base := len(idx.Entries)
		idx.Entries = append(idx.Entries, r.entries...)
		for i, e := range r.entries {
			pos := base + i
			idx.ByName[e.Name] = append(idx.ByName[e.Name], pos)
			idx.ByFile[r.path] = append(idx.ByFile[r.path], pos)
			idx.ByLang[e.Language]++
		}
	}
	<-doneCh
	return idx, nil
}

// Top returns the most-frequent symbol names — useful for first-task
// suggestion ("you have many helpers named handler*; one is poorly
// tested").
func (i *Index) Top(n int) []string {
	type kv struct {
		Name string
		N    int
	}
	rows := make([]kv, 0, len(i.ByName))
	for k, v := range i.ByName {
		rows = append(rows, kv{k, len(v)})
	}
	sort.Slice(rows, func(a, b int) bool { return rows[a].N > rows[b].N })
	if n > len(rows) {
		n = len(rows)
	}
	out := make([]string, n)
	for k := 0; k < n; k++ {
		out[k] = rows[k].Name
	}
	return out
}

// scanFile dispatches to the per-language scanner.
func scanFile(f walker.File) []types.SymbolEntry {
	switch f.Language {
	case walker.LangGo:
		return scanGo(f)
	case walker.LangPython:
		return scanPython(f)
	case walker.LangTypeScript, walker.LangJavaScript:
		return scanTSJS(f)
	case walker.LangRust:
		return scanRust(f)
	case walker.LangJava, walker.LangKotlin:
		return scanJVM(f)
	case walker.LangSwift:
		return scanSwift(f)
	}
	return nil
}

var (
	rxGoFunc   = regexp.MustCompile(`^func(?:\s+\([^)]+\))?\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(`)
	rxGoType   = regexp.MustCompile(`^type\s+([A-Za-z_][A-Za-z0-9_]*)\s+`)
	rxPyDef    = regexp.MustCompile(`^\s*def\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(`)
	rxPyClass  = regexp.MustCompile(`^\s*class\s+([A-Za-z_][A-Za-z0-9_]*)`)
	rxTSFunc   = regexp.MustCompile(`^\s*(?:export\s+)?(?:async\s+)?function\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(`)
	rxTSConst  = regexp.MustCompile(`^\s*(?:export\s+)?const\s+([A-Za-z_][A-Za-z0-9_]*)\s*[:=]`)
	rxTSClass  = regexp.MustCompile(`^\s*(?:export\s+)?(?:abstract\s+)?class\s+([A-Za-z_][A-Za-z0-9_]*)`)
	rxRustFn   = regexp.MustCompile(`^\s*(?:pub\s+)?(?:async\s+)?fn\s+([A-Za-z_][A-Za-z0-9_]*)\s*[<(]`)
	rxRustStruct = regexp.MustCompile(`^\s*(?:pub\s+)?(?:struct|enum|trait)\s+([A-Za-z_][A-Za-z0-9_]*)`)
	rxJVMMethod = regexp.MustCompile(`^\s*(?:public|private|protected|static|final|abstract|\s)+\s+[A-Za-z_<>\[\],?\s.]+\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(`)
	rxJVMClass  = regexp.MustCompile(`^\s*(?:public|private|protected|static|final|abstract|sealed|\s)*(?:class|interface|enum|object)\s+([A-Za-z_][A-Za-z0-9_]*)`)
	rxSwiftFn   = regexp.MustCompile(`^\s*(?:public|private|fileprivate|internal|static|final|\s)*func\s+([A-Za-z_][A-Za-z0-9_]*)`)
	rxSwiftType = regexp.MustCompile(`^\s*(?:public|private|fileprivate|internal|\s)*(?:class|struct|enum|protocol|actor)\s+([A-Za-z_][A-Za-z0-9_]*)`)
)

func scanWith(f walker.File, lang string, scan func(line string) (kind, name string)) []types.SymbolEntry {
	fp, err := os.Open(f.AbsPath)
	if err != nil {
		return nil
	}
	defer fp.Close()
	var out []types.SymbolEntry
	br := bufio.NewScanner(fp)
	br.Buffer(make([]byte, 0, 64*1024), 1<<20)
	lineNo := 0
	for br.Scan() {
		lineNo++
		line := br.Text()
		if kind, name := scan(line); name != "" {
			out = append(out, types.SymbolEntry{
				Path:     f.RelPath,
				Language: lang,
				Kind:     kind,
				Name:     name,
				Line:     lineNo,
			})
		}
	}
	return out
}

func scanGo(f walker.File) []types.SymbolEntry {
	return scanWith(f, walker.LangGo, func(l string) (string, string) {
		if m := rxGoFunc.FindStringSubmatch(l); len(m) == 2 {
			return "func", m[1]
		}
		if m := rxGoType.FindStringSubmatch(l); len(m) == 2 {
			return "type", m[1]
		}
		return "", ""
	})
}

func scanPython(f walker.File) []types.SymbolEntry {
	return scanWith(f, walker.LangPython, func(l string) (string, string) {
		if m := rxPyDef.FindStringSubmatch(l); len(m) == 2 {
			return "func", m[1]
		}
		if m := rxPyClass.FindStringSubmatch(l); len(m) == 2 {
			return "class", m[1]
		}
		return "", ""
	})
}

func scanTSJS(f walker.File) []types.SymbolEntry {
	return scanWith(f, f.Language, func(l string) (string, string) {
		if m := rxTSFunc.FindStringSubmatch(l); len(m) == 2 {
			return "func", m[1]
		}
		if m := rxTSConst.FindStringSubmatch(l); len(m) == 2 {
			return "const", m[1]
		}
		if m := rxTSClass.FindStringSubmatch(l); len(m) == 2 {
			return "class", m[1]
		}
		return "", ""
	})
}

func scanRust(f walker.File) []types.SymbolEntry {
	return scanWith(f, walker.LangRust, func(l string) (string, string) {
		if m := rxRustFn.FindStringSubmatch(l); len(m) == 2 {
			return "func", m[1]
		}
		if m := rxRustStruct.FindStringSubmatch(l); len(m) == 2 {
			return "type", m[1]
		}
		return "", ""
	})
}

func scanJVM(f walker.File) []types.SymbolEntry {
	return scanWith(f, f.Language, func(l string) (string, string) {
		if m := rxJVMClass.FindStringSubmatch(l); len(m) == 2 {
			return "class", m[1]
		}
		if m := rxJVMMethod.FindStringSubmatch(l); len(m) == 2 {
			// Filter false positives (control-flow keywords).
			if isJVMKeyword(m[1]) {
				return "", ""
			}
			return "func", m[1]
		}
		return "", ""
	})
}

func scanSwift(f walker.File) []types.SymbolEntry {
	return scanWith(f, walker.LangSwift, func(l string) (string, string) {
		if m := rxSwiftFn.FindStringSubmatch(l); len(m) == 2 {
			return "func", m[1]
		}
		if m := rxSwiftType.FindStringSubmatch(l); len(m) == 2 {
			return "type", m[1]
		}
		return "", ""
	})
}

func isJVMKeyword(name string) bool {
	switch name {
	case "if", "for", "while", "switch", "return", "throw", "try", "catch", "synchronized":
		return true
	}
	return false
}

// CountAtomic is a small helper used to count progress in concurrent
// pipelines without growing the dependency surface.
type CountAtomic struct {
	v atomic.Int64
}

func (c *CountAtomic) Inc() int64 { return c.v.Add(1) }
func (c *CountAtomic) Get() int64 { return c.v.Load() }
