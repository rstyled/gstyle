package taintflow

import (
	"slices"
)

type summarySolver struct {
	queries   map[string]*query
	summaries map[string]summary
	callers   map[string][]string
	queue     []string
	pending   map[string]bool
}

func solveSummaries(queries map[string]*query, summaries map[string]summary) {
	solver := &summarySolver{queries: queries, summaries: summaries, callers: map[string][]string{}, pending: map[string]bool{}}
	for name := range queries {
		solver.enqueue(name)
	}
	slices.Sort(solver.queue)
	for cursor := 0; cursor < len(solver.queue); cursor++ {
		key := solver.queue[cursor]
		delete(solver.pending, key)
		next := solver.queries[key].analyze(solver)
		if summaryEqual(next, solver.summaries[key]) {
			continue
		}
		solver.summaries[key] = next
		for _, caller := range solver.callers[key] {
			solver.enqueue(caller)
		}
	}
}
func (solver *summarySolver) enqueue(key string) {
	if solver.pending[key] {
		return
	}
	solver.pending[key] = true
	solver.queue = append(solver.queue, key)
}
