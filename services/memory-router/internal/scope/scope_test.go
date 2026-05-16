package scope

import (
	"testing"

	memoryspec "github.com/crucible/memory-spec/go"
)

func TestMatch_RepoExactness(t *testing.T) {
	conv := memoryspec.ScopeFilter{Repo: "acme/payments"}
	q := memoryspec.ScopeFilter{Repo: "acme/billing"}
	if Match(conv, q) {
		t.Fatal("different repos must not match")
	}
}

func TestMatch_FileGlobRecursive(t *testing.T) {
	conv := memoryspec.ScopeFilter{FileGlob: "api/**/*.ts"}
	q := memoryspec.ScopeFilter{FileGlob: "api/handlers/oauth.ts"}
	if !Match(conv, q) {
		t.Fatalf("recursive **/*.ts must match handlers/oauth.ts")
	}
}

func TestMatch_EmptyConventionScopeWildcards(t *testing.T) {
	conv := memoryspec.ScopeFilter{}
	q := memoryspec.ScopeFilter{Repo: "acme/foo", FileGlob: "anywhere.go"}
	if !Match(conv, q) {
		t.Fatal("empty convention scope must match anything")
	}
}

func TestMatch_CategoryNarrowing(t *testing.T) {
	conv := memoryspec.ScopeFilter{Category: "Logging"}
	q := memoryspec.ScopeFilter{Category: "Naming"}
	if Match(conv, q) {
		t.Fatal("disjoint categories must not match")
	}
}

func TestMatch_ConcreteFilenameAgainstStarGlob(t *testing.T) {
	conv := memoryspec.ScopeFilter{FileGlob: "src/db/*.sql"}
	q := memoryspec.ScopeFilter{FileGlob: "src/db/migration_001.sql"}
	if !Match(conv, q) {
		t.Fatal("star glob must match concrete sibling file")
	}
}

func TestMatch_DoubleStarLeadingWildcard(t *testing.T) {
	conv := memoryspec.ScopeFilter{FileGlob: "**/test_*.py"}
	q := memoryspec.ScopeFilter{FileGlob: "tests/auth/test_oauth.py"}
	if !Match(conv, q) {
		t.Fatal("leading ** must match nested directories")
	}
}

func TestNormalize_TrimsAndLowersRepo(t *testing.T) {
	got := Normalize(memoryspec.ScopeFilter{Repo: "  AcMe/Payments  "})
	if got.Repo != "acme/payments" {
		t.Fatalf("normalize repo: got %q", got.Repo)
	}
}

func TestAnyMatch_AcceptsFirstMatch(t *testing.T) {
	convs := []memoryspec.ScopeFilter{
		{Repo: "different/repo"},
		{FileGlob: "src/**/*.go"},
	}
	q := memoryspec.ScopeFilter{Repo: "acme/svc", FileGlob: "src/auth/login.go"}
	if !AnyMatch(convs, q) {
		t.Fatal("AnyMatch must return true on the second scope's glob")
	}
}
