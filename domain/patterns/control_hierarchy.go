package patterns

// Legacy structural rewrites require a hierarchy. A semantic PDG can contain
// several controllers and cycles; do not flatten that information to run a rule.
func hierarchicalControl(graph *MiningGraph) bool {
	parents := make([]int, len(graph.Nodes))
	for index := range parents {
		parents[index] = -1
	}
	for _, edge := range graph.Edges {
		if edge.Kind != Ctrl {
			continue
		}
		if !assignControlParent(parents, edge.From, edge.To) {
			return false
		}
	}
	return acyclicControlParents(parents)
}

func assignControlParent(parents []int, from, to int) bool {
	if from == to || (parents[to] >= 0 && parents[to] != from) {
		return false
	}
	parents[to] = from
	return true
}

func acyclicControlParents(parents []int) bool {
	state := make([]uint8, len(parents))
	for index := range parents {
		if state[index] != 0 {
			continue
		}
		if !visitControlParents(parents, state, index) {
			return false
		}
	}
	return true
}

func visitControlParents(parents []int, state []uint8, start int) bool {
	path := []int{}
	at := start
	for at >= 0 && state[at] == 0 {
		state[at] = 1
		path = append(path, at)
		at = parents[at]
	}
	if at >= 0 && state[at] == 1 {
		return false
	}
	for _, node := range path {
		state[node] = 2
	}
	return true
}
