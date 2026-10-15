package runtimeguard

import (
	"strconv"
	"testing"

	"github.com/rstyled/gstyle/domain/pdg"
)

func TestOneSidedComparisonPreservesConcreteValues(t *testing.T) {
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
		for _, concrete := range values {
			for _, other := range values {
				for flags := 0; flags < 8; flags++ {
					for _, right := range []bool{false, true} {
						view := comparisonTestView(t, strconv.FormatInt(other, 10), flags&4 != 0)
						leftValue, rightValue := concrete, other
						if right {
							view.operands[0] = [2]pdg.Ref{3, 2}
							leftValue, rightValue = other, concrete
						}
						marker := Value{Identity: 179}
						state := State{1: {Integer: true, HasLower: flags&1 != 0, Lower: concrete, HasUpper: flags&2 != 0, Upper: concrete}, 7: marker}
						got := view.comparison(1, op, holds(leftValue, rightValue), state)
						if got == nil {
							t.Fatalf("feasible branch rejected: %d %s %d right=%t flags=%d", leftValue, op, rightValue, right, flags)
						}
						value := got[1]
						if value.HasLower && concrete < value.Lower || value.HasUpper && concrete > value.Upper {
							t.Fatalf("concrete value lost: %d %s %d result=%+v right=%t flags=%d", leftValue, op, rightValue, value, right, flags)
						}
						if got[7] != marker {
							t.Fatal("comparison changes an unrelated binding")
						}
						got[11] = marker
						if state[11] != marker {
							t.Fatal("comparison returns a different map")
						}
					}
				}
			}
		}
	}
}
