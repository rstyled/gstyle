package runtimeguard

import (
	"math"
	"testing"
)

func TestValueMergeFacts(t *testing.T) {
	all := Value{Identity: 7, NonNil: true, Nil: true, Guarded: true, UpperChecked: true,
		Integer: true, Length: true, BelowLength: true, Extent: 12, HasExtent: true}
	cases := []struct {
		name string
		a, b Value
		want Value
	}{
		{"common facts", all, all, Value{Identity: 7, NonNil: true, Nil: true, Guarded: true, UpperChecked: true, Length: true, BelowLength: true}},
		{"left-only facts", all, Value{}, Value{UpperChecked: true}},
		{"right-only facts", Value{}, all, Value{UpperChecked: true}},
		{"different identities", Value{Identity: 7, NonNil: true}, Value{Identity: 8, NonNil: true}, Value{NonNil: true}},
		{"maximum identity", Value{Identity: math.MaxUint64}, Value{Identity: math.MaxUint64}, Value{Identity: math.MaxUint64}},
		{"empty", Value{}, Value{}, Value{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.a.merge(tc.b); got != tc.want {
				t.Fatalf("got=%+v want=%+v", got, tc.want)
			}
		})
	}
}

func TestValueMergePreservesBothIntervalSets(t *testing.T) {
	values := []Value{
		{}, {Lower: math.MaxInt64, Upper: math.MinInt64},
		{Lower: math.MinInt64, HasLower: true}, {Lower: math.MaxInt64, HasLower: true},
		{Upper: math.MinInt64, HasUpper: true}, {Upper: math.MaxInt64, HasUpper: true},
		{Lower: 1, Upper: -1, HasLower: true, HasUpper: true},
		{Lower: -1, Upper: 1, HasLower: true, HasUpper: true},
		{Lower: 0, Upper: 0, HasLower: true, HasUpper: true},
	}
	contains := func(v Value, x int64) bool {
		return (!v.HasLower || v.Lower <= x) && (!v.HasUpper || x <= v.Upper)
	}
	for _, a := range values {
		for _, b := range values {
			got := a.merge(b)
			for _, x := range []int64{math.MinInt64, -2, -1, 0, 1, 2, math.MaxInt64} {
				if (contains(a, x) || contains(b, x)) && !contains(got, x) {
					t.Fatalf("merge drops x=%d: a=%+v b=%+v got=%+v", x, a, b, got)
				}
			}
		}
	}
}
