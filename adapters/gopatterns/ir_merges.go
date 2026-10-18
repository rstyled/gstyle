package gopatterns

import (
	"math/bits"
	"slices"
	"sort"

	"github.com/rstyled/gstyle/domain/pdg"
)

type definitionJoin struct {
	binding    pdg.Ref
	inputs     []pdg.Ref
	definition pdg.Ref
}

// A join is a value entity at the read site, not an executable CFG operation.
func (a *reachingAnalysis) joinedDefinition(binding, target pdg.Ref, definitions []pdg.Ref) pdg.Ref {
	if len(definitions) == 1 {
		return definitions[0]
	}
	if a.builder.generalBindings {
		return a.sharedDefinition(binding, definitions)
	}
	return a.definitionJoin(a.builder.graph.Nodes[target-1].Span, definitions, "Scalar value join at a reachable read site.")
}

func (a *reachingAnalysis) definitionJoin(span pdg.Ref, inputs []pdg.Ref, description string) pdg.Ref {
	b := a.builder
	first := b.graph.Definitions[inputs[0]-1]
	var symbols []pdg.Ref
	if first.Symbol != 0 {
		symbols = []pdg.Ref{first.Symbol}
	}
	node := b.scalarEntity(span, pdg.Merge, pdg.Attributes{Symbols: symbols, Description: b.pool.Text(description)})
	b.storageDefinition(node, first.Symbol, first.Location)
	ref := pdg.Ref(len(b.graph.Definitions))
	for _, input := range inputs {
		a.edge(input, node, 0)
	}
	return ref
}

// Shared inputs retain their first definition's source span. Read operations
// keep their own spans and link directly to the shared value entity.
func (a *reachingAnalysis) sharedDefinition(binding pdg.Ref, inputs []pdg.Ref) pdg.Ref {
	hash := definitionSetHash(binding, inputs)
	if ref := a.findDefinitionJoin(hash, binding, inputs); ref != 0 {
		return ref
	}
	ref := a.definitionTree(inputs)
	if a.joins == nil {
		a.joins = map[uint64][]definitionJoin{}
	}
	a.joins[hash] = append(a.joins[hash], definitionJoin{binding: binding, inputs: inputs, definition: ref})
	return ref
}

func (a *reachingAnalysis) findDefinitionJoin(hash uint64, binding pdg.Ref, inputs []pdg.Ref) pdg.Ref {
	for _, join := range a.joins[hash] {
		if join.binding == binding && slices.Equal(join.inputs, inputs) {
			return join.definition
		}
	}
	return 0
}

// Hash collisions only select a bucket; equality above checks every input.
func definitionSetHash(binding pdg.Ref, inputs []pdg.Ref) uint64 {
	hash := uint64(binding)
	for _, input := range inputs {
		hash = (hash ^ uint64(input)) * 1099511628211
	}
	return hash
}

// Split on definition identity, rather than slice length, so overlapping sets
// reuse the same subtrees even when other inputs are added or removed.
func (a *reachingAnalysis) definitionTree(inputs []pdg.Ref) pdg.Ref {
	if len(inputs) == 1 {
		return inputs[0]
	}
	mask := uint32(1) << uint(bits.Len32(uint32(inputs[0]^inputs[len(inputs)-1]))-1)
	split := sort.Search(len(inputs), func(i int) bool { return uint32(inputs[i])&mask != 0 })
	pair := [2]pdg.Ref{a.definitionTree(inputs[:split]), a.definitionTree(inputs[split:])}
	if ref := a.joinPairs[pair]; ref != 0 {
		return ref
	}
	first := a.builder.graph.Definitions[inputs[0]-1]
	span := a.builder.graph.Nodes[first.Node-1].Span
	ref := a.definitionJoin(span, pair[:], "Shared immutable reaching-definition set.")
	if a.joinPairs == nil {
		a.joinPairs = map[[2]pdg.Ref]pdg.Ref{}
	}
	a.joinPairs[pair] = ref
	return ref
}
