package weak

import "testing"

// TestIsEvenOnlyHappyPath under-tests IsEven on purpose. A mutator
// that flips `x%2 == 0` to `x%2 != 0` would survive this test on
// the single input 2, because IsEven(2) and the mutant !IsEven(2)
// would both pass and fail respectively — only ONE bool value gets
// exercised, so swap-style mutators see no coverage.
func TestIsEvenOnlyHappyPath(t *testing.T) {
	if !IsEven(2) {
		t.Fatal("expected 2 to be even")
	}
}

func TestAverageEmpty(t *testing.T) {
	if Average(nil) != 0 {
		t.Fatal("Average(nil) should be 0")
	}
}
