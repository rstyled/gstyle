package runtimeguard

import "github.com/rstyled/gstyle/domain/pdg"

func (v *graphView) incrementValue(ref, symbol pdg.Ref, fallback Value, state State) Value {
	if v.fact(ref, "length-increment") != "true" {
		return fallback
	}
	previous := v.binding(symbol, state)
	if !previous.BelowLength {
		return fallback
	}
	// Keep the old lower bound as a widening. Increasing it on every trip
	// would prevent convergence. Discard the upper bound and length check:
	// after ++ the value may equal len, so another increment needs a new guard.
	return Value{HasLower: previous.HasLower, Lower: previous.Lower}
}
