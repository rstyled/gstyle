package runtimeguard

import (
	"reflect"
	"testing"

	"github.com/rstyled/gstyle/domain/pdg"
)

func TestBooleanFactBranchAdmission(t *testing.T) {
	cases := []struct {
		value                   string
		rejectTrue, rejectFalse bool
	}{
		{"true", false, true}, {"false", true, false}, {"", false, false},
		{`"true"`, false, false}, {`"false"`, false, false}, {"null", false, false},
		{"0", false, false}, {"1", false, false}, {"[]", false, false}, {"{}", false, false},
		{"True", false, false}, {"FALSE", false, false}, {" true ", false, false},
		{"false\n", false, false}, {"\x00false", false, false}, {"\xff", false, false},
	}
	for _, tc := range cases {
		for _, truth := range []bool{true, false} {
			view := booleanFactView(t, tc.value)
			state := State{42: {Lower: -7, HasLower: true, Guarded: true}}
			want := tc.rejectFalse
			if truth {
				want = tc.rejectTrue
			}
			got := view.assume(1, truth, state)
			if (got == nil) != want {
				t.Fatalf("fact=%q truth=%t nil=%t want=%t", tc.value, truth, got == nil, want)
			}
			if !reflect.DeepEqual(state, State{42: {Lower: -7, HasLower: true, Guarded: true}}) {
				t.Fatal("Boolean admission changes the input state")
			}
		}
	}
}

func booleanFactView(t *testing.T, value string) *graphView {
	t.Helper()
	pool := pdg.NewBuilder()
	attrs := []pdg.Attributes{{Extensions: pool.SingletonExtension("gstyle.runtime:boolean", value)}}
	graph := pdg.Graph{Tables: pool.Tables, Nodes: []pdg.Node{{Category: pdg.Computation, Attributes: 1}}}
	if err := pdg.PackAttributes(&graph, attrs); err != nil {
		t.Fatal(err)
	}
	return &graphView{graph: &graph, operands: [][2]pdg.Ref{{0, 0}}, fields: make([]pdg.Ref, 1)}
}
