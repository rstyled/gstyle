package runtimeguard

import (
	"math"
	"testing"
)

func TestIntegerStepCases(t *testing.T) {
	cases := []struct {
		name      string
		input     Value
		increment bool
		want      Value
	}{
		{"unknown increment", Value{}, true, Value{}},
		{"unknown decrement", Value{}, false, Value{}},
		{"negative increment", Value{Lower: -8, Upper: -1, HasLower: true, HasUpper: true}, true, Value{Lower: -8, HasLower: true}},
		{"negative decrement", Value{Lower: -8, Upper: -1, HasLower: true, HasUpper: true}, false, Value{Upper: -1, HasUpper: true}},
		{"increment can overflow", Value{Lower: 0, HasLower: true}, true, Value{}},
		{"increment maximum", Value{Lower: math.MaxInt64, Upper: math.MaxInt64, HasLower: true, HasUpper: true}, true, Value{}},
		{"decrement minimum", Value{Lower: math.MinInt64, Upper: math.MinInt64, HasLower: true, HasUpper: true}, false, Value{}},
		{"decrement can overflow", Value{Upper: 0, HasUpper: true}, false, Value{}},
		{"guarded increment", Value{Lower: 0, HasLower: true, BelowLength: true}, true, Value{HasLower: true}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := integerStep(tc.input, tc.increment); got != tc.want {
				t.Fatalf("got=%+v want=%+v", got, tc.want)
			}
		})
	}
}

func TestIntegerStepRepresentsGoResults(t *testing.T) {
	points := []int64{math.MinInt64, math.MinInt64 + 1, -8, -1, 0, 1, 8, math.MaxInt64 - 1, math.MaxInt64}
	for flags := 0; flags < 8; flags++ {
		for _, lower := range points {
			for _, upper := range points {
				v := Value{Lower: lower, Upper: upper, HasLower: flags&1 != 0, HasUpper: flags&2 != 0, BelowLength: flags&4 != 0}
				for _, increment := range []bool{false, true} {
					out := integerStep(v, increment)
					for _, x := range points {
						if !numericMember(v, x) || (increment && v.BelowLength && x == math.MaxInt64) {
							continue
						}
						y := x - 1
						if increment {
							y = x + 1
						}
						if !numericMember(out, y) {
							t.Fatalf("lost Go result: input=%+v increment=%v x=%d y=%d output=%+v", v, increment, x, y, out)
						}
					}
				}
			}
		}
	}
}

func TestIntegerStepClearsMetadata(t *testing.T) {
	v := Value{Identity: 17, Lower: -1, Upper: 1, HasLower: true, HasUpper: true,
		NonNil: true, Nil: true, Guarded: true, UpperChecked: true, Integer: true,
		Length: true, BelowLength: true, Extent: 9, HasExtent: true}
	for _, increment := range []bool{false, true} {
		out := integerStep(v, increment)
		if out.Identity != 0 || out.NonNil || out.Nil || out.Guarded || out.UpperChecked || out.Integer || out.Length || out.BelowLength || out.Extent != 0 || out.HasExtent {
			t.Fatalf("stale metadata: %+v", out)
		}
	}
}
