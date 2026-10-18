package bindingflow

import "github.com/rstyled/gstyle/domain/pdg"

func (g *originGraph) step(vertex originVertex) originStep {
	var step originStep
	if vertex.definition {
		step = g.definition(vertex.ref)
	} else {
		step = g.value(vertex)
	}
	return originRepresentation(step, vertex)
}

func (g *originGraph) value(vertex originVertex) originStep {
	if !g.query.validRef(vertex.ref) {
		return originStep{}
	}
	node := g.query.graph.Nodes[vertex.ref-1]
	attrs := g.query.graph.NodeAttributes(node)
	if next, wrapped := g.query.identityOperand(vertex.ref, g.facts); wrapped {
		return oneOrigin(next, vertex.result)
	}
	if node.Category == pdg.Read {
		return g.read(vertex.ref, attrs)
	}
	return originStep{known: true, source: ValueSource{Node: vertex.ref, Result: vertex.result}}
}

func (g *originGraph) read(ref pdg.Ref, attrs pdg.Attributes) originStep {
	if len(attrs.Reads) != 1 || !directBinding(attrs.Reads[0]) {
		return originStep{}
	}
	read := attrs.Reads[0]
	inputs := g.query.readOrigins(ref, read.Symbol, read.Position)
	return originStep{inputs: inputs, known: len(inputs) != 0}
}

func (g *originGraph) definition(ref pdg.Ref) originStep {
	q := g.query
	definition := q.graph.Definitions[ref-1]
	owner := q.graph.Nodes[definition.Node-1]
	switch owner.Category {
	case pdg.Merge:
		return g.merge(definition)
	case pdg.FormalInput:
		return originStep{known: true, source: ValueSource{Node: definition.Node, Symbol: definition.Symbol}}
	case pdg.FormalOutput:
		return originStep{known: true, source: ValueSource{Node: definition.Node, Symbol: definition.Symbol, Zero: true}}
	case pdg.Assignment, pdg.Declaration:
		return g.copy(definition)
	default:
		return originStep{}
	}
}

func (g *originGraph) merge(definition pdg.Definition) originStep {
	inputs := g.query.readOrigins(definition.Node, definition.Symbol, 0)
	return originStep{inputs: inputs, known: len(inputs) != 0}
}

func (g *originGraph) copy(definition pdg.Definition) originStep {
	q := g.query
	owner := q.graph.Nodes[definition.Node-1]
	attrs := q.graph.NodeAttributes(owner)
	if !copyOperation(q.graph, owner, attrs) {
		return originStep{}
	}
	if definition.Symbol == 0 || !q.strongWrite(attrs, definition.Symbol) {
		return originStep{}
	}
	write, found := uniqueOriginWrite(q.Writes(definition.Node), definition.Symbol)
	if !found {
		return originStep{}
	}
	if write.Zero {
		return originStep{known: true, source: ValueSource{Node: definition.Node, Symbol: definition.Symbol, Zero: true}}
	}
	inputs, known := q.originInputs(definition.Node, write.Position)
	return originStep{inputs: inputs, known: known}
}

func oneOrigin(ref pdg.Ref, result pdg.Position) originStep {
	if ref == 0 {
		return originStep{}
	}
	return originStep{known: true, inputs: []originVertex{{ref: ref, result: result}}}
}

func uniqueOriginWrite(writes []Write, symbol pdg.Ref) (Write, bool) {
	var selected Write
	found := false
	for _, write := range writes {
		if write.Symbol != symbol {
			continue
		}
		if found || write.Location != 0 || write.Unresolved {
			return Write{}, false
		}
		selected, found = write, true
	}
	return selected, found
}
