package taintflow

import (
	"github.com/rstyled/gstyle/domain/pdg"
	"testing"
)

func TestCallbackEffectsAreSeparateFromHeapEdges(t *testing.T) {
	root := object{Input: 1}
	unrelated := object{Allocation: "ordinary argument"}
	source := atom{Source: Site{Function: "source", Node: 1}}
	payload := emptyValue()
	payload.Objects[unrelated] = true
	payload.Data[source] = true
	state := emptyState()
	state.Memory[cellKey(root, callbackEffect)] = payload
	base := emptyValue()
	base.Objects[root] = true
	reachable := reachableStore(state, base)
	if reachable.Objects[unrelated] || reachable.Data[source] {
		t.Fatal("pending effect became a heap edge")
	}
}

func TestCallbackEffectProjectsOnlyToCapturedCells(t *testing.T) {
	environment := object{Allocation: "closure environment"}
	captured := object{Captured: true, Allocation: "live cell"}
	ordinary := object{Allocation: "ordinary field"}
	formal := object{Input: 2}
	for _, root := range []object{environment, ordinary, formal} {
		state := emptyState()
		if root == environment {
			reference := emptyValue()
			reference.Objects[captured] = true
			state.Memory[cellKey(environment, "capture:1")] = reference
		}
		actual := emptyValue()
		actual.Objects[root] = true
		e := evaluation{state: state, arguments: map[pdg.Position]value{1: actual}}
		projected := map[cell]value{}
		e.projectWrite(0, cell{Object: object{Input: 1}, Fields: callbackEffect}, emptyValue(), writeProjection{values: projected, weak: map[cell]bool{}})
		switch root {
		case environment:
			if _, ok := projected[cellKey(captured, "*")]; !ok || len(projected) != 1 {
				t.Fatalf("missing captured-cell effect: %v", projected)
			}
		case ordinary:
			if len(projected) != 0 {
				t.Fatal("callback effect mutated ordinary heap")
			}
		case formal:
			if _, ok := projected[cellKey(formal, callbackEffect)]; !ok || len(projected) != 1 {
				t.Fatal("wrapper lost pending callback effect")
			}
		}
	}
}
