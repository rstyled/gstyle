package patterns

import (
	"slices"

	"github.com/rstyled/gstyle/domain/pdg"
)

type projectionSource struct {
	ref  pdg.Ref
	kind pdg.EdgeKind
}

type matchingProjection struct {
	graph    *pdg.Graph
	view     *MiningGraph
	indexes  []int
	incoming [][]pdg.Edge
	kept     []bool
	visited  []uint64
	epoch    uint64
	queue    []pdg.Ref
	edges    map[PdgEdge]bool
	sources  map[projectionSource][]pdg.Ref
	found    []pdg.Ref
}

// Matching keeps its source abstraction labels, but derives dependencies through
// omitted canonical reads and joins. It never restores first-binding guesses.
func newMatchingProjection(graph *pdg.Graph, view *MiningGraph, indexes []int, kept []bool) *matchingProjection {
	p := &matchingProjection{graph: graph, view: view, indexes: indexes, incoming: make([][]pdg.Edge, len(graph.Nodes)), kept: kept, visited: make([]uint64, len(graph.Nodes)), edges: map[PdgEdge]bool{}, sources: map[projectionSource][]pdg.Ref{}}
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
		p.append(edge)
		return
	}
	if edge.Kind == pdg.ControlEdge {
		return
	}
	p.transparent(edge)
}
func (p *matchingProjection) transparent(edge pdg.Edge) {
	for _, ref := range p.sourceSet(edge.Source, edge.Kind) {
		edge.Source = ref
		p.append(edge)
	}
}

func (p *matchingProjection) sourceSet(source pdg.Ref, kind pdg.EdgeKind) []pdg.Ref {
	key := projectionSource{source, kind}
	if sources, ok := p.sources[key]; ok {
		return sources
	}
	sources := p.collectSources(source, kind)
	p.sources[key] = sources
	return sources
}

func (p *matchingProjection) collectSources(source pdg.Ref, kind pdg.EdgeKind) []pdg.Ref {
	p.startTraversal(source)
	for cursor := 0; cursor < len(p.queue); cursor++ {
		ref := p.queue[cursor]
		if p.visited[ref-1] == p.epoch {
			continue
		}
		p.visited[ref-1] = p.epoch
		if p.indexes[ref-1] >= 0 {
			p.found = append(p.found, ref)
			continue
		}
		for _, incoming := range p.incoming[ref-1] {
			if incoming.Kind == kind {
				p.queue = append(p.queue, incoming.Source)
			}
		}
	}
	return slices.Clone(p.found)
}

// Different canonical locations can project to the same abstract dependency.
// Operand positions and dependency kinds remain distinct.
func (p *matchingProjection) append(edge pdg.Edge) {
	projected := semanticEdge(edge)
	projected.From = p.indexes[edge.Source-1]
	projected.To = p.indexes[edge.Target-1]
	if p.edges[projected] {
		return
	}
	p.edges[projected] = true
	p.view.Edges = append(p.view.Edges, projected)
}

func (p *matchingProjection) startTraversal(source pdg.Ref) {
	p.epoch++
	if p.epoch == 0 {
		clear(p.visited)
		p.epoch++
	}
	p.queue = append(p.queue[:0], source)
	p.found = p.found[:0]
}
