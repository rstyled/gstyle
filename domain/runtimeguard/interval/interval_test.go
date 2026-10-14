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
