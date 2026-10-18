package gopatterns

import (
	"slices"
	"testing"

	"github.com/rstyled/gstyle/domain/pdg"
)

func TestMergeDefinitionSets(t *testing.T) {
	for left := 0; left < 32; left++ {
		for right := 0; right < 32; right++ {
			current, incoming := definitionSubset(left), definitionSubset(right)
			beforeCurrent, beforeIncoming := slices.Clone(current), slices.Clone(incoming)
			want := definitionSubset(left | right)
			got := mergeDefinitions(current, incoming)
			if !slices.Equal(got, want) {
				t.Fatalf("%v union %v = %v, want %v", current, incoming, got, want)
			}
			if !slices.Equal(current, beforeCurrent) || !slices.Equal(incoming, beforeIncoming) {
				t.Fatal("merge modifies a predecessor")
			}
			if left|right == left && len(current) > 0 && &got[0] != &current[0] {
				t.Fatal("unchanged union copies the current set")
			}
		}
	}
}

func definitionSubset(bits int) []pdg.Ref {
	var values []pdg.Ref
	for i := 0; i < 5; i++ {
		if bits&(1<<i) != 0 {
			values = append(values, pdg.Ref(i+1))
		}
	}
	return values
}

func TestReachingMergePreservesPredecessorStorage(t *testing.T) {
	storage := []pdg.Ref{1, 3, 5, 7, 9, 11}
	before := slices.Clone(storage)
	predecessor := reachingState{1: storage[:3], 2: storage[1:5]}
	state := reachingState{}
	state.merge(predecessor)
	state.merge(reachingState{1: {2, 8}, 2: {6}})
	if !slices.Equal(state[1], []pdg.Ref{1, 2, 3, 5, 8}) || !slices.Equal(state[2], []pdg.Ref{3, 5, 6, 7, 9}) {
		t.Fatal(state)
	}
	if !slices.Equal(storage, before) {
		t.Fatal("merge overwrites predecessor storage")
	}
	state[1][0] = 99
	if !slices.Equal(storage, before) {
		t.Fatal("merged state aliases a predecessor")
	}
}

func TestReachingMergeSharesImmutableSets(t *testing.T) {
	incoming := []pdg.Ref{1, 3, 5, 7}
	predecessor := reachingState{1: incoming}
	state := reachingState{1: nil}
	equivalent := reachingState{1: slices.Clone(incoming)}
	count := testing.AllocsPerRun(100, func() { state[1] = nil; state.merge(predecessor); state.merge(predecessor); state.merge(equivalent) })
	if count != 0 {
		t.Fatalf("unchanged predecessor allocates %v times", count)
	}
	if &state[1][0] != &incoming[0] {
		t.Fatal("unchanged predecessor copied")
	}
	state.merge(reachingState{1: {2, 8}})
	if !slices.Equal(incoming, []pdg.Ref{1, 3, 5, 7}) {
		t.Fatal("union mutates a shared predecessor")
	}
	if !slices.Equal(state[1], []pdg.Ref{1, 2, 3, 5, 7, 8}) {
		t.Fatal(state)
	}
}

func TestMergeUnchangedDefinitionsDoesNotAllocate(t *testing.T) {
	current := []pdg.Ref{1, 3, 5, 7, 9}
	incoming := []pdg.Ref{3, 7}
	var got []pdg.Ref
	count := testing.AllocsPerRun(100, func() { got = mergeDefinitions(current, incoming) })
	if count != 0 {
		t.Fatalf("unchanged union allocates %v times", count)
	}
	if !slices.Equal(got, current) {
		t.Fatal(got)
	}
}
