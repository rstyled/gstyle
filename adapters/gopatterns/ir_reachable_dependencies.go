package gopatterns

import (
	"github.com/rstyled/gstyle/domain/pdg"
	"slices"
)

// Bodies outside the semantic-control profile can retain initial statement
// anchors. An ordinary-flow proof of unreachability also invalidates those
// anchors; operations absent from this flow (such as unwind invocations) stay.
func pruneUnreachableDependencies(graph *pdg.Graph) {
	dead := map[pdg.Ref]bool{}
	for _, block := range graph.ValueFlow.Blocks {
		for _, ref := range block.Operations {
			dead[ref] = true
		}
	}
	for _, block := range graph.ValueFlow.Blocks {
		if block.Reachable {
			for _, ref := range block.Operations {
				delete(dead, ref)
			}
		}
	}
	graph.Edges = slices.DeleteFunc(graph.Edges, func(edge pdg.Edge) bool {
		return unreachableDependency(edge, dead)
	})
}

func unreachableDependency(edge pdg.Edge, dead map[pdg.Ref]bool) bool {
	if edge.Kind != pdg.Execution && edge.Kind != pdg.ControlEdge {
		return false
	}
	return dead[edge.Source] || dead[edge.Target]
}
