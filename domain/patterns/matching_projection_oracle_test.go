package patterns

import (
	"maps"
	"testing"

	"github.com/rstyled/gstyle/domain/pdg"
)

// Enumerate every directed four-node graph without self edges. The independent
// oracle eliminates only hidden vertices; visible producers stop traversal.
func TestMatchingProjectionMatchesReachabilityOracle(t *testing.T) {
	pool := pdg.NewBuilder()
	label := pool.MatchLabel(pdg.MatchLabel{Kind: pool.Text(string(Call))})
	for layout := 0; layout < 2; layout++ {
		for mask := 0; mask < 1<<12; mask++ {
			graph := &pdg.Graph{Tables: pool.Tables, Nodes: []pdg.Node{
				{Category: pdg.Call, MatchLabel: label}, {Category: pdg.Read},
				{Category: pdg.Merge}, {Category: pdg.Call, MatchLabel: label},
			}}
			bit := 0
			for from := 0; from < 4; from++ {
				for to := 0; to < 4; to++ {
					if from == to {
						continue
					}
					if mask&(1<<bit) != 0 {
						edge := pdg.Edge{Source: pdg.Ref(from + 1), Target: pdg.Ref(to + 1), Kind: pdg.Data, Position: 1}
						if layout == 1 {
							edge.Kind = pdg.EdgeKind(2 * ((from + to) % 2))
							edge.Position += pdg.Position(from % 2)
						}
						graph.Edges = append(graph.Edges, edge)
					}
					bit++
				}
			}
			want := projectionReachabilityOracle(graph)
			for _, got := range [][]PdgEdge{MiningView(graph).Edges, uncachedProjectionEdges(graph)} {
				missing := maps.Clone(want)
				if len(got) != len(missing) {
					t.Fatalf("layout=%d mask=%d: %v, want %v", layout, mask, got, missing)
				}
				for _, edge := range got {
					if !missing[edge] {
						t.Fatalf("layout=%d mask=%d: unexpected or duplicate edge %v", layout, mask, edge)
					}
					delete(missing, edge)
				}
			}
		}
	}
}

func projectionReachabilityOracle(graph *pdg.Graph) map[PdgEdge]bool {
	visible := map[int]int{0: 0, 3: 1}
	want := map[PdgEdge]bool{}
	for _, kind := range []pdg.EdgeKind{pdg.Data, pdg.Execution} {
		var reach [4][4]bool
		for _, edge := range graph.Edges {
			if edge.Kind == kind {
				reach[edge.Source-1][edge.Target-1] = true
			}
		}
		for _, middle := range []int{1, 2} {
			for from := 0; from < 4; from++ {
				for to := 0; to < 4; to++ {
					reach[from][to] = reach[from][to] || reach[from][middle] && reach[middle][to]
				}
			}
		}
		for _, edge := range graph.Edges {
			target, kept := visible[int(edge.Target)-1]
			if !kept || edge.Kind != kind {
				continue
			}
			for source, index := range visible {
				root := int(edge.Source) - 1
				_, direct := visible[root]
				if source == root || !direct && reach[source][root] {
					dependency := Data
					if kind == pdg.Execution {
						dependency = Exec
					}
					want[PdgEdge{From: index, To: target, Kind: dependency, ArgPos: int(edge.Position) - 1}] = true
				}
			}
		}
	}
	return want
}

func uncachedProjectionEdges(graph *pdg.Graph) []PdgEdge {
	indexes := make([]int, len(graph.Nodes))
	view := &MiningGraph{}
	for i, node := range graph.Nodes {
		indexes[i] = -1
		if node.MatchLabel != 0 {
			indexes[i] = len(view.Nodes)
			view.Nodes = append(view.Nodes, PdgNode{})
		}
	}
	projection := newMatchingProjection(graph, view, indexes, nil)
	projection.closures[pdg.Data] = nil
	projection.closures[pdg.Execution] = nil
	for _, edge := range graph.Edges {
		projection.edge(edge)
	}
	projection.project()
	return view.Edges
}

func TestProjectionClosureBudgetFallsBack(t *testing.T) {
	projection := &matchingProjection{view: &MiningGraph{Nodes: make([]PdgNode, 4096)}, indexes: make([]int, projectionCacheWords/64+1)}
	if newProjectionClosure(projection, pdg.Data) != nil {
		t.Fatal("closure exceeds its scratch budget")
	}
	projection.view.Nodes = nil
	if newProjectionClosure(projection, pdg.Data) != nil {
		t.Fatal("empty view allocates a closure")
	}
}

func TestProjectionProducerSetsCrossWordBoundaries(t *testing.T) {
	pool := pdg.NewBuilder()
	label := pool.MatchLabel(pdg.MatchLabel{Kind: pool.Text(string(Call))})
	graph := &pdg.Graph{Tables: pool.Tables}
	for i := 0; i < 130; i++ {
		graph.Nodes = append(graph.Nodes, pdg.Node{Category: pdg.Call, MatchLabel: label})
		graph.Edges = append(graph.Edges, pdg.Edge{Source: pdg.Ref(i + 1), Target: 131, Kind: pdg.Data})
	}
	graph.Nodes = append(graph.Nodes, pdg.Node{Category: pdg.Merge}, pdg.Node{Category: pdg.Call, MatchLabel: label})
	want := map[PdgEdge]bool{}
	for position := 1; position <= 2; position++ {
		graph.Edges = append(graph.Edges, pdg.Edge{Source: 131, Target: 132, Kind: pdg.Data, Position: pdg.Position(position)})
		for source := 0; source < 130; source++ {
			want[PdgEdge{From: source, To: 130, Kind: Data, ArgPos: position - 1}] = true
		}
	}
	got := MiningView(graph).Edges
	if len(got) != len(want) {
		t.Fatalf("edges=%d; want %d", len(got), len(want))
	}
	for _, edge := range got {
		if !want[edge] {
			t.Fatalf("unexpected or duplicate dependency %v", edge)
		}
		delete(want, edge)
	}
}
