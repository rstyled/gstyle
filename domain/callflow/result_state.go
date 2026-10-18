package callflow

import (
	"cmp"
	"github.com/rstyled/gstyle/domain/bindingflow"
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

func concreteResults(values map[pdg.Ref]string) map[pdg.Ref][]bindingflow.Value {
	result := make(map[pdg.Ref][]bindingflow.Value, len(values))
	for ref, tuple := range values {
		result[ref] = concreteTuple(tuple)
	}
	return result
}

func concreteTuple(tuple string) []bindingflow.Value {
	values := decodeTuple(tuple)
	result := make([]bindingflow.Value, len(values))
	for i := range values {
		result[i] = concreteValue(values[i])
	}
	return result
}

func (evaluation *exitEvaluation) callResult(point exitPoint, ref pdg.Ref) exitPoint {
	returned := evaluation.body.calls.returnedValues(evaluation.summaries, ref, evaluation.valueContext(point))
	values := evaluation.values[point.values]
	nodes := map[pdg.Ref]exitValue{}
	maps.Copy(nodes, values.nodes)
	tuples := map[pdg.Ref]string{}
	maps.Copy(tuples, values.tuples)
	delete(nodes, ref)
	delete(tuples, ref)
	storeResult(nodes, tuples, ref, returned)
	point.values = evaluation.internValues(exitValues{bindings: values.bindings, nodes: nodes, tuples: tuples, cells: values.cells, unorderedReads: values.unorderedReads})
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
