package callflow

import (
	"maps"
	"testing"

	"github.com/rstyled/gstyle/domain/pdg"
)

func TestValueStatesResolveHashCollisions(t *testing.T) {
	evaluation := newExitEvaluation(nil, nil)
	evaluation.internValues(exitValues{nodes: map[pdg.Ref]exitValue{99: trueValue}})
	var ids []valueStateID
	for _, value := range []exitValue{nilValue, falseValue, trueValue} {
		state := exitValues{bindings: map[pdg.Ref]exitValue{1: value}}
		ids = append(ids, evaluation.internValueState(state, 42))
	}
	for index, value := range []exitValue{nilValue, falseValue, trueValue} {
		got := evaluation.internValueState(exitValues{bindings: map[pdg.Ref]exitValue{1: value}}, 42)
		if got != ids[index] {
			t.Fatalf("collision changes state %d: got %d want %d", index, got, ids[index])
		}
	}
	if ids[0] == ids[1] || ids[1] == ids[2] || ids[0] == ids[2] {
		t.Fatal("collision merges distinct states")
	}
}

func TestValueStatesRetainEveryComponent(t *testing.T) {
	evaluation := newExitEvaluation(nil, nil)
	states := []exitValues{
		{},
		{heap: environmentHeap{"x": {Value: trueValue}}},
		{heap: environmentHeap{"x": {Value: trueValue, Repeated: true}}},
		{locations: map[pdg.Ref]string{1: "x"}},
		{bindings: map[pdg.Ref]exitValue{1: trueValue}},
		{nodes: map[pdg.Ref]exitValue{1: trueValue}},
		{cells: map[pdg.Ref]exitValue{1: trueValue}},
		{tuples: map[pdg.Ref]string{1: "t"}},
		{unorderedReads: map[pdg.Ref]bool{1: true}},
	}
	seen := map[valueStateID]bool{}
	for _, state := range states {
		id := evaluation.internValues(state)
		if seen[id] {
			t.Fatalf("distinct state shares ID %d", id)
		}
		seen[id] = true
		if got := evaluation.internValues(state); got != id {
			t.Fatalf("state ID changes: %d to %d", id, got)
		}
	}
	if evaluation.internValues(exitValues{nodes: map[pdg.Ref]exitValue{}}) != 0 {
		t.Fatal("empty state has nonzero ID")
	}
}

func TestValueStateOrderAndSnapshot(t *testing.T) {
	evaluation := newExitEvaluation(nil, nil)
	original := exitValues{bindings: map[pdg.Ref]exitValue{1: trueValue, 2: nilValue}}
	id := evaluation.internValues(original)
	reverse := exitValues{bindings: map[pdg.Ref]exitValue{2: nilValue, 1: trueValue}}
	if evaluation.internValues(reverse) != id {
		t.Fatal("map order changes state identity")
	}
	next := original
	next.bindings = maps.Clone(original.bindings)
	next.bindings[1] = falseValue
	if evaluation.internValues(next) == id {
		t.Fatal("changed state reuses old identity")
	}
	if evaluation.values[id].bindings[1] != trueValue {
		t.Fatal("new state changes saved snapshot")
	}
	if count := testing.AllocsPerRun(100, func() { evaluation.internValues(reverse) }); count != 0 {
		t.Fatalf("repeated state lookup allocates %g times", count)
	}
}
