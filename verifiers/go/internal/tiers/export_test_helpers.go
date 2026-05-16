// Exported test-helper shims. These wrap unexported internals so the
// module-root pool_test.go can drive the parser without lifting
// production functions to the public API.
//
// We deliberately give these helpers the *ForTest suffix so a
// production import would stand out in review. They live in a
// non-_test.go file because pool_test.go is in an external package
// (`verifygotest`), and Go's _test.go visibility scoping is
// per-package — external packages can't see internal helpers exposed
// only in foo_test.go.

package tiers

// ParseMutestingOutputForTest is the test-only re-export of
// parseMutestingOutput.
func ParseMutestingOutputForTest(raw []byte) Tier0Stats {
	return parseMutestingOutput(raw)
}

// DiscoverPropertiesForTest is the test-only re-export of
// discoverProperties.
func DiscoverPropertiesForTest(cfg PBTConfig) []string {
	return discoverProperties(cfg)
}

// FuzzTargetForTest mirrors the unexported fuzzTarget so tests can
// assert on the Name field without importing internals reflectively.
type FuzzTargetForTest struct {
	Package string
	Name    string
}

// DiscoverFuzzTargetsForTest is the test-only re-export of
// discoverFuzzTargets, projected to the public shim type.
func DiscoverFuzzTargetsForTest(cfg PBTConfig) []FuzzTargetForTest {
	internal := discoverFuzzTargets(cfg)
	out := make([]FuzzTargetForTest, len(internal))
	for i, t := range internal {
		out[i] = FuzzTargetForTest{Package: t.Package, Name: t.Name}
	}
	return out
}
