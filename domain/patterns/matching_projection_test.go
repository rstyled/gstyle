package patterns

import (
	"github.com/rstyled/gstyle/domain/pdg"
	"reflect"
	"testing"
)

func TestMatchingProjectionPreservesOperandSlotsAndCycles(t *testing.T) {
	graph := semanticFlowFixture(t)
	graph.Edges = append(graph.Edges,
		pdg.Edge{Source: 3, Target: 2, Kind: pdg.Data},
		pdg.Edge{Source: 3, Target: 4, Kind: pdg.Data, Position: 2})
	want := []PdgEdge{{From: 0, To: 1, Kind: Data, ArgPos: 0}, {From: 0, To: 1, Kind: Data, ArgPos: 1}}
	if got := MiningView(graph).Edges; !reflect.DeepEqual(got, want) {
		t.Fatalf("projected operands = %+v, want %+v", got, want)
	}
}

func TestMatchingProjectionDoesNotComposeDifferentDependencies(t *testing.T) {
	for _, kind := range []pdg.EdgeKind{pdg.Execution, pdg.ControlEdge} {
		t.Run(string(rune('a'+kind)), func(t *testing.T) {
			graph := semanticFlowFixture(t)
			graph.Edges[1].Kind = kind
			if got := MiningView(graph).Edges; len(got) != 0 {
				t.Fatalf("mixed dependency path became a data edge: %+v", got)
			}
		})
	}
}

func TestSemanticTaintUsesSharedOperationReachability(t *testing.T) {
	graph := semanticFlowFixture(t)
	graph.Tables.Strings = append(graph.Tables.Strings, "gstyle.pdg:operation-flow", `"0.1.0"`)
	last := pdg.Text(len(graph.Tables.Strings))
	graph.Extensions = []pdg.Extension{{Name: last - 1, Value: last}}
	graph.ValueFlow = &pdg.NormalFlow{Entry: 1, Blocks: []pdg.FlowBlock{
		{Operations: []pdg.Ref{1}, Reachable: true},
		{Operations: []pdg.Ref{2, 3, 4}},
	}}
	fact := &MiningFacts{ID: "unreachable-sink", Canonical: graph, Pdg: MiningView(graph)}
	if flows := FindTaintFlows([]*MiningFacts{fact}); len(flows) != 0 {
		t.Fatalf("unreachable sink produces taint flow: %+v", flows)
	}
	if got := semanticView(graph).Nodes[3].Kind; got != "unreachable" {
		t.Fatalf("sink kind=%s, want unreachable", got)
	}
}
