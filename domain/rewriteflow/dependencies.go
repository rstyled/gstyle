// Package rewriteflow checks canonical dependencies under an established source
// operation correspondence. It does not infer correspondence or prove arbitrary
// program equivalence, call effects, frame independence or rewrite correctness.
package rewriteflow

import (
	"fmt"
	"maps"
	"slices"

	"github.com/rstyled/gstyle/domain/pdg"
)

type CallSubstitution struct{ Left, Right pdg.Ref }

type dependency struct {
	source, target, symbol, definition, location pdg.Ref
	subkind, outcome                             string
	position                                     pdg.Position
	kind                                         pdg.EdgeKind
	loop                                         uint8
}

// SameDependencies requires ordinal operation/binding correspondence supplied by
// the caller's complete source match. Only declared call targets may differ.
// Graph IDs, source spans, display descriptions and edge order are not semantics.
func SameDependencies(a, b *pdg.Graph, substitutions []CallSubstitution) (bool, error) {
	if a == nil || b == nil {
		return false, nil
	}
	if err := validateGraphs(a, b); err != nil {
		return false, err
	}
	allowed, known := callSubstitutions(a, b, substitutions)
	if !known {
		return false, nil
	}
	return sameRelations(a, b, allowed), nil
}

func validateGraphs(a, b *pdg.Graph) error {
	if err := pdg.Validate(a); err != nil {
		return fmt.Errorf("left rewrite graph: %w", err)
	}
	if err := pdg.Validate(b); err != nil {
		return fmt.Errorf("right rewrite graph: %w", err)
	}
	return nil
}

func sameRelations(a, b *pdg.Graph, allowed map[pdg.Ref]bool) bool {
	return sameContexts(a, b) && sameOperations(a, b, allowed) && sameStorage(a, b) && maps.Equal(dependencies(a), dependencies(b))
}

func callSubstitutions(a, b *pdg.Graph, substitutions []CallSubstitution) (map[pdg.Ref]bool, bool) {
	allowed := map[pdg.Ref]bool{}
	for _, pair := range substitutions {
		if pair.Left != pair.Right || !directCall(a, pair.Left) || !directCall(b, pair.Right) {
			return nil, false
		}
		if allowed[pair.Left] {
			return nil, false
		}
		allowed[pair.Left] = true
	}
	return allowed, true
}

func directCall(graph *pdg.Graph, ref pdg.Ref) bool {
	if ref == 0 || int(ref) > len(graph.Nodes) {
		return false
	}
	node := graph.Nodes[ref-1]
	attrs := graph.NodeAttributes(node)
	return node.Category == pdg.Call && attrs.CallForm == pdg.DirectCall && attrs.Callee.Status == pdg.Known
}

func sameContexts(a, b *pdg.Graph) bool {
	if pdg.BodyFlow(a) == nil || pdg.BodyFlow(b) == nil {
		return false
	}
	return sameFlow(a.NormalFlow, b.NormalFlow) && sameFlow(a.ValueFlow, b.ValueFlow) && sameExtensions(a, a.Extensions, b, b.Extensions)
}

func sameFlow(a, b *pdg.NormalFlow) bool {
	if a == nil {
		return b == nil
	}
	if b == nil {
		return false
	}
	return sameFlowHeader(a, b) && sameBlocks(a.Blocks, b.Blocks)
}

func sameFlowHeader(a, b *pdg.NormalFlow) bool {
	return a.Entry == b.Entry && a.ReadBindings == b.ReadBindings && len(a.Blocks) == len(b.Blocks)
}

func sameBlocks(a, b []pdg.FlowBlock) bool {
	for i, block := range a {
		if !sameBlock(block, b[i]) {
			return false
		}
	}
	return true
}

func sameBlock(a, b pdg.FlowBlock) bool {
	return a.Condition == b.Condition && a.Reachable == b.Reachable && slices.Equal(a.Operations, b.Operations) && slices.Equal(a.Successors, b.Successors)
}

func dependencies(graph *pdg.Graph) map[dependency]bool {
	result := make(map[dependency]bool, len(graph.Edges))
	for _, edge := range graph.Edges {
		result[dependency{edge.Source, edge.Target, edge.Symbol, edge.Definition, edge.Location, graph.Tables.Text(edge.Subkind), graph.Tables.Text(edge.Outcome), edge.Position, edge.Kind, edge.LoopCarried}] = true
	}
	return result
}
