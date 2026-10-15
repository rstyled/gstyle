package runtimeguard

import (
	"maps"
	"testing"
)

func TestEqualStateContent(t *testing.T) {
	states := []State{nil, {}, {0: {}}, {0xffffffff: {}}}
	values := []Value{
		{}, {Identity: 1}, {Lower: -1}, {Upper: 2}, {HasLower: true}, {HasUpper: true},
		{NonNil: true}, {Nil: true}, {Guarded: true}, {UpperChecked: true},
		{Integer: true}, {Length: true}, {BelowLength: true}, {Extent: -3}, {HasExtent: true},
	}
	for _, value := range values {
		states = append(states, State{1: value}, State{2: value}, State{1: value, 2: {}}, State{1: {}, 2: value})
	}
	for i, a := range states {
		for j, b := range states {
			beforeA, beforeB := maps.Clone(a), maps.Clone(b)
			if got, want := equalState(a, b), maps.Equal(a, b); got != want {
				t.Fatalf("states %d,%d: equalState=%t want=%t", i, j, got, want)
			}
			if !maps.Equal(a, beforeA) || !maps.Equal(b, beforeB) {
				t.Fatal("state comparison changes an input")
			}
		}
	}
}
