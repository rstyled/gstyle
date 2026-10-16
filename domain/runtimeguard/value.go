// Package runtimeguard derives guard and access facts from the shared PDG.
// Its inputs contain only native nodes, operands, bindings and normal flow.
package runtimeguard

import (
	"github.com/rstyled/gstyle/domain/pdg"
	"github.com/rstyled/gstyle/domain/runtimeguard/interval"
)

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

func (v Value) Bounds() interval.Range {
	return interval.Range{Lower: v.Lower, Upper: v.Upper, HasLower: v.HasLower, HasUpper: v.HasUpper}
}

func (v Value) mergeBounds(other Value) Value {
	r := interval.Join(v.Bounds(), other.Bounds())
	return Value{Lower: r.Lower, Upper: r.Upper, HasLower: r.HasLower, HasUpper: r.HasUpper}
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

type Call struct {
	Node          pdg.Ref
	FirstArgument Value
}

type Report struct {
	Accesses []Access
	Calls    []Call
}

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
		if other, ok := b[symbol]; !ok || other != value {
			return false
		}
	}
	return true
}
