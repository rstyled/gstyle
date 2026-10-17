package gopatterns

import (
	"fmt"
	"testing"

	"github.com/rstyled/gstyle/domain/pdg"
	"github.com/rstyled/gstyle/domain/resourceflow"
)

func resourcePathOracle(next [2][2]int, node int, active [2]bool) bool {
	if node == 2 {
		return true
	}
	if active[node] {
		return false
	}
	active[node] = true
	return resourcePathOracle(next, next[node][0], active) && resourcePathOracle(next, next[node][1], active)
}
func resourceOracleSource(next [2][2]int, replaced bool) string {
	write := ""
	if replaced {
		write = "resource=nil;"
	}
	return fmt.Sprintf(`package p;func f(resource *T,a,b bool){
 if false{goto L0};if false{goto L1};if false{goto L2};goto L0
 L0:%s if a{goto L%d}else{goto L%d}
 L1:if b{goto L%d}else{goto L%d}
 L2:resource.Close()
 };type T struct{};func(*T)Close(){}`, write, next[0][0], next[0][1], next[1][0], next[1][1])
}
func TestResourceFlowAgainstIndependentPathOracle(t *testing.T) {
	checked := 0
	for layout := 0; layout < 81; layout++ {
		digits := layout
		var next [2][2]int
		for i := range next {
			for j := range next[i] {
				next[i][j] = digits % 3
				digits /= 3
			}
		}
		for _, replaced := range []bool{false, true} {
			graph := parseIR(t, resourceOracleSource(next, replaced), pdg.NewBuilder())
			query, err := resourceflow.New(&graph)
			if err != nil {
				t.Fatal(err)
			}
			policy := resourceflow.Policy{Resource: graph.Function.Inputs[0].Symbol}
			got := query.FromEntry(policy)
			want := !replaced && resourcePathOracle(next, 0, [2]bool{})
			if got != want {
				t.Fatalf("next=%v replaced=%v got=%v want=%v", next, replaced, got, want)
			}
			checked++
		}
	}
	t.Logf("checked %d typed-source graph/write layouts against independent path traversal", checked)
}
