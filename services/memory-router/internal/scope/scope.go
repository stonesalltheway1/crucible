// Package scope normalizes ScopeFilter inputs and matches them against
// convention scopes. The matcher is glob-aware (the cartographer writes
// rules with `api/**/*.ts`-style scopes; recall queries may pass
// `api/handlers/oauth.ts` and need to match the broader rule).
package scope

import (
	"path"
	"strings"

	memoryspec "github.com/crucible/memory-spec/go"
	cruciblev1 "github.com/crucible/sdk-go/crucible/v1"
)

// Normalize lowercases the repo + file_glob, trims, and resolves "" to
// the wildcard match. The router stores ScopeFilter rows in their
// normalized form so matching is byte-exact for repo + category and
// glob-aware for file_glob.
func Normalize(s cruciblev1.ScopeFilter) memoryspec.ScopeFilter {
	return memoryspec.ScopeFilter{
		Repo:     strings.ToLower(strings.TrimSpace(s.Repo)),
		FileGlob: strings.TrimSpace(s.FileGlob),
		Category: strings.TrimSpace(s.Category),
	}
}

// Match reports whether the convention's scope covers a query's scope.
//
// The match rules:
//   - empty repo on the convention matches any repo on the query
//   - non-empty repo must equal the query's repo
//   - empty file_glob matches anything
//   - non-empty file_glob is glob-matched against the query's file_glob
//     OR the query's file_glob (treated as a concrete path) is matched
//     against the convention glob
//   - category match is byte-exact when set on the convention
func Match(convention, query memoryspec.ScopeFilter) bool {
	if convention.Repo != "" && query.Repo != "" && convention.Repo != query.Repo {
		return false
	}
	if convention.Category != "" && query.Category != "" && convention.Category != query.Category {
		return false
	}
	if convention.FileGlob == "" || query.FileGlob == "" {
		return true
	}
	if convention.FileGlob == query.FileGlob {
		return true
	}
	// Cheap glob match for the cartographer's standard patterns. Uses
	// path.Match (POSIX); for `**` recursive globs we re-implement
	// below since stdlib doesn't support them.
	if globMatch(convention.FileGlob, query.FileGlob) {
		return true
	}
	if globMatch(query.FileGlob, convention.FileGlob) {
		return true
	}
	return false
}

// globMatch supports `**` for recursive directory matching in addition
// to path.Match's `*` and `?`. Mirrors the semantics commonly used by
// .gitignore and .eslintrc.
func globMatch(pattern, name string) bool {
	if strings.Contains(pattern, "**") {
		// Split on `**`, match each segment via path.Match against the
		// corresponding chunk of name.
		left, right, _ := strings.Cut(pattern, "**")
		left = strings.TrimSuffix(left, "/")
		right = strings.TrimPrefix(right, "/")
		if !strings.HasPrefix(name, left) {
			if left != "" {
				return false
			}
		}
		remainder := name
		if left != "" {
			remainder = strings.TrimPrefix(name, left)
			remainder = strings.TrimPrefix(remainder, "/")
		}
		if right == "" {
			return true
		}
		// Iterate suffixes of remainder, trying to match right against any.
		for i := 0; i <= len(remainder); i++ {
			tail := remainder[i:]
			ok, err := path.Match(right, tail)
			if err == nil && ok {
				return true
			}
		}
		return false
	}
	ok, err := path.Match(pattern, name)
	return err == nil && ok
}

// AnyMatch returns true if any of the convention scopes covers the
// query. Used by the procedural-store traversal when multiple
// (Convention)-[:APPLIES_TO]->(Scope) edges land on the same node.
func AnyMatch(scopes []memoryspec.ScopeFilter, query memoryspec.ScopeFilter) bool {
	for _, s := range scopes {
		if Match(s, query) {
			return true
		}
	}
	return false
}
