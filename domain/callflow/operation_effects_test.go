package callflow

import (
	"github.com/rstyled/gstyle/domain/pdg"
	"testing"
)

func TestNonNilRefinementRetainsOtherFaults(t *testing.T) {
	graph := &pdg.Graph{Nodes: []pdg.Node{{Category: pdg.Computation}}}
	for _, tc := range []struct {
		effect operationEffect
		want   Outcomes
	}{
		{operationEffect{NilInput: true}, Returns},
		{operationEffect{NilInput: true, Panic: true}, Returns | Panics},
		{operationEffect{NilInput: true, Block: true}, Returns | Diverges},
		{operationEffect{NilInput: true, Unknown: true}, UnknownExit},
	} {
		operations := &OperationEffects{graph: graph, effects: map[pdg.Ref]operationEffect{1: tc.effect}}
		if got := operations.OutcomesWithNonNilInput(1); got != tc.want {
			t.Fatalf("effect %+v: got %v want %v", tc.effect, got, tc.want)
		}
	}
}
