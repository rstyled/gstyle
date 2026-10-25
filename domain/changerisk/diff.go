package changerisk

import (
	"maps"
	"slices"
)

func graphDelta(before, after Function) Delta {
	current := currentFunction(before, after)
	old, next := nodesByKey(before.Graph.Nodes), nodesByKey(after.Graph.Nodes)
	delta := Delta{Name: current.Name, Path: current.Path, Line: current.Line,
		Added: difference(after.Graph.Nodes, old), Removed: difference(before.Graph.Nodes, next),
		Context: []Node{}, Edges: []ChangedEdge{}, Ambiguous: before.Graph.Ambiguous || after.Graph.Ambiguous}
	if !delta.Ambiguous {
		delta.Edges = changedEdges(before.Graph.Edges, after.Graph.Edges)
	}
	delta.Context = contextNodes(delta, old, next)
	return delta
}

func difference(nodes []Node, exclude map[string]Node) []Node {
	result := []Node{}
	for _, node := range nodes {
		if _, ok := exclude[node.Key]; !ok {
			result = append(result, node)
		}
	}
	return result
}

func nodesByKey(nodes []Node) map[string]Node {
	result := map[string]Node{}
	for _, node := range nodes {
		result[node.Key] = node
	}
	return result
}

func changedEdges(before, after []Edge) []ChangedEdge {
	old, next := edgesByKey(before), edgesByKey(after)
	result := []ChangedEdge{}
	for _, edge := range before {
		if !next[edge.Key] {
			result = append(result, ChangedEdge{edge, "removed"})
		}
	}
	for _, edge := range after {
		if !old[edge.Key] {
			result = append(result, ChangedEdge{edge, "added"})
		}
	}
	return result
}

func edgesByKey(edges []Edge) map[string]bool {
	result := map[string]bool{}
	for _, edge := range edges {
		result[edge.Key] = true
	}
	return result
}

func contextNodes(delta Delta, old, next map[string]Node) []Node {
	all := maps.Clone(old)
	maps.Copy(all, next)
	changed := nodesByKey(append(slices.Clone(delta.Added), delta.Removed...))
	context := map[string]Node{}
	for _, edge := range delta.Edges {
		for _, key := range []string{edge.From, edge.To} {
			if _, ok := changed[key]; !ok {
				context[key] = all[key]
			}
		}
	}
	var result []Node
	for _, key := range slices.Sorted(maps.Keys(context)) {
		result = append(result, context[key])
	}
	return result
}
