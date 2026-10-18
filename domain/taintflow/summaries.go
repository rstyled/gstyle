package taintflow

import (
	"github.com/rstyled/gstyle/domain/pdg"
	"slices"
)

func solveSummaries(queries map[string]*query, summaries map[string]summary) {
	queue := make([]string, 0, len(queries))
	pending := map[string]bool{}
	for name := range queries {
		queue = append(queue, name)
		pending[name] = true
	}
	slices.Sort(queue)
	callers := summaryCallers(queries)
	for cursor := 0; cursor < len(queue); cursor++ {
		name := queue[cursor]
		delete(pending, name)
		next := queries[name].analyze(queries, summaries)
		if summaryEqual(next, summaries[name]) {
			continue
		}
		summaries[name] = next
		queue = queueCallers(queue, pending, callers[name])
	}
}
func queueCallers(queue []string, pending map[string]bool, callers []string) []string {
	for _, name := range callers {
		if pending[name] {
			continue
		}
		pending[name] = true
		queue = append(queue, name)
	}
	return queue
}
func summaryCallers(queries map[string]*query) map[string][]string {
	result := map[string][]string{}
	for _, q := range queries {
		for _, node := range q.graph.Nodes {
			q.indexCaller(result, queries, node)
		}
	}
	for target := range result {
		slices.Sort(result[target])
		result[target] = slices.Compact(result[target])
	}
	return result
}
func (q *query) indexCaller(result map[string][]string, queries map[string]*query, node pdg.Node) {
	if node.Category != pdg.Call {
		return
	}
	attrs := q.graph.NodeAttributes(node)
	if attrs.CallForm != pdg.DirectCall || attrs.Callee.Status != pdg.Known {
		return
	}
	target := q.graph.Tables.Text(attrs.Callee.Value)
	if queries[target] != nil {
		result[target] = append(result[target], q.name)
	}
}
