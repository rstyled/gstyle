package gopatterns

import (
	"go/token"

	"github.com/rstyled/gstyle/domain/pdg"
	"github.com/rstyled/gstyle/domain/rewriteflow"
)

func matchingParameterizationDependencies(pair *parameterizationPair, holes []hole, fset *token.FileSet) bool {
	substitutions := make([]rewriteflow.CallSubstitution, len(holes))
	for i, hole := range holes {
		substitutions[i] = rewriteflow.CallSubstitution{Left: parameterizationCallRef(pair.left, fset.PositionFor(hole.aExpr.Pos(), false).Offset), Right: parameterizationCallRef(pair.right, fset.PositionFor(hole.bExpr.Pos(), false).Offset)}
	}
	matched, err := rewriteflow.SameDependencies(pair.left, pair.right, substitutions)
	return err == nil && matched
}
func parameterizationCallRef(graph *pdg.Graph, offset int) pdg.Ref {
	for i, node := range graph.Nodes {
		if callAtOffset(graph, node, offset) {
			return pdg.Ref(i + 1)
		}
	}
	return 0
}
