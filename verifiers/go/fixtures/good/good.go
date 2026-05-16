// Package good is a fixture used by the Tier 0/1 end-to-end test
// to demonstrate a well-tested Go package. The functions here are
// covered by both example-based tests AND a rapid property test, so
// the runner's mutation phase finds few survivors and the PBT phase
// returns Passed.
package good

// Reverse reverses s in place and returns it. Pure, deterministic,
// trivially invertible (Reverse(Reverse(s)) == s).
func Reverse(s []int) []int {
	out := make([]int, len(s))
	for i, v := range s {
		out[len(s)-1-i] = v
	}
	return out
}

// Sum returns the sum of xs. Zero for an empty slice.
func Sum(xs []int) int {
	total := 0
	for _, x := range xs {
		total += x
	}
	return total
}

// Max returns the largest element of xs, or zero if xs is empty.
// Returning zero on empty is documented behaviour the tests check.
func Max(xs []int) int {
	if len(xs) == 0 {
		return 0
	}
	m := xs[0]
	for _, x := range xs[1:] {
		if x > m {
			m = x
		}
	}
	return m
}
