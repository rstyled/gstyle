package runtimeguard

import (
	"maps"
	"math"
	"testing"

	"github.com/rstyled/gstyle/domain/pdg"
)

func numericTestView(kind string, tracked, unstable bool) *graphView {
	return &graphView{kinds: []string{kind}, tracked: []bool{tracked}, unstable: []bool{unstable}}
}

func TestNumericConstraintStateCases(t *testing.T) {
	cases := []struct {
		name, kind, op       string
		tracked, unstable    bool
		initial, bound, want Value
		reject               bool
	}{
		{name: "untracked", kind: `"signed"`, op: ">=", bound: Value{Lower: 0, Upper: 0, HasLower: true, HasUpper: true}},
		{name: "lower guard", kind: `"signed"`, tracked: true, op: ">=", bound: Value{HasLower: true, HasUpper: true}, want: Value{HasLower: true}},
		{name: "length guard", kind: `"signed"`, tracked: true, op: "<", bound: Value{Length: true}, want: Value{UpperChecked: true, BelowLength: true}},
		{name: "inclusive length", kind: `"signed"`, tracked: true, op: "<=", bound: Value{Length: true}, want: Value{UpperChecked: true}},
		{name: "unknown bound", kind: `"signed"`, tracked: true, op: ">=", initial: Value{Lower: -3, HasLower: true}, bound: Value{Lower: 7}, want: Value{Lower: -3, HasLower: true}},
		{name: "unsupported operator", kind: `"signed"`, tracked: true, op: "!=", bound: Value{HasLower: true, HasUpper: true}},
		{name: "contradiction", kind: `"signed"`, tracked: true, op: "<", initial: Value{Lower: 0, HasLower: true}, bound: Value{HasLower: true, HasUpper: true}, reject: true},
		{name: "strict minimum", kind: `"signed"`, tracked: true, op: "<", bound: Value{Lower: math.MinInt64, Upper: math.MinInt64, HasLower: true, HasUpper: true}, reject: true},
		{name: "strict maximum", kind: `"signed"`, tracked: true, op: ">", bound: Value{Lower: math.MaxInt64, Upper: math.MaxInt64, HasLower: true, HasUpper: true}, reject: true},
		{name: "unstable discards cache", kind: `"signed"`, tracked: true, unstable: true, op: ">=", initial: Value{Upper: -1, HasUpper: true}, bound: Value{HasLower: true, HasUpper: true}, want: Value{HasLower: true}},
		{name: "unsigned default", kind: `"unsigned"`, tracked: true, unstable: true, op: "other", want: Value{HasLower: true}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			state := State{1: tc.initial, 99: {Identity: 9, Lower: -7, HasLower: true}}
			before := maps.Clone(state)
			got := numericTestView(tc.kind, tc.tracked, tc.unstable).constrain(1, tc.op, tc.bound, state)
			if (got == nil) != tc.reject {
				t.Fatalf("rejected=%v want=%v", got == nil, tc.reject)
			}
			if tc.reject || !tc.tracked {
				if !maps.Equal(state, before) {
					t.Fatal("rejected or untracked constraint changes state")
				}
				return
			}
			if got[1] != tc.want || got[99] != before[99] {
				t.Fatalf("unexpected state: %+v", got)
			}
		})
	}
}

func TestNumericConstraintsPreserveSatisfyingMembers(t *testing.T) {
	values := []Value{{}, {HasLower: true}, {Lower: -2, Upper: 2, HasLower: true, HasUpper: true}, {Lower: 1, Upper: -1, HasLower: true, HasUpper: true}}
	for _, unstable := range []bool{false, true} {
		view := numericTestView(`"signed"`, true, unstable)
		for _, initial := range values {
			for _, bound := range []int64{math.MinInt64, -1, 0, 1, math.MaxInt64} {
				for _, op := range []string{"<", "<=", ">", ">=", "==", "!=", "other"} {
					state := State{1: initial, 99: {Identity: 99}}
					input := view.binding(1, state)
					out := view.constrain(1, op, Value{Lower: bound, Upper: bound, HasLower: true, HasUpper: true}, state)
					for _, x := range []int64{math.MinInt64, -2, -1, 0, 1, 2, math.MaxInt64} {
						if !numericMember(input, x) || !numericSatisfies(op, x, bound) {
							continue
						}
						if out == nil || !numericMember(view.binding(pdg.Ref(1), out), x) {
							t.Fatalf("lost member: initial=%+v unstable=%v op=%s bound=%d x=%d", initial, unstable, op, bound, x)
						}
					}
					if state[99].Identity != 99 {
						t.Fatal("constraint changes another key")
					}
				}
			}
		}
	}
}

func numericMember(v Value, x int64) bool {
	return (!v.HasLower || v.Lower <= x) && (!v.HasUpper || x <= v.Upper)
}

func numericSatisfies(op string, x, bound int64) bool {
	switch op {
	case "<":
		return x < bound
	case "<=":
		return x <= bound
	case ">":
		return x > bound
	case ">=":
		return x >= bound
	case "==":
		return x == bound
	case "!=":
		return x != bound
	default:
		return true
	}
}

func TestNumericConstraintAbsentVersusPresentZero(t *testing.T) {
	for _, present := range []bool{false, true} {
		state := State{}
		if present {
			state[1] = Value{}
		}
		view := numericTestView(`"unsigned"`, true, false)
		out := view.constrain(1, "other", Value{}, state)
		want := Value{HasLower: !present}
		if out == nil || len(out) != 1 || out[1] != want {
			t.Fatalf("present=%v got=%+v want=%+v", present, out, want)
		}
		out[1] = Value{Identity: 42}
		if state[1].Identity != 42 {
			t.Fatal("accepted constraint must return its input map")
		}
	}
	state := State{}
	if got := numericTestView(`"signed"`, false, false).constrain(1, "<", Value{Length: true}, state); got == nil || len(got) != 0 {
		t.Fatalf("untracked absent key must remain absent: %+v", got)
	}
}
