package gopatterns

import (
	"github.com/rstyled/gstyle/domain/runtimeguard"
	"testing"
)

func TestRuntimeEmbeddedPointerAccess(t *testing.T) {
	for _, receiver := range []string{"Outer", "*Outer"} {
		src := `package p; type Inner struct { Value int }; type Outer struct { *Inner }; func f(p ` + receiver + `,q *int) int { if q==nil{return 0}; return p.Value }`
		if got := nilGuardTestCandidates(t, src); len(got) != 1 {
			t.Fatalf("%s: got %d findings, want 1", receiver, len(got))
		}
	}
}

func TestRuntimeConstantBranches(t *testing.T) {
	for _, condition := range []string{"false", "!true", "1 == 2"} {
		graphs := runtimeTestGraphs(t, `package p; func f(i int) int { if `+condition+` { return []int{1,2}[i] }; return 0 }`)
		report, err := runtimeguard.Analyze(&graphs[0].Pdg)
		if err != nil {
			t.Fatal(err)
		}
		if len(report.Accesses) != 0 {
			t.Fatalf("%s: got %d unreachable accesses", condition, len(report.Accesses))
		}
	}
}

func TestRuntimeStringIndex(t *testing.T) {
	graphs := runtimeTestGraphs(t, `package p; func f(s string,i int) byte { if i>=len(s){return 0}; return s[i] }`)
	report, err := runtimeguard.Analyze(&graphs[0].Pdg)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Accesses) != 1 || !report.Accesses[0].Index || !report.Accesses[0].Value.UpperChecked {
		t.Fatalf("string access missing upper-bound facts: %+v", report)
	}
}
