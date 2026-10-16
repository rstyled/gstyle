package gopatterns

import "testing"

func TestRuntimeBranchComposition(t *testing.T) {
	cases := []struct {
		name, source string
		warning      bool
	}{
		{"impossible disjunction", `func At(s []int, i int) int { if i >= 0 { if i < 0 || i < 0 { if i < len(s) { return s[i] } } }; return 0 }`, false},
		{"impossible false conjunction", `func At(s []int, i int) int { if i >= 0 { if i >= 0 && i >= 0 { return 0 }; if i < len(s) { return s[i] } }; return 0 }`, false},
		{"reachable disjunction", `func At(s []int, i int) int { if i < 0 || i >= 0 { if i < len(s) { return s[i] } }; return 0 }`, true},
		{"alternative copies stay independent", `func At(s []int, i, j int) int { if i >= 0 || j >= 0 { if i < len(s) { return s[i] } }; return 0 }`, true},
		{"sequential true conjunction", `func At(s []int, i int) int { if i >= 0 && i < len(s) { return s[i] }; return 0 }`, false},
		{"sequential false disjunction", `func At(s []int, i int) int { if !(i < 0 || i >= len(s)) { return s[i] }; return 0 }`, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := potentialNegativeAccesses(t, "package p; "+tc.source)
			if (len(got) > 0) != tc.warning {
				t.Fatalf("findings=%+v want warning=%v", got, tc.warning)
			}
		})
	}
}
