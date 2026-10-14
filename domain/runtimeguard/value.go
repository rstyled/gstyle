// Package runtimeguard derives guard and access facts from the shared PDG.
// Its inputs contain only native nodes, operands, bindings and normal flow.
package runtimeguard

import "github.com/rstyled/gstyle/domain/pdg"

type Value struct {
	Identity                           uint64
	Lower, Upper                       int64
	HasLower, HasUpper                 bool
	NonNil, Nil, Guarded, UpperChecked bool
	Integer, Length, BelowLength       bool
	Extent                             int64
	HasExtent                          bool
}

func (v Value) merge(other Value) Value {
	out := v.mergeBounds(other)
	out.UpperChecked = v.UpperChecked || other.UpperChecked
	out.Length = v.Length && other.Length
	out.BelowLength = v.BelowLength && other.BelowLength
	return v.mergePointers(other, out)
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

func (v Value) mergePointers(other, out Value) Value {
	out.NonNil, out.Nil = v.NonNil && other.NonNil, v.Nil && other.Nil
	out.Guarded = v.Guarded && other.Guarded
	if v.Identity == other.Identity {
		out.Identity = v.Identity
	}
	return out
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
