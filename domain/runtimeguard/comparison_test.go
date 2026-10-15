package runtimeguard

import (
	"maps"
	"testing"

	"github.com/rstyled/gstyle/domain/pdg"
)

func TestComparisonOperatorTables(t *testing.T) {
	cases := []struct {
		op, inverse, reverse string
		holds                func(int64, int64) bool
	}{
		{"<", ">=", ">", func(a, b int64) bool { return a < b }},
		{"<=", ">", ">=", func(a, b int64) bool { return a <= b }},
		{">", "<=", "<", func(a, b int64) bool { return a > b }},
		{">=", "<", "<=", func(a, b int64) bool { return a >= b }},
		{"==", "!=", "==", func(a, b int64) bool { return a == b }},
		{"!=", "==", "!=", func(a, b int64) bool { return a != b }},
	}
	predicates := map[string]func(int64, int64) bool{}
	for _, tc := range cases {
		predicates[tc.op] = tc.holds
	}
	values := []int64{-9223372036854775808, -1, 0, 1, 9223372036854775807}
	for _, tc := range cases {
		t.Run(tc.op, func(t *testing.T) {
			if inverse[tc.op] != tc.inverse || reverse[tc.op] != tc.reverse {
				t.Fatalf("inverse=%q reverse=%q", inverse[tc.op], reverse[tc.op])
			}
			for _, left := range values {
				for _, right := range values {
					if predicates[inverse[tc.op]](left, right) == tc.holds(left, right) {
						t.Fatalf("inverse does not negate %d %s %d", left, tc.op, right)
					}
					if predicates[reverse[tc.op]](right, left) != tc.holds(left, right) {
						t.Fatalf("reverse changes %d %s %d", left, tc.op, right)
					}
				}
			}
		})
	}
	for _, op := range []string{"", "+", " <", "< ", "<>", "≤", "==\x00"} {
		if inverse[op] != "" || reverse[op] != "" {
			t.Fatalf("unsupported operator %q gets a relation", op)
		}
	}
}

func TestComparisonWithoutBindingsPreservesState(t *testing.T) {
	view := &graphView{operands: [][2]pdg.Ref{{}}}
	for _, op := range []string{"<", "<=", ">", ">=", "==", "!=", "", "+"} {
		for _, truth := range []bool{false, true} {
			for _, state := range []State{nil, {}, {7: {HasLower: true, Lower: -3}}} {
				before := maps.Clone(state)
				got := view.comparison(1, op, truth, state)
				if (got == nil) != (state == nil) || !maps.Equal(got, before) {
					t.Fatalf("op=%q truth=%t state=%+v result=%+v", op, truth, before, got)
				}
				if got == nil {
					continue
				}
				marker := Value{Identity: 89}
				got[11] = marker
				if state[11] != marker {
					t.Fatal("comparison returns a different map")
				}
			}
		}
	}
}

func TestComparisonLeftNumericTransfer(t *testing.T) {
	initial := Value{Integer: true, HasLower: true, Lower: -10, HasUpper: true, Upper: 200}
	cases := []struct {
		name, op, bound string
		truth, tracked  bool
		initial, want   Value
		reject          bool
	}{
		{"less than", "<", "99", true, true, initial, Value{Integer: true, HasLower: true, Lower: -10, HasUpper: true, Upper: 98}, false},
		{"false less than", "<", "99", false, true, initial, Value{Integer: true, HasLower: true, Lower: 99, HasUpper: true, Upper: 200}, false},
		{"equality", "==", "99", true, true, initial, Value{Integer: true, HasLower: true, Lower: 99, HasUpper: true, Upper: 99}, false},
		{"untracked", "<", "99", true, false, initial, initial, false},
		{"unknown operator", "+", "99", true, true, initial, initial, false},
		{"below signed minimum", "<", "-9223372036854775808", true, true, Value{Integer: true}, Value{}, true},
		{"above signed maximum", ">", "9223372036854775807", true, true, Value{Integer: true}, Value{}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			view := comparisonTestView(t, tc.bound, tc.tracked)
			state := State{1: tc.initial}
			got := view.comparison(1, tc.op, tc.truth, state)
			if tc.reject {
				if got != nil || state[1] != tc.initial {
					t.Fatalf("rejected comparison result=%+v state=%+v", got, state)
				}
				return
			}
			if got == nil || got[1] != tc.want || state[1] != tc.want {
				t.Fatalf("result=%+v state=%+v want=%+v", got, state, tc.want)
			}
		})
	}
}

func comparisonTestView(t *testing.T, bound string, tracked bool) *graphView {
	t.Helper()
	view := evaluationTestView(t, "", "")
	view.fields[1] = 1
	view.kinds[0] = `"signed"`
	view.tracked = []bool{tracked}
	id := pdg.Text(len(view.graph.Tables.Strings) + 1)
	view.graph.Tables.Strings = append(view.graph.Tables.Strings, bound)
	entries := view.graph.AttributeTables.Extensions[view.graph.Attributes[2].Extensions-1]
	entries[0].Value = id
	return view
}

func TestComparisonRightNumericTransfer(t *testing.T) {
	initial := Value{Integer: true, HasLower: true, Lower: -10, HasUpper: true, Upper: 200}
	cases := []struct {
		op    string
		truth bool
		want  Value
	}{
		{"<", true, Value{Integer: true, HasLower: true, Lower: 100, HasUpper: true, Upper: 200}},
		{"<", false, Value{Integer: true, HasLower: true, Lower: -10, HasUpper: true, Upper: 99}},
		{"==", true, Value{Integer: true, HasLower: true, Lower: 99, HasUpper: true, Upper: 99}},
		{"+", true, initial},
	}
	for _, tc := range cases {
		view := comparisonTestView(t, "99", true)
		view.operands[0] = [2]pdg.Ref{3, 2}
		state := State{1: initial}
		got := view.comparison(1, tc.op, tc.truth, state)
		if got == nil || got[1] != tc.want || state[1] != tc.want {
			t.Fatalf("op=%q truth=%t result=%+v state=%+v want=%+v", tc.op, tc.truth, got, state, tc.want)
		}
	}
}

func TestComparisonBothNumericTransfers(t *testing.T) {
	span := func(lower, upper int64) Value {
		return Value{Integer: true, HasLower: true, Lower: lower, HasUpper: true, Upper: upper}
	}
	maximal := span(9223372036854775807, 9223372036854775807)
	marked := maximal
	marked.UpperChecked, marked.BelowLength = true, true
	cases := []struct {
		name, op              string
		truth, untrackedRight bool
		rightKey              pdg.Ref
		initial, want         State
		reject                bool
	}{
		{"tighten right", "<", true, false, 2, State{1: span(5, 5), 2: span(0, 10)}, State{1: span(5, 5), 2: span(6, 10)}, false},
		{"tighten left", "<", true, false, 2, State{1: span(0, 10), 2: span(5, 5)}, State{1: span(0, 4), 2: span(5, 5)}, false},
		{"false branch", "<", false, false, 2, State{1: span(5, 5), 2: span(0, 10)}, State{1: span(5, 5), 2: span(0, 5)}, false},
		{"untracked right", "<", true, true, 2, State{1: span(5, 5), 2: span(0, 10)}, State{1: span(5, 5), 2: span(0, 10)}, false},
		{"same key accepted", "==", true, false, 1, State{1: span(5, 5)}, State{1: span(5, 5)}, false},
		{"same key rejected", "<", true, false, 1, State{1: span(5, 5)}, State{1: span(5, 5)}, true},
		{"unknown false branch", "+", false, false, 2, State{1: span(5, 5), 2: span(0, 10)}, State{1: span(5, 5), 2: span(0, 10)}, false},
		{"second rejection keeps first update", "<", true, false, 2,
			State{1: maximal, 2: {Integer: true, Length: true}}, State{1: marked, 2: {Integer: true, Length: true}}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			view := comparisonTestView(t, "0", true)
			view.fields[2] = tc.rightKey
			view.kinds = []string{`"signed"`, `"signed"`}
			view.unstable = []bool{false, false}
			view.tracked = []bool{true, !tc.untrackedRight}
			got := view.comparison(1, tc.op, tc.truth, tc.initial)
			if (got == nil) != tc.reject || !maps.Equal(tc.initial, tc.want) {
				t.Fatalf("result=%+v state=%+v want=%+v rejected=%t", got, tc.initial, tc.want, tc.reject)
			}
			if tc.reject {
				return
			}
			if !maps.Equal(got, tc.want) {
				t.Fatalf("returned state=%+v want=%+v", got, tc.want)
			}
			marker := Value{Identity: 77}
			got[44] = marker
			if tc.initial[44] != marker {
				t.Fatal("comparison returns a different map")
			}
		})
	}
}

func TestConstrainInvalidBindingsPreserveState(t *testing.T) {
	view := &graphView{tracked: []bool{true}}
	for _, key := range []pdg.Ref{0, 2, ^pdg.Ref(0)} {
		for _, bound := range []Value{{Nil: true}, {Integer: true, HasLower: true, HasUpper: true, Lower: 8, Upper: 8}} {
			for _, state := range []State{nil, {}, {1: {Integer: true, HasLower: true, Lower: -3}}} {
				before := maps.Clone(state)
				got := view.constrain(key, "==", bound, state)
				if (got == nil) != (state == nil) || !maps.Equal(state, before) || !maps.Equal(got, before) {
					t.Fatalf("key=%d bound=%+v result=%+v state=%+v", key, bound, got, state)
				}
			}
		}
	}
}

func TestComparisonInvalidCachedBindingsStayConservative(t *testing.T) {
	for _, keys := range [][2]pdg.Ref{{2, 1}, {1, 2}, {^pdg.Ref(0), 1}, {1, ^pdg.Ref(0)}, {2, ^pdg.Ref(0)}} {
		for _, truth := range []bool{false, true} {
			view := comparisonTestView(t, "0", true)
			view.fields[1], view.fields[2] = keys[0], keys[1]
			state := State{1: {Integer: true, HasLower: true, Lower: -10, HasUpper: true, Upper: 200}}
			before := maps.Clone(state)
			got := view.comparison(1, "<", truth, state)
			if got == nil || !maps.Equal(got, before) || !maps.Equal(state, before) {
				t.Fatalf("keys=%v truth=%t result=%+v state=%+v", keys, truth, got, state)
			}
		}
	}
}

func TestComparisonPreservesConcreteNumericValues(t *testing.T) {
	operators := map[string]func(int64, int64) bool{
		"<":  func(a, b int64) bool { return a < b },
		"<=": func(a, b int64) bool { return a <= b },
		">":  func(a, b int64) bool { return a > b },
		">=": func(a, b int64) bool { return a >= b },
		"==": func(a, b int64) bool { return a == b },
		"!=": func(a, b int64) bool { return a != b },
	}
	values := []int64{-9223372036854775808, -1, 0, 1, 9223372036854775807}
	for op, holds := range operators {
		for _, left := range values {
			for _, right := range values {
				for bounds := 0; bounds < 16; bounds++ {
					for tracked := 0; tracked < 4; tracked++ {
						view := comparisonTestView(t, "0", true)
						view.fields[2] = 2
						view.kinds = []string{`"signed"`, `"signed"`}
						view.unstable = []bool{false, false}
						view.tracked = []bool{tracked&1 != 0, tracked&2 != 0}
						marker := Value{Identity: 173}
						state := State{
							1: {Integer: true, HasLower: bounds&1 != 0, Lower: left, HasUpper: bounds&2 != 0, Upper: left},
							2: {Integer: true, HasLower: bounds&4 != 0, Lower: right, HasUpper: bounds&8 != 0, Upper: right},
							3: marker,
						}
						got := view.comparison(1, op, holds(left, right), state)
						if got == nil {
							t.Fatalf("feasible branch rejected: %d %s %d bounds=%d tracked=%d", left, op, right, bounds, tracked)
						}
						for key, concrete := range map[pdg.Ref]int64{1: left, 2: right} {
							value := got[key]
							if value.HasLower && concrete < value.Lower || value.HasUpper && concrete > value.Upper {
								t.Fatalf("concrete value lost: %d %s %d key=%d value=%+v bounds=%d tracked=%d", left, op, right, key, value, bounds, tracked)
							}
						}
						if got[3] != marker {
							t.Fatal("comparison changes an unrelated binding")
						}
					}
				}
			}
		}
	}
}
