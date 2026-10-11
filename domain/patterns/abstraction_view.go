package patterns

// Existing abstraction analyses use value and control dependencies. Execution
// constraints remain available to the separate execution-aware clone profile.
func abstractionEdge(edge PdgEdge) bool {
	return edge.Kind == Data || edge.Kind == Ctrl
}

func abstractionAdjacency(graph *MiningGraph, reverse bool) [][]int {
	adjacency := make([][]int, len(graph.Nodes))
	for _, edge := range graph.Edges {
		if !abstractionEdge(edge) {
			continue
		}
		from, to := edge.From, edge.To
		if reverse {
			from, to = to, from
		}
		adjacency[from] = append(adjacency[from], to)
	}
	return adjacency
}
