// Package diff filters a VerificationRequest's Diff down to the Go
// source files the per-language runner is allowed to operate on.
//
// All tiers consume this filtered list. Mutation testing in
// particular MUST be diff-scoped (otherwise wall-clock explodes on
// large changes — see verifier-pipeline.md and the testreport
// invariant: MutationStats.DiffScoped MUST be true).
package diff

import (
	"path/filepath"
	"strings"

	cruciblev1 "github.com/crucible/sdk-go/crucible/v1"
)

// FileSet is the result of filtering a diff. Source and Test are
// disjoint subsets of the diff's FileChange list.
type FileSet struct {
	// Source is non-test .go files that should be mutated and built.
	Source []cruciblev1.FileChange
	// Test is _test.go files; the PBT/fuzz driver scans these for
	// rapid properties and testing.F fuzz targets.
	Test []cruciblev1.FileChange
}

// FilterGo returns the Go-source / Go-test partition of the diff.
//
// Files with Action "deleted" are excluded — there's nothing to
// mutate, no test to run.
//
// Vendored code (vendor/ prefix) and generated code with a build
// tag of `//go:build generated` are NOT filtered here — call sites
// that want the conservative set should post-filter. For Tier 0 we
// keep them in because survived mutants in generated code is still
// useful signal that the diff didn't add coverage there.
func FilterGo(d cruciblev1.Diff) FileSet {
	var out FileSet
	for _, f := range d.Files {
		if f.Action == "deleted" {
			continue
		}
		if !IsGoFile(f.Path) {
			continue
		}
		if IsTestFile(f.Path) {
			out.Test = append(out.Test, f)
		} else {
			out.Source = append(out.Source, f)
		}
	}
	return out
}

// IsGoFile reports whether p has a .go extension. Case-insensitive
// because Windows.
func IsGoFile(p string) bool {
	return strings.EqualFold(filepath.Ext(p), ".go")
}

// IsTestFile reports whether p is a Go test file (ends _test.go).
func IsTestFile(p string) bool {
	base := filepath.Base(p)
	return strings.HasSuffix(strings.ToLower(base), "_test.go")
}

// SourcePaths returns just the paths of the Source slice.
func (fs FileSet) SourcePaths() []string {
	out := make([]string, len(fs.Source))
	for i, f := range fs.Source {
		out[i] = f.Path
	}
	return out
}

// TestPaths returns just the paths of the Test slice.
func (fs FileSet) TestPaths() []string {
	out := make([]string, len(fs.Test))
	for i, f := range fs.Test {
		out[i] = f.Path
	}
	return out
}

// Packages returns the unique directory parents of fs.Source, useful
// for `go test ./...`-style invocations scoped to the diff.
func (fs FileSet) Packages() []string {
	seen := map[string]struct{}{}
	out := make([]string, 0)
	for _, f := range fs.Source {
		dir := filepath.ToSlash(filepath.Dir(f.Path))
		if dir == "" || dir == "." {
			dir = "."
		}
		if _, ok := seen[dir]; ok {
			continue
		}
		seen[dir] = struct{}{}
		out = append(out, dir)
	}
	return out
}
