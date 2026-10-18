package callflow

import (
	"encoding/json"
	"fmt"

	"github.com/rstyled/gstyle/domain/bindingflow"
	"github.com/rstyled/gstyle/domain/pdg"
)

func frameTargets(graph *pdg.Graph, query *bindingflow.Query, call pdg.Ref) ([]string, bool) {
	attrs := graph.NodeAttributes(graph.Nodes[call-1])
	if attrs.CallForm == pdg.DirectCall && attrs.Callee.Status == pdg.Known {
		return []string{graph.Tables.Text(attrs.Callee.Value)}, true
	}
	target := callTarget(graph, call)
	if target == 0 {
		return nil, false
	}
	origins := query.OriginsOf(target, bindingflow.Facts{})
	targets, known := functionOrigins(graph, origins)
	return targets, origins.Known && known
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
	attrs := graph.NodeAttributes(graph.Nodes[source.Node-1])
	if graph.Tables.Text(attrs.Construct) == "closure" {
		return callableIdentity(graph, attrs.Extensions, "gstyle.closure:body-id")
	}
	return declarationIdentity(graph, attrs.Extensions)
}
func declarationIdentity(graph *pdg.Graph, extensions []pdg.Extension) (string, error) {
	return callableIdentity(graph, extensions, "gstyle.go:function-value")
}

func callableIdentity(graph *pdg.Graph, extensions []pdg.Extension, key string) (string, error) {
	var identity string
	found := false
	for _, extension := range extensions {
		if graph.Tables.Text(extension.Name) != key {
			continue
		}
		if found {
			return "", fmt.Errorf("duplicate callable identity")
		}
		found = true
		if err := json.Unmarshal([]byte(graph.Tables.Text(extension.Value)), &identity); err != nil {
			return "", err
		}
	}
	if identity == "" {
		return "", fmt.Errorf("callable identity is unknown")
	}
	return identity, nil
}
