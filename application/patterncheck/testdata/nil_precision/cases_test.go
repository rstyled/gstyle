package precision

import "testing"

func panics(f func() int) (result bool) {
	defer func() { result = recover() != nil }()
	f()
	return false
}
func TestRuntimeNilCases(t *testing.T) {
	value := 7
	cases := []struct {
		name  string
		call  func() int
		panic bool
	}{
		{"NilAssignedBad", NilAssignedBad, true}, {"NilAssignedSafe", NilAssignedSafe, false},
		{"NilGuardBad", func() int { return NilGuardBad(nil) }, true}, {"NilGuardSafe", func() int { return NilGuardSafe(&value) }, false},
		{"NilAliasBad", func() int { return NilAliasBad(nil) }, true}, {"NilAliasSafe", func() int { return NilAliasSafe(&value) }, false},
		{"NilBranchBad", func() int { return NilBranchBad(nil) }, true}, {"NilBranchSafe", func() int { return NilBranchSafe(&value) }, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := panics(tc.call); got != tc.panic {
				t.Fatalf("panic=%v,want %v", got, tc.panic)
			}
		})
	}
	for _, call := range []func(*int) int{NilGuardSafe, NilAliasSafe, NilBranchSafe} {
		if panics(func() int { return call(nil) }) {
			t.Fatal("safe nil branch panics")
		}
	}
}
