package runtimeguard

import (
	"maps"
	"reflect"
	"testing"

	"github.com/rstyled/gstyle/domain/pdg"
)

func finiteAssumeView(t *testing.T, op string) *graphView {
	t.Helper()
	pool := pdg.NewBuilder()
	ops := []string{op, "parentheses", "!", "<", "", "", "||", "<", "", "", "!", "<", "", ""}
	attrs := make([]pdg.Attributes, len(ops))
	nodes := make([]pdg.Node, len(ops))
	for i, operator := range ops {
		attrs[i].Operator = pdg.Fact{Status: pdg.Known, Value: pool.Text(operator)}
		nodes[i] = pdg.Node{Category: pdg.Computation, Attributes: pdg.Ref(i + 1)}
	}
	for i, value := range map[int]string{5: "0", 9: "5", 13: "10"} {
		attrs[i].Extensions = pool.SingletonExtension("gstyle.runtime:integer", value)
	}
	graph := pdg.Graph{Tables: pool.Tables, Nodes: nodes, Symbols: make([]pdg.Symbol, 2)}
	if err := pdg.PackAttributes(&graph, attrs); err != nil {
		t.Fatal(err)
	}
	fields := make([]pdg.Ref, len(ops))
	fields[4], fields[8], fields[12] = 1, 1, 1
	return &graphView{graph: &graph, fields: fields, kinds: []string{`"signed"`, `"signed"`}, tracked: []bool{true, true}, unstable: []bool{false, false},
		operands: [][2]pdg.Ref{{2, 7}, {3, 0}, {4, 0}, {5, 6}, {}, {}, {8, 11}, {9, 10}, {}, {}, {12, 0}, {13, 14}, {}, {}}}
}

func finiteAssumeTruth(op string, value int64) bool {
	left, right := value >= 0, value < 5 || value >= 10
	if op == "&&" {
		return left && right
	}
	return left || right
}

func TestFiniteAssumePreservesRepresentedValues(t *testing.T) {
	for _, op := range []string{"&&", "||"} {
		for _, truth := range []bool{false, true} {
			for _, bounds := range []Value{{}, {HasLower: true, Lower: -20}, {HasUpper: true, Upper: 20}, {HasLower: true, Lower: -20, HasUpper: true, Upper: 20}} {
				checkFiniteAssumeValues(t, op, truth, bounds)
			}
		}
	}
}

func checkFiniteAssumeValues(t *testing.T, op string, truth bool, bounds Value) {
	t.Helper()
	for _, value := range []int64{-9223372036854775808, -20, -1, 0, 4, 5, 9, 10, 20, 9223372036854775807} {
		if !finiteAssumeContains(bounds, value) || finiteAssumeTruth(op, value) != truth {
			continue
		}
		view := finiteAssumeView(t, op)
		operands := append([][2]pdg.Ref(nil), view.operands...)
		state := State{1: bounds, 2: {HasLower: true, Lower: -7, Guarded: true}}
		before := maps.Clone(state)
		out := view.assume(1, truth, state)
		if out == nil || !finiteAssumeContains(out[1], value) || out[2] != before[2] {
			t.Fatalf("%s truth=%v value=%d before=%+v out=%+v", op, truth, value, before, out)
		}
		if !reflect.DeepEqual(operands, view.operands) {
			t.Fatal("assume changes graph operands")
		}
		if (op == "&&") != truth && !maps.Equal(state, before) {
			t.Fatal("alternative composition changes its input")
		}
	}
}

func TestFiniteAssumeGuardRejectsBeforeRecursiveDispatch(t *testing.T) {
	view := booleanFactView(t, "false")
	view.graph.Tables.Strings = append(view.graph.Tables.Strings, "parentheses")
	view.graph.Attributes[0].Operator = pdg.Fact{Status: pdg.Known, Value: pdg.Text(len(view.graph.Tables.Strings))}
	view.operands[0][0] = 1
	state := State{1: {HasLower: true, Lower: -4}}
	before := maps.Clone(state)
	if got := view.assume(1, true, state); got != nil || !maps.Equal(state, before) {
		t.Fatalf("guard does not reject before the self-reference: %+v", got)
	}
}

func finiteAssumeContains(value Value, concrete int64) bool {
	return (!value.HasLower || value.Lower <= concrete) && (!value.HasUpper || concrete <= value.Upper)
}

func TestFiniteAssumeExactBounds(t *testing.T) {
	for _, tc := range []struct {
		op           string
		truth        bool
		lower, upper int64
	}{
		{"&&", true, 0, 20},
		{"&&", false, -20, 9},
		{"||", true, -20, 20},
	} {
		view := finiteAssumeView(t, tc.op)
		state := State{1: {HasLower: true, Lower: -20, HasUpper: true, Upper: 20}}
		out := view.assume(1, tc.truth, state)
		want := Value{HasLower: true, Lower: tc.lower, HasUpper: true, Upper: tc.upper}
		if out == nil || out[1] != want {
			t.Fatalf("%s truth=%v out=%+v want=%+v", tc.op, tc.truth, out, want)
		}
	}
}

func TestFiniteAssumeRejectsImpossibleComposition(t *testing.T) {
	view := finiteAssumeView(t, "||")
	state := State{1: {HasLower: true, Lower: -20, HasUpper: true, Upper: 20}}
	if out := view.assume(1, false, state); out != nil {
		t.Fatalf("false branch of a tautology survives: %+v", out)
	}
}

func TestFiniteAssumeAllocatedEmptyState(t *testing.T) {
	for _, op := range []string{"&&", "||"} {
		for _, truth := range []bool{false, true} {
			for _, value := range []int64{-1, 0, 5, 10} {
				if finiteAssumeTruth(op, value) != truth {
					continue
				}
				view := finiteAssumeView(t, op)
				state := State{}
				out := view.assume(1, truth, state)
				if out == nil || !finiteAssumeContains(out[1], value) {
					t.Fatalf("empty state loses %d for %s truth=%v: %+v", value, op, truth, out)
				}
			}
		}
	}
}
