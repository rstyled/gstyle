// Package runtimeguard derives guard and access facts from the shared PDG.
// Its inputs contain only native nodes, operands, bindings and normal flow.
package runtimeguard

import "github.com/rstyled/gstyle/domain/pdg"

type Value struct {
	Lower, Upper                       int64
	HasLower, HasUpper                 bool
	NonNil, Nil, Guarded, UpperChecked bool
	Integer, Length                    bool
	Extent                             int64
	HasExtent                          bool
}

func (v Value) merge(other Value) Value {
	out := v.mergeBounds(other)
	out.NonNil, out.Nil = v.NonNil && other.NonNil, v.Nil && other.Nil
	out.Guarded, out.UpperChecked = v.Guarded && other.Guarded, v.UpperChecked && other.UpperChecked
	out.Length = v.Length && other.Length
	return out
}

func (v Value) mergeBounds(other Value) Value {
	return Value{HasLower: v.HasLower && other.HasLower, HasUpper: v.HasUpper && other.HasUpper,
		Lower: min(v.Lower, other.Lower), Upper: max(v.Upper, other.Upper)}.normalized()
}

func (v Value) normalized() Value {
	if !v.HasLower {
		v.Lower = 0
	}
	if !v.HasUpper {
		v.Upper = 0
	}
	return v
}

type State map[pdg.Ref]Value

type Access struct {
	Node, Symbol             pdg.Ref
	Index, Literal, Constant bool
	Value                    Value
	Guards                   []pdg.Ref
}

type Report struct{ Accesses []Access }

func copyState(state State) State {
	out := make(State, len(state))
	for symbol, value := range state {
		out[symbol] = value
	}
	return out
}

func equalState(a, b State) bool {
	if len(a) != len(b) {
		return false
	}
	for symbol, value := range a {
		if b[symbol] != value {
			return false
		}
	}
	return true
}
