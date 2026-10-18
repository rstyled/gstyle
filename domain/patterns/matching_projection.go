package patterns

import (
	"math/bits"

	"github.com/rstyled/gstyle/domain/pdg"
)

type projectionSource struct {
	ref  pdg.Ref
	kind pdg.EdgeKind
}

type projectionTarget struct {
	ref      pdg.Ref
	kind     pdg.EdgeKind
	position pdg.Position
}

type projectionGroup struct {
	edge    pdg.Edge
	sources []pdg.Ref
}

type matchingProjection struct {
	view     *MiningGraph
	indexes  []int
	incoming [][]projectionSource
	kept     []bool
	visited  []uint64
	epoch    uint64
	queue    []pdg.Ref
	groups   []projectionGroup
	targets  map[projectionTarget]int
	closures map[pdg.EdgeKind]*projectionClosure
	reached  []uint64
}

// Matching derives dependencies through omitted canonical reads and joins.
func newMatchingProjection(graph *pdg.Graph, view *MiningGraph, indexes []int, kept []bool) *matchingProjection {
	p := &matchingProjection{view: view, indexes: indexes, incoming: make([][]projectionSource, len(graph.Nodes)), kept: kept, visited: make([]uint64, len(graph.Nodes)), targets: map[projectionTarget]int{}, closures: map[pdg.EdgeKind]*projectionClosure{}, reached: make([]uint64, (len(view.Nodes)+63)/64)}
	for _, edge := range graph.Edges {
		if !normalEdgeRemoved(kept, edge) {
			p.incoming[edge.Target-1] = append(p.incoming[edge.Target-1], projectionSource{edge.Source, edge.Kind})
		}
	}
	return p
}

func (p *matchingProjection) edge(edge pdg.Edge) {
	if p.indexes[edge.Target-1] < 0 || normalEdgeRemoved(p.kept, edge) {
		return
	}
	if edge.Kind == pdg.ControlEdge && p.indexes[edge.Source-1] < 0 {
		return
	}
	key := projectionTarget{edge.Target, edge.Kind, edge.Position}
	index, found := p.targets[key]
	if !found {
		index = len(p.groups)
		p.targets[key] = index
		p.groups = append(p.groups, projectionGroup{edge: edge})
	}
	p.groups[index].sources = append(p.groups[index].sources, edge.Source)
}

// Roots for one consumer slot share their visited set. Overlapping dependency
// paths are walked once, and each visible producer is emitted once per slot.
func (p *matchingProjection) project() {
	for _, group := range p.groups {
		p.startTraversal()
		for _, source := range group.sources {
			edge := group.edge
			edge.Source = source
			p.walk(edge)
		}
		p.emit(group.edge)
	}
}

func (p *matchingProjection) walk(edge pdg.Edge) {
	var closure *projectionClosure
	if p.indexes[edge.Source-1] < 0 {
		closure = p.closure(edge.Kind)
	}
	p.queue = append(p.queue[:0], edge.Source)
	for cursor := 0; cursor < len(p.queue); cursor++ {
		ref := p.queue[cursor]
		if !p.visit(ref, closure) {
			continue
		}
		for _, incoming := range p.incoming[ref-1] {
			if incoming.kind == edge.Kind {
				p.queue = append(p.queue, incoming.ref)
			}
		}
	}
}

func (p *matchingProjection) visit(ref pdg.Ref, closure *projectionClosure) bool {
	if p.visited[ref-1] == p.epoch {
		return false
	}
	p.visited[ref-1] = p.epoch
	if visible := p.indexes[ref-1]; visible >= 0 {
		p.reached[visible/64] |= uint64(1) << uint(visible%64)
		return false
	}
	if closure != nil && closure.resolved[ref-1] {
		unionProjectionBits(p.reached, closure.set(ref))
		return false
	}
	return true
}

func (p *matchingProjection) closure(kind pdg.EdgeKind) *projectionClosure {
	if closure, found := p.closures[kind]; found {
		return closure
	}
	closure := newProjectionClosure(p, kind)
	p.closures[kind] = closure
	return closure
}

func (p *matchingProjection) emit(edge pdg.Edge) {
	projected := semanticEdge(edge)
	projected.To = p.indexes[edge.Target-1]
	for word, value := range p.reached {
		for value != 0 {
			projected.From = word*64 + bits.TrailingZeros64(value)
			p.view.Edges = append(p.view.Edges, projected)
			value &= value - 1
		}
	}
}

func (p *matchingProjection) startTraversal() {
	clear(p.reached)
	p.epoch++
	if p.epoch == 0 {
		clear(p.visited)
		p.epoch++
	}
}
