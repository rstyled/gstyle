package gopatterns

import "github.com/rstyled/gstyle/domain/pdg"

// A join is a value entity at the read site, not an executable CFG operation.
// It is created on demand only when several definitions reach that binding.
func (a *reachingAnalysis) joinedDefinition(symbol, target pdg.Ref, definitions []pdg.Ref) pdg.Ref {
	if len(definitions) == 1 {
		return definitions[0]
	}
	b := a.builder
	definition := b.graph.Definitions[definitions[0]-1]
	var symbols []pdg.Ref
	if definition.Symbol != 0 {
		symbols = []pdg.Ref{definition.Symbol}
	}
	node := b.scalarEntity(b.graph.Nodes[target-1].Span, pdg.Merge, pdg.Attributes{Symbols: symbols, Description: b.pool.Text("Scalar value join at a reachable read site.")})
	b.storageDefinition(node, definition.Symbol, definition.Location)
	for _, definition := range definitions {
		a.edge(definition, node, 0)
	}
	return pdg.Ref(len(b.graph.Definitions))
}
