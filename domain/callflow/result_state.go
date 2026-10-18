package callflow

import (
	"cmp"
	"github.com/rstyled/gstyle/domain/pdg"
	"maps"
	"slices"
)

type tupleEntry struct {
	Ref    pdg.Ref
	Values string
}

func tupleEntries(values map[pdg.Ref]string) []tupleEntry {
	entries := make([]tupleEntry, 0, len(values))
	for ref, value := range values {
		entries = append(entries, tupleEntry{Ref: ref, Values: value})
	}
	slices.SortFunc(entries, func(a, b tupleEntry) int { return cmp.Compare(a.Ref, b.Ref) })
	return entries
}

func (evaluation *exitEvaluation) storeCallResult(point exitPoint, ref pdg.Ref, returned string) exitPoint {
	values := evaluation.values[point.values]
	nodes := map[pdg.Ref]exitValue{}
	maps.Copy(nodes, values.nodes)
	tuples := map[pdg.Ref]string{}
	maps.Copy(tuples, values.tuples)
	delete(nodes, ref)
	delete(tuples, ref)
	storeResult(nodes, tuples, ref, returned)
	values.nodes = nodes
	values.tuples = tuples
	point.values = evaluation.internValues(values)
	return point
}

func storeResult(nodes map[pdg.Ref]exitValue, tuples map[pdg.Ref]string, ref pdg.Ref, returned string) {
	values := decodeTuple(returned)
	if !knownTuple(values) {
		return
	}
	if len(values) == 1 {
		nodes[ref] = values[0]
		return
	}
	tuples[ref] = returned
}
