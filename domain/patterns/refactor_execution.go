package patterns

import "github.com/rstyled/gstyle/domain/pdg"

// Must-precede constraints form a partial order. A cycle cannot authorize a
// rewrite even when all individual edge evidence claims precision.
func acyclicExecution(graph *pdg.Graph) bool {
	successors, indegree := executionTopology(graph)
	queue := executionRoots(indegree)
	// Each operation is queued at most once. A fixed node bound also detects
	// a stalled frontier without retaining another visited set.
	for head := range graph.Nodes {
		if head >= len(queue) {
			return false
		}
		for _, target := range successors[queue[head]] {
			indegree[target-1]--
			if indegree[target-1] == 0 {
				queue = append(queue, int(target-1))
			}
		}
	}
	return true
}

func executionTopology(graph *pdg.Graph) ([][]pdg.Ref, []int) {
	successors := make([][]pdg.Ref, len(graph.Nodes))
	indegree := make([]int, len(graph.Nodes))
	for _, edge := range graph.Edges {
		if edge.Kind != pdg.Execution {
			continue
		}
		successors[edge.Source-1] = append(successors[edge.Source-1], edge.Target)
		indegree[edge.Target-1]++
	}
	return successors, indegree
}

func executionRoots(indegree []int) []int {
	var roots []int
	for i, degree := range indegree {
		if degree == 0 {
			roots = append(roots, i)
		}
	}
	return roots
}
