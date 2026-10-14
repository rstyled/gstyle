package gopatterns

import (
	"fmt"
	"github.com/rstyled/gstyle/domain/patterns"
	"strings"
	"testing"
)

func TestAuditNativeAttentionBounds(t *testing.T) {
	for _, count := range []int{1, 25} {
		var src strings.Builder
		src.WriteString("package p;func f(x int)int{v0:=x;")
		for i := 1; i <= count; i++ {
			fmt.Fprintf(&src, "v%d:=v%d+1;", i, i-1)
		}
		fmt.Fprintf(&src, "return v%d}", count)
		graph := runtimeTestGraphs(t, src.String())[0].Pdg
		facts := []*patterns.MiningFacts{{ID: "f", Name: "f", Canonical: &graph}}
		results := map[string]int{"thin_slice": len(patterns.FindThinSlices(facts)), "barrier_slice": len(patterns.FindBarrierSlices(facts)), "chop": len(patterns.FindChops(facts))}
		for kind, n := range results {
			want := count == 25
			if (n > 0) != want {
				t.Errorf("%s chain=%d findings=%d", kind, count, n)
			}
		}
	}
}
