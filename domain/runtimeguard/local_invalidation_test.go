package runtimeguard

import (
	"maps"
	"testing"

	"github.com/rstyled/gstyle/domain/pdg"
)

func requireLocalInvalidationInput(t *testing.T, view *graphView, state State) {
	t.Helper()
	for key, value := range state {
		if view.fieldRoots[key] != 0 {
			t.Fatalf("key %d has a field root", key)
		}
		if value.Identity != 0 && value.Identity%2 == 0 && value.Identity/2 > uint64(len(view.graph.Symbols)) {
			t.Fatalf("key %d has a field identity", key)
		}
	}
}

func TestInvalidateFieldsPreservesLocalRecords(t *testing.T) {
	for name, roots := range map[string]map[pdg.Ref]pdg.Ref{
		"nil": nil, "empty": {}, "explicit-zero": {1: 0}, "unrelated": {7: 1},
	} {
		t.Run(name, func(t *testing.T) {
			view := &graphView{graph: &pdg.Graph{Symbols: make([]pdg.Symbol, 2)}, fieldRoots: roots}
			state := State{}
			identities := []uint64{0, 1, 2, 4, 5, ^uint64(0)}
			for i, identity := range identities {
				value := assignmentRawValue()
				value.Identity = identity
				value.Lower, value.Upper = -1<<63, 1<<63-1
				value.HasLower, value.HasUpper = i%2 == 0, i%3 == 0
				state[pdg.Ref(i+1)] = value
			}
			requireLocalInvalidationInput(t, view, state)
			before, rootsBefore := maps.Clone(state), maps.Clone(roots)
			for _, ref := range []pdg.Ref{0, 1, ^pdg.Ref(0)} {
				view.invalidateFields(ref, state)
				if !maps.Equal(state, before) || !maps.Equal(roots, rootsBefore) {
					t.Fatalf("ref=%d state=%+v roots=%v", ref, state, roots)
				}
			}
		})
	}
}

func TestInvalidateFieldsPreservesEmptyLocalState(t *testing.T) {
	view := &graphView{graph: &pdg.Graph{}, fieldRoots: map[pdg.Ref]pdg.Ref{1: 2}}
	state := State{}
	view.invalidateFields(0, state)
	if len(state) != 0 {
		t.Fatalf("state=%+v", state)
	}
}

func TestFieldIdentityLocalBoundary(t *testing.T) {
	for _, tc := range []struct {
		identity uint64
		symbols  int
		want     bool
	}{
		{0, 0, false}, {0, 2, false}, {1, 0, false}, {2, 1, false},
		{4, 2, false}, {5, 2, false}, {^uint64(0), 2, false},
		{2, 0, true}, {6, 2, true}, {^uint64(0) - 1, 2, true},
	} {
		if got := fieldIdentity(tc.identity, tc.symbols); got != tc.want {
			t.Fatalf("identity=%d symbols=%d got=%v want=%v", tc.identity, tc.symbols, got, tc.want)
		}
	}
}
