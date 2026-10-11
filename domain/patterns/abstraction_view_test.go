package patterns

import (
	"reflect"
	"slices"
	"testing"
)

func TestExecutionConstraintsDoNotCreateValueSlicesOrCycles(t *testing.T) {
	plain := &MiningGraph{
		Nodes: []PdgNode{{Kind: Param}, {Kind: Op}, {Kind: Return}, {Kind: Return}},
		Edges: []PdgEdge{{From: 0, To: 1, Kind: Data}, {From: 1, To: 2, Kind: Data}, {From: 0, To: 2, Kind: Ctrl}},
	}
	ordered := clonePdg(plain)
	ordered.Edges = append(ordered.Edges, PdgEdge{From: 2, To: 0, Kind: Exec}, PdgEdge{From: 2, To: 3, Kind: Exec})
	if got := BarrierSlice(ordered, 3, nil); !slices.Equal(got, []int{3}) {
		t.Fatalf("execution created a backward value slice: %v", got)
	}
	if got := Chop(ordered, 0, 3); len(got) != 0 {
		t.Fatalf("execution created a value chop: %v", got)
	}
	if got := stronglyConnected(ordered); len(got) != 0 {
		t.Fatalf("execution created a value/control cycle: %v", got)
	}
	if got, want := paramToReturnChop(ordered), paramToReturnChop(plain); !reflect.DeepEqual(got, want) {
		t.Fatalf("report chop includes execution constraints: got %v, want %v", got, want)
	}
	blocked := make([]bool, len(plain.Nodes))
	if got := newReturnTraversal(ordered, false).size(3, blocked); got != 1 {
		t.Fatalf("report traversal includes execution predecessors: %d", got)
	}
	if got := Canonicalize(ordered, Rules{}); !reflect.DeepEqual(got, plain) {
		t.Fatalf("abstraction normalization includes execution: %+v", got)
	}
	if !reflect.DeepEqual(NewWl(ordered).rounds, NewWl(plain).rounds) {
		t.Fatal("abstraction WL uses execution constraints")
	}
	if reflect.DeepEqual(NewWlLight(ordered, nil).hists, NewWlLight(plain, nil).hists) {
		t.Fatal("clone WL skipped execution constraints")
	}
	if len(ordered.Edges) != 5 {
		t.Fatal("projection mutates input graph")
	}
}
