package gopatterns

import "github.com/rstyled/gstyle/domain/pdg"

func (p *packageAnalysis) assessScalarExtraction(source functionGraph) error {
	graph := source.graph
	if source.closure || graph.NormalFlow == nil {
		return nil
	}
	if _, _, err := nativeStage(graph, p.pkg, source.declaration); err != nil {
		graph.Conformance.Reasons = []pdg.Text{p.pool.Text(err.Error())}
		return nil
	}
	return certifyScalarExtraction(graph, p.pool)
}
