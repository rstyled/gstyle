package gopatterns

import (
	"fmt"
	"go/ast"

	"github.com/rstyled/gstyle/domain/pdg"
)

func (p *packageAnalysis) assessScalarExtraction(source functionGraph) error {
	graph := source.graph
	if source.closure || !scalarExtractionCandidate(graph, source.declaration) {
		return nil
	}
	if p.pkg.TypesSizes == nil || graph.Sources[0].ContentIdentity.Status != pdg.Known {
		return nil
	}
	if _, _, err := nativeStage(graph, p.pkg, source.declaration); err != nil {
		return fmt.Errorf("verify scalar extraction %s: %w", graph.Tables.Text(graph.Function.ID), err)
	}
	return certifyScalarExtraction(graph, p.pool)
}

func scalarExtractionCandidate(graph *pdg.Graph, fn *ast.FuncDecl) bool {
	return graph.NormalFlow != nil && len(graph.Function.Outputs) <= 1 && scalarOrderKnown(fn.Body)
}
