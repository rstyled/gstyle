package patterns

import (
	"reflect"
	"testing"
)

func TestSemanticControlIsNotFlattenedForNormalization(t *testing.T) {
	for _, edges := range [][]PdgEdge{
		{{From: 0, To: 2, Kind: Ctrl}, {From: 1, To: 2, Kind: Ctrl}},
		{{From: 0, To: 0, Kind: Ctrl}},
		{{From: 0, To: 1, Kind: Ctrl}, {From: 1, To: 0, Kind: Ctrl}},
	} {
		graph := &MiningGraph{Nodes: []PdgNode{{Kind: Branch}, {Kind: Branch}, {Kind: Return}}, Edges: edges}
		if hierarchicalControl(graph) {
			t.Fatal("nonhierarchical semantic control treated as source containment")
		}
		if got := Canonicalize(graph, RulesAll); !reflect.DeepEqual(got.Edges, edges) {
			t.Fatalf("dependencies changed: %v -> %v", edges, got.Edges)
		}
	}
}

func TestUniqueControllerDoesNotDependOnEdgeOrder(t *testing.T) {
	for _, edges := range [][]PdgEdge{
		{{From: 0, To: 2, Kind: Ctrl}, {From: 1, To: 2, Kind: Ctrl}},
		{{From: 1, To: 2, Kind: Ctrl}, {From: 0, To: 2, Kind: Ctrl}},
	} {
		graph := &MiningGraph{Nodes: make([]PdgNode, 3), Edges: edges}
		if _, ok := ctrlParent(graph, 2); ok {
			t.Fatal("arbitrary controller selected")
		}
	}
}

func TestEnclosingLoopHasNoArbitraryDepthCutoff(t *testing.T) {
	graph := &MiningGraph{Nodes: make([]PdgNode, 9)}
	graph.Nodes[0].Kind = Loop
	for index := 1; index < len(graph.Nodes); index++ {
		graph.Edges = append(graph.Edges, PdgEdge{From: index - 1, To: index, Kind: Ctrl})
	}
	if loop, ok := enclosingLoop(graph, 8); !ok || loop != 0 {
		t.Fatalf("loop=%d,%v", loop, ok)
	}
}
