package taintflow

import (
	"fmt"
	"slices"
)

type summarySolver struct {
	work      *analysisWork
	queries   map[string]*query
	summaries map[string]summary
	callers   map[string][]string
	queue     []string
	pending   map[string]bool
}

func (solver *summarySolver) enqueue(key string) {
	if solver.pending[key] {
		return
	}
	solver.pending[key] = true
	solver.queue = append(solver.queue, key)
}

func solveSummaries(queries map[string]*query, workLimit int) (map[string]summary, error) {
	solver := &summarySolver{work: &analysisWork{remaining: workLimit}, queries: queries, summaries: map[string]summary{}, callers: map[string][]string{}, pending: map[string]bool{}}
	for name := range queries {
		solver.summaries[name] = emptySummary()
		solver.enqueue(name)
	}
	slices.Sort(solver.queue)
	for cursor := 0; cursor < len(solver.queue); cursor++ {
		key := solver.queue[cursor]
		delete(solver.pending, key)
		next := solver.queries[key].analyze(solver)
		if !solver.work.available() {
			return nil, fmt.Errorf("%w in %s", ErrWorkLimit, solver.queries[key].name)
		}
		if summaryEqual(next, solver.summaries[key]) {
			continue
		}
		solver.summaries[key] = next
		for _, caller := range solver.callers[key] {
			solver.enqueue(caller)
		}
	}
	return solver.summaries, nil
}
