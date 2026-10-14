package gopatterns

import (
	"strings"
	"testing"

	"github.com/rstyled/gstyle/domain/patterns"
	"github.com/rstyled/gstyle/domain/runtimeguard"
)

func negativeIndexTestCandidates(t *testing.T, src string) []patterns.Candidate {
	t.Helper()
	var out []patterns.Candidate
	for _, function := range runtimeTestGraphs(t, src) {
		report, err := runtimeguard.Analyze(&function.Pdg)
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, negativeIndexCandidates(function, report)...)
	}

	return out
}

func TestNegativeIndex(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		find       bool
	}{
		{"rejection guard", `func At(s []int, i int) int { if i >= len(s) { return -1 }; return s[i] }`, true},
		{"reversed rejection guard", `func At(s []int, i int) int { if len(s) <= i { return -1 }; return s[i] }`, true},
		{"parenthesized rejection", `func At(s []int, i int) int { if (i >= len(s)) { return -1 }; return s[i] }`, true},
		{"combined rejection", `func At(s []int, i int) int { if i < 0 || i >= len(s) { return -1 }; return s[i] }`, false},
		{"unsigned array index cannot be negative", `func At(s [2]int, i uint) int { if i < uint(len(s)) { return s[i] }; return -1 }`, false},
		{"named unsigned index cannot be negative", `type Index uint; func At(s [2]int, i Index) int { if i < Index(len(s)) { return s[i] }; return -1 }`, false},
		{"shadowed length is not builtin", `func At(s []int, i int) int { len := func([]int) int { return 1 }; if i < len(s) { return s[i] }; return -1 }`, false},
		{"lower check only logs", `func At(s []int,i int) int { if i < 0 { println("negative") }; if i < len(s) { return s[i] }; return -1 }`, true},
		{"later lower check cannot protect access", `func At(s []int,i int) int { if i < len(s) { n := s[i]; if i < 0 { return -1 }; return n }; return -1 }`, true},
		{"lower guard invalidated by assignment", `func At(s []int,i int) int { if i < 0 { return -1 }; i = -2; if i < len(s) { return s[i] }; return -1 }`, true},
		{"range counter has nonnegative PDG definition", `func At(s []int) int { for i:=range s { if i < len(s) { return s[i] } }; return -1 }`, false},
		{"copied index retains lower guard", `func At(s []int,i int) int { if i < 0 { return -1 }; j := i; if j < len(s) { return s[j] }; return -1 }`, false},
		{"named length check", `func At(s []int,i int) int { n := len(s); if i < n { return s[i] }; return -1 }`, true},
		{"map negative keys do not panic", `func At(s map[int]int,i int) int { if i < len(s) { return s[i] }; return -1 }`, false},
		{"missing lower bound", `func At(s []int, i int) int { if i < len(s) { return s[i] }; return -1 }`, true},
		{"reversed comparison", `func At(s []int, i int) int { if len(s) > i { return s[i] }; return -1 }`, true},
		{"combined guard present", `func At(s []int, i int) int { if i >= 0 && i < len(s) { return s[i] }; return -1 }`, false},
		{"separate rejection", `func At(s []int, i int) int { if i < 0 { return -1 }; if i < len(s) { return s[i] }; return -1 }`, false},
		{"no bound check at all", `func At(s []int, i int) int { return s[i] }`, false},
		{"no index use", `func Len(s []int, i int) int { if i < len(s) { return len(s) }; return -1 }`, false},
		{"non-integer index", `func First(s []string, f float64) float64 { if f < float64(len(s)) { return f }; return -1 }`, false},
		{"checked branch without return", `func At(s []int, i int) int { x := 0; if i < len(s) { x = s[i] }; return x }`, true},
		{"different index guarded", `func At(s []int, i, j int) int { if j < len(s) { return s[j] }; if i < len(s) { return s[i] }; return -1 }`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			src := "package example\n" + tc.body + "\n"
			candidates := negativeIndexTestCandidates(t, src)
			if (len(candidates) != 0) != tc.find {
				t.Fatalf("got %d candidates, want finding=%v", len(candidates), tc.find)
			}
			if !tc.find {
				return
			}
			c := candidates[0]
			if c.Kind != patterns.NegativeIndex {
				t.Errorf("kind=%q want negative_index", c.Kind)
			}
			if !strings.Contains(c.Observation, "no lower-bound check") {
				t.Errorf("observation missing lower-bound wording: %s", c.Observation)
			}
			if len(c.CounterEvidence) == 0 {
				t.Error("detection-only finding must carry counter-evidence")
			}
			if c.FixSpec != nil {
				t.Error("detection-only finding must not carry a fix spec")
			}
		})
	}
}

func TestNegativeIndexOneFindingPerBinding(t *testing.T) {
	src := `package example
 func At(s []int, i int) int {
  if i >= len(s) { return -1 }
  if i < len(s) { return s[i] }
  return -1
 }`
	for range 20 {
		got := negativeIndexTestCandidates(t, src)
		if len(got) != 1 {
			t.Fatalf("got %d findings, want one per binding", len(got))
		}
	}
}

func TestNegativeIndexStableOrderWithinGuard(t *testing.T) {
	src := `package example; func At(s []int, i, j int) int { if i >= len(s) && j >= len(s) { return -1 }; return s[i] + s[j] }`
	var first string
	for range 30 {
		got := negativeIndexTestCandidates(t, src)
		if len(got) != 2 {
			t.Fatalf("got %d findings, want two", len(got))
		}
		if first == "" {
			first = got[0].Observation
		}
		if got[0].Observation != first {
			t.Fatalf("unstable order: %s then %s", first, got[0].Observation)
		}
	}
}

func TestNegativeStringIndexReview(t *testing.T) {
	for _, typ := range []string{"string", "Text"} {
		src := `package p; type Text string; func f(s ` + typ + `,i int) byte { if i>=len(s){return 0}; return s[i] }`
		if got := negativeIndexTestCandidates(t, src); len(got) != 1 {
			t.Fatalf("%s: got %d findings, want 1", typ, len(got))
		}
	}
	src := `package p; func f(s string,i int) byte { if i<0 || i>=len(s){return 0}; return s[i] }`
	if got := negativeIndexTestCandidates(t, src); len(got) != 0 {
		t.Fatalf("safe string access: got %d findings", len(got))
	}
}

func TestNegativePointerArrayIndex(t *testing.T) {
	cases := []struct {
		body string
		want int
	}{
		{`if i>=len(s){return 0};return s[i]`, 1},
		{`if i<0 || i>=len(s){return 0};return s[i]`, 0},
		{`for i=range s{if i<len(s){return s[i]}};return 0`, 0},
	}
	for _, tc := range cases {
		src := `package p;func f(s *[3]int,i int)int{` + tc.body + `}`
		if got := negativeIndexTestCandidates(t, src); len(got) != tc.want {
			t.Fatalf("%s: got %d findings,want %d", tc.body, len(got), tc.want)
		}
	}
}
