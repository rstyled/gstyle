package runtimeguard

import "github.com/rstyled/gstyle/domain/pdg"

func (v *graphView) incrementValue(ref, symbol pdg.Ref, fallback Value, state State) Value {
	operator := v.fact(ref, "signed-step64")
	if operator != `"++"` && operator != `"--"` {
		return fallback
	}
	return integerStep(v.binding(symbol, state), operator == `"++"`)
}

func integerStep(value Value, increment bool) Value {
	if increment {
		return incrementInteger(value)
	}
	return decrementInteger(value)
}

func incrementInteger(value Value) Value {
	if !value.BelowLength && (!value.HasUpper || value.Upper == int64(^uint64(0)>>1)) {
		return Value{}
	}
	return Value{HasLower: value.HasLower, Lower: value.Lower}
}

func decrementInteger(value Value) Value {
	if !value.HasLower || value.Lower == -int64(^uint64(0)>>1)-1 {
		return Value{}
	}
	return Value{HasUpper: value.HasUpper, Upper: value.Upper}
}
