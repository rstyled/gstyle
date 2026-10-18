package gopatterns

import (
	"github.com/rstyled/gstyle/domain/pdg"
	"reflect"
	"testing"
)

func TestUnreachableDependencyPruningRespectsFlowInventory(t *testing.T) {
	graph := pdg.Graph{ValueFlow: &pdg.NormalFlow{Blocks: []pdg.FlowBlock{
		{Reachable: false, Operations: []pdg.Ref{1, 2}},
		{Reachable: true, Operations: []pdg.Ref{2, 3}},
	}}}
	for _, kind := range []pdg.EdgeKind{pdg.Execution, pdg.ControlEdge, pdg.Data} {
		for _, pair := range [][2]pdg.Ref{{1, 3}, {3, 1}, {2, 3}, {3, 4}} {
			graph.Edges = append(graph.Edges, pdg.Edge{Source: pair[0], Target: pair[1], Kind: kind})
		}
	}
	var want []pdg.Edge
	for _, edge := range graph.Edges {
		if edge.Kind == pdg.Data || edge.Source != 1 && edge.Target != 1 {
			want = append(want, edge)
		}
	}
	pruneUnreachableDependencies(&graph)
	if !reflect.DeepEqual(graph.Edges, want) {
		t.Fatalf("edges=%+v want %+v", graph.Edges, want)
	}
}
