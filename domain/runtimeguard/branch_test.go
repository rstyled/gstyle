package runtimeguard

import (
	"maps"
	"strconv"
	"testing"

	"github.com/rstyled/gstyle/domain/pdg"
)

func branchTestView(t *testing.T, lhs, rhs bool) *graphView {
	t.Helper()
	pool := pdg.NewBuilder()
	attrs := []pdg.Attributes{{},
		{Extensions: pool.SingletonExtension("gstyle.runtime:boolean", strconv.FormatBool(lhs))},
		{Extensions: pool.SingletonExtension("gstyle.runtime:boolean", strconv.FormatBool(rhs))},
	}
	graph := pdg.Graph{Tables: pool.Tables, Nodes: []pdg.Node{{Category: pdg.Control, Attributes: 1}, {Category: pdg.Computation, Attributes: 2}, {Category: pdg.Computation, Attributes: 3}}, Symbols: make([]pdg.Symbol, 1)}
	if err := pdg.PackAttributes(&graph, attrs); err != nil {
		t.Fatal(err)
	}
	return &graphView{graph: &graph, kinds: []string{`"signed"`}, tracked: []bool{true}, unstable: []bool{false}, fields: make([]pdg.Ref, 3), operands: [][2]pdg.Ref{{2, 3}, {0, 0}, {0, 0}}}
}

func TestLogicalTruthAndAlternativeIsolation(t *testing.T) {
	for _, op := range []string{"&&", "||"} {
		for _, truth := range []bool{false, true} {
			for _, lhs := range []bool{false, true} {
				for _, rhs := range []bool{false, true} {
					view := branchTestView(t, lhs, rhs)
					state := State{1: {Lower: -7, Upper: 9, HasLower: true, HasUpper: true}}
					before := maps.Clone(state)
					out := view.logical(1, op, truth, state)
					condition := lhs || rhs
					if op == "&&" {
						condition = lhs && rhs
					}
					if (out != nil) != (condition == truth) {
						t.Fatalf("op=%s truth=%v lhs=%v rhs=%v out=%+v", op, truth, lhs, rhs, out)
					}
					if !maps.Equal(state, before) {
						t.Fatal("Boolean branch changes input records")
					}
					if (op == "&&") == truth || out == nil {
						continue
					}
					out[99] = Value{Identity: 99}
					if !maps.Equal(state, before) {
						t.Fatal("alternative output aliases its input")
					}
				}
			}
		}
	}
}

func TestLogicalSequentialStopsAtUnreachableLeft(t *testing.T) {
	for _, tc := range []struct {
		op         string
		truth, lhs bool
	}{{"&&", true, false}, {"||", false, true}} {
		view := branchTestView(t, tc.lhs, true)
		view.operands[0][1] = 99
		if got := view.logical(1, tc.op, tc.truth, State{}); got != nil {
			t.Fatalf("right side must not execute: %+v", got)
		}
	}
}

func TestLogicalEmptyStateIsReachable(t *testing.T) {
	for _, lhs := range []bool{false, true} {
		view := branchTestView(t, lhs, true)
		state := State{}
		out := view.logical(1, "||", true, state)
		if out == nil || len(out) != 0 {
			t.Fatalf("allocated empty state lost: %+v", out)
		}
		out[99] = Value{}
		if len(state) != 0 {
			t.Fatal("empty alternative output aliases input")
		}
	}
}

func TestOperandGuardBoundaries(t *testing.T) {
	view := (*graphView)(nil)
	if got := view.operand(0, -1); got != 0 {
		t.Fatalf("zero reference=%d", got)
	}
	if got := view.operand(99, 2); got != 0 {
		t.Fatalf("outside position=%d", got)
	}
	view = &graphView{operands: [][2]pdg.Ref{{7, 9}}}
	if view.operand(1, 0) != 7 || view.operand(1, 1) != 9 {
		t.Fatal("operand order changed")
	}
}
