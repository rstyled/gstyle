package patterns

import "github.com/rstyled/gstyle/domain/pdg"

// This bounds optional scratch memory, not analysis coverage. Larger graphs use
// the exact traversal instead. Closures disappear after one function's view.
const projectionCacheWords = 8 << 20 // 64 MiB

type projectionClosure struct {
	words    int
	sets     []uint64
	resolved []bool
}

func newProjectionClosure(p *matchingProjection, kind pdg.EdgeKind) *projectionClosure {
	words := (len(p.view.Nodes) + 63) / 64
	if words == 0 || len(p.indexes) > projectionCacheWords/words {
		return nil
	}
	closure := &projectionClosure{words: words, sets: make([]uint64, len(p.indexes)*words), resolved: make([]bool, len(p.indexes))}
	builder := closureBuilder{projection: p, closure: closure, kind: kind, pending: make([]int, len(p.indexes)), outgoing: make([][]pdg.Ref, len(p.indexes))}
	builder.index()
	builder.solve()
	return closure
}

func (c *projectionClosure) set(ref pdg.Ref) []uint64 {
	start := (int(ref) - 1) * c.words
	return c.sets[start : start+c.words]
}

func unionProjectionBits(target, source []uint64) {
	for i, word := range source {
		target[i] |= word
	}
}

// Only hidden vertices participate in topological propagation. Visible vertices
// are terminal producers. Cycles and their dependents stay unresolved, so they
// cannot publish an incomplete cached set.
type closureBuilder struct {
	projection *matchingProjection
	closure    *projectionClosure
	kind       pdg.EdgeKind
	pending    []int
	outgoing   [][]pdg.Ref
	queue      []pdg.Ref
}

func (b *closureBuilder) index() {
	for i, visible := range b.projection.indexes {
		if visible >= 0 {
			continue
		}
		for _, source := range b.projection.incoming[i] {
			if source.kind == b.kind {
				b.link(pdg.Ref(i+1), source.ref)
			}
		}
	}
}

func (b *closureBuilder) link(target, source pdg.Ref) {
	if visible := b.projection.indexes[source-1]; visible >= 0 {
		b.closure.set(target)[visible/64] |= uint64(1) << uint(visible%64)
		return
	}
	b.pending[target-1]++
	b.outgoing[source-1] = append(b.outgoing[source-1], target)
}

func (b *closureBuilder) seed() {
	for i, pending := range b.pending {
		if pending == 0 && b.projection.indexes[i] < 0 {
			b.queue = append(b.queue, pdg.Ref(i+1))
		}
	}
}

func (b *closureBuilder) solve() {
	b.seed()
	for cursor := 0; cursor < len(b.queue); cursor++ {
		ref := b.queue[cursor]
		b.closure.resolved[ref-1] = true
		for _, target := range b.outgoing[ref-1] {
			unionProjectionBits(b.closure.set(target), b.closure.set(ref))
			b.pending[target-1]--
			if b.pending[target-1] == 0 {
				b.queue = append(b.queue, target)
			}
		}
	}
}
