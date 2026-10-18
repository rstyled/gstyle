package callflow

import (
	"github.com/rstyled/gstyle/domain/bindingflow"
	"github.com/rstyled/gstyle/domain/pdg"
	"testing"
)

func TestNilInputFaultAlternatives(t *testing.T) {
	for bits := 0; bits < 4; bits++ {
		effect := operationEffect{NilInput: true, Panic: bits&1 != 0, Block: bits&2 != 0}
		if got := nilInputOutcomes(effect, bindingflow.Value{Nil: true}); got != Panics {
			t.Fatalf("nil effect %+v=%v", effect, got)
		}
		want := Returns
		if bits&1 != 0 {
			want |= Panics
		}
		if bits&2 != 0 {
			want |= Diverges
		}
		if got := nilInputOutcomes(effect, bindingflow.Value{NonNil: true}); got != want {
			t.Fatalf("non-nil effect %+v=%v want %v", effect, got, want)
		}
		if got := nilInputOutcomes(effect, bindingflow.Value{}); got != want|Panics {
			t.Fatalf("unknown effect %+v=%v", effect, got)
		}
	}
}

func TestNilInputNeedsAdmittedStableRegion(t *testing.T) {
	graph := &pdg.Graph{Nodes: []pdg.Node{{Category: pdg.Computation}, {Category: pdg.Call}}}
	for _, tc := range []struct {
		name       string
		region     uint32
		members    []pdg.Ref
		incomplete bool
	}{
		{"missing region", 0, []pdg.Ref{1}, false},
		{"missing member", 1, nil, false},
		{"incomplete inventory", 1, []pdg.Ref{1}, true},
		{"sibling invocation", 1, []pdg.Ref{1, 2}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			operations := &OperationEffects{graph: graph, regions: map[pdg.Ref]uint32{1: tc.region}, members: map[uint32][]pdg.Ref{tc.region: tc.members}, incompleteRegions: tc.incomplete, effects: map[pdg.Ref]operationEffect{1: {NilInput: true}}}
			// No operand query exists. Incomplete evidence must return the original
			// may-outcomes without evaluating or inventing an operand value.
			if got := operations.outcomesWithValues(1, bindingflow.Evaluation{}); got != Returns|Panics {
				t.Fatalf("outcomes=%v", got)
			}
		})
	}
}
