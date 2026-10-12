package patterns

import "github.com/rstyled/gstyle/domain/pdg"

// PaperCharacteristics is computed once from canonical source facts. The order
// is declarations, assignments, controls, calls, other nodes, control edges,
// data edges, execution edges, reference bindings. Counts is immutable.
type PaperCharacteristics struct {
	Counts         [9]float64
	ReferenceKnown bool
}

func nativeCharacteristics(g *pdg.Graph) *PaperCharacteristics {
	return nativeCharacteristicsFor(g, pdg.NormalProjection(g))
}

func nativeCharacteristicsFor(g *pdg.Graph, kept []bool) *PaperCharacteristics {
	out := &PaperCharacteristics{ReferenceKnown: g.Function.ReferenceCount.Status == pdg.Known}
	countNativeNodes(g, kept, out)
	countNativeEdges(g, kept, out)
	if out.ReferenceKnown {
		out.Counts[8] = float64(g.Function.ReferenceCount.Value)
	}
	return out
}

func paperCategoryIndex(category pdg.Category) int {
	switch category {
	case pdg.Declaration, pdg.FormalInput:
		return 0
	case pdg.Assignment:
		return 1
	case pdg.Control:
		return 2
	case pdg.Call:
		return 3
	default:
		return 4
	}
}

func paperEdgeIndex(kind pdg.EdgeKind) int {
	switch kind {
	case pdg.ControlEdge:
		return 5
	case pdg.Execution:
		return 7
	default:
		return 6
	}
}

func paperFeatureNode(category pdg.Category) bool {
	switch category {
	case pdg.FormalOutput, pdg.Entry, pdg.Exit:
		return false
	default:
		return true
	}
}

func normalEdgeRemoved(kept []bool, edge pdg.Edge) bool {
	return normalNodeRemoved(kept, int(edge.Source-1)) || normalNodeRemoved(kept, int(edge.Target-1))
}

func nativeFeatureNode(g *pdg.Graph, category pdg.Category) bool {
	if g.NormalFlow != nil {
		return category != pdg.Entry && category != pdg.Exit
	}
	return paperFeatureNode(category)
}

func nativeCategoryIndex(g *pdg.Graph, category pdg.Category) int {
	if g.NormalFlow != nil && category == pdg.FormalInput {
		return 4
	}
	return paperCategoryIndex(category)
}

func countNativeNodes(g *pdg.Graph, kept []bool, out *PaperCharacteristics) {
	for i, node := range g.Nodes {
		if normalNodeRemoved(kept, i) {
			continue
		}
		if !nativeFeatureNode(g, node.Category) {
			continue
		}
		out.Counts[nativeCategoryIndex(g, node.Category)]++
	}
}

func countNativeEdges(g *pdg.Graph, kept []bool, out *PaperCharacteristics) {

	for _, edge := range g.Edges {
		if normalEdgeRemoved(kept, edge) {
			continue
		}
		out.Counts[paperEdgeIndex(edge.Kind)]++
	}
}
