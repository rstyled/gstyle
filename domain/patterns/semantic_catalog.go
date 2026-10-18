package patterns

import "github.com/rstyled/gstyle/domain/pdg"

func semanticCatalog(facts []*MiningFacts, graphs []*pdg.Graph) ([]*MiningFacts, error) {
	result := append([]*MiningFacts(nil), facts...)
	seen := map[string]bool{}
	for _, fact := range facts {
		seen[factID(fact)] = true
	}
	for _, graph := range graphs {
		if graph == nil {
			continue
		}
		if err := pdg.Validate(graph); err != nil {
			return result, err
		}
		name := graph.Tables.Text(graph.Function.ID)
		if seen[name] {
			continue
		}
		seen[name] = true
		result = append(result, callableMiningFact(graph, name))
	}
	return result, nil
}
func callableMiningFact(graph *pdg.Graph, name string) *MiningFacts {
	path := ""
	source := graph.Function.Span.Source
	if source > 0 && int(source) <= len(graph.Sources) {
		path = graph.Tables.Text(graph.Sources[source-1].Path)
	}
	return &MiningFacts{ID: name, Name: name, Path: path, Canonical: graph, Pdg: MiningView(graph)}
}
