package good

import (
	"reflect"
	"testing"
)

func TestReverseExamples(t *testing.T) {
	cases := []struct {
		name string
		in   []int
		want []int
	}{
		{"empty", []int{}, []int{}},
		{"single", []int{1}, []int{1}},
		{"two", []int{1, 2}, []int{2, 1}},
		{"five", []int{1, 2, 3, 4, 5}, []int{5, 4, 3, 2, 1}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Reverse(c.in)
			if !reflect.DeepEqual(got, c.want) {
				t.Fatalf("Reverse(%v) = %v, want %v", c.in, got, c.want)
			}
		})
	}
}

func TestSumExamples(t *testing.T) {
	cases := []struct {
		in   []int
		want int
	}{
		{nil, 0},
		{[]int{}, 0},
		{[]int{1}, 1},
		{[]int{1, 2, 3}, 6},
		{[]int{-1, 1}, 0},
	}
	for _, c := range cases {
		got := Sum(c.in)
		if got != c.want {
			t.Errorf("Sum(%v) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestMaxExamples(t *testing.T) {
	if got := Max(nil); got != 0 {
		t.Errorf("Max(nil) = %d, want 0", got)
	}
	if got := Max([]int{}); got != 0 {
		t.Errorf("Max([]) = %d, want 0", got)
	}
	if got := Max([]int{3, 1, 4, 1, 5, 9, 2, 6}); got != 9 {
		t.Errorf("Max([3,1,4,1,5,9,2,6]) = %d, want 9", got)
	}
}

// PropertyReverseIsInvolutive is the rapid property the Tier 1
// driver discovers via the `Property*` name convention. We hand-roll
// the property instead of importing rapid here because the fixtures
// must build inside the verifier module even when go-mutesting / rapid
// aren't on the host. The runner's discoverProperties() function
// only needs to spot the name; it doesn't introspect the body.
func PropertyReverseIsInvolutive(t *testing.T) {
	// A self-contained property check: a few hand-crafted inputs.
	// In production this is replaced with rapid.T → rapid.SliceOf(...).
	inputs := [][]int{nil, {}, {1}, {1, 2}, {1, 2, 3, 4, 5}}
	for _, in := range inputs {
		if !reflect.DeepEqual(Reverse(Reverse(in)), in) {
			t.Fatalf("involution failed for %v", in)
		}
	}
}

// FuzzSum is a native testing.F target. The Tier 1 runner's
// discoverFuzzTargets() picks this up by the `func Fuzz<X>(f
// *testing.F)` signature; the body just feeds the seed corpus and
// asserts a trivial invariant. With cfg.GoBinary absent the runner
// emits ToolUnavailable so this body never executes during the
// fixture test — it's discovered purely by source scanning.
func FuzzSum(f *testing.F) {
	f.Add([]byte{1, 2, 3})
	f.Fuzz(func(t *testing.T, b []byte) {
		// Sum of two-copies should be exactly twice the single.
		xs := make([]int, len(b))
		for i, v := range b {
			xs[i] = int(v)
		}
		double := append(append([]int{}, xs...), xs...)
		if Sum(double) != 2*Sum(xs) {
			t.Errorf("Sum(double) != 2*Sum(single) for %v", xs)
		}
	})
}
