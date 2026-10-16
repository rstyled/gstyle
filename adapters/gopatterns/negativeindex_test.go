package gopatterns

import (
	"math"
	"strings"
	"testing"

	"github.com/rstyled/gstyle/domain/patterns"
	"github.com/rstyled/gstyle/domain/pdg"
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
		{"negative branch", `func At(s []int, i int) int { if i < 0 { return s[i] }; return 0 }`, true},
		{"safe helper index", `func index() int { return 1 }; func At(s [2]int) int { i := index(); if i < len(s) { return s[i] }; return 0 }`, false},
		{"zero initialized loop", `func Sum(s []int) int { n := 0; for i := 0; i < len(s); i++ { n += s[i] }; return n }`, false},
		{"named int counter", `type Index int; func Sum(s []int) int { n := 0; for i := Index(0); i < Index(len(s)); i++ { n += s[i] }; return n }`, false},
		{"int64 counter", `func Sum(s []int) int { n := 0; for i := int64(0); i < int64(len(s)); i++ { n += s[i] }; return n }`, false},
		{"unknown initial counter", `func Sum(s []int, i int) int { n := 0; for ; i < len(s); i++ { n += s[i] }; return n }`, false},
		{"second increment needs new bound", `func Sum(s []int) int { n := 0; for i := 0; i < len(s); i++ { i++; if i < len(s) { n += s[i] } }; return n }`, false},
		{"negative initialized loop", `func Sum(s []int) int { n := 0; for i := -1; i < len(s); i++ { n += s[i] }; return n }`, false},
		{"loop body invalidates lower bound", `func Sum(s []int) int { n := 0; for i := 0; i < len(s); i++ { i = -1; if i < len(s) { n += s[i] } }; return n }`, true},
		{"narrow counter can overflow", `func Sum(s []int) int { n := 0; for i := int8(0); int(i) < len(s); i++ { if i < int8(len(s)) { n += s[i] } }; return n }`, false},
		{"inclusive length can overflow", `func Sum(s []int) int { n := 0; for i := 0; i <= len(s); i++ { if i < len(s) { n += s[i] } }; return n }`, false},
		{"decrement can become negative", `func Sum(s []int) int { n := 0; for i := 0; i < len(s); i-- { n += s[i] }; return n }`, false},
		{"rejection guard", `func At(s []int, i int) int { if i >= len(s) { return -1 }; return s[i] }`, false},
		{"reversed rejection guard", `func At(s []int, i int) int { if len(s) <= i { return -1 }; return s[i] }`, false},
		{"parenthesized rejection", `func At(s []int, i int) int { if (i >= len(s)) { return -1 }; return s[i] }`, false},
		{"combined rejection", `func At(s []int, i int) int { if i < 0 || i >= len(s) { return -1 }; return s[i] }`, false},
		{"unsigned array index cannot be negative", `func At(s [2]int, i uint) int { if i < uint(len(s)) { return s[i] }; return -1 }`, false},
		{"named unsigned index cannot be negative", `type Index uint; func At(s [2]int, i Index) int { if i < Index(len(s)) { return s[i] }; return -1 }`, false},
		{"shadowed length is not builtin", `func At(s []int, i int) int { len := func([]int) int { return 1 }; if i < len(s) { return s[i] }; return -1 }`, false},
		{"lower check only logs", `func At(s []int,i int) int { if i < 0 { println("negative") }; if i < len(s) { return s[i] }; return -1 }`, false},
		{"later lower check cannot protect access", `func At(s []int,i int) int { if i < len(s) { n := s[i]; if i < 0 { return -1 }; return n }; return -1 }`, false},
		{"lower guard invalidated by assignment", `func At(s []int,i int) int { if i < 0 { return -1 }; i = -2; if i < len(s) { return s[i] }; return -1 }`, true},
		{"range counter has nonnegative PDG definition", `func At(s []int) int { for i:=range s { if i < len(s) { return s[i] } }; return -1 }`, false},
		{"copied index retains lower guard", `func At(s []int,i int) int { if i < 0 { return -1 }; j := i; if j < len(s) { return s[j] }; return -1 }`, false},
		{"named length check", `func At(s []int,i int) int { n := len(s); if i < n { return s[i] }; return -1 }`, false},
		{"map negative keys do not panic", `func At(s map[int]int,i int) int { if i < len(s) { return s[i] }; return -1 }`, false},
		{"missing lower bound", `func At(s []int, i int) int { if i < len(s) { return s[i] }; return -1 }`, false},
		{"reversed comparison", `func At(s []int, i int) int { if len(s) > i { return s[i] }; return -1 }`, false},
		{"combined guard present", `func At(s []int, i int) int { if i >= 0 && i < len(s) { return s[i] }; return -1 }`, false},
		{"separate rejection", `func At(s []int, i int) int { if i < 0 { return -1 }; if i < len(s) { return s[i] }; return -1 }`, false},
		{"no bound check at all", `func At(s []int, i int) int { return s[i] }`, false},
		{"no index use", `func Len(s []int, i int) int { if i < len(s) { return len(s) }; return -1 }`, false},
		{"non-integer index", `func First(s []string, f float64) float64 { if f < float64(len(s)) { return f }; return -1 }`, false},
		{"checked branch without return", `func At(s []int, i int) int { x := 0; if i < len(s) { x = s[i] }; return x }`, false},
		{"different index guarded", `func At(s []int, i, j int) int { if j < len(s) { return s[j] }; if i < len(s) { return s[i] }; return -1 }`, false},
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
			if !strings.Contains(c.Observation, "is negative at this index") {
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
  i = -1
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
	src := `package example; func At(s []int, i, j int) int { i = -1; j = -2; return s[i] + s[j] }`
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
		if got := negativeIndexTestCandidates(t, src); len(got) != 0 {
			t.Fatalf("%s: unknown index produces %d findings", typ, len(got))
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
		{`if i>=len(s){return 0};return s[i]`, 0},
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

func TestNegativeIndexInvalidNodesDoNotHideValidOccurrence(t *testing.T) {
	function := FuncPdg{Name: "At", Path: "index.go", Pdg: pdg.Graph{Nodes: []pdg.Node{{Line: 11}, {Line: 22}}}}
	var report runtimeguard.Report
	for _, ref := range []pdg.Ref{0, 3, pdg.Ref(math.MaxUint32), 2, 1} {
		report.Accesses = append(report.Accesses, runtimeguard.Access{Node: ref, Index: true, Value: runtimeguard.Value{HasUpper: true, Upper: -1}})
	}
	got := negativeIndexCandidates(function, report)
	if len(got) != 1 || got[0].Sites[0].Line != 22 || got[0].Sites[0].Path != "index.go" {
		t.Fatalf("bad node reference hides or relocates the first valid risk: %+v", got)
	}
	function.Pdg.Nodes = nil
	if got := negativeIndexCandidates(function, report); len(got) != 0 {
		t.Fatalf("nil node table produces findings: %+v", got)
	}
}

func TestNegativeIndexCandidateProjection(t *testing.T) {
	function := FuncPdg{Name: "pkg.At", Path: "index.go", Pdg: pdg.Graph{Nodes: []pdg.Node{{Line: math.MaxUint32}}}}
	report := runtimeguard.Report{Accesses: []runtimeguard.Access{{Node: 1, Index: true, Value: runtimeguard.Value{HasUpper: true, Upper: -1}}}}
	candidates := negativeIndexCandidates(function, report)
	if len(candidates) != 1 {
		t.Fatalf("got %d findings", len(candidates))
	}
	got := candidates[0]
	if got.Kind != patterns.NegativeIndex || got.ScoreMilli != 800 || !strings.Contains(got.Observation, "recorded upper bound -1") {
		t.Fatalf("candidate lacks invalid-bound evidence: %+v", got)
	}

	if len(got.CounterEvidence) != 1 || len(got.Sites) != 1 {
		t.Fatalf("candidate drops evidence or source: %+v", got)
	}
	if got.Sites[0] != (patterns.Site{Path: "index.go", Line: math.MaxUint32, EndLine: math.MaxUint32, ID: "pkg.At", Name: "pkg.At"}) {
		t.Fatalf("candidate changes proved location: %+v", got)
	}
	if got.FixSpec != nil || got.Analysis != nil || got.Definitions != nil || got.PossibleRefactor != "" || got.Breakdown != (patterns.Breakdown{}) {
		t.Fatalf("advisory invents a fix or unrelated analysis: %+v", got)
	}
}
