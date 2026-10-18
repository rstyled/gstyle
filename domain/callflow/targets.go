package callflow

import (
	"fmt"

	"github.com/rstyled/gstyle/domain/bindingflow"
	"github.com/rstyled/gstyle/domain/pdg"
)

func frameTargets(graph *pdg.Graph, query *bindingflow.Query, call pdg.Ref) callTargets {
	attrs := graph.NodeAttributes(graph.Nodes[call-1])
	if attrs.CallForm == pdg.DirectCall && attrs.Callee.Status == pdg.Known {
		return callTargets{names: []string{graph.Tables.Text(attrs.Callee.Value)}, known: true}
	}
	target := callTarget(graph, call)
	if target == 0 {
		return callTargets{}
	}
	origins := query.OriginsOf(target, bindingflow.Facts{})
	targets, known := functionOrigins(graph, origins)
	return callTargets{source: target, names: targets, known: origins.Known && known}
}
func callTarget(graph *pdg.Graph, call pdg.Ref) pdg.Ref {
	var source pdg.Ref
	for _, edge := range graph.Edges {
		if !targetEdge(graph, edge, call) {
			continue
		}
		if source != 0 && source != edge.Source {
			return 0
		}
		source = edge.Source
	}
	return source
}
func targetEdge(graph *pdg.Graph, edge pdg.Edge, call pdg.Ref) bool {
	return edge.Kind == pdg.Data && edge.Target == call && graph.Tables.Text(edge.Subkind) == "call-target"
}
func functionOrigins(graph *pdg.Graph, origins bindingflow.Origins) ([]string, bool) {
	known := len(origins.Sources) > 0
	var targets []string
	for _, source := range origins.Sources {
		target, err := functionValue(graph, source)
		if err != nil {
			known = false
			continue
		}
		targets = append(targets, target)
	}
	return targets, known
}
func functionValue(graph *pdg.Graph, source bindingflow.ValueSource) (string, error) {
	if source.Zero || source.Result != 0 {
		return "", fmt.Errorf("function value is not a declaration")
	}
	return bindingflow.CallableIdentity(graph, source.Node)
}
