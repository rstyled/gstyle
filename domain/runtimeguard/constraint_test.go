package runtimeguard

import (
	"math"
	"testing"
)

func TestBoundConstraintsPreserveOtherFacts(t *testing.T) {
	value := Value{Identity: 17, Lower: -9, Upper: 9, HasLower: true, HasUpper: true,
		NonNil: true, Nil: true, Guarded: true, UpperChecked: true,
		Integer: true, Length: true, BelowLength: true, Extent: 8, HasExtent: true}
	for _, bound := range []int64{math.MinInt64, -9, 0, 9, math.MaxInt64} {
		wantLower, wantUpper := value, value
		wantLower.Lower = max(value.Lower, bound)
		wantUpper.Upper = min(value.Upper, bound)
		if got := lowerConstraint(value, bound); got != wantLower {
			t.Fatalf("lower constraint changes other facts: bound=%d got=%+v want=%+v", bound, got, wantLower)
		}
		if got := upperConstraint(value, bound); got != wantUpper {
			t.Fatalf("upper constraint changes other facts: bound=%d got=%+v want=%+v", bound, got, wantUpper)
		}
	}
}

func TestStrictConstraintsPreserveOtherFacts(t *testing.T) {
	value := Value{Identity: 17, Lower: -9, Upper: 9, HasLower: true, HasUpper: true,
		NonNil: true, Nil: true, Guarded: true, UpperChecked: true,
		Integer: true, Length: true, BelowLength: true, Extent: 8, HasExtent: true}
	for _, bound := range []int64{math.MinInt64, math.MinInt64 + 1, -9, 0, 9, math.MaxInt64 - 1, math.MaxInt64} {
		wantLower, wantUpper := value, value
		if bound != math.MaxInt64 {
			wantLower.Lower = max(value.Lower, bound+1)
		}
		if bound != math.MinInt64 {
			wantUpper.Upper = min(value.Upper, bound-1)
		}
		if got := strictLowerConstraint(value, bound); got != wantLower {
			t.Fatalf("strict lower changes other facts: bound=%d got=%+v want=%+v", bound, got, wantLower)
		}
		if got := strictUpperConstraint(value, bound); got != wantUpper {
			t.Fatalf("strict upper changes other facts: bound=%d got=%+v want=%+v", bound, got, wantUpper)
		}
	}
}
