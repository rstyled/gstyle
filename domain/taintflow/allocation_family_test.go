package taintflow

import (
	"github.com/rstyled/gstyle/domain/pdg"
	"testing"
)

// The concrete model selects one cell for the write. A repeated allocation
// represents all cells, so a clean write cannot clear their joined source.
func TestProjectedWritesAgainstConcreteAllocationFamily(t *testing.T) {
	source := atom{Source: Site{Function: "source", Node: 1}}
	for size := 1; size <= 2; size++ {
		for mask := 0; mask < 1<<size; mask++ {
			for _, incoming := range []bool{false, true} {
				want := false
				for selected := 0; selected < size; selected++ {
					concrete := make([]bool, size)
					for i := range concrete {
						concrete[i] = mask&(1<<i) != 0
					}
					concrete[selected] = incoming
					for _, tainted := range concrete {
						want = want || tainted
					}
				}
				root := object{Allocation: "factory cell"}
				state := emptyState()
				state.Repeated[root] = size > 1
				old := emptyValue()
				if mask != 0 {
					old.Data[source] = true
				}
				target := cellKey(root, "value")
				state.Memory[target] = old
				actual := emptyValue()
				actual.Objects[root] = true
				e := evaluation{state: state, arguments: map[pdg.Position]value{1: actual}}
				write := emptyValue()
				if incoming {
					write.Data[source] = true
				}
				projected := map[cell]value{}
				projection := writeProjection{values: projected, weak: map[cell]bool{}}
				e.projectWrite(0, cell{Object: object{Input: 1}, Fields: "value"}, write, projection)
				projection.preserveOriginal(e.state)
				if got := projected[target].Data[source]; got != want {
					t.Fatalf("size=%d initial=%b incoming=%v got=%v want=%v", size, mask, incoming, got, want)
				}
			}
		}
	}
}
