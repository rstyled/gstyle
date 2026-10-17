package patterns

import "github.com/rstyled/gstyle/domain/pdg"

type matchingProjection struct {
	graph    *pdg.Graph
	view     *MiningGraph
	indexes  []int
	incoming [][]pdg.Edge
	kept     []bool
}

// Matching keeps its source abstraction labels, but derives dependencies through
// omitted canonical reads and joins. It never restores first-binding guesses.
func newMatchingProjection(graph *pdg.Graph, view *MiningGraph, indexes []int, kept []bool) *matchingProjection {
	p := &matchingProjection{graph: graph, view: view, indexes: indexes, incoming: make([][]pdg.Edge, len(graph.Nodes)), kept: kept}
	for _, edge := range graph.Edges {
		if !normalEdgeRemoved(kept, edge) {
			p.incoming[edge.Target-1] = append(p.incoming[edge.Target-1], edge)
		}
	}
	return p
}
func (p *matchingProjection) edge(edge pdg.Edge) {
	if p.indexes[edge.Target-1] < 0 || normalEdgeRemoved(p.kept, edge) {
		return
	}
	if p.indexes[edge.Source-1] >= 0 {
		appendMiningEdge(p.graph, p.view, p.indexes, edge)
		return
	}
	if edge.Kind == pdg.ControlEdge {
		return
	}
	p.transparent(edge)
}
func (p *matchingProjection) transparent(edge pdg.Edge) {
	queue := []pdg.Ref{edge.Source}
	seen := make([]bool, len(p.indexes))
	for cursor := 0; cursor < len(queue); cursor++ {
		ref := queue[cursor]
		if seen[ref-1] {
			continue
		}
		seen[ref-1] = true
		if p.visibleSource(ref, edge) {
			continue
		}
		for _, incoming := range p.incoming[ref-1] {
			if incoming.Kind == edge.Kind {
				queue = append(queue, incoming.Source)
			}
		}
	}
}
func (p *matchingProjection) visibleSource(ref pdg.Ref, edge pdg.Edge) bool {
	if p.indexes[ref-1] < 0 {
		return false
	}
	edge.Source = ref
	appendMiningEdge(p.graph, p.view, p.indexes, edge)
	return true
}
