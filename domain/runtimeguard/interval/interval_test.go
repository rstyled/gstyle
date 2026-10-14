package interval

import (
	"math"
	"testing"
)

func members(r Range, x int64) bool {
	return (!r.HasLower || r.Lower <= x) && (!r.HasUpper || x <= r.Upper)
}

func samples() []Range {
	points := []int64{math.MinInt64, -1, 0, 1, math.MaxInt64}
	var out []Range
	for flags := 0; flags < 4; flags++ {
		for _, lower := range points {
			for _, upper := range points {
				out = append(out, Range{Lower: lower, Upper: upper, HasLower: flags&1 != 0, HasUpper: flags&2 != 0})
			}
		}
	}
	return out
}

func TestJoinPreservesBothSets(t *testing.T) {
	points := []int64{math.MinInt64, -2, -1, 0, 1, 2, math.MaxInt64}
	for _, a := range samples() {
		for _, b := range samples() {
			joined := Join(a, b)
			for _, x := range points {
				if (members(a, x) || members(b, x)) && !members(joined, x) {
					t.Fatalf("join drops %d: a=%+v b=%+v result=%+v", x, a, b, joined)
				}
			}
		}
	}
}

func TestNegativeMemberRequiresWarning(t *testing.T) {
	for _, r := range samples() {
		for _, x := range []int64{math.MinInt64, -2, -1} {
			if members(r, x) && !NegativeIndex(true, true, r) {
				t.Fatalf("negative member %d is called safe: %+v", x, r)
			}
		}
		if NegativeIndex(false, true, r) || NegativeIndex(true, false, r) {
			t.Fatalf("warning without the pattern trigger: %+v", r)
		}
	}
}

func TestConstraintsIntersectSets(t *testing.T) {
	points := []int64{math.MinInt64, -2, -1, 0, 1, 2, math.MaxInt64}
	for _, r := range samples() {
		for _, bound := range points {
			lower, upper := Lower(r, bound), Upper(r, bound)
			for _, x := range points {
				if members(lower, x) != (members(r, x) && x >= bound) {
					t.Fatalf("lower intersection: r=%+v bound=%d x=%d result=%+v", r, bound, x, lower)
				}
				if members(upper, x) != (members(r, x) && x <= bound) {
					t.Fatalf("upper intersection: r=%+v bound=%d x=%d result=%+v", r, bound, x, upper)
				}
			}
		}
	}
}

func TestStrictConstraints(t *testing.T) {
	points := []int64{math.MinInt64, math.MinInt64 + 1, -2, -1, 0, 1, 2, math.MaxInt64 - 1, math.MaxInt64}
	for _, r := range samples() {
		for _, bound := range points {
			lower, upper := StrictLower(r, bound), StrictUpper(r, bound)
			if bound == math.MaxInt64 && lower != r {
				t.Fatalf("strict lower overflows: r=%+v result=%+v", r, lower)
			}
			if bound == math.MinInt64 && upper != r {
				t.Fatalf("strict upper overflows: r=%+v result=%+v", r, upper)
			}
			for _, x := range points {
				if members(lower, x) != (members(r, x) && (bound == math.MaxInt64 || x > bound)) {
					t.Fatalf("strict lower: r=%+v bound=%d x=%d result=%+v", r, bound, x, lower)
				}
				if members(upper, x) != (members(r, x) && (bound == math.MinInt64 || x < bound)) {
					t.Fatalf("strict upper: r=%+v bound=%d x=%d result=%+v", r, bound, x, upper)
				}
			}
		}
	}
}

func TestConstrainOperatorSemantics(t *testing.T) {
	cases := []struct {
		operator string
		holds    func(int64, int64) bool
	}{
		{">=", func(x, b int64) bool { return x >= b }},
		{">", func(x, b int64) bool { return b == math.MaxInt64 || x > b }},
		{"<", func(x, b int64) bool { return b == math.MinInt64 || x < b }},
		{"<=", func(x, b int64) bool { return x <= b }},
		{"==", func(x, b int64) bool { return x == b }},
	}
	points := []int64{math.MinInt64, math.MinInt64 + 1, -1, 0, 1, math.MaxInt64 - 1, math.MaxInt64}
	for _, r := range samples() {
		for _, bound := range points {
			for _, tc := range cases {
				got := Constrain(r, tc.operator, bound)
				for _, x := range points {
					if members(got, x) != (members(r, x) && tc.holds(x, bound)) {
						t.Fatalf("operator=%q bound=%d x=%d range=%+v result=%+v", tc.operator, bound, x, r, got)
					}
				}
			}
			for _, operator := range []string{"", "!=", "≥", ">= ", "\x00", "unknown"} {
				if got := Constrain(r, operator, bound); got != r {
					t.Fatalf("unsupported operator %q changes range: got=%+v want=%+v", operator, got, r)
				}
			}
		}
	}
}
