package bindingflow

import (
	"cmp"
	"github.com/rstyled/gstyle/domain/pdg"
	"slices"
)

// ValueSource selects a terminal producer and, for a tuple, its result slot.
// Zero is a compiler zero initialization of Symbol, not a fabricated nil value.
type ValueSource struct {
	Node, Symbol   pdg.Ref
	Result         pdg.Position
	Zero           bool
	Representation Representation
}

type Origins struct {
	Sources []ValueSource
	Known   bool
}

type originVertex struct {
	ref            pdg.Ref
	result         pdg.Position
	definition     bool
	representation Representation
}

type originStep struct {
	inputs []originVertex
	source ValueSource
	known  bool
}

type originGraph struct {
	query *Query
	facts Facts
	steps map[originVertex]originStep
}

func (graph *originGraph) results(productive map[originVertex]bool) Origins {
	result := Origins{Known: true}
	sources := map[ValueSource]bool{}
	for vertex, step := range graph.steps {
		if !step.known || !productive[vertex] {
			result.Known = false
		}
		if step.source.Node != 0 {
			sources[step.source] = true
		}
	}
	for source := range sources {
		result.Sources = append(result.Sources, source)
	}
	slices.SortFunc(result.Sources, sourceOrder)
	return result
}

func (g *originGraph) discover(roots []originVertex) {
	queue := slices.Clone(roots)
	for cursor := 0; cursor < len(queue); cursor++ {
		vertex := queue[cursor]
		if _, seen := g.steps[vertex]; seen {
			continue
		}
		step := g.step(vertex)
		g.steps[vertex] = step
		queue = append(queue, step.inputs...)
	}
}

func (g *originGraph) predecessors() (map[originVertex][]originVertex, []originVertex) {
	reverse := map[originVertex][]originVertex{}
	var queue []originVertex
	for vertex, step := range g.steps {
		if step.source.Node != 0 {
			queue = append(queue, vertex)
		}
		for _, input := range step.inputs {
			reverse[input] = append(reverse[input], vertex)
		}
	}
	return reverse, queue
}

func (g *originGraph) productive() map[originVertex]bool {
	reverse, queue := g.predecessors()
	productive := map[originVertex]bool{}
	for _, vertex := range queue {
		productive[vertex] = true
	}
	for cursor := 0; cursor < len(queue); cursor++ {
		for _, parent := range reverse[queue[cursor]] {
			if productive[parent] {
				continue
			}
			productive[parent] = true
			queue = append(queue, parent)
		}
	}
	return productive
}

// OriginsAt follows actual value inputs, strong copies and reaching definitions.
// Productive definition cycles retain their terminal producers. Missing sources,
// weak writes and cycles without a producer stay unknown.
func (q *Query) OriginsAt(owner pdg.Ref, position pdg.Position, facts Facts) Origins {
	roots, known := q.originInputs(owner, position)
	if !known {
		return Origins{}
	}
	return q.originClosure(roots, facts)
}

func (q *Query) OriginsOf(ref pdg.Ref, facts Facts) Origins {
	if !q.validRef(ref) {
		return Origins{}
	}
	return q.originClosure([]originVertex{{ref: ref}}, facts)
}

// SymbolOrigins is a conservative inventory for an addressed/captured cell.
// It does not assert that every definition reaches the capture operation.
func (q *Query) SymbolOrigins(symbol pdg.Ref, facts Facts) Origins {
	var roots []originVertex
	for i, definition := range q.graph.Definitions {
		if definition.Symbol == symbol {
			roots = append(roots, originVertex{ref: pdg.Ref(i + 1), definition: true})
		}
	}
	return q.originClosure(roots, facts)
}

func (q *Query) originInputs(owner pdg.Ref, position pdg.Position) ([]originVertex, bool) {
	if !q.validRef(owner) || position == 0 {
		return nil, false
	}
	representation := q.inputRepresentation(owner, position)
	if tuple := q.tupleProducer(owner); tuple != 0 {
		return []originVertex{{ref: tuple, result: position, representation: representation}}, true
	}
	if source := q.Operand(owner, position); source != 0 {
		return []originVertex{{ref: source, representation: representation}}, true
	}
	symbol := q.ReadSymbol(owner, position)
	if symbol == 0 {
		return nil, false
	}
	return representOrigins(q.readOrigins(owner, symbol, position), representation), true
}

func (q *Query) tupleProducer(owner pdg.Ref) pdg.Ref {
	source := q.Operand(owner, 1)
	if source == 0 || len(q.operands[owner-1]) != 1 {
		return 0
	}
	if !q.tupleValue(source) {
		return 0
	}
	return source
}

func (q *Query) tupleValue(ref pdg.Ref) bool {
	for _, extension := range q.graph.NodeAttributes(q.graph.Nodes[ref-1]).Extensions {
		if q.graph.Tables.Text(extension.Name) == "gstyle.memory:kind" && q.graph.Tables.Text(extension.Value) == `"tuple"` {
			return true
		}
	}
	return false
}

func (q *Query) readOrigins(ref, symbol pdg.Ref, position pdg.Position) []originVertex {
	var inputs []originVertex
	for _, edge := range q.reaching[ref-1] {
		if edge.Symbol == symbol && edge.Location == 0 && edge.Position == position && edge.Definition != 0 {
			inputs = append(inputs, originVertex{ref: edge.Definition, definition: true})
		}
	}
	return inputs
}

func (q *Query) originClosure(roots []originVertex, facts Facts) Origins {
	if len(roots) == 0 {
		return Origins{}
	}
	graph := originGraph{query: q, facts: facts, steps: map[originVertex]originStep{}}
	graph.discover(roots)
	productive := graph.productive()
	return graph.results(productive)
}

func sourceOrder(a, b ValueSource) int {
	return cmp.Or(cmp.Compare(a.Node, b.Node), cmp.Compare(a.Symbol, b.Symbol), cmp.Compare(a.Result, b.Result), cmp.Compare(a.Representation, b.Representation), cmp.Compare(sourceZeroCode(a.Zero), sourceZeroCode(b.Zero)))
}

func sourceZeroCode(zero bool) int {
	if zero {
		return 1
	}
	return 0
}
