// Package weak is the under-tested counterpart of fixtures/good.
//
// The fixture's branches are deliberately under-covered: the tests
// only exercise the "happy path" for IsEven and the empty-slice
// shortcut for Average. A real mutation tester finds several easy
// survivors here (replace `x%2 == 0` with `x%2 != 0`, replace `> 0`
// with `>= 0`, etc.) and Tier 0 returns Failed.
package weak

// IsEven returns true iff x is even. The test only checks IsEven(2);
// a mutator that flips the operator survives.
func IsEven(x int) bool {
	return x%2 == 0
}

// Average returns the arithmetic mean of xs, or 0 for empty input.
// The test only exercises the empty shortcut.
func Average(xs []int) float64 {
	if len(xs) == 0 {
		return 0
	}
	total := 0
	for _, v := range xs {
		total += v
	}
	return float64(total) / float64(len(xs))
}
