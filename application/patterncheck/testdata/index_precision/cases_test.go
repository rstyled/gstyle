package precision

import "testing"

func panics(f func() int) (result bool) {
	defer func() { result = recover() != nil }()
	f()
	return false
}
func TestRuntimeIndexCases(t *testing.T) {
	cases := []struct {
		name  string
		call  func() int
		panic bool
	}{
		{"IndexEmptyNegatedSafe", func() int { return IndexEmptyNegatedSafe(0) }, false},
		{"IndexEmptyConjunctionSafe", func() int { return IndexEmptyConjunctionSafe(0) }, false},
		{"IndexEmptyDisjunctionSafe", func() int { return IndexEmptyDisjunctionSafe(0) }, false},
		{"IndexGenericBad", func() int { return IndexGenericBad[int]() }, true},
		{"IndexGenericSafe", func() int { return IndexGenericSafe[int]() }, false},
		{"IndexConstantBad", IndexConstantBad, true}, {"IndexConstantSafe", IndexConstantSafe, false},
		{"IndexEmptyBad", func() int { return IndexEmptyBad(0) }, true}, {"IndexEmptySafe", func() int { return IndexEmptySafe(0) }, false},
		{"IndexNegativeBad", IndexNegativeBad, true}, {"IndexNegativeSafe", IndexNegativeSafe, false},
		{"IndexHighBad", IndexHighBad, true}, {"IndexHighSafe", IndexHighSafe, false},
		{"IndexBranchBad", func() int { return IndexBranchBad(2) }, true}, {"IndexBranchSafe", func() int { return IndexBranchSafe(2) }, false},
		{"IndexSliceBad", IndexSliceBad, true}, {"IndexSliceSafe", IndexSliceSafe, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := panics(tc.call); got != tc.panic {
				t.Fatalf("panic=%v,want %v", got, tc.panic)
			}
		})
	}
}
