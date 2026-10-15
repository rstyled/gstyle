package runtimeguard

import (
	"github.com/rstyled/gstyle/domain/pdg"
	"github.com/rstyled/gstyle/domain/runtimeguard/interval"
)

var inverse = map[string]string{"<": ">=", "<=": ">", ">": "<=", ">=": "<", "==": "!=", "!=": "=="}

var reverse = map[string]string{"<": ">", "<=": ">=", ">": "<", ">=": "<=", "==": "==", "!=": "!="}

func (v *graphView) assume(ref pdg.Ref, truth bool, state State) State {
	if value := v.fact(ref, "boolean"); value != "" && (value == "true") != truth {
		return nil
	}
	operator := v.graph.Tables.Text(v.attrs(ref).Operator.Value)
	switch operator {
	case "parentheses":
		return v.assume(v.operand(ref, 0), truth, state)
	case "!":
		return v.assume(v.operand(ref, 0), !truth, state)
	case "&&", "||":
		return v.logical(ref, operator, truth, state)
	default:
		return v.comparison(ref, operator, truth, state)
	}
}

func (v *graphView) logical(ref pdg.Ref, operator string, truth bool, state State) State {
	if (operator == "&&") == truth {
		left := v.assume(v.operand(ref, 0), truth, state)
		if left == nil {
			return nil
		}
		return v.assume(v.operand(ref, 1), truth, left)
	}
	left := v.assume(v.operand(ref, 0), truth, copyState(state))
	right := v.assume(v.operand(ref, 1), truth, copyState(state))
	if left == nil && right == nil {
		return nil
	}
	return v.join(left, right)
}

func (v *graphView) comparison(ref pdg.Ref, operator string, truth bool, state State) State {
	if !truth {
		operator = inverse[operator]
	}
	left, right := v.operand(ref, 0), v.operand(ref, 1)
	leftValue, rightValue := v.value(left, state), v.value(right, state)
	if symbol := v.symbol(left); symbol != 0 {
		state = v.constrain(symbol, operator, rightValue, state)
	}
	if state == nil {
		return nil
	}
	if symbol := v.symbol(right); symbol != 0 {
		state = v.constrain(symbol, reverse[operator], leftValue, state)
	}
	return state
}

func (v *graphView) constrain(symbol pdg.Ref, operator string, bound Value, state State) State {
	if !v.trackedBinding(symbol) {
		return state
	}
	value := v.binding(symbol, state)
	if bound.Nil {
		return v.nilConstraint(symbol, operator, value, state)
	}
	value = lengthConstraint(value, operator, bound)
	if exactInteger(bound) {
		value = integerConstraint(value, operator, bound.Lower)
	}
	if v.infeasibleConstraint(symbol, operator, bound, value) {
		return nil
	}
	state[symbol] = value
	return state
}

func (v *graphView) nilConstraint(symbol pdg.Ref, operator string, value Value, state State) State {
	if v.kinds[symbol-1] != `"pointer"` {
		return state
	}
	switch operator {
	case "==":
		if value.NonNil {
			return nil
		}
		value.NonNil, value.Nil = false, true
	case "!=":
		if value.Nil {
			return nil
		}
		value.NonNil, value.Nil, value.Guarded = true, false, true
	default:
		return state
	}
	v.propagatePointer(symbol, value, state)
	return state
}

func (v *graphView) infeasibleConstraint(symbol pdg.Ref, operator string, bound, value Value) bool {
	return emptyRange(value) || v.impossibleSignedComparison(symbol, operator, bound)
}

func (v *graphView) impossibleSignedComparison(symbol pdg.Ref, operator string, bound Value) bool {
	if v.kinds[symbol-1] != `"signed"` || !exactInteger(bound) {
		return false
	}
	switch operator {
	case "<":
		return bound.Lower == -int64(^uint64(0)>>1)-1
	case ">":
		return bound.Lower == int64(^uint64(0)>>1)
	default:
		return false
	}
}

func integerConstraint(value Value, operator string, bound int64) Value {
	r := interval.Constrain(value.Bounds(), operator, bound)
	value.Lower, value.Upper = r.Lower, r.Upper
	value.HasLower, value.HasUpper = r.HasLower, r.HasUpper
	return value
}

func lengthConstraint(value Value, operator string, bound Value) Value {
	if !bound.Length {
		return value
	}
	if operator != "<" && operator != "<=" {
		return value
	}
	value.UpperChecked = true
	value.BelowLength = value.BelowLength || operator == "<"
	return value
}

func exactInteger(value Value) bool {
	return value.HasLower && value.HasUpper && value.Lower == value.Upper
}

func emptyRange(value Value) bool {
	return value.HasLower && value.HasUpper && value.Lower > value.Upper
}
